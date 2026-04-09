package sonar

import (
	"testing"
)

// TestCalculateHealthScore_PerfectHealth tests health score with all primary selectors
// Expected: H = 1.0 (Green Water)
func TestCalculateHealthScore_PerfectHealth(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   100,
		SecondaryCount: 0,
		FallbackCount:  0,
		Total:          100,
	}

	score := CalculateHealthScore(metrics)

	expected := 1.0
	if score != expected {
		t.Errorf("Perfect health: expected score %.2f, got %.2f", expected, score)
	}

	if score < 0.8 {
		t.Errorf("Perfect health should be Green Water (>= 0.8), got %.2f", score)
	}
}

// TestCalculateHealthScore_MixedHealth tests health score with mixed selectors
// 50% primary, 50% secondary → Expected: H = 0.75
func TestCalculateHealthScore_MixedHealth(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   50,
		SecondaryCount: 50,
		FallbackCount:  0,
		Total:          100,
	}

	score := CalculateHealthScore(metrics)

	// Formula: (50*1.0 + 50*0.5) / 100 = (50 + 25) / 100 = 0.75
	expected := 0.75
	if score != expected {
		t.Errorf("Mixed health: expected score %.2f, got %.2f", expected, score)
	}
}

// TestCalculateHealthScore_PoorHealth tests health score with all fallback selectors
// Expected: H = 0.1 (Choppy Seas approaching Shipwreck)
func TestCalculateHealthScore_PoorHealth(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   0,
		SecondaryCount: 0,
		FallbackCount:  100,
		Total:          100,
	}

	score := CalculateHealthScore(metrics)

	// Formula: (100*0.1) / 100 = 0.1
	expected := 0.1
	if score != expected {
		t.Errorf("Poor health: expected score %.2f, got %.2f", expected, score)
	}

	if score >= 0.2 {
		t.Errorf("Poor health should be < 0.2 (Shipwreck territory), got %.2f", score)
	}
}

// TestCalculateHealthScore_DeathSpiral tests health score with total failure
// All selectors failed (S=0) → Expected: H = 0.0
// CRITICAL: TotalInteractions is incremented even on total failure
func TestCalculateHealthScore_DeathSpiral(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   0,
		SecondaryCount: 0,
		FallbackCount:  0,
		Total:          100, // Still incremented even though all failed
	}

	score := CalculateHealthScore(metrics)

	// Formula: (0*1.0 + 0*0.5 + 0*0.1) / 100 = 0.0
	expected := 0.0
	if score != expected {
		t.Errorf("Death spiral: expected score %.2f, got %.2f", expected, score)
	}

	if score >= 0.2 {
		t.Errorf("Death spiral should be < 0.2 (Shipwreck), got %.2f", score)
	}

	// Verify TotalInteractions was incremented
	if metrics.Total != 100 {
		t.Errorf("TotalInteractions should be incremented even on total failure, got %d", metrics.Total)
	}
}

// TestCalculateHealthScore_ZeroInteractions tests health score with no interactions
// Expected: H = 1.0 (perfect by default)
func TestCalculateHealthScore_ZeroInteractions(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   0,
		SecondaryCount: 0,
		FallbackCount:  0,
		Total:          0,
	}

	score := CalculateHealthScore(metrics)

	expected := 1.0
	if score != expected {
		t.Errorf("Zero interactions: expected score %.2f, got %.2f", expected, score)
	}
}

// TestCalculateHealthScore_SalesforceScenario tests the Salesforce Lightning scenario from the plan
// 10 tier-1, 0 tier-2, 40 tier-3 → Expected: H = 0.28
func TestCalculateHealthScore_SalesforceScenario(t *testing.T) {
	metrics := SelectorMetrics{
		PrimaryCount:   10,
		SecondaryCount: 0,
		FallbackCount:  40,
		Total:          50,
	}

	score := CalculateHealthScore(metrics)

	// Formula: (10*1.0 + 0*0.5 + 40*0.1) / 50 = (10 + 0 + 4) / 50 = 14/50 = 0.28
	expected := 0.28
	if score != expected {
		t.Errorf("Salesforce scenario: expected score %.2f, got %.2f", expected, score)
	}

	// Should be in "Rough Waters" territory
	if score >= 0.5 {
		t.Errorf("Salesforce scenario should be < 0.5 (Rough Waters), got %.2f", score)
	}
}

