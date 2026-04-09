package security

import (
	"testing"
)

func TestNewPIIScanner(t *testing.T) {
	scanner := NewPIIScanner()
	if scanner == nil {
		t.Fatal("NewPIIScanner returned nil")
	}
	if len(scanner.patterns) != 4 {
		t.Errorf("Expected 4 patterns, got %d", len(scanner.patterns))
	}
}

func TestScanCDPMessage_SSN(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "My SSN is 123-45-6789",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	if findings[0].Type != PIITypeSSN {
		t.Errorf("Expected type %s, got %s", PIITypeSSN, findings[0].Type)
	}
}

func TestScanCDPMessage_CreditCard(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "Card: 4111-1111-1111-1111",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	if findings[0].Type != PIITypeCreditCard {
		t.Errorf("Expected type %s, got %s", PIITypeCreditCard, findings[0].Type)
	}
}

func TestScanCDPMessage_Email(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "Email: user@example.com",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	if findings[0].Type != PIITypeEmail {
		t.Errorf("Expected type %s, got %s", PIITypeEmail, findings[0].Type)
	}
}

func TestScanCDPMessage_Password(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "password=MySecret123",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	if findings[0].Type != PIITypePassword {
		t.Errorf("Expected type %s, got %s", PIITypePassword, findings[0].Type)
	}
}

func TestScanCDPMessage_PasswordCaseInsensitive(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "PASSWORD=MySecret123",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
}

func TestScanCDPMessage_MultiplePII(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "Email: user@example.com, SSN: 123-45-6789",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 2 {
		t.Fatalf("Expected 2 findings, got %d", len(findings))
	}
}

func TestScanCDPMessage_NonInputMethod(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "password=MySecret123",
	}

	findings := scanner.ScanCDPMessage("Page.navigate", params)

	if len(findings) != 0 {
		t.Fatalf("Expected 0 findings for non-Input method, got %d", len(findings))
	}
}

func TestScanCDPMessage_NoPII(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{
		"text": "This is just regular text",
	}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 0 {
		t.Fatalf("Expected 0 findings, got %d", len(findings))
	}
}

func TestScanCDPMessage_EmptyParams(t *testing.T) {
	scanner := NewPIIScanner()

	params := map[string]interface{}{}

	findings := scanner.ScanCDPMessage("Input.insertText", params)

	if len(findings) != 0 {
		t.Fatalf("Expected 0 findings, got %d", len(findings))
	}
}

func TestContainsPassword(t *testing.T) {
	scanner := NewPIIScanner()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Password field", "password=secret123", true},
		{"Passwd field", "passwd=secret123", true},
		{"Pwd field", "pwd=secret123", true},
		{"No password", "regular text", false},
		{"Short password", "pwd=short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.ContainsPassword(tt.text)
			if result != tt.expected {
				t.Errorf("ContainsPassword() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskPII(t *testing.T) {
	scanner := NewPIIScanner()

	text := "My SSN is 123-45-6789"
	masked := scanner.MaskPII(text)

	if masked == text {
		t.Error("Expected text to be masked")
	}

	if masked == text {
		t.Errorf("Expected SSN to be masked, got: %s", masked)
	}
}

func TestScanJSONMessage(t *testing.T) {
	scanner := NewPIIScanner()

	jsonStr := `{"method":"Input.insertText","params":{"text":"password=secret123"}}`

	findings, err := scanner.ScanJSONMessage(jsonStr)

	if err != nil {
		t.Fatalf("ScanJSONMessage returned error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
}

func TestScanJSONMessage_InvalidJSON(t *testing.T) {
	scanner := NewPIIScanner()

	jsonStr := `invalid json`

	_, err := scanner.ScanJSONMessage(jsonStr)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGetSeverity(t *testing.T) {
	scanner := NewPIIScanner()

	tests := []struct {
		piiType  PIIType
		expected string
	}{
		{PIITypePassword, "HIGH"},
		{PIITypeSSN, "HIGH"},
		{PIITypeCreditCard, "HIGH"},
		{PIITypeEmail, "MEDIUM"},
	}

	for _, tt := range tests {
		t.Run(string(tt.piiType), func(t *testing.T) {
			severity := scanner.getSeverity(tt.piiType)
			if severity != tt.expected {
				t.Errorf("getSeverity() = %s, want %s", severity, tt.expected)
			}
		})
	}
}

func TestSanitizeContext(t *testing.T) {
	scanner := NewPIIScanner()

	shortText := "short text"
	longText := string(make([]byte, 200))

	sanitizedShort := scanner.sanitizeContext(shortText)
	if sanitizedShort != shortText {
		t.Error("Short text should not be truncated")
	}

	sanitizedLong := scanner.sanitizeContext(longText)
	if len(sanitizedLong) > 103 {
		t.Error("Long text should be truncated")
	}
	if sanitizedLong[len(sanitizedLong)-3:] != "..." {
		t.Error("Truncated text should end with '...'")
	}
}
