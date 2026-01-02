package node

import "errors"

// 连接管理错误
var (
	ErrConnectionNotFound = errors.New("连接未找到")
	ErrTooManyConnections = errors.New("连接数超过限制")
	ErrConnectionClosed   = errors.New("连接已关闭")
	ErrUserNotFound       = errors.New("用户未找到")
)

// 会话管理错误
var (
	ErrSessionNotFound      = errors.New("会话未找到")
	ErrSessionAlreadyExists = errors.New("会话已存在")
	ErrSessionExpired       = errors.New("会话已过期")
)

// 消息分发错误
var (
	ErrHandlerNotFound      = errors.New("处理器未找到")
	ErrHandlerAlreadyExists = errors.New("处理器已存在")
	ErrInvalidRoute         = errors.New("无效的路由")
	ErrInvalidMessage       = errors.New("无效的消息")
)

// 远程通信错误
var (
	ErrNotConnected      = errors.New("未连接到远程服务")
	ErrAlreadySubscribed = errors.New("已订阅该主题")
	ErrNotSubscribed     = errors.New("未订阅该主题")
	ErrPublishFailed     = errors.New("发布消息失败")
)

// 协议适配错误
var (
	ErrUnsupportedProtocol      = errors.New("不支持的协议")
	ErrProtocolConversionFailed = errors.New("协议转换失败")
)
