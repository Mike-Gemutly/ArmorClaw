package browser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SupportedSchemaVersion is the only NavChart version currently accepted.
const SupportedSchemaVersion = 1

// placeholderRe matches the {{field_name}} placeholder format.
var placeholderRe = regexp.MustCompile(`^\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\}$`)

// ValidationIssue describes a single validation problem.
type ValidationIssue struct {
	Field   string // dotted path, e.g. "action_map.action_1.value"
	Message string
}

func (i ValidationIssue) Error() string {
	return fmt.Sprintf("%s: %s", i.Field, i.Message)
}

// ValidationError collects one or more validation issues.
type ValidationError struct {
	Issues []ValidationIssue
}

func (ve *ValidationError) Error() string {
	if len(ve.Issues) == 0 {
		return "validation failed"
	}
	msgs := make([]string, len(ve.Issues))
	for i, iss := range ve.Issues {
		msgs[i] = iss.Error()
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(msgs, "; "))
}

func addIssue(issues *[]ValidationIssue, field, msg string) {
	*issues = append(*issues, ValidationIssue{Field: field, Message: msg})
}

// DomainPolicy checks whether a target domain is allowed for replay.
// Implementations may source the allow-list from config, keystore, etc.
type DomainPolicy interface {
	IsAllowed(domain string) bool
}

// AllowAllDomains is a DomainPolicy that permits every domain (used when no policy is configured).
type AllowAllDomains struct{}

func (AllowAllDomains) IsAllowed(string) bool { return true }

// ChartValidator validates NavChart instances before storage and before replay.
type ChartValidator struct {
	DomainPolicy DomainPolicy
}

// NewChartValidator creates a ChartValidator with the given domain policy.
// If policy is nil, AllowAllDomains is used.
func NewChartValidator(policy DomainPolicy) *ChartValidator {
	if policy == nil {
		policy = AllowAllDomains{}
	}
	return &ChartValidator{DomainPolicy: policy}
}

