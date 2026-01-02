package engines

import (
	"runtime/game/share"
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
	InitializeEngine(roomID string, users map[string]*share.UserInfo) error

	// NotifyEvent 通知游戏事件（入队，由引擎内部串行处理）
	NotifyEvent(event share.GameEvent) // 似乎跟 go 的特性有关，所以这里实际上需要传指针

	// Clone 克隆引擎实例（用于原型模式）
	Clone() Engine

	// Terminate 触发销毁房间（异步请求）
	Terminate()

	// Close 释放引擎内部资源
	Close()
}
