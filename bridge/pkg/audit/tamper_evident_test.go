package audit

import (
	"testing"
	"time"
)

func TestNewTamperEvidentLog(t *testing.T) {
	config := TamperEvidentConfig{
		Enabled:       true,
		MaxEntries:    1000,
		Logger:        nil,
	}

	log := NewTamperEvidentLog(config)

	if log == nil {
		t.Fatal("Expected non-nil log")
	}

	if log.config.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries 1000, got %d", log.config.MaxEntries)
	}

	if log.lastHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("Expected initial hash to be all zeros")
	}
}

func TestTamperEvidentLogEntry(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	actor := Actor{
		Type:      "user",
		ID:        "user-123",
		Name:      "Test User",
		IPAddress: "192.168.1.1",
	}

	resource := Resource{
		Type: "room",
		ID:   "room-456",
		Name: "Test Room",
	}

	compliance := ComplianceFlags{
		Category:      "access",
		Severity:      "low",
		AuditRequired: false,
	}

	entry, err := log.LogEntry("user_access", actor, "view", resource, nil, compliance)
	if err != nil {
		t.Fatalf("Failed to log entry: %v", err)
	}

	if entry.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", entry.Sequence)
	}

	if entry.EventType != "user_access" {
		t.Errorf("Expected event type 'user_access', got %s", entry.EventType)
	}

	if entry.Hash == "" {
		t.Error("Expected non-empty hash")
	}

	if entry.PreviousHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("Expected initial previous hash")
	}
}

func TestTamperEvidentHashChain(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log multiple entries
	for i := 0; i < 5; i++ {
		_, err := log.LogEntry(
			"test_event",
			Actor{Type: "system", ID: "test"},
			"test_action",
			Resource{Type: "test", ID: "test"},
			nil,
			ComplianceFlags{Category: "test", Severity: "low"},
		)
		if err != nil {
			t.Fatalf("Failed to log entry %d: %v", i, err)
		}
	}

	// Verify chain
	result := log.VerifyChain()
	if !result.Valid {
		t.Errorf("Expected valid chain, got invalid: %v", result.InvalidEntries)
	}

	if result.TotalEntries != 5 {
		t.Errorf("Expected 5 entries, got %d", result.TotalEntries)
	}
}

func TestTamperDetection(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log entries
	for i := 0; i < 3; i++ {
		_, err := log.LogEntry(
			"test_event",
			Actor{Type: "system", ID: "test"},
			"test_action",
			Resource{Type: "test", ID: "test"},
			nil,
			ComplianceFlags{Category: "test", Severity: "low"},
		)
		if err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Tamper with an entry
	log.mu.Lock()
	if len(log.entries) > 1 {
		log.entries[1].Action = "tampered_action"
	}
	log.mu.Unlock()

	// Verify chain should detect tampering
	result := log.VerifyChain()
	if result.Valid {
		t.Error("Expected chain to be invalid due to tampering")
	}

	if len(result.InvalidEntries) == 0 {
		t.Error("Expected invalid entries to be reported")
	}

	if result.TamperedAt == nil {
		t.Error("Expected TamperedAt to be set")
	}
}

func TestTamperEvidentMaxEntriesLimit(t *testing.T) {
	maxEntries := 10
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: maxEntries,
	})

	// Log more entries than max
	for i := 0; i < 20; i++ {
		_, err := log.LogEntry(
			"test_event",
			Actor{Type: "system", ID: "test"},
			"test_action",
			Resource{Type: "test", ID: "test"},
			nil,
			ComplianceFlags{Category: "test", Severity: "low"},
		)
		if err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	log.mu.RLock()
	entryCount := len(log.entries)
	log.mu.RUnlock()

	if entryCount > maxEntries {
		t.Errorf("Expected at most %d entries, got %d", maxEntries, entryCount)
	}
}

func TestGetEntriesWithFilter(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log different types of entries
	log.LogEntry("access", Actor{ID: "user1"}, "view", Resource{ID: "res1"}, nil, ComplianceFlags{Severity: "low"})
	log.LogEntry("access", Actor{ID: "user2"}, "view", Resource{ID: "res2"}, nil, ComplianceFlags{Severity: "high"})
	log.LogEntry("modification", Actor{ID: "user1"}, "edit", Resource{ID: "res1"}, nil, ComplianceFlags{Severity: "medium"})
	log.LogEntry("phi_event", Actor{ID: "user1"}, "access", Resource{ID: "res3"}, nil, ComplianceFlags{PHIInvolved: true, Severity: "high"})

	// Filter by actor
	filter := EntryFilter{ActorID: "user1"}
	entries := log.GetEntries(filter)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries for user1, got %d", len(entries))
	}

	// Filter by severity
	filter = EntryFilter{Severity: "high"}
	entries = log.GetEntries(filter)
	if len(entries) != 2 {
		t.Errorf("Expected 2 high severity entries, got %d", len(entries))
	}

	// Filter PHI only
	filter = EntryFilter{PHIOnly: true}
	entries = log.GetEntries(filter)
	if len(entries) != 1 {
		t.Errorf("Expected 1 PHI entry, got %d", len(entries))
	}

	// Filter with limit
	filter = EntryFilter{Limit: 2}
	entries = log.GetEntries(filter)
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries with limit, got %d", len(entries))
	}
}

