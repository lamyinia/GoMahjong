package grpc

import (
	"common/log"
	"context"
	"fmt"
	"time"

	"march/application/service"
	"march/pb"
)

// MatchServer 实现 gRPC MatchService
type MatchServer struct {
	pb.UnimplementedMatchServiceServer
	matchService service.MatchService
}

func NewMatchServer(matchService service.MatchService) *MatchServer {
	return &MatchServer{
		matchService: matchService,
	}
}

// JoinQueue 处理玩家加入匹配队列
func (s *MatchServer) JoinQueue(ctx context.Context, req *pb.JoinQueueRequest) (*pb.JoinQueueResponse, error) {
	if req.GetUserID() == "" || req.GetNodeID() == "" {
		return &pb.JoinQueueResponse{
			Success: false,
			Message: "userID 和 nodeID 不能为空",
		}, nil
	}

	if err := s.matchService.JoinQueue(ctx, req.GetUserID(), req.GetNodeID()); err != nil {
		log.Debug("进入匹配队列失败: %#v", req)
		return &pb.JoinQueueResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	queueID := fmt.Sprintf("%s:%d", req.GetUserID(), time.Now().UnixNano())

	return &pb.JoinQueueResponse{
		Success:          true,
		Message:          "加入匹配队列成功",
		QueueID:          queueID,
		EstimatedSeconds: 0,
	}, nil
}

// LeaveQueue 处理玩家取消匹配
func (s *MatchServer) LeaveQueue(ctx context.Context, req *pb.LeaveQueueRequest) (*pb.LeaveQueueResponse, error) {
	if req.GetUserID() == "" {
		return &pb.LeaveQueueResponse{
			Success: false,
			Message: "userID 不能为空",
		}, nil
	}

	if err := s.matchService.LeaveQueue(ctx, req.GetUserID()); err != nil {
		return &pb.LeaveQueueResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.LeaveQueueResponse{
		Success: true,
		Message: "已取消匹配",
	}, nil
}

// QueryStatus 查询匹配状态（当前返回占位信息，后续可扩展真实数据）
func (s *MatchServer) QueryStatus(ctx context.Context, req *pb.QueryStatusRequest) (*pb.QueryStatusResponse, error) {
	status := pb.QueryStatusResponse_STATUS_UNKNOWN
	if req.GetUserID() != "" {
		status = pb.QueryStatusResponse_STATUS_WAITING
	}

	return &pb.QueryStatusResponse{
		Status:           status,
		Position:         0,
		EstimatedSeconds: 0,
		GameNodeID:       "",
	}, nil
}
