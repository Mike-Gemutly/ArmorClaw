package webrtc

import (
	"testing"
	"time"
)

// TestSessionManager_Create tests creating a new session
func TestSessionManager_Create(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	session, err := sm.Create("test-container", "!testRoom:example.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.ContainerID != "test-container" {
		t.Errorf("Expected ContainerID 'test-container', got '%s'", session.ContainerID)
	}

	if session.RoomID != "!testRoom:example.com" {
		t.Errorf("Expected RoomID '!testRoom:example.com', got '%s'", session.RoomID)
	}

	if session.State != SessionPending {
		t.Errorf("Expected state %v, got %v", SessionPending, session.State)
	}

	if session.IsExpired() {
		t.Error("New session should not be expired")
	}

	// Verify TTL
	remaining := session.RemainingTTL()
	if remaining > 5*time.Minute || remaining < 4*time.Minute {
		t.Errorf("Expected remaining TTL around 5 minutes, got %v", remaining)
	}
}

// TestSessionManager_Get tests retrieving a session
func TestSessionManager_Get(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	session, err := sm.Create("test-container", "!testRoom:example.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the session
	retrieved, ok := sm.Get(session.ID)
	if !ok {
		t.Fatal("Session not found")
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}

	// Try to get non-existent session
	_, ok = sm.Get("non-existent")
	if ok {
		t.Error("Should not find non-existent session")
	}
}

// TestSessionManager_UpdateState tests updating session state
func TestSessionManager_UpdateState(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	session, err := sm.Create("test-container", "!testRoom:example.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Update state to active
	err = sm.UpdateState(session.ID, SessionActive)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Verify state was updated
	retrieved, ok := sm.Get(session.ID)
	if !ok {
		t.Fatal("Session not found")
	}

	if retrieved.State != SessionActive {
		t.Errorf("Expected state %v, got %v", SessionActive, retrieved.State)
	}

	// Try to update non-existent session
	err = sm.UpdateState("non-existent", SessionActive)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

// TestSessionManager_End tests ending a session
func TestSessionManager_End(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	session, err := sm.Create("test-container", "!testRoom:example.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// End the session
	err = sm.End(session.ID)
	if err != nil {
		t.Fatalf("Failed to end session: %v", err)
	}

	// Verify session was removed
	_, ok := sm.Get(session.ID)
	if ok {
		t.Error("Session should be removed after ending")
	}

	// Try to end non-existent session
	err = sm.End("non-existent")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

// TestSessionManager_Fail tests failing a session
func TestSessionManager_Fail(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	session, err := sm.Create("test-container", "!testRoom:example.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Fail the session
	reason := "connection lost"
	err = sm.Fail(session.ID, reason)
	if err != nil {
		t.Fatalf("Failed to fail session: %v", err)
	}

	// Verify session was removed
	_, ok := sm.Get(session.ID)
	if ok {
		t.Error("Session should be removed after failing")
	}

	// Try to fail non-existent session
	err = sm.Fail("non-existent", "test")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

// TestSessionManager_List tests listing all sessions
func TestSessionManager_List(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Initially empty
	sessions := sm.List()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Create some sessions
	session1, _ := sm.Create("container1", "!room1:example.com", 5*time.Minute)
	session2, _ := sm.Create("container2", "!room2:example.com", 5*time.Minute)

	sessions = sm.List()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify sessions
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}

	if !sessionIDs[session1.ID] {
		t.Error("Session 1 not found in list")
	}

	if !sessionIDs[session2.ID] {
		t.Error("Session 2 not found in list")
	}
}

// TestSessionManager_Count tests counting sessions
func TestSessionManager_Count(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          1 * time.Hour,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Initially 0
	count := sm.Count()
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Add sessions
	_, _ = sm.Create("container1", "!room1:example.com", 5*time.Minute)
	_, _ = sm.Create("container2", "!room2:example.com", 5*time.Minute)
	_, _ = sm.Create("container3", "!room3:example.com", 5*time.Minute)

	count = sm.Count()
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// TestSessionManager_TTLValidation tests TTL enforcement
func TestSessionManager_TTLValidation(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      100 * time.Millisecond,
		MaxTTL:          1 * time.Minute,
		CleanupInterval: 50 * time.Millisecond,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session with short TTL
	session, err := sm.Create("test-container", "!testRoom:example.com", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired
	if !session.IsExpired() {
		t.Error("Session should be expired")
	}

	// Cleanup should have removed it
	_, ok := sm.Get(session.ID)
	if ok {
		t.Error("Expired session should be removed by cleanup")
	}
}

// TestSessionManager_MaxTTL tests maximum TTL enforcement
func TestSessionManager_MaxTTL(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      10 * time.Minute,
		MaxTTL:          30 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}

	sm := NewSessionManager(config)
	defer sm.Stop()

	// Try to create session with TTL exceeding max
	_, err := sm.Create("test-container", "!testRoom:example.com", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Session should be created with max TTL
	sessions := sm.List()
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	remaining := session.RemainingTTL()

	// Should be close to max TTL (30 minutes)
	if remaining > 31*time.Minute || remaining < 29*time.Minute {
		t.Errorf("Expected TTL around 30 minutes, got %v", remaining)
	}
}

// TestTokenManager_GenerateValidate tests token generation and validation
func TestTokenManager_GenerateValidate(t *testing.T) {
	secret := "test-secret-key"
	ttl := 15 * time.Minute

	tm := NewTokenManager(secret, ttl)

	// Generate a token
	token, err := tm.Generate("sess-123", "!room:example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Verify token properties
	if token.SessionID != "sess-123" {
		t.Errorf("Expected SessionID 'sess-123', got '%s'", token.SessionID)
	}

	if token.RoomID != "!room:example.com" {
		t.Errorf("Expected RoomID '!room:example.com', got '%s'", token.RoomID)
	}

	if token.Signature == "" {
		t.Error("Token signature is empty")
	}

	// Validate the token
	claims, err := tm.Validate(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.SessionID != "sess-123" {
		t.Errorf("Expected SessionID 'sess-123', got '%s'", claims.SessionID)
	}

	if claims.RoomID != "!room:example.com" {
		t.Errorf("Expected RoomID '!room:example.com', got '%s'", claims.RoomID)
	}
}

// TestTokenManager_Expiration tests token expiration
func TestTokenManager_Expiration(t *testing.T) {
	secret := "test-secret-key"
	ttl := 10 * time.Millisecond // Very short TTL

	tm := NewTokenManager(secret, ttl)

	// Generate a token
	token, err := tm.Generate("sess-123", "!room:example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Token should be expired
	_, err = tm.Validate(token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

// TestTokenManager_InvalidSignature tests token signature validation
func TestTokenManager_InvalidSignature(t *testing.T) {
	secret := "test-secret-key"
	ttl := 15 * time.Minute

	tm := NewTokenManager(secret, ttl)

	// Generate a token
	token, err := tm.Generate("sess-123", "!room:example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Tamper with the signature
	token.Signature = "invalid"

	// Validation should fail
	_, err = tm.Validate(token)
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

// TestTokenManager_TURNCredentials tests TURN credential generation
func TestTokenManager_TURNCredentials(t *testing.T) {
	secret := "turn-shared-secret"
	ttl := 15 * time.Minute

	tm := NewTokenManager(secret, ttl)

	// Generate a token
	token, err := tm.Generate("sess-123", "!room:example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Generate TURN credentials
	turnCreds := tm.GenerateTURNCredentials(token, "turn:example.com:3478", "stun:example.com:3478")

	// Verify TURN credentials format
	if turnCreds.Username == "" {
		t.Error("TURN username is empty")
	}

	if turnCreds.Password == "" {
		t.Error("TURN password is empty")
	}

	if turnCreds.TURNServer != "turn:example.com:3478" {
		t.Errorf("Expected TURN server 'turn:example.com:3478', got '%s'", turnCreds.TURNServer)
	}

	if turnCreds.STUNServer != "stun:example.com:3478" {
		t.Errorf("Expected STUN server 'stun:example.com:3478', got '%s'", turnCreds.STUNServer)
	}

	// Verify username format: <expiry>:<session_id>
	// Should contain the session ID
	if !contains(turnCreds.Username, "sess-123") {
		t.Error("TURN username should contain session ID")
	}
}

// TestSessionState_String tests session state string representation
func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected string
	}{
		{SessionPending, "pending"},
		{SessionActive, "active"},
		{SessionEnded, "ended"},
		{SessionFailed, "failed"},
		{SessionExpired, "expired"},
		{SessionState(99), "unknown"},
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

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
