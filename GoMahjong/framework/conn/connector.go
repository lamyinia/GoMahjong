package conn

import (
	"common/log"
)

/*
	连接器职责：
	1. 正确处理玩家长连接的生命周期、读写事件
	2. 调用 rpc 服务，实现游戏断线重连机制
	3. 调用 nats 服务和 game 节点通信，nats 监听来自 game 节点的消息，实现游戏逻辑
	4. 使用 rpc 服务调用 game 节点的方法，同时 nats 监听来自 hall 节点的消息，实现大厅逻辑
	5. 使用 rpc 服务调用 march 节点的方法，同时 nats 监听来自 march 节点的消息，实现匹配逻辑
			(1)用户在类似游戏对战开始(匹配、创建房间...)的逻辑之前，一定要有查路由的逻辑，检查有没有正在进行的游戏

	6. 设计正确的处理器和路由，收到来自 game、hall、march 节点或者 player 的消息后，如果需要转发，正确转发给目标
	7. 设计正确的处理器，收到来自 game、hall、march 节点或者 player 的消息后，如果需要操作本地内存，正确操作本地内存
*/

type Connector struct {
	manager   *Manager
	handlers  LogicHandler
	isRunning bool
}

func NewConnector() *Connector {
	return &Connector{
		handlers: make(LogicHandler),
	}
}

func (connector *Connector) Run(topic string, maxConn int) {
	if !connector.isRunning {
		log.Info("connector 组件正在配置")
		connector.manager = NewManager()
		connector.manager.LocalHandlers = connector.handlers

		// TODO 使用集群配置文件的 url
		// connector.manager.MiddleWorker.RegisterHandlers(nil)
		connector.manager.MiddleWorker.Run("nats://localhost:4222", topic)

		addr := "localhost:8082"
		connector.manager.Run(addr)
	}
}

func (connector *Connector) Close() {
	if connector.isRunning {
		connector.manager.Close()
	}
}
