// Package budget provides tests for the budget tracker
package budget

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBudgetTracker(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   10.00,
		MonthlyLimitUSD: 100.00,
		AlertThreshold:  80.0,
		HardStop:        true,
	}

	tracker := NewBudgetTracker(config, "")

	if tracker == nil {
		t.Fatal("NewBudgetTracker returned nil")
	}

	if tracker.GetDailyLimit() != 10.00 {
		t.Errorf("Expected daily limit 10.00, got %.2f", tracker.GetDailyLimit())
	}

	if tracker.GetMonthlyLimit() != 100.00 {
		t.Errorf("Expected monthly limit 100.00, got %.2f", tracker.GetMonthlyLimit())
	}
}

func TestRecordUsage(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   10.00,
		MonthlyLimitUSD: 100.00,
		HardStop:        false,
	}

	tracker := NewBudgetTracker(config, "")

	record := UsageRecord{
		SessionID:    "test-session-1",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	err := tracker.RecordUsage(record)
	if err != nil {
		t.Errorf("RecordUsage failed: %v", err)
	}

	// Check usage was recorded
	dailyUsage := tracker.GetDailyUsage()
	if dailyUsage == 0 {
		t.Error("Daily usage should be > 0")
	}

	sessionUsage := tracker.GetSessionUsage("test-session-1")
	if sessionUsage == 0 {
		t.Error("Session usage should be > 0")
	}
}

func TestDailyLimitEnforcement(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   0.01, // Very low limit for testing
		MonthlyLimitUSD: 100.00,
		HardStop:        true,
	}

	tracker := NewBudgetTracker(config, "")

	// First record should succeed (cost: 1500/1M * $0.50 = $0.00075)
	record1 := UsageRecord{
		SessionID:    "test-session-1",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	err := tracker.RecordUsage(record1)
	if err != nil {
		t.Errorf("First record should succeed: %v", err)
	}

	// Second record should fail due to daily limit (cost: 19K/1M * $0.50 = $0.0095, total = $0.01025 > $0.01)
	record2 := UsageRecord{
		SessionID:    "test-session-2",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  14000,
		OutputTokens: 5000,
	}

	err = tracker.RecordUsage(record2)
	if err == nil {
		t.Error("Second record should fail due to daily limit")
	}
}

func TestMonthlyLimitEnforcement(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   100.00,
		MonthlyLimitUSD: 0.01, // Very low limit for testing
		HardStop:        true,
	}

	tracker := NewBudgetTracker(config, "")

	record := UsageRecord{
		SessionID:    "test-session-1",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  20000,
		OutputTokens: 10000,
	}

	err := tracker.RecordUsage(record)
	if err == nil {
		t.Error("Record should fail due to monthly limit")
	}
}

func TestCanStartSession(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   1.00,
		MonthlyLimitUSD: 10.00,
		HardStop:        true,
	}

	tracker := NewBudgetTracker(config, "")

	// Should be able to start initially
	err := tracker.CanStartSession()
	if err != nil {
		t.Errorf("Should be able to start session: %v", err)
	}

	// Record usage that hits the limit
	record := UsageRecord{
		SessionID:    "test-session-1",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  2000000, // 2M tokens = $1.00
		OutputTokens: 0,
	}

	tracker.RecordUsage(record)

	// Should not be able to start new session
	err = tracker.CanStartSession()
	if err == nil {
		t.Error("Should not be able to start session after hitting limit")
	}
}

func TestUsageTracking(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   100.00,
		MonthlyLimitUSD: 1000.00,
		HardStop:        false,
	}

	tracker := NewBudgetTracker(config, "")

	// Record multiple sessions
	records := []UsageRecord{
		{
			SessionID:    "session-1",
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			InputTokens:  1000,
			OutputTokens: 500,
		},
		{
			SessionID:    "session-2",
			Provider:     "anthropic",
			Model:        "claude-3-haiku",
			InputTokens:  2000,
			OutputTokens: 1000,
		},
	}

	for _, record := range records {
		if err := tracker.RecordUsage(record); err != nil {
			t.Errorf("RecordUsage failed: %v", err)
		}
	}

	// Check session-specific usage
	session1Usage := tracker.GetSessionUsage("session-1")
	if session1Usage == 0 {
		t.Error("Session 1 usage should be > 0")
	}

	session2Usage := tracker.GetSessionUsage("session-2")
	if session2Usage == 0 {
		t.Error("Session 2 usage should be > 0")
	}

	// Total daily usage should be sum of both
	dailyUsage := tracker.GetDailyUsage()
	if dailyUsage != session1Usage+session2Usage {
		t.Errorf("Daily usage %.4f != sum of sessions %.4f", dailyUsage, session1Usage+session2Usage)
	}
}

func TestGetCost(t *testing.T) {
	tracker := NewBudgetTracker(BudgetConfig{}, "")

	// Test default costs
	gpt35Cost := tracker.GetCost("gpt-3.5-turbo")
	if gpt35Cost != 0.50 {
		t.Errorf("Expected gpt-3.5-turbo cost 0.50, got %.2f", gpt35Cost)
	}

	// Test unknown model returns default
	unknownCost := tracker.GetCost("unknown-model")
	if unknownCost != 0.001 {
		t.Errorf("Expected unknown model cost 0.001, got %.4f", unknownCost)
	}
}

