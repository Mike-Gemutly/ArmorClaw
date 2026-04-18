package secretary

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/armorclaw/bridge/pkg/capability"
)

// mockBrokerCheck captures broker calls and returns a configured response.
type mockBrokerCheck struct {
	mu       sync.Mutex
	response capability.ActionResponse
	err      error
	calls    []capability.ActionRequest
}

func newMockBrokerCheck(resp capability.ActionResponse, err error) *mockBrokerCheck {
	return &mockBrokerCheck{response: resp, err: err}
}

func (m *mockBrokerCheck) Authorize(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, req)
	return m.response, m.err
}

func (m *mockBrokerCheck) getCalls() []capability.ActionRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]capability.ActionRequest, len(m.calls))
	copy(out, m.calls)
	return out
}

// TestBrokerCheck_StepAllowed verifies that when the broker returns ALLOW,
// executeStep proceeds past the broker check and reaches the "no agent" error
// (since no factory or agents are configured).
func TestBrokerCheck_StepAllowed(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{
		Allowed:        true,
		Classification: capability.RiskAllow,
	}, nil)

	exec := NewStepExecutor(StepExecutorConfig{Broker: broker})
	workflow := &Workflow{ID: "wf-broker-allow"}
	step := WorkflowStep{StepID: "step-allow", Name: "browse_page"}

	result := exec.executeStep(context.Background(), workflow, step)

	assert.NotNil(t, result)
	assert.Equal(t, ErrNoAgentForStep, result.Err, "broker allows but no agents → ErrNoAgentForStep")

	calls := broker.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "wf-broker-allow", calls[0].TeamID)
	assert.Equal(t, "browse_page", calls[0].Action)
}

// TestBrokerCheck_StepDenied verifies that when the broker returns DENY,
// executeStep returns immediately with a denial error.
func TestBrokerCheck_StepDenied(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{
		Allowed:        false,
		Classification: capability.RiskDeny,
		Reason:         "payment requires approval",
	}, nil)

	exec := NewStepExecutor(StepExecutorConfig{
		Broker: broker,
	})

	workflow := &Workflow{ID: "wf-broker-deny"}
	step := WorkflowStep{
		StepID:   "step-deny",
		Name:     "charge_credit_card",
		AgentIDs: []string{"agent-2"},
	}

	result := exec.executeStep(context.Background(), workflow, step)

	require.NotNil(t, result)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "step-deny")
	assert.Contains(t, result.Err.Error(), "capability denied: payment requires approval")
	assert.False(t, result.Recoverable, "broker denial should not be recoverable")
}

// TestBrokerCheck_BrokerError verifies that when the broker returns an error,
// executeStep returns a denial error with the broker error message.
func TestBrokerCheck_BrokerError(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{}, errors.New("db connection lost"))

	exec := NewStepExecutor(StepExecutorConfig{
		Broker: broker,
	})

	workflow := &Workflow{ID: "wf-broker-err"}
	step := WorkflowStep{
		StepID:   "step-err",
		Name:     "send_email",
		AgentIDs: []string{"agent-3"},
	}

	result := exec.executeStep(context.Background(), workflow, step)

	require.NotNil(t, result)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "step-err")
	assert.Contains(t, result.Err.Error(), "broker error: db connection lost")
	assert.False(t, result.Recoverable)
}

// TestBrokerCheck_NilBroker_BackwardCompat verifies that when no broker is
// configured (nil), executeStep skips the broker check entirely and proceeds
// to agent execution (backward compatible).
func TestBrokerCheck_NilBroker_BackwardCompat(t *testing.T) {
	exec := NewStepExecutor(StepExecutorConfig{Broker: nil})
	workflow := &Workflow{ID: "wf-no-broker"}
	step := WorkflowStep{StepID: "step-no-broker", Name: "fetch_url"}

	result := exec.executeStep(context.Background(), workflow, step)

	assert.NotNil(t, result)
	assert.Equal(t, ErrNoAgentForStep, result.Err)
}

// TestBrokerCheck_ContextCancelled verifies that a cancelled context is
// handled properly when passed to the broker check.
func TestBrokerCheck_ContextCancelled(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{
		Allowed:        true,
		Classification: capability.RiskAllow,
	}, nil)

	exec := NewStepExecutor(StepExecutorConfig{Broker: broker})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	workflow := &Workflow{ID: "wf-cancelled"}
	step := WorkflowStep{StepID: "step-cancelled", Name: "risky_action"}

	assert.NotPanics(t, func() {
		exec.executeStep(ctx, workflow, step)
	})
}

