package capability

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Mocks
// ============================================================================

type mockClassifier struct {
	class  RiskClass
	level  RiskLevel
	panic_ bool
}

func (m *mockClassifier) Classify(_ context.Context, _ string, _ map[string]any) (RiskClass, RiskLevel) {
	if m.panic_ {
		panic("classifier exploded")
	}
	return m.class, m.level
}

type mockRegistry struct {
	caps CapabilitySet
	err  error
}

func (m *mockRegistry) GetCapabilities(_ string) (CapabilitySet, error) {
	return m.caps, m.err
}

func (m *mockRegistry) RegisterRole(_ string, _ CapabilitySet) error {
	return nil
}

type mockConsent struct {
	ch chan ConsentResult
}

func newMockConsent() *mockConsent {
	return &mockConsent{
		ch: make(chan ConsentResult, 1),
	}
}

func (m *mockConsent) RequestConsent(_ context.Context, _, _ string, _ []string) (<-chan ConsentResult, error) {
	return m.ch, nil
}

type mockSkillGate struct {
	err  error
	call *ToolCall
}

func (m *mockSkillGate) InterceptToolCall(_ context.Context, call *ToolCall) (*ToolCall, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.call = call
	return call, nil
}

// ============================================================================
// Tests
// ============================================================================

func TestBrokerAuthorize_Allow(t *testing.T) {
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
		Params:  map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW, got DENY: %s", resp.Reason)
	}
	if resp.Classification != RiskAllow {
		t.Errorf("expected classification ALLOW, got %s", resp.Classification)
	}
}

func TestBrokerAuthorize_DenyCapability(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
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
	if !strings.Contains(resp.Reason, "not in role") {
		t.Errorf("expected 'not in role' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_DenyRisk(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDeny},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
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
		t.Errorf("expected classification DENY, got %s", resp.Classification)
	}
}

func TestBrokerAuthorize_DeferConsentGranted(t *testing.T) {
	consent := newMockConsent()
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		DefaultBrokerConfig(),
	)

	var resp ActionResponse
	var err error
	done := make(chan struct{})

	go func() {
		resp, err = b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		close(done)
	}()

	consent.ch <- ConsentResult{Approved: true}
	<-done

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW after consent granted, got DENY: %s", resp.Reason)
	}
	if resp.Reason != "approved by consent" {
		t.Errorf("expected 'approved by consent', got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_DeferConsentDenied(t *testing.T) {
	consent := newMockConsent()
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		DefaultBrokerConfig(),
	)

	var resp ActionResponse
	var err error
	done := make(chan struct{})

	go func() {
		resp, err = b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		close(done)
	}()

	consent.ch <- ConsentResult{Approved: false}
	<-done

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY after consent denied")
	}
	if !strings.Contains(resp.Reason, "denied by consent") {
		t.Errorf("expected 'denied by consent' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_DeferTimeout(t *testing.T) {
	consent := newMockConsent()
	cfg := DefaultBrokerConfig()
	cfg.DeferTimeout = 50 * time.Millisecond

	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		cfg,
	)

	start := time.Now()
	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on timeout")
	}
	if !strings.Contains(resp.Reason, "timeout") {
		t.Errorf("expected 'timeout' in reason, got: %s", resp.Reason)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("timed out too quickly: %v", elapsed)
	}
}

func TestBrokerAuthorize_FailClosed_NilClassifier(t *testing.T) {
	b := NewBroker(
		nil,
		nil,
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		Action:  "anything",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY with nil classifier (fail-closed)")
	}
	if resp.Classification != RiskDeny {
		t.Errorf("expected DENY classification, got %s", resp.Classification)
	}
}

func TestBrokerAuthorize_FailClosed_Panic(t *testing.T) {
	b := NewBroker(
		&mockClassifier{panic_: true},
		nil,
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "anything",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY after panic (fail-closed)")
	}
	if !strings.Contains(resp.Reason, "panic") {
		t.Errorf("expected 'panic' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_FailClosed_NilRegistry(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		nil,
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("expected ALLOW with nil registry (skips cap check), got DENY: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_QueueFull(t *testing.T) {
	consent := newMockConsent()
	cfg := DefaultBrokerConfig()
	cfg.MaxConcurrentDefers = 1
	cfg.DeferTimeout = 5 * time.Second

	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		cfg,
	)

	done1 := make(chan struct{})
	go func() {
		b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		close(done1)
	}()

	time.Sleep(20 * time.Millisecond)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-2",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY when queue full")
	}
	if !strings.Contains(resp.Reason, "consent queue full") {
		t.Errorf("expected 'consent queue full' in reason, got: %s", resp.Reason)
	}

	consent.ch <- ConsentResult{Approved: true}
	<-done1
}

