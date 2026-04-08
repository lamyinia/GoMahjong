package codec

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// LengthFieldLength is the length of the length field in bytes
	LengthFieldLength = 4
	// MaxMessageSize is the maximum allowed message size
	MaxMessageSize = 65536 // 64KB
)

// LengthFieldCodec encodes/decodes messages with a 4-byte big-endian length prefix
type LengthFieldCodec struct{}

// NewLengthFieldCodec creates a new length field codec
func NewLengthFieldCodec() *LengthFieldCodec {
	return &LengthFieldCodec{}
}

// Encode wraps data with a 4-byte big-endian length prefix
func (c *LengthFieldCodec) Encode(data []byte) []byte {
	buf := make([]byte, LengthFieldLength+len(data))
	binary.BigEndian.PutUint32(buf[:LengthFieldLength], uint32(len(data)))
	copy(buf[LengthFieldLength:], data)
	return buf
}

// Decode reads a length-prefixed message from reader
func (c *LengthFieldCodec) Decode(r io.Reader) ([]byte, error) {
	// Read length prefix
	lenBuf := make([]byte, LengthFieldLength)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d", length)
	}

	// Read payload
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return payload, nil
}

// EncodeWithLength encodes data and returns length + data
func (c *LengthFieldCodec) EncodeWithLength(data []byte) (uint32, []byte) {
	return uint32(len(data)), data
}
