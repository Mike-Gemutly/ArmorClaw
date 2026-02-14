// Package sdtw provides Microsoft Teams adapter stub
package sdtw

import (
	"context"
	"fmt"
	"time"
)

// TeamsAdapter implements SDTWAdapter for Microsoft Teams
// This is a stub implementation for Phase 2
type TeamsAdapter struct {
	*BaseAdapter
	initialized bool
}

// TeamsMessage represents a Teams message payload
type TeamsMessage struct {
	Type    string                 `json:"type"`
	ID      string                 `json:"id"`
	From    TeamsUser              `json:"from"`
	To      []TeamsUser            `json:"to"`
	Text    string                 `json:"text"`
	Attachments []TeamsAttachment  `json:"attachments,omitempty"`
}

// TeamsUser represents a Teams user
type TeamsUser struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// TeamsAttachment represents an attachment in Teams
type TeamsAttachment struct {
	ContentType string                 `json:"contentType"`
	Content     map[string]interface{} `json:"content"`
}

// TeamsActivity represents a Teams activity event
type TeamsActivity struct {
	Type         string                 `json:"type"`
	ID           string                 `json:"id"`
	Timestamp    string                 `json:"timestamp"`
	From         TeamsUser              `json:"from"`
	Conversation TeamsConversation      `json:"conversation"`
	Text         string                 `json:"text"`
	Attachments  []TeamsAttachment      `json:"attachments,omitempty"`
}

// TeamsConversation represents a conversation in Teams
type TeamsConversation struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// NewTeamsAdapter creates a new Teams adapter stub
func NewTeamsAdapter() *TeamsAdapter {
	caps := CapabilitySet{
		Read:         false, // Not implemented yet
		Write:        false, // Not implemented yet
		Media:        false,
		Reactions:    false,
		Threads:      false,
		Edit:         false,
		Delete:       false,
		Typing:       false,
		ReadReceipts: false,
	}

	return &TeamsAdapter{
		BaseAdapter: NewBaseAdapter("teams", "0.1.0-stub", caps),
	}
}

// Initialize sets up the Teams adapter with configuration
func (t *TeamsAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	if !config.Enabled {
		return fmt.Errorf("teams adapter is not enabled (Phase 2)")
	}
	return fmt.Errorf("teams adapter not yet implemented - scheduled for Phase 2")
}

// Start begins processing Teams events
func (t *TeamsAdapter) Start(ctx context.Context) error {
	return fmt.Errorf("teams adapter not yet implemented - scheduled for Phase 2")
}

// Shutdown gracefully stops the adapter
func (t *TeamsAdapter) Shutdown(ctx context.Context) error {
	return nil // No-op for stub
}

// SendMessage sends a message to Teams
func (t *TeamsAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	return &SendResult{
		Delivered: false,
		Timestamp: time.Now(),
		Error: NewAdapterError(ErrPlatformError,
			"teams adapter not yet implemented - scheduled for Phase 2", false),
	}, nil
}

// ReceiveEvent handles an incoming Teams event
func (t *TeamsAdapter) ReceiveEvent(event ExternalEvent) error {
	return fmt.Errorf("teams adapter not yet implemented - scheduled for Phase 2")
}

// HealthCheck returns the current health status
func (t *TeamsAdapter) HealthCheck() (HealthStatus, error) {
	return HealthStatus{
		Connected: false,
		Error:     "teams adapter not yet implemented - scheduled for Phase 2",
	}, nil
}

// Metrics returns the current metrics
func (t *TeamsAdapter) Metrics() (AdapterMetrics, error) {
	return AdapterMetrics{}, nil
}

// GetDefaultCapabilities returns the default capabilities for Teams (when implemented)
func GetDefaultTeamsCapabilities() CapabilitySet {
	return CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    true,
		Threads:      true,
		Edit:         true,
		Delete:       true,
		Typing:       true,
		ReadReceipts: true,
	}
}
