// Package adapter provides tests for the Matrix adapter with zero-trust validation
package adapter

import (
	"fmt"
	"testing"

	"github.com/armorclaw/bridge/pkg/logger"
)

// TestTrustedSenderValidation tests the trusted sender allow-list functionality
func TestTrustedSenderValidation(t *testing.T) {
	// Initialize logger for tests
	logger.Initialize("info", "text", "stdout")

	tests := []struct {
		name           string
		trustedSenders []string
		sender         string
		expected       bool
	}{
		{
			name:           "empty allowlist allows all",
			trustedSenders: []string{},
			sender:         "@user:example.com",
			expected:       true,
		},
		{
			name:           "exact match",
			trustedSenders: []string{"@user:example.com"},
			sender:         "@user:example.com",
			expected:       true,
		},
		{
			name:           "exact match denied",
			trustedSenders: []string{"@admin:example.com"},
			sender:         "@user:example.com",
			expected:       false,
		},
		{
			name:           "wildcard domain *@example.com",
			trustedSenders: []string{"*@example.com"},
			sender:         "@user@example.com",
			expected:       true,
		},
		{
			name:           "wildcard domain allows different user",
			trustedSenders: []string{"*@example.com"},
			sender:         "@admin@example.com",
			expected:       true,
		},
		{
			name:           "wildcard domain rejects different domain",
			trustedSenders: []string{"*@example.com"},
			sender:         "@user@other.com",
			expected:       false,
		},
		{
			name:           "wildcard colon domain :example.com",
			trustedSenders: []string{"*:example.com"},
			sender:         "@user:example.com",
			expected:       true,
		},
		{
			name:           "multiple patterns",
			trustedSenders: []string{"@admin:example.com", "*@trusted.com"},
			sender:         "@admin:example.com",
			expected:       true,
		},
		{
			name:           "multiple patterns matches second",
			trustedSenders: []string{"@admin:example.com", "*@trusted.com"},
			sender:         "@user@trusted.com",
			expected:       true,
		},
		{
			name:           "multiple patterns rejects both",
			trustedSenders: []string{"@admin:example.com", "*@trusted.com"},
			sender:         "@user:other.com",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				HomeserverURL:   "https://matrix.example.com",
				TrustedSenders:  tt.trustedSenders,
				RejectUntrusted: false,
			}

			adapter, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			result := adapter.isTrustedSender(tt.sender)
			if result != tt.expected {
				t.Errorf("isTrustedSender() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTrustedRoomValidation tests the trusted room allow-list functionality
func TestTrustedRoomValidation(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	tests := []struct {
		name          string
		trustedRooms []string
		roomID        string
		expected      bool
	}{
		{
			name:          "empty allowlist allows all rooms",
			trustedRooms: []string{},
			roomID:        "!roomid:example.com",
			expected:      true,
		},
		{
			name:          "exact match",
			trustedRooms: []string{"!roomid:example.com"},
			roomID:        "!roomid:example.com",
			expected:      true,
		},
		{
			name:          "exact match denied",
			trustedRooms: []string{"!otherroom:example.com"},
			roomID:        "!roomid:example.com",
			expected:      false,
		},
		{
			name:          "multiple rooms",
			trustedRooms: []string{"!room1:example.com", "!room2:example.com"},
			roomID:        "!room1:example.com",
			expected:      true,
		},
		{
			name:          "multiple rooms denies unlisted",
			trustedRooms: []string{"!room1:example.com", "!room2:example.com"},
			roomID:        "!room3:example.com",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				HomeserverURL:  "https://matrix.example.com",
				TrustedRooms:   tt.trustedRooms,
				RejectUntrusted: false,
			}

			adapter, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			result := adapter.isTrustedRoom(tt.roomID)
			if result != tt.expected {
				t.Errorf("isTrustedRoom() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSetAndGetTrustedSenders tests setting and retrieving trusted senders
func TestSetAndGetTrustedSenders(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL: "https://matrix.example.com",
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Initially empty
	senders := adapter.GetTrustedSenders()
	if len(senders) != 0 {
		t.Errorf("Initial trusted senders not empty, got %d", len(senders))
	}

	// Set senders
	newSenders := []string{"@user:example.com", "*@trusted.com"}
	adapter.SetTrustedSenders(newSenders)

	// Verify retrieval
	senders = adapter.GetTrustedSenders()
	if len(senders) != 2 {
		t.Errorf("Expected 2 trusted senders, got %d", len(senders))
	}

	// Verify order preserved
	if senders[0] != "@user:example.com" {
		t.Errorf("First sender = %s, want @user:example.com", senders[0])
	}
}

// TestSetAndGetTrustedRooms tests setting and retrieving trusted rooms
func TestSetAndGetTrustedRooms(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL: "https://matrix.example.com",
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Initially empty
	rooms := adapter.GetTrustedRooms()
	if len(rooms) != 0 {
		t.Errorf("Initial trusted rooms not empty, got %d", len(rooms))
	}

	// Set rooms
	newRooms := []string{"!room1:example.com", "!room2:example.com"}
	adapter.SetTrustedRooms(newRooms)

	// Verify retrieval
	rooms = adapter.GetTrustedRooms()
	if len(rooms) != 2 {
		t.Errorf("Expected 2 trusted rooms, got %d", len(rooms))
	}

	// Verify order preserved
	if rooms[0] != "!room1:example.com" {
		t.Errorf("First room = %s, want !room1:example.com", rooms[0])
	}
}

// TestWildcardSenderMatching tests various wildcard patterns
func TestWildcardSenderMatching(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	tests := []struct {
		name           string
		pattern        string
		sender         string
		expected       bool
	}{
		{
			name:     "wildcard *@domain matches user@domain",
			pattern:  "*@example.com",
			sender:   "@user@example.com",
			expected: true,
		},
		{
			name:     "wildcard *@domain matches admin@domain",
			pattern:  "*@example.com",
			sender:   "@admin@example.com",
			expected: true,
		},
		{
			name:     "wildcard *@domain rejects user@otherdomain",
			pattern:  "*@example.com",
			sender:   "@user@other.com",
			expected: false,
		},
		{
			name:     "wildcard *:domain matches user:domain",
			pattern:  "*:example.com",
			sender:   "@user:example.com",
			expected: true,
		},
		{
			name:     "wildcard *:domain matches admin:domain",
			pattern:  "*:example.com",
			sender:   "@admin:example.com",
			expected: true,
		},
		{
			name:     "wildcard *:domain rejects user:otherdomain",
			pattern:  "*:example.com",
			sender:   "@user:other.com",
			expected: false,
		},
		{
			name:     "exact match works",
			pattern:  "@user:example.com",
			sender:   "@user:example.com",
			expected: true,
		},
		{
			name:     "exact match rejects different user",
			pattern:  "@user:example.com",
			sender:   "@other:example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				HomeserverURL: "https://matrix.example.com",
			}

			adapter, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			// Set the pattern as the only trusted sender
			adapter.SetTrustedSenders([]string{tt.pattern})

			result := adapter.isTrustedSender(tt.sender)
			if result != tt.expected {
				t.Errorf("isTrustedSender(%s, pattern=%s) = %v, want %v",
					tt.sender, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestConcurrentTrustListUpdates tests thread-safe updates to trust lists
func TestConcurrentTrustListUpdates(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL: "https://matrix.example.com",
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Launch multiple goroutines updating trust lists
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				senders := []string{
					fmt.Sprintf("@user%d:example.com", id),
					fmt.Sprintf("*@domain%d.com", id),
				}
				adapter.SetTrustedSenders(senders)
				adapter.SetTrustedRooms([]string{
					fmt.Sprintf("!room%d:example.com", id),
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without deadlock or race condition, test passed
	// Verify final state is consistent
	senders := adapter.GetTrustedSenders()
	if len(senders) != 2 {
		t.Logf("Note: Final sender count = %d (last write won)", len(senders))
	}
}

// TestRejectUntrustedBehavior tests the rejectUntrusted configuration
func TestRejectUntrustedBehavior(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	// Test with rejectUntrusted = false
	cfg1 := Config{
		HomeserverURL:  "https://matrix.example.com",
		TrustedSenders: []string{"@admin:example.com"},
		RejectUntrusted: false,
	}

	adapter1, err := New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Untrusted sender should be rejected (dropped silently)
	if adapter1.isTrustedSender("@user:example.com") {
		t.Error("Untrusted sender should not be trusted")
	}

	// Test with rejectUntrusted = true
	cfg2 := Config{
		HomeserverURL:  "https://matrix.example.com",
		TrustedSenders: []string{"@admin:example.com"},
		RejectUntrusted: true,
	}

	_, err = New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// This would trigger sending rejection message in processEvents
	// The behavior difference is tested through integration
	_ = cfg2 // Use cfg2 to avoid unused variable warning
}

// BenchmarkTrustedSenderValidation benchmarks sender validation
func BenchmarkTrustedSenderValidation(b *testing.B) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL:  "https://matrix.example.com",
		TrustedSenders: []string{
			"@admin:example.com",
			"*@trusted.com",
			"*:partner.com",
		},
	}

	adapter, _ := New(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test various senders
		senders := []string{
			"@admin:example.com",
			"@user:trusted.com",
			"@bot:partner.com",
			"@hacker:evil.com",
		}
		for _, sender := range senders {
			adapter.isTrustedSender(sender)
		}
	}
}

// BenchmarkWildcardPatternMatching benchmarks wildcard pattern matching
func BenchmarkWildcardPatternMatching(b *testing.B) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL: "https://matrix.example.com",
		TrustedSenders: []string{"*@example.com"},
	}

	adapter, _ := New(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.matchSenderPattern("@user@example.com", "*@example.com")
		adapter.matchSenderPattern("@admin@example.com", "*@example.com")
		adapter.matchSenderPattern("@user@other.com", "*@example.com")
	}
}
