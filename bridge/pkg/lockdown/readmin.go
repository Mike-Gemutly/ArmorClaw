// Package lockdown provides admin recovery and reset operations
package lockdown

import (
	"fmt"
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

// InitiateReadmin starts the readmin process
func (rm *ReadminManager) InitiateReadmin(reason string) error {
	rm.state = &ReadminState{
		Reason:    reason,
		Timestamp: time.Now(),
	}
	return nil
}

// GenerateQR generates a QR code with embedded admin credentials
// TODO: Implement full QR generation with secure password
func (rm *ReadminManager) GenerateQR() (string, string, error) {
	if rm.state == nil {
		return "", "", fmt.Errorf("readmin not initiated")
	}
	// Placeholder - will be implemented in later tasks
	return "", "", fmt.Errorf("not implemented")
}

// WipeData wipes all user data (keystore, sessions, devices, hardening state)
func (rm *ReadminManager) WipeData() error {
	if rm.keystore == nil {
		return fmt.Errorf("keystore not configured")
	}
	return rm.keystore.WipeAllData()
}

// Complete finishes the readmin process and returns to normal operation
func (rm *ReadminManager) Complete() error {
	rm.state = nil
	return nil
}
