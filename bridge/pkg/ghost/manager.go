// Package ghost provides Ghost User lifecycle management for bridged users.
//
// Resolves: Gap - Ghost User Lifecycle Management
//
// Handles creation, tracking, and deactivation of Matrix "Ghost Users"
// that represent external platform users (Slack, Discord, etc.)
package ghost

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// UserEvent represents a user lifecycle event from an external platform
type UserEvent struct {
	Platform   string    // "slack", "discord", "teams", etc.
	UserID     string    // Platform-specific user ID
	EventType  EventType // Type of lifecycle event
	Timestamp  time.Time
	Attributes map[string]string // Optional: display_name, email, etc.
}

// EventType defines the type of user lifecycle event
type EventType int

const (
	EventUserJoined EventType = iota
	EventUserLeft
	EventUserUpdated
	EventUserDeleted
)

// String returns the string representation of the event type
func (e EventType) String() string {
	switch e {
	case EventUserJoined:
		return "user_joined"
	case EventUserLeft:
		return "user_left"
	case EventUserUpdated:
		return "user_updated"
	case EventUserDeleted:
		return "user_deleted"
	default:
		return "unknown"
	}
}

// GhostUser represents a Matrix ghost user for an external platform user
type GhostUser struct {
	MatrixUserID   string    // Full Matrix user ID (@platform_user:homeserver)
	Platform       string    // Source platform
	PlatformUserID string    // Platform-specific user ID
	DisplayName    string    // Current display name
	Active         bool      // Whether user is still active on source platform
	CreatedAt      time.Time
	DeactivatedAt  *time.Time
	LastSyncedAt   time.Time
}

// MatrixClient interface for Matrix operations (dependency injection)
type MatrixClient interface {
	// DeactivateAccount deactivates a Matrix user account
	DeactivateAccount(ctx context.Context, userID string) error

	// UpdateDisplayName updates a user's display name
	UpdateDisplayName(ctx context.Context, userID, displayName string) error

	// GetGhostUsers returns all ghost users for a platform
	GetGhostUsers(ctx context.Context, platform string) ([]GhostUser, error)

	// CreateGhostUser creates a new ghost user (via AppService)
	CreateGhostUser(ctx context.Context, platform, platformUserID, displayName string) (string, error)
}

// PlatformClient interface for external platform operations
type PlatformClient interface {
	// GetUserList returns current active users on the platform
	GetUserList(ctx context.Context) ([]PlatformUser, error)

	// GetUser returns a specific user by ID
	GetUser(ctx context.Context, userID string) (*PlatformUser, error)
}

// PlatformUser represents a user on an external platform
type PlatformUser struct {
	ID          string
	DisplayName string
	Email       string
	Active      bool
}

// Storage interface for ghost user persistence
type Storage interface {
	// UpsertGhostUser creates or updates a ghost user record
	UpsertGhostUser(ctx context.Context, user *GhostUser) error

	// GetGhostUser retrieves a ghost user by platform and platform user ID
	GetGhostUser(ctx context.Context, platform, platformUserID string) (*GhostUser, error)

	// ListGhostUsers lists all ghost users for a platform
	ListGhostUsers(ctx context.Context, platform string) ([]GhostUser, error)

	// ListActiveGhostUsers lists all active ghost users for a platform
	ListActiveGhostUsers(ctx context.Context, platform string) ([]GhostUser, error)

	// MarkDeactivated marks a ghost user as deactivated
	MarkDeactivated(ctx context.Context, matrixUserID string) error
}

// Manager handles ghost user lifecycle operations
type Manager struct {
	logger        *slog.Logger
	matrixClient  MatrixClient
	storage       Storage
	platforms     map[string]PlatformClient
	mu            sync.RWMutex
	syncInterval  time.Duration
	stopSync      chan struct{}
	syncRunning   bool
}

// Config for the Ghost User Manager
type Config struct {
	Logger       *slog.Logger
	MatrixClient MatrixClient
	Storage      Storage
	SyncInterval time.Duration // How often to sync user rosters (default: 24h)
}

// NewManager creates a new Ghost User Manager
func NewManager(config Config) *Manager {
	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "ghost_manager")
	}
	if config.SyncInterval == 0 {
		config.SyncInterval = 24 * time.Hour
	}

	return &Manager{
		logger:       config.Logger,
		matrixClient: config.MatrixClient,
		storage:      config.Storage,
		platforms:    make(map[string]PlatformClient),
		syncInterval: config.SyncInterval,
		stopSync:     make(chan struct{}),
	}
}

// RegisterPlatform registers a platform client for user sync
func (m *Manager) RegisterPlatform(platform string, client PlatformClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.platforms[platform] = client
}

