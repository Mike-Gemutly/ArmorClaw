// Package pii provides Three-Way Consent management for PII access.
// Three-way consent creates a Matrix room with user + agent + bridge participants
// for transparent approval/rejection of PII access requests.
package pii

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/securerandom"
)

var (
	// ErrConsentRoomExists is returned when a consent room already exists for a request
	ErrConsentRoomExists = errors.New("consent room already exists for this request")

	// ErrInvalidApprovalToken is returned when the approval token is invalid
	ErrInvalidApprovalToken = errors.New("invalid approval token")

	// ErrRoomNotFound is returned when a consent room is not found
	ErrRoomNotFound = errors.New("consent room not found")

	// ErrNotAuthorized is returned when the user is not authorized to approve/reject
	ErrNotAuthorized = errors.New("user not authorized for this consent request")

	// ErrRoomNotConsentRoom is returned when the room is not a consent room
	ErrRoomNotConsentRoom = errors.New("room is not a consent room")
)

type ConsentEventType string

const (
	ConsentEventRequest ConsentEventType = "request"
	ConsentEventApprove ConsentEventType = "approve"
	ConsentEventReject  ConsentEventType = "reject"
	ConsentEventExpire  ConsentEventType = "expire"
)

// ConsentRequestEventType is the Matrix event type for consent requests
const ConsentRequestEventType = "app.armorclaw.consent.request"

// ConsentResponseEventType is the Matrix event type for consent responses
const ConsentResponseEventType = "app.armorclaw.consent.response"

// ApprovalReaction is the reaction emoji for approval
const ApprovalReaction = "✅"

// RejectionReaction is the reaction emoji for rejection
const RejectionReaction = "❌"

// ConsentState represents the state of a three-way consent request
type ConsentState string

const (
	ConsentStatePending  ConsentState = "pending"
	ConsentStateApproved ConsentState = "approved"
	ConsentStateRejected ConsentState = "rejected"
	ConsentStateExpired  ConsentState = "expired"
)

// ConsentRoom represents a Matrix room created for three-way consent
type ConsentRoom struct {
	// ID is the unique identifier for this consent room record
	ID string `json:"id"`

	// RequestID is the PII access request ID
	RequestID string `json:"request_id"`

	// MatrixRoomID is the Matrix room ID
	MatrixRoomID string `json:"matrix_room_id"`

	// MatrixRoomAlias is the Matrix room alias (optional)
	MatrixRoomAlias string `json:"matrix_room_alias,omitempty"`

	// UserMatrixID is the user's Matrix ID
	UserMatrixID string `json:"user_matrix_id"`

	// AgentMatrixID is the agent's Matrix ID
	AgentMatrixID string `json:"agent_matrix_id"`

	// BridgeMatrixID is the bridge's Matrix ID
	BridgeMatrixID string `json:"bridge_matrix_id"`

	// ApprovalToken is a secure token for validating approval requests
	ApprovalToken string `json:"approval_token"`

	// State is the current consent state
	State ConsentState `json:"state"`

	// RequestEventID is the Matrix event ID of the consent request message
	RequestEventID string `json:"request_event_id,omitempty"`

	// ResponseEventID is the Matrix event ID of the approval/rejection message
	ResponseEventID string `json:"response_event_id,omitempty"`

	// ApprovedBy is the Matrix ID of who approved (if applicable)
	ApprovedBy string `json:"approved_by,omitempty"`

	// ApprovedFields are the fields that were approved
	ApprovedFields []string `json:"approved_fields,omitempty"`

	// RejectedBy is the Matrix ID of who rejected (if applicable)
	RejectedBy string `json:"rejected_by,omitempty"`

	// RejectionReason is the reason for rejection
	RejectionReason string `json:"rejection_reason,omitempty"`

	// CreatedAt is when the consent room was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the consent request expires
	ExpiresAt time.Time `json:"expires_at"`

	// RespondedAt is when the consent was responded to
	RespondedAt *time.Time `json:"responded_at,omitempty"`

	// mu protects concurrent access
	mu sync.RWMutex `json:"-"`
}

