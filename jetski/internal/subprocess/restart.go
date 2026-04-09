package subprocess

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Restarter struct {
	mu              sync.Mutex
	restartCount    int
	lastRestartTime time.Time
	maxRestarts     int
	maxDelay        time.Duration
	baseDelay       time.Duration
	stableDuration  time.Duration
}

func NewRestarter() *Restarter {
	return &Restarter{
		maxRestarts:    5,
		maxDelay:       30 * time.Second,
		baseDelay:      1 * time.Second,
		stableDuration: 60 * time.Second,
	}
}

func (r *Restarter) calculateBackoffDelay(attempt int) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	exponentialFactor := 1 << uint(attempt)
	backoff := float64(r.baseDelay) * float64(exponentialFactor)
	if backoff > float64(r.maxDelay) {
		backoff = float64(r.maxDelay)
	}
	jitter := backoff * 0.1 * (2*rand.Float64() - 1)
	delay := backoff + jitter
	if delay > float64(r.maxDelay) {
		delay = float64(r.maxDelay)
	}

	return time.Duration(delay)
}

func (r *Restarter) restartEngine(ctx context.Context, restartFunc func(context.Context) error) error {
	r.mu.Lock()
	if r.restartCount >= r.maxRestarts {
		r.mu.Unlock()
		log.Printf("[JETSKI RESTARTER]: Max restarts (%d) exceeded, giving up", r.maxRestarts)
		return ErrMaxRestartsExceeded
	}
	r.mu.Unlock()

	delay := r.calculateBackoffDelay(r.restartCount)
	log.Printf("[JETSKI RESTARTER]: Attempt %d/%d, waiting %v before restart",
		r.restartCount+1, r.maxRestarts, delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	if err := restartFunc(ctx); err != nil {
		r.mu.Lock()
		r.restartCount++
		r.lastRestartTime = time.Now()
		r.mu.Unlock()
		return err
	}

	r.mu.Lock()
	r.restartCount++
	r.lastRestartTime = time.Now()
	r.mu.Unlock()

	return nil
}

func (r *Restarter) resetRestartCounter() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.lastRestartTime) >= r.stableDuration {
		log.Printf("[JETSKI RESTARTER]: Stable operation for %v, resetting restart counter", r.stableDuration)
		r.restartCount = 0
	}
}

// GetRestartCount returns the current restart count (thread-safe)
func (r *Restarter) GetRestartCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.restartCount
}

// ShouldRestart returns true if restart should be attempted
func (r *Restarter) ShouldRestart() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.restartCount < r.maxRestarts
}

var ErrMaxRestartsExceeded = &RestartError{Message: "maximum restart attempts exceeded"}

type RestartError struct {
	Message string
}

func (e *RestartError) Error() string {
	return e.Message
}
