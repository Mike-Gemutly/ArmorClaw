package secretary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupRemovesDirectoryAndContents(t *testing.T) {
	dir := t.TempDir()

	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := cleanupStateDir(dir); err != nil {
		t.Fatalf("cleanupStateDir: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected directory to be removed, stat returned: %v", err)
	}
}

func TestStateDirExistsReturnsFalseAfterCleanup(t *testing.T) {
	dir := t.TempDir()

	eventsFile := filepath.Join(dir, "_events.jsonl")
	if err := os.WriteFile(eventsFile, []byte("[]\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !stateDirExists(dir) {
		t.Fatal("expected stateDirExists to return true before cleanup")
	}

	if err := cleanupStateDir(dir); err != nil {
		t.Fatalf("cleanupStateDir: %v", err)
	}

	if stateDirExists(dir) {
		t.Fatal("expected stateDirExists to return false after cleanup")
	}
}

func TestCleanupEmptyStringReturnsNil(t *testing.T) {
	if err := cleanupStateDir(""); err != nil {
		t.Fatalf("expected nil for empty string, got: %v", err)
	}
}

func TestCleanupNonexistentPathReturnsNil(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does_not_exist_anywhere")
	if err := cleanupStateDir(dir); err != nil {
		t.Fatalf("expected nil for nonexistent path, got: %v", err)
	}
}
