package secretary

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlocked_Transitions(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusRunning)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.BlockWorkflow("wf-1", "approval_required", "waiting for user approval")
	require.NoError(t, err)
	assert.Equal(t, StatusBlocked, workflow.Status)
	assert.Equal(t, "waiting for user approval", workflow.ErrorMessage)

	events := emitter.getEvents()
	require.Len(t, events, 1)
	assert.Equal(t, WorkflowEventBlocked, events[0].eventType)
	assert.Equal(t, "approval_required", events[0].reason)
}

func TestBlocked_BlockedToRunning(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusBlocked)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.validateTransition(StatusBlocked, StatusRunning)
	assert.NoError(t, err)
}

func TestBlocked_BlockedToFailed(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusBlocked)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.validateTransition(StatusBlocked, StatusFailed)
	assert.NoError(t, err)
}

func TestBlocked_BlockedToCancelled(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusBlocked)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.validateTransition(StatusBlocked, StatusCancelled)
	assert.NoError(t, err)
}

func TestBlocked_BlockedToCompleted_Rejected(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusBlocked)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.validateTransition(StatusBlocked, StatusCompleted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestBlocked_RunningToBlocked_Accepted(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusRunning)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	err := orch.validateTransition(StatusRunning, StatusBlocked)
	assert.NoError(t, err)
}

func TestBlocked_BlockWorkflow_NotRunning_ReturnsError(t *testing.T) {
	orch, store, _ := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusPending)
	store.workflows["wf-1"] = workflow

	err := orch.BlockWorkflow("wf-1", "test", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not currently running")
}

func TestBlocked_BlockWorkflow_EmitsEvent(t *testing.T) {
	orch, store, emitter := setupTestOrchestrator(t)

	workflow := createTestWorkflow("wf-1", "tpl-1", StatusRunning)
	store.workflows["wf-1"] = workflow
	orch.activeWorkflows["wf-1"] = &activeWorkflow{
		workflow: workflow,
	}

	emitter.clear()

	err := orch.BlockWorkflow("wf-1", "payment_approval", "credit card needs approval")
	require.NoError(t, err)

	events := emitter.getEvents()
	require.Len(t, events, 1)
	assert.Equal(t, WorkflowEventBlocked, events[0].eventType)
	assert.Equal(t, "payment_approval", events[0].reason)
	assert.Equal(t, workflow, events[0].workflow)
}
