// Package pii provides structured errors for PII/PHI compliance operations
package pii

import (
	"errors"
	"fmt"
)

// Error codes for traceability
const (
	ErrCodeScrubbingFailed     = "PII001"
	ErrCodeBufferOverflow      = "PII002"
	ErrCodeQuarantineFailed    = "PII003"
	ErrCodeInvalidConfig       = "PII004"
	ErrCodePatternCompileError = "PII005"
	ErrCodeContextCanceled     = "PII006"
)

// ComplianceError provides structured error information with traceability
type ComplianceError struct {
	Code      string // Error code for programmatic handling
	Operation string // Operation that failed (e.g., "ScrubPHI", "FlushBuffer")
	Source    string // Source of the content being processed
	Message   string // Human-readable message
	Cause     error  // Underlying error (if any)
}

func (e *ComplianceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s failed for %s: %s (caused by: %v)",
			e.Code, e.Operation, e.Source, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s failed for %s: %s",
		e.Code, e.Operation, e.Source, e.Message)
}

func (e *ComplianceError) Unwrap() error {
	return e.Cause
}

// NewScrubbingError creates an error for scrubbing failures
func NewScrubbingError(source string, cause error) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodeScrubbingFailed,
		Operation: "ScrubPHI",
		Source:    source,
		Message:   "failed to scrub content for PII/PHI",
		Cause:     cause,
	}
}

// NewBufferOverflowError creates an error for buffer overflow
func NewBufferOverflowError(source string, size int, max int) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodeBufferOverflow,
		Operation: "AppendChunk",
		Source:    source,
		Message:   fmt.Sprintf("buffer overflow: %d bytes exceeds max %d", size, max),
	}
}

// NewQuarantineError creates an error for quarantine notification failures
func NewQuarantineError(source string, cause error) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodeQuarantineFailed,
		Operation: "NotifyQuarantine",
		Source:    source,
		Message:   "failed to send quarantine notification",
		Cause:     cause,
	}
}

// NewInvalidConfigError creates an error for invalid configuration
func NewInvalidConfigError(field string, value interface{}) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodeInvalidConfig,
		Operation: "ValidateConfig",
		Source:    "config",
		Message:   fmt.Sprintf("invalid configuration: %s=%v", field, value),
	}
}

// NewPatternCompileError creates an error for pattern compilation failures
func NewPatternCompileError(patternType string, cause error) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodePatternCompileError,
		Operation: "CompilePattern",
		Source:    patternType,
		Message:   "failed to compile PHI detection pattern",
		Cause:     cause,
	}
}

// NewContextCanceledError creates an error for context cancellation
func NewContextCanceledError(source string) *ComplianceError {
	return &ComplianceError{
		Code:      ErrCodeContextCanceled,
		Operation: "ProcessResponse",
		Source:    source,
		Message:   "operation canceled by context",
	}
}

// IsComplianceError checks if an error is a ComplianceError
func IsComplianceError(err error) bool {
	var e *ComplianceError
	return errors.As(err, &e)
}

// GetErrorCode extracts the error code from a ComplianceError
func GetErrorCode(err error) string {
	var e *ComplianceError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
