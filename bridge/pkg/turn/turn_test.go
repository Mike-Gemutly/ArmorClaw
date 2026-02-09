// Package turn tests for TURN/STUN integration and NAT traversal
package turn

import (
	"testing"
	"time"
)

// TestGenerateTURNCredentials tests TURN credential generation
func TestGenerateTURNCredentials(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{
				Host:     "turn.example.com",
				Port:     3478,
				Protocol: "udp",
				Realm:    "example",
			},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)
	defer manager.Stop()

	sessionID := "test-session-123"
	ttl := 5 * time.Minute

	creds, err := manager.GenerateTURNCredentials(sessionID, ttl)
	if err != nil {
		t.Fatalf("Failed to generate credentials: %v", err)
	}

	if len(creds) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(creds))
	}

	cred := creds[0]

	// Verify username format: <expiry>:<session_id>
	if cred.Username == "" {
		t.Error("Username should not be empty")
	}

	// Should contain session ID
	if !contains(cred.Username, sessionID) {
		t.Error("Username should contain session ID")
	}

	// Verify password
	if cred.Password == "" {
		t.Error("Password should not be empty")
	}

	// Verify URLs
	if cred.TURNServer != "turn:turn.example.com:3478" {
		t.Errorf("Expected 'turn:turn.example.com:3478', got '%s'", cred.TURNServer)
	}

	if cred.STUNServer != "stun:turn.example.com:3478" {
		t.Errorf("Expected 'stun:turn.example.com:3478', got '%s'", cred.STUNServer)
	}

	// Verify expiration (should be ~5 minutes from now)
	remaining := time.Until(cred.Expires)
	if remaining < 4*time.Minute || remaining > 6*time.Minute {
		t.Errorf("Expected expiration around 5 minutes, got %v", remaining)
	}
}

// TestValidateTURNCredentials tests TURN credential validation
func TestValidateTURNCredentials(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn.example.com", Port: 3478, Protocol: "udp"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)
	defer manager.Stop()

	sessionID := "test-session-123"

	// Generate credentials
	creds, _ := manager.GenerateTURNCredentials(sessionID, 5*time.Minute)
	cred := creds[0]

	// Validate correct credentials
	validateSessionID, err := manager.ValidateTURNCredentials(cred.Username, cred.Password)
	if err != nil {
		t.Errorf("Failed to validate correct credentials: %v", err)
	}

	if validateSessionID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, validateSessionID)
	}

	// Validate incorrect password
	_, err = manager.ValidateTURNCredentials(cred.Username, "wrong-password")
	if err != ErrTURNInvalidPassword {
		t.Errorf("Expected ErrTURNInvalidPassword, got %v", err)
	}

	// Validate non-existent username
	_, err = manager.ValidateTURNCredentials("non-existent", "password")
	if err != ErrTURNInvalidCredentials {
		t.Errorf("Expected ErrTURNInvalidCredentials, got %v", err)
	}
}

// TestTURNCredentialsExpiry tests TURN credential expiration
func TestTURNCredentialsExpiry(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn.example.com", Port: 3478, Protocol: "udp"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)
	defer manager.Stop()

	// Generate credentials with very short TTL
	creds, _ := manager.GenerateTURNCredentials("test-session", 50*time.Millisecond)
	cred := creds[0]

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err := manager.ValidateTURNCredentials(cred.Username, cred.Password)
	if err != ErrTURNExpired {
		t.Errorf("Expected ErrTURNExpired, got %v", err)
	}
}

