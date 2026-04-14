package secretary

import (
	"context"
	"fmt"
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

	// piiApprovalTimeout is the maximum time to wait for a PII approval response.
	piiApprovalTimeout = 120 * time.Second
)

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

	// Block until response, timeout, or cancellation.
	select {
	case resp := <-responseCh:
		if !resp.Approved {
			return nil, fmt.Errorf("PII approval denied for step %s", stepID)
		}
		return resp.Fields, nil

	case <-time.After(piiApprovalTimeout):
		return nil, fmt.Errorf("PII approval timed out for step %s after %v", stepID, piiApprovalTimeout)

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
