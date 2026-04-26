package browser

import (
	"context"
	"fmt"
	"strings"
)

// ScanPIIFinding carries chart-level provenance for a PII match found in a stored NavChart.
type ScanPIIFinding struct {
	ChartID        string `json:"chart_id"`
	ActionKey      string `json:"action_key"`
	Field          string `json:"field"`
	MatchedPattern string `json:"matched_pattern"`
	Severity       string `json:"severity"`
}

func severityForType(piiType string) string {
	switch piiType {
	case "SSN", "CREDIT_CARD", "PASSWORD":
		return "high"
	case "EMAIL":
		return "medium"
	default:
		return "low"
	}
}

var KnownDomains = []string{
	"https://unknown",
}

// ScanChartsForPII iterates stored charts for the given domains (or KnownDomains
// if empty) and scans every action value, URL, and selector for raw PII.
// Detection only — does not auto-fix.
func ScanChartsForPII(ctx context.Context, store ChartStore, domains []string) ([]ScanPIIFinding, error) {
	if len(domains) == 0 {
		domains = KnownDomains
	}

	var allFindings []ScanPIIFinding

	for _, domain := range domains {
		records, err := store.FindForDomain(ctx, domain, 1000)
		if err != nil {
			return allFindings, fmt.Errorf("scanning domain %q: %w", domain, err)
		}

		for _, rec := range records {
			findings := scanRecordForPII(rec)
			allFindings = append(allFindings, findings...)
		}
	}

	return allFindings, nil
}

// ScanSingleChart scans one ChartRecord for PII.
func ScanSingleChart(rec ChartRecord) []ScanPIIFinding {
	return scanRecordForPII(rec)
}

func scanRecordForPII(rec ChartRecord) []ScanPIIFinding {
	var findings []ScanPIIFinding

	if rec.NavChart != nil {
		return scanNavChart(rec.ChartID, rec.NavChart)
	}

	findings = append(findings, scanField(rec.ChartID, "record", "title", rec.Title)...)

	if rec.Steps != "" {
		findings = append(findings, scanField(rec.ChartID, "record", "steps", rec.Steps)...)
	}

	return findings
}

func scanNavChart(chartID string, chart *NavChart) []ScanPIIFinding {
	var findings []ScanPIIFinding

	for actionKey, action := range chart.ActionMap {
		if action.Value != "" {
			findings = append(findings, scanField(chartID, actionKey, "value", action.Value)...)
		}

		if action.URL != "" {
			findings = append(findings, scanField(chartID, actionKey, "url", action.URL)...)
		}

		if action.Selector != nil {
			if action.Selector.PrimaryCSS != "" {
				findings = append(findings, scanField(chartID, actionKey, "selector_css", action.Selector.PrimaryCSS)...)
			}
			if action.Selector.SecondaryXPath != "" {
				findings = append(findings, scanField(chartID, actionKey, "selector_xpath", action.Selector.SecondaryXPath)...)
			}
			if action.Selector.FallbackJS != "" {
				findings = append(findings, scanField(chartID, actionKey, "selector_js", action.Selector.FallbackJS)...)
			}
		}

		if action.Assertion != nil && action.Assertion.Expected != nil {
			expStr := fmt.Sprintf("%v", action.Assertion.Expected)
			if expStr != "" {
				findings = append(findings, scanField(chartID, actionKey, "assertion_expected", expStr)...)
			}
		}
	}

	return findings
}

func scanField(chartID, actionKey, field, text string) []ScanPIIFinding {
	if isPlaceholder(text) {
		return nil
	}

	var findings []ScanPIIFinding
	for _, f := range DetectPII(text) {
		findings = append(findings, ScanPIIFinding{
			ChartID:        chartID,
			ActionKey:      actionKey,
			Field:          field,
			MatchedPattern: f.Type,
			Severity:       severityForType(f.Type),
		})
	}
	return findings
}

func isPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, ph := range []string{
		PlaceholderSSN, PlaceholderCreditCard, PlaceholderEmail, PlaceholderPassword,
	} {
		if lower == strings.ToLower(ph) {
			return true
		}
	}
	return false
}
