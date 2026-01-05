package database

import (
	"common/config"
	"common/log"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisManager struct {
	Cli        *redis.Client
	ClusterCli *redis.ClusterClient
	scriptSHAs map[string]string
	mu         sync.RWMutex
}

func NewRedis(redisConf config.RedisConf) *RedisManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var clusterCli *redis.ClusterClient
	var cli *redis.Client

	// 构建Redis地址
	var addr string
	if redisConf.Addr != "" {
		addr = redisConf.Addr
	} else if redisConf.Host != "" && redisConf.Port > 0 {
		addr = fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port)
	} else {
		panic("redis 配置出错")
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
		scriptSHAs: make(map[string]string),
	}
}

func (r *RedisManager) GetClient() (redis.Cmdable, error) {
	if r.Cli != nil {
		return r.Cli, nil
	}
	if r.ClusterCli != nil {
		return r.ClusterCli, nil
	}
	return nil, fmt.Errorf("redis 客户端未初始化")
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

func (r *RedisManager) Get(ctx context.Context, key string) *redis.StringCmd {
	if r.Cli != nil {
		return r.Cli.Get(ctx, key)
	}
	if r.ClusterCli != nil {
		return r.ClusterCli.Get(ctx, key)
	}
	return nil
}

func (r *RedisManager) Del(ctx context.Context, keys ...string) error {
	if r.Cli != nil {
		return r.Cli.Del(ctx, keys...).Err()
	}
	if r.ClusterCli != nil {
		return r.ClusterCli.Del(ctx, keys...).Err()
	}
	return nil
}

func (r *RedisManager) Exists(ctx context.Context, key ...string) (int64, error) {
	if r.Cli != nil {
		return r.Cli.Exists(ctx, key...).Result()
	}
	if r.ClusterCli != nil {
		return r.ClusterCli.Exists(ctx, key...).Result()
	}
	return 0, nil
}

func (r *RedisManager) Incr(ctx context.Context, key string) (int64, error) {
	if r.Cli != nil {
		return r.Cli.Incr(ctx, key).Result()
	}
	if r.ClusterCli != nil {
		return r.ClusterCli.Incr(ctx, key).Result()
	}
	return 0, nil
}

func (r *RedisManager) EvalScript(ctx context.Context, scriptName, script string, keys []string, args ...any) (any, error) {
	cli, err := r.GetClient()
	if err != nil {
		return nil, err
	}

	if r.Cli != nil && scriptName != "" {
		r.mu.RLock()
		sha, exists := r.scriptSHAs[scriptName]
		r.mu.RUnlock()
		if exists {
			result, err := r.Cli.EvalSha(ctx, sha, keys, args...).Result()
			if err != nil {
				// SHA 失效，重新加载
				if err.Error() == "NOSCRIPT No matching script. Use EVAL." {
					newSHA, loadErr := r.Cli.ScriptLoad(ctx, script).Result()
					if loadErr != nil {
						return nil, fmt.Errorf("重新加载脚本失败: %w", loadErr)
					}
					r.mu.Lock()
					r.scriptSHAs[scriptName] = newSHA
					r.mu.Unlock()
					return r.Cli.EvalSha(ctx, newSHA, keys, args...).Result()
				}
				return nil, err
			}
			return result, nil
		}
		sha, err := r.Cli.ScriptLoad(ctx, script).Result()
		if err != nil {
			return nil, fmt.Errorf("加载脚本失败: %w", err)
		}
		r.mu.Lock()
		r.scriptSHAs[scriptName] = sha
		r.mu.Unlock()
		return r.Cli.EvalSha(ctx, sha, keys, args...).Result()
	}

	return cli.Eval(ctx, script, keys, args...).Result()
}

func (r *RedisManager) Close() error {
	if r.Cli == nil && r.ClusterCli == nil {
		return nil
	}
	if r.Cli != nil {
		if err := r.Cli.Close(); err != nil {
			log.Error("redis 关闭出错: %v", err)
			return err
		}
	}
	if r.ClusterCli != nil {
		if err := r.ClusterCli.Close(); err != nil {
			log.Error("redisCluster 关闭出错: %v", err)
			return err
		}
	}
	return nil
}
