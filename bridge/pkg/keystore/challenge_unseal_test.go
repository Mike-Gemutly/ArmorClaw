package keystore

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

// TestChallengeBasedUnseal tests the complete challenge-based unseal flow
func TestChallengeBasedUnseal(t *testing.T) {
	// Create challenge manager
	cm, err := NewChallengeManager(ChallengeManagerConfig{
		TTL: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create challenge manager: %v", err)
	}

	// Create sealed keystore with challenge policy
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
		challengeMgr: cm,
	}

	agentID := "test-agent-001"
	reason := "Test unseal request"
	fields := []string{"name", "email"}

	// Step 1: Request unseal (should generate challenge)
	pending, err := sealedKS.RequestUnseal(context.Background(), agentID, reason, fields, "task-001")
	if err != nil {
		t.Fatalf("failed to request unseal: %v", err)
	}

	if pending == nil {
		t.Fatal("expected pending request, got nil")
	}

	if pending.ChallengeID == "" {
		t.Error("expected challenge ID to be set for challenge policy")
	}

	// Step 2: Get the challenge
	challenge, err := sealedKS.GetChallengeForRequest(pending.ID)
	if err != nil {
		t.Fatalf("failed to get challenge: %v", err)
	}

	if challenge.AgentID != agentID {
		t.Errorf("expected agent_id %s, got %s", agentID, challenge.AgentID)
	}

	// Step 3: Generate client key pair
	clientPubKey, clientPrivKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate client key pair: %v", err)
	}

	// Step 4: Build and sign the response
	// Message format: SHA256(nonce || timestamp (as decimal string) || deviceID)
	timestamp := time.Now().Unix()
	deviceID := "device-001"

	h := sha256.New()
	h.Write(challenge.Nonce)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write([]byte(deviceID))
	message := h.Sum(nil)

	signature := ed25519.Sign(clientPrivKey, message)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey,
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// Step 5: Verify and complete unseal
	session, err := sealedKS.VerifyChallengeAndUnseal(context.Background(), response)
	if err != nil {
		t.Fatalf("failed to verify challenge and unseal: %v", err)
	}

	if session == nil {
		t.Fatal("expected session, got nil")
	}

	if session.AgentID != agentID {
		t.Errorf("expected agent_id %s, got %s", agentID, session.AgentID)
	}

	if session.UnsealPolicy != PolicyChallenge {
		t.Errorf("expected policy %s, got %s", PolicyChallenge, session.UnsealPolicy)
	}

	if session.DeviceID != deviceID {
		t.Errorf("expected device_id %s, got %s", deviceID, session.DeviceID)
	}

	// Step 6: Verify keystore is unsealed
	if sealedKS.IsSealed(agentID) {
		t.Error("expected keystore to be unsealed after successful challenge")
	}
}

// TestChallengeBasedUnsealInvalidSignature tests rejection of invalid signatures
func TestChallengeBasedUnsealInvalidSignature(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{TTL: 60 * time.Second})

	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
		challengeMgr: cm,
	}

	// Request unseal
	pending, _ := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	challenge, _ := sealedKS.GetChallengeForRequest(pending.ID)

	// Generate different key pair for signing (not the one we'll verify with)
	_, wrongPrivKey, _ := ed25519.GenerateKey(nil)
	verifyPubKey, _, _ := ed25519.GenerateKey(nil)

	timestamp := time.Now().Unix()
	deviceID := "device-001"

	// Sign with wrong key
	h := sha256.New()
	h.Write(challenge.Nonce)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write([]byte(deviceID))
	message := h.Sum(nil)
	signature := ed25519.Sign(wrongPrivKey, message)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   verifyPubKey, // Different public key
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// Should fail with invalid signature
	_, err := sealedKS.VerifyChallengeAndUnseal(context.Background(), response)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

