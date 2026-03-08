package skills

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// SlackMessageParams represents parameters for sending Slack messages
type SlackMessageParams struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
	BotName     string            `json:"bot_name,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color     string                 `json:"color,omitempty"`
	Fallback  string                 `json:"fallback"`
	Title     string                 `json:"title,omitempty"`
	TitleLink string                 `json:"title_link,omitempty"`
	Text      string                 `json:"text"`
	ImageURL  string                 `json:"image_url,omitempty"`
	Fields    []SlackAttachmentField `json:"fields,omitempty"`
	Timestamp int64                  `json:"ts,omitempty"`
	Markdown  []string               `json:"mrkdwn_in,omitempty"`
}

// SlackAttachmentField represents a field in a Slack attachment
type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// SlackBlock represents a Slack block (newer format)
type SlackBlock struct {
	Type string      `json:"type"`
	Text interface{} `json:"text,omitempty"`
}

// SlackTextBlock represents text in a Slack block
type SlackTextBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// SlackMessageResult represents the result of sending a Slack message
type SlackMessageResult struct {
	OK        bool              `json:"ok"`
	Channel   string            `json:"channel"`
	Timestamp string            `json:"ts"`
	Message   SlackMessage      `json:"message,omitempty"`
	SentAt    time.Time         `json:"sent_at"`
	Provider  string            `json:"provider"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SlackMessage represents a Slack message object
type SlackMessage struct {
	Type        string            `json:"type"`
	Channel     string            `json:"channel"`
	User        string            `json:"user,omitempty"`
	Text        string            `json:"text"`
	Timestamp   string            `json:"ts"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
	BotID       string            `json:"bot_id,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackConfig represents Slack API configuration
type SlackConfig struct {
	Token      string `json:"token"`
	WebhookURL string `json:"webhook_url,omitempty"`
	TeamID     string `json:"team_id,omitempty"`
}

// ExecuteSlackMessage sends a message to Slack
func ExecuteSlackMessage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	slackParams, err := parseSlackMessageParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid slack message parameters: %w", err)
	}

	// Validate parameters
	if err := validateSlackMessageParams(slackParams); err != nil {
		return nil, fmt.Errorf("slack message validation failed: %w", err)
	}

	// Get Slack configuration (in production, this would come from secure storage)
	slackConfig, err := getSlackConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack configuration: %w", err)
	}

	// Send the message
	return sendSlackMessage(ctx, slackParams, slackConfig)
}

// parseSlackMessageParams parses Slack message parameters
func parseSlackMessageParams(params map[string]interface{}) (*SlackMessageParams, error) {
	slackParams := &SlackMessageParams{}

	// Extract required parameters
	if channel, ok := params["channel"].(string); ok {
		slackParams.Channel = strings.TrimSpace(channel)
	} else {
		return nil, fmt.Errorf("channel parameter is required and must be a string")
	}

	if text, ok := params["text"].(string); ok {
		slackParams.Text = text
	} else {
		return nil, fmt.Errorf("text parameter is required and must be a string")
	}

	// Extract optional parameters
	if threadTS, ok := params["thread_ts"].(string); ok {
		slackParams.ThreadTS = strings.TrimSpace(threadTS)
	}

	if botName, ok := params["bot_name"].(string); ok {
		slackParams.BotName = strings.TrimSpace(botName)
	}

	if iconEmoji, ok := params["icon_emoji"].(string); ok {
		slackParams.IconEmoji = strings.TrimSpace(iconEmoji)
	}

	if iconURL, ok := params["icon_url"].(string); ok {
		slackParams.IconURL = strings.TrimSpace(iconURL)
	}

	// Parse attachments if provided
	if attachments, ok := params["attachments"].([]interface{}); ok {
		slackParams.Attachments = make([]SlackAttachment, len(attachments))
		for i, attachment := range attachments {
			if attachmentMap, ok := attachment.(map[string]interface{}); ok {
				slackParams.Attachments[i] = parseSlackAttachment(attachmentMap)
			}
		}
	}

	// Parse blocks if provided
	if blocks, ok := params["blocks"].([]interface{}); ok {
		slackParams.Blocks = make([]SlackBlock, len(blocks))
		for i, block := range blocks {
			if blockMap, ok := block.(map[string]interface{}); ok {
				slackParams.Blocks[i] = parseSlackBlock(blockMap)
			}
		}
	}

	return slackParams, nil
}

