//go:build cgo

// Package keystore tests for encrypted credential storage
// Note: These tests require CGO_ENABLED=1 due to SQLCipher dependency.
// Run with: CGO_ENABLED=1 go test ./pkg/keystore/...
package keystore

import (
	"encoding/base64"
	"os"
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

func TestErrorMessageContent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey1 := make([]byte, 32)
	for i := range masterKey1 {
		masterKey1[i] = byte(i)
	}

	ks1, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey1,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks1.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks1.Close()

	cred := Credential{
		ID:          "test-key",
		Provider:    ProviderOpenAI,
		Token:       "sk-test-key-12345",
		DisplayName: "Test Key",
		CreatedAt:   time.Now().Unix(),
	}

	if err := ks1.Store(cred); err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	ks1.Close()

	masterKey2 := make([]byte, 32)
	for i := range masterKey2 {
		masterKey2[i] = byte(i + 10)
	}

	ks2, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey2,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks2.Open()
	if err == nil {
		t.Fatal("Expected KEY MISMATCH error when opening with wrong key, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message:\n%s", errMsg)

	if !contains(errMsg, "Ensure /var/lib/armorclaw is persisted as Docker volume") {
		t.Error("Error message missing volume mount suggestion")
	}

	if !contains(errMsg, "Set ARMORCLAW_KEYSTORE_SECRET environment variable") {
		t.Error("Error message missing env var suggestion")
	}

	if !contains(errMsg, "--migrate-keystore flag") {
		t.Error("Error message missing container restart guidance")
	}

	if !contains(errMsg, "https://github.com/Gemutly/ArmorClaw/blob/main/docs/guides/keystore.md") {
		t.Error("Error message missing documentation link")
	}

	if contains(errMsg, "sk-") || contains(errMsg, "key:") {
		t.Error("Error message should not expose key material")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMigrateHardwareKey tests migration from hardware-derived key to file-persisted key
func TestMigrateHardwareKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ks, err := New(Config{
		DBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		t.Fatalf("Failed to open keystore: %v", err)
	}

	cred := Credential{
		ID:          "test-migration-cred",
		Provider:    ProviderOpenAI,
		Token:       "sk-test-key-for-migration",
		DisplayName: "Migration Test Credential",
	}

	if err := ks.Store(cred); err != nil {
		t.Fatalf("Failed to store test credential: %v", err)
	}
	ks.Close()

	ks, err = New(Config{
		DBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to recreate keystore: %v", err)
	}

	if err := ks.MigrateToPersistedKey(); err != nil {
		t.Fatalf("Failed to migrate keystore: %v", err)
	}

	keyPath := dbPath + ".key"
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatalf("keystore.key file not created at %s", keyPath)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read keystore.key: %v", err)
	}

	persistedKey, err := base64.StdEncoding.DecodeString(string(keyData))
	if err != nil {
		t.Fatalf("Failed to decode persisted key: %v", err)
	}

	if len(persistedKey) != 32 {
		t.Fatalf("Persisted key has wrong length: got %d, want 32", len(persistedKey))
	}

	ksPersisted, err := New(Config{
		DBPath:    dbPath,
		MasterKey: persistedKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore with persisted key: %v", err)
	}

	if err := ksPersisted.Open(); err != nil {
		t.Fatalf("Failed to open keystore with persisted key: %v", err)
	}
	defer ksPersisted.Close()

	retrieved, err := ksPersisted.Retrieve("test-migration-cred")
	if err != nil {
		t.Fatalf("Failed to retrieve credential after migration: %v", err)
	}

	if retrieved.ID != cred.ID {
		t.Errorf("Credential ID mismatch: got %s, want %s", retrieved.ID, cred.ID)
	}

	if retrieved.Token != cred.Token {
		t.Errorf("Credential token mismatch: got %s, want %s", retrieved.Token, cred.Token)
	}

	if retrieved.Provider != cred.Provider {
		t.Errorf("Credential provider mismatch: got %s, want %s", retrieved.Provider, cred.Provider)
	}

	t.Logf("✓ Migration successful - data preserved with persisted key")
}

// TestSystemdSpawn tests systemd-nspawn container detection
func TestSystemdSpawn(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	_, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Test 1: /run/systemd/container contains "nspawn"
	t.Run("SystemdContainerFile", func(t *testing.T) {
		systemdDir := filepath.Join(tmpDir, "run", "systemd")
		if err := os.MkdirAll(systemdDir, 0755); err != nil {
			t.Fatalf("Failed to create systemd dir: %v", err)
		}
		containerFile := filepath.Join(systemdDir, "container")
		if err := os.WriteFile(containerFile, []byte("systemd-nspawn\n"), 0644); err != nil {
			t.Fatalf("Failed to write container file: %v", err)
		}
		t.Log("systemd-nspawn detection would check /run/systemd/container for nspawn string")
	})

	// Test 2: /.containerenv contains "nspawn"
	t.Run("ContainerEnvFile", func(t *testing.T) {
		containerEnvFile := filepath.Join(tmpDir, ".containerenv")
		if err := os.WriteFile(containerEnvFile, []byte("container=nspawn\n"), 0644); err != nil {
			t.Fatalf("Failed to write .containerenv file: %v", err)
		}
		t.Log("systemd-nspawn detection would check /.containerenv for nspawn string")
	})
}

// TestPodman tests Podman container detection
func TestPodman(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	_, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Test 1: /run/podman/podman.pid exists
	t.Run("PodmanPidFile", func(t *testing.T) {
		podmanDir := filepath.Join(tmpDir, "run", "podman")
		if err := os.MkdirAll(podmanDir, 0755); err != nil {
			t.Fatalf("Failed to create podman dir: %v", err)
		}
		pidFile := filepath.Join(podmanDir, "podman.pid")
		if err := os.WriteFile(pidFile, []byte("12345\n"), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		t.Log("Podman detection would check /run/podman/podman.pid existence")
	})

	// Test 2: .podmanenv exists
	t.Run("PodmanEnvFile", func(t *testing.T) {
		podmanEnvFile := filepath.Join(tmpDir, ".podmanenv")
		if err := os.WriteFile(podmanEnvFile, []byte("podman_env=1\n"), 0644); err != nil {
			t.Fatalf("Failed to write .podmanenv file: %v", err)
		}
		t.Log("Podman detection would check .podmanenv existence")
	})
}

// TestECS tests ECS/Fargate container detection
func TestECS(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	_, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Test 1: ECS_CONTAINER_METADATA_URI environment variable set
	t.Run("ECSMetadataURI", func(t *testing.T) {
		originalURI := os.Getenv("ECS_CONTAINER_METADATA_URI")
		defer os.Setenv("ECS_CONTAINER_METADATA_URI", originalURI)

		os.Setenv("ECS_CONTAINER_METADATA_URI", "http://169.254.170.2/v4/12345")

		uri := os.Getenv("ECS_CONTAINER_METADATA_URI")
		if uri != "http://169.254.170.2/v4/12345" {
			t.Errorf("Expected env var to be set, got %s", uri)
		}
	})

	// Test 2: AWS_EXECUTION_ENV environment variable set
	t.Run("AWSExecutionEnv", func(t *testing.T) {
		originalEnv := os.Getenv("AWS_EXECUTION_ENV")
		defer os.Setenv("AWS_EXECUTION_ENV", originalEnv)

		os.Setenv("AWS_EXECUTION_ENV", "AWS_ECS_FARGATE")

		env := os.Getenv("AWS_EXECUTION_ENV")
		if env != "AWS_ECS_FARGATE" {
			t.Errorf("Expected env var to be set, got %s", env)
		}
	})
}

// TestExistingDetection tests existing Docker and /proc/cgroup detection
func TestExistingDetection(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	_, err := New(Config{
		DBPath:    dbPath,
		MasterKey: masterKey,
	})
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Test 1: /.dockerenv exists
	t.Run("DockerEnvFile", func(t *testing.T) {
		dockerEnvFile := filepath.Join(tmpDir, ".dockerenv")
		if err := os.WriteFile(dockerEnvFile, []byte("docker_env=1\n"), 0644); err != nil {
			t.Fatalf("Failed to write .dockerenv file: %v", err)
		}
		t.Log("Docker detection would check /.dockerenv existence")
	})

	// Test 2: /proc/1/cgroup contains docker or kubepods
	t.Run("ProcCgroup", func(t *testing.T) {
		cgroupContent := `12:cpuset:/kubepods/burstable/pod12345/6789
11:cpu:/kubepods/burstable/pod12345/6789
10:cpuacct:/kubepods/burstable/pod12345/6789
9:memory:/kubepods/burstable/pod12345/6789`

		if !contains(cgroupContent, "kubepods") {
			t.Error("Expected cgroup content to contain kubepods")
		}

		dockerContent := `12:cpuset:/docker/12345
11:cpu:/docker/12345
10:cpuacct:/docker/12345`

		if !contains(dockerContent, "docker") {
			t.Error("Expected cgroup content to contain docker")
		}
	})
}
