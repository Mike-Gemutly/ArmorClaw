// Package sdtw provides WhatsApp adapter implementation
package sdtw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// WhatsAppAdapter implements SDTWAdapter for WhatsApp Business Cloud API
type WhatsAppAdapter struct {
	*BaseAdapter
	client            *http.Client
	accessToken       string
	phoneNumberID     string
	businessAccountID string
	webhookSecret     string
	logger            *slog.Logger
	mu                sync.RWMutex
	running           bool
	ctx               context.Context
	cancel            context.CancelFunc

	// Rate limiting
	rateLimiter *rateLimiter
}

// rateLimiter implements simple token bucket rate limiting
type rateLimiter struct {
	tokens     int
	capacity   int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// WhatsAppMessage represents a WhatsApp message payload
type WhatsAppMessage struct {
	MessagingProduct string            `json:"messaging_product"`
	RecipientType    string            `json:"recipient_type,omitempty"`
	To               string            `json:"to"`
	Type             string            `json:"type"`
	Text             *WhatsAppText     `json:"text,omitempty"`
	Template         *WhatsAppTemplate `json:"template,omitempty"`
	Image            *WhatsAppMedia    `json:"image,omitempty"`
	Document         *WhatsAppMedia    `json:"document,omitempty"`
	Audio            *WhatsAppMedia    `json:"audio,omitempty"`
	Video            *WhatsAppMedia    `json:"video,omitempty"`
}

// WhatsAppText represents text content in WhatsApp
type WhatsAppText struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

// WhatsAppMedia represents media content in WhatsApp
type WhatsAppMedia struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// WhatsAppTemplate represents a template message in WhatsApp
type WhatsAppTemplate struct {
	Name       string                      `json:"name"`
	Language   WhatsAppLanguage            `json:"language"`
	Components []WhatsAppTemplateComponent `json:"components,omitempty"`
}

// WhatsAppLanguage represents language code and policy
type WhatsAppLanguage struct {
	Code   string `json:"code"`
	Policy string `json:"policy,omitempty"` // "deterministic" or "fallback"
}

// WhatsAppTemplateComponent represents a template component
type WhatsAppTemplateComponent struct {
	Type       string              `json:"type"`
	Parameters []WhatsAppParameter `json:"parameters,omitempty"`
}

// WhatsAppParameter represents a template parameter
type WhatsAppParameter struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	Currency *WhatsAppCurrency `json:"currency,omitempty"`
	DateTime *WhatsAppDateTime `json:"date_time,omitempty"`
	Image    *WhatsAppMedia    `json:"image,omitempty"`
	Document *WhatsAppMedia    `json:"document,omitempty"`
	Video    *WhatsAppMedia    `json:"video,omitempty"`
}

// WhatsAppCurrency represents a currency parameter
type WhatsAppCurrency struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int    `json:"amount_1000"`
}

// WhatsAppDateTime represents a datetime parameter
type WhatsAppDateTime struct {
	FallbackValue string `json:"fallback_value"`
}

