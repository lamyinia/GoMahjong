package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"fmt"
	"hall/app"
	"os"

	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "hall",
	Short: "hall 大厅相关的处理",
	Long:  `hall 大厅相关的处理`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Load(configFile); err != nil {
			log.Fatal("文件配置发生错误：%v", err)
		}
		log.InitLog(config.HallNodeConfig.ID, config.HallNodeConfig.LogConf.Level)
		log.Info(fmt.Sprintf("配置文件: %+v", config.HallNodeConfig))

		go func() {
			log.Info("启动监控..., URL: http://localhost:" + fmt.Sprintf("%d", config.HallNodeConfig.MetricPort) + "/debug/statsviz/")
			err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.HallNodeConfig.MetricPort))
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
	rootCmd.Flags().StringVar(&configFile, "configFile", "resource/application.yml", "resource file")
	rootCmd.MarkFlagRequired("configFile")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error("error happen: %#v", err)
		os.Exit(1)
	}
}
