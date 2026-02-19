// Package adapter provides platform adapters for SDTW integration.
// This file implements the Slack adapter for GAP #9.
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/queue"
	"github.com/armorclaw/bridge/pkg/logger"
	"log/slog"
)

// SlackConfig holds Slack adapter configuration
type SlackConfig struct {
	// WorkspaceID is the Slack workspace/team ID (starts with T)
	WorkspaceID string `json:"workspace_id"`

	// BotToken is the xoxb- token for bot authentication
	BotToken string `json:"bot_token"`

	// AppToken is the xapp- token for Socket Mode (optional)
	AppToken string `json:"app_token,omitempty"`

	// Channels to monitor/forward
	Channels []string `json:"channels"`

	// MatrixRoom is the destination Matrix room
	MatrixRoom string `json:"matrix_room"`

	// UseSocketMode enables Socket Mode for real-time events
	UseSocketMode bool `json:"use_socket_mode"`

	// RateLimitRPS is the rate limit in requests per second
	RateLimitRPS int `json:"rate_limit_rps"`
}

// SlackAdapter implements the platform adapter for Slack
type SlackAdapter struct {
	config     SlackConfig
	httpClient *http.Client
	queue      *queue.MessageQueue
	log        *logger.Logger
	securityLog *logger.SecurityLogger

	// State
	mu          sync.RWMutex
	connected   bool
	lastSync    time.Time
	userCache   map[string]*SlackUser
	channelCache map[string]*SlackChannel
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// SlackUser represents a Slack user
type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Profile  struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		ImageURL    string `json:"image_72"`
	} `json:"profile"`
	IsBot bool `json:"is_bot"`
}

// SlackChannel represents a Slack channel
type SlackChannel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsChannel   bool   `json:"is_channel"`
	IsPrivate   bool   `json:"is_private"`
	IsMPIM      bool   `json:"is_mpim"`
	IsGroup     bool   `json:"is_group"`
	NumMembers  int    `json:"num_members"`
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	Type        string `json:"type"`
	Channel     string `json:"channel"`
	User        string `json:"user"`
	Text        string `json:"text"`
	Ts          string `json:"ts"`
	ThreadTs    string `json:"thread_ts,omitempty"`
	Blocks      []interface{} `json:"blocks,omitempty"`
	Attachments []interface{} `json:"attachments,omitempty"`
}

// SlackEvent represents a Slack event
type SlackEvent struct {
	Type    string          `json:"type"`
	Event   json.RawMessage `json:"event"`
	TeamID  string          `json:"team_id"`
	APIAppID string         `json:"api_app_id"`
}

// Slack API response structures
type slackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type slackAuthTestResponse struct {
	slackAPIResponse
	URL     string `json:"url"`
	Team    string `json:"team"`
	User    string `json:"user"`
	TeamID  string `json:"team_id"`
	UserID  string `json:"user_id"`
	BotID   string `json:"bot_id"`
}

type slackConversationsListResponse struct {
	slackAPIResponse
	Channels []SlackChannel `json:"channels"`
}

type slackUsersInfoResponse struct {
	slackAPIResponse
	User SlackUser `json:"user"`
}

type slackPostMessageResponse struct {
	slackAPIResponse
	Ts      string `json:"ts"`
	Channel string `json:"channel"`
	Message SlackMessage `json:"message"`
}

type slackConversationsHistoryResponse struct {
	slackAPIResponse
	Messages   []SlackMessage `json:"messages"`
	HasMore    bool           `json:"has_more"`
	PinCount   int            `json:"pin_count"`
	Latest     SlackMessage   `json:"latest"`
}

var (
	ErrSlackAuthFailed     = errors.New("slack authentication failed")
	ErrSlackRateLimited    = errors.New("slack rate limited")
	ErrSlackChannelNotFound = errors.New("slack channel not found")
	ErrSlackNotConnected   = errors.New("slack adapter not connected")
)

const (
	slackAPIBaseURL = "https://slack.com/api"
	defaultTimeout  = 30 * time.Second
)

