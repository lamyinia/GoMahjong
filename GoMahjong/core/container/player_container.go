package container

import (
	"common/database"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
)

// PlayerContainer player 服务专用容器
// 继承 BaseContainer 的数据库连接，添加 player 服务特定的依赖
type PlayerContainer struct {
	*BaseContainer
	userRepository repository.UserRepository
}

// NewPlayerContainer 创建 player 服务容器
func NewPlayerContainer() *PlayerContainer {
	base := NewBase()
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	// 创建 player 服务需要的仓储
	userRepo := persistence.NewMongoUserRepository(base.mongo)

	return &PlayerContainer{
		BaseContainer:  base,
		userRepository: userRepo,
	}
}

// GetUserRepository 获取用户仓储
func (c *PlayerContainer) GetUserRepository() repository.UserRepository {
	return c.userRepository
}

// GetRedis 获取 Redis 管理器（从基础容器继承）
func (c *PlayerContainer) GetRedis() *database.RedisManager {
	return c.BaseContainer.GetRedis()
}

// GetMongo 获取 Mongo 管理器（从基础容器继承）
func (c *PlayerContainer) GetMongo() *database.MongoManager {
	return c.BaseContainer.GetMongo()
}

// Close 关闭容器资源
func (c *PlayerContainer) Close() error {
	return c.BaseContainer.Close()
}
