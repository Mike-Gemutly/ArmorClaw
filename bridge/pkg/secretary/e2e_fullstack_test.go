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

	"github.com/armorclaw/bridge/internal/events"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// E2E Full-Stack Test Suite
//
// Covers the full Bridge → Secretary → Sidecar data flows:
//   1. Workflow lifecycle: create → start → step execution → completion → event emission
//   2. WebSocket event delivery: publish → subscriber receives
//   3. Inter-step data propagation: step1 produces data → step2 consumes it
//   4. Event flow: MatrixEventBus → WorkflowEventEmitter → Subscriber
//   5. Event emission during step execution
//   6. Concurrent workflow execution
//
// Email HITL tests live in pkg/email/e2e_hitl_test.go (avoids import cycle).
// All tests use in-memory stores and mock Docker clients.
// No real services are started.
//=============================================================================

//=============================================================================
// Shared E2E Test Infrastructure
//=============================================================================

// e2eMockDocker implements studio.DockerClient for E2E tests.
// It supports sequential container creates, state transitions, and completion signals.
type e2eMockDocker struct {
	mu          sync.Mutex
	containers  map[string]*types.ContainerState
	nextID      int
	stopped     []string
	completedCh chan string // signals which container was completed
}

func newE2EMockDocker() *e2eMockDocker {
	return &e2eMockDocker{
		containers:  make(map[string]*types.ContainerState),
		completedCh: make(chan string, 20),
	}
}

func (m *e2eMockDocker) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ any, _ any, _ string) (container.CreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("e2e-container-%d", m.nextID)
	m.containers[id] = &types.ContainerState{Running: true, ExitCode: 0}
	return container.CreateResponse{ID: id}, nil
}

func (m *e2eMockDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *e2eMockDocker) ContainerStop(_ context.Context, cid string, _ container.StopOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, cid)
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *e2eMockDocker) ContainerKill(_ context.Context, cid string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
	}
	return nil
}

func (m *e2eMockDocker) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *e2eMockDocker) ContainerInspect(_ context.Context, cid string) (types.ContainerJSON, error) {
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

func (m *e2eMockDocker) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return nil, nil
}

func (m *e2eMockDocker) completeContainer(cid string) {
	m.mu.Lock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
		s.ExitCode = 0
		s.FinishedAt = "2025-01-01T00:00:00Z"
	}
	m.mu.Unlock()
	select {
	case m.completedCh <- cid:
	default:
	}
}

func (m *e2eMockDocker) failContainer(cid string) {
	m.mu.Lock()
	if s, ok := m.containers[cid]; ok {
		s.Running = false
		s.ExitCode = 1
		s.FinishedAt = "2025-01-01T00:00:00Z"
	}
	m.mu.Unlock()
}

// waitForContainer waits for a container to appear (created by Spawn).
func (m *e2eMockDocker) waitForContainer(t *testing.T, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		for id, s := range m.containers {
			if s.Running {
				m.mu.Unlock()
				return id
			}
		}
		m.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for container to be created")
	return ""
}

// setupE2ETest creates all the plumbing for E2E full-stack tests.
func setupE2ETest(t *testing.T) (
	*OrchestratorIntegration,
	*WorkflowOrchestratorImpl,
	*StepExecutor,
	*orchestratorTestStore,
	*mockEventEmitter,
	*e2eMockDocker,
	*studio.AgentFactory,
	*studio.SQLiteStore,
	string, // tmpDir
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

	mockDocker := newE2EMockDocker()
	factory := studio.NewAgentFactory(studio.FactoryConfig{
		StateDir:     tmpDir,
		DockerClient: mockDocker,
		Store:        studioStore,
	})

	executor := NewStepExecutor(StepExecutorConfig{
		Factory:        factory,
		DefaultTimeout: 30 * time.Second,
		StepRetryCount: 1,
		StateDirBase:   filepath.Join(tmpDir, "agent-state"),
	})

	integration := NewOrchestratorIntegration(IntegrationConfig{
		Orchestrator: orch,
		Executor:     executor,
		Store:        store,
	})

	return integration, orch, executor, store, emitter, mockDocker, factory, studioStore, tmpDir
}

