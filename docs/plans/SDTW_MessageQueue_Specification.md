# SDTW Message Queue Specification

> **Component:** Persistent Message Queue for SDTW Adapters
> **Backend:** SQLite with WAL Mode
> **Status:** Design Approved - Ready for Implementation
> **Date:** 2026-02-12
> **Dependency:** SDTW Adapter Implementation Plan v2.0

---

## Executive Summary

This specification defines a durable, persistent message queue built on SQLite for the SDTW (Slack, Discord, Teams, WhatsApp) adapter system. The queue ensures reliable message delivery with retry logic, dead letter queue handling, and support for high-throughput scenarios while maintaining ACID guarantees.

**Key Design Decisions:**
- **SQLite over BoltDB:** Better concurrency support, WAL mode for read/write parallelism, mature Go ecosystem
- **Per-Adapter Queues:** Isolated queue instances prevent cascade failures
- **Persistent Storage:** Survives bridge restarts and container failures
- **Delivery Guarantees:** At-least-once delivery with deduplication

---

## Requirements

### Functional Requirements

| ID | Requirement | Priority |
|----|-------------|------------|
| F1 | Enqueue messages with unique IDs | P0 |
| F2 | Persistent storage across restarts | P0 |
| F3 | Retry with exponential backoff + jitter | P0 |
| F4 | Dead letter queue for failed messages | P0 |
| F5 | Per-platform queue isolation | P1 |
| F6 | Message priority support | P1 |
| F7 | At-least-once delivery guarantee | P0 |
| F8 | Bulk dequeue for throughput | P1 |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|----------|
| NF1 | Throughput | 1000 messages/minute per adapter |
| NF2 | Latency (p95) | < 50ms for enqueue/dequeue |
| NF3 | Memory footprint | < 100MB per queue instance |
| NF4 | Database size | < 1GB for 10K queued messages |
| NF5 | Concurrent operations | Support 100+ concurrent readers/writers |
| NF6 | Recovery time | < 5 seconds after restart |

---

## Database Schema

### Tables

```sql
-- Main queue table
CREATE TABLE queue_items (
    id              TEXT PRIMARY KEY,           -- UUID v4
    platform         TEXT NOT NULL,              -- 'slack', 'discord', 'teams', 'whatsapp'
    target_room     TEXT NOT NULL,              -- Matrix room ID
    target_channel  TEXT NOT NULL,              -- Platform channel ID
    message_type    TEXT NOT NULL,              -- 'text', 'media', 'file', etc.
    content         TEXT NOT NULL,              -- PII-scrubbed content
    attachments     BLOB,                       -- JSON array of attachments
    reply_to        TEXT,                       -- Parent message ID (threads)
    signature       TEXT,                       -- HMAC-SHA256 integrity
    metadata        BLOB,                       -- Platform-specific data (JSON)
    priority        INTEGER DEFAULT 0,            -- 0=normal, 1=high, -1=low
    attempts        INTEGER DEFAULT 0,            -- Retry counter
    max_attempts    INTEGER DEFAULT 3,            -- Maximum retry limit
    created_at      INTEGER NOT NULL,            -- Unix timestamp (ms)
    next_retry     INTEGER NOT NULL,            -- Unix timestamp (ms)
    last_attempt    INTEGER,                    -- Unix timestamp (ms)
    error_message   TEXT,                       -- Last error
    status          TEXT DEFAULT 'pending',     -- 'pending', 'inflight', 'failed', 'acked'
    expires_at      INTEGER,                    -- TTL for message (optional)
    CHECK (
        status IN ('pending', 'inflight', 'failed', 'acked')
    ),
    CHECK (
        attempts >= 0 AND attempts <= max_attempts
    ),
    CHECK (
        priority >= -1 AND priority <= 1
    )
);

-- Dead letter queue
CREATE TABLE dead_letter_queue (
    id              TEXT PRIMARY KEY,           -- Original queue item ID
    platform         TEXT NOT NULL,
    target_room     TEXT NOT NULL,
    target_channel  TEXT NOT NULL,
    message_type    TEXT NOT NULL,
    content         TEXT NOT NULL,
    attachments     BLOB,
    reply_to        TEXT,
    signature       TEXT,
    metadata        BLOB,
    priority        INTEGER,
    attempts        INTEGER NOT NULL,
    original_id      TEXT,                       -- Original platform message ID if available
    failed_at       INTEGER NOT NULL,            -- Unix timestamp (ms)
    error_message   TEXT NOT NULL,
    error_category  TEXT NOT NULL,             -- 'timeout', 'rate_limit', 'auth_failed', etc.
    reviewed        INTEGER DEFAULT 0,            -- Boolean: has admin reviewed?
    -- Foreign key to original queue item for context
    FOREIGN KEY (id) REFERENCES queue_items(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_queue_status_retry
    ON queue_items (status, next_retry)
    WHERE status IN ('pending', 'inflight');

CREATE INDEX idx_queue_platform_status
    ON queue_items (platform, status)
    WHERE status = 'pending';

CREATE INDEX idx_queue_priority
    ON queue_items (priority DESC, next_retry ASC)
    WHERE status = 'pending';

CREATE INDEX idx_queue_expires
    ON queue_items (expires_at)
    WHERE expires_at IS NOT NULL;

CREATE INDEX idx_dlq_platform
    ON dead_letter_queue (platform, failed_at DESC);

CREATE INDEX idx_dlq_reviewed
    ON dead_letter_queue (reviewed, failed_at DESC);
```

### Triggers

```sql
-- Auto-expire messages past TTL
CREATE TRIGGER trigger_expire_messages
    AFTER UPDATE OF next_retry ON queue_items
    WHEN NEW.next_retry < (strftime('%s', 'now') * 1000)
       AND NEW.expires_at IS NOT NULL
       AND NEW.expires_at < (strftime('%s', 'now') * 1000)
BEGIN
    UPDATE queue_items
    SET status = 'failed',
        error_message = 'Message expired (TTL)',
        attempts = max_attempts
    WHERE id = NEW.id;
END;

-- Auto-move to dead letter queue on max attempts
CREATE TRIGGER trigger_move_to_dlq
    AFTER UPDATE OF attempts ON queue_items
    WHEN NEW.attempts >= NEW.max_attempts
       AND NEW.status != 'acked'
BEGIN
    INSERT INTO dead_letter_queue (
        id, platform, target_room, target_channel,
        message_type, content, attachments, reply_to,
        signature, metadata, priority, attempts,
        failed_at, error_message, error_category
    )
    SELECT
        id, platform, target_room, target_channel,
        message_type, content, attachments, reply_to,
        signature, metadata, priority, attempts,
        (strftime('%s', 'now') * 1000), error_message,
        CASE
            WHEN error_message LIKE '%timeout%' THEN 'timeout'
            WHEN error_message LIKE '%rate limit%' THEN 'rate_limit'
            WHEN error_message LIKE '%auth%' THEN 'auth_failed'
            WHEN error_message LIKE '%network%' THEN 'network_error'
            ELSE 'unknown'
        END
    FROM queue_items
    WHERE id = NEW.id;

    DELETE FROM queue_items WHERE id = NEW.id;
END;
```

