package capability

import (
	"fmt"
	"sync"
)

// RoleResolver resolves an agentID to a team role name.
// Return an empty string with nil error for agents without team membership.
type RoleResolver func(agentID string) (role string, err error)

// RoleLookupFunc looks up a role's capabilities by role name.
// Injected at construction to avoid import cycle with pkg/team.
type RoleLookupFunc func(roleName string) (CapabilitySet, error)

// TeamCapabilityRegistry implements the capabilityRegistry interface by
// delegating capability lookups to the team role system.
//
// Flow: GetCapabilities(agentID) → RoleResolver(agentID) → role name →
// RoleLookupFunc(role) → CapabilitySet
//
// Custom role overrides can be registered via RegisterRole. Overrides take
// precedence over the injected role lookup.
type TeamCapabilityRegistry struct {
	mu        sync.RWMutex
	overrides map[string]CapabilitySet
	resolve   RoleResolver
	lookup    RoleLookupFunc
}

// NewTeamCapabilityRegistry creates a TeamCapabilityRegistry.
// resolver maps agentID → role name. lookup maps role name → CapabilitySet.
// Either may be nil; nil resolver causes GetCapabilities to error,
// nil lookup causes unknown roles to error after overrides are checked.
func NewTeamCapabilityRegistry(resolver RoleResolver, lookup RoleLookupFunc) *TeamCapabilityRegistry {
	return &TeamCapabilityRegistry{
		overrides: make(map[string]CapabilitySet),
		resolve:   resolver,
		lookup:    lookup,
	}
}

// GetCapabilities resolves agentID to a role via the injected RoleResolver,
// then returns the CapabilitySet for that role.
//
// Precedence: overrides map → RoleLookupFunc (built-in).
// If the resolver returns an empty role (agent has no team membership),
// returns nil CapabilitySet with nil error — the broker treats this as
// "no capability restrictions from team" and falls through to risk
// classification.
func (r *TeamCapabilityRegistry) GetCapabilities(agentID string) (CapabilitySet, error) {
	if r.resolve == nil {
		return nil, fmt.Errorf("capability: team registry: no role resolver configured")
	}

	role, err := r.resolve(agentID)
	if err != nil {
		return nil, fmt.Errorf("capability: team registry: resolve agent %q: %w", agentID, err)
	}

	if role == "" {
		return nil, nil
	}

	r.mu.RLock()
	if caps, ok := r.overrides[role]; ok {
		r.mu.RUnlock()
		return caps, nil
	}
	r.mu.RUnlock()

	if r.lookup == nil {
		return nil, fmt.Errorf("capability: team registry: no role lookup configured")
	}
	caps, err := r.lookup(role)
	if err != nil {
		return nil, fmt.Errorf("capability: team registry: %w", err)
	}
	return caps, nil
}

// RegisterRole stores a custom CapabilitySet for the given role name.
// Overrides take precedence over injected role lookups.
func (r *TeamCapabilityRegistry) RegisterRole(role string, caps CapabilitySet) error {
	if role == "" {
		return fmt.Errorf("capability: team registry: role name must not be empty")
	}
	if caps == nil || len(caps) == 0 {
		return fmt.Errorf("capability: team registry: capability set must not be empty")
	}
	r.mu.Lock()
	r.overrides[role] = caps
	r.mu.Unlock()
	return nil
}
