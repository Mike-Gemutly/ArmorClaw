package team

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordTeamEvent_GeneratesEventID(t *testing.T) {
	entry := TeamAuditEntry{
		EventType: EventTeamCreated,
		TeamID:    "team-123",
		AgentID:   "agent-456",
	}
	result := RecordTeamEvent(entry)
	assert.NotEmpty(t, result.EventID, "EventID should be auto-generated")
	assert.Contains(t, result.EventID, "te-", "EventID should have te- prefix")
	assert.Equal(t, EventTeamCreated, result.EventType)
	assert.Equal(t, "team-123", result.TeamID)
	assert.Equal(t, "agent-456", result.AgentID)
}

func TestRecordTeamEvent_PreservesExistingEventID(t *testing.T) {
	entry := TeamAuditEntry{
		EventID:   "custom-id-999",
		EventType: EventMemberAdded,
		TeamID:    "team-123",
	}
	result := RecordTeamEvent(entry)
	assert.Equal(t, "custom-id-999", result.EventID, "existing EventID should be preserved")
}

func TestRecordTeamEvent_SetsTimestamp(t *testing.T) {
	entry := TeamAuditEntry{
		EventType: EventTeamCreated,
		TeamID:    "team-123",
	}
	before := time.Now()
	result := RecordTeamEvent(entry)
	after := time.Now()
	assert.False(t, result.Timestamp.IsZero(), "Timestamp should be set")
	assert.True(t, !result.Timestamp.Before(before) && !result.Timestamp.After(after),
		"Timestamp should be close to now")
}

func TestRecordTeamEvent_PreservesExistingTimestamp(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := TeamAuditEntry{
		EventType: EventTeamDissolved,
		TeamID:    "team-123",
		Timestamp: ts,
	}
	result := RecordTeamEvent(entry)
	assert.Equal(t, ts, result.Timestamp, "existing Timestamp should be preserved")
}

func TestRecordTeamEvent_AllEventConstants(t *testing.T) {
	events := []string{
		EventTeamCreated,
		EventTeamDissolved,
		EventMemberAdded,
		EventMemberRemoved,
		EventRoleAssigned,
		EventDelegationSent,
		EventHandoffComplete,
	}
	for _, eventType := range events {
		t.Run(eventType, func(t *testing.T) {
			entry := TeamAuditEntry{
				EventType: eventType,
				TeamID:    "team-test",
			}
			result := RecordTeamEvent(entry)
			assert.Equal(t, eventType, result.EventType)
			assert.NotEmpty(t, result.EventID)
		})
	}
}

func TestRecordTeamEvent_WithDetails(t *testing.T) {
	entry := TeamAuditEntry{
		EventType: EventMemberAdded,
		TeamID:    "team-123",
		RoleName:  "browser_specialist",
		Details: map[string]string{
			"added_by": "team_lead",
			"reason":   "scaling",
		},
	}
	result := RecordTeamEvent(entry)
	require.NotNil(t, result.Details)
	assert.Equal(t, "team_lead", result.Details["added_by"])
	assert.Equal(t, "scaling", result.Details["reason"])
	assert.Equal(t, "browser_specialist", result.RoleName)
}

func TestRecordTeamEvent_EmptyFields(t *testing.T) {
	entry := TeamAuditEntry{}
	result := RecordTeamEvent(entry)
	assert.NotEmpty(t, result.EventID, "should still generate EventID")
	assert.False(t, result.Timestamp.IsZero(), "should still set Timestamp")
	assert.Empty(t, result.EventType)
	assert.Empty(t, result.TeamID)
}

func TestRecordTeamEvent_DoesNotMutateOriginal(t *testing.T) {
	entry := TeamAuditEntry{
		EventType: EventTeamCreated,
		TeamID:    "team-123",
	}
	RecordTeamEvent(entry)
	assert.Empty(t, entry.EventID, "original entry should not be mutated")
	assert.True(t, entry.Timestamp.IsZero(), "original timestamp should not be mutated")
}
