package secretary

import (
	"fmt"
	"time"

	"github.com/armorclaw/bridge/internal/events"
)

//=============================================================================
// Workflow Event Types
//=============================================================================

const (
	WorkflowEventStarted   = "workflow.started"
	WorkflowEventProgress  = "workflow.progress"
	WorkflowEventBlocked   = "workflow.blocked"
	WorkflowEventCompleted = "workflow.completed"
	WorkflowEventFailed    = "workflow.failed"
	WorkflowEventCancelled = "workflow.cancelled"

	WorkflowEventStepProgress   = "workflow.step_progress"
	WorkflowEventStepError      = "workflow.step_error"
	WorkflowEventBlockerWarning = "workflow.blocker_warning"
)

//=============================================================================
// Workflow Event Types
//=============================================================================

type WorkflowEvent struct {
	WorkflowID  string                 `json:"workflow_id"`
	TemplateID  string                 `json:"template_id,omitempty"`
	Status      WorkflowStatus         `json:"status"`
	StepID      string                 `json:"step_id,omitempty"`
	StepName    string                 `json:"step_name,omitempty"`
	Progress    float64                `json:"progress,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
	Error       string                 `json:"error,omitempty"`
	Recoverable bool                   `json:"recoverable,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

//=============================================================================
// Event Emitter
//=============================================================================

type EventEmitter interface {
	EmitStarted(workflow *Workflow) uint64
	EmitProgress(workflow *Workflow, stepID, stepName string, progress float64) uint64
	EmitBlocked(workflow *Workflow, reason, message string) uint64
	EmitCompleted(workflow *Workflow, result string) uint64
	EmitFailed(workflow *Workflow, stepID string, err error, recoverable bool) uint64
	EmitCancelled(workflow *Workflow, reason string) uint64
}

type WorkflowEventEmitter struct {
	bus    *events.MatrixEventBus
	sender string
}

func NewWorkflowEventEmitter(bus *events.MatrixEventBus) *WorkflowEventEmitter {
	return &WorkflowEventEmitter{
		bus:    bus,
		sender: "orchestrator",
	}
}

func (e *WorkflowEventEmitter) EmitStarted(workflow *Workflow) uint64 {
	event := WorkflowEvent{
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Status:     StatusRunning,
		Timestamp:  workflow.StartedAt.UnixMilli(),
		Metadata: map[string]interface{}{
			"name":        workflow.Name,
			"description": workflow.Description,
			"created_by":  workflow.CreatedBy,
			"agent_ids":   workflow.AgentIDs,
		},
	}

	return e.publish(workflow.CreatedBy, WorkflowEventStarted, event)
}

func (e *WorkflowEventEmitter) EmitProgress(workflow *Workflow, stepID, stepName string, progress float64) uint64 {
	event := WorkflowEvent{
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Status:     StatusRunning,
		StepID:     stepID,
		StepName:   stepName,
		Progress:   progress,
		Timestamp:  time.Now().UnixMilli(),
	}

	return e.publish(workflow.CreatedBy, WorkflowEventProgress, event)
}

func (e *WorkflowEventEmitter) EmitCompleted(workflow *Workflow, result string) uint64 {
	var duration int64
	if workflow.CompletedAt != nil {
		duration = workflow.CompletedAt.Sub(workflow.StartedAt).Milliseconds()
	}

	event := WorkflowEvent{
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Status:     StatusCompleted,
		Result:     result,
		Duration:   duration,
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"name": workflow.Name,
		},
	}

	return e.publish(workflow.CreatedBy, WorkflowEventCompleted, event)
}

func (e *WorkflowEventEmitter) EmitFailed(workflow *Workflow, stepID string, err error, recoverable bool) uint64 {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	event := WorkflowEvent{
		WorkflowID:  workflow.ID,
		TemplateID:  workflow.TemplateID,
		Status:      StatusFailed,
		StepID:      stepID,
		Error:       errorMsg,
		Recoverable: recoverable,
		Timestamp:   time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"name": workflow.Name,
		},
	}

	return e.publish(workflow.CreatedBy, WorkflowEventFailed, event)
}