// NewSlackAdapter creates a new Slack adapter
func NewSlackAdapter(config SlackConfig, msgQueue *queue.MessageQueue) (*SlackAdapter, error) {
	if config.BotToken == "" {
		return nil, errors.New("bot token is required")
	}

	if config.WorkspaceID == "" {
		return nil, errors.New("workspace ID is required")
	}

	if config.RateLimitRPS == 0 {
		config.RateLimitRPS = 1 // Default to 1 request per second
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &SlackAdapter{
		config: config,
		queue:  msgQueue,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		log:        logger.Global().WithComponent("slack"),
		securityLog: logger.NewSecurityLogger(logger.Global().WithComponent("slack")),
		userCache:   make(map[string]*SlackUser),
		channelCache: make(map[string]*SlackChannel),
		ctx:        ctx,
		cancel:     cancel,
	}

	return adapter, nil
}

// Connect authenticates with Slack and establishes connection
func (s *SlackAdapter) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Test authentication
	resp, err := s.authTest(ctx)
	if err != nil {
		return fmt.Errorf("auth test failed: %w", err)
	}

	if !resp.OK {
		return fmt.Errorf("%w: %s", ErrSlackAuthFailed, resp.Error)
	}

	// Verify team ID matches
	if resp.TeamID != s.config.WorkspaceID {
		return fmt.Errorf("workspace ID mismatch: expected %s, got %s",
			s.config.WorkspaceID, resp.TeamID)
	}

	s.connected = true
	s.lastSync = time.Now()

	s.securityLog.LogSecurityEvent("slack_connected",
		slog.String("workspace_id", s.config.WorkspaceID),
		slog.String("team_name", resp.Team),
		slog.String("bot_user", resp.User),
	)

	// Start background sync
	s.wg.Add(1)
	go s.syncLoop()

	return nil
}

// Disconnect closes the Slack connection
func (s *SlackAdapter) Disconnect() error {
	s.mu.Lock()
	s.connected = false
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()

	s.securityLog.LogSecurityEvent("slack_disconnected",
		slog.String("workspace_id", s.config.WorkspaceID),
	)

	return nil
}

// IsConnected returns connection status
func (s *SlackAdapter) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// SendMessage sends a message to a Slack channel
func (s *SlackAdapter) SendMessage(ctx context.Context, channelID, text string, options ...MessageOption) (*MessageResult, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, ErrSlackNotConnected
	}
	s.mu.RUnlock()

	// Build message payload
	payload := map[string]interface{}{
		"channel": channelID,
		"text":    text,
	}

	// Apply options
	for _, opt := range options {
		opt(payload)
	}

	// Send to Slack API
	resp, err := s.postMessage(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to post message: %w", err)
	}

	s.securityLog.LogSecurityEvent("slack_message_sent",
		slog.String("channel_id", channelID),
		slog.String("message_ts", resp.Ts),
	)

	return &MessageResult{
		MessageID: resp.Ts,
		ChannelID: resp.Channel,
		Timestamp: time.Now(),
	}, nil
}

// GetMessages retrieves messages from a channel
func (s *SlackAdapter) GetMessages(ctx context.Context, channelID string, limit int) ([]*PlatformMessage, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, ErrSlackNotConnected
	}
	s.mu.RUnlock()

	resp, err := s.getConversationHistory(ctx, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*PlatformMessage, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// Get user info
		user, _ := s.getUser(ctx, msg.User)

		messages = append(messages, &PlatformMessage{
			ID:        msg.Ts,
			ChannelID: msg.Channel,
			UserID:    msg.User,
			UserName:  s.getUserName(user),
			Text:      msg.Text,
			Timestamp: s.parseSlackTimestamp(msg.Ts),
			ThreadID:  msg.ThreadTs,
		})
	}

	return messages, nil
}

// GetChannels returns available channels
func (s *SlackAdapter) GetChannels(ctx context.Context) ([]*SlackChannel, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, ErrSlackNotConnected
	}
	s.mu.RUnlock()

	resp, err := s.listConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	// Update cache
	s.mu.Lock()
	for _, ch := range resp.Channels {
		s.channelCache[ch.ID] = &ch
	}
	s.mu.Unlock()

	result := make([]*SlackChannel, len(resp.Channels))
	for i := range resp.Channels {
		result[i] = &resp.Channels[i]
	}

	return result, nil
}

// GetUserInfo returns user information
func (s *SlackAdapter) GetUserInfo(ctx context.Context, userID string) (*SlackUser, error) {
	return s.getUser(ctx, userID)
}

// Platform returns the platform type
func (s *SlackAdapter) Platform() string {
	return "slack"
}

// Internal API methods

func (s *SlackAdapter) authTest(ctx context.Context) (*slackAuthTestResponse, error) {
	var resp slackAuthTestResponse
	err := s.apiCall(ctx, "auth.test", nil, &resp)
	return &resp, err
}

