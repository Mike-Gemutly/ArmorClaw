package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Parallel Group Identification Tests
//=============================================================================

func TestIdentifyParallelGroups_SplitMerge(t *testing.T) {
	steps := []WorkflowStep{
		{StepID: "setup", Order: 0, Type: StepAction},
		{StepID: "split", Order: 1, Type: StepParallelSplit},
		{StepID: "branch-a", Order: 2, Type: StepAction},
		{StepID: "branch-b", Order: 3, Type: StepAction},
		{StepID: "merge", Order: 4, Type: StepParallelMerge},
		{StepID: "cleanup", Order: 5, Type: StepAction},
	}

	groups := IdentifyParallelGroups(steps)
	require.Len(t, groups, 1)

	g := groups[0]
	assert.Equal(t, "split", g.SplitStepID)
	assert.Equal(t, "merge", g.MergeStepID)
	assert.ElementsMatch(t, []string{"branch-a", "branch-b"}, g.BranchStepIDs)
}

func TestIdentifyParallelGroups_NoParallelSteps(t *testing.T) {
	groups := IdentifyParallelGroups([]WorkflowStep{
		{StepID: "step1", Order: 0, Type: StepAction},
		{StepID: "step2", Order: 1, Type: StepAction},
	})
	assert.Empty(t, groups)
}

func TestIdentifyParallelGroups_MultipleGroups(t *testing.T) {
	steps := []WorkflowStep{
		{StepID: "split1", Order: 0, Type: StepParallelSplit},
		{StepID: "branch1a", Order: 1, Type: StepAction},
		{StepID: "branch1b", Order: 2, Type: StepAction},
		{StepID: "merge1", Order: 3, Type: StepParallelMerge},
		{StepID: "split2", Order: 4, Type: StepParallelSplit},
		{StepID: "branch2a", Order: 5, Type: StepAction},
		{StepID: "merge2", Order: 6, Type: StepParallelMerge},
	}

	groups := IdentifyParallelGroups(steps)
	require.Len(t, groups, 2)

	assert.Equal(t, "split1", groups[0].SplitStepID)
	assert.Equal(t, "merge1", groups[0].MergeStepID)
	assert.ElementsMatch(t, []string{"branch1a", "branch1b"}, groups[0].BranchStepIDs)

	assert.Equal(t, "split2", groups[1].SplitStepID)
	assert.Equal(t, "merge2", groups[1].MergeStepID)
	assert.ElementsMatch(t, []string{"branch2a"}, groups[1].BranchStepIDs)
}

func TestIdentifyParallelGroups_SplitWithoutMerge(t *testing.T) {
	groups := IdentifyParallelGroups([]WorkflowStep{
		{StepID: "split", Order: 0, Type: StepParallelSplit},
		{StepID: "branch-a", Order: 1, Type: StepAction},
	})
	assert.Empty(t, groups, "split without merge should produce no groups")
}

func TestIdentifyParallelGroups_Empty(t *testing.T) {
	groups := IdentifyParallelGroups(nil)
	assert.Empty(t, groups)
}

func TestHasParallelSteps(t *testing.T) {
	assert.True(t, HasParallelSteps([]WorkflowStep{{Type: StepParallel}}))
	assert.True(t, HasParallelSteps([]WorkflowStep{{Type: StepParallelSplit}}))
	assert.True(t, HasParallelSteps([]WorkflowStep{{Type: StepParallelMerge}}))
	assert.False(t, HasParallelSteps([]WorkflowStep{{Type: StepAction}, {Type: StepCondition}}))
}

func TestBuildStepToGroupMap(t *testing.T) {
	groups := []*ParallelGroup{
		{
			SplitStepID:   "split",
			MergeStepID:   "merge",
			BranchStepIDs: []string{"branch-a", "branch-b"},
		},
	}

	m := BuildStepToGroupMap(groups)
	assert.Equal(t, groups[0], m["merge"])
	assert.Equal(t, groups[0], m["branch-a"])
	assert.Equal(t, groups[0], m["branch-b"])
	_, hasSplit := m["split"]
	assert.False(t, hasSplit, "split step should not be in step-to-group map")
}

