// Package trust provides enforcement middleware using the zero-trust system
package trust

import (
	"context"
	"fmt"
	"sync"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
)

// EnforcementPolicy defines trust requirements for specific operations
type EnforcementPolicy struct {
	// Operation name
	Operation string `json:"operation"`

	// Minimum trust level required
	MinTrustLevel TrustScore `json:"min_trust_level"`

	// Maximum risk score allowed (0-100)
	MaxRiskScore int `json:"max_risk_score"`

	// Whether MFA is required for this operation
	RequireMFA bool `json:"require_mfa"`

	// Whether verified device is required
	RequireVerifiedDevice bool `json:"require_verified_device"`

	// Allowed anomaly flags (empty = none allowed)
	AllowedAnomalies []string `json:"allowed_anomalies"`

	// Bypass trust check for specific conditions
	BypassConditions []string `json:"bypass_conditions,omitempty"`
}

// EnforcementResult represents the result of trust enforcement
type EnforcementResult struct {
	// Whether the operation is allowed
	Allowed bool `json:"allowed"`

	// Reason for denial (if not allowed)
	DenialReason string `json:"denial_reason,omitempty"`

	// Trust level of the request
	TrustLevel TrustScore `json:"trust_level"`

	// Risk score of the request
	RiskScore int `json:"risk_score"`

	// Anomalies detected
	Anomalies []string `json:"anomalies,omitempty"`

	// Required actions for the user
	RequiredActions []string `json:"required_actions,omitempty"`

	// Session ID
	SessionID string `json:"session_id"`
}

// TrustMiddleware provides trust enforcement for operations
type TrustMiddleware struct {
	manager       *ZeroTrustManager
	auditLog      *audit.TamperEvidentLog
	logger        *logger.Logger
	policies      map[string]EnforcementPolicy
	defaultPolicy EnforcementPolicy
	mu            sync.RWMutex
}

// TrustMiddlewareConfig configures the trust middleware
type TrustMiddlewareConfig struct {
	TrustManager  *ZeroTrustManager
	AuditLog      *audit.TamperEvidentLog
	Logger        *logger.Logger
	DefaultPolicy EnforcementPolicy
	Policies      []EnforcementPolicy
}

// NewTrustMiddleware creates a new trust enforcement middleware
func NewTrustMiddleware(cfg TrustMiddlewareConfig) *TrustMiddleware {
	if cfg.Logger == nil {
		cfg.Logger = logger.Global().WithComponent("trust_middleware")
	}

	// Set default policy
	defaultPolicy := cfg.DefaultPolicy
	if defaultPolicy.MinTrustLevel == 0 {
		defaultPolicy.MinTrustLevel = TrustScoreMedium
	}
	if defaultPolicy.MaxRiskScore == 0 {
		defaultPolicy.MaxRiskScore = 50
	}

	tm := &TrustMiddleware{
		manager:       cfg.TrustManager,
		auditLog:      cfg.AuditLog,
		logger:        cfg.Logger,
		policies:      make(map[string]EnforcementPolicy),
		defaultPolicy: defaultPolicy,
	}

	// Register policies
	for _, policy := range cfg.Policies {
		tm.policies[policy.Operation] = policy
	}

	return tm
}

// RegisterPolicy registers an enforcement policy for an operation
func (tm *TrustMiddleware) RegisterPolicy(policy EnforcementPolicy) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.policies[policy.Operation] = policy
}

// GetPolicy returns the policy for an operation
func (tm *TrustMiddleware) GetPolicy(operation string) EnforcementPolicy {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if policy, exists := tm.policies[operation]; exists {
		return policy
	}
	return tm.defaultPolicy
}

