package impl

import (
	"context"
	"errors"
	"fmt"
	"march/domain/repository"
	"march/domain/vo"
	"march/infrastructure/log"
	"march/infrastructure/message/transfer"
	"march/runtime/application/service"
	"strings"
	"time"
)

const (
	rank4Prefix = "classic:rank4"
)

type MatchServiceImpl struct {
	queueRepo repository.MarchQueueRepository
	userRepo  repository.UserRepository
}

func NewMatchService(queueRepo repository.MarchQueueRepository, userRepo repository.UserRepository) service.MatchService {
	return &MatchServiceImpl{
		queueRepo: queueRepo,
		userRepo:  userRepo,
	}
}

func (s *MatchServiceImpl) JoinQueue(ctx context.Context, poolID, userID string) error {
	inQueue, existPool, err := s.queueRepo.IsInQueue(ctx, userID)
	if err != nil {
		return fmt.Errorf("检查队列状态失败:  %s", err)
	}
	if inQueue {
		return errors.Join(transfer.ErrPlayerAlreadyInQueue, fmt.Errorf("已在匹配队列: %s", existPool))
	}

	finalPoolID, err := s.resolvePoolID(ctx, poolID, userID)
	if err != nil {
		return fmt.Errorf("解析匹配池ID失败: %w", err)
	}

	score := float64(time.Now().Unix())

	if err := s.queueRepo.JoinQueue(ctx, finalPoolID, userID, score); err != nil {
		return fmt.Errorf("加入队列失败: %w", err)
	}

	log.Info("玩家 %s 加入匹配池 %s (原始: %s)", userID, finalPoolID, poolID)
	return nil
}

func (s *MatchServiceImpl) resolvePoolID(ctx context.Context, poolID, userID string) (string, error) {
	if !strings.HasPrefix(poolID, rank4Prefix) {
		return poolID, nil
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("查询用户信息失败: %w", err)
	}

	rankingType := vo.GetRankingByScore(user.Ranking)
	rankingString := rankingType.String()

	finalPoolID := fmt.Sprintf("%s:%s", rank4Prefix, rankingString)

	log.Info("玩家 %s ranking=%d, 段位=%s, 匹配池=%s", userID, user.Ranking, rankingString, finalPoolID)
	return finalPoolID, nil
}

func (s *MatchServiceImpl) LeaveQueue(ctx context.Context, userID string) error {
	if err := s.queueRepo.RemoveFromQueue(ctx, userID); err != nil {
		return fmt.Errorf("离开队列失败: %w", err)
	}

	log.Info("玩家 %s 离开匹配队列", userID)
	return nil
}
