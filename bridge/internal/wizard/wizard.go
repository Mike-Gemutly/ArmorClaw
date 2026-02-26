// Package wizard provides an interactive TUI setup wizard for ArmorClaw
// using charmbracelet/huh forms. It collects configuration through multi-page
// forms with validation, then outputs results for the container setup script.
//
// Security: Secrets (API keys, passwords) are kept in memory only and passed
// to the container setup script via environment variables — never written to
// the JSON output file.
package wizard

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

// Version is the wizard version, matching the container setup version.
const Version = "0.3.1"

// Profile represents a deployment profile.
const (
	ProfileQuick      = "quick"
	ProfileEnterprise = "enterprise"
)

// WizardConfig holds non-secret configuration values that are safe to write
// to a JSON file for the container setup script to consume.
type WizardConfig struct {
	Profile       string `json:"profile"`
	APIProvider   string `json:"api_provider"`
	APIBaseURL    string `json:"api_base_url"`
	AdminUser     string `json:"admin_user"`
	ServerName    string `json:"server_name,omitempty"`
	MatrixServer  string `json:"matrix_server,omitempty"`
	MatrixURL     string `json:"matrix_url,omitempty"`
	LogLevel      string `json:"log_level,omitempty"`
	SocketPath    string `json:"socket_path,omitempty"`
	SecurityTier  string `json:"security_tier,omitempty"`
	HIPAAEnabled  bool   `json:"hipaa_enabled,omitempty"`
	Quarantine    bool   `json:"quarantine_enabled,omitempty"`
	AuditRetDays  int    `json:"audit_retention_days,omitempty"`
	WizardVersion string `json:"wizard_version"`
}

// WizardSecrets holds sensitive values that must never be written to disk.
// These are passed to the container setup script via environment variables
// and then injected directly into the SQLCipher-encrypted keystore.
type WizardSecrets struct {
	APIKey        string
	AdminPassword string
	BridgePassword string
}

// WizardResult combines config and secrets from a completed wizard run.
type WizardResult struct {
	Config  WizardConfig
	Secrets WizardSecrets
}

// Run executes the interactive setup wizard and returns the collected
// configuration and secrets. Returns huh.ErrUserAborted if the user cancels.
func Run(accessible bool) (*WizardResult, error) {
	printBanner()

	result := &WizardResult{
		Config: WizardConfig{
			WizardVersion: Version,
			LogLevel:      "info",
			SocketPath:    "/run/armorclaw/bridge.sock",
			SecurityTier:  "enhanced",
			AdminUser:     "admin",
		},
	}

	// Page 1: Profile selection
	profile, err := runProfileForm(accessible)
	if err != nil {
		return nil, err
	}
	result.Config.Profile = profile

	// Route to profile-specific forms
	switch profile {
	case ProfileQuick:
		if err := runQuickStartForms(result, accessible); err != nil {
			return nil, err
		}
	case ProfileEnterprise:
		// Enterprise wizard is a future addition; for now fall through
		// to quick start with a note.
		fmt.Println("  Enterprise profile will be available in a future update.")
		fmt.Println("  Running Quick Start for now.")
		fmt.Println()
		result.Config.Profile = ProfileQuick
		if err := runQuickStartForms(result, accessible); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// WriteConfigJSON writes the non-secret configuration to a JSON file.
// Secrets are intentionally excluded — they are passed via environment
// variables by the caller.
func WriteConfigJSON(path string, cfg *WizardConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wizard config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write wizard config to %s: %w", path, err)
	}

	return nil
}

// SecretEnvVars returns the environment variable assignments for secrets.
// These are used to pass secrets to the container setup script without
// writing them to disk.
func SecretEnvVars(s *WizardSecrets) []string {
	vars := make([]string, 0, 3)
	if s.APIKey != "" {
		vars = append(vars, "ARMORCLAW_WIZARD_API_KEY="+s.APIKey)
	}
	if s.AdminPassword != "" {
		vars = append(vars, "ARMORCLAW_WIZARD_ADMIN_PASSWORD="+s.AdminPassword)
	}
	if s.BridgePassword != "" {
		vars = append(vars, "ARMORCLAW_WIZARD_BRIDGE_PASSWORD="+s.BridgePassword)
	}
	return vars
}

func printBanner() {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════════════╗")
	fmt.Println("  ║        ArmorClaw Container Setup                    ║")
	fmt.Printf("  ║        Version %-40s║\n", Version)
	fmt.Println("  ╚══════════════════════════════════════════════════════╝")
	fmt.Println()
}

// formOpts returns common form options (accessibility, theme).
func formOpts(f *huh.Form, accessible bool) *huh.Form {
	return f.
		WithTheme(ArmorClawTheme()).
		WithAccessible(accessible)
}
