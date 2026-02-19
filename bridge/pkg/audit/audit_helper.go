// Package audit provides audit logging helpers for critical operations
package audit

import (
	"context"
	"sync"
)

// CriticalOperationLogger provides audit logging for critical operations
type CriticalOperationLogger struct {
	auditLog *TamperEvidentLog
	mu       sync.RWMutex
}

// NewCriticalOperationLogger creates a new critical operation logger
func NewCriticalOperationLogger(auditLog *TamperEvidentLog) *CriticalOperationLogger {
	return &CriticalOperationLogger{
		auditLog: auditLog,
	}
}

// SetAuditLog updates the audit log
func (l *CriticalOperationLogger) SetAuditLog(auditLog *TamperEvidentLog) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.auditLog = auditLog
}

// LogContainerStart logs a container start operation
func (l *CriticalOperationLogger) LogContainerStart(ctx context.Context, containerID, image, keyID, sessionID string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type:      "system",
		ID:        "bridge",
		IPAddress: "",
	}
	resource := Resource{
		Type: "container",
		ID:   containerID,
		Name: image,
	}
	details := map[string]interface{}{
		"image":      image,
		"key_id":     keyID,
		"session_id": sessionID,
		"action":     "start",
	}
	compliance := ComplianceFlags{
		Category:      "container_lifecycle",
		Severity:      "medium",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("container_start", actor, "start", resource, details, compliance)
	return err
}

// LogContainerStop logs a container stop operation
func (l *CriticalOperationLogger) LogContainerStop(ctx context.Context, containerID, reason string, exitCode int) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "bridge",
	}
	resource := Resource{
		Type: "container",
		ID:   containerID,
	}
	details := map[string]interface{}{
		"reason":    reason,
		"exit_code": exitCode,
		"action":    "stop",
	}
	compliance := ComplianceFlags{
		Category:      "container_lifecycle",
		Severity:      "medium",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("container_stop", actor, "stop", resource, details, compliance)
	return err
}

// LogContainerError logs a container error
func (l *CriticalOperationLogger) LogContainerError(ctx context.Context, containerID, errorMsg string, errorCode string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "bridge",
	}
	resource := Resource{
		Type: "container",
		ID:   containerID,
	}
	details := map[string]interface{}{
		"error":      errorMsg,
		"error_code": errorCode,
		"action":     "error",
	}
	compliance := ComplianceFlags{
		Category:      "container_lifecycle",
		Severity:      "high",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("container_error", actor, "error", resource, details, compliance)
	return err
}

// LogKeyAccess logs a key access operation
func (l *CriticalOperationLogger) LogKeyAccess(ctx context.Context, keyID, userID, operation string, success bool) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "user",
		ID:   userID,
	}
	resource := Resource{
		Type: "api_key",
		ID:   keyID,
	}
	severity := "medium"
	if !success {
		severity = "high"
	}
	details := map[string]interface{}{
		"operation": operation,
		"success":   success,
	}
	compliance := ComplianceFlags{
		Category:      "key_access",
		Severity:      severity,
		AuditRequired: true,
	}

	eventType := "key_access"
	if !success {
		eventType = "key_access_denied"
	}

	_, err := auditLog.LogEntry(eventType, actor, operation, resource, details, compliance)
	return err
}

// LogKeyCreated logs a key creation operation
func (l *CriticalOperationLogger) LogKeyCreated(ctx context.Context, keyID, provider, userID string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "user",
		ID:   userID,
	}
	resource := Resource{
		Type: "api_key",
		ID:   keyID,
	}
	details := map[string]interface{}{
		"provider": provider,
		"action":   "create",
	}
	compliance := ComplianceFlags{
		Category:      "key_management",
		Severity:      "high",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("key_created", actor, "create", resource, details, compliance)
	return err
}

// LogKeyDeleted logs a key deletion operation
func (l *CriticalOperationLogger) LogKeyDeleted(ctx context.Context, keyID, userID string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "user",
		ID:   userID,
	}
	resource := Resource{
		Type: "api_key",
		ID:   keyID,
	}
	details := map[string]interface{}{
		"action": "delete",
	}
	compliance := ComplianceFlags{
		Category:      "key_management",
		Severity:      "high",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("key_deleted", actor, "delete", resource, details, compliance)
	return err
}

// LogSecretInjection logs a secret injection operation
func (l *CriticalOperationLogger) LogSecretInjection(ctx context.Context, containerID, keyID string, success bool) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "bridge",
	}
	resource := Resource{
		Type: "container",
		ID:   containerID,
	}
	severity := "low"
	if !success {
		severity = "high"
	}
	details := map[string]interface{}{
		"key_id":  keyID,
		"success": success,
		"action":  "inject",
	}
	compliance := ComplianceFlags{
		Category:      "secret_management",
		Severity:      severity,
		AuditRequired: true,
		PHIInvolved:   false, // Secrets are not PHI
	}

	eventType := "secret_injected"
	if !success {
		eventType = "secret_injection_failed"
	}

	_, err := auditLog.LogEntry(eventType, actor, "inject", resource, details, compliance)
	return err
}

