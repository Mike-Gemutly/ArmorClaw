package secretary

import (
	"context"
	"log/slog"
	"time"
)

//=============================================================================
// Audit Logger
//=============================================================================

// AuditLogger logs Secretary operations for compliance
type AuditLogger struct {
	logger *slog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditLogger{
		logger: logger,
	}
}

// LogOperation logs a Secretary operation with details
func (a *AuditLogger) LogOperation(ctx context.Context, operation string, details map[string]interface{}) error {
	timestamp := time.Now()

	logData := make(map[string]interface{})
	logData["timestamp"] = timestamp.UnixMilli()
	logData["operation"] = operation

	for k, v := range details {
		logData[k] = v
	}

	a.logger.InfoContext(ctx, "secretary_operation", "log_data", logData)
	return nil
}

// LogWorkflowCreation logs workflow creation
func (a *AuditLogger) LogWorkflowCreation(ctx context.Context, workflowID, templateID, createdBy string) error {
	return a.LogOperation(ctx, "workflow_created", map[string]interface{}{
		"workflow_id": workflowID,
		"template_id": templateID,
		"created_by":  createdBy,
	})
}

// LogAgentCreation logs agent creation
func (a *AuditLogger) LogAgentCreation(ctx context.Context, agentID, agentName, createdBy string) error {
	return a.LogOperation(ctx, "agent_created", map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agentName,
		"created_by": createdBy,
	})
}

// LogWorkflowStatusUpdate logs workflow status change
func (a *AuditLogger) LogWorkflowStatusUpdate(ctx context.Context, workflowID, oldStatus, newStatus WorkflowStatus) error {
	return a.LogOperation(ctx, "workflow_status_update", map[string]interface{}{
		"workflow_id": workflowID,
		"old_status":  string(oldStatus),
		"new_status":  string(newStatus),
	})
}

// LogPIIRequest logs PII access request
func (a *AuditLogger) LogPIIRequest(ctx context.Context, requestID, piiFields []string, requestedBy string) error {
	return a.LogOperation(ctx, "pii_request", map[string]interface{}{
		"request_id":   requestID,
		"pii_fields":   piiFields,
		"requested_by": requestedBy,
	})
}

// LogPIIApproval logs PII approval decision
func (a *AuditLogger) LogPIIApproval(ctx context.Context, requestID, approvedFields []string, approvedBy string) error {
	return a.LogOperation(ctx, "pii_approval", map[string]interface{}{
		"request_id":      requestID,
		"approved_fields": approvedFields,
		"approved_by":     approvedBy,
	})
}

// LogWorkflowCancellation logs workflow cancellation
func (a *AuditLogger) LogWorkflowCancellation(ctx context.Context, workflowID, reason string) error {
	return a.LogOperation(ctx, "workflow_cancelled", map[string]interface{}{
		"workflow_id": workflowID,
		"reason":      reason,
	})
}

// LogAgentDeletion logs agent deletion
func (a *AuditLogger) LogAgentDeletion(ctx context.Context, agentID string) error {
	return a.LogOperation(ctx, "agent_deleted", map[string]interface{}{
		"agent_id": agentID,
	})
}
