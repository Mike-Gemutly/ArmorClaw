// Package enforcement provides enterprise feature enforcement based on license tiers.
// It ensures that premium features are only accessible to appropriately licensed users.
package enforcement

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/license"
)

// LicenseValidator defines the interface for license validation
type LicenseValidator interface {
	Validate(ctx context.Context, feature string) (bool, error)
	GetCached(feature string) *license.CachedLicense
}

// Feature represents an enterprise feature that requires a specific license tier
type Feature string

const (
	// Platform Bridging Features
	FeatureSlackBridge    Feature = "bridge.slack"
	FeatureDiscordBridge  Feature = "bridge.discord"
	FeatureTeamsBridge    Feature = "bridge.teams"
	FeatureWhatsAppBridge Feature = "bridge.whatsapp"

	// Compliance Features
	FeaturePHIScrubbing    Feature = "compliance.phi_scrubbing"
	FeatureHIPAAMode       Feature = "compliance.hipaa"
	FeatureAuditExport     Feature = "compliance.audit_export"
	FeatureTamperEvidence  Feature = "compliance.tamper_evidence"

	// Security Features
	FeatureSSO             Feature = "security.sso"
	FeatureSAML            Feature = "security.saml"
	FeatureMFAEnforcement  Feature = "security.mfa_enforcement"
	FeatureHardwareKeys    Feature = "security.hardware_keys"

	// Voice Features
	FeatureVoiceCalls      Feature = "voice.calls"
	FeatureVoiceRecording  Feature = "voice.recording"
	FeatureVoiceTranscript Feature = "voice.transcription"

	// Management Features
	FeatureDashboard       Feature = "management.dashboard"
	FeatureAPIAccess       Feature = "management.api"
	FeatureWebhooks        Feature = "management.webhooks"
	FeaturePrioritySupport Feature = "support.priority"

	// Platform Limits
	FeatureUnlimitedBridges Feature = "limits.unlimited_bridges"
	FeatureUnlimitedUsers   Feature = "limits.unlimited_users"
)

// FeatureDefinition describes a feature and its requirements
type FeatureDefinition struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	MinTier      license.Tier  `json:"min_tier"`
	Category     string        `json:"category"`
	Enabled      bool          `json:"enabled"`
	Compliance   bool          `json:"compliance"` // If true, affects PHI/HIPAA handling
}

// ComplianceMode represents the level of compliance enforcement
type ComplianceMode string

const (
	ComplianceModeNone     ComplianceMode = "none"
	ComplianceModeBasic    ComplianceMode = "basic"
	ComplianceModeStandard ComplianceMode = "standard"
	ComplianceModeFull     ComplianceMode = "full"
	ComplianceModeStrict   ComplianceMode = "strict"
)

// PlatformLimit represents limits for platform bridging
type PlatformLimit struct {
	Platform       string `json:"platform"`
	Enabled        bool   `json:"enabled"`
	MaxChannels    int    `json:"max_channels"`    // 0 = unlimited
	MaxUsers       int    `json:"max_users"`       // 0 = unlimited
	MessageLimit   int    `json:"message_limit"`   // per day, 0 = unlimited
	PHIScrubbing   bool   `json:"phi_scrubbing"`
	AuditLogging   bool   `json:"audit_logging"`
}

// EnforcementConfig configures the enforcement manager
type EnforcementConfig struct {
	// License client for tier validation
	LicenseClient LicenseValidator

	// Default compliance mode when license is invalid
	DefaultComplianceMode ComplianceMode

	// Enable grace period for expired licenses
	EnableGracePeriod bool
	GracePeriodDays   int

	// Strict mode - block all features on invalid license
	StrictMode bool

	// Logger
	Logger *slog.Logger
}

// Manager enforces feature access based on license tier
type Manager struct {
	config     EnforcementConfig
	features   map[Feature]*FeatureDefinition
	limits     map[string]*PlatformLimit
	license    *license.CachedLicense
	mu         sync.RWMutex
	logger     *slog.Logger
}

// NewManager creates a new enforcement manager
func NewManager(config EnforcementConfig) (*Manager, error) {
	if config.LicenseClient == nil {
		return nil, fmt.Errorf("license client is required")
	}

	if config.DefaultComplianceMode == "" {
		config.DefaultComplianceMode = ComplianceModeBasic
	}

	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "enforcement")
	}

	m := &Manager{
		config:   config,
		features: make(map[Feature]*FeatureDefinition),
		limits:   make(map[string]*PlatformLimit),
		logger:   config.Logger,
	}

	// Initialize default features
	m.initDefaultFeatures()
	m.initDefaultLimits()

	return m, nil
}

