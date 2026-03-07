package memory

import (
	"sync"
	"time"
)

type BatchWriter struct {
	store     *Store
	pending   []Message
	mu        sync.Mutex
	maxBatch  int
	interval  time.Duration
	stopChan  chan struct{}
	flushChan chan struct{}
}

type BatchWriterConfig struct {
	Store    *Store
	MaxBatch int
	Interval time.Duration
}

func NewBatchWriter(cfg BatchWriterConfig) *BatchWriter {
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = 100
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Second
	}

	bw := &BatchWriter{
		store:     cfg.Store,
		pending:   make([]Message, 0, cfg.MaxBatch),
		maxBatch:  cfg.MaxBatch,
		interval:  cfg.Interval,
		stopChan:  make(chan struct{}),
		flushChan: make(chan struct{}, 1),
	}

	go bw.run()
	return bw
}

func (bw *BatchWriter) Add(msg Message) {
	bw.mu.Lock()
	bw.pending = append(bw.pending, msg)
	shouldFlush := len(bw.pending) >= bw.maxBatch
	bw.mu.Unlock()

	if shouldFlush {
		select {
		case bw.flushChan <- struct{}{}:
		default:
		}
	}
}

func (bw *BatchWriter) Flush() error {
	bw.mu.Lock()
	pending := bw.pending
	bw.pending = make([]Message, 0, bw.maxBatch)
	bw.mu.Unlock()

	if len(pending) == 0 {
		return nil
	}

	for _, msg := range pending {
		if err := bw.store.AddMessage(msg); err != nil {
			return err
		}
	}
	return nil
}

func (bw *BatchWriter) run() {
	ticker := time.NewTicker(bw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-bw.stopChan:
			bw.Flush()
			return
		case <-ticker.C:
			bw.Flush()
		case <-bw.flushChan:
			bw.Flush()
		}
	}
}

func (bw *BatchWriter) Close() error {
	close(bw.stopChan)
	return nil
}