func TestSetCost(t *testing.T) {
	tracker := NewBudgetTracker(BudgetConfig{}, "")

	// Set custom cost
	tracker.SetCost("custom-model", 5.00)

	cost := tracker.GetCost("custom-model")
	if cost != 5.00 {
		t.Errorf("Expected custom model cost 5.00, got %.2f", cost)
	}
}

func TestReset(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   100.00,
		MonthlyLimitUSD: 1000.00,
		HardStop:        false,
	}

	tracker := NewBudgetTracker(config, "")

	// Record some usage
	record := UsageRecord{
		SessionID:    "test-session",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	tracker.RecordUsage(record)

	if tracker.GetDailyUsage() == 0 {
		t.Error("Should have recorded usage")
	}

	// Reset
	tracker.Reset()

	if tracker.GetDailyUsage() != 0 {
		t.Error("Daily usage should be 0 after reset")
	}

	if tracker.GetSessionUsage("test-session") != 0 {
		t.Error("Session usage should be 0 after reset")
	}

	history := tracker.GetUsageHistory()
	if len(history) != 0 {
		t.Error("History should be empty after reset")
	}
}

func TestGetUsageHistory(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   100.00,
		MonthlyLimitUSD: 1000.00,
		HardStop:        false,
	}

	tracker := NewBudgetTracker(config, "")

	// Record multiple usage events
	for i := 0; i < 5; i++ {
		record := UsageRecord{
			SessionID:    "test-session",
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			InputTokens:  1000,
			OutputTokens: 500,
		}
		tracker.RecordUsage(record)
	}

	history := tracker.GetUsageHistory()
	if len(history) != 5 {
		t.Errorf("Expected 5 history entries, got %d", len(history))
	}

	// Verify history is a copy (modifying it shouldn't affect tracker)
	history[0].SessionID = "modified"
	newHistory := tracker.GetUsageHistory()
	if newHistory[0].SessionID == "modified" {
		t.Error("GetUsageHistory should return a copy")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   10.00,
		MonthlyLimitUSD: 100.00,
		HardStop:        false,
	}

	// Create temp directory for state
	tempDir := t.TempDir()

	tracker := NewBudgetTracker(config, tempDir)

	// Record some usage
	record := UsageRecord{
		SessionID:    "test-session",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	tracker.RecordUsage(record)
	expectedDailyUsage := tracker.GetDailyUsage()

	// Save state
	err := tracker.SaveState()
	if err != nil {
		t.Errorf("SaveState failed: %v", err)
	}

	// Create new tracker and load state
	tracker2 := NewBudgetTracker(config, tempDir)
	err = tracker2.LoadState()
	if err != nil {
		t.Errorf("LoadState failed: %v", err)
	}

	// Verify state was loaded
	loadedDailyUsage := tracker2.GetDailyUsage()
	if loadedDailyUsage != expectedDailyUsage {
		t.Errorf("Expected daily usage %.4f, got %.4f", expectedDailyUsage, loadedDailyUsage)
	}

	loadedSessionUsage := tracker2.GetSessionUsage("test-session")
	if loadedSessionUsage == 0 {
		t.Error("Session usage should be loaded")
	}
}

func TestSaveStateCreatesDirectory(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   10.00,
		MonthlyLimitUSD: 100.00,
		HardStop:        false,
	}

	// Use a path that doesn't exist
	tempDir := filepath.Join(os.TempDir(), "armorclaw-test", time.Now().Format("20060102-150405"))

	tracker := NewBudgetTracker(config, tempDir)

	// Record some usage
	record := UsageRecord{
		SessionID:    "test-session",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	tracker.RecordUsage(record)

	// Save should create directory
	err := tracker.SaveState()
	if err != nil {
		t.Errorf("SaveState failed: %v", err)
	}

	// Verify state file exists
	stateFile := filepath.Join(tempDir, "budget_state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Cleanup
	os.RemoveAll(tempDir)
}

func TestCustomProviderCosts(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   10.00,
		MonthlyLimitUSD: 100.00,
		HardStop:        false,
		ProviderCosts: map[string]float64{
			"custom-model": 25.00,
		},
	}

	tracker := NewBudgetTracker(config, "")

	// Check custom cost was applied
	cost := tracker.GetCost("custom-model")
	if cost != 25.00 {
		t.Errorf("Expected custom model cost 25.00, got %.2f", cost)
	}

	// Check default costs still work
	gpt35Cost := tracker.GetCost("gpt-3.5-turbo")
	if gpt35Cost != 0.50 {
		t.Errorf("Expected gpt-3.5-turbo cost 0.50, got %.2f", gpt35Cost)
	}
}

func TestConcurrentUsageRecording(t *testing.T) {
	config := BudgetConfig{
		DailyLimitUSD:   1000.00,
		MonthlyLimitUSD: 10000.00,
		HardStop:        false,
	}

	tracker := NewBudgetTracker(config, "")

	done := make(chan bool)

	// Launch multiple goroutines recording usage
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				record := UsageRecord{
					SessionID:    "session-" + string(rune('0'+id)),
					Provider:     "openai",
					Model:        "gpt-3.5-turbo",
					InputTokens:  100,
					OutputTokens: 50,
				}
				tracker.RecordUsage(record)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without deadlock or race condition, test passed
	dailyUsage := tracker.GetDailyUsage()
	if dailyUsage == 0 {
		t.Error("Should have recorded usage from concurrent operations")
	}
}
