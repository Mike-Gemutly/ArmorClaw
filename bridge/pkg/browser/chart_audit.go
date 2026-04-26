package browser

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Audit event types for chart lifecycle.
const (
	AuditEventCreated  = "chart_created"
	AuditEventUpdated  = "chart_updated"
	AuditEventReplayed = "chart_replayed"
	AuditEventRejected = "chart_rejected"
	AuditEventDeleted  = "chart_deleted"
)

// ChartAuditor records audit entries for chart lifecycle events.
type ChartAuditor struct {
	db *sql.DB
}

// NewChartAuditor creates a new ChartAuditor.
// The caller must ensure the chart_audit table already exists (via initSchema).
func NewChartAuditor(db *sql.DB) *ChartAuditor {
	return &ChartAuditor{db: db}
}

// AuditCreated logs the creation of a new chart.
func (a *ChartAuditor) AuditCreated(ctx context.Context, chartID, domain string, actionCount int, sessionID string) error {
	return a.insert(ctx, AuditEventCreated, chartID, map[string]interface{}{
		"domain":       domain,
		"action_count": actionCount,
		"session_id":   sessionID,
	})
}

// AuditUpdated logs an update to a chart's confidence after outcome recording.
func (a *ChartAuditor) AuditUpdated(ctx context.Context, chartID string, confidence float64, outcome bool) error {
	return a.insert(ctx, AuditEventUpdated, chartID, map[string]interface{}{
		"confidence": confidence,
		"outcome":    outcome,
	})
}

// AuditReplayed logs the replay execution of a chart.
func (a *ChartAuditor) AuditReplayed(ctx context.Context, chartID, jobID string, actionsExecuted, approvalsNeeded, approvalsGranted, approvalsDenied int) error {
	return a.insert(ctx, AuditEventReplayed, chartID, map[string]interface{}{
		"job_id":            jobID,
		"actions_executed":  actionsExecuted,
		"approvals_needed":  approvalsNeeded,
		"approvals_granted": approvalsGranted,
		"approvals_denied":  approvalsDenied,
	})
}

// AuditRejected logs when a chart is rejected (e.g. low confidence threshold).
func (a *ChartAuditor) AuditRejected(ctx context.Context, chartID, reason string) error {
	return a.insert(ctx, AuditEventRejected, chartID, map[string]interface{}{
		"reason": reason,
	})
}

// AuditDeleted logs the deletion of a chart.
func (a *ChartAuditor) AuditDeleted(ctx context.Context, chartID, domain string) error {
	return a.insert(ctx, AuditEventDeleted, chartID, map[string]interface{}{
		"domain": domain,
	})
}

func (a *ChartAuditor) insert(ctx context.Context, eventType, chartID string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal audit details: %w", err)
	}

	_, err = a.db.ExecContext(ctx, `
		INSERT INTO chart_audit (event_type, chart_id, details, created_at)
		VALUES (?, ?, ?, ?)`,
		eventType, chartID, string(detailsJSON), time.Now().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}
