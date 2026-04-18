package team

import (
	"fmt"
	"time"
)

// Team audit event types.
const (
	EventTeamCreated     = "team_created"
	EventTeamDissolved   = "team_dissolved"
	EventMemberAdded     = "member_added"
	EventMemberRemoved   = "member_removed"
	EventRoleAssigned    = "role_assigned"
	EventDelegationSent  = "delegation_sent"
	EventHandoffComplete = "handoff_complete"
)

// TeamAuditEntry represents a team-specific audit event.
type TeamAuditEntry struct {
	EventID   string
	EventType string
	TeamID    string
	AgentID   string
	RoleName  string
	Timestamp time.Time
	Details   map[string]string
}

// RecordTeamEvent creates a TeamAuditEntry with a generated event ID and
// current timestamp. It is a convenience helper for callers that only have
// the business fields.
func RecordTeamEvent(entry TeamAuditEntry) TeamAuditEntry {
	if entry.EventID == "" {
		entry.EventID = fmt.Sprintf("te-%d-%s", time.Now().UnixNano(), entry.TeamID)
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	return entry
}
