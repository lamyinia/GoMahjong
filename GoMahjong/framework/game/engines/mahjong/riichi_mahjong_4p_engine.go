package mahjong

import (
	"fmt"
	"framework/game"
	"framework/game/engines"
	"framework/game/share"
)

type RiichiMahjong4p struct {
	State  engines.GameState
	Worker *game.Worker               // Game Worker（在 GameContainer 创建原型时注入）
	Users  map[string]*share.UserInfo // Room.Users 的引用（Engine 和 Room 共用）
}

// NewRiichiMahjong4p 创建立直麻将 4 人引擎实例
func NewRiichiMahjong4p(worker *game.Worker) *RiichiMahjong4p {
	return &RiichiMahjong4p{
		State:  engines.GameWaiting,
		Worker: worker,
		Users:  nil, // 在 Initialize 时设置
	}
}

// Initialize 初始化游戏引擎
// users: Room.Users map，Engine 和 Room 共用
func (eg *RiichiMahjong4p) Initialize(users map[string]*share.UserInfo) error {
	if len(users) != 4 {
		return fmt.Errorf("立直麻将需要4个玩家，当前: %d", len(users))
	}

	eg.Users = users
	eg.State = engines.GameInProgress
	return nil
}

// CalculateScore 计算分数
func (eg *RiichiMahjong4p) CalculateScore() map[string]int {
	// TODO: 实现分数计算逻辑
	return nil
}

// DriveEngine 驱动游戏逻辑
func (eg *RiichiMahjong4p) DriveEngine(event share.GameEvent) {
	// TODO: 根据事件类型处理游戏逻辑
	// 支持的事件：DropTileEvent、PengTileEvent、GangEvent、HuEvent
}

// Clone 克隆引擎实例（用于原型模式）
func (eg *RiichiMahjong4p) Clone() engines.Engine {
	return &RiichiMahjong4p{
		State:  engines.GameWaiting,
		Worker: eg.Worker, // 克隆后保持相同的 Worker 引用
	}
}
