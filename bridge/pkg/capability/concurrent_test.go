package capability

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrent_100Authorize_Allow(t *testing.T) {
	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&noopSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{AgentID: "agent-1", TeamID: "team-1", Action: "browse"}

	var wg sync.WaitGroup
	var errors atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := broker.Authorize(context.Background(), req)
			if err != nil || !resp.Allowed {
				errors.Add(1)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(0), errors.Load())
}

func TestConcurrent_MixedAllowDenyDefer(t *testing.T) {
	var allowCount, denyCount, deferCount atomic.Int64

	broker := NewBroker(
		&countingClassifier{allowCount: &allowCount, denyCount: &denyCount, deferCount: &deferCount},
		&mockRegistry{caps: CapabilitySet{"browse": true, "pay": true, "pii": true}},
		nil,
		&noopSkillGate{},
		DefaultBrokerConfig(),
	)

	actions := []string{"browse", "pay", "pii"}
	var wg sync.WaitGroup
	for i := 0; i < 99; i++ {
		wg.Add(1)
		action := actions[i%3]
		go func(a string) {
			defer wg.Done()
			req := ActionRequest{AgentID: "agent-1", TeamID: "team-1", Action: a}
			broker.Authorize(context.Background(), req)
		}(action)
	}

	wg.Wait()
	total := allowCount.Load() + denyCount.Load() + deferCount.Load()
	assert.Equal(t, int64(99), total)
}

type countingClassifier struct {
	allowCount *atomic.Int64
	denyCount  *atomic.Int64
	deferCount *atomic.Int64
}

func (c *countingClassifier) Classify(_ context.Context, action string, _ map[string]any) (RiskClass, RiskLevel) {
	switch action {
	case "browse":
		c.allowCount.Add(1)
		return RiskExternalCommunication, RiskAllow
	case "pay":
		c.denyCount.Add(1)
		return RiskPayment, RiskDeny
	case "pii":
		c.deferCount.Add(1)
		return RiskIdentityPII, RiskDefer
	default:
		c.allowCount.Add(1)
		return RiskExternalCommunication, RiskAllow
	}
}

func TestConcurrent_BrokerWithSharedRegistry(t *testing.T) {
	var mu sync.Mutex
	capsMap := map[string]bool{"browse": true, "pay": true}
	reg := &threadSafeRegistry{mu: &mu, caps: capsMap}
	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		reg,
		nil,
		&noopSkillGate{},
		DefaultBrokerConfig(),
	)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := "browse"
			if idx%2 == 0 {
				action = "pay"
			}
			req := ActionRequest{AgentID: "agent-1", Action: action}
			broker.Authorize(context.Background(), req)
		}(i)
	}

	wg.Wait()
}

type threadSafeRegistry struct {
	mu   *sync.Mutex
	caps CapabilitySet
}

func (m *threadSafeRegistry) GetCapabilities(_ string) (CapabilitySet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.caps, nil
}

func (m *threadSafeRegistry) RegisterRole(_ string, _ CapabilitySet) error {
	return nil
}

type noopSkillGate struct{}

func (n *noopSkillGate) InterceptToolCall(_ context.Context, call *toolCall) (*toolCall, error) {
	return call, nil
}

func TestConcurrent_DeferConsentIsolation(t *testing.T) {
	consent := newMockConsent()
	broker := NewBroker(
		&mockClassifier{class: RiskIdentityPII, level: RiskDefer},
		&mockRegistry{caps: CapabilitySet{"pii": true}},
		consent,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	go func() {
		time.Sleep(10 * time.Millisecond)
		consent.ch <- consentResult{Approved: true}
	}()

	req := ActionRequest{AgentID: "agent-1", Action: "access_pii"}
	resp, err := broker.Authorize(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestConcurrent_RapidCreateAuthorize(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := NewBroker(
				&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
				&mockRegistry{caps: CapabilitySet{"test": true}},
				nil,
				&mockSkillGate{},
				DefaultBrokerConfig(),
			)
			req := ActionRequest{AgentID: "agent-1", Action: "test"}
			b.Authorize(context.Background(), req)
		}()
	}

	wg.Wait()
}

func TestConcurrent_TeamRegistryLookups(t *testing.T) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) {
			return "role-" + agentID, nil
		},
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{role: true}, nil
		},
	)

	var wg sync.WaitGroup
	var errors atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", idx%10)
			caps, err := registry.GetCapabilities(agentID)
			if err != nil {
				errors.Add(1)
				return
			}
			expected := fmt.Sprintf("role-agent-%d", idx%10)
			if caps != nil && !caps[expected] {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int64(0), errors.Load())
}

func TestConcurrent_TeamRegistryWithOverrides(t *testing.T) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) { return "editor", nil },
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{"read": true, "write": true}, nil
		},
	)

	registry.RegisterRole("editor", CapabilitySet{"read": true, "write": true, "admin": true})

	var wg sync.WaitGroup
	var hasAdmin atomic.Int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caps, err := registry.GetCapabilities("any-agent")
			if err == nil && caps != nil && caps["admin"] {
				hasAdmin.Add(1)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(50), hasAdmin.Load())
}
