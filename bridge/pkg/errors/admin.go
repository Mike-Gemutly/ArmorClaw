package errors

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// AdminTarget represents a resolved admin recipient
type AdminTarget struct {
	MXID   string `json:"mxid"`
	Source string `json:"source"` // "config", "setup", "room", "fallback"
}

// AdminResolver determines the admin recipient for error notifications
type AdminResolver struct {
	mu sync.RWMutex

	// Resolution chain (priority order)
	configAdminMXID string              // 1. Explicit config override
	setupUserMXID   string              // 2. User who ran initial setup
	adminRoomID     string              // 3. Admin room for member lookup

	// Matrix adapter for room membership queries
	matrixAdapter MatrixAdminAdapter

	// Fallback (used if all else fails)
	fallbackMXID string

	// Cache
	cachedTarget *AdminTarget
	cacheExpiry  time.Time
	cacheTTL     time.Duration
}

// MatrixAdminAdapter is the interface needed for admin resolution from room membership
type MatrixAdminAdapter interface {
	GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error)
}

// RoomMember represents a member of a Matrix room
type RoomMember struct {
	UserID     string `json:"user_id"`
	PowerLevel int    `json:"power_level"`
	Display    string `json:"display_name,omitempty"`
}

// AdminConfig configures the admin resolver
type AdminConfig struct {
	ConfigAdminMXID string             // From armorclaw.toml
	SetupUserMXID   string             // Captured during first-run wizard
	AdminRoomID     string             // From notifier config
	MatrixAdapter   MatrixAdminAdapter // For room membership lookup
	FallbackMXID    string             // Last resort fallback
	CacheTTL        time.Duration      // How long to cache resolved admin
}

// DefaultAdminConfig returns default configuration
func DefaultAdminConfig() AdminConfig {
	return AdminConfig{
		CacheTTL: 5 * time.Minute,
	}
}

// NewAdminResolver creates a new admin resolver
func NewAdminResolver(cfg AdminConfig) *AdminResolver {
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}

	return &AdminResolver{
		configAdminMXID: cfg.ConfigAdminMXID,
		setupUserMXID:   cfg.SetupUserMXID,
		adminRoomID:     cfg.AdminRoomID,
		matrixAdapter:   cfg.MatrixAdapter,
		fallbackMXID:    cfg.FallbackMXID,
		cacheTTL:        cfg.CacheTTL,
	}
}

// Resolve determines the admin recipient using the fallback chain
func (r *AdminResolver) Resolve(ctx context.Context) (*AdminTarget, error) {
	return r.ResolveWithContext(ctx)
}

// ResolveWithContext resolves the admin with context for room membership queries
func (r *AdminResolver) ResolveWithContext(ctx context.Context) (*AdminTarget, error) {
	r.mu.RLock()
	// Check cache
	if r.cachedTarget != nil && time.Now().Before(r.cacheExpiry) {
		defer r.mu.RUnlock()
		return r.cachedTarget, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check cache after acquiring write lock
	if r.cachedTarget != nil && time.Now().Before(r.cacheExpiry) {
		return r.cachedTarget, nil
	}

	// Try resolution chain
	target, err := r.resolveChain(ctx)
	if err != nil {
		return nil, err
	}

	// Cache result
	r.cachedTarget = target
	r.cacheExpiry = time.Now().Add(r.cacheTTL)

	return target, nil
}

// resolveChain attempts each resolution method in priority order
func (r *AdminResolver) resolveChain(ctx context.Context) (*AdminTarget, error) {
	// 1. Check explicit config override
	if r.configAdminMXID != "" {
		return &AdminTarget{
			MXID:   r.configAdminMXID,
			Source: "config",
		}, nil
	}

	// 2. Fallback to first setup user
	if r.setupUserMXID != "" {
		return &AdminTarget{
			MXID:   r.setupUserMXID,
			Source: "setup",
		}, nil
	}

	// 3. Fallback to admin room members
	if r.adminRoomID != "" && r.matrixAdapter != nil {
		members, err := r.matrixAdapter.GetRoomMembers(ctx, r.adminRoomID)
		if err == nil && len(members) > 0 {
			// Find first admin/moderator (power level >= 50)
			for _, m := range members {
				if m.PowerLevel >= 50 {
					return &AdminTarget{
						MXID:   m.UserID,
						Source: "room",
					}, nil
				}
			}
			// Fall back to first member if no admin/moderator
			if len(members) > 0 {
				return &AdminTarget{
					MXID:   members[0].UserID,
					Source: "room",
				}, nil
			}
		}
	}

	// 4. Last resort fallback
	if r.fallbackMXID != "" {
		return &AdminTarget{
			MXID:   r.fallbackMXID,
			Source: "fallback",
		}, nil
	}

	return nil, fmt.Errorf("no admin target could be resolved")
}

// SetConfigAdmin sets the admin from config
func (r *AdminResolver) SetConfigAdmin(mxid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configAdminMXID = mxid
	r.invalidateCache()
}

// SetSetupUser sets the admin from setup wizard
func (r *AdminResolver) SetSetupUser(mxid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setupUserMXID = mxid
	r.invalidateCache()
}

// SetAdminRoom sets the admin room ID
func (r *AdminResolver) SetAdminRoom(roomID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adminRoomID = roomID
	r.invalidateCache()
}

// SetMatrixAdapter sets the Matrix adapter for room membership queries
func (r *AdminResolver) SetMatrixAdapter(adapter MatrixAdminAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matrixAdapter = adapter
	r.invalidateCache()
}

// SetFallback sets the fallback admin MXID
func (r *AdminResolver) SetFallback(mxid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbackMXID = mxid
	r.invalidateCache()
}

// InvalidateCache clears the cached admin target
func (r *AdminResolver) InvalidateCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invalidateCache()
}

