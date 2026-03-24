// Package agent provides PII interceptor middleware for detecting and replacing vault patterns
package agent

import (
	"regexp"
	"sync"
)

// PIIMapping stores the mapping between redacted placeholders and original vault patterns
type PIIMapping struct {
	mu     sync.RWMutex
	mapped map[string]string
}

// NewPIIMapping creates a new PII mapping
func NewPIIMapping() *PIIMapping {
	return &PIIMapping{
		mapped: make(map[string]string),
	}
}

// Count returns the number of mappings
func (m *PIIMapping) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.mapped)
}

// Get retrieves the original pattern for a redacted placeholder
func (m *PIIMapping) Get(redacted string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.mapped[redacted]
	return val, ok
}

// set stores a mapping (internal use)
func (m *PIIMapping) set(redacted, original string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mapped[redacted] = original
}

// PIIInterceptor detects and replaces vault patterns in agent prompts
type PIIInterceptor struct {
	pattern *regexp.Regexp
	mu      sync.RWMutex
}

// vaultPattern matches {{VAULT:hash}} where hash is lowercase hexadecimal
var vaultPattern = regexp.MustCompile(`\{\{VAULT:([a-f0-9]+)\}\}`)

// NewPIIInterceptor creates a new PII interceptor
func NewPIIInterceptor() *PIIInterceptor {
	return &PIIInterceptor{
		pattern: vaultPattern,
	}
}

// Intercept scans a prompt for vault patterns, replaces them with redacted placeholders,
// and returns the modified prompt along with the mapping of placeholders to original patterns
func (i *PIIInterceptor) Intercept(prompt string) (string, *PIIMapping, error) {
	mapping := NewPIIMapping()

	matches := i.pattern.FindAllStringSubmatch(prompt, -1)
	if len(matches) == 0 {
		return prompt, mapping, nil
	}

	modified := prompt
	seenHashes := make(map[string]bool)

	for _, match := range matches {
		fullMatch := match[0]
		hash := match[1]

		if seenHashes[hash] {
			continue
		}
		seenHashes[hash] = true

		redacted := "[REDACTED:" + hash + "]"
		mapping.set(redacted, fullMatch)

		hashPattern := regexp.MustCompile(`\{\{VAULT:` + regexp.QuoteMeta(hash) + `\}\}`)
		modified = hashPattern.ReplaceAllString(modified, redacted)
	}

	return modified, mapping, nil
}
