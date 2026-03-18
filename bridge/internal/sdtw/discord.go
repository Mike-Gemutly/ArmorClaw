// Package sdtw provides Discord adapter implementation
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

// DiscordAdapter implements SDTWAdapter for Discord
type DiscordAdapter struct {
	*BaseAdapter
	client        *http.Client
	botToken      string
	guildID       string
	commandPrefix string
	mu            sync.RWMutex
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// DiscordMessage represents a Discord message payload
type DiscordMessage struct {
	Content          string                   `json:"content"`
	TTS              bool                     `json:"tts,omitempty"`
	Embeds           []DiscordEmbed           `json:"embeds,omitempty"`
	MessageReference *DiscordMessageReference `json:"message_reference,omitempty"`
}

// DiscordEmbed represents a rich embed in Discord
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedField represents a field in an embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordMessageReference represents a message reference for replies
type DiscordMessageReference struct {
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id"`
}

// DiscordEvent represents a Discord event payload
type DiscordEvent struct {
	Op int             `json:"op"` // Opcode
	D  json.RawMessage `json:"d"`  // Event data
	T  string          `json:"t"`  // Event type
	S  int             `json:"s"`  // Sequence number
}

// Discord Gateway opcodes (GatewayOp)
const (
	GatewayOpDispatch            = 0  // Dispatch
	GatewayOpHeartbeat           = 1  // Heartbeat
	GatewayOpIdentify            = 2  // Identify
	GatewayOpPresence            = 3  // Presence Update
	GatewayOpVoiceState          = 4  // Voice State Update
	GatewayOpResume              = 6  // Resume
	GatewayOpReconnect           = 7  // Reconnect
	GatewayOpRequestGuildMembers = 8  // Request Guild Members
	GatewayOpInvalidSession      = 9  // Invalid Session
	GatewayOpHello               = 10 // Hello
	GatewayOpHeartbeatACK        = 11 // Heartbeat ACK
)

// GatewayHello represents Gateway HELLO payload
type GatewayHello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// GatewayIdentify represents Gateway IDENTIFY payload
type GatewayIdentify struct {
	Token      string              `json:"token"`
	Properties GatewayIdentifyConn `json:"properties"`
	Intents    int                 `json:"intents"`
}

// GatewayIdentifyConn represents connection properties for IDENTIFY
type GatewayIdentifyConn struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

// GatewayResume represents Gateway RESUME payload
type GatewayResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int    `json:"seq"`
}

// GatewayHeartbeat represents Gateway HEARTBEAT payload
type GatewayHeartbeat struct {
	Seq int `json:"seq"`
}

// GatewayReady represents Gateway READY event data
type GatewayReady struct {
	Version   int    `json:"v"`
	SessionID string `json:"session_id"`
	User      struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
	} `json:"user"`
}

// DiscordMessageCreate represents a MESSAGE_CREATE event
type DiscordMessageCreate struct {
	ID               string                   `json:"id"`
	ChannelID        string                   `json:"channel_id"`
	Author           DiscordUser              `json:"author"`
	Content          string                   `json:"content"`
	Timestamp        string                   `json:"timestamp"`
	Edited           string                   `json:"edited_timestamp"`
	TTS              bool                     `json:"tts"`
	Mentions         []DiscordUser            `json:"mentions"`
	Embeds           []DiscordEmbed           `json:"embeds"`
	Attachments      []DiscordAttachment      `json:"attachments"`
	MessageReference *DiscordMessageReference `json:"message_reference,omitempty"`
}

// DiscordUser represents a Discord user
type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot"`
}

// DiscordAttachment represents a file attachment in Discord
type DiscordAttachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// DiscordGatewayResponse represents the gateway connection response
type DiscordGatewayResponse struct {
	URL    string `json:"url"`
	Shards int    `json:"shards"`
}