// ValidateForStorage checks that a chart is safe to persist.
//
// Rules:
//   - Schema version must be supported (currently only version 1).
//   - target_domain must be a valid URL with a scheme and a host containing a dot.
//   - action_map must not be empty.
//   - click and input actions must have at least a primary_css selector.
//   - No step value may contain raw PII (SSN, CC, email, password in raw form).
//     Values should use the {{placeholder}} format for sensitive fields.
func (cv *ChartValidator) ValidateForStorage(chart NavChart) error {
	var issues []ValidationIssue

	// 1. Schema version
	if chart.Version != SupportedSchemaVersion {
		addIssue(&issues, "version",
			fmt.Sprintf("unsupported schema version %d (only version %d is supported)", chart.Version, SupportedSchemaVersion))
	}

	// 2. Target domain
	if chart.TargetDomain == "" {
		addIssue(&issues, "target_domain", "must not be empty")
	} else if !isValidTargetDomain(chart.TargetDomain) {
		addIssue(&issues, "target_domain",
			fmt.Sprintf("must be a valid URL with a scheme and a host containing a dot, got %q", chart.TargetDomain))
	}

	// 3. Action map not empty
	if len(chart.ActionMap) == 0 {
		addIssue(&issues, "action_map", "must contain at least one action")
	}

	// 4. Per-action checks
	for key, action := range chart.ActionMap {
		prefix := "action_map." + key

		// 4a. Click/input must have primary_css selector
		if action.ActionType == ActionClick || action.ActionType == ActionInput {
			if action.Selector == nil || action.Selector.PrimaryCSS == "" {
				addIssue(&issues, prefix+".selector.primary_css",
					fmt.Sprintf("action_type %q requires a primary_css selector", action.ActionType))
			}
		}

		// 4b. PII check on values
		if action.Value != "" {
			if findings := DetectPII(action.Value); len(findings) > 0 {
				types := make([]string, len(findings))
				for i, f := range findings {
					types[i] = f.Type
				}
				addIssue(&issues, prefix+".value",
					fmt.Sprintf("contains raw PII (%s); use {{placeholder}} format instead", strings.Join(types, ", ")))
			}
		}

		// 4c. PII check on URLs (navigate actions)
		if action.URL != "" {
			if findings := DetectPII(action.URL); len(findings) > 0 {
				types := make([]string, len(findings))
				for i, f := range findings {
					types[i] = f.Type
				}
				addIssue(&issues, prefix+".url",
					fmt.Sprintf("contains raw PII (%s); use {{placeholder}} format instead", strings.Join(types, ", ")))
			}
		}

		// 4d. PII check on selectors (edge case: selectors could contain PII)
		if action.Selector != nil {
			for _, sel := range []struct {
				val   string
				field string
			}{
				{action.Selector.PrimaryCSS, prefix + ".selector.primary_css"},
				{action.Selector.SecondaryXPath, prefix + ".selector.secondary_xpath"},
				{action.Selector.FallbackJS, prefix + ".selector.fallback_js"},
			} {
				if sel.val != "" {
					if findings := DetectPII(sel.val); len(findings) > 0 {
						types := make([]string, len(findings))
						for i, f := range findings {
							types[i] = f.Type
						}
						addIssue(&issues, sel.field,
							fmt.Sprintf("contains raw PII (%s); use {{placeholder}} format instead", strings.Join(types, ", ")))
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// ValidateForReplay checks that a stored chart is safe to execute with the given PII map.
//
// All storage validations are performed, PLUS:
//   - Every placeholder in action values must have a corresponding entry in availablePII.
//   - Target domain must be allowed by the configured DomainPolicy.
//   - Actions that typically require approval (payment, login) are flagged.
func (cv *ChartValidator) ValidateForReplay(chart NavChart, availablePII map[string]string) error {
	// Run all storage validations first
	if err := cv.ValidateForStorage(chart); err != nil {
		return err
	}

	var issues []ValidationIssue

	// 1. Domain policy check
	if !cv.DomainPolicy.IsAllowed(chart.TargetDomain) {
		addIssue(&issues, "target_domain",
			fmt.Sprintf("domain %q is not in the allowed list", chart.TargetDomain))
	}

	// 2. Placeholder resolution
	for key, action := range chart.ActionMap {
		prefix := "action_map." + key

		// Check value placeholders
		if unresolved := findUnresolvedPlaceholders(action.Value, availablePII); len(unresolved) > 0 {
			addIssue(&issues, prefix+".value",
				fmt.Sprintf("unresolved placeholder(s): %s", strings.Join(unresolved, ", ")))
		}

		// Check URL placeholders
		if unresolved := findUnresolvedPlaceholders(action.URL, availablePII); len(unresolved) > 0 {
			addIssue(&issues, prefix+".url",
				fmt.Sprintf("unresolved placeholder(s): %s", strings.Join(unresolved, ", ")))
		}

		// Check selector placeholders
		if action.Selector != nil {
			for _, sel := range []struct {
				val   string
				field string
			}{
				{action.Selector.PrimaryCSS, prefix + ".selector.primary_css"},
				{action.Selector.SecondaryXPath, prefix + ".selector.secondary_xpath"},
				{action.Selector.FallbackJS, prefix + ".selector.fallback_js"},
			} {
				if unresolved := findUnresolvedPlaceholders(sel.val, availablePII); len(unresolved) > 0 {
					addIssue(&issues, sel.field,
						fmt.Sprintf("unresolved placeholder(s): %s", strings.Join(unresolved, ", ")))
				}
			}
		}
	}

	// 3. Flag actions requiring approval (informational — not a hard rejection)
	// These are recorded as issues but with a distinct prefix "APPROVAL_REQUIRED"
	// so callers can separate hard errors from approval flags.
	for key, action := range chart.ActionMap {
		if requiresApproval(action) {
			addIssue(&issues, "action_map."+key,
				fmt.Sprintf("action_type %q with value may require approval", action.ActionType))
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// RequiresApproval checks if a replay validation error includes approval-required issues.
func RequiresApproval(err error) []ValidationIssue {
	if err == nil {
		return nil
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		return nil
	}
	var approval []ValidationIssue
	for _, iss := range ve.Issues {
		if strings.Contains(iss.Message, "may require approval") {
			approval = append(approval, iss)
		}
	}
	return approval
}

// isValidTargetDomain checks that a domain string looks like a real URL with scheme and dot.
func isValidTargetDomain(domain string) bool {
	u, err := url.Parse(domain)
	if err != nil {
		return false
	}
	if u.Scheme == "" || u.Host == "" {
		return false
	}
	host := u.Hostname()
	// Must contain at least one dot (reject "localhost", IPs are ok with dots)
	return strings.Contains(host, ".")
}

// placeholderRefRe finds all {{name}} tokens in a string.
var placeholderRefRe = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

// findUnresolvedPlaceholders returns placeholder names present in text but missing from availablePII.
func findUnresolvedPlaceholders(text string, availablePII map[string]string) []string {
	if text == "" {
		return nil
	}
	matches := placeholderRefRe.FindAllStringSubmatch(text, -1)
	seen := map[string]bool{}
	var unresolved []string
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		if _, ok := availablePII[name]; !ok {
			unresolved = append(unresolved, "{{"+name+"}}")
		}
	}
	return unresolved
}

// requiresApproval returns true for actions that typically need human approval
// before execution (e.g., payments, login submissions).
func requiresApproval(action ChartAction) bool {
	switch action.ActionType {
	case ActionInput:
		if action.Selector == nil {
			return false
		}
		sel := strings.ToLower(action.Selector.PrimaryCSS)
		// Heuristic: input into payment/login-related fields requires approval
		for _, keyword := range []string{"card", "credit", "payment", "password", "ssn", "passwd", "login"} {
			if strings.Contains(sel, keyword) {
				return true
			}
		}
		return false
	case ActionClick:
		if action.Selector == nil {
			return false
		}
		sel := strings.ToLower(action.Selector.PrimaryCSS)
		for _, keyword := range []string{"submit", "pay", "checkout", "purchase", "confirm"} {
			if strings.Contains(sel, keyword) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
