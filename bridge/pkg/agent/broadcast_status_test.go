package agent

import (
	"context"
	"testing"
	"time"

	"github.com/armorclaw/bridge/internal/events"
)

func TestBroadcastStatus_EmitsMatrixEvent(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	sub := bus.Subscribe()

	coordinator := NewAgentCoordinator()
	coordinator.SetEventBus(bus)

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-001"})
	integration, err := coordinator.RegisterAgent("agent-001", sm)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}
	integration.SetRoomID("!room-001:example.com")

	sm.ForceTransition(StatusBrowsing, StatusMetadata{
		URL: "https://example.com",
	})

	event := StatusEvent{
		AgentID:   "agent-001",
		Status:    StatusBrowsing,
		Previous:  StatusIdle,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:          "https://example.com",
			WorkflowID:   "wf-123",
			Step:         "navigate",
			InferredFrom: "CDP",
		},
	}

	err = coordinator.BroadcastStatus(context.Background(), event)
	if err != nil {
		t.Fatalf("BroadcastStatus returned error: %v", err)
	}

	select {
	case published := <-sub:
		if published.Type != "com.armorclaw.agent.status" {
			t.Errorf("expected event type 'com.armorclaw.agent.status', got %q", published.Type)
		}
		if published.RoomID != "!room-001:example.com" {
			t.Errorf("expected room_id '!room-001:example.com', got %q", published.RoomID)
		}
		if published.Sender != "bridge" {
			t.Errorf("expected sender 'bridge', got %q", published.Sender)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for published event")
	}
}

func TestBroadcastStatus_ContainsAllRequiredFields(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	sub := bus.Subscribe()

	coordinator := NewAgentCoordinator()
	coordinator.SetEventBus(bus)

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-002"})
	integration, _ := coordinator.RegisterAgent("agent-002", sm)
	integration.SetRoomID("!room-002:example.com")

	sm.ForceTransition(StatusFormFilling, StatusMetadata{
		Step:   "fill_payment",
		TaskID: "task-456",
	})

	event := StatusEvent{
		AgentID:   "agent-002",
		Status:    StatusFormFilling,
		Previous:  StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			WorkflowID:   "wf-789",
			Step:         "fill_payment",
			InferredFrom: "workflow",
		},
	}

	_ = coordinator.BroadcastStatus(context.Background(), event)

	select {
	case published := <-sub:
		content, ok := published.Content.(map[string]interface{})
		if !ok {
			t.Fatal("expected content to be map[string]interface{}")
		}

		// Verify state
		if state, ok := content["status"].(string); !ok || state != "FORM_FILLING" {
			t.Errorf("expected status 'FORM_FILLING', got %v", content["status"])
		}

		// Verify agent_id
		if agentID, ok := content["agent_id"].(string); !ok || agentID != "agent-002" {
			t.Errorf("expected agent_id 'agent-002', got %v", content["agent_id"])
		}

		// Verify timestamp
		if ts, ok := content["timestamp"].(int64); !ok || ts == 0 {
			t.Errorf("expected non-zero timestamp, got %v", content["timestamp"])
		}

		// Verify metadata contains workflow_id, step_name, inferred_from
		meta, ok := content["metadata"].(StatusMetadata)
		if !ok {
			t.Fatal("expected metadata to be StatusMetadata")
		}
		if meta.WorkflowID != "wf-789" {
			t.Errorf("expected workflow_id 'wf-789', got %q", meta.WorkflowID)
		}
		if meta.Step != "fill_payment" {
			t.Errorf("expected step 'fill_payment', got %q", meta.Step)
		}
		if meta.InferredFrom != "workflow" {
			t.Errorf("expected inferred_from 'workflow', got %q", meta.InferredFrom)
		}

	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for published event")
	}
}

func TestBroadcastStatus_AgentNotFound(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	coordinator := NewAgentCoordinator()
	coordinator.SetEventBus(bus)

	event := StatusEvent{
		AgentID:   "nonexistent",
		Status:    StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
	}

	err := coordinator.BroadcastStatus(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestBroadcastStatus_EventBusNotConfigured(t *testing.T) {
	coordinator := NewAgentCoordinator()

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-003"})
	_, _ = coordinator.RegisterAgent("agent-003", sm)

	event := StatusEvent{
		AgentID:   "agent-003",
		Status:    StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
	}

	err := coordinator.BroadcastStatus(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when event bus is not configured")
	}
}

func TestBroadcastStatus_EventTypeCorrect(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	sub := bus.Subscribe()

	coordinator := NewAgentCoordinator()
	coordinator.SetEventBus(bus)

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-004"})
	integration, _ := coordinator.RegisterAgent("agent-004", sm)
	integration.SetRoomID("!room-004:example.com")

	sm.ForceTransition(StatusAwaitingApproval)

	event := StatusEvent{
		AgentID:   "agent-004",
		Status:    StatusAwaitingApproval,
		Previous:  StatusFormFilling,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			FieldsRequested: []string{"credit_card_number", "cvv"},
			InferredFrom:    "command",
			WorkflowID:      "wf-approval",
		},
	}

	_ = coordinator.BroadcastStatus(context.Background(), event)

	select {
	case published := <-sub:
		// Verify the event type matches StatusEvent.EventType()
		expectedType := event.EventType()
		if published.Type != expectedType {
			t.Errorf("expected event type %q, got %q", expectedType, published.Type)
		}

		// Verify ID contains event type and agent ID
		if published.ID == "" {
			t.Error("expected non-empty event ID")
		}

	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for published event")
	}
}

func TestBroadcastStatus_StateChangeEmitsEvent(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	sub := bus.Subscribe()

	coordinator := NewAgentCoordinator()
	coordinator.SetEventBus(bus)

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-005"})
	integration, _ := coordinator.RegisterAgent("agent-005", sm)
	integration.SetRoomID("!room-005:example.com")

	// Simulate state inference: IDLE -> INITIALIZING -> BROWSING
	sm.ForceTransition(StatusInitializing)
	sm.ForceTransition(StatusBrowsing, StatusMetadata{
		URL:          "https://example.com",
		InferredFrom: "CDP",
		WorkflowID:   "wf-browse",
		Step:         "page_load",
	})

	lastEvent := sm.LastEvent()
	if lastEvent == nil {
		t.Fatal("expected last event from state machine")
	}

	err := coordinator.BroadcastStatus(context.Background(), *lastEvent)
	if err != nil {
		t.Fatalf("BroadcastStatus returned error: %v", err)
	}

	select {
	case published := <-sub:
		if published.Type != "com.armorclaw.agent.status" {
			t.Errorf("expected type 'com.armorclaw.agent.status', got %q", published.Type)
		}

		content, ok := published.Content.(map[string]interface{})
		if !ok {
			t.Fatal("expected map content")
		}

		if status := content["status"].(string); status != "BROWSING" {
			t.Errorf("expected status BROWSING, got %q", status)
		}

		meta := content["metadata"].(StatusMetadata)
		if meta.InferredFrom != "CDP" {
			t.Errorf("expected inferred_from 'CDP', got %q", meta.InferredFrom)
		}
		if meta.WorkflowID != "wf-browse" {
			t.Errorf("expected workflow_id 'wf-browse', got %q", meta.WorkflowID)
		}

	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for published event")
	}
}