// parseSlackAttachment parses a Slack attachment from map
func parseSlackAttachment(attachmentMap map[string]interface{}) SlackAttachment {
	attachment := SlackAttachment{}

	if color, ok := attachmentMap["color"].(string); ok {
		attachment.Color = color
	}
	if fallback, ok := attachmentMap["fallback"].(string); ok {
		attachment.Fallback = fallback
	}
	if title, ok := attachmentMap["title"].(string); ok {
		attachment.Title = title
	}
	if titleLink, ok := attachmentMap["title_link"].(string); ok {
		attachment.TitleLink = titleLink
	}
	if text, ok := attachmentMap["text"].(string); ok {
		attachment.Text = text
	}
	if imageURL, ok := attachmentMap["image_url"].(string); ok {
		attachment.ImageURL = imageURL
	}
	if timestamp, ok := attachmentMap["ts"].(float64); ok {
		attachment.Timestamp = int64(timestamp)
	}

	// Parse fields
	if fields, ok := attachmentMap["fields"].([]interface{}); ok {
		attachment.Fields = make([]SlackAttachmentField, len(fields))
		for i, field := range fields {
			if fieldMap, ok := field.(map[string]interface{}); ok {
				attachment.Fields[i] = SlackAttachmentField{
					Title: getStringField(fieldMap, "title"),
					Value: getStringField(fieldMap, "value"),
					Short: getBoolField(fieldMap, "short"),
				}
			}
		}
	}

	return attachment
}

// parseSlackBlock parses a Slack block from map
func parseSlackBlock(blockMap map[string]interface{}) SlackBlock {
	block := SlackBlock{}

	if blockType, ok := blockMap["type"].(string); ok {
		block.Type = blockType
	}

	if text, ok := blockMap["text"].(interface{}); ok {
		block.Text = text
	}

	return block
}

// getStringField gets a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getBoolField gets a boolean field from a map
func getBoolField(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

// validateSlackMessageParams validates Slack message parameters
func validateSlackMessageParams(params *SlackMessageParams) error {
	// Validate channel
	if params.Channel == "" {
		return fmt.Errorf("channel cannot be empty")
	}

	// Basic channel format validation
	if !isValidSlackChannel(params.Channel) {
		return fmt.Errorf("invalid channel format: %s", params.Channel)
	}

	// Validate text
	if strings.TrimSpace(params.Text) == "" {
		return fmt.Errorf("text cannot be empty")
	}
	if len(params.Text) > 40000 { // Slack's limit
		return fmt.Errorf("text too long (max 40000 characters)")
	}

	// Validate thread_ts if provided
	if params.ThreadTS != "" {
		if !isValidSlackTimestamp(params.ThreadTS) {
			return fmt.Errorf("invalid thread_ts format: %s", params.ThreadTS)
		}
	}

	// Validate attachments
	for i, attachment := range params.Attachments {
		if err := validateSlackAttachment(attachment); err != nil {
			return fmt.Errorf("invalid attachment at index %d: %w", i, err)
		}
	}

	// Validate bot name and icons
	if params.BotName != "" && len(params.BotName) > 80 {
		return fmt.Errorf("bot_name too long (max 80 characters)")
	}

	if params.IconEmoji != "" && !isValidSlackEmoji(params.IconEmoji) {
		return fmt.Errorf("invalid icon_emoji format: %s", params.IconEmoji)
	}

	if params.IconURL != "" && !isValidURL(params.IconURL) {
		return fmt.Errorf("invalid icon_url: %s", params.IconURL)
	}

	return nil
}

// isValidSlackChannel validates a Slack channel
func isValidSlackChannel(channel string) bool {
	channel = strings.TrimSpace(channel)

	// Accept channels starting with # or @, or direct channel IDs
	if strings.HasPrefix(channel, "#") || strings.HasPrefix(channel, "@") {
		return len(channel) > 1
	}

	// Or just a channel ID (alphanumeric with some special chars)
	for _, char := range channel {
		if !isValidChannelChar(char) {
			return false
		}
	}

	return len(channel) > 0
}

// isValidChannelChar checks if a character is valid in a channel name
func isValidChannelChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '_'
}

