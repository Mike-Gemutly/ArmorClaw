// Package agent provides end-to-end integration tests for PII shadow flow
// Tests the complete flow: prompt → interceptor → injection → browser
package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPIIShadowE2E_SinglePattern tests the full flow with a single PII value
func TestPIIShadowE2E_SinglePattern(t *testing.T) {
	// Setup: Create a temporary socket for testing
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	// Step 1: Create mapping with synthetic PII data (no real secrets)
	mapping := NewPIIMapping()
	syntheticEmail := "test-user@example.com"
	emailHash := "abc123def456"
	mapping.set(emailHash, syntheticEmail)

	// Step 2: Start PII injector with the mapping
	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	// Step 3: Create interceptor
	interceptor := NewPIIInterceptor()

	// Step 4: Intercept prompt containing vault pattern
	prompt := "Please send email to {{VAULT:" + emailHash + "}}"
	modifiedPrompt, interceptedMapping, err := interceptor.Intercept(prompt)
	if err != nil {
		t.Fatalf("Failed to intercept prompt: %v", err)
	}

	// Step 5: Verify interceptor redacted the pattern
	expectedRedacted := "Please send email to [REDACTED:" + emailHash + "]"
	if modifiedPrompt != expectedRedacted {
		t.Errorf("Expected redacted prompt %q, got %q", expectedRedacted, modifiedPrompt)
	}

	// Step 6: Verify LLM never receives actual value (the prompt sent to LLM contains redacted)
	if strings.Contains(modifiedPrompt, syntheticEmail) {
		t.Error("Modified prompt still contains actual PII value - security violation")
	}

	// Step 7: Verify interceptor created mapping
	if interceptedMapping.Count() != 1 {
		t.Errorf("Expected 1 mapping entry, got %d", interceptedMapping.Count())
	}

	// Step 8: Simulate browser requesting the value via socket (like real flow)
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: emailHash, Secret: "test-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Step 9: Verify browser receives actual value (not the LLM)
	if resp.Error != "" {
		t.Errorf("Unexpected error from injector: %s", resp.Error)
	}
	if resp.Value != syntheticEmail {
		t.Errorf("Expected value %q, got %q", syntheticEmail, resp.Value)
	}

	// Step 10: Verify the redacted placeholder is used to lookup original
	redactedKey := "[REDACTED:" + emailHash + "]"
	originalPattern, exists := interceptedMapping.Get(redactedKey)
	if !exists {
		t.Error("Redacted key not found in mapping")
	}
	if originalPattern != "{{VAULT:"+emailHash+"}}" {
		t.Errorf("Expected original pattern %q, got %q", "{{VAULT:"+emailHash+"}}", originalPattern)
	}
}

// TestPIIShadowE2E_MultiplePatterns tests multiple PII values in a single prompt
func TestPIIShadowE2E_MultiplePatterns(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_multi_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	testData := map[string]string{
		"a1b2c3d4": "user@example.com",
		"e5f6a7b8": "secret-password-123",
		"9e0a1b2c": "4111111111111111", // Synthetic credit card
		"d3e4f5a6": "John Doe",
	}

	for hash, value := range testData {
		mapping.set(hash, value)
	}

	injector := NewPIIInjector(socketPath, "multi-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	interceptor := NewPIIInterceptor()

	prompt := "Email: {{VAULT:a1b2c3d4}}, Password: {{VAULT:e5f6a7b8}}, Card: {{VAULT:9e0a1b2c}}, Name: {{VAULT:d3e4f5a6}}"
	modifiedPrompt, interceptedMapping, err := interceptor.Intercept(prompt)
	if err != nil {
		t.Fatalf("Failed to intercept prompt: %v", err)
	}

	// Verify all patterns are redacted
	if strings.Contains(modifiedPrompt, "{{VAULT:") {
		t.Error("Modified prompt still contains vault patterns - security violation")
	}

	// Verify all actual values are absent
	for _, value := range testData {
		if strings.Contains(modifiedPrompt, value) {
			t.Errorf("Modified prompt contains actual PII value %q - security violation", value)
		}
	}

	// Verify correct number of redactions
	if interceptedMapping.Count() != len(testData) {
		t.Errorf("Expected %d mapping entries, got %d", len(testData), interceptedMapping.Count())
	}

	// Verify each redacted value can be retrieved via socket
	for hash, expectedValue := range testData {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to connect to socket: %v", err)
		}

		req := PIIRequest{Hash: hash, Secret: "multi-secret"}
		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(req); err != nil {
			conn.Close()
			t.Fatalf("Failed to send request: %v", err)
		}

		var resp PIIResponse
		decoder := json.NewDecoder(conn)
		if err := decoder.Decode(&resp); err != nil {
			conn.Close()
			t.Fatalf("Failed to read response: %v", err)
		}
		conn.Close()

		if resp.Error != "" {
			t.Errorf("Unexpected error for hash %s: %s", hash, resp.Error)
		}
		if resp.Value != expectedValue {
			t.Errorf("Expected value %q for hash %s, got %q", expectedValue, hash, resp.Value)
		}
	}
}

// TestPIIShadowE2E_NoPatterns tests prompt with no PII patterns
func TestPIIShadowE2E_NoPatterns(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_none_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	interceptor := NewPIIInterceptor()

	prompt := "This is a normal prompt with no vault patterns"
	modifiedPrompt, interceptedMapping, err := interceptor.Intercept(prompt)
	if err != nil {
		t.Fatalf("Failed to intercept prompt: %v", err)
	}

	// Verify prompt unchanged
	if modifiedPrompt != prompt {
		t.Errorf("Expected prompt to remain unchanged, got %q", modifiedPrompt)
	}

	// Verify no mapping entries
	if interceptedMapping.Count() != 0 {
		t.Errorf("Expected 0 mapping entries, got %d", interceptedMapping.Count())
	}
}

