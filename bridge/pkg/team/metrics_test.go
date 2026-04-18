package team

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTeamMetrics(t *testing.T) {
	m := NewTeamMetrics()
	assert.NotNil(t, m)
}

func TestTeamMetrics_RecordTokenUsage(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordTokenUsage("team-1", 100)
	m.RecordTokenUsage("team-1", 50)
	m.RecordTokenUsage("team-2", 200)

	snap1 := m.GetSnapshot("team-1")
	assert.Equal(t, int64(150), snap1.TokenUsage)

	snap2 := m.GetSnapshot("team-2")
	assert.Equal(t, int64(200), snap2.TokenUsage)
}

func TestTeamMetrics_RecordCost(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordCost("team-1", 500)
	m.RecordCost("team-1", 300)

	snap := m.GetSnapshot("team-1")
	assert.Equal(t, int64(800), snap.CostCents)
}

func TestTeamMetrics_RecordLatency(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordLatency("browser_specialist", 100*time.Millisecond)
	m.RecordLatency("browser_specialist", 200*time.Millisecond)
	m.RecordLatency("doc_analyst", 50*time.Millisecond)

	snap := m.GetSnapshot("team-1")
	assert.Equal(t, "team-1", snap.TeamID)
}

func TestTeamMetrics_RecordHandoff(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordHandoff("team-1", true)
	m.RecordHandoff("team-1", true)
	m.RecordHandoff("team-1", false)

	snap := m.GetSnapshot("team-1")
	assert.Equal(t, int64(3), snap.HandoffsTotal)
	assert.Equal(t, int64(1), snap.HandoffsFail)
}

func TestTeamMetrics_RecordSecretAccess(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordSecretAccess("team-1")
	m.RecordSecretAccess("team-1")
	m.RecordSecretAccess("team-1")

	snap := m.GetSnapshot("team-1")
	assert.Equal(t, int64(3), snap.SecretAccesses)
}

func TestTeamMetrics_RecordApproval(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordApproval("team-1", "payment", true)
	m.RecordApproval("team-1", "payment", true)
	m.RecordApproval("team-1", "payment", false)
	m.RecordApproval("team-1", "identity_pii", true)

	snap := m.GetSnapshot("team-1")
	require.NotNil(t, snap.ApprovalsByRisk)
	assert.Equal(t, int64(2), snap.ApprovalsByRisk["payment"])
	assert.Equal(t, int64(1), snap.ApprovalsByRisk["payment:denied"])
	assert.Equal(t, int64(1), snap.ApprovalsByRisk["identity_pii"])
}

func TestTeamMetrics_GetSnapshot_NonExistentTeam(t *testing.T) {
	m := NewTeamMetrics()
	snap := m.GetSnapshot("nonexistent")
	assert.Equal(t, "nonexistent", snap.TeamID)
	assert.Equal(t, int64(0), snap.TokenUsage)
	assert.Equal(t, int64(0), snap.CostCents)
	assert.Equal(t, int64(0), snap.HandoffsTotal)
	assert.Equal(t, int64(0), snap.HandoffsFail)
	assert.Equal(t, int64(0), snap.SecretAccesses)
}

func TestTeamMetrics_MultipleTeams(t *testing.T) {
	m := NewTeamMetrics()
	m.RecordTokenUsage("team-1", 100)
	m.RecordTokenUsage("team-2", 500)
	m.RecordCost("team-1", 10)
	m.RecordCost("team-2", 50)

	snap1 := m.GetSnapshot("team-1")
	snap2 := m.GetSnapshot("team-2")

	assert.Equal(t, int64(100), snap1.TokenUsage)
	assert.Equal(t, int64(10), snap1.CostCents)
	assert.Equal(t, int64(500), snap2.TokenUsage)
	assert.Equal(t, int64(50), snap2.CostCents)
}

func TestTeamMetrics_ConcurrentRecording(t *testing.T) {
	m := NewTeamMetrics()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordTokenUsage("team-1", 1)
			m.RecordCost("team-1", 1)
			m.RecordHandoff("team-1", true)
			m.RecordSecretAccess("team-1")
			m.RecordApproval("team-1", "payment", true)
		}()
	}
	wg.Wait()

	snap := m.GetSnapshot("team-1")
	assert.Equal(t, int64(100), snap.TokenUsage)
	assert.Equal(t, int64(100), snap.CostCents)
	assert.Equal(t, int64(100), snap.HandoffsTotal)
	assert.Equal(t, int64(100), snap.SecretAccesses)
}
