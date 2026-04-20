package container

import (
	"fmt"
	"march/domain/repository"
	"march/infrastructure/config"
	"march/infrastructure/database"
	"march/infrastructure/discovery"
	"march/infrastructure/log"
	"march/infrastructure/persistence"
	"march/infrastructure/realtime"
	"march/runtime"
	"march/runtime/application/service"
	"march/runtime/application/service/impl"
	"sync"
)

type MarchContainer struct {
	mongo *database.MongoManager
	redis *database.RedisManager
	repository.UserRepository
	repository.MarchQueueRepository
	repository.UserRouterRepository
	MarchWorker  *runtime.Worker
	MatchService service.MatchService
	NodeID       string
	nodeSelector *discovery.NodeSelector
	closed       bool
	mu           sync.Mutex
}

func NewContainer() *MarchContainer {
	base := NewBase(config.MarchNodeConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}
	userRepository := persistence.NewUserRepository(base.mongo, base.redis)
	queueRepository := realtime.NewRedisMarchQueueRepository(base.redis)
	routerRepository := realtime.NewRedisUserRouterRepository(base.redis)
	nodeSelector, err := discovery.NewNodeSelector(discovery.LeastLoad, config.MarchNodeConfig.EtcdConf)
	if err != nil {
		log.Fatal("NodeSelector 创建错误err:%#v", err)
	}
	matchService := impl.NewMatchService(queueRepository, userRepository)
	worker := runtime.NewWorker(matchService, config.MarchNodeConfig.ID)
	if err := worker.InitMatchPools(queueRepository, routerRepository, nodeSelector); err != nil {
		log.Fatal("初始化匹配池失败: %v", err)
		return nil
	}

	return &MarchContainer{
		mongo:                base.mongo,
		redis:                base.redis,
		UserRepository:       userRepository,
		MarchQueueRepository: queueRepository,
		UserRouterRepository: routerRepository,
		MarchWorker:          worker,
		MatchService:         matchService,
		NodeID:               config.MarchNodeConfig.ID,
		nodeSelector:         nodeSelector,
	}
}

func (c *MarchContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	var errs []error

	if c.MarchWorker != nil {
		if err := c.MarchWorker.Close(); err != nil {
			log.Error("MarchWorker 关闭失败: %v", err)
			errs = append(errs, err)
		}
	}
	if c.nodeSelector != nil {
		if err := c.nodeSelector.Close(); err != nil {
			log.Error("NodeSelector 关闭失败: %v", err)
			errs = append(errs, err)
		}
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

	log.Info("MarchContainer 已关闭")
	return nil
}

type baseContainer struct {
	mongo *database.MongoManager
	redis *database.RedisManager
}

func NewBase(conf config.DatabaseConf) *baseContainer {
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
