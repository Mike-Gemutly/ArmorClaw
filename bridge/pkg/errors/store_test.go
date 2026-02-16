package errors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewErrorStore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "errors.db")

	store, err := NewErrorStore(StoreConfig{Path: path})
	if err != nil {
		t.Fatalf("NewErrorStore() error = %v", err)
	}
	defer store.Close()

	if store.Path() != path {
		t.Errorf("Path() = %q, want %q", store.Path(), path)
	}

	// Check file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestErrorStore_Store(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	tracedErr := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test error",
		TraceID:   "tr_test_1",
		Timestamp: time.Now(),
	}

	if err := store.Store(context.Background(), tracedErr); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Verify it was stored
	results, err := store.Query(context.Background(), ErrorQuery{Code: "CTX-001"})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Query() returned %d results, want 1", len(results))
	}

	if results[0].Code != "CTX-001" {
		t.Errorf("Code = %q, want CTX-001", results[0].Code)
	}
}

func TestErrorStore_Store_DuplicateCode(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store first error
	err1 := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "first error",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	}
	store.Store(context.Background(), err1)

	// Store second error with same code
	err2 := &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "second error",
		TraceID:   "tr_2",
		Timestamp: time.Now().Add(time.Hour),
	}
	store.Store(context.Background(), err2)

	// Should have updated occurrences, not created new row
	results, _ := store.Query(context.Background(), ErrorQuery{Code: "CTX-001"})
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Occurrences != 2 {
		t.Errorf("Occurrences = %d, want 2", results[0].Occurrences)
	}
}

func TestErrorStore_Query(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store multiple errors
	codes := []string{"CTX-001", "CTX-002", "MAT-001", "RPC-001"}
	for i, code := range codes {
		store.Store(context.Background(), &TracedError{
			Code:      code,
			Category:  getCategoryFromCode(code),
			Severity:  SeverityError,
			Message:   "test",
			TraceID:   "tr_" + code,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	tests := []struct {
		name     string
		query    ErrorQuery
		expected int
	}{
		{"all", ErrorQuery{}, 4},
		{"by code", ErrorQuery{Code: "CTX-001"}, 1},
		{"by category", ErrorQuery{Category: "container"}, 2},
		{"with limit", ErrorQuery{Limit: 2}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Query(context.Background(), tt.query)
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Query() returned %d results, want %d", len(results), tt.expected)
			}
		})
	}
}

func TestErrorStore_Query_Resolved(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store and resolve one error
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	})
	store.Resolve(context.Background(), "tr_1", "@admin:example.com")

	// Store another unresolved
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-002",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_2",
		Timestamp: time.Now(),
	})

	resolved := true
	unresolved := false

	resolvedResults, _ := store.Query(context.Background(), ErrorQuery{Resolved: &resolved})
	if len(resolvedResults) != 1 {
		t.Errorf("Resolved query returned %d, want 1", len(resolvedResults))
	}

	unresolvedResults, _ := store.Query(context.Background(), ErrorQuery{Resolved: &unresolved})
	if len(unresolvedResults) != 1 {
		t.Errorf("Unresolved query returned %d, want 1", len(unresolvedResults))
	}
}

func TestErrorStore_Resolve(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	})

	err := store.Resolve(context.Background(), "tr_1", "@admin:example.com")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Verify it's resolved
	resolved := true
	results, _ := store.Query(context.Background(), ErrorQuery{Resolved: &resolved})
	if len(results) != 1 {
		t.Fatalf("Expected 1 resolved error")
	}

	if !results[0].Resolved {
		t.Error("Error should be marked as resolved")
	}
	if results[0].ResolvedBy != "@admin:example.com" {
		t.Errorf("ResolvedBy = %q, want @admin:example.com", results[0].ResolvedBy)
	}
}

