// Package pii provides Human-in-the-Loop consent management for PII access.
// Skills must request user approval before accessing PII fields.
package pii

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/securerandom"
)

var (
	// ErrRequestNotFound is returned when a request doesn't exist
	ErrRequestNotFound = errors.New("access request not found")

	// ErrRequestExpired is returned when a request has expired
	ErrRequestExpired = errors.New("access request has expired")

	// ErrRequestAlreadyProcessed is returned when trying to process an already handled request
	ErrRequestAlreadyProcessed = errors.New("access request already processed")

	// ErrApprovalTimeout is returned when the approval timeout is exceeded
	ErrApprovalTimeout = errors.New("approval timeout exceeded")
)

// DefaultApprovalTimeout is the default time to wait for user approval
const DefaultApprovalTimeout = 60 * time.Second

// AccessRequest represents a pending PII access request
type AccessRequest struct {
	// ID is the unique identifier for this request
	ID string `json:"id"`

	// SkillID is the skill requesting access
	SkillID string `json:"skill_id"`

	// SkillName is the human-readable skill name
	SkillName string `json:"skill_name"`

	// ProfileID is the profile being accessed
	ProfileID string `json:"profile_id"`

	// RequestedFields is the list of fields being requested
	RequestedFields []string `json:"requested_fields"`

	// RequiredFields is the list of fields that are required
	RequiredFields []string `json:"required_fields"`

	// Status is the current status of the request
	Status AccessRequestStatus `json:"status"`

	// CreatedAt is when the request was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the request expires
	ExpiresAt time.Time `json:"expires_at"`

	// ApprovedAt is when the request was approved (if applicable)
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// ApprovedBy is the user who approved (if applicable)
	ApprovedBy string `json:"approved_by,omitempty"`

	// ApprovedFields is the list of fields that were approved
	ApprovedFields []string `json:"approved_fields,omitempty"`

	// RejectedAt is when the request was rejected (if applicable)
	RejectedAt *time.Time `json:"rejected_at,omitempty"`

	// RejectedBy is the user who rejected (if applicable)
	RejectedBy string `json:"rejected_by,omitempty"`

	// RejectionReason is why the request was rejected
	RejectionReason string `json:"rejection_reason,omitempty"`

	// RoomID is the Matrix room where consent was requested
	RoomID string `json:"room_id,omitempty"`

	// EventID is the Matrix event ID of the consent request message
	EventID string `json:"event_id,omitempty"`

	// Manifest is the original skill manifest
	Manifest *SkillManifest `json:"manifest,omitempty"`

	// approvalChan is used to signal approval completion
	approvalChan chan approvalResult `json:"-"`

	// mu protects concurrent access
	mu sync.RWMutex `json:"-"`
}

// approvalResult carries the result of an approval decision
type approvalResult struct {
	approved      bool
	approvedBy    string
	approvedFields []string
	reason        string
}

// AccessRequestStatus represents the status of an access request
type AccessRequestStatus string

const (
	StatusPending  AccessRequestStatus = "pending"
	StatusApproved AccessRequestStatus = "approved"
	StatusRejected AccessRequestStatus = "rejected"
	StatusExpired  AccessRequestStatus = "expired"
)

// IsExpired checks if the request has expired
func (r *AccessRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsPending checks if the request is still pending
func (r *AccessRequest) IsPending() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Status == StatusPending
}

// HITLConsentManager manages Human-in-the-Loop consent for PII access.
// It coordinates consent requests between skills, the Matrix interface, and users.
type HITLConsentManager struct {
	requests     map[string]*AccessRequest
	mu           sync.RWMutex
	timeout      time.Duration
	auditLogger  *audit.CriticalOperationLogger
	securityLog  *logger.SecurityLogger
	log          *logger.Logger

	// Notification callback for sending consent requests to users
	notifyCallback func(ctx context.Context, request *AccessRequest) error
}

// HITLConfig configures the HITL consent manager
type HITLConfig struct {
	// Timeout is how long to wait for approval
	Timeout time.Duration

	// AuditLogger is the audit logger for consent events
	AuditLogger *audit.CriticalOperationLogger

	// SecurityLog is the security logger
	SecurityLog *logger.SecurityLogger

	// NotifyCallback is called to send consent request notifications
	NotifyCallback func(ctx context.Context, request *AccessRequest) error
}

