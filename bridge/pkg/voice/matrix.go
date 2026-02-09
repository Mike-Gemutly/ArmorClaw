// Package voice provides Matrix integration for WebRTC voice calls
// Handles room-scoped authorization, call signaling, and state management
package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/webrtc"
	"log/slog"
)

// EventType represents Matrix event types for voice calls
type EventType string

const (
	// EventTypeCallInvite is sent when a user invites someone to a call
	EventTypeCallInvite EventType = "m.call.invite"
	// EventTypeCallCandidates is sent with ICE candidates
	EventTypeCallCandidates EventType = "m.call.candidates"
	// EventTypeCallAnswer is sent when the call is answered
	EventTypeCallAnswer EventType = "m.call.answer"
	// EventTypeCallHangup is sent when the call ends
	EventTypeCallHangup EventType = "m.call.hangup"
	// EventTypeCallReject is sent when the call is rejected
	EventTypeCallReject EventType = "m.call.reject"
	// EventTypeCallSelectAnswer is sent when selecting an answer
	EventTypeCallSelectAnswer EventType = "m.call.select_answer"
	// EventTypeCallNegotiate is sent for renegotiation
	EventTypeCallNegotiate EventType = "m.call.negotiate"
)

// CallEvent represents a Matrix call event
type CallEvent struct {
	Type      EventType    `json:"type"`
	CallID    string       `json:"call_id"`
	PartyID   string       `json:"party_id"`
	Version   string       `json:"version"`
	SDP       string       `json:"sdp,omitempty"`
	SDP       json.RawMessage `json:"sdp,omitempty"`
	Candidates []Candidate  `json:"candidates,omitempty"`
	Answer    CallAnswer   `json:"answer,omitempty"`
	Hangup   CallHangup   `json:"hangup,omitempty"`
	Invite   CallInvite   `json:"invite,omitempty"`
	Reject   CallReject   `json:"reject,omitempty"`
	CreateTime time.Time   `json:"create_time"`
	Lifetime  uint32       `json:"lifetime,omitempty"`
}

// Candidate represents an ICE candidate in a call event
type Candidate struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex int    `json:"sdpMLineIndex"`
	SDPMid        string `json:"sdpMid"`
}

// CallInvite represents the invite event content
type CallInvite struct {
	CallID     string      `json:"call_id"`
	Lifetime   uint32      `json:"lifetime"`
	Offer      CallOffer   `json:"offer"`
	PartyID    string      `json:"party_id"`
	CreateTime time.Time   `json:"create_time"`
}

// CallOffer represents the SDP offer
type CallOffer struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	SDP    string `json:"sdp"`
}

// CallAnswer represents the answer event content
type CallAnswer struct {
	CallID  string `json:"call_id"`
	Answer CallOffer `json:"answer"`
	PartyID string   `json:"party_id"`
	Version string   `json:"version"`
}

// CallHangup represents the hangup event content
type CallHangup struct {
	CallID  string `json:"call_id"`
	PartyID string `json:"party_id"`
	Reason  string `json:"reason,omitempty"`
	Version string `json:"version"`
}

// CallReject represents the reject event content
type CallReject struct {
	CallID  string `json:"call_id"`
	PartyID string `json:"party_id"`
	Reason  string `json:"reason,omitempty"`
}

// CallState represents the state of a Matrix call
type CallState int

const (
	// CallStateInvite is when the call is invited
	CallStateInvite CallState = iota
	// CallStateRinging is when the call is ringing
	CallStateRinging
	// CallStateConnected is when the call is connected
	CallStateConnected
	// CallStateEnded is when the call ended
	CallStateEnded
	// CallStateRejected is when the call was rejected
	CallStateRejected
	// CallStateFailed is when the call failed
	CallStateFailed
	// CallStateExpired is when the call timed out
	CallStateExpired
)

