// Package auth provides unified Matrix token authentication for ArmorClaw
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
)

// MatrixAuthProvider provides authentication via Matrix homeserver tokens
type MatrixAuthProvider struct {
	homeserverURL string
	client        *http.Client
	cache         *tokenCache
	auditLog      *audit.TamperEvidentLog
	logger        *logger.Logger
	mu            sync.RWMutex
}

// MatrixAuthProviderConfig configures the Matrix auth provider
type MatrixAuthProviderConfig struct {
	HomeserverURL string
	Timeout       time.Duration
	AuditLog      *audit.TamperEvidentLog
	Logger        *logger.Logger
}

// UserInfo contains user information from Matrix
type UserInfo struct {
	UserID    string `json:"user_id"`
	DeviceID  string `json:"device_id"`
	IsGuest   bool   `json:"is_guest,omitempty"`
	PowerLevel int   `json:"power_level,omitempty"` // For RBAC
}

// tokenCache caches validated tokens to reduce homeserver load
type tokenCache struct {
	tokens map[string]*cachedToken
	mu     sync.RWMutex
}

type cachedToken struct {
	userInfo   *UserInfo
	expiration time.Time
}

// NewMatrixAuthProvider creates a new Matrix authentication provider
func NewMatrixAuthProvider(cfg MatrixAuthProviderConfig) (*MatrixAuthProvider, error) {
	if cfg.HomeserverURL == "" {
		return nil, fmt.Errorf("homeserver URL is required")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	if cfg.Logger == nil {
		cfg.Logger = logger.Global().WithComponent("matrix_auth")
	}

	return &MatrixAuthProvider{
		homeserverURL: strings.TrimSuffix(cfg.HomeserverURL, "/"),
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		cache: &tokenCache{
			tokens: make(map[string]*cachedToken),
		},
		auditLog: cfg.AuditLog,
		logger:   cfg.Logger,
	}, nil
}

// ValidateToken validates a Matrix access token against the homeserver
// Returns UserInfo if valid, error otherwise
func (p *MatrixAuthProvider) ValidateToken(ctx context.Context, token string) (*UserInfo, error) {
	// Check cache first
	if cached := p.cache.get(token); cached != nil {
		p.logger.Debug("token_cache_hit", "user_id", cached.userInfo.UserID)
		return cached.userInfo, nil
	}

	// Query homeserver
	userInfo, err := p.queryWhoami(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Cache the result
	p.cache.set(token, userInfo, 5*time.Minute)

	p.logger.Info("token_validated",
		"user_id", userInfo.UserID,
		"device_id", userInfo.DeviceID,
	)

	return userInfo, nil
}

// queryWhoami queries the Matrix homeserver's whoami endpoint
func (p *MatrixAuthProvider) queryWhoami(ctx context.Context, token string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/_matrix/client/v3/account/whoami", p.homeserverURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homeserver error: %d", resp.StatusCode)
	}

	var whoamiResp struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
		IsGuest  bool   `json:"is_guest"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&whoamiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &UserInfo{
		UserID:   whoamiResp.UserID,
		DeviceID: whoamiResp.DeviceID,
		IsGuest:  whoamiResp.IsGuest,
	}, nil
}

// GetUserPowerLevel fetches the user's power level from a room for RBAC
func (p *MatrixAuthProvider) GetUserPowerLevel(ctx context.Context, token, roomID, userID string) (int, error) {
	url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/state/m.room.power_levels", p.homeserverURL, roomID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get power levels: %d", resp.StatusCode)
	}

	var powerLevels struct {
		Users map[string]int `json:"users"`
		Default int `json:"users_default"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&powerLevels); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}

	if level, ok := powerLevels.Users[userID]; ok {
		return level, nil
	}
	return powerLevels.Default, nil
}

// InvalidateToken removes a token from the cache
func (p *MatrixAuthProvider) InvalidateToken(token string) {
	p.cache.delete(token)
}

// SetAuditLog updates the audit log
func (p *MatrixAuthProvider) SetAuditLog(auditLog *audit.TamperEvidentLog) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.auditLog = auditLog
}

// Cache methods

func (c *tokenCache) get(token string) *cachedToken {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.tokens[token]
	if !exists || time.Now().After(cached.expiration) {
		return nil
	}
	return cached
}

func (c *tokenCache) set(token string, userInfo *UserInfo, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[token] = &cachedToken{
		userInfo:   userInfo,
		expiration: time.Now().Add(ttl),
	}
}