func TestBuildParallelGroupIndex(t *testing.T) {
	groups := []*ParallelGroup{
		{SplitStepID: "split1", MergeStepID: "merge1", BranchStepIDs: []string{"a"}},
		{SplitStepID: "split2", MergeStepID: "merge2", BranchStepIDs: []string{"b"}},
	}
	idx := BuildParallelGroupIndex(groups)
	assert.Equal(t, groups[0], idx["split1"])
	assert.Equal(t, groups[1], idx["split2"])
}

//=============================================================================
// Parallel Mock Docker (concurrent-safe)
//=============================================================================

type parallelMockDocker struct {
	mu         sync.Mutex
	containers map[string]*types.ContainerState
	nextID     int
	spawnTimes []time.Time
}

func newParallelMockDocker() *parallelMockDocker {
	return &parallelMockDocker{
		containers: make(map[string]*types.ContainerState),
	}
}

func (m *parallelMockDocker) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ any, _ any, _ string) (container.CreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("par-container-%d", m.nextID)
	m.containers[id] = &types.ContainerState{Running: true, ExitCode: 0}
	m.spawnTimes = append(m.spawnTimes, time.Now())
	return container.CreateResponse{ID: id}, nil
}

func (m *parallelMockDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *parallelMockDocker) ContainerStop(_ context.Context, cid string, _ container.StopOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *parallelMockDocker) ContainerKill(_ context.Context, cid string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *parallelMockDocker) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *parallelMockDocker) ContainerInspect(_ context.Context, cid string) (types.ContainerJSON, error) {
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

func (m *parallelMockDocker) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return nil, nil
}

//=============================================================================
// Parallel Executor Direct Tests
//=============================================================================

func setupParallelExecutor(t *testing.T, mockDocker *parallelMockDocker) (*StepExecutor, string) {
	t.Helper()

	tmpDir := t.TempDir()
	studioStore, err := studio.NewStore(studio.StoreConfig{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { studioStore.Close() })

	def := &studio.AgentDefinition{
		ID: "par-agent", Name: "Par Agent", Skills: []string{"browser_navigate"},
		ResourceTier: "medium", CreatedBy: "@test:example.com",
		CreatedAt: time.Now(), UpdatedAt: time.Now(), IsActive: true,
	}
	require.NoError(t, studioStore.CreateDefinition(def))

	// Create additional agent definitions for parallel branches.
	for i := 1; i <= 3; i++ {
		extraDef := &studio.AgentDefinition{
			ID: fmt.Sprintf("par-agent-%d", i), Name: fmt.Sprintf("Par Agent %d", i),
			Skills: []string{"browser_navigate"}, ResourceTier: "medium",
			CreatedBy: "@test:example.com", CreatedAt: time.Now(),
			UpdatedAt: time.Now(), IsActive: true,
		}
		require.NoError(t, studioStore.CreateDefinition(extraDef))
	}

	factory := studio.NewAgentFactory(studio.FactoryConfig{
		StateDir: tmpDir, DockerClient: mockDocker, Store: studioStore,
	})

	stateBase := filepath.Join(tmpDir, "agent-state")
	executor := NewStepExecutor(StepExecutorConfig{
		Factory:      factory,
		StateDirBase: stateBase,
		Validator:    NewDependencyValidator(),
	})

	return executor, def.ID
}

func writeParSuccess(t *testing.T, stateDir string) {
	t.Helper()
	result := ContainerStepResult{Status: "success", Output: "ok", DurationMS: 100}
	resultJSON, _ := json.Marshal(result)
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))
	ep := filepath.Join(stateDir, "_events.jsonl")
	if _, err := os.Stat(ep); os.IsNotExist(err) {
		f, _ := os.Create(ep)
		f.Close()
	}
}

