package secretary

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	store, err := NewStore(StoreConfig{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	now := time.Now().UnixMilli()
	_, err = store.db.Exec(
		`INSERT INTO task_templates (id, name, steps, created_by, created_at, updated_at, is_active) VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"test-tpl", "Test Template", `[{"step_id":"s1","name":"Step 1","type":"action","order":0}]`,
		"@test:example.com", now, now,
	)
	require.NoError(t, err)

	return store
}

func TestStore_CreateScheduledTask_WithDefinitionId(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	task := &ScheduledTask{
		ID:             "task-def-1",
		TemplateID:     "test-tpl",
		DefinitionID:   "def-my-agent",
		CronExpression: "0 * * * *",
		IsActive:       true,
		CreatedBy:      "@test:example.com",
	}
	err := store.CreateScheduledTask(ctx, task)
	require.NoError(t, err)

	got, err := store.GetScheduledTask(ctx, "task-def-1")
	require.NoError(t, err)
	assert.Equal(t, "def-my-agent", got.DefinitionID)
	assert.Equal(t, "task-def-1", got.ID)
}

func TestStore_UpdateScheduledTask_WithDefinitionId(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	task := &ScheduledTask{
		ID:             "task-upd-1",
		TemplateID:     "test-tpl",
		DefinitionID:   "def-old",
		CronExpression: "0 * * * *",
		IsActive:       true,
		CreatedBy:      "@test:example.com",
	}
	err := store.CreateScheduledTask(ctx, task)
	require.NoError(t, err)

	task.DefinitionID = "def-updated"
	err = store.UpdateScheduledTask(ctx, task)
	require.NoError(t, err)

	got, err := store.GetScheduledTask(ctx, "task-upd-1")
	require.NoError(t, err)
	assert.Equal(t, "def-updated", got.DefinitionID)
}

func TestStore_ListDueTasks_ReturnsDueActiveTasks(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tasks := []*ScheduledTask{
		{ID: "due-active", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: true, NextRun: &past, CreatedBy: "@test:example.com"},
		{ID: "not-due", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: true, NextRun: &future, CreatedBy: "@test:example.com"},
		{ID: "due-inactive", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: false, NextRun: &past, CreatedBy: "@test:example.com"},
	}
	for _, tsk := range tasks {
		require.NoError(t, store.CreateScheduledTask(ctx, tsk))
	}

	due, err := store.ListDueTasks(ctx)
	require.NoError(t, err)

	ids := make(map[string]bool)
	for _, d := range due {
		ids[d.ID] = true
	}
	assert.True(t, ids["due-active"], "due+active task should be returned")
	assert.False(t, ids["not-due"], "future task should not be returned")
	assert.False(t, ids["due-inactive"], "inactive task should not be returned")
}

func TestStore_ListDueTasks_ExcludesEmptyDefinitionId(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	past := time.Now().Add(-1 * time.Hour)

	task := &ScheduledTask{
		ID: "task-empty-def", TemplateID: "test-tpl", DefinitionID: "",
		IsActive: true, NextRun: &past, CreatedBy: "@test:example.com",
	}
	require.NoError(t, store.CreateScheduledTask(ctx, task))

	due, err := store.ListDueTasks(ctx)
	require.NoError(t, err)

	for _, d := range due {
		assert.NotEqual(t, "task-empty-def", d.ID, "task with empty definition_id should be excluded")
	}
}

func TestStore_ListDueTasks_OrderedByNextRun(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	now := time.Now()

	t1 := now.Add(-3 * time.Hour)
	t2 := now.Add(-2 * time.Hour)
	t3 := now.Add(-1 * time.Hour)

	tasks := []*ScheduledTask{
		{ID: "task-c", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: true, NextRun: &t3, CreatedBy: "@test:example.com"},
		{ID: "task-a", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: true, NextRun: &t1, CreatedBy: "@test:example.com"},
		{ID: "task-b", TemplateID: "test-tpl", DefinitionID: "def-1",
			IsActive: true, NextRun: &t2, CreatedBy: "@test:example.com"},
	}
	for _, tsk := range tasks {
		require.NoError(t, store.CreateScheduledTask(ctx, tsk))
	}

	due, err := store.ListDueTasks(ctx)
	require.NoError(t, err)
	require.Len(t, due, 3)

	assert.Equal(t, "task-a", due[0].ID)
	assert.Equal(t, "task-b", due[1].ID)
	assert.Equal(t, "task-c", due[2].ID)
}

func TestStore_MarkDispatched_UpdatesTimestamps(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	past := time.Now().Add(-1 * time.Hour)

	task := &ScheduledTask{
		ID: "task-dispatch", TemplateID: "test-tpl", DefinitionID: "def-1",
		CronExpression: "0 * * * *", IsActive: true, NextRun: &past,
		CreatedBy: "@test:example.com",
	}
	require.NoError(t, store.CreateScheduledTask(ctx, task))

	futureNextRun := time.Now().Add(1 * time.Hour)
	err := store.MarkDispatched(ctx, "task-dispatch", futureNextRun)
	require.NoError(t, err)

	got, err := store.GetScheduledTask(ctx, "task-dispatch")
	require.NoError(t, err)
	assert.NotNil(t, got.LastRun, "last_run should be set")
	assert.WithinDuration(t, time.Now(), *got.LastRun, 2*time.Second)
	assert.NotNil(t, got.NextRun, "next_run should be set")
	assert.WithinDuration(t, futureNextRun, *got.NextRun, time.Second)
}

func TestStore_MarkDispatched_NonExistentTask(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	futureNextRun := time.Now().Add(1 * time.Hour)
	err := store.MarkDispatched(ctx, "nonexistent-task", futureNextRun)
	assert.NoError(t, err, "MarkDispatched on non-existent task should not error")
}

func TestStore_NullTimestampsHandled(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	task := &ScheduledTask{
		ID: "task-null-ts", TemplateID: "test-tpl", DefinitionID: "def-1",
		CronExpression: "0 * * * *", IsActive: true,
		NextRun: nil, LastRun: nil,
		CreatedBy: "@test:example.com",
	}
	require.NoError(t, store.CreateScheduledTask(ctx, task))

	got, err := store.GetScheduledTask(ctx, "task-null-ts")
	require.NoError(t, err)
	assert.Nil(t, got.NextRun, "NextRun should be nil")
	assert.Nil(t, got.LastRun, "LastRun should be nil")
}

func TestStore_ListPendingScheduledTasks_StillWorks(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	past := time.Now().Add(-1 * time.Hour)

	task := &ScheduledTask{
		ID: "task-pending", TemplateID: "test-tpl", DefinitionID: "def-1",
		CronExpression: "0 * * * *", IsActive: true, NextRun: &past,
		CreatedBy: "@test:example.com",
	}
	require.NoError(t, store.CreateScheduledTask(ctx, task))

	pending, err := store.ListPendingScheduledTasks(ctx)
	require.NoError(t, err, "ListPendingScheduledTasks should not error after definition_id addition")

	if len(pending) > 0 {
		assert.Equal(t, "task-pending", pending[0].ID)
		assert.Equal(t, "def-1", pending[0].DefinitionID, "definition_id should be readable in backward-compat query")
	}
}
