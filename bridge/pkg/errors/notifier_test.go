package errors

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Mock Matrix sender for testing
type mockMatrixSender struct {
	lastRoomID  string
	lastMessage string
	lastType    string
	err         error
	callCount   int
}

func (m *mockMatrixSender) SendMessage(ctx context.Context, roomID, message, msgType string) (string, error) {
	m.callCount++
	m.lastRoomID = roomID
	m.lastMessage = message
	m.lastType = msgType
	if m.err != nil {
		return "", m.err
	}
	return "event_id_123", nil
}

func TestErrorNotifier_Notify(t *testing.T) {
	mockSender := &mockMatrixSender{}

	registry := NewSamplingRegistry(DefaultSamplingConfig())
	resolver := NewAdminResolver(AdminConfig{
		SetupUserMXID: "@admin:example.com",
	})

	notifier := NewErrorNotifier(NotifierConfig{
		Registry:     registry,
		Resolver:     resolver,
		MatrixSender: mockSender,
		Enabled:      true,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "container start failed",
		Function:  "StartContainer",
		File:      "docker/client.go",
		Line:      142,
		TraceID:   "tr_test",
		Timestamp: time.Date(2026, 2, 15, 18, 32, 5, 0, time.UTC),
	}

	notifyErr := notifier.Notify(context.Background(), err)
	if notifyErr != nil {
		t.Fatalf("Notify() error = %v", notifyErr)
	}

	if mockSender.callCount != 1 {
		t.Errorf("SendMessage called %d times, want 1", mockSender.callCount)
	}

	if mockSender.lastRoomID != "@admin:example.com" {
		t.Errorf("SendMessage roomID = %q, want @admin:example.com", mockSender.lastRoomID)
	}

	if mockSender.lastType != "m.notice" {
		t.Errorf("SendMessage type = %q, want m.notice", mockSender.lastType)
	}
}

func TestErrorNotifier_Notify_Disabled(t *testing.T) {
	mockSender := &mockMatrixSender{}

	notifier := NewErrorNotifier(NotifierConfig{
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      false,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	}

	notifyErr := notifier.Notify(context.Background(), err)
	if notifyErr != nil {
		t.Fatalf("Notify() error = %v", notifyErr)
	}

	if mockSender.callCount != 0 {
		t.Errorf("SendMessage should not be called when disabled")
	}
}

func TestErrorNotifier_Notify_RateLimited(t *testing.T) {
	mockSender := &mockMatrixSender{}

	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Hour,
	})

	notifier := NewErrorNotifier(NotifierConfig{
		Registry:     registry,
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      true,
	})

	// First notification
	err1 := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	}
	notifier.Notify(context.Background(), err1)

	// Second notification (should be rate limited)
	err2 := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_2",
		Timestamp: time.Now(),
	}
	notifier.Notify(context.Background(), err2)

	if mockSender.callCount != 1 {
		t.Errorf("SendMessage called %d times, want 1 (rate limited)", mockSender.callCount)
	}
}

func TestErrorNotifier_Notify_CriticalAlways(t *testing.T) {
	mockSender := &mockMatrixSender{}

	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Hour,
	})

	notifier := NewErrorNotifier(NotifierConfig{
		Registry:     registry,
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      true,
	})

	// Multiple critical notifications - all should go through
	for i := 0; i < 3; i++ {
		err := &TracedError{
			Code:      "SYS-001",
			Category:  "system",
			Severity:  SeverityCritical,
			Message:   "critical error",
			TraceID:   "tr_critical",
			Timestamp: time.Now(),
		}
		notifier.Notify(context.Background(), err)
	}

	if mockSender.callCount != 3 {
		t.Errorf("SendMessage called %d times, want 3 (critical always)", mockSender.callCount)
	}
}

