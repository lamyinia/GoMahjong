package share

// Tile 牌的定义
type Tile struct {
	Type int // 牌的类型
	ID   int // 牌的 ID
}

// GameEvent 游戏事件接口
type GameEvent interface {
	GetUserID() string
	GetEventType() string
}

type GameMessageEvent struct {
	UserID string `json:"userID"` // 用户 ID（用于查找座位）
}

func (e *GameMessageEvent) GetUserID() string {
	return e.UserID
}

type DropTileEvent struct {
	GameMessageEvent
	Tile Tile `json:"tile"` // 打出的牌
}

func (e *DropTileEvent) GetEventType() string {
	return "DropTile"
}

func (e *DropTileEvent) GetTile() Tile {
	return e.Tile
}

type PengTileEvent struct {
	GameMessageEvent
}

func (e *PengTileEvent) GetEventType() string {
	return "Peng"
}

type HuEvent struct {
	GameMessageEvent
}

func (e *HuEvent) GetEventType() string {
	return "Hu"
}

type GangEvent struct {
	GameMessageEvent
}

func (e *GangEvent) GetEventType() string {
	return "Gang"
}

type ChiEvent struct {
	GameMessageEvent
}

func (e *ChiEvent) GetEventType() string {
	return "Chi"
}

type RiichiEvent struct {
	GameMessageEvent
}

func (e *RiichiEvent) GetEventType() string {
	return "Riichi"
}
