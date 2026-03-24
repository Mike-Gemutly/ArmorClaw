package trust

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// HardeningStep represents a security hardening step for first-login flow
type HardeningStep string

const (
	// PasswordRotated indicates the user has changed their initial password
	PasswordRotated HardeningStep = "password_rotated"
	// BootstrapWiped indicates temporary bootstrap files have been cleaned up
	BootstrapWiped HardeningStep = "bootstrap_wiped"
	// DeviceVerified indicates the current device has been verified
	DeviceVerified HardeningStep = "device_verified"
	// RecoveryBackedUp indicates recovery keys have been backed up securely
	RecoveryBackedUp HardeningStep = "recovery_backed_up"
	// BiometricsEnabled indicates biometric authentication has been configured
	BiometricsEnabled HardeningStep = "biometrics_enabled"
)

// UserHardeningState tracks the completion status of security hardening steps
type UserHardeningState struct {
	// UserID is the unique identifier for the user
	UserID string `json:"user_id"`

	// PasswordRotated indicates password change step is complete
	PasswordRotated bool `json:"password_rotated"`

	// BootstrapWiped indicates bootstrap cleanup step is complete
	BootstrapWiped bool `json:"bootstrap_wiped"`

	// DeviceVerified indicates device verification step is complete
	DeviceVerified bool `json:"device_verified"`

	// RecoveryBackedUp indicates recovery backup step is complete
	RecoveryBackedUp bool `json:"recovery_backed_up"`

	// BiometricsEnabled indicates biometric setup step is complete (optional)
	BiometricsEnabled bool `json:"biometrics_enabled"`

	// DelegationReady is computed: true when all mandatory steps are complete
	DelegationReady bool `json:"delegation_ready"`

	// CreatedAt is when the hardening state was first created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the hardening state was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// Recompute updates DelegationReady based on mandatory step completion
// DelegationReady is true only when PasswordRotated, BootstrapWiped,
// DeviceVerified, and RecoveryBackedUp are all true.
// BiometricsEnabled is optional and does not affect DelegationReady.
func (s *UserHardeningState) Recompute() {
	s.DelegationReady = s.PasswordRotated &&
		s.BootstrapWiped &&
		s.DeviceVerified &&
		s.RecoveryBackedUp
	s.UpdatedAt = time.Now()
}

// Store defines the interface for persisting and retrieving hardening state
type Store interface {
	// Get retrieves the hardening state for a user
	Get(userID string) (*UserHardeningState, error)

	// Put saves or updates the hardening state
	Put(state *UserHardeningState) error

	// IsDelegationReady checks if all mandatory hardening steps are complete
	IsDelegationReady(userID string) (bool, error)

	// AckStep marks a specific hardening step as complete
	AckStep(userID string, step HardeningStep) error
}

// KeystoreHardeningStore implements Store interface using SQLCipher keystore
type KeystoreHardeningStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewKeystoreHardeningStore creates a new hardening store backed by SQLCipher keystore
func NewKeystoreHardeningStore(db *sql.DB) *KeystoreHardeningStore {
	return &KeystoreHardeningStore{
		db: db,
	}
}

// Get retrieves the hardening state for a user
// Returns default state with all false values if user not found
func (s *KeystoreHardeningStore) Get(userID string) (*UserHardeningState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := &UserHardeningState{
		UserID: userID,
		// Default all flags to false
		PasswordRotated:   false,
		BootstrapWiped:    false,
		DeviceVerified:    false,
		RecoveryBackedUp:  false,
		BiometricsEnabled: false,
		DelegationReady:   false,
	}

	var passwordRotated, bootstrapWiped, deviceVerified int
	var recoveryBackedUp, biometricsEnabled, delegationReady int
	var createdAt, updatedAt int64

	err := s.db.QueryRow(`
		SELECT password_rotated, bootstrap_wiped, device_verified,
		       recovery_backed_up, biometrics_enabled, delegation_ready,
		       created_at, updated_at
		FROM hardening_state
		WHERE user_id = ?
	`, userID).Scan(
		&passwordRotated,
		&bootstrapWiped,
		&deviceVerified,
		&recoveryBackedUp,
		&biometricsEnabled,
		&delegationReady,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// User not found, return default state
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get hardening state: %w", err)
	}

	// Map INTEGER to bool (0=false, 1=true)
	state.PasswordRotated = passwordRotated == 1
	state.BootstrapWiped = bootstrapWiped == 1
	state.DeviceVerified = deviceVerified == 1
	state.RecoveryBackedUp = recoveryBackedUp == 1
	state.BiometricsEnabled = biometricsEnabled == 1
	state.DelegationReady = delegationReady == 1
	state.CreatedAt = time.Unix(createdAt, 0)
	state.UpdatedAt = time.Unix(updatedAt, 0)

	return state, nil
}

// Put saves or updates the hardening state
func (s *KeystoreHardeningStore) Put(state *UserHardeningState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Map bool to INTEGER (false=0, true=1)
	passwordRotated := 0
	if state.PasswordRotated {
		passwordRotated = 1
	}
	bootstrapWiped := 0
	if state.BootstrapWiped {
		bootstrapWiped = 1
	}
	deviceVerified := 0
	if state.DeviceVerified {
		deviceVerified = 1
	}
	recoveryBackedUp := 0
	if state.RecoveryBackedUp {
		recoveryBackedUp = 1
	}
	biometricsEnabled := 0
	if state.BiometricsEnabled {
		biometricsEnabled = 1
	}
	delegationReady := 0
	if state.DelegationReady {
		delegationReady = 1
	}

	// Set timestamps if not set
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now()
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO hardening_state
		(user_id, password_rotated, bootstrap_wiped, device_verified,
		 recovery_backed_up, biometrics_enabled, delegation_ready,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		state.UserID,
		passwordRotated,
		bootstrapWiped,
		deviceVerified,
		recoveryBackedUp,
		biometricsEnabled,
		delegationReady,
		state.CreatedAt.Unix(),
		state.UpdatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to put hardening state: %w", err)
	}

	return nil
}

// IsDelegationReady checks if all mandatory hardening steps are complete
func (s *KeystoreHardeningStore) IsDelegationReady(userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var delegationReady int

	err := s.db.QueryRow(`
		SELECT delegation_ready
		FROM hardening_state
		WHERE user_id = ?
	`, userID).Scan(&delegationReady)

	if err == sql.ErrNoRows {
		// User not found, not ready
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check delegation ready: %w", err)
	}

	return delegationReady == 1, nil
}

// AckStep marks a specific hardening step as complete
// Updates the step, recomputes delegation_ready, and updates timestamp
func (s *KeystoreHardeningStore) AckStep(userID string, step HardeningStep) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Map step to column name
	var columnName string
	switch step {
	case PasswordRotated:
		columnName = "password_rotated"
	case BootstrapWiped:
		columnName = "bootstrap_wiped"
	case DeviceVerified:
		columnName = "device_verified"
	case RecoveryBackedUp:
		columnName = "recovery_backed_up"
	case BiometricsEnabled:
		columnName = "biometrics_enabled"
	default:
		return fmt.Errorf("unknown hardening step: %s", step)
	}

	result, err := s.db.Exec(`
		UPDATE hardening_state
		SET `+columnName+` = 1, updated_at = ?
		WHERE user_id = ?
	`, time.Now().Unix(), userID)

	if err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		now := time.Now().Unix()
		_, err = s.db.Exec(`
			INSERT INTO hardening_state
			(user_id, password_rotated, bootstrap_wiped, device_verified,
			 recovery_backed_up, biometrics_enabled, delegation_ready,
			 created_at, updated_at)
			VALUES (?, 0, 0, 0, 0, 0, 0, ?, ?)
		`, userID, now, now)
		if err != nil {
			return fmt.Errorf("failed to initialize hardening state for new user: %w", err)
		}

		_, err = s.db.Exec(`
			UPDATE hardening_state
			SET `+columnName+` = 1, updated_at = ?
			WHERE user_id = ?
		`, time.Now().Unix(), userID)
		if err != nil {
			return fmt.Errorf("failed to update step after initialization: %w", err)
		}
	}

	var passwordRotated, bootstrapWiped, deviceVerified, recoveryBackedUp int
	err = s.db.QueryRow(`
		SELECT password_rotated, bootstrap_wiped, device_verified, recovery_backed_up
		FROM hardening_state
		WHERE user_id = ?
	`, userID).Scan(&passwordRotated, &bootstrapWiped, &deviceVerified, &recoveryBackedUp)

	if err != nil {
		return fmt.Errorf("failed to fetch steps for recomputation: %w", err)
	}

	delegationReady := 0
	if passwordRotated == 1 && bootstrapWiped == 1 && deviceVerified == 1 && recoveryBackedUp == 1 {
		delegationReady = 1
	}

	_, err = s.db.Exec(`
		UPDATE hardening_state
		SET delegation_ready = ?, updated_at = ?
		WHERE user_id = ?
	`, delegationReady, time.Now().Unix(), userID)

	if err != nil {
		return fmt.Errorf("failed to update delegation ready: %w", err)
	}

	return nil
}
