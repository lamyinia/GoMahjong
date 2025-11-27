package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	servers = "servers.json"
)

type NodeConfig interface {
	Inherit()
	GetID() string
	GetServerType() string
}

var InjectedConfig *Configs

// BaseNodeConfig 所有节点配置的基类，包含通用字段
type BaseNodeConfig struct {
	ID               string `json:"id"`
	ServerType       string `json:"serverType"`
	HandleTimeout    int    `json:"handleTimeOut"` // 统一为 int 类型
	RPCTimeout       int    `json:"rpcTimeOut"`    // 统一为 int 类型
	MaxRunRoutineNum int    `json:"maxRunRoutineNum"`
}

func (b *BaseNodeConfig) Inherit() {
	// 可以在这里设置默认值或验证
	if b.HandleTimeout <= 0 {
		b.HandleTimeout = 10 // 默认值
	}
	if b.RPCTimeout <= 0 {
		b.RPCTimeout = 5 // 默认值
	}
	if b.MaxRunRoutineNum <= 0 {
		b.MaxRunRoutineNum = 10240 // 默认值
	}
}

func (b *BaseNodeConfig) GetID() string {
	return b.ID
}

func (b *BaseNodeConfig) GetServerType() string {
	return b.ServerType
}

type MarchConfig struct {
	BaseNodeConfig
}

type GameConfig struct {
	BaseNodeConfig
}

type ConnectorConfig struct {
	BaseNodeConfig
	Host       string `json:"host"`
	ClientPort int    `json:"clientPort"`
	Frontend   bool   `json:"frontend"`
	HeartTime  int    `json:"heartTime"`
}

type Configs struct {
	Nats           NatsConfig   `json:"nats"`
	ClusterConfigs []NodeConfig `json:"-"` // 不直接从 JSON 解析，通过解析逻辑填充
	LocalConfig    any          `json:"-"` // 当前节点的配置，通过 SetLocalConfig 设置
}

type NatsConfig struct {
	URL string `json:"url" mapstructure:"url"`
}

func InitDynamicConfig(nodeID string) {

	possiblePaths := []string{
		"common/config/json",    // 从项目根目录运行
		"../common/config/json", // 从子目录（如 march/）运行
		"./common/config/json",  // 明确相对路径
		"config/json",           // 如果 common 在 GOPATH 中
	}

	var configDir string
	var err error
	for _, p := range possiblePaths {
		dir, err := os.ReadDir(p)
		if err == nil {
			found := false
			for _, v := range dir {
				if v.Name() == servers {
					found = true
					break
				}
			}
			if found {
				configDir = p
				break
			}
		}
	}

	if configDir == "" {
		cwd, _ := os.Getwd()
		log.Fatal("无法找到配置文件目录，当前工作目录: %s，尝试过的路径: %v", cwd, possiblePaths)
		return
	}

	dir, err := os.ReadDir(configDir)
	if err != nil {
		log.Fatal("读取配置目录失败: %w", err)
	}

	for _, v := range dir {
		configFile := filepath.Join(configDir, v.Name())
		if v.Name() == servers {
			readServersConfig(configFile)
		}
	}

	if err := InjectedConfig.SetLocalConfig(nodeID); err != nil {
		log.Fatal(fmt.Sprintf("设置本地配置失败: %v", err))
		os.Exit(-1)
	}
}

type serversConfigRaw struct {
	Nats    NatsConfig               `json:"nats"`
	Servers []map[string]interface{} `json:"servers"`
}

