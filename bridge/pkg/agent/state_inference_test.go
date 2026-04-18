package agent

import (
	"testing"
)

// --- CDP event helpers ---

func cdpNav() CDPEvent {
	return CDPEvent{
		Method: "Page.frameNavigated",
		Params: map[string]interface{}{"url": "https://example.com"},
	}
}

func cdpFocusInput(nodeName string) CDPEvent {
	return CDPEvent{
		Method: "DOM.focus",
		Params: map[string]interface{}{"nodeName": nodeName},
	}
}

func cdpFocusByType(typ string) CDPEvent {
	return CDPEvent{
		Method: "DOM.focus",
		Params: map[string]interface{}{"type": typ},
	}
}

func cdpUnknown() CDPEvent {
	return CDPEvent{
		Method: "Network.requestWillBeSent",
		Params: map[string]interface{}{"url": "https://example.com/api"},
	}
}

func cdpContextCreated() CDPEvent {
	return CDPEvent{
		Method: "Runtime.executionContextCreated",
		Params: map[string]interface{}{},
	}
}

func cdpDetached() CDPEvent {
	return CDPEvent{
		Method: "Inspector.detached",
		Params: map[string]interface{}{},
	}
}

// --- Test cases covering all 11 AgentStatus states ---

func TestInferState_Idle_NoEvents(t *testing.T) {
	result := InferAgentState(nil, WorkflowStatus{}, StatusIdle)
	if result != StatusIdle {
		t.Errorf("expected IDLE, got %s", result)
	}
}

func TestInferState_Idle_EmptyWorkflow(t *testing.T) {
	result := InferAgentState([]CDPEvent{}, WorkflowStatus{}, StatusIdle)
	if result != StatusIdle {
		t.Errorf("expected IDLE with empty inputs, got %s", result)
	}
}

func TestInferState_Initializing_ContextCreatedFromIdle(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpContextCreated()},
		WorkflowStatus{},
		StatusIdle,
	)
	if result != StatusInitializing {
		t.Errorf("expected INITIALIZING from Runtime.executionContextCreated while IDLE, got %s", result)
	}
}

func TestInferState_Initializing_ContextCreatedFromOffline(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpContextCreated()},
		WorkflowStatus{},
		StatusOffline,
	)
	if result != StatusInitializing {
		t.Errorf("expected INITIALIZING from Runtime.executionContextCreated while OFFLINE, got %s", result)
	}
}

func TestInferState_Browsing_FrameNavigated(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()},
		WorkflowStatus{},
		StatusIdle,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING from Page.frameNavigated, got %s", result)
	}
}

func TestInferState_Browsing_FrameNavigatedOverridesInitializing(t *testing.T) {
	// Both events fire; last one wins for CDP states
	result := InferAgentState(
		[]CDPEvent{cdpContextCreated(), cdpNav()},
		WorkflowStatus{},
		StatusIdle,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (nav overrides init), got %s", result)
	}
}

func TestInferState_FormFilling_FocusInput(t *testing.T) {
	tests := []struct {
		name  string
		event CDPEvent
	}{
		{"INPUT", cdpFocusInput("INPUT")},
		{"TEXTAREA", cdpFocusInput("TEXTAREA")},
		{"SELECT", cdpFocusInput("SELECT")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InferAgentState(
				[]CDPEvent{tc.event},
				WorkflowStatus{},
				StatusBrowsing,
			)
			if result != StatusFormFilling {
				t.Errorf("expected FORM_FILLING from DOM.focus on %s, got %s", tc.name, result)
			}
		})
	}
}

func TestInferState_FormFilling_FocusByType(t *testing.T) {
	types := []string{"text", "password", "email", "tel", "number", "search", "url"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			result := InferAgentState(
				[]CDPEvent{cdpFocusByType(typ)},
				WorkflowStatus{},
				StatusBrowsing,
			)
			if result != StatusFormFilling {
				t.Errorf("expected FORM_FILLING from DOM.focus type=%s, got %s", typ, result)
			}
		})
	}
}

func TestInferState_FormFilling_FocusNonInput(t *testing.T) {
	// DOM.focus on a non-input element should NOT infer FORM_FILLING
	result := InferAgentState(
		[]CDPEvent{cdpFocusInput("DIV")},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (DIV focus is not form filling), got %s", result)
	}
}

func TestInferState_AwaitingCaptcha_WorkflowOverride(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()}, // CDP says browsing
		WorkflowStatus{State: "captcha", WorkflowID: "wf-1"},
		StatusBrowsing,
	)
	if result != StatusAwaitingCaptcha {
		t.Errorf("expected AWAITING_CAPTCHA from workflow override, got %s", result)
	}
}

