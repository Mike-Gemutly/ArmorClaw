package keystore

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestNewChallengeManager(t *testing.T) {
	cm, err := NewChallengeManager(ChallengeManagerConfig{})
	if err != nil {
		t.Fatalf("failed to create challenge manager: %v", err)
	}

	if cm == nil {
		t.Fatal("expected non-nil challenge manager")
	}

	if cm.serverPublicKey == nil {
		t.Error("expected server public key to be generated")
	}
}

func TestNewChallengeManagerWithKey(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(nil)

	cm, err := NewChallengeManager(ChallengeManagerConfig{
		ServerPrivateKey: privKey,
	})
	if err != nil {
		t.Fatalf("failed to create challenge manager: %v", err)
	}

	if !equalPublicKeys(cm.serverPublicKey, pubKey) {
		t.Error("expected server public key to match provided key")
	}
}

func TestGenerateChallenge(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	challenge, err := cm.GenerateChallenge("agent-001", "test reason", []string{"name", "email"})
	if err != nil {
		t.Fatalf("failed to generate challenge: %v", err)
	}

	if challenge.ID == "" {
		t.Error("expected challenge ID")
	}
	if len(challenge.Nonce) != 32 {
		t.Errorf("expected 32-byte nonce, got %d", len(challenge.Nonce))
	}
	if challenge.AgentID != "agent-001" {
		t.Errorf("expected agent_id 'agent-001', got %s", challenge.AgentID)
	}
	if challenge.Reason != "test reason" {
		t.Errorf("expected reason 'test reason', got %s", challenge.Reason)
	}
	if len(challenge.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(challenge.Fields))
	}
	if challenge.Used {
		t.Error("expected challenge to not be used initially")
	}
}

func TestVerifyChallenge(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	// Generate challenge
	challenge, _ := cm.GenerateChallenge("agent-001", "test", nil)

	// Generate client key pair
	clientPubKey, clientPrivKey, _ := ed25519.GenerateKey(nil)

	// Build signature message
	timestamp := time.Now().Unix()
	deviceID := "device-001"
	message := cm.buildSignatureMessage(challenge.Nonce, timestamp, deviceID)

	// Sign the message
	signature := ed25519.Sign(clientPrivKey, message)

	// Create response
	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey,
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// Verify
	verified, err := cm.VerifyChallenge(response)
	if err != nil {
		t.Fatalf("failed to verify challenge: %v", err)
	}

	if verified.ID != challenge.ID {
		t.Errorf("expected challenge ID %s, got %s", challenge.ID, verified.ID)
	}
}

func TestVerifyChallengeInvalidSignature(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	challenge, _ := cm.GenerateChallenge("agent-001", "test", nil)

	// Generate different key pair for signing
	_, wrongPrivKey, _ := ed25519.GenerateKey(nil)

	timestamp := time.Now().Unix()
	deviceID := "device-001"
	message := cm.buildSignatureMessage(challenge.Nonce, timestamp, deviceID)
	signature := ed25519.Sign(wrongPrivKey, message)

	// Use a different public key for verification
	clientPubKey, _, _ := ed25519.GenerateKey(nil)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey, // Different public key
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	_, err := cm.VerifyChallenge(response)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyChallengeNotFound(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	clientPubKey, _, _ := ed25519.GenerateKey(nil)

	response := &ChallengeResponse{
		ChallengeID: "nonexistent",
		Signature:   make([]byte, 64),
		PublicKey:   clientPubKey,
		Timestamp:   time.Now().Unix(),
		DeviceID:    "device-001",
	}

	_, err := cm.VerifyChallenge(response)
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestVerifyChallengeAlreadyUsed(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	challenge, _ := cm.GenerateChallenge("agent-001", "test", nil)
	clientPubKey, clientPrivKey, _ := ed25519.GenerateKey(nil)
	_ = clientPrivKey // Use it

	timestamp := time.Now().Unix()
	deviceID := "device-001"
	message := cm.buildSignatureMessage(challenge.Nonce, timestamp, deviceID)
	signature := ed25519.Sign(clientPrivKey, message)

	response := &ChallengeResponse{
		ChallengeID: challenge.ID,
		Signature:   signature,
		PublicKey:   clientPubKey,
		Timestamp:   timestamp,
		DeviceID:    deviceID,
	}

	// First verification should succeed
	_, err := cm.VerifyChallenge(response)
	if err != nil {
		t.Fatalf("first verification failed: %v", err)
	}

	// Second verification should fail
	_, err = cm.VerifyChallenge(response)
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound (challenge was deleted), got %v", err)
	}
}

func TestGetChallenge(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	challenge, _ := cm.GenerateChallenge("agent-001", "test", nil)

	retrieved, err := cm.GetChallenge(challenge.ID)
	if err != nil {
		t.Fatalf("failed to get challenge: %v", err)
	}

	if retrieved.ID != challenge.ID {
		t.Errorf("expected ID %s, got %s", challenge.ID, retrieved.ID)
	}
}

func TestGetChallengeNotFound(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	_, err := cm.GetChallenge("nonexistent")
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestRegisterDevice(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	pubKey, _, _ := ed25519.GenerateKey(nil)

	err := cm.RegisterDevice("device-001", pubKey)
	if err != nil {
		t.Fatalf("failed to register device: %v", err)
	}

	retrieved, exists := cm.GetDevicePublicKey("device-001")
	if !exists {
		t.Error("expected device to be registered")
	}
	if !equalPublicKeys(retrieved, pubKey) {
		t.Error("expected public key to match")
	}
}

func TestRegisterDeviceInvalidKey(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	err := cm.RegisterDevice("device-001", []byte{1, 2, 3}) // Invalid key
	if err != ErrInvalidPublicKey {
		t.Errorf("expected ErrInvalidPublicKey, got %v", err)
	}
}

func TestUnregisterDevice(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	pubKey, _, _ := ed25519.GenerateKey(nil)
	cm.RegisterDevice("device-001", pubKey)

	cm.UnregisterDevice("device-001")

	_, exists := cm.GetDevicePublicKey("device-001")
	if exists {
		t.Error("expected device to be unregistered")
	}
}

func TestChallengeCleanupExpired(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{
		TTL: 1 * time.Millisecond, // Very short TTL
	})

	cm.GenerateChallenge("agent-001", "test", nil)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	count := cm.CleanupExpired()
	if count < 1 {
		t.Errorf("expected at least 1 expired challenge cleaned, got %d", count)
	}
}

func TestGetServerPublicKey(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	pubKey := cm.GetServerPublicKey()
	if pubKey == nil {
		t.Error("expected non-nil public key")
	}
	if len(pubKey) != ed25519.PublicKeySize {
		t.Errorf("expected %d-byte public key, got %d", ed25519.PublicKeySize, len(pubKey))
	}
}

func TestSignWithServerKey(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	message := []byte("test message")
	signature := cm.SignWithServerKey(message)

	if len(signature) != ed25519.SignatureSize {
		t.Errorf("expected %d-byte signature, got %d", ed25519.SignatureSize, len(signature))
	}

	// Verify signature
	if !ed25519.Verify(cm.serverPublicKey, message, signature) {
		t.Error("signature verification failed")
	}
}

func TestChallengeToMatrixEvent(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})
	challenge, _ := cm.GenerateChallenge("agent-001", "test reason", []string{"name"})

	event := challenge.ToMatrixEvent()

	if event["type"] != "com.armorclaw.keystore.unseal_challenge" {
		t.Errorf("unexpected event type: %v", event["type"])
	}
	if event["challenge_id"] != challenge.ID {
		t.Errorf("unexpected challenge_id: %v", event["challenge_id"])
	}
	if event["agent_id"] != "agent-001" {
		t.Errorf("unexpected agent_id: %v", event["agent_id"])
	}
	if event["reason"] != "test reason" {
		t.Errorf("unexpected reason: %v", event["reason"])
	}
}

