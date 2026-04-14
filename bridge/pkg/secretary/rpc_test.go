package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//=============================================================================
// Mock Store for RPC Tests
//=============================================================================

type rpcTestStore struct {
	sync.RWMutex
	scheduledTasks map[string]*ScheduledTask
}

func newRPCTestStore() *rpcTestStore {
	return &rpcTestStore{
		scheduledTasks: make(map[string]*ScheduledTask),
	}
}

func (s *rpcTestStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.Lock()
	defer s.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *rpcTestStore) GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	t, ok := s.scheduledTasks[id]
	if !ok {
		return nil, fmt.Errorf("scheduled task not found: %s", id)
	}
	return t, nil
}

func (s *rpcTestStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.RLock()
	defer s.RUnlock()
	var result []ScheduledTask
	for _, t := range s.scheduledTasks {
		result = append(result, *t)
	}
	return result, nil
}

func (s *rpcTestStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.Lock()
	defer s.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *rpcTestStore) DeleteScheduledTask(ctx context.Context, id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.scheduledTasks, id)
	return nil
}

func (s *rpcTestStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}

func (s *rpcTestStore) ListDueTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}

func (s *rpcTestStore) MarkDispatched(ctx context.Context, taskID string, nextRun time.Time) error {
	return nil
}

func (s *rpcTestStore) CreateTemplate(ctx context.Context, template *TaskTemplate) error {
	return nil
}
func (s *rpcTestStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	return nil, errors.New("not implemented")
}
func (s *rpcTestStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	return nil, nil
}
func (s *rpcTestStore) UpdateTemplate(ctx context.Context, template *TaskTemplate) error {
	return nil
}
func (s *rpcTestStore) DeleteTemplate(ctx context.Context, id string) error {
	return nil
}
func (s *rpcTestStore) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}
func (s *rpcTestStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *rpcTestStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	return nil, nil
}
func (s *rpcTestStore) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}
func (s *rpcTestStore) DeleteWorkflow(ctx context.Context, id string) error {
	return nil
}
func (s *rpcTestStore) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *rpcTestStore) GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error) {
	return nil, errors.New("not implemented")
}
func (s *rpcTestStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	return nil, nil
}
func (s *rpcTestStore) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}
func (s *rpcTestStore) DeletePolicy(ctx context.Context, id string) error {
	return nil
}
func (s *rpcTestStore) CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *rpcTestStore) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	return nil, errors.New("not implemented")
}
func (s *rpcTestStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	return nil, nil
}
func (s *rpcTestStore) UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}
func (s *rpcTestStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	return nil
}
func (s *rpcTestStore) CreateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *rpcTestStore) GetContact(ctx context.Context, id string) (*Contact, error) {
	return nil, errors.New("not implemented")
}
func (s *rpcTestStore) ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error) {
	return nil, nil
}
func (s *rpcTestStore) UpdateContact(ctx context.Context, contact *Contact) error {
	return nil
}
func (s *rpcTestStore) DeleteContact(ctx context.Context, id string) error {
	return nil
}
func (s *rpcTestStore) Close() error { return nil }

//=============================================================================
// Helpers
//=============================================================================

func setupTestRPCHandler(t *testing.T) (*RPCHandler, *rpcTestStore) {
	store := newRPCTestStore()
	handler := NewRPCHandler(RPCHandlerConfig{
		Store: store,
	})
	return handler, store
}

func callRPC(h *RPCHandler, method string, params interface{}) *RPCResponse {
	paramsJSON, _ := json.Marshal(params)
	return h.Handle(&RPCRequest{
		Method: method,
		Params: paramsJSON,
		UserID: "@test:example.com",
	})
}

//=============================================================================
// task.create Tests
//=============================================================================

func TestHandleTaskCreate_Immediate(t *testing.T) {
	handler, store := setupTestRPCHandler(t)
	before := time.Now()

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"definition_id": "def-1",
		"created_by":    "@test:example.com",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	taskID, _ := result["task_id"].(string)
	assert.NotEmpty(t, taskID)

	nextRunVal, ok := result["next_run"].(int64)
	require.True(t, ok, "next_run should be int64")
	nextRun := time.UnixMilli(nextRunVal)
	assert.WithinDuration(t, before, nextRun, 2*time.Second)

	task, err := store.GetScheduledTask(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "def-1", task.DefinitionID)
	assert.True(t, task.IsActive)
}

func TestHandleTaskCreate_WithCronExpression(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"definition_id":   "def-1",
		"cron_expression": "0 * * * *",
		"created_by":      "@test:example.com",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	nextRunVal, ok := result["next_run"].(int64)
	require.True(t, ok, "next_run should be int64")
	nextRun := time.UnixMilli(nextRunVal)
	assert.True(t, nextRun.After(time.Now().Add(-time.Second)))
}

