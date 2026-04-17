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

//=============================================================================
// Orchestrator Integration Wiring Tests
//=============================================================================

func setupTestIntegration(t *testing.T) (*OrchestratorIntegration, *orchestratorTestStore, *mockEventEmitter) {
	store := newOrchestratorTestStore()
	emitter := newMockEventEmitter()

	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	require.NoError(t, err)

	executor := NewStepExecutor(StepExecutorConfig{})

	integration := NewOrchestratorIntegration(IntegrationConfig{
		Orchestrator: orch,
		Executor:     executor,
		Store:        store,
	})

	return integration, store, emitter
}

func TestStartWorkflowExecution_UsesRoomID(t *testing.T) {
	integration, store, _ := setupTestIntegration(t)

	workflow := &Workflow{
		ID:         "wf-room",
		TemplateID: "",
		Name:       "RoomID Test",
		Status:     StatusRunning,
		CreatedBy:  "@creator:example.com",
		RoomID:     "!target:example.com",
		StartedAt:  time.Now(),
	}
	store.workflows["wf-room"] = workflow

	err := integration.StartWorkflowExecution("wf-room")
	assert.NoError(t, err)

	status := integration.GetExecutionStatus("wf-room")
	assert.True(t, status.IsExecuting, "workflow should be executing")
}

func TestStartWorkflowExecution_WorkflowNotRunning(t *testing.T) {
	integration, store, _ := setupTestIntegration(t)

	workflow := &Workflow{
		ID:        "wf-pending",
		Name:      "Pending Workflow",
		Status:    StatusPending,
		CreatedBy: "@test:example.com",
	}
	store.workflows["wf-pending"] = workflow

	err := integration.StartWorkflowExecution("wf-pending")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestStartWorkflowExecution_AlreadyExecuting(t *testing.T) {
	integration, store, _ := setupTestIntegration(t)

	workflow := &Workflow{
		ID:        "wf-dup",
		Name:      "Dup Workflow",
		Status:    StatusRunning,
		CreatedBy: "@test:example.com",
		StartedAt: time.Now(),
	}
	store.workflows["wf-dup"] = workflow

	err := integration.StartWorkflowExecution("wf-dup")
	require.NoError(t, err)

	err = integration.StartWorkflowExecution("wf-dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already executing")
}

func TestExecuteStep_EmptyRoomID_NoPanic(t *testing.T) {
	integration, store, _ := setupTestIntegration(t)

	workflow := &Workflow{
		ID:        "wf-empty-room",
		Name:      "Empty Room",
		Status:    StatusRunning,
		CreatedBy: "@test:example.com",
		RoomID:    "",
		StartedAt: time.Now(),
	}
	store.workflows["wf-empty-room"] = workflow

	assert.NotPanics(t, func() {
		integration.StartWorkflowExecution("wf-empty-room")
	})

	time.Sleep(100 * time.Millisecond)

	status := integration.GetExecutionStatus("wf-empty-room")
	assert.False(t, status.IsExecuting, "workflow with no steps should complete quickly")
}

func TestCancelWorkflowExecution(t *testing.T) {
	integration, store, _ := setupTestIntegration(t)

	workflow := &Workflow{
		ID:        "wf-cancel",
		Name:      "Cancel Me",
		Status:    StatusRunning,
		CreatedBy: "@test:example.com",
		StartedAt: time.Now(),
	}
	store.workflows["wf-cancel"] = workflow

	err := integration.StartWorkflowExecution("wf-cancel")
	require.NoError(t, err)

	status := integration.GetExecutionStatus("wf-cancel")
	assert.True(t, status.IsExecuting)

	err = integration.CancelWorkflowExecution("wf-cancel")
	assert.NoError(t, err)
}

func TestNewOrchestratorIntegration_NilFields(t *testing.T) {
	integration := NewOrchestratorIntegration(IntegrationConfig{})

	assert.NotPanics(t, func() {
		status := integration.GetExecutionStatus("nonexistent")
		assert.False(t, status.IsExecuting)
	})

	assert.NotPanics(t, func() {
		integration.CancelWorkflowExecution("nonexistent")
	})
}

//=============================================================================
// Event Tailing Integration Tests
//=============================================================================

func TestTailEventsDuringExecution(t *testing.T) {
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "_events.jsonl")

	reader := NewEventReader(tmpDir)

	// Initially no events (file doesn't exist yet).
	events, _, err := reader.ReadNew()
	require.NoError(t, err)
	assert.Nil(t, events)

	// Simulate container writing events incrementally across 5+ poll cycles.
	// Each cycle appends new events; ReadNew must return only new ones.
	var allSeen []StepEvent
	uniqueSeqs := make(map[int]bool)

	for cycle := 0; cycle < 6; cycle++ {
		for i := 0; i < 3; i++ {
			seq := cycle*3 + i + 1
			evt := StepEvent{
				Seq:  seq,
				Type: "action",
				Name: fmt.Sprintf("step_%d", seq),
				TsMs: time.Now().UnixMilli(),
			}
			line, err := json.Marshal(evt)
			require.NoError(t, err)

			f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			require.NoError(t, err)
			_, err = f.WriteString(string(line) + "\n")
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}

		newEvents, _, err := reader.ReadNew()
		require.NoError(t, err)
		require.Len(t, newEvents, 3, "poll cycle %d: expected exactly 3 new events", cycle)

		for _, evt := range newEvents {
			assert.False(t, uniqueSeqs[evt.Seq],
				"duplicate seq %d detected across %d poll cycles", evt.Seq, cycle+1)
			uniqueSeqs[evt.Seq] = true
		}

		allSeen = append(allSeen, newEvents...)
	}

	assert.Len(t, allSeen, 18, "expected 18 total events across 6 cycles")
	assert.Len(t, uniqueSeqs, 18, "expected 18 unique sequence numbers")

	// Subsequent ReadNew after all writes should return zero events.
	dupes, _, err := reader.ReadNew()
	require.NoError(t, err)
	assert.Nil(t, dupes, "no new events should be returned after steady state")
}