func (e *WorkflowEventEmitter) EmitCancelled(workflow *Workflow, reason string) uint64 {
	var duration int64
	if workflow.CompletedAt != nil {
		duration = workflow.CompletedAt.Sub(workflow.StartedAt).Milliseconds()
	}

	event := WorkflowEvent{
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Status:     StatusCancelled,
		Reason:     reason,
		Duration:   duration,
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"name": workflow.Name,
		},
	}

	return e.publish(workflow.CreatedBy, WorkflowEventCancelled, event)
}

func (e *WorkflowEventEmitter) EmitBlocked(workflow *Workflow, reason, message string) uint64 {
	event := WorkflowEvent{
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Status:     StatusBlocked,
		Reason:     reason,
		Result:     message,
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"name":        workflow.Name,
			"currentStep": workflow.CurrentStep,
		},
	}

	return e.publish(workflow.CreatedBy, WorkflowEventBlocked, event)
}

func (e *WorkflowEventEmitter) publish(roomID, eventType string, event WorkflowEvent) uint64 {
	return e.bus.Publish(events.MatrixEvent{
		ID:      fmt.Sprintf("%s-%s-%d", eventType, event.WorkflowID, time.Now().UnixNano()),
		RoomID:  roomID,
		Sender:  e.sender,
		Type:    eventType,
		Content: event,
	})
}

//=============================================================================
// Event Builder
//=============================================================================

type WorkflowEventBuilder struct {
	event WorkflowEvent
}

func NewWorkflowEventBuilder(workflowID string) *WorkflowEventBuilder {
	return &WorkflowEventBuilder{
		event: WorkflowEvent{
			WorkflowID: workflowID,
			Timestamp:  time.Now().UnixMilli(),
			Metadata:   make(map[string]interface{}),
		},
	}
}

func (b *WorkflowEventBuilder) WithTemplateID(templateID string) *WorkflowEventBuilder {
	b.event.TemplateID = templateID
	return b
}

func (b *WorkflowEventBuilder) WithStatus(status WorkflowStatus) *WorkflowEventBuilder {
	b.event.Status = status
	return b
}

func (b *WorkflowEventBuilder) WithStep(stepID, stepName string) *WorkflowEventBuilder {
	b.event.StepID = stepID
	b.event.StepName = stepName
	return b
}

func (b *WorkflowEventBuilder) WithProgress(progress float64) *WorkflowEventBuilder {
	b.event.Progress = progress
	return b
}

func (b *WorkflowEventBuilder) WithError(err error, recoverable bool) *WorkflowEventBuilder {
	if err != nil {
		b.event.Error = err.Error()
	}
	b.event.Recoverable = recoverable
	return b
}

func (b *WorkflowEventBuilder) WithReason(reason string) *WorkflowEventBuilder {
	b.event.Reason = reason
	return b
}

func (b *WorkflowEventBuilder) WithResult(result string) *WorkflowEventBuilder {
	b.event.Result = result
	return b
}

func (b *WorkflowEventBuilder) WithDuration(duration time.Duration) *WorkflowEventBuilder {
	b.event.Duration = duration.Milliseconds()
	return b
}

func (b *WorkflowEventBuilder) WithMetadata(key string, value interface{}) *WorkflowEventBuilder {
	b.event.Metadata[key] = value
	return b
}

func (b *WorkflowEventBuilder) Build() WorkflowEvent {
	return b.event
}

//=============================================================================
// Helper Functions
//=============================================================================

func EmitWorkflowStarted(bus *events.MatrixEventBus, workflow *Workflow) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitStarted(workflow)
}

func EmitWorkflowProgress(bus *events.MatrixEventBus, workflow *Workflow, stepID, stepName string, progress float64) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitProgress(workflow, stepID, stepName, progress)
}

func EmitWorkflowCompleted(bus *events.MatrixEventBus, workflow *Workflow, result string) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitCompleted(workflow, result)
}

