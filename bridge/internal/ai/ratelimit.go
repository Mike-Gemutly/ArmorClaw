package ai

import (
	"sync"
	"time"
)

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now
	
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	
	return false
}

type RateLimiter struct {
	userBuckets   map[string]*tokenBucket
	globalBucket  *tokenBucket
	mu            sync.RWMutex
	userCapacity  float64
	userRefill    float64
}

func NewRateLimiter(userCapacity, globalCapacity, userRefillRate, globalRefillRate float64) *RateLimiter {
	return &RateLimiter{
		userBuckets:   make(map[string]*tokenBucket),
		globalBucket:  newTokenBucket(globalCapacity, globalRefillRate),
		userCapacity:  userCapacity,
		userRefill:    userRefillRate,
	}
}

func DefaultRateLimiter() *RateLimiter {
	userCapacity := float64(UserBurstCapacity)
	globalCapacity := float64(GlobalBurstCapacity)
	userRefillRate := float64(RatePerMinute) / 60.0
	globalRefillRate := float64(GlobalRateLimit) / 60.0
	
	return NewRateLimiter(userCapacity, globalCapacity, userRefillRate, globalRefillRate)
}

func (rl *RateLimiter) Allow(userID string) bool {
	if !rl.globalBucket.allow() {
		return false
	}
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	bucket, ok := rl.userBuckets[userID]
	if !ok {
		bucket = newTokenBucket(rl.userCapacity, rl.userRefill)
		rl.userBuckets[userID] = bucket
	}
	
	return bucket.allow()
}

type ConcurrentLimiter struct {
	sem chan struct{}
}

func NewConcurrentLimiter(maxConcurrent int) *ConcurrentLimiter {
	return &ConcurrentLimiter{
		sem: make(chan struct{}, maxConcurrent),
	}
}

func (cl *ConcurrentLimiter) Acquire() func() {
	cl.sem <- struct{}{}
	return func() {
		<-cl.sem
	}
}

func (cl *ConcurrentLimiter) TryAcquire() (func(), bool) {
	select {
	case cl.sem <- struct{}{}:
		return func() { <-cl.sem }, true
	default:
		return nil, false
	}
}
