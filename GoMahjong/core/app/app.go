package app

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"core/infrastructure"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 1.启动 grpc 服务，优雅启停。 2.启用日志。 3.启用 Etcd。 4.启用数据库。
func Run(ctx context.Context) error {
	server := grpc.NewServer()
	register := discovery.NewRegister()

	go func() {
		log.Info("启动 grpc 服务、etcd 服务、mongodb 服务、redis 服务...")
		lis, err := net.Listen("tcp", config.Conf.GrpcConf.Addr)
		if err != nil {
			log.Fatal("grpc 监听失败")
		}

		err = register.Register(config.Conf.EtcdConf)
		if err != nil {
			log.Fatal("etcd 启动失败")
		}

		manager := infrastructure.New()
		if manager != nil {
			log.Info("mongodb、redis 数据库服务启动成功")
		}

		err = server.Serve(lis)
		if err != nil {
			log.Fatal("grpc 服务启动失败")
		}
	}()

	stop := func() {
		log.Info("正在关闭 grpc 服务")
		time.Sleep(3 * time.Second)
		server.Stop()
		register.Close()
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
