package container

import (
	"common/config"
	"common/log"
	"common/utils"
	"core/domain/repository"
	"core/infrastructure/cache"
	"core/infrastructure/message/node"
	"core/infrastructure/realtime"
	"runtime/conn"
)

type ConnectorContainer struct {
	*BaseContainer
	worker     *conn.Worker
	natsWorker *node.NatsWorker
}

func NewConnectorContainer() *ConnectorContainer {
	base := NewBase(config.ConnectorConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	return &ConnectorContainer{
		BaseContainer: base,
	}
}

// GetNatsWorker 获取或创建 NATS Worker
func (c *ConnectorContainer) GetNatsWorker() *node.NatsWorker {
	if c.natsWorker == nil {
		c.natsWorker = node.NewNatsWorker()
	}
	return c.natsWorker
}

// GetWorker 获取或创建 Worker
func (c *ConnectorContainer) GetWorker() *conn.Worker {
	if c.worker == nil {
		var opts []conn.WorkerOption
		userRepository := realtime.NewRedisUserRouterRepository(c.redis)

		opts = append(opts, WithNatsWorker())
		opts = append(opts, WithRateLimiter(100, 1))
		opts = append(opts, WithGameRouteCache())
		opts = append(opts, WithUserRoute(userRepository))

		c.worker = conn.NewWorkerWithDeps(opts...)
		if c.worker == nil {
			log.Fatal("Worker 初始化失败")
		}
	}
	return c.worker
}

func WithNatsWorker() conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.MiddleWorker = node.NewNatsWorker()
		return nil
	}
}

func WithRateLimiter(rate, burst int) conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.ConnectionRateLimiter = utils.NewRateLimiter(rate, burst)
		return nil
	}
}

func WithGameRouteCache() conn.WorkerOption {
	return func(w *conn.Worker) error {
		gameCache, err := cache.NewGameRouteCache()
		if err != nil {
			return err
		}
		w.GameRouteCache = gameCache
		return nil
	}
}

func WithUserRoute(repository repository.UserRouterRepository) conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.UserRouter = repository
		return nil
	}
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
