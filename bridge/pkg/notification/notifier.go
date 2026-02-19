// Package notification provides notification delivery via Matrix
// This sends budget alerts, security events, and system notifications to Matrix rooms
package notification

import (
	"context"
	"fmt"
	"sync"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/logger"
	"log/slog"
)

// Notifier sends notifications to Matrix rooms
type Notifier struct {
	matrixAdapter *adapter.MatrixAdapter
	adminRoomID   string
	enabled       bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	securityLog   *logger.SecurityLogger
}

// Config holds notification configuration
type Config struct {
	AdminRoomID    string // Matrix room ID for admin notifications
	Enabled        bool   // Whether notifications are enabled
	AlertThreshold float64 // Percentage at which to send alerts (0.0-1.0)
}

// NewNotifier creates a new Matrix notification sender
func NewNotifier(matrixAdapter *adapter.MatrixAdapter, config Config) *Notifier {
	ctx, cancel := context.WithCancel(context.Background())

	return &Notifier{
		matrixAdapter: matrixAdapter,
		adminRoomID:   config.AdminRoomID,
		enabled:       config.Enabled && config.AdminRoomID != "",
		ctx:          ctx,
		cancel:       cancel,
		securityLog:  logger.NewSecurityLogger(logger.Global().WithComponent("notifier")),
	}
}

// Start initializes the notifier
func (n *Notifier) Start() error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.enabled {
		return nil
	}

	n.securityLog.LogSecurityEvent("notifier_started",
		slog.String("admin_room", n.adminRoomID))

	return nil
}

// Stop shuts down the notifier
func (n *Notifier) Stop() {
	n.cancel()
	n.securityLog.LogSecurityEvent("notifier_stopped")
}

// SendBudgetAlert sends a budget alert to Matrix
func (n *Notifier) SendBudgetAlert(alertType string, sessionID string, current, limit float64) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.enabled || n.matrixAdapter == nil {
		// Fallback to logging if Matrix not configured
		n.securityLog.LogSecurityEvent("budget_alert",
			slog.String("alert_type", alertType),
			slog.String("session_id", sessionID),
			slog.Float64("current", current),
			slog.Float64("limit", limit))
		return nil
	}

	percentage := (current / limit) * 100

	message := fmt.Sprintf("🔔 **Budget Alert**\n\n"+
		"**Type:** %s\n"+
		"**Session:** %s\n"+
		"**Current:** $%.2f\n"+
		"**Limit:** $%.2f\n"+
		"**Usage:** %.1f%%\n\n"+
		"Action may be required if this is not expected.",
		alertType, sessionID, current, limit, percentage)

	_, err := n.matrixAdapter.SendMessage(n.adminRoomID, message, "m.notice")
	if err != nil {
		n.securityLog.LogSecurityEvent("budget_alert_failed",
			slog.String("error", err.Error()),
			slog.String("alert_type", alertType),
			slog.String("session_id", sessionID))
		return fmt.Errorf("failed to send budget alert: %w", err)
	}

	n.securityLog.LogSecurityEvent("budget_alert_sent",
		slog.String("alert_type", alertType),
		slog.String("session_id", sessionID),
		slog.String("room_id", n.adminRoomID))

	return nil
}

// SendSecurityAlert sends a security alert to Matrix
func (n *Notifier) SendSecurityAlert(eventType, message string, metadata map[string]interface{}) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.enabled || n.matrixAdapter == nil {
		n.securityLog.LogSecurityEvent("security_alert",
			slog.String("event_type", eventType),
			slog.String("message", message))
		return nil
	}

	// Build message with metadata
	msgText := fmt.Sprintf("🚨 **Security Alert**\n\n**Type:** %s\n\n%s\n", eventType, message)

	// Add metadata if present
	if len(metadata) > 0 {
		msgText += "\n**Details:**\n"
		for key, value := range metadata {
			msgText += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	_, err := n.matrixAdapter.SendMessage(n.adminRoomID, msgText, "m.notice")
	if err != nil {
		n.securityLog.LogSecurityEvent("security_alert_failed",
			slog.String("error", err.Error()),
			slog.String("event_type", eventType))
		return fmt.Errorf("failed to send security alert: %w", err)
	}

	n.securityLog.LogSecurityEvent("security_alert_sent",
		slog.String("event_type", eventType),
		slog.String("room_id", n.adminRoomID))

	return nil
}

// SendContainerAlert sends a container-related alert to Matrix
func (n *Notifier) SendContainerAlert(eventType, containerID, containerName, reason string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.enabled || n.matrixAdapter == nil {
		n.securityLog.LogSecurityEvent("container_alert",
			slog.String("event_type", eventType),
			slog.String("container_id", containerID),
			slog.String("container_name", containerName),
			slog.String("reason", reason))
		return nil
	}

	var emoji string
	switch eventType {
	case "container_started":
		emoji = "✅"
	case "container_stopped":
		emoji = "⏹️"
	case "container_failed":
		emoji = "❌"
	case "container_restarted":
		emoji = "🔄"
	default:
		emoji = "ℹ️"
	}

	message := fmt.Sprintf("%s **Container Event**\n\n"+
		"**Type:** %s\n"+
		"**Container:** %s\n"+
		"**ID:** %s\n"+
		"**Reason:** %s",
		emoji, eventType, containerName, containerID, reason)

	_, err := n.matrixAdapter.SendMessage(n.adminRoomID, message, "m.notice")
	if err != nil {
		n.securityLog.LogSecurityEvent("container_alert_failed",
			slog.String("error", err.Error()),
			slog.String("event_type", eventType),
			slog.String("container_id", containerID))
		return fmt.Errorf("failed to send container alert: %w", err)
	}

	n.securityLog.LogSecurityEvent("container_alert_sent",
		slog.String("event_type", eventType),
		slog.String("container_id", containerID),
		slog.String("room_id", n.adminRoomID))

	return nil
}

// SendSystemAlert sends a system-level alert to Matrix
func (n *Notifier) SendSystemAlert(eventType, message string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.enabled || n.matrixAdapter == nil {
		n.securityLog.LogSecurityEvent("system_alert",
			slog.String("event_type", eventType),
			slog.String("message", message))
		return nil
	}

	msgText := fmt.Sprintf("⚠️ **System Alert**\n\n**Type:** %s\n\n%s", eventType, message)

	_, err := n.matrixAdapter.SendMessage(n.adminRoomID, msgText, "m.notice")
	if err != nil {
		n.securityLog.LogSecurityEvent("system_alert_failed",
			slog.String("error", err.Error()),
			slog.String("event_type", eventType))
		return fmt.Errorf("failed to send system alert: %w", err)
	}

	n.securityLog.LogSecurityEvent("system_alert_sent",
		slog.String("event_type", eventType),
		slog.String("room_id", n.adminRoomID))

	return nil
}

// IsEnabled returns whether notifications are enabled
func (n *Notifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}

// SetEnabled enables or disables notifications
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled && n.adminRoomID != ""
}

// SetAdminRoom sets the admin room for notifications
func (n *Notifier) SetAdminRoom(roomID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.adminRoomID = roomID
	n.enabled = roomID != ""
}

// SetMatrixAdapter sets or updates the Matrix adapter for sending notifications
// This allows the notifier to be created before the Matrix adapter is initialized
func (n *Notifier) SetMatrixAdapter(matrixAdapter *adapter.MatrixAdapter) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.matrixAdapter = matrixAdapter
}
