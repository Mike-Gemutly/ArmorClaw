package secretary

import "time"

//=============================================================================
// Task Dispatch - Payload and Event Type
//=============================================================================

// EventTypeTaskDispatch is the Matrix event type for task dispatch directives
const EventTypeTaskDispatch = "app.armorclaw.task_dispatch"

// TaskDispatchPayload is the payload sent to agents via Matrix room injection
// when a scheduled task is dispatched.
type TaskDispatchPayload struct {
	TaskID       string `json:"task_id"`
	Description  string `json:"description"`
	DispatchedAt int64  `json:"dispatched_at"`
	Source       string `json:"source"`
	WorkflowID   string `json:"workflow_id,omitempty"`
}

// BuildTaskDispatchPayload constructs a dispatch payload from a scheduled task
func BuildTaskDispatchPayload(task ScheduledTask, description string) TaskDispatchPayload {
	return TaskDispatchPayload{
		TaskID:       task.ID,
		Description:  description,
		DispatchedAt: time.Now().UnixMilli(),
		Source:       "scheduler",
	}
}
