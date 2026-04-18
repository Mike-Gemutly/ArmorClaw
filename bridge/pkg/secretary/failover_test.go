package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/armorclaw/bridge/pkg/studio"
)

// failoverMockDocker implements studio.DockerClient for failover tests.
// Containers start Running=true. Fail-agents auto-transition to ExitCode=1 after a delay.
// Success-agents auto-transition to ExitCode=0 after a delay.
type failoverMockDocker struct {
	mu         sync.Mutex
	containers map[string]*types.ContainerState
	nextID     int
	failAgents map[string]bool
	tmpDir     string
	t          *testing.T
}

func newFailoverMockDocker(t *testing.T, failAgents map[string]bool, tmpDir string) *failoverMockDocker {
	return &failoverMockDocker{
		containers: make(map[string]*types.ContainerState),
		failAgents: failAgents,
		tmpDir:     tmpDir,
		t:          t,
	}
}

func (m *failoverMockDocker) ContainerCreate(_ context.Context, cfg *container.Config, _ *container.HostConfig, _ any, _ any, _ string) (container.CreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("failover-ctr-%d", m.nextID)
	m.containers[id] = &types.ContainerState{Running: true, ExitCode: 0}

	agentID := ""
	if cfg != nil && cfg.Labels != nil {
		agentID = cfg.Labels["armorclaw.agent_id"]
	}

	shouldFail := m.failAgents[agentID]

	go func() {
		time.Sleep(300 * time.Millisecond)
		m.mu.Lock()
		if s, ok := m.containers[id]; ok {
			s.Running = false
			if shouldFail {
				s.ExitCode = 1
			} else {
				s.ExitCode = 0
			}
			s.FinishedAt = "2025-01-01T00:00:00Z"
		}
		m.mu.Unlock()

		stateDir := filepath.Join(m.tmpDir, "agent-state", agentID)
		for i := 0; i < 50; i++ {
			if _, err := os.Stat(stateDir); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		writeAgentStateResult(m.t, stateDir, !shouldFail)
	}()

	return container.CreateResponse{ID: id}, nil
}

func (m *failoverMockDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *failoverMockDocker) ContainerStop(_ context.Context, cid string, _ container.StopOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *failoverMockDocker) ContainerKill(_ context.Context, cid string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *failoverMockDocker) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *failoverMockDocker) ContainerInspect(_ context.Context, cid string) (types.ContainerJSON, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.containers[cid]
	if !ok {
		state = &types.ContainerState{Running: true, ExitCode: 0}
	}
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    cid,
			State: state,
		},
	}, nil
}

func (m *failoverMockDocker) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return nil, nil
}

