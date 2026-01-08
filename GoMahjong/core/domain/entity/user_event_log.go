package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserEventLog 用户事件日志
// 记录用户的操作流水，如登录、登出、开始游戏、结束游戏等
type UserEventLog struct {
	ID        primitive.ObjectID     `bson:"_id"`
	UserID    string                 `bson:"user_id"`              // 用户ID
	EventType string                 `bson:"event_type"`           // 事件类型
	Timestamp time.Time              `bson:"timestamp"`            // 发生时间
	IP        string                 `bson:"ip,omitempty"`         // IP地址（可选）
	UserAgent string                 `bson:"user_agent,omitempty"` // 用户代理（可选）
	Metadata  map[string]interface{} `bson:"metadata,omitempty"`   // 扩展数据（灵活存储）
	CreatedAt time.Time              `bson:"created_at"`           // 创建时间（用于TTL索引）
}

// NewUserEventLog 创建用户事件日志
func NewUserEventLog(userID, eventType string) *UserEventLog {
	return &UserEventLog{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		EventType: eventType,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// SetIP 设置IP地址
func (log *UserEventLog) SetIP(ip string) {
	log.IP = ip
}

// SetUserAgent 设置用户代理
func (log *UserEventLog) SetUserAgent(userAgent string) {
	log.UserAgent = userAgent
}

// SetMetadata 设置元数据
func (log *UserEventLog) SetMetadata(key string, value interface{}) {
	if log.Metadata == nil {
		log.Metadata = make(map[string]interface{})
	}
	log.Metadata[key] = value
}

// 事件类型常量
const (
	EventTypeLogin         = "LOGIN"          // 登录
	EventTypeLogout        = "LOGOUT"         // 登出
	EventTypeRegister      = "REGISTER"       // 注册
	EventTypeGameStart     = "GAME_START"     // 开始游戏
	EventTypeGameEnd       = "GAME_END"       // 结束游戏
	EventTypeGameAbort     = "GAME_ABORT"     // 游戏中断
	EventTypeRoomJoin      = "ROOM_JOIN"      // 加入房间
	EventTypeRoomLeave     = "ROOM_LEAVE"     // 离开房间
	EventTypeRoomCreate    = "ROOM_CREATE"    // 创建房间
	EventTypeRankingChange = "RANKING_CHANGE" // 段位变化
	EventTypeProfileUpdate = "PROFILE_UPDATE" // 资料更新
)
