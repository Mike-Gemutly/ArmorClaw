package keystore

import (
	"testing"
	"time"
)

// TestKeyDerivationIntegration tests the key derivation integration
func TestKeyDerivationIntegration(t *testing.T) {
	// Test with default parameters
	kd, err := NewKeyDerivation(DefaultKeyDerivationParams)
	if err != nil {
		t.Fatalf("failed to create key derivation: %v", err)
	}

	password := []byte("test-password-12345")

	// Generate a random key to wrap
	plaintextKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	// Wrap the key
	wrapped, err := kd.WrapKey(plaintextKey, password)
	if err != nil {
		t.Fatalf("failed to wrap key: %v", err)
	}

	// Verify wrapped key structure
	if wrapped.Version != 1 {
		t.Errorf("expected version 1, got %d", wrapped.Version)
	}
	if len(wrapped.Ciphertext) == 0 {
		t.Error("expected non-empty ciphertext")
	}
	if len(wrapped.Nonce) != 24 { // XChaCha20-Poly1305 nonce size
		t.Errorf("expected 24-byte nonce, got %d", len(wrapped.Nonce))
	}
	if len(wrapped.Salt) == 0 {
		t.Error("expected non-empty salt")
	}

	// Unwrap the key
	unwrapped, err := kd.UnwrapKey(wrapped, password)
	if err != nil {
		t.Fatalf("failed to unwrap key: %v", err)
	}

	// Verify unwrapped key matches original
	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original")
	}
}

// TestKeyDerivationWithDifferentParams tests key derivation with custom parameters
func TestKeyDerivationWithDifferentParams(t *testing.T) {
	customParams := KeyDerivationParams{
		Memory:      32 * 1024, // 32 MB
		Iterations:  2,
		Parallelism: 2,
		KeyLength:   32,
		SaltLength:  16,
	}

	kd, err := NewKeyDerivation(customParams)
	if err != nil {
		t.Fatalf("failed to create key derivation: %v", err)
	}

	password := []byte("custom-params-password")
	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	// Wrap and unwrap
	wrapped, err := kd.WrapKey(plaintextKey, password)
	if err != nil {
		t.Fatalf("failed to wrap key: %v", err)
	}

	unwrapped, err := kd.UnwrapKey(wrapped, password)
	if err != nil {
		t.Fatalf("failed to unwrap key: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original")
	}
}

// TestKeyDerivationRekeyFlow tests the rekey workflow
func TestKeyDerivationRekeyFlow(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	oldPassword := []byte("old-password-123")
	newPassword := []byte("new-password-456")

	// Wrap with old password
	wrapped, err := kd.WrapKey(plaintextKey, oldPassword)
	if err != nil {
		t.Fatalf("failed to wrap key: %v", err)
	}

	// Rekey
	rekeyed, err := kd.Rekey(wrapped, oldPassword, newPassword)
	if err != nil {
		t.Fatalf("failed to rekey: %v", err)
	}

	// Verify old password no longer works
	_, err = kd.UnwrapKey(rekeyed, oldPassword)
	if err == nil {
		t.Error("old password should not work after rekey")
	}

	// Verify new password works
	unwrapped, err := kd.UnwrapKey(rekeyed, newPassword)
	if err != nil {
		t.Fatalf("failed to unwrap with new password: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original after rekey")
	}
}

// TestKeyDerivationParamChange tests changing derivation parameters
func TestKeyDerivationParamChange(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	password := []byte("test-password")

	// Wrap with default params
	wrapped, _ := kd.WrapKey(plaintextKey, password)

	// Change to more secure params
	newParams := KeyDerivationParams{
		Memory:      128 * 1024, // 128 MB
		Iterations:  4,
		Parallelism: 8,
		KeyLength:   32,
		SaltLength:  16,
	}

	// Rekey with new params
	rekeyed, err := kd.ChangeParams(wrapped, password, newParams)
	if err != nil {
		t.Fatalf("failed to change params: %v", err)
	}

	// Verify params were changed
	if rekeyed.Params.Memory != newParams.Memory {
		t.Errorf("expected memory %d, got %d", newParams.Memory, rekeyed.Params.Memory)
	}

	// Verify key still works
	unwrapped, _ := kd.UnwrapKey(rekeyed, password)
	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original after param change")
	}
}

// TestKeyDerivationMultipleKeys tests wrapping multiple keys
func TestKeyDerivationMultipleKeys(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	password := []byte("shared-password")

	// Create multiple keys
	keys := make([][]byte, 5)
	wrappedKeys := make([]*WrappedKey, 5)

	for i := 0; i < 5; i++ {
		keys[i], _ = GenerateRandomKey(32)
		wrappedKeys[i], _ = kd.WrapKey(keys[i], password)
	}

	// Verify each key can be unwrapped correctly
	for i := 0; i < 5; i++ {
		unwrapped, err := kd.UnwrapKey(wrappedKeys[i], password)
		if err != nil {
			t.Fatalf("failed to unwrap key %d: %v", i, err)
		}
		if !ConstantTimeCompare(keys[i], unwrapped) {
			t.Errorf("key %d does not match after unwrap", i)
		}
	}
}

