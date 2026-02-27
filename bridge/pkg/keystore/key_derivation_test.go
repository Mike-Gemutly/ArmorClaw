package keystore

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewKeyDerivation(t *testing.T) {
	kd, err := NewKeyDerivation(DefaultKeyDerivationParams)
	if err != nil {
		t.Fatalf("failed to create key derivation: %v", err)
	}

	if kd == nil {
		t.Fatal("expected non-nil key derivation")
	}
}

func TestNewKeyDerivationInvalidParams(t *testing.T) {
	tests := []struct {
		name   string
		params KeyDerivationParams
	}{
		{
			name: "memory too low",
			params: KeyDerivationParams{
				Memory:      4 * 1024, // 4 MB, below 8 MB minimum
				Iterations:  3,
				Parallelism: 4,
				KeyLength:   32,
				SaltLength:  16,
			},
		},
		{
			name: "iterations zero",
			params: KeyDerivationParams{
				Memory:      64 * 1024,
				Iterations:  0,
				Parallelism: 4,
				KeyLength:   32,
				SaltLength:  16,
			},
		},
		{
			name: "parallelism zero",
			params: KeyDerivationParams{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 0,
				KeyLength:   32,
				SaltLength:  16,
			},
		},
		{
			name: "key length too short",
			params: KeyDerivationParams{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				KeyLength:   8, // Below 16 minimum
				SaltLength:  16,
			},
		},
		{
			name: "salt length too short",
			params: KeyDerivationParams{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				KeyLength:   32,
				SaltLength:  4, // Below 8 minimum
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKeyDerivation(tt.params)
			if err == nil {
				t.Error("expected error for invalid params")
			}
		})
	}
}

func TestDeriveKey(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	password := []byte("test-password-123")
	salt := []byte("test-salt-16-byt")

	derived, err := kd.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	if len(derived.Key) != int(DefaultKeyDerivationParams.KeyLength) {
		t.Errorf("expected key length %d, got %d", DefaultKeyDerivationParams.KeyLength, len(derived.Key))
	}

	if string(derived.Salt) != string(salt) {
		t.Error("salt mismatch")
	}

	// Same password + salt should produce same key
	derived2, _ := kd.DeriveKey(password, salt)
	if !ConstantTimeCompare(derived.Key, derived2.Key) {
		t.Error("same password and salt should produce same key")
	}
}

func TestDeriveKeyWithNewSalt(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	password := []byte("test-password-123")

	derived1, err := kd.DeriveKeyWithNewSalt(password)
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	derived2, _ := kd.DeriveKeyWithNewSalt(password)

	// Different salts should produce different keys
	if ConstantTimeCompare(derived1.Key, derived2.Key) {
		t.Error("different salts should produce different keys")
	}

	if len(derived1.Salt) != int(DefaultKeyDerivationParams.SaltLength) {
		t.Errorf("expected salt length %d, got %d", DefaultKeyDerivationParams.SaltLength, len(derived1.Salt))
	}
}

func TestDeriveKeyEmptySalt(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	_, err := kd.DeriveKey([]byte("password"), []byte{})
	if err != ErrInvalidSaltLength {
		t.Errorf("expected ErrInvalidSaltLength, got %v", err)
	}
}

func TestWrapUnwrapKey(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	password := []byte("test-password-123")

	// Wrap the key
	wrapped, err := kd.WrapKey(plaintextKey, password)
	if err != nil {
		t.Fatalf("failed to wrap key: %v", err)
	}

	if wrapped.Version != 1 {
		t.Errorf("expected version 1, got %d", wrapped.Version)
	}

	if len(wrapped.Ciphertext) == 0 {
		t.Error("expected non-empty ciphertext")
	}

	if len(wrapped.Nonce) == 0 {
		t.Error("expected non-empty nonce")
	}

	if len(wrapped.Salt) == 0 {
		t.Error("expected non-empty salt")
	}

	// Unwrap the key
	unwrapped, err := kd.UnwrapKey(wrapped, password)
	if err != nil {
		t.Fatalf("failed to unwrap key: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original")
	}
}

