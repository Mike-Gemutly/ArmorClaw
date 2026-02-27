package keystore

import (
	"context"
	"testing"
	"time"
)

// TestNewPIIRequestManager tests manager creation
func TestNewPIIRequestManager(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
	if mgr.requests == nil {
		t.Error("expected requests map to be initialized")
	}
}

// TestCreatePIIRequest tests request creation
func TestCreatePIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	fields := []PIIFieldRequest{
		{Key: "email", DisplayName: "Email Address", Required: true, Sensitive: false},
		{Key: "card_number", DisplayName: "Credit Card", Required: false, Sensitive: true},
	}

	req, err := mgr.CreateRequest(
		context.Background(),
		"agent-001",
		"skill-001",
		"Flight Booking",
		"profile-001",
		fields,
		"Booking flight to NYC",
		"room-001",
		5*time.Minute,
	)

	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if req.ID == "" {
		t.Error("expected request ID to be set")
	}
	if req.AgentID != "agent-001" {
		t.Errorf("expected agent ID agent-001, got %s", req.AgentID)
	}
	if req.Status != StatusPending {
		t.Errorf("expected status pending, got %s", req.Status)
	}
	if len(req.RequestedFields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(req.RequestedFields))
	}
}

// TestGetPIIRequest tests request retrieval
func TestGetPIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-002",
		"skill-002",
		"Test Skill",
		"profile-002",
		[]PIIFieldRequest{{Key: "name", DisplayName: "Name"}},
		"Testing",
		"room-002",
		5*time.Minute,
	)

	retrieved, err := mgr.GetRequest(created.ID)
	if err != nil {
		t.Fatalf("failed to get request: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}

	// Test non-existent request
	_, err = mgr.GetRequest("non-existent")
	if err != ErrRequestNotFound {
		t.Errorf("expected ErrRequestNotFound, got %v", err)
	}
}

// TestApprovePIIRequest tests request approval
func TestApprovePIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-003",
		"skill-003",
		"Test Skill",
		"profile-003",
		[]PIIFieldRequest{
			{Key: "email", DisplayName: "Email"},
			{Key: "name", DisplayName: "Name"},
		},
		"Testing",
		"room-003",
		5*time.Minute,
	)

	approved, err := mgr.ApproveRequest(
		context.Background(),
		created.ID,
		"user-001",
		[]string{"email"},
	)

	if err != nil {
		t.Fatalf("failed to approve request: %v", err)
	}

	if approved.Status != StatusApproved {
		t.Errorf("expected status approved, got %s", approved.Status)
	}
	if approved.ApprovedBy != "user-001" {
		t.Errorf("expected approved_by user-001, got %s", approved.ApprovedBy)
	}
	if len(approved.ApprovedFields) != 1 {
		t.Errorf("expected 1 approved field, got %d", len(approved.ApprovedFields))
	}
	if approved.ApprovedAt == nil {
		t.Error("expected approved_at to be set")
	}
}

// TestDenyPIIRequest tests request denial
func TestDenyPIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-004",
		"skill-004",
		"Test Skill",
		"profile-004",
		[]PIIFieldRequest{{Key: "ssn", DisplayName: "SSN", Sensitive: true}},
		"Testing",
		"room-004",
		5*time.Minute,
	)

	denied, err := mgr.DenyRequest(
		context.Background(),
		created.ID,
		"user-002",
		"SSN access not allowed",
	)

	if err != nil {
		t.Fatalf("failed to deny request: %v", err)
	}

	if denied.Status != StatusDenied {
		t.Errorf("expected status denied, got %s", denied.Status)
	}
	if denied.DeniedBy != "user-002" {
		t.Errorf("expected denied_by user-002, got %s", denied.DeniedBy)
	}
	if denied.DenyReason != "SSN access not allowed" {
		t.Errorf("expected deny reason, got %s", denied.DenyReason)
	}
	if denied.DeniedAt == nil {
		t.Error("expected denied_at to be set")
	}
}

// TestCancelPIIRequest tests request cancellation
func TestCancelPIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-005",
		"skill-005",
		"Test Skill",
		"profile-005",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-005",
		5*time.Minute,
	)

	err := mgr.CancelRequest(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to cancel request: %v", err)
	}

	retrieved, _ := mgr.GetRequest(created.ID)
	if retrieved.Status != StatusCancelled {
		t.Errorf("expected status cancelled, got %s", retrieved.Status)
	}
}

