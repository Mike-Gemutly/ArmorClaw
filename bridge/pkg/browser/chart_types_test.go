package browser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestActionTypeConstants(t *testing.T) {
	consts := []ActionType{ActionClick, ActionInput, ActionNavigate, ActionWait, ActionAssert}
	want := []string{"click", "input", "navigate", "wait", "assert"}
	for i, c := range consts {
		if string(c) != want[i] {
			t.Errorf("ActionType constant %d = %q, want %q", i, c, want[i])
		}
	}
}

func TestNavChartRoundtrip(t *testing.T) {
	original := NavChart{
		Version:      1,
		TargetDomain: "https://example.com",
		Metadata: ChartMetadata{
			GeneratedBy: "@armorclaw/jetski-chartmaker",
			Timestamp:   "2026-04-26T00:00:00Z",
			SessionID:   "sess-1",
		},
		ActionMap: map[string]ChartAction{
			"login": {
				ActionType: ActionClick,
				Selector: &ChartSelector{
					PrimaryCSS:     "button.login",
					SecondaryXPath: "//button[@class='login']",
					FallbackJS:     "document.querySelector('button.login')",
				},
				PostActionWait: &WaitCondition{
					Type:    "waitForVisible",
					Timeout: 5000,
				},
			},
			"fill_email": {
				ActionType: ActionInput,
				Selector: &ChartSelector{
					PrimaryCSS: "input[type='email']",
				},
				Value: "{email}",
			},
			"goto_home": {
				ActionType: ActionNavigate,
				URL:        "https://example.com/home",
			},
			"check_title": {
				ActionType: ActionAssert,
				Assertion: &Assertion{
					Type:     "includes",
					Expected: "Dashboard",
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded NavChart
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, original.Version)
	}
	if decoded.TargetDomain != original.TargetDomain {
		t.Errorf("TargetDomain = %q, want %q", decoded.TargetDomain, original.TargetDomain)
	}
	if decoded.Metadata.SessionID != original.Metadata.SessionID {
		t.Errorf("Metadata.SessionID = %q, want %q", decoded.Metadata.SessionID, original.Metadata.SessionID)
	}
	if len(decoded.ActionMap) != len(original.ActionMap) {
		t.Fatalf("ActionMap length = %d, want %d", len(decoded.ActionMap), len(original.ActionMap))
	}

	login := decoded.ActionMap["login"]
	if login.ActionType != ActionClick {
		t.Errorf("login.ActionType = %q, want %q", login.ActionType, ActionClick)
	}
	if login.Selector == nil || login.Selector.PrimaryCSS != "button.login" {
		t.Error("login.Selector.PrimaryCSS missing or wrong")
	}
	if login.Selector.SecondaryXPath != "//button[@class='login']" {
		t.Errorf("login.Selector.SecondaryXPath = %q", login.Selector.SecondaryXPath)
	}
	if login.Selector.FallbackJS != "document.querySelector('button.login')" {
		t.Errorf("login.Selector.FallbackJS = %q", login.Selector.FallbackJS)
	}
	if login.PostActionWait == nil || login.PostActionWait.Type != "waitForVisible" {
		t.Error("login.PostActionWait missing or wrong type")
	}

	fill := decoded.ActionMap["fill_email"]
	if fill.Value != "{email}" {
		t.Errorf("fill_email.Value = %q, want %q", fill.Value, "{email}")
	}

	gotoHome := decoded.ActionMap["goto_home"]
	if gotoHome.URL != "https://example.com/home" {
		t.Errorf("goto_home.URL = %q", gotoHome.URL)
	}

	assertion := decoded.ActionMap["check_title"]
	if assertion.Assertion == nil || assertion.Assertion.Expected != "Dashboard" {
		t.Error("check_title.Assertion.Expected missing or wrong")
	}
}

func TestOmitEmpty(t *testing.T) {
	action := ChartAction{ActionType: ActionClick}
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to raw: %v", err)
	}

	for _, key := range []string{"selector", "value", "url", "frame_routing", "post_action_wait", "assertion"} {
		if _, ok := raw[key]; ok {
			t.Errorf("key %q should be omitted when empty", key)
		}
	}
}

func TestStripeChartUnmarshal(t *testing.T) {
	data, err := os.ReadFile("../../../jetski/lighthouse/charts/stripe.com.acsb.json")
	if err != nil {
		t.Skip("stripe.com.acsb.json not available:", err)
	}

	var chart NavChart
	if err := json.Unmarshal(data, &chart); err != nil {
		t.Fatalf("unmarshal stripe chart: %v", err)
	}

	if chart.Version != 1 {
		t.Errorf("Version = %d, want 1", chart.Version)
	}
	if chart.TargetDomain != "https://stripe.com" {
		t.Errorf("TargetDomain = %q, want %q", chart.TargetDomain, "https://stripe.com")
	}
	if chart.Metadata.GeneratedBy != "@armorclaw/jetski-chartmaker" {
		t.Errorf("GeneratedBy = %q", chart.Metadata.GeneratedBy)
	}

	expectedActions := []string{"click_pay", "enter_card_details", "enter_cvc", "enter_expiry", "enter_zip", "submit_payment", "handle_success", "handle_error"}
	for _, name := range expectedActions {
		if _, ok := chart.ActionMap[name]; !ok {
			t.Errorf("missing action %q", name)
		}
	}

	card := chart.ActionMap["enter_card_details"]
	if card.ActionType != ActionInput {
		t.Errorf("enter_card_details.ActionType = %q, want %q", card.ActionType, ActionInput)
	}
	if card.FrameRouting == nil {
		t.Fatal("enter_card_details.FrameRouting is nil")
	}
	if card.FrameRouting.Origin != "https://js.stripe.com" {
		t.Errorf("FrameRouting.Origin = %q", card.FrameRouting.Origin)
	}
	if card.Selector == nil || card.Selector.PrimaryCSS == "" {
		t.Error("enter_card_details.Selector.PrimaryCSS is empty")
	}
	if card.PostActionWait == nil {
		t.Fatal("enter_card_details.PostActionWait is nil")
	}

	submit := chart.ActionMap["submit_payment"]
	if submit.PostActionWait == nil || submit.PostActionWait.Type != "waitForSelector" {
		t.Error("submit_payment.PostActionWait.Type should be waitForSelector")
	}

	success := chart.ActionMap["handle_success"]
	if success.Assertion == nil || success.Assertion.Type != "exists" {
		t.Error("handle_success.Assertion.Type should be exists")
	}
}
