package sidecar

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrQueueFull is returned when the queue is full
	ErrQueueFull = errors.New("request queue is full")

	// ErrQueueShutdown is returned when trying to enqueue after shutdown
	ErrQueueShutdown = errors.New("queue is shutting down")

	// ErrRequestFailed is returned when a request fails after all retries
	ErrRequestFailed = errors.New("request failed after retries")

	// ErrRequestQueued is returned when a request is queued due to sidecar being down
	ErrRequestQueued = errors.New("request queued - sidecar unavailable")
)

// QueuedRequest represents a request waiting to be processed
type QueuedRequest struct {
	Operation  string
	RequestID  string
	UserID     string
	AgentID    string
	SessionID  string
	QueuedAt   time.Time
	RetryCount int32
	Execute    func(ctx context.Context) error
	Callback   func(error)
}

// QueueConfig holds configuration for the request queue
type QueueConfig struct {
	MaxSize             int           // Maximum number of queued requests
	MaxRetryAttempts    int           // Maximum number of retry attempts per request
	InitialBackoff      time.Duration // Initial backoff duration
	MaxBackoff          time.Duration // Maximum backoff duration
	BackoffMultiplier   float64       // Multiplier for exponential backoff
	HealthCheckInterval time.Duration // Interval between health checks
}

// DefaultQueueConfig returns default queue configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		MaxSize:             1000,
		MaxRetryAttempts:    5,
		InitialBackoff:      1 * time.Second,
		MaxBackoff:          30 * time.Second,
		BackoffMultiplier:   2.0,
		HealthCheckInterval: 10 * time.Second,
	}
}

// QueueManager manages the request queue and retry logic
type QueueManager struct {
	config          *QueueConfig
	client          *Client
	auditClient     *AuditClient
	queue           []*QueuedRequest
	mu              sync.Mutex
	shutdown        int32 // atomic
	shutdownCh      chan struct{}
	pendingRequests int32       // atomic
	logger          interface{} // slog.Logger (using interface to avoid import cycle)
}

// NewQueueManager creates a new queue manager
func NewQueueManager(client *Client, auditClient *AuditClient, config *QueueConfig) *QueueManager {
	if config == nil {
		config = DefaultQueueConfig()
	}

	return &QueueManager{
		config:      config,
		client:      client,
		auditClient: auditClient,
		queue:       make([]*QueuedRequest, 0, config.MaxSize),
		shutdownCh:  make(chan struct{}),
	}
}

// Start starts the queue manager background worker
func (qm *QueueManager) Start(ctx context.Context) {
	go qm.processQueue(ctx)
}

// Shutdown gracefully shuts down the queue manager
func (qm *QueueManager) Shutdown() {
	atomic.StoreInt32(&qm.shutdown, 1)
	close(qm.shutdownCh)
}

// Enqueue adds a request to the queue
func (qm *QueueManager) Enqueue(req *QueuedRequest) error {
	if atomic.LoadInt32(&qm.shutdown) != 0 {
		return ErrQueueShutdown
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	if len(qm.queue) >= qm.config.MaxSize {
		return ErrQueueFull
	}

	if req.QueuedAt.IsZero() {
		req.QueuedAt = time.Now()
	}
	req.RetryCount = 0

	qm.queue = append(qm.queue, req)
	atomic.AddInt32(&qm.pendingRequests, 1)

	qm.logDebug("request enqueued",
		"operation", req.Operation,
		"request_id", req.RequestID,
		"queue_size", len(qm.queue),
	)

	return nil
}

// Size returns the current queue size
func (qm *QueueManager) Size() int {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	return len(qm.queue)
}

// PendingCount returns the number of pending requests
func (qm *QueueManager) PendingCount() int32 {
	return atomic.LoadInt32(&qm.pendingRequests)
}

// IsShutdown returns true if the queue is shutting down
func (qm *QueueManager) IsShutdown() bool {
	return atomic.LoadInt32(&qm.shutdown) != 0
}

// processQueue processes queued requests in the background
func (qm *QueueManager) processQueue(ctx context.Context) {
	ticker := time.NewTicker(qm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-qm.shutdownCh:
			qm.logDebug("queue manager shutting down")
			return

		case <-ticker.C:
			qm.tryProcessQueue(ctx)

		case <-ctx.Done():
			qm.logDebug("queue manager context cancelled")
			return
		}
	}
}

// tryProcessQueue attempts to process queued requests
func (qm *QueueManager) tryProcessQueue(ctx context.Context) {
	// Check if sidecar is available
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := qm.client.HealthCheck(healthCtx)
	if err != nil {
		// Sidecar still down, skip processing
		qm.logDebug("sidecar unavailable, skipping queue processing", "error", err)
		return
	}

	// Sidecar is up, process queue
	qm.processPendingRequests(ctx)
}

