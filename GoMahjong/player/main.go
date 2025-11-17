package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"os"
	"player/app"
)

// 加载配置 -> 启动监控 -> 启动 grpc 服务
// 阿里云代理 go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
// 查看兼容的版本 go list -m -versions github.com/arl/statsviz
var configFile = flag.String("config", "resource/application.yml", "resource file")

func main() {
	flag.Parse()

	configPath := *configFile
	config.InitConfig(configPath)

	log.InitLog(config.Conf.AppName)
	log.Info("配置文件: %+v", config.Conf)

	go func() {
		log.Info("启动监控..., URL: http://localhost:%d/debug/statsviz/", config.Conf.MetricPort)
		err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()

	err := app.Run(context.Background())
	if err != nil {
		log.Error("发生异常: {}", err)
		os.Exit(-1)
	}
}

/*
	go get github.com/arl/statsviz@latest
	go get google.golang.org/grpc@1.73.1
	go get github.com/charmbracelet/log@latest
	go get go.etcd.io/etcd/client/v3
	go get go.mongodb.org/mongo-driver/mongo
 	go get github.com/redis/go-redis/v9
*/
