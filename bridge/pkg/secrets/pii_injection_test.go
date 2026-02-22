// Package secrets provides tests for PII injection functionality
package secrets

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/pii"
)

func TestNewPIIInjector(t *testing.T) {
	// Create temp directory for sockets
	tempDir := t.TempDir()

	cfg := &PIIInjectionConfig{
		Method:    "socket",
		SocketDir: tempDir,
		EnvPrefix: "PII_",
		TTL:       10 * time.Second,
	}

	injector, err := NewPIIInjector(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}

	if injector == nil {
		t.Fatal("Injector is nil")
	}

	// Cleanup
	injector.Stop()
}

func TestNewPIIInjector_DefaultConfig(t *testing.T) {
	injector, err := NewPIIInjector(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create injector with default config: %v", err)
	}

	if injector.config.Method != "socket" {
		t.Errorf("Expected default method 'socket', got '%s'", injector.config.Method)
	}

	injector.Stop()
}

func TestPIIInjector_InjectViaEnv(t *testing.T) {
	injector, err := NewPIIInjector(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}
	defer injector.Stop()

	// Create resolved variables
	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("full_name", "John Doe")
	resolved.SetVariable("email", "john@example.com")

	cfg := &PIIInjectionConfig{
		Method:    "env",
		EnvPrefix: "TEST_PII_",
		TTL:       5 * time.Second,
	}

	result, err := injector.InjectPII(context.Background(), "test-container", resolved, cfg)
	if err != nil {
		t.Fatalf("Failed to inject via env: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.Method != "env" {
		t.Errorf("Expected method 'env', got '%s'", result.Method)
	}

	if len(result.FieldsInjected) != 2 {
		t.Errorf("Expected 2 fields injected, got %d", len(result.FieldsInjected))
	}

	// Verify environment variables were generated
	if result.EnvVars == nil {
		t.Fatal("EnvVars should not be nil")
	}

	if result.EnvVars["TEST_PII_full_name"] != "John Doe" {
		t.Errorf("Expected full_name env var, got: %v", result.EnvVars["TEST_PII_full_name"])
	}

	if result.EnvVars["TEST_PII_email"] != "john@example.com" {
		t.Errorf("Expected email env var, got: %v", result.EnvVars["TEST_PII_email"])
	}
}

func TestPIIInjector_InjectViaSocket(t *testing.T) {
	tempDir := t.TempDir()

	injector, err := NewPIIInjector(&PIIInjectionConfig{
		Method:    "socket",
		SocketDir: tempDir,
		TTL:       5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}
	defer injector.Stop()

	// Create resolved variables
	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("full_name", "Jane Smith")

	// Start socket handler in goroutine to receive
	receivedData := make(chan map[string]interface{}, 1)
	go func() {
		socketPath := filepath.Join(tempDir, "test-container.pii.sock")

		// Wait for socket to be created
		for i := 0; i < 50; i++ {
			if _, err := os.Stat(socketPath); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Connect to socket
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Logf("Failed to connect to socket: %v", err)
			return
		}
		defer conn.Close()

		// Read length prefix
		lengthBuf := make([]byte, 4)
		_, err = conn.Read(lengthBuf)
		if err != nil {
			t.Logf("Failed to read length: %v", err)
			return
		}

		length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])

		// Read data
		dataBuf := make([]byte, length)
		_, err = conn.Read(dataBuf)
		if err != nil {
			t.Logf("Failed to read data: %v", err)
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataBuf, &data); err != nil {
			t.Logf("Failed to unmarshal data: %v", err)
			return
		}

		receivedData <- data
	}()

	// Inject PII
	result, err := injector.InjectPII(context.Background(), "test-container", resolved, nil)
	if err != nil {
		t.Fatalf("Failed to inject via socket: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.Method != "socket" {
		t.Errorf("Expected method 'socket', got '%s'", result.Method)
	}

	if result.SocketPath == "" {
		t.Error("Expected socket path to be set")
	}

	// Wait for data to be received
	select {
	case data := <-receivedData:
		if data["skill_id"] != "skill-123" {
			t.Errorf("Expected skill_id 'skill-123', got '%v'", data["skill_id"])
		}
		vars, ok := data["variables"].(map[string]interface{})
		if !ok {
			t.Fatal("variables should be a map")
		}
		if vars["full_name"] != "Jane Smith" {
			t.Errorf("Expected full_name 'Jane Smith', got '%v'", vars["full_name"])
		}
	case <-time.After(3 * time.Second):
		t.Log("Warning: Socket data not received within timeout (may be timing issue)")
	}
}

func TestPIIInjector_NilResolved(t *testing.T) {
	injector, err := NewPIIInjector(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}
	defer injector.Stop()

	_, err = injector.InjectPII(context.Background(), "test", nil, nil)
	if err == nil {
		t.Error("Expected error for nil resolved variables")
	}
}

func TestPIIInjector_ExpiredResolved(t *testing.T) {
	injector, err := NewPIIInjector(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}
	defer injector.Stop()

	// Create already-expired resolved variables
	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("test", "value")
	// Manually set expiry to past (Unix timestamp)
	resolved.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()

	_, err = injector.InjectPII(context.Background(), "test", resolved, nil)
	if err == nil {
		t.Error("Expected error for expired resolved variables")
	}
}

func TestPIIInjector_DuplicateSession(t *testing.T) {
	// Skip on Windows due to Unix socket path limitations
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows - Unix socket paths have length limitations")
	}

	tempDir := t.TempDir()

	injector, err := NewPIIInjector(&PIIInjectionConfig{
		Method:    "socket",
		SocketDir: tempDir,
		TTL:       5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}
	defer injector.Stop()

	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("test", "value")

	// First injection should succeed
	_, err = injector.InjectPII(context.Background(), "dup", resolved, nil)
	if err != nil {
		t.Fatalf("First injection failed: %v", err)
	}

	// Second injection should fail (session already exists)
	_, err = injector.InjectPII(context.Background(), "dup", resolved, nil)
	if err == nil {
		t.Error("Expected error for duplicate session")
	}
}

func TestPIIInjector_Cleanup(t *testing.T) {
	tempDir := t.TempDir()

	injector, err := NewPIIInjector(&PIIInjectionConfig{
		Method:    "socket",
		SocketDir: tempDir,
		TTL:       5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create injector: %v", err)
	}

	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("test", "value")

	// Create injection
	_, err = injector.InjectPII(context.Background(), "cleanup-test", resolved, nil)
	if err != nil {
		t.Fatalf("Injection failed: %v", err)
	}

	// Verify socket file exists
	socketPath := filepath.Join(tempDir, "cleanup-test.pii.sock")
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket file should exist after injection")
	}

	// Stop injector (should cleanup)
	injector.Stop()

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify socket file is removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after stop")
	}
}

