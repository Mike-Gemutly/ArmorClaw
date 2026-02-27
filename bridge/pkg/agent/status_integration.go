// Package agent provides status emission utilities for BlindFill and Browser operations.
// This enables these modules to emit status events without creating import cycles.
package agent

import (
	"context"
	"fmt"
	"time"
)

// StatusEmitter is an interface for emitting status events.
// This can be implemented by Integration or any other component.
type StatusEmitter interface {
	EmitStatus(ctx context.Context, event StatusEvent) error
}

// BlindFillStatusEmitter wraps status emission for BlindFill operations.
// This should be used by the caller (not the pii package) to emit status.
type BlindFillStatusEmitter struct {
	agentID       string
	statusEmitter StatusEmitter
}

// NewBlindFillStatusEmitter creates a new status emitter for BlindFill operations.
func NewBlindFillStatusEmitter(agentID string, emitter StatusEmitter) *BlindFillStatusEmitter {
	return &BlindFillStatusEmitter{
		agentID:       agentID,
		statusEmitter: emitter,
	}
}

// EmitResolvingPII emits status when PII resolution starts
func (e *BlindFillStatusEmitter) EmitResolvingPII(ctx context.Context, skillName string, fieldCount int) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusFormFilling,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:     "resolving_pii",
			Progress: 10,
			TaskType: skillName,
		},
	})
}

// EmitPIIResolved emits status when PII resolution completes
func (e *BlindFillStatusEmitter) EmitPIIResolved(ctx context.Context, grantedCount, deniedCount int) error {
	if e.statusEmitter == nil {
		return nil
	}

	progress := 50
	if grantedCount > 0 {
		progress = 80
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusFormFilling,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:     "pii_resolved",
			Progress: progress,
		},
	})
}

// EmitPIIError emits status when PII resolution fails
func (e *BlindFillStatusEmitter) EmitPIIError(ctx context.Context, errMsg string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusError,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:  "resolving_pii",
			Error: errMsg,
		},
	})
}

// EmitAwaitingApproval emits status when waiting for PII approval
func (e *BlindFillStatusEmitter) EmitAwaitingApproval(ctx context.Context, fields []string, requestID string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusAwaitingApproval,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:            "awaiting_pii_approval",
			Progress:        30,
			TaskID:          requestID,
			FieldsRequested: fields,
		},
	})
}

// EmitApprovalGranted emits status when PII approval is granted
func (e *BlindFillStatusEmitter) EmitApprovalGranted(ctx context.Context, requestID string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusFormFilling,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:     "approval_granted",
			Progress: 50,
			TaskID:   requestID,
		},
	})
}

// EmitApprovalRejected emits status when PII approval is rejected
func (e *BlindFillStatusEmitter) EmitApprovalRejected(ctx context.Context, requestID, reason string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusError,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:   "approval_rejected",
			Error:  fmt.Sprintf("PII access rejected: %s", reason),
			TaskID: requestID,
		},
	})
}

// BrowserStatusEmitter wraps status emission for Browser operations.
type BrowserStatusEmitter struct {
	agentID       string
	statusEmitter StatusEmitter
}

// NewBrowserStatusEmitter creates a new status emitter for Browser operations.
func NewBrowserStatusEmitter(agentID string, emitter StatusEmitter) *BrowserStatusEmitter {
	return &BrowserStatusEmitter{
		agentID:       agentID,
		statusEmitter: emitter,
	}
}

// EmitNavigating emits status when navigation starts
// If the state machine is offline, it will automatically initialize first
func (e *BrowserStatusEmitter) EmitNavigating(ctx context.Context, url string) error {
	if e.statusEmitter == nil {
		return nil
	}

	// First emit initializing status if we have an IntegrationEmitter
	// This handles the OFFLINE -> INITIALIZING -> BROWSING chain
	if adapter, ok := e.statusEmitter.(*IntegrationEmitter); ok && adapter.integration != nil {
		sm := adapter.integration.stateMachine
		if sm != nil && sm.Current() == StatusOffline {
			// Transition to initializing first
			if err := sm.Transition(StatusInitializing, StatusMetadata{}); err != nil {
				// Log but continue - might already be initialized
			}
		}
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:      url,
			Step:     "navigating",
			Progress: 10,
		},
	})
}

