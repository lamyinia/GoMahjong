package app

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/container"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "march/interfaces/grpc"
	"march/pb"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	marchContainer := container.NewMarchContainer()
	if marchContainer == nil {
		log.Fatal("march 容器初始化失败")
		return nil
	}

	// 创建 grpc 注册依赖 -> 拿到监听端口 -> 注册 etcd -> 监听 grpc
	grpcSrv := grpc.NewServer()
	matchProvider := grpcserver.NewMatchProvider(marchContainer.MatchService)
	pb.RegisterMatchServiceServer(grpcSrv, matchProvider)

	var (
		registry *discovery.Registry
		lis      net.Listener
		err      error
	)

	lis, err = net.Listen("tcp", config.MarchNodeConfig.EtcdConf.Register.Addr)
	if err != nil {
		log.Fatal("监听 gRPC 地址失败: %v", err)
		return err
	}

	registry = discovery.NewRegistry()
	if err = registry.Register(config.MarchNodeConfig.EtcdConf, marchContainer.NodeID); err != nil {
		log.Fatal("march 注册 etcd 失败: %v", err)
		return err
	}

	go func() {
		log.Info(fmt.Sprintf("march gRPC 服务启动, addr=%s", config.MarchNodeConfig.EtcdConf.Register.Addr))
		if serveErr := grpcSrv.Serve(lis); serveErr != nil {
			log.Error(fmt.Sprintf("march gRPC 服务退出: %v", serveErr))
		}
	}()

	go func() {
		err := marchContainer.MarchWorker.Start(ctx, config.MarchNodeConfig.NatsConfig.URL)
		if err != nil {
			log.Fatal("match 服务启动失败，err:%#v", err)
		}
	}()

	stop := func() {
		log.Info("正在关闭 march 服务...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done := make(chan struct{})
		go func() {
			if grpcSrv != nil {
				log.Info("关闭 gRPC 服务...")
				grpcSrv.GracefulStop()
			}
			if lis != nil {
				_ = lis.Close()
			}
			if registry != nil {
				log.Info("注销 etcd 服务...")
				registry.Close()
			}
			if err := marchContainer.Close(); err != nil {
				log.Warn("关闭 march 容器失败: %v", err)
			}
			close(done)
		}()

		select {
		case <-done:
			log.Info("march 服务已关闭")
		case <-shutdownCtx.Done():
			log.Warn("关闭 march 服务超时（5秒），defer 会确保资源最终被释放")
		}
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