func TestErrorNotifier_FormatMessage(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{Enabled: true})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "container start failed",
		Function:  "StartContainer",
		File:      "docker/client.go",
		Line:      142,
		TraceID:   "tr_test",
		Timestamp: time.Date(2026, 2, 15, 18, 32, 5, 0, time.UTC),
	}

	admin := &AdminTarget{
		MXID:   "@admin:example.com",
		Source: "setup",
	}

	message := notifier.formatMessage(err, admin)

	// Check key components
	if !strings.Contains(message, "❌ ERROR: CTX-001") {
		t.Error("Message should contain severity header")
	}
	if !strings.Contains(message, "container start failed") {
		t.Error("Message should contain error message")
	}
	if !strings.Contains(message, "StartContainer") {
		t.Error("Message should contain function name")
	}
	if !strings.Contains(message, "docker/client.go") {
		t.Error("Message should contain file name")
	}
	if !strings.Contains(message, "tr_test") {
		t.Error("Message should contain trace ID")
	}
	if !strings.Contains(message, "```json") {
		t.Error("Message should contain JSON code block")
	}
	if !strings.Contains(message, "📋 Copy the JSON block") {
		t.Error("Message should contain LLM instruction")
	}
	if !strings.Contains(message, "@admin:example.com") {
		t.Error("Message should contain admin MXID")
	}
}

func TestErrorNotifier_FormatHeader(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{Enabled: true})

	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "🔴 CRITICAL: CTX-001"},
		{SeverityError, "❌ ERROR: CTX-001"},
		{SeverityWarning, "⚠️ WARNING: CTX-001"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			err := &TracedError{
				Code:     "CTX-001",
				Severity: tt.severity,
			}
			header := notifier.formatHeader(err)
			if header != tt.expected {
				t.Errorf("formatHeader() = %q, want %q", header, tt.expected)
			}
		})
	}
}

func TestErrorNotifier_FormatMetadata(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{Enabled: true})

	err := &TracedError{
		Function:    "StartContainer",
		File:        "docker/client.go",
		Line:        142,
		TraceID:     "tr_test",
		Timestamp:   time.Date(2026, 2, 15, 18, 32, 5, 0, time.UTC),
		RepeatCount: 5,
	}

	admin := &AdminTarget{
		MXID:   "@admin:example.com",
		Source: "config",
	}

	metadata := notifier.formatMetadata(err, admin)

	if !strings.Contains(metadata, "StartContainer") {
		t.Error("Metadata should contain function")
	}
	if !strings.Contains(metadata, "tr_test") {
		t.Error("Metadata should contain trace ID")
	}
	if !strings.Contains(metadata, "Repeated 5 times") {
		t.Error("Metadata should contain repeat count")
	}
	if !strings.Contains(metadata, "@admin:example.com") {
		t.Error("Metadata should contain admin MXID")
	}
}

func TestErrorNotifier_NotifyQuick(t *testing.T) {
	mockSender := &mockMatrixSender{}

	notifier := NewErrorNotifier(NotifierConfig{
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      true,
	})

	err := notifier.NotifyQuick(context.Background(), "CTX-001", "quick test", SeverityWarning)
	if err != nil {
		t.Fatalf("NotifyQuick() error = %v", err)
	}

	if mockSender.callCount != 1 {
		t.Error("SendMessage should be called once")
	}

	if !strings.Contains(mockSender.lastMessage, "quick test") {
		t.Error("Message should contain quick test message")
	}
}

func TestErrorNotifier_SetEnabled(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{Enabled: false})

	if notifier.IsEnabled() {
		t.Error("Should be disabled initially")
	}

	notifier.SetEnabled(true)

	if !notifier.IsEnabled() {
		t.Error("Should be enabled after SetEnabled(true)")
	}
}

func TestErrorNotifier_NoResolver(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{
		MatrixSender: &mockMatrixSender{},
		Enabled:      true,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	}

	notifyErr := notifier.Notify(context.Background(), err)
	if notifyErr == nil {
		t.Error("Notify() should return error when no resolver configured")
	}
}

