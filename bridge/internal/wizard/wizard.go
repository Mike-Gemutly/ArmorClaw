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
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// Version is the wizard version, matching the container setup version.
// Update this when releasing new versions - should match VERSION file in repo root.
const Version = "0.3.3"

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
	APIKey         string
	AdminPassword  string
	BridgePassword string
}

// WizardResult combines config and secrets from a completed wizard run.
type WizardResult struct {
	Config  WizardConfig
	Secrets WizardSecrets
}

// TerminalCheckResult contains information about terminal capabilities.
type TerminalCheckResult struct {
	IsTTY        bool
	Width        int
	Height       int
	StdinOpen    bool
	SupportsColor bool
}

// checkTerminal verifies terminal capabilities for TUI operation.
func checkTerminal() TerminalCheckResult {
	result := TerminalCheckResult{
		IsTTY:        term.IsTerminal(int(os.Stdout.Fd())),
		StdinOpen:    true,
		SupportsColor: true,
	}

	// Check if stdin is open
	stat, err := os.Stdin.Stat()
	if err != nil {
		result.StdinOpen = false
	} else {
		// Check if stdin is a pipe or redirect (not interactive)
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			result.StdinOpen = false
		}
	}

	// Get terminal size if available
	if result.IsTTY {
		result.Width, result.Height, _ = term.GetSize(int(os.Stdout.Fd()))
	}

	// Check for color support via TERM env
	termEnv := os.Getenv("TERM")
	if termEnv == "dumb" || termEnv == "" {
		result.SupportsColor = false
	}

	return result
}

// printTerminalError shows a helpful message when terminal is not suitable for TUI.
func printTerminalError(check TerminalCheckResult) {
	fmt.Println()
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Interactive Terminal Required")
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if !check.IsTTY {
		fmt.Println("  The TUI wizard requires an interactive terminal.")
		fmt.Println()
		fmt.Println("  Solutions:")
		fmt.Println("    1. Run with -it flags: docker run -it ...")
		fmt.Println("    2. Use environment variables (non-interactive):")
		fmt.Println("       -e ARMORCLAW_API_KEY=sk-your-key")
		fmt.Println("       -e ARMORCLAW_PROFILE=quick")
	}

	if !check.StdinOpen && check.IsTTY {
		fmt.Println("  Stdin is not connected (piped input detected).")
		fmt.Println()
		fmt.Println("  Solutions:")
		fmt.Println("    1. Run without piping input")
		fmt.Println("    2. Use environment variables for non-interactive setup")
	}

	if check.Width > 0 && check.Width < 60 {
		fmt.Printf("  Terminal is too narrow (%d columns, need 60+).\n", check.Width)
		fmt.Println("  Please resize your terminal window.")
	}

	fmt.Println()
	fmt.Println("  For non-interactive setup, set these environment variables:")
	fmt.Println("    ARMORCLAW_PROFILE=quick|enterprise")
	fmt.Println("    ARMORCLAW_API_KEY=your-api-key")
	fmt.Println("    ARMORCLAW_SERVER_NAME=your-server (optional, auto-detected)")
	fmt.Println()
}

// Run executes the interactive setup wizard and returns the collected
// configuration and secrets. Returns huh.ErrUserAborted if the user cancels.
func Run(accessible bool) (*WizardResult, error) {
	// Check for ACCESSIBLE env var
	if os.Getenv("ARMORCLAW_ACCESSIBLE") == "true" {
		accessible = true
	}

	// ALWAYS check env vars first - prefer non-interactive if configured
	// This allows users to provide env vars even with proper terminal
	if result := tryNonInteractive(); result != nil {
		return result, nil
	}

	// No env vars - check terminal capabilities before launching TUI
	termCheck := checkTerminal()
	if !termCheck.IsTTY {
		// No env vars AND no TTY - show error
		printTerminalError(termCheck)
		return nil, fmt.Errorf("interactive terminal required (run with -it or provide ARMORCLAW_API_KEY)")
	}

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

// tryNonInteractive attempts to build a WizardResult from environment variables.
// Returns nil if required env vars are missing.
func tryNonInteractive() *WizardResult {
	apiKey := strings.TrimSpace(os.Getenv("ARMORCLAW_API_KEY"))
	if apiKey == "" {
		return nil
	}

	profile := os.Getenv("ARMORCLAW_PROFILE")
	if profile == "" {
		profile = ProfileQuick
	}

	// Determine provider from API base URL
	baseURL := os.Getenv("ARMORCLAW_API_BASE_URL")
	provider := "openai"
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// Detect provider from URL patterns
	switch {
	case strings.Contains(baseURL, "anthropic"):
		provider = "anthropic"
	case strings.Contains(baseURL, "z.ai"):
		provider = "openai" // Z.ai uses OpenAI-compatible API
	}

	// Get server name from env or auto-detect
	serverName := os.Getenv("ARMORCLAW_SERVER_NAME")
	if serverName == "" {
		serverName = detectServerName()
	}

	// Get or generate admin password
	adminPassword := os.Getenv("ARMORCLAW_ADMIN_PASSWORD")
	if adminPassword == "" {
		generated, err := generatePassword(16)
		if err != nil {
			// Log error but don't fail - use a fallback
			fmt.Fprintf(os.Stderr, "Warning: password generation failed: %v\n", err)
			generated = "ChangeMe123!" // Fallback password
		}
		adminPassword = generated
	}

	// Generate bridge password
	bridgePassword, err := generatePassword(16)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: bridge password generation failed: %v\n", err)
		bridgePassword = "BridgePass123!"
	}

	fmt.Println()
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Non-Interactive Mode (Environment Variables)")
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Profile:     %s\n", profile)
	fmt.Printf("  Server Name: %s\n", serverName)
	fmt.Printf("  Provider:    %s\n", provider)
	fmt.Printf("  Base URL:    %s\n", baseURL)
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return &WizardResult{
		Config: WizardConfig{
			WizardVersion: Version,
			Profile:       profile,
			APIProvider:   provider,
			APIBaseURL:    baseURL,
			ServerName:    serverName,
			LogLevel:      "info",
			SocketPath:    "/run/armorclaw/bridge.sock",
			SecurityTier:  "enhanced",
			AdminUser:     "admin",
		},
		Secrets: WizardSecrets{
			APIKey:         apiKey,
			AdminPassword:  adminPassword,
			BridgePassword: bridgePassword,
		},
	}
}

// detectServerName attempts to auto-detect the server's public hostname or IP.
// Falls back to localhost if detection fails.
func detectServerName() string {
	// Try to get public IP (most reliable for VPS deployments)
	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return localAddr.IP.String()
		}
	}

	// Try hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}

	// Fallback to localhost
	return "localhost"
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
