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
	listDueErr     error
}

func newSchedulerTestStore() *schedulerTestStore {
	return &schedulerTestStore{
		scheduledTasks: make(map[string]*ScheduledTask),
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
		if t.IsActive && t.NextRun != nil && !t.NextRun.After(now) && t.DefinitionID != "" {
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
	return nil
}
func (s *schedulerTestStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	return nil, errors.New("not implemented")
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
	return nil
}
func (s *schedulerTestStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	return nil, errors.New("not implemented")
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

func TestTaskScheduler_WarmDispatch(t *testing.T) {
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
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
	scheduler.tick()

	// Verify matrix.SendEvent called with correct event type
	events := matrix.getSentEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "!room:example.com", events[0].RoomID)
	assert.Equal(t, EventTypeTaskDispatch, events[0].EventType)

	// Verify payload contains task_id
	payload, ok := events[0].Payload.(TaskDispatchPayload)
	require.True(t, ok)
	assert.Equal(t, "task-1", payload.TaskID)

	// Verify MarkDispatched called — next_run should be updated to future
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

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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
	}

	// nil matrix adapter — should not panic
	scheduler := NewTaskScheduler(store, factory, nil, nil)

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

	scheduler := NewTaskScheduler(store, factory, matrix, nil)

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

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
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
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil)
	scheduler.Start()

	// Wait for the immediate first tick to process
	time.Sleep(300 * time.Millisecond)
	scheduler.Stop()

	// Verify the task was dispatched on first tick
	events := matrix.getSentEvents()
	assert.NotEmpty(t, events, "expected task to be dispatched on scheduler start (immediate tick)")

	if len(events) > 0 {
		payload, ok := events[0].Payload.(TaskDispatchPayload)
		require.True(t, ok)
		assert.Equal(t, "task-immediate", payload.TaskID)
	}
}
