package share

// UserInfo 玩家信息
// 用于在房间中管理玩家状态
type UserInfo struct {
	UserID          string      // 用户 ID
	ConnectorNodeID string      // connector 的 topic（用于主动推送消息）
	IsOnline        bool        // 是否在线
	SeatIndex       int         // 座位索引（0-3，麻将4人）
	Snapshot        interface{} // 游戏快照（用于断线重连）
}

// NewUserInfo 创建玩家信息
func NewUserInfo(userID, connectorNodeID string, seatIndex int) *UserInfo {
	return &UserInfo{
		UserID:          userID,
		ConnectorNodeID: connectorNodeID,
		IsOnline:        true,
		SeatIndex:       seatIndex,
		Snapshot:        nil,
	}
}

// SetOffline 设置玩家离线
func (pi *UserInfo) SetOffline() {
	pi.IsOnline = false
}

// SetOnline 设置玩家在线
func (pi *UserInfo) SetOnline(connectorNodeID string) {
	pi.IsOnline = true
	pi.ConnectorNodeID = connectorNodeID
}

// SaveSnapshot 保存游戏快照（用于断线重连）
func (pi *UserInfo) SaveSnapshot(snapshot interface{}) {
	pi.Snapshot = snapshot
}

// GetSnapshot 获取游戏快照
func (pi *UserInfo) GetSnapshot() interface{} {
	return pi.Snapshot
}
