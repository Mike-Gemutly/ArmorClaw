// Package push provides push notification gateway for ArmorClaw.
// It integrates with Matrix Sygnal for multi-platform push notifications.
package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// Platform represents a push notification platform
type Platform string

const (
	PlatformFCM      Platform = "fcm"       // Firebase Cloud Messaging (Android/iOS)
	PlatformAPNS     Platform = "apns"      // Apple Push Notification Service
	PlatformWebPush  Platform = "webpush"   // Web Push (VAPID)
	PlatformUnified  Platform = "unified"   // Unified Push
)

// Priority levels for notifications
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// Notification represents a push notification payload
type Notification struct {
	ID          string                 `json:"id"`
	Platform    Platform               `json:"platform"`
	DeviceToken string                 `json:"device_token"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Priority    Priority               `json:"priority"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Badge       int                    `json:"badge,omitempty"`
	Sound       string                 `json:"sound,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Tag         string                 `json:"tag,omitempty"`
	Actions     []NotificationAction   `json:"actions,omitempty"`
	ExpiresAt   time.Time              `json:"expires_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// NotificationAction represents an interactive notification action
type NotificationAction struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Icon    string `json:"icon,omitempty"`
	Trigger string `json:"trigger,omitempty"` // "tap", "background"
}

// DeviceRegistration represents a registered push device
type DeviceRegistration struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Platform     Platform  `json:"platform"`
	DeviceToken  string    `json:"device_token"`
	DeviceType   string    `json:"device_type"`   // "ios", "android", "web"
	DeviceName   string    `json:"device_name"`   // User-friendly name
	AppID        string    `json:"app_id"`        // Application identifier
	PushKey      string    `json:"push_key"`      // Push key for encryption
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastPushAt   time.Time `json:"last_push_at"`
	LastError    string    `json:"last_error,omitempty"`
}

// PushResult represents the result of a push attempt
type PushResult struct {
	NotificationID string    `json:"notification_id"`
	DeviceID       string    `json:"device_id"`
	Success        bool      `json:"success"`
	Error          string    `json:"error,omitempty"`
	DeliveredAt    time.Time `json:"delivered_at,omitempty"`
}

// Config configures the push gateway
type Config struct {
	// FCM configuration
	FCMEnabled    bool
	FCMServerKey  string
	FCMProjectID  string

	// APNS configuration
	APNSEnabled     bool
	APNSCertFile    string
	APNSKeyFile     string
	APNSTopic       string
	APNSEnvironment string // "production" or "sandbox"

	// Web Push configuration
	WebPushEnabled  bool
	WebPushVAPIDKey string
	WebPushSubject  string
	WebPushEmail    string

	// Rate limiting
	MaxNotificationsPerMinute int
	MaxRetries                int
	RetryDelay                time.Duration

	// Sygnal integration
	SygnalURL    string
	SygnalAPIKey string

	// Logger
	Logger *slog.Logger
}

// Gateway manages push notifications for all platforms
type Gateway struct {
	config    Config
	devices   map[string]*DeviceRegistration
	userDevices map[string][]string // user_id -> device_ids
	results   []*PushResult
	mu        sync.RWMutex
	logger    *slog.Logger
	providers map[Platform]PushProvider
}

// PushProvider defines the interface for push notification providers
type PushProvider interface {
	Send(ctx context.Context, notification *Notification) (*PushResult, error)
	ValidateToken(token string) bool
	Platform() Platform
}

