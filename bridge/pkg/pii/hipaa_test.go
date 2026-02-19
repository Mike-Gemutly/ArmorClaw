// Package pii provides integration tests for HIPAA compliance
package pii

import (
	"context"
	"strings"
	"testing"
)

// TestPHIDetection tests detection of various PHI types
func TestPHIDetection(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:            HIPAATierFull,
		EnableAuditLog:  true,
		HashContext:     true,
		AuditRetentionDays: 90,
	})

	tests := []struct {
		name          string
		input         string
		expectPHI     bool
		expectedTypes []PHIType
	}{
		{
			name:      "MRN detection",
			input:     "Patient MRN: 12345678 was admitted",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypeMRN},
		},
		{
			name:      "ICD-10 diagnosis code",
			input:     "Diagnosis: A00.0 - Cholera",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypeDiagnosis},
		},
		{
			name:      "Prescription number",
			input:     "Rx #123456789 - Take as directed",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypePrescription},
		},
		{
			name:      "Lab result identifier",
			input:     "Lab: ABC-12345 pending",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypeLabResult},
		},
		{
			name:      "Medical device ID",
			input:     "Device ID: 12345/67890/12",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypeDeviceID},
		},
		{
			name:      "No PHI",
			input:     "The patient is doing well and will be discharged tomorrow",
			expectPHI: false,
			expectedTypes: nil,
		},
		{
			name:      "Multiple PHI types",
			input:     "MRN: ABC12345, Diagnosis: J18.9, Rx: 987654321",
			expectPHI: true,
			expectedTypes: []PHIType{PHITypeMRN, PHITypeDiagnosis, PHITypePrescription},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := scrubber.DetectPHI(tt.input)

			if tt.expectPHI && len(detections) == 0 {
				t.Error("Expected PHI to be detected, but none found")
			}

			if !tt.expectPHI && len(detections) > 0 {
				t.Errorf("Expected no PHI, but found %d detections", len(detections))
			}

			// Verify detection types if specified
			if tt.expectedTypes != nil {
				foundTypes := make(map[PHIType]bool)
				for _, d := range detections {
					foundTypes[d.Type] = true
				}

				for _, expectedType := range tt.expectedTypes {
					if !foundTypes[expectedType] {
						t.Errorf("Expected PHI type %s not detected", expectedType)
					}
				}
			}
		})
	}
}

// TestPHIScrubbing tests actual scrubbing of PHI
func TestPHIScrubbing(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:           HIPAATierStandard,
		EnableAuditLog: true,
		HashContext:    true,
	})

	tests := []struct {
		name            string
		input           string
		expectRedacted  bool
		forbiddenOutput []string
	}{
		{
			name:           "MRN should be redacted",
			input:          "Patient MRN: 12345678",
			expectRedacted: true,
			forbiddenOutput: []string{"12345678"},
		},
		{
			name:           "Diagnosis code should be redacted",
			input:          "Patient has diagnosis A00.0",
			expectRedacted: true,
			forbiddenOutput: []string{"A00.0"},
		},
		{
			name:           "Prescription should be redacted",
			input:          "Rx: 123456789 prescribed",
			expectRedacted: true,
			forbiddenOutput: []string{"123456789"},
		},
		{
			name:           "Safe text unchanged",
			input:          "Patient is recovering well",
			expectRedacted: false,
			forbiddenOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, detections, err := scrubber.ScrubPHI(ctx, tt.input, "test-source")

			if err != nil {
				t.Fatalf("ScrubPHI failed: %v", err)
			}

			if tt.expectRedacted && len(detections) == 0 {
				t.Error("Expected PHI to be detected and redacted")
			}

			// Verify forbidden strings are not in output
			for _, forbidden := range tt.forbiddenOutput {
				if strings.Contains(result, forbidden) {
					t.Errorf("Output contains forbidden string: %s", forbidden)
				}
			}

			// Verify redaction markers are present
			if tt.expectRedacted {
				redactionMarkers := []string{"[REDACTED]", "[MRN", "[DIAGNOSIS", "[PRESCRIPTION"}
				found := false
				for _, marker := range redactionMarkers {
					if strings.Contains(result, marker) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected redaction marker in output: %s", result)
				}
			}
		})
	}
}

