// Package voice tests for budget enforcement
package voice

import (
	"context"
	"testing"
	"time"
)

// TestNewBudgetTracker tests creating a budget tracker
func TestNewBudgetTracker(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	if tracker == nil {
		t.Error("Budget tracker should not be nil")
	}

	if tracker.config.DefaultTokenLimit != 100000 {
		t.Errorf("Expected default token limit 100000, got %d", tracker.config.DefaultTokenLimit)
	}
}

// TestStartSession tests starting a voice session
func TestStartSession(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	session, err := tracker.StartSession("test-session", "call-123", "room-456", 50000, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	if session.SessionID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", session.SessionID)
	}

	if session.CallID != "call-123" {
		t.Errorf("Expected call ID 'call-123', got '%s'", session.CallID)
	}

	if session.Type != SessionTypeVoice {
		t.Errorf("Expected session type %v, got %v", SessionTypeVoice, session.Type)
	}
}

// TestRecordTokenUsage tests recording token consumption
func TestRecordTokenUsage(t *testing.T) {
	config := DefaultConfig()
	config.HardStop = false // Don't hard stop in tests
	tracker := NewBudgetTracker(config)

	_, err := tracker.StartSession("test-session", "call-123", "room-456", 1000, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Record usage
	err = tracker.RecordTokenUsage("test-session", 100, 200, "gpt-4")
	if err != nil {
		t.Fatalf("Failed to record token usage: %v", err)
	}

	usage, duration, err := tracker.GetUsage("test-session")
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.InputTokens != 100 {
		t.Errorf("Expected 100 input tokens, got %d", usage.InputTokens)
	}

	if usage.OutputTokens != 200 {
		t.Errorf("Expected 200 output tokens, got %d", usage.OutputTokens)
	}

	if usage.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", usage.Model)
	}

	if usage.Requests != 1 {
		t.Errorf("Expected 1 request, got %d", usage.Requests)
	}

	// Duration should be >= 0 (may be 0 if very fast)
	if duration < 0 {
		t.Errorf("Expected non-negative duration, got %v", duration)
	}
}

