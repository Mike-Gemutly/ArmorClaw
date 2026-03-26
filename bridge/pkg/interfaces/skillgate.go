// Package interfaces defines shared interfaces for PII interception and governance
// This package provides the SkillGate interface for secure AI tool call processing
package interfaces

import (
	"context"
	"regexp"
)

// SkillGate defines the interface for PII interception in AI tool calls
// Implementations must ensure AI tools never see raw user data, and restore
// PII only when returning results to the user's secure enclave
type SkillGate interface {
	// InterceptToolCall intercepts and scrubs PII from tool call arguments before
	// execution. Returns modified ToolCall with redacted arguments.
	InterceptToolCall(ctx context.Context, call *ToolCall) (*ToolCall, error)

	// InterceptPrompt scans and redacts PII from user prompts before they reach
	// the AI model. Returns redacted prompt and PIIMapping for restoration.
	InterceptPrompt(ctx context.Context, prompt string) (string, *PIIMapping, error)

	// RestoreOutput restores redacted PII placeholders in AI output with original
	// values from the PIIMapping. Used when returning results to user's enclave.
	RestoreOutput(ctx context.Context, output string, mapping *PIIMapping) (string, error)

	// ValidateArgs validates tool call arguments for PII violations without modifying
	// the call. Returns list of violations found.
	ValidateArgs(ctx context.Context, toolName string, args map[string]interface{}) ([]PIIViolation, error)
}

// ToolCall represents a tool call request with PII-sensitive arguments
type ToolCall struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Priority  int                    `json:"priority,omitempty"`
}

// PIIMapping tracks the relationship between original and redacted PII values
// Enables restoration of PII when returning results to the user's secure enclave
type PIIMapping struct {
	OriginalArgs map[string]interface{} `json:"original_args"`
	RedactedArgs map[string]interface{} `json:"redacted_args"`
	Placeholders map[string]string      `json:"placeholders"`
}

// PIIViolation represents a detected PII policy violation
type PIIViolation struct {
	Field       string `json:"field"`
	PatternType string `json:"pattern_type"`
	Message     string `json:"message"`
	Severity    string `json:"severity"` // "low", "medium", "high", "critical"
}

// PIIPattern defines a PII detection pattern for validation
// Pattern definitions reference bridge/pkg/pii/scrubber.go patterns
type PIIPattern struct {
	Name        string         `json:"name"`
	Pattern     *regexp.Regexp `json:"-"`
	Replacement string         `json:"replacement"`
	Description string         `json:"description"`
}

// SkillGateConfig provides configuration for SkillGate implementations
type SkillGateConfig struct {
	Enabled        bool     `json:"enabled"`
	StrictMode     bool     `json:"strict_mode"`     // Fail on first violation vs collect all
	LogViolations  bool     `json:"log_violations"`  // Log all PII violations
	AllowedTools   []string `json:"allowed_tools"`   // Tools exempt from PII checks
	BlockedTools   []string `json:"blocked_tools"`   // Tools that require PII (e.g., payment)
	RedactionStyle string   `json:"redaction_style"` // "placeholder", "hash", "mask"
}

// DefaultPIIPatterns returns the 7 core PII patterns for SkillGate validation
// References patterns from bridge/pkg/pii/scrubber.go:15-80
func DefaultPIIPatterns() []*PIIPattern {
	return []*PIIPattern{
		{
			Name:        "email",
			Pattern:     regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`),
			Replacement: "[REDACTED_EMAIL]",
			Description: "Email addresses",
		},
		{
			Name:        "phone",
			Pattern:     regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`),
			Replacement: "[REDACTED_PHONE]",
			Description: "US phone numbers (10-digit format)",
		},
		{
			Name:        "ssn",
			Pattern:     regexp.MustCompile(`\b[0-9]{3}-[0-9]{2}-[0-9]{4}\b`),
			Replacement: "[REDACTED_SSN]",
			Description: "US Social Security Numbers",
		},
		{
			Name:        "credit_card",
			Pattern:     regexp.MustCompile(`\b4[0-9]{12}(?:[0-9]{3})?\b|\b5[1-5][0-9]{14}\b|\b6(?:011|5[0-9][0-9])[0-9]{12}\b|\b3[47][0-9]{13}\b`),
			Replacement: "[REDACTED_CREDIT_CARD]",
			Description: "Credit card numbers (Visa, MC, Amex, Discover)",
		},
		{
			Name:        "api_key",
			Pattern:     regexp.MustCompile(`\b(sk_|pk_|ai_)[a-zA-Z0-9_-]{20,}\b`),
			Replacement: "[REDACTED_API_KEY]",
			Description: "API keys with common prefixes",
		},
		{
			Name:        "jwt",
			Pattern:     regexp.MustCompile(`(?i)Bearer\s+[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`),
			Replacement: "[REDACTED_JWT]",
			Description: "JWT bearer tokens",
		},
		{
			Name:        "ip_address",
			Pattern:     regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
			Replacement: "[REDACTED_IP]",
			Description: "IPv4 addresses",
		},
	}
}
