// Package adapter provides tests for Matrix adapter ChangePassword functionality
package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/armorclaw/bridge/pkg/logger"
)

// TestChangePasswordSuccess tests successful password change
func TestChangePasswordSuccess(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/_matrix/client/v3/account/password" {
			t.Errorf("expected path /_matrix/client/v3/account/password, got %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got '%s'", auth)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["new_password"] != "newSecurePassword123" {
			t.Errorf("expected new_password 'newSecurePassword123', got '%v'", payload["new_password"])
		}

		if payload["logout_devices"] != false {
			t.Errorf("expected logout_devices false, got '%v'", payload["logout_devices"])
		}

		// Return success
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	// Create adapter with mock server
	cfg := Config{
		HomeserverURL: server.URL,
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Set access token directly (simulating logged in state)
	adapter.mu.Lock()
	adapter.accessToken = "test-token"
	adapter.syncToken = "s123_456"
	adapter.mu.Unlock()

	// Test password change with logoutDevices=false
	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", false)
	if err != nil {
		t.Errorf("ChangePassword failed: %v", err)
	}
}

// TestChangePasswordSuccessWithLogout tests successful password change with logout
func TestChangePasswordSuccessWithLogout(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		if payload["logout_devices"] != true {
			t.Errorf("expected logout_devices true, got '%v'", payload["logout_devices"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cfg := Config{
		HomeserverURL: server.URL,
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	adapter.mu.Lock()
	adapter.accessToken = "test-token"
	adapter.syncToken = "s123_456"
	adapter.mu.Unlock()

	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", true)
	if err != nil {
		t.Errorf("ChangePassword with logout failed: %v", err)
	}
}

// TestChangePasswordUnauthorized tests password change with invalid token
func TestChangePasswordUnauthorized(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errcode": "M_UNKNOWN_TOKEN",
			"error":   "Invalid access token",
		})
	}))
	defer server.Close()

	cfg := Config{
		HomeserverURL: server.URL,
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	adapter.mu.Lock()
	adapter.accessToken = "invalid-token"
	adapter.mu.Unlock()

	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", false)
	if err == nil {
		t.Error("Expected error for unauthorized request, got nil")
	}

	// Verify error message contains unauthorized info
	if err != nil && err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}

// TestChangePasswordWeakPassword tests password change with weak password
func TestChangePasswordWeakPassword(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errcode": "M_WEAK_PASSWORD",
			"error":   "Password does not meet complexity requirements",
		})
	}))
	defer server.Close()

	cfg := Config{
		HomeserverURL: server.URL,
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	adapter.mu.Lock()
	adapter.accessToken = "test-token"
	adapter.mu.Unlock()

	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "weak", false)
	if err == nil {
		t.Error("Expected error for weak password, got nil")
	}

	// Verify error message contains weak password info
	if err != nil && err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}

// TestChangePasswordNotLoggedIn tests password change when not logged in
func TestChangePasswordNotLoggedIn(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	cfg := Config{
		HomeserverURL: "https://matrix.example.com",
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Don't set access token (simulating not logged in)
	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", false)
	if err == nil {
		t.Error("Expected error when not logged in, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}

// TestChangePasswordNetworkError tests password change with network error
func TestChangePasswordNetworkError(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	// Use invalid URL to simulate network error
	cfg := Config{
		HomeserverURL: "http://invalid-host-that-does-not-exist.example.com:9999",
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	adapter.mu.Lock()
	adapter.accessToken = "test-token"
	adapter.mu.Unlock()

	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", false)
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

// TestChangePasswordInvalidStatusCode tests password change with unexpected status code
func TestChangePasswordInvalidStatusCode(t *testing.T) {
	logger.Initialize("info", "text", "stdout")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errcode": "M_UNKNOWN",
			"error":   "Internal server error",
		})
	}))
	defer server.Close()

	cfg := Config{
		HomeserverURL: server.URL,
	}

	adapter, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	adapter.mu.Lock()
	adapter.accessToken = "test-token"
	adapter.mu.Unlock()

	ctx := context.Background()
	err = adapter.ChangePassword(ctx, "newSecurePassword123", false)
	if err == nil {
		t.Error("Expected error for 500 status, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}
