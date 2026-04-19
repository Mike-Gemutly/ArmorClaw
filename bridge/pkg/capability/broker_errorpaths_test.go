package capability

import (
	"context"
	"errors"
	"testing"
)

func TestBrokerAuthorizeAction_Allow(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	err := b.AuthorizeAction(context.Background(), "agent-1", "browse", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestBrokerAuthorizeAction_Deny(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDeny},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	err := b.AuthorizeAction(context.Background(), "agent-1", "payment", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, err) {
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestBrokerAuthorizeAction_DenyContainsReason(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDeny},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	err := b.AuthorizeAction(context.Background(), "agent-1", "payment", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if len(msg) < 10 {
		t.Errorf("error message too short: %q", msg)
	}
}

func TestBrokerAuthorize_UnknownRiskLevel(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskLevel("UNKNOWN")},
		&mockRegistry{caps: CapabilitySet{"action": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "action",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("unknown risk level should DENY")
	}
	if resp.Reason == "" {
		t.Error("expected reason for unknown risk level")
	}
}

func TestBrokerAuthorize_NoTeamID(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		Action:  "browse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW when no TeamID, got DENY: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_NilParams(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW with nil params, got DENY: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_AllDepsNil(t *testing.T) {
	b := NewBroker(nil, nil, nil, nil, DefaultBrokerConfig())

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "anything",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("all deps nil should DENY (nil classifier → DENY)")
	}
}

func TestBrokerAuthorize_SkillGateModifiesParams(t *testing.T) {
	gate := &modifyingSkillGate{}
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		gate,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"sensitive": "data"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW, got DENY: %s", resp.Reason)
	}
}

type modifyingSkillGate struct{}

func (m *modifyingSkillGate) InterceptToolCall(_ context.Context, call *ToolCall) (*ToolCall, error) {
	call.Arguments["sensitive"] = "[REDACTED]"
	return call, nil
}

func TestBrokerAuthorize_EmptyAction(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	resp, _ := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "",
	})
	if resp.Allowed {
		t.Error("empty action should DENY")
	}
}

func TestBrokerAuthorize_ConcurrentDefersDecrement(t *testing.T) {
	consent := newMockConsent()
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		BrokerConfig{DeferTimeout: 1, MaxConcurrentDefers: 2},
	)

	consent.ch <- ConsentResult{Approved: false}

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY from consent denial")
	}

	if b.activeDefers != 0 {
		t.Errorf("activeDefers should be 0 after consent resolved, got %d", b.activeDefers)
	}
}
