package dto

import "errors"

// 连接相关错误
var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrSendChanFull     = errors.New("send channel full")
	ErrNotConnected     = errors.New("not connected")
	ErrAlreadyConnected = errors.New("already connected")
)

// 会话相关错误
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// 消息相关错误
var (
	ErrInvalidRoute     = errors.New("invalid route")
	ErrHandlerNotFound  = errors.New("handler not found")
	ErrInvalidMessage   = errors.New("invalid message")
	ErrMessageMarshal   = errors.New("message marshal error")
	ErrMessageUnmarshal = errors.New("message unmarshal error")
)

// 远程通信相关错误
var (
	ErrRemoteServiceUnavailable = errors.New("remote service unavailable")
	ErrRemoteSendFailed         = errors.New("remote send failed")
	ErrRemoteTimeout            = errors.New("remote timeout")
)

// 负载均衡相关错误
var (
	ErrNoAvailableInstance = errors.New("no available instance")
	ErrLoadBalanceFailed   = errors.New("load balance failed")
)

// 重连相关错误
var (
	ErrReconnectFailed    = errors.New("reconnect failed")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)
