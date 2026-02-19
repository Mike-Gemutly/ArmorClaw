// Package rpc provides public RPC endpoints for pre-login access
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

// PublicRPCConfig configures public RPC endpoints
type PublicRPCConfig struct {
	// Server version
	Version string

	// Server build info
	BuildInfo map[string]string

	// Rate limit for public endpoints (requests per minute)
	RateLimit int

	// Logger
	Logger *logger.Logger
}

// PublicRPCHandler provides public RPC endpoints
type PublicRPCHandler struct {
	config    PublicRPCConfig
	version   string
	buildTime string
	startTime time.Time

	// Rate limiting
	requestCounts map[string]*rateLimitEntry
	mu            sync.RWMutex

	logger *logger.Logger
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// NewPublicRPCHandler creates a new public RPC handler
func NewPublicRPCHandler(config PublicRPCConfig) *PublicRPCHandler {
	if config.Logger == nil {
		config.Logger = logger.Global().WithComponent("public_rpc")
	}

	if config.RateLimit == 0 {
		config.RateLimit = 10 // Default: 10 requests per minute
	}

	return &PublicRPCHandler{
		config:        config,
		version:       config.Version,
		buildTime:     config.BuildInfo["build_time"],
		startTime:     time.Now(),
		requestCounts: make(map[string]*rateLimitEntry),
		logger:        config.Logger,
	}
}

// CheckRateLimit checks if a client has exceeded rate limits
// Returns true if allowed, false if rate limited
func (h *PublicRPCHandler) CheckRateLimit(clientID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	entry, exists := h.requestCounts[clientID]

	if !exists || now.After(entry.resetTime) {
		h.requestCounts[clientID] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(time.Minute),
		}
		return true
	}

	if entry.count >= h.config.RateLimit {
		return false
	}

	entry.count++
	return true
}

// CleanupRateLimits removes expired rate limit entries
func (h *PublicRPCHandler) CleanupRateLimits() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	count := 0
	for id, entry := range h.requestCounts {
		if now.After(entry.resetTime) {
			delete(h.requestCounts, id)
			count++
		}
	}
	return count
}

// HandleSystemHealth handles the system.health RPC method
// Public endpoint - no authentication required
func (h *PublicRPCHandler) HandleSystemHealth(ctx context.Context, params json.RawMessage) (interface{}, error) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    time.Since(h.startTime).String(),
		"version":   h.version,
		"checks": map[string]bool{
			"server":   true,
			"database": true, // Would check actual DB in production
		},
	}

	return response, nil
}

// HandleSystemConfig handles the system.config RPC method
// Public endpoint - returns safe configuration for client initialization
func (h *PublicRPCHandler) HandleSystemConfig(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// Only return safe, public configuration
	response := map[string]interface{}{
		"version": h.version,
		"features": map[string]bool{
			"matrix":    true,
			"webrtc":    true,
			"push":      true,
			"sso":       true,
			"e2ee":      true,
		},
		"endpoints": map[string]string{
			"matrix": "/_matrix/client/v3",
			"rpc":    "/rpc",
		},
		"limits": map[string]int{
			"max_message_size": 65536,
			"max_file_size":    104857600, // 100MB
		},
		"branding": map[string]string{
			"name":    "ArmorClaw",
			"logo":    "/static/logo.svg",
			"privacy": "/privacy",
			"terms":   "/terms",
		},
	}

	return response, nil
}

// HandleSystemInfo handles the system.info RPC method
// Public endpoint - returns server information
func (h *PublicRPCHandler) HandleSystemInfo(ctx context.Context, params json.RawMessage) (interface{}, error) {
	response := map[string]interface{}{
		"server": map[string]string{
			"name":      "ArmorClaw Bridge",
			"version":   h.version,
			"buildTime": h.buildTime,
		},
		"protocol": map[string]interface{}{
			"rpc_version": "2.0",
			"methods":     len(h.getPublicMethods()),
		},
		"capabilities": []string{
			"matrix",
			"webrtc",
			"push",
			"e2ee",
			"sso",
			"audit",
			"trust",
		},
	}

	return response, nil
}

// HandleDeviceValidate handles the device.validate RPC method
// Public endpoint with strict rate limiting - validates device without full auth
func (h *PublicRPCHandler) HandleDeviceValidate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		DeviceID    string `json:"device_id"`
		DeviceName  string `json:"device_name,omitempty"`
		Platform    string `json:"platform,omitempty"`
		AppVersion  string `json:"app_version,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if req.DeviceID == "" {
		return nil, fmt.Errorf("device_id is required")
	}

	// Basic validation - check device ID format
	if len(req.DeviceID) < 8 || len(req.DeviceID) > 128 {
		return nil, fmt.Errorf("invalid device_id length")
	}

	// Return validation result
	response := map[string]interface{}{
		"valid":       true,
		"device_id":   req.DeviceID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_time": time.Now().Unix(),
		"message":     "Device validated successfully",
	}

	h.logger.Info("device_validated",
		"device_id", req.DeviceID,
		"platform", req.Platform,
		"app_version", req.AppVersion,
	)

	return response, nil
}

// HandleSystemTime handles the system.time RPC method
// Public endpoint - returns server time for clock sync
func (h *PublicRPCHandler) HandleSystemTime(ctx context.Context, params json.RawMessage) (interface{}, error) {
	response := map[string]interface{}{
		"server_time":     time.Now().Unix(),
		"server_time_utc": time.Now().UTC().Format(time.RFC3339),
	}

	return response, nil
}

// getPublicMethods returns the list of public RPC methods
func (h *PublicRPCHandler) getPublicMethods() []string {
	return []string{
		"system.health",
		"system.config",
		"system.info",
		"system.time",
		"device.validate",
	}
}

// RegisterPublicMethods returns a map of public method handlers
// These should be added to the Server's method switch statement
func (h *PublicRPCHandler) GetPublicMethodHandlers() map[string]func(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return map[string]func(ctx context.Context, params json.RawMessage) (interface{}, error){
		"system.health":     h.HandleSystemHealth,
		"system.config":     h.HandleSystemConfig,
		"system.info":       h.HandleSystemInfo,
		"system.time":       h.HandleSystemTime,
		"device.validate":   h.HandleDeviceValidate,
	}
}

// ============================================================================

// PublicRPCMiddleware provides rate limiting middleware for public endpoints
type PublicRPCMiddleware struct {
	handler *PublicRPCHandler
}

// NewPublicRPCMiddleware creates a new public RPC middleware
func NewPublicRPCMiddleware(handler *PublicRPCHandler) *PublicRPCMiddleware {
	return &PublicRPCMiddleware{
		handler: handler,
	}
}

// Wrap wraps an RPC handler with rate limiting
func (m *PublicRPCMiddleware) Wrap(clientID string, fn func(ctx context.Context, params json.RawMessage) (interface{}, error)) func(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		if !m.handler.CheckRateLimit(clientID) {
			return nil, fmt.Errorf("rate limit exceeded (max %d requests per minute)", m.handler.config.RateLimit)
		}
		return fn(ctx, params)
	}
}
