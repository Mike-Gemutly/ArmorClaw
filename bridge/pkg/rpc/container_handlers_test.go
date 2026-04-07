package rpc

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTerminateContainer_Validation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		wantError     bool
		errorCode     int
		errorContains string
	}{
		{
			name:          "missing container_id",
			params:        map[string]interface{}{"user_id": "user-456"},
			wantError:     true,
			errorCode:     InvalidParams,
			errorContains: "container_id is required",
		},
		{
			name:          "empty container_id",
			params:        map[string]interface{}{"container_id": "", "user_id": "user-456"},
			wantError:     true,
			errorCode:     InvalidParams,
			errorContains: "container_id is required",
		},
		{
			name:          "missing user_id",
			params:        map[string]interface{}{"container_id": "test-container-123"},
			wantError:     true,
			errorCode:     InvalidParams,
			errorContains: "user_id is required",
		},
		{
			name:          "empty user_id",
			params:        map[string]interface{}{"container_id": "test-container-123", "user_id": ""},
			wantError:     true,
			errorCode:     InvalidParams,
			errorContains: "user_id is required",
		},
		{
			name:          "docker client not configured",
			params:        map[string]interface{}{"container_id": "test-container", "user_id": "user-456"},
			wantError:     true,
			errorCode:     InternalError,
			errorContains: "docker client not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				handlers: make(map[string]HandlerFunc),
			}

			paramsJSON, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("failed to marshal params: %v", err)
			}

			req := &Request{
				JSONRPC: JSONRPCVersion,
				ID:      1,
				Method:  "container.terminate",
				Params:  paramsJSON,
			}

			result, rpcErr := server.handleTerminateContainer(context.Background(), req)

			if tt.wantError {
				if rpcErr == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errorCode != 0 && rpcErr.Code != tt.errorCode {
					t.Errorf("expected error code %d, got %d", tt.errorCode, rpcErr.Code)
				}
				if tt.errorContains != "" && rpcErr.Message != "" {
					found := false
					for i := 0; i <= len(rpcErr.Message)-len(tt.errorContains); i++ {
						if rpcErr.Message[i:i+len(tt.errorContains)] == tt.errorContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error message to contain %q, got %q", tt.errorContains, rpcErr.Message)
					}
				}
				return
			}

			if rpcErr != nil {
				t.Errorf("unexpected error: %s", rpcErr.Message)
				return
			}

			if result != nil {
				t.Fatalf("expected nil result, got %v", result)
			}
		})
	}
}

func TestContainsArmorClawLabel(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "armorclaw.agent_id",
			key:      "armorclaw.agent_id",
			expected: true,
		},
		{
			name:     "armorclaw.key_id",
			key:      "armorclaw.key_id",
			expected: true,
		},
		{
			name:     "armorclaw.session_id",
			key:      "armorclaw.session_id",
			expected: true,
		},
		{
			name:     "com.armorclaw.agent",
			key:      "com.armorclaw.agent",
			expected: true,
		},
		{
			name:     "com.armorclaw.managed",
			key:      "com.armorclaw.managed",
			expected: true,
		},
		{
			name:     "com.docker.compose.service",
			key:      "com.docker.compose.service",
			expected: false,
		},
		{
			name:     "random.label",
			key:      "random.label",
			expected: false,
		},
		{
			name:     "empty string",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsArmorClawLabel(tt.key)
			if result != tt.expected {
				t.Errorf("containsArmorClawLabel(%q) = %v, expected %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestListContainers_Validation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		wantError     bool
		errorCode     int
		errorContains string
	}{
		{
			name:          "missing user_id",
			params:        map[string]interface{}{"all": true},
			wantError:     true,
			errorCode:     InvalidParams,
			errorContains: "user_id is required",
		},
		{
			name:          "docker client not configured",
			params:        map[string]interface{}{"user_id": "user-456"},
			wantError:     true,
			errorCode:     InternalError,
			errorContains: "docker client not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				handlers: make(map[string]HandlerFunc),
			}

			paramsJSON, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("failed to marshal params: %v", err)
			}

			req := &Request{
				JSONRPC: JSONRPCVersion,
				ID:      1,
				Method:  "container.list",
				Params:  paramsJSON,
			}

			result, rpcErr := server.handleListContainers(context.Background(), req)

			if tt.wantError {
				if rpcErr == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errorCode != 0 && rpcErr.Code != tt.errorCode {
					t.Errorf("expected error code %d, got %d", tt.errorCode, rpcErr.Code)
				}
				if tt.errorContains != "" && rpcErr.Message != "" {
					found := false
					for i := 0; i <= len(rpcErr.Message)-len(tt.errorContains); i++ {
						if rpcErr.Message[i:i+len(tt.errorContains)] == tt.errorContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error message to contain %q, got %q", tt.errorContains, rpcErr.Message)
					}
				}
				return
			}

			if rpcErr != nil {
				t.Errorf("unexpected error: %s", rpcErr.Message)
				return
			}

			if result != nil {
				t.Fatalf("expected nil result, got %v", result)
			}
		})
	}
}
