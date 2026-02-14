// Package queue provides a persistent, reliable message queue for SDTW adapters using SQLite with WAL mode for concurrent access and ACID guarantees.
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// QueueConfig configures queue behavior
type QueueConfig struct {
	DBPath         string
	Platform        string
	MaxRetries      int
	DefaultPriority int
	MaxQueueDepth   int
	RetryBaseDelay  time.Duration
	RetryMaxDelay   time.Duration
	EnableWAL       bool
	ConnectionPool  int
	// Circuit breaker settings
	CircuitBreakerThreshold int           // Consecutive failures before opening
	CircuitBreakerTimeout  time.Duration // Time to wait before trying again
	BatchMaxSize         int           // Maximum batch size for dequeue
}

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota // Normal operation
	CircuitOpen                            // Failures exceeded, not attempting
	CircuitHalfOpen                         // Testing if recovered
)

// Message represents a queued message
type Message struct {
	ID             string
	Platform       string
	TargetRoom     string
	TargetChannel  string
	Type           MessageType
	Content        string
	Attachments    []Attachment
	ReplyTo        string
	Metadata       map[string]string
	Signature       string
	Priority       int
	Attempts       int
	MaxAttempts    int
	CreatedAt      time.Time
	NextRetry      time.Time
	LastAttempt    *time.Time
	ErrorMessage   string
	Status         QueueStatus
	ExpiresAt      *time.Time
}

// QueueStatus represents message status in queue
type QueueStatus string

const (
	StatusPending   QueueStatus = "pending"
	StatusInflight QueueStatus = "inflight"
	StatusFailed   QueueStatus = "failed"
	StatusAcked   QueueStatus = "acked"
)

// EnqueueResult from enqueue operations
type EnqueueResult struct {
	ID      string
	QueuedAt time.Time
	Position int
	Depth    int
}

// DequeueResult from dequeue operations
type DequeueResult struct {
	Message *Message
	Found   bool
	Depth    int
}

// QueueStats for monitoring
type QueueStats struct {
	TotalMessages  int
	PendingDepth    int
	InflightCount int
	FailedCount   int
	DLQCount      int
}

// HealthStatus represents the health of the queue
type HealthStatus struct {
	Healthy       bool      `json:"healthy"`
	Status        string    `json:"status"`
	PendingDepth  int       `json:"pending_depth"`
	InflightCount int       `json:"inflight_count"`
	FailedCount   int       `json:"failed_count"`
	CircuitState  string    `json:"circuit_state"`
	LastFailure   string    `json:"last_failure,omitempty"`
	Uptime        string    `json:"uptime"`
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
	MessageTypeMedia MessageType = "media"
)

// Attachment represents a file attachment
type Attachment struct {
	ID       string
	URL      string
	MimeType string
	Size     int64
	Filename string
}

// CircuitBreaker manages circuit breaker state
type CircuitBreaker struct {
	mu               sync.RWMutex
	state             CircuitBreakerState
	consecutiveErrors int
	threshold        int
	halfOpenAttempts int
	lastFailureTime  time.Time
	timeout          time.Duration
	openUntil        time.Time
	lastStateChange  time.Time  // NEW: Track when state last changed
	db               *sql.DB  // NEW: Database handle for state persistence
}

// MessageQueue manages persistent message queue
type MessageQueue struct {
	config         QueueConfig
	db             *sql.DB
	metrics        *QueueMetrics
	mu             sync.RWMutex
	shutdownChan   chan struct{}
	closed         bool
	circuitBreaker *CircuitBreaker
	startTime      time.Time
}

// Schema version for migrations
const schemaVersion = 1