// TestPIIShadowE2E_InvalidHash tests error case: hash not found in mapping
func TestPIIShadowE2E_InvalidHash(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_invalid_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	mapping.set("validhash", "valid-value")

	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	// Connect to socket and request invalid hash
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: "invalidhash", Secret: "test-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify error response
	if resp.Error != "hash not found" {
		t.Errorf("Expected 'hash not found' error, got %q", resp.Error)
	}
	if resp.Value != "" {
		t.Errorf("Expected no value for invalid hash, got %q", resp.Value)
	}
}

// TestPIIShadowE2E_AuthFailure tests error case: authentication failure
func TestPIIShadowE2E_AuthFailure(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_auth_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	mapping.set("secrethash", "secret-value")

	injector := NewPIIInjector(socketPath, "valid-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	// Connect with wrong secret
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: "secrethash", Secret: "wrong-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify auth error
	if resp.Error != "authentication failed" {
		t.Errorf("Expected 'authentication failed' error, got %q", resp.Error)
	}
	if resp.Value != "" {
		t.Errorf("Expected no value on auth failure, got %q", resp.Value)
	}
}

// TestPIIShadowE2E_DuplicateHash tests that duplicate hashes in a prompt are handled correctly
func TestPIIShadowE2E_DuplicateHash(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_dup_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	duplicateHash := "d1a2b3c4"
	mapping.set(duplicateHash, "duplicate-value")

	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	interceptor := NewPIIInterceptor()

	prompt := "Use {{VAULT:" + duplicateHash + "}} here and {{VAULT:" + duplicateHash + "}} there"
	modifiedPrompt, interceptedMapping, err := interceptor.Intercept(prompt)
	if err != nil {
		t.Fatalf("Failed to intercept prompt: %v", err)
	}

	// Verify both occurrences are redacted
	expectedRedacted := "Use [REDACTED:" + duplicateHash + "] here and [REDACTED:" + duplicateHash + "] there"
	if modifiedPrompt != expectedRedacted {
		t.Errorf("Expected %q, got %q", expectedRedacted, modifiedPrompt)
	}

	// Verify only one mapping entry (deduplication)
	if interceptedMapping.Count() != 1 {
		t.Errorf("Expected 1 mapping entry (deduplication), got %d", interceptedMapping.Count())
	}

	// Verify value can still be retrieved
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: duplicateHash, Secret: "test-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Value != "duplicate-value" {
		t.Errorf("Expected 'duplicate-value', got %q", resp.Value)
	}
}

// TestPIIShadowE2E_LLMIsolation verifies that the LLM truly never sees PII
func TestPIIShadowE2E_LLMIsolation(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_isol_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	sensitiveHash := "s3ns1t1v3"
	sensitiveValue := "this-is-extremely-sensitive-data-123456789"
	mapping.set(sensitiveHash, sensitiveValue)

	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	interceptor := NewPIIInterceptor()

	prompt := "Process the following: {{VAULT:" + sensitiveHash + "}}"
	llmPrompt, _, err := interceptor.Intercept(prompt)
	if err != nil {
		t.Fatalf("Failed to intercept prompt: %v", err)
	}

	// Critical security check: LLM prompt must NOT contain the actual value
	if strings.Contains(llmPrompt, sensitiveValue) {
		t.Error("CRITICAL SECURITY FAILURE: LLM prompt contains actual PII value")
	}

	// LLM prompt must NOT contain any part of the value longer than the hash
	// (this prevents partial leakage)
	words := strings.Fields(sensitiveValue)
	for _, word := range words {
		if len(word) > 8 && strings.Contains(llmPrompt, word) {
			t.Errorf("Potential security issue: LLM prompt contains long word from value: %q", word)
		}
	}

	// Verify value is still retrievable by authorized browser
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	req := PIIRequest{Hash: sensitiveHash, Secret: "test-secret"}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var resp PIIResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Value != sensitiveValue {
		t.Errorf("Expected value %q, got %q", sensitiveValue, resp.Value)
	}
}

// TestPIIShadowE2E_ConcurrentRequests tests concurrent socket requests
func TestPIIShadowE2E_ConcurrentRequests(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("pii_shadow_conc_%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	mapping := NewPIIMapping()
	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		value := fmt.Sprintf("value%d", i)
		mapping.set(hash, value)
	}

	injector := NewPIIInjector(socketPath, "test-secret", mapping)
	if err := injector.Start(); err != nil {
		t.Fatalf("Failed to start PII injector: %v", err)
	}
	defer injector.Stop()

	// Launch concurrent requests
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				errChan <- fmt.Errorf("connection failed: %w", err)
				return
			}
			defer conn.Close()

			hash := fmt.Sprintf("hash%d", idx)
			expectedValue := fmt.Sprintf("value%d", idx)

			req := PIIRequest{Hash: hash, Secret: "test-secret"}
			encoder := json.NewEncoder(conn)
			if err := encoder.Encode(req); err != nil {
				errChan <- fmt.Errorf("encode failed: %w", err)
				return
			}

			var resp PIIResponse
			decoder := json.NewDecoder(conn)
			if err := decoder.Decode(&resp); err != nil {
				errChan <- fmt.Errorf("decode failed: %w", err)
				return
			}

			if resp.Value != expectedValue {
				errChan <- fmt.Errorf("expected %q, got %q", expectedValue, resp.Value)
				return
			}

			errChan <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}
