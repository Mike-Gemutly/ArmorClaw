package team

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrent_CreateTeams_NoPanic(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			team := validTeam(fmt.Sprintf("concurrent-team-%d", idx), fmt.Sprintf("Team %d", idx))
			store.CreateTeam(ctx, team)
		}(i)
	}

	wg.Wait()

	teams, err := store.ListTeams(ctx)
	require.NoError(t, err)
	assert.True(t, len(teams) >= 1, "at least some teams should be created without panicking")
}

func TestConcurrent_AddRemoveMembers(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("team-ar", "AddRemove Team")
	require.NoError(t, store.CreateTeam(ctx, team))

	for i := 0; i < 5; i++ {
		m := validMember("team-ar", fmt.Sprintf("agent-%d", i), "worker")
		require.NoError(t, store.AddMember(ctx, m))
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			store.AddMember(ctx, validMember("team-ar", fmt.Sprintf("extra-%d", idx), "worker"))
		}(i)
		go func(idx int) {
			defer wg.Done()
			store.RemoveMember(ctx, "team-ar", fmt.Sprintf("agent-%d", idx))
		}(i)
	}

	wg.Wait()

	fetched, err := store.GetTeam(ctx, "team-ar")
	require.NoError(t, err)
	assert.NotNil(t, fetched)
}

func TestConcurrent_GetTeamWhileUpdating(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("team-rw", "ReadWrite Team")
	require.NoError(t, store.CreateTeam(ctx, team))

	var wg sync.WaitGroup

	for i := 0; i < 25; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			store.GetTeam(ctx, "team-rw")
		}()
		go func(idx int) {
			defer wg.Done()
			m := validMember("team-rw", fmt.Sprintf("member-%d", idx), "member")
			store.AddMember(ctx, m)
		}(i)
	}

	wg.Wait()
}

func TestConcurrent_MetricsRecording(t *testing.T) {
	metrics := NewTeamMetrics()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			teamID := fmt.Sprintf("team-%d", idx%5)
			metrics.RecordTokenUsage(teamID, int64(idx))
			metrics.RecordCost(teamID, int64(idx))
			metrics.RecordHandoff(teamID, idx%2 == 0)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 5; i++ {
		teamID := fmt.Sprintf("team-%d", i)
		snapshot := metrics.GetSnapshot(teamID)
		assert.NotNil(t, snapshot)
	}
}

func TestConcurrent_GovernanceChecks(t *testing.T) {
	var teamCount atomic.Int64
	enforcer := NewGovernanceEnforcer(GovernanceConfig{
		MaxMembersPerTeam:   10,
		MaxTeamsPerInstance: 100,
		AllowedRoles:        []string{"worker", "lead", "clerk"},
	}, func() int { return int(teamCount.Load()) })

	var wg sync.WaitGroup
	var validationErrors atomic.Int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			role := []string{"worker", "lead", "clerk"}[idx%3]
			if err := enforcer.ValidateMemberAddition(idx%10, role); err != nil {
				validationErrors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(0), validationErrors.Load())
}

func TestConcurrent_TeamDissolveDuringAccess(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		team := validTeam(fmt.Sprintf("dissolve-%d", i), fmt.Sprintf("Dissolve Team %d", i))
		require.NoError(t, store.CreateTeam(ctx, team))
		m := validMember(fmt.Sprintf("dissolve-%d", i), "agent-1", "lead")
		require.NoError(t, store.AddMember(ctx, m))
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(2)
		teamID := fmt.Sprintf("dissolve-%d", i)
		go func(id string) {
			defer wg.Done()
			store.DissolveTeam(ctx, id)
		}(teamID)
		go func(id string) {
			defer wg.Done()
			store.GetTeam(ctx, id)
		}(teamID)
	}

	wg.Wait()
}