// setupE2EAgent creates an agent definition and returns its ID.
func setupE2EAgent(t *testing.T, studioStore *studio.SQLiteStore) string {
	t.Helper()
	def := &studio.AgentDefinition{
		ID:           "e2e-test-agent",
		Name:         "E2E Test Agent",
		Skills:       []string{"browser_navigate"},
		ResourceTier: "medium",
		CreatedBy:    "@test:example.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	require.NoError(t, studioStore.CreateDefinition(def))
	return def.ID
}

// writeE2EResult writes a result.json to the agent state dir.
func writeE2EResult(t *testing.T, stateDir string, result ContainerStepResult) {
	t.Helper()
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "result.json"), resultJSON, 0644))

	// Create empty _events.jsonl so EventReader doesn't error
	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		f, err := os.Create(eventsPath)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
}

// writeE2EEvents writes structured events to _events.jsonl.
func writeE2EEvents(t *testing.T, stateDir string, evts []StepEvent) {
	t.Helper()
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	eventsPath := filepath.Join(stateDir, "_events.jsonl")
	f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	defer f.Close()
	for _, evt := range evts {
		line, err := json.Marshal(evt)
		require.NoError(t, err)
		_, err = f.WriteString(string(line) + "\n")
		require.NoError(t, err)
	}
}

//=============================================================================
// 1. Workflow Lifecycle E2E: create → start → step execution → completion → event
//=============================================================================

func TestE2E_WorkflowLifecycle_FullFlow(t *testing.T) {
	_, orch, executor, store, emitter, mockDocker, _, studioStore, tmpDir := setupE2ETest(t)
	agentID := setupE2EAgent(t, studioStore)
	stateDir := filepath.Join(tmpDir, "agent-state", agentID)

	// Create a 2-step template
	template := &TaskTemplate{
		ID:        "e2e-tpl-lifecycle",
		Name:      "E2E Lifecycle Test",
		CreatedBy: "@test:example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Steps: []WorkflowStep{
			{StepID: "step1", Order: 0, Type: StepAction, Name: "Fetch Data", AgentIDs: []string{agentID}},
			{StepID: "step2", Order: 1, Type: StepAction, Name: "Process Data", AgentIDs: []string{agentID}},
		},
	}
	require.NoError(t, store.CreateTemplate(context.Background(), template))

	// Create workflow
	workflow := &Workflow{
		ID:         "wf-e2e-lifecycle",
		TemplateID: template.ID,
		Name:       "E2E Lifecycle",
		Status:     StatusPending,
		CreatedBy:  "@test:example.com",
		RoomID:     "!room-lifecycle:example.com",
	}
	require.NoError(t, store.CreateWorkflow(context.Background(), workflow))

	// Start workflow
	require.NoError(t, orch.StartWorkflow("wf-e2e-lifecycle"))
	wf, err := orch.GetWorkflow("wf-e2e-lifecycle")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, wf.Status)

	events := emitter.getEvents()
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, WorkflowEventStarted, events[0].eventType)

	// Execute steps manually (simulating what runWorkflow does)
	ctx := context.Background()
	accumulated := make(map[string]map[string]any)

	// --- Step 1: Execute and produce data ---
	go func() {
		time.Sleep(300 * time.Millisecond)
		writeE2EResult(t, stateDir, ContainerStepResult{
			Status: "success",
			Output: "fetched order data",
			Data:   map[string]any{"order_id": "ORD-E2E-001", "total": 299.99},
		})
		mockDocker.completeContainer(mockDocker.waitForContainer(t, 5*time.Second))
	}()

	step1Result := executor.executeStepWithRetry(ctx, wf, template.Steps[0])
	require.NoError(t, step1Result.Err, "step1 should succeed")
	assert.Equal(t, "step1", step1Result.StepID)
	assert.NotNil(t, step1Result.ContainerResult)
	assert.Equal(t, "success", step1Result.ContainerResult.Status)
	assert.Equal(t, "ORD-E2E-001", step1Result.ContainerResult.Data["order_id"])

	// Accumulate step1 data
	if step1Result.ContainerResult != nil && len(step1Result.ContainerResult.Data) > 0 {
		accumulated["step1"] = step1Result.ContainerResult.Data
	}

	require.NoError(t, orch.AdvanceWorkflow("wf-e2e-lifecycle", "step1"))

	// --- Step 2: Execute and consume step1 data ---
	go func() {
		time.Sleep(300 * time.Millisecond)
		writeE2EResult(t, stateDir, ContainerStepResult{
			Status: "success",
			Output: "processed order ORD-E2E-001",
		})
		mockDocker.completeContainer(mockDocker.waitForContainer(t, 5*time.Second))
	}()

	step2Result := executor.executeStepWithRetry(ctx, wf, template.Steps[1])
	require.NoError(t, step2Result.Err, "step2 should succeed")
	assert.Equal(t, "step2", step2Result.StepID)

	require.NoError(t, orch.AdvanceWorkflow("wf-e2e-lifecycle", "step2"))

	// Verify workflow is completed
	wf, err = orch.GetWorkflow("wf-e2e-lifecycle")
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, wf.Status)
	assert.NotNil(t, wf.CompletedAt)

	// Verify event emission order
	events = emitter.getEvents()
	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.eventType
	}
	assert.Contains(t, eventTypes, WorkflowEventStarted)
	assert.Contains(t, eventTypes, WorkflowEventCompleted)
}

