package pii

import (
	"fmt"
	"regexp"
	"strings"
)

type MaskedField struct {
	Placeholder string
	Original    string
	Type        string
	Start       int
	End         int
}

type Masker struct {
	patterns []*piiPattern
}

type piiPattern struct {
	regex   *regexp.Regexp
	piiType string
}

func NewMasker() *Masker {
	m := &Masker{}
	m.patterns = []*piiPattern{
		{regex: regexp.MustCompile(`\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b`), piiType: "ssn"},
		{regex: regexp.MustCompile(`\b4\d{3}[-.\s]?\d{4}[-.\s]?\d{4}[-.\s]?\d{4}\b`), piiType: "credit_card_visa"},
		{regex: regexp.MustCompile(`\b5[1-5]\d{2}[-.\s]?\d{4}[-.\s]?\d{4}[-.\s]?\d{4}\b`), piiType: "credit_card_mc"},
		{regex: regexp.MustCompile(`\b3[47]\d{2}[-.\s]?\d{6}[-.\s]?\d{5}\b`), piiType: "credit_card_amex"},
		{regex: regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`), piiType: "phone"},
		{regex: regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{2,4}\b`), piiType: "date"},
	}
	return m
}

func (m *Masker) MaskPII(text string) (string, []MaskedField) {
	var fields []MaskedField
	result := text
	offset := 0

	for _, pat := range m.patterns {
		matches := pat.regex.FindAllStringIndex(result, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			start, end := matches[i][0], matches[i][1]
			original := result[start:end]
			placeholder := fmt.Sprintf("{{VAULT:%s_%d}}", pat.piiType, len(fields))

			fields = append([]MaskedField{{
				Placeholder: placeholder,
				Original:    original,
				Type:        pat.piiType,
				Start:       start + offset,
				End:         end + offset,
			}}, fields...)

			result = result[:start] + placeholder + result[end:]
		}
	}

	return result, fields
}

func (m *Masker) ResolvePlaceholders(text string, resolutions map[string]string) string {
	result := text
	for placeholder, value := range resolutions {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (m *Masker) ExtractPlaceholders(text string) []string {
	re := regexp.MustCompile(`\{\{VAULT:[^}]+\}\}`)
	return re.FindAllString(text, -1)
}
