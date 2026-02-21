// Package trust provides zero-trust verification mechanisms for ArmorClaw
package trust

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// TrustScore represents the trust level of a device or session
type TrustScore int

const (
	TrustScoreUntrusted TrustScore = iota
	TrustScoreLow
	TrustScoreMedium
	TrustScoreHigh
	TrustScoreVerified
)

func (t TrustScore) String() string {
	switch t {
	case TrustScoreUntrusted:
		return "untrusted"
	case TrustScoreLow:
		return "low"
	case TrustScoreMedium:
		return "medium"
	case TrustScoreHigh:
		return "high"
	case TrustScoreVerified:
		return "verified"
	default:
		return "unknown"
	}
}

// ZeroTrustConfig configures the zero-trust manager
type ZeroTrustConfig struct {
	// Minimum trust level required for operations
	MinimumTrustLevel TrustScore

	// Trust decay interval (how often trust levels decrease)
	TrustDecayInterval time.Duration

	// Trust decay amount
	TrustDecayAmount int

	// Maximum verification attempts before lockout
	MaxVerificationAttempts int

	// Lockout duration after max attempts
	LockoutDuration time.Duration

	// Device fingerprint TTL
	DeviceFingerprintTTL time.Duration

	// Enable continuous verification
	ContinuousVerification bool

	// Verification interval for continuous mode
	VerificationInterval time.Duration

	// Logger
	Logger *slog.Logger
}

// DeviceFingerprintData represents a device's unique identifier
type DeviceFingerprintData struct {
	ID                  string            `json:"id"`
	Hash                string            `json:"hash"`
	UserAgent           string            `json:"user_agent"`
	Platform            string            `json:"platform"`
	FirstSeen           time.Time         `json:"first_seen"`
	LastSeen            time.Time         `json:"last_seen"`
	TrustLevel          TrustScore        `json:"trust_level"`
	VerificationCount   int               `json:"verification_count"`
	FailedVerifications int               `json:"failed_verifications"`
	KnownIPs            map[string]time.Time `json:"known_ips"`
	Verified            bool              `json:"verified"`
	VerificationMethod  string            `json:"verification_method,omitempty"`
}

// TrustedSession represents a user session with trust information
type TrustedSession struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	DeviceID             string     `json:"device_id"`
	IPAddress            string     `json:"ip_address"`
	TrustLevel           TrustScore `json:"trust_level"`
	CreatedAt            time.Time  `json:"created_at"`
	LastActivity         time.Time  `json:"last_activity"`
	ExpiresAt            time.Time  `json:"expires_at"`
	VerificationAttempts int        `json:"verification_attempts"`
	LockedOut            bool       `json:"locked_out"`
	LockoutUntil         *time.Time `json:"lockout_until,omitempty"`
	RiskScore            int        `json:"risk_score"`
	AnomalyFlags         []string   `json:"anomaly_flags,omitempty"`
}

// ZeroTrustRequest represents a verification request
type ZeroTrustRequest struct {
	SessionID         string                 `json:"session_id"`
	UserID            string                 `json:"user_id"`
	DeviceFingerprint DeviceFingerprintInput `json:"device_fingerprint"`
	IPAddress         string                 `json:"ip_address"`
	Action            string                 `json:"action"`
	Resource          string                 `json:"resource"`
	Context           map[string]interface{} `json:"context,omitempty"`
}

