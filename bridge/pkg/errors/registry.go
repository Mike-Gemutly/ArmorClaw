package errors

import (
	"sync"
	"time"
)

// ErrorRecord tracks occurrences of a specific error code
type ErrorRecord struct {
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Count     int       `json:"count"`
	TraceID   string    `json:"trace_id"`
	Notified  bool      `json:"notified"`
}

// SamplingRegistry handles rate limiting and deduplication of error notifications
type SamplingRegistry struct {
	seen           map[string]*ErrorRecord // code -> record
	mu             sync.RWMutex
	rateLimitWindow time.Duration
	retentionPeriod time.Duration
	lastCleanup     time.Time
}

// SamplingConfig configures the sampling registry
type SamplingConfig struct {
	RateLimitWindow time.Duration // Window for rate limiting repeats (default 5m)
	RetentionPeriod time.Duration // How long to keep records (default 24h)
}

// DefaultSamplingConfig returns default configuration
func DefaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		RateLimitWindow: 5 * time.Minute,
		RetentionPeriod: 24 * time.Hour,
	}
}

// NewSamplingRegistry creates a new sampling registry
func NewSamplingRegistry(cfg SamplingConfig) *SamplingRegistry {
	if cfg.RateLimitWindow <= 0 {
		cfg.RateLimitWindow = 5 * time.Minute
	}
	if cfg.RetentionPeriod <= 0 {
		cfg.RetentionPeriod = 24 * time.Hour
	}

	return &SamplingRegistry{
		seen:            make(map[string]*ErrorRecord),
		rateLimitWindow: cfg.RateLimitWindow,
		retentionPeriod: cfg.RetentionPeriod,
		lastCleanup:     time.Now(),
	}
}

// ShouldNotify determines if an error should trigger a notification
// based on severity and rate limiting rules:
// - Critical: Always notify
// - First occurrence of code: Notify
// - Repeat within window: Skip, just count
// - Repeat after window: Notify with accumulated count
func (r *SamplingRegistry) ShouldNotify(err *TracedError) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Periodic cleanup
	r.maybeCleanup()

	// Always notify critical errors
	if err.Severity == SeverityCritical {
		record, exists := r.seen[err.Code]
		if exists {
			record.LastSeen = err.Timestamp
			record.Count++
			record.TraceID = err.TraceID
		} else {
			r.seen[err.Code] = &ErrorRecord{
				FirstSeen: err.Timestamp,
				LastSeen:  err.Timestamp,
				Count:     1,
				TraceID:   err.TraceID,
				Notified:  true,
			}
		}
		return true
	}

	record, exists := r.seen[err.Code]

	// First occurrence of this code
	if !exists {
		r.seen[err.Code] = &ErrorRecord{
			FirstSeen: err.Timestamp,
			LastSeen:  err.Timestamp,
			Count:     1,
			TraceID:   err.TraceID,
			Notified:  true,
		}
		return true
	}

	// Repeat occurrence
	timeSinceLast := err.Timestamp.Sub(record.LastSeen)

	// Within rate limit window - don't notify, just count
	if timeSinceLast < r.rateLimitWindow {
		record.Count++
		record.LastSeen = err.Timestamp
		return false
	}

	// Window has passed - notify with accumulated count
	err.RepeatCount = record.Count
	record.LastSeen = err.Timestamp
	record.Count = 1
	record.TraceID = err.TraceID
	record.Notified = true

	return true
}

// Record records an error occurrence without notification check
// Useful for tracking errors that are handled internally
func (r *SamplingRegistry) Record(err *TracedError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.seen[err.Code]
	if !exists {
		r.seen[err.Code] = &ErrorRecord{
			FirstSeen: err.Timestamp,
			LastSeen:  err.Timestamp,
			Count:     1,
			TraceID:   err.TraceID,
			Notified:  false,
		}
		return
	}

	record.Count++
	record.LastSeen = err.Timestamp
	record.TraceID = err.TraceID
}

// GetRecord returns the record for an error code
func (r *SamplingRegistry) GetRecord(code string) *ErrorRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if record, ok := r.seen[code]; ok {
		// Return a copy
		copy := *record
		return &copy
	}
	return nil
}

