package skills

import (
	"testing"
)

func TestPreExecutionChecks_URLWithQueryParams(t *testing.T) {
	se := NewSkillExecutor()
	params := map[string]interface{}{
		"url": "https://api.example.com/search?q=test&page=1",
	}
	if err := se.checkDangerousPatterns(params); err != nil {
		t.Fatalf("URL with query params should not error: %v", err)
	}
}

func TestPreExecutionChecks_JSONBody(t *testing.T) {
	se := NewSkillExecutor()
	params := map[string]interface{}{
		"body": `{"key": "value"}`,
	}
	if err := se.checkDangerousPatterns(params); err != nil {
		t.Fatalf("JSON body should not error: %v", err)
	}
}

func TestPreExecutionChecks_HTMLContent(t *testing.T) {
	se := NewSkillExecutor()
	params := map[string]interface{}{
		"content": "<p>Hello</p>",
	}
	if err := se.checkDangerousPatterns(params); err != nil {
		t.Fatalf("HTML content should not error: %v", err)
	}
}

func TestPreExecutionChecks_MathExpression(t *testing.T) {
	se := NewSkillExecutor()
	params := map[string]interface{}{
		"expression": "(a + b) * c",
	}
	if err := se.checkDangerousPatterns(params); err != nil {
		t.Fatalf("Math expression should not error: %v", err)
	}
}

func TestPreExecutionChecks_URLWithAmpersand(t *testing.T) {
	se := NewSkillExecutor()
	params := map[string]interface{}{
		"url": "https://example.com?a=1&b=2&c=3",
	}
	if err := se.checkDangerousPatterns(params); err != nil {
		t.Fatalf("URL with ampersand should not error: %v", err)
	}
}