func TestBrokerAuthorize_CircularDependency(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		nil,
		nil,
		nil,
		BrokerConfig{MaxCallDepth: 5},
	)

	ctx := WithCallDepth(context.Background(), 6)

	resp, err := b.Authorize(ctx, ActionRequest{
		AgentID: "agent-1",
		Action:  "browse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY for circular dependency")
	}
	if !strings.Contains(resp.Reason, "circular dependency") {
		t.Errorf("expected 'circular dependency' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_ContextCancelled(t *testing.T) {
	consent := newMockConsent()
	cfg := DefaultBrokerConfig()
	cfg.DeferTimeout = 10 * time.Second

	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		cfg,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	resp, err := b.Authorize(ctx, ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on context cancellation")
	}
	if !strings.Contains(resp.Reason, "context cancelled") {
		t.Errorf("expected 'context cancelled' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_InvalidInput(t *testing.T) {
	b := NewBroker(nil, nil, nil, nil, DefaultBrokerConfig())

	resp, err := b.Authorize(context.Background(), ActionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY for invalid input")
	}
	if resp.Classification != RiskDeny {
		t.Errorf("expected DENY classification, got %s", resp.Classification)
	}
}

func TestBrokerAuthorize_ConsentError(t *testing.T) {
	consent := newMockConsent()
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		consent,
		nil,
		DefaultBrokerConfig(),
	)

	var resp ActionResponse
	var err error
	done := make(chan struct{})

	go func() {
		resp, err = b.Authorize(context.Background(), ActionRequest{
			AgentID: "agent-1",
			TeamID:  "team-1",
			Action:  "payment",
		})
		close(done)
	}()

	consent.ch <- ConsentResult{Error: fmt.Errorf("human rejected")}
	<-done

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on consent error")
	}
	if !strings.Contains(resp.Reason, "consent error") {
		t.Errorf("expected 'consent error' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_DeferNoConsentProvider(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"payment": true}},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY when deferred but no consent provider")
	}
	if !strings.Contains(resp.Reason, "no consent provider") {
		t.Errorf("expected 'no consent provider' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_PIIScrubError(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{err: fmt.Errorf("scrubber crashed")},
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on PII scrub error (fail-closed)")
	}
	if !strings.Contains(resp.Reason, "PII scrub failed") {
		t.Errorf("expected 'PII scrub failed' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_RegistryError(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{err: fmt.Errorf("db connection lost")},
		nil,
		nil,
		DefaultBrokerConfig(),
	)

	resp, err := b.Authorize(context.Background(), ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowed {
		t.Error("expected DENY on registry error (fail-closed)")
	}
	if !strings.Contains(resp.Reason, "registry lookup failed") {
		t.Errorf("expected 'registry lookup failed' in reason, got: %s", resp.Reason)
	}
}

func TestBrokerAuthorize_CircularDependencyAtBoundary(t *testing.T) {
	b := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		nil,
		nil,
		nil,
		BrokerConfig{MaxCallDepth: 5},
	)

	t.Run("at_limit_allowed", func(t *testing.T) {
		ctx := WithCallDepth(context.Background(), 5)
		resp, err := b.Authorize(ctx, ActionRequest{
			AgentID: "agent-1",
			Action:  "browse",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Allowed {
			t.Errorf("depth=5 with MaxCallDepth=5 should be allowed, got DENY: %s", resp.Reason)
		}
	})

	t.Run("over_limit_denied", func(t *testing.T) {
		ctx := WithCallDepth(context.Background(), 6)
		resp, err := b.Authorize(ctx, ActionRequest{
			AgentID: "agent-1",
			Action:  "browse",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Allowed {
			t.Error("depth=6 with MaxCallDepth=5 should be denied")
		}
	})
}

func TestDefaultBrokerConfig(t *testing.T) {
	cfg := DefaultBrokerConfig()
	if cfg.DeferTimeout != 300*time.Second {
		t.Errorf("DeferTimeout = %v, want 300s", cfg.DeferTimeout)
	}
	if cfg.MaxConcurrentDefers != 50 {
		t.Errorf("MaxConcurrentDefers = %d, want 50", cfg.MaxConcurrentDefers)
	}
	if cfg.MaxCallDepth != 5 {
		t.Errorf("MaxCallDepth = %d, want 5", cfg.MaxCallDepth)
	}
}

func TestNewBroker_ZeroConfig(t *testing.T) {
	b := NewBroker(nil, nil, nil, nil, BrokerConfig{})
	if b.config.DeferTimeout != 300*time.Second {
		t.Errorf("DeferTimeout = %v, want 300s", b.config.DeferTimeout)
	}
	if b.config.MaxConcurrentDefers != 50 {
		t.Errorf("MaxConcurrentDefers = %d, want 50", b.config.MaxConcurrentDefers)
	}
	if b.config.MaxCallDepth != 5 {
		t.Errorf("MaxCallDepth = %d, want 5", b.config.MaxCallDepth)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkBrokerAuthorize(b *testing.B) {
	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"url": "https://example.com"},
	}

	var count int64
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := broker.Authorize(context.Background(), req)
			if !resp.Allowed {
				atomic.AddInt64(&count, 1)
			}
		}
	})
}

func BenchmarkBrokerAuthorize_Serial(b *testing.B) {
	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{
		AgentID: "agent-1",
		TeamID:  "team-1",
		Action:  "browse",
		Params:  map[string]any{"url": "https://example.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := broker.Authorize(context.Background(), req)
		if !resp.Allowed {
			b.Fatalf("unexpected DENY: %v", resp.Reason)
		}
	}
}
