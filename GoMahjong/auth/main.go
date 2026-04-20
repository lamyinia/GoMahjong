package main

import (
	"auth/app"
	"auth/infrastructure/config"
	"auth/infrastructure/log"
	"auth/infrastructure/metrics"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "auth",
	Short: "auth 认证服务",
	Long:  `auth 认证服务`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Load(configFile); err != nil {
			fmt.Println("文件配置发生错误：%v", err)
		}
		log.InitLog(config.AuthNodeConfig.ID, config.AuthNodeConfig.LogConf.Level)
		log.Info(fmt.Sprintf("配置文件: %+v", config.AuthNodeConfig))
		go func() {
			log.Info("启动监控..., URL: http://localhost:%d/debug/statsviz/", config.AuthNodeConfig.MetricPort)
			err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.AuthNodeConfig.MetricPort))
			if err != nil {
				panic(err)
			}
		}()
		err := app.Run(context.Background())
		if err != nil {
			log.Error("发生异常: %v", err)
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
