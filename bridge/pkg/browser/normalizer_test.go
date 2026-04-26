package browser

import (
	"encoding/json"
	"testing"
	"time"
)

func mustRawMsg(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func TestNormalize_EmptyFrames(t *testing.T) {
	chart, err := Normalize(nil, "sess-1")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if chart.Version != 1 {
		t.Errorf("Version = %d, want 1", chart.Version)
	}
	if len(chart.ActionMap) != 0 {
		t.Errorf("ActionMap length = %d, want 0", len(chart.ActionMap))
	}
	if chart.Metadata.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", chart.Metadata.SessionID, "sess-1")
	}
	if chart.Metadata.GeneratedBy != "bridge-normalizer" {
		t.Errorf("GeneratedBy = %q, want %q", chart.Metadata.GeneratedBy, "bridge-normalizer")
	}
}

func TestNormalize_EmptySlice(t *testing.T) {
	chart, err := Normalize([]CDPFrame{}, "sess-2")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if len(chart.ActionMap) != 0 {
		t.Errorf("ActionMap length = %d, want 0", len(chart.ActionMap))
	}
}

func TestNormalize_NavigateOnly(t *testing.T) {
	frames := []CDPFrame{
		{
			Timestamp: time.Now(),
			Method:    "Page.navigate",
			Params:    mustRawMsg(t, map[string]string{"url": "https://example.com"}),
			SessionID: "sess-1",
		},
	}

	chart, err := Normalize(frames, "sess-1")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	if chart.TargetDomain != "https://example.com" {
		t.Errorf("TargetDomain = %q, want %q", chart.TargetDomain, "https://example.com")
	}
	if len(chart.ActionMap) != 1 {
		t.Fatalf("ActionMap length = %d, want 1", len(chart.ActionMap))
	}

	action := chart.ActionMap["action_1"]
	if action.ActionType != ActionNavigate {
		t.Errorf("action_1.ActionType = %q, want %q", action.ActionType, ActionNavigate)
	}
	if action.URL != "https://example.com" {
		t.Errorf("action_1.URL = %q, want %q", action.URL, "https://example.com")
	}
	if action.PostActionWait == nil {
		t.Error("action_1.PostActionWait is nil, expected wait condition")
	}
}

func TestNormalize_NavigateFillClick(t *testing.T) {
	frames := []CDPFrame{
		{
			Timestamp: time.Now(),
			Method:    "Page.navigate",
			Params:    mustRawMsg(t, map[string]string{"url": "https://shop.example.com/checkout"}),
			SessionID: "sess-3",
		},
		{
			Timestamp: time.Now(),
			Method:    "Input.insertText",
			Params:    mustRawMsg(t, map[string]string{"text": "user@example.com"}),
			SessionID: "sess-3",
		},
		{
			Timestamp: time.Now(),
			Method:    "Input.dispatchMouseEvent",
			Params:    mustRawMsg(t, map[string]interface{}{"type": "mousePressed", "x": 150, "y": 300}),
			SessionID: "sess-3",
		},
	}

	chart, err := Normalize(frames, "sess-3")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	if chart.TargetDomain != "https://shop.example.com" {
		t.Errorf("TargetDomain = %q, want %q", chart.TargetDomain, "https://shop.example.com")
	}
	if len(chart.ActionMap) != 3 {
		t.Fatalf("ActionMap length = %d, want 3", len(chart.ActionMap))
	}

	nav := chart.ActionMap["action_1"]
	if nav.ActionType != ActionNavigate {
		t.Errorf("action_1 type = %q, want navigate", nav.ActionType)
	}
	if nav.URL != "https://shop.example.com/checkout" {
		t.Errorf("action_1.URL = %q", nav.URL)
	}

	fill := chart.ActionMap["action_2"]
	if fill.ActionType != ActionInput {
		t.Errorf("action_2 type = %q, want input", fill.ActionType)
	}
	if fill.Value != PlaceholderEmail {
		t.Errorf("action_2.Value = %q, want %q (PII should be replaced)", fill.Value, PlaceholderEmail)
	}

	click := chart.ActionMap["action_3"]
	if click.ActionType != ActionClick {
		t.Errorf("action_3 type = %q, want click", click.ActionType)
	}
}

