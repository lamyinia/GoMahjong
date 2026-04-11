package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// MessageCodec provides high-level message encoding/decoding
type MessageCodec struct {
	envelopeCodec *EnvelopeCodec
	seq           uint64
}

// NewMessageCodec creates a new message codec
func NewMessageCodec() *MessageCodec {
	return &MessageCodec{
		envelopeCodec: NewEnvelopeCodec(),
		seq:           0,
	}
}

// Encode encodes a protobuf message with envelope and length prefix
func (c *MessageCodec) Encode(route string, msg proto.Message) ([]byte, error) {
	c.seq++
	return c.envelopeCodec.EncodeEnvelope(route, msg, c.seq)
}

// EncodeRaw encodes raw payload bytes with envelope and length prefix
func (c *MessageCodec) EncodeRaw(route string, payload []byte) ([]byte, error) {
	c.seq++
	return c.envelopeCodec.EncodeRaw(route, payload, c.seq)
}

// Decode decodes a message from raw bytes (after length prefix removed)
func (c *MessageCodec) Decode(data []byte) (*Envelope, error) {
	return c.envelopeCodec.DecodeRaw(data)
}

// UnmarshalPayload unmarshals the envelope payload into a protobuf message
func UnmarshalPayload[T proto.Message](envelope *Envelope, target T) error {
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	return proto.Unmarshal(envelope.Payload, target)
}

// GetRoute returns the route from an envelope
func GetRoute(e *Envelope) string {
	return e.Route
}

// GetPayload returns the raw payload bytes
func GetPayload(e *Envelope) []byte {
	return e.Payload
}

// GetSeq returns the client sequence number
func GetSeq(e *Envelope) uint64 {
	return e.ClientSeq
}
