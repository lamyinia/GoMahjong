package runtime

import (
	"context"
	"fmt"
	"march/domain/repository"
	"march/infrastructure/config"
	"march/infrastructure/discovery"
	"march/infrastructure/log"
	"march/infrastructure/message/node"
	"march/runtime/application/service"
	"strings"
	"sync"
	"time"

	pb "march/pb"
)

const (
	matchInterval = 60 * time.Second
	maxWaitTime   = 10 * time.Minute
)

type Worker struct {
	NodeID          string
	matchService    service.MatchService
	natsWorker      *node.NatsWorker
	gameConnPool    *GameConnPool
	matchPools      []*MatchPool
	matchResultChan chan *service.MatchResult
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

func NewWorker(matchService service.MatchService, nodeID string) *Worker {
	return &Worker{
		NodeID:          nodeID,
		matchService:    matchService,
		natsWorker:      node.NewNatsWorker(),
		gameConnPool:    NewGameConnPool(),
		matchResultChan: make(chan *service.MatchResult, 1024),
		stopChan:        make(chan struct{}),
	}
}

func (w *Worker) InitMatchPools(queueRepo repository.MarchQueueRepository, routerRepo repository.UserRouterRepository, nodeSelector *discovery.NodeSelector) error {
	if len(config.MarchNodeConfig.MarchPoolConfigs) == 0 {
		log.Warn("配置中没有匹配池配置")
		return nil
	}

	pools := make([]*MatchPool, 0, len(config.MarchNodeConfig.MarchPoolConfigs))
	for _, poolConfig := range config.MarchNodeConfig.MarchPoolConfigs {
		pool, err := NewMatchPool(
			poolConfig,
			queueRepo,
			routerRepo,
			nodeSelector,
			w.matchResultChan,
		)
		if err != nil {
			return fmt.Errorf("创建匹配池 [%s] 失败: %w", poolConfig.PoolID, err)
		}
		pools = append(pools, pool)
		log.Info("匹配池 [%s] 初始化成功", poolConfig.PoolID)
	}

	w.matchPools = pools
	log.Info(fmt.Sprintf("March Worker[%s] 初始化 %d 个匹配池", w.NodeID, len(pools)))
	return nil
}

func (w *Worker) Start(ctx context.Context, natsURL string) error {
	err := w.natsWorker.Run(natsURL, w.NodeID)
	if err != nil {
		return fmt.Errorf("启动 NATS 监听失败: %v", err)
	}
	log.Info(fmt.Sprintf("March Worker[%s] 启动 NATS 监听成功, topic: %s", w.NodeID, w.NodeID))

	go w.processMatchResults(ctx)
	for _, pool := range w.matchPools {
		pool.Start()
	}

	log.Info(fmt.Sprintf("March Worker[%s] 启动成功，已启动 %d 个匹配池", w.NodeID, len(w.matchPools)))
	return nil
}

func (w *Worker) processMatchResults(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	log.Info(fmt.Sprintf("March Worker[%s] 匹配结果处理启动", w.NodeID))

	for {
		select {
		case result := <-w.matchResultChan:
			if result == nil {
				continue
			}
			if err := w.handleMatchSuccess(ctx, result); err != nil {
				log.Error(fmt.Sprintf("March Worker[%s] 处理匹配结果失败: %v", w.NodeID, err))
			}
		case <-w.stopChan:
			log.Info(fmt.Sprintf("March Worker[%s] 匹配结果处理收到停止信号", w.NodeID))
			return
		case <-ctx.Done():
			log.Info(fmt.Sprintf("March Worker[%s] 匹配结果处理收到上下文取消信号", w.NodeID))
			return
		}
	}
}

func (w *Worker) handleMatchSuccess(ctx context.Context, result *service.MatchResult) error {
	if err := w.callGameCreateRoom(ctx, result); err != nil {
		return fmt.Errorf("调用 Game 创建房间失败: %w", err)
	}
	log.Info(fmt.Sprintf("March Worker 匹配成功处理完成: poolID=%s, gameNode=%s, players=%d", result.PoolID, result.GameNodeAddr, len(result.Players)))
	return nil
}

func (w *Worker) callGameCreateRoom(ctx context.Context, result *service.MatchResult) error {
	engineType := inferEngineType(result.PoolID)
	client, err := w.gameConnPool.GetClient(result.GameNodeAddr)
	if err != nil {
		return fmt.Errorf("获取 Game 客户端失败: %v", err)
	}

	req := &pb.CreateRoomRequest{
		Players:    result.Players,
		EngineType: engineType,
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.CreateRoom(callCtx, req)
	if err != nil {
		return fmt.Errorf("调用 Game.CreateRoom RPC 失败: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("game 创建房间失败: %s", resp.Message)
	}

	log.Info(fmt.Sprintf("March Worker 通过 gRPC 创建房间成功: poolID=%s, gameNodeAddr=%s, roomID=%s, players=%d",
		result.PoolID, result.GameNodeAddr, resp.RoomID, len(result.Players)))
	return nil
}

func inferEngineType(poolID string) int32 {
	const RIICHI_MAHJONG_4P_ENGINE = int32(0)
	const RIICHI_MAHJONG_3P_ENGINE = int32(1)

	if strings.Contains(poolID, "casual3") {
		return RIICHI_MAHJONG_3P_ENGINE
	}
	return RIICHI_MAHJONG_4P_ENGINE
}

func (w *Worker) Close() error {
	for _, pool := range w.matchPools {
		pool.Stop()
	}

	close(w.stopChan)
	w.wg.Wait()

	if w.gameConnPool != nil {
		if err := w.gameConnPool.Close(); err != nil {
			log.Error(fmt.Sprintf("March Worker[%s] 关闭 gRPC 连接池失败: %v", w.NodeID, err))
		}
	}

	if w.natsWorker != nil {
		w.natsWorker.Close()
	}

	log.Info(fmt.Sprintf("March Worker[%s] 已关闭", w.NodeID))
	return nil
}
