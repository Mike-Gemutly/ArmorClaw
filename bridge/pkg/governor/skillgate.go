// Package governor implements the DefaultSkillGate for PII interception in AI tool calls
package governor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/armorclaw/bridge/pkg/interfaces"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

// NewGovernor creates a new Governor instance with the given configuration
func NewGovernor(cfg *Config, log *logger.Logger) *Governor {
	if cfg == nil {
		cfg = &Config{
			LogViolations:      true,
			LogMaskedPII:       true,
			StrictMode:         false,
			UseShadowMapping:   true,
			PlaceholderPrefix:  "[REDACTED:",
			MaxConcurrentCalls: 100,
			CacheMappings:      true,
		}
	}

	if log == nil {
		log, _ = logger.New(logger.Config{Level: "info"})
	}

	return &Governor{
		scrubber: pii.New(),
		logger:   log,
		config:   cfg,
		mapping:  nil, // Created per-call in InterceptToolCall
	}
}

// InterceptToolCall intercepts and scrubs PII from tool call arguments before execution
// Uses CTO-provided boilerplate as foundation with Shadow Mapping support
func (g *Governor) InterceptToolCall(ctx context.Context, call *interfaces.ToolCall) (*interfaces.ToolCall, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	mapping := &interfaces.PIIMapping{
		OriginalArgs: call.Arguments,
		RedactedArgs: make(map[string]interface{}),
		Placeholders: make(map[string]string),
	}

	for key, value := range call.Arguments {
		strVal, ok := value.(string)
		if !ok {
			mapping.RedactedArgs[key] = value
			continue
		}

		// Scrub PII using existing patterns from pii/scrubber.go
		scrubbed, violations := g.scrubber.Scrub(strVal)
		if len(violations) > 0 {
			// Log violation for security audit (only masked snippets, not raw PII)
			if g.config.LogViolations {
				maskedSnippet := g.maskForLog(strVal)
				g.logger.Warn("PII violation detected in tool call",
					"tool", call.ToolName,
					"key", key,
					"violations", len(violations),
					"masked_snippet", maskedSnippet,
					"pattern_types", g.getPatternTypes(violations))
			}

			// If Shadow Mapping is enabled, use hash-based placeholders
			if g.config.UseShadowMapping {
				hash := g.computeHash(strVal)
				placeholder := g.config.PlaceholderPrefix + hash + "]"
				mapping.Placeholders[placeholder] = strVal
				mapping.RedactedArgs[key] = placeholder
			} else {
				// Otherwise use standard redaction
				mapping.RedactedArgs[key] = scrubbed
			}
		} else {
			mapping.RedactedArgs[key] = value
		}
	}

	// Store mapping for restoration (Governor doesn't persist state across calls)
	g.mapping = mapping
	call.Arguments = mapping.RedactedArgs

	return call, nil
}

// InterceptPrompt scans and redacts PII from user prompts before they reach the AI model
// Returns redacted prompt and PIIMapping for restoration
func (g *Governor) InterceptPrompt(ctx context.Context, prompt string) (string, *interfaces.PIIMapping, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	mapping := &interfaces.PIIMapping{
		OriginalArgs: map[string]interface{}{"prompt": prompt},
		RedactedArgs: make(map[string]interface{}),
		Placeholders: make(map[string]string),
	}

	// Scrub PII using existing patterns from pii/scrubber.go
	scrubbed, violations := g.scrubber.Scrub(prompt)
	if len(violations) > 0 {
		// Log violation for security audit (only masked snippets)
		if g.config.LogViolations {
			maskedSnippet := g.maskForLog(prompt)
			g.logger.Warn("PII violation detected in prompt",
				"violations", len(violations),
				"masked_snippet", maskedSnippet,
				"pattern_types", g.getPatternTypes(violations))

			// Log each violation type count for compliance
			summary := g.scrubber.CreateSummary(violations)
			for patternType, count := range summary {
				g.logger.Info("PII pattern summary", "pattern", patternType, "count", count)
			}
		}

		// Use Shadow Mapping for placeholders
		if g.config.UseShadowMapping {
			// For each violation, create a hash-based placeholder
			modified := prompt
			placeholderMap := make(map[string]string)

			// Process violations in reverse order to maintain positions
			for i := len(violations) - 1; i >= 0; i-- {
				v := violations[i]
				original := prompt[v.Start:v.End]
				hash := g.computeHash(original)
				placeholder := g.config.PlaceholderPrefix + hash + "]"

				// Store in mapping
				placeholderMap[placeholder] = original
				mapping.Placeholders[placeholder] = original

				// Replace in prompt
				modified = modified[:v.Start] + placeholder + modified[v.End:]
			}

			scrubbed = modified
		}
	}

	mapping.RedactedArgs["prompt"] = scrubbed

	return scrubbed, mapping, nil
}

