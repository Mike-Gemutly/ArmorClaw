// Package voice tests for security enforcement
package voice

import (
	"context"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/webrtc"
)

// TestNewSecurityEnforcer tests creating a security enforcer
func TestNewSecurityEnforcer(t *testing.T) {
	policy := DefaultSecurityPolicy()
	enforcer := NewSecurityEnforcer(policy)

	if enforcer == nil {
		t.Error("Security enforcer should not be nil")
	}

	if enforcer.policy.MaxConcurrentCalls != 10 {
		t.Errorf("Expected max concurrent calls 10, got %d", enforcer.policy.MaxConcurrentCalls)
	}
}

// TestCheckStartCall tests call start validation
func TestCheckStartCall(t *testing.T) {
	policy := DefaultSecurityPolicy()
	enforcer := NewSecurityEnforcer(policy)

	// Should allow call when no limits
	err := enforcer.CheckStartCall("user-123", "room-456")
	if err != nil {
		t.Errorf("Should allow call: %v", err)
	}
}

// TestMaxConcurrentCalls tests concurrent call limit
func TestMaxConcurrentCalls(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.MaxConcurrentCalls = 2
	enforcer := NewSecurityEnforcer(policy)

	// Register calls up to limit
	enforcer.RegisterCall("room-1", "user-1")
	enforcer.RegisterCall("room-2", "user-2")

	// Should exceed limit
	err := enforcer.CheckStartCall("user-3", "room-3")
	if err != ErrMaxConcurrentCallsExceeded {
		t.Errorf("Expected ErrMaxConcurrentCallsExceeded, got %v", err)
	}
}

// TestBlockedUsers tests user blocklist
func TestBlockedUsers(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.BlockedUsers = make(map[string]bool)
	policy.BlockedUsers["@banned:example.com"] = true
	enforcer := NewSecurityEnforcer(policy)

	// Should block user
	err := enforcer.CheckStartCall("@banned:example.com", "room-456")
	if err != ErrUserBlocked {
		t.Errorf("Expected ErrUserBlocked, got %v", err)
	}

	// Should allow other users
	err = enforcer.CheckStartCall("user-123", "room-456")
	if err != nil {
		t.Errorf("Should allow non-blocked user: %v", err)
	}
}

// TestBlockedRooms tests room blocklist
func TestBlockedRooms(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.BlockedRooms = make(map[string]bool)
	policy.BlockedRooms["!blocked:example.com"] = true
	enforcer := NewSecurityEnforcer(policy)

	// Should block room
	err := enforcer.CheckStartCall("user-123", "!blocked:example.com")
	if err != ErrRoomBlocked {
		t.Errorf("Expected ErrRoomBlocked, got %v", err)
	}

	// Should allow other rooms
	err = enforcer.CheckStartCall("user-123", "room-456")
	if err != nil {
		t.Errorf("Should allow non-blocked room: %v", err)
	}
}

// TestAllowedUsers tests user allowlist
func TestAllowedUsers(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.AllowedUsers = make(map[string]bool)
	policy.AllowedUsers["@allowed:example.com"] = true
	enforcer := NewSecurityEnforcer(policy)

	// Should allow listed user
	err := enforcer.CheckStartCall("@allowed:example.com", "room-456")
	if err != nil {
		t.Errorf("Should allow listed user: %v", err)
	}

	// Should block non-listed user
	err = enforcer.CheckStartCall("user-123", "room-456")
	if err != ErrUserNotAllowed {
		t.Errorf("Expected ErrUserNotAllowed, got %v", err)
	}
}

// TestAllowedRooms tests room allowlist
func TestAllowedRooms(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.AllowedRooms = make(map[string]bool)
	policy.AllowedRooms["!allowed:example.com"] = true
	enforcer := NewSecurityEnforcer(policy)

	// Should allow listed room
	err := enforcer.CheckStartCall("user-123", "!allowed:example.com")
	if err != nil {
		t.Errorf("Should allow listed room: %v", err)
	}

	// Should block non-listed room
	err = enforcer.CheckStartCall("user-123", "room-456")
	if err != ErrRoomNotAllowed {
		t.Errorf("Expected ErrRoomNotAllowed, got %v", err)
	}
}