// Enforce performs trust enforcement for an operation
func (tm *TrustMiddleware) Enforce(ctx context.Context, operation string, req *ZeroTrustRequest) (*EnforcementResult, error) {
	// If no trust manager, allow all (backward compatible)
	if tm.manager == nil {
		return &EnforcementResult{
			Allowed:      true,
			TrustLevel:   TrustScoreHigh,
			DenialReason: "Trust enforcement disabled",
		}, nil
	}

	// Get policy for this operation
	policy := tm.GetPolicy(operation)

	// Perform verification
	result, err := tm.manager.Verify(ctx, req)
	if err != nil {
		tm.logger.Error("verification_failed",
			"operation", operation,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("trust verification failed: %w", err)
	}

	// Build enforcement result
	enforcement := &EnforcementResult{
		TrustLevel:      result.TrustLevel,
		RiskScore:       result.RiskScore,
		Anomalies:       result.AnomalyFlags,
		RequiredActions: result.RequiredActions,
		SessionID:       result.SessionID,
	}

	// Check trust level
	if result.TrustLevel < policy.MinTrustLevel {
		enforcement.Allowed = false
		enforcement.DenialReason = fmt.Sprintf("Trust level %s below required %s",
			result.TrustLevel.String(), policy.MinTrustLevel.String())
		tm.logEnforcement(ctx, operation, req, enforcement, policy)
		return enforcement, nil
	}

	// Check risk score
	if result.RiskScore > policy.MaxRiskScore {
		enforcement.Allowed = false
		enforcement.DenialReason = fmt.Sprintf("Risk score %d exceeds maximum %d",
			result.RiskScore, policy.MaxRiskScore)
		tm.logEnforcement(ctx, operation, req, enforcement, policy)
		return enforcement, nil
	}

	// Check MFA requirement
	if policy.RequireMFA && !tm.hasMFAVerified(result) {
		enforcement.Allowed = false
		enforcement.DenialReason = "MFA verification required"
		enforcement.RequiredActions = append(enforcement.RequiredActions, "mfa_required")
		tm.logEnforcement(ctx, operation, req, enforcement, policy)
		return enforcement, nil
	}

	// Check verified device requirement
	if policy.RequireVerifiedDevice {
		device, err := tm.manager.GetDeviceData(result.DeviceID)
		if err != nil || !device.Verified {
			enforcement.Allowed = false
			enforcement.DenialReason = "Verified device required"
			enforcement.RequiredActions = append(enforcement.RequiredActions, "device_verification")
			tm.logEnforcement(ctx, operation, req, enforcement, policy)
			return enforcement, nil
		}
	}

	// Check anomaly flags
	if len(result.AnomalyFlags) > 0 && len(policy.AllowedAnomalies) == 0 {
		enforcement.Allowed = false
		enforcement.DenialReason = fmt.Sprintf("Anomalies detected: %v", result.AnomalyFlags)
		tm.logEnforcement(ctx, operation, req, enforcement, policy)
		return enforcement, nil
	}

	// Check for disallowed anomalies
	for _, anomaly := range result.AnomalyFlags {
		allowed := false
		for _, allowedAnomaly := range policy.AllowedAnomalies {
			if anomaly == allowedAnomaly {
				allowed = true
				break
			}
		}
		if !allowed {
			enforcement.Allowed = false
			enforcement.DenialReason = fmt.Sprintf("Disallowed anomaly: %s", anomaly)
			tm.logEnforcement(ctx, operation, req, enforcement, policy)
			return enforcement, nil
		}
	}

	// All checks passed
	enforcement.Allowed = true
	tm.logEnforcement(ctx, operation, req, enforcement, policy)

	return enforcement, nil
}

// hasMFAVerified checks if MFA has been verified
func (tm *TrustMiddleware) hasMFAVerified(result *ZeroTrustResult) bool {
	// Check if MFA verification was done based on required actions
	for _, action := range result.RequiredActions {
		if action == "mfa_challenge" {
			return false
		}
	}
	return true
}

// logEnforcement logs the enforcement decision
func (tm *TrustMiddleware) logEnforcement(ctx context.Context, operation string, req *ZeroTrustRequest, result *EnforcementResult, policy EnforcementPolicy) {
	// Log to component logger
	if result.Allowed {
		tm.logger.Info("trust_enforcement_passed",
			"operation", operation,
			"trust_level", result.TrustLevel.String(),
			"risk_score", result.RiskScore,
			"session_id", result.SessionID,
		)
	} else {
		tm.logger.Warn("trust_enforcement_denied",
			"operation", operation,
			"trust_level", result.TrustLevel.String(),
			"risk_score", result.RiskScore,
			"reason", result.DenialReason,
			"session_id", result.SessionID,
			"anomalies", result.Anomalies,
		)
	}

	// Log to audit log
	if tm.auditLog != nil {
		actor := audit.Actor{
			Type:      "user",
			ID:        req.UserID,
			IPAddress: req.IPAddress,
		}
		resource := audit.Resource{
			Type: "operation",
			ID:   operation,
		}
		severity := "low"
		if !result.Allowed {
			severity = "high"
		}
		compliance := audit.ComplianceFlags{
			Category:      "trust_enforcement",
			Severity:      severity,
			AuditRequired: !result.Allowed,
		}

		eventType := "trust_enforcement_passed"
		if !result.Allowed {
			eventType = "trust_enforcement_denied"
		}

		_, _ = tm.auditLog.LogEntry(eventType, actor, operation, resource, map[string]interface{}{
			"allowed":         result.Allowed,
			"trust_level":     result.TrustLevel.String(),
			"risk_score":      result.RiskScore,
			"denial_reason":   result.DenialReason,
			"anomalies":       result.Anomalies,
			"required_actions": result.RequiredActions,
			"min_trust_required": policy.MinTrustLevel.String(),
			"max_risk_allowed":   policy.MaxRiskScore,
		}, compliance)
	}
}

// SetAuditLog updates the audit log
func (tm *TrustMiddleware) SetAuditLog(auditLog *audit.TamperEvidentLog) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.auditLog = auditLog
}

