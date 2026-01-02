package impl

import (
	"common/discovery"
	"common/log"
	"context"
	"core/domain/repository"
	"core/domain/vo"
	"fmt"
	"runtime/march/application/service"
	"sync"
	"time"
)

const (
	playersPerMatch = 4             // 麻将需要4个玩家
	routerTTL       = 2 * time.Hour // 路由过期时间（游戏结束后清理）
)

type MatchServiceImpl struct {
	userRepo      repository.UserRepository
	queueRepo     repository.MarchQueueRepository
	routerRepo    repository.UserRouterRepository
	nodeSelector  *discovery.NodeSelector
	matchTriggers map[vo.RankingType]chan struct{} // 段位 -> 匹配触发 channel（由 Worker 创建）
	mu            sync.RWMutex                     // 保护 matchTriggers 的并发访问
}

func NewMatchService(
	userRepo repository.UserRepository,
	queueRepo repository.MarchQueueRepository,
	routerRepo repository.UserRouterRepository,
	nodeSelector *discovery.NodeSelector,
) service.MatchService {
	return &MatchServiceImpl{
		userRepo:      userRepo,
		queueRepo:     queueRepo,
		routerRepo:    routerRepo,
		nodeSelector:  nodeSelector,
		matchTriggers: make(map[vo.RankingType]chan struct{}),
	}
}

// SetMatchTriggers 设置匹配触发 channel（由 Worker 调用）
func (s *MatchServiceImpl) SetMatchTriggers(matchTriggers map[vo.RankingType]chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matchTriggers = matchTriggers
	log.Info("MatchService 设置匹配触发 channel，段位数: %d", len(matchTriggers))
}

func (s *MatchServiceImpl) JoinQueue(ctx context.Context, userID, connectorNodeID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	ranking := user.GetRanking()

	inQueue, err := s.queueRepo.IsInQueue(ctx, userID, ranking)
	if err != nil {
		return fmt.Errorf("检查队列状态失败: %w", err)
	}
	if inQueue {
		return repository.ErrPlayerAlreadyInQueue
	}

	score := float64(time.Now().Unix())
	if err := s.queueRepo.JoinQueue(ctx, userID, connectorNodeID, ranking, score); err != nil {
		return fmt.Errorf("加入队列失败: %w", err)
	}

	s.triggerMatch(ranking)

	log.Info(fmt.Sprintf("玩家 %s 加入段位 %s 匹配队列", userID, ranking.GetDisplayName()))
	return nil
}

func (s *MatchServiceImpl) triggerMatch(ranking vo.RankingType) {
	s.mu.RLock()
	trigger, exists := s.matchTriggers[ranking]
	s.mu.RUnlock()

	if !exists {
		return
	}

	select {
	case trigger <- struct{}{}:
		// 成功发送触发信号
	default:
		// channel 已关闭或已满，忽略（避免阻塞）
	}
}

func (s *MatchServiceImpl) LeaveQueue(ctx context.Context, userID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	ranking := user.GetRanking()

	if err := s.queueRepo.RemoveFromQueue(ctx, userID, ranking); err != nil {
		return fmt.Errorf("从队列移除失败: %w", err)
	}

	log.Info(fmt.Sprintf("玩家 %s 离开段位 %s 匹配队列", userID, ranking.GetDisplayName()))
	return nil
}

// MatchByRanking 按段位执行一次匹配
func (s *MatchServiceImpl) MatchByRanking(ctx context.Context, ranking vo.RankingType) (*service.MatchResult, error) {
	players, err := s.queueRepo.PopPlayers(ctx, ranking, playersPerMatch)
	if err != nil {
		return nil, fmt.Errorf("从队列取出玩家失败: %w", err)
	}

	// Lua 脚本已保证，如果队列中玩家不足，不会取出任何玩家，直接返回空，所以这里如果 players 为空，说明队列中玩家不足
	if len(players) == 0 {
		return nil, nil
	}

	// 防御性检查
	if len(players) < playersPerMatch {
		log.Warn(fmt.Sprintf("段位 %s 取出玩家数量异常: 期望 %d 人，实际 %d 人", ranking.GetDisplayName(), playersPerMatch, len(players)))
		return nil, nil
	}

	gameNode, err := s.nodeSelector.SelectGameNode(ctx)
	if err != nil {
		// 选择节点失败，需要将玩家重新放回队列，后续可以实现 Rollback
		return nil, fmt.Errorf("选择 game 节点失败: %w", err)
	}

	nodeID := gameNode.NodeID

	//保存用户路由（用于断线重连）
	for userID, connectorTopic := range players {
		routerInfo := &repository.UserRouterInfo{
			GameTopic:      nodeID,
			ConnectorTopic: connectorTopic,
		}
		if err := s.routerRepo.SaveRouter(ctx, userID, routerInfo, routerTTL); err != nil {
			log.Error(fmt.Sprintf("保存用户路由失败: userID=%s, err=%v", userID, err))
		}
	}

	log.Info(fmt.Sprintf("段位 %s 匹配成功: gameNode=%s, players=%d", ranking.GetDisplayName(), gameNode.Addr, len(players)))

	return &service.MatchResult{
		Players:      players,
		GameNodeID:   nodeID,
		GameNodeAddr: gameNode.Addr,
	}, nil
}

// Match 执行一次匹配（遍历所有段位）
func (s *MatchServiceImpl) Match(ctx context.Context) (*service.MatchResult, error) {
	rankings := vo.GetAllRankings()
	for _, ranking := range rankings {
		result, err := s.MatchByRanking(ctx, ranking)
		if err != nil {
			log.Error(fmt.Sprintf("段位 %s 匹配失败: %v", ranking.GetDisplayName(), err))
			continue
		}
		if result != nil {
			return result, nil
		}
	}

	return nil, nil
}
