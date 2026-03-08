package eventlog

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Record represents a single entry in the durable log
type Record struct {
	Offset    uint64          `json:"offset"`
	Timestamp int64           `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// Log handles durable append-only event storage
type Log struct {
	mu sync.RWMutex

	dir        string
	active     *Segment
	nextOffset uint64

	// Channels for async writing
	appendCh chan *Record
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Configuration
	maxSegmentSize int64
	flushInterval  time.Duration

	// Notifications
	cond *sync.Cond
}

// Config configures the durable log
type Config struct {
	Dir            string
	MaxSegmentSize int64
	FlushInterval  time.Duration
}

// New creates a new durable log
func New(cfg Config) (*Log, error) {
	if cfg.MaxSegmentSize <= 0 {
		cfg.MaxSegmentSize = 64 * 1024 * 1024 // 64MB
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 100 * time.Millisecond
	}

	l := &Log{
		dir:            cfg.Dir,
		maxSegmentSize: cfg.MaxSegmentSize,
		flushInterval:  cfg.FlushInterval,
		appendCh:       make(chan *Record, 1000),
		stopCh:         make(chan struct{}),
	}
	l.cond = sync.NewCond(&l.mu)

	if err := l.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize log: %w", err)
	}

	// Start background writer
	l.wg.Add(1)
	go l.writeLoop()

	return l, nil
}

// Append adds an event to the log (asynchronous)
func (l *Log) Append(eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	record := &Record{
		Timestamp: time.Now().UnixNano(),
		Type:      eventType,
		Payload:   data,
	}

	select {
	case l.appendCh <- record:
	default:
		// Buffer full, event dropped (Phase 1)
	}
}

// Close stops the log and flushes remaining events
func (l *Log) Close() error {
	close(l.stopCh)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active != nil {
		return l.active.Close()
	}
	return nil
}

// WaitForEvents blocks until new events are available after the cursor
func (l *Log) WaitForEvents(ctx context.Context, cursor uint64) ([]*Record, error) {
	for {
		records, err := l.ReadFrom(cursor, 100)
		if err != nil {
			return nil, err
		}

		if len(records) > 0 {
			return records, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
			// Poll interval
		}
	}
}

func (l *Log) initialize() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. Ensure directory exists
	// 2. Scan for latest segment
	// 3. Load nextOffset
	// 4. Open active segment

	// For Phase 1, we'll implement a simple single-file version or
	// minimal segment rotation.

	l.nextOffset = 0 // TODO: Load from disk on recovery

	var err error
	l.active, err = openSegment(l.dir, 0)
	if err != nil {
		return err
	}

	return nil
}

func (l *Log) writeLoop() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	var batch []*Record

	for {
		select {
		case rec := <-l.appendCh:
			l.mu.Lock()
			rec.Offset = l.nextOffset
			l.nextOffset++
			l.mu.Unlock()

			batch = append(batch, rec)
			if len(batch) >= 100 {
				l.flush(batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				l.flush(batch)
				batch = nil
			}

		case <-l.stopCh:
			// Flush remaining
			if len(batch) > 0 {
				l.flush(batch)
			}
			return
		}
	}
}

func (l *Log) flush(batch []*Record) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active == nil {
		return
	}

	for _, rec := range batch {
		if err := l.active.Write(rec); err != nil {
			// Log error, continue
			continue
		}
	}

	l.active.Sync()
	l.cond.Broadcast()
}