// HandleUserEvent processes a user lifecycle event from an external platform
func (m *Manager) HandleUserEvent(ctx context.Context, event UserEvent) error {
	m.logger.Info("handling_user_event",
		"platform", event.Platform,
		"user_id", event.UserID,
		"event_type", event.EventType,
	)

	switch event.EventType {
	case EventUserJoined:
		return m.handleUserJoined(ctx, event)
	case EventUserLeft:
		return m.handleUserLeft(ctx, event)
	case EventUserUpdated:
		return m.handleUserUpdated(ctx, event)
	case EventUserDeleted:
		return m.handleUserDeleted(ctx, event)
	default:
		return fmt.Errorf("unknown event type: %v", event.EventType)
	}
}

// handleUserJoined creates or reactivates a ghost user
func (m *Manager) handleUserJoined(ctx context.Context, event UserEvent) error {
	// Check if ghost user already exists
	existing, err := m.storage.GetGhostUser(ctx, event.Platform, event.UserID)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	displayName := event.Attributes["display_name"]
	if displayName == "" {
		displayName = event.UserID
	}

	if existing != nil {
		// Reactivate if previously deactivated
		if !existing.Active {
			existing.Active = true
			existing.DeactivatedAt = nil
			existing.DisplayName = displayName
			existing.LastSyncedAt = time.Now()
			if err := m.storage.UpsertGhostUser(ctx, existing); err != nil {
				return fmt.Errorf("failed to reactivate user: %w", err)
			}
			m.logger.Info("ghost_user_reactivated",
				"matrix_user_id", existing.MatrixUserID,
				"platform", event.Platform,
			)
		}
		return nil
	}

	// Create new ghost user via AppService
	matrixUserID, err := m.matrixClient.CreateGhostUser(ctx, event.Platform, event.UserID, displayName)
	if err != nil {
		return fmt.Errorf("failed to create ghost user: %w", err)
	}

	ghostUser := &GhostUser{
		MatrixUserID:   matrixUserID,
		Platform:       event.Platform,
		PlatformUserID: event.UserID,
		DisplayName:    displayName,
		Active:         true,
		CreatedAt:      time.Now(),
		LastSyncedAt:   time.Now(),
	}

	if err := m.storage.UpsertGhostUser(ctx, ghostUser); err != nil {
		return fmt.Errorf("failed to store ghost user: %w", err)
	}

	m.logger.Info("ghost_user_created",
		"matrix_user_id", matrixUserID,
		"platform", event.Platform,
		"platform_user_id", event.UserID,
	)

	return nil
}

// handleUserLeft deactivates a ghost user when they leave the source platform
func (m *Manager) handleUserLeft(ctx context.Context, event UserEvent) error {
	ghostUser, err := m.storage.GetGhostUser(ctx, event.Platform, event.UserID)
	if err != nil {
		return fmt.Errorf("failed to get ghost user: %w", err)
	}
	if ghostUser == nil {
		// User never existed, nothing to do
		return nil
	}

	// Update display name to indicate left status
	newDisplayName := fmt.Sprintf("%s [Left %s]", ghostUser.DisplayName, strings.Title(event.Platform))
	if err := m.matrixClient.UpdateDisplayName(ctx, ghostUser.MatrixUserID, newDisplayName); err != nil {
		m.logger.Warn("failed_to_update_display_name",
			"error", err,
			"matrix_user_id", ghostUser.MatrixUserID,
		)
		// Continue with deactivation even if display name update fails
	}

	// Deactivate the Matrix account
	if err := m.matrixClient.DeactivateAccount(ctx, ghostUser.MatrixUserID); err != nil {
		return fmt.Errorf("failed to deactivate matrix account: %w", err)
	}

	// Mark as deactivated in storage
	if err := m.storage.MarkDeactivated(ctx, ghostUser.MatrixUserID); err != nil {
		return fmt.Errorf("failed to mark deactivated: %w", err)
	}

	m.logger.Info("ghost_user_deactivated",
		"matrix_user_id", ghostUser.MatrixUserID,
		"platform", event.Platform,
		"platform_user_id", event.UserID,
	)

	return nil
}

