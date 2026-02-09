// Package adapter provides Matrix client functionality for ArmorClaw bridge.
// This enables agents to communicate via Matrix while keeping credentials isolated.
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

// MatrixAdapter implements Matrix client protocol
type MatrixAdapter struct {
	homeserverURL   string
	userID           string
	accessToken     string
	deviceID         string
	syncToken        string
	trustedSenders   []string
	trustedRooms     []string
	rejectUntrusted  bool
	httpClient       *http.Client
	eventQueue       chan *MatrixEvent
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	piiScrubber      *pii.Scrubber
}

// MatrixEvent represents a Matrix event
type MatrixEvent struct {
	Type    string                 `json:"type"`
	RoomID  string                 `json:"room_id"`
	Sender  string                 `json:"sender"`
	Content map[string]interface{} `json:"content"`
	Origin  string                 `json:"origin_server"`
	EventID string                 `json:"event_id"`
}

// MatrixMessage is a simplified message structure
type MatrixMessage struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
	MsgType string `json:"msgtype"`
}

// SyncResponse represents Matrix sync response
type SyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]struct {
			Timeline struct {
				Events []json.RawMessage `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

// MatrixError represents a Matrix API error response
type MatrixError struct {
	ErrCode string `json:"errcode"`
	Error   string `json:"error"`
}

// Config holds Matrix adapter configuration
type Config struct {
	HomeserverURL string
	Username      string
	Password      string
	DeviceID      string
	TokenFile     string

	// Zero-trust configuration
	TrustedSenders []string // Allowed Matrix user IDs (@user:domain.com, *@trusted.domain, *:domain)
	TrustedRooms   []string // Allowed room IDs (!roomid:domain.com)
	RejectUntrusted bool   // If true, return error to sender; if false, drop silently
}

// New creates a new Matrix adapter
func New(cfg Config) (*MatrixAdapter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &MatrixAdapter{
		homeserverURL:   cfg.HomeserverURL,
		deviceID:        cfg.DeviceID,
		trustedSenders:  cfg.TrustedSenders,
		trustedRooms:    cfg.TrustedRooms,
		rejectUntrusted: cfg.RejectUntrusted,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		eventQueue:  make(chan *MatrixEvent, 100),
		ctx:         ctx,
		cancel:      cancel,
		piiScrubber: pii.New(),
	}, nil
}

// Login authenticates with the Matrix homeserver
func (m *MatrixAdapter) Login(username, password string) error {
	payload := map[string]interface{}{
		"type":     "m.login.password",
		"user":     username,
		"password": password,
		"device_id": m.deviceID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		m.ctx,
		"POST",
		m.homeserverURL+"/_matrix/client/v3/login",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		DeviceID    string `json:"device_id"`
		UserID      string `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	m.mu.Lock()
	m.accessToken = result.AccessToken
	m.userID = result.UserID
	m.mu.Unlock()

	return nil
}

// SendMessage sends a message to a Matrix room
func (m *MatrixAdapter) SendMessage(roomID, message, msgType string) (string, error) {
	m.mu.RLock()
	token := m.accessToken
	m.mu.RUnlock()

	if token == "" {
		return "", fmt.Errorf("not logged in")
	}

	// Scrub PII from outgoing message
	scrubbedMessage, redactions := m.piiScrubber.Scrub(message)
	if len(redactions) > 0 {
		logger.Global().Info("PII scrubbed from outgoing message",
			"room_id", roomID,
			"redaction_count", len(redactions),
		)
	}

	payload := map[string]interface{}{
		"msgtype": msgType,
		"body":    scrubbedMessage,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Generate transaction ID
	txnID := fmt.Sprintf("m%d", time.Now().UnixNano())

	u, err := url.Parse(m.homeserverURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse homeserver URL: %w", err)
	}

	u.Path = fmt.Sprintf("/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		url.PathEscape(roomID), txnID)

	req, err := http.NewRequestWithContext(
		m.ctx,
		"PUT",
		u.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("send failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode send response: %w", err)
	}

	return result.EventID, nil
}

// Sync performs a long-poll sync with the homeserver
func (m *MatrixAdapter) Sync(timeout int) error {
	m.mu.RLock()
	token := m.accessToken
	syncToken := m.syncToken
	m.mu.RUnlock()

	if token == "" {
		return fmt.Errorf("not logged in")
	}

	u, err := url.Parse(m.homeserverURL)
	if err != nil {
		return fmt.Errorf("failed to parse homeserver URL: %w", err)
	}

	u.Path = "/_matrix/client/v3/sync"
	query := url.Values{}
	query.Set("timeout", fmt.Sprintf("%d", timeout*1000))
	if syncToken != "" {
		query.Set("since", syncToken)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		m.ctx,
		"GET",
		u.String(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	// Process events and queue them
	m.processEvents(&syncResp)

	// Update sync token
	m.mu.Lock()
	m.syncToken = syncResp.NextBatch
	m.mu.Unlock()

	return nil
}

// processEvents processes events from sync response
func (m *MatrixAdapter) processEvents(syncResp *SyncResponse) {
	for roomID, room := range syncResp.Rooms.Join {
		for _, rawEvent := range room.Timeline.Events {
			var event MatrixEvent
			if err := json.Unmarshal(rawEvent, &event); err != nil {
				continue
			}

			// Only queue message events
			if event.Type == "m.room.message" {
				// Validate sender (zero-trust)
				if !m.isTrustedSender(event.Sender) {
					// Log security event
					secLog := logger.NewSecurityLogger(logger.Global().WithComponent("matrix"))
					secLog.LogAuthRejected(context.Background(), event.Sender, "sender_not_in_allowlist",
						logger.LogAttr("room_id", roomID),
					)

					// Optionally send rejection back to sender
					if m.rejectUntrusted {
						m.sendRejectionMessage(roomID, event.Sender)
					}
					continue
				}

				// Validate room
				if !m.isTrustedRoom(roomID) {
					secLog := logger.NewSecurityLogger(logger.Global().WithComponent("matrix"))
					secLog.LogAccessDenied(context.Background(), "matrix_message", event.Sender, "room_not_in_allowlist",
						logger.LogAttr("room_id", roomID),
						logger.LogAttr("event_id", event.EventID),
					)
					continue
				}

				// Scrub PII from message content
				event.Content = m.piiScrubber.ScrubMap(event.Content)

				// Log redaction summary if any PII was found
				if body, ok := event.Content["body"].(string); ok {
					scrubbed, redactions := m.piiScrubber.Scrub(body)
					if len(redactions) > 0 {
						event.Content["body"] = scrubbed
						logger.Global().Info("PII scrubbed from message",
							"sender", event.Sender,
							"room_id", roomID,
							"redaction_count", len(redactions),
						)
					}
				}

				event.RoomID = roomID

				select {
				case m.eventQueue <- &event:
				default:
					// Queue full, drop event
				}
			}
		}
	}
}

// sendRejectionMessage sends a rejection message back to the sender
func (m *MatrixAdapter) sendRejectionMessage(roomID, sender string) {
	// Send rejection notice
	rejectionMsg := "🔒 **Message Rejected**\n\nYour message was not processed because you are not on the trusted senders list for this ArmorClaw instance."
	m.SendMessageWithRetry(roomID, rejectionMsg, "m.notice")
}

// ReceiveEvents returns the event channel for receiving messages
func (m *MatrixAdapter) ReceiveEvents() <-chan *MatrixEvent {
	return m.eventQueue
}

// StartSync begins the background sync loop
func (m *MatrixAdapter) StartSync() {
	go m.syncLoop()
}

// syncLoop runs the continuous sync loop
func (m *MatrixAdapter) syncLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.Sync(30); err != nil {
				// Log error but continue
			}
		}
	}
}

// GetUserID returns the current user ID
func (m *MatrixAdapter) GetUserID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userID
}

// GetAccessToken returns the current access token
func (m *MatrixAdapter) GetAccessToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.accessToken
}

