package container

import (
	"common/config"
	"common/discovery"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
	"core/infrastructure/realtime"
	"fmt"
	"runtime/march"
	"runtime/march/application/service"
	"runtime/march/application/service/impl"

	"sync"
)

type MarchContainer struct {
	*BaseContainer
	repository.UserRepository
	repository.MarchQueueRepository
	repository.UserRouterRepository
	MarchWorker  *march.Worker
	MatchService service.MatchService
	NodeID       string
	nodeSelector *discovery.NodeSelector
	closed       bool
	mu           sync.Mutex
}

func NewMarchContainer() *MarchContainer {
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
	worker := march.NewWorker(matchService, config.MarchNodeConfig.ID)
	if err := worker.InitMatchPools(queueRepository, routerRepository, nodeSelector); err != nil {
		log.Fatal("初始化匹配池失败: %v", err)
		return nil
	}

	return &MarchContainer{
		BaseContainer:        base,
		UserRepository:       userRepository,
		MarchQueueRepository: queueRepository,
		UserRouterRepository: routerRepository,
		MarchWorker:          worker,
		MatchService:         matchService,
		NodeID:               config.MarchNodeConfig.ID,
		nodeSelector:         nodeSelector,
	}
}

// Close 关闭容器资源（幂等操作，可以安全地多次调用）
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
	if c.BaseContainer != nil {
		if err := c.BaseContainer.Close(); err != nil {
			log.Error("BaseContainer 关闭失败: %v", err)
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
