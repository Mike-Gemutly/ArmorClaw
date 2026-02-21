// Package notification provides system alert functionality for the Bridge.
//
// Resolves: Gap 4 (Notification Pipeline "Split-Brain")
//
// System alerts are sent as custom Matrix events with distinct UI treatment
// to ensure critical alerts are not lost in the chat stream.
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Event type constants for system alerts
const (
	// SystemAlertEventType is the Matrix event type for system alerts
	SystemAlertEventType = "app.armorclaw.alert"

	// Content fields
	FieldAlertType  = "alert_type"
	FieldSeverity   = "severity"
	FieldTitle      = "title"
	FieldMessage    = "message"
	FieldAction     = "action"
	FieldActionURL  = "action_url"
	FieldTimestamp  = "timestamp"
	FieldMetadata   = "metadata"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityError    AlertSeverity = "ERROR"
	SeverityCritical AlertSeverity = "CRITICAL"
)

// AlertType represents different types of system alerts
type AlertType string

const (
	// Budget alerts
	AlertBudgetWarning  AlertType = "BUDGET_WARNING"
	AlertBudgetExceeded AlertType = "BUDGET_EXCEEDED"

	// License alerts
	AlertLicenseExpiring AlertType = "LICENSE_EXPIRING"
	AlertLicenseExpired  AlertType = "LICENSE_EXPIRED"
	AlertLicenseInvalid  AlertType = "LICENSE_INVALID"

	// Security alerts
	AlertSecurityEvent       AlertType = "SECURITY_EVENT"
	AlertTrustDegraded       AlertType = "TRUST_DEGRADED"
	AlertVerificationRequired AlertType = "VERIFICATION_REQUIRED"
	AlertBridgeSecurityDowngrade AlertType = "BRIDGE_SECURITY_DOWNGRADE"

	// System alerts
	AlertBridgeError     AlertType = "BRIDGE_ERROR"
	AlertBridgeRestarting AlertType = "BRIDGE_RESTARTING"
	AlertMaintenance     AlertType = "MAINTENANCE"

	// Compliance alerts
	AlertComplianceViolation AlertType = "COMPLIANCE_VIOLATION"
	AlertAuditExport        AlertType = "AUDIT_EXPORT"
)