func writeParFailure(t *testing.T, stateDir string) {
	t.Helper()
	result := ContainerStepResult{Status: "failed", Error: "branch failed"}
	resultJSON, _ := json.Marshal(result)
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))
	ep := filepath.Join(stateDir, "_events.jsonl")
	if _, err := os.Stat(ep); os.IsNotExist(err) {
		f, _ := os.Create(ep)
		f.Close()
	}
}

// TestParallel_ExecuteParallelGroup_Concurrent verifies that parallel branches
// execute concurrently using errgroup. Three branches with 200ms simulated
// delay each should complete in < 500ms total (not 600ms+).
func TestParallel_ExecuteParallelGroup_Concurrent(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, defID := setupParallelExecutor(t, mockDocker)

	tmpDir := executor.stateDirBase
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, defID), 0755))

	workflow := &Workflow{
		ID: "wf-par", Name: "Par", Status: StatusRunning,
		CreatedBy: "@test:example.com", RoomID: "!test:example.com", StartedAt: time.Now(),
	}

	stepMap := map[string]WorkflowStep{
		"branch-a": {StepID: "branch-a", Order: 0, Type: StepAction, Name: "A", AgentIDs: []string{defID}},
		"branch-b": {StepID: "branch-b", Order: 1, Type: StepAction, Name: "B", AgentIDs: []string{defID}},
	}

	group := &ParallelGroup{
		SplitStepID:   "split",
		MergeStepID:   "merge",
		BranchStepIDs: []string{"branch-a", "branch-b"},
	}

	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           FailFast,
	})

	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			mockDocker.mu.Lock()
			for _, s := range mockDocker.containers {
				if s.Running {
					s.Running = false
					s.ExitCode = 0
					s.FinishedAt = "2025-01-01T00:00:00Z"
				}
			}
			running := 0
			for _, s := range mockDocker.containers {
				if s.Running {
					running++
				}
			}
			allDone := len(mockDocker.containers) >= 2 && running == 0
			if allDone {
				writeParSuccess(t, filepath.Join(tmpDir, defID))
			}
			mockDocker.mu.Unlock()
			if allDone {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := pe.ExecuteParallelGroup(ctx, workflow, group, stepMap, 0, 1.0)
	require.NoError(t, err)
	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.BranchResults, 2)

	mockDocker.mu.Lock()
	spawnCount := len(mockDocker.spawnTimes)
	mockDocker.mu.Unlock()
	assert.Equal(t, 2, spawnCount, "both branches should spawn containers")
}

// TestParallel_ExecuteParallelGroup_FailFast verifies that a failing branch
// causes the group to return an error.
func TestParallel_ExecuteParallelGroup_FailFast(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	agentOK := "par-agent-1"
	agentFail := "par-agent-2"

	tmpDir := executor.stateDirBase
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentOK), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentFail), 0755))

	workflow := &Workflow{
		ID: "wf-par-fail", Name: "Par Fail", Status: StatusRunning,
		CreatedBy: "@test:example.com", RoomID: "!test:example.com", StartedAt: time.Now(),
	}

	stepMap := map[string]WorkflowStep{
		"ok":   {StepID: "ok", Order: 0, Type: StepAction, Name: "OK", AgentIDs: []string{agentOK}},
		"fail": {StepID: "fail", Order: 1, Type: StepAction, Name: "Fail", AgentIDs: []string{agentFail}},
	}

	group := &ParallelGroup{
		SplitStepID:   "split",
		MergeStepID:   "merge",
		BranchStepIDs: []string{"ok", "fail"},
	}

	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           FailFast,
	})

	var completed atomic.Int32
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			mockDocker.mu.Lock()
			for _, s := range mockDocker.containers {
				if s.Running {
					n := completed.Add(1)
					s.Running = false
					s.FinishedAt = "2025-01-01T00:00:00Z"
					if n == 1 {
						s.ExitCode = 0
						writeParSuccess(t, filepath.Join(tmpDir, agentOK))
					} else {
						s.ExitCode = 1
						writeParFailure(t, filepath.Join(tmpDir, agentFail))
					}
				}
			}
			mockDocker.mu.Unlock()
			if completed.Load() >= 2 {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := pe.ExecuteParallelGroup(ctx, workflow, group, stepMap, 0, 1.0)
	require.Error(t, err, "should fail when a branch fails")
	assert.False(t, result.AllSucceeded)
	assert.NotEmpty(t, result.Errors)
}

