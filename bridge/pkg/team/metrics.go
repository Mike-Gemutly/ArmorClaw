package team

import (
	"sync"
	"time"
)

// TeamMetricsSnapshot is a read-only copy of metrics for a single team.
type TeamMetricsSnapshot struct {
	TeamID        string
	TokenUsage    int64
	CostCents     int64
	AvgLatency    time.Duration
	HandoffsTotal int64
	HandoffsFail  int64
	SecretAccesses int64
	ApprovalsByRisk map[string]int64
}

// TeamMetrics tracks per-team operational metrics.
// All methods are goroutine-safe.
type TeamMetrics struct {
	mu              sync.RWMutex
	tokenUsage      map[string]int64              // teamID → tokens
	costCents       map[string]int64              // teamID → cost in cents
	latencySum      map[string]int64              // role → cumulative nanoseconds
	latencyCount    map[string]int64              // role → count
	handoffsTotal   map[string]int64              // teamID → handoff count
	handoffsFailed  map[string]int64              // teamID → failed handoffs
	secretAccesses  map[string]int64              // teamID → secret access count
	approvalsByRisk map[string]map[string]int64   // teamID → riskClass → count
}

// NewTeamMetrics creates a ready-to-use TeamMetrics.
func NewTeamMetrics() *TeamMetrics {
	return &TeamMetrics{
		tokenUsage:      make(map[string]int64),
		costCents:       make(map[string]int64),
		latencySum:      make(map[string]int64),
		latencyCount:    make(map[string]int64),
		handoffsTotal:   make(map[string]int64),
		handoffsFailed:  make(map[string]int64),
		secretAccesses:  make(map[string]int64),
		approvalsByRisk: make(map[string]map[string]int64),
	}
}

// RecordTokenUsage adds token consumption for a team.
func (m *TeamMetrics) RecordTokenUsage(teamID string, tokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenUsage[teamID] += tokens
}

// RecordCost adds cost in cents for a team.
func (m *TeamMetrics) RecordCost(teamID string, cents int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.costCents[teamID] += cents
}

// RecordLatency records a latency observation for a role.
func (m *TeamMetrics) RecordLatency(role string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencySum[role] += duration.Nanoseconds()
	m.latencyCount[role]++
}

// RecordHandoff records a handoff event. If success is false it also
// increments the failed counter.
func (m *TeamMetrics) RecordHandoff(teamID string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handoffsTotal[teamID]++
	if !success {
		m.handoffsFailed[teamID]++
	}
}

// RecordSecretAccess increments the secret access counter for a team.
func (m *TeamMetrics) RecordSecretAccess(teamID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secretAccesses[teamID]++
}

// RecordApproval records an approval event for a team and risk class.
func (m *TeamMetrics) RecordApproval(teamID, riskClass string, approved bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := riskClass
	if !approved {
		key = riskClass + ":denied"
	}
	if m.approvalsByRisk[teamID] == nil {
		m.approvalsByRisk[teamID] = make(map[string]int64)
	}
	m.approvalsByRisk[teamID][key]++
}

// GetSnapshot returns a point-in-time copy of metrics for the given team.
func (m *TeamMetrics) GetSnapshot(teamID string) TeamMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgLatency time.Duration
	sum := m.latencySum["_team:"+teamID]
	cnt := m.latencyCount["_team:"+teamID]
	if cnt > 0 {
		avgLatency = time.Duration(sum / cnt)
	}

	snap := TeamMetricsSnapshot{
		TeamID:         teamID,
		TokenUsage:     m.tokenUsage[teamID],
		CostCents:      m.costCents[teamID],
		AvgLatency:     avgLatency,
		HandoffsTotal:  m.handoffsTotal[teamID],
		HandoffsFail:   m.handoffsFailed[teamID],
		SecretAccesses: m.secretAccesses[teamID],
	}

	if risks, ok := m.approvalsByRisk[teamID]; ok {
		snap.ApprovalsByRisk = make(map[string]int64, len(risks))
		for k, v := range risks {
			snap.ApprovalsByRisk[k] = v
		}
	}

	return snap
}
