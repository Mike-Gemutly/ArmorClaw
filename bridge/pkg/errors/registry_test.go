package errors

import (
	"testing"
	"time"
)

func TestSamplingRegistry_ShouldNotify_Critical(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 5 * time.Minute,
	})

	// Critical errors always notify
	for i := 0; i < 5; i++ {
		err := &TracedError{
			Code:      "SYS-001",
			Severity:  SeverityCritical,
			Timestamp: time.Now(),
			TraceID:   "tr_critical",
		}

		if !registry.ShouldNotify(err) {
			t.Errorf("Critical error should always notify, attempt %d", i+1)
		}
	}
}

func TestSamplingRegistry_ShouldNotify_FirstOccurrence(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 5 * time.Minute,
	})

	err := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_first",
	}

	// First occurrence should notify
	if !registry.ShouldNotify(err) {
		t.Error("First occurrence should notify")
	}
}

func TestSamplingRegistry_ShouldNotify_RateLimited(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Second,
	})

	// First occurrence
	err1 := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_1",
	}
	if !registry.ShouldNotify(err1) {
		t.Error("First occurrence should notify")
	}

	// Second occurrence within window - should not notify
	err2 := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now().Add(500 * time.Millisecond),
		TraceID:   "tr_2",
	}
	if registry.ShouldNotify(err2) {
		t.Error("Second occurrence within window should NOT notify")
	}

	// Third occurrence within window - should not notify
	err3 := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now().Add(800 * time.Millisecond),
		TraceID:   "tr_3",
	}
	if registry.ShouldNotify(err3) {
		t.Error("Third occurrence within window should NOT notify")
	}
}

func TestSamplingRegistry_ShouldNotify_WindowExpired(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 100 * time.Millisecond,
	})

	// First occurrence
	baseTime := time.Now()
	err1 := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: baseTime,
		TraceID:   "tr_1",
	}
	registry.ShouldNotify(err1)

	// Multiple occurrences within window (3 repeats)
	for i := 0; i < 3; i++ {
		err := &TracedError{
			Code:      "CTX-001",
			Severity:  SeverityError,
			Timestamp: baseTime.Add(time.Duration(i+1) * 20 * time.Millisecond),
			TraceID:   "tr_repeat",
		}
		registry.ShouldNotify(err)
	}

	// After window expires - should notify with accumulated count
	errAfter := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: baseTime.Add(200 * time.Millisecond), // Past 100ms window
		TraceID:   "tr_after",
	}

	if !registry.ShouldNotify(errAfter) {
		t.Error("Occurrence after window should notify")
	}

	// RepeatCount is the accumulated count (1 first + 3 repeats = 4 total)
	if errAfter.RepeatCount != 4 {
		t.Errorf("RepeatCount = %d, want 4", errAfter.RepeatCount)
	}
}

func TestSamplingRegistry_Record(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	err := &TracedError{
		Code:      "CTX-001",
		Timestamp: time.Now(),
		TraceID:   "tr_record",
	}

	registry.Record(err)

	record := registry.GetRecord("CTX-001")
	if record == nil {
		t.Fatal("Record should exist")
	}
	if record.Count != 1 {
		t.Errorf("Count = %d, want 1", record.Count)
	}
	if record.Notified {
		t.Error("Record should not be marked as notified")
	}
}

func TestSamplingRegistry_GetRecord(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	// Non-existent record
	if registry.GetRecord("NONEXISTENT") != nil {
		t.Error("Non-existent record should return nil")
	}

	// Add and retrieve
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_1",
	})

	record := registry.GetRecord("CTX-001")
	if record == nil {
		t.Fatal("Record should exist")
	}
	if record.Count != 1 {
		t.Errorf("Count = %d, want 1", record.Count)
	}
}

func TestSamplingRegistry_GetAllRecords(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	// Add multiple errors
	for i := 0; i < 3; i++ {
		registry.ShouldNotify(&TracedError{
			Code:      "CTX-00" + string(rune('1'+i)),
			Severity:  SeverityError,
			Timestamp: time.Now(),
			TraceID:   "tr",
		})
	}

	records := registry.GetAllRecords()
	if len(records) != 3 {
		t.Errorf("GetAllRecords() returned %d records, want 3", len(records))
	}
}

func TestSamplingRegistry_Clear(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr",
	})

	registry.Clear()

	if registry.GetRecord("CTX-001") != nil {
		t.Error("Record should be cleared")
	}
}

func TestSamplingRegistry_ClearCode(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr",
	})
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-002",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr",
	})

	registry.ClearCode("CTX-001")

	if registry.GetRecord("CTX-001") != nil {
		t.Error("CTX-001 should be cleared")
	}
	if registry.GetRecord("CTX-002") == nil {
		t.Error("CTX-002 should still exist")
	}
}

func TestSamplingRegistry_MarkResolved(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	// First occurrence
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_1",
	})

	// Mark resolved
	registry.MarkResolved("CTX-001")

	// Next occurrence should be treated as first
	err := &TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_2",
	}

	// Should notify (treated as first occurrence)
	if !registry.ShouldNotify(err) {
		t.Error("After resolution, next occurrence should notify")
	}
}

