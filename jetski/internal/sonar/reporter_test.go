package sonar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/armorclaw/jetski/internal/subprocess"
)

// TestNewReporter tests Reporter creation
func TestNewReporter(t *testing.T) {
	buffer := NewCircularBuffer(10)
	outputDir := "/tmp/test-wreckage"
	zigEngine := subprocess.NewProcessManager()

	reporter := NewReporter(buffer, outputDir, zigEngine)

	if reporter == nil {
		t.Fatal("Reporter should not be nil")
	}

	if reporter.outputDir != outputDir {
		t.Errorf("Expected outputDir %s, got %s", outputDir, reporter.outputDir)
	}

	if reporter.buffer != buffer {
		t.Errorf("Buffer not properly set")
	}
}

// TestTriggerWreckage tests the TriggerWreckage function
func TestTriggerWreckage(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Setup
	buffer := NewCircularBuffer(10)
	sessionID := "test-session-123"
	targetURI := "https://github.com/user/repo"

	// Add some frames to buffer
	for i := 0; i < 5; i++ {
		frame := CDPFrame{
			Method:    "Test.method",
			Params:    []byte(`{"index":` + string(rune('0'+i)) + `}`),
			SessionID: sessionID,
			Timestamp: time.Now(),
		}
		buffer.Push(frame)
	}

	// Create reporter
	zigEngine := subprocess.NewProcessManager()
	reporter := NewReporter(buffer, tempDir, zigEngine)

	// Trigger wreckage
	failedSelector := Selector{
		PrimaryCSS: "#submit-button",
		Tier:       1,
	}

	report, err := reporter.TriggerWreckage(failedSelector, sessionID, targetURI)

	if err != nil {
		t.Fatalf("TriggerWreckage failed: %v", err)
	}

	// Verify report
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, report.SessionID)
	}

	if report.TargetURI != targetURI {
		t.Errorf("Expected TargetURI %s, got %s", targetURI, report.TargetURI)
	}

	if report.TargetHostname != "github.com" {
		t.Errorf("Expected TargetHostname 'github.com', got '%s'", report.TargetHostname)
	}

	if len(report.CDPHistory) != 5 {
		t.Errorf("Expected 5 CDP frames, got %d", len(report.CDPHistory))
	}

	if report.DOMSnapshot == "" {
		t.Error("DOMSnapshot should not be empty (even if placeholder)")
	}
}

// TestTriggerWreckage_FileOutput tests that files are created correctly
func TestTriggerWreckage_FileOutput(t *testing.T) {
	tempDir := t.TempDir()

	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	sessionID := "test-session-file"
	targetURI := "https://example.com"
	failedSelector := Selector{
		PrimaryCSS: "#button",
		Tier:       1,
	}

	// Track some interactions
	reporter.TrackInteraction(sessionID, 1)
	reporter.TrackInteraction(sessionID, 1)
	reporter.TrackInteraction(sessionID, 2)

	// Trigger wreckage
	report, err := reporter.TriggerWreckage(failedSelector, sessionID, targetURI)
	if err != nil {
		t.Fatalf("TriggerWreckage failed: %v", err)
	}

	// Verify health score is calculated
	if report.SelectorHealth == 0 {
		t.Error("Health score should be calculated")
	}

	if report.TotalInteractions != 3 {
		t.Errorf("Expected TotalInteractions 3, got %d", report.TotalInteractions)
	}

	// Verify file was created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	jsonFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			jsonFiles++
			// Verify file is valid JSON
			data, err := os.ReadFile(filepath.Join(tempDir, entry.Name()))
			if err != nil {
				t.Errorf("Failed to read wreck file: %v", err)
				continue
			}

			var wreck WreckageReport
			if err := json.Unmarshal(data, &wreck); err != nil {
				t.Errorf("Invalid JSON in wreck file: %v", err)
			}
		}
	}

	if jsonFiles != 1 {
		t.Errorf("Expected 1 wreck file, got %d", jsonFiles)
	}
}

