package errors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ErrorStore persists error traces to SQLite
type ErrorStore struct {
	db       *sql.DB
	path     string
	mu       sync.RWMutex
	retentionDays int
}

// StoreConfig configures the error store
type StoreConfig struct {
	Path          string // Path to SQLite database file
	RetentionDays int    // Days to keep resolved errors (0 = default 30)
}

// DefaultStoreConfig returns default configuration
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		Path:          "/var/lib/armorclaw/errors.db",
		RetentionDays: 30,
	}
}

// NewErrorStore creates a new error store
func NewErrorStore(cfg StoreConfig) (*ErrorStore, error) {
	if cfg.Path == "" {
		cfg.Path = "/var/lib/armorclaw/errors.db"
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", cfg.Path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &ErrorStore{
		db:            db,
		path:          cfg.Path,
		retentionDays: cfg.RetentionDays,
	}

	// Run migrations
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// migrate creates or updates the database schema
func (s *ErrorStore) migrate() error {
	// Create errors table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS errors (
			trace_id     TEXT PRIMARY KEY,
			code         TEXT NOT NULL,
			category     TEXT NOT NULL,
			severity     TEXT NOT NULL,
			message      TEXT NOT NULL,
			trace_json   TEXT NOT NULL,
			first_seen   TIMESTAMP NOT NULL,
			last_seen    TIMESTAMP NOT NULL,
			occurrences  INTEGER DEFAULT 1,
			resolved     BOOLEAN DEFAULT FALSE,
			resolved_by  TEXT,
			resolved_at  TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_errors_code ON errors(code);
		CREATE INDEX IF NOT EXISTS idx_errors_category ON errors(category);
		CREATE INDEX IF NOT EXISTS idx_errors_severity ON errors(severity);
		CREATE INDEX IF NOT EXISTS idx_errors_resolved ON errors(resolved);
		CREATE INDEX IF NOT EXISTS idx_errors_first_seen ON errors(first_seen);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// StoredError represents an error retrieved from the store
type StoredError struct {
	TraceID     string        `json:"trace_id"`
	Code        string        `json:"code"`
	Category    string        `json:"category"`
	Severity    Severity      `json:"severity"`
	Message     string        `json:"message"`
	Trace       *TracedError  `json:"trace,omitempty"`
	FirstSeen   time.Time     `json:"first_seen"`
	LastSeen    time.Time     `json:"last_seen"`
	Occurrences int           `json:"occurrences"`
	Resolved    bool          `json:"resolved"`
	ResolvedBy  string        `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

// Store persists a traced error
func (s *ErrorStore) Store(ctx context.Context, tracedErr *TracedError) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize full trace
	traceJSON, err := json.Marshal(tracedErr)
	if err != nil {
		return fmt.Errorf("failed to serialize trace: %w", err)
	}

	// Check if error with this code already exists (for updating occurrences)
	var existingTraceID string
	var existingOccurrences int
	queryErr := s.db.QueryRowContext(ctx,
		"SELECT trace_id, occurrences FROM errors WHERE code = ? AND resolved = FALSE ORDER BY last_seen DESC LIMIT 1",
		tracedErr.Code,
	).Scan(&existingTraceID, &existingOccurrences)

	if queryErr == nil && existingTraceID != "" {
		// Update existing unresolved error
		_, err = s.db.ExecContext(ctx, `
			UPDATE errors SET
				trace_json = ?,
				last_seen = ?,
				occurrences = occurrences + 1
			WHERE trace_id = ?
		`,
			string(traceJSON),
			tracedErr.Timestamp,
			existingTraceID,
		)
		return err
	}

	// Insert new error
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO errors (trace_id, code, category, severity, message, trace_json, first_seen, last_seen, occurrences)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
	`,
		tracedErr.TraceID,
		tracedErr.Code,
		tracedErr.Category,
		string(tracedErr.Severity),
		tracedErr.Message,
		string(traceJSON),
		tracedErr.Timestamp,
		tracedErr.Timestamp,
	)

	return err
}

// ErrorQuery defines parameters for querying errors
type ErrorQuery struct {
	Code      string     // Filter by error code
	Category  string     // Filter by category
	Severity  Severity   // Filter by severity
	Resolved  *bool      // Filter by resolved status (nil = all)
	Since     time.Time  // Only errors after this time
	Until     time.Time  // Only errors before this time
	Limit     int        // Max results (default 20, max 1000)
	Offset    int        // Pagination offset
	OrderBy   string     // "first_seen", "last_seen", "occurrences" (default "last_seen")
	OrderDesc bool       // Sort descending (default true)
}

// Query retrieves errors matching the query parameters
func (s *ErrorStore) Query(ctx context.Context, q ErrorQuery) ([]StoredError, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Set defaults
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 1000 {
		q.Limit = 1000
	}

	// Build query
	query := "SELECT trace_id, code, category, severity, message, trace_json, first_seen, last_seen, occurrences, resolved, resolved_by, resolved_at FROM errors WHERE 1=1"
	args := []interface{}{}

	if q.Code != "" {
		query += " AND code = ?"
		args = append(args, q.Code)
	}
	if q.Category != "" {
		query += " AND category = ?"
		args = append(args, q.Category)
	}
	if q.Severity != "" {
		query += " AND severity = ?"
		args = append(args, string(q.Severity))
	}
	if q.Resolved != nil {
		query += " AND resolved = ?"
		args = append(args, *q.Resolved)
	}
	if !q.Since.IsZero() {
		query += " AND first_seen >= ?"
		args = append(args, q.Since)
	}
	if !q.Until.IsZero() {
		query += " AND first_seen <= ?"
		args = append(args, q.Until)
	}

	// Ordering
	orderCol := "last_seen"
	switch q.OrderBy {
	case "first_seen", "occurrences":
		orderCol = q.OrderBy
	}
	orderDir := "DESC"
	if !q.OrderDesc {
		orderDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderCol, orderDir)

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []StoredError
	for rows.Next() {
		var se StoredError
		var traceJSON string
		var resolvedAt sql.NullTime
		var resolvedBy sql.NullString

		err := rows.Scan(
			&se.TraceID,
			&se.Code,
			&se.Category,
			&se.Severity,
			&se.Message,
			&traceJSON,
			&se.FirstSeen,
			&se.LastSeen,
			&se.Occurrences,
			&se.Resolved,
			&resolvedBy,
			&resolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Parse full trace
		var trace TracedError
		if json.Unmarshal([]byte(traceJSON), &trace) == nil {
			se.Trace = &trace
		}

		if resolvedBy.Valid {
			se.ResolvedBy = resolvedBy.String
		}
		if resolvedAt.Valid {
			se.ResolvedAt = &resolvedAt.Time
		}

		results = append(results, se)
	}

	return results, rows.Err()
}

// Get retrieves a single error by trace ID
func (s *ErrorStore) Get(ctx context.Context, traceID string) (*StoredError, error) {
	results, err := s.Query(ctx, ErrorQuery{
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("error not found: %s", traceID)
	}
	return &results[0], nil
}

// Resolve marks an error as resolved
func (s *ErrorStore) Resolve(ctx context.Context, traceID, resolvedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		UPDATE errors SET
			resolved = TRUE,
			resolved_by = ?,
			resolved_at = ?
		WHERE trace_id = ?
	`, resolvedBy, time.Now(), traceID)

	if err != nil {
		return fmt.Errorf("resolve failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("error not found: %s", traceID)
	}

	return nil
}

// Unresolve marks an error as unresolved (for reopening)
func (s *ErrorStore) Unresolve(ctx context.Context, traceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		UPDATE errors SET
			resolved = FALSE,
			resolved_by = NULL,
			resolved_at = NULL
		WHERE trace_id = ?
	`, traceID)

	return err
}

// Delete removes an error permanently
func (s *ErrorStore) Delete(ctx context.Context, traceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM errors WHERE trace_id = ?", traceID)
	return err
}

// Cleanup removes old resolved errors based on retention policy
func (s *ErrorStore) Cleanup(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -s.retentionDays)

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM errors WHERE resolved = TRUE AND resolved_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}

	return result.RowsAffected()
}