// LogSecretCleanup logs a secret cleanup operation
func (l *CriticalOperationLogger) LogSecretCleanup(ctx context.Context, containerID string, success bool) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "bridge",
	}
	resource := Resource{
		Type: "container",
		ID:   containerID,
	}
	details := map[string]interface{}{
		"success": success,
		"action":  "cleanup",
	}
	compliance := ComplianceFlags{
		Category:      "secret_management",
		Severity:      "low",
		AuditRequired: false,
	}

	_, err := auditLog.LogEntry("secret_cleaned", actor, "cleanup", resource, details, compliance)
	return err
}

// LogConfigurationChange logs a configuration change
func (l *CriticalOperationLogger) LogConfigurationChange(ctx context.Context, userID, section, key string, oldValue, newValue interface{}) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "user",
		ID:   userID,
	}
	resource := Resource{
		Type: "configuration",
		ID:   section,
	}
	details := map[string]interface{}{
		"key":       key,
		"old_value": oldValue,
		"new_value": newValue,
		"action":    "change",
	}
	compliance := ComplianceFlags{
		Category:      "configuration",
		Severity:      "medium",
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry("config_change", actor, "change", resource, details, compliance)
	return err
}

// LogAuthenticationEvent logs an authentication event
func (l *CriticalOperationLogger) LogAuthenticationEvent(ctx context.Context, userID, method string, success bool, ipAddress string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type:      "user",
		ID:        userID,
		IPAddress: ipAddress,
	}
	resource := Resource{
		Type: "session",
		ID:   userID,
	}
	severity := "low"
	if !success {
		severity = "high"
	}
	details := map[string]interface{}{
		"method":  method,
		"success": success,
	}
	compliance := ComplianceFlags{
		Category:      "authentication",
		Severity:      severity,
		AuditRequired: true,
	}

	eventType := "auth_success"
	if !success {
		eventType = "auth_failure"
	}

	_, err := auditLog.LogEntry(eventType, actor, "authenticate", resource, details, compliance)
	return err
}

// LogSecurityEvent logs a security-related event
func (l *CriticalOperationLogger) LogSecurityEvent(ctx context.Context, eventType, severity string, details map[string]interface{}) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "bridge",
	}
	resource := Resource{
		Type: "security",
		ID:   eventType,
	}
	compliance := ComplianceFlags{
		Category:      "security",
		Severity:      severity,
		AuditRequired: true,
	}

	_, err := auditLog.LogEntry(eventType, actor, "security_event", resource, details, compliance)
	return err
}

// LogBudgetEvent logs a budget-related event
func (l *CriticalOperationLogger) LogBudgetEvent(ctx context.Context, sessionID string, tokensUsed, tokensLimit int, exceeded bool) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "system",
		ID:   "budget_tracker",
	}
	resource := Resource{
		Type: "session",
		ID:   sessionID,
	}
	severity := "low"
	if exceeded {
		severity = "high"
	}
	details := map[string]interface{}{
		"tokens_used":  tokensUsed,
		"tokens_limit": tokensLimit,
		"exceeded":     exceeded,
	}
	compliance := ComplianceFlags{
		Category:      "budget",
		Severity:      severity,
		AuditRequired: exceeded,
	}

	eventType := "budget_warning"
	if exceeded {
		eventType = "budget_exceeded"
	}

	_, err := auditLog.LogEntry(eventType, actor, "budget_check", resource, details, compliance)
	return err
}

// LogPHIAccess logs access to PHI (Protected Health Information)
func (l *CriticalOperationLogger) LogPHIAccess(ctx context.Context, userID, resourceType, resourceID, action string) error {
	l.mu.RLock()
	auditLog := l.auditLog
	l.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := Actor{
		Type: "user",
		ID:   userID,
	}
	resource := Resource{
		Type: resourceType,
		ID:   resourceID,
	}
	details := map[string]interface{}{
		"action": action,
	}
	compliance := ComplianceFlags{
		Category:      "phi_access",
		Severity:      "high",
		AuditRequired: true,
		PHIInvolved:   true,
	}

	_, err := auditLog.LogEntry("phi_access", actor, action, resource, details, compliance)
	return err
}

// Global audit logger instance
var globalAuditLogger *CriticalOperationLogger
var globalAuditMu sync.RWMutex

// SetGlobalAuditLogger sets the global audit logger
func SetGlobalAuditLogger(logger *CriticalOperationLogger) {
	globalAuditMu.Lock()
	defer globalAuditMu.Unlock()
	globalAuditLogger = logger
}

// GetGlobalAuditLogger gets the global audit logger
func GetGlobalAuditLogger() *CriticalOperationLogger {
	globalAuditMu.RLock()
	defer globalAuditMu.RUnlock()
	return globalAuditLogger
}
