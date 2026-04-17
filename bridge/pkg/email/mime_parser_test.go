package email

import (
	"strings"
	"testing"
)

func TestParseMIME_SimplePlainText(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: Hello\r\nContent-Type: text/plain\r\n\r\nHello world"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if parsed.From != "sender@test.com" {
		t.Errorf("From = %q", parsed.From)
	}
	if parsed.Subject != "Hello" {
		t.Errorf("Subject = %q", parsed.Subject)
	}
	if !strings.Contains(parsed.BodyText, "Hello world") {
		t.Errorf("BodyText = %q", parsed.BodyText)
	}
}

func TestParseMIME_MultipartMixed(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: Multipart\r\nContent-Type: multipart/mixed; boundary=\"BOUNDARY\"\r\n\r\n--BOUNDARY\r\nContent-Type: text/plain\r\n\r\nBody text\r\n--BOUNDARY\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"doc.pdf\"\r\n\r\nfake pdf content\r\n--BOUNDARY--\r\n"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if !strings.Contains(parsed.BodyText, "Body text") {
		t.Errorf("BodyText = %q, want body text", parsed.BodyText)
	}
	if len(parsed.Attachments) != 1 {
		t.Fatalf("Attachments count = %d, want 1", len(parsed.Attachments))
	}
	if parsed.Attachments[0].Filename != "doc.pdf" {
		t.Errorf("Filename = %q, want doc.pdf", parsed.Attachments[0].Filename)
	}
	if parsed.Attachments[0].ContentType != "application/pdf" {
		t.Errorf("ContentType = %q", parsed.Attachments[0].ContentType)
	}
}

func TestParseMIME_CCAddresses(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nCc: cc1@test.com, cc2@test.com\r\nSubject: CC Test\r\n\r\nBody"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if len(parsed.To) != 3 {
		t.Errorf("To count = %d, want 3 (To + Cc)", len(parsed.To))
	}
}

func TestParseMIME_EmptyInput(t *testing.T) {
	_, err := ParseMIME([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseMIME_InvalidHeaders(t *testing.T) {
	_, err := ParseMIME([]byte("garbage\r\nmore garbage"))
	if err != nil {
		t.Logf("ParseMIME on garbage returned: %v (acceptable)", err)
	}
}

func TestParseMIME_HTMLBody(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: HTML\r\nContent-Type: text/html\r\n\r\n<b>bold</b>"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if !strings.Contains(parsed.BodyHTML, "<b>bold</b>") {
		t.Errorf("BodyHTML = %q", parsed.BodyHTML)
	}
}

func TestParseMIME_MultipartHTMLAndText(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: Alt\r\nContent-Type: multipart/alternative; boundary=\"ALT\"\r\n\r\n--ALT\r\nContent-Type: text/plain\r\n\r\nPlain text body\r\n--ALT\r\nContent-Type: text/html\r\n\r\n<p>HTML body</p>\r\n--ALT--\r\n"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if !strings.Contains(parsed.BodyText, "Plain text body") {
		t.Errorf("BodyText = %q", parsed.BodyText)
	}
	if !strings.Contains(parsed.BodyHTML, "HTML body") {
		t.Errorf("BodyHTML = %q", parsed.BodyHTML)
	}
}

func TestParseMIME_NestedMultipart(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: Nested\r\nContent-Type: multipart/mixed; boundary=\"OUTER\"\r\n\r\n--OUTER\r\nContent-Type: multipart/alternative; boundary=\"INNER\"\r\n\r\n--INNER\r\nContent-Type: text/plain\r\n\r\nInner text\r\n--INNER--\r\n--OUTER\r\nContent-Type: application/zip\r\nContent-Disposition: attachment; filename=\"data.zip\"\r\n\r\nzipcontent\r\n--OUTER--\r\n"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if !strings.Contains(parsed.BodyText, "Inner text") {
		t.Errorf("BodyText = %q", parsed.BodyText)
	}
	attachCount := len(parsed.Attachments)
	if attachCount < 1 {
		t.Errorf("expected at least 1 attachment, got %d", attachCount)
	}
}

func TestParseMIME_EncodedSubject(t *testing.T) {
	raw := "From: sender@test.com\r\nTo: recv@test.com\r\nSubject: =?UTF-8?B?SGVsbG8gV29ybGQ=?=\r\n\r\nBody"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if parsed.Subject != "Hello World" {
		t.Errorf("Subject = %q, want Hello World", parsed.Subject)
	}
}

func TestParseMIME_NamedAddressFrom(t *testing.T) {
	raw := "From: John Doe <john@test.com>\r\nTo: recv@test.com\r\nSubject: Test\r\n\r\nBody"
	parsed, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if !strings.Contains(parsed.From, "john@test.com") {
		t.Errorf("From = %q, want john@test.com", parsed.From)
	}
}

func TestDecodeHeader_PlainString(t *testing.T) {
	result := decodeHeader("plain text")
	if result != "plain text" {
		t.Errorf("decodeHeader = %q, want plain text", result)
	}
}

func TestExtractFilename_DispositionFirst(t *testing.T) {
	header := map[string][]string{}
	dispParams := map[string]string{"filename": "report.pdf"}
	ctParams := map[string]string{"name": "other.pdf"}
	result := extractFilename(header, dispParams, ctParams)
	if result != "report.pdf" {
		t.Errorf("extractFilename = %q, want report.pdf", result)
	}
}

func TestExtractFilename_ContentTypeName(t *testing.T) {
	header := map[string][]string{}
	dispParams := map[string]string{}
	ctParams := map[string]string{"name": "image.png"}
	result := extractFilename(header, dispParams, ctParams)
	if result != "image.png" {
		t.Errorf("extractFilename = %q, want image.png", result)
	}
}

func TestExtractFilename_Empty(t *testing.T) {
	result := extractFilename(map[string][]string{}, map[string]string{}, map[string]string{})
	if result != "" {
		t.Errorf("extractFilename = %q, want empty", result)
	}
}
