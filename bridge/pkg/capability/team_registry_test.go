package capability

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var builtinRoleLookup RoleLookupFunc = func(roleName string) (CapabilitySet, error) {
	builtin := map[string]CapabilitySet{
		"team_lead": {
			"browser.browse": true, "browser.extract": true, "browser.screenshot": true,
			"browser.fill": true, "secret.request": true,
			"doc.ingest": true, "doc.summarize": true, "doc.reference": true,
			"email.read": true, "email.draft": true, "email.send": true,
			"team.synthesize": true, "team.request_hitl": true, "team.review": true,
		},
		"browser_specialist": {
			"browser.browse": true, "browser.extract": true, "browser.screenshot": true,
		},
		"form_filler": {
			"browser.fill": true, "secret.request": true,
		},
		"doc_analyst": {
			"doc.ingest": true, "doc.summarize": true, "doc.reference": true,
		},
		"email_clerk": {
			"email.read": true, "email.draft": true, "email.send": true,
		},
		"supervisor": {
			"team.synthesize": true, "team.request_hitl": true, "team.review": true,
		},
	}
	caps, ok := builtin[roleName]
	if !ok {
		return nil, errors.New("unknown role")
	}
	return caps, nil
}

func TestTeamCapabilityRegistry_GetCapabilities_KnownRole(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "browser_specialist", nil
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	caps, err := reg.GetCapabilities("agent-browser-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps == nil {
		t.Fatal("expected non-nil CapabilitySet")
	}
	for _, want := range []string{"browser.browse", "browser.extract", "browser.screenshot"} {
		if !caps[want] {
			t.Errorf("expected capability %q in set", want)
		}
	}
}

func TestTeamCapabilityRegistry_GetCapabilities_UnknownRole(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "nonexistent_role", nil
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	_, err := reg.GetCapabilities("agent-x")
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("expected 'unknown role' in error, got: %v", err)
	}
}

func TestTeamCapabilityRegistry_GetCapabilities_NoTeam(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "", nil
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	caps, err := reg.GetCapabilities("lone-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps != nil {
		t.Errorf("expected nil CapabilitySet for agent without team, got %v", caps)
	}
}

func TestTeamCapabilityRegistry_GetCapabilities_ResolverError(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "", errors.New("db down")
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	_, err := reg.GetCapabilities("agent-1")
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
	if !strings.Contains(err.Error(), "resolve agent") {
		t.Errorf("expected 'resolve agent' in error, got: %v", err)
	}
}

func TestTeamCapabilityRegistry_GetCapabilities_NilResolver(t *testing.T) {
	reg := NewTeamCapabilityRegistry(nil, builtinRoleLookup)

	_, err := reg.GetCapabilities("agent-1")
	if err == nil {
		t.Fatal("expected error with nil resolver")
	}
	if !strings.Contains(err.Error(), "no role resolver") {
		t.Errorf("expected 'no role resolver' in error, got: %v", err)
	}
}

func TestTeamCapabilityRegistry_RegisterRole_Override(t *testing.T) {
	resolveCalled := false
	resolver := func(agentID string) (string, error) {
		resolveCalled = true
		return "custom_role", nil
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	customCaps := CapabilitySet{
		"custom.action": true,
	}
	if err := reg.RegisterRole("custom_role", customCaps); err != nil {
		t.Fatalf("RegisterRole failed: %v", err)
	}

	caps, err := reg.GetCapabilities("agent-custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolveCalled {
		t.Error("expected resolver to be called")
	}
	if !caps["custom.action"] {
		t.Error("expected custom.action in capability set")
	}
	if len(caps) != 1 {
		t.Errorf("expected exactly 1 capability, got %d", len(caps))
	}
}

func TestTeamCapabilityRegistry_RegisterRole_OverridesBuiltIn(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "browser_specialist", nil
	}
	reg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	overrideCaps := CapabilitySet{
		"browser.browse": true,
	}
	if err := reg.RegisterRole("browser_specialist", overrideCaps); err != nil {
		t.Fatalf("RegisterRole failed: %v", err)
	}

	caps, err := reg.GetCapabilities("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 1 {
		t.Errorf("override should replace built-in, expected 1 cap, got %d", len(caps))
	}
	if !caps["browser.browse"] {
		t.Error("expected browser.browse in override set")
	}
	if caps["browser.screenshot"] {
		t.Error("browser.screenshot should not be present in override")
	}
}

func TestTeamCapabilityRegistry_RegisterRole_EmptyName(t *testing.T) {
	reg := NewTeamCapabilityRegistry(func(string) (string, error) { return "", nil }, builtinRoleLookup)
	err := reg.RegisterRole("", CapabilitySet{"x": true})
	if err == nil {
		t.Fatal("expected error for empty role name")
	}
}

func TestTeamCapabilityRegistry_RegisterRole_NilCaps(t *testing.T) {
	reg := NewTeamCapabilityRegistry(func(string) (string, error) { return "", nil }, builtinRoleLookup)
	err := reg.RegisterRole("role", nil)
	if err == nil {
		t.Fatal("expected error for nil capability set")
	}
}

func TestTeamCapabilityRegistry_RegisterRole_EmptyCaps(t *testing.T) {
	reg := NewTeamCapabilityRegistry(func(string) (string, error) { return "", nil }, builtinRoleLookup)
	err := reg.RegisterRole("role", CapabilitySet{})
	if err == nil {
		t.Fatal("expected error for empty capability set")
	}
}

func TestBrokerWithTeamRegistry(t *testing.T) {
	resolver := func(agentID string) (string, error) {
		return "browser_specialist", nil
	}
	teamReg := NewTeamCapabilityRegistry(resolver, builtinRoleLookup)

	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		teamReg,
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	t.Run("allowed_action", func(t *testing.T) {
		resp, err := b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "browser.browse",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Allowed {
			t.Errorf("browser_specialist should be allowed to browse, got DENY: %s", resp.Reason)
		}
	})

	t.Run("denied_action", func(t *testing.T) {
		resp, err := b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "email.send",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Allowed {
			t.Error("browser_specialist should NOT be allowed to send email")
		}
		if !strings.Contains(resp.Reason, "not in role") {
			t.Errorf("expected 'not in role' in reason, got: %s", resp.Reason)
		}
	})

	t.Run("no_team_skips_registry", func(t *testing.T) {
		resp, err := b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			Action:  "anything",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Allowed {
			t.Errorf("agent without team should skip registry check, got DENY: %s", resp.Reason)
		}
	})
}
