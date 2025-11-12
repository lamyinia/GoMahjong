package infrastructure

import (
	"common/database"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
)

// Container 是依赖注入容器
type Container struct {
	mongo          *database.MongoManager
	redis          *database.RedisManager
	userRepository repository.UserRepository
}

// New 创建容器并初始化所有依赖
func New() *Container {
	mongo := database.NewMongo()
	redis := database.NewRedis()

	if mongo == nil || redis == nil {
		log.Fatal("数据库初始化失败")
		return nil
	}

	log.Info("mongodb、redis 数据库服务启动成功")

	// 创建仓储实现
	userRepo := persistence.NewMongoUserRepository(mongo)

	return &Container{
		mongo:          mongo,
		redis:          redis,
		userRepository: userRepo,
	}
}

// GetUserRepository 获取用户仓储
func (c *Container) GetUserRepository() repository.UserRepository {
	return c.userRepository
}

// GetRedis 获取 Redis 管理器
func (c *Container) GetRedis() *database.RedisManager {
	return c.redis
}

// GetMongo 获取 Mongo 管理器
func (c *Container) GetMongo() *database.MongoManager {
	return c.mongo
}

// Close 关闭所有资源
func (c *Container) Close() {
	e1 := c.mongo.Close()
	e2 := c.redis.Close()
	if e1 != nil {
		log.Error("mongo 关闭失败: %v", e1)
	}
	if e2 != nil {
		log.Error("redis 关闭失败: %v", e2)
	}
}
