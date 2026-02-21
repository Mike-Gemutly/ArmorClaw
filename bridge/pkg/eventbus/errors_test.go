// Package eventbus provides structured errors for event bus operations
package eventbus

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEventError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *EventError
		contains []string
	}{
		{
			name: "basic error",
			err: &EventError{
				Domain:   DomainPublisher,
				Code:     CodeNilEvent,
				Message:  "test error",
				Severity: SeverityError,
			},
			contains: []string{"[eventbus.publisher:E001]", "test error"},
		},
		{
			name: "error with operation",
			err: &EventError{
				Domain:     DomainPublisher,
				Code:       CodeWrapFailed,
				Message:    "wrap failed",
				Operation:  "Publish",
				Severity:   SeverityError,
			},
			contains: []string{"(Publish)", "wrap failed"},
		},
		{
			name: "error with cause",
			err: &EventError{
				Domain:   DomainSerialize,
				Code:     CodeSerializeFail,
				Message:  "serialize failed",
				Severity: SeverityError,
				Cause:    errors.New("underlying error"),
			},
			contains: []string{"cause: underlying error"},
		},
		{
			name: "error with hint",
			err: &EventError{
				Domain:   DomainWebSocket,
				Code:     CodeWSNotEnabled,
				Message:  "ws not enabled",
				Severity: SeverityInfo,
				Context:  map[string]interface{}{"hint": "enable websocket"},
			},
			contains: []string{"hint: enable websocket"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("Error() = %q, should contain %q", errStr, s)
				}
			}
		})
	}
}

func TestEventError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := &EventError{
		Domain:   DomainPublisher,
		Code:     CodeSerializeFail,
		Message:  "test",
		Severity: SeverityError,
		Cause:    cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is compatibility
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}

func TestEventError_WithMethods(t *testing.T) {
	t.Run("WithCause", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewError(DomainPublisher, CodeNilEvent, "test").WithCause(cause)
		if err.Cause != cause {
			t.Error("WithCause did not set cause")
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		err := NewError(DomainPublisher, CodeNilEvent, "test").
			WithContext("key", "value").
			WithContext("count", 42)

		if err.Context["key"] != "value" {
			t.Error("WithContext did not set string context")
		}
		if err.Context["count"] != 42 {
			t.Error("WithContext did not set int context")
		}
	})

	t.Run("WithOperation", func(t *testing.T) {
		err := NewError(DomainPublisher, CodeNilEvent, "test").WithOperation("Publish")
		if err.Operation != "Publish" {
			t.Error("WithOperation did not set operation")
		}
	})

	t.Run("WithSeverity", func(t *testing.T) {
		err := NewError(DomainPublisher, CodeNilEvent, "test").WithSeverity(SeverityFatal)
		if err.Severity != SeverityFatal {
			t.Error("WithSeverity did not set severity")
		}
	})

	t.Run("WithSource", func(t *testing.T) {
		err := NewError(DomainPublisher, CodeNilEvent, "test").WithSource(0)
		if err.Source == nil {
			t.Error("WithSource should set source")
		}
		if err.Source.File == "" {
			t.Error("WithSource should capture file")
		}
		if err.Source.Line == 0 {
			t.Error("WithSource should capture line")
		}
	})

	t.Run("WithStackTrace", func(t *testing.T) {
		err := NewError(DomainPublisher, CodeNilEvent, "test").WithStackTrace(0)
		if len(err.StackTrace) == 0 {
			t.Error("WithStackTrace should capture stack")
		}
		// Verify runtime frames are filtered out
		for _, frame := range err.StackTrace {
			if strings.Contains(frame, "runtime/") {
				t.Errorf("StackTrace should not contain runtime frames: %s", frame)
			}
		}
	})
}

