package eventbus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/websocket"
	wslib "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBroadcaster captures BroadcastEvent calls for assertion.
type mockBroadcaster struct {
	mu       sync.Mutex
	calls    []broadcastCall
	channels []chan []byte
}

type broadcastCall struct {
	eventType string
	payload   []byte
}

func (m *mockBroadcaster) BroadcastEvent(eventType string, payload []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, broadcastCall{eventType: eventType, payload: payload})
	for _, ch := range m.channels {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (m *mockBroadcaster) addChannel() chan []byte {
	ch := make(chan []byte, 16)
	m.mu.Lock()
	m.channels = append(m.channels, ch)
	m.mu.Unlock()
	return ch
}

func (m *mockBroadcaster) lastPayload() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return nil
	}
	return m.calls[len(m.calls)-1].payload
}

// wsHandler is a minimal WebSocket handler that forwards broadcasts to clients.
type wsHandler struct {
	mu      sync.RWMutex
	clients map[string]chan []byte
	upgrader wslib.Upgrader
}

func newWSHandler() *wsHandler {
	return &wsHandler{
		clients: make(map[string]chan []byte),
		upgrader: wslib.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan []byte, 256)
	id := generateID()
	h.mu.Lock()
	h.clients[id] = ch
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, id)
		h.mu.Unlock()
		close(ch)
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range ch {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(wslib.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	<-done
}

func (h *wsHandler) broadcast(payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

func generateID() string {
	return strings.Repeat("x", 16)
}

// TestWebSocketE2E verifies EventBus events flow through the WebSocket
// adapter to connected clients end-to-end.
func TestWebSocketE2E(t *testing.T) {
	handler := newWSHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsAdapter := websocket.NewServer(websocket.Config{
		Addr:              ts.Listener.Addr().String(),
		Path:              "/ws",
		MaxConnections:    10,
		InactivityTimeout: 5 * time.Minute,
	})

	broadcaster := &mockBroadcaster{}
	wsAdapter.SetBroadcaster(broadcaster)

	busCfg := Config{
		WebSocketEnabled:  true,
		WebSocketAddr:     ts.Listener.Addr().String(),
		WebSocketPath:     "/ws",
		MaxSubscribers:    10,
		InactivityTimeout: 5 * time.Minute,
	}
	bus := NewEventBus(busCfg)
	bus.SetBroadcaster(broadcaster)
	defer bus.Stop()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	wsConn, _, err := wslib.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err, "WebSocket client should connect")
	defer wsConn.Close()

	time.Sleep(100 * time.Millisecond)

	t.Run("agent status event arrives on WebSocket client", func(t *testing.T) {
		event := NewAgentStatusChangedEvent("agent-e2e-001", "idle", "running", 120, 5, 3)

		err := bus.PublishBridgeEvent(event)
		require.NoError(t, err)

		// The EventBus calls wsAdapter.Broadcast → broadcaster.BroadcastEvent.
		// We need to also forward from the broadcaster to the WS handler.
		// In production, the HTTP server IS the broadcaster. Here we bridge them.
		go func() {
			time.Sleep(50 * time.Millisecond)
			payload := broadcaster.lastPayload()
			if payload != nil {
				handler.broadcast(payload)
			}
		}()

		wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := wsConn.ReadMessage()
		require.NoError(t, err, "should receive event within 2s")

		var received map[string]interface{}
		require.NoError(t, json.Unmarshal(msg, &received))
		t.Logf("received: %s", string(msg))

		assert.Equal(t, EventTypeAgentStatusChanged, received["type"])
		assert.Contains(t, received, "timestamp")

		data, ok := received["data"].(map[string]interface{})
		require.True(t, ok, "payload should contain 'data' wrapper")
		assert.Equal(t, EventTypeAgentStatusChanged, data["type"],
			"inner event type should match")
	})

	t.Run("disconnected client does not block publishing", func(t *testing.T) {
		wsConn.Close()
		time.Sleep(100 * time.Millisecond)

		event := NewAgentStatusChangedEvent("agent-e2e-002", "running", "stopped", 300, 10, 8)
		err := bus.PublishBridgeEvent(event)
		assert.NoError(t, err)
	})
}

// TestWebSocketE2EDirectBroadcast verifies the full chain:
// EventBus.PublishBridgeEvent → websocket.Server.Broadcast → EventBroadcaster
func TestWebSocketE2EDirectBroadcast(t *testing.T) {
	var received atomic.Value

	wsAdapter := websocket.NewServer(websocket.Config{
		Addr: "localhost:0",
	})
	wsAdapter.SetBroadcaster(&directBroadcaster{received: &received})

	busCfg := Config{
		WebSocketEnabled:  true,
		WebSocketAddr:     "localhost:0",
		WebSocketPath:     "/ws",
		MaxSubscribers:    10,
		InactivityTimeout: 5 * time.Minute,
	}
	bus := NewEventBus(busCfg)
	bus.SetBroadcaster(&directBroadcaster{received: &received})
	defer bus.Stop()

	event := NewAgentStatusChangedEvent("agent-direct", "idle", "busy", 0, 0, 0)
	err := bus.PublishBridgeEvent(event)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	payload := received.Load()
	require.NotNil(t, payload, "broadcaster should have received the event")

	var wrapper map[string]interface{}
	require.NoError(t, json.Unmarshal(payload.([]byte), &wrapper))
	assert.Equal(t, EventTypeAgentStatusChanged, wrapper["type"])
	assert.Contains(t, wrapper, "timestamp")
	assert.Contains(t, wrapper, "data", "wrapper should contain 'data' field")
}

type directBroadcaster struct {
	received *atomic.Value
}

func (d *directBroadcaster) BroadcastEvent(eventType string, payload []byte) {
	d.received.Store(payload)
}

// TestWebSocketAdapterBroadcastWire verifies the websocket adapter
// correctly delegates Broadcast to the injected broadcaster.
func TestWebSocketAdapterBroadcastWire(t *testing.T) {
	broadcaster := &mockBroadcaster{}

	wsAdapter := websocket.NewServer(websocket.Config{Addr: "localhost:0"})
	wsAdapter.SetBroadcaster(broadcaster)

	payload := []byte(`{"type":"test.wire"}`)
	err := wsAdapter.Broadcast(payload)
	require.NoError(t, err)

	require.Len(t, broadcaster.calls, 1)
	assert.Equal(t, payload, broadcaster.calls[0].payload)
}

// TestWebSocketMultipleClients verifies events reach multiple consumers.
func TestWebSocketMultipleClients(t *testing.T) {
	broadcaster := &mockBroadcaster{}

	wsAdapter := websocket.NewServer(websocket.Config{
		Addr:           "localhost:0",
		MaxConnections: 10,
	})
	wsAdapter.SetBroadcaster(broadcaster)

	busCfg := Config{
		WebSocketEnabled:  true,
		WebSocketAddr:     "localhost:0",
		WebSocketPath:     "/ws",
		MaxSubscribers:    10,
		InactivityTimeout: 5 * time.Minute,
	}
	bus := NewEventBus(busCfg)
	bus.SetBroadcaster(broadcaster)
	defer bus.Stop()

	ch1 := broadcaster.addChannel()
	ch2 := broadcaster.addChannel()
	ch3 := broadcaster.addChannel()

	event := NewAgentStatusChangedEvent("agent-multi", "idle", "busy", 0, 0, 0)
	err := bus.PublishBridgeEvent(event)
	require.NoError(t, err)

	for i, ch := range []chan []byte{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			var received map[string]interface{}
			require.NoError(t, json.Unmarshal(msg, &received), "client %d", i)
			assert.Equal(t, EventTypeAgentStatusChanged, received["type"])
		case <-time.After(2 * time.Second):
			t.Fatalf("client %d timed out", i)
		}
	}
}

