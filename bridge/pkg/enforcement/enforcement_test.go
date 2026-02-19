// Package enforcement provides enterprise feature enforcement based on license tiers.
package enforcement

import (
	"context"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/license"
)

// mockLicenseClient creates a mock license client for testing
type mockLicenseClient struct {
	tier     license.Tier
	valid    bool
	features []string
	cached   *license.CachedLicense
}

func (m *mockLicenseClient) Validate(ctx context.Context, feature string) (bool, error) {
	return m.valid, nil
}

func (m *mockLicenseClient) GetCached(feature string) *license.CachedLicense {
	if m.cached != nil {
		return m.cached
	}
	return &license.CachedLicense{
		Valid:     m.valid,
		Tier:      m.tier,
		Features:  m.features,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CachedAt:  time.Now(),
		GraceUntil: time.Now().Add(72 * time.Hour),
	}
}

func (m *mockLicenseClient) SetCached(feature string, cached *license.CachedLicense) {
	m.cached = cached
}

func TestNewManager(t *testing.T) {
	mockClient := &mockLicenseClient{tier: license.TierFree, valid: true}

	tests := []struct {
		name    string
		config  EnforcementConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: EnforcementConfig{
				LicenseClient: mockClient,
			},
			wantErr: false,
		},
		{
			name: "missing license client",
			config: EnforcementConfig{
				LicenseClient: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mgr == nil {
				t.Error("NewManager() returned nil without error")
			}
		})
	}
}

func TestFeatureRegistration(t *testing.T) {
	mockClient := &mockLicenseClient{tier: license.TierFree, valid: true}
	mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})

	// Test default features are registered
	features := mgr.GetAllFeatures()
	if len(features) == 0 {
		t.Error("No default features registered")
	}

	// Test specific features exist
	expectedFeatures := []Feature{
		FeatureSlackBridge,
		FeatureDiscordBridge,
		FeaturePHIScrubbing,
		FeatureHIPAAMode,
		FeatureSSO,
	}

	for _, f := range expectedFeatures {
		if _, exists := features[f]; !exists {
			t.Errorf("Expected feature %s not registered", f)
		}
	}
}

func TestCheckFeature(t *testing.T) {
	tests := []struct {
		name      string
		tier      license.Tier
		feature   Feature
		wantAllow bool
	}{
		// Free tier tests
		{"free slack bridge", license.TierFree, FeatureSlackBridge, true},
		{"free discord bridge", license.TierFree, FeatureDiscordBridge, false},
		{"free PHI scrubbing", license.TierFree, FeaturePHIScrubbing, false},

		// Pro tier tests
		{"pro slack bridge", license.TierPro, FeatureSlackBridge, true},
		{"pro discord bridge", license.TierPro, FeatureDiscordBridge, true},
		{"pro PHI scrubbing", license.TierPro, FeaturePHIScrubbing, true},
		{"pro HIPAA mode", license.TierPro, FeatureHIPAAMode, false},

		// Enterprise tier tests
		{"ent all features", license.TierEnterprise, FeatureHIPAAMode, true},
		{"ent whatsapp", license.TierEnterprise, FeatureWhatsAppBridge, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockLicenseClient{
				tier:     tt.tier,
				valid:    true,
				features: []string{string(tt.feature)},
			}

			mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
			mgr.license = mockClient.GetCached("")

			allowed, err := mgr.CheckFeature(tt.feature)
			if err != nil {
				t.Errorf("CheckFeature() error = %v", err)
				return
			}

			if allowed != tt.wantAllow {
				t.Errorf("CheckFeature(%s) for tier %s = %v, want %v",
					tt.feature, tt.tier, allowed, tt.wantAllow)
			}
		})
	}
}

func TestGetComplianceMode(t *testing.T) {
	tests := []struct {
		name      string
		tier      license.Tier
		features  []string
		wantMode  ComplianceMode
	}{
		{"free tier", license.TierFree, []string{}, ComplianceModeBasic},
		{"pro no PHI", license.TierPro, []string{}, ComplianceModeBasic},
		{"pro with PHI", license.TierPro, []string{"compliance.phi_scrubbing"}, ComplianceModeStandard},
		{"enterprise no HIPAA", license.TierEnterprise, []string{}, ComplianceModeFull},
		{"enterprise with HIPAA", license.TierEnterprise, []string{"compliance.hipaa"}, ComplianceModeStrict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockLicenseClient{
				tier:     tt.tier,
				valid:    true,
				features: tt.features,
			}

			mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
			mgr.license = mockClient.GetCached("")

			mode := mgr.GetComplianceMode()
			if mode != tt.wantMode {
				t.Errorf("GetComplianceMode() = %s, want %s", mode, tt.wantMode)
			}
		})
	}
}

