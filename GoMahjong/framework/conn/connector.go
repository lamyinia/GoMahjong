package conn

import (
	"common/log"
	"framework/node"
)

/*
	连接器职责：
	1. 正确处理玩家长连接的生命周期、读写事件
	2. 调用 rpc 服务，实现游戏断线重连机制
	3. 调用 nats 服务，监听来自 game 节点的消息，实现游戏逻辑
	4. 调用 rpc 服务，监听来自 hall 节点的消息，实现大厅逻辑
	5. 调用 rpc 服务，监听来自 march 节点的消息，实现匹配逻辑
	6. 正确转发玩家收到的来自 game、hall、march 节点的消息
*/

type Connector struct {
	manager   *Manager
	handlers  node.LogicHandler
	isRunning bool
}

func NewConnector() *Connector {
	return &Connector{
		handlers: make(node.LogicHandler),
	}
}

func (connector *Connector) Run(topic string, maxConn int) {
	if !connector.isRunning {
		log.Info("connector 组件正在配置")
		connector.manager = NewManager()
		connector.manager.ConnectorHandlers = connector.handlers

		connector.manager.MiddleWorker = node.NewNatsClient(topic, connector.manager.MiddleReadChan)
		connector.manager.MiddleWorker.Run("nats://localhost:4222")

		addr := "localhost:8082"
		connector.manager.Run(addr)
	}
}

func (connector *Connector) Close() {
	if connector.isRunning {
		connector.manager.Close()
	}
}