---

## Go Implementation

### Core Types

```go
package queue

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    "modernc.org/sqlite"
    "github.com/google/uuid"
)

// QueueConfig configures queue behavior
type QueueConfig struct {
    DBPath          string        // Path to SQLite database
    Platform         string        // Platform identifier
    MaxRetries       int           // Default: 3
    DefaultPriority   int           // Default: 0 (normal)
    MaxQueueDepth    int           // Soft limit for alerts
    RetryBaseDelay   time.Duration // Base delay: 1s
    RetryMaxDelay    time.Duration // Max delay: 5min
    EnableWAL        bool          // Enable WAL mode (default: true)
    ConnectionPool   int           // SQLite connection pool (default: 10)
}

// Message represents a queued message
type Message struct {
    ID              string                 `json:"id"`
    Platform         string                 `json:"platform"`
    TargetRoom      string                 `json:"target_room"`
    TargetChannel   string                 `json:"target_channel"`
    Type            MessageType            `json:"type"`
    Content         string                 `json:"content"`
    Attachments    []Attachment           `json:"attachments,omitempty"`
    ReplyTo         string                 `json:"reply_to,omitempty"`
    Metadata        map[string]string      `json:"metadata,omitempty"`
    Signature       string                 `json:"signature"`
    Priority        int                    `json:"priority"`
    Attempts        int                    `json:"attempts"`
    MaxAttempts     int                    `json:"max_attempts"`
    CreatedAt       time.Time              `json:"created_at"`
    NextRetry       time.Time              `json:"next_retry"`
    LastAttempt     *time.Time            `json:"last_attempt,omitempty"`
    ErrorMessage    string                 `json:"error_message,omitempty"`
    Status          QueueStatus            `json:"status"`
    ExpiresAt       *time.Time            `json:"expires_at,omitempty"`
}

type MessageType string

const (
    MessageTypeText  MessageType = "text"
    MessageTypeImage MessageType = "image"
    MessageTypeFile  MessageType = "file"
    MessageTypeMedia MessageType = "media"
)

type QueueStatus string

const (
    StatusPending   QueueStatus = "pending"
    StatusInflight QueueStatus = "inflight"
    StatusFailed   QueueStatus = "failed"
    StatusAcked   QueueStatus = "acked"
)

type Attachment struct {
    Type      string `json:"type"`
    URL       string `json:"url"`
    Name      string `json:"name,omitempty"`
    Size       int64  `json:"size,omitempty"`
    MimeType  string `json:"mime_type,omitempty"`
}

// DLQMessage represents a dead letter queue item
type DLQMessage struct {
    ID              string                 `json:"id"`
    Platform         string                 `json:"platform"`
    TargetRoom      string                 `json:"target_room"`
    TargetChannel   string                 `json:"target_channel"`
    Type            MessageType            `json:"type"`
    Content         string                 `json:"content"`
    Attachments    []Attachment           `json:"attachments,omitempty"`
    ReplyTo         string                 `json:"reply_to,omitempty"`
    Metadata        map[string]string      `json:"metadata,omitempty"`
    Signature       string                 `json:"signature"`
    Priority        int                    `json:"priority"`
    Attempts        int                    `json:"attempts"`
    OriginalID      string                 `json:"original_id,omitempty"`
    FailedAt        time.Time              `json:"failed_at"`
    ErrorMessage    string                 `json:"error_message"`
    ErrorCategory   string                 `json:"error_category"`
    Reviewed         bool                   `json:"reviewed"`
}

// EnqueueResult from enqueue operations
type EnqueueResult struct {
    ID        string    `json:"id"`
    QueuedAt   time.Time `json:"queued_at"`
    Position   int       `json:"position"`      // Position in queue
    Depth      int       `json:"depth"`       // Total queue depth
}

// DequeueResult from dequeue operations
type DequeueResult struct {
    Message   *Message   `json:"message"`
    Found      bool       `json:"found"`
    Depth      int        `json:"depth"`     // Remaining depth
}

// QueueStats for monitoring
type QueueStats struct {
    TotalMessages    int           `json:"total_messages"`
    PendingDepth    int           `json:"pending_depth"`
    InflightCount   int           `json:"inflight_count"`
    FailedCount     int           `json:"failed_count"`
    DLQCount        int           `json:"dlq_count"`
    AvgWaitTime     time.Duration `json:"avg_wait_time"`
    P95WaitTime     time.Duration `json:"p95_wait_time"`
    OldestMessage   time.Time     `json:"oldest_message"`
}
```

### Queue Implementation

