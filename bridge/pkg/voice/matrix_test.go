// Package voice tests for Matrix integration with WebRTC voice calls
package voice

import (
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/webrtc"
)

// MockMatrixAdapter is a mock Matrix adapter for testing
type MockMatrixAdapter struct {
	UserID      string
	Members     map[string][]string // roomID -> userIDs
	SentEvents  []MatrixEvent
	mu          sync.RWMutex
}

// MatrixEvent represents a sent Matrix event
type MatrixEvent struct {
	RoomID  string
	Type    string
	Content []byte
}

// NewMockMatrixAdapter creates a new mock Matrix adapter
func NewMockMatrixAdapter(userID string) *MockMatrixAdapter {
	return &MockMatrixAdapter{
		UserID:  userID,
		Members: make(map[string][]string),
	}
}

// SendEvent sends an event to a Matrix room
func (m *MockMatrixAdapter) SendEvent(roomID, eventType string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SentEvents = append(m.SentEvents, MatrixEvent{
		RoomID:  roomID,
		Type:    eventType,
		Content: content,
	})

	return nil
}

// AddMember adds a member to a room
func (m *MockMatrixAdapter) AddMember(roomID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Members[roomID] == nil {
		m.Members[roomID] = make([]string, 0)
	}
	m.Members[roomID] = append(m.Members[roomID], userID)
}

// IsMember checks if a user is a member of a room
func (m *MockMatrixAdapter) IsMember(roomID, userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members, ok := m.Members[roomID]
	if !ok {
		return false
	}

	for _, member := range members {
		if member == userID {
			return true
		}
	}

	return false
}

// TestNewManager tests creating a new voice manager
func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	if manager == nil {
		t.Error("Manager should not be nil")
	}
}

// TestStartCall tests initiating a new voice call
func TestStartCall(t *testing.T) {
	config := DefaultConfig()
	config.AutoAnswer = false
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	roomID := "!testRoom:example.com"
	offerSDP := "v=0\r\no=- host 0 IN IP4 127.0.0.1\r\ns=-\r\nm=audio 0 RTP/SAVPF 111\r\na=rtpmap:111 opus/48000/1"

	call, err := manager.StartCall(roomID, offerSDP)
	if err != nil {
		t.Fatalf("Failed to start call: %v", err)
	}

	if call.ID == "" {
		t.Error("Call ID should not be empty")
	}

	if call.RoomID != roomID {
		t.Errorf("Expected room ID '%s', got '%s'", roomID, call.RoomID)
	}

	if call.CallerID != "@bridge:example.com" {
		t.Errorf("Expected caller ID '@bridge:example.com', got '%s'", call.CallerID)
	}

	if call.State != CallStateInvite {
		t.Errorf("Expected state %v, got %v", CallStateInvite, call.State)
	}

	if call.SDPOffer != offerSDP {
		t.Error("SDP offer not stored")
	}

	// Verify invite was sent to Matrix
	if len(matrix.SentEvents) != 1 {
		t.Errorf("Expected 1 event sent, got %d", len(matrix.SentEvents))
	}

	event := matrix.SentEvents[0]
	if event.Type != "m.call.invite" {
		t.Errorf("Expected event type 'm.call.invite', got '%s'", event.Type)
	}
}

// TestAnswerCall tests answering an incoming call
func TestAnswerCall(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create an incoming call invite
	inviteEvent := &CallEvent{
		Type:    EventTypeCallInvite,
		CallID:  "test-call-123",
		PartyID: "@caller:example.com",
		Version: "0",
		Invite: CallInvite{
			CallID:   "test-call-123",
			Lifetime: 1800,
			Offer: CallOffer{
				Type: "offer",
				CallID: "test-call-123",
				SDP:   "test-sdp-offer",
			},
			PartyID: "@caller:example.com",
		},
		CreateTime: time.Now(),
		Lifetime:  1800,
	}

	roomID := "!testRoom:example.com"
	err := manager.HandleCallEvent(roomID, "event-123", "@caller:example.com", inviteEvent)
	if err != nil {
		t.Fatalf("Failed to handle invite: %v", err)
	}

	// Answer the call
	answerSDP := "v=0\r\no=- host 0 IN IP4 127.0.0.1\r\ns=-\r\nm=audio 0 RTP/SAVPF 111\r\na=rtpmap:111 opus/48000/1"
	err = manager.AnswerCall("test-call-123", answerSDP)
	if err != nil {
		t.Fatalf("Failed to answer call: %v", err)
	}

	// Verify call state
	call, ok := manager.GetCall("test-call-123")
	if !ok {
		t.Fatal("Call not found")
	}

	if call.State != CallStateRinging {
		t.Errorf("Expected state %v, got %v", CallStateRinging, call.State)
	}

	// Verify answer was sent
	foundAnswer := false
	for _, event := range matrix.SentEvents {
		if event.Type == "m.call.answer" {
			foundAnswer = true
			break
		}
	}

	if !foundAnswer {
		t.Error("Answer event not sent to Matrix")
	}
}