// Stats returns statistics about stored errors
func (s *ErrorStore) Stats(ctx context.Context) (StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stats StoreStats

	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM errors",
	).Scan(&stats.TotalErrors)
	if err != nil {
		return stats, err
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM errors WHERE resolved = FALSE",
	).Scan(&stats.UnresolvedErrors)
	if err != nil {
		return stats, err
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT code) FROM errors",
	).Scan(&stats.UniqueCodes)
	if err != nil {
		return stats, err
	}

	// Get counts by severity
	rows, err := s.db.QueryContext(ctx,
		"SELECT severity, COUNT(*) FROM errors GROUP BY severity",
	)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.BySeverity = make(map[Severity]int)
	for rows.Next() {
		var sev string
		var count int
		if err := rows.Scan(&sev, &count); err != nil {
			return stats, err
		}
		stats.BySeverity[Severity(sev)] = count
	}

	// Get counts by category
	rows, err = s.db.QueryContext(ctx,
		"SELECT category, COUNT(*) FROM errors GROUP BY category",
	)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.ByCategory = make(map[string]int)
	for rows.Next() {
		var cat string
		var count int
		if err := rows.Scan(&cat, &count); err != nil {
			return stats, err
		}
		stats.ByCategory[cat] = count
	}

	return stats, nil
}

