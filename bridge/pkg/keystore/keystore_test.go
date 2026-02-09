// Package keystore tests for encrypted credential storage
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
