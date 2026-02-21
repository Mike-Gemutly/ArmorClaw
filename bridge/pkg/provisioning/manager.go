// Package provisioning provides secure device provisioning via QR codes
package provisioning

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TokenStatus represents the state of a provisioning token
type TokenStatus string

const (
	StatusPending  TokenStatus = "pending"
	StatusClaimed  TokenStatus = "claimed"
	StatusExpired  TokenStatus = "expired"
	StatusCanceled TokenStatus = "canceled"
)

// Token represents a provisioning token
type Token struct {
	ID           string      `json:"id"`
	Status       TokenStatus `json:"status"`
	Config       *Config     `json:"config"`
	Signature    string      `json:"signature"`
	CreatedAt    time.Time   `json:"created_at"`
	ExpiresAt    time.Time   `json:"expires_at"`
	ClaimedAt    *time.Time  `json:"claimed_at,omitempty"`
	ClaimedBy    *DeviceInfo `json:"claimed_by,omitempty"`
	OneTimeUse   bool        `json:"one_time_use"`
}

// Config represents the configuration to be provisioned
type Config struct {
	Version          int    `json:"version"`
	TokenID          string `json:"token_id,omitempty"`
	MatrixHomeserver string `json:"matrix_homeserver"`
	RPCURL           string `json:"rpc_url"`
	WSURL            string `json:"ws_url"`
	PushGateway      string `json:"push_gateway"`
	ServerName       string `json:"server_name"`
	Region           string `json:"region,omitempty"`
	BridgePublicKey  string `json:"bridge_public_key,omitempty"`
	ExpiresAt        int64  `json:"expires_at"`
}

// DeviceInfo contains information about a device that claimed a token
type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// Manager handles provisioning token generation and verification
type Manager struct {
	mu            sync.RWMutex
	tokens        map[string]*Token
	signingSecret []byte
	config        *ManagerConfig
}

// ManagerConfig contains configuration for the provisioning manager
type ManagerConfig struct {
	// SigningSecret is the HMAC key for signing configurations
	SigningSecret string
	// DefaultExpirySeconds is the default token lifetime
	DefaultExpirySeconds int
	// MaxExpirySeconds is the maximum allowed token lifetime
	MaxExpirySeconds int
	// OneTimeUse determines if tokens can only be used once
	OneTimeUse bool
	// BridgePublicKey is the bridge's public key for TOFU
	BridgePublicKey string
	// ServerConfig provides server URLs for provisioning
	ServerConfig ServerConfigProvider
}

// ServerConfigProvider provides server configuration
type ServerConfigProvider interface {
	GetMatrixHomeserver() string
	GetRPCURL() string
	GetWSURL() string
	GetPushGateway() string
	GetServerName() string
	GetRegion() string
}

// NewManager creates a new provisioning manager
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config.SigningSecret == "" {
		return nil, fmt.Errorf("signing secret is required")
	}

	if config.DefaultExpirySeconds == 0 {
		config.DefaultExpirySeconds = 60
	}
	if config.MaxExpirySeconds == 0 {
		config.MaxExpirySeconds = 300
	}

	return &Manager{
		tokens:        make(map[string]*Token),
		signingSecret: []byte(config.SigningSecret),
		config:        config,
	}, nil
}

// StartToken generates a new provisioning token
func (m *Manager) StartToken(ctx context.Context, opts *StartTokenOptions) (*Token, error) {
	if opts == nil {
		opts = &StartTokenOptions{}
	}

	// Determine expiry
	expirySeconds := m.config.DefaultExpirySeconds
	if opts.ExpirySeconds > 0 {
		if opts.ExpirySeconds > m.config.MaxExpirySeconds {
			return nil, fmt.Errorf("expiry exceeds maximum of %d seconds", m.config.MaxExpirySeconds)
		}
		expirySeconds = opts.ExpirySeconds
	}

	// Generate token ID
	tokenID, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token ID: %w", err)
	}

	// Create config
	now := time.Now()
	expiresAt := now.Add(time.Duration(expirySeconds) * time.Second)

	config := &Config{
		Version:          1,
		TokenID:          tokenID,
		MatrixHomeserver: m.config.ServerConfig.GetMatrixHomeserver(),
		RPCURL:           m.config.ServerConfig.GetRPCURL(),
		WSURL:            m.config.ServerConfig.GetWSURL(),
		PushGateway:      m.config.ServerConfig.GetPushGateway(),
		ServerName:       m.config.ServerConfig.GetServerName(),
		Region:           m.config.ServerConfig.GetRegion(),
		BridgePublicKey:  m.config.BridgePublicKey,
		ExpiresAt:        expiresAt.Unix(),
	}

	// Generate signature
	signature, err := m.signConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign config: %w", err)
	}

	token := &Token{
		ID:         tokenID,
		Status:     StatusPending,
		Config:     config,
		Signature:  signature,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		OneTimeUse: m.config.OneTimeUse,
	}

	m.mu.Lock()
	m.tokens[tokenID] = token
	m.mu.Unlock()

	// Start expiry cleanup
	go m.cleanupToken(tokenID, expiresAt)

	return token, nil
}