```go
// MessageQueue provides persistent, reliable message queuing
type MessageQueue struct {
    config     QueueConfig
    db         *sql.DB
    mu         sync.RWMutex
    backoff    *BackoffStrategy
    metrics    *QueueMetrics
    ctx        context.Context
    cancel     context.CancelFunc
}

// BackoffStrategy calculates retry delays with jitter
type BackoffStrategy struct {
    baseDelay   time.Duration
    maxDelay    time.Duration
    multiplier   float64
    jitter      float64  // 0.0 to 1.0 (jitter percentage)
}

func (bs *BackoffStrategy) Next(attempt int) time.Duration {
    // Exponential backoff: base * (2 ^ attempt)
    delay := bs.baseDelay * time.Duration(float64(1<<uint(attempt)))

    // Apply jitter to prevent thundering herd
    if bs.jitter > 0 {
        jitterRange := time.Duration(float64(delay) * bs.jitter)
        delay += time.Duration(float64(jitterRange) * (2.0*rand.Float64() - 1.0))
    }

    // Cap at max delay
    if delay > bs.maxDelay {
        delay = bs.maxDelay
    }

    return delay
}

// NewMessageQueue creates a new queue instance
func NewMessageQueue(ctx context.Context, config QueueConfig) (*MessageQueue, error) {
    if config.DBPath == "" {
        return nil, fmt.Errorf("database path required")
    }

    if config.MaxRetries <= 0 {
        config.MaxRetries = 3
    }

    ctx, cancel := context.WithCancel(ctx)

    // Open SQLite with optimization flags
    dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", config.DBPath)
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(config.ConnectionPool)
    db.SetMaxIdleConns(config.ConnectionPool / 2)

    // Enable WAL mode for better concurrency
    if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
        return nil, fmt.Errorf("enable wal: %w", err)
    }

    // Set synchronous mode to NORMAL for performance
    if _, err := db.ExecContext(ctx, "PRAGMA synchronous=NORMAL;"); err != nil {
        return nil, fmt.Errorf("set synchronous: %w", err)
    }

    // Set cache size (10MB)
    if _, err := db.ExecContext(ctx, "PRAGMA cache_size=-10000;"); err != nil {
        return nil, fmt.Errorf("set cache size: %w", err)
    }

    // Set mmap size (1GB)
    if _, err := db.ExecContext(ctx, "PRAGMA mmap_size=1073741824;"); err != nil {
        return nil, fmt.Errorf("set mmap size: %w", err)
    }

    // Initialize schema
    if err := initSchema(ctx, db); err != nil {
        return nil, fmt.Errorf("init schema: %w", err)
    }

    mq := &MessageQueue{
        config:  config,
        db:      db,
        backoff: &BackoffStrategy{
            baseDelay: config.RetryBaseDelay,
            maxDelay:  config.RetryMaxDelay,
            multiplier: 2.0,
            jitter:     0.1, // 10% jitter
        },
        metrics: NewQueueMetrics(),
        ctx:      ctx,
        cancel:   cancel,
    }

    // Start cleanup goroutine
    go mq.cleanupLoop()

    return mq, nil
}

func initSchema(ctx context.Context, db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS queue_items (
        id              TEXT PRIMARY KEY,
        platform         TEXT NOT NULL,
        target_room     TEXT NOT NULL,
        target_channel  TEXT NOT NULL,
        message_type    TEXT NOT NULL,
        content         TEXT NOT NULL,
        attachments     BLOB,
        reply_to        TEXT,
        signature       TEXT,
        metadata        BLOB,
        priority        INTEGER DEFAULT 0,
        attempts        INTEGER DEFAULT 0,
        max_attempts    INTEGER DEFAULT 3,
        created_at      INTEGER NOT NULL,
        next_retry     INTEGER NOT NULL,
        last_attempt    INTEGER,
        error_message   TEXT,
        status          TEXT DEFAULT 'pending',
        expires_at      INTEGER,
        CHECK (status IN ('pending', 'inflight', 'failed', 'acked')),
        CHECK (attempts >= 0 AND attempts <= max_attempts),
        CHECK (priority >= -1 AND priority <= 1)
    );

    CREATE TABLE IF NOT EXISTS dead_letter_queue (
        id              TEXT PRIMARY KEY,
        platform         TEXT NOT NULL,
        target_room     TEXT NOT NULL,
        target_channel  TEXT NOT NULL,
        message_type    TEXT NOT NULL,
        content         TEXT NOT NULL,
        attachments     BLOB,
        reply_to        TEXT,
        signature       TEXT,
        metadata        BLOB,
        priority        INTEGER,
        attempts        INTEGER NOT NULL,
        original_id      TEXT,
        failed_at       INTEGER NOT NULL,
        error_message   TEXT NOT NULL,
        error_category  TEXT NOT NULL,
        reviewed        INTEGER DEFAULT 0,
        FOREIGN KEY (id) REFERENCES queue_items(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_queue_status_retry
        ON queue_items (status, next_retry)
        WHERE status IN ('pending', 'inflight');

    CREATE INDEX IF NOT EXISTS idx_queue_platform_status
        ON queue_items (platform, status)
        WHERE status = 'pending';

    CREATE INDEX IF NOT EXISTS idx_queue_priority
        ON queue_items (priority DESC, next_retry ASC)
        WHERE status = 'pending';

    CREATE INDEX IF NOT EXISTS idx_queue_expires
        ON queue_items (expires_at)
        WHERE expires_at IS NOT NULL;

    CREATE INDEX IF NOT EXISTS idx_dlq_platform
        ON dead_letter_queue (platform, failed_at DESC);

    CREATE INDEX IF NOT EXISTS idx_dlq_reviewed
        ON dead_letter_queue (reviewed, failed_at DESC);
    `

    _, err := db.ExecContext(ctx, schema)
    return err
}

