package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/metrics"
)

type ToolCache struct {
	lru      *LRU
	mu       sync.RWMutex
	hits     int64
	misses   int64
}

type ToolCacheConfig struct {
	MaxSize   int
	TTL       time.Duration
}

func NewToolCache(cfg ToolCacheConfig) *ToolCache {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 500
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 10 * time.Minute
	}
	return &ToolCache{
		lru: NewLRU(LRUConfig{
			MaxSize:   cfg.MaxSize,
			DefaultTTL: cfg.TTL,
			OnEvict: func(key string, value interface{}) {
				metrics.RecordCacheEviction()
			},
		}),
	}
}

func canonicalJSON(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	return data, nil
}

func cacheKey(toolName string, args map[string]interface{}) (string, error) {
	data, err := canonicalJSON(args)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(append([]byte(toolName+":"), data...))
	return string(hash[:]), nil
}

func (c *ToolCache) Get(toolName string, args map[string]interface{}) (string, bool) {
	key, err := cacheKey(toolName, args)
	if err != nil {
		return "", false
	}

	c.mu.RLock()
	val, ok := c.lru.Get(key)
	c.mu.RUnlock()

	if ok {
		c.mu.Lock()
		c.hits++
		c.mu.Unlock()
		metrics.RecordCacheHit()
		return val.(string), true
	}

	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
	metrics.RecordCacheMiss()
	return "", false
}

func (c *ToolCache) Set(toolName string, args map[string]interface{}, result string) {
	key, err := cacheKey(toolName, args)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Set(key, result)
}

func (c *ToolCache) Delete(toolName string, args map[string]interface{}) {
	key, err := cacheKey(toolName, args)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Delete(key)
}

func (c *ToolCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Clear()
	c.hits = 0
	c.misses = 0
}

func (c *ToolCache) Stats() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

func (c *ToolCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}
