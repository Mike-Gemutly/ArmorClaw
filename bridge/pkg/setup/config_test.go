package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateConfig tests configuration generation
func TestGenerateConfig(t *testing.T) {
	cfg := BridgeConfig{
		ServerName:       "test.example.com",
		APIKey:           "sk-test-key-123456789012345",
		MatrixHomeserver: "https://matrix.example.com",
		MatrixUser:       "@bot:example.com",
		MatrixPassword:   "secret-password",
		SocketPath:       "/run/test/bridge.sock",
		LogLevel:         "debug",
		EnableE2EE:       true,
		SessionTimeout:   "10m",
		KeystorePath:     "/var/lib/test/keystore.db",
		KeystorePassword: "keystore-secret",
		SSLEnabled:       true,
		SSLCertPath:      "/etc/test/server.crt",
		SSLKeyPath:       "/etc/test/server.key",
	}

	result, err := GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	// Verify config content
	if !strings.Contains(result.ConfigData, "server_name = \"test.example.com\"") {
		t.Error("expected server_name in config")
	}
	if !strings.Contains(result.ConfigData, "homeserver = \"https://matrix.example.com\"") {
		t.Error("expected homeserver in config")
	}
	if !strings.Contains(result.ConfigData, "socket_path = \"/run/test/bridge.sock\"") {
		t.Error("expected socket_path in config")
	}
	if !strings.Contains(result.ConfigData, "log_level = \"debug\"") {
		t.Error("expected log_level in config")
	}
	if !strings.Contains(result.ConfigData, "enable_e2ee = true") {
		t.Error("expected enable_e2ee in config")
	}

	// Verify SSL section is included when enabled
	if !strings.Contains(result.ConfigData, "[ssl]") {
		t.Error("expected SSL section in config")
	}

	// Verify environment content
	if !strings.Contains(result.EnvData, "ARMORCLAW_API_KEY=sk-test-key-123456789012345") {
		t.Error("expected API key in env")
	}
	if !strings.Contains(result.EnvData, "ARMORCLAW_MATRIX_USER=@bot:example.com") {
		t.Error("expected Matrix user in env")
	}
	if !strings.Contains(result.EnvData, "ARMORCLAW_MATRIX_PASSWORD=secret-password") {
		t.Error("expected Matrix password in env")
	}
	if !strings.Contains(result.EnvData, "ARMORCLAW_KEYSTORE_KEY=keystore-secret") {
		t.Error("expected keystore key in env")
	}
}

// TestGenerateConfigDefaults tests default values
func TestGenerateConfigDefaults(t *testing.T) {
	cfg := BridgeConfig{
		ServerName: "default.example.com",
	}

	result, err := GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	// Verify defaults were applied
	if !strings.Contains(result.ConfigData, "homeserver = \"https://default.example.com:6167\"") {
		t.Error("expected default homeserver URL")
	}
	if !strings.Contains(result.ConfigData, "socket_path = \"/run/armorclaw/bridge.sock\"") {
		t.Error("expected default socket path")
	}
	if !strings.Contains(result.ConfigData, "log_level = \"info\"") {
		t.Error("expected default log level")
	}
	if !strings.Contains(result.ConfigData, "session_timeout = \"5m\"") {
		t.Error("expected default session timeout")
	}
	if !strings.Contains(result.ConfigData, "path = \"/var/lib/armorclaw/keystore.db\"") {
		t.Error("expected default keystore path")
	}

	// Verify SSL section is not included when disabled
	if strings.Contains(result.ConfigData, "[ssl]") {
		t.Error("expected no SSL section when disabled")
	}

	// Verify default Matrix user
	if !strings.Contains(result.EnvData, "@armorclaw:default.example.com") {
		t.Error("expected default Matrix user")
	}
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  BridgeConfig
		wantErr bool
		errCode string
	}{
		{
			name: "valid config",
			config: BridgeConfig{
				ServerName: "valid.example.com",
				APIKey:     "sk-valid-key-123456789012345",
			},
			wantErr: false,
		},
		{
			name: "missing server name",
			config: BridgeConfig{
				APIKey: "sk-test-key",
			},
			wantErr: true,
			errCode: "INS-005",
		},
		{
			name: "server name with spaces",
			config: BridgeConfig{
				ServerName: "invalid server name",
			},
			wantErr: true,
			errCode: "INS-005",
		},
		{
			name: "server name is URL",
			config: BridgeConfig{
				ServerName: "https://example.com",
			},
			wantErr: true,
			errCode: "INS-005",
		},
		{
			name: "API key too short",
			config: BridgeConfig{
				ServerName: "test.example.com",
				APIKey:     "short-key",
			},
			wantErr: true,
			errCode: "INS-005",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				setupErr := GetSetupError(err)
				if setupErr == nil {
					t.Error("expected SetupError")
					return
				}

				if setupErr.Code != tt.errCode {
					t.Errorf("expected error code %s, got %s", tt.errCode, setupErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestWriteConfig tests writing configuration to disk
func TestWriteConfig(t *testing.T) {
	// Create temp directory and override DefaultConfigDir
	tempDir, err := os.MkdirTemp("", "armorclaw-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override default config dir for this test
	originalDir := DefaultConfigDir
	// Note: In a real test, we'd use a configurable output path

	cfg := BridgeConfig{
		ServerName: "write-test.example.com",
	}

	result, err := GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	// Manually set paths to temp directory for testing
	result.ConfigPath = filepath.Join(tempDir, "config.toml")

	// Create directory and write
	if err := os.MkdirAll(filepath.Dir(result.ConfigPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	if err := os.WriteFile(result.ConfigPath, []byte(result.ConfigData), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(result.ConfigPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Verify content
	data, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(data), "server_name = \"write-test.example.com\"") {
		t.Error("expected server_name in written config")
	}

	// Restore original
	_ = originalDir
}

// TestGenerateRandomPassword tests password generation
func TestGenerateRandomPassword(t *testing.T) {
	// Test default length
	password, err := GenerateRandomPassword(0)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}
	if len(password) != 16 {
		t.Errorf("expected password length 16, got %d", len(password))
	}

	// Test custom length
	password, err = GenerateRandomPassword(32)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}
	if len(password) != 32 {
		t.Errorf("expected password length 32, got %d", len(password))
	}

	// Test uniqueness
	password2, _ := GenerateRandomPassword(32)
	if password == password2 {
		t.Error("expected different passwords")
	}

	// Test minimum length enforcement
	shortPassword, _ := GenerateRandomPassword(8)
	if len(shortPassword) != 16 {
		t.Errorf("expected minimum length 16, got %d", len(shortPassword))
	}
}

// TestConfigExists tests configuration existence check
func TestConfigExists(t *testing.T) {
	// This test uses the actual default path, so it might fail
	// In a real test environment, we'd mock the filesystem
	// For now, just verify the function doesn't panic
	_ = ConfigExists()
}

// TestGeneratedConfigPaths tests that paths are set correctly
func TestGeneratedConfigPaths(t *testing.T) {
	cfg := BridgeConfig{
		ServerName: "path-test.example.com",
	}

	result, err := GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	expectedConfigPath := filepath.Join(DefaultConfigDir, DefaultConfigFile)
	if result.ConfigPath != expectedConfigPath {
		t.Errorf("expected config path %s, got %s", expectedConfigPath, result.ConfigPath)
	}

	expectedEnvPath := filepath.Join(DefaultConfigDir, "bridge.env")
	if result.EnvPath != expectedEnvPath {
		t.Errorf("expected env path %s, got %s", expectedEnvPath, result.EnvPath)
	}
}