func TestErrorNotifier_NoMatrixSender(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{
		Resolver: NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		Enabled:  true,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_test",
		Timestamp: time.Now(),
	}

	// Should succeed (no error) but not actually send
	notifyErr := notifier.Notify(context.Background(), err)
	if notifyErr != nil {
		t.Errorf("Notify() error = %v, want nil (no sender is ok)", notifyErr)
	}
}

func TestGlobalNotifier(t *testing.T) {
	mockSender := &mockMatrixSender{}

	notifier := NewErrorNotifier(NotifierConfig{
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      true,
	})

	SetGlobalNotifier(notifier)

	err := &TracedError{
		Code:      "GLOBAL-001",
		Category:  "test",
		Severity:  SeverityError,
		Message:   "global test",
		TraceID:   "tr_global",
		Timestamp: time.Now(),
	}

	notifyErr := GlobalNotify(context.Background(), err)
	if notifyErr != nil {
		t.Fatalf("GlobalNotify() error = %v", notifyErr)
	}

	if mockSender.callCount != 1 {
		t.Error("SendMessage should be called once")
	}
}

func TestGlobalNotifier_NotInitialized(t *testing.T) {
	SetGlobalNotifier(nil)

	err := &TracedError{Code: "TEST-001"}

	notifyErr := GlobalNotify(context.Background(), err)
	if notifyErr == nil {
		t.Error("GlobalNotify should return error when not initialized")
	}
}

func TestErrorNotifier_NotifyAndPanic(t *testing.T) {
	mockSender := &mockMatrixSender{}

	notifier := NewErrorNotifier(NotifierConfig{
		Resolver:     NewAdminResolver(AdminConfig{SetupUserMXID: "@admin:example.com"}),
		MatrixSender: mockSender,
		Enabled:      true,
	})

	err := &TracedError{
		Code:      "SYS-001",
		Category:  "system",
		Severity:  SeverityCritical,
		Message:   "critical failure",
		TraceID:   "tr_panic",
		Timestamp: time.Now(),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("NotifyAndPanic should panic")
		}
		if mockSender.callCount != 1 {
			t.Error("Should send notification before panicking")
		}
	}()

	notifier.NotifyAndPanic(context.Background(), err)
}

func TestErrorNotifier_GetRecentLogs(t *testing.T) {
	// Clear all components first
	ClearAllComponents()

	// Add some events
	TrackEvent("docker", "test_event", nil)
	TrackEvent("matrix", "test_event", nil)
	TrackEvent("rpc", "test_event", nil)

	notifier := NewErrorNotifier(NotifierConfig{Enabled: true})

	// Test container category
	logs := notifier.getRecentLogs("container")
	if len(logs) < 1 {
		t.Error("Should return logs for container category")
	}

	// Test matrix category
	logs = notifier.getRecentLogs("matrix")
	if len(logs) < 1 {
		t.Error("Should return logs for matrix category")
	}
}

func TestErrorNotifier_Setters(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{})

	// Test SetMatrixSender
	sender := &mockMatrixSender{}
	notifier.SetMatrixSender(sender)
	// Verify it's set (indirectly through using it)

	// Test SetResolver
	resolver := NewAdminResolver(AdminConfig{SetupUserMXID: "@test:example.com"})
	notifier.SetResolver(resolver)

	// Test SetStore
	tmpDir := t.TempDir()
	store, _ := NewErrorStore(StoreConfig{Path: tmpDir + "/test.db"})
	notifier.SetStore(store)
	defer store.Close()

	// Test SetRegistry
	registry := NewSamplingRegistry(DefaultSamplingConfig())
	notifier.SetRegistry(registry)
}

func TestErrorNotifier_FormatSummary_WithCause(t *testing.T) {
	notifier := NewErrorNotifier(NotifierConfig{Enabled: true})

	err := &TracedError{
		Message: "container start failed",
		cause:   New("CTX-010", "permission denied"),
	}

	summary := notifier.formatSummary(err)

	if !strings.Contains(summary, "container start failed") {
		t.Error("Summary should contain message")
	}
	if !strings.Contains(summary, "permission denied") {
		t.Error("Summary should contain cause")
	}
}