// SetTrustManager updates the trust manager
func (tm *TrustMiddleware) SetTrustManager(manager *ZeroTrustManager) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.manager = manager
}

// EnforceFunc returns a function that can be used as middleware
func (tm *TrustMiddleware) EnforceFunc(operation string) func(ctx context.Context, userID, ipAddress string, fingerprint DeviceFingerprintInput) (*EnforcementResult, error) {
	return func(ctx context.Context, userID, ipAddress string, fingerprint DeviceFingerprintInput) (*EnforcementResult, error) {
		req := &ZeroTrustRequest{
			UserID:            userID,
			IPAddress:         ipAddress,
			DeviceFingerprint: fingerprint,
			Action:            operation,
		}
		return tm.Enforce(ctx, operation, req)
	}
}

// DefaultPolicies returns sensible default policies for common operations
func DefaultPolicies() []EnforcementPolicy {
	return []EnforcementPolicy{
		{
			Operation:             "container_create",
			MinTrustLevel:        TrustScoreMedium,
			MaxRiskScore:         40,
			RequireVerifiedDevice: false,
		},
		{
			Operation:             "container_exec",
			MinTrustLevel:        TrustScoreHigh,
			MaxRiskScore:         30,
			RequireVerifiedDevice: true,
		},
		{
			Operation:             "secret_access",
			MinTrustLevel:        TrustScoreHigh,
			MaxRiskScore:         25,
			RequireVerifiedDevice: true,
			RequireMFA:           true,
		},
		{
			Operation:             "key_management",
			MinTrustLevel:        TrustScoreVerified,
			MaxRiskScore:         20,
			RequireVerifiedDevice: true,
			RequireMFA:           true,
		},
		{
			Operation:             "config_change",
			MinTrustLevel:        TrustScoreHigh,
			MaxRiskScore:         30,
			RequireVerifiedDevice: true,
		},
		{
			Operation:             "admin_access",
			MinTrustLevel:        TrustScoreVerified,
			MaxRiskScore:         15,
			RequireVerifiedDevice: true,
			RequireMFA:           true,
		},
		{
			Operation:             "message_send",
			MinTrustLevel:        TrustScoreLow,
			MaxRiskScore:         60,
		},
		{
			Operation:             "message_receive",
			MinTrustLevel:        TrustScoreLow,
			MaxRiskScore:         70,
		},
	}
}

// QuickEnforce is a helper for simple trust checks
func (tm *TrustMiddleware) QuickEnforce(ctx context.Context, operation, userID, ipAddress string) (*EnforcementResult, error) {
	req := &ZeroTrustRequest{
		UserID:    userID,
		IPAddress: ipAddress,
		Action:    operation,
	}
	return tm.Enforce(ctx, operation, req)
}

// WrapOperation wraps an operation function with trust enforcement
func (tm *TrustMiddleware) WrapOperation(operation string, fn func(ctx context.Context) error) func(ctx context.Context, userID, ipAddress string, fingerprint DeviceFingerprintInput) error {
	return func(ctx context.Context, userID, ipAddress string, fingerprint DeviceFingerprintInput) error {
		// Enforce trust
		result, err := tm.Enforce(ctx, operation, &ZeroTrustRequest{
			UserID:            userID,
			IPAddress:         ipAddress,
			DeviceFingerprint: fingerprint,
			Action:            operation,
		})
		if err != nil {
			return fmt.Errorf("trust enforcement error: %w", err)
		}
		if !result.Allowed {
			return fmt.Errorf("trust enforcement denied: %s", result.DenialReason)
		}

		// Execute the operation
		return fn(ctx)
	}
}