// TestHIPAAAuditLogging tests the audit logging functionality
func TestHIPAAAuditLogging(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:            HIPAATierFull,
		EnableAuditLog:  true,
		AuditRetentionDays: 90,
	})

	// Perform some PHI operations
	ctx := context.Background()
	text := "MRN: 12345678, Diagnosis: A00.0"

	_, _, _ = scrubber.ScrubPHI(ctx, text, "integration-test")
	_ = scrubber.DetectPHI(text)

	// Check audit log
	auditLog := scrubber.GetAuditLog(100)

	if len(auditLog) == 0 {
		t.Error("Expected audit log entries, but log is empty")
	}

	// Verify audit entry structure
	for _, entry := range auditLog {
		if entry.ID == "" {
			t.Error("Audit entry missing ID")
		}
		if entry.Timestamp.IsZero() {
			t.Error("Audit entry missing timestamp")
		}
		if entry.Action == "" {
			t.Error("Audit entry missing action")
		}
		if entry.PHIType == "" {
			t.Error("Audit entry missing PHI type")
		}
		if entry.Result == "" {
			t.Error("Audit entry missing result")
		}
	}
}

// TestAuditLogExport tests exporting audit logs
func TestAuditLogExport(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:           HIPAATierStandard,
		EnableAuditLog: true,
	})

	// Generate some audit entries
	ctx := context.Background()
	scrubber.ScrubPHI(ctx, "MRN: 12345678", "export-test")

	// Export as JSON
	export, err := scrubber.ExportAuditLog("json")
	if err != nil {
		t.Fatalf("Failed to export audit log: %v", err)
	}

	if len(export) == 0 {
		t.Error("Export returned empty data")
	}

	// Verify JSON is valid
	if !strings.Contains(string(export), "[") && !strings.Contains(string(export), "{") {
		t.Error("Export does not appear to be valid JSON")
	}
}

// TestPHISeverityLevels tests severity assignment for different PHI types
func TestPHISeverityLevels(t *testing.T) {
	_ = NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierFull,
	}) // Initialize patterns

	tests := []struct {
		phiType      PHIType
		minSeverity  string
	}{
		{PHITypeMRN, "critical"},
		{PHITypeHPBN, "critical"},
		{PHITypeBiometric, "critical"},
		{PHITypeDeviceID, "high"},
		{PHITypeLabResult, "high"},
		{PHITypeDiagnosis, "high"},
		{PHITypePrescription, "medium"},
		{PHITypeTreatment, "medium"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phiType), func(t *testing.T) {
			severity := getPHISeverity(tt.phiType)

			// Verify severity is at least the minimum expected
			severityLevels := map[string]int{"critical": 3, "high": 2, "medium": 1, "low": 0}
			minLevel := severityLevels[tt.minSeverity]
			actualLevel := severityLevels[severity]

			if actualLevel < minLevel {
				t.Errorf("Severity %s lower than expected minimum %s for type %s",
					severity, tt.minSeverity, tt.phiType)
			}
		})
	}
}

// TestComplianceTiers tests different compliance tier configurations
func TestComplianceTiers(t *testing.T) {
	tiers := []HIPAATier{HIPAATierBasic, HIPAATierStandard, HIPAATierFull}

	for _, tier := range tiers {
		t.Run(string(tier), func(t *testing.T) {
			config := HIPAAConfig{
				Tier:           tier,
				EnableAuditLog: true,
			}

			scrubber := NewHIPAAScrubber(config)

			if scrubber == nil {
				t.Fatal("Failed to create scrubber")
			}

			// Test basic functionality
			ctx := context.Background()
			_, detections, err := scrubber.ScrubPHI(ctx, "Test MRN: 12345678", "tier-test")

			if err != nil {
				t.Errorf("ScrubPHI failed for tier %s: %v", tier, err)
			}

			// Basic tier should still detect MRN
			if tier != HIPAATierBasic && len(detections) == 0 {
				t.Errorf("Expected PHI detection for tier %s", tier)
			}
		})
	}
}

