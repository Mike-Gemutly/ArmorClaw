package errors

import (
	"fmt"
	"sync"
	"time"
)

// RingBuffer is a thread-safe circular buffer for events
type RingBuffer struct {
	events []ComponentLogEntry
	size   int
	head   int
	count  int
	mu     sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the given capacity
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 10
	}
	return &RingBuffer{
		events: make([]ComponentLogEntry, size),
		size:   size,
		head:   0,
		count:  0,
	}
}

// Add adds an event to the buffer
func (rb *RingBuffer) Add(event ComponentLogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.events[rb.head] = event
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll returns all events in chronological order (oldest first)
func (rb *RingBuffer) GetAll() []ComponentLogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]ComponentLogEntry, rb.count)

	// If buffer is not full, start from index 0
	// If buffer is full, start from head (oldest entry)
	start := 0
	if rb.count >= rb.size {
		start = rb.head
	}

	for i := 0; i < rb.count; i++ {
		idx := (start + i) % rb.size
		result[i] = rb.events[idx]
	}

	return result
}

// GetLast returns the last n events (most recent)
func (rb *RingBuffer) GetLast(n int) []ComponentLogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 || n <= 0 {
		return nil
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]ComponentLogEntry, n)

	// Start from the most recent entry (head - 1)
	for i := 0; i < n; i++ {
		idx := (rb.head - 1 - i + rb.size) % rb.size
		result[n-1-i] = rb.events[idx]
	}

	return result
}

// Clear removes all events from the buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.events = make([]ComponentLogEntry, rb.size)
	rb.head = 0
	rb.count = 0
}

// Count returns the number of events in the buffer
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// ComponentTracker tracks events for a specific component
type ComponentTracker struct {
	name   string
	buffer *RingBuffer
	mu     sync.RWMutex
}

// NewComponentTracker creates a new component tracker
func NewComponentTracker(name string, bufferSize int) *ComponentTracker {
	return &ComponentTracker{
		name:   name,
		buffer: NewRingBuffer(bufferSize),
	}
}

// Event records an event with optional data
func (ct *ComponentTracker) Event(eventType string, data interface{}) {
	entry := ComponentLogEntry{
		Timestamp: time.Now(),
		Component: ct.name,
		Event:     eventType,
		Data:      data,
	}
	ct.buffer.Add(entry)
}

// Eventf records an event with formatted message (data is string)
func (ct *ComponentTracker) Eventf(eventType, format string, args ...interface{}) {
	ct.Event(eventType, formatString(format, args...))
}

// Success records a success event (helper method)
func (ct *ComponentTracker) Success(operation string, data interface{}) {
	ct.Event(operation+"_success", data)
}

// Failure records a failure event (helper method)
func (ct *ComponentTracker) Failure(operation string, err error, data interface{}) {
	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}
	ct.Event(operation+"_failure", mergeData(data, map[string]interface{}{"error": errMsg}))
}

// Start records an operation start event
func (ct *ComponentTracker) Start(operation string, data interface{}) {
	ct.Event(operation+"_start", data)
}

// GetRecent returns the last n events
func (ct *ComponentTracker) GetRecent(n int) []ComponentLogEntry {
	return ct.buffer.GetLast(n)
}

// GetAll returns all events
func (ct *ComponentTracker) GetAll() []ComponentLogEntry {
	return ct.buffer.GetAll()
}

// Clear removes all events
func (ct *ComponentTracker) Clear() {
	ct.buffer.Clear()
}

// Name returns the component name
func (ct *ComponentTracker) Name() string {
	return ct.name
}

// Count returns the number of events
func (ct *ComponentTracker) Count() int {
	return ct.buffer.Count()
}

