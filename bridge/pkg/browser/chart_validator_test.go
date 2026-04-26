package browser

import (
	"strings"
	"testing"
)

func validChart() NavChart {
	return NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		Metadata:     ChartMetadata{GeneratedBy: "test"},
		ActionMap: map[string]ChartAction{
			"action_1": {
				ActionType: ActionClick,
				Selector:   &ChartSelector{PrimaryCSS: "button.next"},
			},
		},
	}
}

func TestValidateForStorage_ValidMinimal(t *testing.T) {
	cv := NewChartValidator(nil)
	if err := cv.ValidateForStorage(validChart()); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateForStorage_UnsupportedVersion(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.Version = 2
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("error should mention unsupported version, got: %v", err)
	}
}

func TestValidateForStorage_EmptyTargetDomain(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.TargetDomain = ""
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for empty target_domain")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error should mention empty domain, got: %v", err)
	}
}

func TestValidateForStorage_NoSchemeDomain(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.TargetDomain = "example.com"
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for domain without scheme")
	}
	if !strings.Contains(err.Error(), "scheme") || !strings.Contains(err.Error(), "dot") {
		t.Errorf("error should mention scheme/dot requirement, got: %v", err)
	}
}

func TestValidateForStorage_LocalhostRejected(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.TargetDomain = "http://localhost"
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for localhost (no dot)")
	}
}

func TestValidateForStorage_EmptyActionMap(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap = map[string]ChartAction{}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for empty action_map")
	}
	if !strings.Contains(err.Error(), "at least one action") {
		t.Errorf("error should mention empty action_map, got: %v", err)
	}
}

func TestValidateForStorage_ClickWithoutSelector(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{ActionType: ActionClick}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for click without selector")
	}
	if !strings.Contains(err.Error(), "primary_css") {
		t.Errorf("error should mention primary_css, got: %v", err)
	}
}

func TestValidateForStorage_ClickWithoutPrimaryCSS(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionClick,
		Selector:   &ChartSelector{SecondaryXPath: "//button"},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for click without primary_css")
	}
}

func TestValidateForStorage_InputWithoutSelector(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{email}}",
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for input without selector")
	}
}

func TestValidateForStorage_InputWithSelector(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{email}}",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='email']"},
	}
	if err := cv.ValidateForStorage(chart); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateForStorage_NavigateNoSelectorNeeded(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionNavigate,
		URL:        "https://example.com/page",
	}
	if err := cv.ValidateForStorage(chart); err != nil {
		t.Fatalf("navigate should not require selector, got: %v", err)
	}
}

func TestValidateForStorage_WaitNoSelectorNeeded(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionWait,
		PostActionWait: &WaitCondition{
			Type:    "waitForSelector",
			Timeout: 3000,
		},
	}
	if err := cv.ValidateForStorage(chart); err != nil {
		t.Fatalf("wait should not require selector, got: %v", err)
	}
}

func TestValidateForStorage_PIIInValue_SSN(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "123-45-6789",
		Selector:   &ChartSelector{PrimaryCSS: "input"},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for SSN in value")
	}
	if !strings.Contains(err.Error(), "SSN") {
		t.Errorf("error should mention SSN, got: %v", err)
	}
}

func TestValidateForStorage_PIIInValue_CreditCard(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "4111222233334444",
		Selector:   &ChartSelector{PrimaryCSS: "input"},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for credit card in value")
	}
	if !strings.Contains(err.Error(), "CREDIT_CARD") {
		t.Errorf("error should mention CREDIT_CARD, got: %v", err)
	}
}

func TestValidateForStorage_PIIInValue_Email(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "user@example.com",
		Selector:   &ChartSelector{PrimaryCSS: "input"},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for email in value")
	}
	if !strings.Contains(err.Error(), "EMAIL") {
		t.Errorf("error should mention EMAIL, got: %v", err)
	}
}

func TestValidateForStorage_PIIInValue_Password(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      `password="SuperSecret123"`,
		Selector:   &ChartSelector{PrimaryCSS: "input"},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for password in value")
	}
	if !strings.Contains(err.Error(), "PASSWORD") {
		t.Errorf("error should mention PASSWORD, got: %v", err)
	}
}

