package browser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"
)

// CDPFrame represents a single Chrome DevTools Protocol message frame.
// Bridge-local type matching jetski/internal/sonar.CDPFrame.
type CDPFrame struct {
	Timestamp time.Time       `json:"timestamp"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	SessionID string          `json:"session_id"`
}

const (
	PlaceholderSSN        = "{{ssn}}"
	PlaceholderCreditCard = "{{credit_card}}"
	PlaceholderEmail      = "{{email}}"
	PlaceholderPassword   = "{{password}}"
)

// PIIFinding represents a detected PII instance.
type PIIFinding struct {
	Type string
	Text string
}

type piiRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Placeholder string
}

// piiRules are the PII detection rules matching jetski/internal/security/pii_scanner.go exactly.
// Order matters: SSN first (most specific format), then credit card, email, password.
var piiRules = []piiRule{
	{
		Name:        "SSN",
		Pattern:     regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Placeholder: PlaceholderSSN,
	},
	{
		Name:        "CREDIT_CARD",
		Pattern:     regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
		Placeholder: PlaceholderCreditCard,
	},
	{
		Name:        "EMAIL",
		Pattern:     regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		Placeholder: PlaceholderEmail,
	},
	{
		Name:        "PASSWORD",
		Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)["']?\s*[:=]\s*["']?[^\s"']{8,}`),
		Placeholder: PlaceholderPassword,
	},
}

var (
	navigateMethods = map[string]bool{
		"Page.navigate":       true,
		"Page.frameNavigated": true,
	}
	interactionMethods = map[string]bool{
		"Input.insertText":         true,
		"Input.dispatchMouseEvent": true,
	}
	selectorMethods = map[string]bool{
		"DOM.querySelector": true,
		"DOM.getDocument":   true,
		"Runtime.evaluate":  true,
	}
	relevantMethods map[string]bool
)

func init() {
	relevantMethods = make(map[string]bool, len(navigateMethods)+len(interactionMethods)+len(selectorMethods))
	for m := range navigateMethods {
		relevantMethods[m] = true
	}
	for m := range interactionMethods {
		relevantMethods[m] = true
	}
	for m := range selectorMethods {
		relevantMethods[m] = true
	}
}

type frameGroup struct {
	primary   CDPFrame
	selectors []CDPFrame
}

// Normalize converts raw CDP frames into a NavChart through a multi-step pipeline:
// Filter → Group → Detect PII → Replace → Extract Selectors → Attach Metadata.
func Normalize(frames []CDPFrame, sessionID string) (*NavChart, error) {
	if len(frames) == 0 {
		return emptyChart(sessionID), nil
	}

	// Step 1: Filter — remove low-value CDP noise
	filtered := filterFrames(frames)
	if len(filtered) == 0 {
		return emptyChart(sessionID), nil
	}

	// Step 2: Group — consecutive frames into semantic steps
	groups := groupFrames(filtered)

	// Steps 3-6: compile groups, detect/replace PII, extract selectors, attach metadata
	actionMap := make(map[string]ChartAction, len(groups))
	targetDomain := "https://unknown"

	for i, g := range groups {
		action := compileGroup(g)

		// Step 3: Detect PII in input values
		if action.Value != "" {
			if findings := DetectPII(action.Value); len(findings) > 0 {
				// Step 4: Replace PII with placeholders
				action.Value = ReplacePII(action.Value)
			}
		}

		// Extract target domain from first navigate action
		if action.ActionType == ActionNavigate && action.URL != "" {
			if d := extractDomain(action.URL); d != "" {
				targetDomain = d
			}
		}

		actionMap[fmt.Sprintf("action_%d", i+1)] = action
	}

	return &NavChart{
		Version:      1,
		TargetDomain: targetDomain,
		Metadata: ChartMetadata{
			GeneratedBy: "bridge-normalizer",
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			SessionID:   sessionID,
		},
		ActionMap: actionMap,
	}, nil
}

func emptyChart(sessionID string) *NavChart {
	return &NavChart{
		Version: 1,
		Metadata: ChartMetadata{
			GeneratedBy: "bridge-normalizer",
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			SessionID:   sessionID,
		},
		ActionMap: map[string]ChartAction{},
	}
}

