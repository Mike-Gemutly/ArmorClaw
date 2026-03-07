package agent

import (
	"context"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type StepType string

const (
	StepTypeReason    StepType = "reason"
	StepTypeToolCall  StepType = "tool_call"
	StepTypeToolResult StepType = "tool_result"
	StepTypeFinal     StepType = "final"
)

type Task struct {
	ID           string            `json:"id"`
	RoomID       string            `json:"room_id"`
	UserID       string            `json:"user_id"`
	Conversation []Message         `json:"conversation"`
	Steps        []Step            `json:"steps"`
	Status       TaskStatus        `json:"status"`
	Result       *Result           `json:"result,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ctx          context.Context   `json:"-"`
}

type Step struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	Type        StepType               `json:"type"`
	Thought     string                 `json:"thought,omitempty"`
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolInput   map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput  string                 `json:"tool_output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	CreatedAt   time.Time              `json:"created_at"`
	IsSpeculative bool                 `json:"is_speculative,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ContentPart struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *ImageURL   `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Result struct {
	TaskID       string    `json:"task_id"`
	Response     string    `json:"response"`
	ToolCalls    int       `json:"tool_calls"`
	TokensUsed   TokenUsage `json:"tokens_used"`
	Duration     time.Duration `json:"duration"`
	Steps        int       `json:"steps"`
	CompletedAt  time.Time `json:"completed_at"`
}

type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
	Total      int `json:"total"`
}

func (t *Task) Context() context.Context {
	if t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

func (t *Task) WithContext(ctx context.Context) *Task {
	t.ctx = ctx
	return t
}

func (t *Task) AddStep(step Step) {
	t.Steps = append(t.Steps, step)
	t.UpdatedAt = time.Now()
}

func (t *Task) LastStep() *Step {
	if len(t.Steps) == 0 {
		return nil
	}
	return &t.Steps[len(t.Steps)-1]
}

func (t *Task) TokenUsage() TokenUsage {
	var total TokenUsage
	for range t.Steps {
	}
	return total
}

func NewTask(id, roomID, userID string) *Task {
	now := time.Now()
	return &Task{
		ID:           id,
		RoomID:       roomID,
		UserID:       userID,
		Conversation: make([]Message, 0),
		Steps:        make([]Step, 0),
		Status:       TaskStatusPending,
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func NewStep(taskID string, stepType StepType) Step {
	return Step{
		ID:        generateStepID(),
		TaskID:    taskID,
		Type:      stepType,
		CreatedAt: time.Now(),
	}
}

func generateStepID() string {
	return time.Now().Format("20060102150405") + "-" + randomSuffix()
}

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	return string(b)
}
