package signing

import (
	"testing"
)

func TestSignVerify(t *testing.T) {
	chartData := []byte(`{"domain":"example.com","version":"1.0.0","author":"test"}`)
	secretKey := "test-secret-key-12345"

	// Sign the chart
	signature := SignChart(chartData, secretKey)

	// Verify signature
	valid := VerifySignature(chartData, signature, secretKey)

	if !valid {
		t.Errorf("Expected signature to be valid, got false")
	}

	// Check signature format
	if len(signature) < 7 || signature[:7] != "sha256=" {
		t.Errorf("Expected signature to start with 'sha256=', got: %s", signature[:min(7, len(signature))])
	}
}

func TestVerifyTampered(t *testing.T) {
	chartData := []byte(`{"domain":"example.com","version":"1.0.0"}`)
	secretKey := "test-secret-key-12345"

	// Sign the original chart
	signature := SignChart(chartData, secretKey)

	// Tamper with the data
	tamperedData := []byte(`{"domain":"evil.com","version":"1.0.0"}`)

	// Verify signature with tampered data
	valid := VerifySignature(tamperedData, signature, secretKey)

	if valid {
		t.Errorf("Expected signature to be invalid for tampered data, got true")
	}
}

func TestVerifyInvalid(t *testing.T) {
	chartData := []byte(`{"domain":"example.com","version":"1.0.0"}`)
	secretKey := "test-secret-key-12345"

	// Verify with wrong signature
	wrongSig := "sha256=0000000000000000000000000000000000000000000000000000000000000000"
	valid := VerifySignature(chartData, wrongSig, secretKey)

	if valid {
		t.Errorf("Expected signature to be invalid for wrong signature, got true")
	}

	// Verify with empty signature
	emptySig := ""
	valid = VerifySignature(chartData, emptySig, secretKey)

	if valid {
		t.Errorf("Expected signature to be invalid for empty signature, got true")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
