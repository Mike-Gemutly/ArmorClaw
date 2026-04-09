package sonar

import (
	"encoding/json"
	"sync"
	"time"
)

// CDPFrame represents a single Chrome DevTools Protocol message frame
type CDPFrame struct {
	Timestamp time.Time       `json:"timestamp"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"` // Preserve exact CDP data
	SessionID string          `json:"session_id"`
}

// CircularBuffer provides thread-safe O(1) circular buffer for CDP frames
// with FIFO eviction when capacity is exceeded
type CircularBuffer struct {
	mu       sync.RWMutex
	frames   []CDPFrame
	capacity int
	writePos int
	count    int
}

// NewCircularBuffer creates a new circular buffer with specified capacity
func NewCircularBuffer(capacity int) *CircularBuffer {
	if capacity <= 0 {
		capacity = 10 // Default capacity
	}
	return &CircularBuffer{
		frames:   make([]CDPFrame, capacity),
		capacity: capacity,
		writePos: 0,
		count:    0,
	}
}

// Push adds a frame to the buffer, evicting oldest if at capacity
// O(1) complexity with thread-safe write operations
func (cb *CircularBuffer) Push(frame CDPFrame) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.frames[cb.writePos] = frame
	cb.writePos = (cb.writePos + 1) % cb.capacity

	if cb.count < cb.capacity {
		cb.count++
	}
}

// GetLastN returns the last n frames from the buffer
// If n > count, returns all available frames
// Thread-safe read operation (uses RLock for concurrent access)
func (cb *CircularBuffer) GetLastN(n int) []CDPFrame {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if n <= 0 || cb.count == 0 {
		return []CDPFrame{}
	}

	if n > cb.count {
		n = cb.count
	}

	result := make([]CDPFrame, n)
	for i := 0; i < n; i++ {
		// Calculate position: (writePos - n + i + capacity) % capacity
		// This wraps around correctly to get frames in reverse chronological order
		pos := (cb.writePos - n + i + cb.capacity) % cb.capacity
		result[i] = cb.frames[pos]
	}

	return result
}

// Clear empties the buffer and resets state
// Thread-safe operation
func (cb *CircularBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.frames = make([]CDPFrame, cb.capacity)
	cb.writePos = 0
	cb.count = 0
}

// Count returns the current number of frames in the buffer
// Thread-safe read operation
func (cb *CircularBuffer) Count() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.count
}

// GetAll returns all frames in the buffer (in insertion order)
func (cb *CircularBuffer) GetAll() []CDPFrame {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.count == 0 {
		return []CDPFrame{}
	}

	result := make([]CDPFrame, cb.count)
	for i := 0; i < cb.count; i++ {
		pos := (cb.writePos - cb.count + i + cb.capacity) % cb.capacity
		result[i] = cb.frames[pos]
	}

	return result
}
