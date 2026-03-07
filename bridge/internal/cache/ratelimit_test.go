package cache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterBasic(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(RateLimitConfig{Rate: 10, Burst: 20})

	if !rl.Allow("key1") {
		t.Errorf("key1 should be allowed")
	}

	if !rl.Allow("key2") {
		t.Errorf("key2 should be allowed")
	}
}

func TestRateLimiterWait(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(RateLimitConfig{Rate: 1, Burst: 1})

	rl.Allow("key1")

	err := rl.WaitTimeout("key1", 50*time.Millisecond)
	if err == nil {
		t.Errorf("should timeout")
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Rate: 100, Burst: 200})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := strconv.Itoa(j)
				rl.Allow(key)
			}
		}(i)
	}
	wg.Wait()
}
