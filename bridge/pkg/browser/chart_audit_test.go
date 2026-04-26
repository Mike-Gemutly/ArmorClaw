package browser

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func setupAuditDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	schema := `
		CREATE TABLE IF NOT EXISTS chart_audit (
			event_id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			chart_id TEXT NOT NULL,
			details TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_chart ON chart_audit(chart_id);
		CREATE INDEX IF NOT EXISTS idx_audit_type ON chart_audit(event_type);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create audit schema: %v", err)
	}
	return db
}

func queryAuditRows(t *testing.T, db *sql.DB, eventType string) []map[string]interface{} {
	t.Helper()
	rows, err := db.Query(`SELECT event_type, chart_id, details FROM chart_audit WHERE event_type = ?`, eventType)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var et, cid, detailsJSON string
		if err := rows.Scan(&et, &cid, &detailsJSON); err != nil {
			t.Fatalf("scan audit row: %v", err)
		}
		var details map[string]interface{}
		if err := json.Unmarshal([]byte(detailsJSON), &details); err != nil {
			t.Fatalf("unmarshal details: %v", err)
		}
		results = append(results, map[string]interface{}{
			"event_type": et,
			"chart_id":   cid,
			"details":    details,
		})
	}
	return results
}

func TestAuditCreated(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	err := auditor.AuditCreated(ctx, "chart-1", "example.com", 5, "sess-abc")
	if err != nil {
		t.Fatalf("AuditCreated: %v", err)
	}

	rows := queryAuditRows(t, db, AuditEventCreated)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	d := rows[0]
	if d["chart_id"] != "chart-1" {
		t.Errorf("expected chart_id=chart-1, got %v", d["chart_id"])
	}
	details := d["details"].(map[string]interface{})
	if details["domain"] != "example.com" {
		t.Errorf("expected domain=example.com, got %v", details["domain"])
	}
	if details["action_count"] != float64(5) {
		t.Errorf("expected action_count=5, got %v", details["action_count"])
	}
	if details["session_id"] != "sess-abc" {
		t.Errorf("expected session_id=sess-abc, got %v", details["session_id"])
	}
}

func TestAuditUpdated(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	err := auditor.AuditUpdated(ctx, "chart-2", 0.8, true)
	if err != nil {
		t.Fatalf("AuditUpdated: %v", err)
	}

	rows := queryAuditRows(t, db, AuditEventUpdated)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	details := rows[0]["details"].(map[string]interface{})
	if details["confidence"] != 0.8 {
		t.Errorf("expected confidence=0.8, got %v", details["confidence"])
	}
	if details["outcome"] != true {
		t.Errorf("expected outcome=true, got %v", details["outcome"])
	}
}

func TestAuditReplayed(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	err := auditor.AuditReplayed(ctx, "chart-3", "job-42", 8, 3, 2, 1)
	if err != nil {
		t.Fatalf("AuditReplayed: %v", err)
	}

	rows := queryAuditRows(t, db, AuditEventReplayed)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	details := rows[0]["details"].(map[string]interface{})
	if details["job_id"] != "job-42" {
		t.Errorf("expected job_id=job-42, got %v", details["job_id"])
	}
	if details["actions_executed"] != float64(8) {
		t.Errorf("expected actions_executed=8, got %v", details["actions_executed"])
	}
	if details["approvals_needed"] != float64(3) {
		t.Errorf("expected approvals_needed=3, got %v", details["approvals_needed"])
	}
	if details["approvals_granted"] != float64(2) {
		t.Errorf("expected approvals_granted=2, got %v", details["approvals_granted"])
	}
	if details["approvals_denied"] != float64(1) {
		t.Errorf("expected approvals_denied=1, got %v", details["approvals_denied"])
	}
}

func TestAuditRejected(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	err := auditor.AuditRejected(ctx, "chart-4", "confidence_below_threshold")
	if err != nil {
		t.Fatalf("AuditRejected: %v", err)
	}

	rows := queryAuditRows(t, db, AuditEventRejected)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	details := rows[0]["details"].(map[string]interface{})
	if details["reason"] != "confidence_below_threshold" {
		t.Errorf("expected reason=confidence_below_threshold, got %v", details["reason"])
	}
}

func TestAuditDeleted(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	err := auditor.AuditDeleted(ctx, "chart-5", "example.com")
	if err != nil {
		t.Fatalf("AuditDeleted: %v", err)
	}

	rows := queryAuditRows(t, db, AuditEventDeleted)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	details := rows[0]["details"].(map[string]interface{})
	if details["domain"] != "example.com" {
		t.Errorf("expected domain=example.com, got %v", details["domain"])
	}
}

func TestAuditMultipleEvents(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	auditor.AuditCreated(ctx, "chart-x", "test.com", 3, "sess-1")
	auditor.AuditReplayed(ctx, "chart-x", "job-1", 3, 1, 1, 0)
	auditor.AuditUpdated(ctx, "chart-x", 0.7, true)
	auditor.AuditDeleted(ctx, "chart-x", "test.com")

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM chart_audit WHERE chart_id = ?`, "chart-x").Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 audit rows for chart-x, got %d", count)
	}
}

func TestAuditNoPIIInDetails(t *testing.T) {
	db := setupAuditDB(t)
	defer db.Close()
	auditor := NewChartAuditor(db)
	ctx := context.Background()

	auditor.AuditCreated(ctx, "chart-pii", "secure-site.com", 2, "sess-sensitive")
	auditor.AuditReplayed(ctx, "chart-pii", "job-secret", 2, 1, 0, 1)
	auditor.AuditRejected(ctx, "chart-pii", "pii_detected_in_selectors")
	auditor.AuditDeleted(ctx, "chart-pii", "secure-site.com")

	rows, err := db.Query(`SELECT details FROM chart_audit WHERE chart_id = ?`, "chart-pii")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	piiPatterns := []string{"password", "email", "ssn", "credit_card", "phone", "address"}
	for rows.Next() {
		var detailsJSON string
		if err := rows.Scan(&detailsJSON); err != nil {
			t.Fatalf("scan: %v", err)
		}
		for _, pattern := range piiPatterns {
			if jsonStrContains(detailsJSON, pattern) {
				t.Errorf("potential PII pattern %q found in audit details: %s", pattern, detailsJSON)
			}
		}
	}
}

func jsonStrContains(s, substr string) bool {
	return len(s) >= len(substr) && containsFold(s, substr)
}

func containsFold(s, substr string) bool {
	ls := len(s)
	lsub := len(substr)
	for i := 0; i <= ls-lsub; i++ {
		match := true
		for j := 0; j < lsub; j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
