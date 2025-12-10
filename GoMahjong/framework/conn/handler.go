package conn

import (
	"framework/dto"
	"framework/node"
	"framework/protocol"
)

type HandlerFunc func(session *Session, body []byte) (any, error)

type MessageTypeHandler map[string]HandlerFunc

func (w *Worker) injectDefaultHandlers() {
	w.clientHandlers[protocol.Handshake] = w.handshakeHandler
	w.clientHandlers[protocol.HandshakeAck] = w.handshakeAckHandler
	w.clientHandlers[protocol.Heartbeat] = w.heartbeatHandler
	w.clientHandlers[protocol.Data] = w.messageHandler
	w.clientHandlers[protocol.Kick] = w.kickHandler

	w.MessageTypeHandlers[dto.JoinQueue] = joinQueueHandler
}

func (w *Worker) injectMiddleWorkerHandler() {
	subHandler := make(node.SubscriberHandler)
	subHandler[dto.MatchingSuccess] = w.handlerMatchSuccess

	w.MiddleWorker.RegisterPushHandler(w.handlePush) // 注册推送路由
	w.MiddleWorker.RegisterHandlers(subHandler)
}
