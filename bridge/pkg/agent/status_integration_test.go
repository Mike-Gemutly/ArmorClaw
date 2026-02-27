package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockStatusEmitter captures emitted status events for testing
type mockStatusEmitter struct {
	mu     sync.Mutex
	events []StatusEvent
}

func (m *mockStatusEmitter) EmitStatus(ctx context.Context, event StatusEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockStatusEmitter) getEvents() []StatusEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]StatusEvent{}, m.events...)
}

// TestBlindFillStatusEmitter_ResolvingPII tests PII resolution status emission
func TestBlindFillStatusEmitter_ResolvingPII(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBlindFillStatusEmitter("agent-001", mock)

	err := emitter.EmitResolvingPII(context.Background(), "flight_booking", 5)
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.AgentID != "agent-001" {
		t.Errorf("expected agent_id agent-001, got %s", event.AgentID)
	}
	if event.Status != StatusFormFilling {
		t.Errorf("expected status %s, got %s", StatusFormFilling, event.Status)
	}
	if event.Metadata.Step != "resolving_pii" {
		t.Errorf("expected step resolving_pii, got %s", event.Metadata.Step)
	}
	if event.Metadata.TaskType != "flight_booking" {
		t.Errorf("expected task_type flight_booking, got %s", event.Metadata.TaskType)
	}
	if event.Metadata.Progress != 10 {
		t.Errorf("expected progress 10, got %d", event.Metadata.Progress)
	}
}

// TestBlindFillStatusEmitter_PIIResolved tests PII resolved status emission
func TestBlindFillStatusEmitter_PIIResolved(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBlindFillStatusEmitter("agent-002", mock)

	// Test with granted fields
	err := emitter.EmitPIIResolved(context.Background(), 3, 1)
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	// Should have progress 80 when fields are granted
	if events[0].Metadata.Progress != 80 {
		t.Errorf("expected progress 80 with granted fields, got %d", events[0].Metadata.Progress)
	}
}

// TestBlindFillStatusEmitter_PIIError tests error status emission
func TestBlindFillStatusEmitter_PIIError(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBlindFillStatusEmitter("agent-003", mock)

	err := emitter.EmitPIIError(context.Background(), "access denied")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Status != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, events[0].Status)
	}
	if events[0].Metadata.Error != "access denied" {
		t.Errorf("expected error message 'access denied', got %s", events[0].Metadata.Error)
	}
}

// TestBlindFillStatusEmitter_AwaitingApproval tests approval waiting status
func TestBlindFillStatusEmitter_AwaitingApproval(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBlindFillStatusEmitter("agent-004", mock)

	fields := []string{"name", "email", "phone"}
	err := emitter.EmitAwaitingApproval(context.Background(), fields, "req-123")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Status != StatusAwaitingApproval {
		t.Errorf("expected status %s, got %s", StatusAwaitingApproval, events[0].Status)
	}
	if events[0].Metadata.TaskID != "req-123" {
		t.Errorf("expected task_id req-123, got %s", events[0].Metadata.TaskID)
	}
	if len(events[0].Metadata.FieldsRequested) != 3 {
		t.Errorf("expected 3 fields requested, got %d", len(events[0].Metadata.FieldsRequested))
	}
}

// TestBlindFillStatusEmitter_NilEmitter tests that nil emitter doesn't panic
func TestBlindFillStatusEmitter_NilEmitter(t *testing.T) {
	emitter := NewBlindFillStatusEmitter("agent-005", nil)

	// Should not panic
	err := emitter.EmitResolvingPII(context.Background(), "test", 1)
	if err != nil {
		t.Errorf("expected nil error with nil emitter, got %v", err)
	}
}

// TestBrowserStatusEmitter_Navigating tests navigation status emission
func TestBrowserStatusEmitter_Navigating(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBrowserStatusEmitter("browser-001", mock)

	err := emitter.EmitNavigating(context.Background(), "https://example.com/login")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Status != StatusBrowsing {
		t.Errorf("expected status %s, got %s", StatusBrowsing, events[0].Status)
	}
	if events[0].Metadata.URL != "https://example.com/login" {
		t.Errorf("expected URL, got %s", events[0].Metadata.URL)
	}
	if events[0].Metadata.Step != "navigating" {
		t.Errorf("expected step navigating, got %s", events[0].Metadata.Step)
	}
}

