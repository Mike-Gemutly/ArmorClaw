package skills

import (
	"context"
	"fmt"
	"testing"

	"github.com/armorclaw/bridge/pkg/governor"
)

type mockAuthorizer struct {
	deny    bool
	err     error
	called  bool
	action  string
	params  map[string]interface{}
}

func (m *mockAuthorizer) AuthorizeAction(_ context.Context, _, action string, params map[string]interface{}) error {
	m.called = true
	m.action = action
	m.params = params
	if m.err != nil {
		return m.err
	}
	if m.deny {
		return fmt.Errorf("capability denied: %s: action not permitted", action)
	}
	return nil
}

func TestExecuteSkill_AuthorizerDeny(t *testing.T) {
	auth := &mockAuthorizer{deny: true}
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate:  governor.NewGovernor(nil, nil),
		Authorizer: auth,
	})
	se.registry.skills["weather"] = &Skill{
		Name:    "weather",
		Domain:  "weather",
		Enabled: true,
	}

	result, err := se.ExecuteSkill(context.Background(), "weather", map[string]interface{}{"city": "London"})
	if err == nil {
		t.Fatal("expected error when authorizer denies")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Type != "error" {
		t.Errorf("expected Type=error, got %s", result.Type)
	}
	if !auth.called {
		t.Error("expected authorizer to be called")
	}
	if auth.action != "weather" {
		t.Errorf("expected action=weather, got %s", auth.action)
	}
}

func TestExecuteSkill_AuthorizerError(t *testing.T) {
	auth := &mockAuthorizer{err: fmt.Errorf("broker unavailable")}
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate:  governor.NewGovernor(nil, nil),
		Authorizer: auth,
	})
	se.registry.skills["weather"] = &Skill{
		Name:    "weather",
		Domain:  "weather",
		Enabled: true,
	}

	result, err := se.ExecuteSkill(context.Background(), "weather", map[string]interface{}{"city": "London"})
	if err == nil {
		t.Fatal("expected error when authorizer errors")
	}
	if result.Success {
		t.Error("expected Success=false")
	}
}

func TestExecuteSkill_NilAuthorizer_BackwardCompat(t *testing.T) {
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate: governor.NewGovernor(nil, nil),
	})
	se.registry.skills["weather"] = &Skill{
		Name:    "weather",
		Domain:  "weather",
		Enabled: true,
	}

	result, err := se.ExecuteSkill(context.Background(), "weather", map[string]interface{}{"city": "London"})
	if err != nil {
		t.Fatalf("expected no error with nil authorizer, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Error("expected Success=true with nil authorizer")
	}
}

func TestExecuteSkill_AuthorizerAllow(t *testing.T) {
	auth := &mockAuthorizer{}
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate:  governor.NewGovernor(nil, nil),
		Authorizer: auth,
	})
	se.registry.skills["weather"] = &Skill{
		Name:    "weather",
		Domain:  "weather",
		Enabled: true,
	}

	result, err := se.ExecuteSkill(context.Background(), "weather", map[string]interface{}{"city": "London"})
	if err != nil {
		t.Fatalf("expected no error when authorizer allows, got %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !auth.called {
		t.Error("expected authorizer to be called")
	}
}
