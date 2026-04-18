// Package config provides configuration tests for ArmorClaw bridge.
package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test server defaults
	if cfg.Server.SocketPath == "" {
		t.Error("SocketPath should not be empty")
	}
	if cfg.Server.Daemonize {
		t.Error("Daemonize should default to false")
	}

	// Test matrix defaults
	if cfg.Matrix.DeviceID != "armorclaw-bridge" {
		t.Errorf("DeviceID should be 'armorclaw-bridge', got %s", cfg.Matrix.DeviceID)
	}
	if cfg.Matrix.SyncInterval != 5 {
		t.Errorf("SyncInterval should be 5, got %d", cfg.Matrix.SyncInterval)
	}

	// Test zero-trust defaults
	if len(cfg.Matrix.ZeroTrust.TrustedSenders) != 0 {
		t.Error("TrustedSenders should default to empty (allow all)")
	}
	if len(cfg.Matrix.ZeroTrust.TrustedRooms) != 0 {
		t.Error("TrustedRooms should default to empty (allow all)")
	}
	if cfg.Matrix.ZeroTrust.RejectUntrusted {
		t.Error("RejectUntrusted should default to false (silent drop)")
	}

	// Test budget defaults
	if cfg.Budget.DailyLimitUSD != 5.00 {
		t.Errorf("DailyLimitUSD should default to 5.00, got %f", cfg.Budget.DailyLimitUSD)
	}
	if cfg.Budget.MonthlyLimitUSD != 100.00 {
		t.Errorf("MonthlyLimitUSD should default to 100.00, got %f", cfg.Budget.MonthlyLimitUSD)
	}
	if cfg.Budget.AlertThreshold != 80.0 {
		t.Errorf("AlertThreshold should default to 80.0, got %f", cfg.Budget.AlertThreshold)
	}
	if !cfg.Budget.HardStop {
		t.Error("HardStop should default to true")
	}
	if cfg.Budget.ProviderCosts == nil {
		t.Error("ProviderCosts should be initialized")
	}
}

func TestValidate(t *testing.T) {
	cfg := DefaultConfig()

	// Valid default config should pass validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig validation failed: %v", err)
	}

	// Test invalid socket path
	cfg.Server.SocketPath = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation error for empty SocketPath")
	}

	// Test invalid log level
	cfg = DefaultConfig()
	cfg.Logging.Level = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation error for invalid log level")
	}
}

func TestToMatrixConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Matrix.ZeroTrust.TrustedSenders = []string{"@user:example.com", "*@trusted.com"}
	cfg.Matrix.ZeroTrust.TrustedRooms = []string{"!room:example.com"}
	cfg.Matrix.ZeroTrust.RejectUntrusted = true

	matrixCfg := cfg.ToMatrixConfig()

	if matrixCfg.HomeserverURL != cfg.Matrix.HomeserverURL {
		t.Error("HomeserverURL not copied correctly")
	}
	if matrixCfg.DeviceID != cfg.Matrix.DeviceID {
		t.Error("DeviceID not copied correctly")
	}

	// Test zero-trust fields
	if len(matrixCfg.TrustedSenders) != 2 {
		t.Errorf("Expected 2 trusted senders, got %d", len(matrixCfg.TrustedSenders))
	}
	if matrixCfg.TrustedSenders[0] != "@user:example.com" {
		t.Errorf("Expected '@user:example.com', got %s", matrixCfg.TrustedSenders[0])
	}
	if len(matrixCfg.TrustedRooms) != 1 {
		t.Errorf("Expected 1 trusted room, got %d", len(matrixCfg.TrustedRooms))
	}
	if !matrixCfg.RejectUntrusted {
		t.Error("RejectUntrusted should be true")
	}
}

func TestToBudgetConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Budget.DailyLimitUSD = 10.00
	cfg.Budget.MonthlyLimitUSD = 200.00
	cfg.Budget.HardStop = false
	cfg.Budget.ProviderCosts = map[string]float64{
		"gpt-4": 30.00,
	}

	budgetCfg := cfg.ToBudgetConfig()

	if budgetCfg.DailyLimitUSD != 10.00 {
		t.Errorf("Expected DailyLimitUSD 10.00, got %f", budgetCfg.DailyLimitUSD)
	}
	if budgetCfg.MonthlyLimitUSD != 200.00 {
		t.Errorf("Expected MonthlyLimitUSD 200.00, got %f", budgetCfg.MonthlyLimitUSD)
	}
	if budgetCfg.HardStop {
		t.Error("Expected HardStop to be false")
	}
	if len(budgetCfg.ProviderCosts) != 1 {
		t.Errorf("Expected 1 provider cost, got %d", len(budgetCfg.ProviderCosts))
	}
}

func TestRejectAuthNone(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Server.Auth = "none"
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation error for auth = 'none'")
	}
}

func TestAcceptTokenAuth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Auth = "token"

	tmpDir := t.TempDir()
	cfg.Server.SocketPath = tmpDir + "/bridge.sock"
	cfg.Keystore.DBPath = tmpDir + "/keystore.db"

	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected no validation error for auth = 'token', got: %v", err)
	}
}

func TestDefaultAuthIsToken(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Auth != "token" {
		t.Errorf("Expected default auth to be 'token', got: %s", cfg.Server.Auth)
	}
}

func TestV6MicrokernelDefaultFalse(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Vault.V6Microkernel {
		t.Error("Vault.V6Microkernel should default to false")
	}

	if cfg.Vault.SocketPath != "/run/armorclaw/keystore.sock" {
		t.Errorf("Vault.SocketPath should default to '/run/armorclaw/keystore.sock', got %s", cfg.Vault.SocketPath)
	}
}

func TestV6MicrokernelTOMLParsing(t *testing.T) {
	input := `
[vault]
v6_microkernel = true
socket_path = "/custom/vault.sock"
`

	cfg := DefaultConfig()
	if _, err := toml.Decode(input, cfg); err != nil {
		t.Fatalf("failed to parse vault TOML: %v", err)
	}

	if !cfg.Vault.V6Microkernel {
		t.Error("expected v6_microkernel = true from TOML")
	}

	if cfg.Vault.SocketPath != "/custom/vault.sock" {
		t.Errorf("expected socket_path '/custom/vault.sock', got %s", cfg.Vault.SocketPath)
	}
}

func TestV6AuditModeDefaultFalse(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Vault.V6AuditMode {
		t.Error("Vault.V6AuditMode should default to false")
	}
}

func TestV6AuditModeTOMLParsing(t *testing.T) {
	input := `
[vault]
v6_microkernel = true
v6_audit_mode = true
socket_path = "/custom/vault.sock"
`

	cfg := DefaultConfig()
	if _, err := toml.Decode(input, cfg); err != nil {
		t.Fatalf("failed to parse vault TOML: %v", err)
	}

	if !cfg.Vault.V6Microkernel {
		t.Error("expected v6_microkernel = true from TOML")
	}

	if !cfg.Vault.V6AuditMode {
		t.Error("expected v6_audit_mode = true from TOML")
	}
}
