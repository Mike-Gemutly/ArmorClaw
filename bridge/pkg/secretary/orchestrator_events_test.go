package secretary

import (
	"testing"

	"github.com/armorclaw/bridge/internal/events"
	"github.com/stretchr/testify/assert"
)

func TestEmitStepProgressPublished(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	emitter := &WorkflowEventEmitter{bus: bus, sender: "test"}
	sub := bus.Subscribe()

	evt := StepEvent{
		Seq:    1,
		Type:   "progress",
		Name:   "navigate",
		TsMs:   1000,
		Detail: map[string]interface{}{"percent": float64(50)},
	}

	emitter.EmitStepProgress("!room:test", evt)

	published := <-sub
	assert.Equal(t, WorkflowEventStepProgress, published.Type)
	assert.Equal(t, "!room:test", published.RoomID)
	assert.Equal(t, "test", published.Sender)

	content, ok := published.Content.(WorkflowEvent)
	requireTrue(t, ok)
	assert.Equal(t, float64(50), content.Progress)
	assert.Equal(t, "navigate", content.StepName)
	assert.Equal(t, StatusRunning, content.Status)
	assert.Equal(t, int64(1000), content.Timestamp)

	meta, ok := content.Metadata["progress_detail"].(ProgressDetail)
	requireTrue(t, ok)
	assert.Equal(t, 1, meta.EventSeq)
	assert.Equal(t, "progress", meta.EventType)
}

func TestEmitStepErrorPublished(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	emitter := &WorkflowEventEmitter{bus: bus, sender: "test"}
	sub := bus.Subscribe()

	evt := StepEvent{
		Seq:    2,
		Type:   "error",
		Name:   "timeout",
		TsMs:   2000,
		Detail: map[string]interface{}{"code": "ETIMEDOUT"},
	}

	emitter.EmitStepError("!room:test", evt)

	published := <-sub
	assert.Equal(t, WorkflowEventStepError, published.Type)

	content, ok := published.Content.(WorkflowEvent)
	requireTrue(t, ok)
	assert.Equal(t, "timeout", content.Error)
	assert.Equal(t, StatusFailed, content.Status)
	assert.Equal(t, int64(2000), content.Timestamp)
	assert.Equal(t, 2, content.Metadata["event_seq"])
	assert.Equal(t, "error", content.Metadata["event_type"])
}

func TestEmitBlockerWarningPublished(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	emitter := &WorkflowEventEmitter{bus: bus, sender: "test"}
	sub := bus.Subscribe()

	evt := StepEvent{
		Seq:  3,
		Type: "blocker",
		Name: "auth_required",
		TsMs: 3000,
		Detail: map[string]interface{}{
			"blocker_type": "auth",
			"message":      "Login required to continue",
		},
	}

	emitter.EmitBlockerWarning("!room:test", evt)

	published := <-sub
	assert.Equal(t, WorkflowEventBlockerWarning, published.Type)

	content, ok := published.Content.(WorkflowEvent)
	requireTrue(t, ok)
	assert.Equal(t, StatusBlocked, content.Status)
	assert.Equal(t, "auth", content.Metadata["blocker_type"])
	assert.Equal(t, "Login required to continue", content.Metadata["message"])
	assert.Equal(t, 3, content.Metadata["event_seq"])
}

func requireTrue(t *testing.T, ok bool) {
	if !ok {
		t.Fatal("assertion failed: expected true")
	}
}
