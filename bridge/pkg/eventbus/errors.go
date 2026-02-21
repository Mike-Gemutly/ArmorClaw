// Package eventbus provides structured errors for event bus operations
// Each error type includes context for debugging and monitoring
package eventbus

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorDomain identifies the component that produced an error
type ErrorDomain string

const (
	DomainEventBus   ErrorDomain = "eventbus"
	DomainPublisher  ErrorDomain = "eventbus.publisher"
	DomainSubscriber ErrorDomain = "eventbus.subscriber"
	DomainWebSocket  ErrorDomain = "eventbus.websocket"
	DomainSerialize  ErrorDomain = "eventbus.serialize"
)

// ErrorCode identifies specific error conditions
type ErrorCode string

const (
	// Publisher errors (E001-E099)
	CodeNilEvent       ErrorCode = "E001" // Nil event passed to publisher
	CodeWrapFailed     ErrorCode = "E002" // Event wrapping failed
	CodeSerializeFail  ErrorCode = "E003" // JSON serialization failed
	CodeBroadcastFail  ErrorCode = "E004" // WebSocket broadcast failed

	// Subscriber errors (E101-E199)
	CodeSubNotFound    ErrorCode = "E101" // Subscriber not found
	CodeSubInactive    ErrorCode = "E102" // Subscriber inactive/timeout
	CodeChannelFull    ErrorCode = "E103" // Event channel buffer full
	CodeSubClosed      ErrorCode = "E104" // Subscriber channel closed

	// WebSocket errors (E201-E299)
	CodeWSNotEnabled   ErrorCode = "E201" // WebSocket not enabled
	CodeWSConnectFail  ErrorCode = "E202" // WebSocket connection failed
	CodeWSMessageFail  ErrorCode = "E203" // WebSocket message failed

	// Filter errors (E301-E399)
	CodeInvalidFilter  ErrorCode = "E301" // Invalid event filter
)

// ErrorSeverity indicates the severity level of the error
type ErrorSeverity string

const (
	SeverityDebug   ErrorSeverity = "debug"
	SeverityInfo    ErrorSeverity = "info"
	SeverityWarning ErrorSeverity = "warning"
	SeverityError   ErrorSeverity = "error"
	SeverityFatal   ErrorSeverity = "fatal"
)

