// Package agent provides PII value injection via Unix domain sockets.
// This allows browser-service to request PII values by hash without
// exposing them to agents or logs.
package agent

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

const (
	PIISocketPath = "/run/armorclaw/pii.sock"
)

func TestSocketCreation(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "test_pii.sock")
	_ = os.Remove(socketPath)

	mapping := NewPIIMapping()
	injector := NewPIIInjector(socketPath, "test-secret", mapping)

	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start injector: %v", err)
	}
	defer injector.Stop()

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Errorf("Socket file does not exist after creation: %v", err)
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	conn.Close()
}

func TestAuth(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "test_auth.sock")
	_ = os.Remove(socketPath)

	mapping := NewPIIMapping()
	mapping.set("abc123", "my-credit-card-number")
	injector := NewPIIInjector(socketPath, "valid-secret", mapping)

	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start injector: %v", err)
	}
	defer injector.Stop()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: "abc123", Secret: "wrong-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != "authentication failed" {
		t.Errorf("Expected authentication failed error, got: %v", resp)
	}
	if resp.Value != "" {
		t.Errorf("Expected no value on auth failure, got: %s", resp.Value)
	}

	conn2, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn2.Close()

	req = PIIRequest{Hash: "abc123", Secret: "valid-secret"}
	encoder = json.NewEncoder(conn2)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp2 PIIResponse
	decoder = json.NewDecoder(conn2)
	if err := decoder.Decode(&resp2); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp2.Error != "" {
		t.Errorf("Expected no error on valid auth, got: %v", resp2.Error)
	}
	if resp2.Value != "my-credit-card-number" {
		t.Errorf("Expected value 'my-credit-card-number', got: %s", resp2.Value)
	}
}

func TestValueLookup(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "test_lookup.sock")
	_ = os.Remove(socketPath)

	mapping := NewPIIMapping()
	mapping.set("hash1", "value1")
	mapping.set("hash2", "value2")
	injector := NewPIIInjector(socketPath, "test-secret", mapping)

	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start injector: %v", err)
	}
	defer injector.Stop()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: "hash1", Secret: "test-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Expected no error for found hash, got: %v", resp.Error)
	}
	if resp.Value != "value1" {
		t.Errorf("Expected value 'value1', got: %s", resp.Value)
	}

	conn2, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn2.Close()

	req = PIIRequest{Hash: "nonexistent", Secret: "test-secret"}
	encoder = json.NewEncoder(conn2)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp2 PIIResponse
	decoder = json.NewDecoder(conn2)
	if err := decoder.Decode(&resp2); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp2.Error != "hash not found" {
		t.Errorf("Expected 'hash not found' error, got: %v", resp2.Error)
	}
	if resp2.Value != "" {
		t.Errorf("Expected no value for not found hash, got: %s", resp2.Value)
	}
}
