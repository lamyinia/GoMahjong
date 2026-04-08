package codec

import (
	"fmt"

	"webtest/proto"

	"google.golang.org/protobuf/proto"
)

// Envelope is an alias to the proto-generated type
type Envelope = proto.Envelope

// EnvelopeCodec handles envelope serialization
type EnvelopeCodec struct {
	lengthCodec *LengthFieldCodec
}

// NewEnvelopeCodec creates a new envelope codec
func NewEnvelopeCodec() *EnvelopeCodec {
	return &EnvelopeCodec{
		lengthCodec: NewLengthFieldCodec(),
	}
}

// EncodeEnvelope encodes an envelope with length prefix
// route: message route (e.g., "auth.login")
// msg: protobuf message to encode
// seq: optional client sequence number
func (c *EnvelopeCodec) EncodeEnvelope(route string, msg proto.Message, seq uint64) ([]byte, error) {
	// Serialize payload
	var payload []byte
	var err error
	if msg != nil {
		payload, err = proto.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	// Create envelope proto
	envelope := &Envelope{
		Route:     route,
		Payload:   payload,
		ClientSeq: seq,
	}

	// Serialize envelope
	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	// Add length prefix
	return c.lengthCodec.Encode(envelopeBytes), nil
}

// DecodeEnvelope decodes a length-prefixed envelope
func (c *EnvelopeCodec) DecodeEnvelope(data []byte) (*Envelope, error) {
	// Decode length prefix
	envelopeBytes, err := c.lengthCodec.Decode(nil)
	if err != nil {
		return nil, err
	}

	// For DecodeEnvelope, data should already be the payload after length
	envelopeBytes = data

	// Parse envelope
	envelope := &Envelope{}
	if err := proto.Unmarshal(envelopeBytes, envelope); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	return envelope, nil
}

// DecodeRaw decodes raw bytes (without length prefix) into envelope
func (c *EnvelopeCodec) DecodeRaw(data []byte) (*Envelope, error) {
	envelope := &Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	return envelope, nil
}

// EncodeRaw encodes envelope without length prefix
func (c *EnvelopeCodec) EncodeRaw(route string, payload []byte, seq uint64) ([]byte, error) {
	envelope := &Envelope{
		Route:     route,
		Payload:   payload,
		ClientSeq: seq,
	}
	return proto.Marshal(envelope)
}
