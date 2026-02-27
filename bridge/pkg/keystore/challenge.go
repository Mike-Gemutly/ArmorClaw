// Package keystore provides challenge-response protocol for secure keystore unsealing.
// This implements a zero-trust model where the mobile device must prove possession
// of the private key before the keystore can be unsealed.
package keystore

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Challenge errors
var (
	ErrChallengeExpired     = errors.New("challenge has expired")
	ErrChallengeNotFound    = errors.New("challenge not found")
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrInvalidPublicKey     = errors.New("invalid public key")
	ErrChallengeAlreadyUsed = errors.New("challenge already used")
)

// ChallengeTTL is the default time-to-live for challenges (60 seconds)
const ChallengeTTL = 60 * time.Second

// Challenge represents a pending unseal challenge
type Challenge struct {
	// ID is the unique challenge identifier
	ID string `json:"id"`

	// Nonce is a random 32-byte value that must be signed
	Nonce []byte `json:"nonce"`

	// ServerPublicKey is the bridge's Ed25519 public key for verification
	ServerPublicKey ed25519.PublicKey `json:"server_public_key"`

	// CreatedAt is when the challenge was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the challenge expires
	ExpiresAt time.Time `json:"expires_at"`

	// AgentID is the agent requesting unseal
	AgentID string `json:"agent_id"`

	// Reason is why unseal is being requested
	Reason string `json:"reason"`

	// Fields being requested (for user approval display)
	Fields []string `json:"fields,omitempty"`

	// Used indicates if challenge was already used
	Used bool `json:"-"`

	// PublicKey of the client that should respond (optional, for binding)
	ExpectedPublicKey ed25519.PublicKey `json:"-"`
}

// ChallengeResponse represents a response to a challenge
type ChallengeResponse struct {
	// ChallengeID is the ID of the challenge being responded to
	ChallengeID string `json:"challenge_id"`

	// Signature is the Ed25519 signature of the nonce
	Signature []byte `json:"signature"`

	// PublicKey is the client's public key
	PublicKey ed25519.PublicKey `json:"public_key"`

	// WrappedKEK is the encrypted key encryption key (optional, for key exchange)
	WrappedKEK []byte `json:"wrapped_kek,omitempty"`

	// Timestamp is when the response was created
	Timestamp int64 `json:"timestamp"`

	// DeviceID identifies the responding device
	DeviceID string `json:"device_id,omitempty"`
}

// ChallengeManager manages challenge generation and verification
type ChallengeManager struct {
	mu         sync.RWMutex
	challenges map[string]*Challenge

	// Server key pair for signing/verification
	serverPrivateKey ed25519.PrivateKey
	serverPublicKey  ed25519.PublicKey

	// TTL for challenges
	ttl time.Duration

	// Registered client public keys (device ID -> public key)
	registeredKeys map[string]ed25519.PublicKey
}

// ChallengeManagerConfig configures the challenge manager
type ChallengeManagerConfig struct {
	// ServerPrivateKey is the Ed25519 private key (generated if nil)
	ServerPrivateKey ed25519.PrivateKey

	// TTL is the challenge time-to-live (default: 60 seconds)
	TTL time.Duration
}

// NewChallengeManager creates a new challenge manager
func NewChallengeManager(cfg ChallengeManagerConfig) (*ChallengeManager, error) {
	if cfg.TTL == 0 {
		cfg.TTL = ChallengeTTL
	}

	var privateKey ed25519.PrivateKey
	var publicKey ed25519.PublicKey

	if cfg.ServerPrivateKey != nil {
		privateKey = cfg.ServerPrivateKey
		publicKey = privateKey.Public().(ed25519.PublicKey)
	} else {
		// Generate new key pair
		var err error
		publicKey, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %w", err)
		}
	}

	return &ChallengeManager{
		challenges:      make(map[string]*Challenge),
		serverPrivateKey: privateKey,
		serverPublicKey:  publicKey,
		ttl:             cfg.TTL,
		registeredKeys:  make(map[string]ed25519.PublicKey),
	}, nil
}