// GetAllRecords returns all error records
func (r *SamplingRegistry) GetAllRecords() map[string]ErrorRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]ErrorRecord, len(r.seen))
	for code, record := range r.seen {
		result[code] = *record
	}
	return result
}

// Clear removes all records
func (r *SamplingRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seen = make(map[string]*ErrorRecord)
}

// ClearCode removes a specific error code record
func (r *SamplingRegistry) ClearCode(code string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.seen, code)
}

// MarkResolved marks an error code as resolved (for external tracking)
// This doesn't affect rate limiting, just allows external systems to track state
func (r *SamplingRegistry) MarkResolved(code string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove the record so next occurrence is treated as "first"
	delete(r.seen, code)
}

// Stats returns statistics about the registry
func (r *SamplingRegistry) Stats() SamplingStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var totalOccurrences int
	var unnotifiedCount int

	for _, record := range r.seen {
		totalOccurrences += record.Count
		if !record.Notified {
			unnotifiedCount++
		}
	}

	return SamplingStats{
		UniqueErrorCodes:  len(r.seen),
		TotalOccurrences:  totalOccurrences,
		UnnotifiedRecords: unnotifiedCount,
		RateLimitWindow:   r.rateLimitWindow,
		RetentionPeriod:   r.retentionPeriod,
	}
}

// SamplingStats holds registry statistics
type SamplingStats struct {
	UniqueErrorCodes  int           `json:"unique_error_codes"`
	TotalOccurrences  int           `json:"total_occurrences"`
	UnnotifiedRecords int           `json:"unnotified_records"`
	RateLimitWindow   time.Duration `json:"rate_limit_window"`
	RetentionPeriod   time.Duration `json:"retention_period"`
}

// maybeCleanup performs periodic cleanup of old records
func (r *SamplingRegistry) maybeCleanup() {
	now := time.Now()

	// Only cleanup every hour
	if now.Sub(r.lastCleanup) < time.Hour {
		return
	}

	r.lastCleanup = now

	// Remove records older than retention period with no recent activity
	for code, record := range r.seen {
		if now.Sub(record.LastSeen) > r.retentionPeriod {
			delete(r.seen, code)
		}
	}
}

// ForceCleanup forces immediate cleanup of old records
func (r *SamplingRegistry) ForceCleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.lastCleanup = now

	for code, record := range r.seen {
		if now.Sub(record.LastSeen) > r.retentionPeriod {
			delete(r.seen, code)
		}
	}
}

// SetRateLimitWindow updates the rate limit window
func (r *SamplingRegistry) SetRateLimitWindow(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rateLimitWindow = d
}

// SetRetentionPeriod updates the retention period
func (r *SamplingRegistry) SetRetentionPeriod(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retentionPeriod = d
}

// Global registry instance
var globalRegistry *SamplingRegistry
var globalRegistryMu sync.RWMutex

// init initializes the global registry
func init() {
	globalRegistry = NewSamplingRegistry(DefaultSamplingConfig())
}

// GetGlobalRegistry returns the global sampling registry
func GetGlobalRegistry() *SamplingRegistry {
	globalRegistryMu.RLock()
	defer globalRegistryMu.RUnlock()
	return globalRegistry
}

// SetGlobalRegistry sets the global sampling registry
func SetGlobalRegistry(registry *SamplingRegistry) {
	globalRegistryMu.Lock()
	defer globalRegistryMu.Unlock()
	globalRegistry = registry
}

// GlobalShouldNotify checks if an error should be notified using the global registry
func GlobalShouldNotify(err *TracedError) bool {
	return GetGlobalRegistry().ShouldNotify(err)
}

// GlobalRecord records an error in the global registry
func GlobalRecord(err *TracedError) {
	GetGlobalRegistry().Record(err)
}

// GlobalGetRecord gets a record from the global registry
func GlobalGetRecord(code string) *ErrorRecord {
	return GetGlobalRegistry().GetRecord(code)
}

// GlobalMarkResolved marks an error as resolved in the global registry
func GlobalMarkResolved(code string) {
	GetGlobalRegistry().MarkResolved(code)
}

// GlobalStats returns statistics for the global registry
func GlobalStats() SamplingStats {
	return GetGlobalRegistry().Stats()
}

// GlobalClear clears the global registry
func GlobalClear() {
	GetGlobalRegistry().Clear()
}