// TestBrokerCheck_ActionRequestFields verifies that the ActionRequest sent to
// the broker contains the correct TeamID, Action, Params, and AgentID from
// the step being executed.
func TestBrokerCheck_ActionRequestFields(t *testing.T) {
	var captured capability.ActionRequest
	broker := &trackingBroker{
		authorize: func(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
			captured = req
			return capability.ActionResponse{
				Allowed: false, Classification: capability.RiskDeny, Reason: "captured",
			}, nil
		},
	}

	exec := NewStepExecutor(StepExecutorConfig{Broker: broker})
	workflow := &Workflow{ID: "wf-fields"}
	step := WorkflowStep{
		StepID:   "step-fields",
		Name:     "submit_form",
		Input:    map[string]any{"url": "https://example.com", "field": "email"},
		AgentIDs: []string{"agent-6", "agent-7"},
	}

	result := exec.executeStep(context.Background(), workflow, step)
	require.Error(t, result.Err)

	assert.Equal(t, "wf-fields", captured.TeamID)
	assert.Equal(t, "submit_form", captured.Action)
	assert.Equal(t, "agent-6", captured.AgentID)
	assert.Equal(t, map[string]any{"url": "https://example.com", "field": "email"}, captured.Params)
}

// TestBrokerCheck_SequentialSteps_AllowThenDeny verifies that in a multi-step
// workflow, the broker is called for each step independently, and a DENY on
// step 2 does not affect step 1's result.
func TestBrokerCheck_SequentialSteps_AllowThenDeny(t *testing.T) {
	callCount := 0
	broker := &trackingBroker{
		authorize: func(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
			callCount++
			if callCount == 1 {
				return capability.ActionResponse{Allowed: true, Classification: capability.RiskAllow}, nil
			}
			return capability.ActionResponse{
				Allowed: false, Classification: capability.RiskDeny, Reason: "too risky",
			}, nil
		},
	}

	exec := NewStepExecutor(StepExecutorConfig{Broker: broker})
	workflow := &Workflow{ID: "wf-sequential"}

	result1 := exec.executeStep(context.Background(), workflow, WorkflowStep{
		StepID: "step-1", Name: "browse",
	})
	assert.Equal(t, ErrNoAgentForStep, result1.Err, "step-1 passes broker but no agents")

	result2 := exec.executeStep(context.Background(), workflow, WorkflowStep{
		StepID: "step-2", Name: "payment",
	})
	assert.Error(t, result2.Err)
	assert.Contains(t, result2.Err.Error(), "capability denied: too risky")

	assert.Equal(t, 2, callCount)
}

// trackingBroker is a flexible mock that delegates to a function.
type trackingBroker struct {
	authorize func(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error)
}

func (tb *trackingBroker) Authorize(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
	return tb.authorize(ctx, req)
}

// TestBrokerCheck_DeferClassification verifies that a DEFER response (not ALLOW)
// is treated as a denial — the step does not proceed.
func TestBrokerCheck_DeferClassification(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{
		Allowed:        false,
		Classification: capability.RiskDefer,
		Reason:         "requires human approval",
	}, nil)

	exec := NewStepExecutor(StepExecutorConfig{
		Broker: broker,
	})

	workflow := &Workflow{ID: "wf-defer"}
	step := WorkflowStep{
		StepID:   "step-defer",
		Name:     "access_pii",
		AgentIDs: []string{"agent-7"},
	}

	result := exec.executeStep(context.Background(), workflow, step)

	require.NotNil(t, result)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "capability denied: requires human approval")
	assert.False(t, result.Recoverable)
}

// TestBrokerCheck_NoAgentIDs_PassesBrokerButNoAgent verifies that even when
// broker allows, a step with no AgentIDs returns ErrNoAgentForStep.
func TestBrokerCheck_NoAgentIDs_PassesBrokerButNoAgent(t *testing.T) {
	broker := newMockBrokerCheck(capability.ActionResponse{
		Allowed:        true,
		Classification: capability.RiskAllow,
	}, nil)

	exec := NewStepExecutor(StepExecutorConfig{
		Broker: broker,
	})

	workflow := &Workflow{ID: "wf-no-agents"}
	step := WorkflowStep{
		StepID: "step-no-agents",
		Name:   "do_something",
		// No AgentIDs
	}

	result := exec.executeStep(context.Background(), workflow, step)

	require.NotNil(t, result)
	assert.Error(t, result.Err)
	assert.Equal(t, ErrNoAgentForStep, result.Err)
}

// TestBrokerCheck_ConcurrentSteps tests that multiple goroutines can call
// executeStep with the broker simultaneously without races.
func TestBrokerCheck_ConcurrentSteps_NoRace(t *testing.T) {
	var mu sync.Mutex
	completed := 0
	broker := &trackingBroker{
		authorize: func(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
			return capability.ActionResponse{Allowed: true, Classification: capability.RiskAllow}, nil
		},
	}

	exec := NewStepExecutor(StepExecutorConfig{Broker: broker})
	workflow := &Workflow{ID: "wf-concurrent"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			step := WorkflowStep{
				StepID: fmt.Sprintf("step-%d", idx),
				Name:   "concurrent_action",
			}
			result := exec.executeStep(context.Background(), workflow, step)
			_ = result
			mu.Lock()
			completed++
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 10, completed)
}