func TestE2E_WorkflowLifecycle_CancelDuringExecution(t *testing.T) {
	_, orch, executor, store, emitter, _, _, studioStore, _ := setupE2ETest(t)
	agentID := setupE2EAgent(t, studioStore)

	template := &TaskTemplate{
		ID:        "e2e-tpl-cancel",
		Name:      "E2E Cancel Test",
		CreatedBy: "@test:example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Steps: []WorkflowStep{
			{StepID: "step1", Order: 0, Type: StepAction, Name: "Long Task", AgentIDs: []string{agentID}},
		},
	}
	require.NoError(t, store.CreateTemplate(context.Background(), template))

	workflow := &Workflow{
		ID:         "wf-e2e-cancel",
		TemplateID: template.ID,
		Name:       "E2E Cancel",
		Status:     StatusPending,
		CreatedBy:  "@test:example.com",
		RoomID:     "!room-cancel:example.com",
	}
	require.NoError(t, store.CreateWorkflow(context.Background(), workflow))

	require.NoError(t, orch.StartWorkflow("wf-e2e-cancel"))

	// Start step execution with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	stepDone := make(chan struct{})
	go func() {
		defer close(stepDone)
		// Step will hang because we never complete the container
		_ = executor.executeStepWithRetry(ctx, workflow, template.Steps[0])
	}()

	// Wait for container to be spawned
	time.Sleep(500 * time.Millisecond)

	// Cancel the workflow
	require.NoError(t, orch.CancelWorkflow("wf-e2e-cancel", "user cancelled"))

	// Cancel the execution context
	cancel()

	<-stepDone // Wait for step to finish

	// Verify cancelled state
	wf, err := orch.GetWorkflow("wf-e2e-cancel")
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, wf.Status)

	// Verify cancel event was emitted
	events := emitter.getEvents()
	cancelledEvents := 0
	for _, e := range events {
		if e.eventType == WorkflowEventCancelled {
			cancelledEvents++
		}
	}
	assert.GreaterOrEqual(t, cancelledEvents, 1)
}

