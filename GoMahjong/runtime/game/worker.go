package game

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/infrastructure/message/node"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/stream"
	"fmt"
	svc "runtime/game/application/service"
	"sync"
	"time"
)

/*
	1.上报 etcd, 让 march 节点知晓本地的玩家数和性能分析
	2.监听来自 nats 的消息，处理逻辑
		(1)设计房间管理对象
		(2)设计玩家到游戏房间对象的路由，收到局内对战消息，导航到正确的游戏房间
		(3)开始游戏前，收到 march 发送的通知(知道哪些玩家开始游戏)，设计路由，创建游戏房间
		(4)收到断线重连通知，给请求短线重连的玩家，发送数据快照
	3.玩家游戏信息推送的消息总线
*/

type Worker struct {
	RoomManager  *RoomManager
	MiddleWorker *node.NatsWorker
	Monitor      *Monitor
	Registry     *discovery.Registry
	GameService  svc.GameService // 游戏服务
	NodeID       string          // 当前 game 节点 ID（用于 NATS topic）

	destroyRoomCh chan string
	destroyMu     sync.Mutex
	destroyClosed bool
}

// NewWorker 创建 Worker
// NodeID: 当前 game 节点 ID（用于 NATS topic 和 etcd 注册）
func NewWorker(nodeID string) *Worker {
	roomManager := NewRoomManager()
	registry := discovery.NewRegistry()
	monitor := NewMonitor(roomManager, registry, 5*time.Second) // 负载上报器

	worker := &Worker{
		RoomManager:   roomManager,
		MiddleWorker:  node.NewNatsWorker(),
		Monitor:       monitor,
		Registry:      registry,
		NodeID:        nodeID,
		destroyRoomCh: make(chan string, 128),
	}

	go worker.destroyRoomLoop()

	return worker
}

func (w *Worker) destroyRoomLoop() {
	for roomID := range w.destroyRoomCh {
		if roomID == "" {
			continue
		}
		err := w.RoomManager.DeleteRoom(roomID)
		if err != nil {
			log.Warn("Worker destroyRoomLoop 删除房间失败: %v", err)
		}
	}
}

func (w *Worker) RequestDestroyRoom(roomID string) {
	if roomID == "" {
		return
	}

	w.destroyMu.Lock()
	if w.destroyClosed {
		w.destroyMu.Unlock()
		return
	}
	ch := w.destroyRoomCh
	w.destroyMu.Unlock()

	select {
	case ch <- roomID:
	default:
		log.Warn("Worker RequestDestroyRoom 队列已满, roomID=%s", roomID)
	}
}

// SetGameService 设置 GameService（由容器注入）
func (w *Worker) SetGameService(gameService svc.GameService) {
	w.GameService = gameService
}

// Start 启动 Worker
// natsURL: NATS 服务地址，如 "nats://localhost:4222"
// etcdConf: etcd 配置
func (w *Worker) Start(ctx context.Context, natsURL string, etcdConf config.EtcdConf) error {
	w.registerHandlers()
	err := w.Registry.Register(etcdConf, w.NodeID)
	if err != nil {
		return fmt.Errorf("注册到 etcd 失败: %v", err)
	}
	log.Info(fmt.Sprintf("Game Worker[%s] 注册到 etcd 成功", w.NodeID))

	err = w.MiddleWorker.Run(natsURL, w.NodeID)
	if err != nil {
		return fmt.Errorf("启动 NATS 监听失败: %v", err)
	}
	log.Info(fmt.Sprintf("Game Worker[%s] 启动 NATS 监听成功, topic: %s", w.NodeID, w.NodeID))

	// 启动 Monitor 负载上报
	go w.Monitor.Report(ctx)

	log.Info(fmt.Sprintf("Game Worker[%s] 启动成功", w.NodeID))
	return nil
}

// registerHandlers 注册消息处理器
func (w *Worker) registerHandlers() {
	handlers := make(node.SubscriberHandler)

	handlers["game.play.droptile"] = w.handleDropTileHandler
	handlers["game.reconnect"] = w.handleReconnect

	w.MiddleWorker.RegisterHandlers(handlers)
	log.Info("Game Worker 注册消息处理器完成")
}

// PushConnector 推送消息给指定的 Connector（由 Engine 使用）
// connectorNodeID: connector 的 topic
// route: 消息路由
// data: 消息数据
func (w *Worker) PushConnector(connectorNodeID, route string, data []byte) error {
	if connectorNodeID == "" {
		return fmt.Errorf("connector topic 不能为空")
	}

	// 构建 ServicePacket
	packet := &stream.ServicePacket{
		Source:      w.NodeID,
		Destination: connectorNodeID,
		Route:       route,
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

	log.Info(fmt.Sprintf("Game Worker 推送消息给 Connector %s, route: %s", connectorNodeID, route))
	return nil
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

// Close 关闭 Worker
func (w *Worker) Close() {
	w.destroyMu.Lock()
	if !w.destroyClosed {
		close(w.destroyRoomCh)
		w.destroyClosed = true
	}
	w.destroyMu.Unlock()

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
