package conn

import (
	"core/infrastructure/message/node"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/transfer"
)

type HandlerFunc func(session *Session, body []byte) (any, error)

type MessageTypeHandler map[string]HandlerFunc

// 玩家消息路由
func (w *Worker) injectDefaultHandlers() {
	w.clientHandlers[protocol.Handshake] = w.handshakeHandler
	w.clientHandlers[protocol.HandshakeAck] = w.handshakeAckHandler
	w.clientHandlers[protocol.Heartbeat] = w.heartbeatHandler
	w.clientHandlers[protocol.Data] = w.messageHandler
	w.clientHandlers[protocol.Kick] = w.kickHandler

	w.MessageTypeHandlers[transfer.JoinQueue] = joinQueueHandler
}

// nats 消息路由
func (w *Worker) injectMiddleWorkerHandler() {
	subHandler := make(node.SubscriberHandler)
	subHandler[transfer.MatchingSuccess] = w.handlerMatchSuccess

	w.MiddleWorker.RegisterPushHandler(w.handlePush)
	w.MiddleWorker.RegisterHandlers(subHandler)
}
