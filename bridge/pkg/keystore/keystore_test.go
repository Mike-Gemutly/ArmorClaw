//go:build cgo

// Package keystore tests for encrypted credential storage
// Note: These tests require CGO_ENABLED=1 due to SQLCipher dependency.
// Run with: CGO_ENABLED=1 go test ./pkg/keystore/...
package keystore

import (
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// TestKeystoreEncryption tests the XChaCha20-Poly1305 encryption/decryption
func TestKeystoreEncryption(t *testing.T) {
	// Create a temporary database for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test master key
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	// Initialize keystore
	ks, err := New(Config{
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

	// Test encryption/decryption
	testData := []byte("This is a secret API key that should never be exposed")

	encrypted, nonce, err := ks.encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("Encrypted data is empty")
	}

	if len(nonce) != 24 {
		t.Fatalf("Nonce has wrong length: got %d, want 24", len(nonce))
	}

	decrypted, err := ks.decrypt(encrypted, nonce)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data mismatch:\ngot:  %s\nwant: %s", string(decrypted), string(testData))
	}

	// Test that wrong nonce fails decryption
	wrongNonce := make([]byte, 24)
	_, err = ks.decrypt(encrypted, wrongNonce)
	if err == nil {
		t.Error("Expected error when decrypting with wrong nonce")
	}
}

// TestStoreAndRetrieve tests storing and retrieving credentials
func TestStoreAndRetrieve(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
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

	// Create a test credential
	cred := Credential{
		ID:          "test-credential-1",
		Provider:    ProviderOpenAI,
		Token:       "sk-test-secret-key-12345",
		DisplayName: "Test OpenAI Key",
		CreatedAt:   time.Now().Unix(),
		Tags:        []string{"test", "development"},
	}

	// Store the credential
	if err := ks.Store(cred); err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Retrieve the credential
	retrieved, err := ks.Retrieve(cred.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	// Verify the credential
	if retrieved.ID != cred.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, cred.ID)
	}

	if retrieved.Provider != cred.Provider {
		t.Errorf("Provider mismatch: got %s, want %s", retrieved.Provider, cred.Provider)
	}

	if retrieved.Token != cred.Token {
		t.Errorf("Token mismatch: got %s, want %s", retrieved.Token, cred.Token)
	}

	if retrieved.DisplayName != cred.DisplayName {
		t.Errorf("DisplayName mismatch: got %s, want %s", retrieved.DisplayName, cred.DisplayName)
	}
}

// TestListKeys tests listing stored keys
func TestListKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
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

	// Store multiple credentials
	credentials := []Credential{
		{
			ID:          "openai-key-1",
			Provider:    ProviderOpenAI,
			Token:       "sk-test-openai-1",
			DisplayName: "OpenAI Key 1",
			CreatedAt:   time.Now().Unix(),
		},
		{
			ID:          "anthropic-key-1",
			Provider:    ProviderAnthropic,
			Token:       "sk-ant-test-anthropic-1",
			DisplayName: "Anthropic Key 1",
			CreatedAt:   time.Now().Unix(),
		},
		{
			ID:          "openai-key-2",
			Provider:    ProviderOpenAI,
			Token:       "sk-test-openai-2",
			DisplayName: "OpenAI Key 2",
			CreatedAt:   time.Now().Unix(),
		},
	}

	for _, cred := range credentials {
		if err := ks.Store(cred); err != nil {
			t.Fatalf("Failed to store credential %s: %v", cred.ID, err)
		}
	}

	// List all keys
	allKeys, err := ks.List("")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(allKeys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(allKeys))
	}

	// List only OpenAI keys
	openaiKeys, err := ks.List(ProviderOpenAI)
	if err != nil {
		t.Fatalf("Failed to list OpenAI keys: %v", err)
	}

	if len(openaiKeys) != 2 {
		t.Errorf("Expected 2 OpenAI keys, got %d", len(openaiKeys))
	}
}

// TestDelete tests credential deletion
func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}

	// Store a credential
	cred := Credential{
		ID:          "to-delete",
		Provider:    ProviderOpenAI,
		Token:       "sk-test-delete",
		DisplayName: "Key to Delete",
		CreatedAt:   time.Now().Unix(),
	}

	if err := ks.Store(cred); err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Verify it exists
	_, err = ks.Retrieve(cred.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve credential before delete: %v", err)
	}

	// Delete it
	if err := ks.Delete(cred.ID); err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	// Verify it's gone
	_, err = ks.Retrieve(cred.ID)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}

	ks.Close()
}

