package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalFSEmailStorage_DefaultDir(t *testing.T) {
	s := NewLocalFSEmailStorage("")
	if s.baseDir != defaultStorageDir {
		t.Errorf("baseDir = %q, want %q", s.baseDir, defaultStorageDir)
	}
}

func TestNewLocalFSEmailStorage_CustomDir(t *testing.T) {
	s := NewLocalFSEmailStorage("/tmp/email-test")
	if s.baseDir != "/tmp/email-test" {
		t.Errorf("baseDir = %q, want /tmp/email-test", s.baseDir)
	}
}

func TestLocalFS_StoreAndRetrieveEmail(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	emailData := []byte("From: test@test.com\r\nSubject: Hi\r\n\r\nBody")
	err := s.StoreEmail("email-001", emailData)
	if err != nil {
		t.Fatalf("StoreEmail: %v", err)
	}

	rawPath := filepath.Join(baseDir, "emails", "email-001", "raw.eml")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("read stored email: %v", err)
	}
	if string(data) != string(emailData) {
		t.Errorf("stored data mismatch")
	}
}

func TestLocalFS_StoreEmail_DirPermissions(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	s.StoreEmail("email-002", []byte("test"))

	emailDir := filepath.Join(baseDir, "emails", "email-002")
	info, err := os.Stat(emailDir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("dir perm = %o, want 0700", info.Mode().Perm())
	}
}

func TestLocalFS_StoreEmail_FilePermissions(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	s.StoreEmail("email-003", []byte("test"))

	emlPath := filepath.Join(baseDir, "emails", "email-003", "raw.eml")
	info, err := os.Stat(emlPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestLocalFS_StoreAndRetrieveAttachment(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	content := []byte("attachment content here")
	fileID, err := s.StoreAttachment("email-010", "report.pdf", content)
	if err != nil {
		t.Fatalf("StoreAttachment: %v", err)
	}
	if fileID == "" {
		t.Fatal("fileID should not be empty")
	}

	retrieved, err := s.GetAttachment(fileID)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(retrieved) != string(content) {
		t.Errorf("retrieved content mismatch: got %q", string(retrieved))
	}
}

func TestLocalFS_GetAttachment_NotFound(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	_, err := s.GetAttachment("nonexistent-file-id")
	if err == nil {
		t.Fatal("expected error for missing attachment")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want not found", err)
	}
}

func TestLocalFS_StoreAttachment_SanitizesFilename(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	content := []byte("safe content")
	fileID, err := s.StoreAttachment("email-011", "../../../etc/passwd", content)
	if err != nil {
		t.Fatalf("StoreAttachment: %v", err)
	}

	attachDir := filepath.Join(baseDir, "attachments", "email-011")
	entries, err := os.ReadDir(attachDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, e := range entries {
		if e.Name() == fileID+".meta" {
			continue
		}
		if strings.Contains(e.Name(), "/") || strings.Contains(e.Name(), "\\") {
			t.Errorf("filename not sanitized: %q", e.Name())
		}
	}
	_ = fileID
}

func TestLocalFS_StoreAttachment_EmptyFilename(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	content := []byte("unnamed file")
	fileID, err := s.StoreAttachment("email-012", "", content)
	if err != nil {
		t.Fatalf("StoreAttachment: %v", err)
	}
	if fileID == "" {
		t.Fatal("fileID should be generated")
	}
}

func TestLocalFS_DeleteEmail(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	s.StoreEmail("email-020", []byte("to delete"))
	s.StoreAttachment("email-020", "file.txt", []byte("attachment"))

	err := s.DeleteEmail("email-020")
	if err != nil {
		t.Fatalf("DeleteEmail: %v", err)
	}

	emailDir := filepath.Join(baseDir, "emails", "email-020")
	if _, err := os.Stat(emailDir); !os.IsNotExist(err) {
		t.Error("email dir should be deleted")
	}
}

func TestLocalFS_DeleteEmail_Nonexistent(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	err := s.DeleteEmail("nonexistent")
	if err != nil {
		t.Fatalf("deleting nonexistent should not error: %v", err)
	}
}

func TestLocalFS_StoreRaw(t *testing.T) {
	baseDir := t.TempDir()
	s := NewLocalFSEmailStorage(baseDir)

	err := s.StoreRaw("email-030", strings.NewReader("raw email content"))
	if err != nil {
		t.Fatalf("StoreRaw: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "emails", "email-030", "raw.eml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "raw email content" {
		t.Errorf("stored data = %q", string(data))
	}
}

func TestSanitizeFilename_PathSeparators(t *testing.T) {
	tests := []struct {
		input    string
		hasSlash bool
	}{
		{"foo/bar.txt", false},
		{"foo\\bar.txt", false},
		{"normal.txt", false},
	}
	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if strings.Contains(result, "/") || strings.Contains(result, "\\") {
			t.Errorf("sanitizeFilename(%q) = %q, contains path separator", tt.input, result)
		}
	}
}

func TestSanitizeFilename_LengthLimit(t *testing.T) {
	long := strings.Repeat("a", 300)
	result := sanitizeFilename(long)
	if len(result) > 255 {
		t.Errorf("sanitizeFilename result length = %d, want <= 255", len(result))
	}
}
