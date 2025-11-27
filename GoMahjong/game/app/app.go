package app

import (
	"common/config"
	"common/log"
	"context"
	"core/container"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(ctx context.Context) error {
	gameContainer := container.NewGameContainer()

	if gameContainer == nil {
		log.Fatal("game 容器初始化失败")
		return nil
	}
	defer func() {
		if err := gameContainer.Close(); err != nil {
			log.Error("关闭 game 容器失败: %v", err)
		}
	}()

	go func() {
		err := gameContainer.GameWorker.Start(
			ctx,
			config.InjectedConfig.Nats.URL,
			config.Conf.EtcdConf,
		)

		if err != nil {
			log.Fatal("worker 启动失败，err:%#v", err)
		}
	}()

	stop := func() {
		log.Info("正在关闭 game 服务...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			if err := gameContainer.Close(); err != nil {
				log.Warn("关闭 game 容器失败: %v", err)
			}
			close(done)
		}()

		select {
		case <-done:
			log.Info("game 服务已关闭")
		case <-shutdownCtx.Done():
			log.Warn("关闭 game 服务超时（5秒），defer 会确保资源最终被释放")
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