// TestFulfillPIIRequest tests request fulfillment
func TestFulfillPIIRequest(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-006",
		"skill-006",
		"Test Skill",
		"profile-006",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-006",
		5*time.Minute,
	)

	// First approve the request
	mgr.ApproveRequest(context.Background(), created.ID, "user-003", []string{"email"})

	// Then fulfill it
	err := mgr.FulfillRequest(context.Background(), created.ID, map[string]string{
		"email": "test@example.com",
	})

	if err != nil {
		t.Fatalf("failed to fulfill request: %v", err)
	}

	retrieved, _ := mgr.GetRequest(created.ID)
	if retrieved.Status != StatusFulfilled {
		t.Errorf("expected status fulfilled, got %s", retrieved.Status)
	}
	if retrieved.FulfilledAt == nil {
		t.Error("expected fulfilled_at to be set")
	}
}

// TestFulfillWithoutApproval tests fulfillment fails without approval
func TestFulfillWithoutApproval(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-007",
		"skill-007",
		"Test Skill",
		"profile-007",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-007",
		5*time.Minute,
	)

	// Try to fulfill without approval
	err := mgr.FulfillRequest(context.Background(), created.ID, map[string]string{
		"email": "test@example.com",
	})

	if err == nil {
		t.Error("expected error when fulfilling non-approved request")
	}
}

// TestPIIRequestExpiration tests request expiration
func TestPIIRequestExpiration(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	// Create a request that expires in 1 millisecond
	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-008",
		"skill-008",
		"Test Skill",
		"profile-008",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-008",
		1*time.Millisecond,
	)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	if !created.IsExpired() {
		t.Error("expected request to be expired")
	}

	// Try to approve expired request
	_, err := mgr.ApproveRequest(context.Background(), created.ID, "user-004", []string{"email"})
	if err != ErrRequestExpired {
		t.Errorf("expected ErrRequestExpired, got %v", err)
	}
}

// TestPIIRequestIsClosed tests IsClosed check
func TestPIIRequestIsClosed(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	// Create request
	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-009",
		"skill-009",
		"Test Skill",
		"profile-009",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-009",
		5*time.Minute,
	)

	if created.IsClosed() {
		t.Error("pending request should not be closed")
	}

	// Approve it
	mgr.ApproveRequest(context.Background(), created.ID, "user-005", []string{"email"})

	if !created.IsClosed() {
		t.Error("approved request should be closed")
	}
}

// TestListPendingPIIRequests tests listing pending requests
func TestListPendingPIIRequests(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	// Create multiple requests
	for i := 0; i < 3; i++ {
		mgr.CreateRequest(
			context.Background(),
			"agent-010",
			"skill-010",
			"Test Skill",
			"profile-010",
			[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
			"Testing",
			"room-010",
			5*time.Minute,
		)
	}

	// Approve one
	allPending := mgr.ListPending()
	if len(allPending) != 3 {
		t.Errorf("expected 3 pending requests, got %d", len(allPending))
	}

	mgr.ApproveRequest(context.Background(), allPending[0].ID, "user-006", []string{"email"})

	pending := mgr.ListPending()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending requests after approval, got %d", len(pending))
	}
}

// TestListPIIRequestsByAgent tests listing requests by agent
func TestListPIIRequestsByAgent(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	// Create requests for different agents
	mgr.CreateRequest(
		context.Background(),
		"agent-alice",
		"skill-011",
		"Test Skill",
		"profile-011",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-011",
		5*time.Minute,
	)
	mgr.CreateRequest(
		context.Background(),
		"agent-bob",
		"skill-012",
		"Test Skill",
		"profile-012",
		[]PIIFieldRequest{{Key: "name", DisplayName: "Name"}},
		"Testing",
		"room-012",
		5*time.Minute,
	)
	mgr.CreateRequest(
		context.Background(),
		"agent-alice",
		"skill-013",
		"Test Skill",
		"profile-013",
		[]PIIFieldRequest{{Key: "phone", DisplayName: "Phone"}},
		"Testing",
		"room-013",
		5*time.Minute,
	)

	aliceRequests := mgr.ListByAgent("agent-alice")
	if len(aliceRequests) != 2 {
		t.Errorf("expected 2 requests for agent-alice, got %d", len(aliceRequests))
	}

	bobRequests := mgr.ListByAgent("agent-bob")
	if len(bobRequests) != 1 {
		t.Errorf("expected 1 request for agent-bob, got %d", len(bobRequests))
	}
}

