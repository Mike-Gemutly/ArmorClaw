// Package logger provides tests for security-specific logging
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// setupTestLogger creates a test logger with a buffer for capturing output
func setupTestLogger() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer

	baseLogger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})

	// Redirect to buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	baseLogger.Logger = slog.New(jsonHandler)

	return baseLogger, &buf
}

// parseLogOutput parses JSON log output
func parseLogOutput(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	return logEntry
}

// TestNewSecurityLogger tests creating a security logger
func TestNewSecurityLogger(t *testing.T) {
	baseLogger, _ := New(Config{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: "base",
	})

	secLog := NewSecurityLogger(baseLogger)
	if secLog == nil {
		t.Fatal("NewSecurityLogger() returned nil")
	}

	if secLog.logger == nil {
		t.Error("Security logger has nil base logger")
	}
}

// TestLogAuthAttempt tests logging authentication attempts
func TestLogAuthAttempt(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAuthAttempt(ctx, "openai", "user@example.com")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "auth_attempt" {
		t.Errorf("event_type = %v, want auth_attempt", logEntry["event_type"])
	}
	if logEntry["provider"] != "openai" {
		t.Errorf("provider = %v, want openai", logEntry["provider"])
	}
	if logEntry["user_id"] != "user@example.com" {
		t.Errorf("user_id = %v, want user@example.com", logEntry["user_id"])
	}
	if logEntry["category"] != "security" {
		t.Errorf("category = %v, want security", logEntry["category"])
	}
}

// TestLogAuthSuccess tests logging successful authentication
func TestLogAuthSuccess(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAuthSuccess(ctx, "anthropic", "admin@example.com")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "auth_success" {
		t.Errorf("event_type = %v, want auth_success", logEntry["event_type"])
	}
	if logEntry["provider"] != "anthropic" {
		t.Errorf("provider = %v, want anthropic", logEntry["provider"])
	}
	if logEntry["user_id"] != "admin@example.com" {
		t.Errorf("user_id = %v, want admin@example.com", logEntry["user_id"])
	}
}

// TestLogAuthFailure tests logging failed authentication
func TestLogAuthFailure(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAuthFailure(ctx, "openai", "hacker@evil.com", "invalid_credentials")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "auth_failure" {
		t.Errorf("event_type = %v, want auth_failure", logEntry["event_type"])
	}
	if logEntry["provider"] != "openai" {
		t.Errorf("provider = %v, want openai", logEntry["provider"])
	}
	if logEntry["user_id"] != "hacker@evil.com" {
		t.Errorf("user_id = %v, want hacker@evil.com", logEntry["user_id"])
	}
	if logEntry["reason"] != "invalid_credentials" {
		t.Errorf("reason = %v, want invalid_credentials", logEntry["reason"])
	}
}

// TestLogAuthRejected tests logging rejected authentication
func TestLogAuthRejected(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAuthRejected(ctx, "@untrusted:bad-server.com", "sender_not_in_allowlist")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "auth_rejected" {
		t.Errorf("event_type = %v, want auth_rejected", logEntry["event_type"])
	}
	if logEntry["sender"] != "@untrusted:bad-server.com" {
		t.Errorf("sender = %v, want @untrusted:bad-server.com", logEntry["sender"])
	}
	if logEntry["reason"] != "sender_not_in_allowlist" {
		t.Errorf("reason = %v, want sender_not_in_allowlist", logEntry["reason"])
	}
}

// TestLogContainerStart tests logging container start
func TestLogContainerStart(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogContainerStart(ctx, "session-123", "container-abc", "armorclaw/agent:v1",
		slog.String("key_id", "openai-default"),
		slog.String("socket_path", "/run/armorclaw/test.sock"),
	)

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "container_start" {
		t.Errorf("event_type = %v, want container_start", logEntry["event_type"])
	}
	if logEntry["session_id"] != "session-123" {
		t.Errorf("session_id = %v, want session-123", logEntry["session_id"])
	}
	if logEntry["container_id"] != "container-abc" {
		t.Errorf("container_id = %v, want container-abc", logEntry["container_id"])
	}
	if logEntry["image"] != "armorclaw/agent:v1" {
		t.Errorf("image = %v, want armorclaw/agent:v1", logEntry["image"])
	}
	if logEntry["key_id"] != "openai-default" {
		t.Errorf("key_id = %v, want openai-default", logEntry["key_id"])
	}

	// Verify timestamp
	if logEntry["timestamp"] == nil {
		t.Error("Missing timestamp")
	} else {
		_, err := time.Parse(time.RFC3339, logEntry["timestamp"].(string))
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	}
}

