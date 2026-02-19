// Package push provides Sygnal integration for Matrix push notifications
package push

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

// SygnalConfig configures the Sygnal push gateway
type SygnalConfig struct {
	// Sygnal server URL
	URL string

	// API key for authentication
	APIKey string

	// Application ID
	AppID string

	// Default pusher URL (where clients send push registrations)
	PusherURL string

	// HTTP client timeout
	Timeout time.Duration

	// Logger
	Logger *slog.Logger
}

// SygnalClient handles communication with Sygnal server
type SygnalClient struct {
	config SygnalConfig
	client *http.Client
	logger *slog.Logger
}

// NewSygnalClient creates a new Sygnal client
func NewSygnalClient(config SygnalConfig) (*SygnalClient, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("Sygnal URL is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "sygnal")
	}

	return &SygnalClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: config.Logger,
	}, nil
}

// PushRequest represents a push notification request to Sygnal
type PushRequest struct {
	Notification MatrixNotification `json:"notification"`
}

// MatrixNotification represents a Matrix push notification
type MatrixNotification struct {
	EventID        string                 `json:"event_id"`
	RoomID         string                 `json:"room_id"`
	Type           string                 `json:"type"`
	Sender         string                 `json:"sender"`
	SenderDisplay  string                 `json:"sender_display_name,omitempty"`
	RoomName       string                 `json:"room_name,omitempty"`
	RoomAlias      string                 `json:"room_alias,omitempty"`
	Prio           string                 `json:"prio,omitempty"`
	Content        map[string]interface{} `json:"content,omitempty"`
	Counts         NotificationCounts     `json:"counts"`
	Devices        []PushDevice           `json:"devices"`
}

// NotificationCounts represents unread counts
type NotificationCounts struct {
	Unread       int `json:"unread"`
	MissedCalls  int `json:"missed_calls,omitempty"`
	Highlight    int `json:"highlight,omitempty"`
}

