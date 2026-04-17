package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ExecutionMode string

const (
	ExecutionModeContainer   ExecutionMode = "container"
	ExecutionModeBridgeLocal ExecutionMode = "bridge_local"
)

type BridgeLocalHandler func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)

type BridgeLocalRegistry struct {
	mu       sync.RWMutex
	handlers map[string]BridgeLocalHandler
}

func NewBridgeLocalRegistry() *BridgeLocalRegistry {
	return &BridgeLocalRegistry{
		handlers: make(map[string]BridgeLocalHandler),
	}
}

func (r *BridgeLocalRegistry) Register(handlerName string, handler BridgeLocalHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[handlerName] = handler
}

func (r *BridgeLocalRegistry) Get(handlerName string) (BridgeLocalHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[handlerName]
	return h, ok
}

func (r *BridgeLocalRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

type StepConfig struct {
	ExecutionMode ExecutionMode   `json:"execution_mode"`
	Handler       string          `json:"handler,omitempty"`
	Timeout       int             `json:"timeout,omitempty"`
	Prompt        string          `json:"prompt,omitempty"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Extra         json.RawMessage `json:"-"`
}

func ParseStepConfig(raw json.RawMessage) (*StepConfig, error) {
	if len(raw) == 0 {
		return &StepConfig{ExecutionMode: ExecutionModeContainer}, nil
	}
	var cfg StepConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse step config: %w", err)
	}
	if cfg.ExecutionMode == "" {
		cfg.ExecutionMode = ExecutionModeContainer
	}
	return &cfg, nil
}

func ExecuteBridgeLocal(ctx context.Context, registry *BridgeLocalRegistry, config json.RawMessage) *StepResult {
	stepCfg, err := ParseStepConfig(config)
	if err != nil {
		return &StepResult{
			Err:         fmt.Errorf("parse config: %w", err),
			Recoverable: false,
		}
	}

	if stepCfg.Handler == "" {
		return &StepResult{
			Err:         fmt.Errorf("bridge_local step missing handler name"),
			Recoverable: false,
		}
	}

	handler, ok := registry.Get(stepCfg.Handler)
	if !ok {
		return &StepResult{
			Err:         fmt.Errorf("bridge_local handler not found: %s", stepCfg.Handler),
			Recoverable: false,
		}
	}

	timeout := time.Duration(stepCfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := handler(execCtx, config)
	if err != nil {
		return &StepResult{
			Err:         fmt.Errorf("bridge_local handler %s failed: %w", stepCfg.Handler, err),
			Recoverable: true,
		}
	}

	return &StepResult{
		ContainerResult: &ContainerStepResult{
			Status:     "success",
			Output:     string(result),
			DurationMS: 0,
		},
		Recoverable: false,
	}
}
