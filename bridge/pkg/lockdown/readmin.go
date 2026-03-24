package lockdown

import (
	"time"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/qr"
	"github.com/armorclaw/bridge/pkg/securerandom"
)

// KeystoreInterface defines methods required for data wipe
type KeystoreInterface interface {
	WipeAllData() error
}

// MatrixPasswordChanger defines methods for password changes
type MatrixPasswordChanger interface {
	ChangePassword(newPassword string, logoutDevices bool) error
}

// ReadminState tracks the state of a readmin session
type ReadminState struct {
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
	NewPassword string    `json:"-"` // Never persist
	QRPath      string    `json:"qr_path"`
}

// ReadminManager handles admin recovery and reset operations
type ReadminManager struct {
	lockdown  *Manager
	qrManager *qr.QRManager
	keystore  KeystoreInterface
	matrix    MatrixPasswordChanger
	state     *ReadminState
}

// NewReadminManager creates a new readmin manager
func NewReadminManager(lockdown *Manager, qrManager *qr.QRManager, ks KeystoreInterface, matrix MatrixPasswordChanger) *ReadminManager {
	return &ReadminManager{
		lockdown:  lockdown,
		qrManager: qrManager,
		keystore:  ks,
		matrix:    matrix,
		state:     nil,
	}
}

// GenerateQR generates a QR code with embedded admin credentials
func (rm *ReadminManager) GenerateQR() (string, string, error) {
	// Generate secure random admin password (24 chars, alphanumeric + symbols)
	const passwordLength = 24
	const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"

	var passwordBuilder strings.Builder
	for i := 0; i < passwordLength; i++ {
		char, err := securerandom.Bytes(passwordChars)
		if err != nil {
			return "", "", fmt.Errorf("failed to generate password: %w", err)
		}
		passwordBuilder.WriteByte(char)
	}
	newPassword := passwordBuilder.String()

	// Embed admin credentials in token payload
	// Note: We're reusing TokenTypeConfig to embed credentials
	token, err := rm.qrManager.CreateToken("admin", newPassword, rm.qrManager.Sign(token))
	if err != nil {
		return "", "", fmt.Errorf("failed to create admin token: %w", err)
	}

	// Generate QR code
	result, err := rm.qrManager.GenerateConfigQR(10*time.Minute, rm.state.QRPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate config QR: %w", err)
	}

	// Save QR image to file
	if err := os.WriteFile(result.QRImage, 0640); err != nil {
		return "", "", fmt.Errorf("failed to write QR image: %w", err)
	}

	return result.ToTerminal(), result.QRPath, nil
}

// WipeData wipes all user data (keystore, sessions, devices, hardening state)
func (rm *ReadminManager) WipeData() error {
	return nil // Placeholder - will be implemented in later tasks
}

// Complete finishes the readmin process and returns to normal operation
func (rm *ReadminManager) Complete() error {
	return nil // Placeholder - will be implemented in later tasks
}

// MatrixPasswordChanger defines methods for password changes
type MatrixPasswordChanger interface {
	ChangePassword(newPassword string, logoutDevices bool) error
}

// ReadminState tracks the state of a readmin session
type ReadminState struct {
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
	NewPassword string    `json:"-"`
	QRPath      string    `json:"qr_path"`
}

// ReadminManager handles admin recovery and reset operations
type ReadminManager struct {
	lockdown  *Manager
	qrManager *qr.QRManager
	keystore  KeystoreInterface
	matrix    MatrixPasswordChanger
	state     *ReadminState
}

// NewReadminManager creates a new readmin manager
func NewReadminManager(lockdown *Manager, qrManager *qr.QRManager, ks KeystoreInterface, matrix MatrixPasswordChanger) *ReadminManager {
	return &ReadminManager{
		lockdown:  lockdown,
		qrManager: qrManager,
		keystore:  ks,
		matrix:    matrix,
		state:     nil,
	}
}

// InitiateReadmin starts the readmin process
func (rm *ReadminManager) InitiateReadmin(reason string) error {
	return nil
}

// GenerateQR generates a QR code with embedded credentials
func (rm *ReadminManager) GenerateQR() (string, string, error) {
	return "", "", nil
}

// WipeData wipes all user data
func (rm *ReadminManager) WipeData() error {
	return nil
}

// Complete finishes the readmin process and returns to normal operation
func (rm *ReadminManager) Complete() error {
	return nil
}
