// Package logger provides security-specific logging helpers for ArmorClaw
package logger

import (
	"context"
	"log/slog"
	"time"
)

// SecurityEventType defines types of security events
type SecurityEventType string

const (
	// Authentication events
	AuthAttempt         SecurityEventType = "auth_attempt"
	AuthSuccess         SecurityEventType = "auth_success"
	AuthFailure         SecurityEventType = "auth_failure"
	AuthRejected        SecurityEventType = "auth_rejected"

	// Container lifecycle events
	ContainerStart      SecurityEventType = "container_start"
	ContainerStop       SecurityEventType = "container_stop"
	ContainerError      SecurityEventType = "container_error"
	ContainerTimeout    SecurityEventType = "container_timeout"

	// Secret access events
	SecretAccess        SecurityEventType = "secret_access"
	SecretInject        SecurityEventType = "secret_inject"
	SecretCleanup       SecurityEventType = "secret_cleanup"

	// Authorization events
	AccessDenied        SecurityEventType = "access_denied"
	AccessGranted       SecurityEventType = "access_granted"

	// PII events
	PIIDetected         SecurityEventType = "pii_detected"
	PIIRedacted         SecurityEventType = "pii_redacted"

	// Budget events
	BudgetWarning       SecurityEventType = "budget_warning"
	BudgetExceeded      SecurityEventType = "budget_exceeded"
	BudgetEnforcement   SecurityEventType = "budget_enforcement"

	// HITL events
	HITLRequired        SecurityEventType = "hitl_required"
	HITLApproved        SecurityEventType = "hitl_approved"
	HITLRejected        SecurityEventType = "hitl_rejected"
	HITLTimeout         SecurityEventType = "hitl_timeout"
)

// SecurityLogger provides security-specific logging methods
type SecurityLogger struct {
	logger *Logger
}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger(baseLogger *Logger) *SecurityLogger {
	return &SecurityLogger{
		logger: baseLogger.WithComponent("security"),
	}
}

// LogAuthAttempt logs an authentication attempt
func (sl *SecurityLogger) LogAuthAttempt(ctx context.Context, provider, userID string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("provider", provider),
		slog.String("user_id", userID),
	}
	sl.logger.SecurityEvent(ctx, string(AuthAttempt), append(baseAttrs, attrs...)...)
}

// LogAuthSuccess logs a successful authentication
func (sl *SecurityLogger) LogAuthSuccess(ctx context.Context, provider, userID string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("provider", provider),
		slog.String("user_id", userID),
	}
	sl.logger.SecurityEvent(ctx, string(AuthSuccess), append(baseAttrs, attrs...)...)
}

// LogAuthFailure logs a failed authentication
func (sl *SecurityLogger) LogAuthFailure(ctx context.Context, provider, userID, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("provider", provider),
		slog.String("user_id", userID),
		slog.String("reason", reason),
	}
	sl.logger.SecurityEvent(ctx, string(AuthFailure), append(baseAttrs, attrs...)...)
}

// LogAuthRejected logs a rejected authentication (untrusted sender)
func (sl *SecurityLogger) LogAuthRejected(ctx context.Context, sender, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("sender", sender),
		slog.String("reason", reason),
	}
	sl.logger.SecurityEvent(ctx, string(AuthRejected), append(baseAttrs, attrs...)...)
}

// LogContainerStart logs a container start event
func (sl *SecurityLogger) LogContainerStart(ctx context.Context, sessionID, containerID, image string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("session_id", sessionID),
		slog.String("container_id", containerID),
		slog.String("image", image),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
	}
	sl.logger.SecurityEvent(ctx, string(ContainerStart), append(baseAttrs, attrs...)...)
}

// LogContainerStop logs a container stop event
func (sl *SecurityLogger) LogContainerStop(ctx context.Context, sessionID, containerID, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("session_id", sessionID),
		slog.String("container_id", containerID),
		slog.String("reason", reason),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
	}
	sl.logger.SecurityEvent(ctx, string(ContainerStop), append(baseAttrs, attrs...)...)
}

// LogContainerError logs a container error event
func (sl *SecurityLogger) LogContainerError(ctx context.Context, sessionID, containerID, errorType, errorMessage string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("session_id", sessionID),
		slog.String("container_id", containerID),
		slog.String("error_type", errorType),
		slog.String("error_message", errorMessage),
	}
	sl.logger.SecurityEvent(ctx, string(ContainerError), append(baseAttrs, attrs...)...)
}

// LogSecretAccess logs a secret access event
func (sl *SecurityLogger) LogSecretAccess(ctx context.Context, keyID, keyType string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("key_id", keyID),
		slog.String("key_type", keyType),
	}
	sl.logger.SecurityEvent(ctx, string(SecretAccess), append(baseAttrs, attrs...)...)
}

