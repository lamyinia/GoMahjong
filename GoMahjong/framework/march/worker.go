package march

import (
	"common/log"
	"context"
	"encoding/json"
	"fmt"
	"march/application/service"
	"sync"
	"time"

	"core/domain/vo"
	"framework/node"
	"framework/protocal"
	"framework/stream"
)

/*
	进入大厅时，如果需要记录 user 的上下线状态、
	游戏进行时，需要记录 user 和 game 节点的映射，为重连机制做保障
*/

/*
	匹配器职责：
	1.设计启发式的匹配算法实现分配逻辑，如等待时间、玩家水平、段位
	2.实时监控 game 节点的对局数量、玩家数量、性能信息
	3.根据监控的信息，对 game 节点的分配做负载均衡
	4.游戏对局的生命周期，管理玩家和 game 节点的路由
	5.设计接口(匹配请求、房间创建、路由查询、游戏结束)，注册 rpc 服务，以供其它节点调用
	6.异步给玩家的长连接器推送通知(如匹配成功)
*/

const (
	matchInterval = 2 * time.Second  // 匹配间隔：2秒
	maxWaitTime   = 10 * time.Minute // 队列超时：10分钟
)

// CreateRoomMessage 发送给 game 节点的房间创建消息
type CreateRoomMessage struct {
	Players map[string]string `json:"players"` // userID -> connectorTopic
}

// MatchSuccessMessage 发送给玩家的匹配成功消息
// 注意：connector 不需要知道 roomID，因为 game 节点内部会自己维护索引
type MatchSuccessMessage struct {
	GameNodeTopic string   `json:"gameNodeTopic"` // game 节点 TopicID（用于后续通信）
	PlayerIDs     []string `json:"playerIDs"`     // 所有玩家 ID
}

type Worker struct {
	matchService  service.MatchService
	natsWorker    *node.NatsWorker
	serverID      string                           // 当前 march 节点 ID（用于 NATS topic）
	stopChan      chan struct{}                    // 停止信号
	matchTriggers map[vo.RankingType]chan struct{} // 段位 -> 匹配触发 channel
	wg            sync.WaitGroup                   // 等待所有 goroutine 结束
}