func TestE2E_WorkflowLifecycle_FailedStep(t *testing.T) {
	_, orch, executor, store, emitter, mockDocker, _, studioStore, tmpDir := setupE2ETest(t)
	agentID := setupE2EAgent(t, studioStore)
	stateDir := filepath.Join(tmpDir, "agent-state", agentID)

	template := &TaskTemplate{
		ID:        "e2e-tpl-fail",
		Name:      "E2E Fail Test",
		CreatedBy: "@test:example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Steps: []WorkflowStep{
			{StepID: "step1", Order: 0, Type: StepAction, Name: "Failing Step", AgentIDs: []string{agentID}},
		},
	}
	require.NoError(t, store.CreateTemplate(context.Background(), template))

	workflow := &Workflow{
		ID:         "wf-e2e-fail",
		TemplateID: template.ID,
		Name:       "E2E Fail",
		Status:     StatusPending,
		CreatedBy:  "@test:example.com",
		RoomID:     "!room-fail:example.com",
	}
	require.NoError(t, store.CreateWorkflow(context.Background(), workflow))

	require.NoError(t, orch.StartWorkflow("wf-e2e-fail"))

	go func() {
		time.Sleep(300 * time.Millisecond)
		writeE2EResult(t, stateDir, ContainerStepResult{
			Status: "failed",
			Error:  "database connection refused",
		})
		cid := mockDocker.waitForContainer(t, 5*time.Second)
		mockDocker.failContainer(cid)
	}()

	ctx := context.Background()
	result := executor.executeStepWithRetry(ctx, workflow, template.Steps[0])
	assert.NotNil(t, result)
	assert.Error(t, result.Err, "step with exit code 1 should report error")

	orch.FailWorkflow("wf-e2e-fail", "step1", fmt.Errorf("step failed: %v", result.Err), false)

	wf, err := orch.GetWorkflow("wf-e2e-fail")
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, wf.Status)

	events := emitter.getEvents()
	failedEvents := 0
	for _, e := range events {
		if e.eventType == WorkflowEventFailed {
			failedEvents++
		}
	}
	assert.GreaterOrEqual(t, failedEvents, 1)
}

//=============================================================================
// 2. WebSocket Event Delivery E2E: EventBus publish → subscriber receives
//=============================================================================

