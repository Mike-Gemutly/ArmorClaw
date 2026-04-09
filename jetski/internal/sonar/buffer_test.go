package sonar

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestCircularBuffer_BasicPushGet tests basic Push and GetLastN operations
func TestCircularBuffer_BasicPushGet(t *testing.T) {
	buffer := NewCircularBuffer(10)

	// Push 5 frames
	for i := 0; i < 5; i++ {
		frame := CDPFrame{
			Timestamp: time.Now(),
			Method:    "Test.method",
			Params:    json.RawMessage(`{"index":` + string(rune('0'+i)) + `}`),
			SessionID: "test-session",
		}
		buffer.Push(frame)
	}

	// Get last 3 frames
	frames := buffer.GetLastN(3)

	if len(frames) != 3 {
		t.Errorf("Expected 3 frames, got %d", len(frames))
	}

	if buffer.Count() != 5 {
		t.Errorf("Expected buffer count to be 5, got %d", buffer.Count())
	}
}

// TestCircularBuffer_FIFOEvection tests that oldest frames are evicted when capacity is exceeded
func TestCircularBuffer_FIFOEvection(t *testing.T) {
	buffer := NewCircularBuffer(10)

	// Push 15 frames (exceeds capacity by 5)
	for i := 0; i < 15; i++ {
		frame := CDPFrame{
			Timestamp: time.Now(),
			Method:    "Test.method",
			Params:    json.RawMessage(`{"index":` + string(rune('0'+(i%10))) + `}`),
			SessionID: "test-session",
		}
		buffer.Push(frame)
	}

	// Buffer should only contain last 10 frames
	if buffer.Count() != 10 {
		t.Errorf("Expected buffer count to be 10 (capacity), got %d", buffer.Count())
	}

	// Get all frames should return exactly 10
	allFrames := buffer.GetAll()
	if len(allFrames) != 10 {
		t.Errorf("Expected 10 frames from GetAll, got %d", len(allFrames))
	}
}

// TestCircularBuffer_Concurrency tests thread-safety with 100 concurrent goroutines
func TestCircularBuffer_Concurrency(t *testing.T) {
	buffer := NewCircularBuffer(10)
	numGoroutines := 100
	pushesPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch 100 goroutines pushing simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < pushesPerGoroutine; j++ {
				frame := CDPFrame{
					Timestamp: time.Now(),
					Method:    "Concurrent.method",
					Params:    json.RawMessage(`{"goroutine":` + string(rune('0'+(id%10))) + `}`),
					SessionID: "test-session",
				}
				buffer.Push(frame)
			}
		}(i)
	}

	wg.Wait()

	// Verify buffer integrity
	if buffer.Count() != 10 {
		t.Errorf("Expected buffer count to be 10 after concurrent operations, got %d", buffer.Count())
	}

	// Verify we can read without issues
	frames := buffer.GetLastN(5)
	if len(frames) != 5 {
		t.Errorf("Expected 5 frames from GetLastN after concurrent operations, got %d", len(frames))
	}
}

// TestCircularBuffer_Clear tests clearing the buffer
func TestCircularBuffer_Clear(t *testing.T) {
	buffer := NewCircularBuffer(10)

	// Push 10 frames
	for i := 0; i < 10; i++ {
		frame := CDPFrame{
			Timestamp: time.Now(),
			Method:    "Test.method",
			Params:    json.RawMessage(`{}`),
			SessionID: "test-session",
		}
		buffer.Push(frame)
	}

	// Verify buffer is full
	if buffer.Count() != 10 {
		t.Errorf("Expected buffer count to be 10 before clear, got %d", buffer.Count())
	}

	// Clear buffer
	buffer.Clear()

	// Verify buffer is empty
	if buffer.Count() != 0 {
		t.Errorf("Expected buffer count to be 0 after clear, got %d", buffer.Count())
	}

	// Verify GetAll returns empty
	frames := buffer.GetAll()
	if len(frames) != 0 {
		t.Errorf("Expected empty slice from GetAll after clear, got %d frames", len(frames))
	}
}

// TestCircularBuffer_EmptyBuffer tests operations on empty buffer
func TestCircularBuffer_EmptyBuffer(t *testing.T) {
	buffer := NewCircularBuffer(10)

	// GetLastN on empty buffer should return empty slice
	frames := buffer.GetLastN(5)
	if len(frames) != 0 {
		t.Errorf("Expected empty slice from GetLastN on empty buffer, got %d frames", len(frames))
	}

	// GetLastN(0) should return empty slice
	frames = buffer.GetLastN(0)
	if len(frames) != 0 {
		t.Errorf("Expected empty slice from GetLastN(0), got %d frames", len(frames))
	}

	// Count should be 0
	if buffer.Count() != 0 {
		t.Errorf("Expected count to be 0 on empty buffer, got %d", buffer.Count())
	}

	// GetAll should return empty slice
	frames = buffer.GetAll()
	if len(frames) != 0 {
		t.Errorf("Expected empty slice from GetAll on empty buffer, got %d frames", len(frames))
	}
}

// TestCircularBuffer_GetLastN_ExceedsCount tests GetLastN when n > count
func TestCircularBuffer_GetLastN_ExceedsCount(t *testing.T) {
	buffer := NewCircularBuffer(10)

	// Push only 3 frames
	for i := 0; i < 3; i++ {
		frame := CDPFrame{
			Timestamp: time.Now(),
			Method:    "Test.method",
			Params:    json.RawMessage(`{}`),
			SessionID: "test-session",
		}
		buffer.Push(frame)
	}

	// Request more frames than available
	frames := buffer.GetLastN(10)

	// Should return all 3 frames
	if len(frames) != 3 {
		t.Errorf("Expected 3 frames when requesting 10, got %d", len(frames))
	}
}

// TestCircularBuffer_GetAll tests GetAll returns all frames in insertion order
func TestCircularBuffer_GetAll(t *testing.T) {
	buffer := NewCircularBuffer(5)

	// Push 3 frames with distinct methods
	methods := []string{"First.method", "Second.method", "Third.method"}
	for _, method := range methods {
		frame := CDPFrame{
			Timestamp: time.Now(),
			Method:    method,
			Params:    json.RawMessage(`{}`),
			SessionID: "test-session",
		}
		buffer.Push(frame)
	}

	// Get all frames
	frames := buffer.GetAll()

	if len(frames) != 3 {
		t.Errorf("Expected 3 frames from GetAll, got %d", len(frames))
	}

	// Verify methods are in insertion order
	for i, frame := range frames {
		if frame.Method != methods[i] {
			t.Errorf("Expected frame %d to have method %s, got %s", i, methods[i], frame.Method)
		}
	}
}

// TestCircularBuffer_ZeroCapacity tests buffer creation with zero capacity
func TestCircularBuffer_ZeroCapacity(t *testing.T) {
	buffer := NewCircularBuffer(0)

	// Should default to capacity 10
	if buffer.Count() != 0 {
		t.Errorf("Expected count to be 0 on new buffer, got %d", buffer.Count())
	}

	// Push a frame
	frame := CDPFrame{
		Timestamp: time.Now(),
		Method:    "Test.method",
		Params:    json.RawMessage(`{}`),
		SessionID: "test-session",
	}
	buffer.Push(frame)

	// Should be able to hold at least one frame
	if buffer.Count() != 1 {
		t.Errorf("Expected count to be 1 after push, got %d", buffer.Count())
	}
}