// SystemAlert represents a system alert to be sent to Matrix
type SystemAlert struct {
	Type       AlertType            `json:"alert_type"`
	Severity   AlertSeverity         `json:"severity"`
	Title      string                `json:"title"`
	Message    string                `json:"message"`
	Action     *string               `json:"action,omitempty"`
	ActionURL  *string               `json:"action_url,omitempty"`
	Timestamp  int64                 `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SystemAlertContent is the Matrix event content for a system alert
type SystemAlertContent struct {
	MsgType   string                `json:"msgtype"`
	AlertType AlertType             `json:"alert_type"`
	Severity  AlertSeverity         `json:"severity"`
	Title     string                `json:"title"`
	Message   string                `json:"message"`
	Action    *string               `json:"action,omitempty"`
	ActionURL *string               `json:"action_url,omitempty"`
	Timestamp int64                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertSender interface for sending alerts to Matrix
type AlertSender interface {
	SendAlert(ctx context.Context, roomID string, alert *SystemAlert) error
}

// AlertManager manages system alerts
type AlertManager struct {
	logger     *slog.Logger
	sender     AlertSender
	alertRoomID string // Room ID for system alerts
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *slog.Logger, sender AlertSender, alertRoomID string) *AlertManager {
	if logger == nil {
		logger = slog.Default().With("component", "alert_manager")
	}

	return &AlertManager{
		logger:      logger,
		sender:      sender,
		alertRoomID: alertRoomID,
	}
}

// SendAlert sends a system alert to the alert room
func (m *AlertManager) SendAlert(ctx context.Context, alert *SystemAlert) error {
	if m.sender == nil {
		return fmt.Errorf("alert sender not configured")
	}

	if alert.Timestamp == 0 {
		alert.Timestamp = time.Now().UnixMilli()
	}

	m.logger.Info("sending_system_alert",
		"alert_type", alert.Type,
		"severity", alert.Severity,
		"title", alert.Title,
	)

	return m.sender.SendAlert(ctx, m.alertRoomID, alert)
}

// SendBudgetWarning sends a budget warning alert
func (m *AlertManager) SendBudgetWarning(ctx context.Context, currentSpend, limit float64, percentage int) error {
	alert := &SystemAlert{
		Type:     AlertBudgetWarning,
		Severity: SeverityWarning,
		Title:    "Budget Warning",
		Message:  fmt.Sprintf("Token usage is at %d%% ($%.2f of $%.2f limit)", percentage, currentSpend, limit),
		Metadata: map[string]interface{}{
			"current_spend": currentSpend,
			"limit":        limit,
			"percentage":   percentage,
		},
	}

	action := "View Usage"
	alert.Action = &action
	actionURL := "armorclaw://dashboard/budget"
	alert.ActionURL = &actionURL

	return m.SendAlert(ctx, alert)
}

// SendBudgetExceeded sends a budget exceeded alert
func (m *AlertManager) SendBudgetExceeded(ctx context.Context, currentSpend, limit float64) error {
	alert := &SystemAlert{
		Type:     AlertBudgetExceeded,
		Severity: SeverityError,
		Title:    "Budget Exceeded",
		Message:  "Token budget has been exceeded. API calls are suspended until the budget resets.",
		Metadata: map[string]interface{}{
			"current_spend": currentSpend,
			"limit":        limit,
			"overage":      currentSpend - limit,
		},
	}

	action := "Upgrade Plan"
	alert.Action = &action
	actionURL := "armorclaw://dashboard/billing"
	alert.ActionURL = &actionURL

	return m.SendAlert(ctx, alert)
}

// SendLicenseExpiring sends a license expiring alert
func (m *AlertManager) SendLicenseExpiring(ctx context.Context, daysRemaining int, expiresAt string) error {
	severity := SeverityWarning
	if daysRemaining <= 7 {
		severity = SeverityError
	}

	alert := &SystemAlert{
		Type:     AlertLicenseExpiring,
		Severity: severity,
		Title:    "License Expiring",
		Message:  fmt.Sprintf("Your license expires in %d days (%s). Renew to avoid service interruption.", daysRemaining, expiresAt),
		Metadata: map[string]interface{}{
			"days_remaining": daysRemaining,
			"expires_at":    expiresAt,
		},
	}

	action := "Renew License"
	alert.Action = &action
	actionURL := "armorclaw://dashboard/license"
	alert.ActionURL = &actionURL

	return m.SendAlert(ctx, alert)
}

// SendLicenseExpired sends a license expired alert
func (m *AlertManager) SendLicenseExpired(ctx context.Context) error {
	alert := &SystemAlert{
		Type:     AlertLicenseExpired,
		Severity: SeverityCritical,
		Title:    "License Expired",
		Message:  "Your license has expired. Bridge functionality is limited. Please renew to restore full access.",
	}

	action := "Renew Now"
	alert.Action = &action
	actionURL := "armorclaw://dashboard/license"
	alert.ActionURL = &actionURL

	return m.SendAlert(ctx, alert)
}

// SendBridgeError sends a bridge error alert
func (m *AlertManager) SendBridgeError(ctx context.Context, component, errorMessage string) error {
	alert := &SystemAlert{
		Type:     AlertBridgeError,
		Severity: SeverityError,
		Title:    "Bridge Error",
		Message:  fmt.Sprintf("An error occurred in %s: %s", component, errorMessage),
		Metadata: map[string]interface{}{
			"component": component,
			"error":    errorMessage,
		},
	}

	action := "View Logs"
	alert.Action = &action
	actionURL := "armorclaw://dashboard/logs"
	alert.ActionURL = &actionURL

	return m.SendAlert(ctx, alert)
}

// ToMatrixContent converts the alert to Matrix event content
func (a *SystemAlert) ToMatrixContent() map[string]interface{} {
	content := map[string]interface{}{
		"msgtype":     "m.notice",
		FieldAlertType: a.Type,
		FieldSeverity:  a.Severity,
		FieldTitle:     a.Title,
		FieldMessage:   a.Message,
		FieldTimestamp: a.Timestamp,
	}

	if a.Action != nil {
		content[FieldAction] = *a.Action
	}
	if a.ActionURL != nil {
		content[FieldActionURL] = *a.ActionURL
	}
	if a.Metadata != nil {
		content[FieldMetadata] = a.Metadata
	}

	return content
}

// ToJSON converts the alert to JSON
func (a *SystemAlert) ToJSON() ([]byte, error) {
	return json.Marshal(a.ToMatrixContent())
}
