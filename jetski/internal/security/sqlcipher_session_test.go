package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func skipNoCGO(t *testing.T) {
	if runtime.Compiler != "gc" {
		t.Skip("CGO not available: compiler is not gc")
	}
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("CGO explicitly disabled")
	}
}

func TestSQLCipherNewStore(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	saltPath := dbPath + ".salt"
	if _, err := os.Stat(saltPath); os.IsNotExist(err) {
		t.Fatal("Expected salt file to be created")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Expected database file to be created")
	}
}

func TestSQLCipherCreateAndGetSession(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	session := Session{
		ID:        "sess-123",
		UserAgent: "Mozilla/5.0 (Test)",
		Cookies:   []byte("cookie-data-here"),
		ExpiresAt: 9999999999,
	}

	err = store.CreateSession(session)
	if err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	got, err := store.GetSession("sess-123")
	if err != nil {
		t.Fatalf("GetSession() failed: %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, session.ID)
	}
	if got.UserAgent != session.UserAgent {
		t.Errorf("UserAgent mismatch: got %q, want %q", got.UserAgent, session.UserAgent)
	}
	if string(got.Cookies) != string(session.Cookies) {
		t.Errorf("Cookies mismatch: got %q, want %q", got.Cookies, session.Cookies)
	}
	if got.ExpiresAt != session.ExpiresAt {
		t.Errorf("ExpiresAt mismatch: got %d, want %d", got.ExpiresAt, session.ExpiresAt)
	}
}

func TestSQLCipherGetSessionNotFound(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	_, err = store.GetSession("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent session")
	}
}

func TestSQLCipherListSessions(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	store.CreateSession(Session{ID: "sess-1", UserAgent: "UA1", Cookies: []byte("c1"), ExpiresAt: 100})
	store.CreateSession(Session{ID: "sess-2", UserAgent: "UA2", Cookies: []byte("c2"), ExpiresAt: 200})

	sessions, err = store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestSQLCipherCloseSession(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	store.CreateSession(Session{ID: "sess-del", UserAgent: "UA", Cookies: []byte("c"), ExpiresAt: 100})

	err = store.CloseSession("sess-del")
	if err != nil {
		t.Fatalf("CloseSession() failed: %v", err)
	}

	_, err = store.GetSession("sess-del")
	if err == nil {
		t.Fatal("Expected error after closing session")
	}
}

func TestSQLCipherWrongPassphrase(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	// Create store with correct passphrase
	store, err := NewSQLCipherSessionStore(dbPath, "correct-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() with correct passphrase failed: %v", err)
	}
	store.CreateSession(Session{ID: "sess-wp", UserAgent: "UA", Cookies: []byte("c"), ExpiresAt: 100})
	store.Close()

	_, err = NewSQLCipherSessionStore(dbPath, "wrong-passphrase")
	if err == nil {
		t.Fatal("Expected error when opening with wrong passphrase")
	}
}

func TestSQLCipherImplementsSessionStore(t *testing.T) {
	skipNoCGO(t)

	// Compile-time interface check
	var _ SessionStore = (*SQLCipherSessionStore)(nil)
}

func TestSQLCipherReopenStore(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	// Create and populate
	store1, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("First NewSQLCipherSessionStore() failed: %v", err)
	}

	original := Session{
		ID:        "sess-reopen",
		UserAgent: "Mozilla/5.0",
		Cookies:   []byte("persisted-cookies"),
		ExpiresAt: 1234567890,
	}
	store1.CreateSession(original)
	store1.Close()

	store2, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("Second NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store2.Close()

	got, err := store2.GetSession("sess-reopen")
	if err != nil {
		t.Fatalf("GetSession() after reopen failed: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID mismatch after reopen: got %q, want %q", got.ID, original.ID)
	}
	if string(got.Cookies) != string(original.Cookies) {
		t.Errorf("Cookies mismatch after reopen")
	}
}

func TestSQLCipherEmptyPassphrase(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	_, err := NewSQLCipherSessionStore(dbPath, "")
	if err == nil {
		t.Fatal("Expected error for empty passphrase")
	}
}

func TestSQLCipherDuplicateSession(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sessions.db")

	store, err := NewSQLCipherSessionStore(dbPath, "test-passphrase")
	if err != nil {
		t.Fatalf("NewSQLCipherSessionStore() failed: %v", err)
	}
	defer store.Close()

	session := Session{ID: "sess-dup", UserAgent: "UA", Cookies: []byte("c"), ExpiresAt: 100}
	err = store.CreateSession(session)
	if err != nil {
		t.Fatalf("First CreateSession() failed: %v", err)
	}

	err = store.CreateSession(session)
	if err == nil {
		t.Fatal("Expected error on duplicate session ID")
	}
}
