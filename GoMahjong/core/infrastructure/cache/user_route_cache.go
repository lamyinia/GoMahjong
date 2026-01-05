package cache

import (
	"common/cache"
	"fmt"
	"time"
)

// GameRouteCache 处理 userID -> gameNodeID 的映射
type GameRouteCache struct {
	cache    *cache.GeneralCache
	routeKey string
}

func NewGameRouteCache() (*GameRouteCache, error) {
	generalCache, err := cache.NewGeneralCache(int64(1<<27), 2*time.Hour) // ttl 过期两小时消失，最大内存大概 135mb
	if err != nil {
		return nil, fmt.Errorf("创建用户路由缓存失败: %w", err)
	}

	return &GameRouteCache{cache: generalCache, routeKey: "user:route"}, nil
}

func (c *GameRouteCache) Set(userID, gameNodeID string) bool {
	if userID == "" || gameNodeID == "" {
		return false
	}
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	return c.cache.SetWithTTL(key, gameNodeID, 2*time.Hour)
}

func (c *GameRouteCache) Get(userID string) (string, bool) {
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	return c.cache.GetString(key)
}

func (c *GameRouteCache) Delete(userID string) {
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	c.cache.Delete(key)
}

func (c *GameRouteCache) DeleteBatch(userIDs []string) {
	for _, userID := range userIDs {
		key := fmt.Sprintf("%s:%s", c.routeKey, userID)
		c.cache.Delete(key)
	}
}

func (c *GameRouteCache) Close() {
	c.cache.Close()
}