// Enqueue adds a message to the queue
func (mq *MessageQueue) Enqueue(ctx context.Context, msg Message) (*EnqueueResult, error) {
    // Validate
    if msg.ID == "" {
        msg.ID = uuid.New().String()
    }
    if msg.CreatedAt.IsZero() {
        msg.CreatedAt = time.Now()
    }
    if msg.NextRetry.IsZero() {
        msg.NextRetry = time.Now()
    }
    if msg.Status == "" {
        msg.Status = StatusPending
    }
    if msg.MaxAttempts <= 0 {
        msg.MaxAttempts = mq.config.MaxRetries
    }

    // Serialize attachments
    var attachmentsBlob []byte
    if len(msg.Attachments) > 0 {
        var err error
        attachmentsBlob, err = json.Marshal(msg.Attachments)
        if err != nil {
            return nil, fmt.Errorf("marshal attachments: %w", err)
        }
    }

    // Serialize metadata
    var metadataBlob []byte
    if len(msg.Metadata) > 0 {
        var err error
        metadataBlob, err = json.Marshal(msg.Metadata)
        if err != nil {
            return nil, fmt.Errorf("marshal metadata: %w", err)
        }
    }

    // Insert
    query := `
        INSERT INTO queue_items (
            id, platform, target_room, target_channel,
            message_type, content, attachments, reply_to,
            signature, metadata, priority, attempts,
            max_attempts, created_at, next_retry, status
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    _, err := mq.db.ExecContext(ctx, query,
        msg.ID, mq.config.Platform, msg.TargetRoom, msg.TargetChannel,
        msg.Type, msg.Content, attachmentsBlob, msg.ReplyTo,
        msg.Signature, metadataBlob, msg.Priority, msg.Attempts,
        msg.MaxAttempts, msg.CreatedAt.UnixMilli(), msg.NextRetry.UnixMilli(), msg.Status,
    )
    if err != nil {
        return nil, fmt.Errorf("insert message: %w", err)
    }

    // Get position and depth
    var position, depth int
    depthQuery := `
        SELECT
            (SELECT COUNT(*) FROM queue_items WHERE status = 'pending' AND next_retry <= ?),
            (SELECT COUNT(*) FROM queue_items WHERE status = 'pending')
        `
    err = mq.db.QueryRowContext(ctx, depthQuery, time.Now().UnixMilli()).Scan(&position, &depth)
    if err != nil {
        return nil, fmt.Errorf("get queue depth: %w", err)
    }

    mq.metrics.RecordEnqueued()

    return &EnqueueResult{
        ID:      msg.ID,
        QueuedAt: msg.CreatedAt,
        Position: position,
        Depth:    depth,
    }, nil
}

// Dequeue retrieves the next pending message
func (mq *MessageQueue) Dequeue(ctx context.Context) (*DequeueResult, error) {
    mq.mu.Lock()
    defer mq.mu.Unlock()

    // Start transaction
    tx, err := mq.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Find next message (priority ordered, within retry time)
    now := time.Now().UnixMilli()
    query := `
        SELECT id, platform, target_room, target_channel,
               message_type, content, attachments, reply_to,
               signature, metadata, priority, attempts,
               max_attempts, created_at, next_retry,
               last_attempt, error_message, status,
               CASE WHEN expires_at IS NOT NULL THEN expires_at ELSE NULL END as expires_at
        FROM queue_items
        WHERE status = 'pending'
          AND next_retry <= ?
        ORDER BY priority DESC, next_retry ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    `

    var msg Message
    var attachmentsBlob, metadataBlob []byte
    var expiresAtMillis sql.NullInt64
    var lastAttemptMillis sql.NullInt64

    err = tx.QueryRowContext(ctx, query, now).Scan(
        &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
        &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
        &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
        &msg.MaxAttempts, &msg.CreatedAt, &msg.NextRetry,
        &lastAttemptMillis, &msg.ErrorMessage, &msg.Status,
        &expiresAtMillis,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            // No messages ready
            return &DequeueResult{Found: false, Depth: mq.getPendingDepth(ctx)}, nil
        }
        return nil, fmt.Errorf("query message: %w", err)
    }

    // Deserialize attachments
    if len(attachmentsBlob) > 0 {
        if err := json.Unmarshal(attachmentsBlob, &msg.Attachments); err != nil {
            return nil, fmt.Errorf("unmarshal attachments: %w", err)
        }
    }

    // Deserialize metadata
    if len(metadataBlob) > 0 {
        if err := json.Unmarshal(metadataBlob, &msg.Metadata); err != nil {
            return nil, fmt.Errorf("unmarshal metadata: %w", err)
        }
    }

    // Set timestamps
    msg.CreatedAt = time.Unix(0, msg.CreatedAt.UnixNano())
    msg.NextRetry = time.Unix(0, msg.NextRetry.UnixNano())
    if lastAttemptMillis.Valid {
        t := time.Unix(0, lastAttemptMillis.Int64*1e6)
        msg.LastAttempt = &t
    }
    if expiresAtMillis.Valid {
        t := time.Unix(0, expiresAtMillis.Int64*1e6)
        msg.ExpiresAt = &t
    }

    // Mark as inflight
    updateQuery := `UPDATE queue_items SET status = 'inflight' WHERE id = ?`
    if _, err := tx.ExecContext(ctx, updateQuery, msg.ID); err != nil {
        return nil, fmt.Errorf("mark inflight: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    mq.metrics.RecordDequeued()

    return &DequeueResult{
        Message: &msg,
        Found:   true,
        Depth:    mq.getPendingDepth(ctx),
    }, nil
}

// Ack marks a message as successfully delivered
func (mq *MessageQueue) Ack(ctx context.Context, id string, platformMessageID string) error {
    query := `
        UPDATE queue_items
        SET status = 'acked'
        WHERE id = ?
    `
    result, err := mq.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("ack message: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("message not found: %s", id)
    }

    mq.metrics.RecordAcked()

    // Optionally store platform message ID in metadata or audit log
    return nil
}

// Nack marks a message as failed and schedules retry
func (mq *MessageQueue) Nack(ctx context.Context, id string, err error) error {
    if err == nil {
        return fmt.Errorf("error required for nack")
    }

    // Get current message
    msg, findErr := mq.Get(ctx, id)
    if findErr != nil {
        return findErr
    }

    msg.Attempts++
    msg.LastAttempt = &[]time.Time{time.Now()}[0]
    msg.ErrorMessage = err.Error()

    // Check if max retries exceeded
    if msg.Attempts >= msg.MaxAttempts {
        // Move to DLQ
        return mq.moveToDLQ(ctx, msg, err)
    }

    // Calculate next retry with backoff
    nextDelay := mq.backoff.Next(msg.Attempts)
    msg.NextRetry = time.Now().Add(nextDelay)

    // Update
    query := `
        UPDATE queue_items
        SET attempts = ?, last_attempt = ?, next_retry = ?, error_message = ?
        WHERE id = ?
    `
    _, updateErr := mq.db.ExecContext(ctx, query,
        msg.Attempts, msg.LastAttempt.UnixMilli(), msg.NextRetry.UnixMilli(),
        msg.ErrorMessage, id,
    )
    if updateErr != nil {
        return fmt.Errorf("nack message: %w", updateErr)
    }

    mq.metrics.RecordRetried()

    return nil
}

// moveToDLQ moves a failed message to dead letter queue
func (mq *MessageQueue) moveToDLQ(ctx context.Context, msg Message, originalErr error) error {
    tx, err := mq.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Serialize data
    attachmentsBlob, _ := json.Marshal(msg.Attachments)
    metadataBlob, _ := json.Marshal(msg.Metadata)

    // Determine error category
    errorCategory := categorizeError(originalErr)

    // Insert into DLQ
    dlqQuery := `
        INSERT INTO dead_letter_queue (
            id, platform, target_room, target_channel,
            message_type, content, attachments, reply_to,
            signature, metadata, priority, attempts,
            failed_at, error_message, error_category
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    _, err = tx.ExecContext(ctx, dlqQuery,
        msg.ID, msg.Platform, msg.TargetRoom, msg.TargetChannel,
        msg.Type, msg.Content, attachmentsBlob, msg.ReplyTo,
        msg.Signature, metadataBlob, msg.Priority, msg.Attempts,
        time.Now().UnixMilli(), msg.ErrorMessage, errorCategory,
    )
    if err != nil {
        return fmt.Errorf("insert to dlq: %w", err)
    }

    // Delete from main queue
    deleteQuery := `DELETE FROM queue_items WHERE id = ?`
    if _, err := tx.ExecContext(ctx, deleteQuery, msg.ID); err != nil {
        return fmt.Errorf("delete from queue: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    mq.metrics.RecordDLQ()

    return nil
}

func categorizeError(err error) string {
    errStr := err.Error()
    switch {
    case contains(errStr, "timeout"):
        return "timeout"
    case contains(errStr, "rate limit"):
        return "rate_limit"
    case contains(errStr, "auth") || contains(errStr, "unauthorized"):
        return "auth_failed"
    case contains(errStr, "network"):
        return "network_error"
    default:
        return "unknown"
    }
}

// Stats returns current queue statistics
func (mq *MessageQueue) Stats(ctx context.Context) (*QueueStats, error) {
    query := `
        SELECT
            COUNT(*) as total,
            SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN status = 'inflight' THEN 1 ELSE 0 END) as inflight,
            SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
            MIN(created_at) as oldest_created
        FROM queue_items
    `

    var total, pending, inflight, failed int
    var oldestCreated sql.NullInt64
    err := mq.db.QueryRowContext(ctx, query).Scan(&total, &pending, &inflight, &failed, &oldestCreated)
    if err != nil {
        return nil, fmt.Errorf("get stats: %w", err)
    }

    // Get DLQ count
    var dlqCount int
    dlqQuery := `SELECT COUNT(*) FROM dead_letter_queue WHERE reviewed = 0`
    err = mq.db.QueryRowContext(ctx, dlqQuery).Scan(&dlqCount)
    if err != nil {
        return nil, fmt.Errorf("get dlq count: %w", err)
    }

    stats := &QueueStats{
        TotalMessages:  total,
        PendingDepth:   pending,
        InflightCount:  inflight,
        FailedCount:    failed,
        DLQCount:       dlqCount,
    }

    if oldestCreated.Valid {
        stats.OldestMessage = time.Unix(0, oldestCreated.Int64*1e6)
    }

    return stats, nil
}

// Shutdown gracefully closes the queue
func (mq *MessageQueue) Shutdown(ctx context.Context) error {
    mq.cancel()

    // Wait for inflight messages to drain
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    timeout := time.After(30 * time.Second)

    for {
        select {
        case <-ticker.C:
            stats, err := mq.Stats(ctx)
            if err != nil {
                return err
            }
            if stats.InflightCount == 0 {
                goto CloseDB
            }
        case <-timeout:
            // Force close after timeout
            goto CloseDB
        case <-ctx.Done():
            return ctx.Err()
        }
    }

CloseDB:
    return mq.db.Close()
}

// Helper methods
func (mq *MessageQueue) getPendingDepth(ctx context.Context) int {
    var depth int
    query := `SELECT COUNT(*) FROM queue_items WHERE status = 'pending' AND next_retry <= ?`
    err := mq.db.QueryRowContext(ctx, query, time.Now().UnixMilli()).Scan(&depth)
    if err != nil {
        return 0
    }
    return depth
}

func (mq *MessageQueue) cleanupLoop() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Clean up old DLQ items (older than 90 days)
            cutoff := time.Now().AddDate(0, 0, -90).UnixMilli()
            mq.db.Exec(`DELETE FROM dead_letter_queue WHERE failed_at < ? AND reviewed = 1`, cutoff)

            // Clean up expired messages
            mq.db.Exec(`DELETE FROM queue_items WHERE expires_at < ?`, time.Now().UnixMilli())

        case <-mq.ctx.Done():
            return
        }
    }
}

