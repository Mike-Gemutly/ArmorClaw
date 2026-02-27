// Package keystore provides PII request lifecycle management.
// This module handles the "Secretary" flow where agents request sensitive data
// and mobile users approve/deny access.
package keystore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

// PIIRequestStatus represents the lifecycle state of a PII request
type PIIRequestStatus string

const (
	// StatusPending means the request is awaiting user approval
	StatusPending PIIRequestStatus = "pending"
	// StatusApproved means the user approved access to specific fields
	StatusApproved PIIRequestStatus = "approved"
	// StatusDenied means the user denied the request
	StatusDenied PIIRequestStatus = "denied"
	// StatusExpired means the request timed out before approval
	StatusExpired PIIRequestStatus = "expired"
	// StatusCancelled means the agent cancelled the request
	StatusCancelled PIIRequestStatus = "cancelled"
	// StatusFulfilled means the approved data was delivered to the agent
	StatusFulfilled PIIRequestStatus = "fulfilled"
)

// Error definitions
var (
	ErrRequestNotFound      = errors.New("PII request not found")
	ErrRequestExpired       = errors.New("PII request has expired")
	ErrRequestAlreadyClosed = errors.New("PII request already closed")
	ErrNoApprovedFields     = errors.New("no fields were approved")
	ErrAgentNotPaused       = errors.New("agent is not in paused state")
)

// PIIRequest represents a pending request for PII access
type PIIRequest struct {
	// Unique identifier for this request
	ID string `json:"id"`

	// Agent making the request
	AgentID string `json:"agent_id"`

	// Skill/task requesting access
	SkillID   string `json:"skill_id"`
	SkillName string `json:"skill_name"`

	// Target profile containing the PII
	ProfileID string `json:"profile_id"`

	// Fields being requested with their metadata
	RequestedFields []PIIFieldRequest `json:"requested_fields"`

	// Context/reason for the request (shown to user)
	Context string `json:"context"`

	// Matrix room to emit approval events to
	RoomID string `json:"room_id"`

	// Current status
	Status PIIRequestStatus `json:"status"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`

	// Approval details (set when approved)
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	ApprovedBy    string     `json:"approved_by,omitempty"`
	ApprovedFields []string  `json:"approved_fields,omitempty"`

	// Denial details (set when denied)
	DeniedAt    *time.Time `json:"denied_at,omitempty"`
	DeniedBy    string     `json:"denied_by,omitempty"`
	DenyReason  string     `json:"deny_reason,omitempty"`

	// Fulfilled details (set when data delivered)
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`

	// Resolved variables (only available after approval)
	resolvedVariables map[string]string
}

// PIIFieldRequest represents a single field request
type PIIFieldRequest struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"` // PCI/PII flag
}

// IsExpired checks if the request has expired
func (r *PIIRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsClosed checks if the request is in a terminal state
func (r *PIIRequest) IsClosed() bool {
	return r.Status == StatusApproved ||
		r.Status == StatusDenied ||
		r.Status == StatusExpired ||
		r.Status == StatusCancelled ||
		r.Status == StatusFulfilled
}

// ToMatrixEvent converts the request to a Matrix event payload
func (r *PIIRequest) ToMatrixEvent() map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(r.RequestedFields))
	for _, f := range r.RequestedFields {
		fields = append(fields, map[string]interface{}{
			"key":          f.Key,
			"display_name": f.DisplayName,
			"required":     f.Required,
			"sensitive":    f.Sensitive,
		})
	}

	return map[string]interface{}{
		"request_id":      r.ID,
		"agent_id":        r.AgentID,
		"skill_id":        r.SkillID,
		"skill_name":      r.SkillName,
		"profile_id":      r.ProfileID,
		"requested_fields": fields,
		"context":         r.Context,
		"status":          string(r.Status),
		"created_at":      r.CreatedAt.UnixMilli(),
		"expires_at":      r.ExpiresAt.UnixMilli(),
	}
}

// PIIRequestManager manages the lifecycle of PII access requests
type PIIRequestManager struct {
	mu       sync.RWMutex
	requests map[string]*PIIRequest
	log      *logger.Logger
	counter  int64 // Counter for unique ID generation

	// Callbacks for integration
	onRequestCreated   func(ctx context.Context, req *PIIRequest) error
	onRequestApproved  func(ctx context.Context, req *PIIRequest) error
	onRequestDenied    func(ctx context.Context, req *PIIRequest) error
	onRequestExpired   func(ctx context.Context, req *PIIRequest) error
}

// PIIRequestManagerConfig holds configuration for the manager
type PIIRequestManagerConfig struct {
	// Default TTL for requests
	DefaultTTL time.Duration
	// Logger for audit trail
	Logger *logger.Logger
}

