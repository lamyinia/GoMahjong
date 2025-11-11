package database

import (
	"common/config"
	"common/log"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisManager struct {
	Cli        *redis.Client
	ClusterCli *redis.ClusterClient
}

func NewRedis() *RedisManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var clusterCli *redis.ClusterClient
	var cli *redis.Client
	redisConf := config.Conf.DatabaseConf.RedisConf

	// 构建Redis地址
	var addr string
	if redisConf.Addr != "" {
		addr = redisConf.Addr
	} else if redisConf.Host != "" && redisConf.Port > 0 {
		addr = fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port)
	} else {
		addr = "localhost:6379" // 默认地址
	}

	if len(redisConf.ClusterAddrs) == 0 {
		cli = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     redisConf.Password, // 如果没有密码，这个字段为空字符串，Redis会忽略
			PoolSize:     redisConf.PoolSize,
			MinIdleConns: redisConf.MinIdleConns,
		})
	} else {
		clusterCli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        redisConf.ClusterAddrs,
			Password:     redisConf.Password, // 如果没有密码，这个字段为空字符串，Redis会忽略
			PoolSize:     redisConf.PoolSize,
			MinIdleConns: redisConf.MinIdleConns,
		})
	}
	if cli != nil {
		if err := cli.Ping(ctx).Err(); err != nil {
			log.Fatal("redis 连接错误: %v", err)
			return nil
		}
	}
	if clusterCli != nil {
		if err := clusterCli.Ping(ctx).Err(); err != nil {
			log.Fatal("redisCluster 连接错误: %v", err)
			return nil
		}
	}

	return &RedisManager{
		Cli:        cli,
		ClusterCli: clusterCli,
	}
}

func (r *RedisManager) Close() {
	if r.Cli != nil {
		if err := r.Cli.Close(); err != nil {
			log.Error("redis 关闭出错: %v", err)
		}
	}
	if r.ClusterCli != nil {
		if err := r.ClusterCli.Close(); err != nil {
			log.Error("redisCluster 关闭出错: %v", err)
		}
	}
}

func (r *RedisManager) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	if r.Cli != nil {
		return r.Cli.Set(ctx, key, value, expiration).Err()
	}
	if r.ClusterCli != nil {
		return r.ClusterCli.Set(ctx, key, value, expiration).Err()
	}
	return nil
}