// RestoreOutput restores redacted PII placeholders in AI output with original values
// from the PIIMapping. Used when returning results to user's secure enclave.
func (g *Governor) RestoreOutput(ctx context.Context, output string, mapping *interfaces.PIIMapping) (string, error) {
	if mapping == nil || len(mapping.Placeholders) == 0 {
		return output, nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	restored := output

	// Restore each placeholder with its original value
	for placeholder, original := range mapping.Placeholders {
		// Replace all occurrences of placeholder
		restored = strings.ReplaceAll(restored, placeholder, original)
	}

	// Log restoration for audit trail
	if g.config.LogViolations {
		g.logger.Info("PII placeholders restored",
			"placeholders_count", len(mapping.Placeholders),
			"output_length", len(output),
			"restored_length", len(restored))
	}

	return restored, nil
}

// ValidateArgs validates tool call arguments for PII violations without modifying the call
// Returns list of violations found
func (g *Governor) ValidateArgs(ctx context.Context, toolName string, args map[string]interface{}) ([]interfaces.PIIViolation, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	violations := []interfaces.PIIViolation{}

	for key, value := range args {
		strVal, ok := value.(string)
		if !ok {
			continue
		}

		// Detect PII without modifying
		_, redactions := g.scrubber.Scrub(strVal)
		for _, redaction := range redactions {
			// Determine severity based on pattern type
			severity := g.determineSeverity(redaction.Type)

			violation := interfaces.PIIViolation{
				Field:       key,
				PatternType: redaction.Type,
				Message:     fmt.Sprintf("PII detected: %s in field '%s'", redaction.Description, key),
				Severity:    severity,
			}

			violations = append(violations, violation)
		}
	}

	// Log validation results
	if len(violations) > 0 && g.config.LogViolations {
		severitySummary := make(map[string]int)
		for _, v := range violations {
			severitySummary[v.Severity]++
		}
		g.logger.Warn("PII validation violations detected",
			"tool", toolName,
			"total_violations", len(violations),
			"severity_breakdown", severitySummary)
	}

	return violations, nil
}

// computeHash creates a SHA256 hash for a string (used for Shadow Mapping)
func (g *Governor) computeHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	// Use first 8 hex chars for shorter placeholder
	return hex.EncodeToString(hash[:])[:8]
}

// maskForLog creates a masked version of PII for safe logging
func (g *Governor) maskForLog(s string) string {
	// Show first 2 chars and last 2 chars, mask the rest
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// getPatternTypes extracts pattern types from violations list
func (g *Governor) getPatternTypes(violations []pii.Redaction) []string {
	types := make([]string, 0, len(violations))
	seen := make(map[string]bool)

	for _, v := range violations {
		if !seen[v.Type] {
			seen[v.Type] = true
			types = append(types, v.Type)
		}
	}

	return types
}

// determineSeverity assigns severity level based on PII pattern type
func (g *Governor) determineSeverity(patternType string) string {
	// Critical patterns
	switch patternType {
	case "credit_card", "aws_secret", "aws_key_id", "api_key_sk", "api_key_pk", "api_key_ai":
		return "critical"
	case "ssn", "github_token":
		return "high"
	case "email", "phone", "ip_address", "bearer_token", "token", "secret", "password":
		return "medium"
	default:
		return "low"
	}
}