func TestTailEvents_OverflowKillsContainer(t *testing.T) {
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "_events.jsonl")

	reader := NewEventReader(tmpDir)

	// Write a file larger than 10MB.
	bigLine := make([]byte, 1024)
	for i := range bigLine {
		bigLine[i] = 'A'
	}

	f, err := os.Create(eventsPath)
	require.NoError(t, err)
	for i := 0; i < 10*1024+1; i++ { // > 10MB of 1KB lines
		_, err := f.Write(append(bigLine, '\n'))
		require.NoError(t, err)
	}
	require.NoError(t, f.Close())

	_, _, err = reader.ReadNew()
	assert.ErrorIs(t, err, ErrEventLogExceeded,
		"ReadNew must return ErrEventLogExceeded for >10MB file")
}

func TestTailEvents_ParseExtendedAfterCompletion(t *testing.T) {
	tmpDir := t.TempDir()

	// Write result.json.
	result := ContainerStepResult{
		Status:     "success",
		Output:     "task completed",
		DurationMS: 1234,
	}
	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "result.json"), resultJSON, 0644))

	// Write _events.jsonl with multiple events.
	eventsPath := filepath.Join(tmpDir, "_events.jsonl")
	f, err := os.Create(eventsPath)
	require.NoError(t, err)
	for i := 1; i <= 5; i++ {
		evt := StepEvent{
			Seq:  i,
			Type: "action",
			Name: fmt.Sprintf("action_%d", i),
			TsMs: time.Now().UnixMilli(),
		}
		line, _ := json.Marshal(evt)
		f.WriteString(string(line) + "\n")
	}
	require.NoError(t, f.Close())

	// ParseExtendedStepResult should combine result.json + _events.jsonl.
	ext, err := ParseExtendedStepResult(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, ext)
	assert.Equal(t, "success", ext.Status)
	assert.Len(t, ext.Events, 5, "should have all 5 events from _events.jsonl")
	assert.Equal(t, 1, ext.Events[0].Seq)
	assert.Equal(t, 5, ext.Events[4].Seq)

	// cleanupStateDir should remove the directory.
	require.NoError(t, cleanupStateDir(tmpDir))
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err), "state dir should be removed after cleanup")
}

