package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"hall/app"
	"os"
)

var configFile = flag.String("resource", "resource/application.yml", "resource file")
var rootCmd = &cobra.Command{
	Use:     "hall",
	Short:   "hall 大厅相关的处理",
	Long:    `hall 大厅相关的处理`,
	Run:     func(cmd *cobra.Command, args []string) {},
	PostRun: func(cmd *cobra.Command, args []string) {},
}
var (
	batchServerConfigFile string
	batchGameConfigFile   string
	uniqueTopic           string
)

func init() {
	rootCmd.Flags().StringVar(&uniqueTopic, "uniqueTopic", "", "subscribed topic and identifier of server required")
	_ = rootCmd.MarkFlagRequired("uniqueTopic")
}

func main() {
	log.InitLog(config.Conf.AppName)

	if err := rootCmd.Execute(); err != nil {
		log.Error("error happen: %#v", err)
		os.Exit(1)
	}

	configPath := *configFile
	config.InitConfig(configPath)

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