// initDB initializes the database schema
func (mq *MessageQueue) initDB(ctx context.Context) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Enable WAL mode for concurrent access
	if mq.config.EnableWAL {
		if _, err := mq.db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
			return fmt.Errorf("enable WAL mode: %w", err)
		}
		if _, err := mq.db.ExecContext(ctx, "PRAGMA busy_timeout=5000;"); err != nil {
			return fmt.Errorf("set busy timeout: %w", err)
		}
	}

	// Set connection pool settings
	mq.db.SetMaxOpenConns(mq.config.ConnectionPool)
	mq.db.SetMaxIdleConns(mq.config.ConnectionPool / 2)

	// Create messages table
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		platform TEXT NOT NULL,
		target_room TEXT NOT NULL,
		target_channel TEXT NOT NULL,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		attachments TEXT,
		reply_to TEXT,
		metadata TEXT,
		signature TEXT,
		priority INTEGER DEFAULT 0,
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		created_at INTEGER NOT NULL,
		next_retry INTEGER,
		last_attempt INTEGER,
		error_message TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		expires_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_status_priority ON messages(status, priority, created_at);
	CREATE INDEX IF NOT EXISTS idx_next_retry ON messages(next_retry) WHERE next_retry IS NOT NULL;
	CREATE TABLE IF NOT EXISTS queue_meta (key TEXT PRIMARY KEY, value TEXT);
	`

	if _, err := mq.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Store schema version
	if _, err := mq.db.ExecContext(ctx, "INSERT OR REPLACE INTO queue_meta (key, value) VALUES ('schema_version', '1');"); err != nil {
		return fmt.Errorf("store schema version: %w", err)
	}

	return nil
}

// NewMessageQueue creates a new queue instance
func NewMessageQueue(ctx context.Context, config QueueConfig) (*MessageQueue, error) {
	// Apply defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.DefaultPriority == 0 {
		config.DefaultPriority = 5
	}
	if config.MaxQueueDepth == 0 {
		config.MaxQueueDepth = 10000
	}
	if config.RetryBaseDelay == 0 {
		config.RetryBaseDelay = time.Second
	}
	if config.RetryMaxDelay == 0 {
		config.RetryMaxDelay = 5 * time.Minute
	}
	if config.ConnectionPool == 0 {
		config.ConnectionPool = 10
	}
	if config.CircuitBreakerThreshold == 0 {
		config.CircuitBreakerThreshold = 5
	}
	if config.CircuitBreakerTimeout == 0 {
		config.CircuitBreakerTimeout = time.Minute
	}
	if config.BatchMaxSize == 0 {
		config.BatchMaxSize = 100
	}

	// Open SQLite database
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode=%s", config.DBPath, map[bool]string{true: "WAL", false: "DELETE"}[config.EnableWAL])
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	cb := &CircuitBreaker{
		state:    CircuitClosed,
		threshold: config.CircuitBreakerThreshold,
		timeout:   config.CircuitBreakerTimeout,
		db:        db,  // NEW: Set database for state persistence
	}

	// NEW: Load circuit breaker state from database if available
	if err := cb.loadState(ctx); err != nil {
		// Log error but don't fail queue initialization
		fmt.Fprintf(os.Stderr, "[WARN] Failed to load circuit breaker state: %v\n", err)
	}

	mq := &MessageQueue{
		config:         config,
		db:             db,
		metrics:        NewQueueMetrics(config.Platform),
		shutdownChan:   make(chan struct{}),
		closed:         false,
		circuitBreaker: cb,
		startTime:      time.Now(),
	}

	if err := mq.initDB(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	return mq, nil
}

// Enqueue adds a message to the queue
func (mq *MessageQueue) Enqueue(ctx context.Context, msg Message) (*EnqueueResult, error) {
	if mq.isClosed() {
		return nil, fmt.Errorf("queue is shutdown")
	}

	// Check circuit breaker
	if !mq.circuitBreaker.canProceed() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Validate message
	if msg.ID == "" {
		return nil, fmt.Errorf("message ID is required")
	}
	if msg.Status == "" {
		msg.Status = StatusPending
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if msg.Priority == 0 {
		msg.Priority = mq.config.DefaultPriority
	}
	if msg.MaxAttempts == 0 {
		msg.MaxAttempts = mq.config.MaxRetries
	}

	// Check queue depth
	stats, err := mq.Stats(ctx)
	if err == nil && stats.PendingDepth >= mq.config.MaxQueueDepth {
		return nil, fmt.Errorf("queue depth exceeded: %d >= %d", stats.PendingDepth, mq.config.MaxQueueDepth)
	}

	// Serialize complex types
	var attachmentsJSON []byte
	if len(msg.Attachments) > 0 {
		attachmentsJSON, _ = json.Marshal(msg.Attachments)
	}
	var metadataJSON []byte
	if len(msg.Metadata) > 0 {
		metadataJSON, _ = json.Marshal(msg.Metadata)
	}

	// Insert message
	query := `
		INSERT INTO messages (
			id, platform, target_room, target_channel, type, content,
			attachments, reply_to, metadata, signature, priority,
			attempts, max_attempts, created_at, status, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = mq.db.ExecContext(ctx, query,
		msg.ID, msg.Platform, msg.TargetRoom, msg.TargetChannel, msg.Type, msg.Content,
		string(attachmentsJSON), msg.ReplyTo, string(metadataJSON), msg.Signature,
		msg.Priority, msg.Attempts, msg.MaxAttempts, msg.CreatedAt.Unix(),
		msg.Status, msg.ExpiresAt.Unix(),
	)

	if err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("enqueue message %s: %w", msg.ID, err)
	}

	// Success - reset circuit breaker
	mq.circuitBreaker.recordSuccess()
	mq.metrics.RecordEnqueued()
	mq.metrics.RecordBatch(1)

	// Update Prometheus gauges
	mq.metrics.UpdateGauges(stats.PendingDepth+1, stats.InflightCount, stats.FailedCount)

	return &EnqueueResult{
		ID:      msg.ID,
		QueuedAt: msg.CreatedAt,
		Position: stats.PendingDepth,
		Depth:    stats.PendingDepth + 1,
	}, nil
}

