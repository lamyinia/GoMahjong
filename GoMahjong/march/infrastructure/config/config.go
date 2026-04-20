package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var MarchNodeConfig MarchConfiguration

type BaseConfig struct {
	ID         string `mapstructure:"id"`
	ServerType string `mapstructure:"serverType"`
	MetricPort int    `mapstructure:"metricPort"`
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

type LogConf struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
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

type JwtConf struct {
	Secret        string `mapstructure:"secret"`
	Expire        int    `mapstructure:"expire"`
	AllowTestPath bool   `mapstructure:"allowTestPath"`
}

type NatsConfig struct {
	URL string `mapstructure:"url"`
}

type Domain struct {
	Name string `mapstructure:"name"`
	Addr string `mapstructure:"addr"`
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

type MatchMode string
type MatchStrategy string

const (
	ModeRank4   MatchMode = "classic:rank4"
	ModeCasual4 MatchMode = "classic:casual4"
	ModeCasual3 MatchMode = "classic:casual3"

	ScorePoll MatchStrategy = "classic:poll"
)

type MarchPoolConfig struct {
	PoolID    MatchMode     `mapstructure:"poolID"`
	Strategy  MatchStrategy `mapstructure:"strategy"`
	BatchSize int           `mapstructure:"batchSize"`
	Internal  int64         `mapstructure:"internal"`
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

	var cfg MarchConfiguration
	if err := v.Unmarshal(&cfg); err != nil {
		return err
	}
	cfg.ID = base.ID
	MarchNodeConfig = cfg

	return nil
}