func TestInferState_Awaiting2FA_WorkflowOverride(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()},
		WorkflowStatus{State: "twofa", WorkflowID: "wf-1"},
		StatusBrowsing,
	)
	if result != StatusAwaiting2FA {
		t.Errorf("expected AWAITING_2FA from workflow override, got %s", result)
	}
}

func TestInferState_AwaitingApproval_PreserveDuringCDPEvents(t *testing.T) {
	// Race condition: CDP events during AWAITING_APPROVAL must NOT transition
	result := InferAgentState(
		[]CDPEvent{cdpNav(), cdpFocusInput("INPUT")},
		WorkflowStatus{},
		StatusAwaitingApproval,
	)
	if result != StatusAwaitingApproval {
		t.Errorf("expected AWAITING_APPROVAL preserved during CDP events, got %s", result)
	}
}

func TestInferState_ProcessingPayment_WorkflowOverride(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()},
		WorkflowStatus{State: "payment", WorkflowID: "wf-1"},
		StatusFormFilling,
	)
	if result != StatusProcessingPayment {
		t.Errorf("expected PROCESSING_PAYMENT from workflow override, got %s", result)
	}
}

func TestInferState_Error_ExitNonzero(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{},
		WorkflowStatus{State: "exit_nonzero"},
		StatusBrowsing,
	)
	if result != StatusError {
		t.Errorf("expected ERROR from exit_nonzero, got %s", result)
	}
}

func TestInferState_Complete_ExitZero(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{},
		WorkflowStatus{State: "exit_0"},
		StatusFormFilling,
	)
	if result != StatusComplete {
		t.Errorf("expected COMPLETE from exit_0, got %s", result)
	}
}

func TestInferState_Offline_WorkflowOverride(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()},
		WorkflowStatus{State: "offline", WorkflowID: "wf-1"},
		StatusBrowsing,
	)
	if result != StatusOffline {
		t.Errorf("expected OFFLINE from workflow override, got %s", result)
	}
}

// --- Unknown CDP events maintain current state ---

func TestInferState_UnknownCDPEvent_NoTransition(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpUnknown()},
		WorkflowStatus{},
		StatusFormFilling,
	)
	if result != StatusFormFilling {
		t.Errorf("expected FORM_FILLING (unknown event preserves state), got %s", result)
	}
}

func TestInferState_MultipleUnknownEvents_NoTransition(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpUnknown(), cdpUnknown(), cdpUnknown()},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (multiple unknown events preserve state), got %s", result)
	}
}

// --- Jetski reconnect: CDP connection drop/reconnect maintains state ---

func TestInferState_InspectorDetached_MaintainsCurrent(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpDetached()},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (Inspector.detached maintains state), got %s", result)
	}
}

func TestInferState_TargetCrashed_MaintainsCurrent(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{{Method: "Inspector.targetCrashed", Params: map[string]interface{}{}}},
		WorkflowStatus{},
		StatusFormFilling,
	)
	if result != StatusFormFilling {
		t.Errorf("expected FORM_FILLING (Inspector.targetCrashed maintains state), got %s", result)
	}
}

// --- Workflow override takes precedence over CDP events ---

func TestInferState_WorkflowOverridesCDP(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav(), cdpFocusInput("INPUT")}, // CDP says FORM_FILLING
		WorkflowStatus{State: "captcha"},             // Workflow says CAPTCHA
		StatusBrowsing,
	)
	if result != StatusAwaitingCaptcha {
		t.Errorf("expected AWAITING_CAPTCHA (workflow overrides CDP), got %s", result)
	}
}

// --- Last CDP event wins for event-driven states ---

func TestInferState_LastCDPEventWins(t *testing.T) {
	// Navigation then form focus → last event wins → FORM_FILLING
	result := InferAgentState(
		[]CDPEvent{cdpNav(), cdpFocusInput("INPUT")},
		WorkflowStatus{},
		StatusIdle,
	)
	if result != StatusFormFilling {
		t.Errorf("expected FORM_FILLING (last CDP event wins), got %s", result)
	}
}

// --- DOM.focus with no params does not infer FORM_FILLING ---

func TestInferState_DomFocusNoParams(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{{Method: "DOM.focus", Params: nil}},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (nil params preserves state), got %s", result)
	}
}

func TestInferState_DomFocusEmptyParams(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{{Method: "DOM.focus", Params: map[string]interface{}{}}},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (empty params preserves state), got %s", result)
	}
}

// --- Runtime.executionContextCreated only initializes from early states ---

func TestInferState_ContextCreatedFromBrowsing_StaysBrowsing(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpContextCreated()},
		WorkflowStatus{},
		StatusBrowsing,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (context created during active state preserves it), got %s", result)
	}
}

func TestInferState_ContextCreatedFromFormFilling_StaysFormFilling(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpContextCreated()},
		WorkflowStatus{},
		StatusFormFilling,
	)
	if result != StatusFormFilling {
		t.Errorf("expected FORM_FILLING (context created during active state preserves it), got %s", result)
	}
}