// Dequeue retrieves the next pending message
func (mq *MessageQueue) Dequeue(ctx context.Context) (*DequeueResult, error) {
	if mq.isClosed() {
		return nil, fmt.Errorf("queue is shutdown")
	}

	// Check circuit breaker
	if !mq.circuitBreaker.canProceed() {
		return &DequeueResult{Found: false, Depth: 0}, nil
	}

	// Start transaction for atomic dequeue
	tx, err := mq.db.BeginTx(ctx, nil)
	if err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find next message with row-level locking
	query := `
		SELECT id, platform, target_room, target_channel, type, content,
			   attachments, reply_to, metadata, signature, priority,
			   attempts, max_attempts, created_at, next_retry, status, expires_at
		FROM messages
		WHERE status = 'pending' AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE;
	`

	var msg Message
	var attachmentsJSON, metadataJSON sql.NullString
	var expiresAt sql.NullInt64

	err = tx.QueryRowContext(ctx, query, time.Now().Unix()).Scan(
		&msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel, &msg.Type, &msg.Content,
		&attachmentsJSON, &msg.ReplyTo, &metadataJSON, &msg.Signature,
		&msg.Priority, &msg.Attempts, &msg.MaxAttempts,
		&msg.CreatedAt, &msg.NextRetry, &msg.Status, &expiresAt,
	)

	if err == sql.ErrNoRows {
		mq.circuitBreaker.recordSuccess()
		return &DequeueResult{Found: false, Depth: 0}, nil
	}
	if err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("query message: %w", err)
	}

	// Deserialize complex types
	if attachmentsJSON.Valid {
		json.Unmarshal([]byte(attachmentsJSON.String), &msg.Attachments)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata)
	}
	if expiresAt.Valid {
		msg.ExpiresAt = &time.Time{}
		*msg.ExpiresAt = time.Unix(expiresAt.Int64, 0)
	}

	// Mark as in-flight
	now := time.Now()
	if _, err := tx.ExecContext(ctx, "UPDATE messages SET status = 'inflight', last_attempt = ? WHERE id = ?", now.Unix(), msg.ID); err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("mark in-flight: %w", err)
	}

	if err := tx.Commit(); err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("commit dequeue: %w", err)
	}

	msg.LastAttempt = &now
	msg.Status = StatusInflight

	mq.circuitBreaker.recordSuccess()
	mq.metrics.RecordDequeued()

	stats, _ := mq.Stats(ctx)
	// Update Prometheus gauges
	mq.metrics.UpdateGauges(stats.PendingDepth, stats.InflightCount, stats.FailedCount)

	return &DequeueResult{
		Message: &msg,
		Found:   true,
		Depth:    stats.PendingDepth,
	}, nil
}