func TestHandleTaskCreate_WithRunAt(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)
	targetTime := "2026-12-25T00:00:00Z"

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"definition_id": "def-1",
		"run_at":        targetTime,
		"created_by":    "@test:example.com",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	nextRunVal, ok := result["next_run"].(int64)
	require.True(t, ok, "next_run should be int64")
	nextRun := time.UnixMilli(nextRunVal)
	expected, _ := time.Parse(time.RFC3339, targetTime)
	assert.WithinDuration(t, expected, nextRun, time.Second)
}

func TestHandleTaskCreate_MissingDefinitionId(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"created_by": "@test:example.com",
	})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrValidation, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "definition_id")
}

func TestHandleTaskCreate_InvalidCronExpression(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"definition_id":   "def-1",
		"cron_expression": "not-valid-cron",
		"created_by":      "@test:example.com",
	})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrValidation, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "cron")
}

func TestHandleTaskCreate_InvalidRunAtFormat(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.create", map[string]interface{}{
		"definition_id": "def-1",
		"run_at":        "not-a-date",
		"created_by":    "@test:example.com",
	})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrValidation, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "run_at")
}

//=============================================================================
// task.list Tests
//=============================================================================

func TestHandleTaskList_ReturnsAllTasks(t *testing.T) {
	handler, store := setupTestRPCHandler(t)
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 3; i++ {
		store.CreateScheduledTask(ctx, &ScheduledTask{
			ID:           fmt.Sprintf("task-%d", i),
			TemplateID:   "test-tpl",
			DefinitionID: fmt.Sprintf("def-%d", i),
			IsActive:     true,
			NextRun:      &now,
			CreatedBy:    "@test:example.com",
		})
	}

	resp := callRPC(handler, "task.list", map[string]interface{}{})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	tasksRaw, _ := result["tasks"].([]ScheduledTask)
	assert.Len(t, tasksRaw, 3)
}

func TestHandleTaskList_FilterByDefinitionId(t *testing.T) {
	handler, store := setupTestRPCHandler(t)
	ctx := context.Background()
	now := time.Now()

	store.CreateScheduledTask(ctx, &ScheduledTask{
		ID: "task-a", TemplateID: "test-tpl", DefinitionID: "def-alpha",
		IsActive: true, NextRun: &now, CreatedBy: "@test:example.com",
	})
	store.CreateScheduledTask(ctx, &ScheduledTask{
		ID: "task-b", TemplateID: "test-tpl", DefinitionID: "def-beta",
		IsActive: true, NextRun: &now, CreatedBy: "@test:example.com",
	})

	resp := callRPC(handler, "task.list", map[string]interface{}{
		"definition_id": "def-alpha",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	tasksRaw, _ := result["tasks"].([]ScheduledTask)
	require.Len(t, tasksRaw, 1)
	assert.Equal(t, "def-alpha", tasksRaw[0].DefinitionID)
}

//=============================================================================
// task.cancel Tests
//=============================================================================

func TestHandleTaskCancel_DeactivatesTask(t *testing.T) {
	handler, store := setupTestRPCHandler(t)
	ctx := context.Background()
	now := time.Now()

	store.CreateScheduledTask(ctx, &ScheduledTask{
		ID: "task-cancel-me", TemplateID: "test-tpl", DefinitionID: "def-1",
		IsActive: true, NextRun: &now, CreatedBy: "@test:example.com",
	})

	resp := callRPC(handler, "task.cancel", map[string]interface{}{
		"task_id": "task-cancel-me",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)

	updated, _ := store.GetScheduledTask(ctx, "task-cancel-me")
	assert.False(t, updated.IsActive)
}

func TestHandleTaskCancel_MissingTaskId(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.cancel", map[string]interface{}{})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrValidation, resp.Error.Code)
}

func TestHandleTaskCancel_TaskNotFound(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.cancel", map[string]interface{}{
		"task_id": "nonexistent",
	})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrNotFound, resp.Error.Code)
}

//=============================================================================
// task.get Tests
//=============================================================================

func TestHandleTaskGet_ReturnsTask(t *testing.T) {
	handler, store := setupTestRPCHandler(t)
	ctx := context.Background()
	now := time.Now()

	store.CreateScheduledTask(ctx, &ScheduledTask{
		ID: "task-get-me", TemplateID: "test-tpl", DefinitionID: "def-1",
		CronExpression: "0 * * * *", IsActive: true, NextRun: &now,
		CreatedBy: "@test:example.com",
	})

	resp := callRPC(handler, "task.get", map[string]interface{}{
		"task_id": "task-get-me",
	})

	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	task, ok := resp.Result.(*ScheduledTask)
	require.True(t, ok)

	assert.Equal(t, "task-get-me", task.ID)
	assert.Equal(t, "def-1", task.DefinitionID)
	assert.Equal(t, "0 * * * *", task.CronExpression)
	assert.True(t, task.IsActive)
}

func TestHandleTaskGet_MissingTaskId(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.get", map[string]interface{}{})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrValidation, resp.Error.Code)
}

func TestHandleTaskGet_TaskNotFound(t *testing.T) {
	handler, _ := setupTestRPCHandler(t)

	resp := callRPC(handler, "task.get", map[string]interface{}{
		"task_id": "nonexistent",
	})

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrNotFound, resp.Error.Code)
}
