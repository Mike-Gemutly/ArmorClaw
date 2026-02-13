// Package queue provides a persistent, reliable message queue for SDTW adapters using SQLite with WAL mode for concurrent access and ACID guarantees.
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
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
}

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

// MessageQueue manages persistent message queue
type MessageQueue struct {
	config      QueueConfig
	db          *sql.DB
	metrics     *QueueMetrics
	mu          sync.RWMutex
	shutdownChan chan struct{}
	closed      bool
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

	// Open SQLite database
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode=%s", config.DBPath, map[bool]string{true: "WAL", false: "DELETE"}[config.EnableWAL])
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	mq := &MessageQueue{
		config:      config,
		db:          db,
		metrics:     NewQueueMetrics(),
		shutdownChan: make(chan struct{}),
		closed:      false,
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = mq.db.ExecContext(ctx, query,
		msg.ID, msg.Platform, msg.TargetRoom, msg.TargetChannel, msg.Type, msg.Content,
		string(attachmentsJSON), msg.ReplyTo, string(metadataJSON), msg.Signature,
		msg.Priority, msg.Attempts, msg.MaxAttempts, msg.CreatedAt.Unix(),
		msg.Status, msg.ExpiresAt.Unix(),
	)

	if err != nil {
		return nil, fmt.Errorf("enqueue message %s: %w", msg.ID, err)
	}

	mq.metrics.RecordEnqueued()
	mq.metrics.RecordBatch(1)

	// Reuse stats from depth check above
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

	// Start transaction for atomic dequeue
	tx, err := mq.db.BeginTx(ctx, nil)
	if err != nil {
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
		return &DequeueResult{Found: false, Depth: 0}, nil
	}
	if err != nil {
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
		return nil, fmt.Errorf("mark in-flight: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit dequeue: %w", err)
	}

	msg.LastAttempt = &now
	msg.Status = StatusInflight

	mq.metrics.RecordDequeued()

	stats, _ := mq.Stats(ctx)
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
		return fmt.Errorf("ack message %s: %w", id, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message %s not found or not in-flight", id)
	}

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

	var messages []*Message

	tx, err := mq.db.BeginTx(ctx, nil)
	if err != nil {
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
		return nil, fmt.Errorf("commit batch: %w", err)
	}

	mq.metrics.RecordBatch(len(messages))
	mq.metrics.RecordDequeued()

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
