// Package eventbus provides event push mechanisms for real-time Matrix events
// This enables containers to receive Matrix events in real-time without polling
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/websocket"
	"log/slog"
)

// EventBus manages real-time event distribution to subscribers
type EventBus struct {
	subscribers     map[string]*Subscriber
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	websocketServer *websocket.Server
	securityLog     *logger.SecurityLogger
}

// Subscriber represents a client subscribed to receive events
type Subscriber struct {
	ID            string
	Filter        EventFilter
	EventChannel  chan *MatrixEventWrapper
	SubscribeTime  time.Time
	LastActivity  time.Time
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// EventFilter defines which events a subscriber wants to receive
type EventFilter struct {
	RoomID    string   // Only events from this room (empty = all rooms)
	SenderID  string   // Only events from this sender (empty = all senders)
	EventType []string // Only these event types (empty = all types)
}

// MatrixEventWrapper wraps a Matrix event for delivery
type MatrixEventWrapper struct {
	Event    *MatrixEvent `json:"event"`
	Received time.Time   `json:"received"`
	Sequence int64      `json:"sequence"`
}

// MatrixEvent represents a Matrix event (simplified)
type MatrixEvent struct {
	Type    string                 `json:"type"`
	RoomID  string                 `json:"room_id"`
	Sender  string                 `json:"sender"`
	Content map[string]interface{} `json:"content"`
	EventID string                 `json:"event_id"`
}

// Config holds event bus configuration
type Config struct {
	WebSocketEnabled  bool          // Enable WebSocket server
	WebSocketAddr     string        // WebSocket listen address
	WebSocketPath     string        // WebSocket path
	MaxSubscribers     int           // Maximum concurrent subscribers
	InactivityTimeout  time.Duration // Disconnect inactive subscribers
}

// DefaultConfig returns default event bus configuration
func DefaultConfig() Config {
	return Config{
		WebSocketEnabled:  false,
		WebSocketAddr:     "0.0.0.0:8444",
		WebSocketPath:     "/events",
		MaxSubscribers:     100,
		InactivityTimeout:  30 * time.Minute,
	}
}

// NewEventBus creates a new event bus
func NewEventBus(config Config) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		subscribers: make(map[string]*Subscriber),
		ctx:         ctx,
		cancel:      cancel,
		securityLog: logger.NewSecurityLogger(logger.Global().WithComponent("eventbus")),
	}

	// Initialize WebSocket server if enabled
	if config.WebSocketEnabled {
		wsConfig := websocket.Config{
			Addr:                config.WebSocketAddr,
			Path:                config.WebSocketPath,
			MaxConnections:     config.MaxSubscribers,
			InactivityTimeout:   config.InactivityTimeout,
			MessageHandler:      bus.handleWebSocketMessage,
			ConnectHandler:      bus.handleWebSocketConnect,
			DisconnectHandler:   bus.handleWebSocketDisconnect,
		}

		wsServer := websocket.NewServer(wsConfig)
		bus.websocketServer = wsServer
	}

	return bus
}

// Start starts the event bus
func (b *EventBus) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Start WebSocket server if configured
	if b.websocketServer != nil {
		if err := b.websocketServer.Start(); err != nil {
			return fmt.Errorf("failed to start WebSocket server: %w", err)
		}
		b.securityLog.LogSecurityEvent("eventbus_started",
			slog.Bool("websocket_enabled", true),
			slog.String("addr", b.websocketServer.Addr()))
	} else {
		b.securityLog.LogSecurityEvent("eventbus_started",
			slog.Bool("websocket_enabled", false))
	}

	// Start cleanup goroutine
	go b.cleanupInactiveSubscribers()

	return nil
}