// Global component registry
var (
	components   = make(map[string]*ComponentTracker)
	componentsMu sync.RWMutex

	// Default buffer sizes per component
	defaultBufferSizes = map[string]int{
		"docker":    10,
		"matrix":    10,
		"rpc":       10,
		"keystore":  5,
		"voice":     10,
		"webrtc":    10,
		"container": 10,
		"budget":    5,
		"audit":     5,
		"secrets":   5,
		"turn":      5,
		"websocket": 10,
	}

	// Default buffer size for unknown components
	defaultBufferSize = 10
)

// init initializes default component trackers
func init() {
	// Create trackers for known components
	for name, size := range defaultBufferSizes {
		components[name] = NewComponentTracker(name, size)
	}
}

// GetComponentTracker returns the tracker for a component, creating if needed
func GetComponentTracker(name string) *ComponentTracker {
	componentsMu.RLock()
	if tracker, ok := components[name]; ok {
		componentsMu.RUnlock()
		return tracker
	}
	componentsMu.RUnlock()

	// Create new tracker
	componentsMu.Lock()
	defer componentsMu.Unlock()

	// Double check after acquiring write lock
	if tracker, ok := components[name]; ok {
		return tracker
	}

	size := defaultBufferSize
	if s, ok := defaultBufferSizes[name]; ok {
		size = s
	}

	tracker := NewComponentTracker(name, size)
	components[name] = tracker
	return tracker
}

// TrackEvent records an event for a component
func TrackEvent(component, eventType string, data interface{}) {
	GetComponentTracker(component).Event(eventType, data)
}

// TrackSuccess records a success event
func TrackSuccess(component, operation string, data interface{}) {
	GetComponentTracker(component).Success(operation, data)
}

// TrackFailure records a failure event
func TrackFailure(component, operation string, err error, data interface{}) {
	GetComponentTracker(component).Failure(operation, err, data)
}

// TrackStart records an operation start
func TrackStart(component, operation string, data interface{}) {
	GetComponentTracker(component).Start(operation, data)
}

// GetRecentEvents returns recent events from a component
func GetRecentEvents(component string, n int) []ComponentLogEntry {
	return GetComponentTracker(component).GetRecent(n)
}

// GetAllComponentEvents returns all events from a component
func GetAllComponentEvents(component string) []ComponentLogEntry {
	return GetComponentTracker(component).GetAll()
}

// GetMultiComponentEvents returns recent events from multiple components
// Returns up to n events per component, merged and sorted by timestamp
func GetMultiComponentEvents(componentNames []string, nPerComponent int) []ComponentLogEntry {
	var allEvents []ComponentLogEntry

	for _, name := range componentNames {
		events := GetRecentEvents(name, nPerComponent)
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp (oldest first)
	sortEventsByTimestamp(allEvents)

	return allEvents
}

// GetAllComponentNames returns all registered component names
func GetAllComponentNames() []string {
	componentsMu.RLock()
	defer componentsMu.RUnlock()

	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	return names
}

// ClearAllComponents clears all component trackers
func ClearAllComponents() {
	componentsMu.RLock()
	defer componentsMu.RUnlock()

	for _, tracker := range components {
		tracker.Clear()
	}
}

// GetComponentStats returns stats for all components
func GetComponentStats() map[string]int {
	componentsMu.RLock()
	defer componentsMu.RUnlock()

	stats := make(map[string]int)
	for name, tracker := range components {
		stats[name] = tracker.Count()
	}
	return stats
}

// Helper functions

func formatString(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func mergeData(base interface{}, extra map[string]interface{}) interface{} {
	if base == nil {
		return extra
	}

	// If base is a map, merge
	if baseMap, ok := base.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for k, v := range baseMap {
			result[k] = v
		}
		for k, v := range extra {
			result[k] = v
		}
		return result
	}

	// Otherwise wrap in a new map
	return map[string]interface{}{
		"data": base,
		"extra": extra,
	}
}

func sortEventsByTimestamp(events []ComponentLogEntry) {
	// Simple insertion sort for small slices
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j].Timestamp.Before(events[j-1].Timestamp); j-- {
			events[j], events[j-1] = events[j-1], events[j]
		}
	}
}
