package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type BaseConfig interface {
	CallID() string
	CallNodeType() string
}

func (cfg *AConfig) CallID() string {
	return cfg.ID
}

func (cfg *AConfig) CallNodeType() string {
	return cfg.ServerType
}

var ConnectorConfig ConnectorConfiguration
var GameNodeConfig GameConfiguration
var GateNodeConfig GateConfiguration
var HallNodeConfig HallConfiguration
var MarchNodeConfig MarchConfiguration
var UserNodeConfig UserConfiguration

type AConfig struct {
	ID         string `mapstructure:"id"`
	ServerType string `mapstructure:"serverType"`
	MetricPort int    `mapstructure:"metricPort"`
}

type ConnectorConfiguration struct {
	AConfig      `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type GameConfiguration struct {
	AConfig      `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type GateConfiguration struct {
	AConfig      `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
	HttpPort     int               `mapstructure:"httpPort"`
}

type HallConfiguration struct {
	AConfig      `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type MarchConfiguration struct {
	AConfig      `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
	NatsConfig   `mapstructure:"nats"`
	Domains      map[string]Domain `mapstructure:"domain"`
}

type UserConfiguration struct {
	AConfig      `mapstructure:",squash"`
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

	var base AConfig
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
