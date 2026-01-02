package game

import (
	"common/log"
	"errors"
	"fmt"
	"runtime/game/engines"
	"sync"
)

// RoomManager 房间管理器
// 管理所有游戏房间实例，使用原型模式管理 Engine
type RoomManager struct {
	rooms            map[string]*Room         // roomID -> Room
	playerRoom       map[string]string        // playerID -> roomID
	enginePrototypes map[int32]engines.Engine // engineType -> Engine 原型
	mu               sync.RWMutex
}

// NewRoomManager 创建房间管理器
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:            make(map[string]*Room),
		playerRoom:       make(map[string]string),
		enginePrototypes: make(map[int32]engines.Engine),
	}
}

// SetEnginePrototype 注入 Engine 原型
// 在 GameContainer 初始化时调用
func (rm *RoomManager) SetEnginePrototype(engineType int32, engine engines.Engine) error {
	if engine == nil {
		return fmt.Errorf("Engine 原型不能为空")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.enginePrototypes[engineType] = engine
	log.Info(fmt.Sprintf("RoomManager 注入 Engine 原型: engineType=%d", engineType))
	return nil
}

// CreateRoom 创建房间并添加玩家（使用原型模式）
// 返回：房间实例和错误
func (rm *RoomManager) CreateRoom(users map[string]string, engineType int32) (*Room, error) {
	pass := false
	if len(users) == 4 && engineType == int32(engines.RIICHI_MAHJONG_4P_ENGINE) {
		pass = true
	}
	if !pass {
		return nil, errors.New("玩家列表异常")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 检查玩家是否已在其他房间中
	for userID := range users {
		if roomID, exists := rm.playerRoom[userID]; exists {
			log.Warn("玩家 %s 已在房间 %s 中", userID, roomID)
		}
	}

	// 步骤 1：从原型克隆 Engine
	prototype, exists := rm.enginePrototypes[engineType]
	if !exists {
		return nil, fmt.Errorf("不支持的引擎类型: %d", engineType)
	}
	engine := prototype.Clone()
	if engine == nil {
		return nil, fmt.Errorf("克隆游戏引擎失败: engineType=%d", engineType)
	}

	// 步骤 2：创建新房间（注入克隆的 Engine 和已分配座位的玩家）
	room, err := NewRoom(engine, users)
	if err != nil {
		return nil, fmt.Errorf("创建房间失败: %v", err)
	}

	// 步骤 3：更新路由映射
	for userID := range users {
		rm.playerRoom[userID] = room.ID
	}

	// 步骤 4：初始化游戏引擎（传入 Room.UserMap）
	if err := room.Engine.InitializeEngine(room.ID, room.Users); err != nil {
		rm.cleanupRoom(room.ID)
		return nil, fmt.Errorf("初始化游戏引擎失败: %v", err)
	}
	rm.rooms[room.ID] = room

	log.Info(fmt.Sprintf("RoomManager 创建房间 %s，玩家数: %d，引擎类型: %d", room.ID, len(users), engineType))
	return room, nil
}

// GetRoom 获取房间
func (rm *RoomManager) GetRoom(roomID string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	return room, exists
}

// GetPlayerRoom 获取玩家所在房间
func (rm *RoomManager) GetPlayerRoom(playerID string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	roomID, exists := rm.playerRoom[playerID]
	if !exists {
		return nil, false
	}

	room, exists := rm.rooms[roomID]
	return room, exists
}

// DeleteRoom 删除房间
// 会清理房间内的所有玩家路由映射
func (rm *RoomManager) DeleteRoom(roomID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return fmt.Errorf("房间 %s 不存在", roomID)
	}

	// 清理所有玩家的路由映射
	room.mu.RLock()
	for playerID := range room.Users {
		delete(rm.playerRoom, playerID)
	}
	room.mu.RUnlock()

	// 关闭房间资源（释放引擎、计时器等）
	room.Close()

	// 删除房间
	delete(rm.rooms, roomID)

	log.Info(fmt.Sprintf("RoomManager 删除房间 %s", roomID))
	return nil
}

// UpdatePlayerConnector 更新玩家的 connector topic（用于重连）
func (rm *RoomManager) UpdatePlayerConnector(userID, newConnectorTopic string) error {
	rm.mu.RLock()
	roomID, exists := rm.playerRoom[userID]
	rm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("玩家 %s 不在任何房间中", userID)
	}

	rm.mu.RLock()
	room, exists := rm.rooms[roomID]
	rm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("房间 %s 不存在", roomID)
	}

	// 更新玩家的 connector topic
	player, exists := room.GetPlayer(userID)
	if !exists {
		return fmt.Errorf("玩家 %s 不在房间 %s 中", userID, roomID)
	}

	player.SetOnline(newConnectorTopic)
	log.Info(fmt.Sprintf("RoomManager 更新玩家 %s 的 connector topic: %s", userID, newConnectorTopic))
	return nil
}

// GetPlayerConnector 获取玩家的 connector topic
// 通过房间查找玩家信息，获取 connectorTopic
func (rm *RoomManager) GetPlayerConnector(userID string) (string, bool) {
	rm.mu.RLock()
	roomID, exists := rm.playerRoom[userID]
	rm.mu.RUnlock()

	if !exists {
		return "", false
	}

	rm.mu.RLock()
	room, exists := rm.rooms[roomID]
	rm.mu.RUnlock()

	if !exists {
		return "", false
	}

	player, exists := room.GetPlayer(userID)
	if !exists {
		return "", false
	}

	return player.ConnectorNodeID, true
}

// GetStats 获取统计信息（房间数、玩家数）
// 供 Monitor 使用
func (rm *RoomManager) GetStats() (gameCount int, playerCount int) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	gameCount = len(rm.rooms)

	// 统计所有房间的玩家数
	playerSet := make(map[string]bool)
	for _, room := range rm.rooms {
		room.mu.RLock()
		for playerID := range room.Users {
			playerSet[playerID] = true
		}
		room.mu.RUnlock()
	}
	playerCount = len(playerSet)

	return gameCount, playerCount
}

// GetAllRooms 获取所有房间列表（返回副本）
func (rm *RoomManager) GetAllRooms() []*Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]*Room, 0, len(rm.rooms))
	for _, room := range rm.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// cleanupRoom 清理房间（内部方法，需要在持有锁的情况下调用）
func (rm *RoomManager) cleanupRoom(roomID string) {
	room, exists := rm.rooms[roomID]
	if !exists {
		return
	}

	// 清理所有玩家的路由映射
	room.mu.RLock()
	for playerID := range room.Users {
		delete(rm.playerRoom, playerID)
	}
	room.mu.RUnlock()

	// 关闭房间资源（释放引擎、计时器等）
	room.Close()

	// 删除房间
	delete(rm.rooms, roomID)
}
