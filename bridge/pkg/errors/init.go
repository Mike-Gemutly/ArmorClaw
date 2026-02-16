package errors

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// System is the main error handling system that coordinates all components
type System struct {
	config    Config
	registry  *SamplingRegistry
	resolver  *AdminResolver
	store     *ErrorStore
	notifier  *ErrorNotifier
	tracker   *ComponentTracker

	mu       sync.RWMutex
	started  bool
}

// Config configures the error handling system
type Config struct {
	// Store configuration
	StorePath      string
	RetentionDays  int

	// Sampling configuration
	RateLimitWindow string // e.g., "5m"
	RetentionPeriod string // e.g., "24h"

	// Admin configuration
	ConfigAdminMXID string
	SetupUserMXID   string
	AdminRoomID     string
	FallbackMXID    string

	// Matrix integration
	MatrixSender   MatrixMessageSender
	MatrixAdapter  MatrixAdminAdapter

	// Feature flags
	Enabled       bool
	StoreEnabled  bool
	NotifyEnabled bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		StorePath:       "/var/lib/armorclaw/errors.db",
		RetentionDays:   30,
		RateLimitWindow: "5m",
		RetentionPeriod: "24h",
		Enabled:         true,
		StoreEnabled:    true,
		NotifyEnabled:   true,
	}
}

// Initialize creates and initializes the error handling system
func Initialize(cfg Config) (*System, error) {
	if cfg.StorePath == "" {
		cfg.StorePath = "/var/lib/armorclaw/errors.db"
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}

	// Parse durations
	rateLimitWindow := parseDuration(cfg.RateLimitWindow, 5*60*1000) // 5 minutes default
	retentionPeriod := parseDuration(cfg.RetentionPeriod, 24*60*60*1000) // 24 hours default

	// Create sampling registry
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: rateLimitWindow,
		RetentionPeriod:  retentionPeriod,
	})

	// Create admin resolver
	resolver := NewAdminResolver(AdminConfig{
		ConfigAdminMXID: cfg.ConfigAdminMXID,
		SetupUserMXID:   cfg.SetupUserMXID,
		AdminRoomID:     cfg.AdminRoomID,
		MatrixAdapter:   cfg.MatrixAdapter,
		FallbackMXID:    cfg.FallbackMXID,
	})

	// Create store (optional)
	var store *ErrorStore
	var storeErr error
	if cfg.StoreEnabled {
		store, storeErr = NewErrorStore(StoreConfig{
			Path:          cfg.StorePath,
			RetentionDays: cfg.RetentionDays,
		})
		if storeErr != nil {
			return nil, fmt.Errorf("failed to create error store: %w", storeErr)
		}
	}

	// Create notifier
	notifier := NewErrorNotifier(NotifierConfig{
		Registry:     registry,
		Resolver:     resolver,
		Store:        store,
		MatrixSender: cfg.MatrixSender,
		Enabled:      cfg.Enabled && cfg.NotifyEnabled,
	})

	// Create component tracker for errors package itself
	tracker := GetComponentTracker("errors")

	system := &System{
		config:   cfg,
		registry: registry,
		resolver: resolver,
		store:    store,
		notifier: notifier,
		tracker:  tracker,
	}

	// Set as global
	SetGlobalRegistry(registry)
	SetGlobalAdminResolver(resolver)
	if store != nil {
		SetGlobalStore(store)
	}
	SetGlobalNotifier(notifier)

	return system, nil
}

// Start starts the error handling system
func (s *System) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	s.tracker.Event("system_start", nil)
	s.started = true

	return nil
}

// Stop stops the error handling system
func (s *System) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.tracker.Event("system_stop", nil)

	if s.store != nil {
		return s.store.Close()
	}

	s.started = false
	return nil
}

// Notify sends an error notification
func (s *System) Notify(ctx context.Context, err *TracedError) error {
	return s.notifier.Notify(ctx, err)
}

// NotifyQuick sends a quick error notification
func (s *System) NotifyQuick(ctx context.Context, code, message string, severity Severity) error {
	return s.notifier.NotifyQuick(ctx, code, message, severity)
}

// Store stores an error without notification
func (s *System) Store(ctx context.Context, err *TracedError) error {
	if s.store == nil {
		return fmt.Errorf("error store not configured")
	}
	return s.store.Store(ctx, err)
}

