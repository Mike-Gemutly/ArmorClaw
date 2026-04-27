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

func TestTailEvents_OverflowReturnsSoftCapError(t *testing.T) {
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
		"ReadNew must return ErrEventLogExceeded for >10MB file (soft cap signal)")
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
	cp := *state
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    "purge-mock-container",
			State: &cp,
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

func TestPurgeOrder_10MBSoftCapPath(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("purge-test-softcap-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	// Create _events.jsonl larger than 10MB to trigger soft cap.
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

	// Write result.json so completion can be parsed.
	resultJSON, err := json.Marshal(ContainerStepResult{
		Status:     "success",
		Output:     "completed after soft cap",
		DurationMS: 1000,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))

	// After 800ms, transition Docker state to completed (container finishes naturally).
	go func() {
		time.Sleep(800 * time.Millisecond)
		mockDocker.setState(&types.ContainerState{Running: false, ExitCode: 0})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)

	// Soft cap: no error, container finishes normally.
	require.NoError(t, waitErr, "soft cap should not return error, container finishes normally")
	assert.NotNil(t, completionResult, "CompletionResult should be returned on soft cap path")
	assert.Equal(t, 0, completionResult.ExitCode)

	// Kill() should NOT be called — that's the whole point of soft cap.
	assert.Empty(t, mockDocker.getKilled(), "Kill() should NOT be called on 10MB soft cap")

	// State directory must be cleaned up after completion.
	_, statErr := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(statErr), "state dir should be cleaned up after soft cap completion: statErr=%v", statErr)
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
	cp := *state
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    cid,
			State: &cp,
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

func TestBlockerLoop_Timeout(t *testing.T) {
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

	// Goroutine: spawn returns blockers, but nobody delivers a response — simulate timeout
	go func() {
		time.Sleep(500 * time.Millisecond)
		writeBlockerResult(t, stateDir, []Blocker{
			{BlockerType: "missing_input", Message: "Need password"},
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
		// Intentionally do NOT deliver blocker response — simulate timeout
	}()

	// Use short timeout instead of waiting full 10 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)

	require.Error(t, err, "should fail when blocker times out")
	assert.Contains(t, err.Error(), "blocker")
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

	// Verify appendBlockerResponse never serializes raw PII into the config map
	updatedConfig := appendBlockerResponse(step.Config, BlockerResponse{
		Input:      piiValue,
		UserID:     "@test:example.com",
		ProvidedAt: time.Now().Unix(),
	})
	assert.NotContains(t, string(updatedConfig), piiValue,
		"Raw PII must never appear in serialized config JSON")

	var updated map[string]interface{}
	require.NoError(t, json.Unmarshal(updatedConfig, &updated))
	require.NotNil(t, updated["_blocker_response"])
	blockerResp := updated["_blocker_response"].(map[string]interface{})
	assert.Equal(t, true, blockerResp["has_input"], "has_input should be true when Input is present")
	assert.Equal(t, "@test:example.com", blockerResp["user_id"])
}

func TestAppendBlockerResponse_NoRawPII(t *testing.T) {
	config := json.RawMessage(`{"url":"https://example.com"}`)
	resp := BlockerResponse{
		Input:      "SSSSUPER-SECRET-PII-DATA",
		UserID:     "@test:example.com",
		ProvidedAt: time.Now().Unix(),
	}
	result := appendBlockerResponse(config, resp)

	assert.NotContains(t, string(result), "SSSSUPER-SECRET-PII-DATA",
		"Raw PII must never be serialized into config JSON")

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &m))
	br := m["_blocker_response"].(map[string]interface{})
	assert.Equal(t, true, br["has_input"])
	assert.Equal(t, "@test:example.com", br["user_id"])

	_, hasInput := br["input"]
	assert.False(t, hasInput, "input field must not exist in serialized config")
}

func TestBlockerLoop_EventLogExceeded(t *testing.T) {
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

		resultJSON, _ := json.Marshal(ContainerStepResult{
			Status:     "success",
			Output:     "completed after soft cap",
			DurationMS: 1000,
		})
		os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644)

		time.Sleep(500 * time.Millisecond)
		mockDocker.completeContainer("blocker-container-1")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := executor.executeStepWithBlockerHandling(ctx, workflow, step, defID, orch)

	require.NoError(t, err, "soft cap should not return error")
	require.NotNil(t, result)
	assert.Equal(t, stepID, result.StepID)
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
		assert.NotContains(t, string(result), "test-input")
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.NotNil(t, m["_blocker_response"])
		resp := m["_blocker_response"].(map[string]interface{})
		assert.Equal(t, true, resp["has_input"])
	})

	t.Run("existing config", func(t *testing.T) {
		original := json.RawMessage(`{"url":"https://example.com","timeout":30}`)
		result := appendBlockerResponse(original, BlockerResponse{
			Input:      "secret-value",
			UserID:     "@test:example.com",
			ProvidedAt: 1234567890,
		})
		assert.NotContains(t, string(result), "secret-value")
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, "https://example.com", m["url"])
		assert.Equal(t, float64(30), m["timeout"])
		assert.NotNil(t, m["_blocker_response"])
		resp := m["_blocker_response"].(map[string]interface{})
		assert.Equal(t, true, resp["has_input"])
		assert.Equal(t, "@test:example.com", resp["user_id"])
	})

	t.Run("invalid json config", func(t *testing.T) {
		original := json.RawMessage(`{invalid json}`)
		result := appendBlockerResponse(original, BlockerResponse{
			Input:      "PII-SENSITIVE-INPUT-XYZ",
			UserID:     "@test:example.com",
			ProvidedAt: 1234567890,
		})
		assert.NotContains(t, string(result), "PII-SENSITIVE-INPUT-XYZ")
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.NotNil(t, m["_blocker_response"])
		resp := m["_blocker_response"].(map[string]interface{})
		assert.Equal(t, true, resp["has_input"])
	})
}

