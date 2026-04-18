package team

import (
	"context"
	"database/sql"
	"testing"

	"github.com/armorclaw/bridge/pkg/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInMemoryStore(t *testing.T) *TeamStore {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_foreign_keys=ON")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	store, err := NewTeamStoreFromDB(db)
	require.NoError(t, err)
	return store
}

func makeTeam(id, name string) *Team {
	return &Team{ID: id, Name: name, LifecycleState: LifecycleActive}
}

func TestIntegration_TeamCreateMemberLookup(t *testing.T) {
	store := newInMemoryStore(t)

	team := makeTeam("t-integ", "Integration Team")
	require.NoError(t, store.CreateTeam(context.Background(), team))

	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID:   team.ID,
		AgentID:  "agent-1",
		RoleName: "browser_specialist",
	}))

	got, err := store.GetTeam(context.Background(), team.ID)
	require.NoError(t, err)
	assert.Len(t, got.Members, 1)
	assert.Equal(t, "agent-1", got.Members[0].AgentID)
	assert.Equal(t, "browser_specialist", got.Members[0].RoleName)
}

func TestIntegration_TeamDissolutionBlocksAddMember(t *testing.T) {
	store := newInMemoryStore(t)

	team := makeTeam("t-dissolve", "Dissolving Team")
	require.NoError(t, store.CreateTeam(context.Background(), team))

	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID: team.ID, AgentID: "agent-1", RoleName: "browser_specialist",
	}))

	require.NoError(t, store.DissolveTeam(context.Background(), team.ID))

	err := store.AddMember(context.Background(), &TeamMember{
		TeamID: team.ID, AgentID: "agent-2", RoleName: "doc_analyst",
	})
	require.Error(t, err, "should not add member to dissolved team")
}

func TestIntegration_AutoDissolveOnLastMemberRemoval(t *testing.T) {
	store := newInMemoryStore(t)

	team := makeTeam("t-autodis", "Auto Dissolve")
	require.NoError(t, store.CreateTeam(context.Background(), team))

	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID: team.ID, AgentID: "agent-1", RoleName: "team_lead",
	}))

	require.NoError(t, store.RemoveMember(context.Background(), team.ID, "agent-1"))

	got, err := store.GetTeam(context.Background(), team.ID)
	require.NoError(t, err)
	assert.Equal(t, LifecycleDissolved, got.LifecycleState)
}

func TestIntegration_MultipleTeamMembership(t *testing.T) {
	store := newInMemoryStore(t)

	team1 := makeTeam("t-alpha", "Team Alpha")
	team2 := makeTeam("t-beta", "Team Beta")
	require.NoError(t, store.CreateTeam(context.Background(), team1))
	require.NoError(t, store.CreateTeam(context.Background(), team2))

	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID: team1.ID, AgentID: "agent-1", RoleName: "browser_specialist",
	}))
	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID: team2.ID, AgentID: "agent-1", RoleName: "doc_analyst",
	}))

	got1, _ := store.GetTeam(context.Background(), team1.ID)
	got2, _ := store.GetTeam(context.Background(), team2.ID)
	assert.Equal(t, "browser_specialist", got1.Members[0].RoleName)
	assert.Equal(t, "doc_analyst", got2.Members[0].RoleName)
}

func TestIntegration_TeamCapabilityRegistryWithStore(t *testing.T) {
	store := newInMemoryStore(t)

	team := makeTeam("t-broker", "Broker Team")
	require.NoError(t, store.CreateTeam(context.Background(), team))
	require.NoError(t, store.AddMember(context.Background(), &TeamMember{
		TeamID: team.ID, AgentID: "agent-1", RoleName: "team_lead",
	}))

	roleLookup := func(role string) (capability.CapabilitySet, error) {
		roles := map[string]capability.CapabilitySet{
			"team_lead": {"browse": true, "fill": true, "payment": true},
		}
		return roles[role], nil
	}
	roleResolver := func(agentID string) (string, error) {
		got, err := store.GetTeam(context.Background(), team.ID)
		if err != nil {
			return "", err
		}
		for _, m := range got.Members {
			if m.AgentID == agentID {
				return m.RoleName, nil
			}
		}
		return "", nil
	}

	teamReg := capability.NewTeamCapabilityRegistry(roleResolver, roleLookup)
	caps, err := teamReg.GetCapabilities("agent-1")
	require.NoError(t, err)
	assert.True(t, caps["payment"], "team_lead should have payment capability")
	assert.True(t, caps["browse"], "team_lead should have browse capability")
}

func TestIntegration_PolicyOverrideViaCapabilityRegistry(t *testing.T) {
	roleLookup := func(role string) (capability.CapabilitySet, error) {
		return capability.CapabilitySet{"browse": true}, nil
	}
	roleResolver := func(agentID string) (string, error) {
		return "browser_specialist", nil
	}

	teamReg := capability.NewTeamCapabilityRegistry(roleResolver, roleLookup)

	require.NoError(t, teamReg.RegisterRole("browser_specialist", capability.CapabilitySet{
		"browse":  true,
		"payment": true,
	}))

	caps, err := teamReg.GetCapabilities("agent-1")
	require.NoError(t, err)
	assert.True(t, caps["payment"], "override should grant payment capability")
}

func TestIntegration_ServiceRoleChangeViaStore(t *testing.T) {
	store := newInMemoryStore(t)
	svc := NewTeamService(store)

	team := makeTeam("t-rolechg", "Role Change Team")
	require.NoError(t, store.CreateTeam(context.Background(), team))

	_, err := svc.AddMember(context.Background(), team.ID, "agent-1", "browser_specialist")
	require.NoError(t, err)

	require.NoError(t, svc.AssignRole(context.Background(), team.ID, "agent-1", "team_lead"))

	got, _ := store.GetTeam(context.Background(), team.ID)
	assert.Equal(t, "team_lead", got.Members[0].RoleName)
}

func TestIntegration_ListTeamsReturnsAll(t *testing.T) {
	store := newInMemoryStore(t)

	for i := 0; i < 5; i++ {
		team := makeTeam(string(rune('A'+i)), string(rune('A'+i)))
		require.NoError(t, store.CreateTeam(context.Background(), team))
	}

	teams, err := store.ListTeams(context.Background())
	require.NoError(t, err)
	assert.Len(t, teams, 5)
}
