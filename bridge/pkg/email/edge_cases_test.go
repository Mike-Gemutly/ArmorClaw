package email

import (
	"testing"

	"github.com/armorclaw/bridge/pkg/pii"
)

func TestEdge_EmptyEmail(t *testing.T) {
	_, err := ParseMIME([]byte{})
	if err == nil {
		t.Error("ParseMIME should fail on empty input")
	}
}

func TestEdge_NoSubject(t *testing.T) {
	raw := []byte("From: a@b.com\r\nTo: c@d.com\r\n\r\nBody text")
	parsed, err := ParseMIME(raw)
	if err != nil {
		t.Fatalf("ParseMIME failed: %v", err)
	}
	if parsed.Subject != "" {
		t.Errorf("Subject should be empty, got %q", parsed.Subject)
	}
}

func TestEdge_LargeEmail(t *testing.T) {
	body := make([]byte, 10*1024*1024)
	for i := range body {
		body[i] = 'A'
	}
	raw := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: Large\r\nContent-Type: text/plain\r\n\r\n")
	raw = append(raw, body...)
	parsed, err := ParseMIME(raw)
	if err != nil {
		t.Fatalf("ParseMIME failed on large email: %v", err)
	}
	if len(parsed.BodyText) < 1024*1024 {
		t.Errorf("BodyText too short: %d", len(parsed.BodyText))
	}
}

func TestEdge_MultipartAlternative(t *testing.T) {
	raw := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: Multipart\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=boundary123\r\n\r\n--boundary123\r\nContent-Type: text/plain\r\n\r\nPlain text body\r\n--boundary123\r\nContent-Type: text/html\r\n\r\n<html>HTML body</html>\r\n--boundary123--\r\n")
	parsed, err := ParseMIME(raw)
	if err != nil {
		t.Fatalf("ParseMIME multipart failed: %v", err)
	}
	if parsed.BodyText != "Plain text body" {
		t.Errorf("BodyText = %q, want plain text", parsed.BodyText)
	}
	if parsed.BodyHTML != "<html>HTML body</html>" {
		t.Errorf("BodyHTML = %q, want html", parsed.BodyHTML)
	}
}

func TestEdge_UnicodeSubject(t *testing.T) {
	raw := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: =?UTF-8?B?w6DDqcO2w63DvA==?=\r\n\r\nBody")
	parsed, err := ParseMIME(raw)
	if err != nil {
		t.Fatalf("ParseMIME failed: %v", err)
	}
	if parsed.Subject == "" {
		t.Error("Subject should not be empty for encoded unicode")
	}
}

func TestEdge_NoPII(t *testing.T) {
	body := "Hello, this is a normal email with no sensitive data."
	_, fields := NewTestMasker().MaskPII(body)
	if len(fields) != 0 {
		t.Errorf("expected 0 PII fields, got %d", len(fields))
	}
}

func TestEdge_MultipleSSN(t *testing.T) {
	body := "SSN1: 123-45-6789 SSN2: 987-65-4321 SSN3: 111-22-3333"
	_, fields := NewTestMasker().MaskPII(body)
	if len(fields) < 3 {
		t.Errorf("expected >= 3 SSN fields, got %d", len(fields))
	}
}

func TestSecurity_SqlInjectionInAddress(t *testing.T) {
	malicious := "'; DROP TABLE emails;--"
	storage := NewLocalFSEmailStorage(t.TempDir())
	err := storage.StoreEmail(malicious, []byte("test"))
	if err != nil {
		t.Fatalf("StoreEmail should handle malicious email ID: %v", err)
	}
}

func TestSecurity_PathTraversalAttachment(t *testing.T) {
	storage := NewLocalFSEmailStorage(t.TempDir())
	_, err := storage.StoreAttachment("email_001", "../../../etc/passwd", []byte("hacked"))
	if err != nil {
		t.Fatalf("StoreAttachment should sanitize path: %v", err)
	}
}

func TestSecurity_AuditHashing(t *testing.T) {
	logger := &EmailAuditLogger{logDir: t.TempDir()}
	logger.LogEmailReceived("test", "sensitive@example.com", "to@test.com", 0, 0)
}

func NewTestMasker() *pii.Masker {
	return pii.NewMasker()
}
