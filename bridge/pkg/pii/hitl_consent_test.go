// Package pii provides tests for HITL consent management
package pii

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestHITLConsentManager_RequestAccess(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 30 * time.Second,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
		{Key: "email", Description: "Your email", Required: false},
	})

	request, err := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")
	if err != nil {
		t.Fatalf("RequestAccess failed: %v", err)
	}

	if request == nil {
		t.Fatal("Request should not be nil")
	}

	if request.Status != StatusPending {
		t.Errorf("Expected status 'pending', got '%s'", request.Status)
	}

	if request.SkillID != "skill-123" {
		t.Errorf("Expected skill_id 'skill-123', got '%s'", request.SkillID)
	}

	if request.ProfileID != "profile-456" {
		t.Errorf("Expected profile_id 'profile-456', got '%s'", request.ProfileID)
	}

	if len(request.RequestedFields) != 2 {
		t.Errorf("Expected 2 requested fields, got %d", len(request.RequestedFields))
	}

	if len(request.RequiredFields) != 1 {
		t.Errorf("Expected 1 required field, got %d", len(request.RequiredFields))
	}
}

func TestHITLConsentManager_GetRequest(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// Retrieve the request
	retrieved, err := manager.GetRequest(request.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}

	if retrieved.ID != request.ID {
		t.Error("Retrieved request ID should match")
	}
}

func TestHITLConsentManager_GetRequest_NotFound(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	_, err := manager.GetRequest("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent request")
	}
}

func TestHITLConsentManager_ApproveRequest(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 5 * time.Second,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
		{Key: "email", Description: "Your email", Required: false},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// Start waiting for approval in goroutine
	var wg sync.WaitGroup
	var approvedRequest *AccessRequest
	var waitErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		approvedRequest, waitErr = manager.WaitForApproval(context.Background(), request.ID)
	}()

	// Give time for WaitForApproval to start
	time.Sleep(10 * time.Millisecond)

	// Approve with required fields
	err := manager.ApproveRequest(context.Background(), request.ID, "@user:matrix.example.com", []string{"full_name", "email"})
	if err != nil {
		t.Fatalf("ApproveRequest failed: %v", err)
	}

	wg.Wait()

	if waitErr != nil {
		t.Fatalf("WaitForApproval failed: %v", waitErr)
	}

	// Verify approval (status is updated by WaitForApproval)
	if approvedRequest.Status != StatusApproved {
		t.Errorf("Expected status 'approved', got '%s'", approvedRequest.Status)
	}

	if approvedRequest.ApprovedBy != "@user:matrix.example.com" {
		t.Errorf("Expected approved_by '@user:matrix.example.com', got '%s'", approvedRequest.ApprovedBy)
	}
}

func TestHITLConsentManager_ApproveRequest_MissingRequiredField(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
		{Key: "email", Description: "Your email", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// Try to approve without all required fields
	err := manager.ApproveRequest(context.Background(), request.ID, "@user:matrix.example.com", []string{"full_name"})
	if err == nil {
		t.Error("Expected error when required field not approved")
	}
}

func TestHITLConsentManager_ApproveRequest_AlreadyProcessed(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// First approval
	manager.ApproveRequest(context.Background(), request.ID, "@user:matrix.example.com", []string{"full_name"})

	// Second approval should fail
	err := manager.ApproveRequest(context.Background(), request.ID, "@user2:matrix.example.com", []string{"full_name"})
	if err == nil {
		t.Error("Expected error for already processed request")
	}
}

func TestHITLConsentManager_RejectRequest(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 5 * time.Second,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// Start waiting for approval in goroutine
	var wg sync.WaitGroup
	var rejectedRequest *AccessRequest
	var waitErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		rejectedRequest, waitErr = manager.WaitForApproval(context.Background(), request.ID)
	}()

	// Give time for WaitForApproval to start
	time.Sleep(10 * time.Millisecond)

	// Reject the request
	err := manager.RejectRequest(context.Background(), request.ID, "@user:matrix.example.com", "I don't trust this skill")
	if err != nil {
		t.Fatalf("RejectRequest failed: %v", err)
	}

	wg.Wait()

	if waitErr != nil {
		t.Fatalf("WaitForApproval failed: %v", waitErr)
	}

	// Verify rejection (status is updated by WaitForApproval)
	if rejectedRequest.Status != StatusRejected {
		t.Errorf("Expected status 'rejected', got '%s'", rejectedRequest.Status)
	}

	if rejectedRequest.RejectedBy != "@user:matrix.example.com" {
		t.Errorf("Expected rejected_by '@user:matrix.example.com', got '%s'", rejectedRequest.RejectedBy)
	}

	if rejectedRequest.RejectionReason != "I don't trust this skill" {
		t.Errorf("Expected rejection reason, got '%s'", rejectedRequest.RejectionReason)
	}
}

func TestHITLConsentManager_ListPendingRequests(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 5 * time.Second,
	})

	// Create multiple requests
	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	req1, _ := manager.RequestAccess(context.Background(), manifest, "profile-1", "!room:matrix.example.com")
	req2, _ := manager.RequestAccess(context.Background(), manifest, "profile-2", "!room:matrix.example.com")

	// Initially both should be pending
	pending := manager.ListPendingRequests()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending requests initially, got %d", len(pending))
	}

	// Expire req1 by waiting for it with a short timeout manager
	shortTimeoutManager := NewHITLConsentManager(HITLConfig{Timeout: 50 * time.Millisecond})
	_, _ = shortTimeoutManager.RequestAccess(context.Background(), manifest, "profile-exp", "!room:matrix.example.com")

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Cleanup expired
	shortTimeoutManager.CleanupExpired()

	// List pending should only have req2 since req1 is not on this manager
	pending = shortTimeoutManager.ListPendingRequests()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending requests after cleanup, got %d", len(pending))
	}

	// Verify our original manager still has both
	pending = manager.ListPendingRequests()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending requests on original manager, got %d", len(pending))
	}

	// Verify both IDs are present
	ids := make(map[string]bool)
	for _, p := range pending {
		ids[p.ID] = true
	}
	if !ids[req1.ID] || !ids[req2.ID] {
		t.Error("Both req1 and req2 should be in pending list")
	}
}

