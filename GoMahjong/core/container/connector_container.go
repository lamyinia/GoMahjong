package container

import (
	"common/config"
	"common/log"
	"common/utils"
	"framework/conn"
	"framework/node"
)

type ConnectorContainer struct {
	*BaseContainer
	worker          *conn.Worker
	natsWorker      *node.NatsWorker
	connectorConfig interface{} // 使用 interface{} 避免循环导入
	rateLimiter     *utils.RateLimiter
}

func NewConnectorContainer() *ConnectorContainer {
	base := NewBase()
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	return &ConnectorContainer{
		BaseContainer: base,
	}
}

// GetConnectorConfig 获取 Connector 配置
func (c *ConnectorContainer) GetConnectorConfig() interface{} {
	if c.connectorConfig == nil {
		cfg, err := config.InjectedConfig.GetConnectorConfig()
		if err != nil || cfg == nil {
			log.Fatal("获取 connector 配置文件失败: %v", err)
			return nil
		}
		c.connectorConfig = cfg
	}
	return c.connectorConfig
}

// GetNatsWorker 获取或创建 NATS Worker
func (c *ConnectorContainer) GetNatsWorker() *node.NatsWorker {
	if c.natsWorker == nil {
		c.natsWorker = node.NewNatsWorker()
		if c.natsWorker == nil {
			log.Fatal("NATS Worker 初始化失败")
		}
	}
	return c.natsWorker
}

// GetRateLimiter 获取或创建速率限制器
func (c *ConnectorContainer) GetRateLimiter() *utils.RateLimiter {
	if c.rateLimiter == nil {
		// 100 个连接/秒的限制
		c.rateLimiter = utils.NewRateLimiter(100, 1)
	}
	return c.rateLimiter
}

// GetWorker 获取或创建 Worker
func (c *ConnectorContainer) GetWorker() *conn.Worker {
	if c.worker == nil {
		c.worker = conn.NewWorkerWithDeps(
			c.GetConnectorConfig(),
			c.GetNatsWorker(),
			c.GetRateLimiter(),
		)
		if c.worker == nil {
			log.Fatal("Worker 初始化失败")
		}
	}
	return c.worker
}

// Close 关闭所有资源
func (c *ConnectorContainer) Close() error {
	if c.worker != nil {
		c.worker.Close()
	}
	if c.natsWorker != nil {
		c.natsWorker.Close()
	}
	return c.BaseContainer.Close()
}
