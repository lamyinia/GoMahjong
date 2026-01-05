package march

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/domain/repository"
	"fmt"
	"runtime/march/application/service"
	"strings"
	"sync"
	"time"
)

// MatchPool 匹配池
// 每个匹配池独立管理自己的匹配逻辑和定时任务
type MatchPool struct {
	poolID          string
	strategy        MatchStrategy
	batchSize       int           // 每次匹配尝试的次数
	interval        time.Duration // 匹配间隔
	requiredPlayers int           // 需要的玩家数量（根据 poolID 推断）

	queueRepo    repository.MarchQueueRepository
	routerRepo   repository.UserRouterRepository
	nodeSelector *discovery.NodeSelector
	resultChan   chan<- *service.MatchResult // 匹配结果 channel（只发送）

	wg       sync.WaitGroup
	stopChan chan struct{}
}

// NewMatchPool 创建匹配池
func NewMatchPool(cfg config.MarchPoolConfig,
	queueRepo repository.MarchQueueRepository,
	routerRepo repository.UserRouterRepository,
	nodeSelector *discovery.NodeSelector,
	resultChan chan<- *service.MatchResult,
) (*MatchPool, error) {
	// 根据 poolID 推断需要的玩家数量
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

// matchLoop 匹配循环（定时触发）
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
			log.Info("匹配池 [%s] 收到停止信号", p.poolID)
			return
		}
	}
}

// doBatchMatch 执行批次匹配
// 尝试 batchSize 次匹配
func (p *MatchPool) doBatchMatch() {
	for i := 0; i < p.batchSize; i++ {
		result, err := p.tryMatch()
		if err != nil {
			log.Error("匹配池 [%s] 匹配尝试失败: %v", p.poolID, err)
			continue
		} else if result == nil {
			// 匹配池已经空了
			break
		}

		// 发送匹配结果到 channel（非阻塞，如果 channel 满了可以优先考虑阻塞）
		select {
		case p.resultChan <- result:
		case <-p.stopChan:
			log.Warn("匹配池 [%s] 收到停止信号，丢弃匹配结果", p.poolID)
			return
			//default:
			//	log.Error("匹配池 [%s] 匹配结果 channel 已满，丢弃匹配结果: %d 个玩家", p.poolID, len(result.Players))
		}
	}
}

// tryMatch 尝试一次匹配
func (p *MatchPool) tryMatch() (*service.MatchResult, error) {
	ctx := context.Background()

	// 使用策略进行匹配
	playerIDs, err := p.strategy.Match(ctx, p.queueRepo, p.poolID, p.requiredPlayers)
	if err != nil {
		return nil, err
	}
	if len(playerIDs) < p.requiredPlayers {
		return nil, nil
	}

	// 获取所有玩家的 connectorTopic
	players := make(map[string]string, len(playerIDs))
	for _, userID := range playerIDs {
		connectorRoute, err := p.routerRepo.GetConnectorRouter(ctx, userID)
		if err != nil {
			return nil, nil
		}
		if connectorRoute == "" {
			log.Warn("匹配池 [%s] 玩家 %s 的 connectorRoute 为空，跳过", p.poolID, userID)
			// 通知匹配失败逻辑
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

// inferRequiredPlayers 根据 poolID 推断需要的玩家数量
func inferRequiredPlayers(poolID string) int {
	if contains(poolID, "rank4") || contains(poolID, "casual4") {
		return 4
	}
	if contains(poolID, "casual3") {
		return 3
	}
	// 默认 4 人
	return 4
}

// createStrategy 根据策略名称创建匹配策略实例
func createStrategy(strategy config.MatchStrategy) (MatchStrategy, error) {
	switch strategy {
	case config.ScorePoll:
		return NewPollStrategy(), nil
	default:
		return nil, fmt.Errorf("不支持的匹配策略: %s", strategy)
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