// TestEndCall tests ending an active call
func TestEndCall(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create a call
	call, _ := manager.StartCall("!testRoom:example.com", "test-sdp")

	// End the call
	err := manager.EndCall(call.ID, "user_hangup")
	if err != nil {
		t.Fatalf("Failed to end call: %v", err)
	}

	// Verify call was removed
	_, ok := manager.GetCall(call.ID)
	if ok {
		t.Error("Call should be removed after ending")
	}

	// Verify hangup was sent
	foundHangup := false
	for _, event := range matrix.SentEvents {
		if event.Type == "m.call.hangup" {
			foundHangup = true
			break
		}
	}

	if !foundHangup {
		t.Error("Hangup event not sent to Matrix")
	}
}

// TestRejectCall tests rejecting an incoming call
func TestRejectCall(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create an incoming call invite
	inviteEvent := &CallEvent{
		Type:    EventTypeCallInvite,
		CallID:  "test-call-456",
		PartyID: "@caller:example.com",
		Version: "0",
		Invite: CallInvite{
			CallID: "test-call-456",
			Offer:  CallOffer{Type: "offer", CallID: "test-call-456", SDP: "test-sdp"},
			PartyID: "@caller:example.com",
		},
		CreateTime: time.Now(),
	}

	roomID := "!testRoom:example.com"
	manager.HandleCallEvent(roomID, "event-123", "@caller:example.com", inviteEvent)

	// Reject the call
	reason := "busy"
	err := manager.RejectCall("test-call-456", reason)
	if err != nil {
		t.Fatalf("Failed to reject call: %v", err)
	}

	// Verify call state
	call, ok := manager.GetCall("test-call-456")
	if !ok {
		t.Fatal("Call not found")
	}

	if call.State != CallStateRejected {
		t.Errorf("Expected state %v, got %v", CallStateRejected, call.State)
	}

	// Verify reject was sent
	foundReject := false
	for _, event := range matrix.SentEvents {
		if event.Type == "m.call.reject" {
			foundReject = true
			break
		}
	}

	if !foundReject {
		t.Error("Reject event not sent to Matrix")
	}
}

// TestSendCandidates tests sending ICE candidates
func TestSendCandidates(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create a call
	call, _ := manager.StartCall("!testRoom:example.com", "test-sdp")

	// Create candidates
	candidates := []Candidate{
		{
			Candidate:     "candidate:1 1 udp 2130706431 192.168.1.100 54321 typ host",
			SDPMLineIndex: 0,
			SDPMid:        "audio",
		},
		{
			Candidate:     "candidate:2 1 udp 1694498815 203.0.113.5 62345 typ srflx raddr 192.168.1.100 rport 54321",
			SDPMLineIndex: 0,
			SDPMid:        "audio",
		},
	}

	err := manager.SendCandidates(call.ID, candidates)
	if err != nil {
		t.Fatalf("Failed to send candidates: %v", err)
	}

	// Verify candidates were stored
	retrievedCall, _ := manager.GetCall(call.ID)
	if len(retrievedCall.Candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(retrievedCall.Candidates))
	}

	// Verify candidates event was sent
	foundCandidates := false
	for _, event := range matrix.SentEvents {
		if event.Type == "m.call.candidates" {
			foundCandidates = true
			break
		}
	}

	if !foundCandidates {
		t.Error("Candidates event not sent to Matrix")
	}
}