// isTrustedSender checks if a sender is in the trusted list
func (m *MatrixAdapter) isTrustedSender(sender string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no allowlist configured, allow all (backward compatible)
	if len(m.trustedSenders) == 0 {
		return true
	}

	// Check against allowlist
	for _, pattern := range m.trustedSenders {
		if m.matchSenderPattern(sender, pattern) {
			return true
		}
	}
	return false
}

// matchSenderPattern supports wildcards for sender matching
func (m *MatrixAdapter) matchSenderPattern(sender, pattern string) bool {
	// Exact match
	if sender == pattern {
		return true
	}

	// Wildcard: *@domain.com - any user from domain
	if len(pattern) > 1 && pattern[0] == '*' && pattern[1] == '@' {
		domain := pattern[2:]
		if len(sender) > len(domain)+1 {
			atIndex := len(sender) - len(domain)
			if sender[atIndex:] == domain && sender[atIndex-1] == '@' {
				return true
			}
		}
	}

	// Wildcard: *:domain.com - any user from domain (alternative format)
	if len(pattern) > 1 && pattern[0] == '*' && pattern[1] == ':' {
		domain := pattern[2:]
		if len(sender) > len(domain)+1 {
			atIndex := len(sender) - len(domain)
			if sender[atIndex:] == domain && sender[atIndex-1] == ':' {
				return true
			}
		}
	}

	return false
}

