package repository

import (
	"context"
	"core/domain/vo"
	"time"
)

// MarchQueueRepository 匹配队列仓储接口
// 用于管理匹配队列，基于 Redis Sorted Set 实现，支持按段位划分队列，每个段位独立队列
type MarchQueueRepository interface {
	// JoinQueue 加入匹配队列（按段位）
	JoinQueue(ctx context.Context, userID, connectorTopic string, ranking vo.RankingType, score float64) error

	// RemoveFromQueue 从队列中移除玩家（按段位）
	RemoveFromQueue(ctx context.Context, userID string, ranking vo.RankingType) error

	// PopPlayers 从队列中取出指定数量的玩家（按段位，按分数排序从低到高）
	PopPlayers(ctx context.Context, ranking vo.RankingType, count int) (map[string]string, error)

	// GetQueueSize 获取队列当前大小（按段位）
	GetQueueSize(ctx context.Context, ranking vo.RankingType) (int, error)

	// IsInQueue 检查玩家是否在队列中（按段位）
	IsInQueue(ctx context.Context, userID string, ranking vo.RankingType) (bool, error)

	// GetPlayerScore 获取玩家在队列中的分数（按段位）
	GetPlayerScore(ctx context.Context, userID string, ranking vo.RankingType) (float64, error)

	// UpdatePlayerScore 更新玩家在队列中的分数（用于动态调整匹配优先级）
	UpdatePlayerScore(ctx context.Context, userID string, ranking vo.RankingType, score float64) error

	// RemoveExpiredPlayers 移除过期的玩家（等待时间超过指定时间，按段位）
	RemoveExpiredPlayers(ctx context.Context, ranking vo.RankingType, maxWaitTime time.Duration) ([]string, error)
}
