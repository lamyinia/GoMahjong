package app

import (
	"common/config"
	"common/http"
	"common/log"
	"context"
	"fmt"
	"gate/api"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(ctx context.Context) error {

	// 使用 common 封装的 gin 库 http-server
	server := http.NewHttpServer(
		http.WithPort(config.Conf.HttpPort),
		http.WithMode(config.Conf.Log.Level),
	)

	// 中间处理器注册
	server.Use(
	// http.CorsMiddleware(),
	// http.LoggerMiddleware(),
	// http.RecoveryMiddleware(),
	// http.RequestIDMiddleware(),
	)

	// 路由注册
	api.RegisterRoutes(server)

	go func() {
		log.Info(fmt.Sprintf("启动 HTTP 服务器，端口: %d", config.Conf.HttpPort))
		if err := server.Start(); err != nil {
			// http.ErrServerClosed 是正常关闭，不需要记录为错误
			if err.Error() != "http: Server closed" {
				log.Fatal(fmt.Sprintf("HTTP 服务器启动失败: %v", err))
			}
		}
	}()

	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error(fmt.Sprintf("HTTP 服务器关闭失败: %v", err))
		} else {
			log.Info("HTTP 服务器已优雅关闭")
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