func (s *SlackAdapter) postMessage(ctx context.Context, payload map[string]interface{}) (*slackPostMessageResponse, error) {
	var resp slackPostMessageResponse
	err := s.apiCall(ctx, "chat.postMessage", payload, &resp)
	return &resp, err
}

func (s *SlackAdapter) getConversationHistory(ctx context.Context, channelID string, limit int) (*slackConversationsHistoryResponse, error) {
	params := map[string]interface{}{
		"channel": channelID,
		"limit":   limit,
	}

	var resp slackConversationsHistoryResponse
	err := s.apiCall(ctx, "conversations.history", params, &resp)
	return &resp, err
}

func (s *SlackAdapter) listConversations(ctx context.Context) (*slackConversationsListResponse, error) {
	params := map[string]interface{}{
		"types":  "public_channel,private_channel,mpim,im",
		"limit":  1000,
	}

	var resp slackConversationsListResponse
	err := s.apiCall(ctx, "conversations.list", params, &resp)
	return &resp, err
}

func (s *SlackAdapter) getUser(ctx context.Context, userID string) (*SlackUser, error) {
	// Check cache first
	s.mu.RLock()
	if user, ok := s.userCache[userID]; ok {
		s.mu.RUnlock()
		return user, nil
	}
	s.mu.RUnlock()

	// Fetch from API
	params := map[string]interface{}{
		"user": userID,
	}

	var resp slackUsersInfoResponse
	if err := s.apiCall(ctx, "users.info", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Error)
	}

	// Cache result
	s.mu.Lock()
	s.userCache[userID] = &resp.User
	s.mu.Unlock()

	return &resp.User, nil
}

func (s *SlackAdapter) apiCall(ctx context.Context, method string, params map[string]interface{}, result interface{}) error {
	// Build URL
	apiURL := fmt.Sprintf("%s/%s", slackAPIBaseURL, method)

	// Build form data
	formData := url.Values{}
	for k, v := range params {
		switch val := v.(type) {
		case string:
			formData.Set(k, val)
		case int:
			formData.Set(k, fmt.Sprintf("%d", val))
		case bool:
			formData.Set(k, fmt.Sprintf("%t", val))
		case []string:
			for _, item := range val {
				formData.Add(k, item)
			}
		default:
			data, _ := json.Marshal(val)
			formData.Set(k, string(data))
		}
	}

	// Create request
	var body io.Reader
	if len(formData) > 0 {
		body = bytes.NewBufferString(formData.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+s.config.BotToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for rate limiting
	if resp.StatusCode == 429 {
		return ErrSlackRateLimited
	}

	return nil
}

func (s *SlackAdapter) syncLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sync()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *SlackAdapter) sync() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Refresh channels
	channels, err := s.GetChannels(ctx)
	if err != nil {
		s.log.Error("failed to sync channels", "error", err)
		return
	}

	s.log.Debug("synced channels", "count", len(channels))
	s.lastSync = time.Now()
}

func (s *SlackAdapter) getUserName(user *SlackUser) string {
	if user == nil {
		return "Unknown"
	}
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.RealName != "" {
		return user.RealName
	}
	return user.Name
}

func (s *SlackAdapter) parseSlackTimestamp(ts string) time.Time {
	// Slack timestamps are Unix timestamps with microseconds
	var sec, usec int64
	fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	return time.Unix(sec, usec*1000)
}

// MessageOption is a functional option for message customization
type MessageOption func(map[string]interface{})

// WithThread_ts sets the thread timestamp for threaded replies
func WithThreadTS(threadTs string) MessageOption {
	return func(payload map[string]interface{}) {
		payload["thread_ts"] = threadTs
	}
}

// WithBlocks adds block kit blocks to the message
func WithBlocks(blocks []interface{}) MessageOption {
	return func(payload map[string]interface{}) {
		payload["blocks"] = blocks
	}
}

// WithAttachments adds attachments to the message
func WithAttachments(attachments []interface{}) MessageOption {
	return func(payload map[string]interface{}) {
		payload["attachments"] = attachments
	}
}

// WithMarkdown enables/disables markdown parsing
func WithMarkdown(enabled bool) MessageOption {
	return func(payload map[string]interface{}) {
		payload["mrkdwn"] = enabled
	}
}

// MessageResult represents the result of sending a message
type MessageResult struct {
	MessageID string
	ChannelID string
	Timestamp time.Time
}

// PlatformMessage represents a message from a platform
type PlatformMessage struct {
	ID        string
	ChannelID string
	UserID    string
	UserName  string
	Text      string
	Timestamp time.Time
	ThreadID  string
}
