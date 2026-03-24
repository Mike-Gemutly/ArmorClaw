package rpc

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMethodRegistration(t *testing.T) {
	criticalMethods := []string{
		"matrix.status",
		"matrix.login",
		"matrix.send",
		"matrix.join_room",
		"ai.chat",
	}

	server := &Server{}
	server.registerHandlers()

	for _, method := range criticalMethods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("critical method %q not registered in handlers map", method)
			}
		})
	}
}

func TestMethodRegistrationCompleteness(t *testing.T) {
	expectedMethods := []string{
		"ai.chat",
		"browser.navigate",
		"browser.fill",
		"browser.click",
		"browser.status",
		"matrix.status",
		"matrix.login",
		"matrix.send",
		"matrix.join_room",
	}

	server := &Server{}
	server.registerHandlers()

	for _, method := range expectedMethods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("expected method %q not registered in handlers map", method)
			}
		})
	}
}

func TestMatrixJoinRoomMissingRoomID(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	handler := server.handlers["matrix.join_room"]
	if handler == nil {
		t.Fatalf("matrix.join_room handler not registered")
	}

	// Test with empty room_id
	req := &Request{
		Params: json.RawMessage(`{"via_servers": [], "reason": "test"}`),
	}

	_, errObj := handler(context.Background(), req)
	if errObj == nil {
		t.Error("expected error for missing room_id, got nil")
	}

	if errObj.Code != InvalidParams {
		t.Errorf("expected InvalidParams code, got %d", errObj.Code)
	}

	if errObj.Message != "room_id is required" {
		t.Errorf("expected 'room_id is required' message, got '%s'", errObj.Message)
	}
}
