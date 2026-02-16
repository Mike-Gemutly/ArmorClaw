// Package errors provides structured error handling with detailed traces
// for LLM-assisted debugging and admin notification.
package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Severity levels for errors
type Severity string

const (
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// ComponentLogEntry represents a recent event from a component tracker
type ComponentLogEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Component string      `json:"component"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data,omitempty"`
}

// TracedError is a structured error with detailed context for debugging
type TracedError struct {
	// Identification
	Code     string   `json:"code"`
	Category string   `json:"category"`
	TraceID  string   `json:"trace_id"`

	// Severity
	Severity Severity `json:"severity"`

	// Error details
	Message  string `json:"message"`
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`

	// Context
	Inputs     map[string]interface{} `json:"inputs,omitempty"`
	State      map[string]interface{} `json:"state,omitempty"`
	Stack      []StackFrame           `json:"stack,omitempty"`
	RecentLogs []ComponentLogEntry    `json:"recent_logs,omitempty"`

	// Tracking
	Timestamp   time.Time `json:"timestamp"`
	RepeatCount int       `json:"repeat_count,omitempty"`

	// Wrapped error
	cause error `json:"-"`
}

// Error implements the error interface
func (e *TracedError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *TracedError) Unwrap() error {
	return e.cause
}

// FormatSummary returns a human-readable summary for notification
func (e *TracedError) FormatSummary() string {
	var sb strings.Builder

	// Severity emoji
	emoji := "⚠️"
	switch e.Severity {
	case SeverityError:
		emoji = "❌"
	case SeverityCritical:
		emoji = "🔴"
	}

	sb.WriteString(fmt.Sprintf("%s %s: %s\n\n", emoji, strings.ToUpper(string(e.Severity)), e.Code))
	sb.WriteString(e.Message)
	if e.cause != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.cause))
	}
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("📍 Location: %s @ %s:%d\n", e.Function, e.File, e.Line))
	sb.WriteString(fmt.Sprintf("🏷️ Trace ID: %s\n", e.TraceID))
	sb.WriteString(fmt.Sprintf("⏰ %s\n", e.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC")))

	if e.RepeatCount > 0 {
		sb.WriteString(fmt.Sprintf("🔁 Repeated %d times\n", e.RepeatCount))
	}

	return sb.String()
}

// FormatJSON returns the full trace as formatted JSON
func (e *TracedError) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatForLLM returns the complete LLM-friendly message
func (e *TracedError) FormatForLLM() string {
	var sb strings.Builder

	sb.WriteString(e.FormatSummary())
	sb.WriteString("\n```json\n")

	jsonStr, err := e.FormatJSON()
	if err != nil {
		sb.WriteString(fmt.Sprintf(`{"error": "failed to serialize: %s"}`, err))
	} else {
		sb.WriteString(jsonStr)
	}

	sb.WriteString("\n```\n\n")
	sb.WriteString("📋 Copy the JSON block above to analyze with an LLM.")

	return sb.String()
}

// ErrorBuilder constructs TracedError instances with fluent API
type ErrorBuilder struct {
	err *TracedError
}

// traceIDGenerator generates unique trace IDs
var (
	traceIDCounter uint64
	traceIDMu      sync.Mutex
)

func generateTraceID() string {
	traceIDMu.Lock()
	defer traceIDMu.Unlock()
	traceIDCounter++
	return fmt.Sprintf("tr_%x_%d", time.Now().UnixNano(), traceIDCounter)
}

// captureStack captures the current call stack, skipping the specified number of frames
func captureStack(skip int) []StackFrame {
	var frames []StackFrame

	// Capture up to 32 frames
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip+2, pcs) // +2 to skip captureStack and builder
	if n == 0 {
		return frames
	}

	pcs = pcs[:n]
	callers := runtime.CallersFrames(pcs)

	for {
		frame, more := callers.Next()
		if frame.Function == "main.main" {
			// Include main.main but stop after it
			frames = append(frames, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
			break
		}

		// Skip runtime internals
		if strings.HasPrefix(frame.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}

		frames = append(frames, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})

		if !more {
			break
		}
	}

	return frames
}

