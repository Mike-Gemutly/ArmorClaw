// Package license provides license validation and runtime state management.
//
// Resolves: Gap - License Expiry Runtime Behavior
//
// Manages license states including grace period handling and runtime polling
// to catch expiry while the server is running, not just on boot.
package license

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LicenseState represents the current state of the license
type LicenseState int

const (
	// StateValid means the license is active and valid
	StateValid LicenseState = iota
	// StateGracePeriod means the license has expired but within grace period
	StateGracePeriod
	// StateExpired means the license has expired and grace period has passed
	StateExpired
	// StateInvalid means the license is malformed or revoked
	StateInvalid
	// StateUnknown means the license status could not be determined
	StateUnknown
)

// String returns the string representation of the license state
func (s LicenseState) String() string {
	switch s {
	case StateValid:
		return "valid"
	case StateGracePeriod:
		return "grace_period"
	case StateExpired:
		return "expired"
	case StateInvalid:
		return "invalid"
	case StateUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// RuntimeBehavior defines how the system should behave for each license state
type RuntimeBehavior int

const (
	// BehaviorNormal allows all operations
	BehaviorNormal RuntimeBehavior = iota
	// BehaviorDegraded allows limited operations with warnings
	BehaviorDegraded
	// BehaviorReadOnly allows read operations only
	BehaviorReadOnly
	// BehaviorBlocked blocks all operations
	BehaviorBlocked
)

// String returns the string representation of the runtime behavior
func (b RuntimeBehavior) String() string {
	switch b {
	case BehaviorNormal:
		return "normal"
	case BehaviorDegraded:
		return "degraded"
	case BehaviorReadOnly:
		return "read_only"
	case BehaviorBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// LicenseInfo contains detailed license information
type LicenseInfo struct {
	State            LicenseState
	Behavior         RuntimeBehavior
	LicenseID        string
	CustomerID       string
	Plan             string
	ExpiresAt        time.Time
	GracePeriodEnds  *time.Time // Set when in grace period
	DaysRemaining    int        // Days until expiry (negative if expired)
	HoursInGrace     int        // Hours remaining in grace period
	Features         []string
	MaxInstances     int
	MaxUsers         int
	LastChecked      time.Time
	ValidationErrors []string
}

// LicenseValidator interface for validating licenses
type LicenseValidator interface {
	// Validate checks the license with the license server
	Validate(ctx context.Context) (*LicenseInfo, error)

	// GetCached returns the last known license state without network call
	GetCached() (*LicenseInfo, error)
}

// AlertSender interface for sending license alerts
type AlertSender interface {
	SendLicenseExpiring(ctx context.Context, daysRemaining int, expiresAt string) error
	SendLicenseExpired(ctx context.Context) error
	SendLicenseInvalid(ctx context.Context, reason string) error
}

// StateManager manages runtime license state and behavior
type StateManager struct {
	logger        *slog.Logger
	validator     LicenseValidator
	alertSender   AlertSender
	config        StateConfig

	mu            sync.RWMutex
	currentState  *LicenseInfo
	stopPolling   chan struct{}
	pollingActive bool
}

// StateConfig configures the license state manager
type StateConfig struct {
	// GracePeriodDuration is how long after expiry before hard blocking
	GracePeriodDuration time.Duration
	// PollInterval is how often to check license status
	PollInterval time.Duration
	// AlertThresholds are days before expiry to send alerts
	AlertThresholds []int // e.g., [30, 14, 7, 1]
	// BlockOnExpired if true, blocks all operations when expired
	BlockOnExpired bool
	// ReadOnlyOnGrace if true, allows read-only during grace period
	ReadOnlyOnGrace bool
}

// DefaultStateConfig returns default configuration
func DefaultStateConfig() StateConfig {
	return StateConfig{
		GracePeriodDuration: 7 * 24 * time.Hour, // 7 days
		PollInterval:        24 * time.Hour,     // Check daily
		AlertThresholds:     []int{30, 14, 7, 1},
		BlockOnExpired:      true,
		ReadOnlyOnGrace:     false, // Allow normal operation during grace
	}
}

// NewStateManager creates a new license state manager
func NewStateManager(validator LicenseValidator, alertSender AlertSender, config StateConfig, logger *slog.Logger) *StateManager {
	if logger == nil {
		logger = slog.Default().With("component", "license_state")
	}

	return &StateManager{
		logger:      logger,
		validator:   validator,
		alertSender: alertSender,
		config:      config,
		stopPolling: make(chan struct{}),
	}
}

// Initialize performs initial license check on startup
func (m *StateManager) Initialize(ctx context.Context) (*LicenseInfo, error) {
	info, err := m.validator.Validate(ctx)
	if err != nil {
		m.logger.Error("license_validation_failed", "error", err)
		// Create unknown state info
		info = &LicenseInfo{
			State:       StateUnknown,
			Behavior:    BehaviorDegraded, // Allow degraded operation
			LastChecked: time.Now(),
			ValidationErrors: []string{err.Error()},
		}
	}

	// Calculate derived state
	m.calculateState(info)

	m.mu.Lock()
	m.currentState = info
	m.mu.Unlock()

	m.logger.Info("license_initialized",
		"state", info.State,
		"behavior", info.Behavior,
		"days_remaining", info.DaysRemaining,
	)

	// Send appropriate alerts
	m.sendInitialAlert(ctx, info)

	return info, nil
}

// calculateState determines the runtime behavior based on license info
func (m *StateManager) calculateState(info *LicenseInfo) {
	now := time.Now()

	// Calculate days remaining
	if !info.ExpiresAt.IsZero() {
		info.DaysRemaining = int(time.Until(info.ExpiresAt).Hours() / 24)
	}

	// Determine state based on expiry
	switch {
	case info.State == StateInvalid:
		info.Behavior = BehaviorBlocked

	case info.State == StateUnknown:
		info.Behavior = BehaviorDegraded

	case info.DaysRemaining < 0:
		// Expired - check grace period
		graceEnd := info.ExpiresAt.Add(m.config.GracePeriodDuration)
		if now.Before(graceEnd) {
			info.State = StateGracePeriod
			info.GracePeriodEnds = &graceEnd
			info.HoursInGrace = int(time.Until(graceEnd).Hours())

			if m.config.ReadOnlyOnGrace {
				info.Behavior = BehaviorReadOnly
			} else {
				info.Behavior = BehaviorDegraded
			}
		} else {
			info.State = StateExpired
			if m.config.BlockOnExpired {
				info.Behavior = BehaviorBlocked
			} else {
				info.Behavior = BehaviorReadOnly
			}
		}

	case info.DaysRemaining <= 7:
		info.State = StateValid
		info.Behavior = BehaviorDegraded // Warning state

	default:
		info.State = StateValid
		info.Behavior = BehaviorNormal
	}

	info.LastChecked = now
}

// sendInitialAlert sends appropriate alert on initialization
func (m *StateManager) sendInitialAlert(ctx context.Context, info *LicenseInfo) {
	if m.alertSender == nil {
		return
	}

	switch info.State {
	case StateGracePeriod:
		_ = m.alertSender.SendLicenseExpiring(ctx, info.DaysRemaining, info.ExpiresAt.Format("2006-01-02"))

	case StateExpired:
		_ = m.alertSender.SendLicenseExpired(ctx)

	case StateInvalid:
		reason := "License validation failed"
		if len(info.ValidationErrors) > 0 {
			reason = info.ValidationErrors[0]
		}
		_ = m.alertSender.SendLicenseInvalid(ctx, reason)
	}
}

// StartPolling begins periodic license checks
func (m *StateManager) StartPolling() {
	m.mu.Lock()
	if m.pollingActive {
		m.mu.Unlock()
		return
	}
	m.pollingActive = true
	m.mu.Unlock()

	ticker := time.NewTicker(m.config.PollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := m.checkAndNotify(ctx); err != nil {
					m.logger.Error("license_poll_failed", "error", err)
				}
				cancel()
			case <-m.stopPolling:
				m.logger.Info("license_polling_stopped")
				return
			}
		}
	}()

	m.logger.Info("license_polling_started", "interval", m.config.PollInterval)
}

// StopPolling stops periodic license checks
func (m *StateManager) StopPolling() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pollingActive {
		close(m.stopPolling)
		m.pollingActive = false
	}
}

// checkAndNotify performs a license check and sends alerts if needed
func (m *StateManager) checkAndNotify(ctx context.Context) error {
	newInfo, err := m.validator.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	m.calculateState(newInfo)

	m.mu.RLock()
	oldState := m.currentState
	m.mu.RUnlock()

	// Check for state transitions that require alerts
	if oldState != nil && m.alertSender != nil {
		// Transition from valid to grace period
		if oldState.State == StateValid && newInfo.State == StateGracePeriod {
			_ = m.alertSender.SendLicenseExpiring(ctx, newInfo.DaysRemaining, newInfo.ExpiresAt.Format("2006-01-02"))
		}

		// Transition from grace period to expired
		if oldState.State == StateGracePeriod && newInfo.State == StateExpired {
			_ = m.alertSender.SendLicenseExpired(ctx)
		}

		// Check alert thresholds
		for _, threshold := range m.config.AlertThresholds {
			if oldState.DaysRemaining > threshold && newInfo.DaysRemaining <= threshold && newInfo.DaysRemaining > 0 {
				_ = m.alertSender.SendLicenseExpiring(ctx, newInfo.DaysRemaining, newInfo.ExpiresAt.Format("2006-01-02"))
				break
			}
		}
	}

	// Update state
	m.mu.Lock()
	m.currentState = newInfo
	m.mu.Unlock()

	m.logger.Info("license_checked",
		"state", newInfo.State,
		"behavior", newInfo.Behavior,
		"days_remaining", newInfo.DaysRemaining,
	)

	return nil
}

// GetCurrentState returns the current license state
func (m *StateManager) GetCurrentState() *LicenseInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentState
}

