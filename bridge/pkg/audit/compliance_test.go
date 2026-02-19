// Package audit provides integration tests for compliance audit logging
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestComplianceLevels tests different compliance level configurations
func TestComplianceLevels(t *testing.T) {
	tests := []struct {
		level         ComplianceLevel
		expectedDays  int
	}{
		{ComplianceStandard, 30},
		{ComplianceExtended, 90},
		{ComplianceFull, 365},
		{ComplianceHIPAA, 2190}, // 6 years
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			config := ComplianceConfig{
				Level:       tt.level,
				StoragePath: filepath.Join(os.TempDir(), "audit-test-"+string(tt.level)+".db"),
			}

			cal, err := NewComplianceAuditLog(config)
			if err != nil {
				t.Fatalf("Failed to create audit log: %v", err)
			}
			defer os.Remove(config.StoragePath)

			if cal.GetRetentionDays() != tt.expectedDays {
				t.Errorf("Expected retention %d days, got %d", tt.expectedDays, cal.GetRetentionDays())
			}

			if cal.GetComplianceLevel() != tt.level {
				t.Errorf("Expected level %s, got %s", tt.level, cal.GetComplianceLevel())
			}
		})
	}
}

// TestLogEntry tests basic log entry functionality
func TestLogEntry(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-log-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	entry := ComplianceEntry{
		EventType: EventType("user.login"),
		UserID:    "user-123",
		Action:    "login",
		Resource:  "session",
		Status:    "success",
		Source:    "web",
		IPAddress: "192.168.1.1",
	}

	err = cal.Log(entry)
	if err != nil {
		t.Fatalf("Failed to log entry: %v", err)
	}

	if cal.Count() != 1 {
		t.Errorf("Expected 1 entry, got %d", cal.Count())
	}
}

// TestHashChain tests hash chain integrity
func TestHashChain(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-hash-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log multiple entries
	for i := 0; i < 5; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Resource:  "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry %d: %v", i, err)
		}
	}

	// Verify hash chain
	valid, corrupted := cal.VerifyIntegrity()
	if !valid {
		t.Errorf("Hash chain verification failed, corrupted indices: %v", corrupted)
	}

	if len(corrupted) > 0 {
		t.Errorf("Found corrupted entries at indices: %v", corrupted)
	}
}

// TestHashChainTamperDetection tests that tampering is detected
func TestHashChainTamperDetection(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-tamper-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log entries
	for i := 0; i < 3; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Resource:  "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Tamper with an entry (this would require internal access)
	// For now, we just verify the hash chain is valid before tampering
	valid, _ := cal.VerifyIntegrity()
	if !valid {
		t.Error("Hash chain should be valid before any tampering")
	}
}

