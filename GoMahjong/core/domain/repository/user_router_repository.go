package repository

import (
	"context"
	"time"
)

// UserRouterInfo 用户路由信息
type UserRouterInfo struct {
	GameTopic      string // game 节点 Topic（必需）
	ConnectorTopic string // connector 节点 Topic（可选，重连时可能为空）
}

// UserRouterRepository 用户路由仓储接口
// 用于管理用户到 game 节点的路由映射，用于断线重连
type UserRouterRepository interface {
	// SaveRouter 保存用户路由
	// userID: 用户 ID
	// info: 路由信息（gameTopic 必需，connectorTopic 可选）
	// ttl: 过期时间（游戏结束后自动清理）
	SaveRouter(ctx context.Context, userID string, info *UserRouterInfo, ttl time.Duration) error

	// GetRouter 获取用户路由
	// userID: 用户 ID
	// 返回：路由信息，如果不存在返回错误
	GetRouter(ctx context.Context, userID string) (*UserRouterInfo, error)

	// DeleteRouter 删除用户路由（游戏结束时调用）
	DeleteRouter(ctx context.Context, userID string) error

	// DeleteRouters 批量删除用户路由（房间内所有玩家）
	DeleteRouters(ctx context.Context, userIDs []string) error

	// ExistsRouter 检查用户路由是否存在
	ExistsRouter(ctx context.Context, userID string) (bool, error)
}