func TestE2E_WebSocketEventDelivery(t *testing.T) {
	bus := eventbus.NewEventBus(eventbus.Config{
		WebSocketEnabled: false,
	})
	defer bus.Stop()

	sub, err := bus.Subscribe(eventbus.EventFilter{})
	require.NoError(t, err)

	workflow := &Workflow{
		ID:         "wf-e2e-ws",
		TemplateID: "tpl-ws",
		Name:       "WebSocket E2E",
		Status:     StatusRunning,
		CreatedBy:  "@ws-test:example.com",
		RoomID:     "!ws-room:example.com",
		StartedAt:  time.Now(),
	}

	bus.Publish(&eventbus.MatrixEvent{
		Type:   WorkflowEventStarted,
		RoomID: "!ws-room:example.com",
		Sender: "orchestrator",
		Content: map[string]interface{}{
			"workflow_id": workflow.ID,
			"status":      string(StatusRunning),
		},
		EventID: "$evt-ws-1",
	})

	select {
	case wrapper := <-sub.EventChannel:
		assert.Equal(t, WorkflowEventStarted, wrapper.Event.Type)
		assert.Equal(t, "!ws-room:example.com", wrapper.Event.RoomID)
		assert.Equal(t, "wf-e2e-ws", wrapper.Event.Content["workflow_id"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for workflow.started event on subscriber")
	}

	bus.Publish(&eventbus.MatrixEvent{
		Type:   WorkflowEventProgress,
		RoomID: "!ws-room:example.com",
		Sender: "orchestrator",
		Content: map[string]interface{}{
			"workflow_id": workflow.ID,
			"step_id":     "step1",
			"progress":    0.5,
		},
		EventID: "$evt-ws-2",
	})

	select {
	case wrapper := <-sub.EventChannel:
		assert.Equal(t, WorkflowEventProgress, wrapper.Event.Type)
		assert.Equal(t, "step1", wrapper.Event.Content["step_id"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for workflow.progress event")
	}

	bus.Publish(&eventbus.MatrixEvent{
		Type:   WorkflowEventCompleted,
		RoomID: "!ws-room:example.com",
		Sender: "orchestrator",
		Content: map[string]interface{}{
			"workflow_id": workflow.ID,
			"status":      string(StatusCompleted),
			"result":      "all steps done",
		},
		EventID: "$evt-ws-3",
	})

	select {
	case wrapper := <-sub.EventChannel:
		assert.Equal(t, WorkflowEventCompleted, wrapper.Event.Type)
		assert.Equal(t, "all steps done", wrapper.Event.Content["result"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for workflow.completed event")
	}
}

func TestE2E_WebSocketFilteredDelivery(t *testing.T) {
	bus := eventbus.NewEventBus(eventbus.Config{
		WebSocketEnabled: false,
	})
	defer bus.Stop()

	// Subscribe to only room1 events
	sub1, err := bus.Subscribe(eventbus.EventFilter{RoomID: "!room1:example.com"})
	require.NoError(t, err)

	// Subscribe to all events
	sub2, err := bus.Subscribe(eventbus.EventFilter{})
	require.NoError(t, err)

	// Publish event for room1
	require.NoError(t, bus.Publish(&eventbus.MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "hello room1"},
		EventID: "$evt1",
	}))

	// Publish event for room2 (should be filtered by sub1)
	require.NoError(t, bus.Publish(&eventbus.MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room2:example.com",
		Sender:  "@bob:example.com",
		Content: map[string]interface{}{"body": "hello room2"},
		EventID: "$evt2",
	}))

	// sub1 should only receive room1 event
	select {
	case wrapper := <-sub1.EventChannel:
		assert.Equal(t, "!room1:example.com", wrapper.Event.RoomID)
	case <-time.After(2 * time.Second):
		t.Fatal("sub1 timed out waiting for room1 event")
	}

	select {
	case <-sub1.EventChannel:
		t.Fatal("sub1 should NOT receive room2 events")
	case <-time.After(200 * time.Millisecond):
		// Expected: no event for room2
	}

	// sub2 should receive both events
	select {
	case wrapper := <-sub2.EventChannel:
		assert.Equal(t, "!room1:example.com", wrapper.Event.RoomID)
	case <-time.After(2 * time.Second):
		t.Fatal("sub2 timed out waiting for room1 event")
	}
	select {
	case wrapper := <-sub2.EventChannel:
		assert.Equal(t, "!room2:example.com", wrapper.Event.RoomID)
	case <-time.After(2 * time.Second):
		t.Fatal("sub2 timed out waiting for room2 event")
	}
}

//=============================================================================
// 3. Inter-Step Data Propagation E2E
//=============================================================================

func TestE2E_InterStepDataPropagation(t *testing.T) {
	_, orch, executor, store, _, mockDocker, _, studioStore, tmpDir := setupE2ETest(t)
	agentID := setupE2EAgent(t, studioStore)
	stateDir := filepath.Join(tmpDir, "agent-state", agentID)

	template := &TaskTemplate{
		ID:        "e2e-tpl-dataprop",
		Name:      "E2E Data Propagation",
		CreatedBy: "@test:example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Steps: []WorkflowStep{
			{
				StepID:   "fetch_order",
				Order:    0,
				Type:     StepAction,
				Name:     "Fetch Order",
				AgentIDs: []string{agentID},
			},
			{
				StepID:   "process_payment",
				Order:    1,
				Type:     StepAction,
				Name:     "Process Payment",
				AgentIDs: []string{agentID},
				Input: map[string]any{
					"order_ref":   "{{steps.fetch_order.data.order_id}}",
					"order_total": "{{steps.fetch_order.data.total}}",
					"customer":    "John Doe", // static value
				},
			},
		},
	}
	require.NoError(t, store.CreateTemplate(context.Background(), template))

	workflow := &Workflow{
		ID:         "wf-e2e-dataprop",
		TemplateID: template.ID,
		Name:       "E2E Data Prop",
		Status:     StatusPending,
		CreatedBy:  "@test:example.com",
		RoomID:     "!room-dataprop:example.com",
	}
	require.NoError(t, store.CreateWorkflow(context.Background(), workflow))

	require.NoError(t, orch.StartWorkflow("wf-e2e-dataprop"))

	ctx := context.Background()
	accumulated := make(map[string]map[string]any)

	// Step 1: Produce data
	go func() {
		time.Sleep(300 * time.Millisecond)
		writeE2EResult(t, stateDir, ContainerStepResult{
			Status:   "success",
			Output:   "order fetched",
			Data:     map[string]any{"order_id": "ORD-DATA-001", "total": 149.99},
			DurationMS: 500,
		})
		mockDocker.completeContainer(mockDocker.waitForContainer(t, 5*time.Second))
	}()

	step1Result := executor.executeStepWithRetry(ctx, workflow, template.Steps[0])
	require.NoError(t, step1Result.Err)
	require.NotNil(t, step1Result.ContainerResult)
	assert.Equal(t, "ORD-DATA-001", step1Result.ContainerResult.Data["order_id"])

	accumulated["fetch_order"] = step1Result.ContainerResult.Data
	require.NoError(t, orch.AdvanceWorkflow("wf-e2e-dataprop", "fetch_order"))

	// Step 2: Consume step1 data via template resolution
	step2 := template.Steps[1]
	resolved, err := resolveStepInput(step2.Input, accumulated)
	require.NoError(t, err)

	// Verify template resolution worked
	assert.Equal(t, "ORD-DATA-001", resolved["order_ref"])
	assert.Equal(t, "149.99", resolved["order_total"])
	assert.Equal(t, "John Doe", resolved["customer"])

	// Inject resolved data into step config
	step2.Config = injectPrevStepData(step2.Config, resolved)
	require.NotNil(t, step2.Config)

	var injectedConfig map[string]interface{}
	require.NoError(t, json.Unmarshal(step2.Config, &injectedConfig))
	prevData := injectedConfig["_prev_step_data"].(map[string]interface{})
	assert.Equal(t, "ORD-DATA-001", prevData["order_ref"])
	assert.Equal(t, "149.99", prevData["order_total"])

	// Execute step2 with the injected data
	go func() {
		time.Sleep(300 * time.Millisecond)
		writeE2EResult(t, stateDir, ContainerStepResult{
			Status:   "success",
			Output:   fmt.Sprintf("payment processed for order %v", prevData["order_ref"]),
			DurationMS: 300,
		})
		mockDocker.completeContainer(mockDocker.waitForContainer(t, 5*time.Second))
	}()

	step2Result := executor.executeStepWithRetry(ctx, workflow, step2)
	require.NoError(t, step2Result.Err)
	assert.Equal(t, "process_payment", step2Result.StepID)

	require.NoError(t, orch.AdvanceWorkflow("wf-e2e-dataprop", "process_payment"))

	wf, err := orch.GetWorkflow("wf-e2e-dataprop")
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, wf.Status)
}

func TestE2E_InterStepDataPropagation_MultipleReferences(t *testing.T) {
	accumulated := map[string]map[string]any{
		"step_a": {"key1": "valueA", "key2": 42},
		"step_b": {"key1": "valueB", "nested": map[string]any{"deep": true}},
	}

	// Test multiple references in one template string
	result := resolveTemplateString(
		"ref1={{steps.step_a.data.key1}} ref2={{steps.step_b.data.key1}} num={{steps.step_a.data.key2}}",
		accumulated,
	)
	assert.Equal(t, "ref1=valueA ref2=valueB num=42", result)

	// Test missing reference preserves template
	result = resolveTemplateString("{{steps.nonexistent.data.key}}", accumulated)
	assert.Equal(t, "{{steps.nonexistent.data.key}}", result)
}

//=============================================================================
// 4. Event Flow E2E: MatrixEventBus → WorkflowEventEmitter → Subscriber
//=============================================================================

func TestE2E_EventFlow_FullLifecycle(t *testing.T) {
	matrixBus := events.NewMatrixEventBus(256)

	// Subscribe to the Matrix event bus
	subCh := matrixBus.Subscribe()

	workflow := &Workflow{
		ID:         "wf-e2e-eventflow",
		TemplateID: "tpl-eventflow",
		Name:       "E2E Event Flow",
		Status:     StatusRunning,
		CreatedBy:  "@eventflow:example.com",
		RoomID:     "!room-eventflow:example.com",
		StartedAt:  time.Now(),
	}

	emitter := NewWorkflowEventEmitter(matrixBus)

	// Emit all lifecycle events
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		emitter.EmitStarted(workflow)
		wg.Done()
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		emitter.EmitProgress(workflow, "step1", "Step 1", 0.33)
		wg.Done()
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		emitter.EmitProgress(workflow, "step2", "Step 2", 0.66)
		wg.Done()
	}()
	go func() {
		time.Sleep(30 * time.Millisecond)
		emitter.EmitProgress(workflow, "step3", "Step 3", 1.0)
		wg.Done()
	}()
	go func() {
		time.Sleep(40 * time.Millisecond)
		emitter.EmitCompleted(workflow, "all done")
		wg.Done()
	}()
	wg.Wait()

	// Collect all events from the subscriber
	var receivedEvents []events.MatrixEvent
	timeout := time.After(3 * time.Second)
	for len(receivedEvents) < 5 {
		select {
		case evt := <-subCh:
			receivedEvents = append(receivedEvents, evt)
		case <-timeout:
			t.Fatalf("timed out waiting for events, got %d/5", len(receivedEvents))
		}
	}

	// Verify event types
	eventTypes := make(map[string]int)
	for _, evt := range receivedEvents {
		eventTypes[evt.Type]++
	}
	assert.Equal(t, 1, eventTypes[WorkflowEventStarted], "should have 1 started event")
	assert.Equal(t, 3, eventTypes[WorkflowEventProgress], "should have 3 progress events")
	assert.Equal(t, 1, eventTypes[WorkflowEventCompleted], "should have 1 completed event")

	// Verify all events have correct workflow ID
	for _, evt := range receivedEvents {
		content, ok := evt.Content.(WorkflowEvent)
		require.True(t, ok, "event content should be WorkflowEvent")
		assert.Equal(t, "wf-e2e-eventflow", content.WorkflowID)
	}
}