// TestKeyDerivationJSONSerialization tests JSON serialization of wrapped keys
func TestKeyDerivationJSONSerialization(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	password := []byte("json-test-password")

	// Wrap key
	wrapped, _ := kd.WrapKey(plaintextKey, password)

	// Serialize to JSON
	jsonData, err := wrapped.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal wrapped key: %v", err)
	}

	// Deserialize from JSON
	var unwrappedWrapped WrappedKey
	err = unwrappedWrapped.UnmarshalJSON(jsonData)
	if err != nil {
		t.Fatalf("failed to unmarshal wrapped key: %v", err)
	}

	// Verify unwrapping works with deserialized data
	unwrapped, err := kd.UnwrapKey(&unwrappedWrapped, password)
	if err != nil {
		t.Fatalf("failed to unwrap deserialized key: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("deserialized key does not match original")
	}
}

// TestKeyDerivationPerformance tests key derivation performance
func TestKeyDerivationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	params := KeyDerivationParams{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 4,
		KeyLength:   32,
		SaltLength:  16,
	}

	kd, _ := NewKeyDerivation(params)
	password := []byte("performance-test-password")
	plaintextKey := make([]byte, 32)

	// Measure wrap time
	start := time.Now()
	_, err := kd.WrapKey(plaintextKey, password)
	wrapDuration := time.Since(start)

	if err != nil {
		t.Fatalf("failed to wrap key: %v", err)
	}

	// Verify reasonable performance (should be under 2 seconds for default params)
	if wrapDuration > 2*time.Second {
		t.Errorf("key wrapping took too long: %v", wrapDuration)
	}

	t.Logf("Key wrap time: %v", wrapDuration)
}

// TestKeyDerivationBoundaryConditions tests boundary conditions
func TestKeyDerivationBoundaryConditions(t *testing.T) {
	// Test minimum valid params
	// Note: KeyLength must be 32 for XChaCha20-Poly1305
	minParams := KeyDerivationParams{
		Memory:      8 * 1024, // 8 MB minimum
		Iterations:  1,
		Parallelism: 1,
		KeyLength:   32, // 32 bytes required for XChaCha20-Poly1305
		SaltLength:  8,  // 8 bytes minimum
	}

	kd, err := NewKeyDerivation(minParams)
	if err != nil {
		t.Fatalf("failed to create key derivation with min params: %v", err)
	}

	// Verify it works
	password := []byte("min-params-password")
	plaintextKey := make([]byte, 32)

	wrapped, err := kd.WrapKey(plaintextKey, password)
	if err != nil {
		t.Fatalf("failed to wrap with min params: %v", err)
	}

	unwrapped, err := kd.UnwrapKey(wrapped, password)
	if err != nil {
		t.Fatalf("failed to unwrap with min params: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original")
	}
}

// TestKeyDerivationDifferentPasswords tests that different passwords produce different wrapped keys
func TestKeyDerivationDifferentPasswords(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	password1 := []byte("password-1")
	password2 := []byte("password-2")

	// Wrap with different passwords
	wrapped1, _ := kd.WrapKey(plaintextKey, password1)
	wrapped2, _ := kd.WrapKey(plaintextKey, password2)

	// Ciphertexts should be different (different salts)
	if ConstantTimeCompare(wrapped1.Ciphertext, wrapped2.Ciphertext) {
		t.Error("different passwords should produce different ciphertexts")
	}

	// Each should unwrap with its own password
	unwrapped1, _ := kd.UnwrapKey(wrapped1, password1)
	unwrapped2, _ := kd.UnwrapKey(wrapped2, password2)

	if !ConstantTimeCompare(plaintextKey, unwrapped1) {
		t.Error("key 1 does not match")
	}
	if !ConstantTimeCompare(plaintextKey, unwrapped2) {
		t.Error("key 2 does not match")
	}
}

// TestEstimateDerivationTimeAccuracy tests estimation accuracy
func TestEstimateDerivationTimeAccuracy(t *testing.T) {
	params := DefaultKeyDerivationParams
	estimated := EstimateDerivationTime(params)

	// Estimated time should be positive
	if estimated <= 0 {
		t.Error("estimated time should be positive")
	}

	// Should be in reasonable range (100ms - 2s for default params)
	if estimated < 100*time.Millisecond || estimated > 2*time.Second {
		t.Errorf("estimated time %v seems unreasonable for default params", estimated)
	}

	// Double memory should roughly double estimate
	doubleParams := params
	doubleParams.Memory = params.Memory * 2
	doubleEstimate := EstimateDerivationTime(doubleParams)

	if doubleEstimate <= estimated {
		t.Error("doubling memory should increase estimated time")
	}
}
