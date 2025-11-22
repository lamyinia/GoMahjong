package game

// PlayerInfo 玩家信息
// 用于在房间中管理玩家状态
type PlayerInfo struct {
	UserID         string      // 用户 ID
	ConnectorTopic string      // connector 的 topic（用于主动推送消息）
	IsOnline       bool        // 是否在线
	SeatIndex      int         // 座位索引（0-3，麻将4人）
	Snapshot       interface{} // 游戏快照（用于断线重连）
}

// NewPlayerInfo 创建玩家信息
func NewPlayerInfo(userID, connectorTopic string, seatIndex int) *PlayerInfo {
	return &PlayerInfo{
		UserID:         userID,
		ConnectorTopic: connectorTopic,
		IsOnline:       true,
		SeatIndex:      seatIndex,
		Snapshot:       nil,
	}
}

// SetOffline 设置玩家离线
func (pi *PlayerInfo) SetOffline() {
	pi.IsOnline = false
}

// SetOnline 设置玩家在线
func (pi *PlayerInfo) SetOnline(connectorTopic string) {
	pi.IsOnline = true
	pi.ConnectorTopic = connectorTopic
}

// SaveSnapshot 保存游戏快照（用于断线重连）
func (pi *PlayerInfo) SaveSnapshot(snapshot interface{}) {
	pi.Snapshot = snapshot
}

// GetSnapshot 获取游戏快照
func (pi *PlayerInfo) GetSnapshot() interface{} {
	return pi.Snapshot
}