// NewHITLConsentManager creates a new HITL consent manager
func NewHITLConsentManager(cfg HITLConfig) *HITLConsentManager {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultApprovalTimeout
	}

	return &HITLConsentManager{
		requests:      make(map[string]*AccessRequest),
		timeout:       cfg.Timeout,
		auditLogger:   cfg.AuditLogger,
		securityLog:   cfg.SecurityLog,
		log:           logger.Global().WithComponent("hitl_consent"),
		notifyCallback: cfg.NotifyCallback,
	}
}

// SetNotifyCallback sets the notification callback for consent requests
func (m *HITLConsentManager) SetNotifyCallback(callback func(ctx context.Context, request *AccessRequest) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifyCallback = callback
}

// RequestAccess creates a new PII access request and sends notification
func (m *HITLConsentManager) RequestAccess(
	ctx context.Context,
	manifest *SkillManifest,
	profileID string,
	roomID string,
) (*AccessRequest, error) {
	// Generate unique request ID
	requestID := GenerateRequestID()

	// Create the access request
	request := &AccessRequest{
		ID:              requestID,
		SkillID:         manifest.SkillID,
		SkillName:       manifest.SkillName,
		ProfileID:       profileID,
		RequestedFields: manifest.GetAllFieldKeys(),
		RequiredFields:  manifest.GetRequiredFields(),
		Status:          StatusPending,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(m.timeout),
		RoomID:          roomID,
		Manifest:        manifest,
		approvalChan:    make(chan approvalResult, 1),
	}

	// Store the request
	m.mu.Lock()
	m.requests[requestID] = request
	m.mu.Unlock()

	// Log the access request
	m.log.Info("pii_access_requested",
		"request_id", requestID,
		"skill_id", manifest.SkillID,
		"skill_name", manifest.SkillName,
		"profile_id", profileID,
		"requested_fields", request.RequestedFields,
		"required_fields", request.RequiredFields,
	)

	// Audit logging
	if m.auditLogger != nil {
		_ = m.auditLogger.LogPIIAccessRequest(ctx, requestID, manifest.SkillID, profileID, request.RequestedFields)
	}

	// Security logging
	if m.securityLog != nil {
		m.securityLog.LogPIIAccessRequest(ctx, requestID, manifest.SkillName, profileID, request.RequestedFields)
	}

	// Send notification via callback
	if m.notifyCallback != nil {
		if err := m.notifyCallback(ctx, request); err != nil {
			m.log.Error("consent_notification_failed",
				"request_id", requestID,
				"error", err.Error(),
			)
			// Don't fail the request - user can still approve via other means
		}
	}

	return request, nil
}