// TestCleanupExpired tests cleanup of expired credentials
func TestCleanupExpired(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn.example.com", Port: 3478, Protocol: "udp"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)

	// Generate credentials with short TTL
	_, _ = manager.GenerateTURNCredentials("test-session-1", 50*time.Millisecond)
	_, _ = manager.GenerateTURNCredentials("test-session-2", 50*time.Millisecond)

	// Stats should show 2 credentials
	stats := manager.GetStats()
	if stats["total_credentials"].(int) != 2 {
		t.Errorf("Expected 2 credentials, got %d", stats["total_credentials"])
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	manager.CleanupExpired()

	// Stats should show 0 credentials
	stats = manager.GetStats()
	if stats["total_credentials"].(int) != 0 {
		t.Errorf("Expected 0 credentials after cleanup, got %d", stats["total_credentials"])
	}
}

// TestGetStats tests getting TURN manager statistics
func TestGetStats(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn1.example.com", Port: 3478, Protocol: "udp"},
			{Host: "turn2.example.com", Port: 3478, Protocol: "udp"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)
	defer manager.Start()
	defer manager.Stop()

	// Generate some credentials
	_, _ = manager.GenerateTURNCredentials("session-1", 5*time.Minute)
	_, _ = manager.GenerateTURNCredentials("session-2", 5*time.Minute)
	_, _ = manager.GenerateTURNCredentials("session-3", 30*time.Second) // Expiring soon

	stats := manager.GetStats()

	if stats["total_credentials"].(int) != 3 {
		t.Errorf("Expected 3 credentials, got %d", stats["total_credentials"])
	}

	if stats["servers"].(int) != 2 {
		t.Errorf("Expected 2 servers, got %d", stats["servers"])
	}
}

// TestMaxTTL tests maximum TTL enforcement
func TestMaxTTL(t *testing.T) {
	config := Config{
		Servers:     []ServerConfig{{Host: "turn.example.com", Port: 3478, Protocol: "udp"}},
		Secret:      "test-secret",
		DefaultTTL:  10 * time.Minute,
		MaxTTL:      30 * time.Minute,
	}

	manager := NewManager(config)
	defer manager.Stop()

	// Request credentials with TTL exceeding max
	creds, _ := manager.GenerateTURNCredentials("test-session", 1*time.Hour)
	cred := creds[0]

	// Should be capped at max TTL
	remaining := time.Until(cred.Expires)
	if remaining > 31*time.Minute || remaining < 29*time.Minute {
		t.Errorf("Expected TTL around 30 minutes (max), got %v", remaining)
	}
}

// TestCreateICEServers tests creating ICE server configurations
func TestCreateICEServers(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "stun.example.com", Port: 3478, Protocol: "udp"},
			{Host: "turn.example.com", Port: 3478, Protocol: "udp", Realm: "example"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	sessionID := "test-session"
	ttl := 5 * time.Minute

	servers, err := CreateICEServers(config, sessionID, ttl)
	if err != nil {
		t.Fatalf("Failed to create ICE servers: %v", err)
	}

	// Should have STUN + TURN servers
	if len(servers) < 2 {
		t.Errorf("Expected at least 2 servers (STUN + TURN), got %d", len(servers))
	}

	// Find STUN server
	foundSTUN := false
	foundTURN := false

	for _, server := range servers {
		if len(server.URLs) > 0 {
			url := server.URLs[0]
			if strings.HasPrefix(url, "stun:") {
				foundSTUN = true
				// STUN should not have username/password
				if server.Username != "" || server.Credential != "" {
					t.Error("STUN server should not have credentials")
				}
			} else if strings.HasPrefix(url, "turn:") {
				foundTURN = true
				// TURN should have username/password
				if server.Username == "" || server.Credential == "" {
					t.Error("TURN server should have credentials")
				}
			}
		}
	}

	if !foundSTUN {
		t.Error("Should have STUN server")
	}

	if !foundTURN {
		t.Error("Should have TURN server")
	}
}

// TestICECandidatePriority tests ICE candidate priority calculation
func TestICECandidatePriority(t *testing.T) {
	tests := []struct {
		candidateType ICECandidateType
		localPref     int
		componentID   int
		expected      uint32
	}{
		{ICECandidateHost, 65535, 1, 2130706431},
		{ICECandidateSRFLX, 65535, 1, 1694498871},
		{ICECandidatePRFLX, 65535, 1, 1865465391},
		{ICECandidateRelay, 65535, 1, 16777215},
	}

	for _, tt := range tests {
		t.Run(tt.candidateType.String(), func(t *testing.T) {
			priority := ICECandidatePriority(tt.candidateType, tt.localPref, tt.componentID)
			if priority != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, priority)
			}
		})
	}
}

