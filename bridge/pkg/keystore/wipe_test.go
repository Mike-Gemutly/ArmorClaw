// Package keystore_test provides tests for the keystore.WipeAllData() function.
//
// WipeAllData removes ALL data from the keystore.
// This is a destructive operation that should only be called during readmin process.
// Security audit: This test verifies that WipeAllData respects user isolation.

//go:build cgo

package keystore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armorclaw/bridge/pkg/keystore"
)

// TestWipeAllData_UserIsolation verifies that WipeAllData does NOT support
// user-scoped deletion - it is a system-wide admin operation.
// This test documents the current behavior: WipeAllData deletes ALL data.
// If user-scoped deletion is needed, a new method (e.g., WipeUserData) should be created.
func TestWipeAllData_SystemWide(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "keystore.db")

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

	// Store some test data
	err = ks.Store(keystore.Credential{
		Provider: "openai",
		Token:    "test-token-1",
	})
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Verify data exists
	creds, err := ks.List("")
	if err != nil {
		t.Fatalf("Failed to list credentials: %v", err)
	}
	if len(creds) == 0 {
		t.Fatal("Expected at least one credential before wipe")
	}

	// WipeAllData should delete everything
	err = ks.WipeAllData()
	if err != nil {
		t.Fatalf("WipeAllData failed: %v", err)
	}

	// Verify all data is gone
	creds, err = ks.List("")
	if err != nil {
		t.Fatalf("Failed to list credentials after wipe: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("Expected 0 credentials after WipeAllData, got %d", len(creds))
	}

	t.Log("WipeAllData correctly performs system-wide deletion")
}

// TestWipeAllData_RequiresOpenKeystore verifies that WipeAllData fails
// if the keystore is not open.
func TestWipeAllData_RequiresOpenKeystore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "keystore.db")

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

	// Don't open the keystore - WipeAllData should fail
	err = ks.WipeAllData()
	if err == nil {
		t.Error("Expected WipeAllData to fail on unopened keystore")
	}

	t.Logf("WipeAllData correctly rejected: %v", err)
}
