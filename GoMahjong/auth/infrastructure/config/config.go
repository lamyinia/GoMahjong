package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var AuthNodeConfig AuthConfiguration

type BaseConfig struct {
	ID         string `mapstructure:"id"`
	ServerType string `mapstructure:"serverType"`
	MetricPort int    `mapstructure:"metricPort"`
}

type AuthConfiguration struct {
	BaseConfig   `mapstructure:",squash"`
	DatabaseConf `mapstructure:"database"`
	JwtConf      `mapstructure:"jwt"`
	EtcdConf     `mapstructure:"etcd"`
	LogConf      `mapstructure:"log"`
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

	var cfg AuthConfiguration
	if err := v.Unmarshal(&cfg); err != nil {
		return err
	}
	cfg.ID = base.ID
	AuthNodeConfig = cfg

	return nil
}
