package cache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestLRUBasic(t *testing.T) {
	lru := NewLRU(LRUConfig{MaxSize: 3, DefaultTTL: 5 * time.Minute})

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	if lru.Len() != 3 {
		t.Errorf("expected 3 items, got %d", lru.Len())
	}

	val, ok := lru.Get("key1")
	if !ok {
		t.Errorf("key1 should exist")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	lru.Delete("key2")
	if lru.Len() != 2 {
		t.Errorf("expected 2 items after delete, got %d", lru.Len())
	}

	lru.Clear()
	if lru.Len() != 0 {
		t.Errorf("expected 0 items after clear, got %d", lru.Len())
	}
}

func TestLRUEviction(t *testing.T) {
	evictCount := 0
	lru := NewLRU(LRUConfig{
		MaxSize:    3,
		DefaultTTL: 5 * time.Minute,
		OnEvict: func(key string, value interface{}) {
			evictCount++
		},
	})

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")
	lru.Set("key4", "value4")

	if lru.Len() != 3 {
		t.Errorf("expected 3 items, got %d", lru.Len())
	}

	if evictCount != 1 {
		t.Errorf("expected 1 eviction, got %d", evictCount)
	}
}

func TestLRUTTL(t *testing.T) {
	lru := NewLRU(LRUConfig{MaxSize: 100, DefaultTTL: 50 * time.Millisecond})

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")

	time.Sleep(100 * time.Millisecond)

	if _, ok := lru.Get("key1"); ok {
		t.Errorf("key1 should be expired")
	}

	if _, ok := lru.Get("key2"); ok {
		t.Errorf("key2 should be expired")
	}
}

func TestLRUPurgeExpired(t *testing.T) {
	lru := NewLRU(LRUConfig{MaxSize: 100, DefaultTTL: 50 * time.Millisecond})

	for i := 0; i < 10; i++ {
		lru.Set(strconv.Itoa(i), i)
	}

	time.Sleep(100 * time.Millisecond)
	count := lru.PurgeExpired()

	if count != 10 {
		t.Errorf("expected 10 expired, got %d", count)
	}

	if lru.Len() != 0 {
		t.Errorf("expected 0 items after purge, got %d", lru.Len())
	}
}

func TestLRUConcurrentAccess(t *testing.T) {
	lru := NewLRU(LRUConfig{MaxSize: 1000, DefaultTTL: 5 * time.Minute})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := strconv.Itoa(j % 50)
				lru.Set(key, j)
				lru.Get(key)
			}
		}(i)
	}
	wg.Wait()
}

func TestLRUGetOrCompute(t *testing.T) {
	lru := NewLRU(LRUConfig{MaxSize: 100, DefaultTTL: 5 * time.Minute})

	called := 0
	val, err := lru.GetOrCompute("key1", func() (interface{}, error) {
		called++
		return "computed", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "computed" {
		t.Errorf("expected computed, got %v", val)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}

	val, err = lru.GetOrCompute("key1", func() (interface{}, error) {
		called++
		return "computed2", nil
	})
	if called != 1 {
		t.Errorf("should not call compute again, got %d calls", called)
	}
}

func BenchmarkLRUGet(b *testing.B) {
	lru := NewLRU(LRUConfig{MaxSize: 10000, DefaultTTL: 5 * time.Minute})
	for i := 0; i < 10000; i++ {
		lru.Set(strconv.Itoa(i), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(strconv.Itoa(i % 10000))
	}
}

func BenchmarkLRUSet(b *testing.B) {
	lru := NewLRU(LRUConfig{MaxSize: 10000, DefaultTTL: 5 * time.Minute})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Set(strconv.Itoa(i), i)
	}
}
