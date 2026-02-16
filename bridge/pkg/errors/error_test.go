package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTracedError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TracedError
		expected string
	}{
		{
			name: "error without cause",
			err: &TracedError{
				Code:    "CTX-001",
				Message: "container start failed",
			},
			expected: "CTX-001: container start failed",
		},
		{
			name: "error with cause",
			err: &TracedError{
				Code:    "CTX-001",
				Message: "container start failed",
				cause:   errors.New("permission denied"),
			},
			expected: "CTX-001: container start failed: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTracedError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &TracedError{
		Code:    "TEST-001",
		Message: "test error",
		cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is compatibility
	if !errors.Is(err, cause) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestTracedError_FormatSummary(t *testing.T) {
	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "container start failed",
		Function:  "StartContainer",
		File:      "docker/client.go",
		Line:      142,
		TraceID:   "tr_abc123",
		Timestamp: time.Date(2026, 2, 15, 18, 32, 5, 0, time.UTC),
	}

	summary := err.FormatSummary()

	// Check key components are present
	if !strings.Contains(summary, "❌") {
		t.Error("Summary should contain error emoji")
	}
	if !strings.Contains(summary, "ERROR") {
		t.Error("Summary should contain severity")
	}
	if !strings.Contains(summary, "CTX-001") {
		t.Error("Summary should contain error code")
	}
	if !strings.Contains(summary, "StartContainer") {
		t.Error("Summary should contain function name")
	}
	if !strings.Contains(summary, "docker/client.go") {
		t.Error("Summary should contain file name")
	}
	if !strings.Contains(summary, "tr_abc123") {
		t.Error("Summary should contain trace ID")
	}
}

func TestTracedError_FormatSummary_Critical(t *testing.T) {
	err := &TracedError{
		Code:     "SYS-001",
		Severity: SeverityCritical,
		Message:  "keystore decryption failed",
	}

	summary := err.FormatSummary()

	if !strings.Contains(summary, "🔴") {
		t.Error("Critical errors should use red circle emoji")
	}
}

func TestTracedError_FormatSummary_Warning(t *testing.T) {
	err := &TracedError{
		Code:     "RPC-001",
		Severity: SeverityWarning,
		Message:  "invalid request",
	}

	summary := err.FormatSummary()

	if !strings.Contains(summary, "⚠️") {
		t.Error("Warning errors should use warning emoji")
	}
}

func TestTracedError_FormatSummary_RepeatCount(t *testing.T) {
	err := &TracedError{
		Code:        "CTX-001",
		Severity:    SeverityError,
		Message:     "container start failed",
		RepeatCount: 5,
	}

	summary := err.FormatSummary()

	if !strings.Contains(summary, "Repeated 5 times") {
		t.Error("Summary should show repeat count when > 0")
	}
}

func TestTracedError_FormatJSON(t *testing.T) {
	err := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "container start failed",
		Function:  "StartContainer",
		TraceID:   "tr_test",
		Timestamp: time.Date(2026, 2, 15, 18, 32, 5, 0, time.UTC),
	}

	json, err2 := err.FormatJSON()
	if err2 != nil {
		t.Fatalf("FormatJSON() error = %v", err2)
	}

	// Check JSON contains expected fields
	if !strings.Contains(json, `"code": "CTX-001"`) {
		t.Error("JSON should contain code field")
	}
	if !strings.Contains(json, `"category": "container"`) {
		t.Error("JSON should contain category field")
	}
	if !strings.Contains(json, `"severity": "error"`) {
		t.Error("JSON should contain severity field")
	}
}

func TestTracedError_FormatForLLM(t *testing.T) {
	err := &TracedError{
		Code:     "CTX-001",
		Severity: SeverityError,
		Message:  "container start failed",
		TraceID:  "tr_test",
	}

	output := err.FormatForLLM()

	// Check structure
	if !strings.Contains(output, "```json") {
		t.Error("LLM format should contain JSON code block")
	}
	if !strings.Contains(output, "📋 Copy the JSON block") {
		t.Error("LLM format should contain copy instruction")
	}
}

func TestErrorBuilder_Build(t *testing.T) {
	err := NewBuilder("CTX-001").
		WithMessage("custom message").
		WithFunction("TestFunc").
		WithLocation("test.go", 100).
		WithInput("container_id", "abc123").
		WithStateValue("connected", true).
		Build()

	if err.Code != "CTX-001" {
		t.Errorf("Code = %q, want %q", err.Code, "CTX-001")
	}
	if err.Message != "custom message" {
		t.Errorf("Message = %q, want %q", err.Message, "custom message")
	}
	if err.Function != "TestFunc" {
		t.Errorf("Function = %q, want %q", err.Function, "TestFunc")
	}
	if err.File != "test.go" {
		t.Errorf("File = %q, want %q", err.File, "test.go")
	}
	if err.Line != 100 {
		t.Errorf("Line = %d, want %d", err.Line, 100)
	}
	if err.Inputs["container_id"] != "abc123" {
		t.Error("Inputs should contain container_id")
	}
	if err.State["connected"] != true {
		t.Error("State should contain connected=true")
	}
}

func TestErrorBuilder_Wrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewBuilder("CTX-001").
		Wrap(cause).
		Build()

	if err.cause != cause {
		t.Error("Wrap should set cause")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should work with wrapped error")
	}
}

func TestErrorBuilder_WithInputs(t *testing.T) {
	inputs := map[string]interface{}{
		"id":    "abc123",
		"count": 42,
	}

	err := NewBuilder("CTX-001").
		WithInputs(inputs).
		Build()

	if err.Inputs["id"] != "abc123" {
		t.Error("Inputs[id] should be set")
	}
	if err.Inputs["count"] != 42 {
		t.Error("Inputs[count] should be set")
	}
}

