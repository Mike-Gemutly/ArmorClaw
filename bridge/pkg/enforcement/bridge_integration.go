// Package enforcement provides integration with the AppService bridge
package enforcement

import (
	"context"
	"fmt"

	"github.com/armorclaw/bridge/pkg/appservice"
)

// BridgeEnforcer enforces license requirements for bridge operations
type BridgeEnforcer struct {
	manager *Manager
}

// NewBridgeEnforcer creates a new bridge enforcer
func NewBridgeEnforcer(manager *Manager) *BridgeEnforcer {
	return &BridgeEnforcer{manager: manager}
}

// CheckBridgeStart checks if bridge can be started with current license
func (b *BridgeEnforcer) CheckBridgeStart() error {
	// Bridge requires at least Slack bridge feature (free tier)
	allowed, err := b.manager.CheckFeature(FeatureSlackBridge)
	if err != nil {
		return fmt.Errorf("failed to check bridge feature: %w", err)
	}
	if !allowed {
		return fmt.Errorf("bridge feature not licensed")
	}
	return nil
}

// CheckPlatformConnect checks if connecting to a platform is allowed
func (b *BridgeEnforcer) CheckPlatformConnect(platform appservice.Platform) error {
	allowed, err := b.manager.CanBridgePlatform(string(platform))
	if err != nil {
		return fmt.Errorf("failed to check platform %s: %w", platform, err)
	}
	if !allowed {
		limit, _ := b.manager.GetPlatformLimit(string(platform))
		return fmt.Errorf("%s bridging not available (requires %s tier)",
			platform, b.getRequiredTierMessage(string(platform), limit))
	}
	return nil
}

// CheckChannelBridge checks if creating a new bridge is allowed
func (b *BridgeEnforcer) CheckChannelBridge(platform appservice.Platform, currentCount int) error {
	// First check if platform is allowed
	if err := b.CheckPlatformConnect(platform); err != nil {
		return err
	}

	// Check channel limit
	limit, err := b.manager.GetPlatformLimit(string(platform))
	if err != nil {
		return fmt.Errorf("failed to get platform limit: %w", err)
	}

	// 0 means unlimited
	if limit.MaxChannels > 0 && currentCount >= limit.MaxChannels {
		return fmt.Errorf("%s channel limit reached (%d/%d). Upgrade to increase limit.",
			platform, currentCount, limit.MaxChannels)
	}

	return nil
}

// CheckPHIScrubbing checks if PHI scrubbing should be applied
func (b *BridgeEnforcer) CheckPHIScrubbing() (bool, ComplianceMode) {
	mode := b.manager.GetComplianceMode()

	// PHI scrubbing required for standard and above
	switch mode {
	case ComplianceModeStandard, ComplianceModeFull, ComplianceModeStrict:
		return true, mode
	default:
		return false, mode
	}
}

// CheckAuditLogging checks if audit logging is required
func (b *BridgeEnforcer) CheckAuditLogging() bool {
	mode := b.manager.GetComplianceMode()

	// Audit logging required for full and strict modes
	switch mode {
	case ComplianceModeFull, ComplianceModeStrict:
		return true
	default:
		return false
	}
}

// GetBridgeLimits returns all limits for bridge operations
func (b *BridgeEnforcer) GetBridgeLimits() *BridgeLimits {
	limits := &BridgeLimits{
		Tier:            string(b.manager.GetTier()),
		ComplianceMode:  b.manager.GetComplianceMode(),
		Platforms:       make(map[string]*PlatformLimitInfo),
	}

	platforms := []string{"slack", "discord", "teams", "whatsapp"}
	for _, p := range platforms {
		limit, _ := b.manager.GetPlatformLimit(p)
		limits.Platforms[p] = &PlatformLimitInfo{
			Enabled:      limit.Enabled,
			MaxChannels:  limit.MaxChannels,
			MaxUsers:     limit.MaxUsers,
			MessageLimit: limit.MessageLimit,
			PHIScrubbing: limit.PHIScrubbing,
			AuditLogging: limit.AuditLogging,
		}
	}

	return limits
}

// getRequiredTierMessage returns a message about required tier
func (b *BridgeEnforcer) getRequiredTierMessage(platform string, limit *PlatformLimit) string {
	if limit == nil || !limit.Enabled {
		switch platform {
		case "discord", "teams":
			return "Pro or higher"
		case "whatsapp":
			return "Enterprise"
		default:
			return "a higher tier"
		}
	}
	return "current tier"
}