func filterFrames(frames []CDPFrame) []CDPFrame {
	result := make([]CDPFrame, 0, len(frames))
	for _, f := range frames {
		if relevantMethods[f.Method] {
			result = append(result, f)
		}
	}
	return result
}

func groupFrames(frames []CDPFrame) []frameGroup {
	var groups []frameGroup
	var pendingSelectors []CDPFrame

	for _, frame := range frames {
		if navigateMethods[frame.Method] || interactionMethods[frame.Method] {
			g := frameGroup{
				primary:   frame,
				selectors: pendingSelectors,
			}
			pendingSelectors = nil
			groups = append(groups, g)
		} else if selectorMethods[frame.Method] {
			pendingSelectors = append(pendingSelectors, frame)
		}
	}

	return groups
}

func compileGroup(g frameGroup) ChartAction {
	switch {
	case navigateMethods[g.primary.Method]:
		return compileNavigate(g)
	case g.primary.Method == "Input.insertText":
		return compileInput(g)
	case g.primary.Method == "Input.dispatchMouseEvent":
		return compileClick(g)
	default:
		return ChartAction{ActionType: ActionWait}
	}
}

func compileNavigate(g frameGroup) ChartAction {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(g.primary.Params, &params); err != nil || params.URL == "" {
		return ChartAction{ActionType: ActionNavigate}
	}
	return ChartAction{
		ActionType: ActionNavigate,
		URL:        params.URL,
		PostActionWait: &WaitCondition{
			Type:    "waitForSelector",
			Timeout: 5000,
		},
	}
}

func compileInput(g frameGroup) ChartAction {
	var params struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(g.primary.Params, &params); err != nil {
		return ChartAction{ActionType: ActionInput}
	}

	action := ChartAction{
		ActionType: ActionInput,
		Value:      params.Text,
	}

	if sel := extractSelectorFromContext(g.selectors); sel != nil {
		action.Selector = sel
	}

	return action
}

func compileClick(g frameGroup) ChartAction {
	action := ChartAction{
		ActionType: ActionClick,
	}

	if sel := extractSelectorFromContext(g.selectors); sel != nil {
		action.Selector = sel
	}

	return action
}

// CSS from DOM.querySelector, JS from Runtime.evaluate.
func extractSelectorFromContext(frames []CDPFrame) *ChartSelector {
	if len(frames) == 0 {
		return nil
	}

	var primaryCSS, xpath, jsFallback string

	for _, f := range frames {
		switch f.Method {
		case "DOM.querySelector":
			var p struct {
				Selector string `json:"selector"`
			}
			if json.Unmarshal(f.Params, &p) == nil && p.Selector != "" {
				primaryCSS = p.Selector
			}
		case "DOM.getDocument":
			var p struct {
				Depth    int    `json:"depth"`
				Selector string `json:"selector,omitempty"`
			}
			if json.Unmarshal(f.Params, &p) == nil && p.Selector != "" {
				if xpath == "" {
					xpath = "//*[@selector='" + p.Selector + "']"
				}
			}
		case "Runtime.evaluate":
			var p struct {
				Expression string `json:"expression"`
			}
			if json.Unmarshal(f.Params, &p) == nil && p.Expression != "" {
				jsFallback = p.Expression
			}
		}
	}

	if primaryCSS == "" && xpath == "" && jsFallback == "" {
		return nil
	}

	return &ChartSelector{
		PrimaryCSS:     primaryCSS,
		SecondaryXPath: xpath,
		FallbackJS:     jsFallback,
	}
}

// DetectPII scans text for PII patterns. Returns all findings.
// Public helper for reuse across the bridge.
func DetectPII(text string) []PIIFinding {
	var findings []PIIFinding
	for _, rule := range piiRules {
		if rule.Pattern.MatchString(text) {
			findings = append(findings, PIIFinding{
				Type: rule.Name,
				Text: rule.Pattern.FindString(text),
			})
		}
	}
	return findings
}

// ReplacePII replaces detected PII values with placeholder tokens.
// Public helper for reuse across the bridge.
func ReplacePII(text string) string {
	result := text
	for _, rule := range piiRules {
		result = rule.Pattern.ReplaceAllString(result, rule.Placeholder)
	}
	return result
}

func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if !hasScheme(rawURL) {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func hasScheme(s string) bool {
	for i, c := range s {
		if c == ':' {
			return i > 0
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.') {
			return false
		}
	}
	return false
}