// TestParseICECandidate tests parsing ICE candidate strings
func TestParseICECandidate(t *testing.T) {
	// Valid host candidate
	candidateStr := "candidate:1 1 udp 2130706431 192.168.1.100 54321 typ host"

	candidate, err := ParseICECandidate(candidateStr)
	if err != nil {
		t.Fatalf("Failed to parse candidate: %v", err)
	}

	if candidate.Foundation != "1" {
		t.Errorf("Expected foundation '1', got '%s'", candidate.Foundation)
	}

	if candidate.ComponentID != 1 {
		t.Errorf("Expected component ID 1, got %d", candidate.ComponentID)
	}

	if candidate.Transport != "udp" {
		t.Errorf("Expected transport 'udp', got '%s'", candidate.Transport)
	}

	if candidate.Priority != 2130706431 {
		t.Errorf("Expected priority 2130706431, got %d", candidate.Priority)
	}

	if candidate.Address != "192.168.1.100" {
		t.Errorf("Expected address '192.168.1.100', got '%s'", candidate.Address)
	}

	if candidate.Port != 54321 {
		t.Errorf("Expected port 54321, got %d", candidate.Port)
	}

	if candidate.Type != "host" {
		t.Errorf("Expected type 'host', got '%s'", candidate.Type)
	}
}

// TestICECandidate_String tests string representation of ICE candidate
func TestICECandidate_String(t *testing.T) {
	candidate := &ICECandidate{
		Foundation:  "1",
		ComponentID: 1,
		Transport:   "udp",
		Priority:    2130706431,
		Address:     "192.168.1.100",
		Port:        54321,
		Type:        "host",
		Protocol:    "udp",
	}

	str := candidate.String()
	expected := "candidate:1 1 udp 2130706431 192.168.1.100 54321 host"

	if str != expected {
		t.Errorf("Expected '%s', got '%s'", expected, str)
	}
}

// TestICECandidateWithRelatedAddress tests candidate with related address
func TestICECandidateWithRelatedAddress(t *testing.T) {
	candidateStr := "candidate:2 2 udp 1694498815 203.0.113.5 62345 typ srflx raddr 192.168.1.100 rport 54321"

	candidate, err := ParseICECandidate(candidateStr)
	if err != nil {
		t.Fatalf("Failed to parse candidate: %v", err)
	}

	if candidate.RelatedAddress != "192.168.1.100" {
		t.Errorf("Expected related address '192.168.1.100', got '%s'", candidate.RelatedAddress)
	}

	if candidate.RelatedPort != 54321 {
		t.Errorf("Expected related port 54321, got %d", candidate.RelatedPort)
	}
}

// TestICEGatherer tests ICE gathering
func TestICEGatherer(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "stun.example.com", Port: 3478, Protocol: "udp"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	gatherer := NewICEGatherer(config)

	// Gather host candidates
	candidates, err := gatherer.GatherHostCandidates()
	if err != nil {
		t.Fatalf("Failed to gather host candidates: %v", err)
	}

	// Should have at least one candidate
	if len(candidates) == 0 {
		t.Error("Should have at least one host candidate")
	}

	// Verify all candidates are host type
	for _, candidate := range candidates {
		if candidate.Type != "host" {
			t.Errorf("Expected host candidate type, got '%s'", candidate.Type)
		}
	}
}

// TestSTUNMessage tests STUN message serialization
func TestSTUNMessage(t *testing.T) {
	transactionID := GenerateTransactionID()

	msg := NewSTUNMessage(STUNMethodBinding, transactionID)

	// Add attribute
	msg.Attributes = append(msg.Attributes, STUNAttribute{
		Type:   STUNAttrUsername,
		Length: 8,
		Value:  []byte("user123"),
	})

	// Serialize
	data := msg.Serialize()

	// Should be at least header length (20)
	if len(data) < 20 {
		t.Errorf("Expected at least 20 bytes, got %d", len(data))
	}

	// Parse back
	parsed, err := ParseSTUNMessage(data)
	if err != nil {
		t.Fatalf("Failed to parse STUN message: %v", err)
	}

	if parsed.Type != STUNMethodBinding {
		t.Errorf("Expected type %d, got %d", STUNMethodBinding, parsed.Type)
	}

	if parsed.TransactionID != transactionID {
		t.Error("Transaction ID mismatch")
	}
}

