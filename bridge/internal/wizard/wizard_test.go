package wizard

import (
	"os"
	"strings"
	"testing"

	"github.com/armorclaw/bridge/pkg/setup"
)

func TestNonInteractiveWithAPIKey(t *testing.T) {
	// Set API key
	os.Setenv("ARMORCLAW_API_KEY", "sk-test-key-1234567890")
	defer os.Unsetenv("ARMORCLAW_API_KEY")

	result := tryNonInteractive()
	if result == nil {
		t.Fatal("expected non-interactive result with API key set")
	}
	if result.Secrets.APIKey != "sk-test-key-1234567890" {
		t.Errorf("expected API key to be set, got %s", result.Secrets.APIKey)
	}
	if result.Config.Profile != ProfileQuick {
		t.Errorf("expected profile to be 'quick', got %s", result.Config.Profile)
	}
}

func TestNonInteractiveWithoutAPIKey(t *testing.T) {
	// Ensure API key is not set
	os.Unsetenv("ARMORCLAW_API_KEY")

	result := tryNonInteractive()
	if result != nil {
		t.Fatal("expected nil result without API key")
	}
}

func TestNonInteractiveWithProfile(t *testing.T) {
	os.Setenv("ARMORCLAW_API_KEY", "sk-test-key-1234567890")
	os.Setenv("ARMORCLAW_PROFILE", "enterprise")
	defer func() {
		os.Unsetenv("ARMORCLAW_API_KEY")
		os.Unsetenv("ARMORCLAW_PROFILE")
	}()

	result := tryNonInteractive()
	if result == nil {
		t.Fatal("expected non-interactive result")
	}
	if result.Config.Profile != "enterprise" {
		t.Errorf("expected profile to be 'enterprise', got %s", result.Config.Profile)
	}
}

func TestNonInteractiveWithServerName(t *testing.T) {
	os.Setenv("ARMORCLAW_API_KEY", "sk-test-key-1234567890")
	os.Setenv("ARMORCLAW_SERVER_NAME", "test.example.com")
	defer func() {
		os.Unsetenv("ARMORCLAW_API_KEY")
		os.Unsetenv("ARMORCLAW_SERVER_NAME")
	}()

	result := tryNonInteractive()
	if result == nil {
		t.Fatal("expected non-interactive result")
	}
	if result.Config.ServerName != "test.example.com" {
		t.Errorf("expected server name to be 'test.example.com', got %s", result.Config.ServerName)
	}
}

func TestNonInteractiveWithAdminPassword(t *testing.T) {
	os.Setenv("ARMORCLAW_API_KEY", "sk-test-key-1234567890")
	os.Setenv("ARMORCLAW_ADMIN_PASSWORD", "MyTestPassword123!")
	defer func() {
		os.Unsetenv("ARMORCLAW_API_KEY")
		os.Unsetenv("ARMORCLAW_ADMIN_PASSWORD")
	}()

	result := tryNonInteractive()
	if result == nil {
		t.Fatal("expected non-interactive result")
	}
	if result.Secrets.AdminPassword != "MyTestPassword123!" {
		t.Errorf("expected admin password to be 'MyTestPassword123!', got %s", result.Secrets.AdminPassword)
	}
}

func TestNonInteractiveProviderDetection(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantProv  string
	}{
		{
			name:     "default to openai",
			baseURL:  "",
			wantProv: "openai",
		},
		{
			name:     "anthropic detection",
			baseURL:  "https://api.anthropic.com/v1",
			wantProv: "anthropic",
		},
		{
			name:     "z.ai detection",
			baseURL:  "https://api.z.ai/api/coding/paas/v4",
			wantProv: "openai", // Z.ai uses OpenAI-compatible API
		},
		{
			name:     "custom openai-compatible",
			baseURL:  "https://custom-provider.com/v1",
			wantProv: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ARMORCLAW_API_KEY", "sk-test-key-1234567890")
			if tt.baseURL != "" {
				os.Setenv("ARMORCLAW_API_BASE_URL", tt.baseURL)
			} else {
				os.Unsetenv("ARMORCLAW_API_BASE_URL")
			}
			defer func() {
				os.Unsetenv("ARMORCLAW_API_KEY")
				os.Unsetenv("ARMORCLAW_API_BASE_URL")
			}()

			result := tryNonInteractive()
			if result == nil {
				t.Fatal("expected non-interactive result")
			}
			if result.Config.APIProvider != tt.wantProv {
				t.Errorf("expected provider to be '%s', got %s", tt.wantProv, result.Config.APIProvider)
			}
		})
	}
}

