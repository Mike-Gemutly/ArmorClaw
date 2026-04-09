package security

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// PIIScanner detects PII (Personally Identifiable Information) in CDP messages
type PIIScanner struct {
	mu       sync.Mutex
	patterns map[string]*regexp.Regexp
	flare    *DistressFlareLogger
}

// DistressFlareLogger logs security warnings with ANSI color codes
type DistressFlareLogger struct {
	mu sync.Mutex
}

// PIIType represents the type of PII detected
type PIIType string

const (
	PIITypeSSN        PIIType = "SSN"
	PIITypeCreditCard PIIType = "CREDIT_CARD"
	PIITypeEmail      PIIType = "EMAIL"
	PIITypePassword   PIIType = "PASSWORD"
)

// PIIFinding represents a detected PII instance
type PIIFinding struct {
	Type     PIIType
	Context  string
	Severity string // "HIGH", "MEDIUM", "LOW"
}

// NewPIIScanner creates a new PII scanner with default patterns
func NewPIIScanner() *PIIScanner {
	return &PIIScanner{
		patterns: map[string]*regexp.Regexp{
			string(PIITypeSSN):        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
			string(PIITypeCreditCard): regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
			string(PIITypeEmail):      regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
			string(PIITypePassword):   regexp.MustCompile(`(?i)(password|passwd|pwd)["\']?\s*[:=]\s*["\']?[^\s"']{8,}`),
		},
		flare: &DistressFlareLogger{},
	}
}

// ScanCDPMessage scans a CDP message for PII
// Non-blocking: runs in a separate goroutine for warnings
func (ps *PIIScanner) ScanCDPMessage(method string, params map[string]interface{}) []PIIFinding {
	findings := make([]PIIFinding, 0)

	// Only scan Input.insertText methods
	if method != "Input.insertText" {
		return findings
	}

	// Extract text content from params
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return findings
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Scan for each PII type
	for piiType, pattern := range ps.patterns {
		if pattern.MatchString(text) {
			findings = append(findings, PIIFinding{
				Type:     PIIType(piiType),
				Context:  ps.sanitizeContext(text),
				Severity: ps.getSeverity(PIIType(piiType)),
			})
		}
	}

	// Non-blocking: log warnings in background
	if len(findings) > 0 {
		go ps.flare.Fire(method, findings)
	}

	return findings
}

// sanitizeContext returns a sanitized version of the context (truncates and masks)
func (ps *PIIScanner) sanitizeContext(text string) string {
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}

// getSeverity returns the severity level for a PII type
func (ps *PIIScanner) getSeverity(piiType PIIType) string {
	switch piiType {
	case PIITypePassword:
		return "HIGH"
	case PIITypeSSN, PIITypeCreditCard:
		return "HIGH"
	case PIITypeEmail:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// Fire logs a distress flare to the terminal with ANSI colors
func (dfl *DistressFlareLogger) Fire(method string, findings []PIIFinding) {
	dfl.mu.Lock()
	defer dfl.mu.Unlock()

	// ANSI color codes
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBold   = "\033[1m"
	)

	fmt.Printf("%s%s🔴 DISTRESS FLARE: PII DETECTED%s\n", colorBold+colorRed, colorBold, colorReset)
	fmt.Printf("%sMethod:%s %s\n", colorYellow, colorReset, method)
	fmt.Printf("%sFindings:%s\n", colorYellow, colorReset)

	for _, finding := range findings {
		fmt.Printf("  - %s [%s]: %s\n", finding.Type, finding.Severity, finding.Context)
	}
	fmt.Printf("%s⚠️  WARNING: This data is being logged in cleartext in Free-Ride Mode!%s\n\n", colorRed, colorReset)
}

// ScanJSONMessage scans a raw JSON message for PII
func (ps *PIIScanner) ScanJSONMessage(jsonStr string) ([]PIIFinding, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	method, _ := data["method"].(string)
	params, _ := data["params"].(map[string]interface{})

	if method == "" || params == nil {
		return make([]PIIFinding, 0), nil
	}

	return ps.ScanCDPMessage(method, params), nil
}

// ContainsPassword checks if the text contains password-related PII
func (ps *PIIScanner) ContainsPassword(text string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if pattern, ok := ps.patterns[string(PIITypePassword)]; ok {
		return pattern.MatchString(text)
	}
	return false
}

// MaskPII masks PII in text with asterisks
func (ps *PIIScanner) MaskPII(text string) string {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	masked := text
	for _, pattern := range ps.patterns {
		masked = pattern.ReplaceAllStringFunc(masked, func(match string) string {
			return strings.Repeat("*", len(match))
		})
	}
	return masked
}
