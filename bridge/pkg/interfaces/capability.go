package interfaces

import (
	"context"

	"github.com/armorclaw/bridge/pkg/capability"
)

// CapabilityBroker is the middleware that evaluates every agent action before execution.
// Implementations MUST be fail-closed: any error returns DENY.
type CapabilityBroker interface {
	// Authorize evaluates an action request and returns the broker's ruling.
	// Returns ActionResponse with Classification set to ALLOW, DENY, or DEFER.
	// On any internal error, MUST return DENY (fail-closed).
	Authorize(ctx context.Context, req capability.ActionRequest) (capability.ActionResponse, error)
}

// RiskClassifier categorizes the risk domain and severity of an action.
type RiskClassifier interface {
	// Classify returns the risk class (payment, identity_pii, etc.) and risk level
	// (ALLOW, DENY, DEFER) for the given action and parameters.
	Classify(ctx context.Context, action string, params map[string]any) (capability.RiskClass, capability.RiskLevel)
}

// CapabilityRegistry maps roles to their permitted capabilities.
type CapabilityRegistry interface {
	// GetCapabilities returns the capability set for a given role.
	GetCapabilities(role string) (capability.CapabilitySet, error)
	// RegisterRole registers a role with its permitted capabilities.
	RegisterRole(role string, caps capability.CapabilitySet) error
}
