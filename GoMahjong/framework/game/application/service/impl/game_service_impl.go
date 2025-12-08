package impl

import (
	"common/log"
	"context"
	"encoding/json"
	"fmt"
	"framework/dto"
	"framework/game"
	"framework/game/application/service"
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

	// 推送匹配成功消息给所有玩家
	matchSuccessMsg := &dto.MatchSuccessDTO{
		GameNodeID: s.worker.NodeID,
		Players:    req.Players,
	}
	msgData, err := json.Marshal(matchSuccessMsg)
	if err != nil {
		log.Error(fmt.Sprintf("GameService 序列化消息失败: %v", err))
		return &service.CreateRoomResp{
			Success: true,
			RoomID:  room.ID,
			Message: "房间创建成功，但推送消息失败",
		}, nil
	}

	// 向每个玩家推送匹配成功消息
	for userID := range req.Players {
		if err := s.worker.PushMessage(userID, dto.MatchingSuccess, msgData); err != nil {
			log.Error(fmt.Sprintf("GameService 推送消息给玩家 %s 失败: %v", userID, err))
			// 继续推送其他玩家，不中断
		}
	}

	log.Info(fmt.Sprintf("GameService 创建房间成功: %s, 玩家数: %d", room.ID, len(req.Players)))

	return &service.CreateRoomResp{
		Success: true,
		RoomID:  room.ID,
		Message: "房间创建成功",
	}, nil
}
