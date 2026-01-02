package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"fmt"
	"march/app"
	"os"

	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "march",
	Short: "march 匹配服务",
	Long:  `march 匹配服务`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Load(configFile); err != nil {
			log.Fatal("文件配置发生错误：%v", err)
		}
		log.InitLog(config.MarchNodeConfig.ID, config.MarchNodeConfig.LogConf.Level)
		log.Info(fmt.Sprintf("配置文件: %+v", config.MarchNodeConfig))

		go func() {
			log.Info("启动监控..., URL: http://localhost:" + fmt.Sprintf("%d", config.MarchNodeConfig.MetricPort) + "/debug/statsviz/")
			err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.MarchNodeConfig.MetricPort))
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
	rootCmd.Flags().StringVar(&configFile, "configFile", "", "resource file")
	rootCmd.MarkFlagRequired("configFile")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error("error happen: %#v", err)
		os.Exit(1)
	}
}
