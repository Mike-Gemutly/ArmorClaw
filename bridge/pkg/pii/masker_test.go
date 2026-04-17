package pii

import (
	"strings"
	"testing"
)

func TestMasker_MaskPII_SSN(t *testing.T) {
	m := NewMasker()
	text := "My SSN is 123-45-6789 please help"
	masked, fields := m.MaskPII(text)
	if len(fields) == 0 {
		t.Fatal("expected at least 1 PII field")
	}
	if fields[0].Type != "ssn" {
		t.Errorf("expected type ssn, got %s", fields[0].Type)
	}
	if fields[0].Original != "123-45-6789" {
		t.Errorf("expected original 123-45-6789, got %s", fields[0].Original)
	}
	if !strings.Contains(masked, "{{VAULT:ssn_") {
		t.Errorf("masked text should contain placeholder, got: %s", masked)
	}
	if strings.Contains(masked, "123-45-6789") {
		t.Error("masked text should not contain original SSN")
	}
}

func TestMasker_MaskPII_VisaCreditCard(t *testing.T) {
	m := NewMasker()
	text := "Card: 4111-1111-1111-1111"
	masked, fields := m.MaskPII(text)
	if len(fields) == 0 {
		t.Fatal("expected at least 1 PII field")
	}
	found := false
	for _, f := range fields {
		if f.Type == "credit_card_visa" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected credit_card_visa, got types: %v", fields)
	}
	if strings.Contains(masked, "4111") {
		t.Error("masked text should not contain card number")
	}
}

func TestMasker_MaskPII_MCCreditCard(t *testing.T) {
	m := NewMasker()
	text := "Card: 5111-1111-1111-1111"
	_, fields := m.MaskPII(text)
	found := false
	for _, f := range fields {
		if f.Type == "credit_card_mc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected credit_card_mc, got types: %v", fields)
	}
}

func TestMasker_MaskPII_AmexCreditCard(t *testing.T) {
	m := NewMasker()
	text := "Amex: 3782-822463-10005"
	_, fields := m.MaskPII(text)
	found := false
	for _, f := range fields {
		if f.Type == "credit_card_amex" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected credit_card_amex, got types: %v", fields)
	}
}

func TestMasker_MaskPII_Phone(t *testing.T) {
	m := NewMasker()
	text := "Call me at 555-123-4567"
	_, fields := m.MaskPII(text)
	found := false
	for _, f := range fields {
		if f.Type == "phone" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected phone, got types: %v", fields)
	}
}

func TestMasker_MaskPII_Date(t *testing.T) {
	m := NewMasker()
	text := "DOB: 01/15/1990"
	_, fields := m.MaskPII(text)
	found := false
	for _, f := range fields {
		if f.Type == "date" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected date, got types: %v", fields)
	}
}

func TestMasker_MaskPII_NoPII(t *testing.T) {
	m := NewMasker()
	text := "Hello, this is a normal email with no sensitive data."
	masked, fields := m.MaskPII(text)
	if len(fields) != 0 {
		t.Errorf("expected 0 PII fields, got %d", len(fields))
	}
	if masked != text {
		t.Error("text should be unchanged when no PII found")
	}
}

func TestMasker_MaskPII_MultipleTypes(t *testing.T) {
	m := NewMasker()
	text := "SSN: 123-45-6789, Card: 4111-1111-1111-1111, Phone: 555-123-4567"
	_, fields := m.MaskPII(text)
	types := make(map[string]int)
	for _, f := range fields {
		types[f.Type]++
	}
	if len(types) < 3 {
		t.Errorf("expected at least 3 distinct PII types, got %d: %v", len(types), types)
	}
}

func TestMasker_MaskPII_MultipleSSN(t *testing.T) {
	m := NewMasker()
	text := "SSN1: 123-45-6789 SSN2: 987-65-4321 SSN3: 111-22-3333"
	_, fields := m.MaskPII(text)
	ssnCount := 0
	for _, f := range fields {
		if f.Type == "ssn" {
			ssnCount++
		}
	}
	if ssnCount < 3 {
		t.Errorf("expected >= 3 SSNs, got %d", ssnCount)
	}
}

func TestMasker_MaskPII_PlaceholderFormat(t *testing.T) {
	m := NewMasker()
	text := "SSN: 123-45-6789"
	_, fields := m.MaskPII(text)
	if len(fields) == 0 {
		t.Fatal("expected at least 1 field")
	}
	p := fields[0].Placeholder
	if !strings.HasPrefix(p, "{{VAULT:ssn_") {
		t.Errorf("placeholder should start with {{VAULT:ssn_, got %s", p)
	}
	if !strings.HasSuffix(p, "}}") {
		t.Errorf("placeholder should end with }}, got %s", p)
	}
}

func TestMasker_ResolvePlaceholders(t *testing.T) {
	m := NewMasker()
	text := "Hello {{VAULT:ssn_0}}, your card {{VAULT:credit_card_visa_1}} is ready"
	resolutions := map[string]string{
		"{{VAULT:ssn_0}}":              "123-45-6789",
		"{{VAULT:credit_card_visa_1}}": "4111-1111-1111-1111",
	}
	resolved := m.ResolvePlaceholders(text, resolutions)
	if resolved != "Hello 123-45-6789, your card 4111-1111-1111-1111 is ready" {
		t.Errorf("unexpected resolved text: %s", resolved)
	}
}

func TestMasker_ResolvePlaceholders_NoMatch(t *testing.T) {
	m := NewMasker()
	text := "Hello {{VAULT:ssn_0}}"
	resolutions := map[string]string{}
	resolved := m.ResolvePlaceholders(text, resolutions)
	if resolved != text {
		t.Errorf("unresolved placeholders should remain, got: %s", resolved)
	}
}

func TestMasker_ResolvePlaceholders_EmptyInput(t *testing.T) {
	m := NewMasker()
	resolved := m.ResolvePlaceholders("", nil)
	if resolved != "" {
		t.Errorf("empty input should return empty, got: %s", resolved)
	}
}

func TestMasker_ExtractPlaceholders(t *testing.T) {
	m := NewMasker()
	text := "Hello {{VAULT:ssn_0}} and {{VAULT:phone_1}} goodbye"
	placeholders := m.ExtractPlaceholders(text)
	if len(placeholders) != 2 {
		t.Fatalf("expected 2 placeholders, got %d", len(placeholders))
	}
	if placeholders[0] != "{{VAULT:ssn_0}}" {
		t.Errorf("first placeholder wrong: %s", placeholders[0])
	}
	if placeholders[1] != "{{VAULT:phone_1}}" {
		t.Errorf("second placeholder wrong: %s", placeholders[1])
	}
}

func TestMasker_ExtractPlaceholders_None(t *testing.T) {
	m := NewMasker()
	text := "No placeholders here"
	placeholders := m.ExtractPlaceholders(text)
	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders, got %d", len(placeholders))
	}
}

func TestMasker_Roundtrip_MaskAndResolve(t *testing.T) {
	m := NewMasker()
	original := "My SSN is 123-45-6789 and my phone is 555-123-4567"
	masked, fields := m.MaskPII(original)
	if len(fields) < 2 {
		t.Fatalf("expected >= 2 fields, got %d", len(fields))
	}

	resolutions := make(map[string]string)
	for _, f := range fields {
		resolutions[f.Placeholder] = f.Original
	}

	resolved := m.ResolvePlaceholders(masked, resolutions)
	if resolved != original {
		t.Errorf("roundtrip mismatch\ngot:  %q\nwant: %q", resolved, original)
	}
}
