package repository

import (
	"context"
	"time"
)

type UserRouterRepository interface {
	SaveConnectorRouter(ctx context.Context, userID, connectorID string, ttl time.Duration) error
	GetConnectorRouter(ctx context.Context, userID string) (string, error)
	DeleteConnectorRouter(ctx context.Context, userID string) error
	SaveGameRouter(ctx context.Context, userID, gameNodeID string, ttl time.Duration) error
	GetGameRouter(ctx context.Context, userID string) (string, error)
	DeleteGameRouter(ctx context.Context, userID string) error
	HasGameRouter(ctx context.Context, userID string) (bool, error)
}
