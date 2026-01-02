package share

// UserInfo 和游戏逻辑隔离的用户信息
type UserInfo struct {
	UserID          string // 用户 ID
	ConnectorNodeID string // connector 的 topic（用于主动推送消息）
	IsOnline        bool   // 是否在线
	SeatIndex       int
}

// NewUserInfo 创建玩家信息
func NewUserInfo(userID, connectorNodeID string) *UserInfo {
	return &UserInfo{
		UserID:          userID,
		ConnectorNodeID: connectorNodeID,
		IsOnline:        true,
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
