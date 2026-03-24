//go:build cgo

// Package trust provides user hardening state management for ArmorClaw
package trust

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func TestHardening_DelegationReadyRequiresAllMandatorySteps(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false, // Optional
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if !state.DelegationReady {
		t.Errorf("Expected DelegationReady=true when all mandatory steps complete, got false")
	}
}

func TestHardening_DelegationReadyWithoutPasswordRotated(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   false, // Missing
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if state.DelegationReady {
		t.Errorf("Expected DelegationReady=false when PasswordRotated is false, got true")
	}
}

func TestHardening_DelegationReadyWithoutBootstrapWiped(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   true,
		BootstrapWiped:    false, // Missing
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if state.DelegationReady {
		t.Errorf("Expected DelegationReady=false when BootstrapWiped is false, got true")
	}
}

func TestHardening_DelegationReadyWithoutDeviceVerified(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    false, // Missing
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if state.DelegationReady {
		t.Errorf("Expected DelegationReady=false when DeviceVerified is false, got true")
	}
}

func TestHardening_DelegationReadyWithoutRecoveryBackedUp(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  false, // Missing
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if state.DelegationReady {
		t.Errorf("Expected DelegationReady=false when RecoveryBackedUp is false, got true")
	}
}

func TestHardening_BiometricsOptional(t *testing.T) {
	tests := []struct {
		name              string
		biometricsEnabled bool
		expectReady       bool
	}{
		{
			name:              "Delegation ready without biometrics",
			biometricsEnabled: false,
			expectReady:       true,
		},
		{
			name:              "Delegation ready with biometrics",
			biometricsEnabled: true,
			expectReady:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &UserHardeningState{
				UserID:            "user123",
				PasswordRotated:   true,
				BootstrapWiped:    true,
				DeviceVerified:    true,
				RecoveryBackedUp:  true,
				BiometricsEnabled: tt.biometricsEnabled,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}

			state.Recompute()

			if state.DelegationReady != tt.expectReady {
				t.Errorf("Expected DelegationReady=%v with BiometricsEnabled=%v, got %v",
					tt.expectReady, tt.biometricsEnabled, state.DelegationReady)
			}
		})
	}
}

func TestHardening_DelegationReadyFalseWithNoStepsComplete(t *testing.T) {
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   false,
		BootstrapWiped:    false,
		DeviceVerified:    false,
		RecoveryBackedUp:  false,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	state.Recompute()

	if state.DelegationReady {
		t.Errorf("Expected DelegationReady=false when no steps complete, got true")
	}
}

func TestHardening_RecomputeUpdatesTimestamp(t *testing.T) {
	initialTime := time.Now()
	state := &UserHardeningState{
		UserID:            "user123",
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         initialTime,
		UpdatedAt:         initialTime,
	}

	// Add a small delay to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)
	state.Recompute()

	if state.UpdatedAt.Equal(initialTime) {
		t.Errorf("Expected UpdatedAt to change after Recompute(), but it remained the same")
	}
}

func TestHardeningStoreGetPut(t *testing.T) {
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

	state := &UserHardeningState{
		UserID:            "@test:example.com",
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    false,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	state.Recompute()

	if err := store.Put(state); err != nil {
		t.Fatalf("Failed to put hardening state: %v", err)
	}

	retrieved, err := store.Get("@test:example.com")
	if err != nil {
		t.Fatalf("Failed to get hardening state: %v", err)
	}

	if retrieved.UserID != state.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", retrieved.UserID, state.UserID)
	}
	if retrieved.PasswordRotated != state.PasswordRotated {
		t.Errorf("PasswordRotated mismatch: got %v, want %v", retrieved.PasswordRotated, state.PasswordRotated)
	}
	if retrieved.BootstrapWiped != state.BootstrapWiped {
		t.Errorf("BootstrapWiped mismatch: got %v, want %v", retrieved.BootstrapWiped, state.BootstrapWiped)
	}
	if retrieved.DeviceVerified != state.DeviceVerified {
		t.Errorf("DeviceVerified mismatch: got %v, want %v", retrieved.DeviceVerified, state.DeviceVerified)
	}
	if retrieved.RecoveryBackedUp != state.RecoveryBackedUp {
		t.Errorf("RecoveryBackedUp mismatch: got %v, want %v", retrieved.RecoveryBackedUp, state.RecoveryBackedUp)
	}
	if retrieved.BiometricsEnabled != state.BiometricsEnabled {
		t.Errorf("BiometricsEnabled mismatch: got %v, want %v", retrieved.BiometricsEnabled, state.BiometricsEnabled)
	}
	if retrieved.DelegationReady != state.DelegationReady {
		t.Errorf("DelegationReady mismatch: got %v, want %v", retrieved.DelegationReady, state.DelegationReady)
	}
}

