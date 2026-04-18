package capability

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestNewSecretHandler_Approval(t *testing.T) {
	requester := func(_ context.Context, req SecretRequestParams) (*SecretResult, error) {
		return &SecretResult{
			RequestID: "secret_001",
			Approved:  true,
		}, nil
	}

	handler := NewSecretHandler(requester)
	input, _ := json.Marshal(SecretRequestParams{
		AgentID:        "agent-1",
		CredentialName: "db_password",
		TargetDomain:   "example.com",
		Reason:         "database connection",
	})

	out, err := handler(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var resp secretHandlerResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if !resp.Approved {
		t.Error("expected approved=true")
	}
	if resp.RequestID != "secret_001" {
		t.Errorf("request_id: got %q, want %q", resp.RequestID, "secret_001")
	}
	wantRef := "{{VAULT:db_password:secret_001}}"
	if resp.SecretRef != wantRef {
		t.Errorf("secret_ref: got %q, want %q", resp.SecretRef, wantRef)
	}
}

func TestNewSecretHandler_Denial(t *testing.T) {
	requester := func(_ context.Context, req SecretRequestParams) (*SecretResult, error) {
		return &SecretResult{
			RequestID: "secret_002",
			Approved:  false,
		}, nil
	}

	handler := NewSecretHandler(requester)
	input, _ := json.Marshal(SecretRequestParams{
		AgentID:        "agent-2",
		CredentialName: "credit_card",
	})

	out, err := handler(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var resp secretHandlerResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Approved {
		t.Error("expected approved=false")
	}
	if resp.SecretRef != "" {
		t.Errorf("expected empty secret_ref on denial, got %q", resp.SecretRef)
	}
	if resp.Reason == "" {
		t.Error("expected denial reason")
	}
}

func TestNewSecretHandler_MissingAgentID(t *testing.T) {
	requester := func(_ context.Context, _ SecretRequestParams) (*SecretResult, error) {
		return nil, errors.New("should not be called")
	}

	handler := NewSecretHandler(requester)
	input, _ := json.Marshal(SecretRequestParams{
		CredentialName: "db_password",
	})

	_, err := handler(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if matched := err.Error(); !contains(matched, "agent_id") {
		t.Errorf("error should mention agent_id, got: %s", matched)
	}
}

func TestNewSecretHandler_MissingCredentialName(t *testing.T) {
	requester := func(_ context.Context, _ SecretRequestParams) (*SecretResult, error) {
		return nil, errors.New("should not be called")
	}

	handler := NewSecretHandler(requester)
	input, _ := json.Marshal(SecretRequestParams{
		AgentID: "agent-1",
	})

	_, err := handler(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing credential_name")
	}
	if matched := err.Error(); !contains(matched, "credential_name") {
		t.Errorf("error should mention credential_name, got: %s", matched)
	}
}

func TestNewSecretHandler_MalformedJSON(t *testing.T) {
	requester := func(_ context.Context, _ SecretRequestParams) (*SecretResult, error) {
		return nil, errors.New("should not be called")
	}

	handler := NewSecretHandler(requester)
	_, err := handler(context.Background(), json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if matched := err.Error(); !contains(matched, "invalid input") {
		t.Errorf("error should mention invalid input, got: %s", matched)
	}
}

func TestNewSecretHandler_RequesterError(t *testing.T) {
	wantErr := errors.New("HITL timeout")
	requester := func(_ context.Context, _ SecretRequestParams) (*SecretResult, error) {
		return nil, wantErr
	}

	handler := NewSecretHandler(requester)
	input, _ := json.Marshal(SecretRequestParams{
		AgentID:        "agent-1",
		CredentialName: "api_key",
	})

	_, err := handler(context.Background(), input)
	if err == nil {
		t.Fatal("expected requester error to propagate")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