// handleUserUpdated updates ghost user attributes
func (m *Manager) handleUserUpdated(ctx context.Context, event UserEvent) error {
	ghostUser, err := m.storage.GetGhostUser(ctx, event.Platform, event.UserID)
	if err != nil {
		return fmt.Errorf("failed to get ghost user: %w", err)
	}
	if ghostUser == nil {
		// User doesn't exist, treat as join
		return m.handleUserJoined(ctx, event)
	}

	displayName := event.Attributes["display_name"]
	if displayName != "" && displayName != ghostUser.DisplayName {
		if err := m.matrixClient.UpdateDisplayName(ctx, ghostUser.MatrixUserID, displayName); err != nil {
			return fmt.Errorf("failed to update display name: %w", err)
		}
		ghostUser.DisplayName = displayName
	}

	ghostUser.LastSyncedAt = time.Now()
	if err := m.storage.UpsertGhostUser(ctx, ghostUser); err != nil {
		return fmt.Errorf("failed to update ghost user: %w", err)
	}

	return nil
}

// handleUserDeleted permanently removes a ghost user
func (m *Manager) handleUserDeleted(ctx context.Context, event UserEvent) error {
	// Deletion is treated the same as leaving for safety
	// (we keep message history, just deactivate the account)
	return m.handleUserLeft(ctx, event)
}

// StartSync begins periodic user roster synchronization
func (m *Manager) StartSync() {
	m.mu.Lock()
	if m.syncRunning {
		m.mu.Unlock()
		return
	}
	m.syncRunning = true
	m.mu.Unlock()

	ticker := time.NewTicker(m.syncInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				if err := m.SyncAllPlatforms(ctx); err != nil {
					m.logger.Error("sync_failed", "error", err)
				}
				cancel()
			case <-m.stopSync:
				m.logger.Info("sync_stopped")
				return
			}
		}
	}()

	m.logger.Info("sync_started", "interval", m.syncInterval)
}

// StopSync stops the periodic sync
func (m *Manager) StopSync() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.syncRunning {
		close(m.stopSync)
		m.syncRunning = false
	}
}

// SyncAllPlatforms synchronizes user rosters for all registered platforms
func (m *Manager) SyncAllPlatforms(ctx context.Context) error {
	m.mu.RLock()
	platforms := make([]string, 0, len(m.platforms))
	for p := range m.platforms {
		platforms = append(platforms, p)
	}
	m.mu.RUnlock()

	for _, platform := range platforms {
		if err := m.SyncPlatform(ctx, platform); err != nil {
			m.logger.Error("platform_sync_failed",
				"platform", platform,
				"error", err,
			)
			// Continue with other platforms
		}
	}

	return nil
}

// SyncPlatform synchronizes the user roster for a specific platform
func (m *Manager) SyncPlatform(ctx context.Context, platform string) error {
	m.mu.RLock()
	client, ok := m.platforms[platform]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("platform %s not registered", platform)
	}

	m.logger.Info("syncing_platform", "platform", platform)

	// Get current users from platform
	platformUsers, err := client.GetUserList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get platform user list: %w", err)
	}

	// Build a map of active platform users
	activeUsers := make(map[string]bool)
	for _, u := range platformUsers {
		if u.Active {
			activeUsers[u.ID] = true
		}
	}

	// Get all ghost users for this platform
	ghostUsers, err := m.storage.ListActiveGhostUsers(ctx, platform)
	if err != nil {
		return fmt.Errorf("failed to list ghost users: %w", err)
	}

	// Find orphaned ghost users (exist on Matrix but not on platform)
	var deactivatedCount int
	for _, ghost := range ghostUsers {
		if !activeUsers[ghost.PlatformUserID] {
			m.logger.Info("deactivating_orphaned_user",
				"matrix_user_id", ghost.MatrixUserID,
				"platform_user_id", ghost.PlatformUserID,
			)

			event := UserEvent{
				Platform:       platform,
				UserID:         ghost.PlatformUserID,
				EventType:      EventUserLeft,
				Timestamp:      time.Now(),
				Attributes:     map[string]string{"display_name": ghost.DisplayName},
			}

			if err := m.handleUserLeft(ctx, event); err != nil {
				m.logger.Error("failed_to_deactivate_orphan",
					"matrix_user_id", ghost.MatrixUserID,
					"error", err,
				)
			} else {
				deactivatedCount++
			}
		}
	}

	m.logger.Info("platform_sync_complete",
		"platform", platform,
		"platform_users", len(activeUsers),
		"ghost_users", len(ghostUsers),
		"deactivated", deactivatedCount,
	)

	return nil
}

// GetGhostUser retrieves a ghost user by platform and platform user ID
func (m *Manager) GetGhostUser(ctx context.Context, platform, platformUserID string) (*GhostUser, error) {
	return m.storage.GetGhostUser(ctx, platform, platformUserID)
}

// ListGhostUsers lists all ghost users for a platform
func (m *Manager) ListGhostUsers(ctx context.Context, platform string) ([]GhostUser, error) {
	return m.storage.ListGhostUsers(ctx, platform)
}