// TestExpiredKey tests expired key rejection
func TestExpiredKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
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

	// Create an expired credential
	cred := Credential{
		ID:        "expired-key",
		Provider:  ProviderOpenAI,
		Token:     "sk-test-expired",
		CreatedAt: time.Now().Add(-24 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	if err := ks.Store(cred); err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Attempt to retrieve expired key
	_, err = ks.Retrieve(cred.ID)
	if err != ErrKeyExpired {
		t.Errorf("Expected ErrKeyExpired, got %v", err)
	}
}

// TestHardeningTableCreated verifies the hardening_state table exists with correct schema
func TestHardeningTableCreated(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
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

	// Verify table exists using PRAGMA table_info
	rows, err := db.Query("PRAGMA table_info(hardening_state)")
	if err != nil {
		t.Fatalf("Failed to query table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]string{
		"user_id":            "TEXT",
		"password_rotated":   "INTEGER",
		"bootstrap_wiped":    "INTEGER",
		"device_verified":    "INTEGER",
		"recovery_backed_up": "INTEGER",
		"biometrics_enabled": "INTEGER",
		"delegation_ready":   "INTEGER",
		"created_at":         "INTEGER",
		"updated_at":         "INTEGER",
	}

	foundColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}

		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		foundColumns[name] = true

		expectedType, ok := expectedColumns[name]
		if !ok {
			t.Errorf("Unexpected column found: %s", name)
			continue
		}

		if ctype != expectedType {
			t.Errorf("Column %s has wrong type: got %s, want %s", name, ctype, expectedType)
		}

		// Verify primary key
		if name == "user_id" && pk != 1 {
			t.Error("user_id should be PRIMARY KEY")
		}

		// Verify NOT NULL constraints
		if (name == "created_at" || name == "updated_at") && notnull != 1 {
			t.Errorf("%s should be NOT NULL", name)
		}
	}

	// Verify all expected columns are present
	for col := range expectedColumns {
		if !foundColumns[col] {
			t.Errorf("Expected column not found: %s", col)
		}
	}
}

// TestDefaultHardeningValues verifies new users have all hardening flags defaulted to 0
func TestDefaultHardeningValues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	ks, err := New(Config{
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

	// Insert a new user with no explicit values (should use defaults)
	testUserID := "@test:example.com"
	now := time.Now().Unix()

	_, err = db.Exec(`
		INSERT INTO hardening_state (user_id, created_at, updated_at)
		VALUES (?, ?, ?)
	`, testUserID, now, now)
	if err != nil {
		t.Fatalf("Failed to insert hardening state: %v", err)
	}

	// Query the row
	row := db.QueryRow(`
		SELECT password_rotated, bootstrap_wiped, device_verified,
		       recovery_backed_up, biometrics_enabled, delegation_ready
		FROM hardening_state
		WHERE user_id = ?
	`, testUserID)

	var passwordRotated, bootstrapWiped, deviceVerified int
	var recoveryBackedUp, biometricsEnabled, delegationReady int

	err = row.Scan(
		&passwordRotated,
		&bootstrapWiped,
		&deviceVerified,
		&recoveryBackedUp,
		&biometricsEnabled,
		&delegationReady,
	)
	if err != nil {
		t.Fatalf("Failed to scan hardening state: %v", err)
	}

	// Verify all defaults are 0 (false)
	if passwordRotated != 0 {
		t.Errorf("password_rotated should default to 0, got %d", passwordRotated)
	}
	if bootstrapWiped != 0 {
		t.Errorf("bootstrap_wiped should default to 0, got %d", bootstrapWiped)
	}
	if deviceVerified != 0 {
		t.Errorf("device_verified should default to 0, got %d", deviceVerified)
	}
	if recoveryBackedUp != 0 {
		t.Errorf("recovery_backed_up should default to 0, got %d", recoveryBackedUp)
	}
	if biometricsEnabled != 0 {
		t.Errorf("biometrics_enabled should default to 0, got %d", biometricsEnabled)
	}
	if delegationReady != 0 {
		t.Errorf("delegation_ready should default to 0, got %d", delegationReady)
	}
}