// isValidSlackTimestamp validates a Slack timestamp
func isValidSlackTimestamp(ts string) bool {
	// Slack timestamps are in format like "1234567890.123456"
	parts := strings.Split(ts, ".")
	if len(parts) != 2 {
		return false
	}

	// Check both parts are numeric
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}

// isValidSlackEmoji validates a Slack emoji format
func isValidSlackEmoji(emoji string) bool {
	// Basic emoji format check: :emoji_name:
	return strings.HasPrefix(emoji, ":") && strings.HasSuffix(emoji, ":") && len(emoji) >= 3
}

// isValidURL validates a URL
func isValidURL(urlStr string) bool {
	return strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://")
}

// validateSlackAttachment validates a Slack attachment
func validateSlackAttachment(attachment SlackAttachment) error {
	if attachment.Fallback == "" {
		return fmt.Errorf("attachment fallback is required")
	}
	if len(attachment.Fallback) > 100 {
		return fmt.Errorf("attachment fallback too long (max 100 characters)")
	}

	if attachment.Title != "" && len(attachment.Title) > 100 {
		return fmt.Errorf("attachment title too long (max 100 characters)")
	}

	if attachment.Text != "" && len(attachment.Text) > 8000 {
		return fmt.Errorf("attachment text too long (max 8000 characters)")
	}

	if len(attachment.Fields) > 20 {
		return fmt.Errorf("too many attachment fields (max 20)")
	}

	return nil
}

// getSlackConfig gets Slack configuration (mock for Phase 2)
func getSlackConfig(ctx context.Context) (*SlackConfig, error) {
	// In production, this would load from secure storage
	// For Phase 2, load from environment variable
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("SLACK_WEBHOOK_URL environment variable not set")
	}

	return &SlackConfig{
		Token:      "xoxb-mock-token-1234567890",
		WebhookURL: webhookURL,
		TeamID:     "T00000000",
	}, nil
}

