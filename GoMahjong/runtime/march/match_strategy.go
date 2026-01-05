package march

import (
	"context"
	"core/domain/repository"
)

// MatchStrategy 匹配策略接口
// 定义如何从队列中匹配玩家
type MatchStrategy interface {
	// Match 执行一次匹配尝试
	Match(ctx context.Context, queueRepo repository.MarchQueueRepository, poolID string, requiredPlayers int) ([]string, error)
}

// PollStrategy 轮询策略（先来先服务）
// 从队列头部按顺序取出玩家
type PollStrategy struct{}

// NewPollStrategy 创建轮询策略实例
func NewPollStrategy() MatchStrategy {
	return &PollStrategy{}
}

func (s *PollStrategy) Match(ctx context.Context, queueRepo repository.MarchQueueRepository, poolID string, requiredPlayers int) ([]string, error) {
	if requiredPlayers <= 0 {
		return nil, nil
	}
	players, err := queueRepo.PopPlayers(ctx, poolID, requiredPlayers)
	if err != nil {
		return nil, err
	}
	return players, nil
}
