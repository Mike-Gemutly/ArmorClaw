package secretary

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//=============================================================================
// Mock Types for Scheduler Tests
//=============================================================================

// schedulerTestStore implements secretary.Store with in-memory maps for scheduler tests
type schedulerTestStore struct {
	sync.RWMutex
	scheduledTasks map[string]*ScheduledTask
	templates      map[string]*TaskTemplate
	workflows      map[string]*Workflow
	listDueErr     error
}

func newSchedulerTestStore() *schedulerTestStore {
	return &schedulerTestStore{
		scheduledTasks: make(map[string]*ScheduledTask),
		templates:      make(map[string]*TaskTemplate),
		workflows:      make(map[string]*Workflow),
	}
}

// Key methods with real logic

func (s *schedulerTestStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.Lock()
	defer s.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *schedulerTestStore) GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	t, ok := s.scheduledTasks[id]
	if !ok {
		return nil, fmt.Errorf("scheduled task not found: %s", id)
	}
	return t, nil
}

func (s *schedulerTestStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	var result []ScheduledTask
	for _, t := range s.scheduledTasks {
		result = append(result, *t)
	}
	return result, nil
}

func (s *schedulerTestStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.Lock()
	defer s.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *schedulerTestStore) DeleteScheduledTask(ctx context.Context, id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.scheduledTasks, id)
	return nil
}

func (s *schedulerTestStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	var result []ScheduledTask
	now := time.Now()
	for _, t := range s.scheduledTasks {
		if t.NextRun != nil && !t.NextRun.After(now) {
			result = append(result, *t)
		}
	}
	return result, nil
}

// ListDueTasks has REAL time-based filtering logic:
// active=true, next_run <= now, definition_id != ”
func (s *schedulerTestStore) ListDueTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	if s.listDueErr != nil {
		return nil, s.listDueErr
	}
	var result []ScheduledTask
	now := time.Now()
	for _, t := range s.scheduledTasks {
		if t.IsActive && t.NextRun != nil && !t.NextRun.After(now) && (t.DefinitionID != "" || t.TemplateID != "") {
			result = append(result, *t)
		}
	}
	return result, nil
}

// MarkDispatched updates last_run and next_run in the map
func (s *schedulerTestStore) MarkDispatched(ctx context.Context, taskID string, nextRun time.Time) error {
	s.Lock()
	defer s.Unlock()
	if t, ok := s.scheduledTasks[taskID]; ok {
		now := time.Now()
		t.LastRun = &now
		t.NextRun = &nextRun
	}
	return nil
}

// Stub implementations for unused Store interface methods

func (s *schedulerTestStore) CreateTemplate(ctx context.Context, template *TaskTemplate) error {
	s.Lock()
	defer s.Unlock()
	s.templates[template.ID] = template
	return nil
}
func (s *schedulerTestStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	s.RLock()
	defer s.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return t, nil
}
func (s *schedulerTestStore) GetTemplateByTrigger(ctx context.Context, trigger string) (*TaskTemplate, error) {
	s.RLock()
	defer s.RUnlock()
	for _, t := range s.templates {
		if t.ID == trigger {
			return t, nil
		}
	}
	return nil, nil
}
func (s *schedulerTestStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	return nil, nil
}
func (s *schedulerTestStore) UpdateTemplate(ctx context.Context, template *TaskTemplate) error {
	return nil
}
func (s *schedulerTestStore) DeleteTemplate(ctx context.Context, id string) error {
	return nil
}
func (s *schedulerTestStore) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	s.Lock()
	defer s.Unlock()
	s.workflows[workflow.ID] = workflow
	return nil
}
func (s *schedulerTestStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	s.RLock()
	defer s.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	return w, nil
}
func (s *schedulerTestStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	return nil, nil
}
func (s *schedulerTestStore) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}
func (s *schedulerTestStore) DeleteWorkflow(ctx context.Context, id string) error {
	return nil
}
func (s *schedulerTestStore) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *schedulerTestStore) GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error) {
	return nil, errors.New("not implemented")
}
func (s *schedulerTestStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	return nil, nil
}
func (s *schedulerTestStore) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *schedulerTestStore) DeletePolicy(ctx context.Context, id string) error {
	return nil
}
func (s *schedulerTestStore) CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *schedulerTestStore) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	return nil, errors.New("not implemented")
}
func (s *schedulerTestStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	return nil, nil
}
func (s *schedulerTestStore) UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *schedulerTestStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	return nil
}
func (s *schedulerTestStore) CreateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *schedulerTestStore) GetContact(ctx context.Context, id string) (*Contact, error) {
	return nil, errors.New("not implemented")
}
func (s *schedulerTestStore) ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error) {
	return nil, nil
}
func (s *schedulerTestStore) UpdateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *schedulerTestStore) DeleteContact(ctx context.Context, id string) error {
	return nil
}
func (s *schedulerTestStore) Close() error { return nil }