// BridgeLimits contains all limits for bridge operations
type BridgeLimits struct {
	Tier           string                      `json:"tier"`
	ComplianceMode ComplianceMode              `json:"compliance_mode"`
	Platforms      map[string]*PlatformLimitInfo `json:"platforms"`
}

// PlatformLimitInfo contains limit info for a platform
type PlatformLimitInfo struct {
	Enabled      bool `json:"enabled"`
	MaxChannels  int  `json:"max_channels"`
	MaxUsers     int  `json:"max_users"`
	MessageLimit int  `json:"message_limit"`
	PHIScrubbing bool `json:"phi_scrubbing"`
	AuditLogging bool `json:"audit_logging"`
}

// BridgeHook provides hooks for the BridgeManager
type BridgeHook struct {
	enforcer *BridgeEnforcer
}

// NewBridgeHook creates a new bridge hook
func NewBridgeHook(enforcer *BridgeEnforcer) *BridgeHook {
	return &BridgeHook{enforcer: enforcer}
}

// BeforeBridgeStart is called before the bridge starts
func (h *BridgeHook) BeforeBridgeStart() error {
	return h.enforcer.CheckBridgeStart()
}

// BeforeAdapterStart is called before starting a platform adapter
func (h *BridgeHook) BeforeAdapterStart(platform string) error {
	return h.enforcer.CheckPlatformConnect(appservice.Platform(platform))
}

// BeforeChannelBridge is called before creating a new bridge
func (h *BridgeHook) BeforeChannelBridge(platform string, currentCount int) error {
	return h.enforcer.CheckChannelBridge(appservice.Platform(platform), currentCount)
}

// ShouldScrubPHI returns whether PHI should be scrubbed
func (h *BridgeHook) ShouldScrubPHI() (bool, ComplianceMode) {
	return h.enforcer.CheckPHIScrubbing()
}

// ShouldAuditLog returns whether audit logging is required
func (h *BridgeHook) ShouldAuditLog() bool {
	return h.enforcer.CheckAuditLogging()
}

// GetComplianceConfig returns compliance configuration
func (h *BridgeHook) GetComplianceConfig() *ComplianceConfig {
	shouldScrub, mode := h.enforcer.CheckPHIScrubbing()
	shouldAudit := h.enforcer.CheckAuditLogging()

	return &ComplianceConfig{
		Mode:              mode,
		PHIScrubbing:      shouldScrub,
		AuditLogging:      shouldAudit,
		TamperEvidence:    mode == ComplianceModeStrict,
		QuarantineEnabled: mode == ComplianceModeStrict,
	}
}

// ComplianceConfig contains compliance settings
type ComplianceConfig struct {
	Mode              ComplianceMode `json:"mode"`
	PHIScrubbing      bool           `json:"phi_scrubbing"`
	AuditLogging      bool           `json:"audit_logging"`
	TamperEvidence    bool           `json:"tamper_evidence"`
	QuarantineEnabled bool           `json:"quarantine_enabled"`
}

// LicenseStatusHandler provides license status for RPC
type LicenseStatusHandler struct {
	manager *Manager
}

// NewLicenseStatusHandler creates a new license status handler
func NewLicenseStatusHandler(manager *Manager) *LicenseStatusHandler {
	return &LicenseStatusHandler{manager: manager}
}

// GetStatus returns the current license status
func (h *LicenseStatusHandler) GetStatus() map[string]interface{} {
	info := h.manager.GetLicenseInfo()
	limits := h.manager.GetComplianceMode()

	status := map[string]interface{}{
		"tier":             info["tier"],
		"valid":            info["valid"],
		"compliance_mode":  limits,
		"features":         h.manager.GetAvailableFeatures(),
		"license_info":     info,
	}

	// Add platform availability
	platforms := map[string]bool{}
	for _, p := range []string{"slack", "discord", "teams", "whatsapp"} {
		allowed, _ := h.manager.CanBridgePlatform(p)
		platforms[p] = allowed
	}
	status["platforms"] = platforms

	return status
}

// RefreshLicense refreshes the license from the server
func (h *LicenseStatusHandler) RefreshLicense(ctx context.Context) error {
	return h.manager.RefreshLicense(ctx)
}

// CheckFeatureAccess checks if a feature is accessible
func (h *LicenseStatusHandler) CheckFeatureAccess(feature string) (bool, error) {
	return h.manager.CheckFeature(Feature(feature))
}

// GetComplianceMode returns the current compliance mode
func (h *LicenseStatusHandler) GetComplianceMode() ComplianceMode {
	return h.manager.GetComplianceMode()
}
