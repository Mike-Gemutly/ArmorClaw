// Package config provides configuration management for ArmorClaw bridge.
// Supports TOML configuration files with environment variable overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/armorclaw/bridge/pkg/budget"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/rpc"
)

// Helper function to validate directory exists or can be created
func validateDirectoryWritable(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(dir, 0750); err != nil {
				return fmt.Errorf("cannot create directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("cannot access directory: %w", err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	// Check if we can write to it
	testFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

var (
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrMissingValue  = errors.New("missing required configuration value")
)

// BudgetConfig holds budget-related configuration
type BudgetConfig struct {
	// DailyLimitUSD is the maximum daily spend in USD (0 = no limit)
	DailyLimitUSD float64 `toml:"daily_limit_usd" env:"ARMORCLAW_DAILY_LIMIT"`

	// MonthlyLimitUSD is the maximum monthly spend in USD (0 = no limit)
	MonthlyLimitUSD float64 `toml:"monthly_limit_usd" env:"ARMORCLAW_MONTHLY_LIMIT"`

	// AlertThreshold is the percentage (0-100) at which to alert (e.g., 80 = warn at 80% of limit)
	AlertThreshold float64 `toml:"alert_threshold" env:"ARMORCLAW_ALERT_THRESHOLD"`

	// HardStop prevents new sessions when limits are exceeded
	HardStop bool `toml:"hard_stop" env:"ARMORCLAW_HARD_STOP"`

	// ProviderCosts allows custom token costs per model
	ProviderCosts map[string]float64 `toml:"provider_costs"`
}

// Config holds all bridge configuration
type Config struct {
	// Server configuration
	Server ServerConfig `toml:"server"`

	// Keystore configuration
	Keystore KeystoreConfig `toml:"keystore"`

	// Matrix configuration
	Matrix MatrixConfig `toml:"matrix"`

	// Budget configuration
	Budget BudgetConfig `toml:"budget"`

	// Logging configuration
	Logging LoggingConfig `toml:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// SocketPath is the path to the Unix domain socket
	SocketPath string `toml:"socket_path" env:"ARMORCLAW_SOCKET"`

	// PidFile is the path to the PID file for daemon mode
	PidFile string `toml:"pid_file" env:"ARMORCLAW_PID_FILE"`

	// Daemonize runs the server as a background daemon
	Daemonize bool `toml:"daemonize" env:"ARMORCLAW_DAEMONIZE"`
}

// KeystoreConfig holds keystore-specific configuration
type KeystoreConfig struct {
	// DBPath is the path to the encrypted keystore database
	DBPath string `toml:"db_path" env:"ARMORCLAW_KEYSTORE_DB"`

	// MasterKey is an optional master key (if not provided, derived from hardware)
	MasterKey string `toml:"master_key" env:"ARMORCLAW_MASTER_KEY"`

	// Provider configuration
	Providers []ProviderConfig `toml:"providers"`
}

// ProviderConfig holds credentials for a specific provider
type ProviderConfig struct {
	ID          string            `toml:"id"`
	Provider    keystore.Provider `toml:"provider"`
	Token       string            `toml:"token" env:"ARMORCLAW_PROVIDER_TOKEN"`
	DisplayName string            `toml:"display_name"`
	ExpiresAt   int64             `toml:"expires_at"`
	Tags        []string          `toml:"tags"`
}

// MatrixConfig holds Matrix-specific configuration
type MatrixConfig struct {
	// Enabled enables Matrix communication
	Enabled bool `toml:"enabled" env:"ARMORCLAW_MATRIX_ENABLED"`

	// HomeserverURL is the Matrix homeserver URL
	HomeserverURL string `toml:"homeserver_url" env:"ARMORCLAW_MATRIX_HOMESERVER"`

	// Username for auto-login
	Username string `toml:"username" env:"ARMORCLAW_MATRIX_USERNAME"`

	// Password for auto-login
	Password string `toml:"password" env:"ARMORCLAW_MATRIX_PASSWORD"`

	// DeviceID for the Matrix client
	DeviceID string `toml:"device_id" env:"ARMORCLAW_MATRIX_DEVICE_ID"`

	// SyncInterval is the interval between syncs in seconds
	SyncInterval int `toml:"sync_interval" env:"ARMORCLAW_MATRIX_SYNC_INTERVAL"`

	// AutoRooms are rooms to automatically join on login
	AutoRooms []string `toml:"auto_rooms"`

	// Retry configuration
	Retry RetryConfig `toml:"retry"`

	// Zero-trust configuration
	ZeroTrust ZeroTrustConfig `toml:"zero_trust"`
}

// RetryConfig holds retry configuration for Matrix operations
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `toml:"max_retries"`

	// RetryDelay is the delay between retries in seconds
	RetryDelay int `toml:"retry_delay"`

	// BackoffMultiplier multiplies the delay after each retry
	BackoffMultiplier float64 `toml:"backoff_multiplier"`
}

// ZeroTrustConfig holds zero-trust security settings
type ZeroTrustConfig struct {
	// TrustedSenders is a list of allowed Matrix user IDs
	// Supports wildcards: @user:domain.com, *@trusted.domain.com, *:domain.com
	TrustedSenders []string `toml:"trusted_senders" env:"ARMORCLAW_TRUSTED_SENDERS"`

	// TrustedRooms restricts message processing to specific rooms
	TrustedRooms []string `toml:"trusted_rooms" env:"ARMORCLAW_TRUSTED_ROOMS"`

	// RejectUntrusted controls behavior for untrusted messages
	// true: send rejection back to sender, false: drop silently with log
	RejectUntrusted bool `toml:"reject_untrusted" env:"ARMORCLAW_REJECT_UNTRUSTED"`
}

// LoggingConfig holds logging-specific configuration
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `toml:"level" env:"ARMORCLAW_LOG_LEVEL"`

	// Format is the log format (json, text)
	Format string `toml:"format" env:"ARMORCLAW_LOG_FORMAT"`

	// Output is the log output (stdout, stderr, or file path)
	Output string `toml:"output" env:"ARMORCLAW_LOG_OUTPUT"`

	// File is the log file path when output is "file"
	File string `toml:"file" env:"ARMORCLAW_LOG_FILE"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Server: ServerConfig{
			SocketPath: "/run/armorclaw/bridge.sock",
			PidFile:    "/run/armorclaw/bridge.pid",
			Daemonize:  false,
		},
		Keystore: KeystoreConfig{
			DBPath:    filepath.Join(homeDir, ".armorclaw", "keystore.db"),
			MasterKey: "",
			Providers: []ProviderConfig{},
		},
		Matrix: MatrixConfig{
			Enabled:      false,
			HomeserverURL: "",
			Username:     "",
			Password:     "",
			DeviceID:     "armorclaw-bridge",
			SyncInterval: 5,
			AutoRooms:    []string{},
			Retry: RetryConfig{
				MaxRetries:       3,
				RetryDelay:       5,
				BackoffMultiplier: 2.0,
			},
			ZeroTrust: ZeroTrustConfig{
				TrustedSenders:  []string{}, // Empty = allow all (backward compatible)
				TrustedRooms:    []string{}, // Empty = allow all rooms
				RejectUntrusted: false,      // Silent drop by default
			},
		},
		Budget: BudgetConfig{
			DailyLimitUSD:   5.00,  // $5/day default
			MonthlyLimitUSD: 100.00, // $100/month default
			AlertThreshold:  80.0,  // Warn at 80%
			HardStop:        true,   // Prevent overages by default
			ProviderCosts:   make(map[string]float64),
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
			File:   "",
		},
	}
}

// ConfigPaths returns the list of default configuration file paths to check
func ConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, ".armorclaw", "config.toml"),
		filepath.Join("/etc", "armorclaw", "config.toml"),
		"./config.toml",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.SocketPath == "" {
		return fmt.Errorf("%w: server.socket_path is required", ErrInvalidConfig)
	}

	// Validate socket directory exists or can be created
	socketDir := filepath.Dir(c.Server.SocketPath)
	if err := validateDirectoryWritable(socketDir); err != nil {
		return fmt.Errorf("%w: socket directory %s: %w", ErrInvalidConfig, socketDir, err)
	}

	// Validate keystore configuration
	if c.Keystore.DBPath == "" {
		return fmt.Errorf("%w: keystore.db_path is required", ErrInvalidConfig)
	}

	// Validate keystore directory exists or can be created
	keystoreDir := filepath.Dir(c.Keystore.DBPath)
	if err := validateDirectoryWritable(keystoreDir); err != nil {
		return fmt.Errorf("%w: keystore directory %s: %w", ErrInvalidConfig, keystoreDir, err)
	}

	// Validate Matrix configuration if enabled
	if c.Matrix.Enabled {
		if c.Matrix.HomeserverURL == "" {
			return fmt.Errorf("%w: matrix.homeserver_url is required when matrix is enabled", ErrInvalidConfig)
		}

		// Validate sync interval
		if c.Matrix.SyncInterval < 1 {
			return fmt.Errorf("%w: matrix.sync_interval must be at least 1 second", ErrInvalidConfig)
		}

		// Validate retry configuration
		if c.Matrix.Retry.MaxRetries < 0 {
			return fmt.Errorf("%w: matrix.retry.max_retries cannot be negative", ErrInvalidConfig)
		}

		if c.Matrix.Retry.RetryDelay < 0 {
			return fmt.Errorf("%w: matrix.retry.retry_delay cannot be negative", ErrInvalidConfig)
		}

		if c.Matrix.Retry.BackoffMultiplier < 1.0 {
			return fmt.Errorf("%w: matrix.retry.backoff_multiplier must be at least 1.0", ErrInvalidConfig)
		}
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("%w: logging.level must be one of: debug, info, warn, error", ErrInvalidConfig)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("%w: logging.format must be one of: json, text", ErrInvalidConfig)
	}

	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
	}
	if !validOutputs[c.Logging.Output] {
		return fmt.Errorf("%w: logging.output must be one of: stdout, stderr, file", ErrInvalidConfig)
	}

	if c.Logging.Output == "file" && c.Logging.File == "" {
		return fmt.Errorf("%w: logging.file is required when logging.output is 'file'", ErrInvalidConfig)
	}

	// Validate budget configuration
	if c.Budget.DailyLimitUSD < 0 {
		return fmt.Errorf("%w: budget.daily_limit_usd cannot be negative", ErrInvalidConfig)
	}

	if c.Budget.MonthlyLimitUSD < 0 {
		return fmt.Errorf("%w: budget.monthly_limit_usd cannot be negative", ErrInvalidConfig)
	}

	if c.Budget.AlertThreshold < 0 || c.Budget.AlertThreshold > 100 {
		return fmt.Errorf("%w: budget.alert_threshold must be between 0 and 100", ErrInvalidConfig)
	}

	return nil
}

