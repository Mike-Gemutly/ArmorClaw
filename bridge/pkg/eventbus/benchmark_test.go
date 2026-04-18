package eventbus

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

//=============================================================================
// Benchmark: WebSocket Event Delivery (Publish → Subscriber Channel)
//=============================================================================

func BenchmarkWebSocketEventDelivery(b *testing.B) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	sub, err := bus.Subscribe(EventFilter{})
	if err != nil {
		b.Fatal(err)
	}

	// Drain events in background to prevent channel backpressure
	go func() {
		for range sub.EventChannel {
		}
	}()

	evt := &MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "benchmark message"},
		EventID: "$bench-event",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := bus.Publish(evt); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebSocketEventDelivery_Filtered(b *testing.B) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	// Subscribe with room filter — only events from !room1 pass
	sub, err := bus.Subscribe(EventFilter{RoomID: "!room1:example.com"})
	if err != nil {
		b.Fatal(err)
	}
	go func() {
		for range sub.EventChannel {
		}
	}()

	matching := &MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "hello"},
		EventID: "$match",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		bus.Publish(matching)
	}
}

func BenchmarkWebSocketEventDelivery_MultiSubscriber(b *testing.B) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	const subscriberCount = 10
	for i := 0; i < subscriberCount; i++ {
		sub, err := bus.Subscribe(EventFilter{})
		if err != nil {
			b.Fatal(err)
		}
		go func(ch chan *MatrixEventWrapper) {
			for range ch {
			}
		}(sub.EventChannel)
	}

	evt := &MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "broadcast"},
		EventID: "$bench-broadcast",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := bus.Publish(evt); err != nil {
			b.Fatal(err)
		}
	}
}

//=============================================================================
// Benchmark: Bridge Event Publishing
//=============================================================================

func BenchmarkPublishBridgeEvent(b *testing.B) {
	broadcaster := &benchBroadcaster{}

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

	event := NewAgentStatusChangedEvent("agent-bench", "idle", "running", 120, 5, 3)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := bus.PublishBridgeEvent(event); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPublishBridgeEvent_MultipleTypes(b *testing.B) {
	broadcaster := &benchBroadcaster{}

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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, evt := range events {
			if err := bus.PublishBridgeEvent(evt); err != nil {
				b.Fatal(err)
			}
		}
	}
}

//=============================================================================
// Benchmark: Event Serialization
//=============================================================================

func BenchmarkEventSerialization(b *testing.B) {
	event := NewAgentStatusChangedEvent("agent-ser", "idle", "running", 120, 5, 3)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event.ToJSON()
	}
}

func BenchmarkEventWrapperSerialization(b *testing.B) {
	event := NewAgentStatusChangedEvent("agent-wrap", "idle", "running", 120, 5, 3)
	wrapper, err := WrapEvent(event)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrapper.ToJSON()
	}
}

//=============================================================================
// Benchmark: Subscribe/Unsubscribe
//=============================================================================

func BenchmarkSubscribeUnsubscribe(b *testing.B) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sub, err := bus.Subscribe(EventFilter{
			RoomID:   fmt.Sprintf("!room-%d:example.com", i%100),
			SenderID: "@alice:example.com",
		})
		if err != nil {
			b.Fatal(err)
		}
		bus.Unsubscribe(sub.ID)
	}
}

//=============================================================================
// Benchmark: Event Filtering
//=============================================================================

func BenchmarkMatchesFilter(b *testing.B) {
	bus := NewEventBus(Config{WebSocketEnabled: false})
	defer bus.Stop()

	filter := EventFilter{
		RoomID:    "!room1:example.com",
		SenderID:  "@alice:example.com",
		EventType: []string{"m.room.message", "m.room.notice"},
	}

	evt := &MatrixEvent{
		Type:    "m.room.message",
		RoomID:  "!room1:example.com",
		Sender:  "@alice:example.com",
		Content: map[string]interface{}{"body": "hello"},
		EventID: "$filter-bench",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		bus.matchesFilter(evt, filter)
	}
}

//=============================================================================
// Benchmark: BridgeEvent Large Payload
//=============================================================================

func BenchmarkPublishBridgeEvent_LargePayload(b *testing.B) {
	broadcaster := &benchBroadcaster{}

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

	// Event with large metadata
	largeEvent := NewAgentStartedEvent("agent-large", "LargeAgent", "browser",
		WithCapabilities([]string{
			"web_browsing", "form_filling", "data_extraction",
			"screenshot", "pdf_generation", "email_sending",
			"calendar_management", "file_upload", "api_integration",
			"code_review", "testing", "deployment",
		}),
		WithMetadata(map[string]string{
			"env":         "production",
			"region":      "us-west-2",
			"version":     "4.8.0",
			"deploy_id":   "deploy-abc123def456",
			"container":   "armorclaw/agent:latest",
			"runtime":     "docker",
			"network":     "none",
			"memory":      "512M",
			"cpu_limit":   "1.0",
			"pid_limit":   "100",
		}),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := bus.PublishBridgeEvent(largeEvent); err != nil {
			b.Fatal(err)
		}
	}
}

//=============================================================================
// Benchmark: JSON Marshal/Unmarshal Round-Trip
//=============================================================================

func BenchmarkEventJSONRoundTrip(b *testing.B) {
	event := NewWorkflowProgressEvent("wf-rt", 3, 10, "step_3", "running", 0.3)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
		var decoded BaseEvent
		if err := json.Unmarshal(data, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}

//=============================================================================
// Helper: Minimal broadcaster for benchmarks
//=============================================================================

type benchBroadcaster struct{}

func (b *benchBroadcaster) BroadcastEvent(eventType string, payload []byte) {}
