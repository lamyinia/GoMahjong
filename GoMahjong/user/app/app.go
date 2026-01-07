package app

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/container"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user/application/service"
	provider "user/interfaces/grpc"
	"user/pb"
)

func Run(ctx context.Context) error {
	playerContainer := container.NewUserContainer()
	if playerContainer == nil {
		log.Fatal("user 容器初始化失败")
		return nil
	}
	defer playerContainer.Close()

	server := grpc.NewServer()
	registry := discovery.NewRegistry()

	go func() {
		log.Info("启动 gRPC 服务、etcd 服务...")

		// 监听端口
		lis, err := net.Listen("tcp", config.UserNodeConfig.EtcdConf.Register.Addr)
		if err != nil {
			log.Fatal("gRPC 监听失败: %v", err)
		}

		// 注册到 etcd
		err = registry.Register(config.UserNodeConfig.EtcdConf, config.UserNodeConfig.ID)
		if err != nil {
			log.Fatal("etcd 注册失败: %v", err)
		}

		// 注册 gRPC 服务
		log.Info("注册 UserService...")
		authService := service.NewAuthService(
			playerContainer.GetUserRepository(),
			playerContainer.GetRedis(),
		)
		authHandler := provider.NewAuthProvider(authService)
		pb.RegisterUserServiceServer(server, authHandler)

		// 启动服务
		if err := server.Serve(lis); err != nil {
			log.Fatal("gRPC 服务启动失败: %v", err)
		}
	}()

	// 5. 优雅关闭
	stop := func() {
		log.Info("正在关闭 user 服务...")
		time.Sleep(2 * time.Second)
		server.Stop()
		registry.Close()
		log.Info("user 服务已关闭")
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
