package transfer

import (
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidSMSCode       = errors.New("invalid or expired sms code")

	// 匹配队列相关错误
	ErrPlayerAlreadyInQueue = errors.New("user already in queue")
	ErrPlayerNotInQueue     = errors.New("user not in queue")
	ErrQueueEmpty           = errors.New("queue is empty")
	ErrNotEnoughPlayers     = errors.New("not enough players in queue")

	// 用户路由相关错误
	ErrRouterNotFound = errors.New("user router not found")

	ErrMongodb = errors.New("mongodb error happen")
	ErrRedis   = errors.New("redis error happen")
)

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
	ErrArgument         = errors.New("argument error")
	ErrService          = errors.New("service error")
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

func MapError(err error) error {
	switch {
	case errors.Is(err, ErrAccountAlreadyExists):
		return status.Error(codes.AlreadyExists, "account already exists")
	case errors.Is(err, ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, "invalid password")
	case errors.Is(err, ErrInvalidSMSCode):
		return status.Error(codes.InvalidArgument, "invalid or expired sms code")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
