package container

import (
	"common/database"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
)

// GameContainer game 服务专用容器
// 继承 BaseContainer 的数据库连接，添加 game 服务特定的依赖
type GameContainer struct {
	*BaseContainer
	userRepository repository.UserRepository
	// TODO: 添加 game 服务特定的仓储
	// gameRepository repository.GameRepository
	// ruleRepository repository.RuleRepository
}

// NewGameContainer 创建 game 服务容器
func NewGameContainer() *GameContainer {
	base := NewBase()
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	// 创建 game 服务需要的仓储
	userRepo := persistence.NewMongoUserRepository(base.mongo)

	return &GameContainer{
		BaseContainer:  base,
		userRepository: userRepo,
		// TODO: 初始化其他仓储
	}
}

// GetUserRepository 获取用户仓储
func (c *GameContainer) GetUserRepository() repository.UserRepository {
	return c.userRepository
}

// GetRedis 获取 Redis 管理器（从基础容器继承）
func (c *GameContainer) GetRedis() *database.RedisManager {
	return c.BaseContainer.GetRedis()
}

// GetMongo 获取 Mongo 管理器（从基础容器继承）
func (c *GameContainer) GetMongo() *database.MongoManager {
	return c.BaseContainer.GetMongo()
}

// Close 关闭容器资源
func (c *GameContainer) Close() error {
	return c.BaseContainer.Close()
}