// ToRPCConfig converts the Config to rpc.Config
func (c *Config) ToRPCConfig() rpc.Config {
	return rpc.Config{
		SocketPath: c.Server.SocketPath,
	}
}

// ToKeystoreConfig converts the Config to keystore.Config
func (c *Config) ToKeystoreConfig() keystore.Config {
	cfg := keystore.Config{
		DBPath: c.Keystore.DBPath,
	}

	// Parse master key if provided
	if c.Keystore.MasterKey != "" {
		// Master key should be hex-encoded
		cfg.MasterKey = []byte(c.Keystore.MasterKey)
	}

	return cfg
}

// ToMatrixConfig converts the Config to adapter.Config
func (c *Config) ToMatrixConfig() adapter.Config {
	return adapter.Config{
		HomeserverURL:  c.Matrix.HomeserverURL,
		DeviceID:       c.Matrix.DeviceID,
		TokenFile:      filepath.Join(filepath.Dir(c.Keystore.DBPath), "matrix_token.json"),
		TrustedSenders: c.Matrix.ZeroTrust.TrustedSenders,
		TrustedRooms:   c.Matrix.ZeroTrust.TrustedRooms,
		RejectUntrusted: c.Matrix.ZeroTrust.RejectUntrusted,
	}
}

// MatrixCredentials returns the Matrix username and password for auto-login
func (c *Config) MatrixCredentials() (username, password string) {
	return c.Matrix.Username, c.Matrix.Password
}

// IsMatrixEnabled returns true if Matrix communication is enabled
func (c *Config) IsMatrixEnabled() bool {
	return c.Matrix.Enabled
}

// GetSyncInterval returns the Matrix sync interval as a Duration
func (c *Config) GetSyncInterval() time.Duration {
	return time.Duration(c.Matrix.SyncInterval) * time.Second
}

// ToBudgetConfig converts the Config to budget.BudgetConfig
func (c *Config) ToBudgetConfig() budget.BudgetConfig {
	return budget.BudgetConfig{
		DailyLimitUSD:   c.Budget.DailyLimitUSD,
		MonthlyLimitUSD: c.Budget.MonthlyLimitUSD,
		AlertThreshold:  c.Budget.AlertThreshold,
		HardStop:        c.Budget.HardStop,
		ProviderCosts:   c.Budget.ProviderCosts,
	}
}