// Query queries stored errors
func (s *System) Query(ctx context.Context, q ErrorQuery) ([]StoredError, error) {
	if s.store == nil {
		return nil, fmt.Errorf("error store not configured")
	}
	return s.store.Query(ctx, q)
}

// Resolve marks an error as resolved
func (s *System) Resolve(ctx context.Context, traceID, resolvedBy string) error {
	if s.store == nil {
		return fmt.Errorf("error store not configured")
	}
	return s.store.Resolve(ctx, traceID, resolvedBy)
}

// Stats returns system statistics
func (s *System) Stats(ctx context.Context) SystemStats {
	stats := SystemStats{
		Sampling: s.registry.Stats(),
	}

	if s.store != nil {
		storeStats, _ := s.store.Stats(ctx)
		stats.Store = &storeStats
	}

	stats.AdminSource = s.resolver.GetConfigAdmin()
	if stats.AdminSource == "" {
		stats.AdminSource = s.resolver.GetSetupUser()
	}
	if stats.AdminSource == "" {
		stats.AdminSource = s.resolver.GetAdminRoom()
	}

	return stats
}

// SystemStats holds statistics about the error system
type SystemStats struct {
	Sampling    SamplingStats `json:"sampling"`
	Store       *StoreStats   `json:"store,omitempty"`
	AdminSource string        `json:"admin_source"`
}

// SetMatrixSender updates the Matrix sender
func (s *System) SetMatrixSender(sender MatrixMessageSender) {
	s.notifier.SetMatrixSender(sender)
}

// SetMatrixAdapter updates the Matrix adapter for admin resolution
func (s *System) SetMatrixAdapter(adapter MatrixAdminAdapter) {
	s.resolver.SetMatrixAdapter(adapter)
}

// SetAdminMXID sets the admin MXID
func (s *System) SetAdminMXID(mxid string) {
	s.resolver.SetConfigAdmin(mxid)
}

// SetSetupUser sets the setup user MXID
func (s *System) SetSetupUser(mxid string) {
	s.resolver.SetSetupUser(mxid)
}

// SetAdminRoom sets the admin room ID
func (s *System) SetAdminRoom(roomID string) {
	s.resolver.SetAdminRoom(roomID)
}

// SetEnabled enables or disables notifications
func (s *System) SetEnabled(enabled bool) {
	s.notifier.SetEnabled(enabled)
}

// IsEnabled returns whether notifications are enabled
func (s *System) IsEnabled() bool {
	return s.notifier.IsEnabled()
}

// GetRegistry returns the sampling registry
func (s *System) GetRegistry() *SamplingRegistry {
	return s.registry
}

// GetResolver returns the admin resolver
func (s *System) GetResolver() *AdminResolver {
	return s.resolver
}

// GetStore returns the error store
func (s *System) GetStore() *ErrorStore {
	return s.store
}

// GetNotifier returns the notifier
func (s *System) GetNotifier() *ErrorNotifier {
	return s.notifier
}

// Cleanup runs cleanup on old resolved errors
func (s *System) Cleanup(ctx context.Context) (int64, error) {
	if s.store == nil {
		return 0, nil
	}
	return s.store.Cleanup(ctx)
}

// Global system instance
var globalSystem *System
var globalSystemMu sync.RWMutex

// SetGlobalSystem sets the global error system
func SetGlobalSystem(system *System) {
	globalSystemMu.Lock()
	defer globalSystemMu.Unlock()
	globalSystem = system
}

// GetGlobalSystem returns the global error system
func GetGlobalSystem() *System {
	globalSystemMu.RLock()
	defer globalSystemMu.RUnlock()
	return globalSystem
}

// Quick helper functions using global system

// Report creates and notifies an error in one call
func Report(ctx context.Context, code string, cause error) error {
	err := Wrap(code, cause)
	return GlobalNotify(ctx, err)
}

// Reportf creates and notifies an error with formatted message
func Reportf(ctx context.Context, code string, format string, args ...interface{}) error {
	err := Newf(code, format, args...)
	return GlobalNotify(ctx, err)
}

// Track tracks an event for the current component
func Track(eventType string, data interface{}) {
	TrackEvent("errors", eventType, data)
}

// Helper to parse duration strings
func parseDuration(s string, defaultMs int64) time.Duration {
	if s == "" {
		return time.Duration(defaultMs) * time.Millisecond
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Duration(defaultMs) * time.Millisecond
	}
	return d
}