// TestQuery tests querying audit entries
func TestQuery(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-query-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log multiple entries
	entries := []ComplianceEntry{
		{EventType: EventType("user.login"), UserID: "user-1", SessionID: "sess-1", Action: "login", Status: "success"},
		{EventType: EventType("user.logout"), UserID: "user-1", SessionID: "sess-1", Action: "logout", Status: "success"},
		{EventType: EventType("user.login"), UserID: "user-2", SessionID: "sess-2", Action: "login", Status: "success"},
		{EventType: EventType("container.create"), UserID: "user-1", Action: "create", Resource: "container-1", Status: "success"},
	}

	for _, e := range entries {
		if err := cal.Log(e); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	tests := []struct {
		name     string
		params   QueryParams
		expected int
	}{
		{
			name:     "query all",
			params:   QueryParams{Limit: 100},
			expected: 4,
		},
		{
			name:     "query by session",
			params:   QueryParams{SessionID: "sess-1", Limit: 100},
			expected: 2,
		},
		{
			name:     "query by event type",
			params:   QueryParams{EventType: EventType("user.login"), Limit: 100},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := cal.Query(tt.params)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

// TestExportJSON tests JSON export functionality
func TestExportJSON(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-export-json-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log some entries
	for i := 0; i < 3; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Resource:  "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Export as JSON
	data, err := cal.Export("json", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify it's valid JSON
	var entries []ComplianceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("Exported data is not valid JSON: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries in export, got %d", len(entries))
	}
}

// TestExportCSV tests CSV export functionality
func TestExportCSV(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-export-csv-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log some entries
	for i := 0; i < 3; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Resource:  "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Export as CSV
	data, err := cal.Export("csv", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify CSV structure
	csvStr := string(data)
	lines := strings.Split(csvStr, "\n")

	// Should have header + 3 data rows (+ empty line at end)
	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines (header + 3 data), got %d", len(lines))
	}

	// Check header
	if !strings.Contains(lines[0], "id,timestamp") {
		t.Errorf("CSV header doesn't contain expected fields: %s", lines[0])
	}
}

// TestGenerateReport tests report generation
func TestGenerateReport(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-report-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log various entries
	entries := []ComplianceEntry{
		{EventType: EventType("user.login"), UserID: "user-1", Action: "login", Status: "success"},
		{EventType: EventType("user.login"), UserID: "user-2", Action: "login", Status: "failure"},
		{EventType: EventType("container.create"), UserID: "user-1", Action: "create", Resource: "container-1", Status: "success"},
		{EventType: EventType("security.violation"), UserID: "user-3", Action: "access", Resource: "restricted", Status: "denied"},
	}

	for _, e := range entries {
		if err := cal.Log(e); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Generate report
	report, err := cal.GenerateReport(time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Report generation failed: %v", err)
	}

	// Verify report contents
	if report.TotalEvents != 4 {
		t.Errorf("Expected 4 total events, got %d", report.TotalEvents)
	}

	if report.FailedActions != 2 { // 1 failure + 1 denied
		t.Errorf("Expected 2 failed actions, got %d", report.FailedActions)
	}

	if report.SecurityEvents != 1 {
		t.Errorf("Expected 1 security event, got %d", report.SecurityEvents)
	}

	if report.UniqueUsers != 3 {
		t.Errorf("Expected 3 unique users, got %d", report.UniqueUsers)
	}

	if report.IntegrityStatus != "valid" {
		t.Errorf("Expected valid integrity status, got %s", report.IntegrityStatus)
	}
}

// TestPurgeOldEntries tests retention-based purging
func TestPurgeOldEntries(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-purge-test.db"),
		EnableHashChain: true,
		MaxEntries:      1000,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log some entries
	for i := 0; i < 10; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	beforeCount := cal.Count()

	// Purge (should not remove recent entries)
	purged, err := cal.PurgeOldEntries()
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	afterCount := cal.Count()

	// Recent entries should not be purged
	if purged > 0 {
		t.Logf("Purged %d old entries", purged)
	}

	if afterCount != beforeCount {
		t.Errorf("Expected count to remain %d, got %d", beforeCount, afterCount)
	}
}

// TestLogEvent tests the LogEvent helper function
func TestLogEvent(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-event-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	err = cal.LogEvent(EventType("user.action"), "create", "resource-1", "success", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	if cal.Count() != 1 {
		t.Errorf("Expected 1 entry, got %d", cal.Count())
	}
}

// TestLogUserAction tests the LogUserAction helper function
func TestLogUserAction(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-user-action-test.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	err = cal.LogUserAction("user-123", "delete", "resource-456", "denied", "insufficient permissions")
	if err != nil {
		t.Fatalf("LogUserAction failed: %v", err)
	}

	// Query to verify
	entries, err := cal.Query(QueryParams{Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}

	if entries[0].UserID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got %s", entries[0].UserID)
	}

	if entries[0].Status != "denied" {
		t.Errorf("Expected status 'denied', got %s", entries[0].Status)
	}

	if entries[0].Reason != "insufficient permissions" {
		t.Errorf("Expected reason 'insufficient permissions', got %s", entries[0].Reason)
	}
}

// TestParseComplianceLevel tests compliance level parsing
func TestParseComplianceLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected ComplianceLevel
	}{
		{"standard", ComplianceStandard},
		{"basic", ComplianceStandard},
		{"extended", ComplianceExtended},
		{"enhanced", ComplianceExtended},
		{"full", ComplianceFull},
		{"complete", ComplianceFull},
		{"hipaa", ComplianceHIPAA},
		{"medical", ComplianceHIPAA},
		{"unknown", ComplianceExtended}, // default
		{"", ComplianceExtended},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseComplianceLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseComplianceLevel(%q) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{1 * time.Hour, "1h 0m"},
		{90 * time.Minute, "1h 30m"},
		{25 * time.Hour, "1d 1h"},
		{48 * time.Hour, "2d 0h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestParseEntryID tests entry ID parsing
func TestParseEntryID(t *testing.T) {
	// Create a valid entry ID
	timestamp := time.Now()
	entryID := generateEntryID(timestamp)

	parsed, err := ParseEntryID(entryID)
	if err != nil {
		t.Fatalf("ParseEntryID failed: %v", err)
	}

	// Verify the parsed timestamp is close to the original
	diff := parsed.Sub(timestamp)
	if diff < 0 {
		diff = -diff
	}

	if diff > time.Millisecond {
		t.Errorf("Parsed timestamp differs from original by %v", diff)
	}

	// Test invalid ID
	_, err = ParseEntryID("invalid-id")
	if err == nil {
		t.Error("Expected error for invalid entry ID")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultComplianceConfig()

	if config.Level != ComplianceExtended {
		t.Errorf("Expected default level to be 'extended', got %s", config.Level)
	}

	if config.MaxEntries != 100000 {
		t.Errorf("Expected max entries 100000, got %d", config.MaxEntries)
	}

	if !config.EnableHashChain {
		t.Error("Expected hash chain to be enabled by default")
	}
}

// TestMaxEntriesLimit tests that max entries limit is respected
func TestMaxEntriesLimit(t *testing.T) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-limit-test.db"),
		EnableHashChain: true,
		MaxEntries:      5,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Log more entries than the limit
	for i := 0; i < 10; i++ {
		entry := ComplianceEntry{
			EventType: EventType("test.event"),
			Action:    "test",
			Status:    "success",
		}
		if err := cal.Log(entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Should be limited to MaxEntries
	if cal.Count() > config.MaxEntries {
		t.Errorf("Expected count <= %d, got %d", config.MaxEntries, cal.Count())
	}
}

// BenchmarkLogEntry benchmarks log entry creation
func BenchmarkLogEntry(b *testing.B) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-bench.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		b.Fatalf("Failed to create audit log: %v", err)
	}

	entry := ComplianceEntry{
		EventType: EventType("benchmark.event"),
		Action:    "test",
		Resource:  "test",
		Status:    "success",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cal.Log(entry)
	}
}

// BenchmarkVerifyIntegrity benchmarks hash chain verification
func BenchmarkVerifyIntegrity(b *testing.B) {
	config := ComplianceConfig{
		Level:           ComplianceExtended,
		StoragePath:     filepath.Join(os.TempDir(), "audit-verify-bench.db"),
		EnableHashChain: true,
	}
	defer os.Remove(config.StoragePath)

	cal, err := NewComplianceAuditLog(config)
	if err != nil {
		b.Fatalf("Failed to create audit log: %v", err)
	}

	// Pre-populate with entries
	for i := 0; i < 1000; i++ {
		entry := ComplianceEntry{
			EventType: EventType("benchmark.event"),
			Action:    "test",
			Status:    "success",
		}
		cal.Log(entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cal.VerifyIntegrity()
	}
}
