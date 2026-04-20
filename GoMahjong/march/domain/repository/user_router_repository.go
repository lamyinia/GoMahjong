package repository

import (
	"context"
	"time"
)

type UserRouterInfo struct {
	GameTopic      string
	ConnectorTopic string
}

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