// TestCallExpiration tests call expiration
func TestCallExpiration(t *testing.T) {
	config := DefaultConfig{
		DefaultLifetime: 100 * time.Millisecond, // Very short TTL
		MaxLifetime:     1 * time.Minute,
		RequireMembership: false,
	}
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)
	manager.Start()
	defer manager.Stop()

	// Create a call
	call, _ := manager.StartCall("!testRoom:example.com", "test-sdp")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Cleanup should have removed it
	_, ok := manager.GetCall(call.ID)
	if ok {
		t.Error("Call should be removed after expiration")
	}
}

// TestListCalls tests listing active calls
func TestListCalls(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Initially no calls
	calls := manager.ListCalls()
	if len(calls) != 0 {
		t.Errorf("Expected 0 calls, got %d", len(calls))
	}

	// Create some calls
	call1, _ := manager.StartCall("!room1:example.com", "sdp1")
	call2, _ := manager.StartCall("!room2:example.com", "sdp2")

	calls = manager.ListCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(calls))
	}

	// Verify calls
	callIDs := make(map[string]bool)
	for _, call := range calls {
		callIDs[call.ID] = true
	}

	if !callIDs[call1.ID] {
		t.Error("Call 1 not found in list")
	}

	if !callIDs[call2.ID] {
		t.Error("Call 2 not found in list")
	}
}

// TestGetStats tests getting voice statistics
func TestGetStats(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create a call
	manager.StartCall("!room:example.com", "sdp")

	// Answer it (makes it active)
	manager.AnswerCall(manager.ListCalls()[0].ID, "answer-sdp")

	stats := manager.GetStats()

	if stats["total_calls"].(int) != 1 {
		t.Errorf("Expected 1 total call, got %d", stats["total_calls"])
	}

	// Active calls are those in ringing state (answered calls transition to connected)
	if stats["active_calls"].(int) != 1 {
		t.Errorf("Expected 1 active call, got %d", stats["active_calls"])
	}
}

// TestRequireMembership tests room membership validation
func TestRequireMembership(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = true
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Try to start call without being member
	roomID := "!testRoom:example.com"
	_, err := manager.StartCall(roomID, "test-sdp")

	if err == nil {
		t.Error("Should fail when not a member of the room")
	}

	// Add bridge as member
	matrix.AddMember(roomID, "@bridge:example.com")

	// Now it should work
	_, err = manager.StartCall(roomID, "test-sdp")
	if err != nil {
		t.Fatalf("Should succeed when member: %v", err)
	}
}

// TestBlockedRooms tests room blocking
func TestBlockedRooms(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	config.BlockedRooms = make(map[string]bool)
	config.BlockedRooms["!blockedRoom:example.com"] = true

	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Try to start call in blocked room
	_, err := manager.StartCall("!blockedRoom:example.com", "test-sdp")

	if err == nil {
		t.Error("Should fail when room is blocked")
	}

	// Try to start call in non-blocked room
	_, err = manager.StartCall("!allowedRoom:example.com", "test-sdp")
	if err != nil {
		t.Fatalf("Should succeed in non-blocked room: %v", err)
	}
}

// TestAllowedRooms tests allowed room filtering
func TestAllowedRooms(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	config.AllowedRooms = make(map[string]bool)
	config.AllowedRooms["!allowedRoom:example.com"] = true

	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Try to start call in non-allowed room
	_, err := manager.StartCall("!otherRoom:example.com", "test-sdp")

	if err == nil {
		t.Error("Should fail when room is not in allowed list")
	}

	// Try to start call in allowed room
	_, err = manager.StartCall("!allowedRoom:example.com", "test-sdp")
	if err != nil {
		t.Fatalf("Should succeed in allowed room: %v", err)
	}
}

// TestCallStateTransitions tests call state transitions
func TestCallStateTransitions(t *testing.T) {
	config := DefaultConfig()
	config.RequireMembership = false
	config.AutoAnswer = false
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Start call (invite state)
	call, _ := manager.StartCall("!testRoom:example.com", "offer")
	if call.State != CallStateInvite {
		t.Errorf("Expected state %v, got %v", CallStateInvite, call.State)
	}

	// Answer call (ringing state)
	manager.AnswerCall(call.ID, "answer")
	if call.State != CallStateRinging {
		t.Errorf("Expected state %v, got %v", CallStateRinging, call.State)
	}

	// End call (ended state)
	manager.EndCall(call.ID, "completed")
	_, ok := manager.GetCall(call.ID)
	if ok {
		t.Error("Call should be removed after ending")
	}
}