// TestGenerateTransactionID tests transaction ID generation
func TestGenerateTransactionID(t *testing.T) {
	id1 := GenerateTransactionID()
	id2 := GenerateTransactionID()

	// Should be different (different timestamps)
	if id1 == id2 {
		t.Error("Transaction IDs should be different")
	}

	// Should be 12 bytes
	if len(id1) != 12 {
		t.Errorf("Expected 12 bytes, got %d", len(id1))
	}
}

// TestParseSTUNMessage tests parsing STUN messages
func TestParseSTUNMessage(t *testing.T) {
	// Create a simple STUN binding request
	data := make([]byte, 20) // Header only
	PutUint16(data[0:2], STUNMethodBinding)
	PutUint16(data[2:4], 0) // Length
	PutUint32(data[4:8], STUNMagicCookie)

	// Add transaction ID
	tid := GenerateTransactionID()
	copy(data[8:20], tid[:])

	msg, err := ParseSTUNMessage(data)
	if err != nil {
		t.Fatalf("Failed to parse STUN message: %v", err)
	}

	if msg.Type != STUNMethodBinding {
		t.Errorf("Expected type %d, got %d", STUNMethodBinding, msg.Type)
	}

	if msg.MagicCookie != STUNMagicCookie {
		t.Error("Magic cookie mismatch")
	}

	if msg.TransactionID != tid {
		t.Error("Transaction ID mismatch")
	}
}

// TestTURNProtocol tests different TURN protocol types
func TestTURNProtocol(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn.example.com", Port: 3478, Protocol: "udp"},
			{Host: "turn.example.com", Port: 5349, Protocol: "tcp"},
			{Host: "turn.example.com", Port: 5350, Protocol: "tls"},
		},
		Secret:     "test-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	manager := NewManager(config)
	defer manager.Stop()

	creds, _ := manager.GenerateTURNCredentials("test-session", 5*time.Minute)

	if len(creds) != 3 {
		t.Errorf("Expected 3 credentials, got %d", len(creds))
	}

	// Check UDP TURN URL
	if creds[0].TURNServer != "turn:turn.example.com:3478" {
		t.Errorf("Expected 'turn:turn.example.com:3478', got '%s'", creds[0].TURNServer)
	}

	// Check TCP TURN URL
	if creds[1].TURNServer != "turn:turn.example.com:5349?transport=tcp" {
		t.Errorf("Expected 'turn:turn.example.com:5349?transport=tcp', got '%s'", creds[1].TURNServer)
	}

	// Check TLS TURN URL
	if creds[2].TURNServer != "turns:turn.example.com:5350" {
		t.Errorf("Expected 'turns:turn.example.com:5350', got '%s'", creds[2].TURNServer)
	}
}

// TestIntegration_TURNLifecycle tests complete TURN credential lifecycle
func TestIntegration_TURNLifecycle(t *testing.T) {
	config := Config{
		Servers: []ServerConfig{
			{Host: "turn.example.com", Port: 3478, Protocol: "udp", Realm: "example"},
		},
		Secret:     "test-secret-key",
		DefaultTTL: 1 * time.Minute,
		MaxTTL:     5 * time.Minute,
	}

	manager := NewManager(config)
	manager.Start()
	defer manager.Stop()

	sessionID := "test-session-lifecycle"

	// 1. Generate credentials
	creds, err := manager.GenerateTURNCredentials(sessionID, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to generate credentials: %v", err)
	}

	cred := creds[0]

	// 2. Validate credentials
	validatedSessionID, err := manager.ValidateTURNCredentials(cred.Username, cred.Password)
	if err != nil {
		t.Fatalf("Failed to validate credentials: %v", err)
	}

	if validatedSessionID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, validatedSessionID)
	}

	// 3. Check stats
	stats := manager.GetStats()
	if stats["total_credentials"].(int) != 1 {
		t.Errorf("Expected 1 credential, got %d", stats["total_credentials"])
	}

	// 4. Wait for expiration (if using short TTL)
	// Note: We use 30 second TTL, so we won't wait for this in the test

	// 5. Get ICE servers for WebRTC
	iceServers, err := CreateICEServers(config, sessionID, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to create ICE servers: %v", err)
	}

	if len(iceServers) == 0 {
		t.Error("Should have at least one ICE server")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfString(s, substr) >= 0
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