func TestPlatformLimits(t *testing.T) {
	mockClient := &mockLicenseClient{tier: license.TierFree, valid: true}
	mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})

	// Test default limits
	slackLimit, err := mgr.GetPlatformLimit("slack")
	if err != nil {
		t.Fatalf("GetPlatformLimit(slack) error = %v", err)
	}

	if !slackLimit.Enabled {
		t.Error("Slack should be enabled by default")
	}
	if slackLimit.MaxChannels != 3 {
		t.Errorf("Free tier slack MaxChannels = %d, want 3", slackLimit.MaxChannels)
	}

	// Test enterprise tier removes limits
	mockClient.tier = license.TierEnterprise
	mgr.license = mockClient.GetCached("")

	entLimit, _ := mgr.GetPlatformLimit("slack")
	if entLimit.MaxChannels != 0 {
		t.Errorf("Enterprise tier slack MaxChannels = %d, want 0 (unlimited)", entLimit.MaxChannels)
	}
	if !entLimit.PHIScrubbing {
		t.Error("Enterprise tier should have PHI scrubbing enabled")
	}
}

func TestCanBridgePlatform(t *testing.T) {
	tests := []struct {
		name       string
		tier       license.Tier
		platform   string
		wantAllow  bool
	}{
		{"free slack", license.TierFree, "slack", true},
		{"free discord", license.TierFree, "discord", false},
		{"pro discord", license.TierPro, "discord", true},
		{"pro whatsapp", license.TierPro, "whatsapp", false},
		{"ent whatsapp", license.TierEnterprise, "whatsapp", true},
		{"unknown platform", license.TierEnterprise, "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockLicenseClient{tier: tt.tier, valid: true}
			mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
			mgr.license = mockClient.GetCached("")

			allowed, err := mgr.CanBridgePlatform(tt.platform)
			if tt.platform == "unknown" {
				if err == nil {
					t.Error("CanBridgePlatform(unknown) should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("CanBridgePlatform() error = %v", err)
				return
			}

			if allowed != tt.wantAllow {
				t.Errorf("CanBridgePlatform(%s) for tier %s = %v, want %v",
					tt.platform, tt.tier, allowed, tt.wantAllow)
			}
		})
	}
}

func TestGetAvailableFeatures(t *testing.T) {
	mockClient := &mockLicenseClient{tier: license.TierPro, valid: true}
	mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
	mgr.license = mockClient.GetCached("")

	available := mgr.GetAvailableFeatures()

	// Pro should have slack, discord, teams but not whatsapp
	hasSlack := false
	hasWhatsApp := false

	for _, f := range available {
		if f == FeatureSlackBridge {
			hasSlack = true
		}
		if f == FeatureWhatsAppBridge {
			hasWhatsApp = true
		}
	}

	if !hasSlack {
		t.Error("Pro tier should have Slack bridge")
	}
	if hasWhatsApp {
		t.Error("Pro tier should not have WhatsApp bridge")
	}
}

func TestGetLicenseInfo(t *testing.T) {
	mockClient := &mockLicenseClient{
		tier:     license.TierEnterprise,
		valid:    true,
		features: []string{"compliance.hipaa"},
	}
	mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
	mgr.license = mockClient.GetCached("")

	info := mgr.GetLicenseInfo()

	if info["tier"] != license.TierEnterprise {
		t.Errorf("GetLicenseInfo() tier = %v, want %s", info["tier"], license.TierEnterprise)
	}

	if info["valid"] != true {
		t.Error("GetLicenseInfo() valid should be true")
	}

	if info["compliance_mode"] != ComplianceModeStrict {
		t.Errorf("GetLicenseInfo() compliance_mode = %v, want %s", info["compliance_mode"], ComplianceModeStrict)
	}
}

func TestRPCEnforcer(t *testing.T) {
	mockClient := &mockLicenseClient{tier: license.TierFree, valid: true}
	mgr, _ := NewManager(EnforcementConfig{LicenseClient: mockClient})
	mgr.license = mockClient.GetCached("")

	enforcer := NewRPCEnforcer(mgr)

	// Test method that should be allowed
	allowed, msg := enforcer.CheckMethod("bridge.channel")
	if !allowed {
		t.Errorf("CheckMethod(bridge.channel) for free tier should be allowed, got: %s", msg)
	}

	// Test method that requires higher tier
	allowed, msg = enforcer.CheckMethod("platform.connect.discord")
	if allowed {
		t.Error("CheckMethod(platform.connect.discord) for free tier should not be allowed")
	}
	if msg == "" {
		t.Error("Expected error message for denied feature")
	}

	// Test stats
	stats := enforcer.GetEnforcementStats()
	if stats == nil {
		t.Error("GetEnforcementStats() should return stats")
	}
}
