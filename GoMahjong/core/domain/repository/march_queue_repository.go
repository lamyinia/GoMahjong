package repository

import (
	"context"
)

// MarchQueueRepository 匹配队列仓储接口
type MarchQueueRepository interface {
	// JoinQueue 加入匹配队列（自动维护 userID -> poolID 映射）
	JoinQueue(ctx context.Context, poolID, userID string, score float64) error

	// RemoveFromQueue 从队列中移除玩家（自动查找 poolID）
	RemoveFromQueue(ctx context.Context, userID string) error

	// IsInQueue 检查玩家是否在队列中（自动查找 poolID）
	// 返回：是否在队列中，所在的 poolID，错误
	IsInQueue(ctx context.Context, userID string) (bool, string, error)

	// GetUserPool 获取用户所在的匹配池
	GetUserPool(ctx context.Context, userID string) (string, error)

	// PopPlayers 从队列中取出指定数量的玩家（同时清理映射）
	PopPlayers(ctx context.Context, poolID string, count int) ([]string, error)

	// GetQueueSize 获取队列当前大小
	GetQueueSize(ctx context.Context, poolID string) (int, error)
}