//=============================================================================
// injectLearnedSkills Tests
//=============================================================================

type mockSkillFinder struct {
	skills []LearnedSkillInfo
	err    error
}

func (m *mockSkillFinder) FindForTask(taskDesc string, limit int) ([]LearnedSkillInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.skills, nil
}

func TestInjectLearnedSkills_WithMatch(t *testing.T) {
	finder := &mockSkillFinder{
		skills: []LearnedSkillInfo{
			{Name: "web-search", Confidence: 0.9, PatternType: "web_browsing", SourceTaskID: "task-123"},
			{Name: "form-fill", Confidence: 0.7, PatternType: "form_filling", SourceTaskID: "task-456"},
		},
	}

	executor := NewStepExecutor(StepExecutorConfig{SkillFinder: finder})

	config := json.RawMessage(`{"timeout": 30}`)
	result := executor.injectLearnedSkills(context.Background(), config, "search the web for restaurants")

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &m))
	assert.Equal(t, float64(30), m["timeout"])

	skills := m["relevant_skills"].([]interface{})
	require.Len(t, skills, 2)

	first := skills[0].(map[string]interface{})
	assert.Equal(t, "web-search", first["name"])
	assert.Equal(t, float64(0.9), first["confidence"])
	assert.Equal(t, "web_browsing", first["pattern"])
	assert.Equal(t, "task-123", first["source"])

	second := skills[1].(map[string]interface{})
	assert.Equal(t, "form-fill", second["name"])
}

func TestInjectLearnedSkills_NilStore(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})

	original := json.RawMessage(`{"timeout": 30}`)
	result := executor.injectLearnedSkills(context.Background(), original, "search the web for restaurants")

	assert.Equal(t, original, result, "nil SkillFinder should return config unchanged")
}

func TestInjectLearnedSkills_NoMatch(t *testing.T) {
	finder := &mockSkillFinder{
		skills: []LearnedSkillInfo{},
	}

	executor := NewStepExecutor(StepExecutorConfig{SkillFinder: finder})

	original := json.RawMessage(`{"timeout": 30}`)
	result := executor.injectLearnedSkills(context.Background(), original, "search the web for restaurants")

	assert.Equal(t, original, result, "no matching skills should return config unchanged")
}

//=============================================================================
// Post-Completion Skill Extraction + RecordOutcome Tests
//=============================================================================

