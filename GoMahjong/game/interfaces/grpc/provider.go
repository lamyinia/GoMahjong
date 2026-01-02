package grpc

import (
	"context"
	pb "game/pb"
	"runtime/game/application/service"
)

type GameProvider struct {
	pb.UnimplementedGameServiceServer
	gameService service.GameService
}

// NewGameProvider 创建 GameProvider 实例
func NewGameProvider(gameService service.GameService) *GameProvider {
	return &GameProvider{
		gameService: gameService,
	}
}

// CreateRoom 实现 GameServiceServer 接口
func (s *GameProvider) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	// 将 proto 请求转换为 service 请求
	serviceReq := &service.CreateRoomReq{
		Players:    req.Players,
		EngineType: req.EngineType,
	}

	// 调用 service 层
	serviceResp, err := s.gameService.CreateRoom(ctx, serviceReq)
	if err != nil {
		return &pb.CreateRoomResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 将 service 响应转换为 proto 响应
	return &pb.CreateRoomResponse{
		Success: serviceResp.Success,
		RoomID:  serviceResp.RoomID,
		Message: serviceResp.Message,
	}, nil
}
