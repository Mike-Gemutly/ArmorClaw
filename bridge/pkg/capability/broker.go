package capability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BrokerConfig holds configuration for the Broker.
type BrokerConfig struct {
	DeferTimeout        time.Duration
	MaxConcurrentDefers int
	MaxCallDepth        int
}

func DefaultBrokerConfig() BrokerConfig {
	return BrokerConfig{
		DeferTimeout:        300 * time.Second,
		MaxConcurrentDefers: 50,
		MaxCallDepth:        5,
	}
}

// Local interfaces to avoid import cycle: pkg/interfaces → pkg/capability.
// These mirror the interfaces in pkg/interfaces/{capability,consent,skillgate}.go.

type riskClassifier interface {
	Classify(ctx context.Context, action string, params map[string]any) (RiskClass, RiskLevel)
}

type capabilityRegistry interface {
	GetCapabilities(role string) (CapabilitySet, error)
	RegisterRole(role string, caps CapabilitySet) error
}

type consentResult struct {
	Approved       bool
	ApprovedFields []string
	DeniedFields   []string
	Error          error
}

type consentProvider interface {
	RequestConsent(ctx context.Context, requestID, reason string, fields []string) (<-chan consentResult, error)
}

type toolCall struct {
	ToolName  string
	Arguments map[string]any
}

type skillGate interface {
	InterceptToolCall(ctx context.Context, call *toolCall) (*toolCall, error)
}

// Broker is fail-closed: any error, panic, or missing dependency → DENY.
// Composes riskClassifier, capabilityRegistry, consentProvider, skillGate (HAS-A).
type Broker struct {
	classifier riskClassifier
	registry   capabilityRegistry
	consent    consentProvider
	skillGate  skillGate
	config     BrokerConfig

	mu           sync.Mutex
	activeDefers int
}

func NewBroker(
	classifier riskClassifier,
	registry capabilityRegistry,
	consent consentProvider,
	skillGate skillGate,
	config BrokerConfig,
) *Broker {
	if config.DeferTimeout == 0 {
		config.DeferTimeout = 300 * time.Second
	}
	if config.MaxConcurrentDefers == 0 {
		config.MaxConcurrentDefers = 50
	}
	if config.MaxCallDepth == 0 {
		config.MaxCallDepth = 5
	}
	return &Broker{
		classifier: classifier,
		registry:   registry,
		consent:    consent,
		skillGate:  skillGate,
		config:     config,
	}
}

// Authorize evaluates an action request through:
// registry lookup → risk classification → consent (if DEFER) → PII scrub (if ALLOW).
// Fail-closed: panics and errors produce DENY, never an error return.
func (b *Broker) Authorize(ctx context.Context, req ActionRequest) (resp ActionResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			resp = ActionResponse{
				Allowed:        false,
				Classification: RiskDeny,
				Reason:         fmt.Sprintf("broker panic: %v", r),
			}
			err = nil
		}
	}()

	if err := req.Validate(); err != nil {
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			Reason:         err.Error(),
		}, nil
	}

	if depth := callDepthFromContext(ctx); depth > b.config.MaxCallDepth {
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			Reason:         fmt.Sprintf("circular dependency detected: call depth %d exceeds max %d", depth, b.config.MaxCallDepth),
		}, nil
	}

	if b.registry != nil && req.TeamID != "" {
		caps, err := b.registry.GetCapabilities(req.AgentID)
		if err != nil {
			return ActionResponse{
				Allowed:        false,
				Classification: RiskDeny,
				Reason:         fmt.Sprintf("registry lookup failed: %v", err),
			}, nil
		}
		if caps != nil && !caps[req.Action] {
			return ActionResponse{
				Allowed:        false,
				Classification: RiskDeny,
				Reason:         fmt.Sprintf("capability %q not in role", req.Action),
			}, nil
		}
	}

	var riskClass RiskClass
	var riskLevel RiskLevel
	if b.classifier != nil {
		riskClass, riskLevel = b.classifier.Classify(ctx, req.Action, req.Params)
	} else {
		riskLevel = RiskDeny
		riskClass = RiskIrreversibleAction
	}

	switch riskLevel {
	case RiskDeny:
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         fmt.Sprintf("action %q classified as %s: denied", req.Action, riskClass),
		}, nil

	case RiskDefer:
		return b.handleDefer(ctx, req, riskClass)

	case RiskAllow:
		if b.skillGate != nil {
			call := &toolCall{
				ToolName:  req.Action,
				Arguments: req.Params,
			}
			scrubbed, scrubErr := b.skillGate.InterceptToolCall(ctx, call)
			if scrubErr != nil {
				return ActionResponse{
					Allowed:        false,
					Classification: RiskDeny,
					RiskClass:      riskClass,
					Reason:         fmt.Sprintf("PII scrub failed: %v", scrubErr),
				}, nil
			}
			if scrubbed != nil {
				req.Params = scrubbed.Arguments
			}
		}

		return ActionResponse{
			Allowed:        true,
			Classification: RiskAllow,
			RiskClass:      riskClass,
		}, nil

	default:
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			Reason:         fmt.Sprintf("unknown risk level: %s", riskLevel),
		}, nil
	}
}

func (b *Broker) handleDefer(ctx context.Context, req ActionRequest, riskClass RiskClass) (ActionResponse, error) {
	b.mu.Lock()
	if b.activeDefers >= b.config.MaxConcurrentDefers {
		b.mu.Unlock()
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         "consent queue full",
		}, nil
	}
	b.activeDefers++
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.activeDefers--
		b.mu.Unlock()
	}()

	if b.consent == nil {
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         "deferred but no consent provider configured",
		}, nil
	}

	consentCh, err := b.consent.RequestConsent(ctx, req.AgentID,
		fmt.Sprintf("action %q classified as %s requires approval", req.Action, riskClass),
		nil,
	)
	if err != nil {
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         fmt.Sprintf("consent request failed: %v", err),
		}, nil
	}

	select {
	case result := <-consentCh:
		if result.Error != nil {
			return ActionResponse{
				Allowed:        false,
				Classification: RiskDeny,
				RiskClass:      riskClass,
				Reason:         fmt.Sprintf("consent error: %v", result.Error),
			}, nil
		}
		if result.Approved {
			return ActionResponse{
				Allowed:        true,
				Classification: RiskAllow,
				RiskClass:      riskClass,
				Reason:         "approved by consent",
			}, nil
		}
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         "denied by consent",
		}, nil

	case <-time.After(b.config.DeferTimeout):
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         fmt.Sprintf("consent timeout (%0.0fs auto-deny)", b.config.DeferTimeout.Seconds()),
		}, nil

	case <-ctx.Done():
		return ActionResponse{
			Allowed:        false,
			Classification: RiskDeny,
			RiskClass:      riskClass,
			Reason:         fmt.Sprintf("context cancelled: %v", ctx.Err()),
		}, nil
	}
}

type callDepthKeyType struct{}

var callDepthKey callDepthKeyType

func WithCallDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, callDepthKey, depth)
}

func callDepthFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	if v := ctx.Value(callDepthKey); v != nil {
		if d, ok := v.(int); ok {
			return d
		}
	}
	return 0
}
