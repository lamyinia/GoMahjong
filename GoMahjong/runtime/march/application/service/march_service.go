package service

import (
	"context"
	"core/domain/vo"
)

// MatchService 匹配服务接口
// 负责匹配逻辑的核心业务
type MatchService interface {
	// Match 执行一次匹配（遍历所有段位）
	// 返回：匹配结果（包含玩家列表、game 节点信息），如果匹配失败返回 nil
	Match(ctx context.Context) (*MatchResult, error)

	// MatchByRanking 按段位执行一次匹配
	// ranking: 段位
	// 返回：匹配结果，如果匹配失败返回 nil
	MatchByRanking(ctx context.Context, ranking vo.RankingType) (*MatchResult, error)

	// JoinQueue 加入匹配队列
	// userID: 用户 ID
	// connectorTopic: connector 的 topic
	JoinQueue(ctx context.Context, userID, connectorNodeID string) error

	// LeaveQueue 离开匹配队列
	LeaveQueue(ctx context.Context, userID string) error

	// SetMatchTriggers 设置匹配触发 channel（由 Worker 调用）
	// matchTriggers: 段位 -> channel 的映射，Worker 创建 channel 后传递给 Service
	SetMatchTriggers(matchTriggers map[vo.RankingType]chan struct{})
}

// MatchResult 匹配结果
type MatchResult struct {
	Players      map[string]string // userID -> connectorTopic
	GameNodeID   string            // game 节点 ID（用于 NATS topic）
	GameNodeAddr string            // game 节点地址
}
