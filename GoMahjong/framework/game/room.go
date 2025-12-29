package game

import (
	"common/log"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"framework/game/engines"
	"framework/game/share"
	"sync"
	"time"
)

// Room 游戏房间, 创建时分配游戏引擎，负责游戏逻辑以外的事务(如观战，路由映射)
type Room struct {
	ID         string                     // 房间 ID
	Users      map[string]*share.UserInfo // userID -> UserInfo（Engine 和 Room 共用）
	AllowWatch bool                       // 是否允许观战
	Engine     engines.Engine             // 游戏引擎
	CreatedAt  time.Time                  // 创建时间
	mu         sync.RWMutex               // 保护 Users 的读写锁
}

// GenerateRoomID 生成房间 ID
// 格式：room_<timestamp>_<random>
func GenerateRoomID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("room_%d_%s", timestamp, randomStr)
}

// Close 关闭房间并释放资源
func (r *Room) Close() {
	// 调用引擎释放资源
	if r.Engine != nil {
		r.Engine.Close()
	}
}

// NewRoom 创建新房间（使用原型模式，Engine 由外部注入）
// engine: 克隆的游戏引擎实例
// users: userID -> UserInfo 的映射（已分配座位）
func NewRoom(engine engines.Engine, users map[string]string) (*Room, error) {
	if engine == nil {
		return nil, fmt.Errorf("游戏引擎不能为空")
	}
	userInfo := make(map[string]*share.UserInfo)
	for k, v := range users {
		userInfo[k] = share.NewUserInfo(k, v)
	}

	room := &Room{
		ID:         GenerateRoomID(),
		Users:      userInfo,
		AllowWatch: false,
		Engine:     engine,
		CreatedAt:  time.Now(),
	}

	return room, nil
}

// RemovePlayer 从房间移除玩家
func (r *Room) RemovePlayer(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Users[userID]; !exists {
		return fmt.Errorf("玩家 %s 不在房间中", userID)
	}

	delete(r.Users, userID)
	log.Info(fmt.Sprintf("Room[%s] 玩家 %s 离开房间", r.ID, userID))
	return nil
}

// GetPlayer 获取玩家信息
func (r *Room) GetPlayer(userID string) (*share.UserInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	player, exists := r.Users[userID]
	return player, exists
}

// GetPlayerCount 获取玩家数量
func (r *Room) GetPlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Users)
}

// GetAllPlayers 获取所有玩家列表（返回副本）
func (r *Room) GetAllPlayers() []*share.UserInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]*share.UserInfo, 0, len(r.Users))
	for _, player := range r.Users {
		players = append(players, player)
	}
	return players
}