// processPendingRequests processes all pending requests in the queue
func (qm *QueueManager) processPendingRequests(ctx context.Context) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	for len(qm.queue) > 0 {
		// Check shutdown
		if atomic.LoadInt32(&qm.shutdown) != 0 {
			break
		}

		req := qm.queue[0]

		// Execute the request
		err := req.Execute(ctx)

		if err == nil {
			// Success - remove from queue
			qm.queue = qm.queue[1:]
			atomic.AddInt32(&qm.pendingRequests, -1)

			qm.logDebug("request succeeded and removed from queue",
				"operation", req.Operation,
				"request_id", req.RequestID,
				"remaining", len(qm.queue),
			)

			// Call callback if provided
			if req.Callback != nil {
				go req.Callback(nil)
			}

		} else {
			// Failure - check retry count
			req.RetryCount++

			if req.RetryCount >= int32(qm.config.MaxRetryAttempts) {
				// Max retries exceeded - remove from queue
				qm.queue = qm.queue[1:]
				atomic.AddInt32(&qm.pendingRequests, -1)

				qm.logWarn("request failed after max retries, removing from queue",
					"operation", req.Operation,
					"request_id", req.RequestID,
					"retry_count", req.RetryCount,
					"error", err,
				)

				// Call callback with error
				if req.Callback != nil {
					go req.Callback(ErrRequestFailed)
				}

			} else {
				backoff := qm.calculateBackoff(req.RetryCount)

				qm.logDebug("request failed, will retry",
					"operation", req.Operation,
					"request_id", req.RequestID,
					"retry_count", req.RetryCount,
					"backoff", backoff,
					"error", err,
				)

				// Rotate: remove from front, add to back
				qm.queue = qm.queue[1:]
				qm.queue = append(qm.queue, req)

				// Schedule retry with backoff
				go func(r *QueuedRequest) {
					time.Sleep(backoff)
					// Log retry event
					if qm.auditClient != nil {
						qm.auditClient.LogRetryEvent(ctx, r.Operation, int(r.RetryCount))
					}
				}(req)

				// Don't process more requests immediately - wait for next tick
				break
			}
		}
	}
}

// calculateBackoff calculates exponential backoff for retries
func (qm *QueueManager) calculateBackoff(retryCount int32) time.Duration {
	backoff := float64(qm.config.InitialBackoff)
	for i := int32(1); i < retryCount; i++ {
		backoff *= qm.config.BackoffMultiplier
		if backoff >= float64(qm.config.MaxBackoff) {
			return qm.config.MaxBackoff
		}
	}

	return time.Duration(backoff)
}

// isTransientError checks if an error is transient and should trigger a queue
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check gRPC status codes
	st, ok := status.FromError(err)
	if !ok {
		// Non-gRPC error - assume transient if connection-related
		return isConnectionError(err)
	}

	// Transient error codes
	switch st.Code() {
	case codes.Unavailable:
	case codes.DeadlineExceeded:
	case codes.Aborted:
	case codes.ResourceExhausted:
		return true
	default:
		return false
	}

	return true
}

// isConnectionError checks if an error is connection-related
func isConnectionError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "no such host") ||
		contains(errStr, "network is unreachable") ||
		contains(errStr, "i/o timeout") ||
		contains(errStr, "broken pipe")
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring is a simple substring finder
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// QueuedClient wraps a sidecar client with request queuing
type QueuedClient struct {
	client       *Client
	queueManager *QueueManager
	auditClient  *AuditClient
}

// NewQueuedClient creates a new queued client wrapper
func NewQueuedClient(client *Client, auditClient *AuditClient, queueConfig *QueueConfig) *QueuedClient {
	queueManager := NewQueueManager(client, auditClient, queueConfig)
	queueManager.Start(context.Background())

	return &QueuedClient{
		client:       client,
		queueManager: queueManager,
		auditClient:  auditClient,
	}
}

// Shutdown shuts down the queued client
func (qc *QueuedClient) Shutdown() {
	qc.queueManager.Shutdown()
}

// executeWithQueue executes a request with queuing on transient errors
func (qc *QueuedClient) executeWithQueue(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	err := fn(ctx)

	if err == nil {
		return nil
	}

	// If error is transient and sidecar appears down, queue the request
	if isTransientError(err) {
		// Try one more health check to confirm sidecar is down
		healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if _, healthErr := qc.client.HealthCheck(healthCtx); healthErr != nil {
			// Sidecar is down - queue the request
			qc.logDebug("sidecar down, queuing request",
				"operation", operation,
				"error", err,
			)

			// Log queue event
			if qc.auditClient != nil {
				qc.auditClient.LogQueueEvent(ctx, operation, qc.queueManager.Size())
			}

			req := &QueuedRequest{
				Operation:  operation,
				Execute:    fn,
				QueuedAt:   time.Now(),
				RetryCount: 0,
			}

			if enqueueErr := qc.queueManager.Enqueue(req); enqueueErr != nil {
				qc.logDebug("failed to enqueue request", "error", enqueueErr)
				return enqueueErr
			}

			return ErrRequestQueued
		}
	}

	return err
}

// logDebug is a placeholder for debug logging
func (qc *QueuedClient) logDebug(msg string, args ...interface{}) {
	// TODO: Implement proper logging
}

func (qm *QueueManager) logDebug(msg string, args ...interface{}) {
	// TODO: Implement proper logging
}

func (qm *QueueManager) logWarn(msg string, args ...interface{}) {
	// TODO: Implement proper logging
}

// GetClient returns the underlying sidecar client
func (qc *QueuedClient) GetClient() *Client {
	return qc.client
}

// GetQueueManager returns the queue manager
func (qc *QueuedClient) GetQueueManager() *QueueManager {
	return qc.queueManager
}
