package lockdown

import (
	"time"

	"github.com/armorclaw/bridge/pkg/qr"
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
