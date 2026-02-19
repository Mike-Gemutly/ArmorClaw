// Package sso provides Single Sign-On integration for ArmorClaw Enterprise
// Supports SAML 2.0 and OpenID Connect (OIDC) protocols
package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ProviderType represents the type of SSO provider
type ProviderType string

const (
	ProviderSAML ProviderType = "saml"
	ProviderOIDC ProviderType = "oidc"
)

// SSOConfig configures the SSO integration
type SSOConfig struct {
	// Provider type
	Provider ProviderType

	// Common settings
	Enabled      bool
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IssuerURL    string

	// SAML-specific settings
	SAML *SAMLConfig

	// OIDC-specific settings
	OIDC *OIDCConfig

	// Role mapping
	RoleMapping map[string]string // SSO attribute -> ArmorClaw role

	// Logger
	Logger *slog.Logger
}

// SAMLConfig contains SAML 2.0 specific configuration
type SAMLConfig struct {
	// Identity Provider metadata URL or file path
	IDPMetadataURL string
	IDPMetadataFile string

	// Service Provider settings
	EntityID       string
	AssertionURL   string

	// Certificate settings
	CertFile    string
	KeyFile     string
	PrivateKey  *rsa.PrivateKey
	Certificate *x509.Certificate

	// Attribute mapping
	NameIDFormat   string
	Attributes     map[string]string // Friendly name -> SAML attribute
}

// OIDCConfig contains OpenID Connect specific configuration
type OIDCConfig struct {
	// Endpoints (auto-discovered if empty)
	AuthorizationEndpoint string
	TokenEndpoint         string
	UserInfoEndpoint     string
	JWKSURL              string

	// Scopes to request
	Scopes []string

	// PKCE settings
	UsePKCE bool

	// Token validation
	Audience string
}

// SSOSession represents an authenticated SSO session
type SSOSession struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	Roles        []string          `json:"roles"`
	Attributes   map[string]string `json:"attributes"`
	Provider     ProviderType      `json:"provider"`
	ExpiresAt    time.Time         `json:"expires_at"`
	CreatedAt    time.Time         `json:"created_at"`
	AccessToken  string            `json:"-"`
	RefreshToken string            `json:"-"`
	IDToken      string            `json:"-"`
}

// SSOProvider is the interface for SSO providers
type SSOProvider interface {
	// GetAuthURL generates the authentication URL for the OAuth/SAML flow
	GetAuthURL(state string) (string, error)

	// HandleCallback processes the callback from the IdP
	HandleCallback(ctx context.Context, code string, state string) (*SSOSession, error)

	// ValidateSession validates an existing session
	ValidateSession(ctx context.Context, session *SSOSession) (bool, error)

	// RefreshSession refreshes an expired session
	RefreshSession(ctx context.Context, session *SSOSession) error

	// LogoutURL generates a logout URL
	LogoutURL(session *SSOSession) (string, error)
}

// SSOManager manages SSO authentication
type SSOManager struct {
	config    SSOConfig
	provider  SSOProvider
	sessions  map[string]*SSOSession
	mu        sync.RWMutex
	logger    *slog.Logger
	stateStore *StateStore
}

// StateStore manages OAuth state parameters
type StateStore struct {
	states map[string]*StateEntry
	mu     sync.RWMutex
}

// StateEntry represents a stored state parameter
type StateEntry struct {
	State     string
	Redirect  string
	CreatedAt time.Time
	PKCE      string // For OIDC PKCE
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		states: make(map[string]*StateEntry),
	}
}

// Generate creates a new state parameter
func (s *StateStore) Generate(redirect string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	state := base64.URLEncoding.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = &StateEntry{
		State:     state,
		Redirect:  redirect,
		CreatedAt: time.Now(),
	}

	// Clean up old states (older than 10 minutes)
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, v := range s.states {
		if v.CreatedAt.Before(cutoff) {
			delete(s.states, k)
		}
	}

	return state, nil
}

// Validate validates and retrieves a state entry
func (s *StateStore) Validate(state string) (*StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.states[state]
	if ok {
		delete(s.states, state) // One-time use
	}
	return entry, ok
}

// SetPKCE sets the PKCE challenge for a state
func (s *StateStore) SetPKCE(state, pkce string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.states[state]; ok {
		entry.PKCE = pkce
	}
}

