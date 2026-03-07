package cache

import (
	"testing"
	"time"
)

func TestToolCacheBasic(t *testing.T) {
	t.Parallel()

	tc := NewToolCache(ToolCacheConfig{
		MaxSize: 100,
		TTL:     10 * time.Minute,
	})

	tc.Set("tool1", map[string]interface{}{"arg": "value1"}, "result1")
	tc.Set("tool2", map[string]interface{}{"arg": "value2"}, "result2")

	val, ok := tc.Get("tool1", map[string]interface{}{"arg": "value1"})
	if !ok {
		t.Errorf("tool1 should exist")
	}
	if val != "result1" {
		t.Errorf("expected result1, got %v", val)
	}
}

func TestToolCacheKeyCollision(t *testing.T) {
	t.Parallel()

	tc := NewToolCache(ToolCacheConfig{
		MaxSize: 100,
		TTL:     10 * time.Minute,
	})

	tc.Set("tool", map[string]interface{}{"arg": "a"}, "result_a")
	tc.Set("tool", map[string]interface{}{"arg": "b"}, "result_b")

	val, ok := tc.Get("tool", map[string]interface{}{"arg": "a"})
	if !ok {
		t.Errorf("should find result_a")
	}
	if val != "result_a" {
		t.Errorf("expected result_a, got %v", val)
	}
}
