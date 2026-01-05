package realtime

import (
	"common/database"
	"context"
	"core/domain/repository"
	"time"
)

const (
	gameRouterKey      = "game:router"      // userID -> GameTopic
	connectorRouterKey = "connector:router" // userID -> ConnectorTopic
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

func (r *RedisUserRouterRepository) SaveGameRouter(ctx context.Context, userID, gameTopic string, ttl time.Duration) error {
	return r.redis.Set(ctx, gameRouterKey+":"+userID, gameTopic, ttl)
}

func (r *RedisUserRouterRepository) GetGameRouter(ctx context.Context, userID string) (string, error) {
	return r.redis.Get(ctx, gameRouterKey+":"+userID).Result()
}

func (r *RedisUserRouterRepository) DeleteGameRouter(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, gameRouterKey+":"+userID)
}

func (r *RedisUserRouterRepository) ExistsGameRouter(ctx context.Context, userID string) (bool, error) {
	count, err := r.redis.Exists(ctx, gameRouterKey+":"+userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RedisUserRouterRepository) DeleteGameRouters(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, gameRouterKey+":"+userID)
	}
	return r.redis.Del(ctx, keys...)
}

func (r *RedisUserRouterRepository) SaveConnectorRouter(ctx context.Context, userID, connectorTopic string, ttl time.Duration) error {
	return r.redis.Set(ctx, connectorRouterKey+":"+userID, connectorTopic, ttl)
}

func (r *RedisUserRouterRepository) GetConnectorRouter(ctx context.Context, userID string) (string, error) {
	return r.redis.Get(ctx, connectorRouterKey+":"+userID).Result()
}

func (r *RedisUserRouterRepository) DeleteConnectorRouter(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, connectorRouterKey+":"+userID)
}

func (r *RedisUserRouterRepository) DeleteConnectorRouters(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, connectorRouterKey+":"+userID)
	}
	return r.redis.Del(ctx, keys...)
}

func (r *RedisUserRouterRepository) ExistsConnectorRouter(ctx context.Context, userID string) (bool, error) {
	count, err := r.redis.Exists(ctx, connectorRouterKey+":"+userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
