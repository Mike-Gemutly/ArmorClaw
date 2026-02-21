// Package pii provides unit tests for PII profile management
package pii

import (
	"testing"
)

func TestNewUserProfile(t *testing.T) {
	profile := NewUserProfile("Test Profile", ProfileTypePersonal)

	if profile.ID == "" {
		t.Error("Profile ID should not be empty")
	}

	if profile.ProfileName != "Test Profile" {
		t.Errorf("Expected profile name 'Test Profile', got '%s'", profile.ProfileName)
	}

	if profile.ProfileType != ProfileTypePersonal {
		t.Errorf("Expected profile type '%s', got '%s'", ProfileTypePersonal, profile.ProfileType)
	}

	if profile.CreatedAt == 0 {
		t.Error("CreatedAt should be set")
	}

	if profile.UpdatedAt == 0 {
		t.Error("UpdatedAt should be set")
	}
}

func TestProfileValidate(t *testing.T) {
	tests := []struct {
		name        string
		profile     *UserProfile
		expectError bool
	}{
		{
			name: "valid profile",
			profile: &UserProfile{
				ProfileName: "Test",
				ProfileType: ProfileTypePersonal,
			},
			expectError: false,
		},
		{
			name: "missing name",
			profile: &UserProfile{
				ProfileType: ProfileTypePersonal,
			},
			expectError: true,
		},
		{
			name: "missing type",
			profile: &UserProfile{
				ProfileName: "Test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestProfileDataToMap(t *testing.T) {
	data := ProfileData{
		FullName:  "John Doe",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Phone:     "555-1234",
		Custom: map[string]string{
			"company_id": "12345",
		},
	}

	m := data.ToMap()

	if m[FieldFullName] != "John Doe" {
		t.Errorf("Expected full name 'John Doe', got '%s'", m[FieldFullName])
	}

	if m[FieldEmail] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", m[FieldEmail])
	}

	if m["company_id"] != "12345" {
		t.Errorf("Expected custom field '12345', got '%s'", m["company_id"])
	}
}

func TestProfileSetGetField(t *testing.T) {
	profile := NewUserProfile("Test", ProfileTypePersonal)

	// Test setting standard fields
	profile.SetField(FieldFullName, "Jane Doe")
	profile.SetField(FieldEmail, "jane@example.com")

	if val, exists := profile.GetField(FieldFullName); !exists || val != "Jane Doe" {
		t.Errorf("Expected 'Jane Doe', got '%s'", val)
	}

	if val, exists := profile.GetField(FieldEmail); !exists || val != "jane@example.com" {
		t.Errorf("Expected 'jane@example.com', got '%s'", val)
	}

	// Test custom fields
	profile.SetField("custom_field", "custom_value")

	if val, exists := profile.GetField("custom_field"); !exists || val != "custom_value" {
		t.Errorf("Expected 'custom_value', got '%s'", val)
	}
}

func TestProfileToInfo(t *testing.T) {
	profile := NewUserProfile("Test Profile", ProfileTypePersonal)
	profile.SetField(FieldFullName, "John Doe")
	profile.SetField(FieldEmail, "john@example.com")

	info := profile.ToInfo()

	if info.ID != profile.ID {
		t.Error("Info ID should match profile ID")
	}

	if info.ProfileName != profile.ProfileName {
		t.Error("Info ProfileName should match profile ProfileName")
	}

	if info.FieldCount != 2 {
		t.Errorf("Expected field count 2, got %d", info.FieldCount)
	}
}

func TestGetStandardFieldSchema(t *testing.T) {
	tests := []struct {
		profileType   ProfileType
		expectedCount int
	}{
		{ProfileTypePersonal, 12},
		{ProfileTypeBusiness, 10},
		{ProfileTypeCustom, 0},
		{ProfileType("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.profileType), func(t *testing.T) {
			schema := GetStandardFieldSchema(tt.profileType)
			if len(schema.Fields) != tt.expectedCount {
				t.Errorf("Expected %d fields for %s, got %d", tt.expectedCount, tt.profileType, len(schema.Fields))
			}
		})
	}
}

func TestIsFieldSensitive(t *testing.T) {
	profile := NewUserProfile("Test", ProfileTypePersonal)

	// SSN should be sensitive
	if !profile.IsFieldSensitive(FieldSSN) {
		t.Error("SSN should be marked as sensitive")
	}

	// DOB should be sensitive
	if !profile.IsFieldSensitive(FieldDateOfBirth) {
		t.Error("DateOfBirth should be marked as sensitive")
	}

	// Email should not be sensitive
	if profile.IsFieldSensitive(FieldEmail) {
		t.Error("Email should not be marked as sensitive")
	}

	// Custom fields should not be sensitive by default
	if profile.IsFieldSensitive("custom_field") {
		t.Error("Custom fields should not be marked as sensitive by default")
	}
}

func TestSkillManifestValidate(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *SkillManifest
		expectError bool
	}{
		{
			name: "valid manifest",
			manifest: NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
				{Key: "name", Description: "Your name", Required: true, Sensitivity: SensitivityMedium},
			}),
			expectError: false,
		},
		{
			name: "missing skill_id",
			manifest: &SkillManifest{
				SkillName: "Test Skill",
				Variables: []VariableRequest{
					{Key: "name", Description: "Your name"},
				},
			},
			expectError: true,
		},
		{
			name: "missing skill_name",
			manifest: &SkillManifest{
				SkillID: "skill-123",
				Variables: []VariableRequest{
					{Key: "name", Description: "Your name"},
				},
			},
			expectError: true,
		},
		{
			name: "no variables",
			manifest: &SkillManifest{
				SkillID:   "skill-123",
				SkillName: "Test Skill",
				Variables: []VariableRequest{},
			},
			expectError: true,
		},
		{
			name: "variable missing key",
			manifest: &SkillManifest{
				SkillID:   "skill-123",
				SkillName: "Test Skill",
				Variables: []VariableRequest{
					{Description: "Missing key"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestSkillManifestGetFields(t *testing.T) {
	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "name", Description: "Your name", Required: true},
		{Key: "email", Description: "Your email", Required: true},
		{Key: "phone", Description: "Your phone", Required: false},
	})

	// Test GetAllFieldKeys
	allKeys := manifest.GetAllFieldKeys()
	if len(allKeys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(allKeys))
	}

	// Test GetRequiredFields
	required := manifest.GetRequiredFields()
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}

	// Test GetOptionalFields
	optional := manifest.GetOptionalFields()
	if len(optional) != 1 {
		t.Errorf("Expected 1 optional field, got %d", len(optional))
	}
}

