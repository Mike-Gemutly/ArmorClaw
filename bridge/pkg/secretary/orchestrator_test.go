package secretary

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//=============================================================================
// Mock Store
//=============================================================================

type orchestratorTestStore struct {
	mu        sync.RWMutex
	workflows map[string]*Workflow
	templates map[string]*TaskTemplate
	updateErr error
	getErr    error
}

func newOrchestratorTestStore() *orchestratorTestStore {
	return &orchestratorTestStore{
		workflows: make(map[string]*Workflow),
		templates: make(map[string]*TaskTemplate),
	}
}

func (s *orchestratorTestStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	w, ok := s.workflows[id]
	if !ok {
		return nil, errors.New("workflow not found")
	}
	return w, nil
}

func (s *orchestratorTestStore) UpdateWorkflow(ctx context.Context, w *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return s.updateErr
	}
	s.workflows[w.ID] = w
	return nil
}

func (s *orchestratorTestStore) CreateWorkflow(ctx context.Context, w *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID] = w
	return nil
}

func (s *orchestratorTestStore) DeleteWorkflow(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workflows, id)
	return nil
}

func (s *orchestratorTestStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Workflow
	for _, w := range s.workflows {
		if filter.Status != nil && w.Status != *filter.Status {
			continue
		}
		if filter.CreatedBy != "" && w.CreatedBy != filter.CreatedBy {
			continue
		}
		result = append(result, *w)
	}
	return result, nil
}

func (s *orchestratorTestStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, errors.New("template not found")
	}
	return t, nil
}

func (s *orchestratorTestStore) CreateTemplate(ctx context.Context, t *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[t.ID] = t
	return nil
}

func (s *orchestratorTestStore) UpdateTemplate(ctx context.Context, t *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[t.ID] = t
	return nil
}

func (s *orchestratorTestStore) DeleteTemplate(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.templates, id)
	return nil
}

func (s *orchestratorTestStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []TaskTemplate
	for _, t := range s.templates {
		if filter.ActiveOnly && !t.IsActive {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

// Stub implementations for unused Store interface methods
func (s *orchestratorTestStore) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *orchestratorTestStore) GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error) {
	return nil, errors.New("not implemented")
}
func (s *orchestratorTestStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	return nil, nil
}
func (s *orchestratorTestStore) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *orchestratorTestStore) DeletePolicy(ctx context.Context, id string) error {
	return nil
}
func (s *orchestratorTestStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	return nil
}
func (s *orchestratorTestStore) GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error) {
	return nil, errors.New("not implemented")
}
func (s *orchestratorTestStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}
func (s *orchestratorTestStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	return nil
}
func (s *orchestratorTestStore) DeleteScheduledTask(ctx context.Context, id string) error {
	return nil
}
func (s *orchestratorTestStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}
func (s *orchestratorTestStore) CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *orchestratorTestStore) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	return nil, errors.New("not implemented")
}
func (s *orchestratorTestStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	return nil, nil
}
func (s *orchestratorTestStore) UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *orchestratorTestStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	return nil
}
func (s *orchestratorTestStore) CreateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *orchestratorTestStore) GetContact(ctx context.Context, id string) (*Contact, error) {
	return nil, errors.New("not implemented")
}
func (s *orchestratorTestStore) ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error) {
	return nil, nil
}
func (s *orchestratorTestStore) UpdateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *orchestratorTestStore) DeleteContact(ctx context.Context, id string) error {
	return nil
}
func (s *orchestratorTestStore) Close() error {
	return nil
}

//=============================================================================
// Mock Event Emitter
//=============================================================================

type capturedEvent struct {
	eventType string
	workflow  *Workflow
	stepID    string
	stepName  string
	progress  float64
	err       error
	reason    string
	result    string
}

type mockEventEmitter struct {
	mu     sync.RWMutex
	events []capturedEvent
}

func newMockEventEmitter() *mockEventEmitter {
	return &mockEventEmitter{
		events: make([]capturedEvent, 0),
	}
}

func (e *mockEventEmitter) EmitStarted(workflow *Workflow) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedEvent{
		eventType: WorkflowEventStarted,
		workflow:  workflow,
	})
	return uint64(len(e.events))
}

func (e *mockEventEmitter) EmitProgress(workflow *Workflow, stepID, stepName string, progress float64) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedEvent{
		eventType: WorkflowEventProgress,
		workflow:  workflow,
		stepID:    stepID,
		stepName:  stepName,
		progress:  progress,
	})
	return uint64(len(e.events))
}

