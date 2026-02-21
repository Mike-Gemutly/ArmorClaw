package lockdown

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// BondingError represents errors during the bonding process
type BondingError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *BondingError) Error() string {
	return e.Message
}

var (
	ErrAlreadyClaimed      = &BondingError{Code: "already_claimed", Message: "ownership already claimed"}
	ErrInvalidChallenge    = &BondingError{Code: "invalid_challenge", Message: "challenge response invalid"}
	ErrChallengeExpired    = &BondingError{Code: "challenge_expired", Message: "challenge has expired"}
	ErrPassphraseTooShort  = &BondingError{Code: "passphrase_too_short", Message: "passphrase must be at least 8 characters"}
	ErrDeviceNameRequired  = &BondingError{Code: "device_name_required", Message: "device name is required"}
	ErrDisplayNameRequired = &BondingError{Code: "display_name_required", Message: "display name is required"}
)

// AdminDevice represents an authorized admin device
type AdminDevice struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Fingerprint  string    `json:"fingerprint"`
	Certificate  string    `json:"certificate,omitempty"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	Trusted      bool      `json:"trusted"`
	IsAdmin      bool      `json:"is_admin"`
	SessionToken string    `json:"-"` // Never persist
	SessionExpiry time.Time `json:"-"`
}

// Admin represents the system administrator
type Admin struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"display_name"`
	Devices     []AdminDevice `json:"devices"`
	CreatedAt   time.Time     `json:"created_at"`
	Tier        string        `json:"tier"` // owner, admin, moderator
}

// BondingRequest is sent by ArmorChat to claim ownership
type BondingRequest struct {
	DisplayName         string `json:"display_name"`
	DeviceName          string `json:"device_name"`
	DeviceFingerprint   string `json:"device_fingerprint"`
	PassphraseCommitment string `json:"passphrase_commitment"` // argon2id hash commitment
	ChallengeResponse   string `json:"challenge_response,omitempty"`
}

// BondingResponse is returned after successful bonding
type BondingResponse struct {
	Status        string    `json:"status"`
	AdminID       string    `json:"admin_id"`
	DeviceID      string    `json:"device_id"`
	Certificate   string    `json:"certificate"`
	SessionToken  string    `json:"session_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	NextStep      string    `json:"next_step"`
	AvailableMethods []string `json:"available_methods"`
}

// Challenge represents a bonding challenge
type Challenge struct {
	Nonce      string    `json:"nonce"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	DeviceFP   string    `json:"device_fingerprint,omitempty"`
}

// BondingManager handles the admin bonding process
type BondingManager struct {
	lockdown  *Manager
	challenges map[string]*Challenge
}

// NewBondingManager creates a new bonding manager
func NewBondingManager(lockdown *Manager) *BondingManager {
	return &BondingManager{
		lockdown:   lockdown,
		challenges: make(map[string]*Challenge),
	}
}

