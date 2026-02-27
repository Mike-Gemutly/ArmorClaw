package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewIntegration(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})

	cfg := IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	}

	integration, err := NewIntegration(cfg)
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	if integration.agentID != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got %s", integration.agentID)
	}
}

func TestNewIntegrationMissingAgentID(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test"})

	cfg := IntegrationConfig{
		StateMachine: sm,
	}

	_, err := NewIntegration(cfg)
	if err == nil {
		t.Error("expected error for missing agent_id")
	}
}

func TestNewIntegrationMissingStateMachine(t *testing.T) {
	cfg := IntegrationConfig{
		AgentID: "test-agent",
	}

	_, err := NewIntegration(cfg)
	if err == nil {
		t.Error("expected error for missing state_machine")
	}
}

func TestStartBrowsing(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	// Initialize the state machine properly
	sm.ForceTransition(StatusInitializing)
	sm.ForceTransition(StatusBrowsing)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	// Already in BROWSING state, update with URL
	err := integration.StartBrowsing("https://example.com")
	if err != nil {
		t.Fatalf("failed to start browsing: %v", err)
	}

	if sm.Current() != StatusBrowsing {
		t.Errorf("expected BROWSING status, got %s", sm.Current())
	}

	meta := sm.Metadata()
	if meta.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %s", meta.URL)
	}
}

func TestUpdateProgress(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusFormFilling)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.UpdateProgress("2/5", 40)
	if err != nil {
		t.Fatalf("failed to update progress: %v", err)
	}

	meta := sm.Metadata()
	if meta.Step != "2/5" {
		t.Errorf("expected step '2/5', got %s", meta.Step)
	}
	if meta.Progress != 40 {
		t.Errorf("expected progress 40, got %d", meta.Progress)
	}
}

func TestWaitForCaptcha(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusBrowsing)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.WaitForCaptcha(context.Background())
	if err != nil {
		t.Fatalf("failed to wait for captcha: %v", err)
	}

	if sm.Current() != StatusAwaitingCaptcha {
		t.Errorf("expected AWAITING_CAPTCHA status, got %s", sm.Current())
	}
}

func TestResolveCaptcha(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusAwaitingCaptcha)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.ResolveCaptcha()
	if err != nil {
		t.Fatalf("failed to resolve captcha: %v", err)
	}

	if sm.Current() != StatusBrowsing {
		t.Errorf("expected BROWSING status after captcha, got %s", sm.Current())
	}
}

func TestWaitFor2FA(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusFormFilling)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.WaitFor2FA()
	if err != nil {
		t.Fatalf("failed to wait for 2FA: %v", err)
	}

	if sm.Current() != StatusAwaiting2FA {
		t.Errorf("expected AWAITING_2FA status, got %s", sm.Current())
	}
}

func TestResolve2FA(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusAwaiting2FA)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.Resolve2FA("123456")
	if err != nil {
		t.Fatalf("failed to resolve 2FA: %v", err)
	}

	if sm.Current() != StatusFormFilling {
		t.Errorf("expected FORM_FILLING status after 2FA, got %s", sm.Current())
	}
}

func TestStartPayment(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusFormFilling)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.StartPayment()
	if err != nil {
		t.Fatalf("failed to start payment: %v", err)
	}

	if sm.Current() != StatusProcessingPayment {
		t.Errorf("expected PROCESSING_PAYMENT status, got %s", sm.Current())
	}
}

func TestCompleteTask(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusProcessingPayment)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.CompleteTask()
	if err != nil {
		t.Fatalf("failed to complete task: %v", err)
	}

	if sm.Current() != StatusComplete {
		t.Errorf("expected COMPLETE status, got %s", sm.Current())
	}
}

func TestFailTask(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusBrowsing)

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	err := integration.FailTask(errors.New("test error"))
	if err != nil {
		t.Fatalf("failed to fail task: %v", err)
	}

	if sm.Current() != StatusError {
		t.Errorf("expected ERROR status, got %s", sm.Current())
	}

	meta := sm.Metadata()
	if meta.Error != "test error" {
		t.Errorf("expected error message 'test error', got %s", meta.Error)
	}
}

func TestGetStatus(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})
	sm.ForceTransition(StatusBrowsing, StatusMetadata{URL: "https://example.com"})

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	status := integration.GetStatus()
	if status.AgentID != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got %s", status.AgentID)
	}
	if status.Status != StatusBrowsing {
		t.Errorf("expected status BROWSING, got %s", status.Status)
	}
}

func TestOnStatusChange(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-agent"})

	integration, _ := NewIntegration(IntegrationConfig{
		AgentID:      "test-agent",
		StateMachine: sm,
	})

	// Set up callback
	receivedEvent := make(chan StatusEvent, 1)
	integration.OnStatusChange(func(ctx context.Context, event StatusEvent) error {
		receivedEvent <- event
		return nil
	})

	// Trigger state change
	sm.ForceTransition(StatusInitializing)

	// Wait for callback
	select {
	case event := <-receivedEvent:
		if event.Status != StatusInitializing {
			t.Errorf("expected status INITIALIZING in callback, got %s", event.Status)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for status change callback")
	}
}

func TestAgentCoordinator(t *testing.T) {
	coordinator := NewAgentCoordinator()

	sm := NewStateMachine(StateMachineConfig{AgentID: "agent-001"})
	integration, err := coordinator.RegisterAgent("agent-001", sm)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	if integration == nil {
		t.Fatal("expected non-nil integration")
	}

	// Get agent
	retrieved, err := coordinator.GetAgent("agent-001")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}

	if retrieved != integration {
		t.Error("expected same integration instance")
	}

	// Get all statuses
	statuses := coordinator.GetAllStatuses()
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}

	// Unregister
	coordinator.UnregisterAgent("agent-001")

	// Verify unregistered
	_, err = coordinator.GetAgent("agent-001")
	if err == nil {
		t.Error("expected error for unregistered agent")
	}
}

func TestAgentCoordinatorDuplicate(t *testing.T) {
	coordinator := NewAgentCoordinator()

	sm1 := NewStateMachine(StateMachineConfig{AgentID: "agent-001"})
	_, _ = coordinator.RegisterAgent("agent-001", sm1)

	sm2 := NewStateMachine(StateMachineConfig{AgentID: "agent-001"})
	_, err := coordinator.RegisterAgent("agent-001", sm2)
	if err == nil {
		t.Error("expected error for duplicate agent registration")
	}
}
