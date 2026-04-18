package team

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Governance tests
// ---------------------------------------------------------------------------

func TestDefaultGovernanceConfig(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	if cfg.MaxMembersPerTeam != 10 {
		t.Errorf("MaxMembersPerTeam = %d, want 10", cfg.MaxMembersPerTeam)
	}
	if cfg.MaxTeamsPerInstance != 5 {
		t.Errorf("MaxTeamsPerInstance = %d, want 5", cfg.MaxTeamsPerInstance)
	}
	if cfg.AllowedRoles != nil {
		t.Errorf("AllowedRoles = %v, want nil", cfg.AllowedRoles)
	}
}

func TestValidateTeamCreation_OK(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 3 })
	if err := g.ValidateTeamCreation(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTeamCreation_Exceeded(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 5 })
	if err := g.ValidateTeamCreation(); err == nil {
		t.Fatal("expected error for exceeded team limit, got nil")
	}
}

func TestValidateMemberAddition_OK(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })
	if err := g.ValidateMemberAddition(3, "worker"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMemberAddition_ExceededMembers(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })
	if err := g.ValidateMemberAddition(10, "worker"); err == nil {
		t.Fatal("expected error for exceeded member limit, got nil")
	}
}

func TestValidateMemberAddition_DisallowedRole(t *testing.T) {
	cfg := GovernanceConfig{
		MaxMembersPerTeam:   10,
		MaxTeamsPerInstance: 5,
		AllowedRoles:        []string{"team_lead", "worker"},
	}
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })
	if err := g.ValidateMemberAddition(3, "supervisor"); err == nil {
		t.Fatal("expected error for disallowed role, got nil")
	}
}

func TestValidateRoleAssignment_Allowed(t *testing.T) {
	cfg := GovernanceConfig{
		AllowedRoles: []string{"team_lead", "worker"},
	}
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })
	if err := g.ValidateRoleAssignment("worker"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRoleAssignment_Disallowed(t *testing.T) {
	cfg := GovernanceConfig{
		AllowedRoles: []string{"team_lead", "worker"},
	}
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })
	if err := g.ValidateRoleAssignment("supervisor"); err == nil {
		t.Fatal("expected error for disallowed role, got nil")
	}
}

// ---------------------------------------------------------------------------
// Policy override tests
// ---------------------------------------------------------------------------

func TestResolveActionPolicy_TeamOverride(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })

	g.SetPolicyOverride(TeamPolicyOverride{
		TeamID: "team-1",
		RiskOverrides: map[string]string{
			"payment": "ALLOW",
		},
	})

	result := g.ResolveActionPolicy("team-1", "payment", "DEFER")
	if result != "ALLOW" {
		t.Errorf("ResolveActionPolicy = %q, want %q", result, "ALLOW")
	}
}

func TestResolveActionPolicy_NoOverride(t *testing.T) {
	cfg := DefaultGovernanceConfig()
	g := NewGovernanceEnforcer(cfg, func() int { return 0 })

	g.SetPolicyOverride(TeamPolicyOverride{
		TeamID: "team-1",
		RiskOverrides: map[string]string{
			"payment": "ALLOW",
		},
	})

	result := g.ResolveActionPolicy("team-1", "credential_use", "DEFER")
	if result != "DEFER" {
		t.Errorf("ResolveActionPolicy = %q, want %q", result, "DEFER")
	}
}

// ---------------------------------------------------------------------------
// Metrics tests
// ---------------------------------------------------------------------------

func TestTeamMetrics_RecordAndGet(t *testing.T) {
	m := NewTeamMetrics()
	teamID := "team-abc"

	m.RecordTokenUsage(teamID, 500)
	m.RecordTokenUsage(teamID, 300)
	m.RecordCost(teamID, 150)
	m.RecordLatency("_team:"+teamID, 100*time.Millisecond)
	m.RecordLatency("_team:"+teamID, 200*time.Millisecond)
	m.RecordHandoff(teamID, true)
	m.RecordHandoff(teamID, false)
	m.RecordSecretAccess(teamID)
	m.RecordSecretAccess(teamID)
	m.RecordApproval(teamID, "payment", true)
	m.RecordApproval(teamID, "payment", false)

	snap := m.GetSnapshot(teamID)
	if snap.TokenUsage != 800 {
		t.Errorf("TokenUsage = %d, want 800", snap.TokenUsage)
	}
	if snap.CostCents != 150 {
		t.Errorf("CostCents = %d, want 150", snap.CostCents)
	}
	if snap.HandoffsTotal != 2 {
		t.Errorf("HandoffsTotal = %d, want 2", snap.HandoffsTotal)
	}
	if snap.HandoffsFail != 1 {
		t.Errorf("HandoffsFail = %d, want 1", snap.HandoffsFail)
	}
	if snap.SecretAccesses != 2 {
		t.Errorf("SecretAccesses = %d, want 2", snap.SecretAccesses)
	}
	if snap.ApprovalsByRisk["payment"] != 1 {
		t.Errorf("ApprovalsByRisk[payment] = %d, want 1", snap.ApprovalsByRisk["payment"])
	}
	if snap.ApprovalsByRisk["payment:denied"] != 1 {
		t.Errorf("ApprovalsByRisk[payment:denied] = %d, want 1", snap.ApprovalsByRisk["payment:denied"])
	}
}

// ---------------------------------------------------------------------------
// Audit constants test
// ---------------------------------------------------------------------------

func TestTeamAuditEntry_Constants(t *testing.T) {
	events := []string{
		EventTeamCreated,
		EventTeamDissolved,
		EventMemberAdded,
		EventMemberRemoved,
		EventRoleAssigned,
		EventDelegationSent,
		EventHandoffComplete,
	}
	if len(events) != 7 {
		t.Fatalf("expected 7 event constants, got %d", len(events))
	}

	expected := map[string]bool{
		"team_created": true, "team_dissolved": true, "member_added": true,
		"member_removed": true, "role_assigned": true, "delegation_sent": true,
		"handoff_complete": true,
	}
	for _, e := range events {
		if !expected[e] {
			t.Errorf("unexpected event constant: %q", e)
		}
	}

	entry := RecordTeamEvent(TeamAuditEntry{
		EventType: EventTeamCreated,
		TeamID:    "t1",
		AgentID:   "a1",
	})
	if entry.EventID == "" {
		t.Error("RecordTeamEvent did not generate EventID")
	}
	if entry.Timestamp.IsZero() {
		t.Error("RecordTeamEvent did not set Timestamp")
	}
}