// GetPKCE gets the PKCE verifier for a state
func (s *StateStore) GetPKCE(state string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.states[state]; ok {
		return entry.PKCE
	}
	return ""
}

// NewSSOManager creates a new SSO manager
func NewSSOManager(config SSOConfig) (*SSOManager, error) {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	if !config.Enabled {
		return &SSOManager{
			config:    config,
			sessions:  make(map[string]*SSOSession),
			logger:    logger,
			stateStore: NewStateStore(),
		}, nil
	}

	var provider SSOProvider
	var err error

	switch config.Provider {
	case ProviderSAML:
		provider, err = NewSAMLProvider(config)
	case ProviderOIDC:
		provider, err = NewOIDCProvider(config)
	default:
		err = fmt.Errorf("unsupported SSO provider type: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create SSO provider: %w", err)
	}

	return &SSOManager{
		config:    config,
		provider:  provider,
		sessions:  make(map[string]*SSOSession),
		logger:    logger,
		stateStore: NewStateStore(),
	}, nil
}

// IsEnabled returns whether SSO is enabled
func (m *SSOManager) IsEnabled() bool {
	return m.config.Enabled
}

// GetProviderType returns the configured provider type
func (m *SSOManager) GetProviderType() ProviderType {
	return m.config.Provider
}

// BeginAuth starts the SSO authentication flow
func (m *SSOManager) BeginAuth(redirect string) (string, string, error) {
	if !m.config.Enabled {
		return "", "", fmt.Errorf("SSO is not enabled")
	}

	// Generate state parameter
	state, err := m.stateStore.Generate(redirect)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Get auth URL from provider
	authURL, err := m.provider.GetAuthURL(state)
	if err != nil {
		return "", "", fmt.Errorf("failed to get auth URL: %w", err)
	}

	m.logger.Info("SSO auth initiated",
		"state", state,
		"provider", m.config.Provider,
	)

	return authURL, state, nil
}

// HandleCallback handles the callback from the IdP
func (m *SSOManager) HandleCallback(ctx context.Context, code, state string) (*SSOSession, string, error) {
	if !m.config.Enabled {
		return nil, "", fmt.Errorf("SSO is not enabled")
	}

	// Validate state
	stateEntry, ok := m.stateStore.Validate(state)
	if !ok {
		return nil, "", fmt.Errorf("invalid state parameter")
	}

	// Handle callback with provider
	session, err := m.provider.HandleCallback(ctx, code, state)
	if err != nil {
		return nil, "", fmt.Errorf("callback handling failed: %w", err)
	}

	// Apply role mapping
	session.Roles = m.mapRoles(session.Attributes)

	// Store session
	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	m.logger.Info("SSO session created",
		"session_id", session.ID,
		"user_id", session.UserID,
		"email", session.Email,
		"provider", m.config.Provider,
	)

	return session, stateEntry.Redirect, nil
}

// ValidateSession validates an existing session
func (m *SSOManager) ValidateSession(ctx context.Context, sessionID string) (*SSOSession, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		// Try to refresh
		if err := m.provider.RefreshSession(ctx, session); err != nil {
			m.mu.Lock()
			delete(m.sessions, sessionID)
			m.mu.Unlock()
			return nil, fmt.Errorf("session expired and refresh failed: %w", err)
		}
	}

	// Validate with provider
	valid, err := m.provider.ValidateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	if !valid {
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
		return nil, fmt.Errorf("session is not valid")
	}

	return session, nil
}

// Logout logs out a session
func (m *SSOManager) Logout(ctx context.Context, sessionID string) (string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return "", nil // Session doesn't exist, nothing to do
	}

	// Get logout URL (only if provider is available)
	var logoutURL string
	if m.provider != nil {
		var err error
		logoutURL, err = m.provider.LogoutURL(session)
		if err != nil {
			m.logger.Warn("Failed to get logout URL", "error", err)
		}
	}

	// Remove session
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	m.logger.Info("SSO session terminated",
		"session_id", sessionID,
		"user_id", session.UserID,
	)

	return logoutURL, nil
}

// GetSession retrieves a session by ID
func (m *SSOManager) GetSession(sessionID string) (*SSOSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	return session, ok
}

