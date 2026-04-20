package realtime

import (
	"connector/domain/repository"
	"connector/infrastructure/database"
	"connector/infrastructure/log"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	connectorRouterPrefix = "user:router:connector:"
	gameRouterPrefix      = "user:router:game:"
)

type RedisUserRouterRepository struct {
	rdb *redis.Client
}

func NewRedisUserRouterRepository(redisManager *database.RedisManager) repository.UserRouterRepository {
	return &RedisUserRouterRepository{
		rdb: redisManager.Cli,
	}
}

func (r *RedisUserRouterRepository) SaveConnectorRouter(ctx context.Context, userID, connectorID string, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s", connectorRouterPrefix, userID)
	if err := r.rdb.Set(ctx, key, connectorID, ttl).Err(); err != nil {
		log.Error("SaveConnectorRouter 保存失败: userID=%s, err=%v", userID, err)
		return err
	}
	return nil
}

func (r *RedisUserRouterRepository) GetConnectorRouter(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("%s%s", connectorRouterPrefix, userID)
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (r *RedisUserRouterRepository) DeleteConnectorRouter(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", connectorRouterPrefix, userID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		log.Error("DeleteConnectorRouter 删除失败: userID=%s, err=%v", userID, err)
		return err
	}
	return nil
}

func (r *RedisUserRouterRepository) SaveGameRouter(ctx context.Context, userID, gameNodeID string, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s", gameRouterPrefix, userID)
	if err := r.rdb.Set(ctx, key, gameNodeID, ttl).Err(); err != nil {
		log.Error("SaveGameRouter 保存失败: userID=%s, err=%v", userID, err)
		return err
	}
	return nil
}

func (r *RedisUserRouterRepository) GetGameRouter(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("%s%s", gameRouterPrefix, userID)
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (r *RedisUserRouterRepository) DeleteGameRouter(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", gameRouterPrefix, userID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		log.Error("DeleteGameRouter 删除失败: userID=%s, err=%v", userID, err)
		return err
	}
	return nil
}

func (r *RedisUserRouterRepository) HasGameRouter(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("%s%s", gameRouterPrefix, userID)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
