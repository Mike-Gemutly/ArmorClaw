// Package pii provides tests for the BlindFillEngine resolver
package pii

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// MockKeystore implements KeystoreInterface for testing
type MockKeystore struct {
	profiles map[string]*UserProfileData
	err      error
}

func NewMockKeystore() *MockKeystore {
	return &MockKeystore{
		profiles: make(map[string]*UserProfileData),
	}
}

func (m *MockKeystore) RetrieveProfile(id string) (*UserProfileData, error) {
	if m.err != nil {
		return nil, m.err
	}
	profile, exists := m.profiles[id]
	if !exists {
		return nil, errors.New("profile not found")
	}
	return profile, nil
}

func (m *MockKeystore) AddProfile(id string, data ProfileData) {
	jsonData, _ := json.Marshal(data)
	m.profiles[id] = &UserProfileData{
		ID:          id,
		ProfileName: "Test Profile",
		ProfileType: "personal",
		Data:        jsonData,
	}
}

func TestBlindFillEngine_ResolveVariables(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("profile-123", ProfileData{
		FullName: "John Doe",
		Email:    "john@example.com",
		Phone:    "555-1234",
	})

	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Your email", Required: true, Sensitivity: SensitivityMedium},
		{Key: "phone", Description: "Your phone", Required: false, Sensitivity: SensitivityMedium},
	})

	resolved, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"profile-123",
		[]string{"full_name", "email", "phone"},
		"req-789",
		"user-001",
	)

	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	if resolved.SkillID != "skill-456" {
		t.Errorf("Expected skill_id 'skill-456', got '%s'", resolved.SkillID)
	}

	if len(resolved.GrantedFields) != 3 {
		t.Errorf("Expected 3 granted fields, got %d", len(resolved.GrantedFields))
	}

	if resolved.Variables["full_name"] != "John Doe" {
		t.Errorf("Expected full_name 'John Doe', got '%s'", resolved.Variables["full_name"])
	}

	if resolved.Variables["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", resolved.Variables["email"])
	}
}

func TestBlindFillEngine_ResolveVariables_PartialApproval(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("profile-123", ProfileData{
		FullName: "John Doe",
		Email:    "john@example.com",
		Phone:    "555-1234",
	})

	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Your email", Required: false, Sensitivity: SensitivityMedium},
		{Key: "phone", Description: "Your phone", Required: false, Sensitivity: SensitivityMedium},
	})

	// Only approve full_name (required) and email (optional)
	resolved, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"profile-123",
		[]string{"full_name", "email"},
		"req-789",
		"user-001",
	)

	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	if len(resolved.GrantedFields) != 2 {
		t.Errorf("Expected 2 granted fields, got %d", len(resolved.GrantedFields))
	}

	if len(resolved.DeniedFields) != 1 {
		t.Errorf("Expected 1 denied field (phone), got %d", len(resolved.DeniedFields))
	}

	if resolved.Variables["phone"] != "" {
		t.Error("Phone should not be in variables since it was not approved")
	}
}

func TestBlindFillEngine_ResolveVariables_MissingRequired(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("profile-123", ProfileData{
		FullName: "John Doe",
		// Email is missing
	})

	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Your email", Required: true, Sensitivity: SensitivityMedium},
	})

	// Try to approve only email (required) but it's missing from profile
	resolved, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"profile-123",
		[]string{"full_name", "email"},
		"req-789",
		"user-001",
	)

	if err != nil {
		t.Fatalf("ResolveVariables should not fail for missing optional data: %v", err)
	}

	// Email should be denied since it's missing from profile
	if len(resolved.DeniedFields) != 1 {
		t.Errorf("Expected 1 denied field (email missing), got %d", len(resolved.DeniedFields))
	}
}

func TestBlindFillEngine_ResolveVariables_RequiredFieldNotApproved(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("profile-123", ProfileData{
		FullName: "John Doe",
		Email:    "john@example.com",
	})

	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true, Sensitivity: SensitivityLow},
		{Key: "email", Description: "Your email", Required: true, Sensitivity: SensitivityMedium},
	})

	// Only approve full_name, but email is required
	_, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"profile-123",
		[]string{"full_name"},
		"req-789",
		"user-001",
	)

	if err == nil {
		t.Error("Expected error when required field not approved")
	}
}

func TestBlindFillEngine_ResolveVariables_ProfileNotFound(t *testing.T) {
	keystore := NewMockKeystore()
	engine := NewBlindFillEngine(keystore, nil)

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	_, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"nonexistent-profile",
		[]string{"full_name"},
		"req-789",
		"user-001",
	)

	if err == nil {
		t.Error("Expected error for non-existent profile")
	}
}

func TestBlindFillEngine_ResolveVariables_InvalidManifest(t *testing.T) {
	keystore := NewMockKeystore()
	engine := NewBlindFillEngine(keystore, nil)

	// Manifest with no variables is invalid
	manifest := &SkillManifest{
		SkillID:   "skill-456",
		SkillName: "Test Skill",
		Variables: []VariableRequest{},
	}

	_, err := engine.ResolveVariables(
		context.Background(),
		manifest,
		"profile-123",
		[]string{},
		"req-789",
		"user-001",
	)

	if err == nil {
		t.Error("Expected error for invalid manifest")
	}
}

func TestBlindFillEngine_ValidateResolution(t *testing.T) {
	keystore := NewMockKeystore()
	engine := NewBlindFillEngine(keystore, nil)

	t.Run("valid resolution", func(t *testing.T) {
		resolved := NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
		err := engine.ValidateResolution(resolved)
		if err != nil {
			t.Errorf("Valid resolution should not error: %v", err)
		}
	})

	t.Run("nil resolution", func(t *testing.T) {
		err := engine.ValidateResolution(nil)
		if err == nil {
			t.Error("Nil resolution should error")
		}
	})

	t.Run("expired resolution", func(t *testing.T) {
		resolved := NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
		resolved.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()
		err := engine.ValidateResolution(resolved)
		if err == nil {
			t.Error("Expired resolution should error")
		}
	})
}

func TestBlindFillEngine_NewProfileResolver(t *testing.T) {
	keystore := NewMockKeystore()
	engine := NewBlindFillEngine(keystore, nil)

	resolver := engine.NewProfileResolver("profile-123")

	if resolver == nil {
		t.Fatal("ProfileResolver should not be nil")
	}

	if resolver.profileID != "profile-123" {
		t.Errorf("Expected profileID 'profile-123', got '%s'", resolver.profileID)
	}
}

func TestHashValue(t *testing.T) {
	hash1 := HashValue("test-value")
	hash2 := HashValue("test-value")
	hash3 := HashValue("different-value")

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("Same input should produce same hash")
	}

	// Different input should produce different hash
	if hash1 == hash3 {
		t.Error("Different input should produce different hash")
	}

	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestProfileResolver_Resolve(t *testing.T) {
	keystore := NewMockKeystore()
	keystore.AddProfile("profile-123", ProfileData{
		FullName: "Jane Smith",
	})

	engine := NewBlindFillEngine(keystore, nil)
	resolver := engine.NewProfileResolver("profile-123")

	manifest := NewSkillManifest("skill-456", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	resolved, err := resolver.Resolve(
		context.Background(),
		manifest,
		[]string{"full_name"},
		"req-789",
		"user-001",
	)

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Variables["full_name"] != "Jane Smith" {
		t.Errorf("Expected full_name 'Jane Smith', got '%s'", resolved.Variables["full_name"])
	}
}
