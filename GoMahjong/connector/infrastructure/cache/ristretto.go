package cache

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

// GeneralCache 通用本地缓存，支持 TTL
type GeneralCache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// NewGeneralCache 创建通用缓存
// maxCost: 最大内存成本（字节），建议 1GB = 1 << 30
// ttl: 默认过期时间
func NewGeneralCache(maxCost int64, ttl time.Duration) (*GeneralCache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // 1000 万个计数器
		MaxCost:     maxCost, // 最大内存成本
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ristretto 缓存失败: %w", err)
	}

	return &GeneralCache{
		cache: cache,
		ttl:   ttl,
	}, nil
}

// Set 设置缓存，使用默认 TTL
func (c *GeneralCache) Set(key string, value interface{}) bool {
	return c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL 设置缓存，指定 TTL
func (c *GeneralCache) SetWithTTL(key string, value interface{}, ttl time.Duration) bool {
	return c.cache.SetWithTTL(key, value, 1, ttl)
}

// Get 获取缓存
func (c *GeneralCache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// GetString 获取字符串缓存
func (c *GeneralCache) GetString(key string) (string, bool) {
	value, ok := c.cache.Get(key)
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// Delete 删除缓存
func (c *GeneralCache) Delete(key string) {
	c.cache.Del(key)
}

// Close 关闭缓存
func (c *GeneralCache) Close() {
	c.cache.Close()
}
