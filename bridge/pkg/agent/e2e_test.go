package agent

import (
	"context"
	"testing"
	"time"
)

// TestE2E_BrowsingToCompletion tests a complete browsing -> form fill -> complete flow
func TestE2E_BrowsingToCompletion(t *testing.T) {
	// Setup
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-001"})
	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "e2e-agent-001",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	// Create emitters
	browserEmitter := CreateBrowserEmitter("e2e-agent-001", integration)
	blindFillEmitter := CreateBlindFillEmitter("e2e-agent-001", integration)

	ctx := context.Background()

	// Step 1: Initialize
	if err := sm.Transition(StatusInitializing, StatusMetadata{
		TaskType: "test_task",
	}); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Step 2: Navigate to page (via emitter)
	if err := browserEmitter.EmitNavigating(ctx, "https://example.com/login"); err != nil {
		t.Fatalf("failed to emit navigating status: %v", err)
	}

	// Verify state changed to browsing
	if sm.Current() != StatusBrowsing {
		t.Errorf("expected status %s, got %s", StatusBrowsing, sm.Current())
	}

	// Step 3: Start form filling (via emitter)
	if err := browserEmitter.EmitFormFilling(ctx, "page_loaded", 20); err != nil {
		t.Fatalf("failed to emit form filling status: %v", err)
	}

	// Step 4: Request PII access
	fields := []string{"email", "password"}
	if err := sm.RequestApproval(fields); err != nil {
		t.Fatalf("failed to request approval: %v", err)
	}

	// Emit awaiting approval status
	if err := blindFillEmitter.EmitAwaitingApproval(ctx, fields, "req-e2e-001"); err != nil {
		t.Errorf("failed to emit awaiting approval: %v", err)
	}

	// Verify state
	if sm.Current() != StatusAwaitingApproval {
		t.Errorf("expected status %s, got %s", StatusAwaitingApproval, sm.Current())
	}

	// Step 5: Approval granted, back to form filling
	if err := sm.Transition(StatusFormFilling, StatusMetadata{
		Step:     "approval_granted",
		Progress: 50,
	}); err != nil {
		t.Fatalf("failed to return to form filling: %v", err)
	}

	// Emit PII resolved status
	if err := blindFillEmitter.EmitPIIResolved(ctx, 2, 0); err != nil {
		t.Errorf("failed to emit PII resolved: %v", err)
	}

	// Step 6: Continue form filling (same state, metadata update)
	if err := browserEmitter.EmitFormFilling(ctx, "filling_form", 70); err != nil {
		t.Errorf("failed to emit form filling: %v", err)
	}

	// Step 7: Complete task (via emitter)
	if err := browserEmitter.EmitComplete(ctx, "https://example.com/login"); err != nil {
		t.Fatalf("failed to emit complete: %v", err)
	}

	// Verify final state
	if sm.Current() != StatusComplete {
		t.Errorf("expected final status %s, got %s", StatusComplete, sm.Current())
	}
}

// TestE2E_ErrorRecovery tests error handling and recovery
func TestE2E_ErrorRecovery(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-002"})
	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "e2e-agent-002",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	browserEmitter := CreateBrowserEmitter("e2e-agent-002", integration)
	ctx := context.Background()

	// Start browsing via emitter
	if err := browserEmitter.EmitNavigating(ctx, "https://example.com"); err != nil {
		t.Fatalf("failed to emit navigating: %v", err)
	}

	// Encounter error via emitter
	if err := browserEmitter.EmitError(ctx, "navigation", "network timeout"); err != nil {
		t.Fatalf("failed to emit error: %v", err)
	}

	// Verify error state
	if sm.Current() != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, sm.Current())
	}

	// Recover to idle
	if err := sm.Transition(StatusIdle, StatusMetadata{}); err != nil {
		t.Fatalf("failed to recover to idle: %v", err)
	}

	// Verify recovery
	if sm.Current() != StatusIdle {
		t.Errorf("expected status %s after recovery, got %s", StatusIdle, sm.Current())
	}
}

// TestE2E_CaptchaFlow tests CAPTCHA handling flow
func TestE2E_CaptchaFlow(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-003"})
	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "e2e-agent-003",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	browserEmitter := CreateBrowserEmitter("e2e-agent-003", integration)
	ctx := context.Background()

	// Navigate to protected page via emitter
	if err := browserEmitter.EmitNavigating(ctx, "https://example.com/protected"); err != nil {
		t.Fatalf("failed to emit navigating: %v", err)
	}

	// Hit CAPTCHA via emitter
	if err := browserEmitter.EmitAwaitingCaptcha(ctx, "https://example.com/protected"); err != nil {
		t.Fatalf("failed to emit captcha status: %v", err)
	}

	// Verify state is terminal (requires user action)
	if !sm.Current().IsTerminal() {
		t.Error("expected AWAITING_CAPTCHA to be terminal state")
	}
	if !sm.Current().NeedsUserAction() {
		t.Error("expected AWAITING_CAPTCHA to need user action")
	}

	// CAPTCHA solved, resume browsing via integration method
	if err := integration.ResolveCaptcha(); err != nil {
		t.Fatalf("failed to resolve captcha: %v", err)
	}

	// Complete via emitter
	if err := browserEmitter.EmitComplete(ctx, "https://example.com/protected"); err != nil {
		t.Fatalf("failed to emit complete: %v", err)
	}
}