// String returns the string representation of the call state
func (s CallState) String() string {
	switch s {
	case CallStateInvite:
		return "invite"
	case CallStateRinging:
		return "ringing"
	case CallStateConnected:
		return "connected"
	case CallStateEnded:
		return "ended"
	case CallStateRejected:
		return "rejected"
	case CallStateFailed:
		return "failed"
	case CallStateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// MatrixCall represents a Matrix WebRTC call
type MatrixCall struct {
	ID            string
	RoomID        string
	CallerID      string
	CalleeID      string
	State         CallState
	SessionID     string    // Associated WebRTC session
	CallEvents    []CallEvent
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	SDPOffer      string
	SDPAnswer     string
	Candidates    []Candidate
	mu            sync.RWMutex
	closeOnce     sync.Once
	closeChan     chan struct{}
}

// Manager manages Matrix voice calls
type Manager struct {
	matrix          *adapter.MatrixAdapter
	sessions        *webrtc.SessionManager
	calls           sync.Map // map[callID]*MatrixCall
	config          Config
	mu              sync.RWMutex
	stopChan        chan struct{}
	wg              sync.WaitGroup
	securityLog     *logger.SecurityLogger
}

// Config holds configuration for Matrix voice calls
type Config struct {
	// Default call lifetime
	DefaultLifetime time.Duration

	// Maximum call lifetime
	MaxLifetime time.Duration

	// Auto-answer calls
	AutoAnswer bool

	// Require room membership for calls
	RequireMembership bool

	// Allowed rooms (empty = all rooms allowed)
	AllowedRooms map[string]bool

	// Blocked rooms
	BlockedRooms map[string]bool
}

// DefaultConfig returns default voice manager configuration
func DefaultConfig() Config {
	return Config{
		DefaultLifetime:    30 * time.Minute,
		MaxLifetime:        2 * time.Hour,
		AutoAnswer:         false,
		RequireMembership:  true,
		AllowedRooms:       make(map[string]bool),
		BlockedRooms:       make(map[string]bool),
	}
}

// NewManager creates a new Matrix voice call manager
func NewManager(matrix *adapter.MatrixAdapter, sessions *webrtc.SessionManager, config Config) *Manager {
	return &Manager{
		matrix:       matrix,
		sessions:     sessions,
		config:       config,
		calls:        sync.Map{},
		stopChan:     make(chan struct{}),
		securityLog:  logger.NewSecurityLogger(logger.Global().WithComponent("voice")),
	}
}

// Start starts the voice call manager
func (m *Manager) Start() error {
	if m.matrix == nil {
		return fmt.Errorf("Matrix adapter not configured")
	}

	// Start event processing
	m.wg.Add(1)
	go m.eventLoop()

	return nil
}

// Stop stops the voice call manager
func (m *Manager) Stop() {
	close(m.stopChan)
	m.wg.Wait()

	// End all active calls
	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)
		m.EndCall(call.ID, "manager_shutdown")
		return true
	})
}

// HandleCallEvent handles an incoming Matrix call event
func (m *Manager) HandleCallEvent(roomID, eventID, senderID string, event *CallEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate sender is in the room
	if m.config.RequireMembership {
		if !m.isMember(roomID, senderID) {
			return fmt.Errorf("sender %s is not a member of room %s", senderID, roomID)
		}
	}

	// Check if room is blocked
	if m.isRoomBlocked(roomID) {
		m.securityLog.LogAccessDenied(context.Background(), "voice_call", roomID,
			slog.String("reason", "room_blocked"),
			slog.String("sender", senderID))
		return fmt.Errorf("room %s is blocked", roomID)
	}

	switch event.Type {
	case EventTypeCallInvite:
		return m.handleInvite(roomID, eventID, senderID, event)

	case EventTypeCallAnswer:
		return m.handleAnswer(roomID, eventID, senderID, event)

	case EventTypeCallHangup:
		return m.handleHangup(roomID, eventID, senderID, event)

	case EventTypeCallReject:
		return m.handleReject(roomID, eventID, senderID, event)

	case EventTypeCallCandidates:
		return m.handleCandidates(roomID, eventID, senderID, event)

	default:
		// Unknown event type, ignore
		return nil
	}
}