func TestNormalize_FiltersNoise(t *testing.T) {
	frames := []CDPFrame{
		{Timestamp: time.Now(), Method: "Network.requestWillBeSent", Params: mustRawMsg(t, map[string]string{"url": "https://cdn.example.com/style.css"}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "Console.messageAdded", Params: mustRawMsg(t, map[string]string{"text": "debug log"}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "CSS.styleChanged", Params: mustRawMsg(t, map[string]string{}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "Log.entryAdded", Params: mustRawMsg(t, map[string]string{}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "DOM.childNodeInserted", Params: mustRawMsg(t, map[string]string{}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "Page.navigate", Params: mustRawMsg(t, map[string]string{"url": "https://example.com"}), SessionID: "s"},
		{Timestamp: time.Now(), Method: "Network.responseReceived", Params: mustRawMsg(t, map[string]string{}), SessionID: "s"},
	}

	chart, err := Normalize(frames, "s")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	if len(chart.ActionMap) != 1 {
		t.Errorf("ActionMap length = %d, want 1 (noise should be filtered)", len(chart.ActionMap))
	}

	action := chart.ActionMap["action_1"]
	if action.ActionType != ActionNavigate {
		t.Errorf("action_1 type = %q, want navigate", action.ActionType)
	}
}

func TestNormalize_SelectorContext(t *testing.T) {
	frames := []CDPFrame{
		{
			Timestamp: time.Now(),
			Method:    "DOM.querySelector",
			Params:    mustRawMsg(t, map[string]interface{}{"nodeId": 1, "selector": "#email-input"}),
			SessionID: "s",
		},
		{
			Timestamp: time.Now(),
			Method:    "Runtime.evaluate",
			Params:    mustRawMsg(t, map[string]string{"expression": "document.querySelector('#email-input')"}),
			SessionID: "s",
		},
		{
			Timestamp: time.Now(),
			Method:    "Input.insertText",
			Params:    mustRawMsg(t, map[string]string{"text": "hello"}),
			SessionID: "s",
		},
	}

	chart, err := Normalize(frames, "s")
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	action := chart.ActionMap["action_1"]
	if action.Selector == nil {
		t.Fatal("action_1.Selector is nil, expected selector from context frames")
	}
	if action.Selector.PrimaryCSS != "#email-input" {
		t.Errorf("PrimaryCSS = %q, want %q", action.Selector.PrimaryCSS, "#email-input")
	}
	if action.Selector.FallbackJS != "document.querySelector('#email-input')" {
		t.Errorf("FallbackJS = %q, want JS expression", action.Selector.FallbackJS)
	}
	if action.Value != "hello" {
		t.Errorf("Value = %q, want %q (non-PII preserved as-is)", action.Value, "hello")
	}
}

func TestPIIDetection(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		hasPII bool
	}{
		{"SSN", "123-45-6789", true},
		{"CreditCard", "4111222233334444", true},
		{"CreditCardDashed", "4111-2222-3333-4444", true},
		{"CreditCardSpaced", "4111 2222 3333 4444", true},
		{"Email", "user@example.com", true},
		{"Password", "password=SuperSecret123", true},
		{"NoPII", "John", false},
		{"NormalText", "click the button", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := DetectPII(tt.input)
			if tt.hasPII && len(findings) == 0 {
				t.Errorf("DetectPII(%q): expected PII detection, got none", tt.input)
			}
			if !tt.hasPII && len(findings) > 0 {
				t.Errorf("DetectPII(%q): expected no PII, got %v", tt.input, findings)
			}
		})
	}
}

func TestReplacePII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"SSN", "123-45-6789", PlaceholderSSN},
		{"CreditCard", "4111222233334444", PlaceholderCreditCard},
		{"Email", "user@example.com", PlaceholderEmail},
		{"NoPII", "John", "John"},
		{"MixedPII", "SSN: 123-45-6789 email: user@example.com", "SSN: {{ssn}} email: {{email}}"},
		{"NonPIIPreserved", "Hello John Smith", "Hello John Smith"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplacePII(tt.input)
			if result != tt.expected {
				t.Errorf("ReplacePII(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/path", "https://example.com"},
		{"http://shop.example.com:8080/checkout", "http://shop.example.com:8080"},
		{"example.com/path", "https://example.com"},
		{"", ""},
		{"not-a-url", "https://not-a-url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractDomain(tt.input)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterFrames(t *testing.T) {
	frames := []CDPFrame{
		{Method: "Page.navigate"},
		{Method: "Network.requestWillBeSent"},
		{Method: "Input.insertText"},
		{Method: "Console.messageAdded"},
		{Method: "DOM.querySelector"},
		{Method: "CSS.styleChanged"},
		{Method: "Input.dispatchMouseEvent"},
	}

	filtered := filterFrames(frames)
	if len(filtered) != 4 {
		t.Errorf("filterFrames: got %d frames, want 4", len(filtered))
	}

	methods := make(map[string]bool, len(filtered))
	for _, f := range filtered {
		methods[f.Method] = true
	}
	for _, want := range []string{"Page.navigate", "Input.insertText", "DOM.querySelector", "Input.dispatchMouseEvent"} {
		if !methods[want] {
			t.Errorf("missing filtered method %q", want)
		}
	}
}

func TestGroupFrames(t *testing.T) {
	frames := []CDPFrame{
		{Method: "DOM.querySelector", Params: mustRawMsg(t, map[string]string{"selector": "#btn"})},
		{Method: "Page.navigate", Params: mustRawMsg(t, map[string]string{"url": "https://example.com"})},
		{Method: "DOM.querySelector", Params: mustRawMsg(t, map[string]string{"selector": "#input"})},
		{Method: "Runtime.evaluate", Params: mustRawMsg(t, map[string]string{"expression": "document.querySelector('#input')"})},
		{Method: "Input.insertText", Params: mustRawMsg(t, map[string]string{"text": "hello"})},
	}

	groups := groupFrames(frames)
	if len(groups) != 2 {
		t.Fatalf("groupFrames: got %d groups, want 2", len(groups))
	}

	if groups[0].primary.Method != "Page.navigate" {
		t.Errorf("group 0 method = %q, want Page.navigate", groups[0].primary.Method)
	}
	if len(groups[0].selectors) != 1 {
		t.Errorf("group 0 selectors = %d, want 1", len(groups[0].selectors))
	}

	if groups[1].primary.Method != "Input.insertText" {
		t.Errorf("group 1 method = %q, want Input.insertText", groups[1].primary.Method)
	}
	if len(groups[1].selectors) != 2 {
		t.Errorf("group 1 selectors = %d, want 2", len(groups[1].selectors))
	}
}