//=============================================================================
// 5. Event Emission During Step Execution
//=============================================================================

func TestE2E_StepEvents_EmittedToMatrix(t *testing.T) {
	_, orch, executor, store, emitter, mockDocker, _, studioStore, tmpDir := setupE2ETest(t)
	agentID := setupE2EAgent(t, studioStore)
	stateDir := filepath.Join(tmpDir, "agent-state", agentID)

	template := &TaskTemplate{
		ID:        "e2e-tpl-events",
		Name:      "E2E Events",
		CreatedBy: "@test:example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
		Steps: []WorkflowStep{
			{StepID: "step1", Order: 0, Type: StepAction, Name: "Produce Events", AgentIDs: []string{agentID}},
		},
	}
	require.NoError(t, store.CreateTemplate(context.Background(), template))

	workflow := &Workflow{
		ID:         "wf-e2e-events",
		TemplateID: template.ID,
		Name:       "E2E Events",
		Status:     StatusPending,
		CreatedBy:  "@test:example.com",
		RoomID:     "!room-events:example.com",
	}
	require.NoError(t, store.CreateWorkflow(context.Background(), workflow))
	require.NoError(t, orch.StartWorkflow("wf-e2e-events"))

	// Step writes events AND result
	go func() {
		time.Sleep(300 * time.Millisecond)

		// Write step events to _events.jsonl
		writeE2EEvents(t, stateDir, []StepEvent{
			{Seq: 1, Type: "step", Name: "initialize", TsMs: time.Now().UnixMilli()},
			{Seq: 2, Type: "progress", Name: "loading", TsMs: time.Now().UnixMilli()},
			{Seq: 3, Type: "checkpoint", Name: "data_loaded", TsMs: time.Now().UnixMilli()},
		})

		writeE2EResult(t, stateDir, ContainerStepResult{
			Status:   "success",
			Output:   "completed with events",
			DurationMS: 1500,
		})
		mockDocker.completeContainer(mockDocker.waitForContainer(t, 5*time.Second))
	}()

	ctx := context.Background()
	result := executor.executeStepWithRetry(ctx, workflow, template.Steps[0])
	require.NoError(t, result.Err)
	assert.NotNil(t, result.ContainerResult)
	assert.Equal(t, "success", result.ContainerResult.Status)

	// Verify orchestrator events were emitted
	events := emitter.getEvents()
	eventTypes := make(map[string]bool)
	for _, e := range events {
		eventTypes[e.eventType] = true
	}
	assert.True(t, eventTypes[WorkflowEventStarted], "should emit started event")
}

