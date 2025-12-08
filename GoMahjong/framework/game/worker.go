package game

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"encoding/json"
	"fmt"
	svc "framework/game/application/service"
	"framework/node"
	"framework/protocol"
	"framework/stream"
	"time"
)

/*
	1.上报 etcd, march 节点知晓本地的玩家数和性能分析
	2.监听来自 nats 的消息，处理逻辑
		(1)设计房间管理对象
		(2)设计玩家到游戏房间对象的路由，收到局内对战消息，导航到正确的游戏房间
		(3)开始游戏前，收到 march 发送的通知(知道哪些玩家开始游戏)，设计路由，创建游戏房间
		(4)收到断线重连通知，给请求短线重连的玩家，发送数据快照
	3.游戏结束时，更新排名积分，持久化对战记录，删除房间实例，推送游戏结束通知
*/

// GameMessage connector 发送的游戏消息
type GameMessage struct {
	UserID string          `json:"userID"` // 用户 ID（用于查找房间）
	Action string          `json:"action"` // 游戏动作（如：playTile, chi, peng, gang, hu）
	Data   json.RawMessage `json:"data"`   // 动作数据
}

// ReconnectMessage 断线重连消息
type ReconnectMessage struct {
	UserID string `json:"userID"`
}

type Worker struct {
	RoomManager  *RoomManager
	MiddleWorker *node.NatsWorker
	Monitor      *Monitor
	Registry     *discovery.Registry
	GameService  svc.GameService // 游戏服务
	NodeID       string          // 当前 game 节点 ID（用于 NATS topic）
}

// NewWorker 创建 Worker
// NodeID: 当前 game 节点 ID（用于 NATS topic 和 etcd 注册）
func NewWorker(nodeID string) *Worker {
	roomManager := NewRoomManager()
	registry := discovery.NewRegistry()
	monitor := NewMonitor(roomManager, registry, 5*time.Second) // 5秒更新一次负载

	worker := &Worker{
		RoomManager:  roomManager,
		MiddleWorker: node.NewNatsWorker(),
		Monitor:      monitor,
		Registry:     registry,
		NodeID:       nodeID,
	}

	return worker
}

// SetGameService 设置 GameService（由容器注入）
func (w *Worker) SetGameService(gameService svc.GameService) {
	w.GameService = gameService
}

// Start 启动 Worker
// natsURL: NATS 服务地址，如 "nats://localhost:4222"
// etcdConf: etcd 配置
func (w *Worker) Start(ctx context.Context, natsURL string, etcdConf config.EtcdConf) error {
	// 1. 注册到 etcd（传入 NodeID 作为 NodeID，用于 NATS 通信）
	err := w.Registry.Register(etcdConf, w.NodeID)
	if err != nil {
		return fmt.Errorf("注册到 etcd 失败: %v", err)
	}
	log.Info(fmt.Sprintf("Game Worker[%s] 注册到 etcd 成功", w.NodeID))

	// 2. 注册消息处理器
	w.registerHandlers()

	// 3. 启动 NATS 监听
	err = w.MiddleWorker.Run(natsURL, w.NodeID)
	if err != nil {
		return fmt.Errorf("启动 NATS 监听失败: %v", err)
	}
	log.Info(fmt.Sprintf("Game Worker[%s] 启动 NATS 监听成功, topic: %s", w.NodeID, w.NodeID))

	// 4. 启动 Monitor 负载上报
	go w.Monitor.Report(ctx)

	log.Info(fmt.Sprintf("Game Worker[%s] 启动成功", w.NodeID))
	return nil
}

// registerHandlers 注册消息处理器
func (w *Worker) registerHandlers() {
	handlers := make(node.SubscriberHandler)

	// 处理来自 connector 的游戏消息
	handlers["game.play"] = w.handleGameMessage

	// 处理断线重连消息
	handlers["game.reconnect"] = w.handleReconnect

	w.MiddleWorker.RegisterHandlers(handlers)
	log.Info("Game Worker 注册消息处理器完成")
}