// NewPIIRequestManager creates a new request manager
func NewPIIRequestManager(cfg PIIRequestManagerConfig) *PIIRequestManager {
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("pii_request")
	}

	return &PIIRequestManager{
		requests: make(map[string]*PIIRequest),
		log:      log,
	}
}

// SetCallbacks sets the callback functions for request lifecycle events
func (m *PIIRequestManager) SetCallbacks(
	onCreated func(ctx context.Context, req *PIIRequest) error,
	onApproved func(ctx context.Context, req *PIIRequest) error,
	onDenied func(ctx context.Context, req *PIIRequest) error,
	onExpired func(ctx context.Context, req *PIIRequest) error,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRequestCreated = onCreated
	m.onRequestApproved = onApproved
	m.onRequestDenied = onDenied
	m.onRequestExpired = onExpired
}

// CreateRequest creates a new PII access request
func (m *PIIRequestManager) CreateRequest(ctx context.Context,
	agentID, skillID, skillName, profileID string,
	fields []PIIFieldRequest,
	context string,
	roomID string,
	ttl time.Duration,
) (*PIIRequest, error) {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	m.mu.Lock()
	m.counter++
	counter := m.counter
	m.mu.Unlock()

	now := time.Now()
	req := &PIIRequest{
		ID:              generatePIIRequestID(agentID, skillID, now, counter),
		AgentID:         agentID,
		SkillID:         skillID,
		SkillName:       skillName,
		ProfileID:       profileID,
		RequestedFields: fields,
		Context:         context,
		RoomID:          roomID,
		Status:          StatusPending,
		CreatedAt:       now,
		ExpiresAt:       now.Add(ttl),
	}

	m.mu.Lock()
	m.requests[req.ID] = req
	callback := m.onRequestCreated
	m.mu.Unlock()

	m.log.Info("pii_request_created",
		"request_id", req.ID,
		"agent_id", agentID,
		"skill_id", skillID,
		"profile_id", profileID,
		"fields_count", len(fields),
		"expires_at", req.ExpiresAt,
	)

	// Notify callback (e.g., emit Matrix event)
	if callback != nil {
		if err := callback(ctx, req); err != nil {
			m.log.Error("pii_request_callback_failed",
				"request_id", req.ID,
				"error", err.Error(),
			)
		}
	}

	return req, nil
}

// GetRequest retrieves a request by ID
func (m *PIIRequestManager) GetRequest(id string) (*PIIRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, exists := m.requests[id]
	if !exists {
		return nil, ErrRequestNotFound
	}
	return req, nil
}

// ApproveRequest approves a PII request with specific fields
func (m *PIIRequestManager) ApproveRequest(ctx context.Context,
	requestID string,
	userID string,
	approvedFields []string,
) (*PIIRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}

	if req.IsClosed() {
		return nil, ErrRequestAlreadyClosed
	}

	if req.IsExpired() {
		req.Status = StatusExpired
		return nil, ErrRequestExpired
	}

	now := time.Now()
	req.Status = StatusApproved
	req.ApprovedAt = &now
	req.ApprovedBy = userID
	req.ApprovedFields = approvedFields

	callback := m.onRequestApproved

	m.log.Info("pii_request_approved",
		"request_id", requestID,
		"approved_by", userID,
		"approved_fields", approvedFields,
	)

	// Notify callback (e.g., resume agent)
	if callback != nil {
		if err := callback(ctx, req); err != nil {
			m.log.Error("pii_approve_callback_failed",
				"request_id", requestID,
				"error", err.Error(),
			)
		}
	}

	return req, nil
}

// DenyRequest denies a PII request
func (m *PIIRequestManager) DenyRequest(ctx context.Context,
	requestID string,
	userID string,
	reason string,
) (*PIIRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}

	if req.IsClosed() {
		return nil, ErrRequestAlreadyClosed
	}

	if req.IsExpired() {
		req.Status = StatusExpired
		return nil, ErrRequestExpired
	}

	now := time.Now()
	req.Status = StatusDenied
	req.DeniedAt = &now
	req.DeniedBy = userID
	req.DenyReason = reason

	callback := m.onRequestDenied

	m.log.Info("pii_request_denied",
		"request_id", requestID,
		"denied_by", userID,
		"reason", reason,
	)

	// Notify callback (e.g., cancel agent task)
	if callback != nil {
		if err := callback(ctx, req); err != nil {
			m.log.Error("pii_deny_callback_failed",
				"request_id", requestID,
				"error", err.Error(),
			)
		}
	}

	return req, nil
}

