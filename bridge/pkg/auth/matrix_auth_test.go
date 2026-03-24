// Package auth provides authentication tests for ArmorClaw bridge.
package auth

import (
	"context"
	"testing"
)

func TestRequireTokenAuth(t *testing.T) {
	cfg := MatrixAuthProviderConfig{
		HomeserverURL: "https://matrix.example.com",
	}

	provider, err := NewMatrixAuthProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create auth provider: %v", err)
	}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		Provider:      provider,
		PublicMethods: []string{"system.health", "system.info"},
	})

	ctx := context.Background()

	testCases := []struct {
		name        string
		method      string
		authToken   string
		expectError bool
	}{
		{
			name:        "public method without token",
			method:      "system.health",
			authToken:   "",
			expectError: false,
		},
		{
			name:        "non-public method without token",
			method:      "ai.chat",
			authToken:   "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := middleware.Authenticate(ctx, tc.method, tc.authToken, "")
			if tc.expectError && result.Error == nil {
				t.Error("Expected authentication error, got nil")
			}
			if !tc.expectError && result.Error != nil {
				t.Errorf("Expected no authentication error, got: %v", result.Error)
			}
		})
	}
}

func TestDefaultPublicMethods(t *testing.T) {
	expectedPublicMethods := []string{
		"system.health",
		"system.config",
		"system.info",
		"device.validate",
	}

	if len(DefaultPublicMethods) != len(expectedPublicMethods) {
		t.Errorf("Expected %d default public methods, got %d", len(expectedPublicMethods), len(DefaultPublicMethods))
	}

	for _, method := range expectedPublicMethods {
		found := false
		for _, m := range DefaultPublicMethods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected public method %s not found in DefaultPublicMethods", method)
		}
	}
}
