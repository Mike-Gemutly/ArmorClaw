//go:build cgo

// Package trust provides user hardening state management for ArmorClaw
package trust

import (
	"path/filepath"
	"testing"

	"github.com/armorclaw/bridge/pkg/keystore"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// TestHardeningStoreAckStepForNonExistentUser tests that AckStep handles new users gracefully
// This test verifies the fix for the "no row in result set" error
func TestHardeningStoreAckStepForNonExistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := keystore.New(keystore.Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	db := ks.GetDB()
	store := NewKeystoreHardeningStore(db)

	userID := "@newuser:example.com"

	state, err := store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if state.DelegationReady {
		t.Error("Non-existent user should not be delegation ready")
	}

	if err := store.AckStep(userID, PasswordRotated); err != nil {
		t.Fatalf("AckStep failed for non-existent user (this is the bug): %v", err)
	}

	state, err = store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after ack: %v", err)
	}

	if !state.PasswordRotated {
		t.Error("PasswordRotated should be true after AckStep")
	}
	if state.DelegationReady {
		t.Error("DelegationReady should be false (only one step complete)")
	}

	if err := store.AckStep(userID, BootstrapWiped); err != nil {
		t.Fatalf("Failed to ack BootstrapWiped: %v", err)
	}

	// Ack another step to verify it works on the now-existent user
	if err := store.AckStep(userID, BootstrapWiped); err != nil {
		t.Fatalf("Failed to ack BootstrapWiped: %v", err)
	}

	state, err = store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after second ack: %v", err)
	}

	if !state.PasswordRotated {
		t.Error("PasswordRotated should still be true")
	}
	if !state.BootstrapWiped {
		t.Error("BootstrapWiped should be true after second AckStep")
	}
}

// TestHardeningStoreIsDelegationReadyForNonExistentUser tests that IsDelegationReady
// returns false for non-existent users (existing behavior, verifying it doesn't break)
func TestHardeningStoreIsDelegationReadyForNonExistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := keystore.New(keystore.Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	db := ks.GetDB()
	store := NewKeystoreHardeningStore(db)

	userID := "@nonexistent:example.com"

	ready, err := store.IsDelegationReady(userID)
	if err != nil {
		t.Fatalf("IsDelegationReady failed: %v", err)
	}
	if ready {
		t.Error("IsDelegationReady should return false for non-existent user")
	}
}

// TestHardeningStoreGetForNonExistentUser tests that Get returns default state
// for non-existent users (existing behavior, verifying it doesn't break)
func TestHardeningStoreGetForNonExistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := keystore.New(keystore.Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	db := ks.GetDB()
	store := NewKeystoreHardeningStore(db)

	userID := "@nonexistent:example.com"

	state, err := store.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if state.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", state.UserID, userID)
	}
	if state.PasswordRotated {
		t.Error("PasswordRotated should be false for non-existent user")
	}
	if state.BootstrapWiped {
		t.Error("BootstrapWiped should be false for non-existent user")
	}
	if state.DeviceVerified {
		t.Error("DeviceVerified should be false for non-existent user")
	}
	if state.RecoveryBackedUp {
		t.Error("RecoveryBackedUp should be false for non-existent user")
	}
	if state.BiometricsEnabled {
		t.Error("BiometricsEnabled should be false for non-existent user")
	}
	if state.DelegationReady {
		t.Error("DelegationReady should be false for non-existent user")
	}
}
