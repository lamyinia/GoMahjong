package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/log"
	bizpb "webtest/proto"

	"google.golang.org/protobuf/proto"
)

// TCPClient connects to the C++ game server
type TCPClient struct {
	host      string
	port      int
	playerID  string
	conn      net.Conn
	connMu    sync.Mutex
	sendCh    chan []byte
	RecvChan  chan *TCPMessage
	done      chan struct{}
	connected bool
	seq       uint64
}

// TCPMessage represents a received message
type TCPMessage struct {
	Route   string
	Payload []byte
}

// NewTCPClient creates a new TCP client
func NewTCPClient(host string, port int, playerID string) *TCPClient {
	return &TCPClient{
		host:     host,
		port:     port,
		playerID: playerID,
		sendCh:   make(chan []byte, 100),
		RecvChan: make(chan *TCPMessage, 100),
		done:     make(chan struct{}),
		seq:      0,
	}
}

// Connect establishes TCP connection to game server
func (c *TCPClient) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Start read/write loops
	go c.readLoop()
	go c.writeLoop()

	log.Debug("TCP connected", "addr", addr, "playerId", c.playerID)
	return nil
}

// Close disconnects from server
func (c *TCPClient) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if !c.connected {
		return
	}

	c.connected = false
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns connection status
func (c *TCPClient) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connected
}

// SendAuth sends authentication request
func (c *TCPClient) SendAuth(token string) error {
	authReq := &bizpb.AuthRequest{
		Token:    token,
		DeviceId: "webtest",
		Version:  "1.0.0",
	}
	return c.SendProtoMessage("auth.login", authReq)
}

// SendProtoMessage sends a protobuf message with envelope wrapper
func (c *TCPClient) SendProtoMessage(route string, msg proto.Message) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Serialize payload
	var payloadBytes []byte
	var err error
	if msg != nil {
		payloadBytes, err = proto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	}

	// Create envelope
	seq := atomic.AddUint64(&c.seq, 1)
	envelope := &bizpb.Envelope{
		Route:     route,
		Payload:   payloadBytes,
		ClientSeq: seq,
	}

	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	// Add length prefix (4 bytes, big endian)
	buf := make([]byte, 4+len(envelopeBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(envelopeBytes)))
	copy(buf[4:], envelopeBytes)

	// Send to write channel
	select {
	case c.sendCh <- buf:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// SendMessage sends a raw payload message with envelope wrapper
func (c *TCPClient) SendMessage(route string, payload interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	var payloadBytes []byte
	var err error

	switch p := payload.(type) {
	case []byte:
		payloadBytes = p
	case proto.Message:
		payloadBytes, err = proto.Marshal(p)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	default:
		return fmt.Errorf("unsupported payload type, use SendProtoMessage for proto messages")
	}

	// 创建 envelope
	seq := atomic.AddUint64(&c.seq, 1)
	envelope := &bizpb.Envelope{
		Route:     route,
		Payload:   payloadBytes,
		ClientSeq: seq,
	}

	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	// Add length prefix (4 bytes, big endian)
	buf := make([]byte, 4+len(envelopeBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(envelopeBytes)))
	copy(buf[4:], envelopeBytes)

	select {
	case c.sendCh <- buf:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// writeLoop sends messages to server
func (c *TCPClient) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case data := <-c.sendCh:
			c.connMu.Lock()
			if c.conn != nil {
				if _, err := c.conn.Write(data); err != nil {
					log.Error("TCP write error", "err", err)
					c.connMu.Unlock()
					c.Close()
					return
				}
			}
			c.connMu.Unlock()
		}
	}
}

// readLoop receives messages from 游戏服务器
func (c *TCPClient) readLoop() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
			lenBuf := make([]byte, 4)
			if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
				if err != io.EOF {
					log.Error("TCP read length error", "err", err)
				}
				return
			}

			length := binary.BigEndian.Uint32(lenBuf)
			if length > 65536 { // Max 64KB
				log.Error("Message too large", "length", length)
				return
			}

			// Read payload
			payloadBuf := make([]byte, length)
			if _, err := io.ReadFull(c.conn, payloadBuf); err != nil {
				log.Error("TCP read payload error", "err", err)
				return
			}
			envelope := &bizpb.Envelope{}
			if err := proto.Unmarshal(payloadBuf, envelope); err != nil {
				log.Error("Parse envelope error", "err", err)
				continue
			}

			msg := &TCPMessage{
				Route:   envelope.Route,
				Payload: envelope.Payload,
			}

			select {
			case c.RecvChan <- msg:
			default:
				log.Warn("Recv buffer full, dropping message", "route", envelope.Route)
			}
		}
	}
}
