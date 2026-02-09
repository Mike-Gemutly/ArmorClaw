// Package config provides configuration loading and management for ArmorClaw bridge.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load loads configuration from a file path
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// If path is empty, search for default config files
	if path == "" {
		for _, p := range ConfigPaths() {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	// If no config file found, warn and return defaults
	if path == "" {
		log.Printf("Warning: No configuration file found in default locations")
		log.Printf("Default locations checked:")
		for _, p := range ConfigPaths() {
			log.Printf("  - %s", p)
		}
		log.Printf("Using default configuration")
		log.Printf("Create a config with: armorclaw-bridge init")
		return cfg, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse TOML using BurntSushi/toml library
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// LoadOrDie loads configuration or exits on error
func LoadOrDie(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(cfg *Config) error {
	// Server overrides
	if v := os.Getenv("ARMORCLAW_SOCKET"); v != "" {
		cfg.Server.SocketPath = v
	}
	if v := os.Getenv("ARMORCLAW_PID_FILE"); v != "" {
		cfg.Server.PidFile = v
	}
	if v := os.Getenv("ARMORCLAW_DAEMONIZE"); v != "" {
		cfg.Server.Daemonize = v == "true" || v == "1"
	}

	// Keystore overrides
	if v := os.Getenv("ARMORCLAW_KEYSTORE_DB"); v != "" {
		cfg.Keystore.DBPath = v
	}
	if v := os.Getenv("ARMORCLAW_MASTER_KEY"); v != "" {
		cfg.Keystore.MasterKey = v
	}

	// Matrix overrides
	if v := os.Getenv("ARMORCLAW_MATRIX_ENABLED"); v != "" {
		cfg.Matrix.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("ARMORCLAW_MATRIX_HOMESERVER"); v != "" {
		cfg.Matrix.HomeserverURL = v
	}
	if v := os.Getenv("ARMORCLAW_MATRIX_USERNAME"); v != "" {
		cfg.Matrix.Username = v
	}
	if v := os.Getenv("ARMORCLAW_MATRIX_PASSWORD"); v != "" {
		cfg.Matrix.Password = v
	}
	if v := os.Getenv("ARMORCLAW_MATRIX_DEVICE_ID"); v != "" {
		cfg.Matrix.DeviceID = v
	}
	if v := os.Getenv("ARMORCLAW_MATRIX_SYNC_INTERVAL"); v != "" {
		var interval int
		if _, err := fmt.Sscanf(v, "%d", &interval); err == nil {
			cfg.Matrix.SyncInterval = interval
		}
	}

	// Logging overrides
	if v := os.Getenv("ARMORCLAW_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("ARMORCLAW_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("ARMORCLAW_LOG_OUTPUT"); v != "" {
		cfg.Logging.Output = v
	}
	if v := os.Getenv("ARMORCLAW_LOG_FILE"); v != "" {
		cfg.Logging.File = v
	}

	return nil
}

// Save saves the configuration to a file
func Save(cfg *Config, path string) error {
	// Validate before saving
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Normalize paths for TOML compatibility (forward slashes, no backslashes)
	// This fixes Windows path parsing issues where \U is interpreted as Unicode escape
	cfgCopy := *cfg // Make a shallow copy
	cfgCopy.Keystore.DBPath = filepath.ToSlash(cfg.Keystore.DBPath)
	cfgCopy.Server.SocketPath = filepath.ToSlash(cfg.Server.SocketPath)
	if cfgCopy.Server.PidFile != "" {
		cfgCopy.Server.PidFile = filepath.ToSlash(cfgCopy.Server.PidFile)
	}
	if cfg.Matrix.DeviceID != "" {
		cfgCopy.Matrix.DeviceID = filepath.ToSlash(cfgCopy.Matrix.DeviceID)
	}

	// Marshal to TOML using BurntSushi/toml library
	data, err := toml.Marshal(&cfgCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GenerateExampleConfig generates an example configuration file
func GenerateExampleConfig(path string) error {
	cfg := DefaultConfig()

	// Add example values
	cfg.Matrix.Enabled = true
	cfg.Matrix.HomeserverURL = "https://matrix.example.com"
	cfg.Matrix.Username = "bridge-bot"
	cfg.Matrix.Password = "change-me"
	cfg.Logging.Level = "info"

	return Save(cfg, path)
}
