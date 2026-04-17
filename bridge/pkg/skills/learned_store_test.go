package skills

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS learned_skills (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE,
			description TEXT,
			source_task_id TEXT,
			source_template_id TEXT,
			pattern_type TEXT NOT NULL,
			pattern_data TEXT NOT NULL,
			trigger_keywords TEXT NOT NULL,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			last_used_at INTEGER,
			created_at INTEGER NOT NULL,
			confidence REAL DEFAULT 0.5
		);
		CREATE INDEX IF NOT EXISTS idx_learned_confidence ON learned_skills(confidence);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db
}

func testSkill(name, keywords string, confidence float64) LearnedSkill {
	return LearnedSkill{
		Name:            name,
		Description:     "test skill " + name,
		PatternType:     "web_browsing",
		PatternData:     `{"steps":["navigate","extract"]}`,
		TriggerKeywords: keywords,
		Confidence:      confidence,
	}
}

// Test 1: Save and FindForTask round-trip
func TestSaveAndFindForTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("search-flights", "search flights booking", 0.8))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	results, err := store.FindForTask("search flights to NYC", 10)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "search-flights" {
		t.Fatalf("expected name 'search-flights', got %q", results[0].Name)
	}
}

// Test 2: Confidence threshold filtering
func TestConfidenceThresholdFiltering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	_, err := store.Save(testSkill("low-conf", "low confidence", 0.3))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	_, err = store.Save(testSkill("high-conf", "high confidence", 0.7))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	results, err := store.FindForTask("confidence test", 10)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (high-conf only), got %d", len(results))
	}
	if results[0].Name != "high-conf" {
		t.Fatalf("expected 'high-conf', got %q", results[0].Name)
	}
}

// Test 3: RecordOutcome increases confidence on success
func TestRecordOutcomeSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("outcome-test", "outcome test", 0.5))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = store.RecordOutcome(saved.ID, true)
	if err != nil {
		t.Fatalf("RecordOutcome failed: %v", err)
	}

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Confidence != 0.6 {
		t.Fatalf("expected confidence 0.6, got %f", results[0].Confidence)
	}
	if results[0].SuccessCount != 1 {
		t.Fatalf("expected success_count 1, got %d", results[0].SuccessCount)
	}
}

// Test 4: RecordOutcome decreases confidence on failure
func TestRecordOutcomeFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("fail-test", "fail test", 0.5))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = store.RecordOutcome(saved.ID, false)
	if err != nil {
		t.Fatalf("RecordOutcome failed: %v", err)
	}

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if results[0].Confidence != 0.3 {
		t.Fatalf("expected confidence 0.3, got %f", results[0].Confidence)
	}
	if results[0].FailureCount != 1 {
		t.Fatalf("expected failure_count 1, got %d", results[0].FailureCount)
	}
}

// Test 5: RecordOutcome caps confidence at 1.0
func TestRecordOutcomeCapAtMax(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("cap-max", "cap max", 0.95))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		store.RecordOutcome(saved.ID, true)
	}

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if results[0].Confidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %f", results[0].Confidence)
	}
}

// Test 6: RecordOutcome floors confidence at 0.0
func TestRecordOutcomeFloorAtMin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("floor-min", "floor min", 0.1))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		store.RecordOutcome(saved.ID, false)
	}

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if results[0].Confidence != 0.0 {
		t.Fatalf("expected confidence 0.0, got %f", results[0].Confidence)
	}
}

// Test 7: Delete removes skill
func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	saved, err := store.Save(testSkill("delete-me", "delete test", 0.8))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = store.Delete(saved.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(results))
	}
}

// Test 8: ListForAgent returns skills ordered by confidence
func TestListForAgent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	_, _ = store.Save(testSkill("low", "low priority", 0.4))
	_, _ = store.Save(testSkill("high", "high priority", 0.9))
	_, _ = store.Save(testSkill("mid", "mid priority", 0.6))

	results, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Confidence < results[1].Confidence || results[1].Confidence < results[2].Confidence {
		t.Fatal("results not ordered by confidence DESC")
	}
}

// Test 9: Empty DB returns no results
func TestEmptyDBReturnsNoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	results, err := store.FindForTask("anything", 10)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results from empty DB, got %d", len(results))
	}

	listed, err := store.ListForAgent(10)
	if err != nil {
		t.Fatalf("ListForAgent failed: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected 0 results from empty DB, got %d", len(listed))
	}
}

// Test 10: Duplicate name handling
func TestDuplicateName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	_, err := store.Save(testSkill("unique-name", "unique", 0.7))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, err = store.Save(testSkill("unique-name", "duplicate", 0.8))
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

// Test 11: FindForTask ranks by keyword overlap
func TestFindForTaskKeywordRanking(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	_, _ = store.Save(testSkill("hotel-booking", "hotel booking reservation", 0.7))
	_, _ = store.Save(testSkill("flight-booking", "flight booking airline", 0.7))
	_, _ = store.Save(testSkill("weather-check", "weather forecast temperature", 0.7))

	results, err := store.FindForTask("book a flight to Paris", 10)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Name != "flight-booking" {
		t.Fatalf("expected 'flight-booking' ranked first, got %q", results[0].Name)
	}
}

