package rpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/trust"
)

type mockStore struct {
	state *trust.UserHardeningState
}

func (m *mockStore) Get(userID string) (*trust.UserHardeningState, error) {
	if m.state == nil {
		m.state = &trust.UserHardeningState{
			UserID: userID,
		}
	}
	return m.state, nil
}

func (m *mockStore) Put(state *trust.UserHardeningState) error {
	m.state = state
	return nil
}

func (m *mockStore) IsDelegationReady(userID string) (bool, error) {
	state, err := m.Get(userID)
	if err != nil {
		return false, err
	}
	return state.DelegationReady, nil
}

func (m *mockStore) AckStep(userID string, step trust.HardeningStep) error {
	state, _ := m.Get(userID)
	switch step {
	case trust.PasswordRotated:
		state.PasswordRotated = true
	case trust.BootstrapWiped:
		state.BootstrapWiped = true
	case trust.DeviceVerified:
		state.DeviceVerified = true
	case trust.RecoveryBackedUp:
		state.RecoveryBackedUp = true
	case trust.BiometricsEnabled:
		state.BiometricsEnabled = true
	}
	state.Recompute()
	return m.Put(state)
}

func TestHardeningHandlerRegistration(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	hardeningMethods := []string{
		"hardening.status",
		"hardening.ack",
		"hardening.rotate_password",
	}

	for _, method := range hardeningMethods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("hardening method %q not registered in handlers map", method)
			}
		})
	}
}

func TestHardeningStatus(t *testing.T) {
	mockStore := &mockStore{}
	matrixAdapter := &adapter.MatrixAdapter{}
	handler := NewHardeningHandler(mockStore, matrixAdapter, "")

	req := &Request{
		Params: json.RawMessage(`{}`),
	}

	_, errObj := handler.handleHardeningStatus(context.Background(), req)
	if errObj != nil {
		if errObj.Message != "Not authenticated" {
			t.Errorf("Expected 'Not authenticated' error when not logged in, got: %s", errObj.Message)
		}
	}
}

func TestHardeningAck(t *testing.T) {
	tests := []struct {
		name         string
		params       string
		expectError  bool
		expectedCode int
		expectedMsg  string
	}{
		{
			name:        "valid step",
			params:      `{"step": "password_rotated"}`,
			expectError: false,
		},
		{
			name:         "missing params",
			params:       `{}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "step parameter cannot be empty",
		},
		{
			name:         "invalid step",
			params:       `{"step": "invalid_step"}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "Invalid step: invalid_step",
		},
		{
			name:         "empty step",
			params:       `{"step": ""}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "step parameter cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockStore{}
			matrixAdapter := &adapter.MatrixAdapter{}
			handler := NewHardeningHandler(mockStore, matrixAdapter, "")

			req := &Request{
				Params: json.RawMessage(tt.params),
			}

			_, errObj := handler.handleHardeningAck(context.Background(), req)

			if tt.expectError {
				if errObj == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.expectedCode != 0 && errObj.Code != tt.expectedCode {
					t.Errorf("Expected error code %d, got %d", tt.expectedCode, errObj.Code)
				}
				if tt.expectedMsg != "" && errObj.Message != tt.expectedMsg {
					t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, errObj.Message)
				}
			} else {
				if errObj != nil && errObj.Message != "Not authenticated" {
					t.Errorf("Unexpected error: %s", errObj.Message)
				}
			}
		})
	}
}

func TestHardeningRotatePasswordValidation(t *testing.T) {
	tests := []struct {
		name         string
		params       string
		expectError  bool
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "missing params",
			params:       `{}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "new_password cannot be empty",
		},
		{
			name:         "empty password",
			params:       `{"new_password": ""}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "new_password cannot be empty",
		},
		{
			name:         "short password",
			params:       `{"new_password": "short"}`,
			expectError:  true,
			expectedCode: InvalidParams,
			expectedMsg:  "new_password must be at least 8 characters",
		},
		{
			name:        "valid password",
			params:      `{"new_password": "new_secure_password"}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockStore{}
			matrixAdapter := &adapter.MatrixAdapter{}
			handler := NewHardeningHandler(mockStore, matrixAdapter, "")

			req := &Request{
				Params: json.RawMessage(tt.params),
			}

			_, errObj := handler.handleHardeningRotatePassword(context.Background(), req)

			if tt.expectError {
				if errObj == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.expectedCode != 0 && errObj.Code != tt.expectedCode {
					t.Errorf("Expected error code %d, got %d", tt.expectedCode, errObj.Code)
				}
				if tt.expectedMsg != "" && errObj.Message != tt.expectedMsg {
					t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, errObj.Message)
				}
			} else {
				if errObj != nil && errObj.Message != "Not authenticated" {
					t.Errorf("Unexpected error: %s", errObj.Message)
				}
			}
		})
	}
}
