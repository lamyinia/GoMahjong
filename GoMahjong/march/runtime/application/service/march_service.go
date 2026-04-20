package service

import (
	"context"
)

type MatchService interface {
	JoinQueue(ctx context.Context, poolID, userID string) error
	LeaveQueue(ctx context.Context, userID string) error
}

type MatchResult struct {
	PoolID       string
	Players      map[string]string
	GameNodeID   string
	GameNodeAddr string
}