// Stop stops the event bus
func (b *EventBus) Stop() {
	b.cancel()

	if b.websocketServer != nil {
		b.websocketServer.Stop()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all subscriber channels
	for id, sub := range b.subscribers {
		sub.cancel()
		close(sub.EventChannel)
	}

	b.securityLog.LogSecurityEvent("eventbus_stopped")
}

// Publish publishes a Matrix event to all matching subscribers
func (b *EventBus) Publish(event *MatrixEvent) error {
	if event == nil {
		return fmt.Errorf("cannot publish nil event")
	}

	wrapper := &MatrixEventWrapper{
		Event:     event,
		Received:  time.Now(),
		Sequence:  time.Now().UnixNano(),
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	publishedCount := 0
	for id, sub := range b.subscribers {
		if b.matchesFilter(event, sub.Filter) {
			select {
			case sub.EventChannel <- wrapper:
				publishedCount++
				sub.mu.Lock()
				sub.LastActivity = time.Now()
				sub.mu.Unlock()
			default:
				// Channel full, subscriber slow - skip
				b.securityLog.LogSecurityEvent("event_skipped",
					slog.String("subscriber_id", id),
					slog.String("reason", "channel_full"))
			}
		}
	}

	b.securityLog.LogSecurityEvent("event_published",
		slog.String("event_type", event.Type),
		slog.String("room_id", event.RoomID),
		slog.Int("subscribers_notified", publishedCount))

	return nil
}

// Subscribe creates a new subscription for receiving events
func (b *EventBus) Subscribe(filter EventFilter) (*Subscriber, error) {
	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	ctx, cancel := context.WithCancel(b.ctx)

	sub := &Subscriber{
		ID:           subID,
		Filter:       filter,
		EventChannel: make(chan *MatrixEventWrapper, 100),
		SubscribeTime: time.Now(),
		LastActivity:  time.Now(),
		ctx:          ctx,
		cancel:       cancel,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[subID] = sub

	b.securityLog.LogSecurityEvent("subscriber_created",
		slog.String("subscriber_id", subID),
		slog.String("room_filter", filter.RoomID),
		slog.String("sender_filter", filter.SenderID))

	return sub, nil
}

// Unsubscribe removes a subscription
func (b *EventBus) Unsubscribe(subscriberID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subscribers[subscriberID]
	if !exists {
		return fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	sub.cancel()
	close(sub.EventChannel)

	delete(b.subscribers, subscriberID)

	b.securityLog.LogSecurityEvent("subscriber_removed",
		slog.String("subscriber_id", subscriberID))

	return nil
}

// matchesFilter checks if an event matches a subscriber's filter
func (b *EventBus) matchesFilter(event *MatrixEvent, filter EventFilter) bool {
	// Check room filter
	if filter.RoomID != "" && event.RoomID != filter.RoomID {
		return false
	}

	// Check sender filter
	if filter.SenderID != "" && event.Sender != filter.SenderID {
		return false
	}

	// Check event type filter
	if len(filter.EventType) > 0 {
		match := false
		for _, eventType := range filter.EventType {
			if event.Type == eventType {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

// handleWebSocketConnect handles new WebSocket connections
func (b *EventBus) handleWebSocketConnect(connID string, conn interface{}) error {
	b.securityLog.LogSecurityEvent("websocket_connected",
		slog.String("connection_id", connID))

	return nil
}

// handleWebSocketDisconnect handles WebSocket disconnections
func (b *EventBus) handleWebSocketDisconnect(connID string) {
	b.securityLog.LogSecurityEvent("websocket_disconnected",
		slog.String("connection_id", connID))
}

// handleWebSocketMessage handles incoming WebSocket messages
func (b *EventBus) handleWebSocketMessage(connID string, message []byte) error {
	// Parse message
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return err
	}

	// Handle subscription requests
	if action, ok := msg["action"].(string); ok {
		switch action {
		case "subscribe":
			// Extract filter parameters
			filter := EventFilter{
				RoomID:   toString(msg["room_id"]),
				SenderID:  toString(msg["sender_id"]),
				EventType: toStringSlice(msg["event_types"]),
			}

			// Create subscription
			sub, err := b.Subscribe(filter)
			if err != nil {
				return err
			}

			// Start sending events to this subscriber
			go b.sendToSubscriber(connID, sub)

			return nil

		case "unsubscribe":
			subID := toString(msg["subscriber_id"])
			if subID == "" {
				return fmt.Errorf("subscriber_id required for unsubscribe")
			}
			return b.Unsubscribe(subID)

		case "ping":
			// Handle ping/pong
			return nil

		default:
			return fmt.Errorf("unknown action: %s", action)
		}
	}

	return fmt.Errorf("invalid message format")
}

// sendToSubscriber sends events to a WebSocket subscriber
func (b *EventBus) sendToSubscriber(connID string, sub *Subscriber) {
	defer func() {
		// Clean up subscriber on exit
		b.Unsubscribe(sub.ID)
	}()

	for {
		select {
		case <-sub.ctx.Done():
			return

		case wrapper, ok := <-sub.EventChannel:
			if !ok {
				return
			}

			// Send event to WebSocket
			message := map[string]interface{}{
				"type":        "event",
				"event":       wrapper.Event,
				"received":    wrapper.Received,
				"sequence":    wrapper.Sequence,
			}

			data, err := json.Marshal(message)
			if err != nil {
				b.securityLog.LogSecurityEvent("event_marshal_failed",
					slog.String("subscriber_id", sub.ID),
					slog.String("error", err.Error()))
				return
			}

			// Send via WebSocket (implementation depends on WebSocket server)
			// This is a placeholder - actual implementation would use the WebSocket connection
			_ = data // Suppress unused warning for now

			sub.mu.Lock()
			sub.LastActivity = time.Now()
			sub.mu.Unlock()

		case <-time.After(30 * time.Second):
			// Send keepalive
			// Implementation depends on WebSocket server
			return
		}
	}
}

// cleanupInactiveSubscribers removes inactive subscribers
func (b *EventBus) cleanupInactiveSubscribers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.mu.Lock()
			now := time.Now()
			for id, sub := range b.subscribers {
				sub.mu.RLock()
				inactive := now.Sub(sub.LastActivity) > 30*time.Minute
				sub.mu.RUnlock()

				if inactive {
					b.securityLog.LogSecurityEvent("subscriber_removed_inactive",
						slog.String("subscriber_id", id),
						slog.Duration("inactive_time", now.Sub(sub.LastActivity)))

					sub.cancel()
					close(sub.EventChannel)
					delete(b.subscribers, id)
				}
			}
			b.mu.Unlock()
		}
	}
}

// GetStats returns event bus statistics
func (b *EventBus) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := map[string]interface{}{
		"active_subscribers": len(b.subscribers),
		"max_subscribers":    100, // Should be configurable
		"websocket_enabled":  b.websocketServer != nil,
	}

	if b.websocketServer != nil {
		stats["websocket_addr"] = b.websocketServer.Addr()
		stats["websocket_path"] = b.websocketServer.Path()
	}

	return stats
}

// toString converts interface{} to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// toStringSlice converts interface{} to []string
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			result = append(result, toString(item))
		}
		return result
	}
	if slice, ok := v.([]string); ok {
		return slice
	}
	return nil
}
