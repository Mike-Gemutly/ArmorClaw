// Package sdtw provides Slack adapter implementation
package sdtw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SlackAdapter implements SDTWAdapter for Slack
type SlackAdapter struct {
	*BaseAdapter
	client      *http.Client
	botToken    string
	webhookURL  string
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text"`
	Username    string            `json:"username"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Title     string `json:"title,omitempty"`
	Text      string `json:"text,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`
	Fallback  string `json:"fallback,omitempty"`
}

// SlackEvent represents a Slack event payload
type SlackEvent struct {
	Token     string          `json:"token"`
	Challenge string          `json:"challenge,omitempty"`
	Type      string          `json:"type"`
	Event     json.RawMessage `json:"event,omitempty"`
}

// SlackMessageEvent represents a message event from Slack
type SlackMessageEvent struct {
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	User      string    `json:"user"`
	Timestamp string    `json:"ts"`
	Text      string    `json:"text"`
	Files     []SlackFile `json:"files,omitempty"`
	ThreadTS  string    `json:"thread_ts,omitempty"`
}

// SlackFile represents a file in Slack
type SlackFile struct {
	ID    string `json:"id"`
	URL   string `json:"url_private"`
	Name  string `json:"name"`
	MimeType string `json:"mimetype"`
	Size  int64  `json:"size"`
}

// NewSlackAdapter creates a new Slack adapter
func NewSlackAdapter() *SlackAdapter {
	caps := CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    true,
		Threads:      true,
		Edit:         false,
		Delete:       false,
		Typing:       true,
		ReadReceipts: false,
	}

	return &SlackAdapter{
		BaseAdapter: NewBaseAdapter("slack", "1.0.0", caps),
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Initialize sets up the Slack adapter with configuration
func (s *SlackAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	if err := s.BaseAdapter.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract credentials (injected from keystore)
	s.botToken = config.Credentials["bot_token"]
	if s.botToken == "" {
		return NewAdapterError(ErrAuthFailed, "bot_token is required", false)
	}

	s.webhookURL = config.WebhookURL
	s.ctx, s.cancel = context.WithCancel(context.Background())

	return nil
}

// Start begins processing Slack events
func (s *SlackAdapter) Start(ctx context.Context) error {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// Start webhook listener if webhook URL is configured
	if s.webhookURL != "" {
		go s.webhookListener()
	}

	// Verify connection
	return s.verifyConnection(ctx)
}

// Shutdown gracefully stops the adapter
func (s *SlackAdapter) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	return nil
}

// SendMessage sends a message to Slack
func (s *SlackAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	if err := ValidateMessage(msg); err != nil {
		return nil, err
	}

	// Build Slack message payload
	slackMsg := SlackMessage{
		Channel:   target.Channel,
		Text:      msg.Content,
		Username:  "ArmorClaw",
		IconEmoji: ":robot_face:",
	}

	// Add thread information if replying
	if msg.ReplyTo != "" {
		slackMsg.ThreadTS = msg.ReplyTo
	}

	// Convert attachments
	if len(msg.Attachments) > 0 {
		for _, att := range msg.Attachments {
			slackMsg.Attachments = append(slackMsg.Attachments, SlackAttachment{
				Title:    att.Filename,
				Text:     fmt.Sprintf("Size: %d bytes", att.Size),
				ImageURL: att.URL,
			})
		}
	}

	// Marshal payload
	payload, err := json.Marshal(slackMsg)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to marshal message", false)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://slack.com/api/chat.postMessage", bytes.NewReader(payload))
	if err != nil {
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		s.RecordError(err)
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to read response", true)
	}

	// Parse response
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Timestamp string `json:"ts"`
		Channel string `json:"channel"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to parse response", true)
	}

	// Check for errors
	if !result.OK {
		s.RecordError(fmt.Errorf("slack API error: %s", result.Error))
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(mapSlackError(result.Error), result.Error, isRetryableSlackError(result.Error)),
		}, nil
	}

	s.RecordSent()

	return &SendResult{
		MessageID: result.Timestamp,
		Delivered: true,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"channel": result.Channel,
			"ts":      result.Timestamp,
		},
	}, nil
}

// ReceiveEvent handles an incoming Slack event
func (s *SlackAdapter) ReceiveEvent(event ExternalEvent) error {
	if event.Platform != s.Platform() {
		return NewAdapterError(ErrValidation, "platform mismatch", false)
	}

	// Verify signature if present
	if event.Signature != "" {
		secret := s.config.Credentials["signing_secret"]
		if !VerifySignature(event.Content, event.Signature, secret) {
			return NewAdapterError(ErrAuthFailed, "invalid signature", false)
		}
	}

	// Parse event
	var slackEvent SlackMessageEvent
	if err := json.Unmarshal([]byte(event.Content), &slackEvent); err != nil {
		return NewAdapterError(ErrValidation, "failed to parse event", false)
	}

	s.RecordReceived()
	return nil
}

// mapSlackError maps Slack API errors to AdapterError codes
func mapSlackError(slackErr string) ErrorCode {
	switch slackErr {
	case "rate_limited":
		return ErrRateLimited
	case "invalid_auth", "account_inactive":
		return ErrAuthFailed
	case "channel_not_found", "channel_is_archived":
		return ErrInvalidTarget
	case "timeout":
		return ErrTimeout
	default:
		return ErrPlatformError
	}
}

// isRetryableSlackError determines if a Slack error is retryable
func isRetryableSlackError(slackErr string) bool {
	retryableErrors := map[string]bool{
		"rate_limited":     true,
		"timeout":          true,
		"server_error":     true,
		"service_unavailable": true,
	}
	return retryableErrors[slackErr]
}

// verifyConnection verifies the Slack API connection
func (s *SlackAdapter) verifyConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://slack.com/api/auth.test", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed: %s", resp.Status)
	}

	return nil
}

// webhookListener listens for Slack webhook events
func (s *SlackAdapter) webhookListener() {
	// This would be implemented when integrating with the RPC server
	// For now, it's a placeholder for webhook handling
}

// GetChannelInfo retrieves information about a channel
func (s *SlackAdapter) GetChannelInfo(ctx context.Context, channelID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://slack.com/api/conversations.info?channel=%s", channelID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool                   `json:"ok"`
		Error string                 `json:"error"`
		Channel map[string]interface{} `json:"channel"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.Channel, nil
}