// TestWreckageReport_CalculateAndSetHealthScore tests the report method
func TestWreckageReport_CalculateAndSetHealthScore(t *testing.T) {
	report := &WreckageReport{
		SessionID: "test-session",
	}

	metrics := SelectorMetrics{
		PrimaryCount:   70,
		SecondaryCount: 20,
		FallbackCount:  10,
		Total:          100,
	}

	report.CalculateAndSetHealthScore(metrics)

	// Formula: (70*1.0 + 20*0.5 + 10*0.1) / 100 = (70 + 10 + 1) / 100 = 0.81
	expectedScore := 0.81
	if report.SelectorHealth != expectedScore {
		t.Errorf("Expected health score %.2f, got %.2f", expectedScore, report.SelectorHealth)
	}

	if report.TotalInteractions != 100 {
		t.Errorf("Expected TotalInteractions to be 100, got %d", report.TotalInteractions)
	}
}

// TestWreckageReport_GetHealthStatus tests the human-readable status
func TestWreckageReport_GetHealthStatus(t *testing.T) {
	tests := []struct {
		health           float64
		expectedContains string
	}{
		{0.9, "Green Water"},
		{1.0, "Green Water"},
		{0.8, "Green Water"},
		{0.6, "Choppy Seas"},
		{0.5, "Choppy Seas"},
		{0.3, "Rough Waters"},
		{0.2, "Rough Waters"},
		{0.1, "Shipwreck"},
		{0.0, "Shipwreck"},
	}

	for _, tt := range tests {
		t.Run(healthName(tt.health), func(t *testing.T) {
			report := &WreckageReport{
				SelectorHealth: tt.health,
			}

			status := report.GetHealthStatus()
			if !contains(status, tt.expectedContains) {
				t.Errorf("Health %.2f: expected status to contain '%s', got '%s'", tt.health, tt.expectedContains, status)
			}
		})
	}
}

// TestExtractHostname tests URI parsing and hostname extraction
func TestExtractHostname(t *testing.T) {
	tests := []struct {
		uri          string
		expectedHost string
		expectError  bool
	}{
		{"https://github.com", "github.com", false},
		{"https://github.com/user/repo", "github.com", false},
		{"http://example.com:8080/path", "example.com", false},
		{"https://salesforce.com/login", "salesforce.com", false},
		{"invalid-uri", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			hostname, err := extractHostname(tt.uri)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URI '%s', but got none", tt.uri)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URI '%s': %v", tt.uri, err)
				return
			}

			if hostname != tt.expectedHost {
				t.Errorf("URI '%s': expected hostname '%s', got '%s'", tt.uri, tt.expectedHost, hostname)
			}
		})
	}
}

// TestNewWreckageReport tests report creation and URI parsing
func TestNewWreckageReport(t *testing.T) {
	buffer := NewCircularBuffer(10)
	sessionID := "test-session-123"
	targetURI := "https://github.com/user/repo"

	selector := Selector{
		PrimaryCSS: "#submit-button",
		Tier:       1,
	}

	report, err := NewWreckageReport(sessionID, targetURI, buffer, selector)

	if err != nil {
		t.Fatalf("Failed to create wreckage report: %v", err)
	}

	if report.TargetURI != targetURI {
		t.Errorf("Expected TargetURI '%s', got '%s'", targetURI, report.TargetURI)
	}

	if report.TargetHostname != "github.com" {
		t.Errorf("Expected TargetHostname 'github.com', got '%s'", report.TargetHostname)
	}

	if report.SessionID != sessionID {
		t.Errorf("Expected SessionID '%s', got '%s'", sessionID, report.SessionID)
	}

	if report.FailedSelector.PrimaryCSS != selector.PrimaryCSS {
		t.Errorf("Failed selector not properly set")
	}
}

// Helper function to generate test names
func healthName(health float64) string {
	return "health-" + string(rune('0'+int(health*10)))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