func TestSamplingRegistry_Stats(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 100 * time.Millisecond,
	})

	baseTime := time.Now()

	// First occurrence
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: baseTime,
		TraceID:   "tr",
	})

	// Two more within window (not notified)
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: baseTime.Add(10 * time.Millisecond),
		TraceID:   "tr",
	})
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: baseTime.Add(20 * time.Millisecond),
		TraceID:   "tr",
	})

	// Another code
	registry.Record(&TracedError{
		Code:      "MAT-001",
		Timestamp: baseTime,
		TraceID:   "tr",
	})

	stats := registry.Stats()

	if stats.UniqueErrorCodes != 2 {
		t.Errorf("UniqueErrorCodes = %d, want 2", stats.UniqueErrorCodes)
	}

	// CTX-001: 3 occurrences, MAT-001: 1 occurrence
	if stats.TotalOccurrences != 4 {
		t.Errorf("TotalOccurrences = %d, want 4", stats.TotalOccurrences)
	}
}

func TestSamplingRegistry_ForceCleanup(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Minute,
		RetentionPeriod: 1 * time.Hour,
	})

	// Add old record
	oldTime := time.Now().Add(-2 * time.Hour)
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: oldTime,
		TraceID:   "tr_old",
	})

	// Add recent record
	registry.ShouldNotify(&TracedError{
		Code:      "CTX-002",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_recent",
	})

	registry.ForceCleanup()

	// Old record should be removed
	if registry.GetRecord("CTX-001") != nil {
		t.Error("Old record should be cleaned up")
	}

	// Recent record should remain
	if registry.GetRecord("CTX-002") == nil {
		t.Error("Recent record should not be cleaned up")
	}
}

func TestSamplingRegistry_SetRateLimitWindow(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())
	registry.SetRateLimitWindow(10 * time.Minute)

	stats := registry.Stats()
	if stats.RateLimitWindow != 10*time.Minute {
		t.Errorf("RateLimitWindow = %v, want 10m", stats.RateLimitWindow)
	}
}

func TestSamplingRegistry_SetRetentionPeriod(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())
	registry.SetRetentionPeriod(48 * time.Hour)

	stats := registry.Stats()
	if stats.RetentionPeriod != 48*time.Hour {
		t.Errorf("RetentionPeriod = %v, want 48h", stats.RetentionPeriod)
	}
}

func TestSamplingRegistry_DefaultConfig(t *testing.T) {
	cfg := DefaultSamplingConfig()

	if cfg.RateLimitWindow != 5*time.Minute {
		t.Errorf("Default RateLimitWindow = %v, want 5m", cfg.RateLimitWindow)
	}
	if cfg.RetentionPeriod != 24*time.Hour {
		t.Errorf("Default RetentionPeriod = %v, want 24h", cfg.RetentionPeriod)
	}
}

func TestSamplingRegistry_ZeroConfig(t *testing.T) {
	// Should use defaults
	registry := NewSamplingRegistry(SamplingConfig{})

	stats := registry.Stats()
	if stats.RateLimitWindow != 5*time.Minute {
		t.Errorf("Zero config RateLimitWindow = %v, want 5m", stats.RateLimitWindow)
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Clear global registry first
	GlobalClear()

	// Test global functions
	err := &TracedError{
		Code:      "TEST-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_global",
	}

	// First occurrence should notify
	if !GlobalShouldNotify(err) {
		t.Error("GlobalShouldNotify should return true for first occurrence")
	}

	// Record should exist
	record := GlobalGetRecord("TEST-001")
	if record == nil {
		t.Error("GlobalGetRecord should return record")
	}

	// Stats should work
	stats := GlobalStats()
	if stats.UniqueErrorCodes < 1 {
		t.Error("GlobalStats should show at least 1 unique error")
	}

	// Mark resolved
	GlobalMarkResolved("TEST-001")
	if GlobalGetRecord("TEST-001") != nil {
		t.Error("After GlobalMarkResolved, record should be gone")
	}
}

func TestSetGlobalRegistry(t *testing.T) {
	customRegistry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Hour,
	})

	originalRegistry := GetGlobalRegistry()
	SetGlobalRegistry(customRegistry)

	// Verify it's using the custom registry
	stats := GlobalStats()
	if stats.RateLimitWindow != 1*time.Hour {
		t.Errorf("Custom registry not in use, RateLimitWindow = %v", stats.RateLimitWindow)
	}

	// Restore original
	SetGlobalRegistry(originalRegistry)
}

func TestSamplingRegistry_Concurrent(t *testing.T) {
	registry := NewSamplingRegistry(SamplingConfig{
		RateLimitWindow: 1 * time.Second,
	})

	done := make(chan bool)

	// Concurrent ShouldNotify calls
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				err := &TracedError{
					Code:      "CTX-001",
					Severity:  SeverityError,
					Timestamp: time.Now(),
					TraceID:   "tr",
				}
				registry.ShouldNotify(err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 100 total occurrences
	record := registry.GetRecord("CTX-001")
	if record == nil {
		t.Fatal("Record should exist")
	}
	if record.Count != 100 {
		t.Errorf("Count = %d, want 100", record.Count)
	}
}

func TestErrorRecord_Copy(t *testing.T) {
	registry := NewSamplingRegistry(DefaultSamplingConfig())

	registry.ShouldNotify(&TracedError{
		Code:      "CTX-001",
		Severity:  SeverityError,
		Timestamp: time.Now(),
		TraceID:   "tr_original",
	})

	record1 := registry.GetRecord("CTX-001")
	record2 := registry.GetRecord("CTX-001")

	// Should be copies, not same pointer
	if record1 == record2 {
		t.Error("GetRecord should return copies, not same pointer")
	}
}