// Ack marks a message as successfully delivered
func (mq *MessageQueue) Ack(ctx context.Context, id string) error {
	if mq.isClosed() {
		return fmt.Errorf("queue is shutdown")
	}

	result, err := mq.db.ExecContext(ctx, "UPDATE messages SET status = 'acked' WHERE id = ? AND status = 'inflight'", id)
	if err != nil {
		mq.circuitBreaker.recordFailure()
		return fmt.Errorf("ack message %s: %w", id, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message %s not found or not in-flight", id)
	}

	mq.circuitBreaker.recordSuccess()
	mq.metrics.RecordAcked()
	return nil
}

// Nack marks a message as failed and schedules retry
func (mq *MessageQueue) Nack(ctx context.Context, id string, nackErr error) error {
	if mq.isClosed() {
		return fmt.Errorf("queue is shutdown")
	}

	mq.mu.RLock()
	defer mq.mu.RUnlock()

	// Get current message
	var msg Message
	var attachmentsJSON, metadataJSON sql.NullString
	err := mq.db.QueryRowContext(ctx, "SELECT * FROM messages WHERE id = ?", id).Scan(
		&msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel, &msg.Type, &msg.Content,
		&attachmentsJSON, &msg.ReplyTo, &metadataJSON, &msg.Signature,
		&msg.Priority, &msg.Attempts, &msg.MaxAttempts,
		&msg.CreatedAt, &msg.NextRetry, &msg.LastAttempt, &msg.ErrorMessage,
		&msg.Status, &msg.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("get message %s: %w", id, err)
	}

	msg.Attempts++

	// Check if max retries exceeded
	if msg.Attempts >= msg.MaxAttempts {
		// Move to DLQ
		if _, err := mq.db.ExecContext(ctx, "UPDATE messages SET status = 'failed', error_message = ? WHERE id = ?", nackErr.Error(), id); err != nil {
			return fmt.Errorf("move to DLQ: %w", err)
		}
		mq.metrics.RecordDLQ()
		return fmt.Errorf("message %s exceeded max retries (%d)", id, msg.MaxAttempts)
	}

	// Calculate next retry with exponential backoff and jitter
	nextRetry := mq.calculateNextRetry(msg.Attempts)

	// Schedule retry
	if _, err := mq.db.ExecContext(ctx, "UPDATE messages SET status = 'pending', next_retry = ?, attempts = ?, error_message = ? WHERE id = ?",
		nextRetry.Unix(), msg.Attempts, nackErr.Error(), id); err != nil {
		return fmt.Errorf("schedule retry: %w", err)
	}

	mq.metrics.RecordRetried()
	mq.metrics.RecordRequeued()
	return nil
}

// calculateNextRetry computes the next retry time with exponential backoff and jitter
func (mq *MessageQueue) calculateNextRetry(attempt int) time.Time {
	// Exponential backoff: base * 2^attempt
	baseDelay := float64(mq.config.RetryBaseDelay)
	expBackoff := baseDelay * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	if expBackoff > float64(mq.config.RetryMaxDelay) {
		expBackoff = float64(mq.config.RetryMaxDelay)
	}

	// Add 10% jitter to prevent thundering herd
	jitter := expBackoff * 0.10 * (rand.Float64()*2 - 1)
	delay := time.Duration(expBackoff + jitter)

	return time.Now().Add(delay)
}

// Stats returns current queue statistics
func (mq *MessageQueue) Stats(ctx context.Context) (*QueueStats, error) {
	var stats QueueStats

	// Count by status
	row := mq.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'inflight' THEN 1 END) as inflight,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(*) as total
		FROM messages
		WHERE expires_at IS NULL OR expires_at > ?
	`, time.Now().Unix())

	err := row.Scan(&stats.PendingDepth, &stats.InflightCount, &stats.FailedCount, &stats.TotalMessages)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}

	stats.DLQCount = stats.FailedCount

	return &stats, nil
}

// Shutdown gracefully closes the queue
func (mq *MessageQueue) Shutdown(ctx context.Context) error {
	mq.mu.Lock()
	if mq.closed {
		mq.mu.Unlock()
		return nil
	}
	mq.closed = true
	mq.mu.Unlock()

	// Signal shutdown
	close(mq.shutdownChan)

	// Wait for inflight messages to complete (brief grace period)
	time.Sleep(100 * time.Millisecond)

	// Close database
	if err := mq.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}

// isClosed checks if queue is shut down
func (mq *MessageQueue) isClosed() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.closed
}

// DequeueBatch retrieves multiple messages for batch processing
func (mq *MessageQueue) DequeueBatch(ctx context.Context, batchSize int) ([]*Message, error) {
	if mq.isClosed() {
		return nil, fmt.Errorf("queue is shutdown")
	}

	// Check circuit breaker
	if !mq.circuitBreaker.canProceed() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Apply batch size limit
	if batchSize > mq.config.BatchMaxSize {
		batchSize = mq.config.BatchMaxSize
	}
	if batchSize <= 0 {
		batchSize = 10 // default
	}

	var messages []*Message

	tx, err := mq.db.BeginTx(ctx, nil)
	if err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		SELECT id, platform, target_room, target_channel, type, content,
			   attachments, reply_to, metadata, signature, priority,
			   attempts, max_attempts, created_at, next_retry, status, expires_at
		FROM messages
		WHERE status = 'pending' AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY priority DESC, created_at ASC
		LIMIT ?
		FOR UPDATE;
	`

	rows, err := tx.QueryContext(ctx, query, time.Now().Unix(), batchSize)
	if err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("query batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var attachmentsJSON, metadataJSON sql.NullString
		var expiresAt sql.NullInt64

		if err := rows.Scan(
			&msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel, &msg.Type, &msg.Content,
			&attachmentsJSON, &msg.ReplyTo, &metadataJSON, &msg.Signature,
			&msg.Priority, &msg.Attempts, &msg.MaxAttempts,
			&msg.CreatedAt, &msg.NextRetry, &msg.Status, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		// Deserialize
		if attachmentsJSON.Valid {
			json.Unmarshal([]byte(attachmentsJSON.String), &msg.Attachments)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata)
		}
		if expiresAt.Valid {
			msg.ExpiresAt = &time.Time{}
			*msg.ExpiresAt = time.Unix(expiresAt.Int64, 0)
		}

		// Mark as in-flight
		now := time.Now()
		if _, err := tx.ExecContext(ctx, "UPDATE messages SET status = 'inflight', last_attempt = ? WHERE id = ?", now.Unix(), msg.ID); err != nil {
			return nil, fmt.Errorf("mark batch in-flight: %w", err)
		}

		msg.LastAttempt = &now
		msg.Status = StatusInflight
		messages = append(messages, &msg)
	}

	if err := tx.Commit(); err != nil {
		mq.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("commit batch: %w", err)
	}

	mq.circuitBreaker.recordSuccess()
	mq.metrics.RecordBatch(len(messages))
	for range messages {
		mq.metrics.RecordDequeued()
	}

	stats, _ := mq.Stats(ctx)
	mq.metrics.UpdateGauges(stats.PendingDepth, stats.InflightCount, stats.FailedCount)

	return messages, nil
}