func TestPostCompletion_SkillExtractedOnSuccess(t *testing.T) {
	_, mockDocker, _, instanceID, executor := setupPurgeTest(t,
		&types.ContainerState{Running: true, ExitCode: 0},
	)

	var extractionCalls []struct {
		taskDesc string
		taskID   string
		tmplID   string
	}
	var extractionMu sync.Mutex

	executor.onSkillExtraction = func(result *ExtendedStepResult, taskDesc, taskID, templateID string) {
		extractionMu.Lock()
		extractionCalls = append(extractionCalls, struct {
			taskDesc string
			taskID   string
			tmplID   string
		}{taskDesc, taskID, templateID})
		extractionMu.Unlock()
	}

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("skill-extract-ok-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	resultJSON, err := json.Marshal(ContainerStepResult{
		Status:     "success",
		Output:     "task completed",
		DurationMS: 500,
	})
	require.NoError(t, err)
	writeStateDirFiles(t, stateDir, resultJSON)

	go func() {
		time.Sleep(800 * time.Millisecond)
		mockDocker.setState(&types.ContainerState{Running: false, ExitCode: 0})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	completionResult, waitErr := executor.waitForCompletion(ctx, instanceID, stateDir)
	require.NoError(t, waitErr)
	require.NotNil(t, completionResult)

	workflow := &Workflow{
		ID:         "wf-skill-test",
		TemplateID: "tmpl-123",
		Name:       "Skill Test",
		Status:     StatusRunning,
		CreatedBy:  "@test:example.com",
		RoomID:     "!test:example.com",
		StartedAt:  time.Now(),
	}
	step := WorkflowStep{
		StepID:   "step-1",
		Order:    0,
		Type:     StepAction,
		Name:     "Test Step",
		AgentIDs: []string{"purge-test-agent"},
		Config:   json.RawMessage(`{"timeout": 30}`),
	}

	stepResult := &StepResult{
		StepID:          step.StepID,
		AgentID:         "purge-test-agent",
		InstanceID:      instanceID,
		ContainerResult: completionResult.ContainerResult,
	}
	stepSuccess := stepResult.Err == nil

	executor.recordSkillOutcomes(step.Config, stepSuccess)
	if stepSuccess && executor.onSkillExtraction != nil && completionResult.ExtendedResult != nil {
		executor.onSkillExtraction(completionResult.ExtendedResult,
			fmt.Sprintf("Workflow %s - Step: %s", workflow.ID, step.Name),
			workflow.ID, workflow.TemplateID)
	}

	extractionMu.Lock()
	require.Len(t, extractionCalls, 1, "OnSkillExtraction should be called once on success")
	assert.Equal(t, "wf-skill-test", extractionCalls[0].taskID)
	assert.Equal(t, "tmpl-123", extractionCalls[0].tmplID)
	assert.Contains(t, extractionCalls[0].taskDesc, "wf-skill-test")
	extractionMu.Unlock()
}

func TestPostCompletion_NoExtractionOnFailure(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})

	extractionCalled := false
	executor.onSkillExtraction = func(result *ExtendedStepResult, taskDesc, taskID, templateID string) {
		extractionCalled = true
	}

	completionResult := &CompletionResult{
		ExitCode: 1,
		ExtendedResult: &ExtendedStepResult{
			ContainerStepResult: &ContainerStepResult{Status: "failed"},
		},
	}

	workflow := &Workflow{
		ID:         "wf-fail-test",
		TemplateID: "tmpl-456",
		Name:       "Fail Test",
		Status:     StatusRunning,
		CreatedBy:  "@test:example.com",
	}
	step := WorkflowStep{
		StepID:   "step-fail",
		Order:    0,
		Type:     StepAction,
		Name:     "Fail Step",
		AgentIDs: []string{"agent-1"},
		Config:   json.RawMessage(`{"timeout": 30}`),
	}

	stepSuccess := false

	executor.recordSkillOutcomes(step.Config, stepSuccess)
	if stepSuccess && executor.onSkillExtraction != nil && completionResult != nil && completionResult.ExtendedResult != nil {
		executor.onSkillExtraction(completionResult.ExtendedResult,
			fmt.Sprintf("Workflow %s - Step: %s", workflow.ID, step.Name),
			workflow.ID, workflow.TemplateID)
	}

	assert.False(t, extractionCalled, "OnSkillExtraction should NOT be called on failure")
}

func TestPostCompletion_RecordOutcomeOnSuccess(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})

	var recordedOutcomes []struct {
		skillID string
		success bool
	}
	var outcomeMu sync.Mutex

	executor.onSkillOutcome = func(skillID string, success bool) error {
		outcomeMu.Lock()
		recordedOutcomes = append(recordedOutcomes, struct {
			skillID string
			success bool
		}{skillID, success})
		outcomeMu.Unlock()
		return nil
	}

	config := json.RawMessage(`{
		"timeout": 30,
		"relevant_skills": [
			{"name": "web-search", "source": "task-100", "confidence": 0.9},
			{"name": "form-fill", "source": "task-200", "confidence": 0.7}
		]
	}`)

	executor.recordSkillOutcomes(config, true)

	outcomeMu.Lock()
	require.Len(t, recordedOutcomes, 2, "should record outcome for each suggested skill")
	assert.Equal(t, "task-100", recordedOutcomes[0].skillID)
	assert.True(t, recordedOutcomes[0].success)
	assert.Equal(t, "task-200", recordedOutcomes[1].skillID)
	assert.True(t, recordedOutcomes[1].success)
	outcomeMu.Unlock()
}

