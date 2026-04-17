package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewBridgeLocalRegistry(t *testing.T) {
	r := NewBridgeLocalRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.handlers == nil {
		t.Error("handlers map should be initialized")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewBridgeLocalRegistry()
	called := false
	handler := func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`{"ok":true}`), nil
	}

	r.Register("test-handler", handler)

	got, ok := r.Get("test-handler")
	if !ok {
		t.Fatal("expected handler to be found")
	}

	result, err := got(context.Background(), nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if string(result) != `{"ok":true}` {
		t.Errorf("handler result = %q, want {\"ok\":true}", string(result))
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewBridgeLocalRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected handler not to be found")
	}
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	r := NewBridgeLocalRegistry()
	v1 := func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`"v1"`), nil
	}
	v2 := func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`"v2"`), nil
	}

	r.Register("handler", v1)
	r.Register("handler", v2)

	got, _ := r.Get("handler")
	result, _ := got(context.Background(), nil)
	if string(result) != `"v2"` {
		t.Errorf("after overwrite, expected v2, got %s", string(result))
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewBridgeLocalRegistry()
	noop := func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return nil, nil
	}

	r.Register("alpha", noop)
	r.Register("beta", noop)
	r.Register("gamma", noop)

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(names))
	}

	set := make(map[string]bool)
	for _, n := range names {
		set[n] = true
	}
	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !set[expected] {
			t.Errorf("expected %q in list", expected)
		}
	}
}

func TestRegistry_ListEmpty(t *testing.T) {
	r := NewBridgeLocalRegistry()
	names := r.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %d", len(names))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewBridgeLocalRegistry()
	noop := func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return nil, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := strings.Repeat("a", i%10+1)
			r.Register(name, noop)
			r.Get(name)
			r.List()
		}(i)
	}
	wg.Wait()
}

func TestParseStepConfig_ValidBridgeLocal(t *testing.T) {
	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "my-handler",
		"timeout": 60,
		"prompt": "do something",
		"max_tokens": 1024
	}`)

	cfg, err := ParseStepConfig(raw)
	if err != nil {
		t.Fatalf("ParseStepConfig: %v", err)
	}
	if cfg.ExecutionMode != ExecutionModeBridgeLocal {
		t.Errorf("ExecutionMode = %q, want bridge_local", cfg.ExecutionMode)
	}
	if cfg.Handler != "my-handler" {
		t.Errorf("Handler = %q, want my-handler", cfg.Handler)
	}
	if cfg.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", cfg.Timeout)
	}
	if cfg.Prompt != "do something" {
		t.Errorf("Prompt = %q, want 'do something'", cfg.Prompt)
	}
	if cfg.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", cfg.MaxTokens)
	}
}

func TestParseStepConfig_EmptyDefaults(t *testing.T) {
	raw := json.RawMessage(`{}`)

	cfg, err := ParseStepConfig(raw)
	if err != nil {
		t.Fatalf("ParseStepConfig: %v", err)
	}
	if cfg.ExecutionMode != ExecutionModeContainer {
		t.Errorf("default ExecutionMode = %q, want container", cfg.ExecutionMode)
	}
}

func TestParseStepConfig_NilInput(t *testing.T) {
	cfg, err := ParseStepConfig(nil)
	if err != nil {
		t.Fatalf("ParseStepConfig(nil): %v", err)
	}
	if cfg.ExecutionMode != ExecutionModeContainer {
		t.Errorf("default ExecutionMode = %q, want container", cfg.ExecutionMode)
	}
}

func TestParseStepConfig_EmptyInput(t *testing.T) {
	cfg, err := ParseStepConfig(json.RawMessage{})
	if err != nil {
		t.Fatalf("ParseStepConfig(empty): %v", err)
	}
	if cfg.ExecutionMode != ExecutionModeContainer {
		t.Errorf("default ExecutionMode = %q, want container", cfg.ExecutionMode)
	}
}

func TestParseStepConfig_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{not valid json}`)
	_, err := ParseStepConfig(raw)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse step config") {
		t.Errorf("error should wrap parse failure, got: %v", err)
	}
}

func TestExecuteBridgeLocal_Success(t *testing.T) {
	r := NewBridgeLocalRegistry()
	r.Register("echo", func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"echo":"pong"}`), nil
	})

	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "echo",
		"timeout": 5
	}`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.ContainerResult == nil {
		t.Fatal("expected ContainerResult")
	}
	if result.ContainerResult.Status != "success" {
		t.Errorf("Status = %q, want success", result.ContainerResult.Status)
	}
	if result.ContainerResult.Output != `{"echo":"pong"}` {
		t.Errorf("Output = %q, want echo pong", result.ContainerResult.Output)
	}
	if result.Recoverable {
		t.Error("success should not be recoverable")
	}
}

func TestExecuteBridgeLocal_MissingHandler(t *testing.T) {
	r := NewBridgeLocalRegistry()
	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": ""
	}`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err == nil {
		t.Fatal("expected error for missing handler")
	}
	if !strings.Contains(result.Err.Error(), "missing handler name") {
		t.Errorf("error = %v, want missing handler", result.Err)
	}
	if result.Recoverable {
		t.Error("config error should not be recoverable")
	}
}

func TestExecuteBridgeLocal_HandlerNotFound(t *testing.T) {
	r := NewBridgeLocalRegistry()
	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "nonexistent"
	}`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err == nil {
		t.Fatal("expected error for handler not found")
	}
	if !strings.Contains(result.Err.Error(), "handler not found") {
		t.Errorf("error = %v, want handler not found", result.Err)
	}
	if result.Recoverable {
		t.Error("missing handler should not be recoverable")
	}
}

func TestExecuteBridgeLocal_InvalidConfig(t *testing.T) {
	r := NewBridgeLocalRegistry()
	raw := json.RawMessage(`{invalid`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err == nil {
		t.Fatal("expected error for invalid config")
	}
	if result.Recoverable {
		t.Error("parse error should not be recoverable")
	}
}

func TestExecuteBridgeLocal_HandlerError(t *testing.T) {
	r := NewBridgeLocalRegistry()
	r.Register("failing", func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		return nil, errors.New("something went wrong")
	})

	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "failing"
	}`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err == nil {
		t.Fatal("expected error from handler")
	}
	if !strings.Contains(result.Err.Error(), "something went wrong") {
		t.Errorf("error = %v, want something went wrong", result.Err)
	}
	if !result.Recoverable {
		t.Error("handler errors should be recoverable")
	}
}

func TestExecuteBridgeLocal_DefaultTimeout(t *testing.T) {
	r := NewBridgeLocalRegistry()
	r.Register("slow", func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("expected context to have a deadline")
			return nil, nil
		}
		maxAllowed := time.Now().Add(301 * time.Second)
		if deadline.After(maxAllowed) {
			t.Errorf("default timeout too high: deadline %v", deadline)
		}
		return json.RawMessage(`"done"`), nil
	})

	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "slow"
	}`)

	result := ExecuteBridgeLocal(context.Background(), r, raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteBridgeLocal_ContextCancellation(t *testing.T) {
	r := NewBridgeLocalRegistry()
	r.Register("blocked", func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	raw := json.RawMessage(`{
		"execution_mode": "bridge_local",
		"handler": "blocked",
		"timeout": 60
	}`)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := ExecuteBridgeLocal(ctx, r, raw)
	if result.Err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !result.Recoverable {
		t.Error("handler errors should be recoverable")
	}
}
