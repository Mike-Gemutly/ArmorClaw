// Package sso provides integration tests for SSO integration
package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestStateStore tests state parameter management
func TestStateStore(t *testing.T) {
	store := NewStateStore()

	// Generate state
	state, err := store.Generate("https://example.com/callback")
	if err != nil {
		t.Fatalf("Failed to generate state: %v", err)
	}

	if state == "" {
		t.Error("State should not be empty")
	}

	// Validate state
	entry, ok := store.Validate(state)
	if !ok {
		t.Error("State validation failed")
	}

	if entry.Redirect != "https://example.com/callback" {
		t.Errorf("Expected redirect URL, got %s", entry.Redirect)
	}

	// State should be one-time use
	_, ok = store.Validate(state)
	if ok {
		t.Error("State should be invalidated after first use")
	}
}

// TestStateStorePKCE tests PKCE storage in state
func TestStateStorePKCE(t *testing.T) {
	store := NewStateStore()

	state, _ := store.Generate("https://example.com/callback")
	store.SetPKCE(state, "verifier123")

	pkce := store.GetPKCE(state)
	if pkce != "verifier123" {
		t.Errorf("Expected PKCE verifier, got %s", pkce)
	}

	// Non-existent state should return empty
	pkce = store.GetPKCE("non-existent")
	if pkce != "" {
		t.Error("Non-existent state should return empty PKCE")
	}
}

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("Failed to generate session ID: %v", err)
	}

	if id1 == "" {
		t.Error("Session ID should not be empty")
	}

	// Generate another and verify uniqueness
	id2, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("Failed to generate second session ID: %v", err)
	}

	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}
}

