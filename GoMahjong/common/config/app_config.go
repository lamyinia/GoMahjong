package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type ConfigIface interface {
	CallID() string
	CallNodeType() string
}

func (cfg *BaseConfig) CallID() string {
	return cfg.ID
}

func (cfg *BaseConfig) CallNodeType() string {
	return cfg.ServerType
}

var ConnectorConfig ConnectorConfiguration
var GameNodeConfig GameConfiguration
var GateNodeConfig GateConfiguration
var HallNodeConfig HallConfiguration
var MarchNodeConfig MarchConfiguration
var UserNodeConfig UserConfiguration

type BaseConfig struct {
	ID         string `mapstructure:"id"`
	ServerType string `mapstructure:"serverType"`
	MetricPort int    `mapstructure:"metricPort"`
}

type ConnectorConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type GameConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type GateConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
	HttpPort     int               `mapstructure:"httpPort"`
}

type HallConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type MarchConfiguration struct {
	BaseConfig       `mapstructure:",squash"`
	DatabaseConf     `mapstructure:"database"`
	JwtConf          `mapstructure:"jwt"`
	EtcdConf         `mapstructure:"etcd"`
	LogConf          `mapstructure:"log"`
	NatsConfig       `mapstructure:"nats"`
	MarchPoolConfigs []MarchPoolConfig `mapstructure:"marchPool"`
	Domains          map[string]Domain `mapstructure:"domain"`
}

type UserConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type LogConf struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type GrpcConf struct {
	Addr string `mapstructure:"addr"`
}

type EtcdConf struct {
	Addrs       []string       `mapstructure:"addrs"`
	RWTimeout   int            `mapstructure:"rwTimeout"`
	DialTimeout int            `mapstructure:"dialTimeout"`
	Register    RegisterServer `mapstructure:"register"`
}

type RegisterServer struct {
	Addr    string `mapstructure:"addr"`
	Domain  string `mapstructure:"domain"`
	Version string `mapstructure:"version"`
	Weight  int    `mapstructure:"weight"`
	Ttl     int    `mapstructure:"ttl"`
}

type ServiceConf struct {
	ID         string `mapstructure:"id"`
	ClientHost string `mapstructure:"clientHost"`
	ClientPort int    `mapstructure:"clientPort"`
}

type Domain struct {
	Name        string `mapstructure:"name"`
	LoadBalance bool   `mapstructure:"loadBalance"`
}

type JwtConf struct {
	Secret        string `mapstructure:"secret"`
	Expire        int    `mapstructure:"expire"`
	AllowTestPath bool   `mapstructure:"allowTestPath"`
}

type DatabaseConf struct {
	MongoConf MongoConf `mapstructure:"mongo"`
	RedisConf RedisConf `mapstructure:"redis"`
}

type MongoConf struct {
	Url         string `mapstructure:"url"`
	Db          string `mapstructure:"db"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	MinPoolSize int    `mapstructure:"minPoolSize"`
	MaxPoolSize int    `mapstructure:"maxPoolSize"`
}

type RedisConf struct {
	Addr         string   `mapstructure:"addr"`
	ClusterAddrs []string `mapstructure:"clusterAddrs"`
	Password     string   `mapstructure:"password"`
	PoolSize     int      `mapstructure:"poolSize"`
	MinIdleConns int      `mapstructure:"minIdleConns"`
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`
}

type NatsConfig struct {
	URL string `json:"url" mapstructure:"url"`
}

type MatchMode string
type MatchStrategy string

const (
	ModeRank4   MatchMode = "classic:rank4"
	ModeCasual4 MatchMode = "classic:casual4"
	ModeCasual3 MatchMode = "classic:casual3"

	ScorePoll MatchStrategy = "classic:poll" // zset 控制 + 先来先服务
)

type MarchPoolConfig struct {
	PoolID    MatchMode     `mapstructure:"poolID"` // 池 ID
	Strategy  MatchStrategy `mapstructure:"strategy"`
	BatchSize int           `mapstructure:"batchSize"`
	Internal  int64         `mapstructure:"internal"` // 单位是毫秒
}

func InitFixedConfig(configFile string) {

	v := viper.New()
	v.SetConfigFile(configFile)
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {

	})
}

func Load(configFile string) error {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	var base BaseConfig
	if err := v.Unmarshal(&base); err != nil {
		return err
	}
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		base.ID = nodeID
	} else {
		return fmt.Errorf("NODE_ID environment variable is required")
	}

	switch base.ServerType {
	case "connector":
		var cfg ConnectorConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		ConnectorConfig = cfg
	case "game":
		var cfg GameConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		GameNodeConfig = cfg
	case "gate":
		var cfg GateConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		GateNodeConfig = cfg
	case "hall":
		var cfg HallConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		HallNodeConfig = cfg
	case "march":
		var cfg MarchConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		MarchNodeConfig = cfg
	case "user":
		var cfg UserConfiguration
		if err := v.Unmarshal(&cfg); err != nil {
			return err
		}
		cfg.ID = base.ID
		UserNodeConfig = cfg
	default:
		return fmt.Errorf("unknown server type: %s", base.ServerType)
	}

	return nil
}