// Test 12: FindForTask respects limit
func TestFindForTaskLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	for i := 0; i < 5; i++ {
		_, _ = store.Save(testSkill("skill-"+string(rune('A'+i)), "test keyword", 0.8))
	}

	results, err := store.FindForTask("test keyword", 2)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (limited), got %d", len(results))
	}
}

// TestSkillsFullLifecycle validates the complete feedback loop:
// save extracted skill → inject (FindForTask) → record outcome → verify confidence adjustment.
// Uses a LearnedSkill struct matching what ExtractFromResult produces for a 3-command sequence,
// since the CGO linker conflict between go-sqlcipher (via secretary→keystore) and go-sqlite3
// prevents testing extraction + store in the same binary.
func TestSkillsFullLifecycle(t *testing.T) {
	// Step 1: Create LearnedStore with in-memory SQLite.
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	// Step 2: Create a LearnedSkill matching ExtractFromResult output for 3 command_run events.
	// ExtractFromResult produces a command_sequence skill with Confidence=0.6, TriggerKeywords=taskDesc.
	extractedSkill := LearnedSkill{
		Name:             "skill_deploy-database_cmdseq",
		SourceTaskID:     "task-1",
		SourceTemplateID: "tpl-1",
		PatternType:      "command_sequence",
		PatternData:      `[{"command":"apt-get update","exit_code":0},{"command":"apt-get install -y postgresql","exit_code":0},{"command":"systemctl start postgresql","exit_code":0}]`,
		Confidence:       0.6,
		TriggerKeywords:  "deploy database",
	}

	// Step 3: Verify skill has correct pattern type.
	if extractedSkill.PatternType != "command_sequence" {
		t.Fatalf("expected pattern_type %q, got %q", "command_sequence", extractedSkill.PatternType)
	}

	// Step 4: Save the extracted skill to the store.
	saved, err := store.Save(extractedSkill)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected non-empty ID after save")
	}

	// Step 5: Inject — FindForTask should return the skill.
	injected, err := store.FindForTask("deploy database", 3)
	if err != nil {
		t.Fatalf("FindForTask after save failed: %v", err)
	}
	if len(injected) != 1 {
		t.Fatalf("expected 1 skill after save, got %d", len(injected))
	}
	if injected[0].ID != saved.ID {
		t.Errorf("expected skill ID %q, got %q", saved.ID, injected[0].ID)
	}

	// Step 6: Record a successful outcome.
	if err := store.RecordOutcome(saved.ID, true); err != nil {
		t.Fatalf("RecordOutcome(success) failed: %v", err)
	}

	// Step 7: Inject again — skill still present (confidence increased from 0.6 to 0.7).
	injected, err = store.FindForTask("deploy database", 3)
	if err != nil {
		t.Fatalf("FindForTask after success failed: %v", err)
	}
	if len(injected) != 1 {
		t.Fatalf("expected 1 skill after success, got %d", len(injected))
	}
	if injected[0].Confidence < 0.69 {
		t.Errorf("expected confidence >= 0.69 after success, got %f", injected[0].Confidence)
	}

	// Step 8: Record 5 failures to drop confidence below 0.4 threshold.
	// Starting from 0.7: 0.7 → 0.5 → 0.3 → 0.1 → 0.0 → 0.0 (floored)
	for i := 0; i < 5; i++ {
		if err := store.RecordOutcome(saved.ID, false); err != nil {
			t.Fatalf("RecordOutcome(failure %d) failed: %v", i+1, err)
		}
	}

	// Step 9: Verify skill is no longer returned (confidence < 0.4).
	injected, err = store.FindForTask("deploy database", 3)
	if err != nil {
		t.Fatalf("FindForTask after failures failed: %v", err)
	}
	if len(injected) != 0 {
		t.Fatalf("expected 0 skills after 5 failures (confidence should be < 0.4), got %d", len(injected))
	}
}

// TestSkillsConfidenceThresholdFiltering verifies that skills below the
// confidence threshold are excluded from FindForTask results.
func TestSkillsConfidenceThresholdFiltering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewLearnedStore(db)

	// Save a skill with confidence 0.5 (the default from ExtractFromResult for file_transform).
	saved, err := store.Save(LearnedSkill{
		Name:            "lifecycle-threshold-test",
		Description:     "test confidence threshold filtering",
		PatternType:     "command_sequence",
		PatternData:     `{"commands":["echo test"]}`,
		TriggerKeywords: "deploy database",
		Confidence:      0.5,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Record 3 failures: 0.5 → 0.3 → 0.1 → 0.0
	for i := 0; i < 3; i++ {
		if err := store.RecordOutcome(saved.ID, false); err != nil {
			t.Fatalf("RecordOutcome(failure %d) failed: %v", i+1, err)
		}
	}

	// FindForTask should return nothing — confidence is 0.0 (well below 0.4).
	results, err := store.FindForTask("deploy database", 5)
	if err != nil {
		t.Fatalf("FindForTask failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results (confidence < 0.4), got %d", len(results))
	}
}