// SourceLocation identifies where the error originated in code
type SourceLocation struct {
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

// String returns a formatted source location
func (s *SourceLocation) String() string {
	if s == nil {
		return ""
	}
	if s.File != "" && s.Line > 0 {
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	}
	return ""
}

// EventError provides structured error information with debugging context
type EventError struct {
	Domain     ErrorDomain            `json:"domain"`
	Code       ErrorCode              `json:"code"`
	Severity   ErrorSeverity          `json:"severity"`
	Message    string                 `json:"message"`
	Operation  string                 `json:"operation,omitempty"` // What operation was being performed
	Source     *SourceLocation        `json:"source,omitempty"`    // Where in code the error occurred
	Cause      error                  `json:"cause,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace []string               `json:"stack_trace,omitempty"`
}

// Error implements the error interface with full context
func (e *EventError) Error() string {
	var sb strings.Builder

	// Format: [DOMAIN:CODE] message (source)
	sb.WriteString(fmt.Sprintf("[%s:%s]", e.Domain, e.Code))

	if e.Operation != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", e.Operation))
	}

	sb.WriteString(fmt.Sprintf(" %s", e.Message))

	if e.Source != nil && e.Source.String() != "" {
		sb.WriteString(fmt.Sprintf(" @ %s", e.Source.String()))
	}

	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf("\n  └─ cause: %v", e.Cause))
	}

	// Add hints if available
	if hint, ok := e.Context["hint"]; ok {
		sb.WriteString(fmt.Sprintf("\n  └─ hint: %v", hint))
	}

	return sb.String()
}

// Unwrap returns the underlying cause for errors.Is/As
func (e *EventError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error (chain errors together)
func (e *EventError) WithCause(cause error) *EventError {
	e.Cause = cause
	return e
}

// WithContext adds contextual information for debugging
func (e *EventError) WithContext(key string, value interface{}) *EventError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithOperation sets the operation that was being performed
func (e *EventError) WithOperation(op string) *EventError {
	e.Operation = op
	return e
}

// WithSeverity sets the error severity level
func (e *EventError) WithSeverity(sev ErrorSeverity) *EventError {
	e.Severity = sev
	return e
}

// WithSource captures the source location (caller of error creator)
func (e *EventError) WithSource(skip int) *EventError {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return e
	}

	// Get function name
	pc, _, _, ok := runtime.Caller(skip + 1)
	var fn string
	if ok {
		if f := runtime.FuncForPC(pc); f != nil {
			fn = f.Name()
		}
	}

	e.Source = &SourceLocation{
		File:     file,
		Line:     line,
		Function: fn,
	}
	return e
}

// WithStackTrace captures the full stack trace
func (e *EventError) WithStackTrace(skip int) *EventError {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return e
	}

	frames := runtime.CallersFrames(pcs[:n])
	e.StackTrace = make([]string, 0, n)

	for {
		frame, more := frames.Next()
		// Skip runtime frames
		if !strings.Contains(frame.File, "runtime/") {
			e.StackTrace = append(e.StackTrace,
				fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line))
		}
		if !more {
			break
		}
	}
	return e
}

// NewError creates a new structured error with source location
func NewError(domain ErrorDomain, code ErrorCode, message string) *EventError {
	e := &EventError{
		Domain:    domain,
		Code:      code,
		Message:   message,
		Severity:  SeverityError,
		Timestamp: time.Now(),
	}
	// Capture source location automatically (skip NewError + constructor)
	e.WithSource(1)
	return e
}

// NewErrorWithOp creates an error with operation context
func NewErrorWithOp(domain ErrorDomain, code ErrorCode, operation, message string) *EventError {
	return NewError(domain, code, message).WithOperation(operation)
}

// Publisher errors

// ErrNilEvent creates an error for nil event publication
func ErrNilEvent() *EventError {
	return NewErrorWithOp(DomainPublisher, CodeNilEvent, "Publish",
		"cannot publish nil event: event must be non-nil").
		WithContext("hint", "check event creation logic").
		WithSeverity(SeverityWarning)
}

// ErrWrapEventFailed creates an error for event wrapping failures
func ErrWrapEventFailed(eventType string, cause error) *EventError {
	return NewErrorWithOp(DomainPublisher, CodeWrapFailed, "WrapEvent",
		"failed to wrap event for transmission").
		WithContext("event_type", eventType).
		WithCause(cause).
		WithSeverity(SeverityError)
}

// ErrSerializeFailed creates an error for JSON serialization failures
func ErrSerializeFailed(eventType string, cause error) *EventError {
	return NewErrorWithOp(DomainSerialize, CodeSerializeFail, "ToJSON",
		"failed to serialize event to JSON").
		WithContext("event_type", eventType).
		WithCause(cause).
		WithSeverity(SeverityError)
}

// ErrBroadcastFailed creates an error for broadcast failures
func ErrBroadcastFailed(connCount int, cause error) *EventError {
	return NewErrorWithOp(DomainWebSocket, CodeBroadcastFail, "Broadcast",
		"failed to broadcast event to WebSocket clients").
		WithContext("connection_count", connCount).
		WithCause(cause).
		WithSeverity(SeverityWarning) // Non-fatal, event still processed
}

// Subscriber errors

// ErrSubscriberNotFound creates an error for missing subscriber
func ErrSubscriberNotFound(subID string) *EventError {
	return NewErrorWithOp(DomainSubscriber, CodeSubNotFound, "Unsubscribe",
		"subscriber not found in registry").
		WithContext("subscriber_id", subID).
		WithSeverity(SeverityWarning)
}

// ErrSubscriberInactive creates an error for inactive subscribers
func ErrSubscriberInactive(subID string, inactiveDuration time.Duration) *EventError {
	return NewErrorWithOp(DomainSubscriber, CodeSubInactive, "CleanupInactive",
		"subscriber inactive for too long").
		WithContext("subscriber_id", subID).
		WithContext("inactive_duration", inactiveDuration.String()).
		WithContext("hint", "subscriber may be disconnected or blocked").
		WithSeverity(SeverityWarning)
}

// ErrChannelFull creates an error when event channel is full
func ErrChannelFull(subID string, eventType string) *EventError {
	return NewErrorWithOp(DomainSubscriber, CodeChannelFull, "Publish",
		"event channel buffer full, event dropped").
		WithContext("subscriber_id", subID).
		WithContext("event_type", eventType).
		WithContext("hint", "subscriber may be slow or blocked; consider increasing buffer size").
		WithSeverity(SeverityWarning)
}

// ErrSubscriberClosed creates an error for closed subscriber
func ErrSubscriberClosed(subID string) *EventError {
	return NewErrorWithOp(DomainSubscriber, CodeSubClosed, "CloseChannel",
		"subscriber channel already closed").
		WithContext("subscriber_id", subID).
		WithSeverity(SeverityWarning)
}

// WebSocket errors

// ErrWebSocketNotEnabled creates an error when WS is not configured
func ErrWebSocketNotEnabled() *EventError {
	return NewErrorWithOp(DomainWebSocket, CodeWSNotEnabled, "Broadcast",
		"WebSocket server not enabled in configuration").
		WithContext("hint", "set WebSocketEnabled=true in Config").
		WithSeverity(SeverityInfo)
}

// ErrWebSocketConnectFailed creates an error for connection failures
func ErrWebSocketConnectFailed(connID string, cause error) *EventError {
	return NewErrorWithOp(DomainWebSocket, CodeWSConnectFail, "Connect",
		"WebSocket connection failed").
		WithContext("connection_id", connID).
		WithCause(cause).
		WithSeverity(SeverityError)
}

// ErrWebSocketMessageFailed creates an error for message send failures
func ErrWebSocketMessageFailed(connID string, cause error) *EventError {
	return NewErrorWithOp(DomainWebSocket, CodeWSMessageFail, "SendMessage",
		"WebSocket message send failed").
		WithContext("connection_id", connID).
		WithCause(cause).
		WithSeverity(SeverityError)
}

// Filter errors

// ErrInvalidEventFilter creates an error for invalid filters
func ErrInvalidEventFilter(reason string) *EventError {
	return NewErrorWithOp(DomainEventBus, CodeInvalidFilter, "Subscribe",
		"invalid event filter specified").
		WithContext("reason", reason).
		WithSeverity(SeverityWarning)
}

// ============================================================================
// Error Code Registry (for documentation and debugging)
// ============================================================================

// ErrorSpec describes an error code for documentation
type ErrorSpec struct {
	Code        ErrorCode     `json:"code"`
	Domain      ErrorDomain   `json:"domain"`
	Message     string        `json:"message"`
	Severity    ErrorSeverity `json:"severity"`
	Description string        `json:"description"`
	Resolution  string        `json:"resolution"`
}

// ErrorRegistry contains all known error codes
var ErrorRegistry = map[ErrorCode]ErrorSpec{
	// Publisher errors
	CodeNilEvent: {
		Code:        CodeNilEvent,
		Domain:      DomainPublisher,
		Message:     "cannot publish nil event",
		Severity:    SeverityWarning,
		Description: "Attempted to publish a nil event pointer",
		Resolution:  "Ensure event is created before calling Publish()",
	},
	CodeWrapFailed: {
		Code:        CodeWrapFailed,
		Domain:      DomainPublisher,
		Message:     "failed to wrap event for transmission",
		Severity:    SeverityError,
		Description: "Event could not be wrapped for WebSocket transmission",
		Resolution:  "Check event structure and serialization",
	},
	CodeSerializeFail: {
		Code:        CodeSerializeFail,
		Domain:      DomainSerialize,
		Message:     "failed to serialize event to JSON",
		Severity:    SeverityError,
		Description: "Event contains fields that cannot be serialized to JSON",
		Resolution:  "Ensure all event fields are JSON-serializable",
	},
	CodeBroadcastFail: {
		Code:        CodeBroadcastFail,
		Domain:      DomainWebSocket,
		Message:     "failed to broadcast event",
		Severity:    SeverityWarning,
		Description: "WebSocket broadcast failed (event may still be processed)",
		Resolution:  "Check WebSocket server status and client connections",
	},

	// Subscriber errors
	CodeSubNotFound: {
		Code:        CodeSubNotFound,
		Domain:      DomainSubscriber,
		Message:     "subscriber not found",
		Severity:    SeverityWarning,
		Description: "Referenced subscriber does not exist in registry",
		Resolution:  "Check subscriber ID or verify subscription was created",
	},
	CodeSubInactive: {
		Code:        CodeSubInactive,
		Domain:      DomainSubscriber,
		Message:     "subscriber inactive",
		Severity:    SeverityWarning,
		Description: "Subscriber has not shown activity for too long",
		Resolution:  "Subscriber may be disconnected; will be cleaned up",
	},
	CodeChannelFull: {
		Code:        CodeChannelFull,
		Domain:      DomainSubscriber,
		Message:     "event channel buffer full",
		Severity:    SeverityWarning,
		Description: "Event dropped because subscriber channel is full",
		Resolution:  "Subscriber is slow; consider increasing buffer size or optimizing handler",
	},
	CodeSubClosed: {
		Code:        CodeSubClosed,
		Domain:      DomainSubscriber,
		Message:     "subscriber channel already closed",
		Severity:    SeverityWarning,
		Description: "Attempted to operate on closed subscriber channel",
		Resolution:  "Check if subscriber was already unsubscribed",
	},

	// WebSocket errors
	CodeWSNotEnabled: {
		Code:        CodeWSNotEnabled,
		Domain:      DomainWebSocket,
		Message:     "WebSocket server not enabled",
		Severity:    SeverityInfo,
		Description: "Attempted WebSocket operation when server is disabled",
		Resolution:  "Set WebSocketEnabled=true in configuration",
	},
	CodeWSConnectFail: {
		Code:        CodeWSConnectFail,
		Domain:      DomainWebSocket,
		Message:     "WebSocket connection failed",
		Severity:    SeverityError,
		Description: "Could not establish WebSocket connection",
		Resolution:  "Check network connectivity and server status",
	},
	CodeWSMessageFail: {
		Code:        CodeWSMessageFail,
		Domain:      DomainWebSocket,
		Message:     "WebSocket message failed",
		Severity:    SeverityError,
		Description: "Failed to send message over WebSocket",
		Resolution:  "Check connection status and reconnect if needed",
	},

	// Filter errors
	CodeInvalidFilter: {
		Code:        CodeInvalidFilter,
		Domain:      DomainEventBus,
		Message:     "invalid event filter",
		Severity:    SeverityWarning,
		Description: "Provided filter is invalid or malformed",
		Resolution:  "Check filter syntax and values",
	},
}

// LookupError returns the spec for an error code
func LookupError(code ErrorCode) (ErrorSpec, bool) {
	spec, ok := ErrorRegistry[code]
	return spec, ok
}

// GetAllErrorCodes returns all registered error codes
func GetAllErrorCodes() []ErrorSpec {
	specs := make([]ErrorSpec, 0, len(ErrorRegistry))
	for _, spec := range ErrorRegistry {
		specs = append(specs, spec)
	}
	return specs
}

// IsErrorCode checks if an error matches a specific code
func IsErrorCode(err error, code ErrorCode) bool {
	var eventErr *EventError
	if AsError(err, &eventErr) {
		return eventErr.Code == code
	}
	return false
}

// AsError attempts to convert an error to EventError
func AsError(err error, target **EventError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*EventError); ok {
		*target = e
		return true
	}
	return false
}

// GetDomain extracts the error domain from an error
func GetDomain(err error) ErrorDomain {
	var eventErr *EventError
	if AsError(err, &eventErr) {
		return eventErr.Domain
	}
	return DomainEventBus
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	var eventErr *EventError
	if AsError(err, &eventErr) {
		return eventErr.Code
	}
	return ""
}