// DeviceFingerprintInput represents input for device fingerprinting
type DeviceFingerprintInput struct {
	UserAgent    string            `json:"user_agent"`
	Platform     string            `json:"platform"`
	ScreenRes    string            `json:"screen_resolution,omitempty"`
	Timezone     string            `json:"timezone,omitempty"`
	Language     string            `json:"language,omitempty"`
	Plugins      []string          `json:"plugins,omitempty"`
	CanvasHash   string            `json:"canvas_hash,omitempty"`
	WebGLHash    string            `json:"webgl_hash,omitempty"`
	AudioHash    string            `json:"audio_hash,omitempty"`
	Fonts        []string          `json:"fonts,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// ZeroTrustResult represents the result of a verification
type ZeroTrustResult struct {
	Passed          bool       `json:"passed"`
	TrustLevel      TrustScore `json:"trust_level"`
	RiskScore       int        `json:"risk_score"`
	SessionID       string     `json:"session_id"`
	DeviceID        string     `json:"device_id"`
	RequiredActions []string   `json:"required_actions,omitempty"`
	AnomalyFlags    []string   `json:"anomaly_flags,omitempty"`
	Message         string     `json:"message,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

// ZeroTrustVerifier is an interface for trust verification plugins
type ZeroTrustVerifier interface {
	Name() string
	Verify(ctx context.Context, req *ZeroTrustRequest) (TrustScore, int, []string, error)
}

// ZeroTrustManager manages zero-trust verification
type ZeroTrustManager struct {
	config      ZeroTrustConfig
	devices     map[string]*DeviceFingerprintData
	sessions    map[string]*TrustedSession
	userDevices map[string]map[string]bool
	mu          sync.RWMutex
	logger      *slog.Logger
	verifiers   []ZeroTrustVerifier
}

// NewZeroTrustManager creates a new zero-trust manager
func NewZeroTrustManager(config ZeroTrustConfig) *ZeroTrustManager {
	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "zero_trust_manager")
	}

	if config.MinimumTrustLevel == 0 {
		config.MinimumTrustLevel = TrustScoreMedium
	}

	if config.TrustDecayInterval == 0 {
		config.TrustDecayInterval = 24 * time.Hour
	}

	if config.MaxVerificationAttempts == 0 {
		config.MaxVerificationAttempts = 5
	}

	if config.LockoutDuration == 0 {
		config.LockoutDuration = 30 * time.Minute
	}

	if config.DeviceFingerprintTTL == 0 {
		config.DeviceFingerprintTTL = 30 * 24 * time.Hour
	}

	if config.VerificationInterval == 0 {
		config.VerificationInterval = 5 * time.Minute
	}

	return &ZeroTrustManager{
		config:      config,
		devices:     make(map[string]*DeviceFingerprintData),
		sessions:    make(map[string]*TrustedSession),
		userDevices: make(map[string]map[string]bool),
		logger:      config.Logger,
		verifiers:   make([]ZeroTrustVerifier, 0),
	}
}

// AddVerifier adds a trust verifier
func (m *ZeroTrustManager) AddVerifier(verifier ZeroTrustVerifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verifiers = append(m.verifiers, verifier)
}

// Verify performs trust verification
func (m *ZeroTrustManager) Verify(ctx context.Context, req *ZeroTrustRequest) (*ZeroTrustResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, sessionExists := m.sessions[req.SessionID]

	if sessionExists && session.LockedOut {
		if session.LockoutUntil != nil && time.Now().Before(*session.LockoutUntil) {
			return &ZeroTrustResult{
				Passed:    false,
				Message:   "Account is temporarily locked",
				RiskScore: 100,
			}, nil
		}
		session.LockedOut = false
		session.LockoutUntil = nil
		session.VerificationAttempts = 0
	}

	device := m.getOrCreateDevice(req.UserID, &req.DeviceFingerprint)

	if device.KnownIPs == nil {
		device.KnownIPs = make(map[string]time.Time)
	}
	device.KnownIPs[req.IPAddress] = time.Now()

	riskScore := m.calculateRiskScore(session, device, req)
	trustLevel := m.calculateTrustLevel(riskScore, device)

	var anomalyFlags []string
	for _, verifier := range m.verifiers {
		vLevel, vRisk, vFlags, err := verifier.Verify(ctx, req)
		if err != nil {
			m.logger.Warn("verifier_error", "verifier", verifier.Name(), "error", err)
			continue
		}
		if vLevel < trustLevel {
			trustLevel = vLevel
		}
		riskScore = (riskScore + vRisk) / 2
		anomalyFlags = append(anomalyFlags, vFlags...)
	}

	anomalyFlags = append(anomalyFlags, m.detectAnomalies(session, device, req)...)

	if !sessionExists {
		session = &TrustedSession{
			ID:           req.SessionID,
			UserID:       req.UserID,
			DeviceID:     device.ID,
			IPAddress:    req.IPAddress,
			TrustLevel:   trustLevel,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			RiskScore:    riskScore,
			AnomalyFlags: anomalyFlags,
		}
		m.sessions[req.SessionID] = session
	} else {
		session.TrustLevel = trustLevel
		session.LastActivity = time.Now()
		session.RiskScore = riskScore
		session.AnomalyFlags = anomalyFlags
	}

	device.LastSeen = time.Now()
	device.TrustLevel = trustLevel

	passed := trustLevel >= m.config.MinimumTrustLevel

	if !passed {
		session.VerificationAttempts++
		if session.VerificationAttempts >= m.config.MaxVerificationAttempts {
			lockoutUntil := time.Now().Add(m.config.LockoutDuration)
			session.LockedOut = true
			session.LockoutUntil = &lockoutUntil
			device.FailedVerifications++
			m.logger.Warn("session_locked",
				"session_id", req.SessionID,
				"user_id", req.UserID,
				"device_id", device.ID,
				"attempts", session.VerificationAttempts,
			)
		}
	} else {
		session.VerificationAttempts = 0
		device.VerificationCount++
	}

	var requiredActions []string
	if trustLevel < m.config.MinimumTrustLevel {
		if !device.Verified {
			requiredActions = append(requiredActions, "device_verification")
		}
		if len(anomalyFlags) > 0 {
			requiredActions = append(requiredActions, "mfa_challenge")
		}
	}

	result := &ZeroTrustResult{
		Passed:          passed,
		TrustLevel:      trustLevel,
		RiskScore:       riskScore,
		SessionID:       session.ID,
		DeviceID:        device.ID,
		RequiredActions: requiredActions,
		AnomalyFlags:    anomalyFlags,
	}

	if !passed {
		result.Message = "Trust verification failed"
	}

	m.logger.Debug("verification_complete",
		"session_id", req.SessionID,
		"user_id", req.UserID,
		"trust_level", trustLevel.String(),
		"risk_score", riskScore,
		"passed", passed,
	)

	return result, nil
}