func TestSourceLocation_String(t *testing.T) {
	tests := []struct {
		name string
		loc  *SourceLocation
		want string
	}{
		{
			name: "nil location",
			loc:  nil,
			want: "",
		},
		{
			name: "empty location",
			loc:  &SourceLocation{},
			want: "",
		},
		{
			name: "file and line",
			loc:  &SourceLocation{File: "test.go", Line: 42},
			want: "test.go:42",
		},
		{
			name: "with function",
			loc:  &SourceLocation{File: "test.go", Line: 42, Function: "TestFunc"},
			want: "test.go:42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.loc.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Run("ErrNilEvent", func(t *testing.T) {
		err := ErrNilEvent()
		if err.Code != CodeNilEvent {
			t.Error("wrong code")
		}
		if err.Domain != DomainPublisher {
			t.Error("wrong domain")
		}
		if err.Severity != SeverityWarning {
			t.Error("wrong severity")
		}
		if _, ok := err.Context["hint"]; !ok {
			t.Error("missing hint")
		}
	})

	t.Run("ErrWrapEventFailed", func(t *testing.T) {
		cause := errors.New("wrap cause")
		err := ErrWrapEventFailed("TestEvent", cause)
		if err.Code != CodeWrapFailed {
			t.Error("wrong code")
		}
		if err.Cause != cause {
			t.Error("wrong cause")
		}
		if err.Context["event_type"] != "TestEvent" {
			t.Error("missing event_type context")
		}
	})

	t.Run("ErrSerializeFailed", func(t *testing.T) {
		cause := errors.New("json error")
		err := ErrSerializeFailed("TestEvent", cause)
		if err.Code != CodeSerializeFail {
			t.Error("wrong code")
		}
		if err.Domain != DomainSerialize {
			t.Error("wrong domain")
		}
	})

	t.Run("ErrBroadcastFailed", func(t *testing.T) {
		cause := errors.New("broadcast error")
		err := ErrBroadcastFailed(5, cause)
		if err.Code != CodeBroadcastFail {
			t.Error("wrong code")
		}
		if err.Context["connection_count"] != 5 {
			t.Error("wrong connection count")
		}
		if err.Severity != SeverityWarning {
			t.Error("broadcast should be warning (non-fatal)")
		}
	})

	t.Run("ErrSubscriberNotFound", func(t *testing.T) {
		err := ErrSubscriberNotFound("sub-123")
		if err.Code != CodeSubNotFound {
			t.Error("wrong code")
		}
		if err.Context["subscriber_id"] != "sub-123" {
			t.Error("wrong subscriber_id")
		}
	})

	t.Run("ErrSubscriberInactive", func(t *testing.T) {
		err := ErrSubscriberInactive("sub-123", 30*time.Minute)
		if err.Code != CodeSubInactive {
			t.Error("wrong code")
		}
		if err.Context["inactive_duration"] != "30m0s" {
			t.Error("wrong duration")
		}
	})

	t.Run("ErrChannelFull", func(t *testing.T) {
		err := ErrChannelFull("sub-123", "m.room.message")
		if err.Code != CodeChannelFull {
			t.Error("wrong code")
		}
		if err.Context["event_type"] != "m.room.message" {
			t.Error("wrong event_type")
		}
	})

	t.Run("ErrSubscriberClosed", func(t *testing.T) {
		err := ErrSubscriberClosed("sub-123")
		if err.Code != CodeSubClosed {
			t.Error("wrong code")
		}
	})

	t.Run("ErrWebSocketNotEnabled", func(t *testing.T) {
		err := ErrWebSocketNotEnabled()
		if err.Code != CodeWSNotEnabled {
			t.Error("wrong code")
		}
		if err.Severity != SeverityInfo {
			t.Error("not enabled should be info, not error")
		}
	})

	t.Run("ErrWebSocketConnectFailed", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := ErrWebSocketConnectFailed("conn-456", cause)
		if err.Code != CodeWSConnectFail {
			t.Error("wrong code")
		}
		if err.Severity != SeverityError {
			t.Error("connect fail should be error")
		}
	})

	t.Run("ErrWebSocketMessageFailed", func(t *testing.T) {
		cause := errors.New("write failed")
		err := ErrWebSocketMessageFailed("conn-456", cause)
		if err.Code != CodeWSMessageFail {
			t.Error("wrong code")
		}
	})

	t.Run("ErrInvalidEventFilter", func(t *testing.T) {
		err := ErrInvalidEventFilter("empty room ID")
		if err.Code != CodeInvalidFilter {
			t.Error("wrong code")
		}
		if err.Context["reason"] != "empty room ID" {
			t.Error("wrong reason")
		}
	})
}