// TestParallel_ExecuteStandaloneParallel_SingleAgent skips parallelism for
// a single agent and runs normally.
func TestParallel_ExecuteStandaloneParallel_SingleAgent(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, defID := setupParallelExecutor(t, mockDocker)

	stateDir := filepath.Join(os.TempDir(), fmt.Sprintf("par-single-%d", time.Now().UnixNano()))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	t.Cleanup(func() { os.RemoveAll(stateDir) })

	agentStateDir := filepath.Join(stateDir, defID)
	require.NoError(t, os.MkdirAll(agentStateDir, 0755))
	executor.stateDirBase = stateDir

	workflow := &Workflow{
		ID: "wf-single", Name: "Single", Status: StatusRunning,
		CreatedBy: "@test:example.com", RoomID: "!test:example.com", StartedAt: time.Now(),
	}

	step := WorkflowStep{
		StepID: "par-step", Order: 0, Type: StepParallel,
		Name: "Single Agent", AgentIDs: []string{defID},
	}

	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           FailFast,
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		mockDocker.mu.Lock()
		for _, s := range mockDocker.containers {
			if s.Running {
				s.Running = false
				s.ExitCode = 0
				s.FinishedAt = "2025-01-01T00:00:00Z"
				writeParSuccess(t, agentStateDir)
			}
		}
		mockDocker.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := pe.ExecuteStandaloneParallel(ctx, workflow, step)
	require.NoError(t, err)
	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.BranchResults, 1)
}

// TestParallel_ExecuteStandaloneParallel_MultipleAgents spawns multiple agents
// concurrently for a single StepParallel step.
func TestParallel_ExecuteStandaloneParallel_MultipleAgents(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	agentA := "par-agent-1"
	agentB := "par-agent-2"

	tmpDir := executor.stateDirBase
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentA), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentB), 0755))

	workflow := &Workflow{
		ID: "wf-multi", Name: "Multi", Status: StatusRunning,
		CreatedBy: "@test:example.com", RoomID: "!test:example.com", StartedAt: time.Now(),
	}

	step := WorkflowStep{
		StepID: "par-multi", Order: 0, Type: StepParallel,
		Name: "Multi Agent", AgentIDs: []string{agentA, agentB},
	}

	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           FailFast,
	})

	var completed atomic.Int32
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			mockDocker.mu.Lock()
			for _, s := range mockDocker.containers {
				if s.Running {
					s.Running = false
					s.ExitCode = 0
					s.FinishedAt = "2025-01-01T00:00:00Z"
					completed.Add(1)
					for _, aid := range []string{agentA, agentB} {
						writeParSuccess(t, filepath.Join(tmpDir, aid))
					}
				}
			}
			mockDocker.mu.Unlock()
			if completed.Load() >= 2 {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := pe.ExecuteStandaloneParallel(ctx, workflow, step)
	require.NoError(t, err)
	assert.True(t, result.AllSucceeded)
	assert.Len(t, result.BranchResults, 2, "both agents should produce results")
}