// NewDiscordAdapter creates a new Discord adapter
func NewDiscordAdapter() *DiscordAdapter {
	caps := CapabilitySet{
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

	return &DiscordAdapter{
		BaseAdapter:   NewBaseAdapter("discord", "1.0.0", caps),
		client:        &http.Client{Timeout: 30 * time.Second},
		commandPrefix: "!",
	}
}

// Initialize sets up the Discord adapter with configuration
func (d *DiscordAdapter) Initialize(ctx context.Context, config AdapterConfig) error {
	if err := d.BaseAdapter.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract credentials (injected from keystore)
	d.botToken = config.Credentials["bot_token"]
	if d.botToken == "" {
		return NewAdapterError(ErrAuthFailed, "bot_token is required", false)
	}

	d.guildID = config.Settings["guild_id"]
	if prefix, ok := config.Settings["command_prefix"]; ok {
		d.commandPrefix = prefix
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())

	return nil
}

// Start begins processing Discord events
func (d *DiscordAdapter) Start(ctx context.Context) error {
	d.mu.Lock()
	d.running = true
	d.mu.Unlock()

	// Discord uses Gateway WebSocket for real-time events
	// For now, we'll verify connection and set up for webhook-style delivery
	return d.verifyConnection(ctx)
}

// Shutdown gracefully stops the adapter
func (d *DiscordAdapter) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	d.running = false
	d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
	}

	return nil
}

// SendMessage sends a message to Discord
func (d *DiscordAdapter) SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error) {
	if err := ValidateMessage(msg); err != nil {
		return nil, err
	}

	// Build Discord message payload
	discordMsg := DiscordMessage{
		Content: msg.Content,
		TTS:     false,
	}

	// Add embed for rich content
	if msg.Metadata != nil {
		if title, ok := msg.Metadata["title"]; ok {
			discordMsg.Embeds = append(discordMsg.Embeds, DiscordEmbed{
				Title:       title,
				Description: msg.Content,
			})
		}
	}

	// Add message reference for replies
	if msg.ReplyTo != "" {
		discordMsg.MessageReference = &DiscordMessageReference{
			MessageID: msg.ReplyTo,
			ChannelID: target.Channel,
		}
	}

	// Marshal payload
	payload, err := json.Marshal(discordMsg)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to marshal message", false)
	}

	// Create request
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", target.Channel)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to read response", true)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		json.Unmarshal(body, &rateLimitError)
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(ErrRateLimited, rateLimitError.Message, true),
		}, nil
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to parse response", true)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		errorMsg := "unknown error"
		if msg, ok := result["message"].(string); ok {
			errorMsg = msg
		}
		d.RecordError(fmt.Errorf("discord API error: %s", errorMsg))
		return &SendResult{
			Delivered: false,
			Timestamp: time.Now(),
			Error:     NewAdapterError(mapDiscordError(resp.StatusCode, errorMsg), errorMsg, true),
		}, nil
	}

	d.RecordSent()

	// Extract message ID from response
	messageID, _ := result["id"].(string)

	return &SendResult{
		MessageID: messageID,
		Delivered: true,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"channel_id": target.Channel,
		},
	}, nil
}

// EditMessage edits an existing message on Discord
func (d *DiscordAdapter) EditMessage(ctx context.Context, target Target, messageID string, newContent string) error {
	payload := map[string]interface{}{"content": newContent}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return NewAdapterError(ErrValidation, "failed to marshal edit payload", false)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", target.Channel, messageID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewAdapterError(ErrPlatformError, "failed to read response", true)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		json.Unmarshal(body, &rateLimitError)
		return NewAdapterError(ErrRateLimited, rateLimitError.Message, true)
	}

	if resp.StatusCode >= 400 {
		var errResponse struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.Unmarshal(body, &errResponse)
		errorMsg := errResponse.Message
		if errorMsg == "" {
			errorMsg = "unknown error"
		}
		d.RecordError(fmt.Errorf("discord API error: %s", errorMsg))
		return NewAdapterError(mapDiscordError(resp.StatusCode, errorMsg), errorMsg, true)
	}

	return nil
}

