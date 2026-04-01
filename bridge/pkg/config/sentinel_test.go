package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerConfigSentinelFields(t *testing.T) {
	t.Run("native mode defaults", func(t *testing.T) {
		// Clear env vars to test default behavior
		os.Unsetenv("ARMORCLAW_SERVER_MODE")
		os.Unsetenv("ARMORCLAW_RPC_TRANSPORT")
		os.Unsetenv("ARMORCLAW_LISTEN_ADDR")
		os.Unsetenv("ARMORCLAW_PUBLIC_BASE_URL")
		os.Unsetenv("ARMORCLAW_ADMIN_TOKEN")

		cfg := DefaultConfig()

		assert.Equal(t, "native", cfg.Server.Mode, "Mode should default to native")
		assert.Equal(t, "unix", cfg.Server.RPCTransport, "RPCTransport should default to unix")
		assert.Equal(t, filepath.Join(os.TempDir(), "armorclaw", "bridge.sock"), cfg.Server.SocketPath, "SocketPath should have default")
		assert.Equal(t, "", cfg.Server.ListenAddr, "ListenAddr should be empty by default")
		assert.Equal(t, "", cfg.Server.PublicBaseURL, "PublicBaseURL should be empty by default")
		assert.Equal(t, "", cfg.Server.AdminToken, "AdminToken should be empty by default")
	})

	t.Run("env var overrides", func(t *testing.T) {
		os.Setenv("ARMORCLAW_SERVER_MODE", "sentinel")
		os.Setenv("ARMORCLAW_RPC_TRANSPORT", "tcp")
		os.Setenv("ARMORCLAW_LISTEN_ADDR", "0.0.0.0:8080")
		os.Setenv("ARMORCLAW_PUBLIC_BASE_URL", "https://test.example.com")
		os.Setenv("ARMORCLAW_ADMIN_TOKEN", "test-token-12345")

		cfg := DefaultConfig()

		assert.Equal(t, "sentinel", cfg.Server.Mode, "Mode should be overridden by env")
		assert.Equal(t, "tcp", cfg.Server.RPCTransport, "RPCTransport should be overridden by env")
		assert.Equal(t, "0.0.0.0:8080", cfg.Server.ListenAddr, "ListenAddr should be overridden by env")
		assert.Equal(t, "https://test.example.com", cfg.Server.PublicBaseURL, "PublicBaseURL should be overridden by env")
		assert.Equal(t, "test-token-12345", cfg.Server.AdminToken, "AdminToken should be overridden by env")
	})
}

func TestValidateSentinelMode(t *testing.T) {
	t.Run("sentinel mode requires ListenAddr", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "sentinel"
		cfg.Server.ListenAddr = ""
		cfg.Server.PublicBaseURL = "https://example.com"
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.listen_addr is required")
	})

	t.Run("sentinel mode requires PublicBaseURL", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "sentinel"
		cfg.Server.ListenAddr = "0.0.0.0:8080"
		cfg.Server.PublicBaseURL = ""
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.public_base_url is required")
	})

	t.Run("sentinel mode with all required fields", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "sentinel"
		cfg.Server.ListenAddr = "0.0.0.0:8080"
		cfg.Server.PublicBaseURL = "https://example.com"
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("native mode requires SocketPath", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "native"
		cfg.Server.SocketPath = ""
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.socket_path is required")
	})

	t.Run("native mode with valid SocketPath", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "native"
		cfg.Server.SocketPath = filepath.Join(os.TempDir(), "armorclaw", "test.sock")
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid mode value", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = "invalid"
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.mode must be 'native' or 'sentinel'")
	})

	t.Run("mode defaults to native", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Server.Mode = ""
		cfg.Server.SocketPath = filepath.Join(os.TempDir(), "armorclaw", "test.sock")
		cfg.Keystore.DBPath = filepath.Join(os.TempDir(), "test-keystore.db")

		err := cfg.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "native", cfg.Server.Mode)
	})
}