// handleInvite handles an incoming call invite
func (m *Manager) handleInvite(roomID, eventID, senderID string, event *CallEvent) error {
	// Check if call already exists
	if _, exists := m.calls.Load(event.CallID); exists {
		return fmt.Errorf("call %s already exists", event.CallID)
	}

	// Create new call
	call := &MatrixCall{
		ID:         event.CallID,
		RoomID:     roomID,
		CallerID:   senderID,
		CalleeID:   m.matrix.UserID(),
		State:      CallStateInvite,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(m.config.DefaultLifetime),
		CallEvents: []CallEvent{*event},
		closeChan:  make(chan struct{}),
	}

	// Extract SDP offer if present
	if event.SDP != nil {
		call.SDPOffer = string(event.SDP)
	}

	// Store call
	m.calls.Store(event.CallID, call)

	// Log security event
	m.securityLog.LogSecurityEvent("call_invite_received",
		slog.String("call_id", event.CallID),
		slog.String("room_id", roomID),
		slog.String("caller", senderID))

	// Auto-answer if configured
	if m.config.AutoAnswer {
		go m.autoAnswerCall(event.CallID)
	} else {
		// Transition to ringing state
		call.State = CallStateRinging
		call.UpdatedAt = time.Now()
	}

	return nil
}

// handleAnswer handles a call answer
func (m *Manager) handleAnswer(roomID, eventID, senderID string, event *CallEvent) error {
	// Get call
	value, exists := m.calls.Load(event.CallID)
	if !exists {
		return fmt.Errorf("call %s not found", event.CallID)
	}

	call := value.(*MatrixCall)

	// Verify answer is from caller
	if senderID != call.CallerID {
		return fmt.Errorf("answer must come from caller")
	}

	// Update call state
	call.State = CallStateConnected
	call.UpdatedAt = time.Now()
	call.SDPAnswer = event.Answer.Answer.SDP

	// Add event to history
	call.CallEvents = append(call.CallEvents, *event)

	// Log security event
	m.securityLog.LogSecurityEvent("call_answered",
		slog.String("call_id", event.CallID),
		slog.String("room_id", roomID))

	return nil
}

// handleHangup handles a call hangup
func (m *Manager) handleHangup(roomID, eventID, senderID string, event *CallEvent) error {
	value, exists := m.calls.Load(event.CallID)
	if !exists {
		return fmt.Errorf("call %s not found", event.CallID)
	}

	call := value.(*MatrixCall)

	// Update call state
	call.State = CallStateEnded
	call.UpdatedAt = time.Now()

	// Add event to history
	call.CallEvents = append(call.CallEvents, *event)

	// Close the call
	call.Close()

	// Remove from active calls
	m.calls.Delete(event.CallID)

	// Log security event
	m.securityLog.LogSecurityEvent("call_ended",
		slog.String("call_id", event.CallID),
		slog.String("room_id", roomID),
		slog.String("ended_by", senderID),
		slog.String("reason", event.Hangup.Reason))

	// End associated WebRTC session
	if call.SessionID != "" {
		m.sessions.End(call.SessionID)
	}

	return nil
}

// handleReject handles a call reject
func (m *Manager) handleReject(roomID, eventID, senderID string, event *CallEvent) error {
	value, exists := m.calls.Load(event.CallID)
	if !exists {
		return fmt.Errorf("call %s not found", event.CallID)
	}

	call := value.(*MatrixCall)

	// Verify reject is from callee
	if senderID != call.CalleeID {
		return fmt.Errorf("reject must come from callee")
	}

	// Update call state
	call.State = CallStateRejected
	call.UpdatedAt = time.Now()

	// Add event to history
	call.CallEvents = append(call.CallEvents, *event)

	// Close the call
	call.Close()

	// Remove from active calls
	m.calls.Delete(event.CallID)

	// Log security event
	m.securityLog.LogSecurityEvent("call_rejected",
		slog.String("call_id", event.CallID),
		slog.String("room_id", roomID),
		slog.String("reason", event.Reject.Reason))

	return nil
}

