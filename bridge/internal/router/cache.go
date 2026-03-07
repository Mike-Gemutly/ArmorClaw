package router

import (
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/cache"
)

type RouterCache struct {
	cache *cache.LRU
	mu   sync.RWMutex
}

type RouterCacheConfig struct {
	MaxSize  int
	TTL      time.Duration
}

func NewRouterCache(cfg RouterCacheConfig) *RouterCache {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1000
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	return &RouterCache{
		cache: cache.NewLRU(cache.LRUConfig{
			MaxSize:    cfg.MaxSize,
			DefaultTTL: cfg.TTL,
		}),
	}
}

func (rc *RouterCache) Get(message string) (*RouteResult, bool) {
	key := hashMessage(message)
	val, ok := rc.cache.Get(key)
	if !ok {
		return nil, false
	}
	result, ok := val.(*RouteResult)
	return result, ok
}

func (rc *RouterCache) Set(message string, result *RouteResult) {
	key := hashMessage(message)
	rc.cache.Set(key, result)
}

func (rc *RouterCache) Clear() {
	rc.cache.Clear()
}

func hashMessage(message string) string {
	if len(message) > 64 {
		message = message[:64]
	}
	return message
}