func readServersConfig(configFile string) {
	// 直接读取 JSON 文件，避免 viper 的键名转换问题
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(fmt.Errorf("读取配置文件失败,err:%v", err))
	}

	var raw serversConfigRaw
	err = json.Unmarshal(data, &raw)
	if err != nil {
		panic(fmt.Errorf("configs 解析 json 错误, err:%v", err))
	}

	parseAndSetConfigs(&raw)

	// 设置文件监听（如果需要热更新）
	v := viper.New()
	v.SetConfigFile(configFile)
	err = v.ReadInConfig() // 需要先读取才能监听
	if err != nil {
		log.Warn("初始化配置文件监听失败", "error", err)
	} else {
		v.WatchConfig()
		v.OnConfigChange(func(in fsnotify.Event) {
			fmt.Println("Configs 配置文件被修改")
			data, err := os.ReadFile(configFile)
			if err != nil {
				log.Warn("重新读取配置文件失败", "error", err)
				return
			}
			var newRaw serversConfigRaw
			if err := json.Unmarshal(data, &newRaw); err != nil {
				log.Warn("重新解析配置文件失败", "error", err)
				return
			}
			parseAndSetConfigs(&newRaw)
		})
	}
}

func parseAndSetConfigs(raw *serversConfigRaw) {
	configs := Configs{
		Nats:           raw.Nats,
		ClusterConfigs: make([]NodeConfig, 0),
		LocalConfig:    nil,
	}

	// 解析 servers 配置
	for _, item := range raw.Servers {
		serverTypeVal, exists := item["serverType"]
		if !exists {
			log.Warn("配置项缺少 serverType 字段", "item", item)
			continue
		}
		serverType, ok := serverTypeVal.(string)
		if !ok {
			log.Warn("serverType 类型错误，期望 string", "serverType", serverTypeVal)
			continue
		}
		var node NodeConfig
		switch serverType {
		case "game":
			game := &GameConfig{}
			if err := parseNodeConfig(item, game); err != nil {
				log.Warn("解析 Game 配置失败", "error", err)
				continue
			}
			game.Inherit()
			node = game
		case "march":
			march := &MarchConfig{}
			if err := parseNodeConfig(item, march); err != nil {
				log.Warn("解析 March 配置失败", "error", err)
				continue
			}
			march.Inherit()
			node = march
		case "connector":
			connector := &ConnectorConfig{}
			if err := parseNodeConfig(item, connector); err != nil {
				log.Warn("解析 connector 配置失败", "error", err)
				continue
			}
			connector.Inherit()
			node = connector
		default:
			log.Warn("未知的 serverType", "serverType", serverType)
			continue
		}
		if node != nil {
			configs.ClusterConfigs = append(configs.ClusterConfigs, node)
		}
	}

	InjectedConfig = &configs
}

// parseNodeConfig 将 map 解析为具体的 NodeConfig 类型
func parseNodeConfig(data map[string]interface{}, target NodeConfig) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal 失败: %w", err)
	}
	return json.Unmarshal(jsonData, target)
}

// SetLocalConfig 根据 nodeID 从 clusterConfigs 中找到对应配置并设置为 localConfig
func (c *Configs) SetLocalConfig(nodeID string) error {
	for _, config := range c.ClusterConfigs {
		if config.GetID() == nodeID {
			c.LocalConfig = config
			return nil
		}
	}
	return fmt.Errorf("未找到 nodeID=%s 的配置", nodeID)
}

// GetMarchConfig 获取 MarchConfig，类型安全
func (c *Configs) GetMarchConfig() (*MarchConfig, error) {
	config, ok := c.LocalConfig.(*MarchConfig)
	if !ok {
		return nil, fmt.Errorf("localConfig 不是 MarchConfig 类型")
	}
	return config, nil
}

// GetGameConfig 获取 GameConfig，类型安全
func (c *Configs) GetGameConfig() (*GameConfig, error) {
	config, ok := c.LocalConfig.(*GameConfig)
	if !ok {
		return nil, fmt.Errorf("localConfig 不是 GameConfig 类型")
	}
	return config, nil
}

// GetConnectorConfig 获取 ConnectorConfig，类型安全
func (c *Configs) GetConnectorConfig() (*ConnectorConfig, error) {
	config, ok := c.LocalConfig.(*ConnectorConfig)
	if !ok {
		return nil, fmt.Errorf("localConfig 不是 ConnectorConfig 类型")
	}
	return config, nil
}
