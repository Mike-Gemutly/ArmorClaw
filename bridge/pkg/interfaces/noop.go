package interfaces

import (
	"context"

	"github.com/armorclaw/bridge/pkg/capability"
)

type noopBroker struct{}

func (n *noopBroker) Authorize(_ context.Context, _ capability.ActionRequest) (capability.ActionResponse, error) {
	return capability.ActionResponse{Classification: capability.RiskDeny}, nil
}

type noopClassifier struct{}

func (n *noopClassifier) Classify(_ context.Context, _ string, _ map[string]any) (capability.RiskClass, capability.RiskLevel) {
	return "", capability.RiskDeny
}

type noopRegistry struct{}

func (n *noopRegistry) GetCapabilities(_ string) (capability.CapabilitySet, error) {
	return nil, nil
}

func (n *noopRegistry) RegisterRole(_ string, _ capability.CapabilitySet) error {
	return nil
}

type noopConsent struct{}

func (n *noopConsent) RequestConsent(_ context.Context, _, _ string, _ []string) (<-chan ConsentResult, error) {
	ch := make(chan ConsentResult, 1)
	ch <- ConsentResult{Approved: false}
	return ch, nil
}

type noopTeamStore struct{}

func (n *noopTeamStore) CreateTeam(_ context.Context, _ Team) error                { return nil }
func (n *noopTeamStore) GetTeam(_ context.Context, _ string) (Team, error)          { return Team{}, nil }
func (n *noopTeamStore) ListTeams(_ context.Context) ([]Team, error)                { return nil, nil }
func (n *noopTeamStore) AddMember(_ context.Context, _, _, _ string) error          { return nil }
func (n *noopTeamStore) RemoveMember(_ context.Context, _, _ string) error          { return nil }
func (n *noopTeamStore) DissolveTeam(_ context.Context, _ string) error             { return nil }

type noopTeamService struct{}

func (n *noopTeamService) AssignRole(_ context.Context, _, _, _ string) error {
	return nil
}

func (n *noopTeamService) GetCapabilitiesForMember(_ context.Context, _, _ string) (capability.CapabilitySet, error) {
	return nil, nil
}

func (n *noopTeamService) ValidateTeamMembership(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// Compile-time interface satisfaction checks.
var (
	_ CapabilityBroker   = (*noopBroker)(nil)
	_ RiskClassifier     = (*noopClassifier)(nil)
	_ CapabilityRegistry = (*noopRegistry)(nil)
	_ ConsentProvider    = (*noopConsent)(nil)
	_ TeamStore          = (*noopTeamStore)(nil)
	_ TeamService        = (*noopTeamService)(nil)
)