func (r *AdminResolver) invalidateCache() {
	r.cachedTarget = nil
	r.cacheExpiry = time.Time{}
}

// GetConfigAdmin returns the configured admin MXID
func (r *AdminResolver) GetConfigAdmin() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configAdminMXID
}

// GetSetupUser returns the setup user MXID
func (r *AdminResolver) GetSetupUser() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.setupUserMXID
}

// GetAdminRoom returns the admin room ID
func (r *AdminResolver) GetAdminRoom() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adminRoomID
}

// SetupUserStorage persists the setup user to disk
type SetupUserStorage struct {
	path string
	mu   sync.RWMutex
}

// NewSetupUserStorage creates storage for setup user
func NewSetupUserStorage(path string) *SetupUserStorage {
	return &SetupUserStorage{path: path}
}

// Save persists the setup user MXID
func (s *SetupUserStorage) Save(mxid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create directory if needed
	dir := s.path[:len(s.path)-len("/setup-user")]
	if lastSlash := len(s.path) - 1; lastSlash > 0 {
		for i := lastSlash - 1; i >= 0; i-- {
			if s.path[i] == '/' || s.path[i] == '\\' {
				dir = s.path[:i]
				break
			}
		}
	}
	_ = dir // Directory creation handled externally

	// Write MXID to file
	data := []byte(mxid)
	return os.WriteFile(s.path, data, 0600)
}

// Load retrieves the setup user MXID
func (s *SetupUserStorage) Load() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Exists checks if setup user is stored
func (s *SetupUserStorage) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.path)
	return err == nil
}

// DefaultSetupUserPath returns the default path for setup user storage
func DefaultSetupUserPath() string {
	return "/var/lib/armorclaw/setup-user"
}

// Global admin resolver
var globalAdminResolver *AdminResolver
var globalAdminResolverMu sync.RWMutex

func init() {
	globalAdminResolver = NewAdminResolver(DefaultAdminConfig())
}

// GetGlobalAdminResolver returns the global admin resolver
func GetGlobalAdminResolver() *AdminResolver {
	globalAdminResolverMu.RLock()
	defer globalAdminResolverMu.RUnlock()
	return globalAdminResolver
}

// SetGlobalAdminResolver sets the global admin resolver
func SetGlobalAdminResolver(resolver *AdminResolver) {
	globalAdminResolverMu.Lock()
	defer globalAdminResolverMu.Unlock()
	globalAdminResolver = resolver
}

// ResolveAdmin resolves the admin using the global resolver
func ResolveAdmin(ctx context.Context) (*AdminTarget, error) {
	return GetGlobalAdminResolver().ResolveWithContext(ctx)
}

// SetGlobalSetupUser sets the setup user in the global resolver
func SetGlobalSetupUser(mxid string) {
	GetGlobalAdminResolver().SetSetupUser(mxid)
}

// SetGlobalConfigAdmin sets the config admin in the global resolver
func SetGlobalConfigAdmin(mxid string) {
	GetGlobalAdminResolver().SetConfigAdmin(mxid)
}

// SetGlobalAdminRoom sets the admin room in the global resolver
func SetGlobalAdminRoom(roomID string) {
	GetGlobalAdminResolver().SetAdminRoom(roomID)
}

// SetGlobalMatrixAdapter sets the Matrix adapter in the global resolver
func SetGlobalMatrixAdapter(adapter MatrixAdminAdapter) {
	GetGlobalAdminResolver().SetMatrixAdapter(adapter)
}