// sendSlackMessage sends a message to Slack
func sendSlackMessage(ctx context.Context, params *SlackMessageParams, config *SlackConfig) (*SlackMessageResult, error) {
	// For Phase 2, we'll simulate sending to Slack
	// In production, this would make a real API call to Slack

	startTime := time.Now()

	// Generate a mock timestamp
	timestamp := generateSlackTimestamp()

	// Create result
	result := &SlackMessageResult{
		OK:        true,
		Channel:   params.Channel,
		Timestamp: timestamp,
		SentAt:    startTime,
		Provider:  "slack",
		Metadata: map[string]string{
			"api_method":       "chat.postMessage",
			"team_id":          config.TeamID,
			"has_thread":       fmt.Sprintf("%t", params.ThreadTS != ""),
			"has_blocks":       fmt.Sprintf("%t", len(params.Blocks) > 0),
			"attachment_count": fmt.Sprintf("%d", len(params.Attachments)),
		},
	}

	// Add message object if successful
	if result.OK {
		result.Message = SlackMessage{
			Type:        "message",
			Channel:     params.Channel,
			Text:        params.Text,
			Timestamp:   timestamp,
			ThreadTS:    params.ThreadTS,
			BotID:       "B00000000",
			Attachments: params.Attachments,
			Blocks:      params.Blocks,
		}
	}

	// In a real implementation, this would be:
	/*
		// Build API request
		apiURL := "https://slack.com/api/chat.postMessage"

		requestBody := map[string]interface{}{
			"channel": params.Channel,
			"text":    params.Text,
		}

		if params.ThreadTS != "" {
			requestBody["thread_ts"] = params.ThreadTS
		}
		if params.BotName != "" {
			requestBody["username"] = params.BotName
		}
		if params.IconEmoji != "" {
			requestBody["icon_emoji"] = params.IconEmoji
		}
		if params.IconURL != "" {
			requestBody["icon_url"] = params.IconURL
		}
		if len(params.Attachments) > 0 {
			requestBody["attachments"] = params.Attachments
		}
		if len(params.Blocks) > 0 {
			requestBody["blocks"] = params.Blocks
		}

		// Marshal request body
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+config.Token)
		req.Header.Set("User-Agent", "ArmorClaw/1.0")

		// Send request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send Slack request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Parse response
		var slackResp SlackMessageResult
		if err := json.Unmarshal(respBody, &slackResp); err != nil {
			return nil, fmt.Errorf("failed to parse Slack response: %w", err)
		}

		// Check for errors
		if !slackResp.OK {
			return nil, fmt.Errorf("Slack API error: %s", string(respBody))
		}

		// Set metadata
		slackResp.SentAt = time.Now()
		slackResp.Provider = "slack"
		if slackResp.Metadata == nil {
			slackResp.Metadata = make(map[string]string)
		}
		slackResp.Metadata["api_method"] = "chat.postMessage"

		return &slackResp, nil
	*/

	return result, nil
}

// generateSlackTimestamp generates a Slack-style timestamp
func generateSlackTimestamp() string {
	return fmt.Sprintf("%d.%06d", time.Now().Unix(), time.Now().Nanosecond()/1000)
}

// ValidateSlackMessageParams validates Slack message parameters
func ValidateSlackMessageParams(params map[string]interface{}) error {
	// Check required parameters
	if _, exists := params["channel"]; !exists {
		return fmt.Errorf("channel parameter is required")
	}
	if _, exists := params["text"]; !exists {
		return fmt.Errorf("text parameter is required")
	}

	// Validate channel
	if channel, ok := params["channel"].(string); ok {
		if !isValidSlackChannel(channel) {
			return fmt.Errorf("invalid channel: %s", channel)
		}
	} else {
		return fmt.Errorf("channel parameter must be a string")
	}

	// Validate text
	if text, ok := params["text"].(string); ok {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("text cannot be empty")
		}
		if len(text) > 40000 {
			return fmt.Errorf("text too long (max 40000 characters)")
		}
	} else {
		return fmt.Errorf("text parameter must be a string")
	}

	// Validate optional thread_ts
	if threadTS, ok := params["thread_ts"].(string); ok && threadTS != "" {
		if !isValidSlackTimestamp(threadTS) {
			return fmt.Errorf("invalid thread_ts: %s", threadTS)
		}
	}

	// Validate optional bot_name
	if botName, ok := params["bot_name"].(string); ok && botName != "" {
		if len(botName) > 80 {
			return fmt.Errorf("bot_name too long (max 80 characters)")
		}
	}

	// Validate optional icon_emoji
	if iconEmoji, ok := params["icon_emoji"].(string); ok && iconEmoji != "" {
		if !isValidSlackEmoji(iconEmoji) {
			return fmt.Errorf("invalid icon_emoji: %s", iconEmoji)
		}
	}

	// Validate optional icon_url
	if iconURL, ok := params["icon_url"].(string); ok && iconURL != "" {
		if !isValidURL(iconURL) {
			return fmt.Errorf("invalid icon_url: %s", iconURL)
		}
	}

	return nil
}
