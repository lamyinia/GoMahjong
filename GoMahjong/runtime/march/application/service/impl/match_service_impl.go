package impl

import (
	"common/log"
	"context"
	"core/domain/repository"
	"core/domain/vo"
	"errors"
	"fmt"
	"runtime/march/application/service"
	"strings"
	"time"
)

const (
	rank4Prefix = "classic:rank4" // 排位4人模式前缀
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
	// 1. 检查是否已在队列中
	inQueue, existPool, err := s.queueRepo.IsInQueue(ctx, userID)
	if err != nil {
		return fmt.Errorf("检查队列状态失败:  %s", err)
	}
	if inQueue {
		return errors.Join(repository.ErrPlayerAlreadyInQueue, fmt.Errorf("已在匹配队列: %s", existPool))
	}

	// 2. 如果是排位模式，需要根据用户 ranking 转换成具体的段位池
	finalPoolID, err := s.resolvePoolID(ctx, poolID, userID)
	if err != nil {
		return fmt.Errorf("解析匹配池ID失败: %w", err)
	}

	// 3. 使用时间戳作为 score（先来先服务）
	score := float64(time.Now().Unix())

	if err := s.queueRepo.JoinQueue(ctx, finalPoolID, userID, score); err != nil {
		return fmt.Errorf("加入队列失败: %w", err)
	}

	log.Info("玩家 %s 加入匹配池 %s (原始: %s)", userID, finalPoolID, poolID)
	return nil
}

// resolvePoolID 解析匹配池ID
// 如果是排位模式（classic:rank4），根据用户 ranking 转换成具体的段位池
// 否则直接返回原始 poolID
func (s *MatchServiceImpl) resolvePoolID(ctx context.Context, poolID, userID string) (string, error) {
	// 如果不是排位模式，直接返回
	if !strings.HasPrefix(poolID, rank4Prefix) {
		return poolID, nil
	}

	// 排位模式：需要查询用户 ranking 并转换成具体段位池
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("查询用户信息失败: %w", err)
	}

	// 根据 ranking 分数获取段位类型
	rankingType := vo.GetRankingByScore(user.Ranking)
	rankingString := rankingType.String()

	// 构建具体的段位池ID：classic:rank4:novice, classic:rank4:guard 等
	finalPoolID := fmt.Sprintf("%s:%s", rank4Prefix, rankingString)

	log.Info("玩家 %s ranking=%d, 段位=%s, 匹配池=%s", userID, user.Ranking, rankingString, finalPoolID)
	return finalPoolID, nil
}

func (s *MatchServiceImpl) LeaveQueue(ctx context.Context, userID string) error {
	// 从队列中移除玩家
	if err := s.queueRepo.RemoveFromQueue(ctx, userID); err != nil {
		return fmt.Errorf("离开队列失败: %w", err)
	}

	log.Info("玩家 %s 离开匹配队列", userID)
	return nil
}
