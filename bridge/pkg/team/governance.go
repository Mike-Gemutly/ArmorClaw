package team

import (
	"errors"
	"fmt"
	"strings"
)

// GovernanceConfig defines team size and role limits.
type GovernanceConfig struct {
	MaxMembersPerTeam   int      `json:"max_members_per_team"`   // default 10
	MaxTeamsPerInstance int      `json:"max_teams_per_instance"` // default 5
	AllowedRoles        []string `json:"allowed_roles"`          // empty = all allowed
}

// DefaultGovernanceConfig returns sensible defaults.
func DefaultGovernanceConfig() GovernanceConfig {
	return GovernanceConfig{
		MaxMembersPerTeam:   10,
		MaxTeamsPerInstance: 5,
		AllowedRoles:        nil, // all allowed
	}
}

// GovernanceEnforcer checks team operations against governance limits.
type GovernanceEnforcer struct {
	config     GovernanceConfig
	teamCount  func() int // function injection for current team count
	overrides  map[string]map[string]string // teamID → riskClass → "ALLOW" or "DEFER"
}

// NewGovernanceEnforcer creates a new enforcer with the given config and
// teamCount function. The teamCount function is called to determine the
// current number of teams in the instance.
func NewGovernanceEnforcer(config GovernanceConfig, teamCount func() int) *GovernanceEnforcer {
	return &GovernanceEnforcer{
		config:    config,
		teamCount: teamCount,
		overrides: make(map[string]map[string]string),
	}
}

var (
	ErrMaxTeamsExceeded   = errors.New("governance: max teams per instance exceeded")
	ErrMaxMembersExceeded = errors.New("governance: max members per team exceeded")
	ErrRoleNotAllowed     = errors.New("governance: role not in allowed list")
)

// ValidateTeamCreation checks that creating a new team would not exceed
// MaxTeamsPerInstance.
func (g *GovernanceEnforcer) ValidateTeamCreation() error {
	if g.teamCount == nil {
		return nil
	}
	if g.config.MaxTeamsPerInstance > 0 && g.teamCount() >= g.config.MaxTeamsPerInstance {
		return ErrMaxTeamsExceeded
	}
	return nil
}

// ValidateMemberAddition checks that adding a member would not exceed
// MaxMembersPerTeam and that the role is in AllowedRoles (if configured).
func (g *GovernanceEnforcer) ValidateMemberAddition(currentMembers int, roleName string) error {
	if g.config.MaxMembersPerTeam > 0 && currentMembers >= g.config.MaxMembersPerTeam {
		return ErrMaxMembersExceeded
	}
	return g.ValidateRoleAssignment(roleName)
}

// ValidateRoleAssignment checks that the role is in AllowedRoles.
// If AllowedRoles is empty/nil, all roles are permitted.
func (g *GovernanceEnforcer) ValidateRoleAssignment(roleName string) error {
	if len(g.config.AllowedRoles) == 0 {
		return nil
	}
	for _, allowed := range g.config.AllowedRoles {
		if strings.EqualFold(allowed, roleName) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrRoleNotAllowed, roleName)
}

// ---------------------------------------------------------------------------
// Task 52: Team Policy Overrides
// ---------------------------------------------------------------------------

// TeamPolicyOverride defines team-specific risk classification overrides.
type TeamPolicyOverride struct {
	TeamID        string
	RiskOverrides map[string]string // riskClass → "ALLOW" or "DEFER"
}

// SetPolicyOverride registers a team-specific override for risk classifications.
func (g *GovernanceEnforcer) SetPolicyOverride(override TeamPolicyOverride) {
	if g.overrides == nil {
		g.overrides = make(map[string]map[string]string)
	}
	g.overrides[override.TeamID] = override.RiskOverrides
}

// ResolveActionPolicy checks team overrides before falling back to global
// policy. If the team has an override for the given risk class it is used;
// otherwise the globalPolicy value is returned.
func (g *GovernanceEnforcer) ResolveActionPolicy(teamID, riskClass, globalPolicy string) string {
	if teamOverrides, ok := g.overrides[teamID]; ok {
		if override, ok := teamOverrides[riskClass]; ok {
			return override
		}
	}
	return globalPolicy
}