// DeleteMessage deletes a message from Discord
func (d *DiscordAdapter) DeleteMessage(ctx context.Context, target Target, messageID string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", target.Channel, messageID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &rateLimitError)
		return NewAdapterError(ErrRateLimited, rateLimitError.Message, true)
	}

	if resp.StatusCode >= 400 {
		var errResponse struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &errResponse)
		errorMsg := errResponse.Message
		if errorMsg == "" {
			errorMsg = "unknown error"
		}
		d.RecordError(fmt.Errorf("discord API error: %s", errorMsg))
		return NewAdapterError(mapDiscordError(resp.StatusCode, errorMsg), errorMsg, true)
	}

	return nil
}

// ReceiveEvent handles an incoming Discord event
func (d *DiscordAdapter) ReceiveEvent(event ExternalEvent) error {
	if event.Platform != d.Platform() {
		return NewAdapterError(ErrValidation, "platform mismatch", false)
	}

	// Verify signature if present
	if event.Signature != "" {
		// Discord uses a different signature format
		secret := d.config.Credentials["public_key"]
		if !VerifySignature(event.Content, event.Signature, secret) {
			return NewAdapterError(ErrAuthFailed, "invalid signature", false)
		}
	}

	// Parse event
	var discordEvent DiscordMessageCreate
	if err := json.Unmarshal([]byte(event.Content), &discordEvent); err != nil {
		return NewAdapterError(ErrValidation, "failed to parse event", false)
	}

	d.RecordReceived()
	return nil
}

// mapDiscordError maps Discord HTTP status codes to AdapterError codes
func mapDiscordError(statusCode int, message string) ErrorCode {
	switch statusCode {
	case 401, 403:
		return ErrAuthFailed
	case 404:
		return ErrInvalidTarget
	case 429:
		return ErrRateLimited
	case 500, 502, 503, 504:
		return ErrNetworkError
	default:
		return ErrPlatformError
	}
}

// verifyConnection verifies the Discord API connection
func (d *DiscordAdapter) verifyConnection(ctx context.Context) error {
	// Get current user to verify token
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed: %s", resp.Status)
	}

	return nil
}

// GetGatewayURL returns the WebSocket gateway URL
func (d *DiscordAdapter) GetGatewayURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://discord.com/api/v10/gateway", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var gatewayResp DiscordGatewayResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayResp); err != nil {
		return "", err
	}

	return gatewayResp.URL, nil
}

// GetChannelInfo retrieves information about a channel
func (d *DiscordAdapter) GetChannelInfo(ctx context.Context, channelID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://discord.com/api/v10/channels/%s", channelID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("channel info failed: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// HandleInteraction handles a Discord slash command interaction
func (d *DiscordAdapter) HandleInteraction(ctx context.Context, payload []byte) (map[string]interface{}, error) {
	var interaction map[string]interface{}
	if err := json.Unmarshal(payload, &interaction); err != nil {
		return nil, err
	}

	// Respond to interaction
	response := map[string]interface{}{
		"type": 4, // ChannelMessageWithSource
		"data": map[string]interface{}{
			"content": "Command received",
		},
	}

	return response, nil
}

// StreamGateway connects to the Discord Gateway for real-time events
func (d *DiscordAdapter) StreamGateway(ctx context.Context, callback func(ExternalEvent) error) error {
	gatewayURL, err := d.GetGatewayURL(ctx)
	if err != nil {
		return err
	}
	_ = gatewayURL // Will be used when WebSocket is implemented

	// Connect via WebSocket
	// This is a placeholder for full WebSocket implementation
	// Would need to:
	// 1. Connect to wss://gateway.discord.gg/?v=10&encoding=json
	// 2. Send heartbeat payload
	// 3. Identify with bot token
	// 4. Dispatch events to callback
	return fmt.Errorf("gateway streaming not implemented: use webhooks for event delivery")
}

// SupportsCapabilities returns true if the adapter supports the given capabilities
func (d *DiscordAdapter) SupportsCapabilities(caps CapabilitySet) bool {
	return d.Capabilities().Read == caps.Read &&
		d.Capabilities().Write == caps.Write &&
		d.Capabilities().Media == caps.Media
}

// GetRateLimitStatus returns current rate limit status
func (d *DiscordAdapter) GetRateLimitStatus(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"global_limit":      50, // requests per second global
		"per_channel_limit": 5,  // requests per second per channel
	}
}