// WaitForApproval blocks until the request is approved, rejected, or expires
func (m *HITLConsentManager) WaitForApproval(ctx context.Context, requestID string) (*AccessRequest, error) {
	m.mu.RLock()
	request, exists := m.requests[requestID]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrRequestNotFound
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Wait for approval, rejection, or timeout
	select {
	case result := <-request.approvalChan:
		request.mu.Lock()
		defer request.mu.Unlock()

		if result.approved {
			request.Status = StatusApproved
			now := time.Now()
			request.ApprovedAt = &now
			request.ApprovedBy = result.approvedBy
			request.ApprovedFields = result.approvedFields

			m.log.Info("pii_access_approved",
				"request_id", requestID,
				"approved_by", result.approvedBy,
				"approved_fields", result.approvedFields,
			)

			if m.auditLogger != nil {
				_ = m.auditLogger.LogPIIAccessGranted(ctx, requestID, request.SkillID, result.approvedBy, result.approvedFields)
			}
		} else {
			request.Status = StatusRejected
			now := time.Now()
			request.RejectedAt = &now
			request.RejectedBy = result.approvedBy
			request.RejectionReason = result.reason

			m.log.Info("pii_access_rejected",
				"request_id", requestID,
				"rejected_by", result.approvedBy,
				"reason", result.reason,
			)

			if m.auditLogger != nil {
				_ = m.auditLogger.LogPIIAccessRejected(ctx, requestID, request.SkillID, result.approvedBy, result.reason)
			}
		}

		return request, nil

	case <-timeoutCtx.Done():
		request.mu.Lock()
		request.Status = StatusExpired
		request.mu.Unlock()

		m.log.Warn("pii_access_expired",
			"request_id", requestID,
		)

		if m.auditLogger != nil {
			_ = m.auditLogger.LogPIIAccessExpired(ctx, requestID, request.SkillID)
		}

		return nil, ErrApprovalTimeout

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ApproveRequest approves an access request with specific fields
func (m *HITLConsentManager) ApproveRequest(
	ctx context.Context,
	requestID string,
	userID string,
	approvedFields []string,
) error {
	m.mu.RLock()
	request, exists := m.requests[requestID]
	m.mu.RUnlock()

	if !exists {
		return ErrRequestNotFound
	}

	request.mu.Lock()
	defer request.mu.Unlock()

	// Check if already processed
	if request.Status != StatusPending {
		return ErrRequestAlreadyProcessed
	}

	// Check if expired
	if request.IsExpired() {
		request.Status = StatusExpired
		return ErrRequestExpired
	}

	// Validate all required fields are approved
	approvedSet := make(map[string]bool)
	for _, f := range approvedFields {
		approvedSet[f] = true
	}

	for _, req := range request.RequiredFields {
		if !approvedSet[req] {
			return fmt.Errorf("required field '%s' not approved", req)
		}
	}

	// Send approval result
	select {
	case request.approvalChan <- approvalResult{
		approved:       true,
		approvedBy:     userID,
		approvedFields: approvedFields,
	}:
	default:
		// Channel full - request might have been processed
		return ErrRequestAlreadyProcessed
	}

	return nil
}

// RejectRequest rejects an access request
func (m *HITLConsentManager) RejectRequest(
	ctx context.Context,
	requestID string,
	userID string,
	reason string,
) error {
	m.mu.RLock()
	request, exists := m.requests[requestID]
	m.mu.RUnlock()

	if !exists {
		return ErrRequestNotFound
	}

	request.mu.Lock()
	defer request.mu.Unlock()

	// Check if already processed
	if request.Status != StatusPending {
		return ErrRequestAlreadyProcessed
	}

	// Check if expired
	if request.IsExpired() {
		request.Status = StatusExpired
		return ErrRequestExpired
	}

	// Send rejection result
	select {
	case request.approvalChan <- approvalResult{
		approved: false,
		approvedBy: userID,
		reason:    reason,
	}:
	default:
		// Channel full - request might have been processed
		return ErrRequestAlreadyProcessed
	}

	return nil
}

// GetRequest retrieves a request by ID
func (m *HITLConsentManager) GetRequest(requestID string) (*AccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, exists := m.requests[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}

	return request, nil
}

// ListPendingRequests returns all pending requests
func (m *HITLConsentManager) ListPendingRequests() []*AccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*AccessRequest
	for _, req := range m.requests {
		req.mu.RLock()
		if req.Status == StatusPending && !req.IsExpired() {
			pending = append(pending, req)
		}
		req.mu.RUnlock()
	}

	return pending
}

// ListRequestsByProfile returns all requests for a specific profile
func (m *HITLConsentManager) ListRequestsByProfile(profileID string) []*AccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*AccessRequest
	for _, req := range m.requests {
		if req.ProfileID == profileID {
			results = append(results, req)
		}
	}

	return results
}

// CleanupExpired removes expired requests from memory
func (m *HITLConsentManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, req := range m.requests {
		req.mu.RLock()
		if req.Status != StatusPending || req.IsExpired() {
			delete(m.requests, id)
			count++
		}
		req.mu.RUnlock()
	}

	return count
}

// RequestAccessAndWait is a convenience method that requests access and waits for approval
func (m *HITLConsentManager) RequestAccessAndWait(
	ctx context.Context,
	manifest *SkillManifest,
	profileID string,
	roomID string,
) (*AccessRequest, error) {
	// Create the request
	request, err := m.RequestAccess(ctx, manifest, profileID, roomID)
	if err != nil {
		return nil, err
	}

	// Wait for approval
	return m.WaitForApproval(ctx, request.ID)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return "pii_req_" + securerandom.MustID(16)
}