// TestTrackInteraction tests selector interaction tracking
func TestTrackInteraction(t *testing.T) {
	tempDir := t.TempDir()
	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	sessionID := "test-session-track"

	// Track tier 1 (primary)
	reporter.TrackInteraction(sessionID, 1)
	reporter.TrackInteraction(sessionID, 1)

	// Track tier 2 (secondary)
	reporter.TrackInteraction(sessionID, 2)

	// Track tier 3 (fallback)
	reporter.TrackInteraction(sessionID, 3)
	reporter.TrackInteraction(sessionID, 3)

	// Verify metrics
	metrics := reporter.GetMetrics(sessionID)

	if metrics.PrimaryCount != 2 {
		t.Errorf("Expected PrimaryCount 2, got %d", metrics.PrimaryCount)
	}

	if metrics.SecondaryCount != 1 {
		t.Errorf("Expected SecondaryCount 1, got %d", metrics.SecondaryCount)
	}

	if metrics.FallbackCount != 2 {
		t.Errorf("Expected FallbackCount 2, got %d", metrics.FallbackCount)
	}

	if metrics.Total != 5 {
		t.Errorf("Expected Total 5, got %d", metrics.Total)
	}
}

// TestGetMetrics tests getting metrics for different sessions
func TestGetMetrics(t *testing.T) {
	tempDir := t.TempDir()
	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	session1 := "session-1"
	session2 := "session-2"

	// Track interactions for session 1
	reporter.TrackInteraction(session1, 1)
	reporter.TrackInteraction(session1, 2)

	// Track interactions for session 2
	reporter.TrackInteraction(session2, 3)
	reporter.TrackInteraction(session2, 3)
	reporter.TrackInteraction(session2, 3)

	// Verify session 1 metrics
	metrics1 := reporter.GetMetrics(session1)
	if metrics1.Total != 2 {
		t.Errorf("Session 1: Expected Total 2, got %d", metrics1.Total)
	}

	// Verify session 2 metrics
	metrics2 := reporter.GetMetrics(session2)
	if metrics2.Total != 3 {
		t.Errorf("Session 2: Expected Total 3, got %d", metrics2.Total)
	}

	// Unknown session should return empty metrics
	metrics3 := reporter.GetMetrics("unknown-session")
	if metrics3.Total != 0 {
		t.Errorf("Unknown session should have Total 0, got %d", metrics3.Total)
	}
}

// TestClearSessionMetrics tests clearing session metrics
func TestClearSessionMetrics(t *testing.T) {
	tempDir := t.TempDir()
	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	sessionID := "test-session-clear"

	// Track some interactions
	reporter.TrackInteraction(sessionID, 1)
	reporter.TrackInteraction(sessionID, 2)
	reporter.TrackInteraction(sessionID, 3)

	// Verify metrics exist
	metrics := reporter.GetMetrics(sessionID)
	if metrics.Total != 3 {
		t.Errorf("Expected Total 3 before clear, got %d", metrics.Total)
	}

	// Clear metrics
	reporter.ClearSessionMetrics(sessionID)

	// Verify metrics are cleared
	metrics = reporter.GetMetrics(sessionID)
	if metrics.Total != 0 {
		t.Errorf("Expected Total 0 after clear, got %d", metrics.Total)
	}
}

// TestListWreckageReports tests listing wreck files
func TestListWreckageReports(t *testing.T) {
	tempDir := t.TempDir()
	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	// Create some wreck files
	for i := 0; i < 3; i++ {
		sessionID := "test-session-" + string(rune('0'+i))
		reporter.TrackInteraction(sessionID, 1)
		_, _ = reporter.TriggerWreckage(Selector{Tier: 1}, sessionID, "https://example.com")
	}

	// List reports
	reports, err := reporter.ListWreckageReports()
	if err != nil {
		t.Fatalf("ListWreckageReports failed: %v", err)
	}

	if len(reports) != 3 {
		t.Errorf("Expected 3 wreck reports, got %d", len(reports))
	}

	// Verify all files are JSON
	for _, reportPath := range reports {
		if filepath.Ext(reportPath) != ".json" {
			t.Errorf("Expected .json extension, got %s", filepath.Ext(reportPath))
		}
	}
}

// TestTriggerWreckage_InvalidURI tests handling of invalid URIs
func TestTriggerWreckage_InvalidURI(t *testing.T) {
	tempDir := t.TempDir()
	buffer := NewCircularBuffer(10)
	reporter := NewReporter(buffer, tempDir, nil)

	sessionID := "test-session-invalid"
	targetURI := "invalid-uri-without-scheme"
	failedSelector := Selector{Tier: 1}

	// Should return error for invalid URI
	_, err := reporter.TriggerWreckage(failedSelector, sessionID, targetURI)

	if err == nil {
		t.Error("Expected error for invalid URI, got nil")
	}
}
