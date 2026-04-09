package cdp

import (
	"encoding/json"
	"testing"
)

func TestFallbackHandler_LogAndDrop(t *testing.T) {
	fallback := NewFallbackHandler()

	msg := &CDPMessage{
		ID:     1,
		Method: "Unknown.unsupportedMethod",
		Params: []byte(`{"param": "value"}`),
	}

	result, err := fallback.HandleUnsupported(msg)
	if err != nil {
		t.Fatalf("HandleUnsupported() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	if result.ID != msg.ID {
		t.Errorf("Expected result ID to match message ID, got %d want %d", result.ID, msg.ID)
	}

	if result.Error == nil {
		t.Error("Expected error to be set for unsupported method")
	}

	if result.Error.Code != -32601 {
		t.Errorf("Expected error code -32601 (method not found), got %d", result.Error.Code)
	}
}

func TestFallbackHandler_DummySuccessResponse(t *testing.T) {
	fallback := NewFallbackHandler()
	fallback.EnableDummySuccess()

	msg := &CDPMessage{
		ID:     2,
		Method: "Some.method",
		Params: []byte(`{}`),
	}

	result, err := fallback.HandleUnsupported(msg)
	if err != nil {
		t.Fatalf("HandleUnsupported() failed: %v", err)
	}

	if result.Error != nil {
		t.Errorf("Expected dummy success (no error), got error: %v", result.Error)
	}

	if result.Result == nil {
		t.Error("Expected result to be set for dummy success response")
	}

	var resultMap map[string]any
	if err := json.Unmarshal(result.Result, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if resultMap["success"] != true {
		t.Error("Expected success field to be true")
	}
}

func TestFallbackHandler_DisableDummySuccess(t *testing.T) {
	fallback := NewFallbackHandler()
	fallback.EnableDummySuccess()
	fallback.DisableDummySuccess()

	msg := &CDPMessage{
		ID:     3,
		Method: "Another.method",
		Params: []byte(`{}`),
	}

	result, err := fallback.HandleUnsupported(msg)
	if err != nil {
		t.Fatalf("HandleUnsupported() failed: %v", err)
	}

	if result.Error == nil {
		t.Error("Expected error to be set when dummy success is disabled")
	}
}

func TestFallbackHandler_PreserveMessageID(t *testing.T) {
	fallback := NewFallbackHandler()

	testCases := []int{0, 1, 100, 9999}

	for _, id := range testCases {
		msg := &CDPMessage{
			ID:     id,
			Method: "Test.method",
			Params: []byte(`{}`),
		}

		result, err := fallback.HandleUnsupported(msg)
		if err != nil {
			t.Fatalf("HandleUnsupported() failed for ID %d: %v", id, err)
		}

		if result.ID != id {
			t.Errorf("Expected result ID to match message ID, got %d want %d", result.ID, id)
		}
	}
}

func TestFallbackHandler_LogMethod(t *testing.T) {
	fallback := NewFallbackHandler()

	msg := &CDPMessage{
		ID:     4,
		Method: "Some.cdp.method",
		Params: []byte(`{}`),
	}

	_, err := fallback.HandleUnsupported(msg)
	if err != nil {
		t.Fatalf("HandleUnsupported() failed: %v", err)
	}

	lastLogged := fallback.GetLastLoggedMethod()
	if lastLogged != "Some.cdp.method" {
		t.Errorf("Expected last logged method to be 'Some.cdp.method', got '%s'", lastLogged)
	}
}
