package container

import (
	"common/config"
	"common/database"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
)

// UserContainer user 服务专用容器
// 继承 BaseContainer 的数据库连接，添加 user 服务特定的依赖
type UserContainer struct {
	*BaseContainer
	userRepository repository.UserRepository
}

// NewUserContainer 创建 user 服务容器
func NewUserContainer() *UserContainer {
	base := NewBase(config.UserNodeConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	// 创建 user 服务需要的仓储
	userRepo := persistence.NewUserRepository(base.mongo, base.redis)

	return &UserContainer{
		BaseContainer:  base,
		userRepository: userRepo,
	}
}

// GetUserRepository 获取用户仓储
func (c *UserContainer) GetUserRepository() repository.UserRepository {
	return c.userRepository
}

// GetRedis 获取 Redis 管理器（从基础容器继承）
func (c *UserContainer) GetRedis() *database.RedisManager {
	return c.BaseContainer.GetRedis()
}

// GetMongo 获取 Mongo 管理器（从基础容器继承）
func (c *UserContainer) GetMongo() *database.MongoManager {
	return c.BaseContainer.GetMongo()
}

// Close 关闭容器资源
func (c *UserContainer) Close() error {
	return c.BaseContainer.Close()
}
