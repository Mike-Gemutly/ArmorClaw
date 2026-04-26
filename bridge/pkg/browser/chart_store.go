package browser

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ChartMeta contains metadata supplied when saving a chart.
type ChartMeta struct {
	Domain             string `json:"domain"`
	Title              string `json:"title"`
	CreatedFromSession string `json:"created_from_session,omitempty"`
	ParentChartID      string `json:"parent_chart_id,omitempty"`
}

// ChartRecord wraps a NavChart together with its persistence metadata.
type ChartRecord struct {
	ChartID            string     `json:"chart_id"`
	Domain             string     `json:"domain"`
	Title              string     `json:"title"`
	Version            int        `json:"version"`
	Steps              string     `json:"steps"`
	Selectors          string     `json:"selectors"`
	Placeholders       string     `json:"placeholders"`
	RequiresApproval   bool       `json:"requires_approval"`
	Confidence         float64    `json:"confidence"`
	CreatedFromSession string     `json:"created_from_session,omitempty"`
	CreatedAt          int64      `json:"created_at"`
	LastUsedAt         *int64     `json:"last_used_at,omitempty"`
	SuccessCount       int        `json:"success_count"`
	FailureCount       int        `json:"failure_count"`
	ParentChartID      string     `json:"parent_chart_id,omitempty"`
	NavChart           *NavChart  `json:"nav_chart,omitempty"`
}

// ChartStore defines the persistence interface for learned navigation charts.
type ChartStore interface {
	// SaveChart serialises a NavChart and persists it. Returns the generated chart_id.
	SaveChart(ctx context.Context, chart NavChart, meta ChartMeta) (string, error)
	// FindForDomain returns charts matching a domain, ordered by confidence descending.
	FindForDomain(ctx context.Context, domain string, limit int) ([]ChartRecord, error)
	// RecordOutcome adjusts confidence counters based on execution outcome.
	RecordOutcome(ctx context.Context, chartID string, success bool) error
	// GetChart retrieves a single chart by ID.
	GetChart(ctx context.Context, chartID string) (*ChartRecord, error)
	// DeleteChart removes a chart by ID.
	DeleteChart(ctx context.Context, chartID string) error
}

// SQLiteChartStore implements ChartStore using the secretary SQLite database.
type SQLiteChartStore struct {
	db *sql.DB
}

// NewSQLiteChartStore creates a new SQLiteChartStore.
// The caller must ensure the learned_charts table already exists (via initSchema).
func NewSQLiteChartStore(db *sql.DB) *SQLiteChartStore {
	return &SQLiteChartStore{db: db}
}

// SaveChart serialises a NavChart into the learned_charts table.
func (s *SQLiteChartStore) SaveChart(ctx context.Context, chart NavChart, meta ChartMeta) (string, error) {
	chartID := uuid.New().String()

	steps, err := json.Marshal(chart.ActionMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal steps: %w", err)
	}

	selectors := extractSelectors(chart)
	placeholders := extractPlaceholders(chart)

	requiresApproval := 0
	if hasApprovalActions(chart) {
		requiresApproval = 1
	}

	now := time.Now().UnixMilli()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO learned_charts (
			chart_id, domain, title, version, steps, selectors, placeholders,
			requires_approval, confidence, created_from_session, created_at,
			last_used_at, success_count, failure_count, parent_chart_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chartID, meta.Domain, meta.Title, chart.Version,
		string(steps), selectors, placeholders,
		requiresApproval, 0.5, nullIfEmpty(meta.CreatedFromSession),
		now, nil, 0, 0, nullIfEmpty(meta.ParentChartID),
	)
	if err != nil {
		return "", fmt.Errorf("failed to save chart: %w", err)
	}

	return chartID, nil
}

// FindForDomain returns charts for the given domain, ordered by confidence.
func (s *SQLiteChartStore) FindForDomain(ctx context.Context, domain string, limit int) ([]ChartRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT chart_id, domain, title, version, steps, selectors, placeholders,
			requires_approval, confidence, created_from_session, created_at,
			last_used_at, success_count, failure_count, parent_chart_id
		FROM learned_charts
		WHERE domain = ?
		ORDER BY confidence DESC
		LIMIT ?`, domain, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query charts for domain %q: %w", domain, err)
	}
	defer rows.Close()

	var records []ChartRecord
	for rows.Next() {
		rec, err := scanChartRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *rec)
	}

	return records, rows.Err()
}

// RecordOutcome updates a chart's confidence based on execution outcome.
func (s *SQLiteChartStore) RecordOutcome(ctx context.Context, chartID string, success bool) error {
	var current float64
	var successes, failures int

	err := s.db.QueryRowContext(ctx, `
		SELECT confidence, success_count, failure_count
		FROM learned_charts WHERE chart_id = ?`, chartID,
	).Scan(&current, &successes, &failures)
	if err != nil {
		return fmt.Errorf("failed to find chart %s: %w", chartID, err)
	}

	if success {
		successes++
		current = current + 0.1
		if current > 1.0 {
			current = 1.0
		}
	} else {
		failures++
		current = current - 0.2
		if current < 0.0 {
			current = 0.0
		}
	}

	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
		UPDATE learned_charts
		SET confidence = ?, success_count = ?, failure_count = ?, last_used_at = ?
		WHERE chart_id = ?`, current, successes, failures, now, chartID,
	)
	if err != nil {
		return fmt.Errorf("failed to update chart outcome: %w", err)
	}

	return nil
}

