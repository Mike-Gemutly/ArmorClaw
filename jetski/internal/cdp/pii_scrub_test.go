package cdp

import (
	"regexp"
	"testing"
)

// scrubTestPatterns returns the same regex patterns used by the real PII scanner.
// These are compiled here for testability without depending on the security package.
func scrubTestPatterns() map[string]*regexp.Regexp {
	return map[string]*regexp.Regexp{
		"SSN":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		"CREDIT_CARD": regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
		"EMAIL":       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		"PASSWORD":    regexp.MustCompile(`(?i)(password|passwd|pwd)["\']?\s*[:=]\s*["\']?[^\s"']{8,}`),
	}
}

func TestPIIScrubTethered_SSN(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"123-45-6789"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if string(result) == string(input) {
		t.Fatal("ScrubPII should have modified the input in tethered mode")
	}
	if !contains(result, "[REDACTED_SSN]") {
		t.Errorf("Expected [REDACTED_SSN] in output, got: %s", string(result))
	}
	if contains(result, "123-45-6789") {
		t.Errorf("SSN should be redacted, got: %s", string(result))
	}
}

func TestPIIScrubTethered_CreditCard(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"4242-4242-4242-4242"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if !contains(result, "[REDACTED_CREDIT_CARD]") {
		t.Errorf("Expected [REDACTED_CREDIT_CARD] in output, got: %s", string(result))
	}
	if contains(result, "4242-4242-4242-4242") {
		t.Errorf("Credit card should be redacted, got: %s", string(result))
	}
}

func TestPIIScrubTethered_Email(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"user@example.com"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if !contains(result, "[REDACTED_EMAIL]") {
		t.Errorf("Expected [REDACTED_EMAIL] in output, got: %s", string(result))
	}
	if contains(result, "user@example.com") {
		t.Errorf("Email should be redacted, got: %s", string(result))
	}
}

func TestPIIScrubTethered_Password(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"password: secretpass123"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if !contains(result, "[REDACTED_PASSWORD]") {
		t.Errorf("Expected [REDACTED_PASSWORD] in output, got: %s", string(result))
	}
	if contains(result, "secretpass123") {
		t.Errorf("Password value should be redacted, got: %s", string(result))
	}
}

func TestPIIScrubTethered_MultiplePII(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"SSN: 123-45-6789, email: user@example.com"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if !contains(result, "[REDACTED_SSN]") {
		t.Errorf("Expected [REDACTED_SSN] in output, got: %s", string(result))
	}
	if !contains(result, "[REDACTED_EMAIL]") {
		t.Errorf("Expected [REDACTED_EMAIL] in output, got: %s", string(result))
	}
	if contains(result, "123-45-6789") {
		t.Errorf("SSN should be redacted, got: %s", string(result))
	}
	if contains(result, "user@example.com") {
		t.Errorf("Email should be redacted, got: %s", string(result))
	}
}

func TestPIIScrubFreeRide_NoChange(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"123-45-6789"}}`)
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, false) // false = free-ride

	result := proxy.ScrubPII(input)

	if string(result) != string(input) {
		t.Errorf("Free-ride mode should not modify data, got: %s", string(result))
	}
}

func TestPIIScrubPreservesNonPII(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"hello world 12345"}}`)
	patterns := scrubTestPatterns()
	result := ScrubPII(input, patterns)

	if string(result) != string(input) {
		t.Errorf("Non-PII data should be unchanged, got: %s", string(result))
	}
}

func TestPIIScrubTethered_ProxyMethod(t *testing.T) {
	input := []byte(`{"method":"Input.insertText","params":{"text":"user@example.com"}}`)
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, true) // true = tethered

	result := proxy.ScrubPII(input)

	if !contains(result, "[REDACTED_EMAIL]") {
		t.Errorf("Tethered mode proxy should redact email, got: %s", string(result))
	}
}

// helper: bytes.Contains on []byte
func contains(data []byte, substr string) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		if string(data[i:i+len(substr)]) == substr {
			return true
		}
	}
	return false
}