func (e *mockEventEmitter) EmitCompleted(workflow *Workflow, result string) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedEvent{
		eventType: WorkflowEventCompleted,
		workflow:  workflow,
		result:    result,
	})
	return uint64(len(e.events))
}

func (e *mockEventEmitter) EmitFailed(workflow *Workflow, stepID string, err error, recoverable bool) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedEvent{
		eventType: WorkflowEventFailed,
		workflow:  workflow,
		stepID:    stepID,
		err:       err,
	})
	return uint64(len(e.events))
}

func (e *mockEventEmitter) EmitCancelled(workflow *Workflow, reason string) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedEvent{
		eventType: WorkflowEventCancelled,
		workflow:  workflow,
		reason:    reason,
	})
	return uint64(len(e.events))
}

func (e *mockEventEmitter) getEvents() []capturedEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]capturedEvent, len(e.events))
	copy(result, e.events)
	return result
}

func (e *mockEventEmitter) clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = make([]capturedEvent, 0)
}

//=============================================================================
// Test Helpers
//=============================================================================

func setupTestOrchestrator(t *testing.T) (*WorkflowOrchestratorImpl, *orchestratorTestStore, *mockEventEmitter) {
	store := newOrchestratorTestStore()
	emitter := newMockEventEmitter()

	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		Factory:  nil,
		EventBus: emitter,
	})
	require.NoError(t, err)

	return orch, store, emitter
}

func createTestWorkflow(id, templateID string, status WorkflowStatus) *Workflow {
	return &Workflow{
		ID:         id,
		TemplateID: templateID,
		Name:       "Test Workflow",
		Status:     status,
		CreatedBy:  "@test:user.com",
		StartedAt:  time.Now(),
	}
}

func createTestTemplate(id string, steps []WorkflowStep) *TaskTemplate {
	return &TaskTemplate{
		ID:        id,
		Name:      "Test Template",
		Steps:     steps,
		CreatedBy: "@test:user.com",
		CreatedAt: time.Now(),
		IsActive:  true,
	}
}

//=============================================================================
// Orchestrator Tests
//=============================================================================

func TestOrchestrator_StartWorkflow_Success(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0},
	})
	store.templates["tpl-1"] = template

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	assert.Equal(t, StatusRunning, workflow.Status)
	assert.Equal(t, "step1", workflow.CurrentStep)
	assert.Equal(t, 1, orch.GetActiveWorkflowCount())

	events := emitter.getEvents()
	require.Len(t, events, 2)
	assert.Equal(t, WorkflowEventStarted, events[0].eventType)
	assert.Equal(t, WorkflowEventProgress, events[1].eventType)
	assert.Equal(t, "step1", events[1].stepID)
}

func TestOrchestrator_StartWorkflow_AlreadyRunning(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusRunning)
	store.workflows["wf-1"] = workflow

	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.StartWorkflow("wf-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestOrchestrator_StartWorkflow_NotFound(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.StartWorkflow("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow")
}

func TestOrchestrator_ValidLifecycleTransition(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0},
	})
	store.templates["tpl-1"] = template

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, workflow.Status)

	err = orch.CompleteWorkflow("wf-1", "done")
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, workflow.Status)
	assert.NotNil(t, workflow.CompletedAt)

	events := emitter.getEvents()
	assert.Equal(t, WorkflowEventCompleted, events[len(events)-1].eventType)
}

func TestOrchestrator_InvalidLifecycleTransition_FromCompleted(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	now := time.Now()
	workflow := &Workflow{
		ID:          "wf-1",
		Status:      StatusCompleted,
		CompletedAt: &now,
	}
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestOrchestrator_InvalidLifecycleTransition_FromFailed(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	now := time.Now()
	workflow := &Workflow{
		ID:          "wf-1",
		Status:      StatusFailed,
		CompletedAt: &now,
	}
	store.workflows["wf-1"] = workflow

	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.CompleteWorkflow("wf-1", "done")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestOrchestrator_CancelWorkflow(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	err = orch.CancelWorkflow("wf-1", "user requested")
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, workflow.Status)
	assert.Equal(t, "user requested", workflow.ErrorMessage)
	assert.NotNil(t, workflow.CompletedAt)
	assert.Equal(t, 0, orch.GetActiveWorkflowCount())

	events := emitter.getEvents()
	assert.Equal(t, WorkflowEventCancelled, events[len(events)-1].eventType)
	assert.Equal(t, "user requested", events[len(events)-1].reason)
}

func TestOrchestrator_FailWorkflow(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	testErr := errors.New("step failed")
	err = orch.FailWorkflow("wf-1", "step1", testErr, false)
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, workflow.Status)
	assert.Equal(t, "step failed", workflow.ErrorMessage)
	assert.NotNil(t, workflow.CompletedAt)

	events := emitter.getEvents()
	assert.Equal(t, WorkflowEventFailed, events[len(events)-1].eventType)
}