//=============================================================================
// 6. Concurrent Workflow Execution
//=============================================================================

func TestE2E_ConcurrentWorkflows(t *testing.T) {
	_, orch, _, store, _, _, _, _, _ := setupE2ETest(t)

	for i := 0; i < 2; i++ {
		template := &TaskTemplate{
			ID:        fmt.Sprintf("e2e-tpl-concurrent-%d", i),
			Name:      fmt.Sprintf("E2E Concurrent %d", i),
			CreatedBy: "@test:example.com",
			CreatedAt: time.Now(),
			IsActive:  true,
			Steps: []WorkflowStep{
				{StepID: fmt.Sprintf("step-c%d", i), Order: 0, Type: StepAction, Name: fmt.Sprintf("Concurrent Step %d", i)},
			},
		}
		require.NoError(t, store.CreateTemplate(context.Background(), template))

		workflow := &Workflow{
			ID:         fmt.Sprintf("wf-concurrent-%d", i),
			TemplateID: template.ID,
			Name:       fmt.Sprintf("Concurrent %d", i),
			Status:     StatusPending,
			CreatedBy:  "@test:example.com",
			RoomID:     fmt.Sprintf("!room-concurrent-%d:example.com", i),
		}
		require.NoError(t, store.CreateWorkflow(context.Background(), workflow))
	}

	require.NoError(t, orch.StartWorkflow("wf-concurrent-0"))
	require.NoError(t, orch.StartWorkflow("wf-concurrent-1"))

	assert.Equal(t, 2, orch.GetActiveWorkflowCount())

	require.NoError(t, orch.AdvanceWorkflow("wf-concurrent-0", "step-c0"))
	require.NoError(t, orch.AdvanceWorkflow("wf-concurrent-1", "step-c1"))

	wf0, _ := orch.GetWorkflow("wf-concurrent-0")
	wf1, _ := orch.GetWorkflow("wf-concurrent-1")
	assert.Equal(t, StatusCompleted, wf0.Status)
	assert.Equal(t, StatusCompleted, wf1.Status)
	assert.Equal(t, 0, orch.GetActiveWorkflowCount())
}
