package errors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ErrorNotifier sends error notifications to admins
type ErrorNotifier struct {
	mu sync.RWMutex

	// Components
	registry *SamplingRegistry
	resolver *AdminResolver
	store    *ErrorStore

	// Matrix sender
	matrixSender MatrixMessageSender

	// Configuration
	enabled bool
}

// MatrixMessageSender is the interface for sending Matrix messages
type MatrixMessageSender interface {
	SendMessage(ctx context.Context, roomID, message, msgType string) (string, error)
}

// NotifierConfig configures the error notifier
type NotifierConfig struct {
	Registry     *SamplingRegistry
	Resolver     *AdminResolver
	Store        *ErrorStore
	MatrixSender MatrixMessageSender
	Enabled      bool
}

// NewErrorNotifier creates a new error notifier
func NewErrorNotifier(cfg NotifierConfig) *ErrorNotifier {
	return &ErrorNotifier{
		registry:     cfg.Registry,
		resolver:     cfg.Resolver,
		store:        cfg.Store,
		matrixSender: cfg.MatrixSender,
		enabled:      cfg.Enabled,
	}
}

// Notify processes an error and sends notification if appropriate
func (n *ErrorNotifier) Notify(ctx context.Context, err *TracedError) error {
	if !n.enabled {
		return nil
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	// Check if we should notify (sampling)
	if n.registry != nil && !n.registry.ShouldNotify(err) {
		// Still store the error even if not notifying
		if n.store != nil {
			_ = n.store.Store(ctx, err)
		}
		return nil
	}

	// Get recent logs from components
	err.RecentLogs = n.getRecentLogs(err.Category)

	// Store the error
	if n.store != nil {
		if storeErr := n.store.Store(ctx, err); storeErr != nil {
			// Log but continue with notification
			_ = storeErr
		}
	}

	// Resolve admin
	if n.resolver == nil {
		return fmt.Errorf("no admin resolver configured")
	}

	admin, err2 := n.resolver.Resolve(ctx)
	if err2 != nil {
		return fmt.Errorf("failed to resolve admin: %w", err2)
	}

	// Format message
	message := n.formatMessage(err, admin)

	// Send notification
	if n.matrixSender != nil {
		// Send as direct message (m.notice for less intrusive)
		_, err2 = n.matrixSender.SendMessage(ctx, admin.MXID, message, "m.notice")
		if err2 != nil {
			return fmt.Errorf("failed to send notification: %w", err2)
		}
	}

	return nil
}

// getRecentLogs retrieves recent logs from relevant components
func (n *ErrorNotifier) getRecentLogs(category string) []ComponentLogEntry {
	// Get logs from the failing component plus related components
	var components []string

	switch category {
	case "container":
		components = []string{"docker", "secrets"}
	case "matrix":
		components = []string{"matrix", "turn"}
	case "rpc":
		components = []string{"rpc", "audit"}
	case "voice":
		components = []string{"voice", "webrtc", "turn"}
	case "budget":
		components = []string{"budget", "audit"}
	default:
		components = []string{category}
	}

	return GetMultiComponentEvents(components, 5)
}

// formatMessage creates the LLM-friendly notification message
func (n *ErrorNotifier) formatMessage(err *TracedError, admin *AdminTarget) string {
	var sb strings.Builder

	// Header with severity indicator
	sb.WriteString(n.formatHeader(err))
	sb.WriteString("\n")

	// Error summary
	sb.WriteString(n.formatSummary(err))
	sb.WriteString("\n\n")

	// Metadata
	sb.WriteString(n.formatMetadata(err, admin))
	sb.WriteString("\n\n")

	// JSON block
	sb.WriteString("```json\n")
	jsonStr, jsonErr := err.FormatJSON()
	if jsonErr != nil {
		sb.WriteString(fmt.Sprintf(`{"error": "failed to serialize: %s"}`, jsonErr))
	} else {
		sb.WriteString(jsonStr)
	}
	sb.WriteString("\n```\n\n")

	// Footer
	sb.WriteString("📋 Copy the JSON block above to analyze with an LLM.")

	return sb.String()
}

// formatHeader creates the notification header
func (n *ErrorNotifier) formatHeader(err *TracedError) string {
	var emoji string
	switch err.Severity {
	case SeverityCritical:
		emoji = "🔴"
	case SeverityError:
		emoji = "❌"
	case SeverityWarning:
		emoji = "⚠️"
	default:
		emoji = "ℹ️"
	}

	severity := strings.ToUpper(string(err.Severity))
	return fmt.Sprintf("%s %s: %s", emoji, severity, err.Code)
}

// formatSummary creates a brief error summary
func (n *ErrorNotifier) formatSummary(err *TracedError) string {
	var sb strings.Builder

	sb.WriteString(err.Message)

	if err.cause != nil {
		sb.WriteString(": ")
		sb.WriteString(err.cause.Error())
	}

	return sb.String()
}

// formatMetadata creates the metadata section
func (n *ErrorNotifier) formatMetadata(err *TracedError, admin *AdminTarget) string {
	var lines []string

	if err.Function != "" {
		lines = append(lines, fmt.Sprintf("📍 Location: %s @ %s:%d", err.Function, err.File, err.Line))
	}

	lines = append(lines, fmt.Sprintf("🏷️ Trace ID: %s", err.TraceID))
	lines = append(lines, fmt.Sprintf("⏰ %s", err.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC")))

	if err.RepeatCount > 0 {
		lines = append(lines, fmt.Sprintf("🔁 Repeated %d times since last notification", err.RepeatCount))
	}

	if admin != nil {
		lines = append(lines, fmt.Sprintf("👤 Admin: %s (via %s)", admin.MXID, admin.Source))
	}

	return strings.Join(lines, "\n")
}

// NotifyQuick sends a quick notification without full trace
func (n *ErrorNotifier) NotifyQuick(ctx context.Context, code, message string, severity Severity) error {
	err := &TracedError{
		Code:      code,
		Category:  Lookup(code).Category,
		Severity:  severity,
		Message:   message,
		TraceID:   generateTraceID(),
		Timestamp: time.Now(),
	}
	return n.Notify(ctx, err)
}

// SetEnabled enables or disables notifications
func (n *ErrorNotifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// IsEnabled returns whether notifications are enabled
func (n *ErrorNotifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}

// SetMatrixSender sets or updates the Matrix sender
func (n *ErrorNotifier) SetMatrixSender(sender MatrixMessageSender) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.matrixSender = sender
}

// SetResolver sets or updates the admin resolver
func (n *ErrorNotifier) SetResolver(resolver *AdminResolver) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.resolver = resolver
}

// SetStore sets or updates the error store
func (n *ErrorNotifier) SetStore(store *ErrorStore) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.store = store
}

// SetRegistry sets or updates the sampling registry
func (n *ErrorNotifier) SetRegistry(registry *SamplingRegistry) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.registry = registry
}