// NewBuilder creates a new error builder for the given code
func NewBuilder(code string) *ErrorBuilder {
	// Get caller info for default location
	_, file, line, _ := runtime.Caller(1)

	def := Lookup(code)

	return &ErrorBuilder{
		err: &TracedError{
			Code:       code,
			Category:   def.Category,
			Severity:   def.Severity,
			Message:    def.Message,
			TraceID:    generateTraceID(),
			Timestamp:  time.Now(),
			File:       file,
			Line:       line,
			Inputs:     make(map[string]interface{}),
			State:      make(map[string]interface{}),
			Stack:      captureStack(1),
			RepeatCount: 0,
		},
	}
}

// Wrap wraps an existing error with this code
func (b *ErrorBuilder) Wrap(cause error) *ErrorBuilder {
	b.err.cause = cause
	if b.err.Message == "" && cause != nil {
		b.err.Message = cause.Error()
	}
	return b
}

// WithMessage sets a custom message
func (b *ErrorBuilder) WithMessage(msg string) *ErrorBuilder {
	b.err.Message = msg
	return b
}

// WithMessagef sets a formatted custom message
func (b *ErrorBuilder) WithMessagef(format string, args ...interface{}) *ErrorBuilder {
	b.err.Message = fmt.Sprintf(format, args...)
	return b
}

// WithSeverity overrides the default severity
func (b *ErrorBuilder) WithSeverity(sev Severity) *ErrorBuilder {
	b.err.Severity = sev
	return b
}

// WithFunction sets the function name where the error occurred
func (b *ErrorBuilder) WithFunction(fn string) *ErrorBuilder {
	b.err.Function = fn
	return b
}

// WithLocation sets the file and line explicitly
func (b *ErrorBuilder) WithLocation(file string, line int) *ErrorBuilder {
	b.err.File = file
	b.err.Line = line
	return b
}

// WithInputs sets the function inputs/parameters
func (b *ErrorBuilder) WithInputs(inputs map[string]interface{}) *ErrorBuilder {
	if inputs != nil {
		b.err.Inputs = inputs
	}
	return b
}

// WithInput adds a single input parameter
func (b *ErrorBuilder) WithInput(key string, value interface{}) *ErrorBuilder {
	b.err.Inputs[key] = value
	return b
}

// WithState sets the state snapshot at time of error
func (b *ErrorBuilder) WithState(state map[string]interface{}) *ErrorBuilder {
	if state != nil {
		b.err.State = state
	}
	return b
}

// WithStateValue adds a single state value
func (b *ErrorBuilder) WithStateValue(key string, value interface{}) *ErrorBuilder {
	b.err.State[key] = value
	return b
}

// WithRecentLogs sets the recent component logs
func (b *ErrorBuilder) WithRecentLogs(logs []ComponentLogEntry) *ErrorBuilder {
	b.err.RecentLogs = logs
	return b
}

// WithRepeatCount sets the repeat count for rate-limited notifications
func (b *ErrorBuilder) WithRepeatCount(count int) *ErrorBuilder {
	b.err.RepeatCount = count
	return b
}

// Build creates the final TracedError
func (b *ErrorBuilder) Build() *TracedError {
	// Clean up empty maps to reduce JSON size
	if len(b.err.Inputs) == 0 {
		b.err.Inputs = nil
	}
	if len(b.err.State) == 0 {
		b.err.State = nil
	}
	if len(b.err.RecentLogs) == 0 {
		b.err.RecentLogs = nil
	}

	return b.err
}

// Error returns the built error as an error interface
func (b *ErrorBuilder) Error() error {
	return b.Build()
}

// Quick constructors for common use cases

// New creates a new traced error with just a code and message
func New(code, message string) *TracedError {
	return NewBuilder(code).WithMessage(message).Build()
}

// Newf creates a new traced error with formatted message
func Newf(code, format string, args ...interface{}) *TracedError {
	return NewBuilder(code).WithMessagef(format, args...).Build()
}

// Wrap wraps an error with a code
func Wrap(code string, cause error) *TracedError {
	return NewBuilder(code).Wrap(cause).Build()
}

// WrapWithMessage wraps an error with a code and custom message
func WrapWithMessage(code string, cause error, message string) *TracedError {
	return NewBuilder(code).Wrap(cause).WithMessage(message).Build()
}
