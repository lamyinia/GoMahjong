package container

import (
	"auth/domain/repository"
	"auth/infrastructure/config"
	"auth/infrastructure/database"
	"auth/infrastructure/log"
	"auth/infrastructure/persistence"
)

// Container auth 服务容器
// 管理数据库连接和 auth 服务特定的依赖
type Container struct {
	mongo          *database.MongoManager
	redis          *database.RedisManager
	userRepository repository.UserRepository
}

// NewContainer 创建 auth 服务容器
func NewContainer() *Container {
	conf := config.AuthNodeConfig.DatabaseConf

	mongo := database.NewMongo(conf.MongoConf)
	redis := database.NewRedis(conf.RedisConf)

	if mongo == nil || redis == nil {
		log.Fatal("数据库初始化失败")
		return nil
	}

	log.Info("mongodb、redis 数据库服务启动成功")

	// 创建 auth 服务需要的仓储
	userRepo := persistence.NewUserRepository(mongo, redis)

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

// Close 关闭容器资源
func (c *Container) Close() error {
	e1 := c.mongo.Close()
	e2 := c.redis.Close()
	if e1 != nil {
		log.Error("mongo 关闭失败: %v", e1)
	}
	if e2 != nil {
		log.Error("redis 关闭失败: %v", e2)
	}
	if e1 != nil {
		return e1
	}
	return e2
}