func TestPostCompletion_RecordOutcomeOnFailure(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})

	var recordedOutcomes []struct {
		skillID string
		success bool
	}
	var outcomeMu sync.Mutex

	executor.onSkillOutcome = func(skillID string, success bool) error {
		outcomeMu.Lock()
		recordedOutcomes = append(recordedOutcomes, struct {
			skillID string
			success bool
		}{skillID, success})
		outcomeMu.Unlock()
		return nil
	}

	config := json.RawMessage(`{
		"timeout": 30,
		"relevant_skills": [
			{"name": "web-search", "source": "task-100", "confidence": 0.9}
		]
	}`)

	executor.recordSkillOutcomes(config, false)

	outcomeMu.Lock()
	require.Len(t, recordedOutcomes, 1, "should record outcome even on failure")
	assert.Equal(t, "task-100", recordedOutcomes[0].skillID)
	assert.False(t, recordedOutcomes[0].success)
	outcomeMu.Unlock()
}

//=============================================================================
// Sequential Step Data Propagation Tests
//=============================================================================

func TestResolveTemplateString_BasicRef(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"order_id": "ORD-123", "total": 99.5},
	}

	result := resolveTemplateString("order={{steps.step_1.data.order_id}}", accumulated)
	assert.Equal(t, "order=ORD-123", result)
}

func TestResolveTemplateString_MultipleRefs(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"order_id": "ORD-456"},
		"step_2": {"tracking": "TRK-789"},
	}

	result := resolveTemplateString(
		"order={{steps.step_1.data.order_id}} track={{steps.step_2.data.tracking}}",
		accumulated,
	)
	assert.Equal(t, "order=ORD-456 track=TRK-789", result)
}

func TestResolveTemplateString_MissingStep(t *testing.T) {
	accumulated := map[string]map[string]any{}

	result := resolveTemplateString("{{steps.step_1.data.order_id}}", accumulated)
	assert.Equal(t, "{{steps.step_1.data.order_id}}", result)
}

func TestResolveTemplateString_MissingKey(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"other_key": "value"},
	}

	result := resolveTemplateString("{{steps.step_1.data.missing}}", accumulated)
	assert.Equal(t, "{{steps.step_1.data.missing}}", result)
}

func TestResolveStepInput_NilInput(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"order_id": "ORD-123"},
	}

	result, err := resolveStepInput(nil, accumulated)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestResolveStepInput_EmptyInput(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"order_id": "ORD-123"},
	}

	result, err := resolveStepInput(map[string]any{}, accumulated)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestResolveStepInput_MixedValues(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"order_id": "ORD-789"},
	}

	input := map[string]any{
		"order_ref": "{{steps.step_1.data.order_id}}",
		"static":    "hardcoded",
		"number":    42,
	}

	result, err := resolveStepInput(input, accumulated)
	require.NoError(t, err)
	assert.Equal(t, "ORD-789", result["order_ref"])
	assert.Equal(t, "hardcoded", result["static"])
	assert.Equal(t, 42, result["number"])
}

func TestInjectPrevStepData_NilConfig(t *testing.T) {
	input := map[string]any{"order_id": "ORD-123"}

	result := injectPrevStepData(nil, input)
	require.NotNil(t, result)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &parsed))
	assert.Equal(t, "ORD-123", parsed["_prev_step_data"].(map[string]interface{})["order_id"])
}

func TestInjectPrevStepData_ExistingConfig(t *testing.T) {
	config := json.RawMessage(`{"timeout": 30}`)
	input := map[string]any{"order_id": "ORD-456"}

	result := injectPrevStepData(config, input)
	require.NotNil(t, result)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &parsed))
	assert.Equal(t, float64(30), parsed["timeout"])
	assert.Equal(t, "ORD-456", parsed["_prev_step_data"].(map[string]interface{})["order_id"])
}

func TestInjectPrevStepData_EmptyInput(t *testing.T) {
	config := json.RawMessage(`{"timeout": 30}`)

	result := injectPrevStepData(config, nil)
	assert.Equal(t, config, result)
}

