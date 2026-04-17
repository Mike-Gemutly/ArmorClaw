package email

import (
	"testing"
)

func TestOutlookClient_Provider(t *testing.T) {
	client := &OutlookClient{}
	if client.Provider() != "outlook" {
		t.Errorf("Provider() = %q, want outlook", client.Provider())
	}
}

func TestOutlookClient_ImplementsEmailSender(t *testing.T) {
	var _ EmailSender = (*OutlookClient)(nil)
}

func TestNewOutlookClient_DefaultGraphBase(t *testing.T) {
	client := NewOutlookClient(OutlookClientConfig{From: "user@test.com"})
	if client.graphBase != "https://graph.microsoft.com/v1.0" {
		t.Errorf("graphBase = %q, want default Microsoft Graph URL", client.graphBase)
	}
	if client.from != "user@test.com" {
		t.Errorf("from = %q, want user@test.com", client.from)
	}
}
