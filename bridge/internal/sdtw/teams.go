// Package sdtw provides Microsoft Teams adapter implementation
package sdtw

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// TeamsAdapter implements SDTWAdapter for Microsoft Teams
type TeamsAdapter struct {
	*BaseAdapter
	config      TeamsConfig
	client      *http.Client
	logger      *slog.Logger
	initialized bool
	running     bool
	mu          sync.RWMutex

	// Message handling
	eventChan chan ExternalEvent
}

// TeamsConfig holds Teams adapter configuration
type TeamsConfig struct {
	// Azure AD app registration
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id"`

	// Bot configuration
	BotID       string `json:"bot_id"`
	BotName     string `json:"bot_name"`
	ServiceURL  string `json:"service_url"`

	// OAuth tokens (obtained via OAuth flow)
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`

	// Webhook configuration
	WebhookURL   string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`

	// Feature flags
	Enabled      bool     `json:"enabled"`
	AllowPrivate bool     `json:"allow_private"`
	AllowTeams   []string `json:"allow_teams"`
}

// TeamsMessage represents a Teams message payload
type TeamsMessage struct {
	Type        string            `json:"type"`
	ID          string            `json:"id"`
	From        TeamsUser         `json:"from"`
	To          []TeamsUser       `json:"to"`
	Text        string            `json:"text"`
	Attachments []TeamsAttachment `json:"attachments,omitempty"`
}

// TeamsUser represents a Teams user
type TeamsUser struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	AADID string `json:"aad_id,omitempty"`
}

