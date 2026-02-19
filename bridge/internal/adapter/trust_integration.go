// Package adapter provides trust integration for Matrix adapter
package adapter

import (
	"context"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/trust"
)

// TrustVerifier integrates ZeroTrustManager with Matrix adapter
type TrustVerifier struct {
	trustManager *trust.ZeroTrustManager
	auditLog     *audit.TamperEvidentLog
	logger       *logger.Logger
}

// TrustVerifierConfig configures the trust verifier
type TrustVerifierConfig struct {
	TrustManager *trust.ZeroTrustManager
	AuditLog     *audit.TamperEvidentLog
	Logger       *logger.Logger
}

// NewTrustVerifier creates a new trust verifier
func NewTrustVerifier(cfg TrustVerifierConfig) *TrustVerifier {
	if cfg.Logger == nil {
		cfg.Logger = logger.Global().WithComponent("trust_verifier")
	}
	return &TrustVerifier{
		trustManager: cfg.TrustManager,
		auditLog:     cfg.AuditLog,
		logger:       cfg.Logger,
	}
}

// VerifyEvent verifies a Matrix event using zero-trust principles
// Returns true if the event should be processed, false if it should be rejected
func (tv *TrustVerifier) VerifyEvent(ctx context.Context, event *MatrixEvent, deviceFingerprint trust.DeviceFingerprintInput, ipAddress string) (*trust.ZeroTrustResult, error) {
	if tv.trustManager == nil {
		// No trust manager configured, allow all (backward compatible)
		return &trust.ZeroTrustResult{Passed: true, TrustLevel: trust.TrustScoreHigh}, nil
	}

	// Create verification request
	req := &trust.ZeroTrustRequest{
		SessionID:         event.EventID, // Use event ID as session ID
		UserID:            event.Sender,
		DeviceFingerprint: deviceFingerprint,
		IPAddress:         ipAddress,
		Action:            "matrix_message",
		Resource:          event.RoomID,
		Context: map[string]interface{}{
			"event_type": event.Type,
			"event_id":   event.EventID,
		},
	}

	// Perform verification
	result, err := tv.trustManager.Verify(ctx, req)
	if err != nil {
		tv.logger.Error("trust verification failed", "error", err, "event_id", event.EventID)
		return nil, err
	}

	// Log to audit if configured
	if tv.auditLog != nil {
		actor := audit.Actor{
			Type:      "matrix_user",
			ID:        event.Sender,
			IPAddress: ipAddress,
		}
		resource := audit.Resource{
			Type: "matrix_room",
			ID:   event.RoomID,
		}
		compliance := audit.ComplianceFlags{
			Category:      "trust_verification",
			Severity:      "medium",
			AuditRequired: result.TrustLevel < trust.TrustScoreMedium,
		}

		_, _ = tv.auditLog.LogEntry("trust_verification", actor, "verify", resource, map[string]interface{}{
			"trust_level":   result.TrustLevel.String(),
			"risk_score":    result.RiskScore,
			"passed":        result.Passed,
			"anomaly_flags": result.AnomalyFlags,
		}, compliance)
	}

	// Log result
	tv.logger.Debug("trust verification complete",
		"event_id", event.EventID,
		"sender", event.Sender,
		"trust_level", result.TrustLevel.String(),
		"risk_score", result.RiskScore,
		"passed", result.Passed,
		"anomaly_flags", result.AnomalyFlags,
	)

	return result, nil
}

// VerifyDevice verifies a device for a user
func (tv *TrustVerifier) VerifyDevice(deviceID, method string) error {
	if tv.trustManager == nil {
		return nil
	}

	err := tv.trustManager.VerifyDevice(deviceID, method)
	if err != nil {
		return err
	}

	// Log to audit
	if tv.auditLog != nil {
		actor := audit.Actor{
			Type: "system",
			ID:   "trust_verifier",
		}
		resource := audit.Resource{
			Type: "device",
			ID:   deviceID,
		}
		compliance := audit.ComplianceFlags{
			Category:      "device_verification",
			Severity:      "high",
			AuditRequired: true,
		}

		_, _ = tv.auditLog.LogEntry("device_verified", actor, "verify", resource, map[string]interface{}{
			"method": method,
		}, compliance)
	}

	tv.logger.Info("device verified", "device_id", deviceID, "method", method)
	return nil
}

// RevokeDevice revokes trust for a device
func (tv *TrustVerifier) RevokeDevice(deviceID string) error {
	if tv.trustManager == nil {
		return nil
	}

	err := tv.trustManager.RevokeTrustDevice(deviceID)
	if err != nil {
		return err
	}

	// Log to audit
	if tv.auditLog != nil {
		actor := audit.Actor{
			Type: "system",
			ID:   "trust_verifier",
		}
		resource := audit.Resource{
			Type: "device",
			ID:   deviceID,
		}
		compliance := audit.ComplianceFlags{
			Category:      "device_revocation",
			Severity:      "high",
			AuditRequired: true,
		}

		_, _ = tv.auditLog.LogEntry("device_revoked", actor, "revoke", resource, nil, compliance)
	}

	tv.logger.Warn("device revoked", "device_id", deviceID)
	return nil
}