func setupFailoverTest(t *testing.T, failAgents map[string]bool, policy FailoverPolicy) (
	*StepExecutor,
	*failoverMockDocker,
	*Workflow,
	string,
) {
	t.Helper()

	tmpDir := t.TempDir()
	store, err := studio.NewStore(studio.StoreConfig{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	for agentID := range failAgents {
		def := &studio.AgentDefinition{
			ID:           agentID,
			Name:         fmt.Sprintf("Agent %s", agentID),
			Skills:       []string{"browser_navigate"},
			ResourceTier: "medium",
			CreatedBy:    "@test:example.com",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}
		require.NoError(t, store.CreateDefinition(def))
	}

	mockDocker := newFailoverMockDocker(t, failAgents, tmpDir)
	stateBase := filepath.Join(tmpDir, "agent-state")
	factory := studio.NewAgentFactory(studio.FactoryConfig{
		StateDir:     tmpDir,
		DockerClient: mockDocker,
		Store:        store,
	})

	executor := NewStepExecutor(StepExecutorConfig{
		Factory:        factory,
		StateDirBase:   stateBase,
		FailoverPolicy: policy,
		DefaultTimeout: 5 * time.Second,
		StepRetryCount: 0,
	})

	workflow := &Workflow{
		ID:         "wf-failover-test",
		Name:       "Failover Test",
		Status:     StatusRunning,
		CreatedBy:  "@test:example.com",
		RoomID:     "!test:example.com",
		StartedAt:  time.Now(),
		TemplateID: "tmpl-failover",
	}

	return executor, mockDocker, workflow, tmpDir
}

func writeAgentStateResult(t *testing.T, stateDir string, success bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	status := "success"
	output := "completed"
	if !success {
		status = "failed"
		output = "agent failed"
	}

	resultJSON, err := json.Marshal(ContainerStepResult{
		Status:     status,
		Output:     output,
		DurationMS: 300,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))

	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	f, err := os.Create(eventsPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestFailover_PrimaryFailFallbackSucceeds(t *testing.T) {
	executor, _, workflow, _ := setupFailoverTest(t, map[string]bool{
		"agent-fail":    true,
		"agent-success": false,
	}, FailoverRetry)

	step := WorkflowStep{
		StepID:   "step-failover-1",
		Order:    0,
		Type:     StepAction,
		Name:     "Failover Step",
		AgentIDs: []string{"agent-fail", "agent-success"},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := executor.executeStep(ctx, workflow, step)

	assert.NoError(t, result.Err, "should succeed after failover to agent-success")
	assert.Equal(t, "agent-success", result.AgentID, "should report the successful fallback agent")
	assert.Equal(t, "step-failover-1", result.StepID)
}

func TestFailover_AllAgentsExhausted(t *testing.T) {
	executor, _, workflow, _ := setupFailoverTest(t, map[string]bool{
		"agent-fail-1": true,
		"agent-fail-2": true,
		"agent-fail-3": true,
	}, FailoverRetry)

	step := WorkflowStep{
		StepID:   "step-all-fail",
		Order:    0,
		Type:     StepAction,
		Name:     "All Fail Step",
		AgentIDs: []string{"agent-fail-1", "agent-fail-2", "agent-fail-3"},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := executor.executeStep(ctx, workflow, step)

	require.Error(t, result.Err, "should fail when all agents are exhausted")

	var aggErr *FailoverAggregatedError
	require.ErrorAs(t, result.Err, &aggErr, "error should be FailoverAggregatedError")
	assert.Equal(t, "step-all-fail", aggErr.StepID)
	require.Len(t, aggErr.Attempts, 3, "should have 3 failover attempts")
	assert.Equal(t, "agent-fail-1", aggErr.Attempts[0].AgentID)
	assert.Equal(t, "agent-fail-2", aggErr.Attempts[1].AgentID)
	assert.Equal(t, "agent-fail-3", aggErr.Attempts[2].AgentID)
	assert.Equal(t, "agent-fail-3", aggErr.LastAgent)
	assert.ErrorIs(t, result.Err, ErrAllAgentsFailed)
}

func TestFailover_ImmediateFailPolicy(t *testing.T) {
	executor, _, workflow, _ := setupFailoverTest(t, map[string]bool{
		"agent-fail":    true,
		"agent-success": false,
	}, FailoverImmediateFail)

	step := WorkflowStep{
		StepID:   "step-immediate",
		Order:    0,
		Type:     StepAction,
		Name:     "Immediate Fail Step",
		AgentIDs: []string{"agent-fail", "agent-success"},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := executor.executeStep(ctx, workflow, step)

	require.Error(t, result.Err, "should fail immediately with FailoverImmediateFail policy")
	assert.Equal(t, "agent-fail", result.AgentID, "should only attempt primary agent")

	var aggErr *FailoverAggregatedError
	assert.NotErrorAs(t, result.Err, &aggErr, "should not wrap in aggregated error for immediate fail")
}

func TestFailover_SingleAgentSuccess(t *testing.T) {
	executor, _, workflow, _ := setupFailoverTest(t, map[string]bool{
		"agent-only": false,
	}, FailoverRetry)

	step := WorkflowStep{
		StepID:   "step-single",
		Order:    0,
		Type:     StepAction,
		Name:     "Single Agent Step",
		AgentIDs: []string{"agent-only"},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := executor.executeStep(ctx, workflow, step)

	assert.NoError(t, result.Err, "single agent should succeed")
	assert.Equal(t, "agent-only", result.AgentID)
}

func TestFailover_NoAgents(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{
		FailoverPolicy: FailoverRetry,
	})

	workflow := &Workflow{
		ID:     "wf-no-agents",
		Status: StatusRunning,
	}

	step := WorkflowStep{
		StepID:   "step-no-agents",
		Order:    0,
		Type:     StepAction,
		Name:     "No Agents Step",
		AgentIDs: []string{},
	}

	ctx := context.Background()
	result := executor.executeStep(ctx, workflow, step)

	assert.ErrorIs(t, result.Err, ErrNoAgentForStep)
	assert.False(t, result.Recoverable)
}

func TestFailoverAggregatedError_ErrorString(t *testing.T) {
	err := &FailoverAggregatedError{
		StepID: "step-1",
		Attempts: []FailoverAttempt{
			{AgentID: "agent-a", Err: fmt.Errorf("timeout")},
			{AgentID: "agent-b", Err: fmt.Errorf("crash")},
		},
		LastAgent: "agent-b",
	}

	msg := err.Error()
	assert.Contains(t, msg, "all agents failed")
	assert.Contains(t, msg, "step-1")
	assert.Contains(t, msg, "agent-a")
	assert.Contains(t, msg, "agent-b")
	assert.ErrorIs(t, err, ErrAllAgentsFailed)
}