func TestHardeningStoreGetDefaults(t *testing.T) {
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

	state, err := store.Get("@nonexistent:example.com")
	if err != nil {
		t.Fatalf("Failed to get hardening state: %v", err)
	}

	if state.UserID != "@nonexistent:example.com" {
		t.Errorf("UserID mismatch: got %s, want @nonexistent:example.com", state.UserID)
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

func TestHardeningStoreAckStep(t *testing.T) {
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

	userID := "@test:example.com"

	initialState := &UserHardeningState{
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Put(initialState); err != nil {
		t.Fatalf("Failed to put initial state: %v", err)
	}

	if err := store.AckStep(userID, PasswordRotated); err != nil {
		t.Fatalf("Failed to ack PasswordRotated: %v", err)
	}

	state, err := store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after ack: %v", err)
	}

	if !state.PasswordRotated {
		t.Error("PasswordRotated should be true after AckStep")
	}
	if state.BootstrapWiped {
		t.Error("BootstrapWiped should still be false")
	}
	if state.DelegationReady {
		t.Error("DelegationReady should be false (not all mandatory steps complete)")
	}

	if err := store.AckStep(userID, BootstrapWiped); err != nil {
		t.Fatalf("Failed to ack BootstrapWiped: %v", err)
	}

	if err := store.AckStep(userID, DeviceVerified); err != nil {
		t.Fatalf("Failed to ack DeviceVerified: %v", err)
	}

	if err := store.AckStep(userID, RecoveryBackedUp); err != nil {
		t.Fatalf("Failed to ack RecoveryBackedUp: %v", err)
	}

	state, err = store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after all acks: %v", err)
	}

	if !state.PasswordRotated {
		t.Error("PasswordRotated should be true")
	}
	if !state.BootstrapWiped {
		t.Error("BootstrapWiped should be true")
	}
	if !state.DeviceVerified {
		t.Error("DeviceVerified should be true")
	}
	if !state.RecoveryBackedUp {
		t.Error("RecoveryBackedUp should be true")
	}
	if !state.DelegationReady {
		t.Error("DelegationReady should be true after all mandatory steps are complete")
	}

	if err := store.AckStep(userID, BiometricsEnabled); err != nil {
		t.Fatalf("Failed to ack BiometricsEnabled: %v", err)
	}

	state, err = store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after biometrics ack: %v", err)
	}

	if !state.BiometricsEnabled {
		t.Error("BiometricsEnabled should be true")
	}
	if !state.DelegationReady {
		t.Error("DelegationReady should still be true (biometrics is optional)")
	}
}

func TestHardeningStoreIsDelegationReady(t *testing.T) {
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

	userID := "@test:example.com"

	ready, err := store.IsDelegationReady(userID)
	if err != nil {
		t.Fatalf("Failed to check delegation ready for non-existent user: %v", err)
	}
	if ready {
		t.Error("IsDelegationReady should return false for non-existent user")
	}

	state := &UserHardeningState{
		UserID:            userID,
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	state.Recompute()

	if err := store.Put(state); err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	ready, err = store.IsDelegationReady(userID)
	if err != nil {
		t.Fatalf("Failed to check delegation ready: %v", err)
	}
	if !ready {
		t.Error("IsDelegationReady should return true when all mandatory steps are complete")
	}

	state2 := &UserHardeningState{
		UserID:           userID + "2",
		PasswordRotated:  true,
		BootstrapWiped:   true,
		DeviceVerified:   false,
		RecoveryBackedUp: true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	state2.Recompute()

	if err := store.Put(state2); err != nil {
		t.Fatalf("Failed to put state2: %v", err)
	}

	ready, err = store.IsDelegationReady(userID + "2")
	if err != nil {
		t.Fatalf("Failed to check delegation ready for user2: %v", err)
	}
	if ready {
		t.Error("IsDelegationReady should return false when mandatory steps are incomplete")
	}
}

func TestHardeningStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks1, err := keystore.New(keystore.Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore 1: %v", err)
	}

	if err := ks1.Open(); err != nil {
		t.Fatalf("Failed to open keystore 1: %v", err)
	}

	db1 := ks1.GetDB()
	store1 := NewKeystoreHardeningStore(db1)

	userID := "@test:example.com"

	state := &UserHardeningState{
		UserID:            userID,
		PasswordRotated:   true,
		BootstrapWiped:    true,
		DeviceVerified:    true,
		RecoveryBackedUp:  true,
		BiometricsEnabled: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	state.Recompute()

	if err := store1.Put(state); err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	if err := ks1.Close(); err != nil {
		t.Fatalf("Failed to close keystore 1: %v", err)
	}

	ks2, err := keystore.New(keystore.Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore 2: %v", err)
	}

	if err := ks2.Open(); err != nil {
		t.Fatalf("Failed to open keystore 2: %v", err)
	}
	defer ks2.Close()

	db2 := ks2.GetDB()
	store2 := NewKeystoreHardeningStore(db2)

	retrieved, err := store2.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state after reopen: %v", err)
	}

	if retrieved.UserID != state.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", retrieved.UserID, state.UserID)
	}
	if retrieved.PasswordRotated != state.PasswordRotated {
		t.Errorf("PasswordRotated mismatch: got %v, want %v", retrieved.PasswordRotated, state.PasswordRotated)
	}
	if retrieved.DelegationReady != state.DelegationReady {
		t.Errorf("DelegationReady mismatch: got %v, want %v", retrieved.DelegationReady, state.DelegationReady)
	}
}

func TestHardeningStoreDefaultValues(t *testing.T) {
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

	userID := "@existinguser:example.com"

	_, err = db.Exec(`
		INSERT INTO hardening_state (user_id, created_at, updated_at)
		VALUES (?, ?, ?)
	`, userID, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to insert default state: %v", err)
	}

	ready, err := store.IsDelegationReady(userID)
	if err != nil {
		t.Fatalf("Failed to check delegation ready: %v", err)
	}
	if ready {
		t.Error("Default user should have delegation_ready=false")
	}

	state, err := store.Get(userID)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if state.PasswordRotated {
		t.Error("Default PasswordRotated should be false")
	}
	if state.BootstrapWiped {
		t.Error("Default BootstrapWiped should be false")
	}
	if state.DeviceVerified {
		t.Error("Default DeviceVerified should be false")
	}
	if state.RecoveryBackedUp {
		t.Error("Default RecoveryBackedUp should be false")
	}
	if state.BiometricsEnabled {
		t.Error("Default BiometricsEnabled should be false")
	}
	if state.DelegationReady {
		t.Error("Default DelegationReady should be false")
	}
}