// ParseEvent parses raw event data into an ExternalEvent
func (d *DiscordAdapter) ParseEvent(raw json.RawMessage) (*ExternalEvent, error) {
	var discordEvent DiscordMessageCreate
	if err := json.Unmarshal(raw, &discordEvent); err != nil {
		return nil, err
	}

	content, _ := json.Marshal(discordEvent)

	return &ExternalEvent{
		Platform:  d.Platform(),
		EventType: "message",
		Timestamp: time.Now(),
		Source:    discordEvent.ChannelID,
		Content:   string(content),
		Metadata: map[string]string{
			"user_id":    discordEvent.Author.ID,
			"username":   discordEvent.Author.Username,
			"message_id": discordEvent.ID,
		},
	}, nil
}

// FormatMessage formats a Message for Discord API
func (d *DiscordAdapter) FormatMessage(msg Message) (interface{}, error) {
	discordMsg := DiscordMessage{
		Content: msg.Content,
	}

	if msg.ReplyTo != "" {
		discordMsg.MessageReference = &DiscordMessageReference{
			MessageID: msg.ReplyTo,
		}
	}

	for _, att := range msg.Attachments {
		discordMsg.Embeds = append(discordMsg.Embeds, DiscordEmbed{
			Title: att.Filename,
			Fields: []DiscordEmbedField{
				{Name: "Size", Value: fmt.Sprintf("%d bytes", att.Size)},
			},
		})
	}

	return discordMsg, nil
}

// ValidateConfig validates the adapter configuration
func (d *DiscordAdapter) ValidateConfig(config AdapterConfig) error {
	if config.Credentials["bot_token"] == "" {
		return fmt.Errorf("bot_token is required")
	}
	if config.Credentials["client_id"] == "" {
		return fmt.Errorf("client_id is recommended")
	}
	return nil
}

// Ping sends a ping to verify connectivity
func (d *DiscordAdapter) Ping(ctx context.Context) error {
	return d.verifyConnection(ctx)
}

