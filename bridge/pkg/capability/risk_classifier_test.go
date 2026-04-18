package capability

import (
	"context"
	"testing"
)

func TestRiskClassify_ExactMatch(t *testing.T) {
	rc := NewRiskClassifier()
	ctx := context.Background()

	tests := []struct {
		action         string
		wantClass      RiskClass
		wantLevel      RiskLevel
	}{
		{"browser.browse", RiskExternalCommunication, RiskAllow},
		{"browser.navigate", RiskExternalCommunication, RiskAllow},
		{"browser.screenshot", RiskExternalCommunication, RiskAllow},
		{"browser.fill_forms", RiskExternalCommunication, RiskDefer},
		{"browser.submit", RiskExternalCommunication, RiskDefer},
		{"browser.click", RiskExternalCommunication, RiskAllow},
		{"email.send", RiskExternalCommunication, RiskDefer},
		{"email.draft", RiskExternalCommunication, RiskAllow},
		{"email.read", RiskExternalCommunication, RiskAllow},
		{"secret.access", RiskCredentialUse, RiskDefer},
		{"secret.request", RiskCredentialUse, RiskDefer},
		{"secret.list", RiskCredentialUse, RiskDefer},
		{"payment.process", RiskPayment, RiskDefer},
		{"payment.refund", RiskPayment, RiskDefer},
		{"payment.view", RiskPayment, RiskAllow},
		{"pii.read", RiskIdentityPII, RiskDefer},
		{"pii.export", RiskIdentityPII, RiskDefer},
		{"pii.mask", RiskIdentityPII, RiskAllow},
		{"doc.query", RiskFileExfiltration, RiskAllow},
		{"doc.upload", RiskFileExfiltration, RiskAllow},
		{"doc.delete", RiskFileExfiltration, RiskDefer},
		{"doc.download", RiskFileExfiltration, RiskDefer},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			gotClass, gotLevel := rc.Classify(ctx, tt.action, nil)
			if gotClass != tt.wantClass {
				t.Errorf("Classify(%q) class = %q, want %q", tt.action, gotClass, tt.wantClass)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("Classify(%q) level = %q, want %q", tt.action, gotLevel, tt.wantLevel)
			}
		})
	}
}

func TestRiskClassify_PrefixMatch(t *testing.T) {
	rc := NewRiskClassifier()
	ctx := context.Background()

	tests := []struct {
		action    string
		wantClass RiskClass
		wantLevel RiskLevel
	}{
		{"browser.custom_action", RiskExternalCommunication, RiskAllow},
		{"browser.scroll", RiskExternalCommunication, RiskAllow},
		{"email.forward", RiskExternalCommunication, RiskDefer},
		{"email.search", RiskExternalCommunication, RiskDefer},
		{"secret.rotate", RiskCredentialUse, RiskDefer},
		{"payment.history", RiskPayment, RiskDefer},
		{"pii.anonymize", RiskIdentityPII, RiskDefer},
		{"doc.archive", RiskFileExfiltration, RiskAllow},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			gotClass, gotLevel := rc.Classify(ctx, tt.action, nil)
			if gotClass != tt.wantClass {
				t.Errorf("Classify(%q) class = %q, want %q", tt.action, gotClass, tt.wantClass)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("Classify(%q) level = %q, want %q", tt.action, gotLevel, tt.wantLevel)
			}
		})
	}
}

func TestRiskClassify_Unknown(t *testing.T) {
	rc := NewRiskClassifier()

	actions := []string{"unknown.action", "totally.new.thing", "system.reboot", "", "random"}
	for _, action := range actions {
		gotClass, gotLevel := rc.Classify(context.Background(), action, nil)
		if gotClass != RiskIrreversibleAction {
			t.Errorf("Classify(%q) class = %q, want %q", action, gotClass, RiskIrreversibleAction)
		}
		if gotLevel != RiskDeny {
			t.Errorf("Classify(%q) level = %q, want %q", action, gotLevel, RiskDeny)
		}
	}
}

func TestRiskClassify_ParamsIgnored(t *testing.T) {
	rc := NewRiskClassifier()
	ctx := context.Background()

	params := map[string]any{
		"url":      "https://evil.example.com",
		"amount":   999999,
		"override": true,
	}

	gotClass, gotLevel := rc.Classify(ctx, "browser.browse", params)
	if gotClass != RiskExternalCommunication {
		t.Errorf("class = %q, want %q", gotClass, RiskExternalCommunication)
	}
	if gotLevel != RiskAllow {
		t.Errorf("level = %q, want %q", gotLevel, RiskAllow)
	}
}

func TestRiskClassify_NilContext(t *testing.T) {
	rc := NewRiskClassifier()

	gotClass, gotLevel := rc.Classify(nil, "payment.process", nil)
	if gotClass != RiskPayment {
		t.Errorf("class = %q, want %q", gotClass, RiskPayment)
	}
	if gotLevel != RiskDefer {
		t.Errorf("level = %q, want %q", gotLevel, RiskDefer)
	}
}

func TestRiskClassify_SpecificCases(t *testing.T) {
	rc := NewRiskClassifier()
	ctx := context.Background()

	cases := []struct {
		action    string
		wantClass RiskClass
		wantLevel RiskLevel
	}{
		{"browser.browse", RiskExternalCommunication, RiskAllow},
		{"browser.fill_forms", RiskExternalCommunication, RiskDefer},
		{"email.send", RiskExternalCommunication, RiskDefer},
		{"payment.process", RiskPayment, RiskDefer},
		{"secret.access", RiskCredentialUse, RiskDefer},
		{"doc.delete", RiskFileExfiltration, RiskDefer},
		{"unknown.action", RiskIrreversibleAction, RiskDeny},
	}

	for _, tt := range cases {
		t.Run(tt.action, func(t *testing.T) {
			gotClass, gotLevel := rc.Classify(ctx, tt.action, nil)
			if gotClass != tt.wantClass {
				t.Errorf("class = %q, want %q", gotClass, tt.wantClass)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("level = %q, want %q", gotLevel, tt.wantLevel)
			}
		})
	}
}