// TestTokenLimitWarning tests token limit warning
func TestTokenLimitWarning(t *testing.T) {
	config := DefaultConfig()
	config.WarningThreshold = 0.8
	config.HardStop = false
	tracker := NewBudgetTracker(config)

	// Set low limit for testing
	_, err := tracker.StartSession("test-session", "call-123", "room-456", 1000, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Record usage that exceeds warning threshold (80% of 1000 = 800)
	err = tracker.RecordTokenUsage("test-session", 400, 401, "gpt-4")
	if err != nil {
		t.Fatalf("Failed to record token usage: %v", err)
	}

	// Should trigger warning but not error
	usage, _, _ := tracker.GetUsage("test-session")
	totalTokens := usage.InputTokens + usage.OutputTokens
	if totalTokens != 801 {
		t.Errorf("Expected 801 total tokens, got %d", totalTokens)
	}
}

// TestTokenLimitExceeded tests hard stop on token limit
func TestTokenLimitExceeded(t *testing.T) {
	config := DefaultConfig()
	config.HardStop = true
	tracker := NewBudgetTracker(config)

	// Set low limit for testing
	_, err := tracker.StartSession("test-session", "call-123", "room-456", 1000, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Record usage that exceeds limit
	err = tracker.RecordTokenUsage("test-session", 500, 501, "gpt-4")
	if err != ErrBudgetExceeded {
		t.Errorf("Expected ErrBudgetExceeded, got %v", err)
	}

	// Wait for async session termination
	time.Sleep(100 * time.Millisecond)

	// Session should be deleted after async termination
	_, exists := tracker.GetSession("test-session")
	if exists {
		t.Error("Session should be deleted after exceeding budget")
	}
}

// TestDurationLimit tests duration limit enforcement
func TestDurationLimit(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	// Set very short duration for testing
	_, err := tracker.StartSession("test-session", "call-123", "room-456", 100000, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Wait for duration to exceed
	time.Sleep(150 * time.Millisecond)

	// Check duration
	err = tracker.CheckDuration("test-session")
	if err != ErrDurationExceeded {
		t.Errorf("Expected ErrDurationExceeded, got %v", err)
	}
}

// TestEndSession tests ending a session
func TestEndSession(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	_, err := tracker.StartSession("test-session", "call-123", "room-456", 100000, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Record some usage
	tracker.RecordTokenUsage("test-session", 100, 200, "gpt-4")

	// End session
	err = tracker.EndSession("test-session")
	if err != nil {
		t.Fatalf("Failed to end session: %v", err)
	}

	// Session should be deleted after ending
	_, exists := tracker.GetSession("test-session")
	if exists {
		t.Error("Session should be deleted after ending")
	}

	// Should not be able to record more usage
	err = tracker.RecordTokenUsage("test-session", 10, 20, "gpt-4")
	if err == nil {
		t.Error("Should fail to record usage on closed session")
	}
}

// TestGetAllSessions tests retrieving all active sessions
func TestGetAllSessions(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	// Start multiple sessions
	tracker.StartSession("session-1", "call-1", "room-1", 100000, 10*time.Minute)
	tracker.StartSession("session-2", "call-2", "room-2", 100000, 10*time.Minute)
	tracker.StartSession("session-3", "call-3", "room-3", 100000, 10*time.Minute)

	// End one
	tracker.EndSession("session-2")

	sessions := tracker.GetAllSessions()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 active sessions, got %d", len(sessions))
	}
}

// TestVoiceBudgetGetStats tests budget tracker statistics
func TestVoiceBudgetGetStats(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	// Start sessions and record usage
	tracker.StartSession("session-1", "call-1", "room-1", 100000, 10*time.Minute)
	tracker.RecordTokenUsage("session-1", 100, 200, "gpt-4")
	tracker.RecordTokenUsage("session-1", 50, 100, "gpt-4")

	tracker.StartSession("session-2", "call-2", "room-2", 100000, 10*time.Minute)
	tracker.RecordTokenUsage("session-2", 75, 150, "gpt-4")

	stats := tracker.GetStats()

	if stats["active_sessions"] != 2 {
		t.Errorf("Expected 2 active sessions, got %v", stats["active_sessions"])
	}

	// Total tokens: (100+200+50+100) + (75+150) = 450 + 225 = 675
	if stats["total_tokens_used"] != uint64(675) {
		t.Errorf("Expected 675 total tokens, got %v", stats["total_tokens_used"])
	}
}

// TestEnforceLimits tests enforcement loop
func TestEnforceLimits(t *testing.T) {
	config := DefaultConfig()
	config.HardStop = true
	tracker := NewBudgetTracker(config)

	// Create session with low limit
	tracker.StartSession("test-session", "call-123", "room-456", 100, 1*time.Hour)
	tracker.RecordTokenUsage("test-session", 50, 51, "gpt-4")

	// Run enforcement
	err := tracker.EnforceLimits(context.Background())
	if err != nil {
		t.Fatalf("EnforceLimits failed: %v", err)
	}

	// Wait for async session termination
	time.Sleep(100 * time.Millisecond)

	// Session should be deleted after enforcement
	_, exists := tracker.GetSession("test-session")
	if exists {
		t.Error("Session should be deleted after budget enforcement")
	}
}

// TestSessionStateString tests session state string representation
func TestSessionStateString(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected string
	}{
		{SessionStateUnknown, "unknown"},
		{SessionStateActive, "active"},
		{SessionStateEnded, "ended"},
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

// TestStop tests stopping the budget tracker
func TestStop(t *testing.T) {
	config := DefaultConfig()
	tracker := NewBudgetTracker(config)

	// Start sessions
	tracker.StartSession("session-1", "call-1", "room-1", 100000, 10*time.Minute)
	tracker.StartSession("session-2", "call-2", "room-2", 100000, 10*time.Minute)

	// Stop tracker
	tracker.Stop()

	// All sessions should be ended
	sessions := tracker.GetAllSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 active sessions after stop, got %d", len(sessions))
	}
}
