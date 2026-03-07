package memory

import (
	"time"
)

type Checkpointer struct {
	store    *Store
	interval time.Duration
	stopChan chan struct{}
}

type CheckpointerConfig struct {
	Store    *Store
	Interval time.Duration
}

func NewCheckpointer(cfg CheckpointerConfig) *Checkpointer {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}

	cp := &Checkpointer{
		store:    cfg.Store,
		interval: cfg.Interval,
		stopChan: make(chan struct{}),
	}

	go cp.run()
	return cp
}

func (c *Checkpointer) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			c.store.Checkpoint()
			return
		case <-ticker.C:
			c.store.Checkpoint()
		}
	}
}

func (c *Checkpointer) Close() error {
	close(c.stopChan)
	return nil
}
