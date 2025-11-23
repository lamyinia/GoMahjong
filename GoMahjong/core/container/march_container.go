package container

import (
	"common/config"
	"common/discovery"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
	"core/infrastructure/realtime"
	"framework/march"
	"march/application/service"
)

type MarchContainer struct {
	*BaseContainer
	repository.UserRepository
	realtime.RedisMarchQueueRepository
	realtime.RedisUserRouterRepository
}

func NewMarchContainer() *MarchContainer {
	base := NewBase()
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	userRepository := persistence.NewMongoUserRepository(base.mongo)
	queueRepository := realtime.NewRedisMarchQueueRepository(base.redis)
	routerRepository := realtime.NewRedisUserRouterRepository(base.redis)
	nodeSelector, err := march.NewNodeSelector(discovery.LeastLoad)
	if err != nil {
		log.Fatal("NodeSelector 创建错误err:%#v", err)
	}

	matchService := service.NewMatchService(userRepository, queueRepository, routerRepository, nodeSelector)

	// 从 LocalConfig 获取 serverID
	marchConfig, err := config.InjectedConfig.Configs.GetMarchConfig()
	if err != nil {
		log.Fatal("获取 MarchConfig 失败: %v", err)
		return nil
	}
	march.NewWorker(matchService, marchConfig.GetID())

	return &MarchContainer{
		BaseContainer: base,
	}
}
