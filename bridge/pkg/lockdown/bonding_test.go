package lockdown

import (
	"path/filepath"
	"testing"
)

func TestNewBondingManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	lockdownMgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() returned error: %v", err)
	}

	bm := NewBondingManager(lockdownMgr)
	if bm == nil {
		t.Fatal("NewBondingManager() returned nil")
	}
}

func TestGetChallenge(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	lockdownMgr, _ := NewManager(cfg)
	bm := NewBondingManager(lockdownMgr)

	challenge, err := bm.GetChallenge()
	if err != nil {
		t.Fatalf("GetChallenge() returned error: %v", err)
	}

	if challenge.Nonce == "" {
		t.Error("Challenge nonce should not be empty")
	}

	if challenge.CreatedAt.IsZero() {
		t.Error("Challenge created time should be set")
	}

	if challenge.ExpiresAt.IsZero() {
		t.Error("Challenge expires time should be set")
	}
}

func TestValidateChallenge(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	lockdownMgr, _ := NewManager(cfg)
	bm := NewBondingManager(lockdownMgr)

	challenge, _ := bm.GetChallenge()

	// Validate valid challenge
	validated, err := bm.ValidateChallenge(challenge.Nonce)
	if err != nil {
		t.Errorf("ValidateChallenge() returned error for valid challenge: %v", err)
	}

	if validated.Nonce != challenge.Nonce {
		t.Error("Validated challenge nonce mismatch")
	}

	// Validate invalid challenge
	_, err = bm.ValidateChallenge("invalid_nonce")
	if err == nil {
		t.Error("Expected error for invalid challenge")
	}
}

func TestValidateSession(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	lockdownMgr, _ := NewManager(cfg)
	bm := NewBondingManager(lockdownMgr)

	// Test with no admin established
	_, err := bm.ValidateSession("sometoken")
	if err == nil {
		t.Error("Expected error when no admin established")
	}

	// Set admin established
	lockdownMgr.state.mu.Lock()
	lockdownMgr.state.AdminEstablished = true
	lockdownMgr.state.AdminDeviceID = "device_123"
	lockdownMgr.state.mu.Unlock()

	// Test with empty token
	_, err = bm.ValidateSession("")
	if err == nil {
		t.Error("Expected error for empty token")
	}

	// Test with short token
	_, err = bm.ValidateSession("short")
	if err == nil {
		t.Error("Expected error for short token")
	}

	// Test with valid hex token
	device, err := bm.ValidateSession("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Errorf("ValidateSession() returned error for valid token: %v", err)
	}

	if device == nil {
		t.Fatal("Expected device to be returned")
	}

	if !device.IsAdmin {
		t.Error("Expected device to be admin")
	}

	if !device.Trusted {
		t.Error("Expected device to be trusted")
	}

	// Test with invalid hex characters
	_, err = bm.ValidateSession("0123456789abcdef0123456789abcdefZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	if err == nil {
		t.Error("Expected error for invalid hex token")
	}
}

func TestValidateSession_TokenLength(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	lockdownMgr, _ := NewManager(cfg)
	bm := NewBondingManager(lockdownMgr)

	// Set admin established
	lockdownMgr.state.mu.Lock()
	lockdownMgr.state.AdminEstablished = true
	lockdownMgr.state.AdminDeviceID = "device_123"
	lockdownMgr.state.mu.Unlock()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"too short", "0123456789", true},
		{"min length", "0123456789abcdef0123456789abcdef", false},
		{"valid length", "0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"too long", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bm.ValidateSession(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