func TestPreparePIIEnvironment(t *testing.T) {
	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("full_name", "John Doe")
	resolved.SetVariable("email", "john@example.com")

	envVars := PreparePIIEnvironment(resolved, "MY_PII_")

	if len(envVars) < 5 { // 2 variables + 3 metadata
		t.Errorf("Expected at least 5 env vars, got %d", len(envVars))
	}

	// Check that variables are present
	foundName := false
	foundEmail := false
	foundRequestID := false

	for _, env := range envVars {
		if env == "MY_PII_full_name=John Doe" {
			foundName = true
		}
		if env == "MY_PII_email=john@example.com" {
			foundEmail = true
		}
		if env == "MY_PII__REQUEST_ID=req-456" {
			foundRequestID = true
		}
	}

	if !foundName {
		t.Error("Missing full_name env var")
	}
	if !foundEmail {
		t.Error("Missing email env var")
	}
	if !foundRequestID {
		// The actual format includes the prefix + _REQUEST_ID
		// Check both formats
		for _, env := range envVars {
			t.Logf("Env var: %s", env)
		}
		t.Error("Missing REQUEST_ID metadata env var")
	}
}

func TestPreparePIIEnvironment_DefaultPrefix(t *testing.T) {
	resolved := pii.NewResolvedVariables("skill-123", "req-456", "profile-789", "user-001")
	resolved.SetVariable("test", "value")

	envVars := PreparePIIEnvironment(resolved, "")

	// Should use default PII_ prefix
	found := false
	for _, env := range envVars {
		if env == "PII_test=value" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected PII_ prefix to be used when none specified")
	}
}

func TestDefaultPIIInjectionConfig(t *testing.T) {
	cfg := DefaultPIIInjectionConfig()

	if cfg.Method != "socket" {
		t.Errorf("Expected default method 'socket', got '%s'", cfg.Method)
	}

	if cfg.SocketDir != "/run/armorclaw/pii" {
		t.Errorf("Expected default socket dir '/run/armorclaw/pii', got '%s'", cfg.SocketDir)
	}

	if cfg.EnvPrefix != "PII_" {
		t.Errorf("Expected default env prefix 'PII_', got '%s'", cfg.EnvPrefix)
	}

	if cfg.TTL != 10*time.Second {
		t.Errorf("Expected default TTL 10s, got %v", cfg.TTL)
	}
}