// LogSecretInject logs a secret injection event
func (sl *SecurityLogger) LogSecretInject(ctx context.Context, sessionID, keyID string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("session_id", sessionID),
		slog.String("key_id", keyID),
	}
	sl.logger.SecurityEvent(ctx, string(SecretInject), append(baseAttrs, attrs...)...)
}

// LogSecretCleanup logs a secret cleanup event
func (sl *SecurityLogger) LogSecretCleanup(ctx context.Context, sessionID, keyID string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("session_id", sessionID),
		slog.String("key_id", keyID),
	}
	sl.logger.SecurityEvent(ctx, string(SecretCleanup), append(baseAttrs, attrs...)...)
}

// LogAccessDenied logs an access denied event
func (sl *SecurityLogger) LogAccessDenied(ctx context.Context, resource, actor, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("resource", resource),
		slog.String("actor", actor),
		slog.String("reason", reason),
	}
	sl.logger.SecurityEvent(ctx, string(AccessDenied), append(baseAttrs, attrs...)...)
}

// LogAccessGranted logs an access granted event
func (sl *SecurityLogger) LogAccessGranted(ctx context.Context, resource, actor string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("resource", resource),
		slog.String("actor", actor),
	}
	sl.logger.SecurityEvent(ctx, string(AccessGranted), append(baseAttrs, attrs...)...)
}

// LogPIIDetected logs PII detection events
func (sl *SecurityLogger) LogPIIDetected(ctx context.Context, piiType, count string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("pii_type", piiType),
		slog.String("count", count),
	}
	sl.logger.SecurityEvent(ctx, string(PIIDetected), append(baseAttrs, attrs...)...)
}

// LogPIIRedacted logs PII redaction events
func (sl *SecurityLogger) LogPIIRedacted(ctx context.Context, piiType, count, strategy string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("pii_type", piiType),
		slog.String("count", count),
		slog.String("strategy", strategy),
	}
	sl.logger.SecurityEvent(ctx, string(PIIRedacted), append(baseAttrs, attrs...)...)
}

// LogBudgetWarning logs a budget warning event
func (sl *SecurityLogger) LogBudgetWarning(ctx context.Context, budgetType string, current, limit float64, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("budget_type", budgetType),
		slog.Float64("current_usd", current),
		slog.Float64("limit_usd", limit),
		slog.Float64("percentage", (current/limit)*100),
	}
	sl.logger.SecurityEvent(ctx, string(BudgetWarning), append(baseAttrs, attrs...)...)
}

// LogBudgetExceeded logs a budget exceeded event
func (sl *SecurityLogger) LogBudgetExceeded(ctx context.Context, budgetType string, current, limit float64, action string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("budget_type", budgetType),
		slog.Float64("current_usd", current),
		slog.Float64("limit_usd", limit),
		slog.String("action_taken", action),
	}
	sl.logger.SecurityEvent(ctx, string(BudgetExceeded), append(baseAttrs, attrs...)...)
}

// LogHITLRequired logs a HITL required event
func (sl *SecurityLogger) LogHITLRequired(ctx context.Context, confirmationID, toolName, severity string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("confirmation_id", confirmationID),
		slog.String("tool_name", toolName),
		slog.String("severity", severity),
	}
	sl.logger.SecurityEvent(ctx, string(HITLRequired), append(baseAttrs, attrs...)...)
}

// LogHITLApproved logs a HITL approved event
func (sl *SecurityLogger) LogHITLApproved(ctx context.Context, confirmationID, toolName, approver string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("confirmation_id", confirmationID),
		slog.String("tool_name", toolName),
		slog.String("approver", approver),
	}
	sl.logger.SecurityEvent(ctx, string(HITLApproved), append(baseAttrs, attrs...)...)
}

// LogHITLRejected logs a HITL rejected event
func (sl *SecurityLogger) LogHITLRejected(ctx context.Context, confirmationID, toolName, rejecter, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("confirmation_id", confirmationID),
		slog.String("tool_name", toolName),
		slog.String("rejecter", rejecter),
		slog.String("reason", reason),
	}
	sl.logger.SecurityEvent(ctx, string(HITLRejected), append(baseAttrs, attrs...)...)
}

// LogHITLTimeout logs a HITL timeout event
func (sl *SecurityLogger) LogHITLTimeout(ctx context.Context, confirmationID, toolName string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("confirmation_id", confirmationID),
		slog.String("tool_name", toolName),
	}
	sl.logger.SecurityEvent(ctx, string(HITLTimeout), append(baseAttrs, attrs...)...)
}