// --- ApplyInferredState with StateMachine integration ---

func TestApplyInferredState_StateChange(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-apply"})
	sm.ForceTransition(StatusBrowsing)

	changed := ApplyInferredState(sm, []CDPEvent{cdpFocusInput("INPUT")}, WorkflowStatus{})
	if !changed {
		t.Error("expected state change to be true")
	}
	if sm.Current() != StatusFormFilling {
		t.Errorf("expected FORM_FILLING, got %s", sm.Current())
	}
}

func TestApplyInferredState_NoChange(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-nochange"})
	sm.ForceTransition(StatusBrowsing)

	changed := ApplyInferredState(sm, []CDPEvent{cdpNav()}, WorkflowStatus{})
	if changed {
		t.Error("expected no state change (already BROWSING)")
	}
}

func TestApplyInferredState_WorkflowForceTransition(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-workflow"})
	sm.ForceTransition(StatusBrowsing)

	changed := ApplyInferredState(sm, nil, WorkflowStatus{State: "payment"})
	if !changed {
		t.Error("expected state change from workflow")
	}
	if sm.Current() != StatusProcessingPayment {
		t.Errorf("expected PROCESSING_PAYMENT, got %s", sm.Current())
	}
}

// --- Unknown workflow state falls through to CDP ---

func TestInferState_UnknownWorkflowState_FallsThroughToCDP(t *testing.T) {
	result := InferAgentState(
		[]CDPEvent{cdpNav()},
		WorkflowStatus{State: "unknown_state", WorkflowID: "wf-1"},
		StatusIdle,
	)
	if result != StatusBrowsing {
		t.Errorf("expected BROWSING (unknown workflow state falls through to CDP), got %s", result)
	}
}

// --- Complete inference table coverage count ---

func TestInferState_AllElevenStates(t *testing.T) {
	// Verify every AgentStatus constant is reachable through inference.
	// Not every state needs explicit CDP events (some use workflow or exit).
	cases := []struct {
		name         string
		events       []CDPEvent
		workflow     WorkflowStatus
		currentState AgentStatus
		want         AgentStatus
	}{
		{"IDLE", nil, WorkflowStatus{}, StatusIdle, StatusIdle},
		{"INITIALIZING", []CDPEvent{cdpContextCreated()}, WorkflowStatus{}, StatusIdle, StatusInitializing},
		{"BROWSING", []CDPEvent{cdpNav()}, WorkflowStatus{}, StatusIdle, StatusBrowsing},
		{"FORM_FILLING", []CDPEvent{cdpFocusInput("INPUT")}, WorkflowStatus{}, StatusBrowsing, StatusFormFilling},
		{"AWAITING_CAPTCHA", nil, WorkflowStatus{State: "captcha"}, StatusBrowsing, StatusAwaitingCaptcha},
		{"AWAITING_2FA", nil, WorkflowStatus{State: "twofa"}, StatusBrowsing, StatusAwaiting2FA},
		{"AWAITING_APPROVAL", []CDPEvent{cdpNav()}, WorkflowStatus{}, StatusAwaitingApproval, StatusAwaitingApproval},
		{"PROCESSING_PAYMENT", nil, WorkflowStatus{State: "payment"}, StatusFormFilling, StatusProcessingPayment},
		{"ERROR", nil, WorkflowStatus{State: "exit_nonzero"}, StatusBrowsing, StatusError},
		{"COMPLETE", nil, WorkflowStatus{State: "exit_0"}, StatusFormFilling, StatusComplete},
		{"OFFLINE", nil, WorkflowStatus{State: "offline"}, StatusBrowsing, StatusOffline},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := InferAgentState(tc.events, tc.workflow, tc.currentState)
			if result != tc.want {
				t.Errorf("expected %s, got %s", tc.want, result)
			}
		})
	}
}

// --- Concurrent safety: ApplyInferredState with ForceTransition ---

func TestApplyInferredState_ConcurrentSafe(t *testing.T) {
	sm := NewStateMachine(StateMachineConfig{AgentID: "test-concurrent"})
	sm.ForceTransition(StatusBrowsing)

	// Simulate concurrent inference calls
	done := make(chan bool, 2)

	go func() {
		ApplyInferredState(sm, []CDPEvent{cdpNav()}, WorkflowStatus{State: "captcha"})
		done <- true
	}()
	go func() {
		ApplyInferredState(sm, []CDPEvent{cdpFocusInput("INPUT")}, WorkflowStatus{State: "payment"})
		done <- true
	}()

	<-done
	<-done

	// State should be one of the two — just verify no panic/race
	current := sm.Current()
	if current != StatusAwaitingCaptcha && current != StatusProcessingPayment {
		t.Errorf("expected AWAITING_CAPTCHA or PROCESSING_PAYMENT, got %s", current)
	}
}