func TestExport(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log some entries
	for i := 0; i < 3; i++ {
		_, err := log.LogEntry(
			"test_event",
			Actor{Type: "system", ID: "test"},
			"test_action",
			Resource{Type: "test", ID: "test"},
			nil,
			ComplianceFlags{Category: "test", Severity: "low"},
		)
		if err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Export as JSON
	data, err := log.Export("json")
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty export data")
	}

	// Verify it's valid JSON (basic check)
	if string(data[0:1]) != "{" {
		t.Error("Expected JSON object start")
	}
}

func TestGetStats(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log different events
	log.LogEntry("access", Actor{ID: "user1"}, "view", Resource{ID: "res1"}, nil, ComplianceFlags{Severity: "low"})
	log.LogEntry("access", Actor{ID: "user2"}, "view", Resource{ID: "res2"}, nil, ComplianceFlags{Severity: "high"})
	log.LogEntry("modification", Actor{ID: "user1"}, "edit", Resource{ID: "res1"}, nil, ComplianceFlags{Severity: "medium", PHIInvolved: true})
	log.LogSecurityEvent("critical", "auth_failure", map[string]interface{}{"alert": "test"})

	stats := log.GetStats()

	totalEntries, ok := stats["total_entries"].(int)
	if !ok || totalEntries != 4 {
		t.Errorf("Expected 4 total entries, got %v", stats["total_entries"])
	}

	chainValid, ok := stats["chain_valid"].(bool)
	if !ok || !chainValid {
		t.Error("Expected valid chain in stats")
	}

	phiCount, ok := stats["phi_involved"].(int)
	if !ok || phiCount != 1 {
		t.Errorf("Expected 1 PHI entry, got %v", stats["phi_involved"])
	}
}

func TestConvenienceMethods(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Test LogUserAccess
	entry, err := log.LogUserAccess("user-123", "John Doe", "login", "system", "session-1", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogUserAccess failed: %v", err)
	}
	if entry.EventType != "user_access" {
		t.Errorf("Expected user_access event, got %s", entry.EventType)
	}

	// Test LogSecurityEvent
	entry, err = log.LogSecurityEvent("high", "auth_failure", map[string]interface{}{"attempts": 3})
	if err != nil {
		t.Fatalf("LogSecurityEvent failed: %v", err)
	}
	if entry.EventType != "security_event" {
		t.Errorf("Expected security_event, got %s", entry.EventType)
	}
	if !entry.Compliance.AuditRequired {
		t.Error("Expected AuditRequired for security event")
	}

	// Test LogPHIEvent
	entry, err = log.LogPHIEvent("user-123", "view_record", "patient", "patient-456", map[string]interface{}{"record_type": "medical"})
	if err != nil {
		t.Fatalf("LogPHIEvent failed: %v", err)
	}
	if !entry.Compliance.PHIInvolved {
		t.Error("Expected PHIInvolved to be true")
	}
	if entry.Compliance.Severity != "high" {
		t.Errorf("Expected high severity for PHI, got %s", entry.Compliance.Severity)
	}

	// Test LogConfigurationChange
	entry, err = log.LogConfigurationChange("admin-1", "update_rate_limit", "system", "config", map[string]interface{}{"old": 100, "new": 200})
	if err != nil {
		t.Fatalf("LogConfigurationChange failed: %v", err)
	}
	if entry.EventType != "config_change" {
		t.Errorf("Expected config_change event, got %s", entry.EventType)
	}
}

func TestEntryFilterTimeRange(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	// Log entries
	log.LogEntry("event1", Actor{ID: "user1"}, "action", Resource{ID: "res1"}, nil, ComplianceFlags{})
	time.Sleep(10 * time.Millisecond)
	log.LogEntry("event2", Actor{ID: "user2"}, "action", Resource{ID: "res2"}, nil, ComplianceFlags{})
	time.Sleep(10 * time.Millisecond)
	log.LogEntry("event3", Actor{ID: "user3"}, "action", Resource{ID: "res3"}, nil, ComplianceFlags{})

	// Get all entries to find timestamps
	allEntries := log.GetEntries(EntryFilter{})
	if len(allEntries) < 3 {
		t.Fatalf("Expected at least 3 entries, got %d", len(allEntries))
	}

	// Filter by time range (middle entry only)
	startTime := allEntries[0].Timestamp.Add(5 * time.Millisecond)
	endTime := allEntries[2].Timestamp.Add(-5 * time.Millisecond)

	filter := EntryFilter{
		StartTime: startTime,
		EndTime:   endTime,
	}

	filtered := log.GetEntries(filter)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 entry in time range, got %d", len(filtered))
	}
}

func TestEmptyLogVerification(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	result := log.VerifyChain()
	if !result.Valid {
		t.Error("Empty log should be valid")
	}

	if result.TotalEntries != 0 {
		t.Errorf("Expected 0 entries, got %d", result.TotalEntries)
	}
}

func TestCalculateHashDeterminism(t *testing.T) {
	log := NewTamperEvidentLog(TamperEvidentConfig{
		Enabled:    true,
		MaxEntries: 100,
	})

	entry := &TamperEvidentEntry{
		Sequence:     1,
		Timestamp:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EventType:    "test",
		Actor:        Actor{Type: "user", ID: "123"},
		Action:       "action",
		Resource:     Resource{Type: "room", ID: "456"},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	hash1 := log.calculateHash(entry)
	hash2 := log.calculateHash(entry)

	if hash1 != hash2 {
		t.Error("Hash calculation should be deterministic")
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}