func TestOrchestrator_CompleteWorkflow(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	err = orch.CompleteWorkflow("wf-1", "all steps done")
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, workflow.Status)
	assert.NotNil(t, workflow.CompletedAt)
	assert.Equal(t, 0, orch.GetActiveWorkflowCount())

	events := emitter.getEvents()
	assert.Equal(t, WorkflowEventCompleted, events[len(events)-1].eventType)
	assert.Equal(t, "all steps done", events[len(events)-1].result)
}

func TestOrchestrator_ProgressEventEmission(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step One", Type: StepAction, Order: 0},
		{StepID: "step2", Name: "Step Two", Type: StepAction, Order: 1},
		{StepID: "step3", Name: "Step Three", Type: StepAction, Order: 2},
	})
	store.templates["tpl-1"] = template

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	err = orch.UpdateProgress("wf-1", "step2", 0.5)
	require.NoError(t, err)

	events := emitter.getEvents()
	var progressEvents []capturedEvent
	for _, e := range events {
		if e.eventType == WorkflowEventProgress {
			progressEvents = append(progressEvents, e)
		}
	}
	assert.GreaterOrEqual(t, len(progressEvents), 1)

	lastProgress := progressEvents[len(progressEvents)-1]
	assert.Equal(t, "step2", lastProgress.stepID)
	assert.Equal(t, "Step Two", lastProgress.stepName)
	assert.Equal(t, 0.5, lastProgress.progress)
}

//=============================================================================
// Dependency Validator Tests
//=============================================================================

func TestDependencyValidator_CircularDependencyRejection(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, NextStepID: "step3"},
		{StepID: "step3", Name: "Step 3", Type: StepAction, NextStepID: "step1"},
	})

	result := validator.ValidateTemplate(template)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrCircularDependency))
	assert.NotEmpty(t, result.Errors[0].CyclePath)
}

func TestDependencyValidator_NoCircularDependency(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, NextStepID: "step3"},
		{StepID: "step3", Name: "Step 3", Type: StepAction},
	})

	result := validator.ValidateTemplate(template)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDependencyValidator_TopologicalExecutionOrder(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, Order: 1, NextStepID: "step3"},
		{StepID: "step3", Name: "Step 3", Type: StepAction, Order: 2},
	})

	result := validator.ValidateTemplate(template)
	require.True(t, result.Valid)
	require.Len(t, result.ExecutionOrder, 3)

	assert.Equal(t, "step1", result.ExecutionOrder[0])
	assert.Equal(t, "step2", result.ExecutionOrder[1])
	assert.Equal(t, "step3", result.ExecutionOrder[2])
}

func TestDependencyValidator_MissingDependency(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, NextStepID: "nonexistent"},
	})

	result := validator.ValidateTemplate(template)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrMissingDependency))
	assert.Equal(t, "step1", result.Errors[0].StepID)
	assert.Equal(t, "nonexistent", result.Errors[0].Dependency)
}

func TestDependencyValidator_DuplicateStepID(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction},
		{StepID: "step1", Name: "Duplicate", Type: StepAction},
	})

	result := validator.ValidateTemplate(template)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrDuplicateStepID))
}

func TestDependencyValidator_EmptyStepID(t *testing.T) {
	validator := NewDependencyValidator()

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "", Name: "No ID", Type: StepAction},
	})

	result := validator.ValidateTemplate(template)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrEmptyStepID))
}

//=============================================================================
// Workflow Transition Validation Tests
//=============================================================================

func TestOrchestrator_ValidateTransition_PendingToRunning(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusPending, StatusRunning)
	assert.NoError(t, err)
}

func TestOrchestrator_ValidateTransition_PendingToCancelled(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusPending, StatusCancelled)
	assert.NoError(t, err)
}

func TestOrchestrator_ValidateTransition_RunningToCompleted(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusRunning, StatusCompleted)
	assert.NoError(t, err)
}

func TestOrchestrator_ValidateTransition_RunningToFailed(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusRunning, StatusFailed)
	assert.NoError(t, err)
}

func TestOrchestrator_ValidateTransition_RunningToCancelled(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusRunning, StatusCancelled)
	assert.NoError(t, err)
}

