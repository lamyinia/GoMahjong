package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"gate/app"
	"os"
)

var configFile = flag.String("config", "resource/application.yml", "config file")

func main() {
	flag.Parse()

	configPath := *configFile
	config.InitConfig(configPath)

	log.InitLog(config.Conf.AppName)
	log.Info(fmt.Sprintf("配置文件: %+v", config.Conf))

	go func() {
		log.Info("启动监控..., URL: http://localhost:" + fmt.Sprintf("%d", config.Conf.MetricPort) + "/debug/statsviz/")
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