// PushDevice represents a device to push to
type PushDevice struct {
	AppID     string `json:"app_id"`
	PushKey   string `json:"pushkey"`
	PushKeyTS int64  `json:"pushkey_ts,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Tweaks    map[string]interface{} `json:"tweaks,omitempty"`
}

// PushResponse represents a response from Sygnal
type PushResponse struct {
	Rejected []string `json:"rejected"`
}

// SendPush sends a Matrix push notification through Sygnal
func (s *SygnalClient) SendPush(ctx context.Context, notif *MatrixNotification) (*PushResponse, error) {
	req := PushRequest{
		Notification: *notif,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := s.config.URL + "/_matrix/push/v1/notify"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if s.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Sygnal error (%d): %s", resp.StatusCode, string(respBody))
	}

	var pushResp PushResponse
	if err := json.Unmarshal(respBody, &pushResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	s.logger.Info("push_sent",
		"event_id", notif.EventID,
		"room_id", notif.RoomID,
		"devices", len(notif.Devices),
		"rejected", len(pushResp.Rejected),
	)

	return &pushResp, nil
}

// MatrixPushGateway handles push notifications from Matrix homeserver
type MatrixPushGateway struct {
	gateway *Gateway
	sygnal  *SygnalClient
	logger  *slog.Logger
}

// NewMatrixPushGateway creates a new Matrix push gateway
func NewMatrixPushGateway(gateway *Gateway, sygnal *SygnalClient) *MatrixPushGateway {
	return &MatrixPushGateway{
		gateway: gateway,
		sygnal:  sygnal,
		logger:  slog.Default().With("component", "matrix_push_gateway"),
	}
}

// HandlePushEvent handles a push event from the homeserver
func (g *MatrixPushGateway) HandlePushEvent(ctx context.Context, event MatrixPushEvent) error {
	// Convert to Matrix notification format
	notif := &MatrixNotification{
		EventID:       event.EventID,
		RoomID:        event.RoomID,
		Type:          event.Type,
		Sender:        event.Sender,
		SenderDisplay: event.SenderDisplay,
		RoomName:      event.RoomName,
		Content:       event.Content,
		Counts: NotificationCounts{
			Unread:    event.UnreadCount,
			Highlight: event.HighlightCount,
		},
	}

	// Get devices for all users in the room
	devices := g.getPushDevicesForRoom(event.RoomID, event.Sender)
	if len(devices) == 0 {
		g.logger.Debug("no_devices_for_room", "room_id", event.RoomID)
		return nil
	}

	notif.Devices = devices

	// Send through Sygnal if available
	if g.sygnal != nil {
		_, err := g.sygnal.SendPush(ctx, notif)
		return err
	}

	// Fallback: send directly through gateway
	return g.sendDirectPush(ctx, notif)
}

// getPushDevicesForRoom gets push devices for all users in a room except sender
func (g *MatrixPushGateway) getPushDevicesForRoom(roomID, sender string) []PushDevice {
	// In production, this would query the room membership from Matrix
	// For now, return empty slice - devices would be registered via Matrix pushers
	return []PushDevice{}
}

// sendDirectPush sends push notifications directly through the gateway
func (g *MatrixPushGateway) sendDirectPush(ctx context.Context, notif *MatrixNotification) error {
	for _, device := range notif.Devices {
		// Get device from gateway
		reg, err := g.gateway.GetDevice(device.PushKey)
		if err != nil {
			g.logger.Warn("device_not_found", "push_key", device.PushKey)
			continue
		}

		// Create notification
		pushNotif := &Notification{
			Platform:    reg.Platform,
			DeviceToken: reg.DeviceToken,
			Title:       formatSenderName(notif.Sender),
			Body:        extractMessageBody(notif.Content),
			Priority:    PriorityHigh,
			Data: map[string]interface{}{
				"event_id": notif.EventID,
				"room_id":  notif.RoomID,
				"type":     notif.Type,
			},
		}

		// Apply tweaks
		if device.Tweaks != nil {
			if sound, ok := device.Tweaks["sound"].(string); ok {
				pushNotif.Sound = sound
			}
			if highlight, ok := device.Tweaks["highlight"].(bool); ok && highlight {
				pushNotif.Priority = PriorityHigh
			}
		}

		// Send
		_, err = g.gateway.Send(ctx, pushNotif)
		if err != nil {
			g.logger.Warn("push_failed",
				"device_id", reg.ID,
				"error", err,
			)
		}
	}

	return nil
}

// MatrixPushEvent represents a push event from Matrix
type MatrixPushEvent struct {
	EventID        string                 `json:"event_id"`
	RoomID         string                 `json:"room_id"`
	Type           string                 `json:"type"`
	Sender         string                 `json:"sender"`
	SenderDisplay  string                 `json:"sender_display_name"`
	RoomName       string                 `json:"room_name"`
	Content        map[string]interface{} `json:"content"`
	UnreadCount    int                    `json:"unread_count"`
	HighlightCount int                    `json:"highlight_count"`
}

// PusherRegistration represents a Matrix pusher registration
type PusherRegistration struct {
	PushKey         string                 `json:"pushkey"`
	Kind            string                 `json:"kind"` // "http" or "email"
	AppID           string                 `json:"app_id"`
	AppDisplayName  string                 `json:"app_display_name"`
	DeviceDisplayName string               `json:"device_display_name"`
	ProfileTag      string                 `json:"profile_tag,omitempty"`
	Lang            string                 `json:"lang"`
	Data            map[string]interface{} `json:"data"`
}

// PusherManager manages Matrix pusher registrations
type PusherManager struct {
	pushers map[string][]*PusherRegistration // user_id -> pushers
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewPusherManager creates a new pusher manager
func NewPusherManager() *PusherManager {
	return &PusherManager{
		pushers: make(map[string][]*PusherRegistration),
		logger:  slog.Default().With("component", "pusher_manager"),
	}
}

// RegisterPusher registers a pusher for a user
func (m *PusherManager) RegisterPusher(userID string, pusher *PusherRegistration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate pusher
	if pusher.PushKey == "" {
		return fmt.Errorf("pushkey is required")
	}
	if pusher.AppID == "" {
		return fmt.Errorf("app_id is required")
	}

	// Check for existing pusher with same pushkey
	for _, p := range m.pushers[userID] {
		if p.PushKey == pusher.PushKey && p.AppID == pusher.AppID {
			// Update existing
			*p = *pusher
			m.logger.Info("pusher_updated", "user_id", userID, "app_id", pusher.AppID)
			return nil
		}
	}

	// Add new pusher
	m.pushers[userID] = append(m.pushers[userID], pusher)
	m.logger.Info("pusher_registered", "user_id", userID, "app_id", pusher.AppID)

	return nil
}

// UnregisterPusher removes a pusher
func (m *PusherManager) UnregisterPusher(userID, pushKey, appID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pushers := m.pushers[userID]
	for i, p := range pushers {
		if p.PushKey == pushKey && (appID == "" || p.AppID == appID) {
			m.pushers[userID] = append(pushers[:i], pushers[i+1:]...)
			m.logger.Info("pusher_unregistered", "user_id", userID, "app_id", appID)
			return nil
		}
	}

	return fmt.Errorf("pusher not found")
}

// GetPushers returns all pushers for a user
func (m *PusherManager) GetPushers(userID string) []*PusherRegistration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pushers := m.pushers[userID]
	result := make([]*PusherRegistration, len(pushers))
	copy(result, pushers)
	return result
}

// extractMessageBody extracts the message body from Matrix content
func extractMessageBody(content map[string]interface{}) string {
	msgtype, hasType := content["msgtype"].(string)

	// Check body first
	if body, ok := content["body"].(string); ok {
		if hasType && msgtype == "m.emote" {
			return "*" + body
		}
		return body
	}

	if hasType {
		switch msgtype {
		case "m.image":
			return "📷 Image"
		case "m.video":
			return "🎬 Video"
		case "m.audio":
			return "🎵 Audio"
		case "m.file":
			return "📎 File"
		}
	}
	return "New message"
}