func TestValidateForStorage_PlaceholderAllowed(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{ssn}}",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='ssn']"},
	}
	if err := cv.ValidateForStorage(chart); err != nil {
		t.Fatalf("placeholder should be allowed, got: %v", err)
	}
}

func TestValidateForStorage_NonPIILiteralAllowed(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "John",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='first_name']"},
	}
	if err := cv.ValidateForStorage(chart); err != nil {
		t.Fatalf("non-PII literal 'John' should be allowed, got: %v", err)
	}
}

func TestValidateForStorage_PIIInURL(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionNavigate,
		URL:        "https://example.com/login?email=user@example.com",
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for email in URL")
	}
	if !strings.Contains(err.Error(), "PII") {
		t.Errorf("error should mention PII, got: %v", err)
	}
}

func TestValidateForStorage_PIIInSelector(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionClick,
		Selector: &ChartSelector{
			PrimaryCSS: "input[value='user@example.com']",
		},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected error for email in selector")
	}
}

func TestValidateForStorage_MultipleIssues(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := NavChart{
		Version:      99,
		TargetDomain: "",
		ActionMap:    map[string]ChartAction{},
	}
	err := cv.ValidateForStorage(chart)
	if err == nil {
		t.Fatal("expected multiple errors")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Issues) < 3 {
		t.Errorf("expected at least 3 issues, got %d: %v", len(ve.Issues), ve.Issues)
	}
}

func TestValidateForReplay_ValidWithPII(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{email}}",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='email']"},
	}
	pii := map[string]string{"email": "user@example.com"}
	if err := cv.ValidateForReplay(chart, pii); err != nil {
		t.Fatalf("expected valid replay, got: %v", err)
	}
}

func TestValidateForReplay_UnresolvedPlaceholder(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{credit_card}}",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='cc']"},
	}
	pii := map[string]string{"email": "user@example.com"}
	err := cv.ValidateForReplay(chart, pii)
	if err == nil {
		t.Fatal("expected error for unresolved placeholder")
	}
	if !strings.Contains(err.Error(), "unresolved") {
		t.Errorf("error should mention unresolved, got: %v", err)
	}
}

func TestValidateForReplay_DomainPolicyRejects(t *testing.T) {
	policy := &mockDomainPolicy{allowed: map[string]bool{
		"https://trusted.com": true,
	}}
	cv := NewChartValidator(policy)
	chart := validChart()
	chart.TargetDomain = "https://evil.com"
	err := cv.ValidateForReplay(chart, nil)
	if err == nil {
		t.Fatal("expected error for domain not in allowed list")
	}
	if !strings.Contains(err.Error(), "not in the allowed list") {
		t.Errorf("error should mention allowed list, got: %v", err)
	}
}

func TestValidateForReplay_DomainPolicyAllows(t *testing.T) {
	policy := &mockDomainPolicy{allowed: map[string]bool{
		"https://example.com": true,
	}}
	cv := NewChartValidator(policy)
	if err := cv.ValidateForReplay(validChart(), nil); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateForReplay_StorageFailsFirst(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := NavChart{Version: 99}
	err := cv.ValidateForReplay(chart, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("should fail on storage validation first, got: %v", err)
	}
}

func TestValidateForReplay_FlagsApprovalActions(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionInput,
		Value:      "{{credit_card}}",
		Selector:   &ChartSelector{PrimaryCSS: "input[name='card_number']"},
	}
	pii := map[string]string{"credit_card": "4111222233334444"}
	err := cv.ValidateForReplay(chart, pii)
	if err == nil {
		t.Fatal("expected approval flag")
	}
	approval := RequiresApproval(err)
	if len(approval) == 0 {
		t.Error("expected approval-required issues")
	}
	if !strings.Contains(approval[0].Message, "may require approval") {
		t.Errorf("approval message wrong, got: %v", approval[0])
	}
}

func TestValidateForReplay_ClickSubmitApproval(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionClick,
		Selector:   &ChartSelector{PrimaryCSS: "button#submit-payment"},
	}
	err := cv.ValidateForReplay(chart, nil)
	if err == nil {
		t.Fatal("expected approval flag for submit click")
	}
	if len(RequiresApproval(err)) == 0 {
		t.Error("click on submit should require approval")
	}
}