// CancelRequest cancels a pending request (agent-initiated)
func (m *PIIRequestManager) CancelRequest(ctx context.Context, requestID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return ErrRequestNotFound
	}

	if req.IsClosed() {
		return ErrRequestAlreadyClosed
	}

	req.Status = StatusCancelled

	m.log.Info("pii_request_cancelled",
		"request_id", requestID,
	)

	return nil
}

// FulfillRequest marks a request as fulfilled (data delivered to agent)
func (m *PIIRequestManager) FulfillRequest(ctx context.Context,
	requestID string,
	resolvedVars map[string]string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return ErrRequestNotFound
	}

	if req.Status != StatusApproved {
		return fmt.Errorf("request must be approved before fulfillment, current status: %s", req.Status)
	}

	now := time.Now()
	req.Status = StatusFulfilled
	req.FulfilledAt = &now
	req.resolvedVariables = resolvedVars

	m.log.Info("pii_request_fulfilled",
		"request_id", requestID,
		"fields_count", len(resolvedVars),
	)

	return nil
}

// ListPending returns all pending requests
func (m *PIIRequestManager) ListPending() []*PIIRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending := make([]*PIIRequest, 0)
	for _, req := range m.requests {
		if req.Status == StatusPending && !req.IsExpired() {
			pending = append(pending, req)
		}
	}
	return pending
}

// ListByAgent returns all requests for a specific agent
func (m *PIIRequestManager) ListByAgent(agentID string) []*PIIRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]*PIIRequest, 0)
	for _, req := range m.requests {
		if req.AgentID == agentID {
			requests = append(requests, req)
		}
	}
	return requests
}

// CleanupExpired removes expired requests
func (m *PIIRequestManager) CleanupExpired(ctx context.Context) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0
	callback := m.onRequestExpired

	for id, req := range m.requests {
		if req.Status == StatusPending && now.After(req.ExpiresAt) {
			req.Status = StatusExpired

			m.log.Info("pii_request_expired",
				"request_id", id,
			)

			// Notify callback
			if callback != nil {
				_ = callback(ctx, req)
			}

			// Remove from active list but keep for audit
			delete(m.requests, id)
			removed++
		}
	}

	return removed
}

// GetStats returns statistics about PII requests
func (m *PIIRequestManager) GetStats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     len(m.requests),
		"pending":   0,
		"approved":  0,
		"denied":    0,
		"expired":   0,
		"cancelled": 0,
		"fulfilled": 0,
	}

	for _, req := range m.requests {
		switch req.Status {
		case StatusPending:
			stats["pending"]++
		case StatusApproved:
			stats["approved"]++
		case StatusDenied:
			stats["denied"]++
		case StatusExpired:
			stats["expired"]++
		case StatusCancelled:
			stats["cancelled"]++
		case StatusFulfilled:
			stats["fulfilled"]++
		}
	}

	return stats
}

// generatePIIRequestID generates a unique ID for a PII request
func generatePIIRequestID(agentID, skillID string, timestamp time.Time, counter int64) string {
	// Include counter for guaranteed uniqueness
	data := fmt.Sprintf("%s:%s:%d:%d:%s", agentID, skillID, timestamp.UnixNano(), counter, randomHex(8))
	hash := sha256.Sum256([]byte(data))
	return "pii_" + hex.EncodeToString(hash[:12])
}

// randomHex generates a random hex string of the specified byte length
func randomHex(n int) string {
	// Use a mix of timestamp and process-specific data for uniqueness
	data := fmt.Sprintf("%d:%d:%s", time.Now().UnixNano(), time.Now().Nanosecond(), "pii_salt")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:n])
}

// MarshalJSON custom marshals PIIRequest for API responses
func (r *PIIRequest) MarshalJSON() ([]byte, error) {
	type Alias PIIRequest
	return json.Marshal(&struct {
		*Alias
		CreatedAt    int64  `json:"created_at"`
		ExpiresAt    int64  `json:"expires_at"`
		ApprovedAt   *int64 `json:"approved_at,omitempty"`
		DeniedAt     *int64 `json:"denied_at,omitempty"`
		FulfilledAt  *int64 `json:"fulfilled_at,omitempty"`
	}{
		Alias:       (*Alias)(r),
		CreatedAt:   r.CreatedAt.UnixMilli(),
		ExpiresAt:   r.ExpiresAt.UnixMilli(),
		ApprovedAt:  timeToMillis(r.ApprovedAt),
		DeniedAt:    timeToMillis(r.DeniedAt),
		FulfilledAt: timeToMillis(r.FulfilledAt),
	})
}

func timeToMillis(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ms := t.UnixMilli()
	return &ms
}
