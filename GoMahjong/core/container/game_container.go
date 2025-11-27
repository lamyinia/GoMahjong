package container

import (
	"common/config"
	"common/log"
	"core/domain/repository"
	"core/infrastructure/persistence"
	"fmt"
	"framework/game"
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

	// 从 LocalConfig 获取 serverID
	gameConfig, err := config.InjectedConfig.GetGameConfig()
	if err != nil {
		log.Fatal("获取 GameConfig 失败: %v", err)
		return nil
	}

	// 创建 GameWorker
	worker := game.NewWorker(gameConfig.GetID())

	return &GameContainer{
		BaseContainer:  base,
		userRepository: userRepo,
		GameWorker:     worker,
	}
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
