package transfer

import (
	"connector/infrastructure/message/protocol"
)

type SessionData struct {
	SingleData map[string]any
	AllData    map[string]any
}

type ServicePacket struct {
	Body        *protocol.Message
	Source      string
	Destination string
	Route       string
	SessionData *SessionData
	PushUser    []string
}
