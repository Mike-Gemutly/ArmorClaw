package email

import (
	"testing"
)

func TestNewSMTPClient_DefaultPort(t *testing.T) {
	client := NewSMTPClient(SMTPClientConfig{
		Host:     "smtp.test.com",
		Username: "user",
		Password: "pass",
		From:     "from@test.com",
	})
	if client.port != "587" {
		t.Errorf("default port = %q, want 587", client.port)
	}
}

func TestNewSMTPClient_DefaultTLSEnabled(t *testing.T) {
	client := NewSMTPClient(SMTPClientConfig{
		Host: "smtp.test.com",
		Port: "587",
	})
	if !client.useTLS {
		t.Error("TLS should be enabled for port 587")
	}
}

func TestNewSMTPClient_CustomPort(t *testing.T) {
	client := NewSMTPClient(SMTPClientConfig{
		Host:   "smtp.test.com",
		Port:   "2525",
		UseTLS: false,
	})
	if client.port != "2525" {
		t.Errorf("port = %q, want 2525", client.port)
	}
}

func TestNewSMTPClient_Port465_NoTLSFlag(t *testing.T) {
	client := NewSMTPClient(SMTPClientConfig{
		Host: "smtp.test.com",
		Port: "465",
	})
	if client.useTLS {
		t.Error("port 465 without UseTLS flag should not auto-enable TLS (only 587 does)")
	}
}

func TestSMTPClient_Provider(t *testing.T) {
	client := &SMTPClient{}
	if client.Provider() != "smtp" {
		t.Errorf("Provider() = %q, want smtp", client.Provider())
	}
}

func TestSMTPClient_ImplementsEmailSender(t *testing.T) {
	var _ EmailSender = (*SMTPClient)(nil)
}