// TestLogContainerStop tests logging container stop
func TestLogContainerStop(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogContainerStop(ctx, "session-456", "container-def", "user_requested")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "container_stop" {
		t.Errorf("event_type = %v, want container_stop", logEntry["event_type"])
	}
	if logEntry["session_id"] != "session-456" {
		t.Errorf("session_id = %v, want session-456", logEntry["session_id"])
	}
	if logEntry["container_id"] != "container-def" {
		t.Errorf("container_id = %v, want container-def", logEntry["container_id"])
	}
	if logEntry["reason"] != "user_requested" {
		t.Errorf("reason = %v, want user_requested", logEntry["reason"])
	}
}

// TestLogContainerError tests logging container errors
func TestLogContainerError(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogContainerError(ctx, "session-789", "container-ghi", "timeout", "operation timed out")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "container_error" {
		t.Errorf("event_type = %v, want container_error", logEntry["event_type"])
	}
	if logEntry["session_id"] != "session-789" {
		t.Errorf("session_id = %v, want session-789", logEntry["session_id"])
	}
	if logEntry["container_id"] != "container-ghi" {
		t.Errorf("container_id = %v, want container-ghi", logEntry["container_id"])
	}
	if logEntry["error_type"] != "timeout" {
		t.Errorf("error_type = %v, want timeout", logEntry["error_type"])
	}
	if logEntry["error_message"] != "operation timed out" {
		t.Errorf("error_message = %v, want 'operation timed out'", logEntry["error_message"])
	}
}

// TestLogSecretAccess tests logging secret access
func TestLogSecretAccess(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogSecretAccess(ctx, "openai-default", "openai")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "secret_access" {
		t.Errorf("event_type = %v, want secret_access", logEntry["event_type"])
	}
	if logEntry["key_id"] != "openai-default" {
		t.Errorf("key_id = %v, want openai-default", logEntry["key_id"])
	}
	if logEntry["key_type"] != "openai" {
		t.Errorf("key_type = %v, want openai", logEntry["key_type"])
	}
}

// TestLogSecretInject tests logging secret injection
func TestLogSecretInject(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogSecretInject(ctx, "session-xyz", "anthropic-prod")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "secret_inject" {
		t.Errorf("event_type = %v, want secret_inject", logEntry["event_type"])
	}
	if logEntry["session_id"] != "session-xyz" {
		t.Errorf("session_id = %v, want session-xyz", logEntry["session_id"])
	}
	if logEntry["key_id"] != "anthropic-prod" {
		t.Errorf("key_id = %v, want anthropic-prod", logEntry["key_id"])
	}
}

// TestLogSecretCleanup tests logging secret cleanup
func TestLogSecretCleanup(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogSecretCleanup(ctx, "session-cleanup", "test-key")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "secret_cleanup" {
		t.Errorf("event_type = %v, want secret_cleanup", logEntry["event_type"])
	}
	if logEntry["session_id"] != "session-cleanup" {
		t.Errorf("session_id = %v, want session-cleanup", logEntry["session_id"])
	}
	if logEntry["key_id"] != "test-key" {
		t.Errorf("key_id = %v, want test-key", logEntry["key_id"])
	}
}

// TestLogAccessDenied tests logging access denied
func TestLogAccessDenied(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAccessDenied(ctx, "admin_endpoint", "untrusted-user", "insufficient_permissions")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "access_denied" {
		t.Errorf("event_type = %v, want access_denied", logEntry["event_type"])
	}
	if logEntry["resource"] != "admin_endpoint" {
		t.Errorf("resource = %v, want admin_endpoint", logEntry["resource"])
	}
	if logEntry["actor"] != "untrusted-user" {
		t.Errorf("actor = %v, want untrusted-user", logEntry["actor"])
	}
	if logEntry["reason"] != "insufficient_permissions" {
		t.Errorf("reason = %v, want insufficient_permissions", logEntry["reason"])
	}
}

// TestLogAccessGranted tests logging access granted
func TestLogAccessGranted(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogAccessGranted(ctx, "user_endpoint", "trusted-user")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "access_granted" {
		t.Errorf("event_type = %v, want access_granted", logEntry["event_type"])
	}
	if logEntry["resource"] != "user_endpoint" {
		t.Errorf("resource = %v, want user_endpoint", logEntry["resource"])
	}
	if logEntry["actor"] != "trusted-user" {
		t.Errorf("actor = %v, want trusted-user", logEntry["actor"])
	}
}

