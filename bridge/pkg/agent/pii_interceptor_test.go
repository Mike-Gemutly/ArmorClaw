// Package agent provides PII interceptor middleware for detecting and replacing vault patterns
package agent

import (
	"testing"
)

func TestPIIInterceptor_SinglePattern(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "Please use {{VAULT:abc123}} to login"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedModified := "Please use [REDACTED:abc123] to login"
	if modified != expectedModified {
		t.Errorf("Expected modified prompt to be %q, got %q", expectedModified, modified)
	}

	if mapping.Count() == 0 {
		t.Error("Expected mapping to have entries")
	}

	actualValue, exists := mapping.Get("[REDACTED:abc123]")
	if !exists {
		t.Error("Expected mapping to contain key [REDACTED:abc123]")
	}

	if actualValue != "{{VAULT:abc123}}" {
		t.Errorf("Expected mapping value to be {{VAULT:abc123}}, got %q", actualValue)
	}
}

func TestPIIInterceptor_MultiplePatterns(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "Use {{VAULT:abc123}} for email and {{VAULT:def456}} for password"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedModified := "Use [REDACTED:abc123] for email and [REDACTED:def456] for password"
	if modified != expectedModified {
		t.Errorf("Expected modified prompt to be %q, got %q", expectedModified, modified)
	}

	if mapping.Count() != 2 {
		t.Errorf("Expected mapping to have 2 entries, got %d", mapping.Count())
	}

	val1, exists1 := mapping.Get("[REDACTED:abc123]")
	if !exists1 || val1 != "{{VAULT:abc123}}" {
		t.Errorf("Expected [REDACTED:abc123] to map to {{VAULT:abc123}}, got %q (exists: %v)", val1, exists1)
	}

	val2, exists2 := mapping.Get("[REDACTED:def456]")
	if !exists2 || val2 != "{{VAULT:def456}}" {
		t.Errorf("Expected [REDACTED:def456] to map to {{VAULT:def456}}, got %q (exists: %v)", val2, exists2)
	}
}

func TestPIIInterceptor_NoMatch(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "This prompt has no vault patterns"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if modified != prompt {
		t.Errorf("Expected prompt to remain unchanged, got %q", modified)
	}

	if mapping.Count() != 0 {
		t.Errorf("Expected mapping to be empty, got %d entries", mapping.Count())
	}
}

func TestPIIInterceptor_DuplicateHash(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "Use {{VAULT:abc123}} twice: {{VAULT:abc123}}"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedModified := "Use [REDACTED:abc123] twice: [REDACTED:abc123]"
	if modified != expectedModified {
		t.Errorf("Expected modified prompt to be %q, got %q", expectedModified, modified)
	}

	if mapping.Count() != 1 {
		t.Errorf("Expected mapping to have 1 unique entry, got %d", mapping.Count())
	}
}

func TestPIIInterceptor_InvalidPattern(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "Use {{VAULT:invalid}} for something"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if modified != prompt {
		t.Errorf("Expected prompt to remain unchanged for invalid pattern, got %q", modified)
	}

	if mapping.Count() != 0 {
		t.Errorf("Expected mapping to be empty for invalid pattern, got %d entries", mapping.Count())
	}
}

func TestPIIInterceptor_LongHash(t *testing.T) {
	interceptor := NewPIIInterceptor()

	prompt := "Use {{VAULT:a1b2c3d4e5f6}} for something"
	modified, mapping, err := interceptor.Intercept(prompt)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedModified := "Use [REDACTED:a1b2c3d4e5f6] for something"
	if modified != expectedModified {
		t.Errorf("Expected modified prompt to be %q, got %q", expectedModified, modified)
	}

	if mapping.Count() != 1 {
		t.Errorf("Expected mapping to have 1 entry, got %d", mapping.Count())
	}
}
