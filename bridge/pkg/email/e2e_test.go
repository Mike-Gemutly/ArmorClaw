package email

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armorclaw/bridge/pkg/pii"
)

func TestE2E_PipelineIngestToEvent(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalFSEmailStorage(tmpDir)

	rawEmail := []byte("From: sender@example.com\r\nTo: agent@example.com\r\nSubject: Test Subject\r\nContent-Type: text/plain\r\n\r\nHello world with SSN 123-45-6789")

	parsed, err := ParseMIME(rawEmail)
	if err != nil {
		t.Fatalf("ParseMIME failed: %v", err)
	}

	if parsed.From != "sender@example.com" {
		t.Errorf("From = %q, want sender@example.com", parsed.From)
	}
	if parsed.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want Test Subject", parsed.Subject)
	}
	if parsed.BodyText == "" {
		t.Error("BodyText is empty")
	}

	emailID := "e2e_test_001"
	if err := storage.StoreEmail(emailID, rawEmail); err != nil {
		t.Fatalf("StoreEmail failed: %v", err)
	}

	emailPath := filepath.Join(tmpDir, "emails", emailID, "raw.eml")
	if _, err := os.Stat(emailPath); os.IsNotExist(err) {
		t.Errorf("stored email not found at %s", emailPath)
	}

	masker := pii.NewMasker()
	masked, fields := masker.MaskPII(parsed.BodyText)
	if len(fields) == 0 {
		t.Error("MaskPII found no PII fields in body containing SSN")
	}
	if masked == parsed.BodyText {
		t.Error("MaskPII did not mask the body")
	}
}

func TestE2E_OutboundFlow(t *testing.T) {
	masker := pii.NewMasker()

	body := "Call me at 555-123-4567 about order 123-45-6789"
	masked, fields := masker.MaskPII(body)
	if len(fields) < 2 {
		t.Errorf("expected >= 2 PII fields, got %d", len(fields))
	}

	resolutions := make(map[string]string)
	for _, f := range fields {
		resolutions[f.Placeholder] = f.Original
	}

	resolved := masker.ResolvePlaceholders(masked, resolutions)
	if resolved != body {
		t.Errorf("ResolvePlaceholders mismatch\ngot:  %q\nwant: %q", resolved, body)
	}
}

func TestE2E_StorageAttachmentRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalFSEmailStorage(tmpDir)

	content := []byte("attachment content here")
	fileID, err := storage.StoreAttachment("email_001", "report.pdf", content)
	if err != nil {
		t.Fatalf("StoreAttachment failed: %v", err)
	}
	if fileID == "" {
		t.Error("StoreAttachment returned empty fileID")
	}

	retrieved, err := storage.GetAttachment(fileID)
	if err != nil {
		t.Fatalf("GetAttachment failed: %v", err)
	}
	if string(retrieved) != string(content) {
		t.Errorf("GetAttachment content mismatch")
	}
}

func TestE2E_AuditLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &EmailAuditLogger{logDir: tmpDir}

	logger.LogEmailReceived("test_001", "from@test.com", "to@test.com", 2, 3)
	logger.LogEmailSent("test_001", "to@test.com", "gmail", "msg_123", true)
	logger.LogApprovalRequested("test_001", "approval_001", 2)
	logger.LogApprovalDecision("test_001", "approval_001", "approved")
	logger.LogPIIMasking("test_001", 3, []string{"ssn", "phone"})
	logger.LogYARAScan("test_001", "attachment.exe", false)
}