func TestUnwrapKeyWrongPassword(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	wrapped, _ := kd.WrapKey(plaintextKey, []byte("correct-password"))

	_, err := kd.UnwrapKey(wrapped, []byte("wrong-password"))
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestUnwrapKeyNilWrapped(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	_, err := kd.UnwrapKey(nil, []byte("password"))
	if err != ErrInvalidWrappedKey {
		t.Errorf("expected ErrInvalidWrappedKey, got %v", err)
	}
}

func TestUnwrapKeyInvalidVersion(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	wrapped := &WrappedKey{
		Version:    99,
		Ciphertext: make([]byte, 48),
		Nonce:      make([]byte, 24),
		Salt:       make([]byte, 16),
	}

	_, err := kd.UnwrapKey(wrapped, []byte("password"))
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestWrapKeyInvalidLength(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	_, err := kd.WrapKey([]byte("short"), []byte("password"))
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestVerifyPassword(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	correctPassword := []byte("correct-password")
	wrongPassword := []byte("wrong-password")

	wrapped, _ := kd.WrapKey(plaintextKey, correctPassword)

	if !kd.VerifyPassword(wrapped, correctPassword) {
		t.Error("expected password verification to succeed")
	}

	if kd.VerifyPassword(wrapped, wrongPassword) {
		t.Error("expected password verification to fail with wrong password")
	}
}

func TestRekey(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	oldPassword := []byte("old-password")
	newPassword := []byte("new-password")

	wrapped, _ := kd.WrapKey(plaintextKey, oldPassword)

	// Rekey
	rekeyed, err := kd.Rekey(wrapped, oldPassword, newPassword)
	if err != nil {
		t.Fatalf("failed to rekey: %v", err)
	}

	// Should not unwrap with old password
	if kd.VerifyPassword(rekeyed, oldPassword) {
		t.Error("should not verify with old password after rekey")
	}

	// Should unwrap with new password
	unwrapped, err := kd.UnwrapKey(rekeyed, newPassword)
	if err != nil {
		t.Fatalf("failed to unwrap with new password: %v", err)
	}

	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match original after rekey")
	}
}

func TestRekeyWrongOldPassword(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	wrapped, _ := kd.WrapKey(plaintextKey, []byte("correct-password"))

	_, err := kd.Rekey(wrapped, []byte("wrong-password"), []byte("new-password"))
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestChangeParams(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	password := []byte("test-password")
	wrapped, _ := kd.WrapKey(plaintextKey, password)

	// Change to new params
	newParams := KeyDerivationParams{
		Memory:      128 * 1024, // 128 MB
		Iterations:  4,
		Parallelism: 8,
		KeyLength:   32,
		SaltLength:  16,
	}

	rekeyed, err := kd.ChangeParams(wrapped, password, newParams)
	if err != nil {
		t.Fatalf("failed to change params: %v", err)
	}

	if rekeyed.Params.Memory != newParams.Memory {
		t.Errorf("expected memory %d, got %d", newParams.Memory, rekeyed.Params.Memory)
	}

	// Should still unwrap correctly
	unwrapped, _ := kd.UnwrapKey(rekeyed, password)
	if !ConstantTimeCompare(plaintextKey, unwrapped) {
		t.Error("unwrapped key does not match after param change")
	}
}

func TestChangeParamsInvalid(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	wrapped, _ := kd.WrapKey(plaintextKey, []byte("password"))

	invalidParams := KeyDerivationParams{
		Memory:      1024, // Too low
		Iterations:  3,
		Parallelism: 4,
		KeyLength:   32,
		SaltLength:  16,
	}

	_, err := kd.ChangeParams(wrapped, []byte("password"), invalidParams)
	if err == nil {
		t.Error("expected error for invalid params")
	}
}

func TestWrappedKeyJSON(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	plaintextKey := make([]byte, 32)
	for i := range plaintextKey {
		plaintextKey[i] = byte(i)
	}

	wrapped, _ := kd.WrapKey(plaintextKey, []byte("password"))

	// Marshal to JSON
	data, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var unwrapped WrappedKey
	err = json.Unmarshal(data, &unwrapped)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields match
	if unwrapped.Version != wrapped.Version {
		t.Errorf("version mismatch: %d vs %d", unwrapped.Version, wrapped.Version)
	}

	if !ConstantTimeCompare(unwrapped.Ciphertext, wrapped.Ciphertext) {
		t.Error("ciphertext mismatch")
	}

	if !ConstantTimeCompare(unwrapped.Nonce, wrapped.Nonce) {
		t.Error("nonce mismatch")
	}

	if !ConstantTimeCompare(unwrapped.Salt, wrapped.Salt) {
		t.Error("salt mismatch")
	}
}

func TestGenerateRandomKey(t *testing.T) {
	key, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("expected key length 32, got %d", len(key))
	}

	// Generate another key - should be different
	key2, _ := GenerateRandomKey(32)
	if ConstantTimeCompare(key, key2) {
		t.Error("two random keys should not be equal")
	}
}

func TestGenerateRandomKeyTooShort(t *testing.T) {
	_, err := GenerateRandomKey(8)
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestGetDefaultParams(t *testing.T) {
	params := GetDefaultParams()

	if params.Memory != 64*1024 {
		t.Errorf("expected memory 64MB, got %d", params.Memory)
	}

	if params.Iterations != 3 {
		t.Errorf("expected iterations 3, got %d", params.Iterations)
	}

	if params.Parallelism != 4 {
		t.Errorf("expected parallelism 4, got %d", params.Parallelism)
	}
}

func TestGetSetParams(t *testing.T) {
	kd, _ := NewKeyDerivation(DefaultKeyDerivationParams)

	params := kd.GetParams()
	if params.Memory != DefaultKeyDerivationParams.Memory {
		t.Errorf("expected default memory, got %d", params.Memory)
	}

	newParams := KeyDerivationParams{
		Memory:      128 * 1024,
		Iterations:  4,
		Parallelism: 8,
		KeyLength:   32,
		SaltLength:  16,
	}

	err := kd.SetParams(newParams)
	if err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	params = kd.GetParams()
	if params.Memory != newParams.Memory {
		t.Errorf("expected memory %d, got %d", newParams.Memory, params.Memory)
	}
}

func TestEstimateDerivationTime(t *testing.T) {
	params := DefaultKeyDerivationParams

	estimated := EstimateDerivationTime(params)

	// Default params should take around 200ms
	if estimated < 100*time.Millisecond || estimated > 1*time.Second {
		t.Errorf("unexpected derivation time estimate: %v", estimated)
	}

	// Double memory should roughly double time
	doubleMemory := params
	doubleMemory.Memory = params.Memory * 2
	estimatedDouble := EstimateDerivationTime(doubleMemory)

	if estimatedDouble <= estimated {
		t.Error("expected double memory to increase estimated time")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	a := []byte("test-value-123")
	b := []byte("test-value-123")
	c := []byte("different-value")

	if !ConstantTimeCompare(a, b) {
		t.Error("expected equal slices to match")
	}

	if ConstantTimeCompare(a, c) {
		t.Error("expected different slices to not match")
	}

	if ConstantTimeCompare(a, []byte("short")) {
		t.Error("expected different length slices to not match")
	}
}

func TestKeyDerivationParamsValidate(t *testing.T) {
	tests := []struct {
		name    string
		params  KeyDerivationParams
		wantErr bool
	}{
		{
			name:    "valid defaults",
			params:  DefaultKeyDerivationParams,
			wantErr: false,
		},
		{
			name: "minimum memory",
			params: KeyDerivationParams{
				Memory:      8 * 1024,
				Iterations:  1,
				Parallelism: 1,
				KeyLength:   16,
				SaltLength:  8,
			},
			wantErr: false,
		},
		{
			name: "memory too low",
			params: KeyDerivationParams{
				Memory:      7 * 1024,
				Iterations:  3,
				Parallelism: 4,
				KeyLength:   32,
				SaltLength:  16,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
