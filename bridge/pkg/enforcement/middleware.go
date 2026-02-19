// Package enforcement provides middleware for enforcing license requirements
package enforcement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/armorclaw/bridge/pkg/license"
)

// Middleware provides HTTP middleware for license enforcement
type Middleware struct {
	manager *Manager
}

// NewMiddleware creates a new enforcement middleware
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{manager: manager}
}

// RequireFeature returns middleware that checks for a specific feature
func (m *Middleware) RequireFeature(feature Feature) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := m.manager.CheckFeature(feature)
			if err != nil {
				m.writeError(w, http.StatusInternalServerError, "FEATURE_CHECK_ERROR", err.Error())
				return
			}

			if !allowed {
				m.writeLicenseError(w, feature)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTier returns middleware that checks for a minimum tier
func (m *Middleware) RequireTier(minTier license.Tier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentTier := m.manager.GetTier()

			if !m.manager.tierSatisfies(currentTier, minTier) {
				m.writeTierError(w, minTier, currentTier)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireCompliance returns middleware that checks for minimum compliance mode
func (m *Middleware) RequireCompliance(minMode ComplianceMode) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentMode := m.manager.GetComplianceMode()

			if !m.complianceSatisfies(currentMode, minMode) {
				m.writeComplianceError(w, minMode, currentMode)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PlatformBridgeMiddleware checks if platform bridging is allowed
func (m *Middleware) PlatformBridgeMiddleware(platform string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := m.manager.CanBridgePlatform(platform)
			if err != nil {
				m.writeError(w, http.StatusInternalServerError, "PLATFORM_CHECK_ERROR", err.Error())
				return
			}

			if !allowed {
				m.writePlatformError(w, platform)
				return
			}

			// Add platform limit to context
			limit, _ := m.manager.GetPlatformLimit(platform)
			ctx := context.WithValue(r.Context(), "platform_limit", limit)
			ctx = context.WithValue(ctx, "platform", platform)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeLicenseError writes a license-required error response
func (m *Middleware) writeLicenseError(w http.ResponseWriter, feature Feature) {
	def := m.manager.features[feature]
	if def == nil {
		m.writeError(w, http.StatusForbidden, "FEATURE_NOT_FOUND", "Feature not found")
		return
	}

	response := map[string]interface{}{
		"error":             "LICENSE_REQUIRED",
		"message":           fmt.Sprintf("Feature '%s' requires %s tier or higher", def.Name, def.MinTier),
		"feature":           feature,
		"feature_name":      def.Name,
		"required_tier":     def.MinTier,
		"current_tier":      m.manager.GetTier(),
		"upgrade_url":       "https://armorclaw.com/pricing",
		"documentation_url": "https://docs.armorclaw.com/features/" + def.Category,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired) // 402 Payment Required
	json.NewEncoder(w).Encode(response)
}

// writeTierError writes a tier-required error response
func (m *Middleware) writeTierError(w http.ResponseWriter, required, current license.Tier) {
	response := map[string]interface{}{
		"error":         "TIER_REQUIRED",
		"message":       fmt.Sprintf("This action requires %s tier or higher", required),
		"required_tier": required,
		"current_tier":  current,
		"upgrade_url":   "https://armorclaw.com/pricing",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	json.NewEncoder(w).Encode(response)
}

// writeComplianceError writes a compliance-mode error response
func (m *Middleware) writeComplianceError(w http.ResponseWriter, required, current ComplianceMode) {
	response := map[string]interface{}{
		"error":            "COMPLIANCE_REQUIRED",
		"message":          fmt.Sprintf("This action requires %s compliance mode", required),
		"required_mode":    required,
		"current_mode":     current,
		"upgrade_url":      "https://armorclaw.com/pricing",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(response)
}

// writePlatformError writes a platform-not-licensed error response
func (m *Middleware) writePlatformError(w http.ResponseWriter, platform string) {
	limit, _ := m.manager.GetPlatformLimit(platform)

	response := map[string]interface{}{
		"error":      "PLATFORM_NOT_LICENSED",
		"message":    fmt.Sprintf("%s bridging is not available on your current tier", platform),
		"platform":   platform,
		"limit":      limit,
		"upgrade_url": "https://armorclaw.com/pricing",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	json.NewEncoder(w).Encode(response)
}

// writeError writes a generic error response
func (m *Middleware) writeError(w http.ResponseWriter, code int, errorType, message string) {
	response := map[string]interface{}{
		"error":   errorType,
		"message": message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

// complianceSatisfies checks if current mode meets required minimum
func (m *Middleware) complianceSatisfies(current, required ComplianceMode) bool {
	modeOrder := map[ComplianceMode]int{
		ComplianceModeNone:     0,
		ComplianceModeBasic:    1,
		ComplianceModeStandard: 2,
		ComplianceModeFull:     3,
		ComplianceModeStrict:   4,
	}

	return modeOrder[current] >= modeOrder[required]
}

// RPCEnforcer provides license enforcement for RPC methods
type RPCEnforcer struct {
	manager *Manager
}

// NewRPCEnforcer creates a new RPC enforcer
func NewRPCEnforcer(manager *Manager) *RPCEnforcer {
	return &RPCEnforcer{manager: manager}
}

// CheckMethod checks if an RPC method is allowed
func (r *RPCEnforcer) CheckMethod(method string) (bool, string) {
	// Map RPC methods to required features
	methodFeatures := map[string]Feature{
		"bridge.channel":          FeatureSlackBridge, // Base bridge feature
		"bridge.list_channels":    FeatureSlackBridge,
		"bridge.start":            FeatureSlackBridge,

		// Platform-specific methods
		"platform.connect.discord": FeatureDiscordBridge,
		"platform.connect.teams":   FeatureTeamsBridge,
		"platform.connect.whatsapp": FeatureWhatsAppBridge,

		// Compliance methods
		"compliance.phi_scrub":     FeaturePHIScrubbing,
		"compliance.hipaa_mode":    FeatureHIPAAMode,
		"compliance.audit_export":  FeatureAuditExport,

		// Security methods
		"security.sso_configure":   FeatureSSO,
		"security.saml_configure":  FeatureSAML,
		"security.mfa_enforce":     FeatureMFAEnforcement,
		"security.hardware_keys":   FeatureHardwareKeys,

		// Voice methods
		"voice.record":    FeatureVoiceRecording,
		"voice.transcribe": FeatureVoiceTranscript,

		// Management methods
		"dashboard.access": FeatureDashboard,
		"api.generate_key": FeatureAPIAccess,
		"webhook.create":   FeatureWebhooks,
	}

	// Check if method requires a feature
	feature, requiresFeature := methodFeatures[method]
	if !requiresFeature {
		// Method doesn't require a specific feature
		return true, ""
	}

	allowed, err := r.manager.CheckFeature(feature)
	if err != nil {
		return false, fmt.Sprintf("Feature check error: %s", err.Error())
	}

	if !allowed {
		def := r.manager.features[feature]
		if def != nil {
			return false, fmt.Sprintf("Feature '%s' requires %s tier", def.Name, def.MinTier)
		}
		return false, fmt.Sprintf("Feature %s not licensed", feature)
	}

	return true, ""
}

// CheckPlatformMethod checks if a platform-specific RPC method is allowed
func (r *RPCEnforcer) CheckPlatformMethod(platform string) (bool, string) {
	allowed, err := r.manager.CanBridgePlatform(platform)
	if err != nil {
		return false, err.Error()
	}

	if !allowed {
		return false, fmt.Sprintf("%s bridging requires %s tier (current: %s)",
			platform, r.getRequiredTier(platform), r.manager.GetTier())
	}

	return true, ""
}

// getRequiredTier returns the tier required for a platform
func (r *RPCEnforcer) getRequiredTier(platform string) license.Tier {
	feature := FeatureSlackBridge // default
	switch platform {
	case "slack":
		feature = FeatureSlackBridge
	case "discord":
		feature = FeatureDiscordBridge
	case "teams":
		feature = FeatureTeamsBridge
	case "whatsapp":
		feature = FeatureWhatsAppBridge
	}

	if def, exists := r.manager.features[feature]; exists {
		return def.MinTier
	}
	return license.TierEnterprise
}

// GetEnforcementStats returns statistics about enforcement
func (r *RPCEnforcer) GetEnforcementStats() map[string]interface{} {
	return map[string]interface{}{
		"tier":             r.manager.GetTier(),
		"compliance_mode":  r.manager.GetComplianceMode(),
		"license_info":     r.manager.GetLicenseInfo(),
		"available_features": len(r.manager.GetAvailableFeatures()),
		"total_features":   len(r.manager.GetAllFeatures()),
	}
}