func TestErrorBuilder_WithState(t *testing.T) {
	state := map[string]interface{}{
		"running": true,
		"uptime":  3600,
	}

	err := NewBuilder("CTX-001").
		WithState(state).
		Build()

	if err.State["running"] != true {
		t.Error("State[running] should be set")
	}
	if err.State["uptime"] != 3600 {
		t.Error("State[uptime] should be set")
	}
}

func TestErrorBuilder_WithRecentLogs(t *testing.T) {
	logs := []ComponentLogEntry{
		{Component: "docker", Event: "start_attempt"},
		{Component: "docker", Event: "pull_image"},
	}

	err := NewBuilder("CTX-001").
		WithRecentLogs(logs).
		Build()

	if len(err.RecentLogs) != 2 {
		t.Errorf("RecentLogs count = %d, want 2", len(err.RecentLogs))
	}
}

func TestErrorBuilder_EmptyMapsCleanedUp(t *testing.T) {
	err := NewBuilder("CTX-001").
		Build()

	// Empty maps should be nil to reduce JSON size
	if err.Inputs != nil {
		t.Error("Empty Inputs should be nil")
	}
	if err.State != nil {
		t.Error("Empty State should be nil")
	}
	if err.RecentLogs != nil {
		t.Error("Empty RecentLogs should be nil")
	}
}

func TestErrorBuilder_WithSeverity(t *testing.T) {
	err := NewBuilder("CTX-001").
		WithSeverity(SeverityCritical).
		Build()

	if err.Severity != SeverityCritical {
		t.Errorf("Severity = %q, want %q", err.Severity, SeverityCritical)
	}
}

func TestErrorBuilder_WithRepeatCount(t *testing.T) {
	err := NewBuilder("CTX-001").
		WithRepeatCount(10).
		Build()

	if err.RepeatCount != 10 {
		t.Errorf("RepeatCount = %d, want 10", err.RepeatCount)
	}
}

func TestQuickConstructors(t *testing.T) {
	// Test New
	err1 := New("CTX-001", "test message")
	if err1.Code != "CTX-001" {
		t.Error("New() should set code")
	}
	if err1.Message != "test message" {
		t.Error("New() should set message")
	}

	// Test Newf
	err2 := Newf("CTX-001", "test %s", "formatted")
	if err2.Message != "test formatted" {
		t.Errorf("Newf() message = %q, want %q", err2.Message, "test formatted")
	}

	// Test Wrap
	cause := errors.New("cause")
	err3 := Wrap("CTX-001", cause)
	if err3.cause != cause {
		t.Error("Wrap() should set cause")
	}

	// Test WrapWithMessage
	err4 := WrapWithMessage("CTX-001", cause, "custom message")
	if err4.Message != "custom message" {
		t.Error("WrapWithMessage() should set message")
	}
	if err4.cause != cause {
		t.Error("WrapWithMessage() should set cause")
	}
}

func TestCaptureStack(t *testing.T) {
	err := NewBuilder("CTX-001").Build()

	if len(err.Stack) == 0 {
		t.Error("Stack should be captured")
	}

	// Stack should contain this test function
	found := false
	for _, frame := range err.Stack {
		if strings.Contains(frame.Function, "TestCaptureStack") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Stack should contain TestCaptureStack")
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()

	if id1 == id2 {
		t.Error("Trace IDs should be unique")
	}
	if !strings.HasPrefix(id1, "tr_") {
		t.Errorf("Trace ID should start with 'tr_', got %q", id1)
	}
}

func TestLookupKnownCode(t *testing.T) {
	def := Lookup("CTX-001")

	if def.Code != "CTX-001" {
		t.Errorf("Code = %q, want %q", def.Code, "CTX-001")
	}
	if def.Category != "container" {
		t.Errorf("Category = %q, want %q", def.Category, "container")
	}
	if def.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestLookupUnknownCode(t *testing.T) {
	def := Lookup("UNKNOWN-999")

	if def.Code != "UNKNOWN-999" {
		t.Errorf("Code = %q, want %q", def.Code, "UNKNOWN-999")
	}
	if def.Category != "unknown" {
		t.Errorf("Category = %q, want 'unknown'", def.Category)
	}
}

func TestRegister(t *testing.T) {
	customCode := ErrorCodeDefinition{
		Code:     "CUSTOM-001",
		Category: "custom",
		Severity: SeverityWarning,
		Message:  "custom error",
		Help:     "custom help",
	}

	Register(customCode)

	def := Lookup("CUSTOM-001")
	if def.Code != "CUSTOM-001" {
		t.Error("Register should add code to registry")
	}
	if def.Category != "custom" {
		t.Error("Registered code should have correct category")
	}
}

func TestAllCodes(t *testing.T) {
	codes := AllCodes()

	if len(codes) == 0 {
		t.Error("AllCodes should return registered codes")
	}

	// Should contain at least the default codes
	if _, ok := codes["CTX-001"]; !ok {
		t.Error("AllCodes should contain CTX-001")
	}
}

func TestCodesByCategory(t *testing.T) {
	containerCodes := CodesByCategory("container")

	if len(containerCodes) == 0 {
		t.Error("CodesByCategory should return container codes")
	}

	for _, code := range containerCodes {
		if code.Category != "container" {
			t.Errorf("Expected container category, got %q", code.Category)
		}
	}
}

func TestCodesBySeverity(t *testing.T) {
	criticalCodes := CodesBySeverity(SeverityCritical)

	for _, code := range criticalCodes {
		if code.Severity != SeverityCritical {
			t.Errorf("Expected critical severity, got %q", code.Severity)
		}
	}
}