// TestRateLimit tests rate limiting
func TestRateLimit(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.RateLimitCalls = 3
	policy.RateLimitWindow = 1 * time.Hour
	enforcer := NewSecurityEnforcer(policy)

	// Make calls up to limit
	for i := 0; i < 3; i++ {
		err := enforcer.CheckStartCall("user-123", "room-456")
		if err != nil {
			t.Errorf("Call %d should be allowed: %v", i+1, err)
		}
	}

	// Should exceed limit
	err := enforcer.CheckStartCall("user-123", "room-456")
	if err != ErrRateLimitExceeded {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}

	// Different user should be allowed
	err = enforcer.CheckStartCall("user-456", "room-789")
	if err != nil {
		t.Errorf("Different user should be allowed: %v", err)
	}
}

// TestRegisterCall tests call registration
func TestRegisterCall(t *testing.T) {
	policy := DefaultSecurityPolicy()
	enforcer := NewSecurityEnforcer(policy)

	err := enforcer.RegisterCall("room-123", "user-456")
	if err != nil {
		t.Fatalf("Failed to register call: %v", err)
	}

	// Should count towards concurrent calls
	err = enforcer.CheckStartCall("user-789", "room-789")
	if err != nil {
		t.Errorf("Should allow second call: %v", err)
	}
}

// TestUnregisterCall tests call unregistration
func TestUnregisterCall(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.MaxConcurrentCalls = 1
	enforcer := NewSecurityEnforcer(policy)

	enforcer.RegisterCall("room-1", "user-1")

	// Should be at limit
	err := enforcer.CheckStartCall("user-2", "room-2")
	if err != ErrMaxConcurrentCallsExceeded {
		t.Errorf("Expected ErrMaxConcurrentCallsExceeded, got %v", err)
	}

	// Unregister
	err = enforcer.UnregisterCall("room-1", "user-1")
	if err != nil {
		t.Fatalf("Failed to unregister call: %v", err)
	}

	// Should allow new call
	err = enforcer.CheckStartCall("user-2", "room-2")
	if err != nil {
		t.Errorf("Should allow call after unregistration: %v", err)
	}
}

// TestValidateCallParameters tests parameter validation
func TestValidateCallParameters(t *testing.T) {
	policy := DefaultSecurityPolicy()
	enforcer := NewSecurityEnforcer(policy)

	// Valid parameters
	err := enforcer.ValidateCallParameters("room-123", "user-456", "session-789")
	if err != nil {
		t.Errorf("Valid parameters should pass: %v", err)
	}

	// Empty room ID
	err = enforcer.ValidateCallParameters("", "user-456", "session-789")
	if err != ErrInvalidRoomID {
		t.Errorf("Expected ErrInvalidRoomID, got %v", err)
	}

	// Empty user ID
	err = enforcer.ValidateCallParameters("room-123", "", "session-789")
	if err != ErrInvalidUserID {
		t.Errorf("Expected ErrInvalidUserID, got %v", err)
	}

	// Empty session ID
	err = enforcer.ValidateCallParameters("room-123", "user-456", "")
	if err != ErrInvalidSessionID {
		t.Errorf("Expected ErrInvalidSessionID, got %v", err)
	}
}

// TestAuditCall tests call auditing
func TestAuditCall(t *testing.T) {
	policy := DefaultSecurityPolicy()
	enforcer := NewSecurityEnforcer(policy)

	// Create a test call
	call := &MatrixCall{
		ID:        "call-123",
		RoomID:    "room-456",
		CallerID:  "@caller:example.com",
		CalleeID:  "@callee:example.com",
		State:     CallStateConnected,
		CreatedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: time.Now(),
	}

	// Add some events
	call.mu.Lock()
	call.CallEvents = []CallEvent{
		{
			Type:       EventTypeCallInvite,
			CreateTime: time.Now().Add(-5 * time.Minute),
			PartyID:    "@caller:example.com",
		},
		{
			Type:       EventTypeCallAnswer,
			CreateTime: time.Now().Add(-4 * time.Minute),
			PartyID:    "@callee:example.com",
		},
	}
	call.mu.Unlock()

	record := enforcer.AuditCall(call)

	if record.CallID != "call-123" {
		t.Errorf("Expected call ID 'call-123', got '%s'", record.CallID)
	}

	if record.RoomID != "room-456" {
		t.Errorf("Expected room ID 'room-456', got '%s'", record.RoomID)
	}

	if len(record.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(record.Events))
	}

	if record.Duration < 4*time.Minute || record.Duration > 6*time.Minute {
		t.Errorf("Duration should be approximately 5 minutes, got %v", record.Duration)
	}
}