func TestValidateForReplay_UnresolvedURLPlaceholder(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionNavigate,
		URL:        "https://example.com/page?id={{session_id}}",
	}
	err := cv.ValidateForReplay(chart, map[string]string{})
	if err == nil {
		t.Fatal("expected error for unresolved URL placeholder")
	}
	if !strings.Contains(err.Error(), "session_id") {
		t.Errorf("error should mention session_id, got: %v", err)
	}
}

func TestValidateForReplay_UnresolvedSelectorPlaceholder(t *testing.T) {
	cv := NewChartValidator(nil)
	chart := validChart()
	chart.ActionMap["action_1"] = ChartAction{
		ActionType: ActionClick,
		Selector: &ChartSelector{
			PrimaryCSS: "{{dynamic_selector}}",
		},
	}
	err := cv.ValidateForReplay(chart, map[string]string{})
	if err == nil {
		t.Fatal("expected error for unresolved selector placeholder")
	}
	if !strings.Contains(err.Error(), "dynamic_selector") {
		t.Errorf("error should mention dynamic_selector, got: %v", err)
	}
}

func TestRequiresApproval_NilError(t *testing.T) {
	if issues := RequiresApproval(nil); issues != nil {
		t.Errorf("expected nil, got: %v", issues)
	}
}

func TestRequiresApproval_NonValidationError(t *testing.T) {
	if issues := RequiresApproval(errFake("some error")); issues != nil {
		t.Errorf("expected nil for non-*ValidationError, got: %v", issues)
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }

func TestValidationIssue_Error(t *testing.T) {
	iss := ValidationIssue{Field: "version", Message: "bad"}
	want := "version: bad"
	if got := iss.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		Issues: []ValidationIssue{
			{Field: "a", Message: "x"},
			{Field: "b", Message: "y"},
		},
	}
	got := ve.Error()
	if !strings.Contains(got, "a: x") || !strings.Contains(got, "b: y") {
		t.Errorf("Error() should contain both issues, got: %s", got)
	}
}

func TestAllowAllDomains(t *testing.T) {
	policy := AllowAllDomains{}
	if !policy.IsAllowed("https://anything.evil") {
		t.Error("AllowAllDomains should allow everything")
	}
}

func TestNewChartValidator_DefaultPolicy(t *testing.T) {
	cv := NewChartValidator(nil)
	if cv.DomainPolicy == nil {
		t.Error("DomainPolicy should not be nil")
	}
}

func TestIsValidTargetDomain(t *testing.T) {
	tests := []struct {
		domain string
		valid  bool
	}{
		{"https://example.com", true},
		{"http://sub.example.com", true},
		{"ftp://files.example.org", true},
		{"https://192.168.1.1", true},
		{"example.com", false},
		{"http://localhost", false},
		{"", false},
		{"noscheme", false},
	}
	for _, tt := range tests {
		got := isValidTargetDomain(tt.domain)
		if got != tt.valid {
			t.Errorf("isValidTargetDomain(%q) = %v, want %v", tt.domain, got, tt.valid)
		}
	}
}

func TestFindUnresolvedPlaceholders(t *testing.T) {
	pii := map[string]string{"email": "a@b.com", "ssn": "123-45-6789"}
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"plain text", 0},
		{"{{email}}", 0},
		{"{{email}} and {{ssn}}", 0},
		{"{{email}} and {{credit_card}}", 1},
		{"{{unknown}}", 1},
		{"prefix {{email}} suffix {{missing}} end", 1},
		{"{{a}} {{b}} {{c}}", 3},
	}
	for _, tt := range tests {
		unresolved := findUnresolvedPlaceholders(tt.text, pii)
		if len(unresolved) != tt.want {
			t.Errorf("findUnresolvedPlaceholders(%q) = %v, want %d unresolved", tt.text, unresolved, tt.want)
		}
	}
}

type mockDomainPolicy struct {
	allowed map[string]bool
}

func (m *mockDomainPolicy) IsAllowed(domain string) bool {
	return m.allowed[domain]
}