// handleCandidates handles ICE candidates
func (m *Manager) handleCandidates(roomID, eventID, senderID string, event *CallEvent) error {
	value, exists := m.calls.Load(event.CallID)
	if !exists {
		return fmt.Errorf("call %s not found", event.CallID)
	}

	call := value.(*MatrixCall)

	// Verify candidate is from caller
	if senderID != call.CallerID {
		return fmt.Errorf("candidates must come from caller")
	}

	// Add candidates
	call.Candidates = append(call.Candidates, event.Candidates...)
	call.UpdatedAt = time.Now()

	// Add event to history
	call.CallEvents = append(call.CallEvents, *event)

	return nil
}

// StartCall initiates a new voice call from the bridge
func (m *Manager) StartCall(roomID string, offerSDP string) (*MatrixCall, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify bridge is member of the room
	if m.config.RequireMembership {
		if !m.isMember(roomID, m.matrix.UserID()) {
			return nil, fmt.Errorf("bridge is not a member of room %s", roomID)
		}
	}

	// Check if room is blocked
	if m.isRoomBlocked(roomID) {
		m.securityLog.LogAccessDenied(context.Background(), "voice_call_start", roomID,
			slog.String("reason", "room_blocked"))
		return nil, fmt.Errorf("room %s is blocked", roomID)
	}

	// Generate call ID
	callID := generateCallID()

	// Create invite event
	lifetime := uint32(m.config.DefaultLifetime.Seconds())
	invite := CallInvite{
		CallID:     callID,
		Lifetime:   lifetime,
		Offer:      CallOffer{Type: "offer", SDP: offerSDP},
		PartyID:    m.matrix.UserID(),
		CreateTime: time.Now(),
	}

	// Create call
	call := &MatrixCall{
		ID:         callID,
		RoomID:     roomID,
		CallerID:   m.matrix.UserID(),
		State:      CallStateInvite,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(m.config.DefaultLifetime),
		SDPOffer:   offerSDP,
		closeChan:  make(chan struct{}),
	}

	// Store call
	m.calls.Store(callID, call)

	// Send invite event via Matrix
	inviteEvent := CallEvent{
		Type:      EventTypeCallInvite,
		CallID:    callID,
		PartyID:   m.matrix.UserID(),
		Version:   "0",
		Invite:    invite,
		CreateTime: time.Now(),
		Lifetime:  lifetime,
	}

	// Serialize and send
	eventJSON, err := json.Marshal(inviteEvent)
	if err != nil {
		m.calls.Delete(callID)
		return nil, fmt.Errorf("failed to marshal invite event: %w", err)
	}

	// Send to Matrix room
	err = m.matrix.SendEvent(roomID, "m.call.invite", eventJSON)
	if err != nil {
		m.calls.Delete(callID)
		return nil, fmt.Errorf("failed to send invite event: %w", err)
	}

	// Log security event
	m.securityLog.LogSecurityEvent("call_started",
		slog.String("call_id", callID),
		slog.String("room_id", roomID))

	return call, nil
}

// AnswerCall answers an incoming call
func (m *Manager) AnswerCall(callID string, answerSDP string) error {
	value, exists := m.calls.Load(callID)
	if !exists {
		return fmt.Errorf("call %s not found", callID)
	}

	call := value.(*MatrixCall)

	// Verify call is in appropriate state
	if call.State != CallStateInvite && call.State != CallStateRinging {
		return fmt.Errorf("call is not in a state that can be answered")
	}

	// Create answer event
	answer := CallAnswer{
		CallID:  callID,
		Answer:  CallOffer{Type: "answer", SDP: answerSDP},
		PartyID: m.matrix.UserID(),
		Version: "0",
	}

	answerEvent := CallEvent{
		Type:    EventTypeCallAnswer,
		CallID:  callID,
		PartyID: m.matrix.UserID(),
		Version: "0",
		Answer:  answer,
	}

	// Serialize and send
	eventJSON, err := json.Marshal(answerEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal answer event: %w", err)
	}

	// Send to Matrix room
	err = m.matrix.SendEvent(call.RoomID, "m.call.answer", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to send answer event: %w", err)
	}

	// Update call state
	call.State = CallStateRinging
	call.UpdatedAt = time.Now()
	call.SDPAnswer = answerSDP

	// Log security event
	m.securityLog.LogSecurityEvent("call_answered",
		slog.String("call_id", callID),
		slog.String("room_id", call.RoomID))

	return nil
}