// IsExpired checks if the consent request has expired
func (r *ConsentRoom) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsPending checks if the consent request is still pending
func (r *ConsentRoom) IsPending() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.State == ConsentStatePending
}

// ConsentRequestEvent is the content of a consent request Matrix event
type ConsentRequestEvent struct {
	RequestID       string   `json:"request_id"`
	SkillID         string   `json:"skill_id"`
	SkillName       string   `json:"skill_name"`
	ProfileID       string   `json:"profile_id"`
	RequestedFields []string `json:"requested_fields"`
	RequiredFields  []string `json:"required_fields"`
	ApprovalToken   string   `json:"approval_token"`
	ExpiresAt       string   `json:"expires_at"`
}

// ConsentResponseEvent is the content of a consent response Matrix event
type ConsentResponseEvent struct {
	RequestID      string       `json:"request_id"`
	State          ConsentState `json:"state"`
	ApprovedBy     string       `json:"approved_by,omitempty"`
	ApprovedFields []string     `json:"approved_fields,omitempty"`
	RejectedBy     string       `json:"rejected_by,omitempty"`
	Reason         string       `json:"reason,omitempty"`
	Timestamp      string       `json:"timestamp"`
}


// MatrixAdapter interface for Matrix operations needed by three-way consent
type MatrixAdapter interface {
	CreateRoom(ctx context.Context, params map[string]interface{}) (roomID, roomAlias string, err error)
	InviteUser(ctx context.Context, roomID, userID, reason string) error
	SendMessage(roomID, message, msgType string) (string, error)
	SendEvent(roomID, eventType string, content []byte) error
	GetUserID() string
}

// ThreeWayConsentManager manages three-way consent via Matrix rooms
type ThreeWayConsentManager struct {
	rooms       map[string]*ConsentRoom // requestID -> ConsentRoom
	byRoomID    map[string]*ConsentRoom // matrixRoomID -> ConsentRoom
	byToken     map[string]*ConsentRoom // approvalToken -> ConsentRoom
	mu          sync.RWMutex
	hitlManager *HITLConsentManager
	matrix      MatrixAdapter
	auditLogger *audit.CriticalOperationLogger
	securityLog *logger.SecurityLogger
	log         *logger.Logger
	timeout     time.Duration
}

// ThreeWayConfig configures the three-way consent manager
type ThreeWayConfig struct {
	// HITLManager is the existing HITL consent manager to integrate with
	HITLManager *HITLConsentManager

	// Matrix is the Matrix adapter for room operations
	Matrix MatrixAdapter

	// AuditLogger is the audit logger for consent events
	AuditLogger *audit.CriticalOperationLogger

	// SecurityLog is the security logger
	SecurityLog *logger.SecurityLogger

	// Timeout is how long to wait for approval
	Timeout time.Duration
}