// TestPIIRequestStats tests statistics
func TestPIIRequestStats(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	// Create and process several requests
	req1, _ := mgr.CreateRequest(
		context.Background(),
		"agent-014",
		"skill-014",
		"Test Skill",
		"profile-014",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-014",
		5*time.Minute,
	)
	req2, _ := mgr.CreateRequest(
		context.Background(),
		"agent-015",
		"skill-015",
		"Test Skill",
		"profile-015",
		[]PIIFieldRequest{{Key: "name", DisplayName: "Name"}},
		"Testing",
		"room-015",
		5*time.Minute,
	)
	_, _ = mgr.CreateRequest(
		context.Background(),
		"agent-016",
		"skill-016",
		"Test Skill",
		"profile-016",
		[]PIIFieldRequest{{Key: "phone", DisplayName: "Phone"}},
		"Testing",
		"room-016",
		5*time.Minute,
	)

	// Approve one
	mgr.ApproveRequest(context.Background(), req1.ID, "user-007", []string{"email"})
	// Deny one
	mgr.DenyRequest(context.Background(), req2.ID, "user-008", "Not allowed")

	stats := mgr.GetStats()

	if stats["total"] != 3 {
		t.Errorf("expected total 3, got %d", stats["total"])
	}
	if stats["approved"] != 1 {
		t.Errorf("expected approved 1, got %d", stats["approved"])
	}
	if stats["denied"] != 1 {
		t.Errorf("expected denied 1, got %d", stats["denied"])
	}
	if stats["pending"] != 1 {
		t.Errorf("expected pending 1, got %d", stats["pending"])
	}
}

// TestPIIRequestToMatrixEvent tests Matrix event conversion
func TestPIIRequestToMatrixEvent(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-017",
		"skill-017",
		"Flight Booking",
		"profile-017",
		[]PIIFieldRequest{
			{Key: "email", DisplayName: "Email", Required: true},
			{Key: "passport", DisplayName: "Passport", Sensitive: true},
		},
		"Booking flight to Paris",
		"room-017",
		5*time.Minute,
	)

	event := created.ToMatrixEvent()

	if event["request_id"] != created.ID {
		t.Errorf("expected request_id, got %v", event["request_id"])
	}
	if event["skill_name"] != "Flight Booking" {
		t.Errorf("expected skill_name, got %v", event["skill_name"])
	}
	if event["context"] != "Booking flight to Paris" {
		t.Errorf("expected context, got %v", event["context"])
	}

	fields, ok := event["requested_fields"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected requested_fields to be a slice")
	}
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}

// TestApproveAlreadyClosed tests approval of already closed request
func TestApproveAlreadyClosed(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	created, _ := mgr.CreateRequest(
		context.Background(),
		"agent-018",
		"skill-018",
		"Test Skill",
		"profile-018",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-018",
		5*time.Minute,
	)

	// Approve once
	mgr.ApproveRequest(context.Background(), created.ID, "user-009", []string{"email"})

	// Try to approve again
	_, err := mgr.ApproveRequest(context.Background(), created.ID, "user-010", []string{"email"})
	if err != ErrRequestAlreadyClosed {
		t.Errorf("expected ErrRequestAlreadyClosed, got %v", err)
	}
}

// TestCallbackInvocation tests that callbacks are invoked
func TestCallbackInvocation(t *testing.T) {
	mgr := NewPIIRequestManager(PIIRequestManagerConfig{})

	createdCalled := false
	approvedCalled := false
	deniedCalled := false

	mgr.SetCallbacks(
		func(ctx context.Context, req *PIIRequest) error {
			createdCalled = true
			return nil
		},
		func(ctx context.Context, req *PIIRequest) error {
			approvedCalled = true
			return nil
		},
		func(ctx context.Context, req *PIIRequest) error {
			deniedCalled = true
			return nil
		},
		nil,
	)

	req, _ := mgr.CreateRequest(
		context.Background(),
		"agent-019",
		"skill-019",
		"Test Skill",
		"profile-019",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-019",
		5*time.Minute,
	)

	if !createdCalled {
		t.Error("expected create callback to be invoked")
	}

	mgr.ApproveRequest(context.Background(), req.ID, "user-011", []string{"email"})

	if !approvedCalled {
		t.Error("expected approve callback to be invoked")
	}

	req2, _ := mgr.CreateRequest(
		context.Background(),
		"agent-020",
		"skill-020",
		"Test Skill",
		"profile-020",
		[]PIIFieldRequest{{Key: "email", DisplayName: "Email"}},
		"Testing",
		"room-020",
		5*time.Minute,
	)

	mgr.DenyRequest(context.Background(), req2.ID, "user-012", "Denied")

	if !deniedCalled {
		t.Error("expected deny callback to be invoked")
	}
}