func TestChallengeMarshalJSON(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})
	challenge, _ := cm.GenerateChallenge("agent-001", "test", nil)

	data, err := json.Marshal(challenge)
	if err != nil {
		t.Fatalf("failed to marshal challenge: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["id"] != challenge.ID {
		t.Errorf("expected id %s, got %v", challenge.ID, parsed["id"])
	}
	if parsed["agent_id"] != "agent-001" {
		t.Errorf("expected agent_id 'agent-001', got %v", parsed["agent_id"])
	}
}

func TestChallengeResponseUnmarshalJSON(t *testing.T) {
	clientPubKey, clientPrivKey, _ := ed25519.GenerateKey(nil)

	sig := ed25519.Sign(clientPrivKey, []byte("test"))

	jsonData := map[string]interface{}{
		"challenge_id": "chal_abc123",
		"signature":    base64.RawURLEncoding.EncodeToString(sig),
		"public_key":   base64.RawURLEncoding.EncodeToString(clientPubKey),
		"timestamp":    1234567890,
		"device_id":    "device-001",
	}

	data, _ := json.Marshal(jsonData)

	var response ChallengeResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.ChallengeID != "chal_abc123" {
		t.Errorf("expected challenge_id 'chal_abc123', got %s", response.ChallengeID)
	}
	if response.DeviceID != "device-001" {
		t.Errorf("expected device_id 'device-001', got %s", response.DeviceID)
	}
	if response.Timestamp != 1234567890 {
		t.Errorf("expected timestamp 1234567890, got %d", response.Timestamp)
	}
}

func TestListPendingChallenges(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	cm.GenerateChallenge("agent-001", "test 1", nil)
	cm.GenerateChallenge("agent-001", "test 2", nil)
	cm.GenerateChallenge("agent-002", "test 3", nil)

	// List all
	all := cm.ListPendingChallenges("")
	if len(all) != 3 {
		t.Errorf("expected 3 challenges, got %d", len(all))
	}

	// List by agent
	agent1 := cm.ListPendingChallenges("agent-001")
	if len(agent1) != 2 {
		t.Errorf("expected 2 challenges for agent-001, got %d", len(agent1))
	}
}

func TestGetStats(t *testing.T) {
	cm, _ := NewChallengeManager(ChallengeManagerConfig{})

	cm.GenerateChallenge("agent-001", "test", nil)
	cm.RegisterDevice("device-001", generateTestPublicKey())

	stats := cm.GetStats()

	if stats["pending_challenges"].(int) != 1 {
		t.Errorf("expected 1 pending challenge, got %v", stats["pending_challenges"])
	}
	if stats["registered_devices"].(int) != 1 {
		t.Errorf("expected 1 registered device, got %v", stats["registered_devices"])
	}
}

func generateTestPublicKey() ed25519.PublicKey {
	pubKey, _, _ := ed25519.GenerateKey(nil)
	return pubKey
}