func EmitWorkflowFailed(bus *events.MatrixEventBus, workflow *Workflow, stepID string, err error, recoverable bool) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitFailed(workflow, stepID, err, recoverable)
}

func EmitWorkflowCancelled(bus *events.MatrixEventBus, workflow *Workflow, reason string) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitCancelled(workflow, reason)
}

//=============================================================================
// Step Event Methods (from container _events.jsonl)
//=============================================================================

// ProgressDetail carries structured metadata for step progress events.
type ProgressDetail struct {
	EventSeq  int                    `json:"event_seq"`
	EventType string                 `json:"event_type"`
	StepName  string                 `json:"step_name"`
	ElapsedMs int64                  `json:"elapsed_ms"`
	Detail    map[string]interface{} `json:"detail,omitempty"`
}

// EmitStepProgress publishes a WorkflowEventStepProgress event derived from a StepEvent.
// It extracts progress percent from event.Detail["percent"] when present.
func (e *WorkflowEventEmitter) EmitStepProgress(roomID string, event StepEvent) uint64 {
	var progress float64
	if v, ok := event.Detail["percent"]; ok {
		if f, ok := v.(float64); ok {
			progress = f
		}
	}

	var elapsed int64
	if event.DurationMs != nil {
		elapsed = int64(*event.DurationMs)
	}

	wfEvent := WorkflowEvent{
		Progress:  progress,
		StepName:  event.Name,
		Timestamp: event.TsMs,
		Status:    StatusRunning,
		Metadata: map[string]interface{}{
			"progress_detail": ProgressDetail{
				EventSeq:  event.Seq,
				EventType: event.Type,
				StepName:  event.Name,
				ElapsedMs: elapsed,
				Detail:    event.Detail,
			},
		},
	}

	return e.publish(roomID, WorkflowEventStepProgress, wfEvent)
}

// EmitStepError publishes a WorkflowEventStepError event derived from a StepEvent.
func (e *WorkflowEventEmitter) EmitStepError(roomID string, event StepEvent) uint64 {
	wfEvent := WorkflowEvent{
		Error:     event.Name,
		Timestamp: event.TsMs,
		Status:    StatusFailed,
		Metadata: map[string]interface{}{
			"event_seq":  event.Seq,
			"event_type": event.Type,
			"detail":     event.Detail,
		},
	}

	return e.publish(roomID, WorkflowEventStepError, wfEvent)
}

// EmitBlockerWarning publishes a WorkflowEventBlockerWarning event derived from a StepEvent.
func (e *WorkflowEventEmitter) EmitBlockerWarning(roomID string, event StepEvent) uint64 {
	wfEvent := WorkflowEvent{
		Status:    StatusBlocked,
		Timestamp: event.TsMs,
		Metadata: map[string]interface{}{
			"blocker_type": event.Detail["blocker_type"],
			"message":      event.Detail["message"],
			"event_seq":    event.Seq,
			"event_type":   event.Type,
		},
	}

	return e.publish(roomID, WorkflowEventBlockerWarning, wfEvent)
}

//=============================================================================
// Standalone Step Event Helpers
//=============================================================================

// EmitStepProgressEvent is a standalone helper that creates an emitter and
// publishes a step progress event in one call.
func EmitStepProgressEvent(bus *events.MatrixEventBus, roomID string, event StepEvent) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitStepProgress(roomID, event)
}

// EmitStepErrorEvent is a standalone helper that creates an emitter and
// publishes a step error event in one call.
func EmitStepErrorEvent(bus *events.MatrixEventBus, roomID string, event StepEvent) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitStepError(roomID, event)
}

// EmitBlockerWarningEvent is a standalone helper that creates an emitter and
// publishes a blocker warning event in one call.
func EmitBlockerWarningEvent(bus *events.MatrixEventBus, roomID string, event StepEvent) uint64 {
	emitter := NewWorkflowEventEmitter(bus)
	return emitter.EmitBlockerWarning(roomID, event)
}
