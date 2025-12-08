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

const (
	MaxPlayers = 4 // 麻将4人游戏
)

// Room 游戏房间 管理房间内的玩家和游戏状态
type Room struct {
	ID        string                       // 房间 ID
	Players   map[string]*share.PlayerInfo // playerID -> PlayerInfo
	Status    RoomStatus                   // 房间状态
	Engine    engines.Engine               // 游戏引擎
	Snapshot  interface{}                  // 游戏快照（用于断线重连）
	CreatedAt time.Time                    // 创建时间
	mu        sync.RWMutex                 // 保护 Players 的读写锁
}

// RoomStatus 房间状态
type RoomStatus int

const (
	RoomStatusWaiting  RoomStatus = iota // 等待中
	RoomStatusPlaying                    // 游戏中
	RoomStatusFinished                   // 已结束
)

// GenerateRoomID 生成房间 ID
// 格式：room_<timestamp>_<random>
func GenerateRoomID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("room_%d_%s", timestamp, randomStr)
}

// NewRoom 创建新房间
// engineType: 游戏引擎类型
func NewRoom(engineType int32) (*Room, error) {
	// 创建游戏引擎
	engine, err := engines.NewEngine(engineType)
	if err != nil {
		return nil, fmt.Errorf("创建游戏引擎失败: %v", err)
	}

	return &Room{
		ID:        GenerateRoomID(),
		Players:   make(map[string]*share.PlayerInfo),
		Status:    RoomStatusWaiting,
		Engine:    engine,
		Snapshot:  nil,
		CreatedAt: time.Now(),
	}, nil
}

// AddPlayer 添加玩家到房间
// userID: 用户 ID
// connectorNodeID: connector 的 topic（用于主动推送消息）
// 返回：座位索引和错误
func (r *Room) AddPlayer(userID, connectorNodeID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查玩家是否已在房间中
	if _, exists := r.Players[userID]; exists {
		return -1, fmt.Errorf("玩家 %s 已在房间中", userID)
	}

	// 检查房间状态
	if r.Status != RoomStatusWaiting {
		return -1, fmt.Errorf("房间状态不允许加入玩家，当前状态: %v", r.Status)
	}

	// 分配座位索引（0-3）
	seatIndex := r.findAvailableSeat()
	if seatIndex < 0 {
		return -1, fmt.Errorf("没有可用座位")
	}

	// 创建玩家信息
	player := share.NewPlayerInfo(userID, connectorNodeID, seatIndex)
	r.Players[userID] = player

	log.Info(fmt.Sprintf("Room[%s] 玩家 %s 加入房间，座位: %d", r.ID, userID, seatIndex))
	return seatIndex, nil
}

// RemovePlayer 从房间移除玩家
func (r *Room) RemovePlayer(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Players[userID]; !exists {
		return fmt.Errorf("玩家 %s 不在房间中", userID)
	}

	delete(r.Players, userID)
	log.Info(fmt.Sprintf("Room[%s] 玩家 %s 离开房间", r.ID, userID))
	return nil
}

// GetPlayer 获取玩家信息
func (r *Room) GetPlayer(userID string) (*share.PlayerInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	player, exists := r.Players[userID]
	return player, exists
}

// GetPlayerCount 获取玩家数量
func (r *Room) GetPlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Players)
}

// UpdateStatus 更新房间状态
func (r *Room) UpdateStatus(status RoomStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldStatus := r.Status
	r.Status = status
	log.Info(fmt.Sprintf("Room[%s] 状态更新: %v -> %v", r.ID, oldStatus, status))
}

// GetStatus 获取房间状态
func (r *Room) GetStatus() RoomStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.Status
}

// SaveSnapshot 保存游戏快照（用于断线重连）
func (r *Room) SaveSnapshot(snapshot interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Snapshot = snapshot
	log.Info(fmt.Sprintf("Room[%s] 保存游戏快照", r.ID))
}

// GetSnapshot 获取游戏快照
func (r *Room) GetSnapshot() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.Snapshot
}

// GetPlayerSnapshot 获取指定玩家的游戏快照
func (r *Room) GetPlayerSnapshot(userID string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	player, exists := r.Players[userID]
	if !exists {
		return nil
	}

	return player.GetSnapshot()
}

// SavePlayerSnapshot 保存指定玩家的游戏快照
func (r *Room) SavePlayerSnapshot(userID string, snapshot interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.Players[userID]
	if !exists {
		return fmt.Errorf("玩家 %s 不在房间中", userID)
	}

	player.SaveSnapshot(snapshot)
	log.Info(fmt.Sprintf("Room[%s] 保存玩家 %s 的游戏快照", r.ID, userID))
	return nil
}

// GetAllPlayers 获取所有玩家列表（返回副本）
func (r *Room) GetAllPlayers() []*share.PlayerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]*share.PlayerInfo, 0, len(r.Players))
	for _, player := range r.Players {
		players = append(players, player)
	}
	return players
}

// findAvailableSeat 查找可用座位索引
// 返回：可用座位索引（0-3），如果没有可用座位返回 -1
func (r *Room) findAvailableSeat() int {
	// 获取已占用的座位
	occupiedSeats := make(map[int]bool)
	for _, player := range r.Players {
		occupiedSeats[player.SeatIndex] = true
	}

	// 查找第一个可用座位
	for i := 0; i < MaxPlayers; i++ {
		if !occupiedSeats[i] {
			return i
		}
	}

	return -1
}