// EmitPageLoaded emits status when page loads
func (e *BrowserStatusEmitter) EmitPageLoaded(ctx context.Context, url string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusBrowsing,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:      url,
			Step:     "page_loaded",
			Progress: 100,
		},
	})
}

// EmitFormFilling emits status during form filling
func (e *BrowserStatusEmitter) EmitFormFilling(ctx context.Context, step string, progress int) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusFormFilling,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:     step,
			Progress: progress,
		},
	})
}

// EmitAwaitingCaptcha emits status when waiting for captcha
func (e *BrowserStatusEmitter) EmitAwaitingCaptcha(ctx context.Context, url string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusAwaitingCaptcha,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:  url,
			Step: "awaiting_captcha",
		},
	})
}

// EmitAwaiting2FA emits status when waiting for 2FA
func (e *BrowserStatusEmitter) EmitAwaiting2FA(ctx context.Context, url string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusAwaiting2FA,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:  url,
			Step: "awaiting_2fa",
		},
	})
}

// EmitProcessingPayment emits status during payment processing
func (e *BrowserStatusEmitter) EmitProcessingPayment(ctx context.Context) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusProcessingPayment,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:     "processing_payment",
			Progress: 90,
		},
	})
}

// EmitComplete emits status when task completes
func (e *BrowserStatusEmitter) EmitComplete(ctx context.Context, url string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusComplete,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			URL:      url,
			Step:     "complete",
			Progress: 100,
		},
	})
}

// EmitError emits status when an error occurs
func (e *BrowserStatusEmitter) EmitError(ctx context.Context, step, errMsg string) error {
	if e.statusEmitter == nil {
		return nil
	}

	return e.statusEmitter.EmitStatus(ctx, StatusEvent{
		AgentID:   e.agentID,
		Status:    StatusError,
		Timestamp: time.Now().UnixMilli(),
		Metadata: StatusMetadata{
			Step:  step,
			Error: errMsg,
		},
	})
}

// IntegrationEmitter adapts Integration to StatusEmitter interface
type IntegrationEmitter struct {
	integration *Integration
}

// NewIntegrationEmitter creates an adapter from Integration
func NewIntegrationEmitter(integration *Integration) *IntegrationEmitter {
	return &IntegrationEmitter{
		integration: integration,
	}
}

// EmitStatus implements StatusEmitter interface
func (e *IntegrationEmitter) EmitStatus(ctx context.Context, event StatusEvent) error {
	if e.integration == nil {
		return nil
	}

	// Handle different statuses
	// Note: We use ForceTransition-compatible methods to allow same-state updates
	switch event.Status {
	case StatusBrowsing:
		// StartBrowsing uses ForceTransition internally for same-state updates
		return e.integration.StartBrowsing(event.Metadata.URL)
	case StatusFormFilling:
		// UpdateProgress uses ForceTransition for metadata updates
		return e.integration.UpdateProgress(event.Metadata.Step, event.Metadata.Progress)
	case StatusAwaitingCaptcha:
		return e.integration.WaitForCaptcha(ctx)
	case StatusAwaiting2FA:
		return e.integration.WaitFor2FA()
	case StatusAwaitingApproval:
		// Approval waiting is handled by RequestPIIAccess
		return nil
	case StatusProcessingPayment:
		return e.integration.StartPayment()
	case StatusComplete:
		return e.integration.CompleteTask()
	case StatusError:
		return e.integration.FailTask(fmt.Errorf("%s", event.Metadata.Error))
	case StatusIdle:
		// Allow transitioning to idle
		return e.integration.stateMachine.Transition(StatusIdle, StatusMetadata{})
	}

	return nil
}

// CreateBlindFillEmitter creates a BlindFillStatusEmitter from an Integration
func CreateBlindFillEmitter(agentID string, integration *Integration) *BlindFillStatusEmitter {
	adapter := NewIntegrationEmitter(integration)
	return NewBlindFillStatusEmitter(agentID, adapter)
}

// CreateBrowserEmitter creates a BrowserStatusEmitter from an Integration
func CreateBrowserEmitter(agentID string, integration *Integration) *BrowserStatusEmitter {
	adapter := NewIntegrationEmitter(integration)
	return NewBrowserStatusEmitter(agentID, adapter)
}