func NewWorker(matchService service.MatchService, serverID string) *Worker {
	return &Worker{
		matchService: matchService,
		natsWorker:   node.NewNatsWorker(),
		serverID:     serverID,
		stopChan:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context, natsURL string) error {
	err := w.natsWorker.Run(natsURL, w.serverID)
	if err != nil {
		return fmt.Errorf("启动 NATS 监听失败: %v", err)
	}
	log.Info(fmt.Sprintf("March Worker[%s] 启动 NATS 监听成功, topic: %s", w.serverID, w.serverID))

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

	log.Info(fmt.Sprintf("March Worker[%s] 启动成功", w.serverID))
	return nil
}

// matchLoopByRanking 按段位的匹配循环（事件驱动 + 定时兜底）
func (w *Worker) matchLoopByRanking(ctx context.Context, ranking vo.RankingType) {
	defer w.wg.Done()

	ticker := time.NewTicker(matchInterval)
	defer ticker.Stop()

	matchTrigger := w.matchTriggers[ranking]
	displayName := ranking.GetDisplayName()

	log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 匹配循环启动，间隔: %v", w.serverID, displayName, matchInterval))

	for {
		select {
		case <-matchTrigger:
			w.doMatch(ctx, ranking, displayName)
		case <-ticker.C:
			w.doMatch(ctx, ranking, displayName)
		case <-w.stopChan:
			log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 匹配循环停止", w.serverID, displayName))
			return
		case <-ctx.Done():
			log.Info(fmt.Sprintf("March Worker[%s] 段位 %s 收到上下文取消信号，停止匹配循环", w.serverID, displayName))
			return
		}
	}
}

// doMatch 执行一次匹配（按段位）
func (w *Worker) doMatch(ctx context.Context, ranking vo.RankingType, displayName string) {
	result, err := w.matchService.MatchByRanking(ctx, ranking)
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
	// 匹配失败（队列中玩家不足4人），立即返回，等待下一轮
}

// handleMatchSuccess 处理匹配成功
// 1. 发送房间创建消息到 game 节点
// 2. 推送匹配成功消息给所有玩家
func (w *Worker) handleMatchSuccess(ctx context.Context, result *service.MatchResult) error {
	// 1. 发送房间创建消息到 game 节点
	gameTopic := fmt.Sprintf("game:%s", result.GameNodeID)
	if err := w.sendCreateRoomMessage(ctx, gameTopic, result.Players); err != nil {
		return fmt.Errorf("发送房间创建消息失败: %w", err)
	}

	// 2. 推送匹配成功消息给所有玩家
	matchSuccessMsg := MatchSuccessMessage{
		GameNodeTopic: gameTopic,
		PlayerIDs:     make([]string, 0, len(result.Players)),
	}
	for userID := range result.Players {
		matchSuccessMsg.PlayerIDs = append(matchSuccessMsg.PlayerIDs, userID)
	}

	msgData, err := json.Marshal(matchSuccessMsg)
	if err != nil {
		return fmt.Errorf("序列化匹配成功消息失败: %w", err)
	}

	// 向每个玩家推送匹配成功消息
	for userID, connectorTopic := range result.Players {
		if err := w.pushMatchSuccessMessage(userID, connectorTopic, msgData); err != nil {
			log.Error(fmt.Sprintf("March Worker 推送匹配成功消息给玩家 %s 失败: %v", userID, err))
			// 继续推送其他玩家，不中断
		}
	}

	log.Info(fmt.Sprintf("March Worker 匹配成功处理完成: gameNode=%s, players=%d", result.GameNodeID, len(result.Players)))
	return nil
}

// sendCreateRoomMessage 发送房间创建消息到 game 节点
func (w *Worker) sendCreateRoomMessage(ctx context.Context, gameTopic string, players map[string]string) error {
	createRoomMsg := CreateRoomMessage{
		Players: players,
	}

	msgData, err := json.Marshal(createRoomMsg)
	if err != nil {
		return fmt.Errorf("序列化房间创建消息失败: %w", err)
	}

	packet := &stream.ServicePacket{
		Source:      w.serverID,
		Destination: gameTopic,
		Route:       "game.room.create",
		Body: &protocal.Message{
			Type: protocal.Notify, // 通知类型，不需要响应
			Data: msgData,
		},
	}

	if err := w.natsWorker.PushMessage(packet); err != nil {
		return fmt.Errorf("推送房间创建消息失败: %w", err)
	}

	log.Info(fmt.Sprintf("March Worker 发送房间创建消息到 game 节点: %s, 玩家数: %d", gameTopic, len(players)))
	return nil
}

// pushMatchSuccessMessage 推送匹配成功消息给玩家
func (w *Worker) pushMatchSuccessMessage(userID, connectorTopic string, msgData []byte) error {
	packet := &stream.ServicePacket{
		Source:      w.serverID,
		Destination: connectorTopic,
		Route:       "march.match.success",
		UserID:      userID,
		Body: &protocal.Message{
			Type: protocal.Push, // 推送类型
			Data: msgData,
		},
	}

	if err := w.natsWorker.PushMessage(packet); err != nil {
		return fmt.Errorf("推送匹配成功消息失败: %w", err)
	}

	log.Debug(fmt.Sprintf("March Worker 推送匹配成功消息给玩家: userID=%s, connectorTopic=%s", userID, connectorTopic))
	return nil
}

// cleanupExpiredPlayers 清理过期玩家（每5分钟执行一次）
func (w *Worker) cleanupExpiredPlayers(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Info(fmt.Sprintf("March Worker[%s] 过期玩家清理启动，间隔: 5分钟", w.serverID))

	for {
		select {
		case <-ticker.C:
			// TODO: 调用 MatchService 清理过期玩家
			// 这里需要 MatchService 提供清理方法，或者直接通过 Repository 清理
			log.Debug("March Worker 执行过期玩家清理")
		case <-w.stopChan:
			log.Info(fmt.Sprintf("March Worker[%s] 过期玩家清理停止", w.serverID))
			return
		case <-ctx.Done():
			log.Info(fmt.Sprintf("March Worker[%s] 收到上下文取消信号，停止过期玩家清理", w.serverID))
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
		log.Debug(fmt.Sprintf("March Worker[%s] 关闭段位 %s 的匹配触发 channel", w.serverID, ranking.GetDisplayName()))
	}

	// 3. 等待所有匹配循环 goroutine 结束
	w.wg.Wait()

	// 4. 关闭 NATS Worker
	if w.natsWorker != nil {
		w.natsWorker.Close()
	}

	log.Info(fmt.Sprintf("March Worker[%s] 已关闭", w.serverID))
	return nil
}