func TestTailEvents_CleanupOnCancel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a marker file to confirm cleanup occurs.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "result.json"), []byte(`{"status":"ok"}`), 0644))

	// cleanupStateDir on a valid path should remove it.
	require.NoError(t, cleanupStateDir(tmpDir))
	_, err := os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))

	// cleanupStateDir on empty string or nonexistent path is a no-op.
	assert.NoError(t, cleanupStateDir(""))
	assert.NoError(t, cleanupStateDir("/nonexistent/path/12345"))
}

//=============================================================================
// Purge Order Tests — verify Parse → Cleanup → Notify ordering
// and state-directory absence after every exit path.
//=============================================================================

// purgeMockDocker implements studio.DockerClient for purge order tests.
// It allows controlling the container inspect result dynamically.
type purgeMockDocker struct {
	mu             sync.Mutex
	containerState *types.ContainerState
	stopped        []string
	killed         []struct {
		id     string
		signal string
	}
}

func (m *purgeMockDocker) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ any, _ any, _ string) (container.CreateResponse, error) {
	return container.CreateResponse{ID: "purge-mock-container"}, nil
}

func (m *purgeMockDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *purgeMockDocker) ContainerStop(_ context.Context, cid string, _ container.StopOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, cid)
	return nil
}

func (m *purgeMockDocker) ContainerKill(_ context.Context, cid string, signal string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed = append(m.killed, struct {
		id     string
		signal string
	}{id: cid, signal: signal})
	return nil
}

func (m *purgeMockDocker) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *purgeMockDocker) ContainerInspect(_ context.Context, _ string) (types.ContainerJSON, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.containerState
	if state == nil {
		state = &types.ContainerState{Running: true, ExitCode: 0}
	}
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    "purge-mock-container",
			State: state,
		},
	}, nil
}

func (m *purgeMockDocker) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return nil, nil
}

func (m *purgeMockDocker) setState(state *types.ContainerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containerState = state
}

func (m *purgeMockDocker) getStopped() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func (m *purgeMockDocker) getKilled() []struct {
	id     string
	signal string
} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.killed
}

