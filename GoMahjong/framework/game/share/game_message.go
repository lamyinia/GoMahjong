package share

// GameEvent 游戏事件接口
type GameEvent interface {
	GetUserID() string
	GetEventType() string
}

type GameMessageEvent struct {
	UserID string `json:"userID"` // 用户 ID（用于查找房间）
}

func (e *GameMessageEvent) GetUserID() string {
	return e.UserID
}

type DropTileEvent struct {
	GameMessageEvent
}

func (e *DropTileEvent) GetEventType() string {
	return "DropTile"
}

type PengTileEvent struct {
	GameMessageEvent
}

func (e *PengTileEvent) GetEventType() string {
	return "PengTile"
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
