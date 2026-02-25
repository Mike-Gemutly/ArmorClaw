// Package pii provides end-to-end tests for the BlindFill pipeline.
//
// Test scenario:
//   1. Skill requests PII variables (SSN=critical, name=low, email=medium)
//   2. HITL consent grants specific fields
//   3. BlindFillEngine resolves from encrypted profile
//   4. Resolved variables are formatted for container env injection
//   5. Audit log captures field names only (never values)
package pii

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// MockAuditLogger captures audit events for verification
type MockAuditLogger struct {
	Events []AuditEvent
}

type AuditEvent struct {
	RequestID     string
	SkillID       string
	GrantedBy     string
	GrantedFields []string
	Timestamp     time.Time
}

func (m *MockAuditLogger) LogPIIAccessGranted(ctx context.Context, requestID, skillID, grantedBy string, fields []string) error {
	m.Events = append(m.Events, AuditEvent{
		RequestID:     requestID,
		SkillID:       skillID,
		GrantedBy:     grantedBy,
		GrantedFields: fields,
		Timestamp:     time.Now(),
	})
	return nil
}

// TestBlindFillE2E_FullPipeline tests the complete blind fill flow:
// Skill manifest → HITL approval → resolve → container env → audit
func TestBlindFillE2E_FullPipeline(t *testing.T) {
	// === Setup: Encrypted profile in keystore ===
	keystore := NewMockKeystore()
	keystore.AddProfile("user-alice", ProfileData{
		FullName:  "Alice Johnson",
		Email:     "alice@armorclaw.io",
		Phone:     "555-0100",
		SSN:       "123-45-6789",
		Address:   "123 Main St, Springfield",
		DateOfBirth: "1990-01-15",
	})

	engine := NewBlindFillEngine(keystore, nil)

	// === Step 1: Skill declares PII requirements via manifest ===
	manifest := NewSkillManifest("book_flight", "Flight Booking Skill", []VariableRequest{
		{Key: "full_name", Description: "Passenger name for ticket", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Booking confirmation email", Required: true, Sensitivity: SensitivityMedium},
		{Key: "ssn", Description: "TSA identification number", Required: true, Sensitivity: SensitivityCritical},
		{Key: "phone", Description: "Contact number", Required: false, Sensitivity: SensitivityMedium},
	})

	if err := manifest.Validate(); err != nil {
		t.Fatalf("Manifest validation failed: %v", err)
	}

	// Verify manifest field classification
	required := manifest.GetRequiredFields()
	if len(required) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(required))
	}
	optional := manifest.GetOptionalFields()
	if len(optional) != 1 {
		t.Errorf("Expected 1 optional field, got %d", len(optional))
	}

	// === Step 2: HITL consent — user approves specific fields ===
	// Simulates: User sees SystemAlertCard in Matrix room, clicks "Approve"
	// User approves all required + phone (optional)
	approvedFields := []string{"full_name", "email", "ssn", "phone"}
	requestID := "req_e2e_test_001"
	grantedBy := "user-alice"

	// === Step 3: BlindFillEngine resolves from encrypted keystore ===
	resolved, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"user-alice",
		approvedFields,
		requestID,
		grantedBy,
	)
	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	// === Step 4: Validate resolved variables ===
	// 4a: All approved fields present
	if len(resolved.GrantedFields) != 4 {
		t.Errorf("Expected 4 granted fields, got %d: %v", len(resolved.GrantedFields), resolved.GrantedFields)
	}
	if len(resolved.DeniedFields) != 0 {
		t.Errorf("Expected 0 denied fields, got %d: %v", len(resolved.DeniedFields), resolved.DeniedFields)
	}

	// 4b: Values match profile
	expectations := map[string]string{
		"full_name": "Alice Johnson",
		"email":     "alice@armorclaw.io",
		"ssn":       "123-45-6789",
		"phone":     "555-0100",
	}
	for key, expected := range expectations {
		actual, exists := resolved.GetVariable(key)
		if !exists {
			t.Errorf("Variable %q not found in resolved set", key)
			continue
		}
		if actual != expected {
			t.Errorf("Variable %q: expected %q, got %q", key, expected, actual)
		}
	}

	// 4c: Metadata correct
	if resolved.SkillID != "book_flight" {
		t.Errorf("Expected skill_id 'book_flight', got %q", resolved.SkillID)
	}
	if resolved.RequestID != requestID {
		t.Errorf("Expected request_id %q, got %q", requestID, resolved.RequestID)
	}
	if resolved.ProfileID != "user-alice" {
		t.Errorf("Expected profile_id 'user-alice', got %q", resolved.ProfileID)
	}

	// === Step 5: Simulate container env injection ===
	envVars := buildContainerEnvVars(resolved)

	// Verify env var format
	expectedEnvVars := map[string]string{
		"PII_FULL_NAME": "Alice Johnson",
		"PII_EMAIL":     "alice@armorclaw.io",
		"PII_SSN":       "123-45-6789",
		"PII_PHONE":     "555-0100",
	}
	for key, expected := range expectedEnvVars {
		if envVars[key] != expected {
			t.Errorf("Env var %s: expected %q, got %q", key, expected, envVars[key])
		}
	}

	// === Step 6: Verify expiration ===
	if resolved.IsExpired() {
		t.Error("Resolution should not be expired immediately after creation")
	}
	if err := engine.ValidateResolution(resolved); err != nil {
		t.Errorf("ValidateResolution should pass for fresh resolution: %v", err)
	}

	// === Step 7: Verify safe JSON excludes actual values ===
	safeJSON, err := resolved.ToSafeJSON()
	if err != nil {
		t.Fatalf("ToSafeJSON failed: %v", err)
	}
	safeStr := string(safeJSON)

	// Safe JSON must NOT contain actual PII values
	sensitiveValues := []string{"Alice Johnson", "alice@armorclaw.io", "123-45-6789", "555-0100"}
	for _, val := range sensitiveValues {
		if strings.Contains(safeStr, val) {
			t.Errorf("ToSafeJSON leaked PII value: %q found in safe output", val)
		}
	}

	// Safe JSON must contain field names (for audit)
	var safeMap map[string]interface{}
	if err := json.Unmarshal(safeJSON, &safeMap); err != nil {
		t.Fatalf("Failed to parse safe JSON: %v", err)
	}
	if safeMap["field_count"].(float64) != 4 {
		t.Errorf("Safe JSON field_count should be 4, got %v", safeMap["field_count"])
	}
}