// GenerateChallenge creates a new challenge for unseal request
func (cm *ChallengeManager) GenerateChallenge(agentID, reason string, fields []string) (*Challenge, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate challenge ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("failed to generate challenge ID: %w", err)
	}
	challengeID := "chal_" + base64.RawURLEncoding.EncodeToString(idBytes)

	// Generate nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	now := time.Now()
	challenge := &Challenge{
		ID:             challengeID,
		Nonce:          nonce,
		ServerPublicKey: cm.serverPublicKey,
		CreatedAt:      now,
		ExpiresAt:      now.Add(cm.ttl),
		AgentID:        agentID,
		Reason:         reason,
		Fields:         fields,
		Used:           false,
	}

	cm.challenges[challengeID] = challenge
	return challenge, nil
}

// VerifyChallenge verifies a challenge response
func (cm *ChallengeManager) VerifyChallenge(response *ChallengeResponse) (*Challenge, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	challenge, exists := cm.challenges[response.ChallengeID]
	if !exists {
		return nil, ErrChallengeNotFound
	}

	// Check if already used
	if challenge.Used {
		return nil, ErrChallengeAlreadyUsed
	}

	// Check if expired
	if time.Now().After(challenge.ExpiresAt) {
		delete(cm.challenges, response.ChallengeID)
		return nil, ErrChallengeExpired
	}

	// Verify public key if expected
	if challenge.ExpectedPublicKey != nil {
		if !equalPublicKeys(response.PublicKey, challenge.ExpectedPublicKey) {
			return nil, ErrInvalidPublicKey
		}
	}

	// Verify signature
	// The signature should be over: nonce || timestamp || device_id
	message := cm.buildSignatureMessage(challenge.Nonce, response.Timestamp, response.DeviceID)
	if !ed25519.Verify(response.PublicKey, message, response.Signature) {
		return nil, ErrInvalidSignature
	}

	// Mark as used
	challenge.Used = true

	// Clean up
	delete(cm.challenges, response.ChallengeID)

	return challenge, nil
}

// GetChallenge retrieves a challenge by ID
func (cm *ChallengeManager) GetChallenge(challengeID string) (*Challenge, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	challenge, exists := cm.challenges[challengeID]
	if !exists {
		return nil, ErrChallengeNotFound
	}

	// Return copy without sensitive data
	challengeCopy := *challenge
	return &challengeCopy, nil
}

// RegisterDevice registers a device public key
func (cm *ChallengeManager) RegisterDevice(deviceID string, publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.registeredKeys[deviceID] = publicKey
	return nil
}

// UnregisterDevice removes a device public key
func (cm *ChallengeManager) UnregisterDevice(deviceID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.registeredKeys, deviceID)
}

// GetDevicePublicKey returns the registered public key for a device
func (cm *ChallengeManager) GetDevicePublicKey(deviceID string) (ed25519.PublicKey, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key, exists := cm.registeredKeys[deviceID]
	return key, exists
}

// CleanupExpired removes expired challenges
func (cm *ChallengeManager) CleanupExpired() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	count := 0

	for id, challenge := range cm.challenges {
		if now.After(challenge.ExpiresAt) {
			delete(cm.challenges, id)
			count++
		}
	}

	return count
}

// GetServerPublicKey returns the server's public key
func (cm *ChallengeManager) GetServerPublicKey() ed25519.PublicKey {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.serverPublicKey
}

// SignWithServerKey signs a message with the server's private key
func (cm *ChallengeManager) SignWithServerKey(message []byte) []byte {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return ed25519.Sign(cm.serverPrivateKey, message)
}

// buildSignatureMessage builds the message to be signed
func (cm *ChallengeManager) buildSignatureMessage(nonce []byte, timestamp int64, deviceID string) []byte {
	// Build deterministic message: sha256(nonce || timestamp || deviceID)
	h := sha256.New()
	h.Write(nonce)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write([]byte(deviceID))
	return h.Sum(nil)
}

