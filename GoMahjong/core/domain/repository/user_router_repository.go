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
	SaveGameRouter(ctx context.Context, userID, gameTopic string, ttl time.Duration) error
	GetGameRouter(ctx context.Context, userID string) (string, error)
	DeleteGameRouter(ctx context.Context, userID string) error
	ExistsGameRouter(ctx context.Context, userID string) (bool, error)

	SaveConnectorRouter(ctx context.Context, userID, connectorTopic string, ttl time.Duration) error
	GetConnectorRouter(ctx context.Context, userID string) (string, error)
	DeleteConnectorRouter(ctx context.Context, userID string) error
	ExistsConnectorRouter(ctx context.Context, userID string) (bool, error)

	DeleteGameRouters(ctx context.Context, userIDs []string) error
	DeleteConnectorRouters(ctx context.Context, userIDs []string) error
}
