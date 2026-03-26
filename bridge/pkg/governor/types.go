// Package governor provides PII interception for AI tool calls through SkillGate interface
package governor

import (
	"sync"

	"github.com/armorclaw/bridge/pkg/agent"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

// Governor is the main PII interception engine that implements SkillGate interface
// It uses the scrubber to detect and redact PII from AI tool calls and prompts
type Governor struct {
	scrubber *pii.Scrubber     // PII detection and redaction engine
	logger   *logger.Logger    // Structured logger for events and violations
	config   *Config           // Governor configuration settings
	mapping  *agent.PIIMapping // PII placeholder to original value mapping

	// Internal state
	mu sync.RWMutex // Protects concurrent access
}

// Config holds configuration for the Governor
type Config struct {
	// Logging settings
	LogViolations bool // Log PII violations to audit trail
	LogMaskedPII  bool // Include masked snippets in logs (safe for audit)

	// Scrubbing settings
	StrictMode    bool     // If true, block all tool calls with any PII detected
	AllowPatterns []string // List of pattern names to allow despite detection
	BlockPatterns []string // List of pattern names to always block

	// Shadow Mapping settings
	UseShadowMapping  bool   // If true, use Shadow Mapping 2.0 (placeholder replacement)
	PlaceholderPrefix string // Prefix for placeholders (default: "[REDACTED:")

	// Performance settings
	MaxConcurrentCalls int  // Maximum concurrent tool calls to process
	CacheMappings      bool // Cache PII mappings for performance
}

// PatternMatchInfo stores information about a detected PII pattern match
type PatternMatchInfo struct {
	PatternName string // Name of the pattern that matched (e.g., "email", "ssn")
	Start       int    // Start position in the text
	End         int    // End position in the text
	Original    string // Original matched text
	Replacement string // Replacement text (placeholder or redacted)
	Seriousness int    // Severity level (1=low, 2=medium, 3=high)
}

// ViolationSummary provides a summary of PII violations detected
type ViolationSummary struct {
	TotalViolations int                // Total number of violations detected
	Patterns        map[string]int     // Count per pattern type
	Matches         []PatternMatchInfo // Detailed match information
	Severity        int                // Maximum severity level (1-3)
	IsBlocked       bool               // Whether the call should be blocked
	Reason          string             // Reason for blocking (if applicable)
}

// ScrubbingResult contains the result of scrubbing a text or tool call
type ScrubbingResult struct {
	Original   string            // Original text
	Scrubbed   string            // Redacted text
	Violations []pii.Redaction   // List of PII violations found
	Summary    *ViolationSummary // Aggregated summary
	Mapping    *agent.PIIMapping // Shadow mapping (if enabled)
}

// GovernorStats tracks statistics about Governor operations
type GovernorStats struct {
	TotalCalls      int64 // Total number of tool calls processed
	ViolationsFound int64 // Total violations detected
	CallsBlocked    int64 // Number of calls blocked due to PII
	PIIRedactions   int64 // Total PII items redacted
	CacheHits       int64 // Mapping cache hits (if caching enabled)
	CacheMisses     int64 // Mapping cache misses
}