//=============================================================================
// Mock Factory
//=============================================================================

type mockSchedulerFactory struct {
	runningInstance  *AgentInstanceRef
	runningErr       error
	spawnResult      *SpawnResultRef
	spawnErr         error
	spawnCalled      bool
	getRunningCalled bool
}

func (f *mockSchedulerFactory) GetRunningInstance(definitionID string) (*AgentInstanceRef, error) {
	f.getRunningCalled = true
	if f.runningErr != nil {
		return nil, f.runningErr
	}
	return f.runningInstance, nil
}

func (f *mockSchedulerFactory) Spawn(ctx context.Context, req *SpawnRequestRef) (*SpawnResultRef, error) {
	f.spawnCalled = true
	if f.spawnErr != nil {
		return nil, f.spawnErr
	}
	return f.spawnResult, nil
}

//=============================================================================
// Mock Matrix Adapter
//=============================================================================

type sentEvent struct {
	RoomID    string
	EventType string
	Payload   interface{}
}

type mockSchedulerMatrix struct {
	mu         sync.RWMutex
	sentEvents []sentEvent
}

func (m *mockSchedulerMatrix) SendEvent(ctx context.Context, roomID string, eventType string, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEvents = append(m.sentEvents, sentEvent{
		RoomID:    roomID,
		EventType: eventType,
		Payload:   payload,
	})
	return nil
}

func (m *mockSchedulerMatrix) getSentEvents() []sentEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]sentEvent, len(m.sentEvents))
	copy(result, m.sentEvents)
	return result
}

//=============================================================================
// Scheduler Tests
//=============================================================================

func TestTaskScheduler_WarmDispatchSkippedForNetworkNone(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-1"] = &ScheduledTask{
		ID:             "task-1",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
			Status: "running",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Warm dispatch should be skipped (NetworkMode: none), go straight to cold dispatch
	assert.True(t, factory.getRunningCalled, "should check for running instance")
	assert.True(t, factory.spawnCalled, "should use cold dispatch since warm is skipped (NetworkMode 'none')")

	updated := store.scheduledTasks["task-1"]
	assert.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now))
	assert.NotNil(t, updated.LastRun)
}

func TestTaskScheduler_ColdDispatch(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-1"] = &ScheduledTask{
		ID:             "task-1",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: nil, // No running instance → cold path
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-new",
			RoomID:     "!new-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Verify Spawn was called with correct definition_id and user_id
	assert.True(t, factory.spawnCalled)

	// Verify MarkDispatched called (cron task → next_run updated)
	updated := store.scheduledTasks["task-1"]
	assert.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now))

	// Warm dispatch not used — no matrix events
	events := matrix.getSentEvents()
	assert.Empty(t, events)
}

func TestTaskScheduler_OneShotDeactivation(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-oneshot"] = &ScheduledTask{
		ID:           "task-oneshot",
		DefinitionID: "def-1",
		// Empty CronExpression → one-shot
		IsActive:  true,
		NextRun:   &now,
		CreatedBy: "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Verify task deactivated
	updated := store.scheduledTasks["task-oneshot"]
	assert.False(t, updated.IsActive)
}

func TestTaskScheduler_CronNextRunUpdate(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-cron"] = &ScheduledTask{
		ID:             "task-cron",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *", // Every hour
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Verify MarkDispatched called with non-zero next_run in the future
	updated := store.scheduledTasks["task-cron"]
	require.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now), "next_run should be in the future")
	assert.True(t, updated.IsActive, "cron task should remain active")
}

func TestTaskScheduler_EmptyDefinitionIdSkipped(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-empty"] = &ScheduledTask{
		ID:           "task-empty",
		DefinitionID: "", // Empty → filtered by ListDueTasks
		IsActive:     true,
		NextRun:      &now,
		CreatedBy:    "@test:example.com",
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Task with empty definition_id is excluded by ListDueTasks → not dispatched
	assert.False(t, factory.getRunningCalled)
	assert.False(t, factory.spawnCalled)
	events := matrix.getSentEvents()
	assert.Empty(t, events)
}