// GetPendingRetryCount returns count of messages pending retry
func (mq *MessageQueue) GetPendingRetryCount(ctx context.Context) (int, error) {
	var count int
	err := mq.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages
		WHERE status = 'pending' AND next_retry IS NOT NULL AND next_retry <= ?
	`, time.Now().Unix()).Scan(&count)

	return count, err
}

// ProcessRetryQueue processes messages due for retry
func (mq *MessageQueue) ProcessRetryQueue(ctx context.Context) (int, error) {
	if mq.isClosed() {
		return 0, fmt.Errorf("queue is shutdown")
	}

	result, err := mq.db.ExecContext(ctx, `
		UPDATE messages SET status = 'pending', next_retry = NULL
		WHERE status = 'pending' AND next_retry IS NOT NULL AND next_retry <= ?
	`, time.Now().Unix())

	if err != nil {
		return 0, fmt.Errorf("process retries: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CleanupExpired removes expired messages
func (mq *MessageQueue) CleanupExpired(ctx context.Context) (int, error) {
	if mq.isClosed() {
		return 0, fmt.Errorf("queue is shutdown")
	}

	result, err := mq.db.ExecContext(ctx, `
		DELETE FROM messages WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now().Unix())

	if err != nil {
		return 0, fmt.Errorf("cleanup expired: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// Health returns the current health status of the queue
func (mq *MessageQueue) Health(ctx context.Context) (*HealthStatus, error) {
	if mq.isClosed() {
		return &HealthStatus{Healthy: false, Status: "shutdown"}, nil
	}

	stats, err := mq.Stats(ctx)
	if err != nil {
		return &HealthStatus{Healthy: false, Status: "error"}, err
	}

	// Update Prometheus gauges
	mq.metrics.UpdateGauges(stats.PendingDepth, stats.InflightCount, stats.FailedCount)

	// Determine circuit breaker state
	cbState := mq.circuitBreaker.state.String()
	var lastFailure string
	if !mq.circuitBreaker.lastFailureTime.IsZero() {
		lastFailure = mq.circuitBreaker.lastFailureTime.Format(time.RFC3339)
	}

	uptime := time.Since(mq.startTime).String()

	healthy := stats.InflightCount < mq.config.ConnectionPool &&
		mq.circuitBreaker.state != CircuitOpen &&
		stats.FailedCount < stats.TotalMessages/10

	status := "healthy"
	if !healthy {
		status = "degraded"
	}
	if mq.circuitBreaker.state == CircuitOpen {
		status = "unhealthy"
	}

	return &HealthStatus{
		Healthy:       healthy,
		Status:        status,
		PendingDepth:  stats.PendingDepth,
		InflightCount: stats.InflightCount,
		FailedCount:   stats.FailedCount,
		CircuitState:  cbState,
		LastFailure:   lastFailure,
		Uptime:        uptime,
	}, nil
}

// HealthHandler is an HTTP handler for health checks
func (mq *MessageQueue) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	health, err := mq.Health(ctx)

	w.Header().Set("Content-Type", "application/json")
	if err != nil || !health.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(health)
}

// MetricsHandler is an HTTP handler for Prometheus metrics scraping
// Note: In production, use prometheus.HandlerFor(registry) to properly expose metrics
func (mq *MessageQueue) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	// Return a simple metrics summary for now
	// Full Prometheus integration requires prometheus package import and registry
	stats, _ := mq.Stats(r.Context())
	fmt.Fprintf(w, "# HELP sdtw_queue_depth Current depth of message queue\n")
	fmt.Fprintf(w, "# TYPE sdtw_queue_depth gauge\n")
	fmt.Fprintf(w, "sdtw_queue_depth{platform=\"%s\",state=\"pending\"} %d\n", mq.config.Platform, stats.PendingDepth)
	fmt.Fprintf(w, "sdtw_queue_depth{platform=\"%s\",state=\"inflight\"} %d\n", mq.config.Platform, stats.InflightCount)
	fmt.Fprintf(w, "sdtw_queue_depth{platform=\"%s\",state=\"failed\"} %d\n", mq.config.Platform, stats.FailedCount)

	fmt.Fprintf(w, "\n# HELP sdtw_queue_inflight Number of messages currently in flight\n")
	fmt.Fprintf(w, "# TYPE sdtw_queue_inflight gauge\n")
	fmt.Fprintf(w, "sdtw_queue_inflight{platform=\"%s\"} %d\n", mq.config.Platform, stats.InflightCount)
}

