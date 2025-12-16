package march

import (
	"common/log"
	"context"
	"fmt"
	"framework/march/application/service"
	"sync"
	"time"

	"core/domain/vo"
	"framework/node"
	pb "game/pb"
)

/*
	进入大厅时，需要记录 user 的上下线状态
	游戏进行时，需要记录 user 和 game 节点的映射，为重连机制做保障
*/

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
	matchService  service.MatchService
	natsWorker    *node.NatsWorker
	gameConnPool  *GameConnPool                    // gRPC 连接池
	NodeID        string                           // 当前 march 节点 ID（用于 NATS topic）
	stopChan      chan struct{}                    // 停止信号
	matchTriggers map[vo.RankingType]chan struct{} // 段位 -> 匹配触发 channel
	wg            sync.WaitGroup                   // 等待所有 goroutine 结束
}

func NewWorker(matchService service.MatchService, nodeID string) *Worker {
	return &Worker{
		matchService: matchService,
		natsWorker:   node.NewNatsWorker(),
		gameConnPool: NewGameConnPool(),
		NodeID:       nodeID,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动 nats 服务
func (w *Worker) Start(ctx context.Context, natsURL string) error {
	err := w.natsWorker.Run(natsURL, w.NodeID)
	if err != nil {
		return fmt.Errorf("启动 NATS 监听失败: %v", err)
	}
	log.Info(fmt.Sprintf("March Worker[%s] 启动 NATS 监听成功, topic: %s", w.NodeID, w.NodeID))

	w.matchTriggers = make(map[vo.RankingType]chan struct{})
	rankings := vo.GetAllRankings()
	for _, ranking := range rankings {
		w.matchTriggers[ranking] = make(chan struct{})
	}

	w.matchService.SetMatchTriggers(w.matchTriggers)

	for _, ranking := range rankings {
		w.wg.Add(1)
		go w.matchLoopByRanking(ctx, ranking)
	}

	go w.cleanupExpiredPlayers(ctx)

	log.Info(fmt.Sprintf("March Worker[%s] 启动成功", w.NodeID))
	return nil
}

// matchLoopByRanking 按段位的匹配循环（事件驱动 + 定时兜底）
func (w *Worker) matchLoopByRanking(ctx context.Context, ranking vo.RankingType) {
	defer w.wg.Done()

	ticker := time.NewTicker(matchInterval)
	defer ticker.Stop()

	matchTrigger := w.matchTriggers[ranking]
	displayName := ranking.GetDisplayName()

	log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 匹配循环启动，间隔: %v", w.NodeID, displayName, matchInterval))

	for {
		select {
		case <-matchTrigger:
			w.doMatch(ctx, ranking, displayName)
		case <-ticker.C:
			w.doMatch(ctx, ranking, displayName)
		case <-w.stopChan:
			log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 匹配循环停止", w.NodeID, displayName))
			return
		case <-ctx.Done():
			log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 收到上下文取消信号，停止匹配循环", w.NodeID, displayName))
			return
		}
	}
}

// doMatch 对某一种匹配队列，执行一次匹配
func (w *Worker) doMatch(ctx context.Context, rankingType vo.RankingType, displayName string) {
	result, err := w.matchService.MatchByRanking(ctx, rankingType)
	if err != nil {
		log.Error(fmt.Sprintf("March Worker 段位 %s 匹配失败: %v", displayName, err))
		return
	}

	// 匹配成功，处理结果
	if result != nil {
		if err := w.handleMatchSuccess(ctx, result); err != nil {
			log.Error(fmt.Sprintf("March Worker 段位 %s 处理匹配成功失败: %v", displayName, err))
			// 注意：这里不 return，因为匹配已经成功，只是推送失败，可以考虑重试机制或回滚匹配
		}
	}
}

// handleMatchSuccess 处理匹配成功
// 通过 gRPC 调用 Game 节点创建房间
func (w *Worker) handleMatchSuccess(ctx context.Context, result *service.MatchResult) error {
	err := w.callGameCreateRoom(ctx, result.GameNodeAddr, result.Players)
	if err != nil {
		// 这里可以考虑重新放进匹配队列，或者通知匹配异常
		return fmt.Errorf("调用 Game 创建房间失败: %w", err)
	}

	log.Info(fmt.Sprintf("March Worker 匹配成功处理完成: gameNode=%s, players=%d", result.GameNodeAddr, len(result.Players)))
	return nil
}

// callGameCreateRoom 通过 gRPC 调用 Game 节点创建房间
func (w *Worker) callGameCreateRoom(ctx context.Context, gameNodeAddr string, players map[string]string) error {
	// 使用立直麻将 4 人引擎（暂时硬编码，后续可配置）
	const RIICHI_MAHJONG_4P_ENGINE = int32(0)

	// 获取 Game 节点的 gRPC 客户端
	client, err := w.gameConnPool.GetClient(gameNodeAddr)
	if err != nil {
		return fmt.Errorf("获取 Game 客户端失败: %v", err)
	}

	// 构建 gRPC 请求
	req := &pb.CreateRoomRequest{
		Players:    players,
		EngineType: RIICHI_MAHJONG_4P_ENGINE,
	}

	// 调用 Game 的 CreateRoom RPC（超时 5 秒）
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.CreateRoom(callCtx, req)
	if err != nil {
		return fmt.Errorf("调用 Game.CreateRoom RPC 失败: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("Game 创建房间失败: %s", resp.Message)
	}

	log.Info(fmt.Sprintf("March Worker 通过 gRPC 创建房间成功: gameNodeAddr=%s, roomID=%s, players=%d", gameNodeAddr, resp.RoomID, len(players)))
	return nil
}

// cleanupExpiredPlayers 清理过期玩家（每5分钟执行一次）
func (w *Worker) cleanupExpiredPlayers(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Info(fmt.Sprintf("March Worker[%s] 过期玩家清理启动，间隔: 5分钟", w.NodeID))

	for {
		select {
		case <-ticker.C:
			// TODO: 调用 MatchService 清理过期玩家
			// 这里需要 MatchService 提供清理方法，或者直接通过 Repository 清理
			log.Debug("March Worker 执行过期玩家清理")
		case <-w.stopChan:
			log.Info(fmt.Sprintf("March Worker[%s] 过期玩家清理停止", w.NodeID))
			return
		case <-ctx.Done():
			log.Info(fmt.Sprintf("March Worker[%s] 收到上下文取消信号，停止过期玩家清理", w.NodeID))
			return
		}
	}
}

// Close 关闭 Worker
func (w *Worker) Close() error {
	// 1. 关闭停止信号 channel
	close(w.stopChan)

	// 2. 关闭所有匹配触发 channel
	for ranking, trigger := range w.matchTriggers {
		close(trigger)
		log.Debug(fmt.Sprintf("March Worker[%s] 关闭段位 %s 的匹配触发 channel", w.NodeID, ranking.GetDisplayName()))
	}

	// 3. 等待所有匹配循环 goroutine 结束
	w.wg.Wait()

	// 4. 关闭 gRPC 连接池
	if w.gameConnPool != nil {
		if err := w.gameConnPool.Close(); err != nil {
			log.Error(fmt.Sprintf("March Worker[%s] 关闭 gRPC 连接池失败: %v", w.NodeID, err))
		}
	}

	// 5. 关闭 NATS Worker
	if w.natsWorker != nil {
		w.natsWorker.Close()
	}

	log.Info(fmt.Sprintf("March Worker[%s] 已关闭", w.NodeID))
	return nil
}
