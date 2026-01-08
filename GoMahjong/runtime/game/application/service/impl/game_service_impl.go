package impl

import (
	"common/log"
	"context"
	"fmt"
	"runtime/game"
	"runtime/game/application/service"
)

type GameServiceImpl struct {
	roomManager *game.RoomManager
	worker      *game.Worker
}

// NewGameService 创建 GameService 实例
func NewGameService(roomManager *game.RoomManager, worker *game.Worker) service.GameService {
	return &GameServiceImpl{
		roomManager: roomManager,
		worker:      worker,
	}
}

// CreateRoom 创建游戏房间
func (s *GameServiceImpl) CreateRoom(ctx context.Context, req *service.CreateRoomReq) (*service.CreateRoomResp, error) {
	if req == nil {
		return &service.CreateRoomResp{
			Success: false,
			Message: "请求不能为空",
		}, nil
	}

	if len(req.Players) == 0 {
		return &service.CreateRoomResp{
			Success: false,
			Message: "玩家列表不能为空",
		}, nil
	}

	// 创建房间
	room, err := s.roomManager.CreateRoom(req.Players, req.EngineType)
	if err != nil {
		log.Error(fmt.Sprintf("GameService 创建房间失败: %v", err))
		return &service.CreateRoomResp{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 推送逻辑已迁移到 Engine.InitializeEngine 中
	// 避免 GetPlayerConnector 的锁竞争，提升性能
	// 如果 Engine 初始化失败，推送也会失败，这是合理的

	log.Info(fmt.Sprintf("GameService 创建房间成功: %s, 玩家数: %d", room.ID, len(req.Players)))

	return &service.CreateRoomResp{
		Success: true,
		RoomID:  room.ID,
		Message: "房间创建成功",
	}, nil
}