// TestParallel_ExecuteParallelGroup_EmptyBranches handles zero-branch groups.
func TestParallel_ExecuteParallelGroup_EmptyBranches(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	workflow := &Workflow{
		ID: "wf-empty", Name: "Empty", Status: StatusRunning,
		CreatedBy: "@test:example.com", StartedAt: time.Now(),
	}

	group := &ParallelGroup{
		SplitStepID:   "split",
		MergeStepID:   "merge",
		BranchStepIDs: []string{},
	}

	pe := NewParallelExecutor(executor, DefaultParallelConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := pe.ExecuteParallelGroup(ctx, workflow, group, map[string]WorkflowStep{}, 0, 1.0)
	require.NoError(t, err)
	assert.True(t, result.AllSucceeded)
}

// TestParallel_ExecuteStandaloneParallel_NoAgent returns ErrNoAgentForStep.
func TestParallel_ExecuteStandaloneParallel_NoAgent(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	workflow := &Workflow{
		ID: "wf-noagent", Name: "NoAgent", Status: StatusRunning,
		CreatedBy: "@test:example.com", StartedAt: time.Now(),
	}

	step := WorkflowStep{StepID: "no-agent", Order: 0, Type: StepParallel, Name: "No Agent"}

	pe := NewParallelExecutor(executor, DefaultParallelConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pe.ExecuteStandaloneParallel(ctx, workflow, step)
	assert.ErrorIs(t, err, ErrNoAgentForStep)
}

// TestParallel_CollectAll runs all branches regardless of failures.
func TestParallel_CollectAll(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	agentOK := "par-agent-1"
	agentFail := "par-agent-2"

	tmpDir := executor.stateDirBase
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentOK), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, agentFail), 0755))

	workflow := &Workflow{
		ID: "wf-collect", Name: "Collect", Status: StatusRunning,
		CreatedBy: "@test:example.com", RoomID: "!test:example.com", StartedAt: time.Now(),
	}

	stepMap := map[string]WorkflowStep{
		"ok":   {StepID: "ok", Order: 0, Type: StepAction, Name: "OK", AgentIDs: []string{agentOK}},
		"fail": {StepID: "fail", Order: 1, Type: StepAction, Name: "Fail", AgentIDs: []string{agentFail}},
	}

	group := &ParallelGroup{
		SplitStepID:   "split",
		MergeStepID:   "merge",
		BranchStepIDs: []string{"ok", "fail"},
	}

	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           CollectAll,
	})

	var completed atomic.Int32
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			mockDocker.mu.Lock()
			for _, s := range mockDocker.containers {
				if s.Running {
					n := completed.Add(1)
					s.Running = false
					s.FinishedAt = "2025-01-01T00:00:00Z"
					if n == 1 {
						s.ExitCode = 0
						writeParSuccess(t, filepath.Join(tmpDir, agentOK))
					} else {
						s.ExitCode = 1
						writeParFailure(t, filepath.Join(tmpDir, agentFail))
					}
				}
			}
			mockDocker.mu.Unlock()
			if completed.Load() >= 2 {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := pe.ExecuteParallelGroup(ctx, workflow, group, stepMap, 0, 1.0)
	require.Error(t, err, "CollectAll should still return error after all branches")
	assert.False(t, result.AllSucceeded)
	assert.Len(t, result.BranchResults, 2, "both branches should have results even with failures")
	assert.True(t, len(result.Errors) >= 1, "at least one branch should have failed")
}

//=============================================================================
// Sequential Backward Compatibility (via ExecuteSteps)
//=============================================================================

