package config

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear environment variables to ensure defaults are used
	os.Unsetenv("LIGHTHOUSE_PORT")
	os.Unsetenv("LIGHTHOUSE_SECRET_KEY")
	os.Unsetenv("LIGHTHOUSE_DATABASE_PATH")

	config, err := Load()

	if err != nil {
		t.Fatalf("Load() should not return error, got: %v", err)
	}

	if config.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", config.Port)
	}

	if config.SecretKey != "change-me-in-production" {
		t.Errorf("Expected SecretKey 'change-me-in-production', got '%s'", config.SecretKey)
	}

	if config.DatabasePath != "./data/lighthouse.db" {
		t.Errorf("Expected DatabasePath './data/lighthouse.db', got '%s'", config.DatabasePath)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("LIGHTHOUSE_PORT", "9000")
	os.Setenv("LIGHTHOUSE_SECRET_KEY", "env-secret-key")
	os.Setenv("LIGHTHOUSE_DATABASE_PATH", "/custom/path.db")
	defer func() {
		os.Unsetenv("LIGHTHOUSE_PORT")
		os.Unsetenv("LIGHTHOUSE_SECRET_KEY")
		os.Unsetenv("LIGHTHOUSE_DATABASE_PATH")
	}()

	config, err := Load()

	if err != nil {
		t.Fatalf("Load() should not return error, got: %v", err)
	}

	if config.Port != 9000 {
		t.Errorf("Expected Port 9000 from env, got %d", config.Port)
	}

	if config.SecretKey != "env-secret-key" {
		t.Errorf("Expected SecretKey 'env-secret-key' from env, got '%s'", config.SecretKey)
	}

	if config.DatabasePath != "/custom/path.db" {
		t.Errorf("Expected DatabasePath '/custom/path.db' from env, got '%s'", config.DatabasePath)
	}
}