// canProceed checks if the circuit breaker allows operations
func (cb *CircuitBreaker) canProceed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Check if we're in open state and timeout has passed
	if cb.state == CircuitOpen && time.Now().After(cb.openUntil) {
		// Transition to half-open for recovery
		cb.state = CircuitHalfOpen
		cb.halfOpenAttempts = 0
		return true
	}

	// Check if we're in half-open state and timeout has passed
	if cb.state == CircuitHalfOpen && time.Now().After(cb.openUntil) {
		// Transition to closed after timeout expires
		cb.state = CircuitClosed
	}

	return cb.state != CircuitOpen
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors = 0
	if cb.state == CircuitHalfOpen {
		cb.halfOpenAttempts++
		// After 3 successful attempts in half-open, close circuit
		if cb.halfOpenAttempts >= 3 {
			cb.state = CircuitClosed
		}
	}

	// NEW: Persist state change to queue_meta
	if err := cb.saveState(context.Background()); err != nil {
		// Log error but don't fail operation
		fmt.Fprintf(os.Stderr, "[WARN] Failed to persist circuit breaker state: %v\n", err)
	}
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors++
	cb.lastFailureTime = time.Now()

	// Open circuit if threshold exceeded
	if cb.consecutiveErrors >= cb.threshold {
		cb.state = CircuitOpen
		cb.openUntil = time.Now().Add(cb.timeout)
	}

	// Persist state change
	if err := cb.saveState(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to persist circuit breaker state: %v\n", err)
	}

	// saveState persists circuit breaker state to queue_meta table