// StartTokenOptions contains options for starting a token
type StartTokenOptions struct {
	ExpirySeconds int
	DeviceName    string
}

// GetToken retrieves a token by ID
func (m *Manager) GetToken(tokenID string) (*Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	token, ok := m.tokens[tokenID]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	// Check if expired
	if time.Now().After(token.ExpiresAt) && token.Status == StatusPending {
		token.Status = StatusExpired
	}

	return token, nil
}

// ClaimToken claims a provisioning token
func (m *Manager) ClaimToken(ctx context.Context, tokenID string, device *DeviceInfo) (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, ok := m.tokens[tokenID]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	// Check status
	if token.Status != StatusPending {
		return nil, fmt.Errorf("token is %s", token.Status)
	}

	// Check expiry
	if time.Now().After(token.ExpiresAt) {
		token.Status = StatusExpired
		return nil, fmt.Errorf("token has expired")
	}

	// Claim the token
	now := time.Now()
	token.Status = StatusClaimed
	token.ClaimedAt = &now
	token.ClaimedBy = device

	// Remove one-time-use tokens after claiming
	if token.OneTimeUse {
		go func() {
			time.Sleep(5 * time.Second) // Allow time for client to read response
			m.mu.Lock()
			delete(m.tokens, tokenID)
			m.mu.Unlock()
		}()
	}

	return token, nil
}

// CancelToken cancels a pending token
func (m *Manager) CancelToken(tokenID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, ok := m.tokens[tokenID]
	if !ok {
		return fmt.Errorf("token not found")
	}

	if token.Status != StatusPending {
		return fmt.Errorf("token is %s, cannot cancel", token.Status)
	}

	token.Status = StatusCanceled
	return nil
}

// ListTokens lists all tokens with optional status filter
func (m *Manager) ListTokens(statusFilter ...TokenStatus) []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Token
	for _, token := range m.tokens {
		if len(statusFilter) == 0 {
			result = append(result, token)
			continue
		}
		for _, s := range statusFilter {
			if token.Status == s {
				result = append(result, token)
				break
			}
		}
	}
	return result
}

// RotateSecret rotates the signing secret
func (m *Manager) RotateSecret(newSecret string) error {
	if newSecret == "" {
		return fmt.Errorf("new secret cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.signingSecret = []byte(newSecret)

	// Invalidate all pending tokens (they were signed with old secret)
	for _, token := range m.tokens {
		if token.Status == StatusPending {
			token.Status = StatusCanceled
		}
	}

	return nil
}

// GetQRData returns the QR code data for a token
func (m *Manager) GetQRData(token *Token) (string, error) {
	// Create signed config with signature
	signedConfig := struct {
		*Config
		Signature string `json:"signature"`
	}{
		Config:    token.Config,
		Signature: token.Signature,
	}

	jsonData, err := json.Marshal(signedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Base64 encode
	encoded := hex.EncodeToString(jsonData)

	return fmt.Sprintf("armorclaw://config?d=%s", encoded), nil
}

// signConfig generates an HMAC signature for a config
func (m *Manager) signConfig(config *Config) (string, error) {
	// Get canonical JSON (without signature field)
	canonical, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, m.signingSecret)
	mac.Write(canonical)
	signature := mac.Sum(nil)

	return "hmac-sha256:" + hex.EncodeToString(signature), nil
}

// VerifySignature verifies a config signature
func (m *Manager) VerifySignature(config *Config, signature string) bool {
	if len(signature) < 11 || signature[:11] != "hmac-sha256:" {
		return false
	}

	expectedSig, err := m.signConfig(config)
	if err != nil {
		return false
	}

	// Constant-time comparison
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// cleanupToken removes expired tokens
func (m *Manager) cleanupToken(tokenID string, expiresAt time.Time) {
	time.Sleep(time.Until(expiresAt) + time.Minute)

	m.mu.Lock()
	defer m.mu.Unlock()

	if token, ok := m.tokens[tokenID]; ok {
		if token.Status == StatusPending {
			token.Status = StatusExpired
		}
		// Remove expired tokens after cleanup
		if token.Status == StatusExpired {
			delete(m.tokens, tokenID)
		}
	}
}

// generateTokenID generates a random token ID
func generateTokenID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
