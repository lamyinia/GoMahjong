package runtime

import (
	"context"
	"march/domain/repository"
)

type MatchStrategy interface {
	Match(ctx context.Context, queueRepo repository.MarchQueueRepository, poolID string, requiredPlayers int) ([]string, error)
}

type PollStrategy struct{}

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