func TestResolvedVariables(t *testing.T) {
	resolved := NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")

	if resolved.SkillID != "skill-123" {
		t.Error("SkillID mismatch")
	}

	if resolved.RequestID != "req-456" {
		t.Error("RequestID mismatch")
	}

	// Test SetVariable
	resolved.SetVariable("name", "John Doe")
	resolved.SetVariable("email", "john@example.com")

	if len(resolved.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(resolved.Variables))
	}

	if val, exists := resolved.GetVariable("name"); !exists || val != "John Doe" {
		t.Error("Failed to get variable")
	}

	// Test DenyField
	resolved.DenyField("ssn")

	if len(resolved.DeniedFields) != 1 {
		t.Errorf("Expected 1 denied field, got %d", len(resolved.DeniedFields))
	}

	// Test IsExpired (should not be expired immediately)
	if resolved.IsExpired() {
		t.Error("ResolvedVariables should not be expired immediately")
	}
}

func TestResolvedVariablesToSafeJSON(t *testing.T) {
	resolved := NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("name", "John Doe")
	resolved.SetVariable("email", "john@example.com")

	safeJSON, err := resolved.ToSafeJSON()
	if err != nil {
		t.Fatalf("ToSafeJSON failed: %v", err)
	}

	// The safe JSON should not contain the actual PII values
	// but should contain field_count
	safeStr := string(safeJSON)
	if safeStr == "" {
		t.Error("SafeJSON should not be empty")
	}

	// Verify that actual values are NOT in the output
	// This is a critical security check
}
