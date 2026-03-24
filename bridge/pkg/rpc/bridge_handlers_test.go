package rpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHeartbeatEndpoint(t *testing.T) {
	server := &Server{
		handlers: make(map[string]HandlerFunc),
	}
	server.registerHandlers()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorCode   int
		checkResult func(t *testing.T, result interface{})
	}{
		{
			name: "valid heartbeat",
			params: map[string]interface{}{
				"user_id": "test-user-123",
			},
			wantError: false,
			checkResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map[string]interface{}, got %T", result)
				}
				if userID, ok := resultMap["user_id"].(string); !ok || userID != "test-user-123" {
					t.Errorf("expected user_id test-user-123, got %v", resultMap["user_id"])
				}
				if timestamp, ok := resultMap["timestamp"].(int64); !ok || timestamp == 0 {
					t.Errorf("expected non-zero timestamp, got %v", resultMap["timestamp"])
				}
			},
		},
		{
			name:        "missing user_id",
			params:      map[string]interface{}{},
			wantError:   true,
			errorCode:   InvalidParams,
			checkResult: nil,
		},
		{
			name: "empty user_id",
			params: map[string]interface{}{
				"user_id": "",
			},
			wantError:   true,
			errorCode:   InvalidParams,
			checkResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("failed to marshal params: %v", err)
			}

			req := &Request{
				JSONRPC: JSONRPCVersion,
				ID:      1,
				Method:  "mobile.heartbeat",
				Params:  paramsJSON,
			}

			result, rpcErr := server.handleMobileHeartbeat(context.Background(), req)

			if tt.wantError {
				if rpcErr == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if rpcErr.Code != tt.errorCode {
					t.Errorf("expected error code %d, got %d", tt.errorCode, rpcErr.Code)
				}
				return
			}

			if rpcErr != nil {
				t.Errorf("unexpected error: %s", rpcErr.Message)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestHeartbeatTracking(t *testing.T) {
	server := &Server{
		handlers: make(map[string]HandlerFunc),
	}
	server.registerHandlers()

	userID := "test-user-tracking"

	// First heartbeat
	params := map[string]interface{}{
		"user_id": userID,
	}
	paramsJSON, _ := json.Marshal(params)
	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      1,
		Method:  "mobile.heartbeat",
		Params:  paramsJSON,
	}

	result1, err := server.handleMobileHeartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("first heartbeat failed: %v", err)
	}
	resultMap1 := result1.(map[string]interface{})
	timestamp1 := resultMap1["timestamp"].(int64)
	time1 := time.Unix(0, timestamp1)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Second heartbeat
	result2, err := server.handleMobileHeartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("second heartbeat failed: %v", err)
	}
	resultMap2 := result2.(map[string]interface{})
	timestamp2 := resultMap2["timestamp"].(int64)
	time2 := time.Unix(0, timestamp2)

	// Verify timestamps are different
	if time2.Before(time1) || time2.Equal(time1) {
		t.Errorf("second heartbeat should be after first, got time1=%v, time2=%v", time1, time2)
	}

	// Verify GetLastHeartbeat returns the latest timestamp
	lastHeartbeat := server.GetLastHeartbeat(userID)
	if !lastHeartbeat.Equal(time2) {
		t.Errorf("GetLastHeartbeat should return latest timestamp, got %v, want %v", lastHeartbeat, time2)
	}

	// Verify non-existent user returns zero time
	nonExistentUser := "non-existent-user-999"
	nonExistentHeartbeat := server.GetLastHeartbeat(nonExistentUser)
	if !nonExistentHeartbeat.IsZero() {
		t.Errorf("GetLastHeartbeat for non-existent user should return zero time, got %v", nonExistentHeartbeat)
	}

	// Verify empty userID returns zero time
	emptyUserHeartbeat := server.GetLastHeartbeat("")
	if !emptyUserHeartbeat.IsZero() {
		t.Errorf("GetLastHeartbeat for empty userID should return zero time, got %v", emptyUserHeartbeat)
	}
}
