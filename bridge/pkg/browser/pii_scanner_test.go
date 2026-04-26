package browser

import (
	"context"
	"errors"
	"testing"
)

type mockChartStore struct {
	charts map[string][]ChartRecord
	err    error
}

func (m *mockChartStore) SaveChart(_ context.Context, _ NavChart, _ ChartMeta) (string, error) {
	return "mock-id", nil
}

func (m *mockChartStore) FindForDomain(_ context.Context, domain string, _ int) ([]ChartRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.charts[domain], nil
}

func (m *mockChartStore) RecordOutcome(_ context.Context, _ string, _ bool) error { return nil }
func (m *mockChartStore) GetChart(_ context.Context, _ string) (*ChartRecord, error) {
	return nil, errors.New("not implemented")
}
func (m *mockChartStore) DeleteChart(_ context.Context, _ string) error { return nil }

func cleanChart(chart *NavChart) *NavChart {
	if chart == nil {
		return nil
	}
	return &NavChart{
		Version:      chart.Version,
		TargetDomain: chart.TargetDomain,
		Metadata:     chart.Metadata,
		ActionMap:    chart.ActionMap,
	}
}

func TestScanChartsForPII_EmptyStore(t *testing.T) {
	store := &mockChartStore{charts: map[string][]ChartRecord{}}
	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestScanChartsForPII_CleanChart(t *testing.T) {
	cleanNav := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionNavigate, URL: "https://example.com/page"},
			"action_2": {ActionType: ActionInput, Value: PlaceholderSSN},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c1", NavChart: cleanNav}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean chart, got %d: %+v", len(findings), findings)
	}
}

func TestScanChartsForPII_SSNInValue(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionInput, Value: "123-45-6789"},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c1", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.ChartID != "c1" {
		t.Errorf("chart_id = %q, want %q", f.ChartID, "c1")
	}
	if f.ActionKey != "action_1" {
		t.Errorf("action_key = %q, want %q", f.ActionKey, "action_1")
	}
	if f.Field != "value" {
		t.Errorf("field = %q, want %q", f.Field, "value")
	}
	if f.MatchedPattern != "SSN" {
		t.Errorf("matched_pattern = %q, want %q", f.MatchedPattern, "SSN")
	}
	if f.Severity != "high" {
		t.Errorf("severity = %q, want %q", f.Severity, "high")
	}
}

func TestScanChartsForPII_CreditCardInValue(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionInput, Value: "4111-1111-1111-1111"},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c2", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].MatchedPattern != "CREDIT_CARD" {
		t.Errorf("matched_pattern = %q, want CREDIT_CARD", findings[0].MatchedPattern)
	}
	if findings[0].Severity != "high" {
		t.Errorf("severity = %q, want high", findings[0].Severity)
	}
}

func TestScanChartsForPII_EmailInValue(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionInput, Value: "user@example.com"},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c3", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].MatchedPattern != "EMAIL" {
		t.Errorf("matched_pattern = %q, want EMAIL", findings[0].MatchedPattern)
	}
	if findings[0].Severity != "medium" {
		t.Errorf("severity = %q, want medium", findings[0].Severity)
	}
}

func TestScanChartsForPII_PIIInURL(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionNavigate, URL: "https://example.com?email=user@test.com"},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c4", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Field != "url" {
		t.Errorf("field = %q, want url", findings[0].Field)
	}
}

func TestScanChartsForPII_PIIInSelector(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {
				ActionType: ActionClick,
				Selector:   &ChartSelector{PrimaryCSS: "input[name='123-45-6789']"},
			},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c5", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Field != "selector_css" {
		t.Errorf("field = %q, want selector_css", findings[0].Field)
	}
}

func TestScanChartsForPII_MultipleFindings(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionInput, Value: "123-45-6789"},
			"action_2": {ActionType: ActionInput, Value: "user@example.com"},
			"action_3": {ActionType: ActionNavigate, URL: "https://safe.com"},
		},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://unknown": {{ChartID: "c6", NavChart: chart}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
}

func TestScanChartsForPII_MultipleDomains(t *testing.T) {
	chart1 := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://a.com",
		ActionMap:    map[string]ChartAction{"action_1": {ActionType: ActionInput, Value: "123-45-6789"}},
	})
	chart2 := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://b.com",
		ActionMap:    map[string]ChartAction{"action_1": {ActionType: ActionInput, Value: "4111-1111-1111-1111"}},
	})

	store := &mockChartStore{charts: map[string][]ChartRecord{
		"https://a.com": {{ChartID: "ca", NavChart: chart1}},
		"https://b.com": {{ChartID: "cb", NavChart: chart2}},
	}}

	findings, err := ScanChartsForPII(context.Background(), store, []string{"https://a.com", "https://b.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings across domains, got %d", len(findings))
	}
}

func TestScanChartsForPII_StoreError(t *testing.T) {
	store := &mockChartStore{err: errors.New("db locked")}
	_, err := ScanChartsForPII(context.Background(), store, []string{"https://fail.com"})
	if err == nil {
		t.Fatal("expected error from store, got nil")
	}
}

func TestScanSingleChart_FallbackRawSteps(t *testing.T) {
	rec := ChartRecord{
		ChartID: "raw1",
		Steps:   `{"action_1":{"value":"123-45-6789"}}`,
	}

	findings := ScanSingleChart(rec)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding from raw steps, got %d", len(findings))
	}
	if findings[0].Field != "steps" {
		t.Errorf("field = %q, want steps", findings[0].Field)
	}
}

func TestScanSingleChart_CleanPlaceholder(t *testing.T) {
	chart := cleanChart(&NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		ActionMap: map[string]ChartAction{
			"action_1": {ActionType: ActionInput, Value: PlaceholderSSN},
			"action_2": {ActionType: ActionInput, Value: PlaceholderEmail},
		},
	})

	rec := ChartRecord{ChartID: "clean1", NavChart: chart}
	findings := ScanSingleChart(rec)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for placeholder-only chart, got %d", len(findings))
	}
}

func TestIsPlaceholder(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{PlaceholderSSN, true},
		{PlaceholderCreditCard, true},
		{PlaceholderEmail, true},
		{PlaceholderPassword, true},
		{"{{SSN}}", true},
		{"123-45-6789", false},
		{"user@example.com", false},
		{"hello world", false},
	}
	for _, tt := range tests {
		got := isPlaceholder(tt.text)
		if got != tt.want {
			t.Errorf("isPlaceholder(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestSeverityForType(t *testing.T) {
	tests := []struct {
		pType string
		want  string
	}{
		{"SSN", "high"},
		{"CREDIT_CARD", "high"},
		{"PASSWORD", "high"},
		{"EMAIL", "medium"},
		{"UNKNOWN", "low"},
	}
	for _, tt := range tests {
		got := severityForType(tt.pType)
		if got != tt.want {
			t.Errorf("severityForType(%q) = %q, want %q", tt.pType, got, tt.want)
		}
	}
}