// TestBrowserStatusEmitter_FormFilling tests form filling status emission
func TestBrowserStatusEmitter_FormFilling(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBrowserStatusEmitter("browser-002", mock)

	err := emitter.EmitFormFilling(context.Background(), "filling_email", 30)
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if events[0].Status != StatusFormFilling {
		t.Errorf("expected status %s, got %s", StatusFormFilling, events[0].Status)
	}
	if events[0].Metadata.Progress != 30 {
		t.Errorf("expected progress 30, got %d", events[0].Metadata.Progress)
	}
}

// TestBrowserStatusEmitter_AwaitingCaptcha tests captcha waiting status
func TestBrowserStatusEmitter_AwaitingCaptcha(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBrowserStatusEmitter("browser-003", mock)

	err := emitter.EmitAwaitingCaptcha(context.Background(), "https://example.com/protected")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if events[0].Status != StatusAwaitingCaptcha {
		t.Errorf("expected status %s, got %s", StatusAwaitingCaptcha, events[0].Status)
	}
}

// TestBrowserStatusEmitter_Complete tests completion status emission
func TestBrowserStatusEmitter_Complete(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBrowserStatusEmitter("browser-004", mock)

	err := emitter.EmitComplete(context.Background(), "https://example.com/success")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if events[0].Status != StatusComplete {
		t.Errorf("expected status %s, got %s", StatusComplete, events[0].Status)
	}
	if events[0].Metadata.Progress != 100 {
		t.Errorf("expected progress 100 on complete, got %d", events[0].Metadata.Progress)
	}
}

// TestBrowserStatusEmitter_Error tests error status emission
func TestBrowserStatusEmitter_Error(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBrowserStatusEmitter("browser-005", mock)

	err := emitter.EmitError(context.Background(), "form_submission", "connection timeout")
	if err != nil {
		t.Fatalf("failed to emit status: %v", err)
	}

	events := mock.getEvents()
	if events[0].Status != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, events[0].Status)
	}
	if events[0].Metadata.Error != "connection timeout" {
		t.Errorf("expected error message, got %s", events[0].Metadata.Error)
	}
}

// TestBrowserStatusEmitter_NilEmitter tests nil emitter doesn't panic
func TestBrowserStatusEmitter_NilEmitter(t *testing.T) {
	emitter := NewBrowserStatusEmitter("browser-006", nil)

	// Should not panic
	err := emitter.EmitNavigating(context.Background(), "https://example.com")
	if err != nil {
		t.Errorf("expected nil error with nil emitter, got %v", err)
	}
}

// TestStatusEventTimestamp tests that timestamps are set correctly
func TestStatusEventTimestamp(t *testing.T) {
	mock := &mockStatusEmitter{}
	emitter := NewBlindFillStatusEmitter("agent-ts", mock)

	before := time.Now().UnixMilli()
	emitter.EmitResolvingPII(context.Background(), "test", 1)
	after := time.Now().UnixMilli()

	events := mock.getEvents()
	if events[0].Timestamp < before || events[0].Timestamp > after {
		t.Errorf("timestamp %d not in expected range [%d, %d]", events[0].Timestamp, before, after)
	}
}

// TestIntegrationEmitter tests the IntegrationEmitter adapter
func TestIntegrationEmitter(t *testing.T) {
	// Create a state machine and integration
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-integration"})

	// First transition from OFFLINE to INITIALIZING
	if err := sm.Transition(StatusInitializing, StatusMetadata{}); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "test-integration",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	emitter := NewIntegrationEmitter(integration)

	// Test browsing status
	err = emitter.EmitStatus(context.Background(), StatusEvent{
		AgentID:   "test-integration",
		Status:    StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL: "https://example.com",
		},
	})
	if err != nil {
		t.Errorf("failed to emit browsing status: %v", err)
	}

	// Verify state changed
	if sm.Current() != StatusBrowsing {
		t.Errorf("expected status %s, got %s", StatusBrowsing, sm.Current())
	}
}