// TestChallengeBasedUnsealExpired tests handling of expired challenges
func TestChallengeBasedUnsealExpired(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{
		TTL: 1 * time.Millisecond, // Very short TTL
	})

	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
		challengeMgr: cm,
	}

	// Request unseal
	pending, _ := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	challenge, _ := sealedKS.GetChallengeForRequest(pending.ID)

	// Wait for challenge to expire
	time.Sleep(10 * time.Millisecond)

	// Generate valid signature
	clientPubKey, clientPrivKey, _ := ed25519.GenerateKey(nil)
	timestamp := time.Now().Unix()
	deviceID := "device-001"

	h := sha256.New()
	h.Write(challenge.Nonce)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write([]byte(deviceID))
	message := h.Sum(nil)
	signature := ed25519.Sign(clientPrivKey, message)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey,
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// Should fail because challenge expired
	_, err := sealedKS.VerifyChallengeAndUnseal(context.Background(), response)
	if err == nil {
		t.Error("expected error for expired challenge")
	}
}

// TestChallengeBasedUnsealAlreadyUsed tests that challenges can only be used once
func TestChallengeBasedUnsealAlreadyUsed(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{TTL: 60 * time.Second})

	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
		challengeMgr: cm,
	}

	// Request unseal and complete it
	pending, _ := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	challenge, _ := sealedKS.GetChallengeForRequest(pending.ID)

	clientPubKey, clientPrivKey, _ := ed25519.GenerateKey(nil)
	timestamp := time.Now().Unix()
	deviceID := "device-001"

	h := sha256.New()
	h.Write(challenge.Nonce)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write([]byte(deviceID))
	message := h.Sum(nil)
	signature := ed25519.Sign(clientPrivKey, message)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey,
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// First use should succeed
	_, err := sealedKS.VerifyChallengeAndUnseal(context.Background(), response)
	if err != nil {
		t.Fatalf("first unseal failed: %v", err)
	}

	// Second use should fail (challenge already used/deleted)
	_, err = sealedKS.VerifyChallengeAndUnseal(context.Background(), response)
	if err == nil {
		t.Error("expected error for already used challenge")
	}
}

// TestChallengePolicyRequiresChallengeManager tests that challenge policy requires manager
func TestChallengePolicyRequiresChallengeManager(t *testing.T) {
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
		challengeMgr: nil, // No challenge manager
	}

	_, err := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	if err == nil {
		t.Error("expected error when challenge manager not configured")
	}
}

// TestSetChallengeManager tests setting the challenge manager
func TestSetChallengeManager(t *testing.T) {
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyChallenge,
	}

	if sealedKS.GetChallengeManager() != nil {
		t.Error("expected nil challenge manager initially")
	}

	cm, _ := NewChallengeManager(ChallengeManagerConfig{})
	sealedKS.SetChallengeManager(cm)

	if sealedKS.GetChallengeManager() != cm {
		t.Error("challenge manager not set correctly")
	}
}

// TestGetChallengeForRequestNotFound tests error handling for invalid request ID
func TestGetChallengeForRequestNotFound(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		challengeMgr: cm,
	}

	_, err := sealedKS.GetChallengeForRequest("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

// TestMobileApprovalPolicyStillWorks tests that mobile_approval policy still works
func TestMobileApprovalPolicyStillWorks(t *testing.T) {
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyMobileApproval,
	}

	// Request unseal should work without challenge manager
	pending, err := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	if err != nil {
		t.Fatalf("failed to request unseal: %v", err)
	}

	if pending == nil {
		t.Fatal("expected pending request")
	}

	if pending.ChallengeID != "" {
		t.Error("mobile_approval policy should not generate challenge")
	}
}

// TestAutoPolicyStillWorks tests that auto policy still works
func TestAutoPolicyStillWorks(t *testing.T) {
	sealedKS := &SealedKeystore{
		sessions:     make(map[string]*SealedSession),
		agentSession: make(map[string]string),
		pending:      make(map[string]*PendingUnsealRequest),
		defaultTTL:   5 * time.Minute,
		policy:       PolicyAuto,
	}

	// Request unseal should auto-approve
	pending, err := sealedKS.RequestUnseal(context.Background(), "agent-001", "test", nil, "")
	if err != nil {
		t.Fatalf("failed to request unseal: %v", err)
	}

	// Should be auto-unsealed
	if sealedKS.IsSealed("agent-001") {
		t.Error("expected keystore to be unsealed with auto policy")
	}

	_ = pending // May be nil or have request info
}
