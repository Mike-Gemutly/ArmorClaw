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

	errsys "github.com/armorclaw/bridge/pkg/errors"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

// matrixTracker tracks component events for the Matrix adapter
var matrixTracker = errsys.GetComponentTracker("matrix")

// EventPublisher interface for publishing events to external systems
type EventPublisher interface {
	Publish(event *MatrixEvent) error
}

// MatrixAdapter implements Matrix client protocol
type MatrixAdapter struct {
	homeserverURL   string
	userID           string
	accessToken     string
	refreshToken     string           // P1-HIGH-1: Refresh token for long-lived sessions
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
	eventPublisher   EventPublisher // Event bus for real-time event publishing
	lastExpiryCheck  time.Time          // P1-HIGH-1: Track last token expiry check
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
	RefreshToken    string   // P1-HIGH-1: Encrypted refresh token from login
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
		refreshToken:    cfg.RefreshToken,    // P1-HIGH-1: Initialize refresh token
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
	matrixTracker.Event("login", map[string]any{"user": username})

	payload := map[string]interface{}{
		"type":     "m.login.password",
		"user":     username,
		"password": password,
		"device_id": m.deviceID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("failed to marshal login request: %w", err)).
			WithFunction("Login").
			WithInputs(map[string]any{"user": username}).
			Build()
		matrixTracker.Failure("login", err, map[string]any{"reason": "marshal_failed"})
		return err
	}

	req, err := http.NewRequestWithContext(
		m.ctx,
		"POST",
		m.homeserverURL+"/_matrix/client/v3/login",
		bytes.NewReader(body),
	)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("failed to create request: %w", err)).
			WithFunction("Login").
			WithInputs(map[string]any{"user": username, "homeserver": m.homeserverURL}).
			Build()
		matrixTracker.Failure("login", err, map[string]any{"reason": "request_create_failed"})
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("login request failed: %w", err)).
			WithFunction("Login").
			WithInputs(map[string]any{"user": username, "homeserver": m.homeserverURL}).
			Build()
		matrixTracker.Failure("login", err, map[string]any{"reason": "request_failed"})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("login failed: status %d, response: %s", resp.StatusCode, string(body))).
			WithFunction("Login").
			WithInputs(map[string]any{"user": username, "status": resp.StatusCode}).
			Build()
		matrixTracker.Failure("login", err, map[string]any{"reason": "auth_failed", "status": resp.StatusCode})
		return err
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		DeviceID    string `json:"device_id"`
		UserID      string `json:"user_id"`
		RefreshToken string `json:"refresh_token"` // P1-HIGH-1: Capture refresh token
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("failed to decode login response: %w", err)).
			WithFunction("Login").
			WithInputs(map[string]any{"user": username}).
			Build()
		matrixTracker.Failure("login", err, map[string]any{"reason": "decode_failed"})
		return err
	}

	m.mu.Lock()
	m.accessToken = result.AccessToken
	m.userID = result.UserID
	m.refreshToken = result.RefreshToken // P1-HIGH-1: Store refresh token for long-lived sessions
	m.mu.Unlock()

	matrixTracker.Success("login", map[string]any{"user_id": result.UserID})
	return nil
}

// SendMessage sends a message to a Matrix room
func (m *MatrixAdapter) SendMessage(roomID, message, msgType string) (string, error) {
	matrixTracker.Event("send_message", map[string]any{"room_id": roomID, "msg_type": msgType})

	// P1-HIGH-1: Ensure token is valid before sending
	if err := m.ensureValidToken(); err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("token validation failed: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "token_validation_failed"})
		return "", err
	}

	m.mu.RLock()
	token := m.accessToken
	m.mu.RUnlock()

	if token == "" {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("not logged in")).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "not_authenticated"})
		return "", err
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
		err := errsys.NewBuilder("MAT-021").
			Wrap(fmt.Errorf("failed to marshal message: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "marshal_failed"})
		return "", err
	}

	// Generate transaction ID
	txnID := fmt.Sprintf("m%d", time.Now().UnixNano())

	u, err := url.Parse(m.homeserverURL)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("failed to parse homeserver URL: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID, "homeserver": m.homeserverURL}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "url_parse_failed"})
		return "", err
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
		err := errsys.NewBuilder("MAT-021").
			Wrap(fmt.Errorf("failed to create request: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "request_create_failed"})
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		err := errsys.NewBuilder("MAT-021").
			Wrap(fmt.Errorf("send request failed: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "request_failed"})
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := errsys.NewBuilder("MAT-021").
			Wrap(fmt.Errorf("send failed: status %d, response: %s", resp.StatusCode, string(body))).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID, "status": resp.StatusCode}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "send_failed", "status": resp.StatusCode})
		return "", err
	}

	var result struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err := errsys.NewBuilder("MAT-021").
			Wrap(fmt.Errorf("failed to decode send response: %w", err)).
			WithFunction("SendMessage").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("send_message", err, map[string]any{"reason": "decode_failed"})
		return "", err
	}

	matrixTracker.Success("send_message", map[string]any{"room_id": roomID, "event_id": result.EventID})
	return result.EventID, nil
}