// TestAutoAnswer tests auto-answer functionality
func TestAutoAnswer(t *testing.T) {
	config := DefaultConfig()
	config.AutoAnswer = true
	config.RequireMembership = false

	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)

	// Create an incoming call invite
	inviteEvent := &CallEvent{
		Type:    EventTypeCallInvite,
		CallID:  "auto-call-123",
		PartyID: "@caller:example.com",
		Version: "0",
		Invite: CallInvite{
			CallID: "auto-call-123",
			Offer:  CallOffer{Type: "offer", SDP: "test-sdp"},
			PartyID: "@caller:example.com",
		},
		CreateTime: time.Now(),
		Lifetime:  1800,
	}

	roomID := "!testRoom:example.com"
	manager.HandleCallEvent(roomID, "event-123", "@caller:example.com", inviteEvent)

	// Wait for auto-answer
	time.Sleep(700 * time.Millisecond)

	// Verify call was answered
	call, ok := manager.GetCall("auto-call-123")
	if !ok {
		t.Fatal("Call not found")
	}

	if call.State != CallStateRinging {
		t.Errorf("Expected state %v, got %v", CallStateRinging, call.State)
	}
}

// TestMatrixCall_SessionAssociation tests associating WebRTC sessions
func TestMatrixCall_SessionAssociation(t *testing.T) {
	call := &MatrixCall{
		ID:      "test-call",
		RoomID:  "!testRoom:example.com",
		State:   CallStateInvite,
		closeChan: make(chan struct{}),
	}

	// Initially no session
	if call.GetSession() != "" {
		t.Error("Session ID should be empty initially")
	}

	// Add session
	call.AddSession("webrtc-session-123")

	if call.GetSession() != "webrtc-session-123" {
		t.Errorf("Expected session ID 'webrtc-session-123', got '%s'", call.GetSession())
	}
}

// TestCallStateString tests call state string representation
func TestCallStateString(t *testing.T) {
	tests := []struct {
		state    CallState
		expected string
	}{
		{CallStateInvite, "invite"},
		{CallStateRinging, "ringing"},
		{CallStateConnected, "connected"},
		{CallStateEnded, "ended"},
		{CallStateRejected, "rejected"},
		{CallStateFailed, "failed"},
		{CallStateExpired, "expired"},
		{CallState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCleanupExpiredCalls tests cleanup of expired calls
func TestCleanupExpiredCalls(t *testing.T) {
	config := DefaultConfig{
		DefaultLifetime: 100 * time.Millisecond,
		MaxLifetime:     1 * time.Minute,
	}
	matrix := NewMockMatrixAdapter("@bridge:example.com")
	sessions := webrtc.NewSessionManager(config.DefaultLifetime)

	manager := NewManager(matrix, sessions, config)
	manager.Start()
	defer manager.Stop()

	// Create calls with short TTL
	call1, _ := manager.StartCall("!room1:example.com", "sdp1")
	call2, _ := manager.StartCall("!room2:example.com", "sdp2")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Trigger cleanup
	manager.cleanupExpiredCalls()

	// Both should be removed
	_, ok := manager.GetCall(call1.ID)
	if ok {
		t.Error("Call 1 should be removed after expiration")
	}

	_, ok = manager.GetCall(call2.ID)
	if ok {
		t.Error("Call 2 should be removed after expiration")
	}
}

// TestErrors tests error values
func TestErrors(t *testing.T) {
	errors := []struct {
		err  error
		msg  string
	}{
		{ErrCallNotFound, "call not found"},
		{ErrCallExpired, "call expired"},
		{ErrCallEnded, "call has ended"},
		{ErrRoomBlocked, "room is blocked"},
	}

	for _, tt := range errors {
		if tt.err.Error() == "" {
			t.Errorf("Error message should not be empty: %v", tt.err)
		}

		if tt.msg != "" && tt.err.Error() != tt.msg {
			t.Errorf("Expected '%s', got '%s'", tt.msg, tt.err.Error())
		}
	}
}
