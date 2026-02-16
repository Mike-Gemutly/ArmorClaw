package errors

import (
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_AddAndGetAll(t *testing.T) {
	rb := NewRingBuffer(5)

	// Add events
	for i := 1; i <= 3; i++ {
		rb.Add(ComponentLogEntry{
			Timestamp: time.Now(),
			Component: "test",
			Event:     "event",
			Data:      i,
		})
	}

	events := rb.GetAll()
	if len(events) != 3 {
		t.Errorf("GetAll() returned %d events, want 3", len(events))
	}

	// Check order (oldest first)
	for i := 0; i < 3; i++ {
		if events[i].Data.(int) != i+1 {
			t.Errorf("Event %d has data %v, want %d", i, events[i].Data, i+1)
		}
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add more events than capacity
	for i := 1; i <= 5; i++ {
		rb.Add(ComponentLogEntry{
			Timestamp: time.Now(),
			Data:      i,
		})
	}

	events := rb.GetAll()
	if len(events) != 3 {
		t.Errorf("GetAll() returned %d events, want 3", len(events))
	}

	// Should have events 3, 4, 5 (oldest 1, 2 were overwritten)
	for i, expected := range []int{3, 4, 5} {
		if events[i].Data.(int) != expected {
			t.Errorf("Event %d has data %v, want %d", i, events[i].Data, expected)
		}
	}
}

func TestRingBuffer_GetLast(t *testing.T) {
	rb := NewRingBuffer(10)

	for i := 1; i <= 5; i++ {
		rb.Add(ComponentLogEntry{
			Timestamp: time.Now(),
			Data:      i,
		})
	}

	// Get last 3
	last3 := rb.GetLast(3)
	if len(last3) != 3 {
		t.Errorf("GetLast(3) returned %d events, want 3", len(last3))
	}

	// Most recent should be last
	if last3[2].Data.(int) != 5 {
		t.Errorf("Most recent event should be 5, got %v", last3[2].Data)
	}

	// Get last 10 (more than available)
	last10 := rb.GetLast(10)
	if len(last10) != 5 {
		t.Errorf("GetLast(10) returned %d events, want 5", len(last10))
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)

	for i := 1; i <= 3; i++ {
		rb.Add(ComponentLogEntry{Data: i})
	}

	rb.Clear()

	if rb.Count() != 0 {
		t.Errorf("After Clear(), Count() = %d, want 0", rb.Count())
	}

	events := rb.GetAll()
	if len(events) != 0 {
		t.Errorf("After Clear(), GetAll() returned %d events, want 0", len(events))
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(5)

	if rb.Count() != 0 {
		t.Error("Empty buffer should have count 0")
	}

	events := rb.GetAll()
	if events != nil {
		t.Error("Empty buffer GetAll() should return nil")
	}

	last := rb.GetLast(5)
	if last != nil {
		t.Error("Empty buffer GetLast() should return nil")
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				rb.Add(ComponentLogEntry{
					Component: "test",
					Data:      id*10 + j,
				})
			}
		}(i)
	}

	wg.Wait()

	// Should have 100 events
	if rb.Count() != 100 {
		t.Errorf("Count() = %d, want 100", rb.Count())
	}
}

func TestComponentTracker_Event(t *testing.T) {
	ct := NewComponentTracker("test", 10)

	ct.Event("test_event", map[string]interface{}{"key": "value"})
	ct.Event("another_event", "simple data")

	events := ct.GetAll()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].Component != "test" {
		t.Errorf("Component = %q, want %q", events[0].Component, "test")
	}
}

func TestComponentTracker_SuccessFailure(t *testing.T) {
	ct := NewComponentTracker("test", 10)

	ct.Start("operation", map[string]interface{}{"id": "123"})
	ct.Success("operation", map[string]interface{}{"result": "ok"})
	ct.Failure("operation", nil, map[string]interface{}{"reason": "test"})

	events := ct.GetAll()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Check event names
	expectedEvents := []string{"operation_start", "operation_success", "operation_failure"}
	for i, expected := range expectedEvents {
		if events[i].Event != expected {
			t.Errorf("Event %d = %q, want %q", i, events[i].Event, expected)
		}
	}
}

func TestComponentTracker_GetRecent(t *testing.T) {
	ct := NewComponentTracker("test", 10)

	for i := 1; i <= 5; i++ {
		ct.Event("event", i)
	}

	recent := ct.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("GetRecent(3) returned %d events, want 3", len(recent))
	}

	// Most recent should be 5
	if recent[2].Data.(int) != 5 {
		t.Errorf("Most recent event should be 5, got %v", recent[2].Data)
	}
}

func TestComponentTracker_Clear(t *testing.T) {
	ct := NewComponentTracker("test", 10)

	ct.Event("event", "data")
	ct.Clear()

	if ct.Count() != 0 {
		t.Errorf("After Clear(), Count() = %d, want 0", ct.Count())
	}
}

func TestGetComponentTracker(t *testing.T) {
	// Get existing tracker (docker is pre-registered)
	dockerTracker := GetComponentTracker("docker")
	if dockerTracker == nil {
		t.Error("GetComponentTracker(docker) returned nil")
	}
	if dockerTracker.Name() != "docker" {
		t.Errorf("Name() = %q, want %q", dockerTracker.Name(), "docker")
	}

	// Get new tracker
	customTracker := GetComponentTracker("custom_component")
	if customTracker == nil {
		t.Error("GetComponentTracker(custom_component) returned nil")
	}

	// Second call should return same instance
	customTracker2 := GetComponentTracker("custom_component")
	if customTracker != customTracker2 {
		t.Error("Second call should return same tracker instance")
	}
}