func (c *tokenCache) delete(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tokens, token)
}

// CleanupExpired removes expired tokens from the cache
func (c *tokenCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0
	for token, cached := range c.tokens {
		if now.After(cached.expiration) {
			delete(c.tokens, token)
			count++
		}
	}
	return count
}

// ============================================================================

// RPCAuthMiddleware provides authentication middleware for RPC handlers
type RPCAuthMiddleware struct {
	provider       *MatrixAuthProvider
	publicMethods  map[string]bool
	adminMethods   map[string]bool
	adminPowerLevel int
}

// RPCAuthMiddlewareConfig configures the RPC auth middleware
type RPCAuthMiddlewareConfig struct {
	Provider       *MatrixAuthProvider
	PublicMethods  []string // Methods that don't require auth
	AdminMethods   []string // Methods that require admin power level
	AdminPowerLevel int     // Minimum power level for admin access (default: 50)
}

// NewRPCAuthMiddleware creates a new RPC auth middleware
func NewRPCAuthMiddleware(cfg RPCAuthMiddlewareConfig) *RPCAuthMiddleware {
	publicMethods := make(map[string]bool)
	for _, m := range cfg.PublicMethods {
		publicMethods[m] = true
	}

	adminMethods := make(map[string]bool)
	for _, m := range cfg.AdminMethods {
		adminMethods[m] = true
	}

	adminPowerLevel := cfg.AdminPowerLevel
	if adminPowerLevel == 0 {
		adminPowerLevel = 50 // Matrix default admin level
	}

	return &RPCAuthMiddleware{
		provider:        cfg.Provider,
		publicMethods:   publicMethods,
		adminMethods:    adminMethods,
		adminPowerLevel: adminPowerLevel,
	}
}

// AuthResult represents the result of authentication
type AuthResult struct {
	Authenticated bool
	UserInfo      *UserInfo
	Error         error
	IsAdmin       bool
}

// Authenticate authenticates an RPC request
func (m *RPCAuthMiddleware) Authenticate(ctx context.Context, method string, authToken string, adminRoomID string) *AuthResult {
	// Check if method is public
	if m.publicMethods[method] {
		return &AuthResult{
			Authenticated: true,
			UserInfo:      nil,
			IsAdmin:       false,
		}
	}

	// Require token for non-public methods
	if authToken == "" {
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("authentication required"),
		}
	}

	// Validate token
	userInfo, err := m.provider.ValidateToken(ctx, authToken)
	if err != nil {
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("invalid token: %w", err),
		}
	}

	// Check admin access if required
	isAdmin := false
	if m.adminMethods[method] {
		if adminRoomID == "" {
			return &AuthResult{
				Authenticated: false,
				Error:         fmt.Errorf("admin room not configured"),
			}
		}

		powerLevel, err := m.provider.GetUserPowerLevel(ctx, authToken, adminRoomID, userInfo.UserID)
		if err != nil {
			return &AuthResult{
				Authenticated: false,
				Error:         fmt.Errorf("failed to verify admin status: %w", err),
			}
		}

		if powerLevel < m.adminPowerLevel {
			return &AuthResult{
				Authenticated: false,
				Error:         fmt.Errorf("admin access required (power level %d, have %d)", m.adminPowerLevel, powerLevel),
			}
		}
		isAdmin = true
	}

	return &AuthResult{
		Authenticated: true,
		UserInfo:      userInfo,
		IsAdmin:       isAdmin,
	}
}

// IsPublicMethod checks if a method is public
func (m *RPCAuthMiddleware) IsPublicMethod(method string) bool {
	return m.publicMethods[method]
}

// IsAdminMethod checks if a method requires admin access
func (m *RPCAuthMiddleware) IsAdminMethod(method string) bool {
	return m.adminMethods[method]
}

// ============================================================================

// DefaultPublicMethods are RPC methods that don't require authentication
var DefaultPublicMethods = []string{
	"system.health",
	"system.config",
	"system.info",
	"device.validate",
}

// DefaultAdminMethods are RPC methods that require admin access
var DefaultAdminMethods = []string{
	"license.activate",
	"license.deactivate",
	"license.update",
	"sso.configure",
	"sso.enable",
	"sso.disable",
	"admin.users",
	"admin.rooms",
	"admin.settings",
	"security.upgrade_tier",
	"audit.export",
	"config.update",
}

// ============================================================================

// ExtractBearerToken extracts a Bearer token from an Authorization header
func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}