// GetChallenge generates a new bonding challenge
func (bm *BondingManager) GetChallenge() (*Challenge, error) {
	nonce, err := securerandom.Bytes(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	challenge := &Challenge{
		Nonce:     hex.EncodeToString(nonce),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	bm.challenges[challenge.Nonce] = challenge

	return challenge, nil
}

// ValidateChallenge checks if a challenge is valid
func (bm *BondingManager) ValidateChallenge(nonce string) (*Challenge, error) {
	challenge, exists := bm.challenges[nonce]
	if !exists {
		return nil, ErrInvalidChallenge
	}

	if time.Now().After(challenge.ExpiresAt) {
		delete(bm.challenges, nonce)
		return nil, ErrChallengeExpired
	}

	return challenge, nil
}

// ClaimOwnership processes an ownership claim from ArmorChat
func (bm *BondingManager) ClaimOwnership(req BondingRequest) (*BondingResponse, error) {
	// Check if already claimed
	if bm.lockdown.IsAdminEstablished() {
		return nil, ErrAlreadyClaimed
	}

	// Validate request
	if err := bm.validateBondingRequest(req); err != nil {
		return nil, err
	}

	// Validate challenge if provided
	if req.ChallengeResponse != "" {
		if _, err := bm.ValidateChallenge(req.ChallengeResponse); err != nil {
			return nil, err
		}
		delete(bm.challenges, req.ChallengeResponse)
	}

	// Generate IDs
	adminID := generateID("admin")
	deviceID := generateID("device")

	// Create admin device
	device := AdminDevice{
		ID:          deviceID,
		Name:        req.DeviceName,
		Fingerprint: req.DeviceFingerprint,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Trusted:     true,
		IsAdmin:     true,
	}

	// Generate certificate (simplified - in production use proper PKI)
	cert, err := generateDeviceCertificate(deviceID, req.DeviceFingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}
	device.Certificate = cert

	// Generate session token
	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	device.SessionToken = sessionToken
	device.SessionExpiry = time.Now().Add(24 * time.Hour)

	// Create admin
	admin := Admin{
		ID:          adminID,
		DisplayName: req.DisplayName,
		Devices:     []AdminDevice{device},
		CreatedAt:   time.Now(),
		Tier:        "owner",
	}

	// Update lockdown state
	if err := bm.lockdown.setAdmin(admin, device); err != nil {
		return nil, fmt.Errorf("failed to set admin: %w", err)
	}

	// Transition to bonding mode
	if err := bm.lockdown.Transition(ModeBonding); err != nil {
		return nil, fmt.Errorf("failed to transition: %w", err)
	}

	// Then to configuring mode
	if err := bm.lockdown.Transition(ModeConfiguring); err != nil {
		return nil, fmt.Errorf("failed to transition to configuring: %w", err)
	}

	return &BondingResponse{
		Status:         "claimed",
		AdminID:        adminID,
		DeviceID:       deviceID,
		Certificate:    cert,
		SessionToken:   sessionToken,
		ExpiresAt:      device.SessionExpiry,
		NextStep:       "security_configuration",
		AvailableMethods: []string{"security.get_categories", "security.set_category", "security.get_tiers"},
	}, nil
}

// ValidateSession checks if a session token is valid
func (bm *BondingManager) ValidateSession(token string) (*AdminDevice, error) {
	state := bm.lockdown.GetState()
	if !state.AdminEstablished {
		return nil, errors.New("no admin established")
	}

	if token == "" {
		return nil, errors.New("invalid session token")
	}

	// Validate token format (must be hex-encoded and reasonable length)
	if len(token) < 32 || len(token) > 128 {
		return nil, errors.New("invalid session token format")
	}

	// Check for valid hex characters
	for _, r := range token {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return nil, errors.New("invalid session token format")
		}
	}

	// Return device info based on lockdown state
	// In production, this would look up the session from secure storage
	return &AdminDevice{
		ID:        state.AdminDeviceID,
		IsAdmin:   true,
		Trusted:   true,
		LastSeen:  time.Now(),
	}, nil
}

// AuthorizeDevice adds a new device for an existing admin
func (bm *BondingManager) AuthorizeDevice(adminID string, device AdminDevice) error {
	if !bm.lockdown.IsAdminEstablished() {
		return errors.New("no admin established")
	}

	// Generate certificate
	cert, err := generateDeviceCertificate(device.ID, device.Fingerprint)
	if err != nil {
		return err
	}
	device.Certificate = cert

	// Update state
	state := bm.lockdown.GetState()
	bm.lockdown.state.mu.Lock()
	bm.lockdown.state.AuthorizedDevices = append(state.AuthorizedDevices, device.ID)
	bm.lockdown.state.mu.Unlock()

	return bm.lockdown.save()
}

// validateBondingRequest validates the bonding request fields
func (bm *BondingManager) validateBondingRequest(req BondingRequest) error {
	if req.DisplayName == "" {
		return ErrDisplayNameRequired
	}
	if req.DeviceName == "" {
		return ErrDeviceNameRequired
	}
	if len(req.PassphraseCommitment) < 16 {
		return ErrPassphraseTooShort
	}
	return nil
}

// setAdmin updates the lockdown state with admin information
func (m *Manager) setAdmin(admin Admin, device AdminDevice) error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.AdminEstablished = true
	m.state.AdminID = admin.ID
	m.state.AdminDeviceID = device.ID
	m.state.AdminClaimedAt = time.Now()
	m.state.AuthorizedDevices = []string{device.ID}

	return m.save()
}

// generateID creates a unique identifier with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, securerandom.MustID(16))
}

// generateDeviceCertificate creates a device certificate
func generateDeviceCertificate(deviceID, fingerprint string) (string, error) {
	// Simplified certificate generation
	// In production, use proper X.509 certificates
	data := fmt.Sprintf("%s:%s:%d", deviceID, fingerprint, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// generateSessionToken creates a secure session token
func generateSessionToken() (string, error) {
	return securerandom.ID(32)
}

// GetAdmin returns the current admin information
func (bm *BondingManager) GetAdmin() (*Admin, error) {
	state := bm.lockdown.GetState()
	if !state.AdminEstablished {
		return nil, errors.New("no admin established")
	}

	return &Admin{
		ID:          state.AdminID,
		DisplayName: "Admin", // Would be loaded from storage
		Tier:        "owner",
		CreatedAt:   state.AdminClaimedAt,
	}, nil
}

// ToJSON returns the bonding status as JSON
func (bm *BondingManager) ToJSON() ([]byte, error) {
	state := bm.lockdown.GetState()
	return json.Marshal(map[string]interface{}{
		"mode":              state.Mode,
		"admin_established": state.AdminEstablished,
		"can_claim":         !state.AdminEstablished,
		"available_methods": bm.getAvailableMethods(state.Mode),
	})
}

func (bm *BondingManager) getAvailableMethods(mode Mode) []string {
	switch mode {
	case ModeLockdown:
		return []string{
			"lockdown.status",
			"lockdown.get_challenge",
			"lockdown.claim_ownership",
		}
	case ModeBonding, ModeConfiguring:
		return []string{
			"lockdown.status",
			"security.get_categories",
			"security.set_category",
			"security.get_tiers",
			"security.set_tier",
			"adapters.list",
			"adapters.configure",
			"skills.list",
			"skills.configure",
		}
	case ModeHardening:
		return []string{
			"lockdown.status",
			"secrets.prepare_add",
			"secrets.submit",
		}
	default:
		return []string{"lockdown.status"}
	}
}
