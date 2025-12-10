package game

import (
	"common/log"
	"errors"
	"fmt"
	"framework/game/engines"
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
// players: 玩家列表，格式为 map[userID]connectorTopic
// engineType: 游戏引擎类型
// 返回：房间实例和错误
func (rm *RoomManager) CreateRoom(players map[string]string, engineType int32) (*Room, error) {
	if len(players) == 0 {
		return nil, errors.New("玩家列表不能为空")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 检查玩家是否已在其他房间中
	for userID := range players {
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

	// 步骤 2：创建新房间（注入克隆的 Engine）
	room, err := NewRoom(engine)
	if err != nil {
		return nil, fmt.Errorf("创建房间失败: %v", err)
	}

	// 步骤 3：添加玩家到房间
	for userID, connectorNodeID := range players {
		seatIndex, err := room.AddPlayer(userID, connectorNodeID)
		if err != nil {
			// 如果添加失败，清理已添加的玩家
			rm.cleanupRoom(room.ID)
			return nil, fmt.Errorf("添加玩家 %s 失败: %v", userID, err)
		}

		// 更新路由映射
		rm.playerRoom[userID] = room.ID
		log.Info(fmt.Sprintf("RoomManager 玩家 %s 路由到房间 %s，connector: %s，座位: %d", userID, room.ID, connectorNodeID, seatIndex))
	}

	// 步骤 4：初始化游戏引擎（所有玩家都添加后，传入 Room.Users）
	if err := room.Engine.Initialize(room.Users); err != nil {
		rm.cleanupRoom(room.ID)
		return nil, fmt.Errorf("初始化游戏引擎失败: %v", err)
	}

	// 保存房间
	rm.rooms[room.ID] = room

	log.Info(fmt.Sprintf("RoomManager 创建房间 %s，玩家数: %d，引擎类型: %d", room.ID, len(players), engineType))
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

	// 删除房间
	delete(rm.rooms, roomID)

	log.Info(fmt.Sprintf("RoomManager 删除房间 %s", roomID))
	return nil
}

// AddPlayerToRoom 添加玩家到房间（更新路由）
// 用于玩家重新连接或加入已有房间
func (rm *RoomManager) AddPlayerToRoom(roomID, userID, connectorTopic string) (int, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return -1, fmt.Errorf("房间 %s 不存在", roomID)
	}

	// 检查玩家是否已在其他房间中
	if existingRoomID, exists := rm.playerRoom[userID]; exists && existingRoomID != roomID {
		return -1, fmt.Errorf("玩家 %s 已在房间 %s 中", userID, existingRoomID)
	}

	// 添加玩家到房间
	seatIndex, err := room.AddPlayer(userID, connectorTopic)
	if err != nil {
		return -1, err
	}

	// 更新路由映射
	rm.playerRoom[userID] = roomID

	log.Info(fmt.Sprintf("RoomManager 玩家 %s 添加到房间 %s，connector: %s，座位: %d", userID, roomID, connectorTopic, seatIndex))
	return seatIndex, nil
}

// RemovePlayerFromRoom 从房间移除玩家（更新路由）
func (rm *RoomManager) RemovePlayerFromRoom(roomID, userID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return fmt.Errorf("房间 %s 不存在", roomID)
	}

	// 从房间移除玩家
	err := room.RemovePlayer(userID)
	if err != nil {
		return err
	}

	// 清理路由映射
	delete(rm.playerRoom, userID)

	log.Info(fmt.Sprintf("RoomManager 玩家 %s 从房间 %s 移除", userID, roomID))
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

	// 删除房间
	delete(rm.rooms, roomID)
}