// Get retrieves a specific message by ID
func (mq *MessageQueue) Get(ctx context.Context, id string) (*Message, error) {
    query := `
        SELECT id, platform, target_room, target_channel,
               message_type, content, attachments, reply_to,
               signature, metadata, priority, attempts,
               max_attempts, created_at, next_retry,
               last_attempt, error_message, status,
               CASE WHEN expires_at IS NOT NULL THEN expires_at ELSE NULL END as expires_at
        FROM queue_items
        WHERE id = ?
    `

    var msg Message
    var attachmentsBlob, metadataBlob []byte
    var expiresAtMillis, lastAttemptMillis sql.NullInt64

    err := mq.db.QueryRowContext(ctx, query, id).Scan(
        &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
        &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
        &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
        &msg.MaxAttempts, &msg.CreatedAt, &msg.NextRetry,
        &lastAttemptMillis, &msg.ErrorMessage, &msg.Status,
        &expiresAtMillis,
    )
    if err != nil {
        return nil, fmt.Errorf("get message: %w", err)
    }

    // Deserialize attachments
    if len(attachmentsBlob) > 0 {
        json.Unmarshal(attachmentsBlob, &msg.Attachments)
    }

    // Deserialize metadata
    if len(metadataBlob) > 0 {
        json.Unmarshal(metadataBlob, &msg.Metadata)
    }

    // Set timestamps
    msg.CreatedAt = time.Unix(0, msg.CreatedAt.UnixNano())
    msg.NextRetry = time.Unix(0, msg.NextRetry.UnixNano())
    if lastAttemptMillis.Valid {
        t := time.Unix(0, lastAttemptMillis.Int64*1e6)
        msg.LastAttempt = &t
    }
    if expiresAtMillis.Valid {
        t := time.Unix(0, expiresAtMillis.Int64*1e6)
        msg.ExpiresAt = &t
    }

    return &msg, nil
}

// DequeueBatch retrieves multiple messages at once
func (mq *MessageQueue) DequeueBatch(ctx context.Context, batchSize int) ([]Message, error) {
    mq.mu.Lock()
    defer mq.mu.Unlock()

    tx, err := mq.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    now := time.Now().UnixMilli()
    query := `
        SELECT id, platform, target_room, target_channel,
               message_type, content, attachments, reply_to,
               signature, metadata, priority, attempts,
               max_attempts, created_at, next_retry,
               last_attempt, error_message, status
        FROM queue_items
        WHERE status = 'pending' AND next_retry <= ?
        ORDER BY priority DESC, next_retry ASC
        LIMIT ?
        FOR UPDATE SKIP LOCKED
    `

    rows, err := tx.QueryContext(ctx, query, now, batchSize)
    if err != nil {
        return nil, fmt.Errorf("query batch: %w", err)
    }
    defer rows.Close()

    var messages []Message
    for rows.Next() {
        var msg Message
        var attachmentsBlob, metadataBlob []byte
        var lastAttemptMillis sql.NullInt64

        err := rows.Scan(
            &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
            &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
            &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
            &msg.MaxAttempts, &msg.CreatedAt, &msg.NextRetry,
            &lastAttemptMillis, &msg.ErrorMessage, &msg.Status,
        )
        if err != nil {
            return nil, fmt.Errorf("scan row: %w", err)
        }

        // Deserialize attachments
        if len(attachmentsBlob) > 0 {
            json.Unmarshal(attachmentsBlob, &msg.Attachments)
        }

        // Deserialize metadata
        if len(metadataBlob) > 0 {
            json.Unmarshal(metadataBlob, &msg.Metadata)
        }

        // Set timestamps
        msg.CreatedAt = time.Unix(0, msg.CreatedAt.UnixNano())
        msg.NextRetry = time.Unix(0, msg.NextRetry.UnixNano())
        if lastAttemptMillis.Valid {
            t := time.Unix(0, lastAttemptMillis.Int64*1e6)
            msg.LastAttempt = &t
        }

        messages = append(messages, msg)

        // Mark as inflight
        updateQuery := `UPDATE queue_items SET status = 'inflight' WHERE id = ?`
        if _, err := tx.ExecContext(ctx, updateQuery, msg.ID); err != nil {
            return nil, fmt.Errorf("mark inflight: %w", err)
        }
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    mq.metrics.RecordDequeued()
    mq.metrics.RecordBatch(len(messages))

    return messages, nil
}

// Peek returns the next message without marking it as inflight
func (mq *MessageQueue) Peek(ctx context.Context) (*Message, error) {
    query := `
        SELECT id, platform, target_room, target_channel,
               message_type, content, attachments, reply_to,
               signature, metadata, priority, attempts,
               max_attempts, created_at, next_retry,
               last_attempt, error_message, status
        FROM queue_items
        WHERE status = 'pending' AND next_retry <= ?
        ORDER BY priority DESC, next_retry ASC
        LIMIT 1
    `

    var msg Message
    var attachmentsBlob, metadataBlob []byte
    var lastAttemptMillis sql.NullInt64

    err := mq.db.QueryRowContext(ctx, query, time.Now().UnixMilli()).Scan(
        &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
        &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
        &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
        &msg.MaxAttempts, &msg.CreatedAt, &msg.NextRetry,
        &lastAttemptMillis, &msg.ErrorMessage, &msg.Status,
    )
    if err != nil {
        return nil, fmt.Errorf("peek message: %w", err)
    }

    // Deserialize attachments
    if len(attachmentsBlob) > 0 {
        json.Unmarshal(attachmentsBlob, &msg.Attachments)
    }

    // Deserialize metadata
    if len(metadataBlob) > 0 {
        json.Unmarshal(metadataBlob, &msg.Metadata)
    }

    // Set timestamps
    msg.CreatedAt = time.Unix(0, msg.CreatedAt.UnixNano())
    msg.NextRetry = time.Unix(0, msg.NextRetry.UnixNano())
    if lastAttemptMillis.Valid {
        t := time.Unix(0, lastAttemptMillis.Int64*1e6)
        msg.LastAttempt = &t
    }

    return &msg, nil
}

// Requeue moves a message back to pending state
func (mq *MessageQueue) Requeue(ctx context.Context, id string) error {
    query := `
        UPDATE queue_items
        SET status = 'pending', next_retry = ?
        WHERE id = ?
    `
    _, err := mq.db.ExecContext(ctx, query, time.Now().UnixMilli(), id)
    if err != nil {
        return fmt.Errorf("requeue message: %w", err)
    }

    mq.metrics.RecordRequeued()

    return nil
}

// GetDLQMessages retrieves messages from dead letter queue
func (mq *MessageQueue) GetDLQMessages(ctx context.Context, platform string, limit int) ([]DLQMessage, error) {
    query := `
        SELECT id, platform, target_room, target_channel,
               message_type, content, attachments, reply_to,
               signature, metadata, priority, attempts,
               original_id, failed_at, error_message, error_category, reviewed
        FROM dead_letter_queue
        WHERE platform = ? OR ? = ''
        ORDER BY failed_at DESC
        LIMIT ?
    `

    rows, err := mq.db.QueryContext(ctx, query, platform, limit)
    if err != nil {
        return nil, fmt.Errorf("get dlq messages: %w", err)
    }
    defer rows.Close()

    var messages []DLQMessage
    for rows.Next() {
        var msg DLQMessage
        var attachmentsBlob, metadataBlob []byte

        err := rows.Scan(
            &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
            &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
            &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
            &msg.OriginalID, &msg.FailedAt, &msg.ErrorMessage, &msg.ErrorCategory, &msg.Reviewed,
        )
        if err != nil {
            return nil, fmt.Errorf("scan dlq row: %w", err)
        }

        // Deserialize attachments
        if len(attachmentsBlob) > 0 {
            json.Unmarshal(attachmentsBlob, &msg.Attachments)
        }

        // Deserialize metadata
        if len(metadataBlob) > 0 {
            json.Unmarshal(metadataBlob, &msg.Metadata)
        }

        msg.FailedAt = time.Unix(0, msg.FailedAt.UnixNano())

        messages = append(messages, msg)
    }

    return messages, nil
}

// MarkDLQReviewed marks a DLQ message as reviewed
func (mq *MessageQueue) MarkDLQReviewed(ctx context.Context, id string) error {
    query := `UPDATE dead_letter_queue SET reviewed = 1 WHERE id = ?`
    _, err := mq.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("mark dlq reviewed: %w", err)
    }

    mq.metrics.RecordDLQReviewed()

    return nil
}

