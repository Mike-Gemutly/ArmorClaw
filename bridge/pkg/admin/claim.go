// Package admin provides admin claiming functionality through Matrix.
// This allows desktop users to claim admin rights via Element X.
package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// ClaimStatus represents the state of an admin claim
type ClaimStatus string

const (
	ClaimPending    ClaimStatus = "pending"
	ClaimApproved   ClaimStatus = "approved"
	ClaimRejected   ClaimStatus = "rejected"
	ClaimExpired    ClaimStatus = "expired"
	ClaimRevoked    ClaimStatus = "revoked"
)

// ClaimRequest represents a pending admin claim
type ClaimRequest struct {
	mu sync.RWMutex

	ID           string      `json:"id"`
	Token        string      `json:"token"`
	MatrixUserID string      `json:"matrix_user_id"`
	DeviceID     string      `json:"device_id"`
	Status       ClaimStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	ExpiresAt    time.Time   `json:"expires_at"`

	// Challenge/response for verification
	Challenge     string `json:"challenge"`
	ChallengeType string `json:"challenge_type"`
	ChallengeSent bool   `json:"challenge_sent"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`

	// Device info
	UserAgent  string `json:"user_agent"`
	IPAddress  string `json:"ip_address"`
	DeviceName string `json:"device_name"`

	// Approval tracking
	ApprovedBy  string     `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedBy  string     `json:"rejected_by,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	RejectReason string    `json:"reject_reason,omitempty"`

	// Signature for verification
	Signature string `json:"signature"`
}

// ClaimManager handles admin claiming via Matrix
type ClaimManager struct {
	mu       sync.RWMutex
	requests map[string]*ClaimRequest
	signingKey []byte
	config    ClaimConfig

	// Callbacks
	onClaimApproved func(*ClaimRequest)
	onClaimRejected func(*ClaimRequest)
}

// ClaimConfig configures the claim process
type ClaimConfig struct {
	// TokenExpiration is how long a claim token is valid
	TokenExpiration time.Duration
	// ChallengeTimeout is how long to wait for challenge response
	ChallengeTimeout time.Duration
	// MaxPendingClaims is the maximum number of pending claims
	MaxPendingClaims int
	// RequireTOTP requires TOTP for additional security
	RequireTOTP bool
}

// DefaultClaimConfig returns sensible defaults
func DefaultClaimConfig() ClaimConfig {
	return ClaimConfig{
		TokenExpiration:  10 * time.Minute,
		ChallengeTimeout: 5 * time.Minute,
		MaxPendingClaims: 3,
		RequireTOTP:      false,
	}
}

// NewClaimManager creates a new claim manager
func NewClaimManager(signingKey []byte, config ClaimConfig) *ClaimManager {
	if len(signingKey) == 0 {
		signingKey = securerandom.MustBytes(32)
	}

	return &ClaimManager{
		requests:   make(map[string]*ClaimRequest),
		signingKey: signingKey,
		config:     config,
	}
}

// InitiateClaim starts a new admin claim process
func (m *ClaimManager) InitiateClaim(matrixUserID, deviceID, userAgent, ipAddress, deviceName string) (*ClaimRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check pending claim limit
	pendingCount := 0
	for _, req := range m.requests {
		if req.Status == ClaimPending {
			pendingCount++
		}
	}
	if pendingCount >= m.config.MaxPendingClaims {
		return nil, errors.New("maximum pending claims reached")
	}

	// Generate claim ID and token
	claimID := "claim_" + securerandom.MustID(16)
	token := securerandom.MustToken(12)
	challenge := securerandom.MustToken(8)

	now := time.Now()
	expiresAt := now.Add(m.config.TokenExpiration)

	claim := &ClaimRequest{
		ID:            claimID,
		Token:         token,
		MatrixUserID:  matrixUserID,
		DeviceID:      deviceID,
		Status:        ClaimPending,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
		Challenge:     challenge,
		ChallengeType: "display_code",
		UserAgent:     userAgent,
		IPAddress:     ipAddress,
		DeviceName:    deviceName,
	}

	// Generate signature
	claim.Signature = m.signClaim(claim)

	m.requests[claimID] = claim

	return claim, nil
}

// ValidateToken validates a claim token and returns the claim request
func (m *ClaimManager) ValidateToken(token string) (*ClaimRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var claim *ClaimRequest
	for _, req := range m.requests {
		if req.Token == token {
			claim = req
			break
		}
	}

	if claim == nil {
		return nil, errors.New("invalid claim token")
	}

	claim.mu.RLock()
	defer claim.mu.RUnlock()

	// Check expiration
	if time.Now().After(claim.ExpiresAt) {
		return nil, errors.New("claim token has expired")
	}

	// Check status
	if claim.Status != ClaimPending {
		return nil, fmt.Errorf("claim is %s", claim.Status)
	}

	// Verify signature
	expectedSig := m.signClaim(claim)
	if !hmac.Equal([]byte(claim.Signature), []byte(expectedSig)) {
		return nil, errors.New("invalid claim signature")
	}

	return claim, nil
}

// RespondChallenge responds to a claim challenge
func (m *ClaimManager) RespondChallenge(token, response string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var claim *ClaimRequest
	for _, req := range m.requests {
		if req.Token == token {
			claim = req
			break
		}
	}

	if claim == nil {
		return errors.New("claim not found")
	}

	claim.mu.Lock()
	defer claim.mu.Unlock()

	if claim.Status != ClaimPending {
		return fmt.Errorf("claim is %s", claim.Status)
	}

	if time.Now().After(claim.ExpiresAt) {
		claim.Status = ClaimExpired
		return errors.New("claim has expired")
	}

	// Verify challenge response
	if !hmac.Equal([]byte(claim.Challenge), []byte(response)) {
		return errors.New("invalid challenge response")
	}

	now := time.Now()
	claim.RespondedAt = &now

	return nil
}