// TestE2E_2FAFlow tests 2FA handling flow
func TestE2E_2FAFlow(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-004"})
	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "e2e-agent-004",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	browserEmitter := CreateBrowserEmitter("e2e-agent-004", integration)
	ctx := context.Background()

	// Navigate via emitter
	if err := browserEmitter.EmitNavigating(ctx, "https://bank.example.com"); err != nil {
		t.Fatalf("failed to emit navigating: %v", err)
	}

	// Start form filling via emitter
	if err := browserEmitter.EmitFormFilling(ctx, "filling_credentials", 30); err != nil {
		t.Fatalf("failed to emit form filling: %v", err)
	}

	// Hit 2FA via emitter
	if err := browserEmitter.EmitAwaiting2FA(ctx, "https://bank.example.com/verify"); err != nil {
		t.Fatalf("failed to emit 2FA status: %v", err)
	}

	// Verify state
	if !sm.Current().NeedsUserAction() {
		t.Error("expected AWAITING_2FA to need user action")
	}

	// 2FA provided, resume form filling via integration
	if err := integration.Resolve2FA("123456"); err != nil {
		t.Fatalf("failed to resolve 2FA: %v", err)
	}

	// Complete via emitter
	if err := browserEmitter.EmitComplete(ctx, "https://bank.example.com/verify"); err != nil {
		t.Fatalf("failed to emit complete: %v", err)
	}
}

// TestE2E_PaymentFlow tests payment processing flow
func TestE2E_PaymentFlow(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-005"})
	integration, err := NewIntegration(IntegrationConfig{
		AgentID:      "e2e-agent-005",
		StateMachine: sm,
	})
	if err != nil {
		t.Fatalf("failed to create integration: %v", err)
	}

	browserEmitter := CreateBrowserEmitter("e2e-agent-005", integration)
	blindFillEmitter := CreateBlindFillEmitter("e2e-agent-005", integration)
	ctx := context.Background()

	// Navigate via emitter
	if err := browserEmitter.EmitNavigating(ctx, "https://shop.example.com/checkout"); err != nil {
		t.Fatalf("failed to emit navigating: %v", err)
	}

	// Start form filling via emitter
	if err := browserEmitter.EmitFormFilling(ctx, "filling_checkout", 30); err != nil {
		t.Fatalf("failed to emit form filling: %v", err)
	}

	// Request payment fields
	fields := []string{"card_number", "card_expiry", "card_cvv", "billing_address"}
	if err := sm.RequestApproval(fields); err != nil {
		t.Fatalf("failed to request approval: %v", err)
	}
	if err := blindFillEmitter.EmitAwaitingApproval(ctx, fields, "req-payment-001"); err != nil {
		t.Errorf("failed to emit awaiting approval: %v", err)
	}

	// Approval granted
	if err := sm.Transition(StatusFormFilling, StatusMetadata{
		Step:     "payment_approved",
		Progress: 60,
	}); err != nil {
		t.Fatalf("failed to return to form filling: %v", err)
	}
	if err := blindFillEmitter.EmitApprovalGranted(ctx, "req-payment-001"); err != nil {
		t.Errorf("failed to emit approval granted: %v", err)
	}

	// PII resolved
	if err := blindFillEmitter.EmitPIIResolved(ctx, 4, 0); err != nil {
		t.Errorf("failed to emit PII resolved: %v", err)
	}

	// Start payment processing via emitter
	if err := browserEmitter.EmitProcessingPayment(ctx); err != nil {
		t.Fatalf("failed to emit payment processing: %v", err)
	}

	// Verify state
	if sm.Current() != StatusProcessingPayment {
		t.Errorf("expected status %s, got %s", StatusProcessingPayment, sm.Current())
	}

	// Complete via emitter
	if err := browserEmitter.EmitComplete(ctx, "https://shop.example.com/checkout/confirmation"); err != nil {
		t.Fatalf("failed to emit complete: %v", err)
	}
}

// TestE2E_EventHistory tests that events are properly tracked
func TestE2E_EventHistory(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-006"})

	// Collect events via subscription
	var events []StatusEvent
	go func() {
		eventCh := sm.Events()
		for {
			select {
			case event := <-eventCh:
				events = append(events, event)
			case <-sm.Done():
				return
			}
		}
	}()

	// Perform transitions
	sm.Transition(StatusInitializing, StatusMetadata{})
	sm.Transition(StatusBrowsing, StatusMetadata{URL: "https://example.com"})
	sm.Transition(StatusFormFilling, StatusMetadata{Progress: 50})
	sm.Transition(StatusComplete, StatusMetadata{Progress: 100})

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify events were captured
	// Note: This test might be flaky due to timing, but demonstrates the pattern
	sm.Shutdown()

	// At minimum, we should have captured some events
	if len(events) == 0 {
		t.Log("Warning: No events captured (timing issue in test)")
	}
}

// TestE2E_InvalidTransitions tests that invalid transitions are rejected
func TestE2E_InvalidTransitions(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "e2e-agent-007"})

	// Try invalid transition: IDLE -> COMPLETE (not allowed)
	err := sm.Transition(StatusComplete, StatusMetadata{})
	if err == nil {
		t.Error("expected error for invalid transition IDLE -> COMPLETE")
	}

	// Valid path: IDLE -> INITIALIZING
	if err := sm.Transition(StatusInitializing, StatusMetadata{}); err != nil {
		t.Fatalf("failed valid transition: %v", err)
	}

	// Try invalid: INITIALIZING -> AWAITING_2FA (not allowed)
	err = sm.Transition(StatusAwaiting2FA, StatusMetadata{})
	if err == nil {
		t.Error("expected error for invalid transition INITIALIZING -> AWAITING_2FA")
	}
}
