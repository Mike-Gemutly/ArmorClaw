// Package license provides license validation for ArmorClaw premium features.
// It implements an offline-first caching strategy with grace period support.
package license

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// Default server URLs
const (
	DefaultServerURL = "https://api.armorclaw.com/v1"
	StagingServerURL = "https://api-staging.armorclaw.com/v1"

	// Default grace period in days
	DefaultGracePeriodDays = 3

	// Cache refresh interval
	DefaultRefreshInterval = 1 * time.Hour

	// Request timeout
	DefaultTimeout = 10 * time.Second
)

// Tier represents license tier levels
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "ent"
)

// License status
type Status string

const (
	StatusActive    Status = "active"
	StatusExpired   Status = "expired"
	StatusRevoked   Status = "revoked"
	StatusSuspended Status = "suspended"
)

// ValidationRequest is sent to the license server
type ValidationRequest struct {
	LicenseKey string `json:"license_key"`
	InstanceID string `json:"instance_id"`
	Feature    string `json:"feature"`
	Version    string `json:"version"`
}

// ValidationResponse is received from the license server
type ValidationResponse struct {
	Valid            bool     `json:"valid"`
	Tier             Tier     `json:"tier"`
	Features         []string `json:"features"`
	ExpiresAt        string   `json:"expires_at"`
	InstanceID       string   `json:"instance_id"`
	GracePeriodDays  int      `json:"grace_period_days"`
	FeatureValid     bool     `json:"feature_valid,omitempty"`
	ErrorCode        string   `json:"error_code,omitempty"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	AvailableFeatures []string `json:"available_features,omitempty"`
}

// CachedLicense stores a validated license with cache metadata
type CachedLicense struct {
	Valid       bool      `json:"valid"`
	Tier        Tier      `json:"tier"`
	Features    []string  `json:"features"`
	ExpiresAt   time.Time `json:"expires_at"`
	CachedAt    time.Time `json:"cached_at"`
	GraceUntil  time.Time `json:"grace_until"`
	InstanceID  string    `json:"instance_id"`
	LastChecked time.Time `json:"last_checked"`
}

// IsValid checks if the cached license is still valid
func (c *CachedLicense) IsValid() bool {
	if c == nil {
		return false
	}
	now := time.Now()

	// Check if still within server-specified expiration
	if now.Before(c.ExpiresAt) {
		return c.Valid
	}

	// Check if within grace period
	if now.Before(c.GraceUntil) {
		return c.Valid
	}

	return false
}

// HasFeature checks if the license includes a specific feature
func (c *CachedLicense) HasFeature(feature string) bool {
	if c == nil || !c.Valid {
		return false
	}
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// ShouldRefresh determines if the license should be re-validated
func (c *CachedLicense) ShouldRefresh() bool {
	if c == nil {
		return true
	}

	now := time.Now()

	// If still valid per server, no refresh needed
	if now.Before(c.ExpiresAt) {
		return false
	}

	// Within grace period? No refresh required yet
	if now.Before(c.GraceUntil) {
		return false
	}

	// Grace period expired, must refresh
	return true
}

// ClientConfig configures the license client
type ClientConfig struct {
	// Server URL (defaults to production)
	ServerURL string

	// License key to validate
	LicenseKey string

	// Instance ID (auto-generated if empty)
	InstanceID string

	// Bridge version
	Version string

	// HTTP client timeout
	Timeout time.Duration

	// Grace period in days (default: 3)
	GracePeriodDays int

	// Enable offline mode (never contact server)
	OfflineMode bool

	// Logger
	Logger *slog.Logger
}

// Client manages license validation
type Client struct {
	config ClientConfig
	client *http.Client
	cache  map[string]*CachedLicense
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewClient creates a new license client
func NewClient(config ClientConfig) (*Client, error) {
	if config.ServerURL == "" {
		config.ServerURL = DefaultServerURL
	}

	if config.InstanceID == "" {
		instanceID, err := generateInstanceID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate instance ID: %w", err)
		}
		config.InstanceID = instanceID
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.GracePeriodDays == 0 {
		config.GracePeriodDays = DefaultGracePeriodDays
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	return &Client{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		cache:  make(map[string]*CachedLicense),
		logger: logger,
	}, nil
}

// Validate checks if a feature is available under the current license
func (c *Client) Validate(ctx context.Context, feature string) (bool, error) {
	// Check cache first
	cached := c.GetCached(feature)

	// If cached and valid, return immediately
	if cached != nil && !cached.ShouldRefresh() {
		c.logger.Debug("using cached license",
			"feature", feature,
			"valid", cached.Valid,
			"cached_at", cached.CachedAt,
		)
		return cached.Valid, nil
	}

	// In offline mode, use cache only
	if c.config.OfflineMode {
		if cached != nil {
			c.logger.Warn("offline mode: using cached license",
				"feature", feature,
				"grace_until", cached.GraceUntil,
			)
			return cached.Valid, nil
		}
		return false, fmt.Errorf("offline mode: no cached license for %s", feature)
	}

	// Validate with server
	result, err := c.validateWithServer(ctx, feature)
	if err != nil {
		// Server unreachable, use cache if available
		if cached != nil {
			c.logger.Warn("server unreachable, using cached license",
				"feature", feature,
				"error", err,
			)
			return cached.Valid, nil
		}
		return false, fmt.Errorf("no cached license and server unreachable: %w", err)
	}

	// Update cache
	c.SetCached(feature, result)

	return result.Valid, nil
}

// validateWithServer calls the license server API
func (c *Client) validateWithServer(ctx context.Context, feature string) (*CachedLicense, error) {
	reqBody := ValidationRequest{
		LicenseKey: c.config.LicenseKey,
		InstanceID: c.config.InstanceID,
		Feature:    feature,
		Version:    c.config.Version,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.config.ServerURL + "/licenses/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.LicenseKey != "" {
		req.Header.Set("X-License-Key", c.config.LicenseKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("server error: %s", resp.Status)
		}
		return nil, fmt.Errorf("server error: %s", errResp.Error)
	}

	var result ValidationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse expiration time
	var expiresAt time.Time
	if result.ExpiresAt != "" {
		expiresAt, err = time.Parse(time.RFC3339, result.ExpiresAt)
		if err != nil {
			c.logger.Warn("failed to parse expires_at, using default",
				"value", result.ExpiresAt,
				"error", err,
			)
			expiresAt = time.Now().Add(24 * time.Hour)
		}
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	// Calculate grace period
	graceDays := result.GracePeriodDays
	if graceDays == 0 {
		graceDays = c.config.GracePeriodDays
	}
	graceUntil := expiresAt.Add(time.Duration(graceDays) * 24 * time.Hour)

	cached := &CachedLicense{
		Valid:       result.Valid,
		Tier:        result.Tier,
		Features:    result.Features,
		ExpiresAt:   expiresAt,
		CachedAt:    time.Now(),
		GraceUntil:  graceUntil,
		InstanceID:  result.InstanceID,
		LastChecked: time.Now(),
	}

	c.logger.Info("license validated",
		"feature", feature,
		"valid", result.Valid,
		"tier", result.Tier,
		"expires_at", expiresAt,
		"grace_until", graceUntil,
	)

	return cached, nil
}

// GetCached returns the cached license for a feature
func (c *Client) GetCached(feature string) *CachedLicense {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.cache[feature]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	copy := *cached
	return &copy
}

// SetCached stores a license validation result
func (c *Client) SetCached(feature string, cached *CachedLicense) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[feature] = cached
}

// HasFeature checks if a specific feature is available
func (c *Client) HasFeature(ctx context.Context, feature string) (bool, error) {
	valid, err := c.Validate(ctx, feature)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}

	cached := c.GetCached(feature)
	if cached == nil {
		return false, nil
	}

	return cached.HasFeature(feature), nil
}

// GetTier returns the current license tier
func (c *Client) GetTier(ctx context.Context) (Tier, error) {
	// Validate any feature to get tier info
	valid, err := c.Validate(ctx, "license-info")
	if err != nil {
		return TierFree, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find any cached entry with tier info
	for _, cached := range c.cache {
		if cached.Valid {
			return cached.Tier, nil
		}
	}

	if valid {
		return TierFree, nil
	}
	return TierFree, fmt.Errorf("no valid license found")
}

// GetFeatures returns all available features
func (c *Client) GetFeatures(ctx context.Context) ([]string, error) {
	// Validate to get feature list
	_, err := c.Validate(ctx, "license-info")
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Collect features from all cached entries
	features := make(map[string]bool)
	for _, cached := range c.cache {
		if cached.Valid {
			for _, f := range cached.Features {
				features[f] = true
			}
		}
	}

	result := make([]string, 0, len(features))
	for f := range features {
		result = append(result, f)
	}
	return result, nil
}

// ClearCache removes all cached license data
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CachedLicense)
}

// GetInstanceID returns the client's instance ID
func (c *Client) GetInstanceID() string {
	return c.config.InstanceID
}

// SetLicenseKey updates the license key
func (c *Client) SetLicenseKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.LicenseKey != key {
		c.config.LicenseKey = key
		// Clear cache when key changes
		c.cache = make(map[string]*CachedLicense)
	}
}

// generateInstanceID creates a unique instance identifier
func generateInstanceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
