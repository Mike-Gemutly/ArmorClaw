package main

import (
	"context"
	"log"

	"github.com/armorclaw/bridge/pkg/capability"
	"github.com/armorclaw/bridge/pkg/interfaces"
)

// consentAdapter wraps interfaces.ConsentProvider to satisfy capability.consentProvider.
// Required because capability defines its own local types to avoid import cycles.
type consentAdapter struct {
	inner interfaces.ConsentProvider
}

func (a *consentAdapter) RequestConsent(ctx context.Context, requestID, reason string, fields []string) (<-chan capability.ConsentResult, error) {
	ch, err := a.inner.RequestConsent(ctx, requestID, reason, fields)
	if err != nil {
		return nil, err
	}
	out := make(chan capability.ConsentResult, 1)
	go func() {
		defer close(out)
		r := <-ch
		out <- capability.ConsentResult{
			Approved:       r.Approved,
			ApprovedFields: r.ApprovedFields,
			DeniedFields:   r.DeniedFields,
			Error:          r.Error,
		}
	}()
	return out, nil
}

// skillGateAdapter wraps interfaces.SkillGate to satisfy capability.skillGate.
type skillGateAdapter struct {
	inner interfaces.SkillGate
}

func (a *skillGateAdapter) InterceptToolCall(ctx context.Context, call *capability.ToolCall) (*capability.ToolCall, error) {
	ifc := &interfaces.ToolCall{
		ToolName:  call.ToolName,
		Arguments: call.Arguments,
	}
	result, err := a.inner.InterceptToolCall(ctx, ifc)
	if err != nil {
		return nil, err
	}
	return &capability.ToolCall{
		ToolName:  result.ToolName,
		Arguments: result.Arguments,
	}, nil
}

// setupBroker creates the CapabilityBroker with its dependencies.
// Returns nil when governance is not enabled (broker is optional).
func setupBroker(classifier interfaces.RiskClassifier, registry interfaces.CapabilityRegistry, consent interfaces.ConsentProvider, skillGate interfaces.SkillGate) *capability.Broker {
	if classifier == nil {
		log.Println("[BROKER] no risk classifier — broker not created")
		return nil
	}

	cfg := capability.DefaultBrokerConfig()
	broker := capability.NewBroker(classifier, registry, &consentAdapter{inner: consent}, &skillGateAdapter{inner: skillGate}, cfg)
	log.Println("[BROKER] capability broker created with default config")
	return broker
}
