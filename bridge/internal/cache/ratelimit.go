package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

type RateLimitConfig struct {
	Rate  float64
	Burst int
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.Rate <= 0 {
		cfg.Rate = 10
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 20
	}
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(cfg.Rate),
		burst:    cfg.Burst,
	}
}

func (r *RateLimiter) getLimiter(key string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if limiter, exists = r.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(r.rate, r.burst)
	r.limiters[key] = limiter
	return limiter
}

func (r *RateLimiter) Allow(key string) bool {
	return r.getLimiter(key).Allow()
}

func (r *RateLimiter) Wait(key string) error {
	return r.getLimiter(key).Wait(nil)
}

func (r *RateLimiter) WaitTimeout(key string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return ErrRateLimitTimeout
	default:
		return r.getLimiter(key).Wait(ctx)
	}
}

var ErrRateLimitTimeout = &RateLimitError{Message: "rate limit wait timeout"}

type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

func (r *RateLimiter) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.limiters, key)
}

func (r *RateLimiter) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limiters = make(map[string]*rate.Limiter)
}

func (r *RateLimiter) Reserve(key string) *rate.Reservation {
	return r.getLimiter(key).Reserve()
}

func (r *RateLimiter) SetRate(newRate float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rate = rate.Limit(newRate)
	for _, limiter := range r.limiters {
		limiter.SetLimit(r.rate)
	}
}

func (r *RateLimiter) SetBurst(newBurst int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.burst = newBurst
	for _, limiter := range r.limiters {
		limiter.SetBurst(r.burst)
	}
}