func (m *ZeroTrustManager) getOrCreateDevice(userID string, input *DeviceFingerprintInput) *DeviceFingerprintData {
	hash := m.calculateDeviceHash(input)

	for id, device := range m.devices {
		if device.Hash == hash {
			if m.userDevices[userID] != nil && m.userDevices[userID][id] {
				return device
			}
		}
	}

	device := &DeviceFingerprintData{
		ID:         generateZeroTrustDeviceID(),
		Hash:       hash,
		UserAgent:  input.UserAgent,
		Platform:   input.Platform,
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		TrustLevel: TrustScoreLow,
		KnownIPs:   make(map[string]time.Time),
	}

	m.devices[device.ID] = device

	if m.userDevices[userID] == nil {
		m.userDevices[userID] = make(map[string]bool)
	}
	m.userDevices[userID][device.ID] = true

	m.logger.Info("new_device_registered",
		"device_id", device.ID,
		"user_id", userID,
		"platform", input.Platform,
	)

	return device
}

func (m *ZeroTrustManager) calculateDeviceHash(input *DeviceFingerprintInput) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%v|%v",
		input.UserAgent,
		input.Platform,
		input.ScreenRes,
		input.Timezone,
		input.Language,
		input.CanvasHash,
		input.WebGLHash,
		input.Plugins,
		input.Fonts,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (m *ZeroTrustManager) calculateRiskScore(session *TrustedSession, device *DeviceFingerprintData, req *ZeroTrustRequest) int {
	score := 0

	if device.VerificationCount == 0 {
		score += 30
	}

	if !device.Verified {
		score += 20
	}

	if len(device.KnownIPs) > 0 {
		knownIP := false
		for ip := range device.KnownIPs {
			if ip == req.IPAddress {
				knownIP = true
				break
			}
		}
		if !knownIP {
			score += 15
		}
	}

	if device.FailedVerifications > 3 {
		score += 25
	}

	if session != nil {
		sessionAge := time.Since(session.CreatedAt)
		if sessionAge < 5*time.Minute {
			score += 10
		}
		if session.VerificationAttempts > 0 {
			score += session.VerificationAttempts * 5
		}
	}

	if session != nil && session.IPAddress != req.IPAddress {
		score += 20
	}

	if score > 100 {
		score = 100
	}

	return score
}

func (m *ZeroTrustManager) calculateTrustLevel(riskScore int, device *DeviceFingerprintData) TrustScore {
	if device.Verified {
		if riskScore < 20 {
			return TrustScoreVerified
		}
		if riskScore < 40 {
			return TrustScoreHigh
		}
	}

	switch {
	case riskScore < 20:
		return TrustScoreHigh
	case riskScore < 40:
		return TrustScoreMedium
	case riskScore < 60:
		return TrustScoreLow
	default:
		return TrustScoreUntrusted
	}
}