// GetTrustLevel returns the trust level for a user's session
func (tv *TrustVerifier) GetTrustLevel(sessionID string) (trust.TrustScore, error) {
	if tv.trustManager == nil {
		return trust.TrustScoreHigh, nil
	}

	session, err := tv.trustManager.GetTrustedSession(sessionID)
	if err != nil {
		return trust.TrustScoreUntrusted, err
	}

	return session.TrustLevel, nil
}

// GetUserDevices returns all trusted devices for a user
func (tv *TrustVerifier) GetUserDevices(userID string) []*trust.DeviceFingerprintData {
	if tv.trustManager == nil {
		return nil
	}

	return tv.trustManager.GetUserTrustedDevices(userID)
}

// GetStats returns trust statistics
func (tv *TrustVerifier) GetStats() map[string]interface{} {
	if tv.trustManager == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := tv.trustManager.GetZeroTrustStats()
	stats["enabled"] = true
	return stats
}

// SetTrustManager allows updating the trust manager after initialization
func (m *MatrixAdapter) SetTrustVerifier(verifier *TrustVerifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trustVerifier = verifier
}

// GetTrustVerifier returns the current trust verifier
func (m *MatrixAdapter) GetTrustVerifier() *TrustVerifier {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trustVerifier
}

// SetAuditLog sets the audit log for the Matrix adapter
func (m *MatrixAdapter) SetAuditLog(auditLog *audit.TamperEvidentLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auditLog = auditLog
}

// LogAuditEvent logs an event to the audit log if configured
func (m *MatrixAdapter) LogAuditEvent(eventType, action string, resource audit.Resource, details map[string]interface{}, compliance audit.ComplianceFlags) error {
	m.mu.RLock()
	auditLog := m.auditLog
	m.mu.RUnlock()

	if auditLog == nil {
		return nil
	}

	actor := audit.Actor{
		Type:      "matrix_user",
		ID:        m.userID,
		IPAddress: "", // IP would be set by the caller if available
	}

	_, err := auditLog.LogEntry(eventType, actor, action, resource, details, compliance)
	return err
}

// DeviceFingerprintFromMatrix creates a device fingerprint from Matrix event context
func DeviceFingerprintFromMatrix(sender, userAgent, platform string) trust.DeviceFingerprintInput {
	return trust.DeviceFingerprintInput{
		UserAgent: userAgent,
		Platform:  platform,
		CustomFields: map[string]string{
			"matrix_sender": sender,
		},
	}
}

// TrustEnforcementResult represents the result of trust enforcement
type TrustEnforcementResult struct {
	Allowed         bool
	TrustLevel      trust.TrustScore
	RiskScore       int
	RequiredActions []string
	AnomalyFlags    []string
	Message         string
}

// ShouldEnforceTrust determines if trust enforcement should block the event
func (tv *TrustVerifier) ShouldEnforceTrust(result *trust.ZeroTrustResult, minTrustLevel trust.TrustScore) TrustEnforcementResult {
	if result == nil {
		return TrustEnforcementResult{
			Allowed:    true,
			TrustLevel: trust.TrustScoreHigh,
			Message:    "No trust verification performed",
		}
	}

	enforcement := TrustEnforcementResult{
		TrustLevel:      result.TrustLevel,
		RiskScore:       result.RiskScore,
		RequiredActions: result.RequiredActions,
		AnomalyFlags:    result.AnomalyFlags,
	}

	if !result.Passed || result.TrustLevel < minTrustLevel {
		enforcement.Allowed = false
		enforcement.Message = result.Message
		if enforcement.Message == "" {
			enforcement.Message = "Trust verification failed: trust level below minimum"
		}
		return enforcement
	}

	enforcement.Allowed = true
	enforcement.Message = "Trust verification passed"
	return enforcement
}

// CleanupExpiredSessions cleans up expired trust sessions
func (tv *TrustVerifier) CleanupExpiredSessions() int {
	if tv.trustManager == nil {
		return 0
	}
	return tv.trustManager.CleanupExpired()
}

// StartCleanupRoutine starts a background routine to clean up expired sessions
func (tv *TrustVerifier) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	if tv.trustManager == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleaned := tv.CleanupExpiredSessions()
				if cleaned > 0 {
					tv.logger.Info("cleaned up expired trust sessions", "count", cleaned)
				}
			}
		}
	}()
}