// RetryDLQMessage retries a message from dead letter queue
func (mq *MessageQueue) RetryDLQMessage(ctx context.Context, id string) error {
    tx, err := mq.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get DLQ message
    dlqQuery := `SELECT * FROM dead_letter_queue WHERE id = ?`
    var msg Message
    var attachmentsBlob, metadataBlob []byte
    var failedAt int64

    err = tx.QueryRowContext(ctx, dlqQuery, id).Scan(
        &msg.ID, &msg.Platform, &msg.TargetRoom, &msg.TargetChannel,
        &msg.Type, &msg.Content, &attachmentsBlob, &msg.ReplyTo,
        &msg.Signature, &msg.Metadata, &msg.Priority, &msg.Attempts,
        &failedAt, &msg.ErrorMessage,
    )
    if err != nil {
        return fmt.Errorf("get dlq message: %w", err)
    }

    // Deserialize attachments
    if len(attachmentsBlob) > 0 {
        json.Unmarshal(attachmentsBlob, &msg.Attachments)
    }

    // Deserialize metadata
    if len(metadataBlob) > 0 {
        json.Unmarshal(metadataBlob, &msg.Metadata)
    }

    // Reset for retry
    msg.Status = StatusPending
    msg.Attempts = 0
    msg.NextRetry = time.Now()
    msg.CreatedAt = time.Now()

    // Insert back into main queue
    enqueueQuery := `
        INSERT INTO queue_items (
            id, platform, target_room, target_channel,
            message_type, content, attachments, reply_to,
            signature, metadata, priority, attempts,
            max_attempts, created_at, next_retry, status
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    _, err = tx.ExecContext(ctx, enqueueQuery,
        msg.ID, msg.Platform, msg.TargetRoom, msg.TargetChannel,
        msg.Type, msg.Content, attachmentsBlob, msg.ReplyTo,
        msg.Signature, metadataBlob, msg.Priority, msg.Attempts,
        msg.MaxAttempts, msg.CreatedAt.UnixMilli(), msg.NextRetry.UnixMilli(), msg.Status,
    )
    if err != nil {
        return fmt.Errorf("re-enqueue message: %w", err)
    }

    // Delete from DLQ
    deleteQuery := `DELETE FROM dead_letter_queue WHERE id = ?`
    if _, err := tx.ExecContext(ctx, deleteQuery, id); err != nil {
        return fmt.Errorf("delete from dlq: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    mq.metrics.RecordDLQRetried()

    return nil
}

// ClearDLQ clears all reviewed messages from dead letter queue
func (mq *MessageQueue) ClearDLQ(ctx context.Context, olderThan time.Duration) (int64, error) {
    cutoff := time.Now().Add(-olderThan).UnixMilli()
    result, err := mq.db.ExecContext(ctx, `DELETE FROM dead_letter_queue WHERE failed_at < ? AND reviewed = 1`, cutoff)
    if err != nil {
        return 0, fmt.Errorf("clear dlq: %w", err)
    }

    rows, _ := result.RowsAffected()
    mq.metrics.RecordDLQCleared(int(rows))

    return rows, nil
}

// QueueMetrics tracks queue performance
type QueueMetrics struct {
    enqueued      int64
    dequeued      int64
    acked         int64
    requeued      int64
    retried       int64
    dlq           int64
    dlqReviewed   int64
    dlqRetried    int64
    dlqCleared    int64
    batchSize     int
    mu            sync.RWMutex
}

func NewQueueMetrics() *QueueMetrics {
    return &QueueMetrics{}
}

func (qm *QueueMetrics) RecordEnqueued() {
    qm.mu.Lock()
    qm.enqueued++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordDequeued() {
    qm.mu.Lock()
    qm.dequeued++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordAcked() {
    qm.mu.Lock()
    qm.acked++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordRequeued() {
    qm.mu.Lock()
    qm.requeued++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordRetried() {
    qm.mu.Lock()
    qm.retried++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordDLQ() {
    qm.mu.Lock()
    qm.dlq++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordDLQReviewed() {
    qm.mu.Lock()
    qm.dlqReviewed++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordDLQRetried() {
    qm.mu.Lock()
    qm.dlqRetried++
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordDLQCleared(count int) {
    qm.mu.Lock()
    qm.dlqCleared += int64(count)
    qm.mu.Unlock()
}

func (qm *QueueMetrics) RecordBatch(size int) {
    qm.mu.Lock()
    qm.batchSize = size
    qm.mu.Unlock()
}

// ToPrometheusMetrics returns metrics in Prometheus format
func (qm *QueueMetrics) ToPrometheusMetrics(platform string) []string {
    qm.mu.RLock()
    defer qm.mu.RUnlock()

    return []string{
        fmt.Sprintf(`sdtw_queue_enqueued_total{platform="%s"} %d`, platform, qm.enqueued),
        fmt.Sprintf(`sdtw_queue_dequeued_total{platform="%s"} %d`, platform, qm.dequeued),
        fmt.Sprintf(`sdtw_queue_acked_total{platform="%s"} %d`, platform, qm.acked),
        fmt.Sprintf(`sdtw_queue_retried_total{platform="%s"} %d`, platform, qm.retried),
        fmt.Sprintf(`sdtw_queue_dlq_total{platform="%s"} %d`, platform, qm.dlq),
        fmt.Sprintf(`sdtw_queue_dlq_reviewed_total{platform="%s"} %d`, platform, qm.dlqReviewed),
        fmt.Sprintf(`sdtw_queue_dlq_retried_total{platform="%s"} %d`, platform, qm.dlqRetried),
        fmt.Sprintf(`sdtw_queue_batch_size{platform="%s"} %d`, platform, qm.batchSize),
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
package queue_test

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMessageQueue_Enqueue(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        MaxRetries:     3,
        EnableWAL:      true,
        ConnectionPool:  5,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    msg := Message{
        Platform:       "test",
        TargetRoom:     "!room:id",
        TargetChannel:  "C12345",
        Type:           MessageTypeText,
        Content:        "Hello, world!",
        Priority:       0,
    }

    result, err := mq.Enqueue(ctx, msg)
    assert.NoError(t, err)
    assert.NotEmpty(t, result.ID)
    assert.Equal(t, 1, result.Depth)
}

func TestMessageQueue_Dequeue(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        MaxRetries:     3,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Enqueue a message
    msg := Message{
        Platform:       "test",
        TargetRoom:     "!room:id",
        TargetChannel:  "C12345",
        Type:           MessageTypeText,
        Content:        "Test message",
        Priority:       0,
    }
    _, err = mq.Enqueue(ctx, msg)
    require.NoError(t, err)

    // Dequeue
    result, err := mq.Dequeue(ctx)
    assert.NoError(t, err)
    assert.True(t, result.Found)
    assert.Equal(t, "Test message", result.Message.Content)
    assert.Equal(t, StatusInflight, result.Message.Status)
}

func TestMessageQueue_RetryWithBackoff(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        MaxRetries:     3,
        RetryBaseDelay: time.Second,
        RetryMaxDelay:  time.Minute,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    msg := Message{
        Platform:       "test",
        TargetRoom:     "!room:id",
        TargetChannel:  "C12345",
        Type:           MessageTypeText,
        Content:        "Test message",
    }
    result, err := mq.Enqueue(ctx, msg)
    require.NoError(t, err)

    // First retry
    err = mq.Nack(ctx, result.ID, fmt.Errorf("temporary error"))
    assert.NoError(t, err)

    retrieved, _ := mq.Get(ctx, result.ID)
    assert.Equal(t, 1, retrieved.Attempts)

    // Check next retry is delayed with backoff
    expectedRetry := time.Now().Add(time.Second) // Base delay
    assert.WithinDuration(t, expectedRetry, retrieved.NextRetry, 100*time.Millisecond)
}

func TestMessageQueue_DeadLetterQueue(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        MaxRetries:     3,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    msg := Message{
        Platform:       "test",
        TargetRoom:     "!room:id",
        TargetChannel:  "C12345",
        Type:           MessageTypeText,
        Content:        "Test message",
        MaxAttempts:    3,
    }
    result, err := mq.Enqueue(ctx, msg)
    require.NoError(t, err)

    // Exhaust retries
    for i := 0; i < 3; i++ {
        err = mq.Nack(ctx, result.ID, fmt.Errorf("attempt %d failed", i+1))
        assert.NoError(t, err)
    }

    // Verify in DLQ
    dlqMessages, err := mq.GetDLQMessages(ctx, "test", 10)
    assert.NoError(t, err)
    assert.Len(t, dlqMessages, 1)
    assert.Equal(t, result.ID, dlqMessages[0].ID)
}

func TestMessageQueue_PriorityOrdering(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Enqueue messages with different priorities
    messages := []Message{
        {Platform: "test", TargetRoom: "!room", Content: "Low priority", Priority: -1},
        {Platform: "test", TargetRoom: "!room", Content: "Normal priority 1", Priority: 0},
        {Platform: "test", TargetRoom: "!room", Content: "High priority", Priority: 1},
        {Platform: "test", TargetRoom: "!room", Content: "Normal priority 2", Priority: 0},
    }

    for _, msg := range messages {
        _, err := mq.Enqueue(ctx, msg)
        require.NoError(t, err)
    }

    // Dequeue all
    var dequeued []string
    for i := 0; i < 4; i++ {
        result, err := mq.Dequeue(ctx)
        require.NoError(t, err)
        if result.Found {
            dequeued = append(dequeued, result.Message.Content)
        }
    }

    // Verify priority order
    assert.Equal(t, "High priority", dequeued[0])
    assert.Equal(t, "Normal priority 1", dequeued[1])
    assert.Equal(t, "Normal priority 2", dequeued[2])
    assert.Equal(t, "Low priority", dequeued[3])
}

func TestMessageQueue_BatchDequeue(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Enqueue 10 messages
    for i := 0; i < 10; i++ {
        msg := Message{
            Platform:      "test",
            TargetRoom:    "!room",
            Content:       fmt.Sprintf("Message %d", i),
            Priority:      0,
        }
        _, err := mq.Enqueue(ctx, msg)
        require.NoError(t, err)
    }

    // Batch dequeue
    messages, err := mq.DequeueBatch(ctx, 5)
    assert.NoError(t, err)
    assert.Len(t, messages, 5)

    // All should be marked inflight
    for _, msg := range messages {
        assert.Equal(t, StatusInflight, msg.Status)
    }
}

func TestMessageQueue_Statistics(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Enqueue some messages
    for i := 0; i < 5; i++ {
        msg := Message{
            Platform:    "test",
            TargetRoom:  "!room",
            Content:     fmt.Sprintf("Message %d", i),
        }
        _, err := mq.Enqueue(ctx, msg)
        require.NoError(t, err)
    }

    stats, err := mq.Stats(ctx)
    assert.NoError(t, err)
    assert.Equal(t, int64(5), stats.TotalMessages)
    assert.Equal(t, 5, stats.PendingDepth)
}

func TestMessageQueue_DeadLetterQueueRetry(t *testing.T) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        MaxRetries:     3,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Move message to DLQ
    msg := Message{
        Platform:    "test",
        TargetRoom:  "!room",
        Content:     "Failed message",
    }
    result, err := mq.Enqueue(ctx, msg)
    require.NoError(t, err)

    for i := 0; i < 3; i++ {
        mq.Nack(ctx, result.ID, fmt.Errorf("failed %d", i))
    }

    // Retry from DLQ
    err = mq.RetryDLQMessage(ctx, result.ID)
    assert.NoError(t, err)

    // Verify back in main queue
    retrieved, err := mq.Get(ctx, result.ID)
    assert.NoError(t, err)
    assert.Equal(t, StatusPending, retrieved.Status)
    assert.Equal(t, 0, retrieved.Attempts)
}
```

### Integration Tests

```go
func TestMessageQueue_Concurrency(t *testing.T) {
    // Test concurrent enqueue/dequeue operations
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "test",
        ConnectionPool:  50,
    }

    mq, err := NewMessageQueue(ctx, config)
    require.NoError(t, err)
    defer mq.Shutdown(ctx)

    // Spawn 100 goroutines
    const goroutines = 100
    const messagesPerGoroutine = 10

    var wg sync.WaitGroup
    errors := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            for j := 0; j < messagesPerGoroutine; j++ {
                msg := Message{
                    Platform:    "test",
                    TargetRoom:  "!room",
                    Content:     fmt.Sprintf("Goroutine %d, Message %d", id, j),
                }
                _, err := mq.Enqueue(ctx, msg)
                if err != nil {
                    errors <- err
                    return
                }
            }
        }(i)
    }

    // Wait for completion
    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent operation error: %v", err)
    }

    // Verify final state
    stats, _ := mq.Stats(ctx)
    assert.Equal(t, goroutines*messagesPerGoroutine, stats.TotalMessages)
}
```

### Benchmarks

```go
func BenchmarkMessageQueue_Enqueue(b *testing.B) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "bench",
        EnableWAL:      true,
        ConnectionPool:  10,
    }

    mq, _ := NewMessageQueue(ctx, config)
    defer mq.Shutdown(ctx)

    msg := Message{
        Platform:    "bench",
        TargetRoom:  "!room",
        Content:     "Benchmark message",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mq.Enqueue(ctx, msg)
    }
}

func BenchmarkMessageQueue_Dequeue(b *testing.B) {
    ctx := context.Background()
    config := QueueConfig{
        DBPath:        ":memory:",
        Platform:       "bench",
    }

    mq, _ := NewMessageQueue(ctx, config)
    defer mq.Shutdown(ctx)

    // Pre-populate queue
    for i := 0; i < 1000; i++ {
        msg := Message{
            Platform:    "bench",
            TargetRoom:  "!room",
            Content:     fmt.Sprintf("Message %d", i),
        }
        mq.Enqueue(ctx, msg)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mq.Dequeue(ctx)
    }
}
```

---

## Prometheus Metrics

```go
// CollectMetrics returns metrics for Prometheus scraping
func (mq *MessageQueue) CollectMetrics() []prometheus.Metric {
    stats, err := mq.Stats(context.Background())
    if err != nil {
        return nil
    }

    platformMetrics := mq.metrics.ToPrometheusMetrics(mq.config.Platform)

    return append(platformMetrics,
        prometheus.NewGauge(
            "sdtw_queue_depth",
            float64(stats.PendingDepth),
            prometheus.Labels{"platform": mq.config.Platform},
        ),
        prometheus.NewGauge(
            "sdtw_queue_inflight",
            float64(stats.InflightCount),
            prometheus.Labels{"platform": mq.config.Platform},
        ),
        prometheus.NewGauge(
            "sdtw_queue_failed",
            float64(stats.FailedCount),
            prometheus.Labels{"platform": mq.config.Platform},
        ),
    )
}
```

---

## Deployment Considerations

### Database Path Configuration

```toml
[queue.sdtw]
type = "sqlite"
path = "/var/lib/armorclaw/queue/sdtw.db"
pool_size = 10
wal_mode = true
cache_size_mb = 10
mmap_size_gb = 1
```

### Migration Strategy

```go
// MigrateFromBoltDB migrates from BoltDB to SQLite
func MigrateFromBoltDB(ctx context.Context, boltPath, sqlitePath string) error {
    // Open BoltDB
    db, err := bolt.Open(boltPath, 0600, nil)
    if err != nil {
        return fmt.Errorf("open boltdb: %w", err)
    }
    defer db.Close()

    // Create new SQLite queue
    config := QueueConfig{DBPath: sqlitePath}
    mq, err := NewMessageQueue(ctx, config)
    if err != nil {
        return fmt.Errorf("create sqlite queue: %w", err)
    }
    defer mq.Shutdown(ctx)

    // Migrate each bucket
    buckets := []string{"slack", "discord", "teams", "whatsapp"}
    for _, platform := range buckets {
        err := db.View(func(tx *bolt.Tx) error {
            bucket := tx.Bucket([]byte(platform))
            if bucket == nil {
                return nil
            }

            return bucket.ForEach(func(k, v []byte) error {
                var msg Message
                if err := json.Unmarshal(v, &msg); err != nil {
                    return err
                }
                msg.Platform = platform
                _, err = mq.Enqueue(ctx, msg)
                return err
            })
        })
        if err != nil {
            return fmt.Errorf("migrate %s: %w", platform, err)
        }
    }

    return nil
}
```

### Backup Strategy

```bash
#!/bin/bash
# Backup script for SQLite queue databases

BACKUP_DIR="/var/backups/armorclaw/queue"
DB_DIR="/var/lib/armorclaw/queue"
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup each queue database
for db in "$DB_DIR"/*.db; do
    filename=$(basename "$db")
    timestamp=$(date +%Y%m%d_%H%M%S)
    backup_path="$BACKUP_DIR/${filename}.${timestamp}.bak"

    # Use SQLite backup command (online backup)
    sqlite3 "$db" ".backup '$backup_path'"

    # Compress
    gzip "$backup_path"
    rm "$backup_path"
done

# Clean up old backups
find "$BACKUP_DIR" -name "*.bak.gz" -mtime +$RETENTION_DAYS -delete
```

---

## Success Criteria

- [ ] All 20+ unit tests passing
- [ ] Concurrency test handles 1000+ concurrent operations
- [ ] Benchmarks show <50ms p95 for enqueue/dequeue
- [ ] Dead letter queue correctly moves failed messages
- [ ] Priority ordering enforced
- [ ] Batch dequeue improves throughput by 5x
- [ ] Prometheus metrics exported correctly
- [ ] Graceful shutdown drains inflight messages
- [ ] Database survives unclean shutdown (WAL recovery)
- [ ] Memory footprint <100MB per queue instance
- [ ] Cleanup loop removes old DLQ messages

---

**Document Version:** 1.0
**Last Updated:** 2026-02-12
**Status:** Ready for Implementation
**Dependencies:** SDTW Adapter Implementation Plan v2.0
