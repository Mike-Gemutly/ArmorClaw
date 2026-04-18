package team

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/armorclaw/bridge/pkg/capability"
)

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// CollectionCreator is a function that creates a vector collection for a team.
// It is injected by the caller (typically the sidecar client adapter) to keep
// the team package free of direct Qdrant imports.
type CollectionCreator func(ctx context.Context, collectionID string) error

// Service implements the team business logic layer. It wraps a TeamStore
// and adds validation (roles, membership) and ID generation on top.
type Service struct {
	store        *TeamStore
	OnTeamCreated CollectionCreator // optional — nil means no collection
}

// NewTeamService creates a new Service backed by the given TeamStore.
func NewTeamService(store *TeamStore) *Service {
	return &Service{store: store}
}

// compile-time check that Service satisfies the interface.
var _ TeamService = (*Service)(nil)

// TeamService is the local mirror of interfaces.TeamService so the package
// does not need to import interfaces (avoiding cycles).
type TeamService interface {
	AssignRole(ctx context.Context, teamID, agentID, role string) error
	GetCapabilitiesForMember(ctx context.Context, teamID, agentID string) (capability.CapabilitySet, error)
	ValidateTeamMembership(ctx context.Context, teamID, agentID string) (bool, error)
}

// ---------------------------------------------------------------------------
// Team CRUD (wraps store with business logic)
// ---------------------------------------------------------------------------

// CreateTeam creates a new team with a generated UUID, validates it, and
// persists it through the store. If OnTeamCreated is set, a Qdrant collection
// is created and its reference is stored in the team's SharedContext.
func (s *Service) CreateTeam(ctx context.Context, name, templateID string) (*Team, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("team: generate id: %w", err)
	}

	t := &Team{
		ID:             id,
		Name:           name,
		TemplateID:     templateID,
		LifecycleState: LifecycleActive,
	}

	if err := s.store.CreateTeam(ctx, t); err != nil {
		return nil, fmt.Errorf("team: create team: %w", err)
	}

	if s.OnTeamCreated != nil {
		collectionID := "team_" + t.ID
		if err := s.OnTeamCreated(ctx, collectionID); err != nil {
			log.Printf("team: collection creation failed for team %s: %v", t.ID, err)
		} else {
			ctxData := map[string]string{"qdrant_collection": collectionID}
			ctxJSON, jsonErr := json.Marshal(ctxData)
			if jsonErr != nil {
				log.Printf("team: marshal shared context for team %s: %v", t.ID, jsonErr)
			} else {
				t.SharedContext = string(ctxJSON)
				if updateErr := s.store.UpdateTeam(ctx, t); updateErr != nil {
					log.Printf("team: update shared context for team %s: %v", t.ID, updateErr)
				}
			}
		}
	}

	return t, nil
}

// GetTeam retrieves a team by ID.
func (s *Service) GetTeam(ctx context.Context, id string) (*Team, error) {
	t, err := s.store.GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team: get team: %w", err)
	}
	return t, nil
}

// ListTeams returns all teams.
func (s *Service) ListTeams(ctx context.Context) ([]Team, error) {
	teams, err := s.store.ListTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("team: list teams: %w", err)
	}
	return teams, nil
}

// DissolveTeam marks a team as dissolved. Data is preserved for audit.
func (s *Service) DissolveTeam(ctx context.Context, teamID string) error {
	if err := s.store.DissolveTeam(ctx, teamID); err != nil {
		return fmt.Errorf("team: dissolve team: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Members
// ---------------------------------------------------------------------------

// AddMember adds a member to a team after validating the role exists and the
// assignment is valid (e.g., no duplicate team_lead).
func (s *Service) AddMember(ctx context.Context, teamID, agentID, roleName string) (*TeamMember, error) {
	if _, err := GetRole(roleName); err != nil {
		return nil, fmt.Errorf("team: add member: %w", err)
	}

	existingRoles, err := s.existingRoles(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if err := ValidateRoleAssignment(roleName, existingRoles); err != nil {
		return nil, fmt.Errorf("team: add member: %w", err)
	}

	m := &TeamMember{
		TeamID:   teamID,
		AgentID:  agentID,
		RoleName: roleName,
	}

	if err := s.store.AddMember(ctx, m); err != nil {
		return nil, fmt.Errorf("team: add member: %w", err)
	}

	return m, nil
}

// RemoveMember removes a member from a team. If zero members remain, the team
// is automatically dissolved.
func (s *Service) RemoveMember(ctx context.Context, teamID, agentID string) error {
	if err := s.store.RemoveMember(ctx, teamID, agentID); err != nil {
		return fmt.Errorf("team: remove member: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Interface methods
// ---------------------------------------------------------------------------

// AssignRole changes a member's role by updating the role_name column directly.
// This avoids the auto-dissolve that RemoveMember triggers when the member is
// the last one in the team.
func (s *Service) AssignRole(ctx context.Context, teamID, agentID, role string) error {
	if _, err := GetRole(role); err != nil {
		return fmt.Errorf("team: assign role: %w", err)
	}

	if role == "team_lead" {
		team, err := s.store.GetTeam(ctx, teamID)
		if err != nil {
			return fmt.Errorf("team: assign role: %w", err)
		}
		for _, m := range team.Members {
			if m.RoleName == "team_lead" && m.AgentID != agentID {
				return fmt.Errorf("team: assign role: team: duplicate team_lead assignment")
			}
		}
	}

	result, err := s.store.DB().ExecContext(ctx,
		"UPDATE team_members SET role_name = ? WHERE team_id = ? AND agent_id = ?",
		role, teamID, agentID,
	)
	if err != nil {
		return fmt.Errorf("team: assign role update: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}

	return nil
}

// GetCapabilitiesForMember returns the capability set associated with a
// member's current role.
func (s *Service) GetCapabilitiesForMember(ctx context.Context, teamID, agentID string) (capability.CapabilitySet, error) {
	t, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("team: get capabilities: %w", err)
	}

	var memberRole string
	for _, m := range t.Members {
		if m.AgentID == agentID {
			memberRole = m.RoleName
			break
		}
	}

	if memberRole == "" {
		return nil, fmt.Errorf("team: agent %q is not a member of team %q", agentID, teamID)
	}

	role, err := GetRole(memberRole)
	if err != nil {
		return nil, fmt.Errorf("team: get capabilities: %w", err)
	}

	return role.Capabilities, nil
}

// ValidateTeamMembership checks that a team exists and the given agent is a
// member. Returns (true, nil) when valid, (false, nil) when not a member,
// and (false, err) on store errors.
func (s *Service) ValidateTeamMembership(ctx context.Context, teamID, agentID string) (bool, error) {
	t, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return false, fmt.Errorf("team: validate membership: %w", err)
	}

	for _, m := range t.Members {
		if m.AgentID == agentID {
			return true, nil
		}
	}

	return false, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// existingRoles returns the role names of all current members in a team.
func (s *Service) existingRoles(ctx context.Context, teamID string) ([]string, error) {
	t, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("team: get team for role check: %w", err)
	}

	roles := make([]string, 0, len(t.Members))
	for _, m := range t.Members {
		roles = append(roles, m.RoleName)
	}
	return roles, nil
}

// generateID creates a random 16-byte hex string suitable for use as a team ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
