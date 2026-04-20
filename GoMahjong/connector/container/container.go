package container

import (
	"connector/domain/repository"
	"connector/infrastructure/cache"
	"connector/infrastructure/config"
	"connector/infrastructure/database"
	"connector/infrastructure/log"
	"connector/infrastructure/message/node"
	"connector/infrastructure/ratelimiter"
	"connector/infrastructure/realtime"
	"connector/runtime"
	"fmt"
	"sync"
)

type ConnectorContainer struct {
	mongo      *database.MongoManager
	redis      *database.RedisManager
	worker     *conn.Worker
	natsWorker *node.NatsWorker
	closed     bool
	mu         sync.Mutex
}

func NewContainer() *ConnectorContainer {
	base := newBase(config.ConnectorConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	return &ConnectorContainer{
		mongo: base.mongo,
		redis: base.redis,
	}
}

func (c *ConnectorContainer) GetNatsWorker() *node.NatsWorker {
	if c.natsWorker == nil {
		c.natsWorker = node.NewNatsWorker()
	}
	return c.natsWorker
}

func (c *ConnectorContainer) GetWorker() *conn.Worker {
	if c.worker == nil {
		var opts []conn.WorkerOption
		userRepository := realtime.NewRedisUserRouterRepository(c.redis)

		opts = append(opts, withNatsWorker())
		opts = append(opts, withRateLimiter(100, 1))
		opts = append(opts, withGameRouteCache())
		opts = append(opts, withUserRoute(userRepository))

		c.worker = conn.NewWorkerWithDeps(opts...)
		if c.worker == nil {
			log.Fatal("Worker 初始化失败")
		}
	}
	return c.worker
}

func withNatsWorker() conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.MiddleWorker = node.NewNatsWorker()
		return nil
	}
}

func withRateLimiter(rate, burst int) conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.ConnectionRateLimiter = ratelimiter.NewRateLimiter(rate, burst)
		return nil
	}
}

func withGameRouteCache() conn.WorkerOption {
	return func(w *conn.Worker) error {
		gameCache, err := cache.NewGameRouteCache()
		if err != nil {
			return err
		}
		w.GameRouteCache = gameCache
		return nil
	}
}

func withUserRoute(repo repository.UserRouterRepository) conn.WorkerOption {
	return func(w *conn.Worker) error {
		w.UserRouter = repo
		return nil
	}
}

func (c *ConnectorContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	var errs []error

	if c.worker != nil {
		c.worker.Close()
	}
	if c.natsWorker != nil {
		c.natsWorker.Close()
	}
	if c.mongo != nil {
		if err := c.mongo.Close(); err != nil {
			log.Error("mongo 关闭失败: %v", err)
			errs = append(errs, err)
		}
	}
	if c.redis != nil {
		if err := c.redis.Close(); err != nil {
			log.Error("redis 关闭失败: %v", err)
			errs = append(errs, err)
		}
	}

	c.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("关闭资源时发生 %d 个错误", len(errs))
	}

	log.Info("ConnectorContainer 已关闭")
	return nil
}

type baseContainer struct {
	mongo *database.MongoManager
	redis *database.RedisManager
}

func newBase(conf config.DatabaseConf) *baseContainer {
	mongo := database.NewMongo(conf.MongoConf)
	redis := database.NewRedis(conf.RedisConf)

	if mongo == nil || redis == nil {
		log.Fatal("数据库初始化失败")
		return nil
	}

	log.Info("mongodb、redis 数据库服务启动成功")

	return &baseContainer{
		mongo: mongo,
		redis: redis,
	}
}