// NewThreeWayConsentManager creates a new three-way consent manager
func NewThreeWayConsentManager(cfg ThreeWayConfig) (*ThreeWayConsentManager, error) {
	if cfg.HITLManager == nil {
		return nil, errors.New("HITL manager is required")
	}
	if cfg.Matrix == nil {
		return nil, errors.New("Matrix adapter is required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultApprovalTimeout
	}

	m := &ThreeWayConsentManager{
		rooms:       make(map[string]*ConsentRoom),
		byRoomID:    make(map[string]*ConsentRoom),
		byToken:     make(map[string]*ConsentRoom),
		hitlManager: cfg.HITLManager,
		matrix:      cfg.Matrix,
		auditLogger: cfg.AuditLogger,
		securityLog: cfg.SecurityLog,
		log:         logger.Global().WithComponent("three_way_consent"),
		timeout:     cfg.Timeout,
	}

	// Set up notification callback on HITL manager
	cfg.HITLManager.SetNotifyCallback(m.NotifyConsentRequest)

	return m, nil
}

// CreateConsentRoom creates a Matrix room for three-way consent
func (m *ThreeWayConsentManager) CreateConsentRoom(
	ctx context.Context,
	request *AccessRequest,
	userMatrixID string,
	agentMatrixID string,
) (*ConsentRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if room already exists for this request
	if _, exists := m.rooms[request.ID]; exists {
		return nil, ErrConsentRoomExists
	}

	// Get bridge's Matrix ID
	bridgeMatrixID := m.matrix.GetUserID()

	// Generate secure approval token
	approvalToken := securerandom.MustToken(32)

	// Generate unique room ID suffix
	roomSuffix := securerandom.MustID(8)

	// Create room name and topic
	roomName := fmt.Sprintf("🔐 Consent: %s", request.SkillName)
	roomTopic := fmt.Sprintf("PII Access Request %s - Expires: %s",
		request.ID, request.ExpiresAt.Format(time.RFC3339))

	// Create Matrix room with invite list
	createParams := map[string]interface{}{
		"name":       roomName,
		"topic":      roomTopic,
		"preset":     "private_chat",
		"visibility": "private",
		"initial_state": []map[string]interface{}{
			{
				"type":      "m.room.guest_access",
				"state_key": "",
				"content": map[string]interface{}{
					"guest_access": "forbidden",
				},
			},
			{
				"type":      "m.room.history_visibility",
				"state_key": "",
				"content": map[string]interface{}{
					"history_visibility": "invited",
				},
			},
		},
		"invite":          []string{userMatrixID},
		"is_direct":       false,
		"room_alias_name": fmt.Sprintf("consent-%s", roomSuffix),
	}

	// Create the room
	roomID, roomAlias, err := m.matrix.CreateRoom(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Matrix room: %w", err)
	}

	// Invite agent if different from user and bridge
	if agentMatrixID != "" && agentMatrixID != userMatrixID && agentMatrixID != bridgeMatrixID {
		if err := m.matrix.InviteUser(ctx, roomID, agentMatrixID, "PII Consent Request"); err != nil {
			m.log.Warn("failed_to_invite_agent", "error", err, "agent_id", agentMatrixID)
			// Don't fail - user can still approve
		}
	}

	// Create consent room record
	consentRoom := &ConsentRoom{
		ID:              "consent_" + securerandom.MustID(8),
		RequestID:       request.ID,
		MatrixRoomID:    roomID,
		MatrixRoomAlias: roomAlias,
		UserMatrixID:    userMatrixID,
		AgentMatrixID:   agentMatrixID,
		BridgeMatrixID:  bridgeMatrixID,
		ApprovalToken:   approvalToken,
		State:           ConsentStatePending,
		CreatedAt:       time.Now(),
		ExpiresAt:       request.ExpiresAt,
	}

	// Send consent request event to room
	eventContent := ConsentRequestEvent{
		RequestID:       request.ID,
		SkillID:         request.SkillID,
		SkillName:       request.SkillName,
		ProfileID:       request.ProfileID,
		RequestedFields: request.RequestedFields,
		RequiredFields:  request.RequiredFields,
		ApprovalToken:   approvalToken,
		ExpiresAt:       request.ExpiresAt.Format(time.RFC3339),
	}

	eventBytes, err := json.Marshal(eventContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal consent event: %w", err)
	}

	if err := m.matrix.SendEvent(roomID, ConsentRequestEventType, eventBytes); err != nil {
		return nil, fmt.Errorf("failed to send consent event: %w", err)
	}

	// Send human-readable message with instructions
	message := m.formatConsentRequestMessage(request, approvalToken)
	eventID, err := m.matrix.SendMessage(roomID, message, "m.text")
	if err != nil {
		return nil, fmt.Errorf("failed to send consent message: %w", err)
	}

	consentRoom.RequestEventID = eventID

	// Store the room
	m.rooms[request.ID] = consentRoom
	m.byRoomID[roomID] = consentRoom
	m.byToken[approvalToken] = consentRoom

	// Log creation
	m.log.Info("consent_room_created",
		"request_id", request.ID,
		"room_id", roomID,
		"user_id", userMatrixID,
		"agent_id", agentMatrixID,
	)

	if m.auditLogger != nil {
		_ = m.auditLogger.LogPIIAccessRequest(ctx, request.ID, request.SkillID, request.ProfileID, request.RequestedFields)
	}

	return consentRoom, nil
}

// NotifyConsentRequest is the callback for HITL manager notifications
func (m *ThreeWayConsentManager) NotifyConsentRequest(ctx context.Context, request *AccessRequest) error {
	// This is called by HITL manager when a new request is created
	// The actual room creation is done separately via CreateConsentRoom
	// This callback can be used for additional notifications if needed
	m.log.Info("consent_request_notified",
		"request_id", request.ID,
		"skill_name", request.SkillName,
	)
	return nil
}

// HandleReaction handles a Matrix reaction event for approval/rejection
func (m *ThreeWayConsentManager) HandleReaction(
	ctx context.Context,
	roomID string,
	userMatrixID string,
	reactionKey string,
) error {
	m.mu.RLock()
	consentRoom, exists := m.byRoomID[roomID]
	m.mu.RUnlock()

	if !exists {
		return ErrRoomNotConsentRoom
	}

	consentRoom.mu.Lock()
	defer consentRoom.mu.Unlock()

	// Check if already processed
	if consentRoom.State != ConsentStatePending {
		return nil // Already processed, ignore
	}

	// Check if expired
	if consentRoom.IsExpired() {
		consentRoom.State = ConsentStateExpired
		return ErrRequestExpired
	}

	// Verify user is authorized (must be the user, not agent or bridge)
	if userMatrixID != consentRoom.UserMatrixID {
		return ErrNotAuthorized
	}

	// Handle reaction
	switch reactionKey {
	case ApprovalReaction:
		return m.approveInRoom(ctx, consentRoom, userMatrixID, consentRoom.RequestID)
	case RejectionReaction:
		return m.rejectInRoom(ctx, consentRoom, userMatrixID, "")
	default:
		// Unknown reaction, ignore
		return nil
	}
}

// HandleApprovalCommand handles an approval command from Matrix
func (m *ThreeWayConsentManager) HandleApprovalCommand(
	ctx context.Context,
	roomID string,
	userMatrixID string,
	approvalToken string,
	fields []string,
) error {
	m.mu.RLock()
	consentRoom, exists := m.byRoomID[roomID]
	m.mu.RUnlock()

	if !exists {
		return ErrRoomNotConsentRoom
	}

	consentRoom.mu.Lock()
	defer consentRoom.mu.Unlock()

	// Validate approval token
	if consentRoom.ApprovalToken != approvalToken {
		return ErrInvalidApprovalToken
	}

	// Check if already processed
	if consentRoom.State != ConsentStatePending {
		return ErrRequestAlreadyProcessed
	}

	// Check if expired
	if consentRoom.IsExpired() {
		consentRoom.State = ConsentStateExpired
		return ErrRequestExpired
	}

	// Verify user is authorized
	if userMatrixID != consentRoom.UserMatrixID {
		return ErrNotAuthorized
	}

	return m.approveInRoom(ctx, consentRoom, userMatrixID, consentRoom.RequestID)
}

// HandleRejectionCommand handles a rejection command from Matrix
func (m *ThreeWayConsentManager) HandleRejectionCommand(
	ctx context.Context,
	roomID string,
	userMatrixID string,
	approvalToken string,
	reason string,
) error {
	m.mu.RLock()
	consentRoom, exists := m.byRoomID[roomID]
	m.mu.RUnlock()

	if !exists {
		return ErrRoomNotConsentRoom
	}

	consentRoom.mu.Lock()
	defer consentRoom.mu.Unlock()

	// Validate approval token
	if consentRoom.ApprovalToken != approvalToken {
		return ErrInvalidApprovalToken
	}

	// Check if already processed
	if consentRoom.State != ConsentStatePending {
		return ErrRequestAlreadyProcessed
	}

	// Check if expired
	if consentRoom.IsExpired() {
		consentRoom.State = ConsentStateExpired
		return ErrRequestExpired
	}

	// Verify user is authorized
	if userMatrixID != consentRoom.UserMatrixID {
		return ErrNotAuthorized
	}

	return m.rejectInRoom(ctx, consentRoom, userMatrixID, reason)
}

// approveInRoom approves the consent request (must be called with lock held)
func (m *ThreeWayConsentManager) approveInRoom(
	ctx context.Context,
	consentRoom *ConsentRoom,
	userMatrixID string,
	requestID string,
) error {
	// Get the access request from HITL manager
	request, err := m.hitlManager.GetRequest(requestID)
	if err != nil {
		return fmt.Errorf("failed to get access request: %w", err)
	}

	// Default to all requested fields if none specified
	approvedFields := request.RequestedFields

	// Approve via HITL manager
	if err := m.hitlManager.ApproveRequest(ctx, requestID, userMatrixID, approvedFields); err != nil {
		return fmt.Errorf("failed to approve request: %w", err)
	}

	// Update consent room state
	now := time.Now()
	consentRoom.State = ConsentStateApproved
	consentRoom.ApprovedBy = userMatrixID
	consentRoom.ApprovedFields = approvedFields
	consentRoom.RespondedAt = &now

	// Send consent response event
	responseEvent := ConsentResponseEvent{
		RequestID:      requestID,
		State:          ConsentStateApproved,
		ApprovedBy:     userMatrixID,
		ApprovedFields: approvedFields,
		Timestamp:      now.Format(time.RFC3339),
	}

	eventBytes, err := json.Marshal(responseEvent)
	if err != nil {
		m.log.Warn("failed_to_marshal_response_event", "error", err)
	} else {
		_ = m.matrix.SendEvent(consentRoom.MatrixRoomID, ConsentResponseEventType, eventBytes)
	}

	// Send approval message
	message := fmt.Sprintf("✅ **Consent Approved**\n\nRequest `%s` has been approved by %s.\nApproved fields: %s",
		requestID, userMatrixID, formatFieldList(approvedFields))
	eventID, err := m.matrix.SendMessage(consentRoom.MatrixRoomID, message, "m.notice")
	if err != nil {
		m.log.Warn("failed_to_send_approval_message", "error", err)
	} else {
		consentRoom.ResponseEventID = eventID
	}

	// Log approval
	m.log.Info("consent_approved",
		"request_id", requestID,
		"approved_by", userMatrixID,
		"approved_fields", approvedFields,
	)

	if m.auditLogger != nil {
		_ = m.auditLogger.LogPIIAccessGranted(ctx, requestID, request.SkillID, userMatrixID, approvedFields)
	}

	return nil
}

// rejectInRoom rejects the consent request (must be called with lock held)
func (m *ThreeWayConsentManager) rejectInRoom(
	ctx context.Context,
	consentRoom *ConsentRoom,
	userMatrixID string,
	reason string,
) error {
	requestID := consentRoom.RequestID

	// Reject via HITL manager
	if err := m.hitlManager.RejectRequest(ctx, requestID, userMatrixID, reason); err != nil {
		return fmt.Errorf("failed to reject request: %w", err)
	}

	// Update consent room state
	now := time.Now()
	consentRoom.State = ConsentStateRejected
	consentRoom.RejectedBy = userMatrixID
	consentRoom.RejectionReason = reason
	consentRoom.RespondedAt = &now

	// Send consent response event
	responseEvent := ConsentResponseEvent{
		RequestID:  requestID,
		State:      ConsentStateRejected,
		RejectedBy: userMatrixID,
		Reason:     reason,
		Timestamp:  now.Format(time.RFC3339),
	}

	eventBytes, err := json.Marshal(responseEvent)
	if err != nil {
		m.log.Warn("failed_to_marshal_response_event", "error", err)
	} else {
		_ = m.matrix.SendEvent(consentRoom.MatrixRoomID, ConsentResponseEventType, eventBytes)
	}

	// Send rejection message
	displayReason := reason
	if displayReason == "" {
		displayReason = "No reason provided"
	}
	message := fmt.Sprintf("❌ **Consent Rejected**\n\nRequest `%s` has been rejected by %s.\nReason: %s",
		requestID, userMatrixID, displayReason)
	eventID, err := m.matrix.SendMessage(consentRoom.MatrixRoomID, message, "m.notice")
	if err != nil {
		m.log.Warn("failed_to_send_rejection_message", "error", err)
	} else {
		consentRoom.ResponseEventID = eventID
	}

	// Log rejection
	m.log.Info("consent_rejected",
		"request_id", requestID,
		"rejected_by", userMatrixID,
		"reason", reason,
	)

	// Get request for audit logging
	request, err := m.hitlManager.GetRequest(requestID)
	if err == nil && m.auditLogger != nil {
		_ = m.auditLogger.LogPIIAccessRejected(ctx, requestID, request.SkillID, userMatrixID, reason)
	}

	return nil
}

// GetConsentRoom retrieves a consent room by request ID
func (m *ThreeWayConsentManager) GetConsentRoom(requestID string) (*ConsentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.rooms[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}
	return room, nil
}

// GetConsentRoomByMatrixID retrieves a consent room by Matrix room ID
func (m *ThreeWayConsentManager) GetConsentRoomByMatrixID(roomID string) (*ConsentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.byRoomID[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

// ValidateApprovalToken validates an approval token for a room
func (m *ThreeWayConsentManager) ValidateApprovalToken(roomID string, token string) (*ConsentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.byRoomID[roomID]
	if !exists {
		return nil, ErrRoomNotConsentRoom
	}

	if room.ApprovalToken != token {
		return nil, ErrInvalidApprovalToken
	}

	return room, nil
}

// ListPendingRooms returns all pending consent rooms
func (m *ThreeWayConsentManager) ListPendingRooms() []*ConsentRoom {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*ConsentRoom
	for _, room := range m.rooms {
		room.mu.RLock()
		if room.State == ConsentStatePending && !room.IsExpired() {
			pending = append(pending, room)
		}
		room.mu.RUnlock()
	}

	return pending
}

// CleanupExpired removes expired consent rooms from memory
func (m *ThreeWayConsentManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, room := range m.rooms {
		room.mu.RLock()
		if room.State != ConsentStatePending || room.IsExpired() {
			delete(m.rooms, id)
			delete(m.byRoomID, room.MatrixRoomID)
			delete(m.byToken, room.ApprovalToken)
			count++
		}
		room.mu.RUnlock()
	}

	return count
}

// formatConsentRequestMessage formats a human-readable consent request message
func (m *ThreeWayConsentManager) formatConsentRequestMessage(request *AccessRequest, approvalToken string) string {
	message := fmt.Sprintf(`## 🔐 PII Access Request

**Skill:** %s (%s)
**Request ID:** %s
**Profile:** %s

### Requested Fields

**Required:**
%s

**Optional:**
%s

⏱️ **Expires at:** %s

---

### To Approve
React with ✅ to this message, or use:
`+"```"+`
!consent approve %s
`+"```"+`

### To Reject
React with ❌ to this message, or use:
`+"```"+`
!consent reject %s [reason]
`+"```"+`
`,
		request.SkillName,
		request.SkillID,
		request.ID,
		request.ProfileID,
		formatFieldList(request.RequiredFields),
		formatFieldList(getOptionalFields(request)),
		request.ExpiresAt.Format(time.RFC3339),
		approvalToken,
		approvalToken,
	)

	return message
}

// formatFieldList formats a list of fields for display
func formatFieldList(fields []string) string {
	if len(fields) == 0 {
		return "(none)"
	}
	result := ""
	for _, f := range fields {
		result += fmt.Sprintf("- %s\n", f)
	}
	return result
}

// getOptionalFields returns fields that are not required
func getOptionalFields(request *AccessRequest) []string {
	requiredSet := make(map[string]bool)
	for _, f := range request.RequiredFields {
		requiredSet[f] = true
	}

	var optional []string
	for _, f := range request.RequestedFields {
		if !requiredSet[f] {
			optional = append(optional, f)
		}
	}

	return optional
}
