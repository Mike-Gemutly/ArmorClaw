package sonar

import (
	"encoding/json"
	"testing"
	"time"
)

// TestRecordFrame verifies that RecordFrame creates a CDPFrame and pushes it to the buffer.
func TestRecordFrame(t *testing.T) {
	buf := NewCircularBuffer(10)

	RecordFrame(buf, "Page.navigate", json.RawMessage(`{"url":"https://example.com"}`), "sess-1")

	frames := buf.GetAll()
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}

	frame := frames[0]
	if frame.Method != "Page.navigate" {
		t.Errorf("expected method Page.navigate, got %s", frame.Method)
	}
	if frame.SessionID != "sess-1" {
		t.Errorf("expected session sess-1, got %s", frame.SessionID)
	}
	if string(frame.Params) != `{"url":"https://example.com"}` {
		t.Errorf("unexpected params: %s", string(frame.Params))
	}
	if frame.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if time.Since(frame.Timestamp) > time.Second {
		t.Error("timestamp should be recent")
	}
}

// TestRecordFrame_NilBuffer verifies RecordFrame is safe with nil buffer.
func TestRecordFrame_NilBuffer(t *testing.T) {
	// Must not panic
	RecordFrame(nil, "Page.navigate", json.RawMessage(`{}`), "sess-1")
}

// TestRecordFrame_CircularEviction verifies frames wrap around correctly.
func TestRecordFrame_CircularEviction(t *testing.T) {
	buf := NewCircularBuffer(3)

	for i := 0; i < 5; i++ {
		RecordFrame(buf, "Method"+string(rune('A'+i)), json.RawMessage(`{}`), "sess")
	}

	// Only last 3 should remain
	if buf.Count() != 3 {
		t.Fatalf("expected count 3, got %d", buf.Count())
	}

	frames := buf.GetAll()
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}

	// First frame should be MethodC (index 2), last should be MethodE (index 4)
	if frames[0].Method != "MethodC" {
		t.Errorf("expected first remaining frame MethodC, got %s", frames[0].Method)
	}
	if frames[2].Method != "MethodE" {
		t.Errorf("expected last remaining frame MethodE, got %s", frames[2].Method)
	}
}

// TestRecordFrame_EmptyParams verifies recording with nil/empty params.
func TestRecordFrame_EmptyParams(t *testing.T) {
	buf := NewCircularBuffer(10)

	RecordFrame(buf, "Network.enable", nil, "")

	frames := buf.GetAll()
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}

	if frames[0].Method != "Network.enable" {
		t.Errorf("expected method Network.enable, got %s", frames[0].Method)
	}
}
