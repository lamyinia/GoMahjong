package grpc

import (
	"context"
	"framework/game/application/service"
	pb "game/pb"
)

type GameServer struct {
	pb.UnimplementedGameServiceServer
	gameService service.GameService
}

// NewGameServer 创建 GameServer 实例
func NewGameServer(gameService service.GameService) *GameServer {
	return &GameServer{
		gameService: gameService,
	}
}

// CreateRoom 实现 GameServiceServer 接口
func (s *GameServer) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
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