func TestErrorStore_Resolve_NotFound(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	err := store.Resolve(context.Background(), "nonexistent", "@admin:example.com")
	if err == nil {
		t.Error("Resolve() should return error for nonexistent trace ID")
	}
}

func TestErrorStore_Unresolve(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	})
	store.Resolve(context.Background(), "tr_1", "@admin:example.com")

	// Unresolve
	err := store.Unresolve(context.Background(), "tr_1")
	if err != nil {
		t.Fatalf("Unresolve() error = %v", err)
	}

	// Verify it's unresolved
	resolved := true
	results, _ := store.Query(context.Background(), ErrorQuery{Resolved: &resolved})
	if len(results) != 0 {
		t.Error("Error should be unresolved")
	}
}

func TestErrorStore_Delete(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	})

	err := store.Delete(context.Background(), "tr_1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	results, _ := store.Query(context.Background(), ErrorQuery{})
	if len(results) != 0 {
		t.Error("Error should be deleted")
	}
}

func TestErrorStore_Cleanup(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store and resolve an old error
	oldTime := time.Now().Add(-60 * 24 * time.Hour) // 60 days ago
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "old error",
		TraceID:   "tr_old",
		Timestamp: oldTime,
	})

	// Manually set resolved_at to be old (Resolve sets it to now)
	store.mu.Lock()
	store.db.Exec("UPDATE errors SET resolved = TRUE, resolved_by = ?, resolved_at = ? WHERE trace_id = ?",
		"@admin:example.com", oldTime, "tr_old")
	store.mu.Unlock()

	// Store a recent error (unresolved)
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-002",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "recent error",
		TraceID:   "tr_recent",
		Timestamp: time.Now(),
	})

	// Cleanup (default 30 day retention)
	deleted, err := store.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if deleted != 1 {
		t.Errorf("Cleanup() deleted %d, want 1", deleted)
	}

	// Verify old resolved error is gone, recent unresolved remains
	results, _ := store.Query(context.Background(), ErrorQuery{})
	if len(results) != 1 {
		t.Errorf("Expected 1 error remaining, got %d", len(results))
	}
}

func TestErrorStore_Stats(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store multiple errors
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-001",
		Category:  "container",
		Severity:  SeverityCritical,
		Message:   "critical",
		TraceID:   "tr_1",
		Timestamp: time.Now(),
	})
	store.Store(context.Background(), &TracedError{
		Code:      "MAT-001",
		Category:  "matrix",
		Severity:  SeverityError,
		Message:   "error",
		TraceID:   "tr_2",
		Timestamp: time.Now(),
	})
	store.Store(context.Background(), &TracedError{
		Code:      "RPC-001",
		Category:  "rpc",
		Severity:  SeverityWarning,
		Message:   "warning",
		TraceID:   "tr_3",
		Timestamp: time.Now(),
	})
	store.Resolve(context.Background(), "tr_3", "@admin:example.com")

	stats, err := store.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.TotalErrors != 3 {
		t.Errorf("TotalErrors = %d, want 3", stats.TotalErrors)
	}
	if stats.UnresolvedErrors != 2 {
		t.Errorf("UnresolvedErrors = %d, want 2", stats.UnresolvedErrors)
	}
	if stats.UniqueCodes != 3 {
		t.Errorf("UniqueCodes = %d, want 3", stats.UniqueCodes)
	}
	if stats.BySeverity[SeverityCritical] != 1 {
		t.Errorf("BySeverity[critical] = %d, want 1", stats.BySeverity[SeverityCritical])
	}
	if stats.ByCategory["container"] != 1 {
		t.Errorf("ByCategory[container] = %d, want 1", stats.ByCategory["container"])
	}
}