// ApproveClaim approves a pending claim
func (m *ClaimManager) ApproveClaim(claimID, approvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	claim, exists := m.requests[claimID]
	if !exists {
		return errors.New("claim not found")
	}

	claim.mu.Lock()
	defer claim.mu.Unlock()

	if claim.Status != ClaimPending {
		return fmt.Errorf("claim is %s", claim.Status)
	}

	now := time.Now()
	claim.Status = ClaimApproved
	claim.ApprovedBy = approvedBy
	claim.ApprovedAt = &now

	// Notify callback
	if m.onClaimApproved != nil {
		go m.onClaimApproved(claim)
	}

	return nil
}

// RejectClaim rejects a pending claim
func (m *ClaimManager) RejectClaim(claimID, rejectedBy, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	claim, exists := m.requests[claimID]
	if !exists {
		return errors.New("claim not found")
	}

	claim.mu.Lock()
	defer claim.mu.Unlock()

	if claim.Status != ClaimPending {
		return fmt.Errorf("claim is %s", claim.Status)
	}

	now := time.Now()
	claim.Status = ClaimRejected
	claim.RejectedBy = rejectedBy
	claim.RejectedAt = &now
	claim.RejectReason = reason

	// Notify callback
	if m.onClaimRejected != nil {
		go m.onClaimRejected(claim)
	}

	return nil
}

// RevokeClaim revokes an approved claim
func (m *ClaimManager) RevokeClaim(claimID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	claim, exists := m.requests[claimID]
	if !exists {
		return errors.New("claim not found")
	}

	claim.mu.Lock()
	defer claim.mu.Unlock()

	claim.Status = ClaimRevoked

	return nil
}

// GetClaim retrieves a claim by ID
func (m *ClaimManager) GetClaim(claimID string) (*ClaimRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	claim, exists := m.requests[claimID]
	if !exists {
		return nil, errors.New("claim not found")
	}

	return claim, nil
}

// GetClaimByToken retrieves a claim by token
func (m *ClaimManager) GetClaimByToken(token string) (*ClaimRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, claim := range m.requests {
		claim.mu.RLock()
		if claim.Token == token {
			claim.mu.RUnlock()
			return claim, nil
		}
		claim.mu.RUnlock()
	}

	return nil, errors.New("claim not found")
}

// ListPendingClaims returns all pending claims
func (m *ClaimManager) ListPendingClaims() []*ClaimRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*ClaimRequest
	for _, claim := range m.requests {
		claim.mu.RLock()
		if claim.Status == ClaimPending {
			pending = append(pending, claim)
		}
		claim.mu.RUnlock()
	}

	return pending
}

// CleanupExpired removes expired claims
func (m *ClaimManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0

	for id, claim := range m.requests {
		claim.mu.Lock()
		if claim.Status == ClaimPending && now.After(claim.ExpiresAt) {
			claim.Status = ClaimExpired
			count++
		}
		claim.mu.Unlock()

		// Remove old expired/rejected claims (24+ hours old)
		if claim.Status != ClaimPending && claim.Status != ClaimApproved {
			if now.Sub(claim.CreatedAt) > 24*time.Hour {
				delete(m.requests, id)
			}
		}
	}

	return count
}

// SetCallbacks sets the claim callbacks
func (m *ClaimManager) SetCallbacks(onApproved func(*ClaimRequest), onRejected func(*ClaimRequest)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onClaimApproved = onApproved
	m.onClaimRejected = onRejected
}

// signClaim generates a signature for a claim
func (m *ClaimManager) signClaim(claim *ClaimRequest) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s",
		claim.ID,
		claim.Token,
		claim.MatrixUserID,
		claim.DeviceID,
		claim.CreatedAt.Format(time.RFC3339),
	)

	h := hmac.New(sha256.New, m.signingKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// ToJSON returns the claim as JSON
func (c *ClaimRequest) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.MarshalIndent(c, "", "  ")
}

// Summary returns a summary of the claim
func (c *ClaimRequest) Summary() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"id":             c.ID,
		"token":          c.Token,
		"matrix_user_id": c.MatrixUserID,
		"device_id":      c.DeviceID,
		"status":         string(c.Status),
		"created_at":     c.CreatedAt,
		"expires_at":     c.ExpiresAt,
		"device_name":    c.DeviceName,
		"challenge_sent": c.ChallengeSent,
	}
}

// IsExpired checks if the claim is expired
func (c *ClaimRequest) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().After(c.ExpiresAt)
}

// ClaimsStats returns statistics about claims
func (m *ClaimManager) ClaimsStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":    len(m.requests),
		"pending":  0,
		"approved": 0,
		"rejected": 0,
		"expired":  0,
		"revoked":  0,
	}

	for _, claim := range m.requests {
		claim.mu.RLock()
		switch claim.Status {
		case ClaimPending:
			stats["pending"]++
		case ClaimApproved:
			stats["approved"]++
		case ClaimRejected:
			stats["rejected"]++
		case ClaimExpired:
			stats["expired"]++
		case ClaimRevoked:
			stats["revoked"]++
		}
		claim.mu.RUnlock()
	}

	return map[string]interface{}{
		"status": stats,
	}
}

// StartCleanupRoutine starts a background cleanup goroutine
func (m *ClaimManager) StartCleanupRoutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.CleanupExpired()
			}
		}
	}()
}
