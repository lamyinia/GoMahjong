package stream

import "framework/protocal"

type SessionType int
type DataType int

type SessionData struct {
	SingleData map[string]any //只保存当前cid
	AllData    map[string]any //所有cid 都需要保存
}

type ServicePacket struct {
	ConnID      string
	Body        *protocal.Message
	Source      string
	Destination string
	Route       string
	UserID      string
	ConnectorID string
	SessionData *SessionData
	SessionType SessionType
	PushUser    []string
}

const (
	Single DataType = iota
	All
)

const (
	Normal SessionType = iota
	Session
)
