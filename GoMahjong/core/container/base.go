package container

import (
	"common/config"
	"common/database"
	"common/log"
)

// BaseContainer 基础容器，管理所有服务共享的资源（数据库连接等）
type BaseContainer struct {
	mongo *database.MongoManager
	redis *database.RedisManager
}

// NewBase 创建基础容器并初始化所有共享依赖
func NewBase(conf config.DatabaseConf) *BaseContainer {
	mongo := database.NewMongo(conf.MongoConf)
	redis := database.NewRedis(conf.RedisConf)

	if mongo == nil || redis == nil {
		log.Fatal("数据库初始化失败")
		return nil
	}

	log.Info("mongodb、redis 数据库服务启动成功")

	return &BaseContainer{
		mongo: mongo,
		redis: redis,
	}
}

// GetMongo 获取 Mongo 管理器
func (c *BaseContainer) GetMongo() *database.MongoManager {
	return c.mongo
}

// GetRedis 获取 Redis 管理器
func (c *BaseContainer) GetRedis() *database.RedisManager {
	return c.redis
}

// Close 关闭所有资源
func (c *BaseContainer) Close() error {
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
