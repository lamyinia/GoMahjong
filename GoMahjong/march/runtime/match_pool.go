package runtime

import (
	"context"
	"fmt"
	"march/domain/repository"
	"march/infrastructure/config"
	"march/infrastructure/discovery"
	"march/infrastructure/log"
	"march/runtime/application/service"
	"strings"
	"sync"
	"time"
)

type MatchPool struct {
	poolID          string
	strategy        MatchStrategy
	batchSize       int
	interval        time.Duration
	requiredPlayers int

	queueRepo    repository.MarchQueueRepository
	routerRepo   repository.UserRouterRepository
	nodeSelector *discovery.NodeSelector
	resultChan   chan<- *service.MatchResult

	wg       sync.WaitGroup
	stopChan chan struct{}
}

func NewMatchPool(cfg config.MarchPoolConfig,
	queueRepo repository.MarchQueueRepository,
	routerRepo repository.UserRouterRepository,
	nodeSelector *discovery.NodeSelector,
	resultChan chan<- *service.MatchResult,
) (*MatchPool, error) {
	requiredPlayers := inferRequiredPlayers(string(cfg.PoolID))
	strategy, err := createStrategy(cfg.Strategy)
	if err != nil {
		return nil, err
	}

	return &MatchPool{
		poolID:          string(cfg.PoolID),
		strategy:        strategy,
		batchSize:       cfg.BatchSize,
		interval:        time.Duration(cfg.Internal) * time.Millisecond,
		requiredPlayers: requiredPlayers,
		queueRepo:       queueRepo,
		routerRepo:      routerRepo,
		nodeSelector:    nodeSelector,
		resultChan:      resultChan,
		stopChan:        make(chan struct{}),
	}, nil
}

func (p *MatchPool) Start() {
	go p.matchLoop()
	log.Info("匹配池 [%s] 启动，间隔: %v, 批次大小: %d, 需要玩家数: %d", p.poolID, p.interval, p.batchSize, p.requiredPlayers)
}

func (p *MatchPool) matchLoop() {
	p.wg.Add(1)
	ticker := time.NewTicker(p.interval)

	defer p.wg.Done()
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.doBatchMatch()
		case <-p.stopChan:
			log.Debug("匹配池 [%s] 收到停止信号", p.poolID)
			return
		}
	}
}

func (p *MatchPool) doBatchMatch() {
	for i := 0; i < p.batchSize; i++ {
		result, err := p.tryMatch()
		if err != nil {
			log.Error("匹配池 [%s] 匹配尝试失败: %v", p.poolID, err)
			continue
		} else if result == nil {
			break
		}

		select {
		case p.resultChan <- result:
		case <-p.stopChan:
			log.Warn("匹配池 [%s] 收到停止信号，丢弃匹配结果", p.poolID)
			return
		default:
		}
	}
}

func (p *MatchPool) tryMatch() (*service.MatchResult, error) {
	ctx := context.Background()

	playerIDs, err := p.strategy.Match(ctx, p.queueRepo, p.poolID, p.requiredPlayers)
	if err != nil {
		return nil, err
	}
	if len(playerIDs) < p.requiredPlayers {
		return nil, nil
	}

	players := make(map[string]string, len(playerIDs))
	for _, userID := range playerIDs {
		connectorRoute, err := p.routerRepo.GetConnectorRouter(ctx, userID)
		if err != nil {
			return nil, nil
		}
		if connectorRoute == "" {
			log.Warn("匹配池 [%s] 玩家 %s 的 connectorRoute 为空，跳过", p.poolID, userID)
			return nil, nil
		}
		players[userID] = connectorRoute
	}

	gameNode, err := p.nodeSelector.SelectGameNode(ctx)
	if err != nil {
		return nil, err
	}

	return &service.MatchResult{
		PoolID:       p.poolID,
		Players:      players,
		GameNodeID:   gameNode.NodeID,
		GameNodeAddr: gameNode.Addr,
	}, nil
}

func (p *MatchPool) Stop() {
	close(p.stopChan)
	p.wg.Wait()
	log.Info("匹配池 [%s] 已停止", p.poolID)
}

func inferRequiredPlayers(poolID string) int {
	if contains(poolID, "rank4") || contains(poolID, "casual4") {
		return 4
	}
	if contains(poolID, "casual3") {
		return 3
	}
	return 4
}

func createStrategy(strategy config.MatchStrategy) (MatchStrategy, error) {
	switch strategy {
	case config.ScorePoll:
		return NewPollStrategy(), nil
	default:
		return nil, fmt.Errorf("不支持的匹配策略: %s", strategy)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