// initDefaultFeatures registers all feature definitions
func (m *Manager) initDefaultFeatures() {
	// Platform Bridging Features
	m.RegisterFeature(FeatureSlackBridge, FeatureDefinition{
		Name:        "Slack Bridge",
		Description: "Bridge Matrix rooms to Slack channels",
		MinTier:     license.TierFree,
		Category:    "bridging",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureDiscordBridge, FeatureDefinition{
		Name:        "Discord Bridge",
		Description: "Bridge Matrix rooms to Discord channels",
		MinTier:     license.TierPro,
		Category:    "bridging",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureTeamsBridge, FeatureDefinition{
		Name:        "Microsoft Teams Bridge",
		Description: "Bridge Matrix rooms to Teams channels",
		MinTier:     license.TierPro,
		Category:    "bridging",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureWhatsAppBridge, FeatureDefinition{
		Name:        "WhatsApp Bridge",
		Description: "Bridge Matrix rooms to WhatsApp",
		MinTier:     license.TierEnterprise,
		Category:    "bridging",
		Enabled:     true,
	})

	// Compliance Features
	m.RegisterFeature(FeaturePHIScrubbing, FeatureDefinition{
		Name:        "PHI Scrubbing",
		Description: "Automatic PHI detection and redaction",
		MinTier:     license.TierPro,
		Category:    "compliance",
		Enabled:     true,
		Compliance:  true,
	})

	m.RegisterFeature(FeatureHIPAAMode, FeatureDefinition{
		Name:        "HIPAA Mode",
		Description: "Full HIPAA compliance mode with audit trails",
		MinTier:     license.TierEnterprise,
		Category:    "compliance",
		Enabled:     true,
		Compliance:  true,
	})

	m.RegisterFeature(FeatureAuditExport, FeatureDefinition{
		Name:        "Audit Export",
		Description: "Export audit logs for compliance reporting",
		MinTier:     license.TierPro,
		Category:    "compliance",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureTamperEvidence, FeatureDefinition{
		Name:        "Tamper Evidence",
		Description: "Hash chain logging for tamper detection",
		MinTier:     license.TierEnterprise,
		Category:    "compliance",
		Enabled:     true,
		Compliance:  true,
	})

	// Security Features
	m.RegisterFeature(FeatureSSO, FeatureDefinition{
		Name:        "Single Sign-On",
		Description: "SSO integration (OIDC)",
		MinTier:     license.TierPro,
		Category:    "security",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureSAML, FeatureDefinition{
		Name:        "SAML 2.0",
		Description: "SAML 2.0 authentication",
		MinTier:     license.TierEnterprise,
		Category:    "security",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureMFAEnforcement, FeatureDefinition{
		Name:        "MFA Enforcement",
		Description: "Enforce multi-factor authentication",
		MinTier:     license.TierPro,
		Category:    "security",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureHardwareKeys, FeatureDefinition{
		Name:        "Hardware Security Keys",
		Description: "FIDO2 hardware key support",
		MinTier:     license.TierEnterprise,
		Category:    "security",
		Enabled:     true,
	})

	// Voice Features
	m.RegisterFeature(FeatureVoiceCalls, FeatureDefinition{
		Name:        "Voice Calls",
		Description: "Real-time voice communication",
		MinTier:     license.TierFree,
		Category:    "voice",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureVoiceRecording, FeatureDefinition{
		Name:        "Voice Recording",
		Description: "Record voice calls for compliance",
		MinTier:     license.TierEnterprise,
		Category:    "voice",
		Enabled:     true,
		Compliance:  true,
	})

	m.RegisterFeature(FeatureVoiceTranscript, FeatureDefinition{
		Name:        "Voice Transcription",
		Description: "Transcribe voice calls",
		MinTier:     license.TierEnterprise,
		Category:    "voice",
		Enabled:     true,
	})

	// Management Features
	m.RegisterFeature(FeatureDashboard, FeatureDefinition{
		Name:        "Web Dashboard",
		Description: "Web-based management interface",
		MinTier:     license.TierPro,
		Category:    "management",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureAPIAccess, FeatureDefinition{
		Name:        "REST API",
		Description: "REST API access for automation",
		MinTier:     license.TierPro,
		Category:    "management",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureWebhooks, FeatureDefinition{
		Name:        "Webhooks",
		Description: "Outbound webhook notifications",
		MinTier:     license.TierPro,
		Category:    "management",
		Enabled:     true,
	})

	m.RegisterFeature(FeaturePrioritySupport, FeatureDefinition{
		Name:        "Priority Support",
		Description: "Priority support access",
		MinTier:     license.TierEnterprise,
		Category:    "support",
		Enabled:     true,
	})

	// Platform Limits
	m.RegisterFeature(FeatureUnlimitedBridges, FeatureDefinition{
		Name:        "Unlimited Bridges",
		Description: "No limit on number of bridged channels",
		MinTier:     license.TierPro,
		Category:    "limits",
		Enabled:     true,
	})

	m.RegisterFeature(FeatureUnlimitedUsers, FeatureDefinition{
		Name:        "Unlimited Users",
		Description: "No limit on number of users",
		MinTier:     license.TierEnterprise,
		Category:    "limits",
		Enabled:     true,
	})
}

// initDefaultLimits sets platform-specific limits by tier
func (m *Manager) initDefaultLimits() {
	// Free tier limits
	m.SetPlatformLimit("slack", PlatformLimit{
		Platform:     "slack",
		Enabled:      true,
		MaxChannels:  3,
		MaxUsers:     10,
		MessageLimit: 1000,
		PHIScrubbing: false,
		AuditLogging: false,
	})

	// Discord requires Pro
	m.SetPlatformLimit("discord", PlatformLimit{
		Platform:     "discord",
		Enabled:      false, // Requires Pro
		MaxChannels:  10,
		MaxUsers:     50,
		MessageLimit: 5000,
		PHIScrubbing: false,
		AuditLogging: false,
	})

	// Teams requires Pro
	m.SetPlatformLimit("teams", PlatformLimit{
		Platform:     "teams",
		Enabled:      false, // Requires Pro
		MaxChannels:  10,
		MaxUsers:     50,
		MessageLimit: 5000,
		PHIScrubbing: false,
		AuditLogging: false,
	})

	// WhatsApp requires Enterprise
	m.SetPlatformLimit("whatsapp", PlatformLimit{
		Platform:     "whatsapp",
		Enabled:      false, // Requires Enterprise
		MaxChannels:  0,     // Unlimited
		MaxUsers:     0,     // Unlimited
		MessageLimit: 0,     // Unlimited
		PHIScrubbing: true,
		AuditLogging: true,
	})
}

// RegisterFeature registers a feature definition
func (m *Manager) RegisterFeature(feature Feature, def FeatureDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.features[feature] = &def
}

// SetPlatformLimit sets limits for a platform
func (m *Manager) SetPlatformLimit(platform string, limit PlatformLimit) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limits[platform] = &limit
}

// RefreshLicense refreshes the cached license from the server
func (m *Manager) RefreshLicense(ctx context.Context) error {
	// Validate with an empty feature to get the full license status
	_, err := m.config.LicenseClient.Validate(ctx, "")
	if err != nil {
		m.logger.Warn("license_validation_failed", "error", err)
		// Use cached license if available and within grace period
		if m.license != nil && m.license.IsValid() {
			m.logger.Info("using_cached_license", "grace_until", m.license.GraceUntil)
			return nil
		}
		return fmt.Errorf("license validation failed: %w", err)
	}

	// Get the cached license
	cached := m.config.LicenseClient.GetCached("")
	if cached == nil {
		return fmt.Errorf("license not cached after validation")
	}

	m.mu.Lock()
	m.license = cached
	m.mu.Unlock()

	m.logger.Info("license_refreshed",
		"tier", cached.Tier,
		"valid", cached.Valid,
		"expires", cached.ExpiresAt,
		"features", len(cached.Features),
	)

	return nil
}

// CheckFeature checks if a feature is accessible with the current license
func (m *Manager) CheckFeature(feature Feature) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get feature definition
	def, exists := m.features[feature]
	if !exists {
		return false, fmt.Errorf("unknown feature: %s", feature)
	}

	// Feature disabled globally
	if !def.Enabled {
		return false, nil
	}

	// Check license
	if m.license == nil {
		// No license - use free tier defaults
		return def.MinTier == license.TierFree, nil
	}

	// License invalid or expired
	if !m.license.IsValid() {
		if m.config.StrictMode {
			return false, fmt.Errorf("license invalid or expired")
		}
		// Grace mode - allow free tier features
		return def.MinTier == license.TierFree, nil
	}

	// Check tier requirement
	return m.tierSatisfies(m.license.Tier, def.MinTier), nil
}

// CheckFeatureOrPanic checks feature access and panics if denied
// Use for critical features that should never be called without validation
func (m *Manager) CheckFeatureOrPanic(feature Feature) bool {
	allowed, err := m.CheckFeature(feature)
	if err != nil {
		panic(fmt.Sprintf("feature check failed: %v", err))
	}
	if !allowed {
		panic(fmt.Sprintf("feature %s not licensed", feature))
	}
	return true
}

// GetComplianceMode returns the current compliance mode based on license
func (m *Manager) GetComplianceMode() ComplianceMode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// No license - basic mode only
	if m.license == nil || !m.license.IsValid() {
		return m.config.DefaultComplianceMode
	}

	// Determine compliance mode by tier
	switch m.license.Tier {
	case license.TierEnterprise:
		// Check for HIPAA feature
		if m.license.HasFeature(string(FeatureHIPAAMode)) {
			return ComplianceModeStrict
		}
		return ComplianceModeFull
	case license.TierPro:
		// Check for PHI scrubbing feature
		if m.license.HasFeature(string(FeaturePHIScrubbing)) {
			return ComplianceModeStandard
		}
		return ComplianceModeBasic
	default:
		return ComplianceModeBasic
	}
}

// GetPlatformLimit returns limits for a platform based on license
func (m *Manager) GetPlatformLimit(platform string) (*PlatformLimit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limit, exists := m.limits[platform]
	if !exists {
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}

	// Create a copy to avoid mutation
	result := *limit

	// Adjust based on license tier
	if m.license != nil && m.license.IsValid() {
		switch m.license.Tier {
		case license.TierEnterprise:
			// All platforms enabled, all limits removed
			result.Enabled = true
			result.MaxChannels = 0
			result.MaxUsers = 0
			result.MessageLimit = 0
			result.PHIScrubbing = true
			result.AuditLogging = true
		case license.TierPro:
			// Pro platforms enabled with increased limits
			if platform == "discord" || platform == "teams" {
				result.Enabled = true
				result.MaxChannels = 50
				result.MaxUsers = 200
				result.MessageLimit = 50000
			}
			if platform == "slack" {
				result.MaxChannels = 20
				result.MaxUsers = 100
				result.MessageLimit = 25000
			}
		}
	}

	return &result, nil
}

// CanBridgePlatform checks if bridging is allowed for a platform
func (m *Manager) CanBridgePlatform(platform string) (bool, error) {
	feature := Feature(platform + "_bridge")
	switch platform {
	case "slack":
		feature = FeatureSlackBridge
	case "discord":
		feature = FeatureDiscordBridge
	case "teams":
		feature = FeatureTeamsBridge
	case "whatsapp":
		feature = FeatureWhatsAppBridge
	default:
		return false, fmt.Errorf("unknown platform: %s", platform)
	}

	// Check feature flag
	allowed, err := m.CheckFeature(feature)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	// Check platform limits
	limit, err := m.GetPlatformLimit(platform)
	if err != nil {
		return false, err
	}

	return limit.Enabled, nil
}

// GetTier returns the current license tier
func (m *Manager) GetTier() license.Tier {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return license.TierFree
	}
	return m.license.Tier
}

// GetLicenseInfo returns license information for display
func (m *Manager) GetLicenseInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := map[string]interface{}{
		"tier":             license.TierFree,
		"valid":            false,
		"compliance_mode":  m.config.DefaultComplianceMode,
		"features":         []string{},
	}

	if m.license != nil {
		info["tier"] = m.license.Tier
		info["valid"] = m.license.IsValid()
		info["features"] = m.license.Features
		info["expires_at"] = m.license.ExpiresAt
		info["grace_until"] = m.license.GraceUntil
	}

	info["compliance_mode"] = m.GetComplianceMode()

	return info
}

// tierSatisfies checks if actual tier meets required minimum
func (m *Manager) tierSatisfies(actual, required license.Tier) bool {
	tierOrder := map[license.Tier]int{
		license.TierFree:       0,
		license.TierPro:        1,
		license.TierEnterprise: 2,
	}

	return tierOrder[actual] >= tierOrder[required]
}

// GetAllFeatures returns all registered features
func (m *Manager) GetAllFeatures() map[Feature]*FeatureDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[Feature]*FeatureDefinition)
	for k, v := range m.features {
		copy := *v
		result[k] = &copy
	}
	return result
}

// GetFeaturesByCategory returns features filtered by category
func (m *Manager) GetFeaturesByCategory(category string) []Feature {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Feature
	for f, def := range m.features {
		if def.Category == category {
			result = append(result, f)
		}
	}
	return result
}

// GetAvailableFeatures returns features available with current license
func (m *Manager) GetAvailableFeatures() []Feature {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Feature
	for f, def := range m.features {
		if m.tierSatisfies(m.GetTier(), def.MinTier) {
			result = append(result, f)
		}
	}
	return result
}

// StartPeriodicRefresh starts a background goroutine to refresh the license
func (m *Manager) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.RefreshLicense(ctx); err != nil {
					m.logger.Error("periodic_license_refresh_failed", "error", err)
				}
			}
		}
	}()
}
