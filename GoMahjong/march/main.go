package main

import (
	"common/config"
	"common/log"
	"common/metrics"
	"connector/app"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
	identifier string
)

var rootCmd = &cobra.Command{
	Use:   "march",
	Short: "march 匹配服务",
	Long:  `march 匹配服务`,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig(configFile)
		log.InitLog(identifier, logLevel)
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
	},
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "resource", "resource/application.yml", "resource file")
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
