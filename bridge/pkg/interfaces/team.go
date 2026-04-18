package interfaces

import (
	"context"

	"github.com/armorclaw/bridge/pkg/capability"
)

// Team represents a group of agents working together.
type Team struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Members     []Member `json:"members,omitempty"`
	Active      bool     `json:"active"`
}

// Member represents an agent in a team.
type Member struct {
	AgentID string `json:"agent_id"`
	Role    string `json:"role"`
}

// TeamStore persists team data. Implementations may use SQLCipher, memory, etc.
type TeamStore interface {
	CreateTeam(ctx context.Context, team Team) error
	GetTeam(ctx context.Context, teamID string) (Team, error)
	ListTeams(ctx context.Context) ([]Team, error)
	AddMember(ctx context.Context, teamID, agentID, role string) error
	RemoveMember(ctx context.Context, teamID, agentID string) error
	DissolveTeam(ctx context.Context, teamID string) error
}

// TeamService provides team business logic.
type TeamService interface {
	AssignRole(ctx context.Context, teamID, agentID, role string) error
	GetCapabilitiesForMember(ctx context.Context, teamID, agentID string) (capability.CapabilitySet, error)
	ValidateTeamMembership(ctx context.Context, teamID, agentID string) (bool, error)
}