// equalPublicKeys compares two Ed25519 public keys
func equalPublicKeys(a, b ed25519.PublicKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ToMatrixEvent converts a challenge to a Matrix event for mobile notification
func (c *Challenge) ToMatrixEvent() map[string]interface{} {
	return map[string]interface{}{
		"type":             "com.armorclaw.keystore.unseal_challenge",
		"challenge_id":     c.ID,
		"nonce":            base64.RawURLEncoding.EncodeToString(c.Nonce),
		"server_public_key": base64.RawURLEncoding.EncodeToString(c.ServerPublicKey),
		"agent_id":         c.AgentID,
		"reason":           c.Reason,
		"fields":           c.Fields,
		"expires_at":       c.ExpiresAt.UnixMilli(),
	}
}

// MarshalJSON implements json.Marshaler for Challenge
func (c *Challenge) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID               string   `json:"id"`
		Nonce            string   `json:"nonce"`
		ServerPublicKey  string   `json:"server_public_key"`
		CreatedAt        int64    `json:"created_at"`
		ExpiresAt        int64    `json:"expires_at"`
		AgentID          string   `json:"agent_id"`
		Reason           string   `json:"reason"`
		Fields           []string `json:"fields,omitempty"`
	}

	return json.Marshal(&Alias{
		ID:               c.ID,
		Nonce:            base64.RawURLEncoding.EncodeToString(c.Nonce),
		ServerPublicKey:  base64.RawURLEncoding.EncodeToString(c.ServerPublicKey),
		CreatedAt:        c.CreatedAt.Unix(),
		ExpiresAt:        c.ExpiresAt.Unix(),
		AgentID:          c.AgentID,
		Reason:           c.Reason,
		Fields:           c.Fields,
	})
}

// UnmarshalJSON implements json.Unmarshaler for ChallengeResponse
func (r *ChallengeResponse) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ChallengeID string `json:"challenge_id"`
		Signature   string `json:"signature"`
		PublicKey   string `json:"public_key"`
		WrappedKEK  string `json:"wrapped_kek,omitempty"`
		Timestamp   int64  `json:"timestamp"`
		DeviceID    string `json:"device_id,omitempty"`
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	r.ChallengeID = alias.ChallengeID
	r.Timestamp = alias.Timestamp
	r.DeviceID = alias.DeviceID

	// Decode signature
	sig, err := base64.RawURLEncoding.DecodeString(alias.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}
	r.Signature = sig

	// Decode public key
	pubKey, err := base64.RawURLEncoding.DecodeString(alias.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key encoding: %w", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	r.PublicKey = ed25519.PublicKey(pubKey)

	// Decode wrapped KEK if present
	if alias.WrappedKEK != "" {
		wrappedKEK, err := base64.RawURLEncoding.DecodeString(alias.WrappedKEK)
		if err != nil {
			return fmt.Errorf("invalid wrapped KEK encoding: %w", err)
		}
		r.WrappedKEK = wrappedKEK
	}

	return nil
}

// StartCleanupRoutine starts a background goroutine to clean up expired challenges
func (cm *ChallengeManager) StartCleanupRoutine(done <-chan struct{}, interval time.Duration) {
	if interval == 0 {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				cm.CleanupExpired()
			}
		}
	}()
}

// ListPendingChallenges returns all pending challenges (for admin/debug)
func (cm *ChallengeManager) ListPendingChallenges(agentID string) []*Challenge {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var challenges []*Challenge
	for _, c := range cm.challenges {
		if agentID == "" || c.AgentID == agentID {
			// Return copy
			challengeCopy := *c
			challenges = append(challenges, &challengeCopy)
		}
	}
	return challenges
}

// GetStats returns challenge manager statistics
func (cm *ChallengeManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"pending_challenges": len(cm.challenges),
		"registered_devices": len(cm.registeredKeys),
		"ttl_seconds":        cm.ttl.Seconds(),
	}
}