// TestSSOManagerDisabled tests SSO manager when disabled
func TestSSOManagerDisabled(t *testing.T) {
	config := SSOConfig{
		Enabled: false,
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	if manager.IsEnabled() {
		t.Error("SSO should be disabled")
	}

	// Should return error when trying to begin auth
	_, _, err = manager.BeginAuth("https://example.com/callback")
	if err == nil {
		t.Error("Expected error when SSO is disabled")
	}
}

// TestSSOManagerOIDCConfig tests OIDC configuration
func TestSSOManagerOIDCConfig(t *testing.T) {
	// Create a mock OIDC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"authorization_endpoint": "https://idp.example.com/authorize",
				"token_endpoint": "https://idp.example.com/token",
				"userinfo_endpoint": "https://idp.example.com/userinfo",
				"jwks_uri": "https://idp.example.com/jwks"
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := SSOConfig{
		Enabled:      true,
		Provider:     ProviderOIDC,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/callback",
		IssuerURL:    server.URL,
		OIDC: &OIDCConfig{
			Scopes: []string{"openid", "profile", "email"},
		},
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	if manager.GetProviderType() != ProviderOIDC {
		t.Errorf("Expected OIDC provider, got %s", manager.GetProviderType())
	}
}

// TestOIDCProviderGetAuthURL tests OIDC authorization URL generation
func TestOIDCProviderGetAuthURL(t *testing.T) {
	config := SSOConfig{
		Provider:     ProviderOIDC,
		ClientID:     "test-client",
		RedirectURL:  "https://app.example.com/callback",
		OIDC: &OIDCConfig{
			AuthorizationEndpoint: "https://idp.example.com/authorize",
			TokenEndpoint:         "https://idp.example.com/token",
			Scopes:                []string{"openid", "profile", "email"},
		},
	}

	provider, err := NewOIDCProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OIDC provider: %v", err)
	}

	authURL, err := provider.GetAuthURL("test-state")
	if err != nil {
		t.Fatalf("Failed to get auth URL: %v", err)
	}

	// Verify URL contains required parameters
	if !strings.Contains(authURL, "client_id=test-client") {
		t.Error("Auth URL should contain client_id")
	}

	if !strings.Contains(authURL, "redirect_uri=") {
		t.Error("Auth URL should contain redirect_uri")
	}

	if !strings.Contains(authURL, "response_type=code") {
		t.Error("Auth URL should contain response_type=code")
	}

	if !strings.Contains(authURL, "state=test-state") {
		t.Error("Auth URL should contain state parameter")
	}

	if !strings.Contains(authURL, "scope=") {
		t.Error("Auth URL should contain scope")
	}
}

// TestSAMLProviderGetAuthURL tests SAML authorization URL generation
func TestSAMLProviderGetAuthURL(t *testing.T) {
	config := SSOConfig{
		Provider: ProviderSAML,
		SAML: &SAMLConfig{
			EntityID:        "https://sp.example.com",
			AssertionURL:    "https://sp.example.com/sso/saml",
			IDPMetadataURL:  "https://idp.example.com/metadata",
		},
	}

	provider, err := NewSAMLProvider(config)
	if err != nil {
		t.Fatalf("Failed to create SAML provider: %v", err)
	}

	authURL, err := provider.GetAuthURL("test-state")
	if err != nil {
		t.Fatalf("Failed to get auth URL: %v", err)
	}

	// Verify URL contains SAMLRequest
	if !strings.Contains(authURL, "SAMLRequest=") {
		t.Error("Auth URL should contain SAMLRequest")
	}

	if !strings.Contains(authURL, "RelayState=test-state") {
		t.Error("Auth URL should contain RelayState")
	}
}

// TestRoleMapping tests role mapping from SSO attributes
func TestRoleMapping(t *testing.T) {
	config := SSOConfig{
		Enabled:  false, // Disabled to avoid needing provider config
		RoleMapping: map[string]string{
			"is_admin":  "admin",
			"is_editor": "editor",
			"department": "department_role",
		},
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	// Test role mapping logic
	roles := manager.mapRoles(map[string]string{
		"is_admin":   "true",
		"is_editor":  "false",
		"department": "engineering",
	})

	// Should include admin role
	foundAdmin := false
	for _, r := range roles {
		if r == "admin" {
			foundAdmin = true
			break
		}
	}

	if !foundAdmin {
		t.Error("Expected admin role to be mapped")
	}
}

// TestSessionManagement tests session creation and retrieval
func TestSessionManagement(t *testing.T) {
	config := SSOConfig{
		Enabled:  false, // Disabled for basic tests
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	// Create a mock session
	session := &SSOSession{
		ID:        "session-123",
		UserID:    "user-456",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"user"},
		Provider:  ProviderOIDC,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Store session
	manager.mu.Lock()
	manager.sessions[session.ID] = session
	manager.mu.Unlock()

	// Retrieve session
	retrieved, ok := manager.GetSession(session.ID)
	if !ok {
		t.Error("Failed to retrieve session")
	}

	if retrieved.Email != session.Email {
		t.Errorf("Expected email %s, got %s", session.Email, retrieved.Email)
	}

	// Non-existent session
	_, ok = manager.GetSession("non-existent")
	if ok {
		t.Error("Should not find non-existent session")
	}
}

// TestSessionCleanup tests expired session cleanup
func TestSessionCleanup(t *testing.T) {
	config := SSOConfig{
		Enabled: false,
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	// Add expired session
	expiredSession := &SSOSession{
		ID:        "expired-session",
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	// Add active session
	activeSession := &SSOSession{
		ID:        "active-session",
		UserID:    "user-2",
		ExpiresAt: time.Now().Add(1 * time.Hour), // Active
	}

	manager.mu.Lock()
	manager.sessions[expiredSession.ID] = expiredSession
	manager.sessions[activeSession.ID] = activeSession
	manager.mu.Unlock()

	// Run cleanup
	cleaned := manager.CleanupSessions()

	if cleaned != 1 {
		t.Errorf("Expected 1 session cleaned, got %d", cleaned)
	}

	// Verify only active session remains
	_, expiredOk := manager.GetSession(expiredSession.ID)
	_, activeOk := manager.GetSession(activeSession.ID)

	if expiredOk {
		t.Error("Expired session should have been cleaned up")
	}

	if !activeOk {
		t.Error("Active session should still exist")
	}
}

// TestLogout tests session logout
func TestLogout(t *testing.T) {
	config := SSOConfig{
		Enabled: false,
	}

	manager, err := NewSSOManager(config)
	if err != nil {
		t.Fatalf("Failed to create SSO manager: %v", err)
	}

	// Add a session
	session := &SSOSession{
		ID:        "logout-session",
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	manager.mu.Lock()
	manager.sessions[session.ID] = session
	manager.mu.Unlock()

	// Logout
	_, err = manager.Logout(context.Background(), session.ID)
	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}

	// Verify session is gone
	_, ok := manager.GetSession(session.ID)
	if ok {
		t.Error("Session should be removed after logout")
	}

	// Logout non-existent session should not error
	_, err = manager.Logout(context.Background(), "non-existent")
	if err != nil {
		t.Errorf("Logout of non-existent session should not error: %v", err)
	}
}

// TestPKCEGeneration tests PKCE code generation
func TestPKCEGeneration(t *testing.T) {
	verifier, challenge := generatePKCE()

	if verifier == "" {
		t.Error("Verifier should not be empty")
	}

	if challenge == "" {
		t.Error("Challenge should not be empty")
	}

	// Verifier and challenge should be different
	if verifier == challenge {
		t.Error("Verifier and challenge should be different")
	}

	// Generate multiple and verify uniqueness
	v2, _ := generatePKCE()
	if verifier == v2 {
		t.Error("Verifiers should be unique")
	}
}

// TestSAMLIDGeneration tests SAML ID generation
func TestSAMLIDGeneration(t *testing.T) {
	id1 := generateSAMLID()
	id2 := generateSAMLID()

	if id1 == "" {
		t.Error("SAML ID should not be empty")
	}

	if !strings.HasPrefix(id1, "_") {
		t.Error("SAML ID should start with underscore")
	}

	if id1 == id2 {
		t.Error("SAML IDs should be unique")
	}
}

// TestSAMLAuthnRequest tests SAML AuthnRequest building
func TestSAMLAuthnRequest(t *testing.T) {
	req := buildSAMLAuthnRequest("https://sp.example.com", "https://sp.example.com/sso/saml", "state123")

	if req.ID == "" {
		t.Error("AuthnRequest should have ID")
	}

	if req.Version != "2.0" {
		t.Errorf("Expected SAML version 2.0, got %s", req.Version)
	}

	if req.Issuer != "https://sp.example.com" {
		t.Errorf("Expected issuer, got %s", req.Issuer)
	}

	if req.AssertionURL != "https://sp.example.com/sso/saml" {
		t.Errorf("Expected assertion URL, got %s", req.AssertionURL)
	}
}

// TestSAMLLogoutRequest tests SAML LogoutRequest building
func TestSAMLLogoutRequest(t *testing.T) {
	req := buildSAMLLogoutRequest("https://sp.example.com", "user-123")

	if req.ID == "" {
		t.Error("LogoutRequest should have ID")
	}

	if req.Version != "2.0" {
		t.Errorf("Expected SAML version 2.0, got %s", req.Version)
	}

	if req.NameID == nil {
		t.Error("LogoutRequest should have NameID")
	}

	if req.NameID.Value != "user-123" {
		t.Errorf("Expected NameID value, got %s", req.NameID.Value)
	}
}

// BenchmarkStateGeneration benchmarks state parameter generation
func BenchmarkStateGeneration(b *testing.B) {
	store := NewStateStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Generate("https://example.com/callback")
	}
}

// BenchmarkSessionIDGeneration benchmarks session ID generation
func BenchmarkSessionIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSessionID()
	}
}
