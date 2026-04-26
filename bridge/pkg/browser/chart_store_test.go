package browser

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS learned_charts (
			chart_id TEXT PRIMARY KEY,
			domain TEXT NOT NULL,
			title TEXT NOT NULL,
			version INTEGER DEFAULT 1,
			steps TEXT NOT NULL,
			selectors TEXT NOT NULL,
			placeholders TEXT NOT NULL,
			requires_approval INTEGER DEFAULT 0,
			confidence REAL DEFAULT 0.5,
			created_from_session TEXT,
			created_at INTEGER NOT NULL,
			last_used_at INTEGER,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			parent_chart_id TEXT,
			CHECK (confidence >= 0.0 AND confidence <= 1.0)
		);
		CREATE INDEX IF NOT EXISTS idx_charts_domain ON learned_charts(domain);
		CREATE INDEX IF NOT EXISTS idx_charts_confidence ON learned_charts(confidence DESC);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func sampleChart() NavChart {
	return NavChart{
		Version:      1,
		TargetDomain: "example.com",
		Metadata:     ChartMetadata{GeneratedBy: "test", SessionID: "sess-1"},
		ActionMap: map[string]ChartAction{
			"navigate_home": {
				ActionType: ActionNavigate,
				URL:        "https://example.com",
			},
			"fill_search": {
				ActionType: ActionInput,
				Selector: &ChartSelector{
					PrimaryCSS:     "#search",
					SecondaryXPath: "//input[@id='search']",
				},
				Value: "{{query}}",
			},
			"click_submit": {
				ActionType: ActionClick,
				Selector: &ChartSelector{
					PrimaryCSS: "button[type='submit']",
				},
			},
		},
	}
}

func TestSaveChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	chart := sampleChart()
	meta := ChartMeta{
		Domain:             "example.com",
		Title:              "Search on Example",
		CreatedFromSession: "sess-1",
	}

	id, err := store.SaveChart(ctx, chart, meta)
	if err != nil {
		t.Fatalf("SaveChart: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty chart_id")
	}
}

func TestGetChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	chart := sampleChart()
	meta := ChartMeta{Domain: "example.com", Title: "Test Chart"}

	id, _ := store.SaveChart(ctx, chart, meta)

	rec, err := store.GetChart(ctx, id)
	if err != nil {
		t.Fatalf("GetChart: %v", err)
	}
	if rec.ChartID != id {
		t.Errorf("expected chart_id %s, got %s", id, rec.ChartID)
	}
	if rec.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", rec.Domain)
	}
	if rec.Title != "Test Chart" {
		t.Errorf("expected title 'Test Chart', got %s", rec.Title)
	}
	if rec.Confidence != 0.5 {
		t.Errorf("expected default confidence 0.5, got %f", rec.Confidence)
	}
	if !rec.RequiresApproval {
		t.Error("expected requires_approval=true because chart has input actions")
	}
	if rec.CreatedAt == 0 {
		t.Error("expected non-zero created_at")
	}
}

func TestFindForDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	chart1 := sampleChart()
	chart2 := sampleChart()
	chart2.TargetDomain = "other.com"

	store.SaveChart(ctx, chart1, ChartMeta{Domain: "example.com", Title: "Chart A"})
	store.SaveChart(ctx, chart2, ChartMeta{Domain: "other.com", Title: "Chart B"})
	store.SaveChart(ctx, chart1, ChartMeta{Domain: "example.com", Title: "Chart C"})

	results, err := store.FindForDomain(ctx, "example.com", 10)
	if err != nil {
		t.Fatalf("FindForDomain: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Domain != "example.com" {
			t.Errorf("expected domain example.com, got %s", r.Domain)
		}
	}

	otherResults, _ := store.FindForDomain(ctx, "other.com", 10)
	if len(otherResults) != 1 {
		t.Fatalf("expected 1 result for other.com, got %d", len(otherResults))
	}
}

func TestRecordOutcomeSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	id, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Test"})

	err := store.RecordOutcome(ctx, id, true)
	if err != nil {
		t.Fatalf("RecordOutcome: %v", err)
	}

	rec, _ := store.GetChart(ctx, id)
	if rec.SuccessCount != 1 {
		t.Errorf("expected success_count=1, got %d", rec.SuccessCount)
	}
	if rec.Confidence != 0.6 {
		t.Errorf("expected confidence=0.6, got %f", rec.Confidence)
	}
	if rec.LastUsedAt == nil {
		t.Error("expected last_used_at to be set")
	}
}

func TestRecordOutcomeFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	id, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Test"})

	store.RecordOutcome(ctx, id, false)

	rec, _ := store.GetChart(ctx, id)
	if rec.FailureCount != 1 {
		t.Errorf("expected failure_count=1, got %d", rec.FailureCount)
	}
	if rec.Confidence != 0.3 {
		t.Errorf("expected confidence=0.3, got %f", rec.Confidence)
	}
}

func TestRecordOutcomeConfidenceBounds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	id, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Test"})

	for i := 0; i < 10; i++ {
		store.RecordOutcome(ctx, id, false)
	}
	rec, _ := store.GetChart(ctx, id)
	if rec.Confidence != 0.0 {
		t.Errorf("expected confidence floored at 0.0, got %f", rec.Confidence)
	}

	for i := 0; i < 20; i++ {
		store.RecordOutcome(ctx, id, true)
	}
	rec, _ = store.GetChart(ctx, id)
	if rec.Confidence != 1.0 {
		t.Errorf("expected confidence capped at 1.0, got %f", rec.Confidence)
	}
}

func TestDeleteChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	id, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Test"})

	err := store.DeleteChart(ctx, id)
	if err != nil {
		t.Fatalf("DeleteChart: %v", err)
	}

	_, err = store.GetChart(ctx, id)
	if err == nil {
		t.Error("expected error after deleting chart")
	}
}

func TestDeleteNonExistentChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	err := store.DeleteChart(ctx, "nonexistent-id")
	if err != nil {
		t.Errorf("deleting non-existent chart should not error, got: %v", err)
	}
}

func TestGetNonExistentChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	_, err := store.GetChart(ctx, "nonexistent-id")
	if err == nil {
		t.Error("expected error for non-existent chart")
	}
}

func TestSaveChartWithParentAndSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	parentID, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{
		Domain:             "example.com",
		Title:              "Parent",
		CreatedFromSession: "sess-parent",
	})

	childID, err := store.SaveChart(ctx, sampleChart(), ChartMeta{
		Domain:             "example.com",
		Title:              "Child",
		CreatedFromSession: "sess-child",
		ParentChartID:      parentID,
	})
	if err != nil {
		t.Fatalf("SaveChart child: %v", err)
	}

	child, _ := store.GetChart(ctx, childID)
	if child.ParentChartID != parentID {
		t.Errorf("expected parent_chart_id=%s, got %s", parentID, child.ParentChartID)
	}
	if child.CreatedFromSession != "sess-child" {
		t.Errorf("expected created_from_session=sess-child, got %s", child.CreatedFromSession)
	}
}

func TestFindForDomainOrderByConfidence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteChartStore(db)
	ctx := context.Background()

	id1, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Low"})
	id2, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "High"})
	id3, _ := store.SaveChart(ctx, sampleChart(), ChartMeta{Domain: "example.com", Title: "Mid"})

	store.RecordOutcome(ctx, id2, true)
	store.RecordOutcome(ctx, id2, true)
	store.RecordOutcome(ctx, id3, true)

	results, _ := store.FindForDomain(ctx, "example.com", 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].ChartID != id2 {
		t.Errorf("expected highest confidence chart first, got %s", results[0].Title)
	}
	if results[1].ChartID != id3 {
		t.Errorf("expected mid confidence chart second, got %s", results[1].Title)
	}
	if results[2].ChartID != id1 {
		t.Errorf("expected lowest confidence chart last, got %s", results[2].Title)
	}
}
