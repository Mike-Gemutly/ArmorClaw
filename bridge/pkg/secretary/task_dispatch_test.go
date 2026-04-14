package secretary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildTaskDispatchPayload_AllFieldsPopulated(t *testing.T) {
	now := time.Now()
	task := ScheduledTask{
		ID:             "task-123",
		DefinitionID:   "def-1",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@test:example.com",
	}

	payload := BuildTaskDispatchPayload(task, "hourly check")

	assert.Equal(t, "task-123", payload.TaskID)
	assert.Equal(t, "hourly check", payload.Description)
	assert.Equal(t, "scheduler", payload.Source)
	assert.WithinDuration(t, time.Now(), time.UnixMilli(payload.DispatchedAt), 2*time.Second)
}

func TestBuildTaskDispatchPayload_ZeroValueTask(t *testing.T) {
	task := ScheduledTask{}

	payload := BuildTaskDispatchPayload(task, "")

	assert.Equal(t, "", payload.TaskID)
	assert.Equal(t, "", payload.Description)
	assert.Equal(t, "scheduler", payload.Source)
	assert.NotZero(t, payload.DispatchedAt)
}

func TestBuildTaskDispatchPayload_WithCronExpression(t *testing.T) {
	task := ScheduledTask{
		ID:             "task-cron",
		CronExpression: "*/5 * * * *",
	}

	payload := BuildTaskDispatchPayload(task, task.CronExpression)

	assert.Equal(t, "task-cron", payload.TaskID)
	assert.Equal(t, "*/5 * * * *", payload.Description)
	assert.Equal(t, "scheduler", payload.Source)
	assert.NotZero(t, payload.DispatchedAt)
}
