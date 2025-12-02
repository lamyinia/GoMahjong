package engines

import (
	"fmt"
	"framework/game"
)

type engineType int32

const (
	RIICHI_MAHJONG_4P_ENGINE engineType = iota // 立直麻将4人 游戏引擎
)

type GameState int

const (
	GameWaiting    GameState = iota // 等待开始
	GameInProgress                  // 进行中
	GamePaused                      // 暂停
	GameFinished                    // 结束
)

// Engine 使用工厂模式，每个游戏房间都有一个游戏引擎
type Engine interface {
	Initialize(players []*game.PlayerInfo) error

	CalculateScore() map[string]int
}

// NewEngine 工厂方法，根据 engineType 创建对应的引擎
func NewEngine(engineType int32) (Engine, error) {
	switch engineType {
	case int32(RIICHI_MAHJONG_4P_ENGINE):
		// 暂时返回 nil，游戏逻辑部分后续实现
		return nil, fmt.Errorf("立直麻将引擎实现中，敬请期待")
	default:
		return nil, fmt.Errorf("不支持的引擎类型: %d", engineType)
	}
}