// TestLogPIIDetected tests logging PII detection
func TestLogPIIDetected(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogPIIDetected(ctx, "email", "3")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "pii_detected" {
		t.Errorf("event_type = %v, want pii_detected", logEntry["event_type"])
	}
	if logEntry["pii_type"] != "email" {
		t.Errorf("pii_type = %v, want email", logEntry["pii_type"])
	}
	if logEntry["count"] != "3" {
		t.Errorf("count = %v, want 3", logEntry["count"])
	}
}

// TestLogPIIRedacted tests logging PII redaction
func TestLogPIIRedacted(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogPIIRedacted(ctx, "credit_card", "2", "synthetic")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "pii_redacted" {
		t.Errorf("event_type = %v, want pii_redacted", logEntry["event_type"])
	}
	if logEntry["pii_type"] != "credit_card" {
		t.Errorf("pii_type = %v, want credit_card", logEntry["pii_type"])
	}
	if logEntry["count"] != "2" {
		t.Errorf("count = %v, want 2", logEntry["count"])
	}
	if logEntry["strategy"] != "synthetic" {
		t.Errorf("strategy = %v, want synthetic", logEntry["strategy"])
	}
}

// TestLogBudgetWarning tests logging budget warnings
func TestLogBudgetWarning(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogBudgetWarning(ctx, "daily", 4.50, 5.00)

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "budget_warning" {
		t.Errorf("event_type = %v, want budget_warning", logEntry["event_type"])
	}
	if logEntry["budget_type"] != "daily" {
		t.Errorf("budget_type = %v, want daily", logEntry["budget_type"])
	}

	current, ok := logEntry["current_usd"].(float64)
	if !ok {
		t.Fatal("current_usd is not a number")
	}
	if current != 4.50 {
		t.Errorf("current_usd = %v, want 4.50", current)
	}

	limit, ok := logEntry["limit_usd"].(float64)
	if !ok {
		t.Fatal("limit_usd is not a number")
	}
	if limit != 5.00 {
		t.Errorf("limit_usd = %v, want 5.00", limit)
	}

	percentage, ok := logEntry["percentage"].(float64)
	if !ok {
		t.Fatal("percentage is not a number")
	}
	expectedPercentage := (4.50 / 5.00) * 100
	if percentage != expectedPercentage {
		t.Errorf("percentage = %v, want %v", percentage, expectedPercentage)
	}
}

// TestLogBudgetExceeded tests logging budget exceeded
func TestLogBudgetExceeded(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogBudgetExceeded(ctx, "monthly", 125.00, 100.00, "hard_stop")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "budget_exceeded" {
		t.Errorf("event_type = %v, want budget_exceeded", logEntry["event_type"])
	}
	if logEntry["budget_type"] != "monthly" {
		t.Errorf("budget_type = %v, want monthly", logEntry["budget_type"])
	}
	if logEntry["action_taken"] != "hard_stop" {
		t.Errorf("action_taken = %v, want hard_stop", logEntry["action_taken"])
	}
}

// TestLogHITLRequired tests logging HITL required
func TestLogHITLRequired(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogHITLRequired(ctx, "conf-abc123", "delete_file", "dangerous")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "hitl_required" {
		t.Errorf("event_type = %v, want hitl_required", logEntry["event_type"])
	}
	if logEntry["confirmation_id"] != "conf-abc123" {
		t.Errorf("confirmation_id = %v, want conf-abc123", logEntry["confirmation_id"])
	}
	if logEntry["tool_name"] != "delete_file" {
		t.Errorf("tool_name = %v, want delete_file", logEntry["tool_name"])
	}
	if logEntry["severity"] != "dangerous" {
		t.Errorf("severity = %v, want dangerous", logEntry["severity"])
	}
}

// TestLogHITLApproved tests logging HITL approval
func TestLogHITLApproved(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogHITLApproved(ctx, "conf-xyz789", "send_email", "admin@example.com")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "hitl_approved" {
		t.Errorf("event_type = %v, want hitl_approved", logEntry["event_type"])
	}
	if logEntry["confirmation_id"] != "conf-xyz789" {
		t.Errorf("confirmation_id = %v, want conf-xyz789", logEntry["confirmation_id"])
	}
	if logEntry["tool_name"] != "send_email" {
		t.Errorf("tool_name = %v, want send_email", logEntry["tool_name"])
	}
	if logEntry["approver"] != "admin@example.com" {
		t.Errorf("approver = %v, want admin@example.com", logEntry["approver"])
	}
}

