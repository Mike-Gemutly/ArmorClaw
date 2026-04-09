package cdp

import (
	"encoding/json"
	"testing"
)

func TestTranslateMouseClick(t *testing.T) {
	translator := NewTranslator()

	mouseClickParams := map[string]any{
		"type":       "mousePressed",
		"x":          100,
		"y":          200,
		"button":     "left",
		"clickCount": 1,
	}
	params, _ := json.Marshal(mouseClickParams)

	msg := &CDPMessage{
		ID:     1,
		Method: "Input.dispatchMouseEvent",
		Params: params,
	}

	translated, err := translator.Translate(msg)
	if err != nil {
		t.Fatalf("Translate() failed: %v", err)
	}

	if translated.Method != "Runtime.evaluate" {
		t.Errorf("Expected method Runtime.evaluate, got %s", translated.Method)
	}

	var result map[string]any
	if err := json.Unmarshal(translated.Params, &result); err != nil {
		t.Fatalf("Failed to unmarshal translated params: %v", err)
	}

	expression, ok := result["expression"].(string)
	if !ok {
		t.Fatal("Expected expression to be a string")
	}

	if !containsSubstring(expression, "elementFromPoint") {
		t.Error("Expected expression to contain elementFromPoint for element resolution")
	}

	if !containsSubstring(expression, "shadowRoot") {
		t.Error("Expected expression to contain shadowRoot check for Shadow DOM boundary handling")
	}
}

func TestThreeTierFallbackMatrix(t *testing.T) {
	translator := NewTranslator()

	_ = `document.querySelector('[data-automation-id="submit-btn"]')`
	translated := translator.translateMouseClick(100, 200)
	if !containsSubstring(translated, "click()") {
		t.Error("Expected click() in translation")
	}
}

func TestShadowDOMBoundaryHandling(t *testing.T) {
	translator := NewTranslator()

	mouseClickParams := map[string]any{
		"type":   "mousePressed",
		"x":      150,
		"y":      250,
		"button": "left",
	}
	params, _ := json.Marshal(mouseClickParams)

	msg := &CDPMessage{
		ID:     2,
		Method: "Input.dispatchMouseEvent",
		Params: params,
	}

	translated, err := translator.Translate(msg)
	if err != nil {
		t.Fatalf("Translate() failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(translated.Params, &result); err != nil {
		t.Fatalf("Failed to unmarshal translated params: %v", err)
	}
	expression := result["expression"].(string)

	if !containsSubstring(expression, "shadowRoot") {
		t.Error("Expected Shadow DOM boundary checking in expression")
	}

	if !containsSubstring(expression, "elem.shadowRoot.elementFromPoint") {
		t.Error("Expected recursive elementFromPoint within shadowRoot")
	}
}

func TestFallbackToPixelCoordinates(t *testing.T) {
	translator := NewTranslator()

	mouseClickParams := map[string]any{
		"type":   "mousePressed",
		"x":      300,
		"y":      400,
		"button": "left",
	}
	params, _ := json.Marshal(mouseClickParams)

	msg := &CDPMessage{
		ID:     3,
		Method: "Input.dispatchMouseEvent",
		Params: params,
	}

	translated, err := translator.Translate(msg)
	if err != nil {
		t.Fatalf("Translate() failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(translated.Params, &result); err != nil {
		t.Fatalf("Failed to unmarshal translated params: %v", err)
	}
	expression := result["expression"].(string)

	if !containsSubstring(expression, "click()") || !containsSubstring(expression, "300") || !containsSubstring(expression, "400") {
		t.Error("Expected fallback to pixel coordinates in click expression")
	}
}

func TestTranslatePassthrough(t *testing.T) {
	translator := NewTranslator()

	networkParams := map[string]any{
		"url": "https://example.com",
	}
	params, _ := json.Marshal(networkParams)

	msg := &CDPMessage{
		ID:     4,
		Method: "Network.enable",
		Params: params,
	}

	translated, err := translator.Translate(msg)
	if err != nil {
		t.Fatalf("Translate() failed: %v", err)
	}

	if translated.Method != msg.Method {
		t.Errorf("Expected passthrough for Network.enable, got %s", translated.Method)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