// setupPurgeTest creates a factory with mock Docker, an in-memory store,
// a pre-seeded definition and running instance, plus a StepExecutor.
func setupPurgeTest(t *testing.T, initialState *types.ContainerState) (
	*studio.AgentFactory,
	*purgeMockDocker,
	*studio.SQLiteStore,
	string, // instanceID
	*StepExecutor,
) {
	t.Helper()

	store, err := studio.NewStore(studio.StoreConfig{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	def := &studio.AgentDefinition{
		ID:           "purge-test-agent",
		Name:         "Purge Test Agent",
		Skills:       []string{"browser_navigate"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	require.NoError(t, store.CreateDefinition(def))

	// Spawn via factory to get a real instance in the store.
	mockDocker := &purgeMockDocker{containerState: initialState}
	factory := studio.NewAgentFactory(studio.FactoryConfig{
		StateDir:     os.TempDir(),
		DockerClient: mockDocker,
		Store:        store,
	})

	ctx := context.Background()
	result, err := factory.Spawn(ctx, &studio.SpawnRequest{
		DefinitionID: def.ID,
		UserID:       "@test:example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Instance)

	instanceID := result.Instance.ID

	executor := NewStepExecutor(StepExecutorConfig{
		Factory: factory,
	})

	return factory, mockDocker, store, instanceID, executor
}

// writeStateDirFiles creates result.json and _events.jsonl in the state dir.
func writeStateDirFiles(t *testing.T, stateDir string, resultJSON []byte) {
	t.Helper()
	if resultJSON != nil {
		require.NoError(t, os.MkdirAll(stateDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))
	}
	// Always create _events.jsonl (even if empty) so EventReader has something.
	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	f, err := os.Create(eventsPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestPurgeOrder_CompletionPath(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	// Use a real temp dir (not t.TempDir which auto-cleans) because we
	// need to verify cleanup.  We clean up manually in the assertion.
	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-complete-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) }) // safety net

	resultJSON, err := json.Marshal(ContainerStepResult{
		Status:     "success",
		Output:     "task completed",
		DurationMS: 500,
	})
	require.NoError(t, err)
	writeStateDirFiles(t, stateDir, resultJSON)

	// After 800ms, transition Docker state to completed.
	go func() {
		time.Sleep(800 * time.Millisecond)
		mockDocker.setState(&types.ContainerState{Running: false, ExitCode: 0})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	require.NoError(t, waitErr, "waitForCompletion should succeed on completion path")
	require.NotNil(t, completionResult, "CompletionResult should not be nil")
	assert.Equal(t, 0, completionResult.ExitCode, "exit code should be 0 on success")
	require.NotNil(t, completionResult.ExtendedResult, "ExtendedResult should be parsed before cleanup")
	assert.Equal(t, "success", completionResult.ExtendedResult.Status)

	// State directory must be gone after waitForCompletion returns.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up after completion: statErr=%v", statErr)
}

func TestPurgeOrder_FailurePath(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-fail-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	resultJSON, err := json.Marshal(ContainerStepResult{
		Status: "failed",
		Error:  "something went wrong",
	})
	require.NoError(t, err)
	writeStateDirFiles(t, stateDir, resultJSON)

	// After 800ms, transition to failed (non-zero exit code).
	go func() {
		time.Sleep(800 * time.Millisecond)
		mockDocker.setState(&types.ContainerState{Running: false, ExitCode: 1})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	require.NoError(t, waitErr, "waitForCompletion should not return error on failure path")
	require.NotNil(t, completionResult, "CompletionResult should not be nil")
	assert.Equal(t, 1, completionResult.ExitCode, "exit code should be 1 on failure")
	require.NotNil(t, completionResult.ExtendedResult, "ExtendedResult should be parsed before cleanup")
	assert.Equal(t, "failed", completionResult.ExtendedResult.Status)

	// State directory must be gone.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up after failure: statErr=%v", statErr)
}

func TestPurgeOrder_CancelPath(t *testing.T) {
	_, _, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-cancel-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	// Write a small events file so state dir has content.
	writeStateDirFiles(t, stateDir, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context almost immediately.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	assert.ErrorIs(t, waitErr, context.Canceled, "should return context.Canceled")
	assert.Nil(t, completionResult, "CompletionResult should be nil on cancel")

	// State directory must be gone even on cancel.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up after cancel: statErr=%v", statErr)
}

func TestPurgeOrder_10MBKillPath(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-kill-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	// Create _events.jsonl larger than 10MB.
	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	bigLine := make([]byte, 1024)
	for i := range bigLine {
		bigLine[i] = 'X'
	}
	f, err := os.Create(eventsPath)
	require.NoError(t, err)
	for i := 0; i < 10*1024+1; i++ { // > 10MB of 1KB lines
		_, err := f.Write(append(bigLine, '\n'))
		require.NoError(t, err)
	}
	require.NoError(t, f.Close())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	require.Error(t, waitErr, "should return error for 10MB overflow")
	assert.Contains(t, waitErr.Error(), "event log exceeded", "error should mention 10MB cap")

	// Kill() should have been called (not Stop).
	assert.NotEmpty(t, mockDocker.getKilled(), "Kill() should be called on 10MB overflow")
	assert.Empty(t, mockDocker.getStopped(), "Stop() should NOT be called on 10MB overflow (Kill instead)")

	// CompletionResult should be nil (error path).
	assert.Nil(t, completionResult, "CompletionResult should be nil on kill path")

	// State directory must be gone even on kill path.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up after 10MB kill: statErr=%v", statErr)
}

func TestPurgeOrder_CommentsAfterPurge(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-comments-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	// result.json with _comments field.
	type extendedResult struct {
		ContainerStepResult
		Comments []string `json:"_comments"`
	}
	resultJSON, err := json.Marshal(extendedResult{
		ContainerStepResult: ContainerStepResult{
			Status:     "success",
			Output:     "comments test",
			DurationMS: 300,
		},
		Comments: []string{"First observation", "Second observation", "Final note"},
	})
	require.NoError(t, err)
	writeStateDirFiles(t, stateDir, resultJSON)

	// After 800ms, transition to completed.
	go func() {
		time.Sleep(800 * time.Millisecond)
		mockDocker.setState(&types.ContainerState{Running: false, ExitCode: 0})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	require.NoError(t, waitErr)
	require.NotNil(t, completionResult)
	require.NotNil(t, completionResult.ExtendedResult, "ExtendedResult must be parsed before purge")

	// Comments should be available in memory even though state dir is gone.
	assert.Equal(t, []string{"First observation", "Second observation", "Final note"},
		completionResult.ExtendedResult.Comments,
		"comments must be available after state dir purge")

	// State directory must be gone — data lives only in memory now.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up: statErr=%v", statErr)
}

//=============================================================================
// Blocker Loop Tests
//=============================================================================

// blockerMockDocker manages sequential container lifecycle for blocker tests.
// Each ContainerCreate returns a new ID; transitions are controlled externally.
type blockerMockDocker struct {
	mu           sync.Mutex
	containers   map[string]*types.ContainerState
	nextID       int
	stopped      []string
	killed       []string
	transitionCh chan string // signals which container to transition
}

func newBlockerMockDocker() *blockerMockDocker {
	return &blockerMockDocker{
		containers:   make(map[string]*types.ContainerState),
		transitionCh: make(chan string, 10),
	}
}

func (m *blockerMockDocker) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ any, _ any, _ string) (container.CreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("blocker-container-%d", m.nextID)
	m.containers[id] = &types.ContainerState{Running: true, ExitCode: 0}
	return container.CreateResponse{ID: id}, nil
}

func (m *blockerMockDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *blockerMockDocker) ContainerStop(_ context.Context, cid string, _ container.StopOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, cid)
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *blockerMockDocker) ContainerKill(_ context.Context, cid string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed = append(m.killed, cid)
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *blockerMockDocker) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *blockerMockDocker) ContainerInspect(_ context.Context, cid string) (types.ContainerJSON, error) {
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

func (m *blockerMockDocker) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return nil, nil
}

func (m *blockerMockDocker) completeContainer(cid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
		s.ExitCode = 0
		s.FinishedAt = "2025-01-01T00:00:00Z"
	}
}

// setupBlockerTest creates all the plumbing for blocker loop tests.
func setupBlockerTest(t *testing.T) (
	*StepExecutor,
	*WorkflowOrchestratorImpl,
	*blockerMockDocker,
	*orchestratorTestStore,
	string,
	string,
	string,
) {
	t.Helper()

	tmpDir := t.TempDir()
	store := newOrchestratorTestStore()
	emitter := newMockEventEmitter()

	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	require.NoError(t, err)

	studioStore, err := studio.NewStore(studio.StoreConfig{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { studioStore.Close() })

	def := &studio.AgentDefinition{
		ID:           "blocker-test-agent",
		Name:         "Blocker Test Agent",
		Skills:       []string{"browser_navigate"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	require.NoError(t, studioStore.CreateDefinition(def))

	mockDocker := newBlockerMockDocker()
	stateBase := filepath.Join(tmpDir, "agent-state")
	factory := studio.NewAgentFactory(studio.FactoryConfig{
		StateDir:     tmpDir,
		DockerClient: mockDocker,
		Store:        studioStore,
	})

	executor := NewStepExecutor(StepExecutorConfig{
		Factory:      factory,
		StateDirBase: stateBase,
	})

	workflow := &Workflow{
		ID:        "wf-blocker-test",
		Name:      "Blocker Test",
		Status:    StatusRunning,
		CreatedBy: "@test:example.com",
		RoomID:    "!test:example.com",
		StartedAt: time.Now(),
	}
	store.workflows[workflow.ID] = workflow

	orch.mu.Lock()
	orch.activeWorkflows[workflow.ID] = &activeWorkflow{
		workflow:   workflow,
		cancelFunc: func() {},
		startedAt:  time.Now(),
	}
	orch.mu.Unlock()

	step := WorkflowStep{
		StepID:   "step-1",
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{def.ID},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}

	return executor, orch, mockDocker, store, workflow.ID, step.StepID, tmpDir
}

// writeBlockerResult writes a result.json with blockers to the state dir.
func writeBlockerResult(t *testing.T, stateDir string, blockers []Blocker) {
	t.Helper()
	type extendedResult struct {
		ContainerStepResult
		Blockers []Blocker `json:"_blockers"`
	}
	resultJSON, err := json.Marshal(extendedResult{
		ContainerStepResult: ContainerStepResult{
			Status:     "blocked",
			Output:     "needs input",
			DurationMS: 500,
		},
		Blockers: blockers,
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))

	// Create empty _events.jsonl so EventReader doesn't error
	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		f, err := os.Create(eventsPath)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
}

// writeSuccessResult writes a result.json without blockers.
func writeSuccessResult(t *testing.T, stateDir string) {
	t.Helper()
	resultJSON, err := json.Marshal(ContainerStepResult{
		Status:     "success",
		Output:     "completed",
		DurationMS: 300,
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))

	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		f, err := os.Create(eventsPath)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
}

func TestBlockerLoop_ResolveFirstAttempt(t *testing.T) {
	executor, orch, mockDocker, store, workflowID, stepID, tmpDir := setupBlockerTest(t)

	defID := "blocker-test-agent"
	stateDir := filepath.Join(tmpDir, "agent-state", defID)

	step := WorkflowStep{
		StepID:   stepID,
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{defID},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}
	workflow := store.workflows[workflowID]

	// Phase 1: First spawn returns blockers, then completes
	go func() {
		time.Sleep(500 * time.Millisecond)

		writeBlockerResult(t, stateDir, []Blocker{
			{BlockerType: "missing_input", Message: "Please provide credit card number", Field: "cc_number"},
		})

		// Complete the container
		mockDocker.mu.Lock()
		for _, s := range mockDocker.containers {
			if s.Running {
				s.Running = false
				s.ExitCode = 0
				s.FinishedAt = "2025-01-01T00:00:00Z"
				break
			}
		}
		mockDocker.mu.Unlock()

		// Wait for blocker to be registered, then deliver response
		time.Sleep(1 * time.Second)
		DeliverBlockerResponse(workflowID, stepID, BlockerResponse{
			Input:      "4242424242424242",
			Note:       "test card",
			UserID:     "@test:example.com",
			ProvidedAt: time.Now().Unix(),
		})

		// Phase 2: Second spawn succeeds
		time.Sleep(500 * time.Millisecond)

		// Clear old result, write success
		writeSuccessResult(t, stateDir)

		// Complete the second container
		mockDocker.mu.Lock()
		for id, s := range mockDocker.containers {
			if s.Running {
				s.Running = false
				s.ExitCode = 0
				s.FinishedAt = "2025-01-01T00:00:00Z"
				t.Logf("Transitioned container %s to completed (2nd)", id)
				break
			}
		}
		mockDocker.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)

	require.NoError(t, err, "blocker loop should resolve on first retry")
	require.NotNil(t, result)
	assert.Equal(t, stepID, result.StepID)
	assert.Equal(t, defID, result.AgentID)
	assert.NoError(t, result.Err)
}

func TestBlockerLoop_MaxRetriesExceeded(t *testing.T) {
	executor, orch, mockDocker, store, workflowID, stepID, tmpDir := setupBlockerTest(t)

	defID := "blocker-test-agent"
	stateDir := filepath.Join(tmpDir, "agent-state", defID)

	step := WorkflowStep{
		StepID:   stepID,
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{defID},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}
	workflow := store.workflows[workflowID]

	// Reactive goroutine: watch for running containers, complete each with blockers
	go func() {
		for attempt := 0; attempt < MaxBlockerRetries; attempt++ {
			var found bool
			for !found {
				time.Sleep(50 * time.Millisecond)
				mockDocker.mu.Lock()
				for cid, s := range mockDocker.containers {
					if s.Running {
						t.Logf("goroutine: found running container %s for attempt %d", cid, attempt)
						found = true
						break
					}
				}
				mockDocker.mu.Unlock()
			}

			writeBlockerResult(t, stateDir, []Blocker{
				{BlockerType: "missing_input", Message: "Still need input"},
			})
			t.Logf("goroutine: wrote blocker result for attempt %d", attempt)
			if data, err := os.ReadFile(filepath.Join(stateDir, "result.json")); err != nil {
				t.Logf("goroutine: ERROR reading back result.json: %v", err)
			} else {
				t.Logf("goroutine: result.json exists, size=%d, content=%.100s", len(data), string(data))
			}

			mockDocker.mu.Lock()
			for cid, s := range mockDocker.containers {
				if s.Running {
					s.Running = false
					s.ExitCode = 0
					s.FinishedAt = "2025-01-01T00:00:00Z"
					t.Logf("goroutine: completed container %s for attempt %d", cid, attempt)
					break
				}
			}
			mockDocker.mu.Unlock()

			time.Sleep(1 * time.Second)
			ok := DeliverBlockerResponse(workflowID, stepID, BlockerResponse{
				Input:      fmt.Sprintf("response-%d", attempt),
				UserID:     "@test:example.com",
				ProvidedAt: time.Now().Unix(),
			})
			t.Logf("goroutine: delivered blocker response for attempt %d, ok=%v", attempt, ok)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)

	require.Error(t, err, "should fail after max retries")
	assert.Contains(t, err.Error(), "max blocker retries")
	assert.Nil(t, result)
}

func TestBlockerLoop_PII_NotLogged(t *testing.T) {
	executor, orch, mockDocker, store, workflowID, stepID, tmpDir := setupBlockerTest(t)

	defID := "blocker-test-agent"
	stateDir := filepath.Join(tmpDir, "agent-state", defID)

	step := WorkflowStep{
		StepID:   stepID,
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{defID},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}
	workflow := store.workflows[workflowID]

	piiValue := "SSSSUPER-SECRET-PII-DATA-1234567890"

	go func() {
		time.Sleep(500 * time.Millisecond)

		writeBlockerResult(t, stateDir, []Blocker{
			{BlockerType: "missing_input", Message: "Need PII"},
		})

		mockDocker.mu.Lock()
		for _, s := range mockDocker.containers {
			if s.Running {
				s.Running = false
				s.ExitCode = 0
				s.FinishedAt = "2025-01-01T00:00:00Z"
				break
			}
		}
		mockDocker.mu.Unlock()

		time.Sleep(1 * time.Second)
		DeliverBlockerResponse(workflowID, stepID, BlockerResponse{
			Input:      piiValue,
			UserID:     "@test:example.com",
			ProvidedAt: time.Now().Unix(),
		})

		time.Sleep(500 * time.Millisecond)
		writeSuccessResult(t, stateDir)

		mockDocker.mu.Lock()
		for _, s := range mockDocker.containers {
			if s.Running {
				s.Running = false
				s.ExitCode = 0
				s.FinishedAt = "2025-01-01T00:00:00Z"
				break
			}
		}
		mockDocker.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the config passed to second spawn contains the PII in memory
	// (we can't inspect what Spawn received without more instrumentation,
	// but we verify the PII value does NOT appear in any file on disk)
	fileContents, _ := os.ReadFile(filepath.Join(stateDir, "result.json"))
	assert.NotContains(t, string(fileContents), piiValue,
		"PII must never appear in result.json on disk")

	// Also verify the appendBlockerResponse helper works correctly
	var testConfig map[string]interface{}
	require.NoError(t, json.Unmarshal(step.Config, &testConfig))
	updatedConfig := appendBlockerResponse(step.Config, BlockerResponse{
		Input:      piiValue,
		UserID:     "@test:example.com",
		ProvidedAt: time.Now().Unix(),
	})
	var updated map[string]interface{}
	require.NoError(t, json.Unmarshal(updatedConfig, &updated))
	require.NotNil(t, updated["_blocker_response"])
	blockerResp := updated["_blocker_response"].(map[string]interface{})
	assert.Equal(t, piiValue, blockerResp["input"], "PII should be in memory config")
}

func TestBlockerLoop_EventLogExceeded(t *testing.T) {
	executor, orch, _, store, workflowID, stepID, tmpDir := setupBlockerTest(t)

	defID := "blocker-test-agent"
	stateDir := filepath.Join(tmpDir, "agent-state", defID)

	step := WorkflowStep{
		StepID:   stepID,
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{defID},
		Config:   json.RawMessage(`{"url":"https://example.com"}`),
	}
	workflow := store.workflows[workflowID]

	// The goroutine writes the oversized file AFTER Spawn creates the state dir.
	// Spawn happens synchronously in executeStepWithBlockerHandling before
	// waitForCompletion is called, so we delay just enough for Spawn to run.
	go func() {
		time.Sleep(200 * time.Millisecond)

		eventsPath := filepath.Join(stateDir, "_events.jsonl")
		bigLine := make([]byte, 1024)
		for i := range bigLine {
			bigLine[i] = 'Z'
		}
		f, err := os.Create(eventsPath)
		if err != nil {
			t.Logf("Failed to create events file: %v", err)
			return
		}
		for i := 0; i < 10*1024+1; i++ {
			f.Write(append(bigLine, '\n'))
		}
		f.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)

	require.Error(t, err, "should return error for event log exceeded")
	assert.ErrorIs(t, err, ErrEventLogExceeded, "error should wrap ErrEventLogExceeded")
	assert.Nil(t, result, "result should be nil on event log exceeded")
}

func TestDeliverBlockerResponse_NoWaiter(t *testing.T) {
	ok := DeliverBlockerResponse("nonexistent-wf", "nonexistent-step", BlockerResponse{
		Input: "test",
	})
	assert.False(t, ok, "should return false when no waiter exists")
}

func TestAppendBlockerResponse(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		result := appendBlockerResponse(nil, BlockerResponse{
			Input:      "test-input",
			UserID:     "@test:example.com",
			ProvidedAt: 1234567890,
		})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.NotNil(t, m["_blocker_response"])
	})

	t.Run("existing config", func(t *testing.T) {
		original := json.RawMessage(`{"url":"https://example.com","timeout":30}`)
		result := appendBlockerResponse(original, BlockerResponse{
			Input:      "secret-value",
			UserID:     "@test:example.com",
			ProvidedAt: 1234567890,
		})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, "https://example.com", m["url"])
		assert.Equal(t, float64(30), m["timeout"])
		assert.NotNil(t, m["_blocker_response"])
		resp := m["_blocker_response"].(map[string]interface{})
		assert.Equal(t, "secret-value", resp["input"])
		assert.Equal(t, "@test:example.com", resp["user_id"])
	})

	t.Run("invalid json config", func(t *testing.T) {
		original := json.RawMessage(`{invalid json}`)
		result := appendBlockerResponse(original, BlockerResponse{
			Input:      "test",
			UserID:     "@test:example.com",
			ProvidedAt: 1234567890,
		})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.NotNil(t, m["_blocker_response"])
	})
}
