package subprocess

import (
	"context"
	"log"
	"sync"
	"time"
)

type Watchdog struct {
	mu           sync.Mutex
	healthTicker *time.Ticker
	failures     int
	maxFailures  int
	checker      HealthChecker
	restartFunc  func(context.Context)
}

type HealthChecker interface {
	Check() bool
}

func NewWatchdog(maxFailures int, checker HealthChecker, restartFunc func(context.Context)) *Watchdog {
	return &Watchdog{
		maxFailures: maxFailures,
		checker:     checker,
		restartFunc: restartFunc,
	}
}

func (w *Watchdog) Start(ctx context.Context) {
	w.mu.Lock()
	w.healthTicker = time.NewTicker(5 * time.Second)
	w.mu.Unlock()

	go w.watchdogLoop(ctx)
}

func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.healthTicker != nil {
		w.healthTicker.Stop()
	}
}

func (w *Watchdog) watchdogLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Stop()
			return
		case <-w.healthTicker.C:
			if !w.checker.Check() {
				w.mu.Lock()
				w.failures++
				currentFailures := w.failures
				w.mu.Unlock()

				log.Printf("[JETSKI WATCHDOG]: Health check failed (%d/%d)", currentFailures, w.maxFailures)

				if currentFailures >= w.maxFailures {
					log.Println("[JETSKI WATCHDOG]: Circuit breaker tripped. Initiating restart...")
					w.restartFunc(ctx)
					w.ResetFailures()
				}
			} else {
				w.ResetFailures()
			}
		}
	}
}

func (w *Watchdog) ResetFailures() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.failures = 0
}