func TestErrorRegistry(t *testing.T) {
	t.Run("LookupError", func(t *testing.T) {
		spec, ok := LookupError(CodeNilEvent)
		if !ok {
			t.Error("CodeNilEvent should be in registry")
		}
		if spec.Code != CodeNilEvent {
			t.Error("wrong code in spec")
		}
		if spec.Description == "" {
			t.Error("spec should have description")
		}
		if spec.Resolution == "" {
			t.Error("spec should have resolution")
		}
	})

	t.Run("GetAllErrorCodes", func(t *testing.T) {
		codes := GetAllErrorCodes()
		if len(codes) == 0 {
			t.Error("registry should not be empty")
		}
		// Verify we have the expected error codes
		foundCodes := make(map[ErrorCode]bool)
		for _, spec := range codes {
			foundCodes[spec.Code] = true
		}
		expectedCodes := []ErrorCode{
			CodeNilEvent, CodeWrapFailed, CodeSerializeFail, CodeBroadcastFail,
			CodeSubNotFound, CodeSubInactive, CodeChannelFull, CodeSubClosed,
			CodeWSNotEnabled, CodeWSConnectFail, CodeWSMessageFail,
			CodeInvalidFilter,
		}
		for _, code := range expectedCodes {
			if !foundCodes[code] {
				t.Errorf("missing code in registry: %s", code)
			}
		}
	})
}

func TestIsErrorCode(t *testing.T) {
	err := ErrNilEvent()
	if !IsErrorCode(err, CodeNilEvent) {
		t.Error("IsErrorCode should match")
	}
	if IsErrorCode(err, CodeWrapFailed) {
		t.Error("IsErrorCode should not match different code")
	}
	if IsErrorCode(errors.New("plain error"), CodeNilEvent) {
		t.Error("IsErrorCode should return false for non-EventError")
	}
}

func TestAsError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		original := ErrNilEvent()
		var target *EventError
		if !AsError(original, &target) {
			t.Error("AsError should succeed")
		}
		if target != original {
			t.Error("target should point to original")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		var target *EventError
		if AsError(nil, &target) {
			t.Error("AsError should return false for nil")
		}
	})

	t.Run("non-EventError", func(t *testing.T) {
		var target *EventError
		if AsError(errors.New("plain"), &target) {
			t.Error("AsError should return false for non-EventError")
		}
	})
}

func TestGetDomainAndCode(t *testing.T) {
	err := ErrNilEvent()

	if GetDomain(err) != DomainPublisher {
		t.Error("GetDomain wrong")
	}

	if GetCode(err) != CodeNilEvent {
		t.Error("GetCode wrong")
	}

	// Test with non-EventError
	plainErr := errors.New("plain")
	if GetDomain(plainErr) != DomainEventBus {
		t.Error("GetDomain should return default for non-EventError")
	}
	if GetCode(plainErr) != "" {
		t.Error("GetCode should return empty for non-EventError")
	}
}

func TestNewErrorWithOp(t *testing.T) {
	err := NewErrorWithOp(DomainPublisher, CodeNilEvent, "Publish", "cannot publish nil")
	if err.Operation != "Publish" {
		t.Error("operation not set")
	}
	if err.Message != "cannot publish nil" {
		t.Error("message not set")
	}
}

func TestErrorChaining(t *testing.T) {
	// Test fluent API for building errors
	err := NewError(DomainPublisher, CodeSerializeFail, "failed to serialize").
		WithOperation("Publish").
		WithSeverity(SeverityError).
		WithContext("event_type", "m.room.message").
		WithCause(errors.New("json: unsupported type")).
		WithSource(0)

	// Verify all fields set
	if err.Operation != "Publish" {
		t.Error("operation not set")
	}
	if err.Severity != SeverityError {
		t.Error("severity not set")
	}
	if err.Context["event_type"] != "m.room.message" {
		t.Error("context not set")
	}
	if err.Cause == nil {
		t.Error("cause not set")
	}
	if err.Source == nil {
		t.Error("source not set")
	}

	// Verify error string contains key info
	errStr := err.Error()
	if !strings.Contains(errStr, "Publish") {
		t.Error("error string missing operation")
	}
	if !strings.Contains(errStr, "json: unsupported type") {
		t.Error("error string missing cause")
	}
}

func TestErrorTimestamp(t *testing.T) {
	before := time.Now()
	err := NewError(DomainPublisher, CodeNilEvent, "test")
	after := time.Now()

	if err.Timestamp.Before(before) {
		t.Error("timestamp should be after NewError call started")
	}
	if err.Timestamp.After(after) {
		t.Error("timestamp should be before NewError call ended")
	}
}
