package subprocess

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestCalculateBackoffDelay(t *testing.T) {
	r := NewRestarter()

	tests := []struct {
		name        string
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "Attempt 0 - no backoff",
			attempt:     0,
			expectedMin: 900 * time.Millisecond,
			expectedMax: 1100 * time.Millisecond,
		},
		{
			name:        "Attempt 1 - 2s base with jitter",
			attempt:     1,
			expectedMin: 1800 * time.Millisecond,
			expectedMax: 2200 * time.Millisecond,
		},
		{
			name:        "Attempt 2 - 4s base with jitter",
			attempt:     2,
			expectedMin: 3600 * time.Millisecond,
			expectedMax: 4400 * time.Millisecond,
		},
		{
			name:        "Attempt 3 - 8s base with jitter",
			attempt:     3,
			expectedMin: 7200 * time.Millisecond,
			expectedMax: 8800 * time.Millisecond,
		},
		{
			name:        "Attempt 4 - 16s base with jitter",
			attempt:     4,
			expectedMin: 14400 * time.Millisecond,
			expectedMax: 17600 * time.Millisecond,
		},
		{
			name:        "Attempt 5 - capped at 30s",
			attempt:     5,
			expectedMin: 27000 * time.Millisecond,
			expectedMax: 33000 * time.Millisecond,
		},
		{
			name:        "Attempt 10 - still capped at 30s",
			attempt:     10,
			expectedMin: 27000 * time.Millisecond,
			expectedMax: 33000 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := r.calculateBackoffDelay(tt.attempt)

			if delay < tt.expectedMin {
				t.Errorf("Delay %v is less than expected minimum %v", delay, tt.expectedMin)
			}
			if delay > tt.expectedMax {
				t.Errorf("Delay %v is greater than expected maximum %v", delay, tt.expectedMax)
			}
		})
	}
}

func TestBackoffJitter(t *testing.T) {
	r := NewRestarter()
	attempt := 2

	delays := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		delays[i] = r.calculateBackoffDelay(attempt)
	}

	exponentialFactor := 1 << uint(attempt)
	baseDelay := float64(r.baseDelay) * float64(exponentialFactor)
	minExpected := time.Duration(baseDelay * 0.9)
	maxExpected := time.Duration(baseDelay * 1.1)

	var hasVariation bool
	for _, delay := range delays {
		if delay < minExpected || delay > maxExpected {
			t.Errorf("Delay %v outside expected jitter range [%v, %v]", delay, minExpected, maxExpected)
		}
		if delay != delays[0] {
			hasVariation = true
		}
	}

	if !hasVariation {
		t.Error("No jitter variation detected across 100 attempts")
	}
}

func TestMaxDelayCap(t *testing.T) {
	r := NewRestarter()

	for attempt := 0; attempt < 20; attempt++ {
		delay := r.calculateBackoffDelay(attempt)
		if delay > r.maxDelay {
			t.Errorf("Attempt %d: Delay %v exceeds max delay %v", attempt, delay, r.maxDelay)
		}
	}
}

func TestRestartCounter(t *testing.T) {
	r := NewRestarter()
	r.baseDelay = 1 * time.Millisecond

	if r.GetRestartCount() != 0 {
		t.Errorf("Initial restart count should be 0, got %d", r.GetRestartCount())
	}

	if !r.ShouldRestart() {
		t.Error("ShouldRestart should return true initially")
	}

	for i := 0; i < r.maxRestarts-1; i++ {
		err := r.restartEngine(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Unexpected error on restart %d: %v", i, err)
		}
	}

	if r.GetRestartCount() != r.maxRestarts-1 {
		t.Errorf("Restart count should be %d, got %d", r.maxRestarts-1, r.GetRestartCount())
	}

	if !r.ShouldRestart() {
		t.Error("ShouldRestart should still return true before max restarts")
	}

	err := r.restartEngine(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Unexpected error on restart: %v", err)
	}

	if r.GetRestartCount() != r.maxRestarts {
		t.Errorf("Restart count should be %d, got %d", r.maxRestarts, r.GetRestartCount())
	}

	if r.ShouldRestart() {
		t.Error("ShouldRestart should return false after max restarts")
	}

	err = r.restartEngine(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != ErrMaxRestartsExceeded {
		t.Errorf("Expected ErrMaxRestartsExceeded, got %v", err)
	}
}

func TestRestartCounterReset(t *testing.T) {
	r := NewRestarter()
	r.baseDelay = 1 * time.Millisecond
	r.stableDuration = 100 * time.Millisecond

	for i := 0; i < r.maxRestarts-1; i++ {
		err := r.restartEngine(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Unexpected error on restart %d: %v", i, err)
		}
	}

	if r.GetRestartCount() != r.maxRestarts-1 {
		t.Errorf("Restart count should be %d, got %d", r.maxRestarts-1, r.GetRestartCount())
	}

	r.resetRestartCounter()
	if r.GetRestartCount() != r.maxRestarts-1 {
		t.Error("Restart counter should not reset immediately")
	}

	time.Sleep(r.stableDuration + 50*time.Millisecond)
	r.resetRestartCounter()

	if r.GetRestartCount() != 0 {
		t.Errorf("Restart counter should reset to 0 after stable period, got %d", r.GetRestartCount())
	}
}

func TestRestartEngineWithFailure(t *testing.T) {
	r := NewRestarter()
	callCount := 0

	restartFunc := func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return ErrMaxRestartsExceeded
		}
		return nil
	}

	err := r.restartEngine(context.Background(), restartFunc)
	if err == nil {
		t.Error("Expected error on failed restart")
	}

	if callCount != 1 {
		t.Errorf("Restart function should be called 1 time, was called %d times", callCount)
	}

	if r.GetRestartCount() != 1 {
		t.Errorf("Restart count should be 1, got %d", r.GetRestartCount())
	}
}

func TestRestartEngineWithCancel(t *testing.T) {
	r := NewRestarter()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.restartEngine(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if r.GetRestartCount() != 0 {
		t.Errorf("Restart count should be 0 after cancel, got %d", r.GetRestartCount())
	}
}

func TestBackoffDelayStability(t *testing.T) {
	r := NewRestarter()
	attempt := 3

	var sum time.Duration
	iterations := 100

	for i := 0; i < iterations; i++ {
		delay := r.calculateBackoffDelay(attempt)
		sum += delay
	}

	average := time.Duration(float64(sum) / float64(iterations))
	exponentialFactor := 1 << uint(attempt)
	baseDelay := float64(r.baseDelay) * float64(exponentialFactor)

	expectedAverage := time.Duration(baseDelay)

	diff := math.Abs(float64(average) - float64(expectedAverage))
	tolerance := float64(expectedAverage) * 0.1

	if diff > tolerance {
		t.Errorf("Average delay %v differs from expected %v by more than tolerance", average, expectedAverage)
	}
}
