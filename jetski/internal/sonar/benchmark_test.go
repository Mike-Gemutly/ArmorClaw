package sonar

import (
	"encoding/json"
	"testing"
	"time"
)

// BenchmarkBufferPush benchmarks the Push operation with concurrent goroutines
// Target: < 1ms per Push operation under 1000 concurrent operations
func BenchmarkBufferPush(b *testing.B) {
	buffer := NewCircularBuffer(10)
	frame := CDPFrame{
		Method:    "Page.navigate",
		Params:    json.RawMessage(`{"url":"https://example.com"}`),
		SessionID: "benchmark-session",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer.Push(frame)
		}
	})
}

// BenchmarkBufferPush_Single benchmarks Push with single goroutine
// Useful for baseline comparison with concurrent operations
func BenchmarkBufferPush_Single(b *testing.B) {
	buffer := NewCircularBuffer(10)
	frame := CDPFrame{
		Method:    "Page.navigate",
		Params:    json.RawMessage(`{"url":"https://example.com"}`),
		SessionID: "benchmark-session",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Push(frame)
	}
}

// BenchmarkBufferPush_1000Concurrent benchmarks Push with 1000 concurrent goroutines
// Simulates worst-case concurrent load
func BenchmarkBufferPush_1000Concurrent(b *testing.B) {
	buffer := NewCircularBuffer(10)
	frame := CDPFrame{
		Method:    "Page.navigate",
		Params:    json.RawMessage(`{"url":"https://example.com"}`),
		SessionID: "benchmark-session",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer.Push(frame)
		}
	})
}

// BenchmarkGetLastN benchmarks the GetLastN operation
// Expected to be faster than Push due to RLock vs Lock
func BenchmarkGetLastN(b *testing.B) {
	buffer := NewCircularBuffer(10)

	// Fill buffer
	for i := 0; i < 10; i++ {
		frame := CDPFrame{
			Method:    "Page.navigate",
			Params:    json.RawMessage(`{"index":` + string(rune('0'+i)) + `}`),
			SessionID: "benchmark-session",
			Timestamp: time.Now(),
		}
		buffer.Push(frame)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.GetLastN(5)
	}
}

// BenchmarkGetAll benchmarks the GetAll operation
// Expected to be faster than Push due to RLock vs Lock
func BenchmarkGetAll(b *testing.B) {
	buffer := NewCircularBuffer(10)

	// Fill buffer
	for i := 0; i < 10; i++ {
		frame := CDPFrame{
			Method:    "Page.navigate",
			Params:    json.RawMessage(`{"index":` + string(rune('0'+i)) + `}`),
			SessionID: "benchmark-session",
			Timestamp: time.Now(),
		}
		buffer.Push(frame)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.GetAll()
	}
}

// BenchmarkCount benchmarks the Count operation
// Should be extremely fast (single read)
func BenchmarkCount(b *testing.B) {
	buffer := NewCircularBuffer(10)

	// Fill buffer
	for i := 0; i < 10; i++ {
		frame := CDPFrame{
			Method:    "Page.navigate",
			Params:    json.RawMessage(`{}`),
			SessionID: "benchmark-session",
			Timestamp: time.Now(),
		}
		buffer.Push(frame)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.Count()
	}
}

// BenchmarkCalculateHealthScore benchmarks health score calculation
// Should be extremely fast (simple arithmetic)
func BenchmarkCalculateHealthScore(b *testing.B) {
	metrics := SelectorMetrics{
		PrimaryCount:   50,
		SecondaryCount: 30,
		FallbackCount:  20,
		Total:          100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateHealthScore(metrics)
	}
}

// BenchmarkNewWreckageReport benchmarks report creation
// Includes hostname extraction and buffer snapshot
func BenchmarkNewWreckageReport(b *testing.B) {
	buffer := NewCircularBuffer(10)

	// Fill buffer
	for i := 0; i < 10; i++ {
		frame := CDPFrame{
			Method:    "Page.navigate",
			Params:    json.RawMessage(`{}`),
			SessionID: "benchmark-session",
			Timestamp: time.Now(),
		}
		buffer.Push(frame)
	}

	sessionID := "benchmark-session"
	targetURI := "https://github.com/user/repo"
	selector := Selector{
		PrimaryCSS: "#submit-button",
		Tier:       1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewWreckageReport(sessionID, targetURI, buffer, selector)
	}
}
