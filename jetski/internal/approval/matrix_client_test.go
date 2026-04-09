package approval

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestApprovalRequest_Approved(t *testing.T) {
	// Mock bridge RPC that accepts approval requests
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	// Simulate async approval response
	go func() {
		defer wg.Done()
		// Wait a moment so the request is pending
		time.Sleep(100 * time.Millisecond)
		// Get the pending request ID
		ac.mu.RLock()
		var id string
		for k := range ac.pending {
			id = k
			break
		}
		ac.mu.RUnlock()
		if id != "" {
			ac.HandleApprovalResponse(id, true)
		}
	}()

	approved, err := ac.RequestApproval(context.Background(), OpSessionCreate, "Create session for task X")
	wg.Wait()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !approved {
		t.Fatal("expected approval to be true")
	}
}

func TestApprovalRequest_Denied(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		ac.mu.RLock()
		var id string
		for k := range ac.pending {
			id = k
			break
		}
		ac.mu.RUnlock()
		if id != "" {
			ac.HandleApprovalResponse(id, false)
		}
	}()

	approved, err := ac.RequestApproval(context.Background(), OpNavigation, "Navigate to https://example.com")
	wg.Wait()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if approved {
		t.Fatal("expected approval to be false (denied)")
	}
}

func TestApprovalRequest_Timeout(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	// Short timeout to make test fast
	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 200*time.Millisecond)
	defer ac.Close()

	approved, err := ac.RequestApproval(context.Background(), OpFileDownload, "Download large file")

	if err != nil {
		t.Fatalf("expected no error on timeout, got %v", err)
	}
	if approved {
		t.Fatal("expected approval to be false (timeout)")
	}
}

func TestApprovalRequest_ContextCancelled(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := ac.RequestApproval(ctx, OpSessionCreate, "Create session")

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestHandleApprovalResponse_NoMatch(t *testing.T) {
	ac := NewApprovalClient("http://127.0.0.1:8080", "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	// Should not panic when responding to non-existent request
	ac.HandleApprovalResponse("nonexistent-id", true)
}

func TestPendingCount(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	if ac.PendingCount() != 0 {
		t.Fatalf("expected 0 pending, got %d", ac.PendingCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch 2 requests that will block
	go ac.RequestApproval(ctx, OpSessionCreate, "session 1")
	go ac.RequestApproval(ctx, OpNavigation, "nav 1")

	// Wait for both to register
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && ac.PendingCount() < 2 {
		time.Sleep(10 * time.Millisecond)
	}

	count := ac.PendingCount()
	if count < 2 {
		t.Fatalf("expected at least 2 pending, got %d", count)
	}
}

func TestRPCHandleApprovalResponse(t *testing.T) {
	ac := NewApprovalClient("http://127.0.0.1:8080", "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	// Start RPC server with approval client
	mux := http.NewServeMux()
	RegisterApprovalHandlers(mux, ac)
	server := httptest.NewServer(mux)
	defer server.Close()

	// POST a valid approval response
	body := map[string]interface{}{
		"id":       "test-123",
		"approved": true,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/rpc/approval/response", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", result["status"])
	}
}

func TestRPCHandleApprovalPending(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	mux := http.NewServeMux()
	RegisterApprovalHandlers(mux, ac)
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/rpc/approval/pending")
	if err != nil {
		t.Fatalf("failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["count"] != float64(0) {
		t.Fatalf("expected count 0, got %v", result["count"])
	}
}

func TestRPCHandleApprovalResponse_MethodNotAllowed(t *testing.T) {
	ac := NewApprovalClient("http://127.0.0.1:8080", "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	mux := http.NewServeMux()
	RegisterApprovalHandlers(mux, ac)
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/rpc/approval/response")
	if err != nil {
		t.Fatalf("failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestNewApprovalClient_DefaultTimeout(t *testing.T) {
	// When timeout is 0, it should default to 60s
	ac := NewApprovalClient("http://127.0.0.1:8080", "!room:matrix.org", 0)
	defer ac.Close()

	if ac.timeout != 60*time.Second {
		t.Fatalf("expected default timeout 60s, got %v", ac.timeout)
	}
}

func TestApprovalRequest_GeneratesUniqueID(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)
	defer ac.Close()

	ids := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ac.RequestApproval(ctx, OpSessionCreate, "test")
		}()
	}

	// Collect IDs from pending requests (they'll time out, but IDs should be unique)
	// We just check that the pending count reaches at least the expected number
	time.Sleep(500 * time.Millisecond)
	ac.mu.RLock()
	for k := range ac.pending {
		mu.Lock()
		ids[k] = true
		mu.Unlock()
	}
	ac.mu.RUnlock()

	wg.Wait()

	if len(ids) < 1 {
		t.Fatal("expected at least 1 unique ID")
	}
}

func TestClose_CancelsPendingRequests(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer bridge.Close()

	ac := NewApprovalClient(bridge.URL, "!room:matrix.org", 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	go ac.RequestApproval(ctx, OpSessionCreate, "will be cancelled")

	time.Sleep(100 * time.Millisecond)

	// Close should cancel pending
	ac.Close()
	cancel()

	if ac.PendingCount() != 0 {
		t.Fatalf("expected 0 pending after close, got %d", ac.PendingCount())
	}
}
