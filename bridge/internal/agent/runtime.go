package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/cache"
	"github.com/armorclaw/bridge/internal/executor"
	"github.com/armorclaw/bridge/internal/memory"
	"github.com/armorclaw/bridge/internal/metrics"
	"github.com/armorclaw/bridge/internal/router"
	"github.com/armorclaw/bridge/internal/speculative"
)

type RuntimeConfig struct {
	MaxSteps          int
	MaxTokens         int
	Timeout           time.Duration
	EnableSpeculation bool
	MaxParallelTools  int
}

type Runtime struct {
	config       RuntimeConfig
	executor     *executor.ToolExecutor
	cache        *cache.ToolCache
	router      *router.Router
	memory       *memory.Store
	speculative  *speculative.SpeculativeExecutor
	mu             sync.RWMutex
	pending      map[string]bool
	running      map[string]*Task
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		MaxSteps:          10,
		MaxTokens:         4096,
		Timeout:           30 * time.Second,
		EnableSpeculation: true,
		MaxParallelTools:  3,
	}
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxParallelTools <= 0 {
		cfg.MaxParallelTools = 3
	}

	exec := executor.NewToolExecutor(executor.ToolExecutorConfig{
		Timeout:    cfg.Timeout,
		MaxWorkers: cfg.MaxParallelTools,
	})

	tc := cache.NewToolCache(cache.ToolCacheConfig{
		MaxSize: 500,
		TTL:       10 * time.Minute,
	})

	rtr := router.NewRouter(router.RouterConfig{})

	mem, _ := memory.NewStore(memory.StoreConfig{})

	rt := &Runtime{
		config:      cfg,
		executor:    exec,
		cache:       tc,
		router:      rtr,
		memory:      mem,
		pending:     make(map[string]bool),
		running:     make(map[string]*Task),
	}

	if cfg.EnableSpeculation {
		rt.speculative = speculative.NewSpeculativeExecutor(speculative.SpeculativeExecutorConfig{
			Executor:       exec,
			MaxPredictions: cfg.MaxParallelTools,
		})
	}

	return rt
}

func (r *Runtime) Run(ctx context.Context, task *Task) (*Result, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}

	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	metrics.RecordTaskStart(task.RoomID)
	defer func() {
		metrics.RecordTaskComplete(string(task.Status), task.RoomID, time.Since(task.CreatedAt).Seconds())
	}()

	routeResult := r.router.Route(task.RoomID, task.UserID)
	if len(routeResult.Tools) == 0 {
		return r.executeWithoutTools(task)
	}

	for i := 0; i < r.config.MaxSteps; i++ {
		select {
		case <-ctx.Done():
			task.Status = TaskStatusCancelled
			return r.buildResult(task, "context cancelled")
		default:
		}

		step := NewStep(task.ID, StepTypeReason)
		task.AddStep(step)

		predictions := r.generatePredictions(routeResult, step)
		if len(predictions) > 0 && r.speculative != nil {
			r.speculative.AddPredictions(predictions)
		}

		toolCalls := r.extractToolCalls(step)
		if len(toolCalls) == 0 {
			return r.buildResult(task, "no tool calls extracted")
		}

		var results []*executor.ToolResult
		for _, call := range toolCalls {
			result, err := r.executor.Execute(ctx, call)
			if err != nil {
				task.Status = TaskStatusFailed
				return r.buildResult(task, fmt.Sprintf("tool execution failed: %v", err))
			}
			results = append(results, result)
		}

		for _, result := range results {
			task.AddStep(Step{
				Type:       StepTypeToolResult,
				ToolName:   result.Name,
				ToolOutput: result.Output,
				Duration:   result.Duration,
			})
		}
	}

	result := &Result{
		TaskID:     task.ID,
		Response:   "completed",
		ToolCalls:  len(task.Steps),
		TokensUsed: TokenUsage{},
		Duration:   time.Since(task.CreatedAt),
		Steps:      len(task.Steps),
		CompletedAt: time.Now(),
	}

	task.Status = TaskStatusCompleted
	task.Result = result

	return result, nil
}

func (r *Runtime) executeWithoutTools(task *Task) (*Result, error) {
    return &Result{
        TaskID:       task.ID,
        Response:    "no tools available for this request",
        ToolCalls:    0,
        Duration:    time.Since(task.CreatedAt),
        Steps:        len(task.Steps),
        CompletedAt:  time.Now(),
    }, nil
}

func (r *Runtime) generatePredictions(routeResult *router.RouteResult, step Step) []executor.ToolCall {
	return nil
}

func (r *Runtime) extractToolCalls(step Step) []executor.ToolCall {
	return nil
}

func (r *Runtime) buildResult(task *Task, errMsg string) (*Result, error) {
	return &Result{
		TaskID:      task.ID,
		Response:    "",
		TokensUsed:  TokenUsage{},
		Duration:    time.Since(task.CreatedAt),
		Steps:       len(task.Steps),
		CompletedAt: time.Now(),
	}, fmt.Errorf("task failed: %s", errMsg)
}

func (r *Runtime) Stop() {
	if r.executor != nil {
		r.executor.Close()
	}
	if r.speculative != nil {
		r.speculative.Close()
	}
	if r.cache != nil {
		r.cache.Clear()
	}
}
