package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"fmt"
	"os"
	"player/app"

	"github.com/spf13/cobra"
)

// 加载配置 -> 启动监控 -> 启动 grpc 服务
// 阿里云代理 go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
// 查看兼容的版本 go list -m -versions github.com/arl/statsviz

var (
	configFile string
	logLevel   string
	identifier string
)

var rootCmd = &cobra.Command{
	Use:   "player",
	Short: "player 玩家服务",
	Long:  `player 玩家服务`,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitFixedConfig(configFile)
		log.InitLog(identifier, logLevel)
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
	},
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "resource/application.yml", "resource file")
	rootCmd.Flags().StringVar(&logLevel, "logLevel", "info", "log level: debug, info, warn, error")
	rootCmd.Flags().StringVar(&identifier, "identifier", "", "subscribed topic and identifier of server required")
	rootCmd.MarkFlagRequired("identifier")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error("error happen: %#v", err)
		os.Exit(1)
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