// isTrustedRoom checks if a room is in the trusted list
func (m *MatrixAdapter) isTrustedRoom(roomID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no allowlist configured, allow all rooms
	if len(m.trustedRooms) == 0 {
		return true
	}

	for _, trusted := range m.trustedRooms {
		if roomID == trusted {
			return true
		}
	}
	return false
}

// SetTrustedSenders sets the list of trusted senders
func (m *MatrixAdapter) SetTrustedSenders(senders []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trustedSenders = senders
}

// SetTrustedRooms sets the list of trusted rooms
func (m *MatrixAdapter) SetTrustedRooms(rooms []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trustedRooms = rooms
}

// GetTrustedSenders returns the list of trusted senders
func (m *MatrixAdapter) GetTrustedSenders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.trustedSenders...)
}

// GetTrustedRooms returns the list of trusted rooms
func (m *MatrixAdapter) GetTrustedRooms() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.trustedRooms...)
}

// Close closes the adapter and stops sync
func (m *MatrixAdapter) Close() error {
	m.cancel()
	return nil
}

// isRetryableHTTPError checks if an HTTP error is retryable
func isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context-related errors
	if containsAny(err.Error(), "context deadline exceeded", "context canceled") {
		return true
	}

	// Check for network-related errors
	if containsAny(err.Error(), "connection refused", "connection reset",
		"connection timed out", "temporary failure", "network is unreachable") {
		return true
	}

	return false
}

// isRetryableStatusCode checks if an HTTP status code is retryable
func isRetryableStatusCode(statusCode int) bool {
	// 5xx server errors are typically retryable
	return statusCode >= 500 && statusCode < 600
}

// SendMessageWithRetry sends a message with retry logic for transient failures
func (m *MatrixAdapter) SendMessageWithRetry(roomID, message, msgType string) (string, error) {
	const maxAttempts = 3
	const baseDelay = 1 * time.Second

	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		eventID, err := m.SendMessage(roomID, message, msgType)
		if err == nil {
			return eventID, nil
		}

		// Check if error is retryable
		if !isRetryableHTTPError(err) {
			return "", err // Non-retryable error
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < maxAttempts-1 {
			backoff := baseDelay * time.Duration(1<<uint(attempt))

			select {
			case <-time.After(backoff):
			case <-m.ctx.Done():
				return "", m.ctx.Err()
			}
		}
	}

	return "", fmt.Errorf("send message failed after %d attempts: %w", maxAttempts, lastErr)
}

// SyncWithRetry performs sync with retry logic
func (m *MatrixAdapter) SyncWithRetry(timeout int) error {
	const maxAttempts = 3
	const baseDelay = 1 * time.Second

	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := m.Sync(timeout)
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !isRetryableHTTPError(err) {
			return err // Non-retryable error
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < maxAttempts-1 {
			backoff := baseDelay * time.Duration(1<<uint(attempt))

			select {
			case <-time.After(backoff):
			case <-m.ctx.Done():
				return m.ctx.Err()
			}
		}
	}

	return fmt.Errorf("sync failed after %d attempts: %w", maxAttempts, lastErr)
}

// isTokenExpired checks if the access token might be expired
// Matrix tokens typically expire after inactivity periods
func (m *MatrixAdapter) isTokenExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If we have a token but no sync token, we haven't successfully synced
	return m.accessToken != "" && m.syncToken == ""
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) && findSubstring(s, sub) >= 0 {
			return true
		}
	}
	return false
}

// findSubstring finds the index of a substring
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