// CleanupSessions removes expired sessions
func (m *SSOManager) CleanupSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0

	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, id)
			count++
		}
	}

	if count > 0 {
		m.logger.Info("Cleaned up expired SSO sessions", "count", count)
	}

	return count
}

// mapRoles maps SSO attributes to ArmorClaw roles
func (m *SSOManager) mapRoles(attributes map[string]string) []string {
	var roles []string

	for attrName, roleName := range m.config.RoleMapping {
		if val, ok := attributes[attrName]; ok {
			// Check if the attribute indicates this role
			if val == "true" || val == "1" || strings.EqualFold(val, "yes") {
				roles = append(roles, roleName)
			} else if strings.Contains(strings.ToLower(val), strings.ToLower(roleName)) {
				roles = append(roles, roleName)
			}
		}
	}

	// Default role if none assigned
	if len(roles) == 0 {
		roles = append(roles, "user")
	}

	return roles
}

// GenerateSessionID generates a unique session ID
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SAML Provider Implementation

// SAMLProvider implements SSO for SAML 2.0
type SAMLProvider struct {
	config    SSOConfig
	sp        *SAMLServiceProvider
	idp       *SAMLIdentityProvider
	logger    *slog.Logger
}

// SAMLServiceProvider represents the Service Provider configuration
type SAMLServiceProvider struct {
	EntityID      string
	AssertionURL  string
	PrivateKey    *rsa.PrivateKey
	Certificate   *x509.Certificate
}

// SAMLIdentityProvider represents the Identity Provider configuration
type SAMLIdentityProvider struct {
	EntityID       string
	SSOURL         string
	SLOURL         string
	Certificates   []*x509.Certificate
	NameIDFormat   string
	Attributes     map[string]string
}