// Global notifier
var globalNotifier *ErrorNotifier
var globalNotifierMu sync.RWMutex

// SetGlobalNotifier sets the global error notifier
func SetGlobalNotifier(notifier *ErrorNotifier) {
	globalNotifierMu.Lock()
	defer globalNotifierMu.Unlock()
	globalNotifier = notifier
}

// GetGlobalNotifier returns the global error notifier
func GetGlobalNotifier() *ErrorNotifier {
	globalNotifierMu.RLock()
	defer globalNotifierMu.RUnlock()
	return globalNotifier
}

// GlobalNotify sends a notification using the global notifier
func GlobalNotify(ctx context.Context, err *TracedError) error {
	notifier := GetGlobalNotifier()
	if notifier == nil {
		return fmt.Errorf("global error notifier not initialized")
	}
	return notifier.Notify(ctx, err)
}

// GlobalNotifyQuick sends a quick notification using the global notifier
func GlobalNotifyQuick(ctx context.Context, code, message string, severity Severity) error {
	notifier := GetGlobalNotifier()
	if notifier == nil {
		return fmt.Errorf("global error notifier not initialized")
	}
	return notifier.NotifyQuick(ctx, code, message, severity)
}

// NotifyAndPanic sends a critical notification then panics
func (n *ErrorNotifier) NotifyAndPanic(ctx context.Context, err *TracedError) {
	// Ensure notification is sent even if we panic
	_ = n.Notify(ctx, err)
	panic(err)
}

// NotifyAndLog sends a notification and returns the error for logging
func (n *ErrorNotifier) NotifyAndLog(ctx context.Context, err *TracedError) error {
	notifyErr := n.Notify(ctx, err)
	if notifyErr != nil {
		return fmt.Errorf("notification failed: %w (original error: %s)", notifyErr, err.Code)
	}
	return err
}
