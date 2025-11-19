package config

import (
	"common/log"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
)

const (
	gameConfig = "gameConfig.json"
	servers    = "servers.json"
)

var InjectedConfig *GlobalConfig

type GameConfigValue map[string]any

type GlobalConfig struct {
	GameConfig map[string]GameConfigValue `json:"gameConfig"`
	configs    Configs                    `json:"serversConf"`
}

type Configs struct {
	Nats      NatsConfig         `json:"nats" `
	Connector []*ConnectorConfig `json:"connector" `
	Servers   []*ServersConfig   `json:"servers" `
	ConfigMap map[string][]*ServersConfig
}

type ServersConfig struct {
	ID               string `json:"id"`
	UniqueTopic      string `json:"uniqueTopic"`
	HandleTimeout    string `json:"handleTimeout"`
	RPCTimeout       string `json:"RPCTimeout"`
	MaxRunRoutineNum int    `json:"maxRunRoutineNum"`
}

type ConnectorConfig struct {
	ID         string `json:"id" `
	Host       string `json:"host" `
	ClientPort int    `json:"clientPort" `
	Frontend   bool   `json:"frontend" `
	ServerType string `json:"serverType" `
}

type NatsConfig struct {
	URL string `json:"url" mapstructure:"url"`
}

func InitGlobalConfig() {
	configDir := "json"
	InjectedConfig = new(GlobalConfig)

	dir, err := os.ReadDir(configDir)
	if err != nil {
		log.Fatal("batch_config error happen %w", err)
	}

	for _, v := range dir {
		configFile := path.Join(configDir, v.Name())
		if v.Name() == gameConfig {
			readGameConfig(configFile)
		}
		if v.Name() == servers {
			readServersConfig(configFile)
		}
	}
}

func readServersConfig(configFile string) {
	var configs Configs
	v := viper.New()
	v.SetConfigFile(configFile)
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {
		log.Info("Configs 配置文件被修改")
		err := v.Unmarshal(&configs)
		if err != nil {
			panic(fmt.Errorf("configs 解析 json 错误, err:%v \n", err))
		}
		InjectedConfig.configs = configs
	})

	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("configs 读取配置文件出错,err:%v \n", err))
	}

	InjectedConfig.configs = configs

	if len(InjectedConfig.configs.Servers) > 0 {
		if InjectedConfig.configs.ConfigMap == nil {
			InjectedConfig.configs.ConfigMap = make(map[string][]*ServersConfig)
		}
		for _, v := range InjectedConfig.configs.Servers {
			if InjectedConfig.configs.ConfigMap[v.UniqueTopic] == nil {
				InjectedConfig.configs.ConfigMap[v.UniqueTopic] = make([]*ServersConfig, 0)
			}
			InjectedConfig.configs.ConfigMap[v.UniqueTopic] = append(InjectedConfig.configs.ConfigMap[v.UniqueTopic], v)
		}
	}
}

func readGameConfig(configFile string) {
	file, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var gc map[string]GameConfigValue
	err = json.Unmarshal(data, &gc)
	if err != nil {
		panic(err)
	}
	InjectedConfig.GameConfig = gc

	/*	gc := make(map[string]GameConfigValue)
		v := viper.New()
		v.SetConfigFile(configFile)
		v.WatchConfig()
		v.OnConfigChange(func(in fsnotify.Event) {
			log.Println("gameConfig配置文件被修改了")
			err := v.Unmarshal(&gc)
			if err != nil {
				panic(fmt.Errorf("gameConfig Unmarshal change config data,err:%v \n", err))
			}
			Conf.GameConfig = gc
		})
		err := v.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("gameConfig 读取配置文件出错,err:%v \n", err))
		}
		log.Println("%v", v.AllKeys())
		err = v.Unmarshal(&gc)
		if err != nil {
			panic(fmt.Errorf("gameConfig Unmarshal config data,err:%v \n", err))
		}
		Conf.GameConfig = gc*/
}
