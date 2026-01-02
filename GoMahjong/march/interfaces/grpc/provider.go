package grpc

import (
	"common/log"
	"context"
	"fmt"
	"runtime/march/application/service"
	"time"

	"march/pb"
)

// MatchProvider 实现 gRPC MatchService
type MatchProvider struct {
	pb.UnimplementedMatchServiceServer
	matchService service.MatchService
}

func NewMatchProvider(matchService service.MatchService) *MatchProvider {
	return &MatchProvider{
		matchService: matchService,
	}
}

// JoinQueue 处理玩家加入匹配队列
func (p *MatchProvider) JoinQueue(ctx context.Context, req *pb.JoinQueueRequest) (*pb.JoinQueueResponse, error) {
	if req.GetUserID() == "" || req.GetNodeID() == "" {
		return &pb.JoinQueueResponse{
			Success: false,
			Message: "userID 和 nodeID 不能为空",
		}, nil
	}

	if err := p.matchService.JoinQueue(ctx, req.GetUserID(), req.GetNodeID()); err != nil {
		log.Warn("进入匹配队列失败: %#v", req)
		return &pb.JoinQueueResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	queueID := fmt.Sprintf("%p:%d", req.GetUserID(), time.Now().UnixNano())

	return &pb.JoinQueueResponse{
		Success:          true,
		Message:          "加入匹配队列成功",
		QueueID:          queueID,
		EstimatedSeconds: 0,
	}, nil
}

// LeaveQueue 处理玩家取消匹配
func (p *MatchProvider) LeaveQueue(ctx context.Context, req *pb.LeaveQueueRequest) (*pb.LeaveQueueResponse, error) {
	if req.GetUserID() == "" {
		return &pb.LeaveQueueResponse{
			Success: false,
			Message: "userID 不能为空",
		}, nil
	}

	if err := p.matchService.LeaveQueue(ctx, req.GetUserID()); err != nil {
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
func (p *MatchProvider) QueryStatus(ctx context.Context, req *pb.QueryStatusRequest) (*pb.QueryStatusResponse, error) {
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