// NewGateway creates a new push notification gateway
func NewGateway(config Config) (*Gateway, error) {
	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "push_gateway")
	}

	if config.MaxNotificationsPerMinute == 0 {
		config.MaxNotificationsPerMinute = 100
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	gw := &Gateway{
		config:      config,
		devices:     make(map[string]*DeviceRegistration),
		userDevices: make(map[string][]string),
		results:     make([]*PushResult, 0),
		logger:      config.Logger,
		providers:   make(map[Platform]PushProvider),
	}

	// Initialize providers
	if config.FCMEnabled {
		gw.providers[PlatformFCM] = NewFCMProvider(config.FCMServerKey, config.FCMProjectID)
	}

	if config.APNSEnabled {
		gw.providers[PlatformAPNS] = NewAPNSProvider(config.APNSCertFile, config.APNSKeyFile, config.APNSTopic, config.APNSEnvironment)
	}

	if config.WebPushEnabled {
		gw.providers[PlatformWebPush] = NewWebPushProvider(config.WebPushVAPIDKey, config.WebPushSubject, config.WebPushEmail)
	}

	gw.logger.Info("push_gateway_initialized",
		"fcm_enabled", config.FCMEnabled,
		"apns_enabled", config.APNSEnabled,
		"webpush_enabled", config.WebPushEnabled,
	)

	return gw, nil
}

// RegisterDevice registers a device for push notifications
func (g *Gateway) RegisterDevice(userID string, platform Platform, deviceToken, deviceType, deviceName string) (*DeviceRegistration, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Generate device ID
	deviceID := generateDeviceID()

	// Check for existing device with same token
	for _, dev := range g.devices {
		if dev.DeviceToken == deviceToken && dev.UserID == userID {
			// Update existing device
			dev.UpdatedAt = time.Now()
			dev.Enabled = true
			g.logger.Info("device_reregistered", "device_id", dev.ID, "user_id", userID)
			return dev, nil
		}
	}

	// Create new registration
	device := &DeviceRegistration{
		ID:          deviceID,
		UserID:      userID,
		Platform:    platform,
		DeviceToken: deviceToken,
		DeviceType:  deviceType,
		DeviceName:  deviceName,
		AppID:       "com.armorclaw.bridge",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	g.devices[deviceID] = device
	g.userDevices[userID] = append(g.userDevices[userID], deviceID)

	g.logger.Info("device_registered",
		"device_id", deviceID,
		"user_id", userID,
		"platform", platform,
		"device_type", deviceType,
	)

	return device, nil
}

// UnregisterDevice removes a device registration
func (g *Gateway) UnregisterDevice(deviceID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	device, exists := g.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Remove from user devices list
	userDevs := g.userDevices[device.UserID]
	for i, id := range userDevs {
		if id == deviceID {
			g.userDevices[device.UserID] = append(userDevs[:i], userDevs[i+1:]...)
			break
		}
	}

	// Delete device
	delete(g.devices, deviceID)

	g.logger.Info("device_unregistered", "device_id", deviceID)
	return nil
}

// SendToUser sends a notification to all devices for a user
func (g *Gateway) SendToUser(ctx context.Context, userID string, notification *Notification) ([]*PushResult, error) {
	g.mu.RLock()
	deviceIDs := g.userDevices[userID]
	g.mu.RUnlock()

	if len(deviceIDs) == 0 {
		return nil, fmt.Errorf("no devices registered for user: %s", userID)
	}

	results := make([]*PushResult, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		device, exists := g.devices[deviceID]
		if !exists || !device.Enabled {
			continue
		}

		// Create device-specific notification
		deviceNotif := *notification
		deviceNotif.Platform = device.Platform
		deviceNotif.DeviceToken = device.DeviceToken
		deviceNotif.ID = generateNotificationID()

		result, err := g.Send(ctx, &deviceNotif)
		if err != nil {
			g.logger.Warn("push_failed",
				"device_id", deviceID,
				"error", err,
			)
			results = append(results, &PushResult{
				NotificationID: deviceNotif.ID,
				DeviceID:       deviceID,
				Success:        false,
				Error:          err.Error(),
			})
			continue
		}

		results = append(results, result)

		// Update device last push time
		g.mu.Lock()
		if dev, ok := g.devices[deviceID]; ok {
			dev.LastPushAt = time.Now()
		}
		g.mu.Unlock()
	}

	return results, nil
}

// Send sends a notification through the appropriate provider
func (g *Gateway) Send(ctx context.Context, notification *Notification) (*PushResult, error) {
	provider, exists := g.providers[notification.Platform]
	if !exists {
		return nil, fmt.Errorf("no provider for platform: %s", notification.Platform)
	}

	// Validate token
	if !provider.ValidateToken(notification.DeviceToken) {
		return nil, fmt.Errorf("invalid device token for platform: %s", notification.Platform)
	}

	// Set defaults
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	if notification.Priority == "" {
		notification.Priority = PriorityNormal
	}

	// Send with retries
	var lastErr error
	for i := 0; i < g.config.MaxRetries; i++ {
		result, err := provider.Send(ctx, notification)
		if err == nil {
			g.mu.Lock()
			g.results = append(g.results, result)
			// Keep last 1000 results
			if len(g.results) > 1000 {
				g.results = g.results[len(g.results)-1000:]
			}
			g.mu.Unlock()
			return result, nil
		}

		lastErr = err
		g.logger.Warn("push_retry",
			"attempt", i+1,
			"error", err,
			"notification_id", notification.ID,
		)

		// Wait before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(g.config.RetryDelay):
		}
	}

	return nil, fmt.Errorf("push failed after %d retries: %w", g.config.MaxRetries, lastErr)
}