// HandleWebhook handles an incoming webhook request from Slack
func (s *SlackAdapter) HandleWebhook(r *http.Request) (*SlackEvent, error) {
	var event SlackEvent

	// Verify URL verification token
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body.Close()

	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}

	// Verify token
	verificationToken := s.config.Credentials["verification_token"]
	if event.Token != verificationToken {
		return nil, NewAdapterError(ErrAuthFailed, "invalid verification token", false)
	}

	return &event, nil
}

// StreamEvents starts streaming events from Slack (via WebSocket)
func (s *SlackAdapter) StreamEvents(ctx context.Context, callback func(ExternalEvent) error) error {
	// Slack uses a WebSocket-based RTM (Real Time Messaging) API
	// This is a placeholder for WebSocket implementation
	// Full implementation would:
	// 1. Connect to wss://slack.com/api/rtm.connect
	// 2. Parse incoming JSON events
	// 3. Convert to ExternalEvent and call callback
	return fmt.Errorf("not implemented: use webhooks for event delivery")
}

// SupportsCapabilities returns true if the adapter supports the given capabilities
func (s *SlackAdapter) SupportsCapabilities(caps CapabilitySet) bool {
	return s.Capabilities().Read == caps.Read &&
		s.Capabilities().Write == caps.Write &&
		s.Capabilities().Media == caps.Media
}

// GetRateLimitStatus returns current rate limit status
func (s *SlackAdapter) GetRateLimitStatus(ctx context.Context) map[string]interface{} {
	// Slack returns rate limit info in headers
	// This would make a call to check current status
	return map[string]interface{}{
		"tier":         s.config.Settings["rate_limit_tier"],
		"requests_per_second": s.config.RateLimits.RequestsPerSecond,
	}
}

// ParseEvent parses raw event data into an ExternalEvent
func (s *SlackAdapter) ParseEvent(raw json.RawMessage) (*ExternalEvent, error) {
	var slackEvent SlackMessageEvent
	if err := json.Unmarshal(raw, &slackEvent); err != nil {
		return nil, err
	}

	content, _ := json.Marshal(slackEvent)

	return &ExternalEvent{
		Platform:  s.Platform(),
		EventType: "message",
		Timestamp: time.Now(),
		Source:    slackEvent.Channel,
		Content:   string(content),
		Metadata: map[string]string{
			"user":      slackEvent.User,
			"timestamp": slackEvent.Timestamp,
			"thread_ts": slackEvent.ThreadTS,
		},
	}, nil
}

// FormatMessage formats a Message for Slack API
func (s *SlackAdapter) FormatMessage(msg Message) (interface{}, error) {
	slackMsg := SlackMessage{
		Text:      msg.Content,
		Username:  "ArmorClaw",
		IconEmoji: ":robot_face:",
	}

	if msg.ReplyTo != "" {
		slackMsg.ThreadTS = msg.ReplyTo
	}

	for _, att := range msg.Attachments {
		slackMsg.Attachments = append(slackMsg.Attachments, SlackAttachment{
			Title: att.Filename,
			Text:  fmt.Sprintf("%d bytes", att.Size),
		})
	}

	return slackMsg, nil
}

// ValidateConfig validates the adapter configuration
func (s *SlackAdapter) ValidateConfig(config AdapterConfig) error {
	if config.Credentials["bot_token"] == "" {
		return fmt.Errorf("bot_token is required")
	}
	if config.Settings["team_id"] == "" {
		return fmt.Errorf("team_id is recommended")
	}
	return nil
}

// Ping sends a ping to verify connectivity
func (s *SlackAdapter) Ping(ctx context.Context) error {
	return s.verifyConnection(ctx)
}

// GetDefaultCapabilities returns the default capabilities for Slack
func GetDefaultSlackCapabilities() CapabilitySet {
	return CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    true,
		Threads:      true,
		Edit:         true,
		Delete:       true,
		Typing:       true,
		ReadReceipts: false,
	}
}

// NewSlackAdapterWithCaps creates a new Slack adapter with custom capabilities
func NewSlackAdapterWithCaps(caps CapabilitySet) *SlackAdapter {
	return &SlackAdapter{
		BaseAdapter: NewBaseAdapter("slack", "1.0.0", caps),
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}
