package app

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/container"
	"net"
	"os"
	"os/signal"
	"player/application/service"
	grpchandler "player/interfaces/grpc"
	"player/pb"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

// Run 1.启用数据库。2.启动 grpc 服务，优雅启停。 3.启用 Etcd。
func Run(ctx context.Context) error {
	// 1. 初始化 player 服务专用容器
	playerContainer := container.NewPlayerContainer()
	if playerContainer == nil {
		log.Fatal("player 容器初始化失败")
		return nil
	}
	defer playerContainer.Close()

	// 2. 创建 gRPC 服务器
	server := grpc.NewServer()

	// 3. 创建服务发现注册器
	registry := discovery.NewRegistry()

	// 4. 启动 gRPC 服务（异步）
	go func() {
		log.Info("启动 gRPC 服务、etcd 服务...")

		// 监听端口
		lis, err := net.Listen("tcp", config.Conf.GrpcConf.Addr)
		if err != nil {
			log.Fatal("gRPC 监听失败: %v", err)
		}

		// 注册到 etcd
		err = registry.Register(config.Conf.EtcdConf)
		if err != nil {
			log.Fatal("etcd 注册失败: %v", err)
		}

		// 注册 gRPC 服务
		log.Info("注册 UserService...")
		authService := service.NewAuthService(
			playerContainer.GetUserRepository(),
			playerContainer.GetRedis(),
		)
		authHandler := grpchandler.NewAuthHandler(authService)
		pb.RegisterUserServiceServer(server, authHandler)

		// 启动服务
		if err := server.Serve(lis); err != nil {
			log.Fatal("gRPC 服务启动失败: %v", err)
		}
	}()

	// 5. 优雅关闭
	stop := func() {
		log.Info("正在关闭 player 服务...")
		time.Sleep(2 * time.Second)
		server.Stop()
		registry.Close()
		log.Info("player 服务已关闭")
	}

	// 6. 监听信号
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
				log.Info("收到中断信号，服务停止")
				return nil
			case syscall.SIGHUP:
				stop()
				log.Info("收到挂起信号，服务停止")
				return nil
			default:
				return nil
			}
		}
	}
}