// TestLogHITLRejected tests logging HITL rejection
func TestLogHITLRejected(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogHITLRejected(ctx, "conf-def456", "make_purchase", "user@example.com", "too_expensive")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "hitl_rejected" {
		t.Errorf("event_type = %v, want hitl_rejected", logEntry["event_type"])
	}
	if logEntry["confirmation_id"] != "conf-def456" {
		t.Errorf("confirmation_id = %v, want conf-def456", logEntry["confirmation_id"])
	}
	if logEntry["tool_name"] != "make_purchase" {
		t.Errorf("tool_name = %v, want make_purchase", logEntry["tool_name"])
	}
	if logEntry["rejecter"] != "user@example.com" {
		t.Errorf("rejecter = %v, want user@example.com", logEntry["rejecter"])
	}
	if logEntry["reason"] != "too_expensive" {
		t.Errorf("reason = %v, want too_expensive", logEntry["reason"])
	}
}

// TestLogHITLTimeout tests logging HITL timeout
func TestLogHITLTimeout(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)

	ctx := context.Background()
	secLog.LogHITLTimeout(ctx, "conf-timeout", "execute_command")

	logEntry := parseLogOutput(t, buf)

	if logEntry["event_type"] != "hitl_timeout" {
		t.Errorf("event_type = %v, want hitl_timeout", logEntry["event_type"])
	}
	if logEntry["confirmation_id"] != "conf-timeout" {
		t.Errorf("confirmation_id = %v, want conf-timeout", logEntry["confirmation_id"])
	}
	if logEntry["tool_name"] != "execute_command" {
		t.Errorf("tool_name = %v, want execute_command", logEntry["tool_name"])
	}
}

// TestAllSecurityEventTypes tests that all security event types are defined
func TestAllSecurityEventTypes(t *testing.T) {
	expectedTypes := []SecurityEventType{
		AuthAttempt, AuthSuccess, AuthFailure, AuthRejected,
		ContainerStart, ContainerStop, ContainerError, ContainerTimeout,
		SecretAccess, SecretInject, SecretCleanup,
		AccessDenied, AccessGranted,
		PIIDetected, PIIRedacted,
		BudgetWarning, BudgetExceeded,
		HITLRequired, HITLApproved, HITLRejected, HITLTimeout,
	}

	// Verify all types have string values
	for _, eventType := range expectedTypes {
		if string(eventType) == "" {
			t.Errorf("Security event type %v has empty string value", eventType)
		}
	}
}

// TestSecurityEventConsistency tests that all security events have consistent fields
func TestSecurityEventConsistency(t *testing.T) {
	logger, buf := setupTestLogger()
	secLog := NewSecurityLogger(logger)
	ctx := context.Background()

	// Test a sample of security events
	tests := []struct {
		name     string
		logFunc  func()
		required []string
	}{
		{
			name: "auth_success",
			logFunc: func() {
				secLog.LogAuthSuccess(ctx, "test", "user@test.com")
			},
			required: []string{"event_type", "provider", "user_id", "category"},
		},
		{
			name: "container_start",
			logFunc: func() {
				secLog.LogContainerStart(ctx, "sess1", "cont1", "image:latest")
			},
			required: []string{"event_type", "session_id", "container_id", "image", "timestamp"},
		},
		{
			name: "secret_access",
			logFunc: func() {
				secLog.LogSecretAccess(ctx, "key1", "openai")
			},
			required: []string{"event_type", "key_id", "key_type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			logEntry := parseLogOutput(t, buf)

			// Verify all required fields are present
			for _, field := range tt.required {
				if logEntry[field] == nil {
					t.Errorf("Missing required field: %s", field)
				}
			}

			// Verify category is always "security"
			if logEntry["category"] != "security" {
				t.Errorf("category = %v, want 'security'", logEntry["category"])
			}
		})
	}
}

// TestSecurityLoggingPerformance benchmarks security logging
func BenchmarkSecurityLogging(b *testing.B) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "bench",
	})
	secLog := NewSecurityLogger(logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		secLog.LogContainerStart(ctx, "session", "container", "image:latest")
	}
}

// TestConcurrentSecurityLogging tests concurrent security logging
func TestConcurrentSecurityLogging(t *testing.T) {
	logger, _ := New(Config{
		Level:     "info",
		Format:    "json",
		Output:    "stdout",
		Component: "test",
	})
	secLog := NewSecurityLogger(logger)
	ctx := context.Background()

	// Launch multiple goroutines logging concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				secLog.LogAuthAttempt(ctx, "provider", "user@example.com")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without race conditions or panics, test passed
}
