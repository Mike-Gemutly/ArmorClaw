// Package team defines core types for multi-agent team management.
// Every struct carries JSON tags for wire serialization and a Validate
// method for input checking.
package team

import (
	"fmt"

	"github.com/armorclaw/bridge/pkg/capability"
)

// ---------------------------------------------------------------------------
// Lifecycle state
// ---------------------------------------------------------------------------

// LifecycleState represents the lifecycle state of a team.
type LifecycleState string

const (
	LifecycleActive    LifecycleState = "active"
	LifecycleSuspended LifecycleState = "suspended"
	LifecycleDissolved LifecycleState = "dissolved"
)

// ---------------------------------------------------------------------------
// Team
// ---------------------------------------------------------------------------

// Team represents a multi-agent team.
type Team struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	TemplateID     string         `json:"template_id,omitempty"`
	SharedContext  string         `json:"shared_context,omitempty"`
	LifecycleState LifecycleState `json:"lifecycle_state"`
	Budgets        *TeamBudgets   `json:"budgets,omitempty"`
	Version        int            `json:"version"` // optimistic locking
}

// Validate ensures ID, Name are non-empty and LifecycleState is recognised.
func (v *Team) Validate() error {
	if v.ID == "" {
		return fmt.Errorf("team: %T: field id is required", v)
	}
	if v.Name == "" {
		return fmt.Errorf("team: %T: field name is required", v)
	}
	switch v.LifecycleState {
	case LifecycleActive, LifecycleSuspended, LifecycleDissolved:
	default:
		return fmt.Errorf("team: %T: field lifecycle_state has invalid value %q", v, v.LifecycleState)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TeamBudgets
// ---------------------------------------------------------------------------

// TeamBudgets tracks resource limits for a team. All fields are optional.
type TeamBudgets struct {
	MaxTokenUsage   int64   `json:"max_token_usage,omitempty"`
	MaxCost         float64 `json:"max_cost,omitempty"`
	MaxDuration     string  `json:"max_duration,omitempty"` // Go duration string
	MaxSecretAccess int     `json:"max_secret_access,omitempty"`
}

// Validate always returns nil — all fields are optional.
func (v *TeamBudgets) Validate() error {
	return nil
}

// ---------------------------------------------------------------------------
// TeamMember
// ---------------------------------------------------------------------------

// TeamMember represents an agent's membership in a team.
type TeamMember struct {
	TeamID                string   `json:"team_id"`
	AgentID               string   `json:"agent_id"`
	RoleName              string   `json:"role_name"`
	AllowedTools          []string `json:"allowed_tools,omitempty"`
	AllowedSecretPrefixes []string `json:"allowed_secret_prefixes,omitempty"`
	BrowserContextID      string   `json:"browser_context_id,omitempty"`
	Priority              int      `json:"priority,omitempty"`
}

// Validate ensures TeamID, AgentID, and RoleName are non-empty.
func (v *TeamMember) Validate() error {
	if v.TeamID == "" {
		return fmt.Errorf("team: %T: field team_id is required", v)
	}
	if v.AgentID == "" {
		return fmt.Errorf("team: %T: field agent_id is required", v)
	}
	if v.RoleName == "" {
		return fmt.Errorf("team: %T: field role_name is required", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TeamRole
// ---------------------------------------------------------------------------

// TeamRole defines a named role with capabilities.
type TeamRole struct {
	Name         string                   `json:"name"`
	Capabilities capability.CapabilitySet `json:"capabilities"`
	Description  string                   `json:"description,omitempty"`
}

// Validate ensures Name is non-empty and Capabilities, when non-nil, is not
// an empty map.
func (v *TeamRole) Validate() error {
	if v.Name == "" {
		return fmt.Errorf("team: %T: field name is required", v)
	}
	if err := v.Capabilities.Validate(); err != nil {
		return fmt.Errorf("team: %T: %w", v, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TeamTemplate
// ---------------------------------------------------------------------------

// TeamTemplate defines a reusable team configuration.
type TeamTemplate struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Roles          []TeamRole `json:"roles"`
	DefaultContext string     `json:"default_context,omitempty"`
}

// Validate ensures ID, Name are non-empty and Roles has at least one entry.
func (v *TeamTemplate) Validate() error {
	if v.ID == "" {
		return fmt.Errorf("team: %T: field id is required", v)
	}
	if v.Name == "" {
		return fmt.Errorf("team: %T: field name is required", v)
	}
	if len(v.Roles) == 0 {
		return fmt.Errorf("team: %T: at least one role is required", v)
	}
	return nil
}
