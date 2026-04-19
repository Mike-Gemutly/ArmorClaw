package skills

import (
	"testing"
	"time"
)

func TestPolicyEnforcer_AllowExplicit(t *testing.T) {
	if !IsToolAllowed("weather") {
		t.Error("expected weather to be allowed")
	}
	if !IsToolAllowed("web.fetch") {
		t.Error("expected web.fetch to be allowed")
	}
}

func TestPolicyEnforcer_BlockExplicit(t *testing.T) {
	if ValidateScheme("weather", "ftp") {
		t.Error("expected ftp scheme to be blocked for weather")
	}
	if ValidateScheme("web.fetch", "http") {
		t.Error("expected http scheme to be blocked for web.fetch")
	}
}

func TestPolicyEnforcer_DenyUnknown(t *testing.T) {
	if IsToolAllowed("unknown_tool_xyz") {
		t.Error("expected unknown tool to be denied")
	}
	if IsToolAllowed("") {
		t.Error("expected empty tool name to be denied")
	}
	_, exists := GetPolicy("nonexistent_tool")
	if exists {
		t.Error("expected GetPolicy to return false for unknown tool")
	}
}

// Explicit policy overrides default deny-by-default risk level
func TestPolicyEnforcer_AllowOverridesBlock(t *testing.T) {
	risk := GetRiskLevel("weather")
	if risk != "low" {
		t.Errorf("expected risk=low for weather, got %s", risk)
	}

	unknownRisk := GetRiskLevel("totally_fake_tool")
	if unknownRisk != "high" {
		t.Errorf("expected risk=high for unknown tool, got %s", unknownRisk)
	}

	if !ValidateTimeout("weather", 3*time.Second) {
		t.Error("expected 3s timeout to be allowed for weather (limit 5s)")
	}
	if ValidateTimeout("weather", 10*time.Second) {
		t.Error("expected 10s timeout to be blocked for weather (limit 5s)")
	}
}

func TestPolicyEnforcer_AllSkillPoliciesHaveRisk(t *testing.T) {
	for name, policy := range Policy {
		if policy.Risk == "" {
			t.Errorf("policy %q has empty Risk field", name)
		}
	}
}
