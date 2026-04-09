package cdp

import (
	"encoding/json"
	"testing"
)

func TestRouterDelegatesMouseClickToTranslator(t *testing.T) {
	translator := NewTranslator()
	router := NewMethodRouter(translator)

	mouseParams := map[string]any{
		"type":       "mousePressed",
		"x":          100,
		"y":          200,
		"button":     "left",
		"clickCount": 1,
	}
	params, _ := json.Marshal(mouseParams)

	msg := &CDPMessage{
		ID:     1,
		Method: "Input.dispatchMouseEvent",
		Params: params,
	}

	route := router.Route("Input.dispatchMouseEvent")
	if route == nil {
		t.Fatal("Expected route for Input.dispatchMouseEvent")
	}
	if route.Handler == nil {
		t.Fatal("Expected handler for Input.dispatchMouseEvent")
	}

	result, err := route.Handler(msg)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result.Method != "Runtime.evaluate" {
		t.Errorf("Expected translated method Runtime.evaluate, got %s", result.Method)
	}

	var translatedParams map[string]any
	if err := json.Unmarshal(result.Params, &translatedParams); err != nil {
		t.Fatalf("Failed to unmarshal translated params: %v", err)
	}

	if _, ok := translatedParams["expression"]; !ok {
		t.Error("Expected 'expression' key in translated params")
	}
}

func TestRouterDelegatesKeyInputToTranslator(t *testing.T) {
	translator := NewTranslator()
	router := NewMethodRouter(translator)

	keyParams := map[string]any{
		"type":                  "keyDown",
		"key":                   "Enter",
		"code":                  "Enter",
		"windowsVirtualKeyCode": 13,
	}
	params, _ := json.Marshal(keyParams)

	msg := &CDPMessage{
		ID:     2,
		Method: "Input.dispatchKeyEvent",
		Params: params,
	}

	route := router.Route("Input.dispatchKeyEvent")
	if route == nil || route.Handler == nil {
		t.Fatal("Expected handler for Input.dispatchKeyEvent")
	}

	result, err := route.Handler(msg)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result.Method != "Input.dispatchKeyEvent" {
		t.Errorf("Expected passthrough method Input.dispatchKeyEvent, got %s", result.Method)
	}
}

func TestRouterDelegatesTextInsertToTranslator(t *testing.T) {
	translator := NewTranslator()
	router := NewMethodRouter(translator)

	textParams := map[string]any{
		"text": "hello world",
	}
	params, _ := json.Marshal(textParams)

	msg := &CDPMessage{
		ID:     3,
		Method: "Input.insertText",
		Params: params,
	}

	route := router.Route("Input.insertText")
	if route == nil || route.Handler == nil {
		t.Fatal("Expected handler for Input.insertText")
	}

	result, err := route.Handler(msg)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result.Method != "Input.insertText" {
		t.Errorf("Expected passthrough method Input.insertText, got %s", result.Method)
	}
}

func TestRouterUnknownEventPassthrough(t *testing.T) {
	translator := NewTranslator()
	router := NewMethodRouter(translator)

	msg := &CDPMessage{
		ID:     99,
		Method: "Some.Unknown.Method",
		Params: json.RawMessage(`{"foo":"bar"}`),
	}

	route := router.Route("Some.Unknown.Method")
	if route == nil {
		t.Fatal("Expected route for unknown method")
	}

	if route.Action != ActionUnsupported {
		t.Errorf("Expected ActionUnsupported for unknown method, got %s", route.Action)
	}

	if route.Handler == nil {
		t.Fatal("Expected handler for unsupported method")
	}

	result, err := route.Handler(msg)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result.Method != msg.Method {
		t.Errorf("Expected original method %s, got %s", msg.Method, result.Method)
	}
}
