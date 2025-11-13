package app

import (
	"common/log"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(ctx context.Context) error {

	go func() {

	}()

	stop := func() {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
