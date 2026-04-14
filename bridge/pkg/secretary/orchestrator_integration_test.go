package secretary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