// GetBehavior returns the current runtime behavior
func (m *StateManager) GetBehavior() RuntimeBehavior {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentState == nil {
		return BehaviorDegraded
	}
	return m.currentState.Behavior
}

// CanPerformOperation checks if an operation is allowed given the license state
func (m *StateManager) CanPerformOperation(operation Operation) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentState == nil {
		return false, "license state unknown"
	}

	switch m.currentState.Behavior {
	case BehaviorNormal:
		return true, ""

	case BehaviorDegraded:
		// Allow most operations in degraded mode
		// but block sensitive ones
		if operation == OperationAdminAccess || operation == OperationConfigChange {
			return false, "admin operations limited during license warning period"
		}
		return true, ""

	case BehaviorReadOnly:
		// Only allow read operations
		if operation == OperationRead {
			return true, ""
		}
		return false, "license expired - read-only mode"

	case BehaviorBlocked:
		return false, "license expired - service paused"

	default:
		return false, "unknown license state"
	}
}

// Operation represents an operation type for license checking
type Operation int

const (
	OperationRead Operation = iota
	OperationWrite
	OperationMessageSend
	OperationMessageReceive
	OperationContainerCreate
	OperationContainerExec
	OperationAdminAccess
	OperationConfigChange
	OperationRPC
)

// GetStateSummary returns a human-readable summary of the license state
func (m *StateManager) GetStateSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentState == nil {
		return "License status unknown"
	}

	switch m.currentState.State {
	case StateValid:
		if m.currentState.DaysRemaining <= 7 {
			return fmt.Sprintf("License expires in %d days (%s)",
				m.currentState.DaysRemaining,
				m.currentState.ExpiresAt.Format("Jan 2, 2006"))
		}
		return "License valid"

	case StateGracePeriod:
		return fmt.Sprintf("License expired - Grace period ends in %d hours",
			m.currentState.HoursInGrace)

	case StateExpired:
		return "License expired - Service paused"

	case StateInvalid:
		return "License invalid - Please contact support"

	default:
		return "License status unknown"
	}
}

// ShouldShowDashboardError returns true if the web dashboard should show an error page
func (m *StateManager) ShouldShowDashboardError() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentState == nil {
		return true
	}

	return m.currentState.Behavior == BehaviorBlocked
}

// GetDashboardError returns the error message for the web dashboard
func (m *StateManager) GetDashboardError() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentState == nil {
		return "Service Paused: License Validation Failed"
	}

	switch m.currentState.State {
	case StateExpired:
		return "Service Paused: License Required"
	case StateInvalid:
		return "Service Paused: Invalid License"
	case StateGracePeriod:
		return fmt.Sprintf("License Expired: Grace period ends in %d hours", m.currentState.HoursInGrace)
	default:
		return "Service Paused: License Issue"
	}
}
