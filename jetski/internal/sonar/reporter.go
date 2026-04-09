package sonar

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jetski-browser/internal/subprocess"
)

// Reporter generates and saves WreckageReports to disk
// Implements atomic write pattern for safe concurrent operations
type Reporter struct {
	buffer     *CircularBuffer
	outputDir  string
	zigEngine  *subprocess.ProcessManager
	mu         sync.Mutex
	selectorMu sync.Mutex
	metrics    map[string]SelectorMetrics // Per-session metrics
}

// NewReporter creates a new Reporter instance
func NewReporter(buffer *CircularBuffer, outputDir string, zigEngine *subprocess.ProcessManager) *Reporter {
	return &Reporter{
		buffer:    buffer,
		outputDir: outputDir,
		zigEngine: zigEngine,
		metrics:   make(map[string]SelectorMetrics),
	}
}

// TriggerWreckage generates a complete WreckageReport and saves it to disk
// This is called when a selector failure occurs, capturing the "Black Box" data
func (r *Reporter) TriggerWreckage(failedSelector Selector, sessionID string, targetURI string) (*WreckageReport, error) {
	// Create the wreckage report
	report, err := NewWreckageReport(sessionID, targetURI, r.buffer, failedSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to create wreckage report: %w", err)
	}

	// Calculate and set the health score from metrics
	r.selectorMu.Lock()
	metrics, exists := r.metrics[sessionID]
	if !exists {
		metrics = SelectorMetrics{Total: 0}
	}
	r.selectorMu.Unlock()

	report.CalculateAndSetHealthScore(metrics)

	// Capture DOM snapshot (placeholder - would use Zig engine in Sprint B)
	// For now, we'll store a placeholder indicating Zig integration is pending
	report.DOMSnapshot = "// Zig DOM snapshot pending - Sprint B integration"

	// Save the report to disk
	if err := r.saveWreckage(report); err != nil {
		return nil, fmt.Errorf("failed to save wreckage report: %w", err)
	}

	// Log with nautical branding
	log.Printf("🕯️ Sonar wreckage report saved: %s (Health: %.2f) - %s",
		report.SessionID,
		report.SelectorHealth,
		report.GetHealthStatus(),
	)

	return report, nil
}

// saveWreckage writes the report to disk using atomic write pattern
// Creates a temporary file first, then renames to ensure consistency
func (r *Reporter) saveWreckage(report *WreckageReport) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename: {session-id}-{unix-timestamp}.json
	filename := fmt.Sprintf("%s-%d.json", report.SessionID, time.Now().Unix())
	tempPath := filepath.Join(r.outputDir, "."+filename+".tmp")
	finalPath := filepath.Join(r.outputDir, filename)

	// Marshal report with indentation for readability
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to temporary file atomically
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename to final filename
	if err := os.Rename(tempPath, finalPath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// TrackInteraction records a selector interaction for health score calculation
// Should be called for every selector attempt (success or failure)
func (r *Reporter) TrackInteraction(sessionID string, tier int) {
	r.selectorMu.Lock()
	defer r.selectorMu.Unlock()

	if _, exists := r.metrics[sessionID]; !exists {
		r.metrics[sessionID] = SelectorMetrics{}
	}

	metrics := r.metrics[sessionID]
	metrics.Total++

	switch tier {
	case 1:
		metrics.PrimaryCount++
	case 2:
		metrics.SecondaryCount++
	case 3:
		metrics.FallbackCount++
	}

	r.metrics[sessionID] = metrics
}

// GetMetrics returns the current selector metrics for a session
func (r *Reporter) GetMetrics(sessionID string) SelectorMetrics {
	r.selectorMu.Lock()
	defer r.selectorMu.Unlock()

	if metrics, exists := r.metrics[sessionID]; exists {
		return metrics
	}
	return SelectorMetrics{}
}

// ClearSessionMetrics removes metrics for a specific session
// Useful for testing or session cleanup
func (r *Reporter) ClearSessionMetrics(sessionID string) {
	r.selectorMu.Lock()
	defer r.selectorMu.Unlock()

	delete(r.metrics, sessionID)
}

// ListWreckageReports returns all saved wreckage report file paths
func (r *Reporter) ListWreckageReports() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(r.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read wreckage directory: %w", err)
	}

	var reports []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			reports = append(reports, filepath.Join(r.outputDir, entry.Name()))
		}
	}

	return reports, nil
}
