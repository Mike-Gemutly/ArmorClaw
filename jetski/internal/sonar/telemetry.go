package sonar

import (
	"fmt"
	"net/url"
	"time"
)

// Selector represents a 3-tier fallback matrix for DOM element location
type Selector struct {
	PrimaryCSS     string `json:"primary_css"`
	SecondaryXPath string `json:"secondary_xpath,omitempty"`
	FallbackJS     string `json:"fallback_js,omitempty"`
	Tier           int    `json:"tier"` // 1=primary, 2=secondary, 3=fallback
}

// SelectorMetrics tracks selector interaction counts across tiers
type SelectorMetrics struct {
	PrimaryCount   int
	SecondaryCount int
	FallbackCount  int
	Total          int
}

// WreckageReport is the "Black Box" flight recorder for browser failures
// Contains all data needed to reconstruct the failure scene
type WreckageReport struct {
	Timestamp         time.Time  `json:"timestamp"`
	TargetURI         string     `json:"target_uri"`      // Full URI (https://github.com)
	TargetHostname    string     `json:"target_hostname"` // Extracted hostname (github.com)
	DOMSnapshot       string     `json:"dom_snapshot"`
	FailedSelector    Selector   `json:"failed_selector"`
	CDPHistory        []CDPFrame `json:"cdp_history"`
	SelectorHealth    float64    `json:"selector_health_score"`
	TotalInteractions int        `json:"total_interactions"`
	SessionID         string     `json:"session_id"`
}

// CalculateHealthScore computes the selector health score using the formula:
// H = (S_primary + 0.5*S_secondary + 0.1*S_fallback) / TotalInteractions
//
// Interpretation:
// - 1.0 = Green Water (pristine navigation)
// - 0.5 = Choppy Seas (significant drift, CSS shifted)
// - < 0.2 = Shipwreck (Red Alert, Nav-Chart obsolete)
//
// CRITICAL: TotalInteractions is incremented even on total failure (S=0)
// This ensures the health score reflects the "Death Spiral" of failing selectors
func CalculateHealthScore(metrics SelectorMetrics) float64 {
	if metrics.Total == 0 {
		return 1.0 // Perfect health by default (no interactions yet)
	}

	// Weighted sum: primary (1.0) + secondary (0.5) + fallback (0.1)
	weightedSum := float64(metrics.PrimaryCount) +
		0.5*float64(metrics.SecondaryCount) +
		0.1*float64(metrics.FallbackCount)

	health := weightedSum / float64(metrics.Total)

	// Clamp to [0, 1] range to handle any edge cases
	if health > 1.0 {
		return 1.0
	}
	if health < 0.0 {
		return 0.0
	}

	return health
}

// CalculateAndSetHealthScore computes the health score from metrics and updates the report
func (wr *WreckageReport) CalculateAndSetHealthScore(metrics SelectorMetrics) {
	wr.SelectorHealth = CalculateHealthScore(metrics)
	wr.TotalInteractions = metrics.Total
}

// NewWreckageReport creates a new WreckageReport with the given parameters
// Automatically extracts hostname from the full URI
func NewWreckageReport(sessionID, targetURI string, buffer *CircularBuffer, failedSelector Selector) (*WreckageReport, error) {
	hostname, err := extractHostname(targetURI)
	if err != nil {
		return nil, fmt.Errorf("failed to extract hostname from URI: %w", err)
	}

	return &WreckageReport{
		Timestamp:      time.Now(),
		TargetURI:      targetURI,
		TargetHostname: hostname,
		FailedSelector: failedSelector,
		CDPHistory:     buffer.GetAll(),
		SessionID:      sessionID,
	}, nil
}

// extractHostname extracts and returns the hostname from a full URI
func extractHostname(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	// Validate that the URI has a scheme and host
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid URI: missing scheme or host")
	}
	return parsed.Hostname(), nil
}

// GetHealthStatus returns a human-readable health status
func (wr *WreckageReport) GetHealthStatus() string {
	switch {
	case wr.SelectorHealth >= 0.8:
		return "Green Water (pristine navigation)"
	case wr.SelectorHealth >= 0.5:
		return "Choppy Seas (moderate drift)"
	case wr.SelectorHealth >= 0.2:
		return "Rough Waters (significant drift)"
	default:
		return "Shipwreck (Red Alert, Nav-Chart obsolete)"
	}
}