// PII-specific security event types
const (
	PIIAccessRequest  SecurityEventType = "pii_access_request"
	PIIAccessGranted  SecurityEventType = "pii_access_granted"
	PIIAccessRejected SecurityEventType = "pii_access_rejected"
	PIIAccessExpired  SecurityEventType = "pii_access_expired"
	PIIInjected       SecurityEventType = "pii_injected"
)

// LogPIIAccessRequest logs when a skill requests access to PII fields
// CRITICAL: Never log actual PII values - only field names
func (sl *SecurityLogger) LogPIIAccessRequest(ctx context.Context, requestID, skillName, profileID string, requestedFields []string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("skill_name", skillName),
		slog.String("profile_id", profileID),
		// Only log field names, never values
		slog.Any("requested_fields", requestedFields),
	}
	sl.logger.SecurityEvent(ctx, string(PIIAccessRequest), append(baseAttrs, attrs...)...)
}

// LogPIIAccessGranted logs when PII access is granted by user
// CRITICAL: Never log actual PII values - only field names and approver
func (sl *SecurityLogger) LogPIIAccessGranted(ctx context.Context, requestID, skillName, userID string, approvedFields []string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("skill_name", skillName),
		slog.String("approver", userID),
		// Only log field names, never values
		slog.Any("approved_fields", approvedFields),
	}
	sl.logger.SecurityEvent(ctx, string(PIIAccessGranted), append(baseAttrs, attrs...)...)
}

// LogPIIAccessRejected logs when PII access is rejected by user
func (sl *SecurityLogger) LogPIIAccessRejected(ctx context.Context, requestID, skillName, userID, reason string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("skill_name", skillName),
		slog.String("rejecter", userID),
		slog.String("reason", reason),
	}
	sl.logger.SecurityEvent(ctx, string(PIIAccessRejected), append(baseAttrs, attrs...)...)
}

// LogPIIAccessExpired logs when a PII access request expires
func (sl *SecurityLogger) LogPIIAccessExpired(ctx context.Context, requestID, skillName string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("skill_name", skillName),
	}
	sl.logger.SecurityEvent(ctx, string(PIIAccessExpired), append(baseAttrs, attrs...)...)
}

// LogPIIInjected logs when PII is injected into a container
// CRITICAL: Never log actual PII values - only field names and container ID
func (sl *SecurityLogger) LogPIIInjected(ctx context.Context, containerID, skillName string, fieldsInjected []string, method string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("container_id", containerID),
		slog.String("skill_name", skillName),
		// Only log field names, never values
		slog.Any("fields_injected", fieldsInjected),
		slog.String("injection_method", method),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
	}
	sl.logger.SecurityEvent(ctx, string(PIIInjected), append(baseAttrs, attrs...)...)
}

// PCI-DSS Compliance events
const (
	PCIViolationDetected     SecurityEventType = "pci_violation_detected"
	PCIViolationAcknowledged SecurityEventType = "pci_violation_acknowledged"
)

// LogPCIViolationAcknowledged logs when a user acknowledges PCI-DSS violations
// This is critical for compliance auditing - tracks when users store prohibited data
func (sl *SecurityLogger) LogPCIViolationAcknowledged(
	ctx context.Context,
	profileName string,
	profileType string,
	pciWarnings []map[string]string,
	attrs ...slog.Attr,
) {
	// Extract warning levels for logging (never log actual values)
	var violationFields []string
	var prohibitedFields []string
	var cautionFields []string

	for _, w := range pciWarnings {
		switch w["level"] {
		case "prohibited":
			prohibitedFields = append(prohibitedFields, w["field"])
		case "violation":
			violationFields = append(violationFields, w["field"])
		case "caution":
			cautionFields = append(cautionFields, w["field"])
		}
	}

	baseAttrs := []slog.Attr{
		slog.String("profile_name", profileName),
		slog.String("profile_type", profileType),
		slog.Any("violation_fields", violationFields),
		slog.Any("prohibited_fields", prohibitedFields),
		slog.Any("caution_fields", cautionFields),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("compliance_note", "User acknowledged PCI-DSS risks"),
	}
	sl.logger.SecurityEvent(ctx, string(PCIViolationAcknowledged), append(baseAttrs, attrs...)...)
}

// LogSecurityEvent logs a generic security event with custom event type
// This provides flexibility for events that don't fit the predefined categories
func (sl *SecurityLogger) LogSecurityEvent(eventType string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("event_type", eventType),
	}
	sl.logger.SecurityEvent(context.Background(), eventType, append(baseAttrs, attrs...)...)
}