// handleGameMessage 处理游戏消息（来自 connector）
func (w *Worker) handleGameMessage(data []byte) interface{} {
	var msg GameMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Error(fmt.Sprintf("Game Worker 解析游戏消息失败: %v", err))
		return nil
	}

	if msg.UserID == "" {
		log.Error("Game Worker 游戏消息缺少 UserID")
		return map[string]interface{}{
			"success": false,
			"error":   "缺少 UserID",
		}
	}

	// 查找玩家所在房间
	room, exists := w.RoomManager.GetPlayerRoom(msg.UserID)
	if !exists {
		log.Error(fmt.Sprintf("Game Worker 玩家 %s 不在任何房间中", msg.UserID))
		return map[string]interface{}{
			"success": false,
			"error":   "玩家不在任何房间中",
		}
	}

	log.Info(fmt.Sprintf("Game Worker 收到游戏消息: userID=%s, roomID=%s, action=%s", msg.UserID, room.ID, msg.Action))

	// TODO: 根据 action 处理游戏逻辑（出牌、吃、碰、杠、胡等）
	// 这里先返回成功，后续在 Engine 中实现具体逻辑
	// 示例：room.Engine.HandleAction(msg.Action, msg.Data)

	return map[string]interface{}{
		"success": true,
		"action":  msg.Action,
		"roomID":  room.ID,
	}
}

// handleReconnect 处理断线重连消息
func (w *Worker) handleReconnect(data []byte) interface{} {
	var msg ReconnectMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Error(fmt.Sprintf("Game Worker 解析重连消息失败: %v", err))
		return map[string]interface{}{
			"success": false,
			"error":   "消息格式错误",
		}
	}

	// 查找玩家所在房间
	room, exists := w.RoomManager.GetPlayerRoom(msg.UserID)
	if !exists {
		return map[string]interface{}{
			"success": false,
			"error":   "玩家不在任何房间中",
		}
	}

	// 获取游戏快照
	snapshot := room.GetPlayerSnapshot(msg.UserID)
	if snapshot == nil {
		// 如果没有玩家快照，返回房间快照
		snapshot = room.GetSnapshot()
	}

	log.Info(fmt.Sprintf("Game Worker 处理重连请求: userID=%s, roomID=%s", msg.UserID, room.ID))

	return map[string]interface{}{
		"success":  true,
		"roomID":   room.ID,
		"snapshot": snapshot,
	}
}

// PushMessage 主动推送消息给指定玩家
func (w *Worker) PushMessage(userID, route string, data []byte) error {
	// 获取玩家的 connector topic
	dest, exists := w.RoomManager.GetPlayerConnector(userID)
	if !exists {
		return fmt.Errorf("玩家 %s 不在任何房间中或 connector topic 不存在", userID)
	}

	// 构建 ServicePacket
	packet := &stream.ServicePacket{
		Source:      w.NodeID,
		Destination: dest,
		Route:       route,
		PushUser:    []string{userID},
		Body: &protocol.Message{
			Type:  protocol.Push,
			Route: route,
			Data:  data,
		},
	}

	// 通过 NatsWorker 推送消息
	err := w.MiddleWorker.PushMessage(packet)
	if err != nil {
		return fmt.Errorf("推送消息失败: %v", err)
	}

	log.Info(fmt.Sprintf("Game Worker 推送消息给玩家 %s, route: %s, connector: %s", userID, route, dest))
	return nil
}

// PushMessageToRoom 推送消息给房间内的所有玩家
// roomID: 房间 ID
// route: 消息路由
// data: 消息数据
func (w *Worker) PushMessageToRoom(roomID, route string, data []byte) error {
	room, exists := w.RoomManager.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("房间 %s 不存在", roomID)
	}

	players := room.GetAllPlayers()
	for _, player := range players {
		if err := w.PushMessage(player.UserID, route, data); err != nil {
			log.Error(fmt.Sprintf("Game Worker 推送消息给玩家 %s 失败: %v", player.UserID, err))
			// 继续推送其他玩家，不中断
		}
	}

	return nil
}

// Close 关闭 Worker
func (w *Worker) Close() {
	if w.Monitor != nil {
		w.Monitor.Stop()
	}
	if w.Registry != nil {
		w.Registry.Close()
	}
	if w.MiddleWorker != nil {
		w.MiddleWorker.Close()
	}
	log.Info(fmt.Sprintf("Game Worker[%s] 已关闭", w.NodeID))
}
