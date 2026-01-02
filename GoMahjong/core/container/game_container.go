package container

import (
	"common/config"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
	"fmt"
	"runtime/game"
	"runtime/game/application/service/impl"
	"runtime/game/engines"
	"runtime/game/engines/mahjong"

	"sync"
)

// GameContainer game 服务专用容器
// 继承 BaseContainer 的数据库连接，添加 game 服务特定的依赖
type GameContainer struct {
	*BaseContainer
	userRepository repository.UserRepository
	GameWorker     *game.Worker
	closed         bool
	mu             sync.Mutex
}

// NewGameContainer 创建 game 服务容器
func NewGameContainer() *GameContainer {
	base := NewBase(config.GameNodeConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	// 创建 game 服务需要的仓储
	userRepo := persistence.NewMongoUserRepository(base.mongo)
	// 创建 GameWorker
	worker := game.NewWorker(config.GameNodeConfig.ID)
	// 步骤 1：创建 Engine 原型（使用原型模式，注入 Worker）,目前只支持立直麻将 4 人引擎
	enginePrototypes := createEnginePrototypes(worker)
	// 步骤 2：注入 Engine 原型到 RoomManager
	for engineType, engine := range enginePrototypes {
		if err := worker.RoomManager.SetEnginePrototype(engineType, engine); err != nil {
			log.Fatal("注入 Engine 原型失败: %v", err)
			return nil
		}
	}

	// 步骤 3：创建 GameService 并注入 Worker
	gameService := impl.NewGameService(worker.RoomManager, worker)
	worker.SetGameService(gameService)

	return &GameContainer{
		BaseContainer:  base,
		userRepository: userRepo,
		GameWorker:     worker,
	}
}

// createEnginePrototypes 创建所有 Engine 原型
// worker: Game Worker（注入到 Engine 原型中）
// 返回：map[engineType]Engine 原型
func createEnginePrototypes(worker *game.Worker) map[int32]engines.Engine {
	prototypes := make(map[int32]engines.Engine)

	prototypes[int32(engines.RIICHI_MAHJONG_4P_ENGINE)] = mahjong.NewRiichiMahjong4p(worker)

	log.Info("GameContainer 创建 Engine 原型完成，共 %d 个引擎", len(prototypes))
	return prototypes
}

// Close 关闭容器资源（幂等操作，可以安全地多次调用）
// 关闭顺序：1. GameWorker 2. BaseContainer（数据库连接）
func (c *GameContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	var errs []error

	if c.GameWorker != nil {
		c.GameWorker.Close()
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

	log.Info("GameContainer 已关闭")
	return nil
}
