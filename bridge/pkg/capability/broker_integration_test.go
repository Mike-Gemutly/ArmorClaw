package capability

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_FullAllowFlow(t *testing.T) {
	registry := &mockRegistry{caps: CapabilitySet{"browse": true}}
	classifier := &mockClassifier{class: RiskExternalCommunication, level: RiskAllow}
	gate := &mockSkillGate{}

	broker := NewBroker(classifier, registry, nil, gate, BrokerConfig{
		DeferTimeout:        1 * time.Second,
		MaxConcurrentDefers: 10,
		MaxCallDepth:        5,
	})

	resp, err := broker.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW, got DENY: %s", resp.Reason)
	}
	if resp.Classification != RiskAllow {
		t.Errorf("expected ALLOW classification, got %s", resp.Classification)
	}
	if resp.RiskClass != RiskExternalCommunication {
		t.Errorf("expected external_communication risk class, got %s", resp.RiskClass)
	}
}

func TestIntegration_FullDenyFlow(t *testing.T) {
	registry := &mockRegistry{caps: CapabilitySet{"browse": true}}
	classifier := &mockClassifier{class: RiskPayment, level: RiskDeny}

	broker := NewBroker(classifier, registry, nil, nil, DefaultBrokerConfig())

	resp, err := broker.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY, got ALLOW")
	}
	if resp.Classification != RiskDeny {
		t.Errorf("expected DENY classification, got %s", resp.Classification)
	}
}

func TestIntegration_DeferToApproveFlow(t *testing.T) {
	consent := newMockConsent()
	registry := &mockRegistry{caps: CapabilitySet{"payment": true}}
	classifier := &mockClassifier{class: RiskPayment, level: RiskDefer}

	broker := NewBroker(classifier, registry, consent, nil, BrokerConfig{
		DeferTimeout:        5 * time.Second,
		MaxConcurrentDefers: 10,
		MaxCallDepth:        5,
	})

	done := make(chan ActionResponse, 1)
	go func() {
		resp, _ := broker.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		done <- resp
	}()

	consent.ch <- consentResult{Approved: true}

	resp := <-done
	if !resp.Allowed {
		t.Errorf("expected ALLOW after consent, got DENY: %s", resp.Reason)
	}
	if resp.RiskClass != RiskPayment {
		t.Errorf("expected payment risk class, got %s", resp.RiskClass)
	}
}

func TestIntegration_DeferToDenyFlow(t *testing.T) {
	consent := newMockConsent()
	registry := &mockRegistry{caps: CapabilitySet{"payment": true}}
	classifier := &mockClassifier{class: RiskPayment, level: RiskDefer}

	broker := NewBroker(classifier, registry, consent, nil, BrokerConfig{
		DeferTimeout:        5 * time.Second,
		MaxConcurrentDefers: 10,
		MaxCallDepth:        5,
	})

	done := make(chan ActionResponse, 1)
	go func() {
		resp, _ := broker.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		done <- resp
	}()

	consent.ch <- consentResult{Approved: false}

	resp := <-done
	if resp.Allowed {
		t.Error("expected DENY after consent denial")
	}
}

func TestIntegration_DeferTimeoutFlow(t *testing.T) {
	consent := newMockConsent()
	registry := &mockRegistry{caps: CapabilitySet{"payment": true}}
	classifier := &mockClassifier{class: RiskPayment, level: RiskDefer}

	broker := NewBroker(classifier, registry, consent, nil, BrokerConfig{
		DeferTimeout:        50 * time.Millisecond,
		MaxConcurrentDefers: 10,
		MaxCallDepth:        5,
	})

	resp, err := broker.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on timeout")
	}
}

func TestIntegration_TeamRegistryResolvesRole(t *testing.T) {
	roleLookup := func(role string) (CapabilitySet, error) {
		roles := map[string]CapabilitySet{
			"browser_specialist": {"browse": true, "fill": true},
			"team_lead":         {"browse": true, "fill": true, "payment": true},
		}
		if caps, ok := roles[role]; ok {
			return caps, nil
		}
		return nil, nil
	}
	roleResolver := func(agentID string) (string, error) {
		if agentID == "agent-lead" {
			return "team_lead", nil
		}
		return "browser_specialist", nil
	}

	teamReg := NewTeamCapabilityRegistry(roleResolver, roleLookup)
	classifier := &mockClassifier{class: RiskPayment, level: RiskDefer}
	consent := newMockConsent()

	broker := NewBroker(classifier, teamReg, consent, nil, BrokerConfig{
		DeferTimeout:        5 * time.Second,
		MaxConcurrentDefers: 10,
		MaxCallDepth:        5,
	})

	done := make(chan ActionResponse, 1)
	go func() {
		resp, _ := broker.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-lead",
			TeamID:  "team-1",
			Action:  "payment",
		})
		done <- resp
	}()

	consent.ch <- consentResult{Approved: true}

	resp := <-done
	if !resp.Allowed {
		t.Errorf("team_lead should have payment capability, got DENY: %s", resp.Reason)
	}
}

func TestIntegration_TeamRegistryDeniesMissingCap(t *testing.T) {
	roleLookup := func(role string) (CapabilitySet, error) {
		return CapabilitySet{"browse": true}, nil
	}
	roleResolver := func(agentID string) (string, error) {
		return "browser_specialist", nil
	}

	teamReg := NewTeamCapabilityRegistry(roleResolver, roleLookup)
	classifier := &mockClassifier{class: RiskPayment, level: RiskAllow}

	broker := NewBroker(classifier, teamReg, nil, nil, DefaultBrokerConfig())

	resp, _ := broker.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-browser",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if resp.Allowed {
		t.Error("browser_specialist should NOT have payment capability")
	}
}

func TestIntegration_SkillGatePIIScrubFlow(t *testing.T) {
	registry := &mockRegistry{caps: CapabilitySet{"browse": true}}
	classifier := &mockClassifier{class: RiskExternalCommunication, level: RiskAllow}
	gate := &trackingSkillGate{}

	broker := NewBroker(classifier, registry, nil, gate, DefaultBrokerConfig())

	resp, err := broker.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW, got DENY: %s", resp.Reason)
	}
	if gate.lastCall == nil {
		t.Fatal("skill gate should have been called")
	}
	if gate.lastCall.ToolName != "browse" {
		t.Errorf("expected tool name 'browse', got %q", gate.lastCall.ToolName)
	}
}

func TestIntegration_ArtifactRoundTrip(t *testing.T) {
	registry := &mockRegistry{caps: CapabilitySet{"browse": true}}
	classifier := &mockClassifier{class: RiskExternalCommunication, level: RiskAllow}

	broker := NewBroker(classifier, registry, nil, &mockSkillGate{}, DefaultBrokerConfig())

	req := ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"url": "https://example.com", "depth": 2},
	}

	resp, err := broker.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW, got DENY: %s", resp.Reason)
	}
	if resp.RiskClass == "" {
		t.Error("expected non-empty risk class in response")
	}
}

type trackingSkillGate struct {
	lastCall *toolCall
}

func (t *trackingSkillGate) InterceptToolCall(_ context.Context, call *toolCall) (*toolCall, error) {
	t.lastCall = call
	return call, nil
}
