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
		"health.check",
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
		"health.check",
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

func TestHealthCheckHandler(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	handler := server.handlers["health.check"]
	if handler == nil {
		t.Fatalf("health.check handler not registered")
	}

	req := &Request{
		Params: json.RawMessage(`{}`),
	}

	result, errObj := handler(context.Background(), req)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var healthResp HealthCheckResponse
	if err := json.Unmarshal(resultBytes, &healthResp); err != nil {
		t.Fatalf("failed to unmarshal HealthCheckResponse: %v", err)
	}

	if healthResp.Status == "" {
		t.Error("expected status to be set, got empty string")
	}

	if healthResp.Components == nil {
		t.Error("expected components to be set, got nil")
	}

	if _, ok := healthResp.Components["bridge"]; !ok {
		t.Error("expected components to contain 'bridge' key")
	}

	if _, ok := healthResp.Components["matrix"]; !ok {
		t.Error("expected components to contain 'matrix' key")
	}

	if _, ok := healthResp.Components["keystore"]; !ok {
		t.Error("expected components to contain 'keystore' key")
	}

	validStatuses := []string{"healthy", "degraded", "unhealthy"}
	valid := false
	for _, vs := range validStatuses {
		if healthResp.Status == vs {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("expected status to be one of %v, got %s", validStatuses, healthResp.Status)
	}
}
