package email

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

func newTestApprovalManager(timeout time.Duration) *EmailApprovalManager {
	log, _ := logger.New(logger.Config{Output: "stdout"})
	return NewEmailApprovalManager(EmailApprovalConfig{
		Timeout:       timeout,
		Log:           log,
		SendMatrixMsg: func(roomID, eventType, body string) error { return nil },
	})
}

func TestApproval_RequestAndApprove(t *testing.T) {
	mgr := newTestApprovalManager(5 * time.Second)

	var approvalID string
	var wg sync.WaitGroup
	var result *ApprovalDecision

	wg.Add(1)
	go func() {
		defer wg.Done()
		req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e1"}
		decision, err := mgr.RequestApproval(context.Background(), req)
		if err != nil {
			t.Logf("RequestApproval error: %v", err)
			return
		}
		result = decision
	}()

	time.Sleep(50 * time.Millisecond)

	mgr.mu.RLock()
	for id := range mgr.pending {
		approvalID = id
		break
	}
	mgr.mu.RUnlock()

	if approvalID == "" {
		t.Fatal("no pending approval found")
	}

	err := mgr.HandleApprovalResponse(approvalID, true, "@user:test.com")
	if err != nil {
		t.Fatalf("HandleApprovalResponse: %v", err)
	}

	wg.Wait()

	if result == nil {
		t.Fatal("no decision received")
	}
	if !result.Approved {
		t.Error("expected approval")
	}
	if result.ApprovedBy != "@user:test.com" {
		t.Errorf("ApprovedBy = %q", result.ApprovedBy)
	}
}

func TestApproval_RequestAndDeny(t *testing.T) {
	mgr := newTestApprovalManager(5 * time.Second)

	var approvalID string
	var wg sync.WaitGroup
	var result *ApprovalDecision

	wg.Add(1)
	go func() {
		defer wg.Done()
		req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e2"}
		decision, _ := mgr.RequestApproval(context.Background(), req)
		result = decision
	}()

	time.Sleep(50 * time.Millisecond)

	mgr.mu.RLock()
	for id := range mgr.pending {
		approvalID = id
		break
	}
	mgr.mu.RUnlock()

	mgr.HandleApprovalResponse(approvalID, false, "@admin:test.com")

	wg.Wait()

	if result.Approved {
		t.Error("expected denial")
	}
	if result.ApprovedBy != "@admin:test.com" {
		t.Errorf("ApprovedBy = %q", result.ApprovedBy)
	}
}

func TestApproval_TimeoutAutoReject(t *testing.T) {
	mgr := newTestApprovalManager(100 * time.Millisecond)

	req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e3"}
	decision, err := mgr.RequestApproval(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if decision.Approved {
		t.Error("expected auto-reject on timeout")
	}
}

func TestApproval_HandleExpiredApproval(t *testing.T) {
	mgr := newTestApprovalManager(50 * time.Millisecond)

	req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e4"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mgr.RequestApproval(ctx, req)

	time.Sleep(100 * time.Millisecond)

	err := mgr.HandleApprovalResponse("nonexistent-id", true, "@user:test.com")
	if err == nil {
		t.Fatal("expected error for expired/unknown approval")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want not found", err)
	}
}

func TestApproval_PendingCount(t *testing.T) {
	mgr := newTestApprovalManager(5 * time.Second)

	if mgr.PendingCount() != 0 {
		t.Errorf("PendingCount = %d, want 0", mgr.PendingCount())
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e-count"}
			mgr.RequestApproval(ctx, req)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	count := mgr.PendingCount()
	if count != 3 {
		t.Errorf("PendingCount = %d, want 3", count)
	}

	cancel()
	wg.Wait()
}

func TestApproval_ContextCancellation(t *testing.T) {
	mgr := newTestApprovalManager(30 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	var resultErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := &OutboundRequest{To: "test@test.com", From: "from@test.com", EmailID: "e-cancel"}
		_, resultErr = mgr.RequestApproval(ctx, req)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	if resultErr == nil {
		t.Error("expected context cancelled error")
	}
	if resultErr != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", resultErr)
	}
}

func TestApproval_DefaultTimeout(t *testing.T) {
	mgr := NewEmailApprovalManager(EmailApprovalConfig{
		Log:           nil,
		SendMatrixMsg: func(string, string, string) error { return nil },
	})
	if mgr.timeout != defaultApprovalTimeout {
		t.Errorf("timeout = %v, want %v", mgr.timeout, defaultApprovalTimeout)
	}
}

func TestApproval_MatrixMessage(t *testing.T) {
	var capturedBody string
	mgr := NewEmailApprovalManager(EmailApprovalConfig{
		Timeout: 5 * time.Second,
		SendMatrixMsg: func(roomID, eventType, body string) error {
			capturedBody = body
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := &OutboundRequest{
		To:        "test@test.com",
		From:      "from@test.com",
		EmailID:   "e-matrix",
		PIIFields: []string{"ssn"},
	}
	mgr.RequestApproval(ctx, req)

	if capturedBody == "" {
		t.Fatal("expected Matrix message to be sent")
	}
	if !strings.Contains(capturedBody, "approval_") {
		t.Errorf("body missing approval ID: %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "e-matrix") {
		t.Errorf("body missing email ID: %q", capturedBody)
	}
}
