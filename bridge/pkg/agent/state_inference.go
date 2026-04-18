// Package agent provides bridge-side state inference from CDP/workflow signals.
//
// The inference engine maps Chrome DevTools Protocol events and workflow status
// to AgentStatus values. It uses ForceTransition on the StateMachine because
// inferred transitions may not follow the normal valid-transitions graph.
//
// This is a bridge-side library only. It does NOT replace container-driven
// state reporting (which does not exist yet) or command-driven transitions
// (RequestPIIAccess, etc.).
package agent

// CDPEvent represents a Chrome DevTools Protocol event observed from Jetski.
type CDPEvent struct {
	// Method is the CDP method name, e.g. "Page.frameNavigated", "DOM.focus".
	Method string
	// Params contains the event parameters.
	Params map[string]interface{}
}

// WorkflowStatus represents the current workflow state reported by the
// workflow engine side-channel (not observable via CDP).
type WorkflowStatus struct {
	// State is the workflow engine's internal state name.
	// Recognized values: "captcha", "twofa", "payment", "offline".
	State string
	// StepName is the human-readable step name within the workflow.
	StepName string
	// WorkflowID identifies the workflow instance.
	WorkflowID string
}

// workflowStateToAgent maps workflow engine states to AgentStatus.
// These are "invisible" states not observable via CDP events.
var workflowStateToAgent = map[string]AgentStatus{
	"captcha": StatusAwaitingCaptcha,
	"twofa":   StatusAwaiting2FA,
	"payment": StatusProcessingPayment,
	"offline": StatusOffline,
}

// InferAgentState determines the likely agent state from CDP events and
// workflow status. It returns the inferred AgentStatus without modifying
// any state machine — the caller should apply it via ForceTransition.
//
// Inference priority:
//  1. Workflow side-channel states (captcha, 2FA, payment, offline) take
//     precedence because they represent intentional blocking states.
//  2. CDP event-driven states (browsing, form_filling) are inferred from
//     the most recent relevant CDP event.
//  3. If the workflow has exited but CDP shows no activity, infer from
//     exit code semantics: exit 0 → COMPLETE, exit non-zero → ERROR.
//  4. Unknown events maintain the current state (no transition).
func InferAgentState(cdpEvents []CDPEvent, workflowState WorkflowStatus, currentState AgentStatus) AgentStatus {
	// Priority 1: Workflow side-channel overrides everything.
	// These states are invisible to CDP and represent intentional blocks.
	if workflowState.State != "" {
		if mapped, ok := workflowStateToAgent[workflowState.State]; ok {
			return mapped
		}
	}

	// Priority 2: Exit-driven states from workflow completion.
	// The workflow engine sets State to "exit_0" or "exit_nonzero".
	switch workflowState.State {
	case "exit_0":
		return StatusComplete
	case "exit_nonzero":
		return StatusError
	}

	// Priority 3: If currently in AWAITING_APPROVAL, do NOT transition away
	// based on CDP events. Approval state is managed by RequestPIIAccess RPC.
	if currentState == StatusAwaitingApproval {
		return StatusAwaitingApproval
	}

	// Priority 4: CDP event-driven inference.
	// Process events from oldest to newest; the last matching event wins.
	inferred := currentState
	for _, evt := range cdpEvents {
		switch evt.Method {
		case "Page.frameNavigated":
			inferred = StatusBrowsing
		case "DOM.focus":
			// DOM.focus on an input/textarea/select element → FORM_FILLING
			if isInputElement(evt.Params) {
				inferred = StatusFormFilling
			}
		case "Runtime.executionContextCreated":
			// Tool dispatch with no browser activity → INITIALIZING
			// This fires when a new JS context is created (page reload, nav).
			// Only infer INITIALIZING if we're still early-state.
			if currentState == StatusIdle || currentState == StatusOffline {
				inferred = StatusInitializing
			}
		case "Inspector.detached", "Inspector.targetCrashed":
			// Jetski restart: CDP connection drops. Maintain current state.
			// Do NOT transition — the connection will reconnect.
		default:
			// Unknown CDP events → maintain current state (do NOT transition).
		}
	}

	return inferred
}

// isInputElement checks whether the DOM.focus event target is an input element.
// It inspects the CDP event params for "nodeType" or "nodeName" fields.
func isInputElement(params map[string]interface{}) bool {
	if params == nil {
		return false
	}

	// Check for explicit nodeName (e.g., "INPUT", "TEXTAREA", "SELECT")
	if nodeName, ok := params["nodeName"].(string); ok {
		switch nodeName {
		case "INPUT", "TEXTAREA", "SELECT":
			return true
		}
	}

	// Check for "type" field indicating input-like behavior
	if typ, ok := params["type"].(string); ok {
		switch typ {
		case "text", "password", "email", "tel", "number", "search", "url":
			return true
		}
	}

	return false
}

// ApplyInferredState applies the inferred state to a StateMachine using
// ForceTransition. This is a convenience function that combines InferAgentState
// with the actual state machine update.
//
// Returns true if the state changed, false if it remained the same.
func ApplyInferredState(sm *StateMachine, cdpEvents []CDPEvent, workflowState WorkflowStatus) bool {
	current := sm.Current()
	inferred := InferAgentState(cdpEvents, workflowState, current)

	if inferred == current {
		return false
	}

	sm.ForceTransition(inferred)
	return true
}