// EndCall ends an active call
func (m *Manager) EndCall(callID, reason string) error {
	value, exists := m.calls.Load(callID)
	if !exists {
		return fmt.Errorf("call %s not found", callID)
	}

	call := value.(*MatrixCall)

	// Create hangup event
	hangup := CallHangup{
		CallID:  callID,
		PartyID: m.matrix.UserID(),
		Reason:  reason,
		Version: "0",
	}

	hangupEvent := CallEvent{
		Type:   EventTypeCallHangup,
		CallID: callID,
		PartyID: m.matrix.UserID(),
		Hangup: hangup,
	}

	// Serialize and send
	eventJSON, err := json.Marshal(hangupEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal hangup event: %w", err)
	}

	// Send to Matrix room
	err = m.matrix.SendEvent(call.RoomID, "m.call.hangup", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to send hangup event: %w", err)
	}

	// Update call state
	call.State = CallStateEnded
	call.UpdatedAt = time.Now()

	// Close call
	call.Close()

	// Remove from active calls
	m.calls.Delete(callID)

	// Log security event
	m.securityLog.LogSecurityEvent("call_ended",
		slog.String("call_id", callID),
		slog.String("room_id", call.RoomID),
		slog.String("reason", reason))

	return nil
}

// RejectCall rejects an incoming call
func (m *Manager) RejectCall(callID, reason string) error {
	value, exists := m.calls.Load(callID)
	if !exists {
		return fmt.Errorf("call %s not found", callID)
	}

	call := value.(*MatrixCall)

	// Create reject event
	reject := CallReject{
		CallID: callID,
		PartyID: m.matrix.UserID(),
		Reason: reason,
	}

	rejectEvent := CallEvent{
		Type:   EventTypeCallReject,
		CallID: callID,
		PartyID: m.matrix.UserID(),
		Reject: reject,
	}

	// Serialize and send
	eventJSON, err := json.Marshal(rejectEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal reject event: %w", err)
	}

	// Send to Matrix room
	err = m.matrix.SendEvent(call.RoomID, "m.call.reject", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to send reject event: %w", err)
	}

	// Update call state
	call.State = CallStateRejected
	call.UpdatedAt = time.Now()

	// Close call
	call.Close()

	// Remove from active calls
	m.calls.Delete(callID)

	// Log security event
	m.securityLog.LogSecurityEvent("call_rejected",
		slog.String("call_id", callID),
		slog.String("room_id", call.RoomID),
		slog.String("reason", reason))

	return nil
}

// SendCandidates sends ICE candidates for a call
func (m *Manager) SendCandidates(callID string, candidates []Candidate) error {
	value, exists := m.calls.Load(callID)
	if !exists {
		return fmt.Errorf("call %s not found", callID)
	}

	call := value.(*MatrixCall)

	// Create candidates event
	candidatesEvent := CallEvent{
		Type:       EventTypeCallCandidates,
		CallID:     callID,
		PartyID:    m.matrix.UserID(),
		Version:    "0",
		Candidates: candidates,
	}

	// Serialize and send
	eventJSON, err := json.Marshal(candidatesEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates event: %w", err)
	}

	// Send to Matrix room
	err = m.matrix.SendEvent(call.RoomID, "m.call.candidates", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to send candidates event: %w", err)
	}

	// Update call
	call.Candidates = append(call.Candidates, candidates...)
	call.UpdatedAt = time.Now()

	return nil
}