func TestOrchestrator_ValidateTransition_InvalidFromCompleted(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusCompleted, StatusRunning)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestOrchestrator_ValidateTransition_InvalidFromFailed(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusFailed, StatusRunning)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestOrchestrator_ValidateTransition_InvalidFromCancelled(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusCancelled, StatusRunning)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestOrchestrator_ValidateTransition_PendingToCompleted_Invalid(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	err := orch.validateTransition(StatusPending, StatusCompleted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

//=============================================================================
// GetWorkflow Tests
//=============================================================================

func TestOrchestrator_GetWorkflow_Active(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	retrieved, err := orch.GetWorkflow("wf-1")
	require.NoError(t, err)
	assert.Equal(t, "wf-1", retrieved.ID)
	assert.Equal(t, StatusRunning, retrieved.Status)
}

func TestOrchestrator_GetWorkflow_FromStore(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "", StatusCompleted)
	store.workflows["wf-1"] = workflow

	retrieved, err := orch.GetWorkflow("wf-1")
	require.NoError(t, err)
	assert.Equal(t, "wf-1", retrieved.ID)
	assert.Equal(t, StatusCompleted, retrieved.Status)
}

func TestOrchestrator_GetWorkflow_NotFound(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	_, err := orch.GetWorkflow("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

//=============================================================================
// AdvanceWorkflow Tests
//=============================================================================

func TestOrchestrator_AdvanceWorkflow_StepProgress(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, Order: 1, NextStepID: "step3"},
		{StepID: "step3", Name: "Step 3", Type: StepAction, Order: 2},
	})
	store.templates["tpl-1"] = template

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)
	assert.Equal(t, "step1", workflow.CurrentStep)

	err = orch.AdvanceWorkflow("wf-1", "step1")
	require.NoError(t, err)
	assert.Equal(t, "step2", workflow.CurrentStep)

	events := emitter.getEvents()
	progressEvents := 0
	for _, e := range events {
		if e.eventType == WorkflowEventProgress {
			progressEvents++
		}
	}
	assert.GreaterOrEqual(t, progressEvents, 2)
}

func TestOrchestrator_AdvanceWorkflow_CompletesOnLastStep(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, Order: 1},
	})
	store.templates["tpl-1"] = template

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	err = orch.AdvanceWorkflow("wf-1", "step1")
	require.NoError(t, err)

	err = orch.AdvanceWorkflow("wf-1", "step2")
	require.NoError(t, err)

	assert.Equal(t, StatusCompleted, workflow.Status)
	assert.Equal(t, 0, orch.GetActiveWorkflowCount())

	events := emitter.getEvents()
	assert.Equal(t, WorkflowEventCompleted, events[len(events)-1].eventType)
}

func TestOrchestrator_AdvanceWorkflow_WrongStep(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	template := createTestTemplate("tpl-1", []WorkflowStep{
		{StepID: "step1", Name: "Step 1", Type: StepAction, Order: 0, NextStepID: "step2"},
		{StepID: "step2", Name: "Step 2", Type: StepAction, Order: 1},
	})
	store.templates["tpl-1"] = template

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)

	err = orch.AdvanceWorkflow("wf-1", "wrong-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not on step")
}

//=============================================================================
// Shutdown Tests
//=============================================================================

func TestOrchestrator_Shutdown_CancelsActiveWorkflows(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow1 := createTestWorkflow("wf-1", "", StatusPending)
	workflow2 := createTestWorkflow("wf-2", "", StatusPending)
	store.workflows["wf-1"] = workflow1
	store.workflows["wf-2"] = workflow2

	err := orch.StartWorkflow("wf-1")
	require.NoError(t, err)
	err = orch.StartWorkflow("wf-2")
	require.NoError(t, err)

	assert.Equal(t, 2, orch.GetActiveWorkflowCount())

	orch.Shutdown()

	assert.Equal(t, 0, orch.GetActiveWorkflowCount())
	assert.Equal(t, StatusCancelled, workflow1.Status)
	assert.Equal(t, StatusCancelled, workflow2.Status)
	assert.Equal(t, "orchestrator shutdown", workflow1.ErrorMessage)

	events := emitter.getEvents()
	cancelledCount := 0
	for _, e := range events {
		if e.eventType == WorkflowEventCancelled {
			cancelledCount++
		}
	}
	assert.Equal(t, 2, cancelledCount)
}

//=============================================================================
// IsRunning Tests
//=============================================================================

func TestOrchestrator_IsRunning(t *testing.T) {
	orch, _, _ := setupTestOrchestrator(t)

	assert.True(t, orch.IsRunning())

	orch.Shutdown()

	assert.False(t, orch.IsRunning())
}