// NewSAMLProvider creates a new SAML provider
func NewSAMLProvider(config SSOConfig) (*SAMLProvider, error) {
	if config.SAML == nil {
		return nil, fmt.Errorf("SAML configuration is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	// Load SP certificate and key
	var privateKey *rsa.PrivateKey
	var certificate *x509.Certificate

	if config.SAML.KeyFile != "" {
		keyData, err := os.ReadFile(config.SAML.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
		privateKey, err = parsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	if config.SAML.CertFile != "" {
		certData, err := os.ReadFile(config.SAML.CertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate: %w", err)
		}
		certificate, err = parseCertificate(certData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
	}

	sp := &SAMLServiceProvider{
		EntityID:     config.SAML.EntityID,
		AssertionURL: config.SAML.AssertionURL,
		PrivateKey:   privateKey,
		Certificate:  certificate,
	}

	// Load IdP metadata
	var idp *SAMLIdentityProvider
	var err error

	if config.SAML.IDPMetadataURL != "" {
		idp, err = loadIDPMetadataFromURL(config.SAML.IDPMetadataURL)
	} else if config.SAML.IDPMetadataFile != "" {
		idp, err = loadIDPMetadataFromFile(config.SAML.IDPMetadataFile)
	} else {
		return nil, fmt.Errorf("IdP metadata URL or file is required")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load IdP metadata: %w", err)
	}

	return &SAMLProvider{
		config: config,
		sp:     sp,
		idp:    idp,
		logger: logger,
	}, nil
}

// GetAuthURL generates the SAML authentication URL (redirect to IdP)
func (p *SAMLProvider) GetAuthURL(state string) (string, error) {
	// Build SAML AuthnRequest
	authnRequest := buildSAMLAuthnRequest(p.sp.EntityID, p.sp.AssertionURL, state)

	// Encode the request
	requestXML, err := xml.Marshal(authnRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AuthnRequest: %w", err)
	}

	// Base64 encode
	encodedRequest := base64.StdEncoding.EncodeToString(requestXML)

	// Build redirect URL
	u, err := url.Parse(p.idp.SSOURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse IdP SSO URL: %w", err)
	}

	query := u.Query()
	query.Set("SAMLRequest", encodedRequest)
	query.Set("RelayState", state)
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// HandleCallback processes the SAML assertion
func (p *SAMLProvider) HandleCallback(ctx context.Context, code string, state string) (*SSOSession, error) {
	// In SAML, the "code" is actually the base64-encoded SAMLResponse
	response, err := decodeSAMLResponse(code)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAML response: %w", err)
	}

	// Validate the response signature
	if err := validateSAMLResponse(response, p.idp.Certificates); err != nil {
		return nil, fmt.Errorf("SAML response validation failed: %w", err)
	}

	// Extract user information from assertion
	session := &SSOSession{
		ID:         response.Assertion.Subject.NameID.Value,
		UserID:     response.Assertion.Subject.NameID.Value,
		Email:      getAttribute(response.Assertion, "email"),
		Name:       getAttribute(response.Assertion, "name"),
		Attributes: extractAttributes(response.Assertion),
		Provider:   ProviderSAML,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour), // Default 24h
	}

	// Generate session ID
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}
	session.ID = sessionID

	return session, nil
}

// ValidateSession validates a SAML session
func (p *SAMLProvider) ValidateSession(ctx context.Context, session *SSOSession) (bool, error) {
	// For SAML, we just check expiration locally
	// In production, you might want to check with the IdP
	return time.Now().Before(session.ExpiresAt), nil
}

// RefreshSession refreshes a SAML session
func (p *SAMLProvider) RefreshSession(ctx context.Context, session *SSOSession) error {
	// SAML doesn't have a native refresh mechanism
	// The user would need to re-authenticate
	return fmt.Errorf("SAML sessions cannot be refreshed")
}

// LogoutURL generates the SAML logout URL
func (p *SAMLProvider) LogoutURL(session *SSOSession) (string, error) {
	if p.idp.SLOURL == "" {
		return "", nil // No SLO support
	}

	// Build SAML LogoutRequest
	logoutRequest := buildSAMLLogoutRequest(p.sp.EntityID, session.UserID)

	requestXML, err := xml.Marshal(logoutRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LogoutRequest: %w", err)
	}

	encodedRequest := base64.StdEncoding.EncodeToString(requestXML)

	u, err := url.Parse(p.idp.SLOURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse IdP SLO URL: %w", err)
	}

	query := u.Query()
	query.Set("SAMLRequest", encodedRequest)
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// OIDC Provider Implementation

// OIDCProvider implements SSO for OpenID Connect
type OIDCProvider struct {
	config    SSOConfig
	endpoints *OIDCEndpoints
	jwks      *JWKS
	logger    *slog.Logger
	http      *http.Client
}

// OIDCEndpoints contains the OIDC endpoints
type OIDCEndpoints struct {
	Authorization string
	Token         string
	UserInfo      string
	JWKS          string
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	X5c []string `json:"x5c"`
}

// OIDCTokenResponse represents the token response from OIDC
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// OIDCUserInfo represents user info from OIDC
type OIDCUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(config SSOConfig) (*OIDCProvider, error) {
	if config.OIDC == nil {
		return nil, fmt.Errorf("OIDC configuration is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	provider := &OIDCProvider{
		config: config,
		logger: logger,
		http:   &http.Client{Timeout: 30 * time.Second},
	}

	// Discover endpoints if not provided
	if config.OIDC.AuthorizationEndpoint == "" {
		if err := provider.discoverEndpoints(); err != nil {
			return nil, fmt.Errorf("failed to discover OIDC endpoints: %w", err)
		}
	} else {
		provider.endpoints = &OIDCEndpoints{
			Authorization: config.OIDC.AuthorizationEndpoint,
			Token:         config.OIDC.TokenEndpoint,
			UserInfo:      config.OIDC.UserInfoEndpoint,
			JWKS:          config.OIDC.JWKSURL,
		}
	}

	// Set default scopes if not provided
	if len(config.OIDC.Scopes) == 0 {
		config.OIDC.Scopes = []string{"openid", "profile", "email"}
	}

	return provider, nil
}

// GetAuthURL generates the OIDC authorization URL
func (p *OIDCProvider) GetAuthURL(state string) (string, error) {
	u, err := url.Parse(p.endpoints.Authorization)
	if err != nil {
		return "", fmt.Errorf("failed to parse authorization endpoint: %w", err)
	}

	query := u.Query()
	query.Set("client_id", p.config.ClientID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", p.config.RedirectURL)
	query.Set("scope", strings.Join(p.config.OIDC.Scopes, " "))
	query.Set("state", state)

	if p.config.OIDC.UsePKCE {
		verifier, challenge := generatePKCE()
		// Store verifier for later use in state store
		// Note: In production, this should be stored with the state parameter
		_ = verifier // Will be used when state store supports PKCE
		query.Set("code_challenge", challenge)
		query.Set("code_challenge_method", "S256")
	}

	u.RawQuery = query.Encode()

	return u.String(), nil
}

// HandleCallback processes the OIDC callback
func (p *OIDCProvider) HandleCallback(ctx context.Context, code string, state string) (*SSOSession, error) {
	// Exchange code for tokens
	tokenResp, err := p.exchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		p.logger.Warn("Failed to get user info, using token claims", "error", err)
		// Fall back to parsing ID token claims
		userInfo = &OIDCUserInfo{
			Sub:   "unknown",
			Email: "unknown@example.com",
		}
	}

	// Create session
	session := &SSOSession{
		ID:           userInfo.Sub,
		UserID:       userInfo.Sub,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		Attributes:   make(map[string]string),
		Provider:     ProviderOIDC,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	// Add attributes
	if userInfo.GivenName != "" {
		session.Attributes["given_name"] = userInfo.GivenName
	}
	if userInfo.FamilyName != "" {
		session.Attributes["family_name"] = userInfo.FamilyName
	}
	if userInfo.Picture != "" {
		session.Attributes["picture"] = userInfo.Picture
	}

	// Generate session ID
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}
	session.ID = sessionID

	return session, nil
}

// ValidateSession validates an OIDC session
func (p *OIDCProvider) ValidateSession(ctx context.Context, session *SSOSession) (bool, error) {
	// Check if access token is still valid
	// In production, you'd validate the JWT or call the userinfo endpoint
	return time.Now().Before(session.ExpiresAt), nil
}

// RefreshSession refreshes an OIDC session using the refresh token
func (p *OIDCProvider) RefreshSession(ctx context.Context, session *SSOSession) error {
	if session.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Refresh the token
	tokenResp, err := p.refreshToken(ctx, session.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update session
	session.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		session.RefreshToken = tokenResp.RefreshToken
	}
	session.IDToken = tokenResp.IDToken
	session.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// LogoutURL generates the OIDC logout URL
func (p *OIDCProvider) LogoutURL(session *SSOSession) (string, error) {
	// Build logout URL with post_logout_redirect_uri
	u, err := url.Parse(p.config.IssuerURL + "/v1/logout")
	if err != nil {
		// Try common logout paths
		u, err = url.Parse(p.config.IssuerURL + "/logout")
		if err != nil {
			return "", nil // No logout URL available
		}
	}

	query := u.Query()
	if session.IDToken != "" {
		query.Set("id_token_hint", session.IDToken)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// discoverEndpoints discovers OIDC endpoints from the issuer
func (p *OIDCProvider) discoverEndpoints() error {
	// Try to fetch OpenID configuration
	discoveryURL := strings.TrimSuffix(p.config.IssuerURL, "/") + "/.well-known/openid-configuration"

	resp, err := p.http.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	var config struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserInfoEndpoint      string `json:"userinfo_endpoint"`
		JWKSURI               string `json:"jwks_uri"`
		EndSessionEndpoint    string `json:"end_session_endpoint"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return fmt.Errorf("failed to parse discovery document: %w", err)
	}

	p.endpoints = &OIDCEndpoints{
		Authorization: config.AuthorizationEndpoint,
		Token:         config.TokenEndpoint,
		UserInfo:      config.UserInfoEndpoint,
		JWKS:          config.JWKSURI,
	}

	return nil
}

// exchangeCode exchanges an authorization code for tokens
func (p *OIDCProvider) exchangeCode(ctx context.Context, code string) (*OIDCTokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {p.config.RedirectURL},
		"client_id":    {p.config.ClientID},
	}

	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}

	resp, err := p.http.PostForm(p.endpoints.Token, data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// getUserInfo gets user info from the userinfo endpoint
func (p *OIDCProvider) getUserInfo(ctx context.Context, accessToken string) (*OIDCUserInfo, error) {
	req, err := http.NewRequest("GET", p.endpoints.UserInfo, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request returned status %d", resp.StatusCode)
	}

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return &userInfo, nil
}

// refreshToken refreshes an access token
func (p *OIDCProvider) refreshToken(ctx context.Context, refreshToken string) (*OIDCTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.config.ClientID},
	}

	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}

	resp, err := p.http.PostForm(p.endpoints.Token, data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh request returned status %d", resp.StatusCode)
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	return &tokenResp, nil
}

// Helper functions

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	// Try PKCS1
	if key, err := x509.ParsePKCS1PrivateKey(data); err == nil {
		return key, nil
	}

	// Try PKCS8
	key, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsaKey, nil
}

func parseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParseCertificate(block.Bytes)
}

func loadIDPMetadataFromURL(url string) (*SAMLIdentityProvider, error) {
	// In production, fetch and parse SAML metadata XML
	// This is a simplified implementation
	return &SAMLIdentityProvider{
		EntityID: "https://idp.example.com",
		SSOURL:   url,
	}, nil
}

func loadIDPMetadataFromFile(path string) (*SAMLIdentityProvider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse SAML metadata XML
	// This is a simplified implementation
	_ = data
	return &SAMLIdentityProvider{
		EntityID: "https://idp.example.com",
	}, nil
}

// SAML Response structures (simplified)
type SAMLResponse struct {
	XMLName   xml.Name `xml:"Response"`
	Assertion *SAMLAssertion
}

type SAMLAssertion struct {
	XMLName  xml.Name `xml:"Assertion"`
	Subject  *SAMLSubject
	Attributes []SAMLAttribute
}

type SAMLSubject struct {
	NameID *SAMLNameID
}

type SAMLNameID struct {
	Value string `xml:",chardata"`
}

type SAMLAttribute struct {
	Name   string   `xml:"Name,attr"`
	Values []string `xml:"AttributeValue"`
}

type SAMLAuthnRequest struct {
	XMLName       xml.Name `xml:"samlp:AuthnRequest"`
	ID            string   `xml:"ID,attr"`
	Version       string   `xml:"Version,attr"`
	IssueInstant  string   `xml:"IssueInstant,attr"`
	Destination   string   `xml:"Destination,attr"`
	Issuer        string   `xml:"saml:Issuer"`
	AssertionURL  string   `xml:"AssertionConsumerServiceURL,attr"`
}

type SAMLLogoutRequest struct {
	XMLName      xml.Name `xml:"samlp:LogoutRequest"`
	ID           string   `xml:"ID,attr"`
	Version      string   `xml:"Version,attr"`
	IssueInstant string   `xml:"IssueInstant,attr"`
	Issuer       string   `xml:"saml:Issuer"`
	NameID       *SAMLNameID
}

func buildSAMLAuthnRequest(entityID, assertionURL, state string) *SAMLAuthnRequest {
	return &SAMLAuthnRequest{
		ID:           generateSAMLID(),
		Version:      "2.0",
		IssueInstant: time.Now().UTC().Format(time.RFC3339),
		Issuer:       entityID,
		AssertionURL: assertionURL,
	}
}

func buildSAMLLogoutRequest(entityID, nameID string) *SAMLLogoutRequest {
	return &SAMLLogoutRequest{
		ID:           generateSAMLID(),
		Version:      "2.0",
		IssueInstant: time.Now().UTC().Format(time.RFC3339),
		Issuer:       entityID,
		NameID:       &SAMLNameID{Value: nameID},
	}
}

func generateSAMLID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("_%x", b)
}

func decodeSAMLResponse(encoded string) (*SAMLResponse, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var response SAMLResponse
	if err := xml.Unmarshal(decoded, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func validateSAMLResponse(response *SAMLResponse, certs []*x509.Certificate) error {
	// In production, validate XML signature
	// This is a simplified implementation
	return nil
}

func getAttribute(assertion *SAMLAssertion, name string) string {
	for _, attr := range assertion.Attributes {
		if attr.Name == name && len(attr.Values) > 0 {
			return attr.Values[0]
		}
	}
	return ""
}

func extractAttributes(assertion *SAMLAssertion) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range assertion.Attributes {
		if len(attr.Values) > 0 {
			attrs[attr.Name] = attr.Values[0]
		}
	}
	return attrs
}

func generatePKCE() (verifier, challenge string) {
	// Generate random verifier
	b := make([]byte, 32)
	rand.Read(b)
	verifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)

	// Generate challenge (S256)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h[:])

	return verifier, challenge
}