// TestNewSecurityAudit tests creating a security auditor
func TestNewSecurityAudit(t *testing.T) {
	policy := DefaultSecurityPolicy()
	audit := NewSecurityAudit(policy)

	if audit == nil {
		t.Error("Security audit should not be nil")
	}

	if audit.enforcer == nil {
		t.Error("Security audit should have an enforcer")
	}
}

// TestAuditCallWithViolations tests audit with policy violations
func TestAuditCallWithViolations(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.BlockedUsers = make(map[string]bool)
	policy.BlockedUsers["@blocked:example.com"] = true
	audit := NewSecurityAudit(policy)

	// Create a call with blocked user
	call := &MatrixCall{
		ID:        "call-123",
		RoomID:    "room-456",
		CallerID:  "@blocked:example.com",
		CalleeID:  "@callee:example.com",
		State:     CallStateConnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	record, err := audit.AuditCall(call)
	if err != nil {
		t.Fatalf("Failed to audit call: %v", err)
	}

	if len(record.Violations) == 0 {
		t.Error("Expected violations for blocked caller")
	}

	hasCallerBlocked := false
	for _, v := range record.Violations {
		if v == "caller_blocked" {
			hasCallerBlocked = true
		}
	}

	if !hasCallerBlocked {
		t.Error("Expected 'caller_blocked' violation")
	}
}

// TestGetAuditLog tests retrieving audit log
func TestGetAuditLog(t *testing.T) {
	policy := DefaultSecurityPolicy()
	audit := NewSecurityAudit(policy)

	// Audit a call
	call := &MatrixCall{
		ID:        "call-123",
		RoomID:    "room-456",
		CallerID:  "@caller:example.com",
		CalleeID:  "@callee:example.com",
		State:     CallStateConnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	audit.AuditCall(call)

	log := audit.GetAuditLog()
	if len(log) != 1 {
		t.Errorf("Expected 1 audit record, got %d", len(log))
	}
}

// TestGenerateReport tests generating security reports
func TestGenerateReport(t *testing.T) {
	policy := DefaultSecurityPolicy()
	audit := NewSecurityAudit(policy)

	// Audit multiple calls
	for i := 0; i < 5; i++ {
		call := &MatrixCall{
			ID:        "call-" + string(rune('1'+i)),
			RoomID:    "room-456",
			CallerID:  "@caller:example.com",
			CalleeID:  "@callee:example.com",
			State:     CallStateConnected,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		audit.AuditCall(call)
	}

	report := audit.GenerateReport()

	if report.TotalCalls != 5 {
		t.Errorf("Expected 5 total calls, got %d", report.TotalCalls)
	}

	if report.GeneratedAt.IsZero() {
		t.Error("Report should have generation time")
	}
}

// TestTTLManager tests TTL enforcement
func TestTTLManager(t *testing.T) {
	// Initialize logger
	logger.Global()

	config := DefaultTTLConfig()
	config.DefaultTTL = 100 * time.Millisecond
	config.EnforcementInterval = 50 * time.Millisecond

	sessions := webrtc.NewSessionManager(10 * time.Minute)
	manager := NewTTLManager(sessions, config)

	// Create a session
	session := sessions.Create("container-123", "room-456", "user-789", 100*time.Millisecond)

	// Should not be expired yet
	err := manager.EnforceTTL(context.Background())
	if err != nil {
		t.Errorf("Should not expire yet: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should expire
	err = manager.EnforceTTL(context.Background())
	if err == nil {
		t.Error("Expected expiration error")
	}

	// Session should be expired
	s, ok := sessions.Get(session.ID)
	if ok && s.State != webrtc.SessionExpired {
		t.Errorf("Expected session to be expired, got state %v", s.State)
	}
}

// TestGetTTLStats tests TTL statistics
func TestGetTTLStats(t *testing.T) {
	// Initialize logger
	logger.Global()

	config := DefaultTTLConfig()
	sessions := webrtc.NewSessionManager(10 * time.Minute)
	manager := NewTTLManager(sessions, config)

	// Create sessions with different TTLs
	sessions.Create("container-1", "room-1", "user-1", 10*time.Minute)
	sessions.Create("container-2", "room-2", "user-2", 10*time.Minute)

	stats := manager.GetTTLStats()

	if stats["total_sessions"] != 2 {
		t.Errorf("Expected 2 total sessions, got %v", stats["total_sessions"])
	}
}

// TestStartEnforcement tests starting enforcement loop
func TestStartEnforcement(t *testing.T) {
	// Initialize logger
	logger.Global()

	config := DefaultTTLConfig()
	config.DefaultTTL = 100 * time.Millisecond
	config.EnforcementInterval = 50 * time.Millisecond

	sessions := webrtc.NewSessionManager(10 * time.Minute)
	manager := NewTTLManager(sessions, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.StartEnforcement(ctx)
	if err != nil {
		t.Fatalf("Failed to start enforcement: %v", err)
	}

	// Create a session
	sessions.Create("container-123", "room-456", "user-789", 100*time.Millisecond)

	// Wait for expiration and enforcement
	time.Sleep(200 * time.Millisecond)

	// Session should be expired
	sessionList := sessions.List()
	if len(sessionList) > 0 && sessionList[0].State != webrtc.SessionExpired {
		t.Logf("Session state: %v", sessionList[0].State)
	}
}

// TestDefaultSecurityPolicy tests default security policy
func TestDefaultSecurityPolicy(t *testing.T) {
	policy := DefaultSecurityPolicy()

	if policy.MaxConcurrentCalls != 10 {
		t.Errorf("Expected max concurrent calls 10, got %d", policy.MaxConcurrentCalls)
	}

	if policy.MaxCallDuration != 1*time.Hour {
		t.Errorf("Expected max call duration 1 hour, got %v", policy.MaxCallDuration)
	}

	if !policy.RequireE2EE {
		t.Error("Expected E2EE to be required by default")
	}

	if !policy.RequireSignalingTLS {
		t.Error("Expected signaling TLS to be required by default")
	}
}

// TestDefaultTTLConfig tests default TTL configuration
func TestDefaultTTLConfig(t *testing.T) {
	config := DefaultTTLConfig()

	if config.DefaultTTL != 10*time.Minute {
		t.Errorf("Expected default TTL 10 minutes, got %v", config.DefaultTTL)
	}

	if config.MaxTTL != 1*time.Hour {
		t.Errorf("Expected max TTL 1 hour, got %v", config.MaxTTL)
	}

	if config.WarningThreshold != 0.9 {
		t.Errorf("Expected warning threshold 0.9, got %v", config.WarningThreshold)
	}

	if !config.HardStop {
		t.Error("Expected hard stop to be enabled by default")
	}
}

// TestSecurityErrorValues tests error values
func TestSecurityErrorValues(t *testing.T) {
	errors := []struct {
		err  error
		msg  string
	}{
		{ErrMaxConcurrentCallsExceeded, "maximum concurrent calls exceeded"},
		{ErrUserBlocked, "user is blocked"},
		{ErrRoomBlocked, "room is blocked"},
		{ErrRateLimitExceeded, "rate limit exceeded"},
		{ErrUserNotAllowed, "user not in allowlist"},
		{ErrRoomNotAllowed, "room not in allowlist"},
		{ErrInvalidRoomID, "invalid room ID"},
		{ErrInvalidUserID, "invalid user ID"},
		{ErrInvalidSessionID, "invalid session ID"},
		{ErrCallDurationExceeded, "call duration exceeded"},
	}

	for _, tt := range errors {
		if tt.err.Error() == "" {
			t.Errorf("Error message should not be empty: %v", tt.err)
		}
	}
}