// StoreStats holds statistics about the error store
type StoreStats struct {
	TotalErrors      int              `json:"total_errors"`
	UnresolvedErrors int              `json:"unresolved_errors"`
	UniqueCodes      int              `json:"unique_codes"`
	BySeverity       map[Severity]int `json:"by_severity"`
	ByCategory       map[string]int   `json:"by_category"`
}

// Close closes the database connection
func (s *ErrorStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// Path returns the database file path
func (s *ErrorStore) Path() string {
	return s.path
}

// Global error store
var globalStore *ErrorStore
var globalStoreMu sync.RWMutex

// SetGlobalStore sets the global error store
func SetGlobalStore(store *ErrorStore) {
	globalStoreMu.Lock()
	defer globalStoreMu.Unlock()
	globalStore = store
}

// GetGlobalStore returns the global error store
func GetGlobalStore() *ErrorStore {
	globalStoreMu.RLock()
	defer globalStoreMu.RUnlock()
	return globalStore
}

// GlobalStore stores an error in the global store
func GlobalStore(ctx context.Context, err *TracedError) error {
	store := GetGlobalStore()
	if store == nil {
		return fmt.Errorf("global error store not initialized")
	}
	return store.Store(ctx, err)
}

// GlobalQuery queries the global store
func GlobalQuery(ctx context.Context, q ErrorQuery) ([]StoredError, error) {
	store := GetGlobalStore()
	if store == nil {
		return nil, fmt.Errorf("global error store not initialized")
	}
	return store.Query(ctx, q)
}

// GlobalResolve resolves an error in the global store
func GlobalResolve(ctx context.Context, traceID, resolvedBy string) error {
	store := GetGlobalStore()
	if store == nil {
		return fmt.Errorf("global error store not initialized")
	}
	return store.Resolve(ctx, traceID, resolvedBy)
}

// GlobalStoreStats returns stats from the global store
func GlobalStoreStats(ctx context.Context) (StoreStats, error) {
	store := GetGlobalStore()
	if store == nil {
		return StoreStats{}, fmt.Errorf("global error store not initialized")
	}
	return store.Stats(ctx)
}
