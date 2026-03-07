package speculative

import (
	"context"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/cache"
	"github.com/armorclaw/bridge/internal/executor"
	"github.com/armorclaw/bridge/internal/metrics"
)

type SpeculativeExecutorConfig struct {
	Executor         *executor.ToolExecutor
	Cache            *cache.LRU
	MaxWorkers       int
	MaxPredictions   int
	EnableSpeculation bool
}

type SpeculativeExecutor struct {
	config       SpeculativeExecutorConfig
	mu            sync.RWMutex
	cache         *cache.LRU
	executor      *executor.ToolExecutor
	predictions   map[string]time.Time
	results       map[string]*executor.ToolResult
	pendingCalls  []executor.ToolCall
}

func NewSpeculativeExecutor(cfg SpeculativeExecutorConfig) *SpeculativeExecutor {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 5
	}
	if cfg.MaxPredictions <= 0 {
		cfg.MaxPredictions = 3
	}
	if cfg.Cache == nil {
		cfg.Cache = cache.NewLRU(cache.LRUConfig{
			MaxSize:    1000,
			DefaultTTL: 5 * time.Minute,
		})
	}

	return &SpeculativeExecutor{
		config:       cfg,
		executor:    cfg.Executor,
		cache:       cfg.Cache,
		predictions: make(map[string]time.Time),
		results:       make(map[string]*executor.ToolResult),
		pendingCalls: make([]executor.ToolCall, 0),
	}
}

func (se *SpeculativeExecutor) Predict(ctx context.Context, call executor.ToolCall) (*executor.ToolResult, bool) {
	se.mu.RLock()
	cached, ok := se.results[call.ID]
	if ok {
		metrics.RecordSpeculativeCall("hit")
		metrics.RecordCacheHit()
		se.mu.RUnlock()
		return cached, true
	}
	se.mu.RUnlock()

	if se.executor == nil {
		return nil, false
	}

	result, err := se.executor.Execute(ctx, call)
	if err != nil {
		metrics.RecordSpeculativeCall("error")
		se.mu.Lock()
		delete(se.results, call.ID)
		se.mu.Unlock()
		return nil, false
	}

	metrics.RecordSpeculativeCall("miss")
	metrics.RecordCacheMiss()

	se.mu.Lock()
	se.results[call.ID] = result
	se.predictions[call.ID] = time.Now()
	se.mu.Unlock()

	return result, true
}

func (se *SpeculativeExecutor) ExecuteBatch(ctx context.Context, calls []executor.ToolCall) ([]*executor.ToolResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	results := make([]*executor.ToolResult, len(calls))
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c executor.ToolCall) {
			defer wg.Done()
			result, ok := se.Predict(ctx, c)
			if !ok {
				errOnce.Do(func() {
					firstErr = ErrPredictionFailed
				})
				return
			}
			results[idx] = result
		}(i, call)
	}

	wg.Wait()
	return results, firstErr
}

func (se *SpeculativeExecutor) AddPredictions(calls []executor.ToolCall) {
	if len(calls) == 0 {
	 return
	 }

    se.mu.Lock()
    defer se.mu.Unlock()

    for _, call := range calls {
        se.pendingCalls = append(se.pendingCalls, call)
        se.predictions[call.ID] = time.Now()
    }
}

func (se *SpeculativeExecutor) ClearPredictions() {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.predictions = make(map[string]time.Time)
	se.results = make(map[string]*executor.ToolResult)
	se.pendingCalls = make([]executor.ToolCall, 0)
}

func (se *SpeculativeExecutor) GetCachedResult(callID string) (*executor.ToolResult, bool) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	result, ok := se.results[callID]
	return result, ok
}

func (se *SpeculativeExecutor) Close() error {
	se.ClearPredictions()
	if se.cache != nil {
		se.cache.Clear()
	}
	return nil
}

var ErrPredictionFailed = &SpeculationError{Message: "prediction failed"}

type SpeculationError struct {
	Message string
}

func (e *SpeculationError) Error() string {
	return e.Message
}
