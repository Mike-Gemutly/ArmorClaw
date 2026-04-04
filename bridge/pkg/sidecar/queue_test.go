package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultQueueConfig(t *testing.T) {
	cfg := DefaultQueueConfig()

	if cfg.MaxSize != 1000 {
		t.Errorf("DefaultQueueConfig() MaxSize = %v, want 1000", cfg.MaxSize)
	}
	if cfg.MaxRetryAttempts != 5 {
		t.Errorf("DefaultQueueConfig() MaxRetryAttempts = %v, want 5", cfg.MaxRetryAttempts)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("DefaultQueueConfig() InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("DefaultQueueConfig() MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("DefaultQueueConfig() BackoffMultiplier = %v, want 2.0", cfg.BackoffMultiplier)
	}
}

func TestQueueManager_New(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)
	qm := NewQueueManager(client, auditClient, nil)

	if qm == nil {
		t.Fatal("NewQueueManager() returned nil")
	}
	if qm.Size() != 0 {
		t.Errorf("NewQueueManager() size = %v, want 0", qm.Size())
	}
	if qm.IsShutdown() {
		t.Error("NewQueueManager() is shutdown, want false")
	}
}

func TestQueueManager_Enqueue(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)
	qm := NewQueueManager(client, auditClient, nil)

	req := &QueuedRequest{
		Operation: "TestOperation",
		Execute:   func(ctx context.Context) error { return nil },
	}

	err := qm.Enqueue(req)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if qm.Size() != 1 {
		t.Errorf("Enqueue() size = %v, want 1", qm.Size())
	}

	if qm.PendingCount() != 1 {
		t.Errorf("Enqueue() pending = %v, want 1", qm.PendingCount())
	}
}

func TestQueueManager_EnqueueFull(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)

	config := &QueueConfig{
		MaxSize:           2,
		MaxRetryAttempts:  3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	qm := NewQueueManager(client, auditClient, config)

	// Fill queue
	for i := 0; i < 2; i++ {
		req := &QueuedRequest{
			Operation: "TestOperation",
			Execute:   func(ctx context.Context) error { return nil },
		}
		err := qm.Enqueue(req)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	// Try to enqueue when full
	req := &QueuedRequest{
		Operation: "TestOperation",
		Execute:   func(ctx context.Context) error { return nil },
	}

	err := qm.Enqueue(req)
	if err != ErrQueueFull {
		t.Errorf("Enqueue() error = %v, want %v", err, ErrQueueFull)
	}
}

func TestQueueManager_EnqueueShutdown(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)
	qm := NewQueueManager(client, auditClient, nil)

	// Shutdown the queue
	qm.Shutdown()

	req := &QueuedRequest{
		Operation: "TestOperation",
		Execute:   func(ctx context.Context) error { return nil },
	}

	err := qm.Enqueue(req)
	if err != ErrQueueShutdown {
		t.Errorf("Enqueue() error = %v, want %v", err, ErrQueueShutdown)
	}
}

func TestQueueManager_Size(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)
	qm := NewQueueManager(client, auditClient, nil)

	if qm.Size() != 0 {
		t.Errorf("Size() = %v, want 0", qm.Size())
	}

	for i := 0; i < 3; i++ {
		req := &QueuedRequest{
			Operation: "TestOperation",
			Execute:   func(ctx context.Context) error { return nil },
		}
		qm.Enqueue(req)
	}

	if qm.Size() != 3 {
		t.Errorf("Size() = %v, want 3", qm.Size())
	}
}

func TestQueueManager_Shutdown(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)
	qm := NewQueueManager(client, auditClient, nil)

	if qm.IsShutdown() {
		t.Error("IsShutdown() = true, want false before Shutdown()")
	}

	qm.Shutdown()

	if !qm.IsShutdown() {
		t.Error("IsShutdown() = false, want true after Shutdown()")
	}
}

func TestCalculateBackoff(t *testing.T) {
	client := NewClient(nil)
	auditClient := NewAuditClient(client, nil)

	config := &QueueConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	qm := NewQueueManager(client, auditClient, config)

	tests := []struct {
		name       string
		retryCount int32
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "first retry",
			retryCount: 1,
			wantMin:    100 * time.Millisecond,
			wantMax:    100 * time.Millisecond,
		},
		{
			name:       "second retry",
			retryCount: 2,
			wantMin:    200 * time.Millisecond,
			wantMax:    200 * time.Millisecond,
		},
		{
			name:       "third retry",
			retryCount: 3,
			wantMin:    400 * time.Millisecond,
			wantMax:    400 * time.Millisecond,
		},
		{
			name:       "capped at max",
			retryCount: 10,
			wantMin:    1 * time.Second,
			wantMax:    1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qm.calculateBackoff(tt.retryCount)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateBackoff(%v) = %v, want [%v, %v]", tt.retryCount, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-transient",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "no such host",
			err:  errors.New("no such host"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTransientError(tt.err); got != tt.want {
				t.Errorf("isTransientError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "no such host",
			err:  errors.New("no such host"),
			want: true,
		},
		{
			name: "network is unreachable",
			err:  errors.New("network is unreachable"),
			want: true,
		},
		{
			name: "i/o timeout",
			err:  errors.New("i/o timeout"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  errors.New("broken pipe"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConnectionError(tt.err); got != tt.want {
				t.Errorf("isConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}