func TestDetectServerName(t *testing.T) {
	name := detectServerName()
	if name == "" {
		t.Error("expected non-empty server name")
	}
	// Should not be "localhost" in most cases (but could be in some environments)
	t.Logf("Detected server name: %s", name)
}

func TestSecretEnvVars(t *testing.T) {
	secrets := &WizardSecrets{
		APIKey:         "sk-test-key",
		AdminPassword:  "admin-pass",
		BridgePassword: "bridge-pass",
	}

	vars := SecretEnvVars(secrets)

	if len(vars) != 3 {
		t.Errorf("expected 3 env vars, got %d", len(vars))
	}

	expected := []string{
		"ARMORCLAW_WIZARD_API_KEY=sk-test-key",
		"ARMORCLAW_WIZARD_ADMIN_PASSWORD=admin-pass",
		"ARMORCLAW_WIZARD_BRIDGE_PASSWORD=bridge-pass",
	}

	for i, exp := range expected {
		if vars[i] != exp {
			t.Errorf("expected %s, got %s", exp, vars[i])
		}
	}
}

func TestSecretEnvVarsPartial(t *testing.T) {
	secrets := &WizardSecrets{
		APIKey: "sk-test-key",
		// AdminPassword and BridgePassword are empty
	}

	vars := SecretEnvVars(secrets)

	if len(vars) != 1 {
		t.Errorf("expected 1 env var, got %d", len(vars))
	}

	if vars[0] != "ARMORCLAW_WIZARD_API_KEY=sk-test-key" {
		t.Errorf("expected API key env var, got %s", vars[0])
	}
}

func TestWriteConfigJSON(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "wizard-config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &WizardConfig{
		WizardVersion: "test-version",
		Profile:       ProfileQuick,
		APIProvider:   "openai",
		APIBaseURL:    "https://api.openai.com/v1",
		ServerName:    "test.example.com",
	}

	err = WriteConfigJSON(tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, `"profile": "quick"`) {
		t.Errorf("expected profile in config, got: %s", content)
	}
	if !strings.Contains(content, `"server_name": "test.example.com"`) {
		t.Errorf("expected server_name in config, got: %s", content)
	}
}

func TestWriteConfigJSONPermissionDenied(t *testing.T) {
	// Try to write to a directory that doesn't exist
	cfg := &WizardConfig{
		WizardVersion: "test-version",
		Profile:       ProfileQuick,
	}

	err := WriteConfigJSON("/nonexistent/directory/config.json", cfg)
	if err == nil {
		t.Error("expected error when writing to nonexistent directory")
	}

	// Check if it's a SetupError
	setupErr := setup.GetSetupError(err)
	if setupErr == nil {
		t.Error("expected SetupError type")
	} else if setupErr.Code != "INS-003" {
		t.Errorf("expected INS-003 error code, got %s", setupErr.Code)
	}
}

func TestCheckTerminal(t *testing.T) {
	result := checkTerminal()

	// In a test environment, we can't guarantee TTY
	// Just verify the function runs without panic
	t.Logf("Terminal check: IsTTY=%v, Width=%d, Height=%d, StdinOpen=%v, SupportsColor=%v",
		result.IsTTY, result.Width, result.Height, result.StdinOpen, result.SupportsColor)
}

func TestWizardVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if !strings.Contains(Version, ".") {
		t.Error("Version should contain dots (semver format)")
	}
}

func TestProfileConstants(t *testing.T) {
	if ProfileQuick != "quick" {
		t.Errorf("expected ProfileQuick to be 'quick', got %s", ProfileQuick)
	}
	if ProfileEnterprise != "enterprise" {
		t.Errorf("expected ProfileEnterprise to be 'enterprise', got %s", ProfileEnterprise)
	}
}

func TestGeneratePassword(t *testing.T) {
	password, err := generatePassword(16)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	if len(password) != 16 {
		t.Errorf("expected password length 16, got %d", len(password))
	}

	// Generate multiple passwords and verify uniqueness
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pwd, err := generatePassword(16)
		if err != nil {
			t.Fatal(err)
		}
		if passwords[pwd] {
			t.Error("generated duplicate password")
		}
		passwords[pwd] = true
	}
}

func TestGeneratePasswordDifferentLengths(t *testing.T) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		password, err := generatePassword(length)
		if err != nil {
			t.Errorf("failed to generate password of length %d: %v", length, err)
		}
		if len(password) != length {
			t.Errorf("expected password length %d, got %d", length, len(password))
		}
	}
}
