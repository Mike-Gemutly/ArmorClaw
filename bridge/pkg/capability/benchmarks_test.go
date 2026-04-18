package capability

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkBrokerAuthorize_WithTeamRegistry(b *testing.B) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) { return "editor", nil },
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{"browse": true, "edit": true, "admin": true}, nil
		},
	)

	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		registry,
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{AgentID: "agent-1", TeamID: "team-1", Action: "browse"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Authorize(context.Background(), req)
	}
}

func BenchmarkTeamCapabilityRegistry_GetCapabilities(b *testing.B) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) {
			roles := []string{"viewer", "editor", "admin", "lead", "clerk", "worker"}
			return roles[len(agentID)%len(roles)], nil
		},
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{role: true, "common": true}, nil
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentID := fmt.Sprintf("agent-%d", i%100)
		registry.GetCapabilities(agentID)
	}
}

func BenchmarkRiskClassifier_Classify(b *testing.B) {
	classifier := &mockClassifier{class: RiskExternalCommunication, level: RiskAllow}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.Classify(context.Background(), "browse", map[string]any{"url": "https://example.com"})
	}
}

func BenchmarkTeamCapabilityRegistry_WithOverrides(b *testing.B) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) { return "editor", nil },
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{"browse": true, "edit": true}, nil
		},
	)

	for i := 0; i < 10; i++ {
		role := fmt.Sprintf("role-%d", i)
		registry.RegisterRole(role, CapabilitySet{role: true, "override": true})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.GetCapabilities(fmt.Sprintf("agent-%d", i%100))
	}
}

func BenchmarkBrokerAuthorize_DenyPath(b *testing.B) {
	broker := NewBroker(
		&mockClassifier{class: RiskPayment, level: RiskDeny},
		&mockRegistry{caps: CapabilitySet{"pay": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{AgentID: "agent-1", TeamID: "team-1", Action: "charge"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Authorize(context.Background(), req)
	}
}

func BenchmarkBrokerAuthorize_Parallel(b *testing.B) {
	broker := NewBroker(
		&mockClassifier{class: RiskExternalCommunication, level: RiskAllow},
		&mockRegistry{caps: CapabilitySet{"browse": true}},
		nil,
		&mockSkillGate{},
		DefaultBrokerConfig(),
	)

	req := ActionRequest{AgentID: "agent-1", TeamID: "team-1", Action: "browse"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			broker.Authorize(context.Background(), req)
		}
	})
}

func BenchmarkTeamCapabilityRegistry_ParallelLookup(b *testing.B) {
	registry := NewTeamCapabilityRegistry(
		func(agentID string) (string, error) { return "editor", nil },
		func(role string) (CapabilitySet, error) {
			return CapabilitySet{"browse": true, "edit": true}, nil
		},
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			registry.GetCapabilities(fmt.Sprintf("agent-%d", i%100))
			i++
		}
	})
}
