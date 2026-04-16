package secretary

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/events"
)

//=============================================================================
// PII Approval Event Types
//=============================================================================

const (
	PIIRequestEventType  = "app.armorclaw.pii_request"
	PIIResponseEventType = "app.armorclaw.pii_response"

	// DefaultPIIApprovalTimeout is the fallback when no config is provided.
	DefaultPIIApprovalTimeout = 120 * time.Second

	// MaxPIIApprovalTimeout is the hard upper bound (15 minutes).
	MaxPIIApprovalTimeout = 900 * time.Second
)

// ApprovalTimeout holds the configured timeout duration for PII approvals.
// It can be set via Bridge TOML [secretary] block. Defaults to 120s, max 900s.
var ApprovalTimeout = DefaultPIIApprovalTimeout

// piiAlertDispatcher is called after publishing a PII request event to trigger
// push notifications. Set via SetPIIAlertDispatcher during Bridge startup.
var piiAlertDispatcher func(roomID, stepID string, fields []string)

// SetPIIAlertDispatcher registers the callback that fires a push notification
// immediately after a PII approval request is published.
func SetPIIAlertDispatcher(fn func(roomID, stepID string, fields []string)) {
	piiAlertDispatcher = fn
}

// SetApprovalTimeout configures the PII approval timeout, clamped to MaxPIIApprovalTimeout.
func SetApprovalTimeout(d time.Duration) {
	if d <= 0 {
		ApprovalTimeout = DefaultPIIApprovalTimeout
		return
	}
	if d > MaxPIIApprovalTimeout {
		slog.Warn("PII approval timeout exceeds maximum, clamping",
			"requested", d, "maximum", MaxPIIApprovalTimeout)
		ApprovalTimeout = MaxPIIApprovalTimeout
		return
	}
	ApprovalTimeout = d
}

//=============================================================================
// Pending Approval Registry
//=============================================================================

type piiResponse struct {
	Approved bool
	Fields   []string
}

var (
	pendingMut  sync.Mutex
	pendingApps = make(map[string]chan piiResponse)
)

//=============================================================================
// PendingApproval
//=============================================================================

// PendingApproval emits a PII approval request via the Matrix event bus and
// blocks until a matching response is received, the context is cancelled, or
// the 120-second timeout expires.
//
// On approval it returns the approved field list. On denial or timeout it
// returns an error.
func PendingApproval(ctx context.Context, eventBus *events.MatrixEventBus, roomID, stepID string, requiredFields []string) ([]string, error) {
	responseCh := make(chan piiResponse, 1)

	// Register the pending approval so HandlePIIResponse can deliver the result.
	pendingMut.Lock()
	pendingApps[stepID] = responseCh
	pendingMut.Unlock()

	// Ensure cleanup on any exit path.
	defer func() {
		pendingMut.Lock()
		delete(pendingApps, stepID)
		pendingMut.Unlock()
	}()

	// Build and publish the PII request event.
	payload := map[string]interface{}{
		"step_id":         stepID,
		"required_fields": requiredFields,
		"timestamp":       time.Now().UnixMilli(),
	}

	eventBus.Publish(events.MatrixEvent{
		ID:      fmt.Sprintf("%s-%s-%d", PIIRequestEventType, stepID, time.Now().UnixNano()),
		RoomID:  roomID,
		Sender:  "orchestrator",
		Type:    PIIRequestEventType,
		Content: payload,
	})

	// Fire push notification so the user sees the approval request immediately,
	// even if they're not actively watching the Matrix room.
	if piiAlertDispatcher != nil {
		piiAlertDispatcher(roomID, stepID, requiredFields)
	}

	// Block until response, timeout, or cancellation.
	select {
	case resp := <-responseCh:
		if !resp.Approved {
			return nil, fmt.Errorf("PII approval denied for step %s", stepID)
		}
		return resp.Fields, nil

	case <-time.After(ApprovalTimeout):
		return nil, fmt.Errorf("PII approval timed out for step %s after %v", stepID, ApprovalTimeout)

	case <-ctx.Done():
		return nil, fmt.Errorf("PII approval cancelled for step %s: %w", stepID, ctx.Err())
	}
}

//=============================================================================
// HandlePIIResponse
//=============================================================================

// HandlePIIResponse delivers a PII approval response to the goroutine blocked
// in PendingApproval. It is called by the RPC handler when an
// app.armorclaw.pii_response Matrix event arrives from the client.
func HandlePIIResponse(stepID string, approved bool, fields []string) {
	pendingMut.Lock()
	ch, exists := pendingApps[stepID]
	pendingMut.Unlock()

	if !exists {
		return
	}

	select {
	case ch <- piiResponse{Approved: approved, Fields: fields}:
	default:
		// Channel full or already satisfied — discard.
	}
}
