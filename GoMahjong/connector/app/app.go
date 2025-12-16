package app

import (
	"common/log"
	"common/rpc"
	"context"
	"core/container"
	"fmt"
	matchpb "march/pb"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(ctx context.Context) error {
	connectorContainer := container.NewConnectorContainer()
	defer connectorContainer.Close()

	connectorConfig := connectorContainer.GetConnectorConfig()
	cfg, ok := connectorConfig.(interface{ GetID() string })
	if !ok || cfg == nil {
		log.Fatal("connector 配置类型错误")
		return nil
	}

	// 初始化 RPC 客户端并检测 march gRPC 是否可达
	rpc.Init()
	if err := healthCheckMarch(cfg.GetID()); err != nil {
		log.Fatal("march RPC 健康检查失败: %v", err)
	}

	worker := connectorContainer.GetWorker()
	if worker == nil {
		log.Fatal("Worker 获取失败")
		return nil
	}
	go func() {
		addr := "localhost:8082"
		if err := worker.Run(cfg.GetID(), 5000, addr); err != nil {
			log.Fatal("worker 启动失败: %v", err)
		}
	}()

	stop := func() {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		worker.Close()
		log.Info("Worker 已关闭")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			stop()
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				stop()
				log.Info("中断信号，服务停止")
				return nil
			case syscall.SIGHUP:
				stop()
				log.Info("挂起信号，服务停止")
				return nil
			default:
				return nil
			}
		}
	}
}

// healthCheckMarch 确保 march gRPC 可用
func healthCheckMarch(connectorID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := &matchpb.QueryStatusRequest{
		UserID: "health-check",
	}

	resp, err := rpc.MatchClient.QueryStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("QueryStatus 调用失败: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("QueryStatus 返回为空")
	}
	// 如果返回未知状态，再尝试 JoinQueue + LeaveQueue，确保写入链路正常
	if resp.GetStatus() == matchpb.QueryStatusResponse_STATUS_UNKNOWN {
		joinReq := &matchpb.JoinQueueRequest{
			UserID: "health-check",
			NodeID: connectorID,
		}
		if _, err := rpc.MatchClient.JoinQueue(ctx, joinReq); err != nil {
			return fmt.Errorf("JoinQueue 调用失败: %w", err)
		}
		// 最佳努力地移除，避免污染队列
		_, _ = rpc.MatchClient.LeaveQueue(ctx, &matchpb.LeaveQueueRequest{
			UserID: "health-check",
		})
	}

	log.Info("march RPC 健康检查通过")
	return nil
}

func rpcSelfTest(connectorID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	userID := "67473d339411b23c61f5a001"

	//joinResp, err := rpc.MatchClient.JoinQueue(ctx, &matchpb.JoinQueueRequest{
	//	UserID: userID,
	//	NodeID: connectorID,
	//})
	//if err != nil || !joinResp.GetSuccess() {
	//	return fmt.Errorf("JoinQueue 失败, resp=%+v err=%w", joinResp, err)
	//}

	leaveResp, err := rpc.MatchClient.LeaveQueue(ctx, &matchpb.LeaveQueueRequest{
		UserID: userID,
	})
	if err != nil || !leaveResp.GetSuccess() {
		return fmt.Errorf("LeaveQueue 失败, resp=%+v err=%w", leaveResp, err)
	}
	return nil
}