func TestSequentialDataPropagation_Integration(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {
			"order_id": "ORD-SEQUENTIAL",
			"amount":   199.99,
		},
	}

	step := WorkflowStep{
		StepID:   "step_2",
		Order:    1,
		Type:     StepAction,
		Name:     "Confirm Order",
		AgentIDs: []string{"agent-1"},
		Input: map[string]any{
			"order_ref": "{{steps.step_1.data.order_id}}",
			"amount":    "{{steps.step_1.data.amount}}",
		},
		Config: json.RawMessage(`{"action":"confirm"}`),
	}

	resolved, err := resolveStepInput(step.Input, accumulated)
	require.NoError(t, err)

	enrichedConfig := injectPrevStepData(step.Config, resolved)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(enrichedConfig, &parsed))

	assert.Equal(t, "confirm", parsed["action"])

	prevData, ok := parsed["_prev_step_data"].(map[string]interface{})
	require.True(t, ok, "config should contain _prev_step_data")
	assert.Equal(t, "ORD-SEQUENTIAL", prevData["order_ref"])
	assert.Equal(t, "199.99", prevData["amount"])
}

func TestSequentialDataPropagation_NoDataProduced(t *testing.T) {
	accumulated := map[string]map[string]any{}

	step := WorkflowStep{
		StepID:   "step_2",
		Order:    1,
		Type:     StepAction,
		Name:     "Step Two",
		AgentIDs: []string{"agent-1"},
		Input: map[string]any{
			"ref": "{{steps.step_1.data.missing}}",
		},
		Config: json.RawMessage(`{"action":"use"}`),
	}

	resolved, err := resolveStepInput(step.Input, accumulated)
	require.NoError(t, err)

	enrichedConfig := injectPrevStepData(step.Config, resolved)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(enrichedConfig, &parsed))

	assert.Equal(t, "use", parsed["action"])

	prevData := parsed["_prev_step_data"].(map[string]interface{})
	assert.Equal(t, "{{steps.step_1.data.missing}}", prevData["ref"])
}

func TestSequentialDataPropagation_NilInputSkipped(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_1": {"key": "val"},
	}

	step := WorkflowStep{
		StepID:   "step_2",
		Order:    1,
		Type:     StepAction,
		Name:     "Step Two",
		AgentIDs: []string{"agent-1"},
		Config:   json.RawMessage(`{"action":"run"}`),
	}

	resolved, err := resolveStepInput(step.Input, accumulated)
	require.NoError(t, err)
	assert.Nil(t, resolved)

	enrichedConfig := injectPrevStepData(step.Config, resolved)
	assert.Equal(t, step.Config, enrichedConfig, "config should be unchanged when Input is nil")
}

func TestDataPass_AccumulatorCollectsResults(t *testing.T) {
	accumulated := make(map[string]map[string]any)

	result1 := &ContainerStepResult{
		Status: "success",
		Data:   map[string]any{"order_id": "A1", "total": 50.0},
	}
	if result1.Data != nil {
		accumulated["step_1"] = result1.Data
	}

	result2 := &ContainerStepResult{
		Status: "success",
		Data:   map[string]any{"tracking": "TRK-XYZ"},
	}
	if result2.Data != nil {
		accumulated["step_2"] = result2.Data
	}

	assert.Equal(t, "A1", accumulated["step_1"]["order_id"])
	assert.Equal(t, 50.0, accumulated["step_1"]["total"])
	assert.Equal(t, "TRK-XYZ", accumulated["step_2"]["tracking"])

	input := map[string]any{
		"order":   "{{steps.step_1.data.order_id}}",
		"track":   "{{steps.step_2.data.tracking}}",
		"verbose": "yes",
	}

	resolved, err := resolveStepInput(input, accumulated)
	require.NoError(t, err)
	assert.Equal(t, "A1", resolved["order"])
	assert.Equal(t, "TRK-XYZ", resolved["track"])
	assert.Equal(t, "yes", resolved["verbose"])
}

func TestDataPass_EmptyDataNotStored(t *testing.T) {
	accumulated := make(map[string]map[string]any)

	result := &ContainerStepResult{
		Status: "success",
		Output: "done",
	}
	if result.Data != nil && len(result.Data) > 0 {
		accumulated["step_1"] = result.Data
	}

	_, exists := accumulated["step_1"]
	assert.False(t, exists, "steps with no Data should not be stored in accumulator")
}
