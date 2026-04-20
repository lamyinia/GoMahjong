package protocol

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"game/infrastructure/log"
	"io"
	"strings"
)

var (
	routes = make(map[string]uint16)
	codes  = make(map[uint16]string)
)

type PackageType byte
type MessageType byte

const (
	None         PackageType = 0x00
	Handshake    PackageType = 0x01
	HandshakeAck PackageType = 0x02
	Heartbeat    PackageType = 0x03
	Data         PackageType = 0x04
	Kick         PackageType = 0x05
)

const (
	Request  MessageType = 0x00
	Notify   MessageType = 0x01
	Response MessageType = 0x02
	Push     MessageType = 0x03
)

const (
	RouteCompressMask = 0x01
	MessageHeadLength = 0x02
	TypeMask          = 0x07
	GZIPMask          = 0x10
	ErrorMask         = 0x20
)

const (
	HeaderLen     = 4
	MaxPacketSize = 1 << 24
)

const (
	messageFlagBytes = 1
)

type Packet struct {
	Type PackageType
	Len  uint32
	Body any
}

func (p Packet) ParseBody() *Message {
	if p.Type == Data {
		body := p.Body.(Message)
		return &body
	}
	return nil
}

type HandshakeBody struct {
	Sys Sys `json:"sys"`
}

type Sys struct {
	Type         string            `json:"type"`
	Version      string            `json:"version"`
	ProtoVersion uint8             `json:"protoVersion"`
	Heartbeat    uint8             `json:"heartbeat"`
	Dict         map[string]uint16 `json:"dict"`
	Serializer   string            `json:"serializer"`
}

type HandshakeResponse struct {
	Code uint16 `json:"code"`
	Sys  Sys    `json:"sys"`
}

type Message struct {
	Type            MessageType
	ID              uint
	Route           string
	Data            []byte
	routeCompressed bool
	Error           bool
}

func Wrap(packageType PackageType, body []byte) ([]byte, error) {
	if packageType == None {
		return nil, errors.New("encode unsupported packageType")
	}
	if len(body) > MaxPacketSize {
		return nil, errors.New("encode body size too big")
	}
	buf := make([]byte, len(body)+HeaderLen)
	buf[0] = byte(packageType)
	copy(buf[1:HeaderLen], IntToBytes(len(body)))
	copy(buf[HeaderLen:], body)
	return buf, nil
}

func Decode(payload []byte) (*Packet, error) {
	if len(payload) < HeaderLen {
		return nil, errors.New("decode 格式错误")
	}
	p := &Packet{}
	p.Type = PackageType(payload[0])
	p.Len = uint32(BytesToInt(payload[1:HeaderLen]))

	if p.Type == Handshake {
		var body HandshakeBody
		err := json.Unmarshal(payload[HeaderLen:], &body)
		if err != nil {
			return nil, err
		}
		if body.Sys.Dict != nil {
			SetDictionary(body.Sys.Dict)
		}
		p.Body = body
	}
	if p.Type == Data {
		m, err := MessageDecode(payload[HeaderLen:])
		if err != nil {
			return nil, err
		}
		p.Body = m
	}
	return p, nil
}

func MessageEncode(m *Message) ([]byte, error) {
	code, compressed := routes[m.Route]
	buf := make([]byte, 0)
	buf = encodeMessageFlag(m.Type, compressed, buf)
	if messageHasID(m.Type) {
		buf = encodeMessageID(m, buf)
	}
	if messageHasRoute(m.Type) {
		buf = encodeMessageRoute(code, compressed, m.Route, buf)
	}
	if m.Data != nil {
		buf = append(buf, m.Data...)
	}
	return buf, nil
}

func MessageDecode(body []byte) (Message, error) {
	m := Message{}
	flag := body[0]
	m.Type = MessageType((flag >> 1) & TypeMask)
	if m.Type < Request || m.Type > Push {
		return m, errors.New("invalid transfer type")
	}
	offset := 1
	dataLen := len(body)
	if m.Type == Request || m.Type == Response {
		id := uint(0)
		for i := offset; i < dataLen; i++ {
			b := body[i]
			id += uint(b&0x7F) << uint(7*(i-offset))
			if b < 128 {
				offset = i + 1
				break
			}
		}
		m.ID = id
	}
	if offset > dataLen {
		return m, errors.New("invalid transfer")
	}
	m.Error = flag&ErrorMask == ErrorMask
	if m.Type == Request || m.Type == Notify || m.Type == Push {
		if flag&RouteCompressMask == 1 {
			m.routeCompressed = true
			code := binary.BigEndian.Uint16(body[offset:(offset + 2)])
			route, found := GetRoute(code)
			if !found {
				return m, errors.New("route info not found in dictionary")
			}
			m.Route = route
			offset += 2
		} else {
			m.routeCompressed = false
			rl := body[offset]
			offset++
			m.Route = string(body[offset:(offset + int(rl))])
			offset += int(rl)
		}
	}
	if offset > dataLen {
		return m, errors.New("invalid transfer")
	}
	m.Data = body[offset:]
	var err error
	if flag&GZIPMask == GZIPMask {
		m.Data, err = InflateData(m.Data)
		if err != nil {
			return m, err
		}
	}
	return m, nil
}

func messageHasRoute(t MessageType) bool {
	return t == Request || t == Notify || t == Push
}

func messageHasID(messageType MessageType) bool {
	return messageType == Request || messageType == Response
}

func encodeMessageID(m *Message, buf []byte) []byte {
	id := m.ID
	for {
		b := byte(id % 128)
		id >>= 7
		if id != 0 {
			buf = append(buf, b+128)
		} else {
			buf = append(buf, b)
			break
		}
	}
	return buf
}

func encodeMessageFlag(t MessageType, compressed bool, buf []byte) []byte {
	flag := byte(t) << 1
	if compressed {
		flag |= RouteCompressMask
	}
	return append(buf, flag)
}

func encodeMessageRoute(code uint16, compressed bool, route string, buf []byte) []byte {
	if compressed {
		buf = append(buf, byte((code>>8)&0xFF))
		buf = append(buf, byte(code&0xFF))
	} else {
		buf = append(buf, byte(len(route)))
		buf = append(buf, []byte(route)...)
	}
	return buf
}

func SetDictionary(dict map[string]uint16) {
	if dict == nil {
		return
	}
	for route, code := range dict {
		r := strings.TrimSpace(route)
		if _, ok := routes[r]; ok {
			log.Error(fmt.Sprintf("重复路由1(route: %s, code: %d)", r, code))
			return
		}
		if _, ok := codes[code]; ok {
			log.Error(fmt.Sprintf("重复路由2(route: %s, code: %d)", r, code))
			return
		}
		routes[r] = code
		codes[code] = r
	}
}

func GetRoute(code uint16) (route string, found bool) {
	route, found = codes[code]
	return route, found
}

func InflateData(data []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}