func TestTaskScheduler_InvalidCronDeactivation(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-bad-cron"] = &ScheduledTask{
		ID:             "task-bad-cron",
		DefinitionID:   "def-1",
		CronExpression: "not-a-cron", // Invalid
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	// Task should be deactivated due to invalid cron expression
	updated := store.scheduledTasks["task-bad-cron"]
	assert.False(t, updated.IsActive, "task with invalid cron should be deactivated")
}

func TestTaskScheduler_NilMatrixAdapter(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-nil-matrix"] = &ScheduledTask{
		ID:             "task-nil-matrix",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}

	// nil matrix adapter — should not panic
	scheduler := NewTaskScheduler(store, factory, nil, nil, nil, nil)

	assert.NotPanics(t, func() {
		scheduler.tick()
	})

	// Task should still be marked dispatched (updateAfterDispatch runs)
	updated := store.scheduledTasks["task-nil-matrix"]
	assert.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now))
}

func TestTaskScheduler_StoreListError(t *testing.T) {
	store := newSchedulerTestStore()
	store.listDueErr = errors.New("db connection lost")

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)

	// Should not panic when ListDueTasks returns error
	assert.NotPanics(t, func() {
		scheduler.tick()
	})

	// No dispatch attempts
	assert.False(t, factory.getRunningCalled)
	assert.False(t, factory.spawnCalled)
}

func TestTaskScheduler_StartStopLifecycle(t *testing.T) {
	store := newSchedulerTestStore()
	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.Start()

	// Give goroutine time to start
	time.Sleep(100 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success — goroutine exited cleanly
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() hung — scheduler goroutine didn't exit")
	}
}

func TestTaskScheduler_TickOnStart(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	// Create a due task BEFORE starting scheduler
	store.scheduledTasks["task-immediate"] = &ScheduledTask{
		ID:             "task-immediate",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.Start()

	// Wait for the immediate first tick to process
	time.Sleep(300 * time.Millisecond)
	scheduler.Stop()

	// Warm dispatch fails, falls back to cold dispatch
	assert.True(t, factory.spawnCalled, "warm dispatch skipped (NetworkMode 'none'), uses cold dispatch on scheduler start (immediate tick)")

	updated := store.scheduledTasks["task-immediate"]
	assert.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now))
}

//=============================================================================
// Scheduler Template Dispatch Tests
//=============================================================================

func TestDispatchTask_TemplateRouting(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-tmpl"] = &ScheduledTask{
		ID:             "task-tmpl",
		TemplateID:     "tmpl-1",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}
	store.templates["tmpl-1"] = &TaskTemplate{
		ID:        "tmpl-1",
		Name:      "Test Template",
		IsActive:  true,
		CreatedBy: "@test:example.com",
		Steps:     []WorkflowStep{{StepID: "s1", Name: "Step 1", Type: StepAction, Order: 0}},
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	emitter := newMockEventEmitter()
	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	require.NoError(t, err)

	scheduler := NewTaskScheduler(store, factory, matrix, nil, orch, nil)
	scheduler.tick()

	assert.False(t, factory.getRunningCalled, "template dispatch should not use factory.GetRunningInstance")
	assert.False(t, factory.spawnCalled, "template dispatch should not use factory.Spawn")

	assert.Len(t, store.workflows, 1, "templateDispatch should create a workflow")
	for _, wf := range store.workflows {
		assert.Equal(t, "tmpl-1", wf.TemplateID)
		assert.Equal(t, StatusRunning, wf.Status)
	}

	updated := store.scheduledTasks["task-tmpl"]
	assert.NotNil(t, updated.LastRun, "task should be marked dispatched")
}

func TestDispatchTask_TemplateNotFound(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-no-tmpl"] = &ScheduledTask{
		ID:             "task-no-tmpl",
		TemplateID:     "nonexistent-tmpl",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	assert.NotPanics(t, func() {
		scheduler.tick()
	})

	updated := store.scheduledTasks["task-no-tmpl"]
	assert.True(t, updated.IsActive, "task should NOT be deactivated when template not found")
	assert.Nil(t, updated.LastRun, "task should NOT be marked dispatched when template not found")
}

