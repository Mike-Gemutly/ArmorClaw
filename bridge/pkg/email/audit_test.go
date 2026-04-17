package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashForAudit_LongValue(t *testing.T) {
	result := hashForAudit("user@example.com")
	if result == "user@example.com" {
		t.Error("value should be hashed, not returned as-is")
	}
	if !strings.HasPrefix(result, "us") {
		t.Errorf("expected prefix 'us', got %q", result[:2])
	}
	if !strings.HasSuffix(result, "om") {
		t.Errorf("expected suffix 'om', got %q", result[len(result)-2:])
	}
	if !strings.Contains(result, "***") {
		t.Error("expected *** mask in middle")
	}
}

func TestHashForAudit_ShortValue(t *testing.T) {
	result := hashForAudit("ab")
	if result != "***" {
		t.Errorf("short value should return ***, got %q", result)
	}
}

func TestHashForAudit_FiveChars(t *testing.T) {
	result := hashForAudit("abcde")
	if !strings.Contains(result, "***") {
		t.Error("5-char value should be masked")
	}
	if !strings.HasPrefix(result, "ab") {
		t.Errorf("expected prefix 'ab', got %q", result[:2])
	}
	if !strings.HasSuffix(result, "de") {
		t.Errorf("expected suffix 'de', got %q", result[len(result)-2:])
	}
}

func TestHashForAudit_EmptyValue(t *testing.T) {
	result := hashForAudit("")
	if result != "***" {
		t.Errorf("empty value should return ***, got %q", result)
	}
}

func TestNewEmailAuditLogger(t *testing.T) {
	logger := NewEmailAuditLogger()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.logDir == "" {
		t.Error("logDir should not be empty")
	}
}

func TestAuditLogger_LogEmailReceived(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogEmailReceived("email-123", "from@test.com", "to@test.com", 2, 3)

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_LogEmailSent(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogEmailSent("email-456", "recipient@test.com", "gmail", "msg-id-789", true)

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_LogApprovalRequested(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogApprovalRequested("email-123", "approval-456", 2)

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_LogApprovalDecision(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogApprovalDecision("email-123", "approval-456", "approved")

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_LogPIIMasking(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogPIIMasking("email-123", 3, []string{"ssn", "credit_card", "phone"})

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_LogYARAScan(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogYARAScan("email-123", "invoice.pdf", true)

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected audit log file to be created")
	}
}

func TestAuditLogger_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogEmailReceived("email-001", "a@b.com", "c@d.com", 0, 0)

	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected log file")
	}

	info, err := os.Stat(filepath.Join(tmpDir, files[0].Name()))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("log file perm = %o, want 0600", info.Mode().Perm())
	}
}