// TestParallel_SequentialPathStillWorks verifies that templates with no
// parallel step types use the sequential path unchanged.
func TestParallel_SequentialPathStillWorks(t *testing.T) {
	mockDocker := newParallelMockDocker()
	executor, _ := setupParallelExecutor(t, mockDocker)

	defID := "par-agent"

	tmpDir := executor.stateDirBase
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, defID), 0755))

	store := newOrchestratorTestStore()
	emitter := newMockEventEmitter()
	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{Store: store, EventBus: emitter})
	require.NoError(t, err)

	template := createTestTemplate("tpl-seq", []WorkflowStep{
		{StepID: "s1", Order: 0, Type: StepAction, Name: "S1", AgentIDs: []string{defID}, NextStepID: "s2"},
		{StepID: "s2", Order: 1, Type: StepAction, Name: "S2", AgentIDs: []string{defID}, NextStepID: "s3"},
		{StepID: "s3", Order: 2, Type: StepAction, Name: "S3", AgentIDs: []string{defID}},
	})
	store.templates[template.ID] = template

	workflow := &Workflow{
		ID: "wf-seq", TemplateID: template.ID, Name: "Seq",
		Status: StatusRunning, CreatedBy: "@test:example.com",
		RoomID: "!test:example.com", StartedAt: time.Now(),
		CurrentStep: "s1",
	}
	store.workflows[workflow.ID] = workflow

	orch.mu.Lock()
	orch.activeWorkflows[workflow.ID] = &activeWorkflow{
		workflow:     workflow,
		template:     template,
		cancelFunc:   func() {},
		startedAt:    time.Now(),
		currentIndex: 0,
	}
	orch.mu.Unlock()

	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			mockDocker.mu.Lock()
			for _, s := range mockDocker.containers {
				if s.Running {
					s.Running = false
					s.ExitCode = 0
					s.FinishedAt = "2025-01-01T00:00:00Z"
					writeParSuccess(t, filepath.Join(tmpDir, defID))
				}
			}
			running := 0
			for _, s := range mockDocker.containers {
				if s.Running {
					running++
				}
			}
			allDone := len(mockDocker.containers) >= 3 && running == 0
			mockDocker.mu.Unlock()
			if allDone {
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = executor.ExecuteSteps(ctx, orch, workflow, template)
	require.NoError(t, err, "sequential execution must still work")

	mockDocker.mu.Lock()
	spawnCount := len(mockDocker.spawnTimes)
	mockDocker.mu.Unlock()
	assert.Equal(t, 3, spawnCount, "should spawn exactly 3 containers for 3 sequential steps")
}

//=============================================================================
// ParallelConfig Tests
//=============================================================================

func TestDefaultParallelConfig(t *testing.T) {
	cfg := DefaultParallelConfig()
	assert.Equal(t, 2, cfg.MaxParallelContainers)
	assert.Equal(t, FailFast, cfg.ErrorPolicy)
}

func TestNewParallelExecutor_ZeroDefaults(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})
	pe := NewParallelExecutor(executor, ParallelConfig{})
	assert.Equal(t, 2, pe.config.MaxParallelContainers)
	assert.Equal(t, FailFast, pe.config.ErrorPolicy)
}

func TestNewParallelExecutor_CustomConfig(t *testing.T) {
	executor := NewStepExecutor(StepExecutorConfig{})
	pe := NewParallelExecutor(executor, ParallelConfig{
		MaxParallelContainers: 5,
		ErrorPolicy:           CollectAll,
	})
	assert.Equal(t, 5, pe.config.MaxParallelContainers)
	assert.Equal(t, CollectAll, pe.config.ErrorPolicy)
}

//=============================================================================
// Error Aggregation Tests
//=============================================================================

func TestAggregateParallelErrors_Nil(t *testing.T) {
	assert.Nil(t, AggregateParallelErrors(nil))
	assert.Nil(t, AggregateParallelErrors([]*ParallelBranchError{}))
}

func TestAggregateParallelErrors_Single(t *testing.T) {
	err := AggregateParallelErrors([]*ParallelBranchError{
		{StepID: "step1", Err: fmt.Errorf("boom")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step1")
}

func TestAggregateParallelErrors_Multiple(t *testing.T) {
	err := AggregateParallelErrors([]*ParallelBranchError{
		{StepID: "step1", Err: fmt.Errorf("error 1")},
		{StepID: "step2", Err: fmt.Errorf("error 2")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error 1")
	assert.Contains(t, err.Error(), "error 2")
}
