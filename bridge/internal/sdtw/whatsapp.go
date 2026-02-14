// Package sdtw provides WhatsApp adapter stub
package sdtw

import (
	"context"
	"fmt"
	"time"
)

// WhatsAppAdapter implements SDTWAdapter for WhatsApp
// This is a stub implementation for Phase 2
type WhatsAppAdapter struct {
	*BaseAdapter
	phoneNumberID string
	businessAccountID string
	initialized bool
}

// WhatsAppMessage represents a WhatsApp message payload
type WhatsAppMessage struct {
	MessagingProduct string               `json:"messaging_product"`
	To               string               `json:"to"`
	Type             string               `json:"type"`
	Text             *WhatsAppText        `json:"text,omitempty"`
	Template         *WhatsAppTemplate    `json:"template,omitempty"`
}

// WhatsAppText represents text content in WhatsApp
type WhatsAppText struct {
	Body string `json:"body"`
	PreviewURL bool `json:"preview_url,omitempty"`
}

// WhatsAppTemplate represents a template message in WhatsApp
type WhatsAppTemplate struct {
	Name     string                 `json:"name"`
	Language map[string]string      `json:"language"`
	Components []WhatsAppTemplateComponent `json:"components,omitempty"`
}

// WhatsAppTemplateComponent represents a template component
type WhatsAppTemplateComponent struct {
	Type       string                 `json:"type"`
	Parameters []map[string]interface{} `json:"parameters,omitempty"`
}

// WhatsAppEvent represents a WhatsApp webhook event
type WhatsAppEvent struct {
	Object string `json:"object"`
	Entry []struct {
		ID   string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID     string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
				} `json:"messages,omitempty"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// NewWhatsAppAdapter creates a new WhatsApp adapter stub
func NewWhatsAppAdapter() *WhatsAppAdapter {
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

	return &WhatsAppAdapter{
		BaseAdapter: NewBaseAdapter("whatsapp", "0.1.0-stub", caps),
	}
}

// Initialize sets up the WhatsApp adapter with configuration
func (w *WhatsAppAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	if !config.Enabled {
		return fmt.Errorf("whatsapp adapter is not enabled (Phase 2)")
	}

	w.phoneNumberID = config.Settings["phone_number_id"]
	w.businessAccountID = config.Settings["business_account_id"]

	return fmt.Errorf("whatsapp adapter not yet implemented - scheduled for Phase 2")
}

// Start begins processing WhatsApp events
func (w *WhatsAppAdapter) Start(ctx context.Context) error {
	return fmt.Errorf("whatsapp adapter not yet implemented - scheduled for Phase 2")
}

// Shutdown gracefully stops the adapter
func (w *WhatsAppAdapter) Shutdown(ctx context.Context) error {
	return nil // No-op for stub
}

// SendMessage sends a message to WhatsApp
func (w *WhatsAppAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	return &SendResult{
		Delivered: false,
		Timestamp: time.Now(),
		Error: NewAdapterError(ErrPlatformError,
			"whatsapp adapter not yet implemented - scheduled for Phase 2", false),
	}, nil
}

// ReceiveEvent handles an incoming WhatsApp event
func (w *WhatsAppAdapter) ReceiveEvent(event ExternalEvent) error {
	return fmt.Errorf("whatsapp adapter not yet implemented - scheduled for Phase 2")
}

// HealthCheck returns the current health status
func (w *WhatsAppAdapter) HealthCheck() (HealthStatus, error) {
	return HealthStatus{
		Connected: false,
		Error:     "whatsapp adapter not yet implemented - scheduled for Phase 2",
	}, nil
}

// Metrics returns the current metrics
func (w *WhatsAppAdapter) Metrics() (AdapterMetrics, error) {
	return AdapterMetrics{}, nil
}

// SendTemplateMessage sends a template message to WhatsApp (when implemented)
func (w *WhatsAppAdapter) SendTemplateMessage(ctx context.Context, to, templateName string, language string, components []WhatsAppTemplateComponent) (*SendResult, error) {
	return &SendResult{
		Delivered: false,
		Timestamp: time.Now(),
		Error: NewAdapterError(ErrPlatformError,
			"whatsapp adapter not yet implemented - scheduled for Phase 2", false),
	}, nil
}

// VerifyWebhook verifies a WhatsApp webhook signature (when implemented)
func (w *WhatsAppAdapter) VerifyWebhook(payload []byte, signature string, secret string) bool {
	return VerifySignature(string(payload), signature, secret)
}

// GetDefaultCapabilities returns the default capabilities for WhatsApp (when implemented)
func GetDefaultWhatsAppCapabilities() CapabilitySet {
	return CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    false, // WhatsApp doesn't have reactions
		Threads:      false,
		Edit:         false, // Messages can't be edited
		Delete:       true,
		Typing:       false,
		ReadReceipts: true,
	}
}
