package repository

import (
	"context"
)

type MarchQueueRepository interface {
	JoinQueue(ctx context.Context, poolID, userID string, score float64) error
	RemoveFromQueue(ctx context.Context, userID string) error
	IsInQueue(ctx context.Context, userID string) (bool, string, error)
	GetUserPool(ctx context.Context, userID string) (string, error)
	PopPlayers(ctx context.Context, poolID string, count int) ([]string, error)
	GetQueueSize(ctx context.Context, poolID string) (int, error)
}