func (cb *CircuitBreaker) saveState(ctx context.Context) error {
	// Serialize state as JSON for storage
	stateJSON, err := json.Marshal(map[string]interface{}{
		"state":       cb.state.String(),
		"open_until":  cb.openUntil.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Store in queue_meta table
	_, err = cb.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO queue_meta (key, value) VALUES (?, ?)",
		"circuit_breaker_state", // key
		cb.openUntil.Format(time.RFC3339), // value
	)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Update lastStateChange timestamp
	cb.lastStateChange = time.Now()

	return nil
}

// loadState loads circuit breaker state from queue_meta table
func (cb *CircuitBreaker) loadState(ctx context.Context) error {
	// Query state from queue_meta
	var stateStr, openUntilStr, lastChangeStr string
	var state CircuitBreakerState
	var openUntil time.Time

	err := cb.db.QueryRowContext(ctx,
		"SELECT value FROM queue_meta WHERE key = 'circuit_breaker_state'",
	).Scan(
		&stateStr,  // value: open_until timestamp
	)

	if err != nil {
		// If no state exists, start with Closed state (default)
		if errors.Is(err, sql.ErrNoRows) {
			state = CircuitClosed
			openUntil = time.Time{}
			cb.lastStateChange = time.Now()
			return nil
		}
		// Parse state
		switch stateStr {
		case "closed":
			state = CircuitClosed
		case "open":
			state = CircuitOpen
		case "half_open":
			state = CircuitHalfOpen
		default:
			state = CircuitClosed // Default to closed on parse error
		}

	// Parse openUntil timestamp
	if openUntilStr != "" {
		openUntil, err = time.Parse(time.RFC3339, openUntilStr)
		if err != nil {
			return fmt.Errorf("failed to parse open_until: %w", err)
		}
	}

	// Parse lastStateChange timestamp
	if lastChangeStr != "" {
		cb.lastStateChange, err = time.Parse(time.RFC3339, lastChangeStr)
		if err != nil {
			return fmt.Errorf("failed to parse last_state_change: %w", err)
		}
	}

	// Update circuit breaker fields
	cb.state = state
	cb.openUntil = openUntil

	return nil
}

// String returns string representation of circuit state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}