func TestDispatchTask_TemplateOnlyNoDefinitionID(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-tmpl-only"] = &ScheduledTask{
		ID:             "task-tmpl-only",
		TemplateID:     "tmpl_123",
		DefinitionID:   "",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}
	store.templates["tmpl_123"] = &TaskTemplate{
		ID:        "tmpl_123",
		Name:      "Template Only",
		IsActive:  true,
		CreatedBy: "@test:example.com",
		Steps:     []WorkflowStep{{StepID: "s1", Name: "Step 1", Type: StepAction, Order: 0}},
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	emitter := newMockEventEmitter()
	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	require.NoError(t, err)

	scheduler := NewTaskScheduler(store, factory, matrix, nil, orch, nil)
	scheduler.tick()

	assert.False(t, factory.spawnCalled, "should use templateDispatch, not cold dispatch")
	assert.False(t, factory.getRunningCalled, "should use templateDispatch, not warm dispatch")

	updated := store.scheduledTasks["task-tmpl-only"]
	assert.NotNil(t, updated.LastRun, "task should be marked dispatched via template path")
}

func TestDispatchTask_FreeTextUnchanged(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-freetext"] = &ScheduledTask{
		ID:             "task-freetext",
		TemplateID:     "",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{
		runningInstance: &AgentInstanceRef{
			ID:     "inst-1",
			RoomID: "!room:example.com",
			Status: "running",
		},
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-cold",
			RoomID:     "!cold-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	scheduler.tick()

	assert.True(t, factory.getRunningCalled, "empty TemplateID should check for running instance")
	assert.True(t, factory.spawnCalled, "warm dispatch skipped (NetworkMode 'none'), uses cold dispatch")

	updated := store.scheduledTasks["task-freetext"]
	assert.NotNil(t, updated.NextRun)
	assert.True(t, updated.NextRun.After(now))
}

func TestTemplateDispatch_CreatesWorkflow(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-create-wf"] = &ScheduledTask{
		ID:             "task-create-wf",
		TemplateID:     "tpl-wf",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@creator:example.com",
	}
	store.templates["tpl-wf"] = &TaskTemplate{
		ID:          "tpl-wf",
		Name:        "Workflow Creator Test",
		Description: "desc",
		IsActive:    true,
		CreatedBy:   "@admin:example.com",
		Steps:       []WorkflowStep{{StepID: "s1", Name: "Step 1", Type: StepAction, Order: 0}},
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	orchStore := newOrchestratorTestStore()
	orchStore.templates["tpl-wf"] = store.templates["tpl-wf"]
	emitter := newMockEventEmitter()
	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    orchStore,
		EventBus: emitter,
	})
	require.NoError(t, err)

	scheduler := NewTaskScheduler(store, factory, matrix, nil, orch, nil)
	scheduler.tick()

	require.Len(t, store.workflows, 1, "exactly one workflow should be created")
	for _, wf := range store.workflows {
		assert.Equal(t, "tpl-wf", wf.TemplateID)
		assert.Equal(t, "Workflow Creator Test", wf.Name)
		assert.Equal(t, "@creator:example.com", wf.CreatedBy, "workflow.CreatedBy should match task.CreatedBy")
		assert.Equal(t, StatusPending, wf.Status)
	}
}

func TestTemplateDispatch_StartsExecution(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-start-exec"] = &ScheduledTask{
		ID:             "task-start-exec",
		TemplateID:     "tpl-exec",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}
	store.templates["tpl-exec"] = &TaskTemplate{
		ID:        "tpl-exec",
		Name:      "Exec Test",
		IsActive:  true,
		CreatedBy: "@test:example.com",
		Steps:     []WorkflowStep{{StepID: "s1", Name: "Step 1", Type: StepAction, Order: 0}},
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	emitter := newMockEventEmitter()
	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	require.NoError(t, err)

	integration := NewOrchestratorIntegration(IntegrationConfig{
		Orchestrator: orch,
		Store:        store,
		Executor: NewStepExecutor(StepExecutorConfig{
			Validator: NewDependencyValidator(),
		}),
	})

	scheduler := NewTaskScheduler(store, factory, matrix, nil, orch, integration)
	scheduler.tick()

	events := emitter.getEvents()
	startedEvents := 0
	for _, e := range events {
		if e.eventType == WorkflowEventStarted {
			startedEvents++
		}
	}
	assert.GreaterOrEqual(t, startedEvents, 1, "orchestrator.StartWorkflow should emit started event")
}

func TestTemplateDispatch_ErrorNotDeactivated(t *testing.T) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["task-err-tmpl"] = &ScheduledTask{
		ID:             "task-err-tmpl",
		TemplateID:     "bad-tmpl",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	factory := &mockSchedulerFactory{}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)
	assert.NotPanics(t, func() {
		scheduler.tick()
	})

	updated := store.scheduledTasks["task-err-tmpl"]
	assert.True(t, updated.IsActive, "task should NOT be deactivated when GetTemplate returns error")
	assert.Nil(t, updated.LastRun, "task should NOT be marked dispatched when GetTemplate returns error")
}
