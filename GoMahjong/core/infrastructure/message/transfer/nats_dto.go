package transfer

import (
	"core/infrastructure/message/protocol"
)

type SessionData struct {
	SingleData map[string]any //只保存当前 connID
	AllData    map[string]any //所有 connID 都需要保存
}

// ServicePacket 用于服务节点之间通信，有两层路由
type ServicePacket struct {
	Body        *protocol.Message
	Source      string
	Destination string
	Route       string
	SessionData *SessionData
	PushUser    []string
}