func TestTrackEvent(t *testing.T) {
	TrackEvent("test", "tracked_event", map[string]interface{}{"test": true})

	events := GetRecentEvents("test", 1)
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Event != "tracked_event" {
		t.Errorf("Event = %q, want %q", events[0].Event, "tracked_event")
	}
}

func TestTrackSuccessFailure(t *testing.T) {
	// Clear the test component first
	GetComponentTracker("test").Clear()

	TrackSuccess("test", "op", nil)
	TrackFailure("test", "op", nil, nil)

	events := GetRecentEvents("test", 2)
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// GetLast returns most recent last
	if events[0].Event != "op_success" {
		t.Errorf("First event = %q, want op_success", events[0].Event)
	}
	if events[1].Event != "op_failure" {
		t.Errorf("Second event = %q, want op_failure", events[1].Event)
	}
}

func TestGetMultiComponentEvents(t *testing.T) {
	// Clear any existing events
	ClearAllComponents()

	// Add events to multiple components
	TrackEvent("docker", "docker_event", 1)
	TrackEvent("matrix", "matrix_event", 2)
	TrackEvent("docker", "docker_event2", 3)

	events := GetMultiComponentEvents([]string{"docker", "matrix"}, 5)

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Should be sorted by timestamp
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Error("Events should be sorted by timestamp (oldest first)")
		}
	}
}

func TestGetAllComponentNames(t *testing.T) {
	names := GetAllComponentNames()

	if len(names) == 0 {
		t.Error("GetAllComponentNames() returned empty list")
	}

	// Should contain pre-registered components
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	expected := []string{"docker", "matrix", "rpc", "keystore"}
	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("Expected component %q not found", exp)
		}
	}
}

func TestClearAllComponents(t *testing.T) {
	// Add some events
	TrackEvent("docker", "event", 1)
	TrackEvent("matrix", "event", 2)

	ClearAllComponents()

	stats := GetComponentStats()
	for name, count := range stats {
		if count != 0 {
			t.Errorf("Component %q has %d events after ClearAllComponents()", name, count)
		}
	}
}

func TestGetComponentStats(t *testing.T) {
	ClearAllComponents()

	TrackEvent("docker", "event1", nil)
	TrackEvent("docker", "event2", nil)
	TrackEvent("matrix", "event1", nil)

	stats := GetComponentStats()

	if stats["docker"] != 2 {
		t.Errorf("docker count = %d, want 2", stats["docker"])
	}
	if stats["matrix"] != 1 {
		t.Errorf("matrix count = %d, want 1", stats["matrix"])
	}
}

func TestRingBuffer_ZeroSize(t *testing.T) {
	rb := NewRingBuffer(0)

	// Should default to 10
	rb.Add(ComponentLogEntry{Data: 1})
	rb.Add(ComponentLogEntry{Data: 2})

	events := rb.GetAll()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestRingBuffer_NegativeSize(t *testing.T) {
	rb := NewRingBuffer(-5)

	// Should default to 10
	rb.Add(ComponentLogEntry{Data: 1})

	if rb.Count() != 1 {
		t.Errorf("Count() = %d, want 1", rb.Count())
	}
}

func TestComponentTracker_Name(t *testing.T) {
	ct := NewComponentTracker("my_component", 5)

	if ct.Name() != "my_component" {
		t.Errorf("Name() = %q, want %q", ct.Name(), "my_component")
	}
}

func TestMergeData(t *testing.T) {
	tests := []struct {
		name        string
		base        interface{}
		extra       map[string]interface{}
		checkKeys   map[string]interface{}
	}{
		{
			name:  "nil base",
			base:  nil,
			extra: map[string]interface{}{"key": "value"},
			checkKeys: map[string]interface{}{"key": "value"},
		},
		{
			name:  "map base",
			base:  map[string]interface{}{"a": 1},
			extra: map[string]interface{}{"b": 2},
			checkKeys: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:  "non-map base",
			base:  "string_value",
			extra: map[string]interface{}{"key": "value"},
			checkKeys: map[string]interface{}{"data": "string_value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeData(tt.base, tt.extra)

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("Result is not a map")
			}

			for k, v := range tt.checkKeys {
				if resultMap[k] != v {
					t.Errorf("mergeData()[%q] = %v, want %v", k, resultMap[k], v)
				}
			}
		})
	}
}

func TestSortEventsByTimestamp(t *testing.T) {
	now := time.Now()

	events := []ComponentLogEntry{
		{Timestamp: now.Add(2 * time.Second), Event: "third"},
		{Timestamp: now.Add(-1 * time.Second), Event: "first"},
		{Timestamp: now, Event: "second"},
	}

	sortEventsByTimestamp(events)

	if events[0].Event != "first" {
		t.Errorf("First event = %q, want first", events[0].Event)
	}
	if events[1].Event != "second" {
		t.Errorf("Second event = %q, want second", events[1].Event)
	}
	if events[2].Event != "third" {
		t.Errorf("Third event = %q, want third", events[2].Event)
	}
}
