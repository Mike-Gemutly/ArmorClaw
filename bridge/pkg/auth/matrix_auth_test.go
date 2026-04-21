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

type mockAdminTokenValidator struct {
	tokens map[string]struct {
		userID string
		role   string
	}
}

func (m *mockAdminTokenValidator) ValidateAdminToken(token string) (string, string, bool) {
	entry, ok := m.tokens[token]
	if !ok {
		return "", "", false
	}
	return entry.userID, entry.role, true
}

func TestAdminTokenValidOwner(t *testing.T) {
	mock := &mockAdminTokenValidator{
		tokens: map[string]struct {
			userID string
			role   string
		}{
			"aat_validtoken123": {userID: "u_abc123", role: "OWNER"},
		},
	}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "device.list", "aat_validtoken123", "")
	if !result.Authenticated {
		t.Fatalf("Expected authenticated, got error: %v", result.Error)
	}
	if !result.IsAdmin {
		t.Error("Expected IsAdmin=true for OWNER")
	}
	if result.AdminUserID != "u_abc123" {
		t.Errorf("Expected AdminUserID=u_abc123, got %s", result.AdminUserID)
	}
	if result.AdminRole != "OWNER" {
		t.Errorf("Expected AdminRole=OWNER, got %s", result.AdminRole)
	}
}

func TestAdminTokenInvalidToken(t *testing.T) {
	mock := &mockAdminTokenValidator{
		tokens: map[string]struct {
			userID string
			role   string
		}{},
	}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "device.list", "aat_unknown", "")
	if result.Authenticated {
		t.Error("Expected authentication to fail for invalid token")
	}
	if result.Error == nil {
		t.Error("Expected error for invalid admin token")
	}
}

func TestAdminTokenMissingToken(t *testing.T) {
	mock := &mockAdminTokenValidator{tokens: map[string]struct {
		userID string
		role   string
	}{}}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "device.list", "", "")
	if result.Authenticated {
		t.Error("Expected authentication to fail with no token")
	}
}

func TestAdminTokenNonAdminRoleRejected(t *testing.T) {
	mock := &mockAdminTokenValidator{
		tokens: map[string]struct {
			userID string
			role   string
		}{
			"aat_moderatortoken": {userID: "u_mod123", role: "MODERATOR"},
		},
	}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "device.approve", "aat_moderatortoken", "")
	if result.Authenticated {
		t.Error("Expected MODERATOR to be rejected for admin method")
	}
}

func TestAdminTokenNonAatPrefixFallsThrough(t *testing.T) {
	mock := &mockAdminTokenValidator{tokens: map[string]struct {
		userID string
		role   string
	}{}}

	cfg := MatrixAuthProviderConfig{
		HomeserverURL: "https://matrix.example.com",
	}
	provider, _ := NewMatrixAuthProvider(cfg)

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		Provider:            provider,
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "ai.chat", "syt_not_an_admin_token", "")
	if result.Error == nil || result.Authenticated {
		t.Error("Expected non-aat token to fall through to Matrix auth and fail")
	}
}

func TestAatPrefixCheck(t *testing.T) {
	mock := &mockAdminTokenValidator{tokens: map[string]struct {
		userID string
		role   string
	}{}}

	cfg := MatrixAuthProviderConfig{HomeserverURL: "https://matrix.example.com"}
	provider, _ := NewMatrixAuthProvider(cfg)

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		Provider:            provider,
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()

	result := middleware.Authenticate(ctx, "ai.chat", "aat_", "")
	if result.Error == nil || result.Authenticated {
		t.Error("Expected short aat_ prefix token to fail validation")
	}
}

func TestDefaultAdminMethodsContainDeviceInvite(t *testing.T) {
	expected := []string{
		"device.list", "device.get", "device.approve", "device.reject",
		"invite.create", "invite.list", "invite.revoke", "invite.validate",
	}

	for _, method := range expected {
		found := false
		for _, m := range DefaultAdminMethods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected admin method %s not found in DefaultAdminMethods", method)
		}
	}
}

func TestAdminTokenAdminRoleAccepted(t *testing.T) {
	mock := &mockAdminTokenValidator{
		tokens: map[string]struct {
			userID string
			role   string
		}{
			"aat_admintoken": {userID: "u_admin1", role: "ADMIN"},
		},
	}

	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		AdminTokenValidator: mock,
		PublicMethods:       []string{"system.health"},
		AdminMethods:        DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "invite.create", "aat_admintoken", "")
	if !result.Authenticated {
		t.Fatalf("Expected authenticated for ADMIN role, got error: %v", result.Error)
	}
	if !result.IsAdmin {
		t.Error("Expected IsAdmin=true for ADMIN role")
	}
}

func TestNoAdminTokenValidator(t *testing.T) {
	middleware := NewRPCAuthMiddleware(RPCAuthMiddlewareConfig{
		PublicMethods: []string{"system.health"},
		AdminMethods:  DefaultAdminMethods,
	})

	ctx := context.Background()
	result := middleware.Authenticate(ctx, "device.list", "aat_some_token", "")
	if result.Authenticated {
		t.Error("Expected auth to fail when no AdminTokenValidator configured")
	}
	if result.Error == nil {
		t.Error("Expected error when no AdminTokenValidator configured")
	}
}
