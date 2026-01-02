package realtime

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/repository"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis Key 前缀
	userRouterKey = "march:router" // Hash: 用户路由 (userID -> JSON{gameTopic, connectorTopic})
)

// RedisUserRouterRepository Redis 实现的用户路由仓储
type RedisUserRouterRepository struct {
	redis *database.RedisManager
}

// NewRedisUserRouterRepository 创建 Redis 用户路由仓储
func NewRedisUserRouterRepository(redis *database.RedisManager) repository.UserRouterRepository {
	return &RedisUserRouterRepository{
		redis: redis,
	}
}

// getClient 获取 Redis 客户端（单机或集群）
func (r *RedisUserRouterRepository) getClient() redis.Cmdable {
	if r.redis.Cli != nil {
		return r.redis.Cli
	}
	return r.redis.ClusterCli
}

// SaveRouter 保存用户路由
func (r *RedisUserRouterRepository) SaveRouter(ctx context.Context, userID string, info *repository.UserRouterInfo, ttl time.Duration) error {
	cli := r.getClient()
	if cli == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	if info == nil {
		return fmt.Errorf("路由信息不能为空")
	}
	if info.GameTopic == "" {
		return fmt.Errorf("gameTopic 不能为空")
	}

	// 将路由信息序列化为 JSON
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("序列化路由信息失败: %w", err)
	}

	// 保存到 Hash
	key := fmt.Sprintf("%s:%s", userRouterKey, userID)
	err = cli.Set(ctx, key, string(data), ttl).Err()
	if err != nil {
		return fmt.Errorf("保存用户路由失败: %w", err)
	}

	log.Debug(fmt.Sprintf("保存用户路由: userID=%s, gameTopic=%s, connectorTopic=%s, ttl=%v",
		userID, info.GameTopic, info.ConnectorTopic, ttl))
	return nil
}

// GetRouter 获取用户路由
func (r *RedisUserRouterRepository) GetRouter(ctx context.Context, userID string) (*repository.UserRouterInfo, error) {
	cli := r.getClient()
	if cli == nil {
		return nil, fmt.Errorf("Redis 客户端未初始化")
	}

	key := fmt.Sprintf("%s:%s", userRouterKey, userID)
	data, err := cli.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repository.ErrRouterNotFound
		}
		return nil, fmt.Errorf("获取用户路由失败: %w", err)
	}

	// 反序列化
	var info repository.UserRouterInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("反序列化路由信息失败: %w", err)
	}

	return &info, nil
}

// DeleteRouter 删除用户路由
func (r *RedisUserRouterRepository) DeleteRouter(ctx context.Context, userID string) error {
	cli := r.getClient()
	if cli == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	key := fmt.Sprintf("%s:%s", userRouterKey, userID)
	err := cli.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除用户路由失败: %w", err)
	}

	log.Debug(fmt.Sprintf("删除用户路由: userID=%s", userID))
	return nil
}

// DeleteRouters 批量删除用户路由
func (r *RedisUserRouterRepository) DeleteRouters(ctx context.Context, userIDs []string) error {
	cli := r.getClient()
	if cli == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	if len(userIDs) == 0 {
		return nil
	}

	// 构建 keys
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, fmt.Sprintf("%s:%s", userRouterKey, userID))
	}

	// 批量删除
	err := cli.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("批量删除用户路由失败: %w", err)
	}

	log.Debug(fmt.Sprintf("批量删除用户路由: 共 %d 个", len(userIDs)))
	return nil
}

// ExistsRouter 检查用户路由是否存在
func (r *RedisUserRouterRepository) ExistsRouter(ctx context.Context, userID string) (bool, error) {
	cli := r.getClient()
	if cli == nil {
		return false, fmt.Errorf("Redis 客户端未初始化")
	}

	key := fmt.Sprintf("%s:%s", userRouterKey, userID)
	count, err := cli.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("检查用户路由失败: %w", err)
	}

	return count > 0, nil
}