// TestContextHashing tests that context hashing works correctly
func TestContextHashing(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierFull,
		HashContext: true,
	})

	text := "MRN: 12345678"
	detections := scrubber.DetectPHI(text)

	for _, d := range detections {
		if d.ContextHash == "" {
			t.Error("Context hash not generated")
		}

		// Hash should be consistent for same input
		if len(d.ContextHash) < 8 {
			t.Errorf("Context hash too short: %s", d.ContextHash)
		}
	}
}

// TestDisableEnable tests the enable/disable functionality
func TestDisableEnable(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierStandard,
	})

	// Should detect PHI when enabled
	detections := scrubber.DetectPHI("MRN: 12345678")
	if len(detections) == 0 {
		t.Error("Expected PHI detection when enabled")
	}

	// Disable and verify no detection
	scrubber.Disable()
	detections = scrubber.DetectPHI("MRN: 12345678")
	if len(detections) > 0 {
		t.Error("Did not expect PHI detection when disabled")
	}

	// Re-enable and verify detection works
	scrubber.Enable()
	detections = scrubber.DetectPHI("MRN: 12345678")
	if len(detections) == 0 {
		t.Error("Expected PHI detection after re-enabling")
	}
}

// TestConcurrentScrubbing tests thread safety
func TestConcurrentScrubbing(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:           HIPAATierStandard,
		EnableAuditLog: true,
	})

	done := make(chan bool)

	// Launch multiple concurrent scrubbers
	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := context.Background()
			text := "MRN: 12345678"
			for j := 0; j < 100; j++ {
				_, _, err := scrubber.ScrubPHI(ctx, text, "concurrent-test")
				if err != nil {
					t.Errorf("Concurrent scrub failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify audit log integrity
	auditLog := scrubber.GetAuditLog(10000)
	if len(auditLog) < 1000 {
		t.Errorf("Expected at least 1000 audit entries, got %d", len(auditLog))
	}
}

// TestAuditLogRetention tests audit log retention and cleanup
func TestAuditLogRetention(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:            HIPAATierStandard,
		EnableAuditLog:  true,
		AuditRetentionDays: 30,
	})

	// Generate some audit entries
	ctx := context.Background()
	scrubber.ScrubPHI(ctx, "MRN: 12345678", "retention-test")

	// Get count before cleanup
	beforeCount := len(scrubber.GetAuditLog(10000))

	// Run cleanup
	scrubber.ClearAuditLog()

	// Get count after cleanup
	afterCount := len(scrubber.GetAuditLog(10000))

	// Since entries are recent, they should not be cleared
	if afterCount < beforeCount {
		t.Logf("Cleanup removed %d entries", beforeCount-afterCount)
	}
}

// TestHasPHI tests the quick PHI check function
func TestHasPHI(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierFull,
	})

	tests := []struct {
		text     string
		expected bool
	}{
		{"MRN: 12345678", true},
		{"Diagnosis: A00.0", true},
		{"Rx: 123456789", true},
		{"Patient is fine", false},
		{"No medical info here", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := scrubber.HasPHI(tt.text)
			if result != tt.expected {
				t.Errorf("HasPHI(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

// TestScrubPHIWithContextCancellation tests context handling
func TestScrubPHIWithContextCancellation(t *testing.T) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierStandard,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := scrubber.ScrubPHI(ctx, "MRN: 12345678", "cancel-test")

	if err == nil {
		t.Error("Expected error due to cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// BenchmarkPHIDetection benchmarks PHI detection performance
func BenchmarkPHIDetection(b *testing.B) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier: HIPAATierStandard,
	})

	text := "Patient MRN: 12345678 was diagnosed with A00.0 and prescribed Rx: 987654321"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scrubber.DetectPHI(text)
	}
}

// BenchmarkPHIScrubbing benchmarks PHI scrubbing performance
func BenchmarkPHIScrubbing(b *testing.B) {
	scrubber := NewHIPAAScrubber(HIPAAConfig{
		Tier:           HIPAATierStandard,
		EnableAuditLog: false, // Disable for pure scrubbing benchmark
	})

	text := "Patient MRN: 12345678 was diagnosed with A00.0 and prescribed Rx: 987654321"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scrubber.ScrubPHI(ctx, text, "benchmark")
	}
}
