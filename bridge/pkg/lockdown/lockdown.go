// Package lockdown provides first-boot security lockdown functionality.
// ArmorClaw starts in a maximally secure state and transitions to operational
// mode only after admin-guided security configuration.
package lockdown

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Mode represents the current lockdown state
type Mode string

const (
	// ModeLockdown is the initial state - no network, single device only
	ModeLockdown Mode = "lockdown"
	// ModeBonding is when admin is being established
	ModeBonding Mode = "bonding"
	// ModeConfiguring is when security settings are being configured
	ModeConfiguring Mode = "configuring"
	// ModeHardening is when final security hardening is applied
	ModeHardening Mode = "hardening"
	// ModeOperational is the final state - configured and running
	ModeOperational Mode = "operational"
)

// State represents the complete lockdown state
type State struct {
	mu sync.RWMutex

	// Current mode
	Mode Mode `json:"mode"`

	// Admin information
	AdminEstablished bool      `json:"admin_established"`
	AdminID          string    `json:"admin_id,omitempty"`
	AdminDeviceID    string    `json:"admin_device_id,omitempty"`
	AdminClaimedAt   time.Time `json:"admin_claimed_at,omitempty"`

	// Device information
	SingleDeviceMode bool     `json:"single_device_mode"`
	AuthorizedDevices []string `json:"authorized_devices,omitempty"`

	// Communication restrictions
	AllowedCommunication []string `json:"allowed_communication"`

	// Setup progress
	SetupComplete         bool   `json:"setup_complete"`
	SecurityConfigured    bool   `json:"security_configured"`
	KeystoreInitialized   bool   `json:"keystore_initialized"`
	SecretsInjected       bool   `json:"secrets_injected"`
	HardeningComplete     bool   `json:"hardening_complete"`

	// Timestamps
	StartedAt     time.Time `json:"started_at"`
	ConfiguredAt  time.Time `json:"configured_at,omitempty"`
	OperationalAt time.Time `json:"operational_at,omitempty"`

	// Persistence
	stateFile string
}

// Config for lockdown initialization
type Config struct {
	StateFile string `json:"state_file"`
}

// Manager handles lockdown state transitions
type Manager struct {
	state *State
}

// NewManager creates a new lockdown manager
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		state: &State{
			Mode:                ModeLockdown,
			SingleDeviceMode:    true,
			AllowedCommunication: []string{"unix"},
			StartedAt:           time.Now(),
			stateFile:           cfg.StateFile,
		},
	}

	// Try to load existing state
	if cfg.StateFile != "" {
		if err := m.load(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load lockdown state: %w", err)
		}
	}

	return m, nil
}

// GetState returns a copy of the current state (without mutex)
func (m *Manager) GetState() State {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	// Create a copy without the mutex field
	return State{
		Mode:                 m.state.Mode,
		AdminEstablished:     m.state.AdminEstablished,
		AdminID:              m.state.AdminID,
		AdminDeviceID:        m.state.AdminDeviceID,
		AdminClaimedAt:       m.state.AdminClaimedAt,
		SingleDeviceMode:     m.state.SingleDeviceMode,
		AuthorizedDevices:    append([]string(nil), m.state.AuthorizedDevices...),
		AllowedCommunication: append([]string(nil), m.state.AllowedCommunication...),
		SetupComplete:        m.state.SetupComplete,
		SecurityConfigured:   m.state.SecurityConfigured,
		KeystoreInitialized:  m.state.KeystoreInitialized,
		SecretsInjected:      m.state.SecretsInjected,
		HardeningComplete:    m.state.HardeningComplete,
		StartedAt:            m.state.StartedAt,
		ConfiguredAt:         m.state.ConfiguredAt,
		OperationalAt:        m.state.OperationalAt,
	}
}

// GetMode returns the current lockdown mode
func (m *Manager) GetMode() Mode {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	return m.state.Mode
}

// CanTransition checks if a transition to the target mode is valid
func (m *Manager) CanTransition(target Mode) error {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	current := m.state.Mode

	// Define valid transitions
	validTransitions := map[Mode][]Mode{
		ModeLockdown:    {ModeBonding},
		ModeBonding:     {ModeLockdown, ModeConfiguring},
		ModeConfiguring: {ModeBonding, ModeHardening},
		ModeHardening:   {ModeConfiguring, ModeOperational},
		ModeOperational: {}, // No transitions from operational
	}

	allowed, exists := validTransitions[current]
	if !exists {
		return fmt.Errorf("unknown current mode: %s", current)
	}

	for _, allowedTarget := range allowed {
		if allowedTarget == target {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", current, target)
}

// Transition moves to a new lockdown mode
func (m *Manager) Transition(target Mode) error {
	if err := m.CanTransition(target); err != nil {
		return err
	}

	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	m.state.Mode = target

	switch target {
	case ModeConfiguring:
		if m.state.ConfiguredAt.IsZero() {
			m.state.ConfiguredAt = time.Now()
		}
	case ModeOperational:
		m.state.SetupComplete = true
		m.state.SingleDeviceMode = false
		if m.state.OperationalAt.IsZero() {
			m.state.OperationalAt = time.Now()
		}
	}

	return m.save()
}

// IsAdminEstablished returns true if an admin has claimed ownership
func (m *Manager) IsAdminEstablished() bool {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	return m.state.AdminEstablished
}

// IsSetupComplete returns true if the entire setup flow is complete
func (m *Manager) IsSetupComplete() bool {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	return m.state.SetupComplete
}

// IsSingleDeviceMode returns true if only the admin device can connect
func (m *Manager) IsSingleDeviceMode() bool {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	return m.state.SingleDeviceMode
}

// GetAllowedCommunication returns the allowed communication methods
func (m *Manager) GetAllowedCommunication() []string {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	result := make([]string, len(m.state.AllowedCommunication))
	copy(result, m.state.AllowedCommunication)
	return result
}

// IsCommunicationAllowed checks if a communication method is allowed
func (m *Manager) IsCommunicationAllowed(method string) bool {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()
	for _, allowed := range m.state.AllowedCommunication {
		if allowed == method {
			return true
		}
	}
	return false
}

// SetSecurityConfigured marks security configuration as complete
func (m *Manager) SetSecurityConfigured() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	m.state.SecurityConfigured = true
	return m.save()
}

// SetKeystoreInitialized marks keystore initialization as complete
func (m *Manager) SetKeystoreInitialized() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	m.state.KeystoreInitialized = true
	return m.save()
}

