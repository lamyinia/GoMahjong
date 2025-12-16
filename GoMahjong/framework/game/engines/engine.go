package engines

import (
	"framework/game/share"
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

// Engine 使用原型模式，每个游戏房间都有一个游戏引擎
type Engine interface {
	// InitializeEngine 初始化游戏引擎
	// users: Room.UserMap map，Engine 和 Room 共用
	InitializeEngine(users map[string]*share.UserInfo) error

	// CalculateScore 计算分数
	CalculateScore() map[string]int

	// DriveEngine 驱动游戏逻辑
	DriveEngine(event share.GameEvent) // 似乎跟 go 的特性有关，所以这里实际上需要传指针

	// Clone 克隆引擎实例（用于原型模式）
	Clone() Engine
}
