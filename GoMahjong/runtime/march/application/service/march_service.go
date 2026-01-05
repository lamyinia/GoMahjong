package service

import (
	"context"
)

// MatchService 匹配服务接口
type MatchService interface {
	JoinQueue(ctx context.Context, poolID, userID string) error
	LeaveQueue(ctx context.Context, userID string) error
}

// MatchResult 匹配结果
type MatchResult struct {
	PoolID       string            // 匹配池 ID（用于推断引擎类型）
	Players      map[string]string // userID -> connectorTopic
	GameNodeID   string            // game 节点 ID（用于 NATS topic）
	GameNodeAddr string            // game 节点地址
}