func TestHITLConsentManager_ListRequestsByProfile(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	manager.RequestAccess(context.Background(), manifest, "profile-1", "!room:matrix.example.com")
	manager.RequestAccess(context.Background(), manifest, "profile-2", "!room:matrix.example.com")
	manager.RequestAccess(context.Background(), manifest, "profile-1", "!room:matrix.example.com")

	// List requests for profile-1
	requests := manager.ListRequestsByProfile("profile-1")
	if len(requests) != 2 {
		t.Errorf("Expected 2 requests for profile-1, got %d", len(requests))
	}
}

func TestHITLConsentManager_CleanupExpired(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 100 * time.Millisecond,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	manager.RequestAccess(context.Background(), manifest, "profile-1", "!room:matrix.example.com")

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Cleanup should remove expired requests
	count := manager.CleanupExpired()
	if count != 1 {
		t.Errorf("Expected 1 expired request cleaned up, got %d", count)
	}

	// Verify list is empty
	pending := manager.ListPendingRequests()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending requests after cleanup, got %d", len(pending))
	}
}

func TestHITLConsentManager_WaitForApproval(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 5 * time.Second,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	var wg sync.WaitGroup
	var approvedRequest *AccessRequest
	var waitErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		approvedRequest, waitErr = manager.WaitForApproval(context.Background(), request.ID)
	}()

	// Approve after a short delay
	time.Sleep(50 * time.Millisecond)
	manager.ApproveRequest(context.Background(), request.ID, "@user:matrix.example.com", []string{"full_name"})

	wg.Wait()

	if waitErr != nil {
		t.Fatalf("WaitForApproval failed: %v", waitErr)
	}

	if approvedRequest.Status != StatusApproved {
		t.Errorf("Expected status 'approved', got '%s'", approvedRequest.Status)
	}
}

func TestHITLConsentManager_WaitForApproval_Timeout(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 100 * time.Millisecond,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	request, _ := manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	// Wait for approval without approving - should timeout
	_, err := manager.WaitForApproval(context.Background(), request.ID)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !errors.Is(err, ErrApprovalTimeout) {
		t.Errorf("Expected ErrApprovalTimeout, got: %v", err)
	}
}

func TestHITLConsentManager_RequestAccessAndWait(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{
		Timeout: 5 * time.Second,
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	var wg sync.WaitGroup
	var result *AccessRequest
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err = manager.RequestAccessAndWait(context.Background(), manifest, "profile-456", "!room:matrix.example.com")
	}()

	// Wait for request to be created, then approve
	time.Sleep(50 * time.Millisecond)
	pending := manager.ListPendingRequests()
	if len(pending) > 0 {
		manager.ApproveRequest(context.Background(), pending[0].ID, "@user:matrix.example.com", []string{"full_name"})
	}

	wg.Wait()

	if err != nil {
		t.Fatalf("RequestAccessAndWait failed: %v", err)
	}

	if result.Status != StatusApproved {
		t.Errorf("Expected status 'approved', got '%s'", result.Status)
	}
}

func TestAccessRequest_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		request := &AccessRequest{
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if request.IsExpired() {
			t.Error("Request should not be expired")
		}
	})

	t.Run("expired", func(t *testing.T) {
		request := &AccessRequest{
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		if !request.IsExpired() {
			t.Error("Request should be expired")
		}
	})
}

func TestAccessRequest_IsPending(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		request := &AccessRequest{
			Status:    StatusPending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if !request.IsPending() {
			t.Error("Request should be pending")
		}
	})

	t.Run("approved", func(t *testing.T) {
		request := &AccessRequest{
			Status:    StatusApproved,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if request.IsPending() {
			t.Error("Approved request should not be pending")
		}
	})
}

func TestHITLConsentManager_NotifyCallback(t *testing.T) {
	var callbackCalled bool
	var receivedRequest *AccessRequest

	manager := NewHITLConsentManager(HITLConfig{
		NotifyCallback: func(ctx context.Context, request *AccessRequest) error {
			callbackCalled = true
			receivedRequest = request
			return nil
		},
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	if !callbackCalled {
		t.Error("NotifyCallback should have been called")
	}

	if receivedRequest == nil {
		t.Fatal("Received request should not be nil")
	}

	if receivedRequest.SkillID != "skill-123" {
		t.Errorf("Expected skill_id 'skill-123', got '%s'", receivedRequest.SkillID)
	}
}

func TestHITLConsentManager_SetNotifyCallback(t *testing.T) {
	manager := NewHITLConsentManager(HITLConfig{})

	var callbackCalled bool

	manager.SetNotifyCallback(func(ctx context.Context, request *AccessRequest) error {
		callbackCalled = true
		return nil
	})

	manifest := NewSkillManifest("skill-123", "Test Skill", []VariableRequest{
		{Key: "full_name", Description: "Your name", Required: true},
	})

	manager.RequestAccess(context.Background(), manifest, "profile-456", "!room:matrix.example.com")

	if !callbackCalled {
		t.Error("SetNotifyCallback should have set the callback")
	}
}