// TestBlindFillE2E_PartialApproval_CriticalDenied tests when user denies
// critical sensitivity fields — skill should still receive approved low fields
func TestBlindFillE2E_PartialApproval_CriticalDenied(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("user-bob", ProfileData{
		FullName: "Bob Smith",
		Email:    "bob@example.com",
		SSN:      "987-65-4321",
	})

	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("form_fill", "Form Autofill Skill", []VariableRequest{
		{Key: "full_name", Description: "Name field", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Email field", Required: true, Sensitivity: SensitivityMedium},
		{Key: "ssn", Description: "SSN field", Required: false, Sensitivity: SensitivityCritical},
	})

	// User approves name and email but DENIES SSN
	resolved, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"user-bob",
		[]string{"full_name", "email"}, // SSN not approved
		"req_e2e_partial",
		"user-bob",
	)
	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	// Name and email should be resolved
	if len(resolved.GrantedFields) != 2 {
		t.Errorf("Expected 2 granted fields, got %d", len(resolved.GrantedFields))
	}

	// SSN should be in denied list
	if len(resolved.DeniedFields) != 1 {
		t.Errorf("Expected 1 denied field, got %d", len(resolved.DeniedFields))
	}
	if len(resolved.DeniedFields) > 0 && resolved.DeniedFields[0] != "ssn" {
		t.Errorf("Expected denied field 'ssn', got %q", resolved.DeniedFields[0])
	}

	// SSN value must NOT be in resolved variables
	if _, exists := resolved.GetVariable("ssn"); exists {
		t.Error("SSN should NOT be in resolved variables when denied")
	}

	// Container env should NOT contain SSN
	envVars := buildContainerEnvVars(resolved)
	if _, exists := envVars["PII_SSN"]; exists {
		t.Error("PII_SSN env var should NOT exist when SSN was denied")
	}
}

// TestBlindFillE2E_ExpiredResolution tests that expired resolutions are rejected
func TestBlindFillE2E_ExpiredResolution(t *testing.T) {
	engine := NewBlindFillEngine(NewMockKeystore(), nil)

	// Create a resolution with expired timestamp
	resolved := &ResolvedVariables{
		SkillID:   "test_skill",
		RequestID: "req_expired",
		Variables: map[string]string{"name": "test"},
		ExpiresAt: time.Now().Add(-1 * time.Minute).Unix(), // Expired 1 min ago
	}

	if !resolved.IsExpired() {
		t.Error("Resolution should be expired")
	}

	if err := engine.ValidateResolution(resolved); err == nil {
		t.Error("ValidateResolution should fail for expired resolution")
	}
}

// TestBlindFillE2E_HashValue tests that PII hashing works for audit comparisons
func TestBlindFillE2E_HashValue(t *testing.T) {
	hash1 := HashValue("123-45-6789")
	hash2 := HashValue("123-45-6789")
	hash3 := HashValue("987-65-4321")

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("Same value should produce same hash")
	}

	// Different input should produce different hash
	if hash1 == hash3 {
		t.Error("Different values should produce different hashes")
	}

	// Hash should not contain original value
	if strings.Contains(hash1, "123-45-6789") {
		t.Error("Hash should not contain original value")
	}
}

// buildContainerEnvVars converts resolved variables to container environment format.
// This simulates what the bridge does when spawning an OpenClaw container.
func buildContainerEnvVars(resolved *ResolvedVariables) map[string]string {
	envVars := make(map[string]string)
	for key, value := range resolved.Variables {
		envKey := "PII_" + strings.ToUpper(key)
		envVars[envKey] = value
	}
	return envVars
}