// SetSecretsInjected marks secrets injection as complete
func (m *Manager) SetSecretsInjected() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	m.state.SecretsInjected = true
	return m.save()
}

// SetHardeningComplete marks final hardening as complete
func (m *Manager) SetHardeningComplete() error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	m.state.HardeningComplete = true
	return m.save()
}

// AddAllowedCommunication adds a communication method
func (m *Manager) AddAllowedCommunication(method string) error {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	// Check if already present
	for _, existing := range m.state.AllowedCommunication {
		if existing == method {
			return nil
		}
	}

	m.state.AllowedCommunication = append(m.state.AllowedCommunication, method)
	return m.save()
}

// load reads state from disk
func (m *Manager) load() error {
	if m.state.stateFile == "" {
		return nil
	}

	data, err := os.ReadFile(m.state.stateFile)
	if err != nil {
		return err
	}

	m.state.mu.Lock()
	defer m.state.mu.Unlock()

	return json.Unmarshal(data, m.state)
}

// save writes state to disk
// NOTE: This function does NOT acquire locks internally. Callers must hold
// appropriate locks (RLock for read-only operations, Lock for modifications).
// This prevents deadlock when save() is called from already-locked methods.
func (m *Manager) save() error {
	if m.state.stateFile == "" {
		return nil
	}

	// Caller must hold lock - marshal without acquiring another lock
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(m.state.stateFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	// Write atomically
	tmpFile := m.state.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0640); err != nil {
		return err
	}

	return os.Rename(tmpFile, m.state.stateFile)
}

// Status returns a summary of the lockdown state
func (m *Manager) Status() map[string]interface{} {
	state := m.GetState()
	return map[string]interface{}{
		"mode":                 state.Mode,
		"admin_established":    state.AdminEstablished,
		"single_device_mode":   state.SingleDeviceMode,
		"allowed_communication": state.AllowedCommunication,
		"setup_complete":       state.SetupComplete,
		"security_configured":  state.SecurityConfigured,
		"keystore_initialized": state.KeystoreInitialized,
		"secrets_injected":     state.SecretsInjected,
		"hardening_complete":   state.HardeningComplete,
		"started_at":           state.StartedAt,
		"configured_at":        state.ConfiguredAt,
		"operational_at":       state.OperationalAt,
	}
}

// CheckReadyForTransition verifies prerequisites for transitioning modes
func (m *Manager) CheckReadyForTransition(target Mode) error {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	switch target {
	case ModeBonding:
		// Can always enter bonding from lockdown
		if m.state.Mode != ModeLockdown {
			return fmt.Errorf("can only enter bonding from lockdown mode")
		}

	case ModeConfiguring:
		if !m.state.AdminEstablished {
			return fmt.Errorf("admin must be established before configuration")
		}

	case ModeHardening:
		if !m.state.SecurityConfigured {
			return fmt.Errorf("security must be configured before hardening")
		}

	case ModeOperational:
		if !m.state.KeystoreInitialized {
			return fmt.Errorf("keystore must be initialized")
		}
		if !m.state.HardeningComplete {
			return fmt.Errorf("hardening must be complete")
		}
	}

	return nil
}

// ValidateForOperational performs final validation before going operational
func (m *Manager) ValidateForOperational() error {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	var validationErrors []string

	if !m.state.AdminEstablished {
		validationErrors = append(validationErrors, "admin not established")
	}
	if !m.state.SecurityConfigured {
		validationErrors = append(validationErrors, "security not configured")
	}
	if !m.state.KeystoreInitialized {
		validationErrors = append(validationErrors, "keystore not initialized")
	}
	if !m.state.HardeningComplete {
		validationErrors = append(validationErrors, "hardening not complete")
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("not ready for operational: %v", validationErrors)
	}

	return nil
}

// Context returns a context with lockdown state values
func (m *Manager) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, lockdownKey{}, m)
}

type lockdownKey struct{}

// FromContext retrieves the lockdown manager from context
func FromContext(ctx context.Context) (*Manager, bool) {
	m, ok := ctx.Value(lockdownKey{}).(*Manager)
	return m, ok
}
