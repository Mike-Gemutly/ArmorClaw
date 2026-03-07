package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/metrics"
	"github.com/armorclaw/bridge/internal/petg"
)

type ToolExecutor struct {
	mu       sync.RWMutex
	timeout  time.Duration
	pool     *ToolPool
	petg     *petg.Gateway
	skills   SkillRegistry
}

type ToolExecutorConfig struct {
	Timeout       time.Duration
	MaxWorkers    int
	PETG          *petg.Gateway
	SkillRegistry SkillRegistry
}

type SkillRegistry interface {
	GetSkill(name string) (*Skill, bool)
}

type Skill struct {
	Name       string
	Command    string
	Timeout    time.Duration
	Parameters map[string]Param
}

type Param struct {
	Type        string
	Description string
	Required    bool
}

type ToolCall struct {
	ID       string
	Name     string
	Args     map[string]interface{}
	Priority int
}

type ToolResult struct {
	CallID   string
	Name     string
	Output   string
	Error    error
	Duration time.Duration
	Cached   bool
}

func NewToolExecutor(cfg ToolExecutorConfig) *ToolExecutor {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 10
	}

	te := &ToolExecutor{
		timeout: cfg.Timeout,
		petg:    cfg.PETG,
		skills:  cfg.SkillRegistry,
	}

	te.pool = NewToolPool(ToolPoolConfig{
		MaxWorkers: cfg.MaxWorkers,
		Executor:   te.executeDirect,
	})

	return te
}

func (te *ToolExecutor) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	start := time.Now()

	if te.skills != nil {
		if _, exists := te.skills.GetSkill(call.Name); !exists {
			return nil, fmt.Errorf("unknown tool: %s", call.Name)
		}
	}

	args := call.Args
	if te.petg != nil {
		if err := te.petg.ValidateToolCall(ctx, call.Name, args); err != nil {
			return nil, fmt.Errorf("tool call rejected by security gateway: %w", err)
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, te.timeout)
	defer cancel()

	result, err := te.pool.Execute(execCtx, call)
	if err != nil {
		metrics.RecordToolCall(call.Name, "error", time.Since(start).Seconds())
		return nil, err
	}

	metrics.RecordToolCall(call.Name, "success", time.Since(start).Seconds())
	return result, nil
}

func (te *ToolExecutor) executeDirect(ctx context.Context, call ToolCall) (*ToolResult, error) {
	start := time.Now()

	switch call.Name {
	case "shell":
		return te.executeShell(ctx, call, start)
	default:
		return nil, fmt.Errorf("unsupported tool: %s", call.Name)
	}
}

func (te *ToolExecutor) executeShell(ctx context.Context, call ToolCall, start time.Time) (*ToolResult, error) {
	commandStr, ok := call.Args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'command' argument")
	}

	shell := "/bin/sh"
	if _, err := exec.LookPath("bash"); err == nil {
		shell = "/bin/bash"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", commandStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := &ToolResult{
		CallID:   call.ID,
		Name:     call.Name,
		Duration: duration,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("tool execution timed out after %v", te.timeout)
		} else {
			result.Error = fmt.Errorf("tool execution failed: %w", err)
		}
		result.Output = stderr.String()
	} else {
		result.Output = stdout.String()
	}

	if te.petg != nil {
		result.Output = te.petg.FilterOutput(result.Output)
	}

	return result, nil
}

func (te *ToolExecutor) Close() error {
	if te.pool != nil {
		return te.pool.Close()
	}
	return nil
}

type ToolPool struct {
	mu         sync.RWMutex
	tasks      chan poolTask
	workers    int
	maxWorkers int
	wg         sync.WaitGroup
	executor   func(ctx context.Context, call ToolCall) (*ToolResult, error)
}

type poolTask struct {
	ctx  context.Context
	call ToolCall
	resp chan *poolResponse
}

type poolResponse struct {
	result *ToolResult
	err    error
}

type ToolPoolConfig struct {
	MaxWorkers int
	Executor   func(ctx context.Context, call ToolCall) (*ToolResult, error)
}

func NewToolPool(cfg ToolPoolConfig) *ToolPool {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 10
	}

	p := &ToolPool{
		tasks:      make(chan poolTask, cfg.MaxWorkers*2),
		workers:    0,
		maxWorkers: cfg.MaxWorkers,
		executor:   cfg.Executor,
	}

	for i := 0; i < cfg.MaxWorkers/2; i++ {
		p.startWorker()
	}

	return p
}

func (p *ToolPool) startWorker() {
	p.mu.Lock()
	p.workers++
	p.mu.Unlock()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for task := range p.tasks {
			result, err := p.executor(task.ctx, task.call)
			task.resp <- &poolResponse{result: result, err: err}
		}
	}()
}

func (p *ToolPool) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	respChan := make(chan *poolResponse, 1)

	select {
	case p.tasks <- poolTask{ctx: ctx, call: call, resp: respChan}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case resp := <-respChan:
		return resp.result, resp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *ToolPool) ExecuteBatch(ctx context.Context, calls []ToolCall) ([]*ToolResult, error) {
	var wg sync.WaitGroup
	results := make([]*ToolResult, len(calls))
	errs := make([]error, len(calls))

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c ToolCall) {
			defer wg.Done()
			results[idx], errs[idx] = p.Execute(ctx, c)
		}(i, call)
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

func (p *ToolPool) Close() error {
	close(p.tasks)
	p.wg.Wait()
	return nil
}

func (te *ToolExecutor) ExecuteWithTimeout(ctx context.Context, call ToolCall, timeout time.Duration) (*ToolResult, error) {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return te.Execute(execCtx, call)
}

func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... (truncated)"
}

func parseCommand(cmdStr string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	var quoteChar rune

	for _, r := range cmdStr {
		switch {
		case r == '"' || r == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func readOutput(r io.Reader, maxBytes int64) (string, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxBytes {
		return string(data[:maxBytes]) + "\n... (truncated)", nil
	}
	return string(data), nil
}