// GetChart retrieves a single chart by ID.
func (s *SQLiteChartStore) GetChart(ctx context.Context, chartID string) (*ChartRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chart_id, domain, title, version, steps, selectors, placeholders,
			requires_approval, confidence, created_from_session, created_at,
			last_used_at, success_count, failure_count, parent_chart_id
		FROM learned_charts
		WHERE chart_id = ?`, chartID,
	)

	rec, err := scanChartRecordRow(row)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart %s: %w", chartID, err)
	}
	return rec, nil
}

// DeleteChart removes a chart by ID.
func (s *SQLiteChartStore) DeleteChart(ctx context.Context, chartID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM learned_charts WHERE chart_id = ?`, chartID)
	if err != nil {
		return fmt.Errorf("failed to delete chart %s: %w", chartID, err)
	}
	return nil
}

func scanChartRecord(rows *sql.Rows) (*ChartRecord, error) {
	var rec ChartRecord
	var requiresApproval int
	var lastUsedAt sql.NullInt64
	var createdFromSession, parentChartID sql.NullString

	err := rows.Scan(
		&rec.ChartID, &rec.Domain, &rec.Title, &rec.Version,
		&rec.Steps, &rec.Selectors, &rec.Placeholders,
		&requiresApproval, &rec.Confidence, &createdFromSession,
		&rec.CreatedAt, &lastUsedAt,
		&rec.SuccessCount, &rec.FailureCount, &parentChartID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan chart record: %w", err)
	}

	rec.RequiresApproval = requiresApproval != 0
	if createdFromSession.Valid {
		rec.CreatedFromSession = createdFromSession.String
	}
	if parentChartID.Valid {
		rec.ParentChartID = parentChartID.String
	}
	if lastUsedAt.Valid {
		rec.LastUsedAt = &lastUsedAt.Int64
	}

	return &rec, nil
}

func scanChartRecordRow(row *sql.Row) (*ChartRecord, error) {
	var rec ChartRecord
	var requiresApproval int
	var lastUsedAt sql.NullInt64
	var createdFromSession, parentChartID sql.NullString

	err := row.Scan(
		&rec.ChartID, &rec.Domain, &rec.Title, &rec.Version,
		&rec.Steps, &rec.Selectors, &rec.Placeholders,
		&requiresApproval, &rec.Confidence, &createdFromSession,
		&rec.CreatedAt, &lastUsedAt,
		&rec.SuccessCount, &rec.FailureCount, &parentChartID,
	)
	if err != nil {
		return nil, err
	}

	rec.RequiresApproval = requiresApproval != 0
	if createdFromSession.Valid {
		rec.CreatedFromSession = createdFromSession.String
	}
	if parentChartID.Valid {
		rec.ParentChartID = parentChartID.String
	}
	if lastUsedAt.Valid {
		rec.LastUsedAt = &lastUsedAt.Int64
	}

	return &rec, nil
}

func extractSelectors(chart NavChart) string {
	var selectors []string
	for _, action := range chart.ActionMap {
		if action.Selector != nil {
			selectors = append(selectors, action.Selector.PrimaryCSS)
			if action.Selector.SecondaryXPath != "" {
				selectors = append(selectors, action.Selector.SecondaryXPath)
			}
			if action.Selector.FallbackJS != "" {
				selectors = append(selectors, action.Selector.FallbackJS)
			}
		}
	}
	b, _ := json.Marshal(selectors)
	return string(b)
}

func extractPlaceholders(chart NavChart) string {
	var phs []string
	for _, action := range chart.ActionMap {
		if action.ActionType == ActionInput && action.Value != "" {
			phs = append(phs, action.Value)
		}
	}
	b, _ := json.Marshal(phs)
	return string(b)
}

func hasApprovalActions(chart NavChart) bool {
	for _, action := range chart.ActionMap {
		if action.ActionType == ActionInput {
			return true
		}
	}
	return false
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