func (m *ZeroTrustManager) detectAnomalies(session *TrustedSession, device *DeviceFingerprintData, req *ZeroTrustRequest) []string {
	var anomalies []string

	if session != nil && session.IPAddress != "" && session.IPAddress != req.IPAddress {
		anomalies = append(anomalies, "ip_change")
	}

	if session != nil {
		timeSinceLastActivity := time.Since(session.LastActivity)
		if timeSinceLastActivity < time.Second && session.IPAddress != req.IPAddress {
			anomalies = append(anomalies, "impossible_travel")
		}
	}

	if device.VerificationCount < 3 {
		if req.Action == "admin_access" || req.Action == "sensitive_data" {
			anomalies = append(anomalies, "new_device_sensitive_access")
		}
	}

	if device.FailedVerifications > 2 {
		anomalies = append(anomalies, "multiple_failed_verifications")
	}

	return anomalies
}

// VerifyDevice marks a device as verified
func (m *ZeroTrustManager) VerifyDevice(deviceID, method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	device.Verified = true
	device.VerificationMethod = method
	device.TrustLevel = TrustScoreVerified

	m.logger.Info("device_verified",
		"device_id", deviceID,
		"method", method,
	)

	return nil
}

// RevokeTrustDevice revokes a device's verification
func (m *ZeroTrustManager) RevokeTrustDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	device.Verified = false
	device.TrustLevel = TrustScoreLow

	m.logger.Warn("device_revoked", "device_id", deviceID)

	return nil
}

// GetTrustedSession gets a session by ID
func (m *ZeroTrustManager) GetTrustedSession(sessionID string) (*TrustedSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetDeviceData gets a device by ID
func (m *ZeroTrustManager) GetDeviceData(deviceID string) (*DeviceFingerprintData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return device, nil
}

// GetUserTrustedDevices gets all devices for a user
func (m *ZeroTrustManager) GetUserTrustedDevices(userID string) []*DeviceFingerprintData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deviceIDs := m.userDevices[userID]
	if deviceIDs == nil {
		return nil
	}

	devices := make([]*DeviceFingerprintData, 0, len(deviceIDs))
	for id := range deviceIDs {
		if device, exists := m.devices[id]; exists {
			devices = append(devices, device)
		}
	}

	return devices
}

// InvalidateTrustedSession invalidates a session
func (m *ZeroTrustManager) InvalidateTrustedSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.ExpiresAt = time.Now()
	delete(m.sessions, sessionID)

	m.logger.Info("session_invalidated", "session_id", sessionID)

	return nil
}

// CleanupExpired cleans up expired sessions and old devices
func (m *ZeroTrustManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for id, session := range m.sessions {
		if session.ExpiresAt.Before(now) {
			delete(m.sessions, id)
			cleaned++
		}
	}

	fingerprintExpiry := now.Add(-m.config.DeviceFingerprintTTL)
	for id, device := range m.devices {
		if device.LastSeen.Before(fingerprintExpiry) && !device.Verified {
			delete(m.devices, id)
			for userID, devices := range m.userDevices {
				delete(devices, id)
				if len(devices) == 0 {
					delete(m.userDevices, userID)
				}
			}
			cleaned++
		}
	}

	if cleaned > 0 {
		m.logger.Info("cleanup_complete", "cleaned", cleaned)
	}

	return cleaned
}

// GetZeroTrustStats returns trust manager statistics
func (m *ZeroTrustManager) GetZeroTrustStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	verifiedDevices := 0
	trustLevelCounts := make(map[TrustScore]int)

	for _, device := range m.devices {
		if device.Verified {
			verifiedDevices++
		}
		trustLevelCounts[device.TrustLevel]++
	}

	activeSessions := 0
	lockedSessions := 0
	now := time.Now()

	for _, session := range m.sessions {
		if session.ExpiresAt.After(now) {
			activeSessions++
			if session.LockedOut {
				lockedSessions++
			}
		}
	}

	return map[string]interface{}{
		"total_devices":        len(m.devices),
		"verified_devices":     verifiedDevices,
		"total_sessions":       len(m.sessions),
		"active_sessions":      activeSessions,
		"locked_sessions":      lockedSessions,
		"trust_level_counts":   trustLevelCounts,
		"minimum_trust_level":  m.config.MinimumTrustLevel.String(),
		"verifiers_registered": len(m.verifiers),
	}
}

// generateZeroTrustDeviceID creates a unique device identifier
func generateZeroTrustDeviceID() string {
	b := securerandom.MustBytes(16)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:8])
}

// CheckPrivateIP checks if an IP address is private
func CheckPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
