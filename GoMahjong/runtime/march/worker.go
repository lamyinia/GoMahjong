package march

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/domain/repository"
	"core/infrastructure/message/node"
	"fmt"
	"runtime/march/application/service"
	"strings"
	"sync"
	"time"

	pb "game/pb"
)

/*
	匹配器职责：
	1.设计启发式的匹配算法实现分配逻辑，如等待时间、玩家水平、段位
	2.实时监控 game 节点的对局数量、玩家数量、性能信息
	3.根据监控的信息，对 game 节点的分配做负载均衡
	4.管理玩家和 game 节点的路由
	5.设计接口(匹配请求、房间创建、路由查询、游戏结束)，注册 rpc 服务，以供其它节点调用
*/

const (
	matchInterval = 60 * time.Second // 匹配间隔：60秒
	maxWaitTime   = 10 * time.Minute // 队列超时：10分钟
)

type Worker struct {
	NodeID          string
	matchService    service.MatchService
	natsWorker      *node.NatsWorker
	gameConnPool    *GameConnPool // gRPC 连接池
	matchPools      []*MatchPool
	matchResultChan chan *service.MatchResult // 统一的结果 channel
	stopChan        chan struct{}             // 停止信号
	wg              sync.WaitGroup            // 等待所有 goroutine 结束
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

// InitMatchPools 从配置初始化匹配池
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

// Start 启动 nats 服务和匹配结果处理
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

// processMatchResults 统一处理所有匹配结果
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
				// fixme 通知客户端匹配失败
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

// handleMatchSuccess 处理匹配成功
// 通过 gRPC 调用 Game 节点创建房间
func (w *Worker) handleMatchSuccess(ctx context.Context, result *service.MatchResult) error {
	if err := w.callGameCreateRoom(ctx, result); err != nil {
		// fixme 通知匹配异常
		return fmt.Errorf("调用 Game 创建房间失败: %w", err)
	}
	log.Info(fmt.Sprintf("March Worker 匹配成功处理完成: poolID=%s, gameNode=%s, players=%d", result.PoolID, result.GameNodeAddr, len(result.Players)))
	return nil
}

// callGameCreateRoom 通过 gRPC 调用 Game 节点创建房间
func (w *Worker) callGameCreateRoom(ctx context.Context, result *service.MatchResult) error {
	// 根据 poolID 推断引擎类型
	engineType := inferEngineType(result.PoolID)
	// 获取 Game 节点的 gRPC 客户端
	client, err := w.gameConnPool.GetClient(result.GameNodeAddr)
	if err != nil {
		// 考虑推送匹配失败
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

// inferEngineType 根据 poolID 推断引擎类型
func inferEngineType(poolID string) int32 {
	const RIICHI_MAHJONG_4P_ENGINE = int32(0)
	const RIICHI_MAHJONG_3P_ENGINE = int32(1) // 假设 3 人引擎类型为 1

	if strings.Contains(poolID, "casual3") {
		return RIICHI_MAHJONG_3P_ENGINE
	}
	return RIICHI_MAHJONG_4P_ENGINE
}

// Close 关闭 Worker
func (w *Worker) Close() error {
	// 停止所有匹配池
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