// GetDefaultCapabilities returns the default capabilities for Discord
func GetDefaultDiscordCapabilities() CapabilitySet {
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

// ExecuteSlashCommand executes a slash command on Discord
func (d *DiscordAdapter) ExecuteSlashCommand(ctx context.Context, guildID, commandID, commandName string, options map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/commands/%s/%s", guildID, commandID, commandName)

	payload, err := json.Marshal(map[string]interface{}{
		"type": 1, // Chat input
		"data": map[string]interface{}{
			"name":    commandName,
			"options": options,
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}

// NewDiscordAdapterWithCaps creates a new Discord adapter with custom capabilities
func NewDiscordAdapterWithCaps(caps CapabilitySet) *DiscordAdapter {
	return &DiscordAdapter{
		BaseAdapter: NewBaseAdapter("discord", "1.0.0", caps),
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// SendReaction adds a reaction to a Discord message
// Discord API: PUT /channels/{channel.id}/messages/{message.id}/reactions/{emoji}/@me
func (d *DiscordAdapter) SendReaction(ctx context.Context, target Target, messageID string, emoji string) error {
	if messageID == "" {
		return NewAdapterError(ErrValidation, "message ID is required", false)
	}

	// Format emoji - remove <:name:id> format for custom emojis
	formattedEmoji := formatDiscordEmoji(emoji)

	// Build URL
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s/reactions/%s/@me",
		target.Channel, messageID, formattedEmoji)

	// Create request (PUT method for adding reaction)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rateLimitError); err == nil {
			return NewAdapterError(ErrRateLimited, rateLimitError.Message, true)
		}
		return NewAdapterError(ErrRateLimited, "rate limited", true)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		d.RecordError(fmt.Errorf("discord API error: %s", errorResp.Message))
		return NewAdapterError(mapDiscordError(resp.StatusCode, errorResp.Message), errorResp.Message, true)
	}

	return nil
}

// RemoveReaction removes a reaction from a Discord message
// Discord API: DELETE /channels/{channel.id}/messages/{message.id}/reactions/{emoji}/@me
func (d *DiscordAdapter) RemoveReaction(ctx context.Context, target Target, messageID string, emoji string) error {
	if messageID == "" {
		return NewAdapterError(ErrValidation, "message ID is required", false)
	}

	// Format emoji - remove <:name:id> format for custom emojis
	formattedEmoji := formatDiscordEmoji(emoji)

	// Build URL
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s/reactions/%s/@me",
		target.Channel, messageID, formattedEmoji)

	// Create request (DELETE method for removing reaction)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rateLimitError); err == nil {
			return NewAdapterError(ErrRateLimited, rateLimitError.Message, true)
		}
		return NewAdapterError(ErrRateLimited, "rate limited", true)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		d.RecordError(fmt.Errorf("discord API error: %s", errorResp.Message))
		return NewAdapterError(mapDiscordError(resp.StatusCode, errorResp.Message), errorResp.Message, true)
	}

	return nil
}

// GetReactions retrieves current reactions for a Discord message
// Discord API: GET /channels/{channel.id}/messages/{message.id}/reactions/{emoji}
func (d *DiscordAdapter) GetReactions(ctx context.Context, target Target, messageID string) ([]Reaction, error) {
	if messageID == "" {
		return nil, NewAdapterError(ErrValidation, "message ID is required", false)
	}

	// Build URL (without emoji to get all reactions)
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s/reactions",
		target.Channel, messageID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		d.RecordError(err)
		return nil, NewAdapterError(ErrNetworkError, err.Error(), true)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitError struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retry_after"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rateLimitError); err == nil {
			return nil, NewAdapterError(ErrRateLimited, rateLimitError.Message, true)
		}
		return nil, NewAdapterError(ErrRateLimited, "rate limited", true)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		d.RecordError(fmt.Errorf("discord API error: %s", errorResp.Message))
		return nil, NewAdapterError(mapDiscordError(resp.StatusCode, errorResp.Message), errorResp.Message, true)
	}

	// Parse response - Discord returns an array of user objects
	var users []struct {
		ID   string `json:"id"`
		Name string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, NewAdapterError(ErrPlatformError, "failed to parse reactions response", true)
	}

	// Build reaction list
	reactions := make([]Reaction, 0, len(users))
	now := time.Now()
	formattedDiscordEmoji := formatDiscordEmoji(emoji)

	for _, user := range users {
		isCustom := strings.HasPrefix(formattedDiscordEmoji, ":")
		reactions = append(reactions, Reaction{
			Emoji:     formattedDiscordEmoji,
			Count:     len(users), // Count per emoji
			UserIDs:   []string{user.ID},
			Timestamp: now,
			IsCustom:  isCustom,
			CustomURL: "", // Would require additional lookup for emoji URL
		})
	}

	return reactions, nil
}

// formatDiscordEmoji formats emoji for Discord API
// Handles <:name:id> format by extracting the name
func formatDiscordEmoji(emoji string) string {
	// If it's a custom emoji in <:name:id> format, extract just the name
	if strings.HasPrefix(emoji, ":") && strings.HasSuffix(emoji, ">") {
		parts := strings.SplitN(emoji[1:len(emoji)-1], ":", 2)
		if len(parts) == 2 {
			return parts[0] // Return just the name part
		}
	}
	// For regular emojis, return as-is
	return emoji
}