// GetCall retrieves a call by ID
func (m *Manager) GetCall(callID string) (*MatrixCall, bool) {
	value, exists := m.calls.Load(callID)
	if !exists {
		return nil, false
	}
	return value.(*MatrixCall), true
}

// ListCalls returns all active calls
func (m *Manager) ListCalls() []*MatrixCall {
	calls := make([]*MatrixCall, 0)

	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)
		calls = append(calls, call)
		return true
	})

	return calls
}

// GetStats returns statistics about voice calls
func (m *Manager) GetStats() map[string]interface{} {
	activeCalls := 0
	totalCalls := 0
	expiringSoon := 0
	now := time.Now()

	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)
		totalCalls++

		if call.State == CallStateConnected || call.State == CallStateRinging {
			activeCalls++
		}

		// Count expiring within 5 minutes
		if call.ExpiresAt.Sub(now) < 5*time.Minute {
			expiringSoon++
		}

		return true
	})

	return map[string]interface{}{
		"active_calls":  activeCalls,
		"total_calls":   totalCalls,
		"expiring_soon":  expiringSoon,
	}
}

// isMember checks if a user is a member of the room
func (m *Manager) isMember(roomID, userID string) bool {
	// In a full implementation, this would check Matrix room state
	// For now, return true (assume validation happens elsewhere)
	return true
}

// isRoomBlocked checks if a room is blocked for voice calls
func (m *Manager) isRoomBlocked(roomID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if room is in blocked list
	if m.config.BlockedRooms[roomID] {
		return true
	}

	// If allowed rooms is configured and room is not in it, block
	if len(m.config.AllowedRooms) > 0 && !m.config.AllowedRooms[roomID] {
		return true
	}

	return false
}

// autoAnswerCall automatically answers a call
func (m *Manager) autoAnswerCall(callID string) {
	// Wait a moment before answering
	time.Sleep(500 * time.Millisecond)

	// Get call
	call, exists := m.GetCall(callID)
	if !exists {
		return
	}

	// Create SDP answer (in a full implementation, this would be generated)
	answerSDP := ""

	// Answer the call
	err := m.AnswerCall(callID, answerSDP)
	if err != nil {
		// Log error but don't fail
		logger.Warn("auto-answer failed", "call_id", callID, "error", err)
	}
}

// eventLoop processes Matrix events for voice calls
func (m *Manager) eventLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for expired calls
			m.cleanupExpiredCalls()

		case <-m.stopChan:
			return
		}
	}
}

// cleanupExpiredCalls removes expired calls
func (m *Manager) cleanupExpiredCalls() {
	now := time.Now()

	m.calls.Range(func(key, value interface{}) bool {
		call := value.(*MatrixCall)

		if now.After(call.ExpiresAt) {
			// Call has expired
			call.State = CallStateExpired
			call.Close()
			m.calls.Delete(key)

			// Log security event
			m.securityLog.LogSecurityEvent("call_expired",
				slog.String("call_id", call.ID),
				slog.String("room_id", call.RoomID))
		}

		return true
	})
}

// Close closes the call
func (c *MatrixCall) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
}

// Closed returns a channel that's closed when the call ends
func (c *MatrixCall) Closed() <-chan struct{} {
	return c.closeChan
}

// AddSession associates a WebRTC session with the call
func (c *MatrixCall) AddSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SessionID = sessionID
}

// GetSession returns the associated WebRTC session ID
func (c *MatrixCall) GetSession() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SessionID
}

// generateCallID generates a unique call ID
func generateCallID() string {
	return fmt.Sprintf("call_%d", time.Now().UnixNano())
}

// Errors
var (
	// ErrCallNotFound is returned when a call is not found
	ErrCallNotFound = fmt.Errorf("call not found")

	// ErrCallExpired is returned when a call has expired
	ErrCallExpired = fmt.Errorf("call expired")

	// ErrCallEnded is returned when attempting to operate on an ended call
	ErrCallEnded = fmt.Errorf("call has ended")

	// ErrRoomBlocked is returned when a room is blocked for voice calls
	ErrRoomBlocked = fmt.Errorf("room is blocked")
)
