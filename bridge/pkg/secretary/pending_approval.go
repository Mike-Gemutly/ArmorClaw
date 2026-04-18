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

//=============================================================================
// PendingApprovalManager
//=============================================================================

type piiResponse struct {
	Approved bool
	Fields   []string
}

// PendingApprovalManager manages PII approval lifecycle.
// Encapsulates what were previously package-level globals.
type PendingApprovalManager struct {
	timeout time.Duration
	alertFn func(roomID, stepID string, fields []string)
	pending map[string]chan piiResponse
	mu      sync.Mutex
}

// NewPendingApprovalManager creates a manager with default timeout.
func NewPendingApprovalManager() *PendingApprovalManager {
	return &PendingApprovalManager{
		timeout: DefaultPIIApprovalTimeout,
		pending: make(map[string]chan piiResponse),
	}
}

// SetTimeout configures the PII approval timeout, clamped to MaxPIIApprovalTimeout.
func (m *PendingApprovalManager) SetTimeout(d time.Duration) {
	if d <= 0 {
		m.timeout = DefaultPIIApprovalTimeout
		return
	}
	if d > MaxPIIApprovalTimeout {
		slog.Warn("PII approval timeout exceeds maximum, clamping",
			"requested", d, "maximum", MaxPIIApprovalTimeout)
		m.timeout = MaxPIIApprovalTimeout
		return
	}
	m.timeout = d
}

// SetAlertDispatcher registers the callback for push notifications.
func (m *PendingApprovalManager) SetAlertDispatcher(fn func(roomID, stepID string, fields []string)) {
	m.alertFn = fn
}

// RequestApproval emits a PII approval request via the Matrix event bus and
// blocks until a matching response is received, the context is cancelled, or
// the configured timeout expires.
//
// On approval it returns the approved field list. On denial or timeout it
// returns an error.
func (m *PendingApprovalManager) RequestApproval(ctx context.Context, eventBus *events.MatrixEventBus, roomID, stepID string, requiredFields []string) ([]string, error) {
	responseCh := make(chan piiResponse, 1)

	// Register the pending approval so HandleResponse can deliver the result.
	m.mu.Lock()
	m.pending[stepID] = responseCh
	m.mu.Unlock()

	// Ensure cleanup on any exit path.
	defer func() {
		m.mu.Lock()
		delete(m.pending, stepID)
		m.mu.Unlock()
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
	if m.alertFn != nil {
		m.alertFn(roomID, stepID, requiredFields)
	}

	// Block until response, timeout, or cancellation.
	select {
	case resp := <-responseCh:
		if !resp.Approved {
			return nil, fmt.Errorf("PII approval denied for step %s", stepID)
		}
		return resp.Fields, nil

	case <-time.After(m.timeout):
		return nil, fmt.Errorf("PII approval timed out for step %s after %v", stepID, m.timeout)

	case <-ctx.Done():
		return nil, fmt.Errorf("PII approval cancelled for step %s: %w", stepID, ctx.Err())
	}
}

// HandleResponse delivers a PII approval response to the goroutine blocked
// in RequestApproval. It is called by the RPC handler when an
// app.armorclaw.pii_response Matrix event arrives from the client.
func (m *PendingApprovalManager) HandleResponse(stepID string, approved bool, fields []string) {
	m.mu.Lock()
	ch, exists := m.pending[stepID]
	m.mu.Unlock()

	if !exists {
		return
	}

	select {
	case ch <- piiResponse{Approved: approved, Fields: fields}:
	default:
		// Channel full or already satisfied — discard.
	}
}

//=============================================================================
// Backward-Compatible Package-Level Functions
//=============================================================================

// defaultManager is the default instance used by package-level functions.
// This preserves backward compatibility during migration.
var defaultManager = NewPendingApprovalManager()

// PendingApproval is the backward-compatible wrapper around defaultManager.RequestApproval.
func PendingApproval(ctx context.Context, eventBus *events.MatrixEventBus, roomID, stepID string, requiredFields []string) ([]string, error) {
	return defaultManager.RequestApproval(ctx, eventBus, roomID, stepID, requiredFields)
}

// HandlePIIResponse is the backward-compatible wrapper around defaultManager.HandleResponse.
func HandlePIIResponse(stepID string, approved bool, fields []string) {
	defaultManager.HandleResponse(stepID, approved, fields)
}

// SetPIIAlertDispatcher is the backward-compatible wrapper around defaultManager.SetAlertDispatcher.
func SetPIIAlertDispatcher(fn func(roomID, stepID string, fields []string)) {
	defaultManager.SetAlertDispatcher(fn)
}

// SetApprovalTimeout is the backward-compatible wrapper around defaultManager.SetTimeout.
func SetApprovalTimeout(d time.Duration) {
	defaultManager.SetTimeout(d)
}

// ApprovalTimeout returns the current timeout from the defaultManager.
// Kept for backward compatibility with code that reads the global.
func ApprovalTimeout() time.Duration {
	return defaultManager.timeout
}