// GetUserDevices returns all devices for a user
func (g *Gateway) GetUserDevices(userID string) []*DeviceRegistration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	deviceIDs := g.userDevices[userID]
	devices := make([]*DeviceRegistration, 0, len(deviceIDs))

	for _, id := range deviceIDs {
		if dev, ok := g.devices[id]; ok {
			devices = append(devices, dev)
		}
	}

	return devices
}

// GetDevice returns a device by ID
func (g *Gateway) GetDevice(deviceID string) (*DeviceRegistration, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	device, exists := g.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}
	return device, nil
}

// GetStats returns gateway statistics
func (g *Gateway) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	platformCounts := make(map[Platform]int)
	for _, dev := range g.devices {
		platformCounts[dev.Platform]++
	}

	successCount := 0
	failCount := 0
	for _, r := range g.results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	return map[string]interface{}{
		"total_devices":       len(g.devices),
		"total_users":         len(g.userDevices),
		"devices_by_platform": platformCounts,
		"providers_enabled":   len(g.providers),
		"notifications_sent":  successCount,
		"notifications_failed": failCount,
		"fcm_enabled":         g.config.FCMEnabled,
		"apns_enabled":        g.config.APNSEnabled,
		"webpush_enabled":     g.config.WebPushEnabled,
	}
}

// CreateMatrixPushNotification creates a push notification from Matrix event
func (g *Gateway) CreateMatrixPushNotification(roomID, eventID, sender, content string, devices []*DeviceRegistration) []*Notification {
	notifications := make([]*Notification, 0, len(devices))

	for _, device := range devices {
		notif := &Notification{
			ID:          generateNotificationID(),
			Platform:    device.Platform,
			DeviceToken: device.DeviceToken,
			Title:       formatSenderName(sender),
			Body:        truncateContent(content, 200),
			Priority:    PriorityHigh,
			CreatedAt:   time.Now(),
			Data: map[string]interface{}{
				"room_id":  roomID,
				"event_id": eventID,
				"sender":   sender,
				"type":     "m.room.message",
			},
			Sound: "default",
			Badge: 1,
		}

		notifications = append(notifications, notif)
	}

	return notifications
}

// Helper functions

func generateDeviceID() string {
	return "dev_" + securerandom.MustID(16)
}

func generateNotificationID() string {
	return "notif_" + securerandom.MustID(16)
}

func formatSenderName(sender string) string {
	// Extract display name from Matrix ID
	if len(sender) > 0 && sender[0] == '@' {
		parts := splitMatrixID(sender)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return sender
}

func splitMatrixID(mxid string) []string {
	// Remove @ prefix
	if len(mxid) > 0 && mxid[0] == '@' {
		mxid = mxid[1:]
	}
	// Split at :
	for i, c := range mxid {
		if c == ':' {
			return []string{mxid[:i], mxid[i+1:]}
		}
	}
	return []string{mxid}
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// MarshalJSON custom marshals notification for logging
func (n *Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"created_at"`
		ExpiresAt string `json:"expires_at,omitempty"`
	}{
		Alias:     (*Alias)(n),
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
		ExpiresAt: n.ExpiresAt.Format(time.RFC3339),
	})
}
