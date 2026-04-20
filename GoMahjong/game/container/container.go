package container

import (
	"game/infrastructure/config"
	"game/infrastructure/database"
	"game/infrastructure/log"
	"game/infrastructure/persistence"
	gameRuntime "game/runtime"
	"game/runtime/application/service/impl"
	"game/runtime/engines"
	"game/runtime/engines/mahjong"
	"sync"
)

type GameContainer struct {
	mongo      *database.MongoManager
	redis      *database.RedisManager
	GameWorker *gameRuntime.Worker

	closed bool
	mu     sync.Mutex
}

func NewContainer() *GameContainer {
	mongo := database.NewMongo(config.GameNodeConfig.DatabaseConf.MongoConf)
	redis := database.NewRedis(config.GameNodeConfig.DatabaseConf.RedisConf)

	if mongo == nil || redis == nil {
		log.Fatal("数据库初始化失败")
		return nil
	}

	gameRecordRepo := persistence.NewGameRecordRepository(mongo)

	worker := gameRuntime.NewWorker(config.GameNodeConfig.ID)
	worker.SetGameRecordRepository(gameRecordRepo)

	enginePrototypes := createEnginePrototypes(worker)
	for engineType, engine := range enginePrototypes {
		if err := worker.RoomManager.SetEnginePrototype(engineType, engine); err != nil {
			log.Fatal("注入 Engine 原型失败: %v", err)
			return nil
		}
	}

	gameService := impl.NewGameService(worker.RoomManager, worker)
	worker.SetGameService(gameService)

	return &GameContainer{
		mongo:      mongo,
		redis:      redis,
		GameWorker: worker,
	}
}

func createEnginePrototypes(worker *gameRuntime.Worker) map[int32]engines.Engine {
	prototypes := make(map[int32]engines.Engine)
	prototypes[int32(engines.RIICHI_MAHJONG_4P_ENGINE)] = mahjong.NewRiichiMahjong4p(worker)
	log.Info("GameContainer 创建 Engine 原型完成，共 %d 个引擎", len(prototypes))
	return prototypes
}

func (c *GameContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	if c.GameWorker != nil {
		c.GameWorker.Close()
	}
	if c.mongo != nil {
		_ = c.mongo.Close()
	}
	if c.redis != nil {
		_ = c.redis.Close()
	}

	c.closed = true
	log.Info("GameContainer 已关闭")
	return nil
}
