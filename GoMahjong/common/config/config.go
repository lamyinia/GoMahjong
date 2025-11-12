package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf *Config

type Config struct {
	AppName      string                 `mapstructure:"appName"`
	Log          LogConf                `mapstructure:"log"`
	HttpPort     int                    `mapstructure:"httpPort"`
	WsPort       int                    `mapstructure:"wsPort"`
	MetricPort   int                    `mapstructure:"metricPort"`
	GrpcConf     GrpcConf               `mapstructure:"grpc"`
	EtcdConf     EtcdConf               `mapstructure:"etcd"`
	ServiceConf  ServiceConf            `mapstructure:"service"`
	JwtConf      JwtConf                `mapstructure:"jwt"`
	DatabaseConf DatabaseConf           `mapstructure:"database"`
	Domain       map[string]Domain      `mapstructure:"domain"`
	Services     map[string]ServiceConf `mapstructure:"services"`
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
	Name    string `mapstructure:"name"`
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
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
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

func InitConfig(configFile string) {
	Conf = new(Config)
	v := viper.New()
	v.SetConfigFile(configFile)
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {
		err := v.Unmarshal(&Conf)
		if err != nil {
			panic(fmt.Errorf("解析配置文件出错 2, err:%v", err))
		}
	})

	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("读取配置文件出错, err:%v", err))
	}

	err = v.Unmarshal(&Conf)
	if err != nil {
		panic(fmt.Errorf("解析配置文件出错 1, err:%v", err))
	}
}