// TeamsAttachment represents an attachment in Teams
type TeamsAttachment struct {
	ContentType string                 `json:"contentType"`
	ContentURL  string                 `json:"contentUrl,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Content     map[string]interface{} `json:"content,omitempty"`
}

// TeamsActivity represents a Teams activity event (Bot Framework format)
type TeamsActivity struct {
	Type         string            `json:"type"`
	ID           string            `json:"id"`
	Timestamp    string            `json:"timestamp"`
	ServiceURL   string            `json:"serviceUrl"`
	ChannelID    string            `json:"channelId"`
	From         TeamsUser         `json:"from"`
	Conversation TeamsConversation `json:"conversation"`
	Text         string            `json:"text"`
	TextFormat   string            `json:"textFormat,omitempty"`
	Attachments  []TeamsAttachment `json:"attachments,omitempty"`
	Entities     []TeamsEntity     `json:"entities,omitempty"`
	Recipient    TeamsUser         `json:"recipient,omitempty"`
}

// TeamsConversation represents a conversation in Teams
type TeamsConversation struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	IsGroup   bool   `json:"isGroup,omitempty"`
	ConversationType string `json:"conversationType,omitempty"`
	TenantID  string `json:"tenantId,omitempty"`
}

// TeamsEntity represents an entity in a Teams message
type TeamsEntity struct {
	Type     string `json:"type"`
	Mentioned TeamsUser `json:"mentioned,omitempty"`
	Text     string `json:"text,omitempty"`
}

// TokenResponse represents Azure AD token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// GraphMessage represents a Microsoft Graph message
type GraphMessage struct {
	ID           string            `json:"id"`
	Subject      string            `json:"subject,omitempty"`
	Body         GraphBody         `json:"body,omitempty"`
	From         GraphRecipient    `json:"from,omitempty"`
	Received     time.Time         `json:"receivedDateTime,omitempty"`
	Conversation GraphConversation `json:"conversation,omitempty"`
}

// GraphBody represents message body content
type GraphBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// GraphRecipient represents a message recipient
type GraphRecipient struct {
	EmailAddress GraphEmail `json:"emailAddress"`
}

// GraphEmail represents an email address
type GraphEmail struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// GraphConversation represents a conversation reference
type GraphConversation struct {
	ID string `json:"id"`
}

// GraphSendRequest represents a request to send a message via Graph API
type GraphSendRequest struct {
	Body GraphBody `json:"body"`
}

// NewTeamsAdapter creates a new Teams adapter
func NewTeamsAdapter(config TeamsConfig) *TeamsAdapter {
	caps := GetDefaultTeamsCapabilities()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &TeamsAdapter{
		BaseAdapter: NewBaseAdapter("teams", "1.0.0", caps),
		config:      config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		eventChan: make(chan ExternalEvent, 100),
	}
}

// Initialize sets up the Teams adapter with configuration
func (t *TeamsAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.initialized {
		return nil
	}

	// Validate required configuration
	if t.config.ClientID == "" {
		return fmt.Errorf("client_id is required for Teams adapter")
	}
	if t.config.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for Teams adapter")
	}
	if t.config.TenantID == "" {
		return fmt.Errorf("tenant_id is required for Teams adapter")
	}

	// Initialize base adapter
	if err := t.BaseAdapter.Initialize(ctx, config); err != nil {
		return fmt.Errorf("base adapter initialization failed: %w", err)
	}

	// Get initial access token if not provided
	if t.config.AccessToken == "" {
		if err := t.refreshAccessToken(ctx); err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}
	}

	t.initialized = true
	t.logger.Info("Teams adapter initialized",
		"client_id", t.config.ClientID,
		"tenant_id", t.config.TenantID,
	)

	return nil
}

// Start begins processing Teams events
func (t *TeamsAdapter) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		return fmt.Errorf("adapter not initialized")
	}
	if t.running {
		return nil
	}

	// Start token refresh goroutine
	go t.tokenRefreshLoop(ctx)

	t.running = true
	t.logger.Info("Teams adapter started")

	return nil
}

// Shutdown gracefully stops the adapter
func (t *TeamsAdapter) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false
	close(t.eventChan)
	t.logger.Info("Teams adapter shutdown complete")

	return nil
}

// SendMessage sends a message to Teams
func (t *TeamsAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	if !t.initialized || !t.running {
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(ErrPlatformError, "adapter not running", false),
		}, nil
	}

	// Validate target
	if target.Channel == "" && target.UserID == "" {
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrInvalidTarget, "channel or user required", false),
		}, nil
	}

	// Check token validity
	if err := t.ensureValidToken(ctx); err != nil {
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrAuthFailed, err.Error(), true),
		}, nil
	}

	// Get team ID from metadata
	teamID := ""
	if target.Metadata != nil {
		teamID = target.Metadata["team_id"]
	}

	// Determine endpoint based on target type
	var endpoint string
	if target.Channel != "" {
		// Send to channel (team)
		if teamID == "" {
			return &SendResult{
				Delivered: false,
				Error:     NewAdapterError(ErrInvalidTarget, "team_id required for channel messages", false),
			}, nil
		}
		endpoint = fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages",
			teamID, target.Channel)
	} else {
		// Send direct message via chat
		endpoint = fmt.Sprintf("https://graph.microsoft.com/v1.0/chats/%s/messages", target.UserID)
	}

	// Build message payload
	payload := GraphSendRequest{
		Body: GraphBody{
			ContentType: "text",
			Content:     msg.Content,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrValidation, "failed to marshal message", false),
		}, nil
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrValidation, "failed to create request", false),
		}, nil
	}

	req.Header.Set("Authorization", "Bearer "+t.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrNetworkError, err.Error(), true),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &SendResult{
			Delivered: false,
			Error:     NewAdapterError(ErrPlatformError, fmt.Sprintf("API error: %s - %s", resp.Status, string(respBody)), true),
		}, nil
	}

	return &SendResult{
		Delivered: true,
		Timestamp: time.Now(),
	}, nil
}

// ReceiveEvent handles an incoming Teams event
func (t *TeamsAdapter) ReceiveEvent(event ExternalEvent) error {
	if !t.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	select {
	case t.eventChan <- event:
		return nil
	default:
		return fmt.Errorf("event channel full")
	}
}

// HandleWebhook processes incoming webhook requests from Teams
func (t *TeamsAdapter) HandleWebhook(body []byte, signature string) (*TeamsActivity, error) {
	// Validate webhook signature
	if t.config.WebhookSecret != "" {
		if !t.validateWebhookSignature(body, signature) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	var activity TeamsActivity
	if err := json.Unmarshal(body, &activity); err != nil {
		return nil, fmt.Errorf("failed to parse activity: %w", err)
	}

	return &activity, nil
}

// HealthCheck returns the current health status
func (t *TeamsAdapter) HealthCheck() (HealthStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.initialized {
		return HealthStatus{
			Connected: false,
			Error:     "adapter not initialized",
		}, nil
	}

	if !t.running {
		return HealthStatus{
			Connected: false,
			Error:     "adapter not running",
		}, nil
	}

	// Check token validity
	if time.Now().After(t.config.TokenExpiry) {
		return HealthStatus{
			Connected: false,
			Error:     "access token expired",
		}, nil
	}

	return HealthStatus{
		Connected: true,
		Latency:   0,
	}, nil
}

// Metrics returns the current metrics
func (t *TeamsAdapter) Metrics() (AdapterMetrics, error) {
	return t.BaseAdapter.Metrics()
}

// GetEventChannel returns the channel for receiving events
func (t *TeamsAdapter) GetEventChannel() <-chan ExternalEvent {
	return t.eventChan
}

// refreshAccessToken obtains a new access token from Azure AD
func (t *TeamsAdapter) refreshAccessToken(ctx context.Context) error {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", t.config.TenantID)

	data := map[string]string{
		"client_id":     t.config.ClientID,
		"client_secret": t.config.ClientSecret,
		"scope":         "https://graph.microsoft.com/.default",
		"grant_type":    "client_credentials",
	}

	if t.config.RefreshToken != "" {
		data["grant_type"] = "refresh_token"
		data["refresh_token"] = t.config.RefreshToken
	}

	formData := make([]byte, 0)
	for k, v := range data {
		if len(formData) > 0 {
			formData = append(formData, '&')
		}
		formData = append(formData, fmt.Sprintf("%s=%s", k, v)...)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(formData))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	t.config.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		t.config.RefreshToken = tokenResp.RefreshToken
	}
	t.config.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	t.logger.Info("Access token refreshed",
		"expires_in", tokenResp.ExpiresIn,
		"expires_at", t.config.TokenExpiry,
	)

	return nil
}

// ensureValidToken refreshes the token if it's about to expire
func (t *TeamsAdapter) ensureValidToken(ctx context.Context) error {
	// Refresh if token expires in less than 5 minutes
	if time.Until(t.config.TokenExpiry) < 5*time.Minute {
		return t.refreshAccessToken(ctx)
	}
	return nil
}

// tokenRefreshLoop periodically refreshes the access token
func (t *TeamsAdapter) tokenRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.RLock()
			running := t.running
			t.mu.RUnlock()

			if !running {
				return
			}

			if err := t.ensureValidToken(ctx); err != nil {
				t.logger.Error("Failed to refresh token", "error", err)
			}
		}
	}
}

// validateWebhookSignature validates the HMAC signature of incoming webhooks
func (t *TeamsAdapter) validateWebhookSignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(t.config.WebhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSig := base64.StdEncoding.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// processActivity converts a Teams activity to an SDTW message
func (t *TeamsAdapter) processActivity(activity *TeamsActivity) (*Message, error) {
	msg := &Message{
		ID:        activity.ID,
		Content:   activity.Text,
		Timestamp: time.Now(),
		Type:      MessageTypeText,
		Metadata:  make(map[string]string),
	}

	if activity.From.ID != "" {
		msg.Metadata["sender_id"] = activity.From.ID
		msg.Metadata["sender_name"] = activity.From.Name
	}

	if activity.Conversation.ID != "" {
		msg.Metadata["channel_id"] = activity.Conversation.ID
	}

	return msg, nil
}

// GetDefaultTeamsCapabilities returns the default capabilities for Teams
func GetDefaultTeamsCapabilities() CapabilitySet {
	return CapabilitySet{
		Read:         true,
		Write:        true,
		Media:        true,
		Reactions:    true,
		Threads:      true,
		Edit:         true,
		Delete:       true,
		Typing:       false, // Not supported via Graph API
		ReadReceipts: true,
	}
}
