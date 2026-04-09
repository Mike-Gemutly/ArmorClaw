package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionEncryptionRoundTrip(t *testing.T) {
	session := Session{
		ID:        "test-session-123",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("session-cookie-data"),
	}

	passphrase := "test-passphrase-123"

	encrypted, err := EncryptSession(session, passphrase)
	if err != nil {
		t.Fatalf("EncryptSession() failed: %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("Expected encrypted data to not be empty")
	}

	decrypted, err := DecryptSession(encrypted, passphrase)
	if err != nil {
		t.Fatalf("DecryptSession() failed: %v", err)
	}

	if decrypted.ID != session.ID {
		t.Errorf("Expected decrypted ID to match, got %s want %s", decrypted.ID, session.ID)
	}

	if decrypted.UserAgent != session.UserAgent {
		t.Errorf("Expected decrypted UserAgent to match, got %s want %s", decrypted.UserAgent, session.UserAgent)
	}

	if string(decrypted.Cookies) != string(session.Cookies) {
		t.Error("Expected decrypted Cookies to match")
	}
}

func TestSessionFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "test-session.enc")

	session := Session{
		ID:        "test-session-perms",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("cookie-data"),
	}

	passphrase := "test-passphrase"

	err := SaveSession(sessionPath, session, passphrase)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	info, err := os.Stat(sessionPath)
	if err != nil {
		t.Fatalf("Failed to stat session file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Expected file permissions 0600, got %04o", mode)
	}
}

func TestSessionDecryptionWithWrongPassphrase(t *testing.T) {
	session := Session{
		ID:        "test-session-wrong-pass",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("cookie-data"),
	}

	correctPassphrase := "correct-passphrase"
	wrongPassphrase := "wrong-passphrase"

	encrypted, err := EncryptSession(session, correctPassphrase)
	if err != nil {
		t.Fatalf("EncryptSession() failed: %v", err)
	}

	_, err = DecryptSession(encrypted, wrongPassphrase)
	if err == nil {
		t.Error("Expected decryption to fail with wrong passphrase")
	}
}

func TestLoadSession(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "test-load-session.enc")

	originalSession := Session{
		ID:        "test-load-session",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("load-cookie-data"),
	}

	passphrase := "test-passphrase"

	err := SaveSession(sessionPath, originalSession, passphrase)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	loadedSession, err := LoadSession(sessionPath, passphrase)
	if err != nil {
		t.Fatalf("LoadSession() failed: %v", err)
	}

	if loadedSession.ID != originalSession.ID {
		t.Errorf("Expected loaded ID to match, got %s want %s", loadedSession.ID, originalSession.ID)
	}

	if loadedSession.UserAgent != originalSession.UserAgent {
		t.Errorf("Expected loaded UserAgent to match, got %s want %s", loadedSession.UserAgent, originalSession.UserAgent)
	}

	if string(loadedSession.Cookies) != string(originalSession.Cookies) {
		t.Error("Expected loaded Cookies to match")
	}
}
