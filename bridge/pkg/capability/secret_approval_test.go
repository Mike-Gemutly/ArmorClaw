package capability

import (
	"context"
	"strings"
	"testing"
)

func TestClassifyRisk_Payment(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	for _, name := range []string{
		"payment_gateway",
		"credit_card_number",
		"card_cvv",
	} {
		got := policy.ClassifyRisk(name)
		if got != RiskDeny {
			t.Errorf("ClassifyRisk(%q) = %q, want %q", name, got, RiskDeny)
		}
	}
}

func TestClassifyRisk_Identity(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	for _, name := range []string{
		"ssn",
		"passport_number",
		"id_document",
	} {
		got := policy.ClassifyRisk(name)
		if got != RiskDeny {
			t.Errorf("ClassifyRisk(%q) = %q, want %q", name, got, RiskDeny)
		}
	}
}

func TestClassifyRisk_APIKey(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	for _, name := range []string{
		"api_key",
		"bearer_token",
		"encryption_key",
	} {
		got := policy.ClassifyRisk(name)
		if got != RiskAllow {
			t.Errorf("ClassifyRisk(%q) = %q, want %q", name, got, RiskAllow)
		}
	}
}

func TestClassifyRisk_Default(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	for _, name := range []string{
		"unknown_secret",
		"database_url",
		"random_field",
	} {
		got := policy.ClassifyRisk(name)
		if got != RiskDeny {
			t.Errorf("ClassifyRisk(%q) = %q, want %q (conservative default)", name, got, RiskDeny)
		}
	}
}

func TestShouldAutoApprove_Payment(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	if policy.ShouldAutoApprove("payment_secret") {
		t.Error("ShouldAutoApprove(payment_secret) = true, want false")
	}
	if policy.ShouldAutoApprove("credit_card") {
		t.Error("ShouldAutoApprove(credit_card) = true, want false")
	}
}

func TestShouldAutoApprove_APIKey(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	if !policy.ShouldAutoApprove("api_key_openai") {
		t.Error("ShouldAutoApprove(api_key_openai) = false, want true")
	}
	if !policy.ShouldAutoApprove("auth_token") {
		t.Error("ShouldAutoApprove(auth_token) = false, want true")
	}
}

func TestStoreApprovedSecret_WithStorer(t *testing.T) {
	var called bool
	mockStorer := func(_ context.Context, credentialName, value string) (string, error) {
		called = true
		return "{{VAULT:" + credentialName + ":abc123}}", nil
	}

	policy := NewSecretApprovalPolicy(SecretApprovalConfig{SecretStorer: mockStorer})
	ref, err := policy.StoreApprovedSecret(context.Background(), "api_key", "secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("storer was not called")
	}
	if ref != "{{VAULT:api_key:abc123}}" {
		t.Errorf("ref = %q, want {{VAULT:api_key:abc123}}", ref)
	}
}

func TestStoreApprovedSecret_NilStorer(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	ref, err := policy.StoreApprovedSecret(context.Background(), "api_key", "mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref == "" {
		t.Error("ref is empty")
	}
	if !strings.HasPrefix(ref, "{{VAULT:api_key:") {
		t.Errorf("ref = %q, want prefix {{VAULT:api_key:", ref)
	}
	hashPart := ref[len("{{VAULT:api_key:") : len(ref)-2]
	if len(hashPart) != 16 {
		t.Errorf("hash prefix length = %d, want 16 (got %q)", len(hashPart), hashPart)
	}
}

func TestStoreApprovedSecret_RefFormat(t *testing.T) {
	policy := NewSecretApprovalPolicy(SecretApprovalConfig{})
	ref, err := policy.StoreApprovedSecret(context.Background(), "db_password", "hunter2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(ref, "{{VAULT:") {
		t.Errorf("ref = %q, want prefix {{VAULT:", ref)
	}
	if !strings.HasSuffix(ref, "}}") {
		t.Errorf("ref = %q, want suffix }}", ref)
	}
}
