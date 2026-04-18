package secretary

import (
	"context"
	"testing"
	"time"

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
		Name: "Login required to continue",
		TsMs: 3000,
		Detail: map[string]interface{}{
			"blocker_type": "auth",
			"suggestion":   "Provide credentials",
			"field":        "password",
		},
	}

	emitter.EmitBlockerWarning("!room:test", evt, "wf-42", "step-1")

	published := <-sub
	assert.Equal(t, WorkflowEventBlockerWarning, published.Type)

	content, ok := published.Content.(WorkflowEvent)
	requireTrue(t, ok)
	assert.Equal(t, StatusBlocked, content.Status)
	assert.Equal(t, "auth", content.Metadata["blocker_type"])
	assert.Equal(t, "Login required to continue", content.Metadata["message"])
	assert.Equal(t, "Provide credentials", content.Metadata["suggestion"])
	assert.Equal(t, "password", content.Metadata["field"])
	assert.Equal(t, "wf-42", content.Metadata["workflow_id"])
	assert.Equal(t, "step-1", content.Metadata["step_id"])
	assert.Equal(t, 3, content.Metadata["event_seq"])
}

func requireTrue(t *testing.T, ok bool) {
	if !ok {
		t.Fatal("assertion failed: expected true")
	}
}

//=============================================================================
// MatrixEventForwarder Tests
//=============================================================================

func TestMatrixForwardStepProgress(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	var sentMessages []struct {
		roomID  string
		message string
	}
	sendFunc := func(_ context.Context, roomID, message string) error {
		sentMessages = append(sentMessages, struct {
			roomID  string
			message string
		}{roomID, message})
		return nil
	}

	forwarder := NewMatrixEventForwarder(bus, sendFunc)
	forwarder.Start()
	defer forwarder.Stop()

	bus.Publish(events.MatrixEvent{
		ID:      "test-progress-1",
		RoomID:  "!room:test",
		Sender:  "orchestrator",
		Type:    WorkflowEventStepProgress,
		Content: WorkflowEvent{StepName: "navigate", Progress: 0.5, Status: StatusRunning},
	})

	assert.Eventually(t, func() bool {
		return len(sentMessages) == 1
	}, time.Second, 10*time.Millisecond, "expected 1 forwarded message")

	assert.Equal(t, "!room:test", sentMessages[0].roomID)
	assert.Equal(t, "🔹 Step: navigate (50%)", sentMessages[0].message)
}

func TestMatrixForwardStepError(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	var sentMessages []struct {
		roomID  string
		message string
	}
	sendFunc := func(_ context.Context, roomID, message string) error {
		sentMessages = append(sentMessages, struct {
			roomID  string
			message string
		}{roomID, message})
		return nil
	}

	forwarder := NewMatrixEventForwarder(bus, sendFunc)
	forwarder.Start()
	defer forwarder.Stop()

	bus.Publish(events.MatrixEvent{
		ID:      "test-error-1",
		RoomID:  "!room:test",
		Sender:  "orchestrator",
		Type:    WorkflowEventStepError,
		Content: WorkflowEvent{Error: "connection refused", Status: StatusFailed},
	})

	assert.Eventually(t, func() bool {
		return len(sentMessages) == 1
	}, time.Second, 10*time.Millisecond, "expected 1 forwarded message")

	assert.Equal(t, "!room:test", sentMessages[0].roomID)
	assert.Equal(t, "❌ Error: connection refused", sentMessages[0].message)
}

func TestMatrixForwardBlockerWarning(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	var sentMessages []struct {
		roomID  string
		message string
	}
	sendFunc := func(_ context.Context, roomID, message string) error {
		sentMessages = append(sentMessages, struct {
			roomID  string
			message string
		}{roomID, message})
		return nil
	}

	forwarder := NewMatrixEventForwarder(bus, sendFunc)
	forwarder.Start()
	defer forwarder.Stop()

	bus.Publish(events.MatrixEvent{
		ID:     "test-blocker-1",
		RoomID: "!room:test",
		Sender: "orchestrator",
		Type:   WorkflowEventBlockerWarning,
		Content: WorkflowEvent{
			Status: StatusBlocked,
			Metadata: map[string]interface{}{
				"blocker_type": "auth",
				"message":      "Login required",
			},
		},
	})

	assert.Eventually(t, func() bool {
		return len(sentMessages) == 1
	}, time.Second, 10*time.Millisecond, "expected 1 forwarded blocker message")

	assert.Equal(t, "!room:test", sentMessages[0].roomID)
	assert.Equal(t, "⚠️ Blocker: Login required", sentMessages[0].message)
}

func TestMatrixForwardIgnoresOtherEvents(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	sentCount := 0
	sendFunc := func(_ context.Context, _, _ string) error {
		sentCount++
		return nil
	}

	forwarder := NewMatrixEventForwarder(bus, sendFunc)
	forwarder.Start()
	defer forwarder.Stop()

	bus.Publish(events.MatrixEvent{
		ID:      "other-1",
		RoomID:  "!room:test",
		Sender:  "orchestrator",
		Type:    WorkflowEventStarted,
		Content: WorkflowEvent{Status: StatusRunning},
	})
	bus.Publish(events.MatrixEvent{
		ID:      "other-2",
		RoomID:  "!room:test",
		Sender:  "orchestrator",
		Type:    WorkflowEventCompleted,
		Content: WorkflowEvent{Status: StatusCompleted},
	})

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, sentCount, "should not forward non-step events")
}

func TestMatrixForwardSkipsEmptyRoom(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	sentCount := 0
	sendFunc := func(_ context.Context, _, _ string) error {
		sentCount++
		return nil
	}

	forwarder := NewMatrixEventForwarder(bus, sendFunc)
	forwarder.Start()
	defer forwarder.Stop()

	bus.Publish(events.MatrixEvent{
		ID:      "no-room",
		RoomID:  "",
		Sender:  "orchestrator",
		Type:    WorkflowEventStepProgress,
		Content: WorkflowEvent{StepName: "navigate", Progress: 0.75, Status: StatusRunning},
	})

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, sentCount, "should not forward when RoomID is empty")
}

func TestMatrixForwardStop(t *testing.T) {
	bus := events.NewMatrixEventBus(64)

	forwarder := NewMatrixEventForwarder(bus, func(_ context.Context, _, _ string) error { return nil })
	forwarder.Start()
	forwarder.Stop()

	assert.NotPanics(t, func() {
		forwarder.Stop()
	})
}