func TestErrorStore_Query_Pagination(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store 10 errors with different codes (to avoid deduplication)
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("CTX-%03d", i)
		store.Store(context.Background(), &TracedError{
			Code:      code,
			Category:  "container",
			Severity:  SeverityError,
			Message:   "test",
			TraceID:   "tr_" + code,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	// First page
	page1, _ := store.Query(context.Background(), ErrorQuery{Limit: 5, Offset: 0})
	if len(page1) != 5 {
		t.Errorf("Page 1 returned %d, want 5", len(page1))
	}

	// Second page
	page2, _ := store.Query(context.Background(), ErrorQuery{Limit: 5, Offset: 5})
	if len(page2) != 5 {
		t.Errorf("Page 2 returned %d, want 5", len(page2))
	}
}

func TestErrorStore_Query_OrderBy(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Store errors with different occurrence counts (by storing same code multiple times)
	for i := 0; i < 5; i++ {
		store.Store(context.Background(), &TracedError{
			Code:      "CTX-001",
			Category:  "container",
			Severity:  SeverityError,
			Message:   "test",
			TraceID:   "tr_1",
			Timestamp: time.Now(),
		})
	}
	store.Store(context.Background(), &TracedError{
		Code:      "CTX-002",
		Category:  "container",
		Severity:  SeverityError,
		Message:   "test",
		TraceID:   "tr_2",
		Timestamp: time.Now(),
	})

	// Order by occurrences
	results, _ := store.Query(context.Background(), ErrorQuery{
		OrderBy:   "occurrences",
		OrderDesc: true,
	})

	if len(results) < 2 {
		t.Fatal("Need at least 2 results")
	}

	// CTX-001 should be first (5 occurrences vs 1)
	if results[0].Code != "CTX-001" {
		t.Errorf("First result = %s, want CTX-001", results[0].Code)
	}
}

func TestStoreConfig_Defaults(t *testing.T) {
	cfg := DefaultStoreConfig()

	if cfg.Path != "/var/lib/armorclaw/errors.db" {
		t.Errorf("Default path = %q, want /var/lib/armorclaw/errors.db", cfg.Path)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("Default retention = %d, want 30", cfg.RetentionDays)
	}
}

func TestGlobalStore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "global_errors.db")

	store, err := NewErrorStore(StoreConfig{Path: path})
	if err != nil {
		t.Fatalf("NewErrorStore() error = %v", err)
	}
	defer store.Close()

	SetGlobalStore(store)

	// Test global functions
	err = GlobalStore(context.Background(), &TracedError{
		Code:      "GLOBAL-001",
		Category:  "test",
		Severity:  SeverityError,
		Message:   "global test",
		TraceID:   "tr_global",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("GlobalStore() error = %v", err)
	}

	results, err := GlobalQuery(context.Background(), ErrorQuery{Code: "GLOBAL-001"})
	if err != nil {
		t.Fatalf("GlobalQuery() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("GlobalQuery() returned %d results, want 1", len(results))
	}

	stats, err := GlobalStoreStats(context.Background())
	if err != nil {
		t.Fatalf("GlobalStoreStats() error = %v", err)
	}
	if stats.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", stats.TotalErrors)
	}
}

func TestGlobalStore_NotInitialized(t *testing.T) {
	SetGlobalStore(nil)

	err := GlobalStore(context.Background(), &TracedError{})
	if err == nil {
		t.Error("GlobalStore should return error when not initialized")
	}

	_, err = GlobalQuery(context.Background(), ErrorQuery{})
	if err == nil {
		t.Error("GlobalQuery should return error when not initialized")
	}
}

// Helper functions

func newTestStore(t *testing.T) *ErrorStore {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_errors.db")

	store, err := NewErrorStore(StoreConfig{Path: path})
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return store
}

func getCategoryFromCode(code string) string {
	switch {
	case len(code) >= 3 && code[:3] == "CTX":
		return "container"
	case len(code) >= 3 && code[:3] == "MAT":
		return "matrix"
	case len(code) >= 3 && code[:3] == "RPC":
		return "rpc"
	case len(code) >= 3 && code[:3] == "SYS":
		return "system"
	default:
		return "unknown"
	}
}