// Sync performs a long-poll sync with the homeserver
func (m *MatrixAdapter) Sync(timeout int) error {
	matrixTracker.Event("sync", map[string]any{"timeout": timeout})

	// P1-HIGH-1: Ensure token is valid before syncing
	if err := m.ensureValidToken(); err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("token validation failed: %w", err)).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "token_validation_failed"})
		return err
	}

	m.mu.RLock()
	token := m.accessToken
	syncToken := m.syncToken
	m.mu.RUnlock()

	if token == "" {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("not logged in")).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "not_authenticated"})
		return err
	}

	u, err := url.Parse(m.homeserverURL)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("failed to parse homeserver URL: %w", err)).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout, "homeserver": m.homeserverURL}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "url_parse_failed"})
		return err
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
		err := errsys.NewBuilder("MAT-003").
			Wrap(fmt.Errorf("failed to create request: %w", err)).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "request_create_failed"})
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		err := errsys.NewBuilder("MAT-003").
			Wrap(fmt.Errorf("sync request failed: %w", err)).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "request_failed"})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := errsys.NewBuilder("MAT-003").
			Wrap(fmt.Errorf("sync failed: status %d, response: %s", resp.StatusCode, string(body))).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout, "status": resp.StatusCode}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "sync_failed", "status": resp.StatusCode})
		return err
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		err := errsys.NewBuilder("MAT-003").
			Wrap(fmt.Errorf("failed to decode sync response: %w", err)).
			WithFunction("Sync").
			WithInputs(map[string]any{"timeout": timeout}).
			Build()
		matrixTracker.Failure("sync", err, map[string]any{"reason": "decode_failed"})
		return err
	}

	// Process events and queue them
	m.processEvents(&syncResp)

	// Update sync token
	m.mu.Lock()
	m.syncToken = syncResp.NextBatch
	m.lastExpiryCheck = time.Now() // Update expiry check on successful sync
	m.mu.Unlock()

	matrixTracker.Success("sync", map[string]any{"next_batch": syncResp.NextBatch})
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

				// Publish to event bus if configured (real-time push)
				m.mu.RLock()
				publisher := m.eventPublisher
				m.mu.RUnlock()

				if publisher != nil {
					// Publish event asynchronously to avoid blocking sync
					go func(e *MatrixEvent) {
						if err := publisher.Publish(e); err != nil {
							// Log but don't block on publish failures
							logger.Global().Warn("Failed to publish event to event bus",
								"error", err,
								"event_id", e.EventID,
								"room_id", e.RoomID,
								"sender", e.Sender,
							)
						}
					}(&event)
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

// SetEventPublisher sets the event publisher for real-time event distribution
// This allows the Matrix adapter to publish events to an event bus
func (m *MatrixAdapter) SetEventPublisher(publisher EventPublisher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventPublisher = publisher
}

// GetRoomMembers retrieves the list of members in a Matrix room
// This implements the errsys.MatrixAdminAdapter interface
func (m *MatrixAdapter) GetRoomMembers(ctx context.Context, roomID string) ([]errsys.RoomMember, error) {
	matrixTracker.Event("get_room_members", map[string]any{"room_id": roomID})

	// P1-HIGH-1: Ensure token is valid before API call
	if err := m.ensureValidToken(); err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("token validation failed: %w", err)).
			WithFunction("GetRoomMembers").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("get_room_members", err, map[string]any{"reason": "token_validation_failed"})
		return nil, err
	}

	m.mu.RLock()
	accessToken := m.accessToken
	m.mu.RUnlock()

	// Build URL for room members API
	url := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/members", m.homeserverURL, roomID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		err := errsys.NewBuilder("MAT-020").
			Wrap(fmt.Errorf("failed to create request: %w", err)).
			WithFunction("GetRoomMembers").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("get_room_members", err, map[string]any{"reason": "request_create_failed"})
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("failed to get room members: %w", err)).
			WithFunction("GetRoomMembers").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("get_room_members", err, map[string]any{"reason": "request_failed"})
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := errsys.NewBuilder("MAT-020").
			Wrap(fmt.Errorf("failed to get room members: %s - %s", resp.Status, string(body))).
			WithFunction("GetRoomMembers").
			WithInputs(map[string]any{"room_id": roomID, "status": resp.StatusCode}).
			Build()
		matrixTracker.Failure("get_room_members", err, map[string]any{"reason": "api_failed", "status": resp.StatusCode})
		return nil, err
	}

	// Parse response
	var membersResp struct {
		Chunk []struct {
			Content struct {
				Membership  string `json:"membership"`
				Displayname string `json:"displayname"`
			} `json:"content"`
			Sender   string `json:"sender"`
			StateKey string `json:"state_key"`
		} `json:"chunk"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&membersResp); err != nil {
		err := errsys.NewBuilder("MAT-020").
			Wrap(fmt.Errorf("failed to parse room members response: %w", err)).
			WithFunction("GetRoomMembers").
			WithInputs(map[string]any{"room_id": roomID}).
			Build()
		matrixTracker.Failure("get_room_members", err, map[string]any{"reason": "decode_failed"})
		return nil, err
	}

	// Convert to errsys.RoomMember slice
	var members []errsys.RoomMember
	for _, event := range membersResp.Chunk {
		if event.Content.Membership == "join" {
			members = append(members, errsys.RoomMember{
				UserID:      event.StateKey,
				PowerLevel:  0, // Default power level, would need separate state event for actual value
				Display:     event.Content.Displayname,
			})
		}
	}

	matrixTracker.Success("get_room_members", map[string]any{"room_id": roomID, "member_count": len(members)})
	return members, nil
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

	matrixTracker.Event("send_message_retry", map[string]any{"room_id": roomID, "max_attempts": maxAttempts})

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
				err := errsys.NewBuilder("MAT-021").
					Wrap(m.ctx.Err()).
					WithFunction("SendMessageWithRetry").
					WithInputs(map[string]any{"room_id": roomID, "attempt": attempt}).
					Build()
				matrixTracker.Failure("send_message_retry", err, map[string]any{"reason": "context_canceled"})
				return "", err
			}
		}
	}

	err := errsys.NewBuilder("MAT-021").
		Wrap(fmt.Errorf("send message failed after %d attempts: %w", maxAttempts, lastErr)).
		WithFunction("SendMessageWithRetry").
		WithInputs(map[string]any{"room_id": roomID, "attempts": maxAttempts}).
		Build()
	matrixTracker.Failure("send_message_retry", err, map[string]any{"reason": "max_retries_exceeded"})
	return "", err
}

// SyncWithRetry performs sync with retry logic
func (m *MatrixAdapter) SyncWithRetry(timeout int) error {
	const maxAttempts = 3
	const baseDelay = 1 * time.Second

	matrixTracker.Event("sync_retry", map[string]any{"timeout": timeout, "max_attempts": maxAttempts})

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
				err := errsys.NewBuilder("MAT-003").
					Wrap(m.ctx.Err()).
					WithFunction("SyncWithRetry").
					WithInputs(map[string]any{"timeout": timeout, "attempt": attempt}).
					Build()
				matrixTracker.Failure("sync_retry", err, map[string]any{"reason": "context_canceled"})
				return err
			}
		}
	}

	err := errsys.NewBuilder("MAT-003").
		Wrap(fmt.Errorf("sync failed after %d attempts: %w", maxAttempts, lastErr)).
		WithFunction("SyncWithRetry").
		WithInputs(map[string]any{"timeout": timeout, "attempts": maxAttempts}).
		Build()
	matrixTracker.Failure("sync_retry", err, map[string]any{"reason": "max_retries_exceeded"})
	return err
}

// isTokenExpired checks if the access token might be expired
// P1-HIGH-1: Tracks token expiry using lastExpiryCheck timestamp
// Matrix access tokens expire after 7 days of inactivity
func (m *MatrixAdapter) isTokenExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// No token at all means not logged in
	if m.accessToken == "" {
		return true
	}

	// P1-HIGH-1: Check if we have a refresh token to support long-lived sessions
	if m.refreshToken != "" {
		// With refresh token support, check if 7 days have passed since last check
		if m.lastExpiryCheck.IsZero() {
			// First check - if we have a sync token, we've successfully synced
			return m.syncToken == ""
		}
		// Check if 7 days have passed since last expiry check
		expiryThreshold := 7 * 24 * time.Hour
		return time.Since(m.lastExpiryCheck) > expiryThreshold
	}

	// Without refresh token: if we have a token but no sync token, we haven't successfully synced
	return m.syncToken == ""
}

// P1-HIGH-1: RefreshAccessToken uses refresh_token to get a new access token
// This enables long-lived sessions without requiring re-login with credentials
func (m *MatrixAdapter) RefreshAccessToken() error {
	matrixTracker.Event("refresh_token", nil)

	m.mu.RLock()
	refreshToken := m.refreshToken
	m.mu.RUnlock()

	if refreshToken == "" {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("no refresh token available")).
			WithFunction("RefreshAccessToken").
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "no_refresh_token"})
		return err
	}

	// Prepare refresh request according to Matrix spec
	// https://spec.matrix.org/v1.4/client-server-api/#post_matrixclientv3refresh
	payload := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("failed to marshal refresh request: %w", err)).
			WithFunction("RefreshAccessToken").
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "marshal_failed"})
		return err
	}

	req, err := http.NewRequestWithContext(
		m.ctx,
		"POST",
		m.homeserverURL+"/_matrix/client/v3/refresh",
		bytes.NewReader(body),
	)
	if err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("failed to create refresh request: %w", err)).
			WithFunction("RefreshAccessToken").
			WithInputs(map[string]any{"homeserver": m.homeserverURL}).
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "request_create_failed"})
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		err := errsys.NewBuilder("MAT-001").
			Wrap(fmt.Errorf("refresh request failed: %w", err)).
			WithFunction("RefreshAccessToken").
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "request_failed"})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("token refresh failed: status %d, response: %s", resp.StatusCode, string(body))).
			WithFunction("RefreshAccessToken").
			WithInputs(map[string]any{"status": resp.StatusCode}).
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "refresh_failed", "status": resp.StatusCode})
		return err
	}

	// Parse refresh response
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"` // May be rotated
		ExpiresIn    int64  `json:"expires_in_ms"`    // Token lifetime in milliseconds
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		err := errsys.NewBuilder("MAT-002").
			Wrap(fmt.Errorf("failed to decode refresh response: %w", err)).
			WithFunction("RefreshAccessToken").
			Build()
		matrixTracker.Failure("refresh_token", err, map[string]any{"reason": "decode_failed"})
		return err
	}

	// Update tokens and reset expiry check timer
	m.mu.Lock()
	m.accessToken = result.AccessToken
	if result.RefreshToken != "" {
		m.refreshToken = result.RefreshToken // Update if rotated
	}
	m.lastExpiryCheck = time.Now() // Reset expiry check timer
	m.mu.Unlock()

	logger.Global().Info("Matrix access token refreshed via refresh_token",
		"expires_in_ms", result.ExpiresIn,
	)

	matrixTracker.Success("refresh_token", map[string]any{"expires_in_ms": result.ExpiresIn})
	return nil
}

// ensureValidToken checks and refreshes access token if needed
// P1-HIGH-1: Call this before API operations to ensure valid credentials
func (m *MatrixAdapter) ensureValidToken() error {
	if m.isTokenExpired() {
		// Try to refresh using refresh token if available
		m.mu.RLock()
		hasRefreshToken := m.refreshToken != ""
		m.mu.RUnlock()

		if hasRefreshToken {
			if err := m.RefreshAccessToken(); err != nil {
				return errsys.NewBuilder("MAT-002").
					Wrap(fmt.Errorf("failed to refresh expired token: %w", err)).
					WithFunction("ensureValidToken").
					Build()
			}
		} else {
			return errsys.NewBuilder("MAT-002").
				Wrap(fmt.Errorf("access token expired and no refresh token available")).
				WithFunction("ensureValidToken").
				Build()
		}
	}
	return nil
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
