package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveSession(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "test-session.enc")

	session := Session{
		ID:        "test-session-save",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("cookie-data"),
	}

	passphrase := "test-passphrase"

	err := SaveSession(sessionPath, session, passphrase)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	dbPath := sessionPath[:len(sessionPath)-4] + ".db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Expected database file to be created")
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
		t.Errorf("Expected loaded ID to match, got %q want %q", loadedSession.ID, originalSession.ID)
	}

	if loadedSession.UserAgent != originalSession.UserAgent {
		t.Errorf("Expected loaded UserAgent to match, got %q want %q", loadedSession.UserAgent, originalSession.UserAgent)
	}

	if string(loadedSession.Cookies) != string(originalSession.Cookies) {
		t.Error("Expected loaded Cookies to match")
	}
}

func TestLoadSessionWrongPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "test-wrong-pass.enc")

	session := Session{
		ID:        "test-wrong-pass",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("cookie-data"),
	}

	err := SaveSession(sessionPath, session, "correct-passphrase")
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	_, err = LoadSession(sessionPath, "wrong-passphrase")
	if err == nil {
		t.Fatal("Expected error when loading with wrong passphrase")
	}
}