func TestWebSocketNoBroadcaster(t *testing.T) {
	wsAdapter := websocket.NewServer(websocket.Config{Addr: "localhost:0"})

	err := wsAdapter.Broadcast([]byte(`{"type":"test"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no EventBroadcaster wired")
}

func TestWebSocketStartWithoutBroadcaster(t *testing.T) {
	wsAdapter := websocket.NewServer(websocket.Config{Addr: "localhost:0"})

	err := wsAdapter.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no EventBroadcaster wired")
}

func TestWebSocketAdapterSetBroadcasterAndStart(t *testing.T) {
	wsAdapter := websocket.NewServer(websocket.Config{Addr: "localhost:0"})
	wsAdapter.SetBroadcaster(&mockBroadcaster{})

	err := wsAdapter.Start()
	assert.NoError(t, err)

	err = wsAdapter.Stop()
	assert.NoError(t, err)
}

func TestWebSocketAdapterStopIdempotent(t *testing.T) {
	wsAdapter := websocket.NewServer(websocket.Config{Addr: "localhost:0"})
	assert.NoError(t, wsAdapter.Stop())
	assert.NoError(t, wsAdapter.Stop())
}

func TestWebSocketAdapterAddrPath(t *testing.T) {
	wsAdapter := websocket.NewServer(websocket.Config{
		Addr: "0.0.0.0:9999",
		Path: "/custom-ws",
	})
	assert.Equal(t, "0.0.0.0:9999", wsAdapter.Addr())
	assert.Equal(t, "/custom-ws", wsAdapter.Path())
}

func TestEventBusStopIdempotent(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	bus.Stop()
	bus.Stop()
}

func TestEventBusGetStats(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	stats := bus.GetStats()
	assert.Equal(t, 0, stats["active_subscribers"])
	assert.Equal(t, false, stats["websocket_enabled"])
}

func TestEventBusPublishNilEvent(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	err := bus.Publish(nil)
	assert.Error(t, err)
}

func TestEventBusSubscribePublishFlow(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	sub, err := bus.Subscribe(EventFilter{})
	require.NoError(t, err)

	evt := &MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "hello"},
		EventID: "$event1",
	}
	require.NoError(t, bus.Publish(evt))

	select {
	case wrapper := <-sub.EventChannel:
		assert.Equal(t, "m.room.message", wrapper.Event.Type)
		assert.Equal(t, "!room1:example.com", wrapper.Event.RoomID)
		assert.Equal(t, "@alice:example.com", wrapper.Event.Sender)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on subscriber channel")
	}
}

func TestEventBusPublishBridgeEventNoClients(t *testing.T) {
	broadcaster := &mockBroadcaster{}

	busCfg := Config{
		WebSocketEnabled:  true,
		WebSocketAddr:     "localhost:0",
		WebSocketPath:     "/ws",
		MaxSubscribers:    10,
		InactivityTimeout: 5 * time.Minute,
	}
	bus := NewEventBus(busCfg)
	bus.SetBroadcaster(broadcaster)
	defer bus.Stop()

	event := NewAgentStartedEvent("agent-no-client", "TestAgent", "browser",
		WithCapabilities([]string{"web_browsing"}),
	)
	err := bus.PublishBridgeEvent(event)
	assert.NoError(t, err)

	require.Len(t, broadcaster.calls, 1)

	var wrapper map[string]interface{}
	require.NoError(t, json.Unmarshal(broadcaster.calls[0].payload, &wrapper))
	assert.Equal(t, EventTypeAgentStarted, wrapper["type"])
}

func TestEventBusPublishMultipleEventTypes(t *testing.T) {
	broadcaster := &mockBroadcaster{}

	busCfg := Config{
		WebSocketEnabled:  true,
		WebSocketAddr:     "localhost:0",
		WebSocketPath:     "/ws",
		MaxSubscribers:    10,
		InactivityTimeout: 5 * time.Minute,
	}
	bus := NewEventBus(busCfg)
	bus.SetBroadcaster(broadcaster)
	defer bus.Stop()

	events := []BridgeEvent{
		NewAgentStartedEvent("a1", "Agent1", "browser"),
		NewAgentStoppedEvent("a1", "completed"),
		NewAgentStatusChangedEvent("a1", "running", "idle", 60, 3, 2),
		NewWorkflowStartedEvent("wf1", "browse", 5, nil),
		NewWorkflowProgressEvent("wf1", 2, 5, "step2", "running", 0.4),
		NewWorkflowCompletedEvent("wf1", true, "done", 10*time.Second, 5, 5),
		NewHitlPendingEvent("gate1", "file_access", "needs approval", time.Now().Add(5*time.Minute), "high"),
		NewBudgetAlertEvent("openai", 80.5, 100.0, 80.5, "warning"),
	}

	for _, event := range events {
		err := bus.PublishBridgeEvent(event)
		require.NoError(t, err, "PublishBridgeEvent should succeed for %s", event.EventType())
	}

	assert.Len(t, broadcaster.calls, len(events))

	for i, call := range broadcaster.calls {
		var wrapper map[string]interface{}
		require.NoError(t, json.Unmarshal(call.payload, &wrapper))
		assert.Equal(t, events[i].EventType(), wrapper["type"], "event %d type mismatch", i)
	}
}

func TestEventBusSubscribeWithFilter(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	roomFilter := EventFilter{RoomID: "!room1:example.com"}
	sub, err := bus.Subscribe(roomFilter)
	require.NoError(t, err)

	matching := &MatrixEvent{
		Type: "m.room.message", RoomID: "!room1:example.com",
		Sender: "@alice:example.com", Content: map[string]interface{}{"body": "hello"},
		EventID: "$evt1",
	}
	nonMatching := &MatrixEvent{
		Type: "m.room.message", RoomID: "!room2:example.com",
		Sender: "@bob:example.com", Content: map[string]interface{}{"body": "hi"},
		EventID: "$evt2",
	}

	require.NoError(t, bus.Publish(nonMatching))
	require.NoError(t, bus.Publish(matching))

	select {
	case wrapper := <-sub.EventChannel:
		assert.Equal(t, "!room1:example.com", wrapper.Event.RoomID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out — should only receive filtered events")
	}

	select {
	case <-sub.EventChannel:
		t.Fatal("should not receive non-matching events")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	sub, err := bus.Subscribe(EventFilter{})
	require.NoError(t, err)

	err = bus.Unsubscribe(sub.ID)
	assert.NoError(t, err)

	stats := bus.GetStats()
	assert.Equal(t, 0, stats["active_subscribers"])

	err = bus.Unsubscribe("nonexistent")
	assert.Error(t, err)
}
