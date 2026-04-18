package team

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRequestSecret_Approved(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 5 * time.Second,
	})

	req := &SecretRequest{
		AgentID:        "agent-1",
		TeamID:         "team-1",
		CredentialName: "api_key",
		TargetDomain:   "example.com",
		Reason:         "fetch data",
		RiskClass:      "credential_use",
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var resp *SecretResponse
	var err error

	go func() {
		defer wg.Done()
		resp, err = mgr.RequestSecret(context.Background(), req)
	}()

	// Wait for request to be registered.
	time.Sleep(50 * time.Millisecond)

	// Find the pending request and approve it.
	if mgr.PendingCount() != 1 {
		t.Fatalf("expected 1 pending request, got %d", mgr.PendingCount())
	}

	// Extract request_id from the pending map.
	mgr.mu.RLock()
	var requestID string
	for id := range mgr.pending {
		requestID = id
		break
	}
	mgr.mu.RUnlock()

	handleErr := mgr.HandleResponse(requestID, true, "user@example.com", "sk-12345")
	if handleErr != nil {
		t.Fatalf("HandleResponse returned error: %v", handleErr)
	}

	wg.Wait()

	if err != nil {
		t.Fatalf("RequestSecret returned error: %v", err)
	}
	if !resp.Approved {
		t.Error("expected Approved=true")
	}
	if resp.SecretValue != "sk-12345" {
		t.Errorf("expected SecretValue=sk-12345, got %s", resp.SecretValue)
	}
	if resp.RespondedBy != "user@example.com" {
		t.Errorf("expected RespondedBy=user@example.com, got %s", resp.RespondedBy)
	}
}

func TestRequestSecret_Denied(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 5 * time.Second,
	})

	req := &SecretRequest{
		AgentID:        "agent-2",
		TeamID:         "team-1",
		CredentialName: "password",
		TargetDomain:   "bank.com",
		Reason:         "login",
		RiskClass:      "credential_use",
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var resp *SecretResponse
	var err error

	go func() {
		defer wg.Done()
		resp, err = mgr.RequestSecret(context.Background(), req)
	}()

	time.Sleep(50 * time.Millisecond)

	mgr.mu.RLock()
	var requestID string
	for id := range mgr.pending {
		requestID = id
		break
	}
	mgr.mu.RUnlock()

	handleErr := mgr.HandleResponse(requestID, false, "admin@company.com", "")
	if handleErr != nil {
		t.Fatalf("HandleResponse returned error: %v", handleErr)
	}

	wg.Wait()

	if err != nil {
		t.Fatalf("RequestSecret returned error: %v", err)
	}
	if resp.Approved {
		t.Error("expected Approved=false")
	}
	if resp.SecretValue != "" {
		t.Errorf("expected empty SecretValue, got %s", resp.SecretValue)
	}
}

func TestRequestSecret_Timeout(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 100 * time.Millisecond,
	})

	req := &SecretRequest{
		AgentID:        "agent-3",
		CredentialName: "ssh_key",
		TargetDomain:   "server.com",
		RiskClass:      "credential_use",
	}

	resp, err := mgr.RequestSecret(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestSecret returned error: %v", err)
	}
	if resp.Approved {
		t.Error("expected auto-deny on timeout")
	}
	if resp.RequestID == "" {
		t.Error("expected non-empty RequestID")
	}
}

func TestRequestSecret_ContextCancelled(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var err error

	go func() {
		defer wg.Done()
		_, err = mgr.RequestSecret(ctx, &SecretRequest{
			AgentID:        "agent-4",
			CredentialName: "token",
			TargetDomain:   "api.com",
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	wg.Wait()

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestHandleResponse_NotFound(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 5 * time.Second,
	})

	err := mgr.HandleResponse("nonexistent_id", true, "user", "val")
	if err == nil {
		t.Fatal("expected error for unknown request_id")
	}
}

func TestPendingCount(t *testing.T) {
	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 5 * time.Second,
	})

	if count := mgr.PendingCount(); count != 0 {
		t.Fatalf("expected 0 pending, got %d", count)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		mgr.RequestSecret(context.Background(), &SecretRequest{
			AgentID:        "agent-5",
			CredentialName: "key",
			TargetDomain:   "host",
		})
	}()

	time.Sleep(50 * time.Millisecond)

	if count := mgr.PendingCount(); count != 1 {
		t.Errorf("expected 1 pending, got %d", count)
	}

	mgr.mu.RLock()
	var requestID string
	for id := range mgr.pending {
		requestID = id
		break
	}
	mgr.mu.RUnlock()

	mgr.HandleResponse(requestID, false, "user", "")
	wg.Wait()

	if count := mgr.PendingCount(); count != 0 {
		t.Errorf("expected 0 pending after response, got %d", count)
	}
}

func TestMatrixEventPublished(t *testing.T) {
	var capturedEventType string
	var capturedBody string
	var mu sync.Mutex

	mgr := NewSecretRequestManager(SecretRequestConfig{
		Timeout: 100 * time.Millisecond,
		SendMatrixMsg: func(roomID, eventType, body string) error {
			mu.Lock()
			capturedEventType = eventType
			capturedBody = body
			mu.Unlock()
			return nil
		},
	})

	_, _ = mgr.RequestSecret(context.Background(), &SecretRequest{
		AgentID:        "agent-6",
		TeamID:         "team-6",
		CredentialName: "oauth_token",
		TargetDomain:   "provider.com",
		Reason:         "authenticate",
		RiskClass:      "credential_use",
	})

	mu.Lock()
	defer mu.Unlock()

	if capturedEventType != "app.armorclaw.secret_request" {
		t.Errorf("expected event type app.armorclaw.secret_request, got %s", capturedEventType)
	}
	if capturedBody == "" {
		t.Error("expected non-empty body")
	}
	if capturedEventType != "" && capturedBody != "" {
		// Verify the body contains expected fields.
		for _, substr := range []string{`"agent_id":"agent-6"`, `"credential_name":"oauth_token"`, `"target_domain":"provider.com"`} {
			if !contains(capturedBody, substr) {
				t.Errorf("body missing expected substring: %s", substr)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
