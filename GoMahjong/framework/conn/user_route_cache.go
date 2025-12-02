package conn

import (
	"common/cache"
	"fmt"
	"time"
)

type UserRouteCache struct {
	cache    *cache.GeneralCache
	routeKey string
}

func NewUserRouteCache() (*UserRouteCache, error) {
	generalCache, err := cache.NewGeneralCache(int64(1<<27), 2*time.Hour) // ttl 过期两消失，最大内存大概 135mb
	if err != nil {
		return nil, fmt.Errorf("创建用户路由缓存失败: %w", err)
	}

	return &UserRouteCache{cache: generalCache, routeKey: "user:route"}, nil
}

func (c *UserRouteCache) Set(userID, gameNodeID string) bool {
	if userID == "" || gameNodeID == "" {
		return false
	}
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	return c.cache.SetWithTTL(key, gameNodeID, 2*time.Hour)
}

func (c *UserRouteCache) Get(userID string) (string, bool) {
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	return c.cache.GetString(key)
}

func (c *UserRouteCache) Delete(userID string) {
	key := fmt.Sprintf("%s:%s", c.routeKey, userID)
	c.cache.Delete(key)
}

func (c *UserRouteCache) DeleteBatch(userIDs []string) {
	for _, userID := range userIDs {
		key := fmt.Sprintf("%s:%s", c.routeKey, userID)
		c.cache.Delete(key)
	}
}

func (c *UserRouteCache) Close() {
	c.cache.Close()
}