// WhatsAppEvent represents a WhatsApp webhook event
type WhatsAppEvent struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From      string              `json:"from"`
					ID        string              `json:"id"`
					Timestamp string              `json:"timestamp"`
					Type      string              `json:"type"`
					Text      *WhatsAppEventText  `json:"text,omitempty"`
					Image     *WhatsAppEventMedia `json:"image,omitempty"`
					Document  *WhatsAppEventMedia `json:"document,omitempty"`
					Audio     *WhatsAppEventMedia `json:"audio,omitempty"`
					Video     *WhatsAppEventMedia `json:"video,omitempty"`
				} `json:"messages,omitempty"`
				Statuses []struct {
					ID        string `json:"id"`
					Recipient string `json:"recipient"`
					Timestamp string `json:"timestamp"`
					Status    string `json:"status"` // sent, delivered, read, failed
					Errors    []struct {
						Code    int    `json:"code"`
						Title   string `json:"title"`
						Message string `json:"message"`
					} `json:"errors,omitempty"`
				} `json:"statuses,omitempty"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// WhatsAppEventText represents text in an incoming event
type WhatsAppEventText struct {
	Body string `json:"body"`
}

// WhatsAppEventMedia represents media in an incoming event
type WhatsAppEventMedia struct {
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	ID       string `json:"id"`
}

// WhatsAppAPIResponse represents a response from WhatsApp API
type WhatsAppAPIResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts,omitempty"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
		TraceID string `json:"error_data_trace_id"`
	} `json:"error,omitempty"`
}

// NewWhatsAppAdapter creates a new WhatsApp adapter
func NewWhatsAppAdapter() *WhatsAppAdapter {
	caps := GetDefaultWhatsAppCapabilities()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &WhatsAppAdapter{
		BaseAdapter: NewBaseAdapter("whatsapp", "1.0.0", caps),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger,
		rateLimiter: newRateLimiter(100, time.Second), // 100 requests per second default
	}
}

// Initialize sets up the WhatsApp adapter with configuration
func (w *WhatsAppAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.BaseAdapter.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract credentials (injected from keystore)
	w.accessToken = config.Credentials["access_token"]
	if w.accessToken == "" {
		return NewAdapterError(ErrAuthFailed, "access_token is required", false)
	}

	w.phoneNumberID = config.Settings["phone_number_id"]
	if w.phoneNumberID == "" {
		return NewAdapterError(ErrValidation, "phone_number_id is required", false)
	}

	w.businessAccountID = config.Settings["business_account_id"]
	w.webhookSecret = config.Credentials["webhook_secret"]

	// Configure rate limiting from config
	if config.RateLimits.RequestsPerSecond > 0 {
		w.rateLimiter = newRateLimiter(config.RateLimits.RequestsPerSecond, time.Second)
	}

	w.ctx, w.cancel = context.WithCancel(context.Background())

	return nil
}

// Start begins processing WhatsApp events
func (w *WhatsAppAdapter) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = true

	// Verify connection
	return w.verifyConnection(ctx)
}

// Shutdown gracefully stops the adapter
func (w *WhatsAppAdapter) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = false

	if w.cancel != nil {
		w.cancel()
	}

	return nil
}

// SendMessage sends a message to WhatsApp
func (w *WhatsAppAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	if err := ValidateMessage(msg); err != nil {
		return nil, err
	}

	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(ErrPlatformError, "adapter not running", false),
		}, nil
	}
	w.mu.RUnlock()

	// Wait for rate limit
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return nil, NewAdapterError(ErrRateLimited, "rate limit exceeded", true)
	}

	// Determine phone number to send to
	toPhone := target.Channel
	if toPhone == "" {
		toPhone = target.UserID
	}

	if toPhone == "" {
		return nil, NewAdapterError(ErrInvalidTarget, "channel or user ID (phone number) is required", false)
	}

	// Build WhatsApp message payload
	whatsappMsg := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               toPhone,
	}

	// Handle template messages
	if templateName, ok := msg.Metadata["template_name"]; ok {
		languageCode := "en_US"
		if lang, ok := msg.Metadata["template_language"]; ok {
			languageCode = lang
		}

		whatsappMsg.Type = "template"
		whatsappMsg.Template = &WhatsAppTemplate{
			Name:     templateName,
			Language: WhatsAppLanguage{Code: languageCode},
		}

		// Add template components if present
		if componentsData, ok := msg.Metadata["template_components"]; ok {
			var components []WhatsAppTemplateComponent
			if err := json.Unmarshal([]byte(componentsData), &components); err == nil {
				whatsappMsg.Template.Components = components
			}
		}
	} else {
		// Handle text or media messages
		switch msg.Type {
		case MessageTypeImage:
			if len(msg.Attachments) > 0 {
				att := msg.Attachments[0]
				whatsappMsg.Type = "image"
				whatsappMsg.Image = &WhatsAppMedia{
					ID:      att.ID,
					Link:    att.URL,
					Caption: msg.Content,
				}
			} else {
				// Fallback to text if no attachment
				whatsappMsg.Type = "text"
				whatsappMsg.Text = &WhatsAppText{Body: msg.Content}
			}
		case MessageTypeFile:
			if len(msg.Attachments) > 0 {
				att := msg.Attachments[0]
				whatsappMsg.Type = "document"
				whatsappMsg.Document = &WhatsAppMedia{
					ID:       att.ID,
					Link:     att.URL,
					Filename: att.Filename,
					Caption:  msg.Content,
				}
			} else {
				whatsappMsg.Type = "text"
				whatsappMsg.Text = &WhatsAppText{Body: msg.Content}
			}
		default:
			// Default to text
			whatsappMsg.Type = "text"
			whatsappMsg.Text = &WhatsAppText{
				Body:       msg.Content,
				PreviewURL: true,
			}
		}
	}

	// Marshal payload
	payload, err := json.Marshal(whatsappMsg)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to marshal message", false)
	}

	// Create request
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", w.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		w.RecordError(err)
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to read response", true)
	}

	// Parse response
	var apiResp WhatsAppAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to parse response", true)
	}

	// Check for API errors
	if apiResp.Error != nil {
		w.RecordError(fmt.Errorf("WhatsApp API error: %s", apiResp.Error.Message))
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(mapWhatsAppError(apiResp.Error.Code), apiResp.Error.Message, isRetryableWhatsAppError(apiResp.Error.Code)),
		}, nil
	}

	// Extract message ID from response
	var messageID string
	if len(apiResp.Messages) > 0 {
		messageID = apiResp.Messages[0].ID
	}

	w.RecordSent()

	return &SendResult{
		MessageID: messageID,
		Delivered: true,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"phone_number_id": w.phoneNumberID,
			"to":              toPhone,
		},
	}, nil
}

// ReceiveEvent handles an incoming WhatsApp event
func (w *WhatsAppAdapter) ReceiveEvent(event ExternalEvent) error {
	if event.Platform != w.Platform() {
		return NewAdapterError(ErrValidation, "platform mismatch", false)
	}

	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return NewAdapterError(ErrPlatformError, "adapter not running", false)
	}
	w.mu.RUnlock()

	// Verify webhook signature if present
	if event.Signature != "" && w.webhookSecret != "" {
		if !VerifySignature(event.Content, event.Signature, w.webhookSecret) {
			return NewAdapterError(ErrAuthFailed, "invalid webhook signature", false)
		}
	}

	// Parse WhatsApp event
	var whatsappEvent WhatsAppEvent
	if err := json.Unmarshal([]byte(event.Content), &whatsappEvent); err != nil {
		return NewAdapterError(ErrValidation, "failed to parse event", false)
	}

	// Process messages
	for _, entry := range whatsappEvent.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			for _, msg := range change.Value.Messages {
				// Extract message content
				content := ""
				msgType := msg.Type

				if msg.Text != nil {
					content = msg.Text.Body
				} else if msg.Image != nil {
					content = msg.Image.Caption
					msgType = "image"
				} else if msg.Document != nil {
					content = msg.Document.Caption
					msgType = "document"
				} else if msg.Audio != nil {
					msgType = "audio"
				} else if msg.Video != nil {
					msgType = "video"
				}

				// Create attachments for media messages
				var attachments []Attachment
				if msg.Image != nil {
					attachments = append(attachments, Attachment{
						ID:       msg.Image.ID,
						MimeType: msg.Image.MimeType,
					})
				} else if msg.Document != nil {
					attachments = append(attachments, Attachment{
						ID:       msg.Document.ID,
						MimeType: msg.Document.MimeType,
					})
				} else if msg.Audio != nil {
					attachments = append(attachments, Attachment{
						ID:       msg.Audio.ID,
						MimeType: msg.Audio.MimeType,
					})
				} else if msg.Video != nil {
					attachments = append(attachments, Attachment{
						ID:       msg.Video.ID,
						MimeType: msg.Video.MimeType,
					})
				}

				// Update external event with parsed content
				event.Content = content
				event.EventType = msgType
				event.Source = msg.From
				event.Attachments = attachments
				event.Metadata = map[string]string{
					"message_id":      msg.ID,
					"timestamp":       msg.Timestamp,
					"phone_number_id": change.Value.Metadata.PhoneNumberID,
				}

				w.RecordReceived()
			}

			// Process status updates (delivery receipts)
			for _, status := range change.Value.Statuses {
				if status.Status == "read" {
					// Mark message as read in the system
					w.RecordReceived()
				}
			}
		}
	}

	return nil
}

// EditMessage marks a message as read (WhatsApp doesn't support editing content)
func (w *WhatsAppAdapter) EditMessage(ctx context.Context, target Target, messageID string, newContent string) error {
	// WhatsApp doesn't support editing message content
	// Use this method to mark messages as read instead
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return NewAdapterError(ErrPlatformError, "adapter not running", false)
	}
	w.mu.RUnlock()

	// Wait for rate limit
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return NewAdapterError(ErrRateLimited, "rate limit exceeded", true)
	}

	// Build mark as read payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return NewAdapterError(ErrPlatformError, "failed to marshal payload", false)
	}

	// Create request
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", w.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		w.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return NewAdapterError(ErrPlatformError, fmt.Sprintf("mark as read failed: %s - %s", resp.Status, string(respBody)), true)
	}

	return nil
}

// DeleteMessage deletes a message from WhatsApp (within time window)
func (w *WhatsAppAdapter) DeleteMessage(ctx context.Context, target Target, messageID string) error {
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return NewAdapterError(ErrPlatformError, "adapter not running", false)
	}
	w.mu.RUnlock()

	// Wait for rate limit
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return NewAdapterError(ErrRateLimited, "rate limit exceeded", true)
	}

	// Create request
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s", messageID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		w.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		w.RecordError(fmt.Errorf("delete failed: %s - %s", resp.Status, string(respBody)))

		// Handle specific errors
		if resp.StatusCode == http.StatusNotFound {
			return NewAdapterError(ErrInvalidTarget, "message not found or expired", false)
		}
		if resp.StatusCode == http.StatusForbidden {
			return NewAdapterError(ErrAuthFailed, "cannot delete message (time window expired or insufficient permissions)", false)
		}

		return NewAdapterError(ErrPlatformError, fmt.Sprintf("delete failed: %s", resp.Status), true)
	}

	w.RecordSent()
	return nil
}

// HealthCheck returns the current health status
func (w *WhatsAppAdapter) HealthCheck() (HealthStatus, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.running {
		return HealthStatus{
			Connected: false,
			Error:     "adapter not running",
		}, nil
	}

	return HealthStatus{
		Connected:   true,
		LastPing:    time.Now(),
		LastMessage: w.metrics.LastErrorTime,
		ErrorRate:   0.0,
		QueueDepth:  0,
	}, nil
}

// Metrics returns the current metrics
func (w *WhatsAppAdapter) Metrics() (AdapterMetrics, error) {
	return w.BaseAdapter.Metrics()
}

// SendTemplateMessage sends a template message to WhatsApp
func (w *WhatsAppAdapter) SendTemplateMessage(ctx context.Context, to, templateName, language string, components []WhatsAppTemplateComponent) (*SendResult, error) {
	msg := Message{
		ID:        generateMessageID(),
		Content:   "",
		Type:      MessageTypeText,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"template_name":     templateName,
			"template_language": language,
		},
	}

	if components != nil {
		componentsData, err := json.Marshal(components)
		if err == nil {
			msg.Metadata["template_components"] = string(componentsData)
		}
	}

	target := Target{
		Platform: "whatsapp",
		Channel:  to,
	}

	return w.SendMessage(ctx, target, msg)
}

// VerifyWebhook verifies a WhatsApp webhook signature
func (w *WhatsAppAdapter) VerifyWebhook(payload []byte, signature string, secret string) bool {
	return VerifySignature(string(payload), signature, secret)
}

// verifyConnection verifies the WhatsApp API connection
func (w *WhatsAppAdapter) verifyConnection(ctx context.Context) error {
	// Test connection by getting phone number info
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s", w.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed: %s", resp.Status)
	}

	return nil
}

// mapWhatsAppError maps WhatsApp API error codes to AdapterError codes
func mapWhatsAppError(errorCode int) ErrorCode {
	switch errorCode {
	case 131021, 131026:
		return ErrAuthFailed
	case 131047:
		return ErrInvalidTarget
	case 131052:
		return ErrRateLimited
	case 131014, 131015:
		return ErrInvalidTarget
	case 131001:
		return ErrNetworkError
	default:
		return ErrPlatformError
	}
}

// isRetryableWhatsAppError determines if a WhatsApp error is retryable
func isRetryableWhatsAppError(errorCode int) bool {
	retryableErrors := map[int]bool{
		131052: true, // Rate limit
		131001: true, // Network error
	}
	return retryableErrors[errorCode]
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(tokens int, refillRate time.Duration) *rateLimiter {
	return &rateLimiter{
		tokens:     tokens,
		capacity:   tokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (rl *rateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		now := time.Now()

		// Refill tokens
		elapsed := now.Sub(rl.lastRefill)
		if elapsed >= rl.refillRate {
			// Refill to capacity
			rl.tokens = rl.capacity
			rl.lastRefill = now
		}

		// Check if token available
		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time
		waitTime := rl.refillRate - elapsed
		rl.mu.Unlock()

		// Wait or timeout
		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetDefaultCapabilities returns the default capabilities for WhatsApp
func GetDefaultWhatsAppCapabilities() CapabilitySet {
	return CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    false, // WhatsApp doesn't have reactions
		Threads:      false, // No threading support
		Edit:         false, // Messages can't be edited (only marked as read)
		Delete:       true,  // Can delete within time window
		Typing:       false,
		ReadReceipts: true,
	}
}

// Helper function to generate message IDs
func generateMessageID() string {
	return fmt.Sprintf("wa_msg_%d", time.Now().UnixNano())
}
