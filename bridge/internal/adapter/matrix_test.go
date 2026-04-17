package adapter

import (
	"encoding/json"
	"testing"

	"github.com/armorclaw/bridge/internal/events"
)

func buildTestSyncResponse(roomID string, eventMaps []map[string]interface{}) *SyncResponse {
	rawEvents := make([]json.RawMessage, len(eventMaps))
	for i, evt := range eventMaps {
		data, _ := json.Marshal(evt)
		rawEvents[i] = data
	}

	syncJSON := map[string]interface{}{
		"next_batch": "test_batch",
		"rooms": map[string]interface{}{
			"join": map[string]interface{}{
				roomID: map[string]interface{}{
					"timeline": map[string]interface{}{
						"events": rawEvents,
					},
				},
			},
		},
	}
	data, _ := json.Marshal(syncJSON)
	var resp SyncResponse
	json.Unmarshal(data, &resp)
	return &resp
}

func TestProcessEvents_WorkflowEventHandled(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	bus := events.NewMatrixEventBus(64)
	m.SetEventBus(bus)
	sub := bus.Subscribe()

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "workflow.progress",
			"room_id":       "!test:localhost",
			"sender":        "@agent:localhost",
			"content":       map[string]interface{}{"step": 1, "total": 5},
			"event_id":      "$wf1",
			"origin_server": "matrix",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed, got %d", processed)
	}

	select {
	case evt := <-sub:
		if evt.Type != "workflow.progress" {
			t.Errorf("expected workflow.progress event, got %s", evt.Type)
		}
		if evt.RoomID != "!test:localhost" {
			t.Errorf("expected room !test:localhost, got %s", evt.RoomID)
		}
	default:
		t.Error("workflow event was not published to event bus")
	}
}

func TestProcessEvents_AgentEventHandled(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	bus := events.NewMatrixEventBus(64)
	m.SetEventBus(bus)
	sub := bus.Subscribe()

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "agent.comment",
			"room_id":       "!test:localhost",
			"sender":        "@bot:localhost",
			"content":       map[string]interface{}{"body": "note"},
			"event_id":      "$ag1",
			"origin_server": "matrix",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed, got %d", processed)
	}

	select {
	case evt := <-sub:
		if evt.Type != "agent.comment" {
			t.Errorf("expected agent.comment event, got %s", evt.Type)
		}
	default:
		t.Error("agent event was not published to event bus")
	}
}

func TestProcessEvents_BlockerEventHandled(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	bus := events.NewMatrixEventBus(64)
	m.SetEventBus(bus)
	sub := bus.Subscribe()

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "blocker.required",
			"room_id":       "!test:localhost",
			"sender":        "@system:localhost",
			"content":       map[string]interface{}{"reason": "approval needed"},
			"event_id":      "$bl1",
			"origin_server": "matrix",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed, got %d", processed)
	}

	select {
	case evt := <-sub:
		if evt.Type != "blocker.required" {
			t.Errorf("expected blocker.required event, got %s", evt.Type)
		}
	default:
		t.Error("blocker event was not published to event bus")
	}
}

func TestProcessEvents_MessageUnchanged(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	bus := events.NewMatrixEventBus(64)
	m.SetEventBus(bus)
	sub := bus.Subscribe()

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "m.room.message",
			"room_id":       "!test:localhost",
			"sender":        "@user:localhost",
			"content":       map[string]interface{}{"msgtype": "m.text", "body": "hello"},
			"event_id":      "$msg1",
			"origin_server": "matrix",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed, got %d", processed)
	}

	select {
	case evt := <-sub:
		if evt.Type != "m.room.message" {
			t.Errorf("expected m.room.message event, got %s", evt.Type)
		}
	default:
		t.Error("m.room.message event was not published to event bus")
	}

	select {
	case evt := <-m.ReceiveEvents():
		if evt.Type != "m.room.message" {
			t.Errorf("expected m.room.message in queue, got %s", evt.Type)
		}
	default:
		t.Error("m.room.message event was not queued")
	}
}

func TestProcessEvents_UnknownCustomLogged(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	bus := events.NewMatrixEventBus(64)
	m.SetEventBus(bus)
	sub := bus.Subscribe()

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "custom.unknown",
			"room_id":       "!test:localhost",
			"sender":        "@test:localhost",
			"content":       map[string]interface{}{},
			"event_id":      "$unk1",
			"origin_server": "matrix",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed (not silently dropped), got %d", processed)
	}

	select {
	case evt := <-sub:
		t.Errorf("unknown custom event should not be published to event bus, got type=%s", evt.Type)
	default:
	}
}

func TestProcessEvents_MatrixStateEventNotCustom(t *testing.T) {
	m, err := New(Config{
		HomeserverURL: "http://localhost:6167",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	syncResp := buildTestSyncResponse("!test:localhost", []map[string]interface{}{
		{
			"type":          "m.room.member",
			"room_id":       "!test:localhost",
			"sender":        "@user:localhost",
			"content":       map[string]interface{}{"membership": "join"},
			"event_id":      "$mem1",
			"origin_server": "matrix",
			"state_key":     "@user:localhost",
		},
	})

	processed := m.processEvents(syncResp)
	if processed != 1 {
		t.Errorf("expected 1 event processed, got %d", processed)
	}
}
