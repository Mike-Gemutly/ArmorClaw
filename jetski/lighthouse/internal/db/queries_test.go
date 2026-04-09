package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/armorclaw/lighthouse/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = RunMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestListCharts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	chart1 := &models.Chart{
		Domain:    "example.com",
		Version:   "1.0.0",
		Author:    "alice",
		ChartData: `{"name": "chart1"}`,
		Signature: "sig1",
		Blessed:   false,
		Downloads: 10,
		CreatedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
	}

	chart2 := &models.Chart{
		Domain:    "example.com",
		Version:   "1.1.0",
		Author:    "bob",
		ChartData: `{"name": "chart2"}`,
		Signature: "sig2",
		Blessed:   true,
		Downloads: 20,
		CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}

	err := InsertChart(db, chart1)
	if err != nil {
		t.Fatalf("Failed to insert chart1: %v", err)
	}

	err = InsertChart(db, chart2)
	if err != nil {
		t.Fatalf("Failed to insert chart2: %v", err)
	}

	charts, err := ListChartsByDomain(db, "example.com")
	if err != nil {
		t.Fatalf("ListChartsByDomain failed: %v", err)
	}

	if len(charts) != 2 {
		t.Errorf("Expected 2 charts, got %d", len(charts))
	}

	if charts[0].Version != "1.1.0" || charts[1].Version != "1.0.0" {
		t.Errorf("Charts not ordered correctly by created_at DESC: got %v, %v", charts[0].Version, charts[1].Version)
	}

	charts, err = ListChartsByDomain(db, "other.com")
	if err != nil {
		t.Fatalf("ListChartsByDomain failed for other domain: %v", err)
	}

	if len(charts) != 0 {
		t.Errorf("Expected 0 charts for other domain, got %d", len(charts))
	}
}

func TestInsertGetChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	chart := &models.Chart{
		Domain:    "example.com",
		Version:   "2.0.0",
		Author:    "charlie",
		ChartData: `{"name": "chart3"}`,
		Signature: "sig3",
		Blessed:   true,
		Downloads: 5,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	err := InsertChart(db, chart)
	if err != nil {
		t.Fatalf("InsertChart failed: %v", err)
	}

	if chart.ID == 0 {
		t.Errorf("Expected chart ID to be set, got 0")
	}

	retrieved, err := GetChartByVersion(db, "example.com", "2.0.0")
	if err != nil {
		t.Fatalf("GetChartByVersion failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve chart, got nil")
	}

	if retrieved.Domain != chart.Domain {
		t.Errorf("Domain mismatch: expected %s, got %s", chart.Domain, retrieved.Domain)
	}

	if retrieved.Version != chart.Version {
		t.Errorf("Version mismatch: expected %s, got %s", chart.Version, retrieved.Version)
	}

	if retrieved.Author != chart.Author {
		t.Errorf("Author mismatch: expected %s, got %s", chart.Author, retrieved.Author)
	}

	if retrieved.ChartData != chart.ChartData {
		t.Errorf("ChartData mismatch: expected %s, got %s", chart.ChartData, retrieved.ChartData)
	}

	if retrieved.Signature != chart.Signature {
		t.Errorf("Signature mismatch: expected %s, got %s", chart.Signature, retrieved.Signature)
	}

	if retrieved.Blessed != chart.Blessed {
		t.Errorf("Blessed mismatch: expected %v, got %v", chart.Blessed, retrieved.Blessed)
	}

	if retrieved.Downloads != chart.Downloads {
		t.Errorf("Downloads mismatch: expected %d, got %d", chart.Downloads, retrieved.Downloads)
	}

	nonExistent, err := GetChartByVersion(db, "example.com", "99.99.99")
	if err != nil {
		t.Fatalf("GetChartByVersion failed for non-existent chart: %v", err)
	}

	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent chart, got %v", nonExistent)
	}
}

func TestGetLatestBlessedChart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now()

	oldBlessed := &models.Chart{
		Domain:    "example.com",
		Version:   "1.0.0",
		Author:    "alice",
		ChartData: `{"name": "old"}`,
		Signature: "sig1",
		Blessed:   true,
		Downloads: 10,
		CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
	}

	newBlessed := &models.Chart{
		Domain:    "example.com",
		Version:   "2.0.0",
		Author:    "bob",
		ChartData: `{"name": "new"}`,
		Signature: "sig2",
		Blessed:   true,
		Downloads: 20,
		CreatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
	}

	unblessed := &models.Chart{
		Domain:    "example.com",
		Version:   "3.0.0",
		Author:    "charlie",
		ChartData: `{"name": "unblessed"}`,
		Signature: "sig3",
		Blessed:   false,
		Downloads: 30,
		CreatedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
	}

	InsertChart(db, oldBlessed)
	InsertChart(db, newBlessed)
	InsertChart(db, unblessed)

	latestBlessed, err := GetLatestBlessedChart(db, "example.com")
	if err != nil {
		t.Fatalf("GetLatestBlessedChart failed: %v", err)
	}

	if latestBlessed == nil {
		t.Fatal("Expected to retrieve blessed chart, got nil")
	}

	if latestBlessed.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0 (latest blessed), got %s", latestBlessed.Version)
	}

	if latestBlessed.Author != "bob" {
		t.Errorf("Expected author bob, got %s", latestBlessed.Author)
	}

	noBlessed, err := GetLatestBlessedChart(db, "other.com")
	if err != nil {
		t.Fatalf("GetLatestBlessedChart failed for other domain: %v", err)
	}

	if noBlessed != nil {
		t.Errorf("Expected nil for domain with no blessed charts, got %v", noBlessed)
	}
}
