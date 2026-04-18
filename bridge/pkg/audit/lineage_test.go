package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLineageTracker_RecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	lt := NewLineageTracker(filepath.Join(dir, "lineage.json"))

	ctx := context.Background()

	err := lt.LogArtifactLineage(ctx, "prompt", "src-001", "llm_generate", "document", "out-001", "agent-1")
	if err != nil {
		t.Fatalf("LogArtifactLineage: %v", err)
	}

	chain, err := lt.GetArtifactLineage(ctx, "out-001")
	if err != nil {
		t.Fatalf("GetArtifactLineage: %v", err)
	}

	if len(chain) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(chain))
	}

	if chain[0].SourceID != "src-001" {
		t.Errorf("expected SourceID src-001, got %s", chain[0].SourceID)
	}
	if chain[0].OutputID != "out-001" {
		t.Errorf("expected OutputID out-001, got %s", chain[0].OutputID)
	}
	if chain[0].Transformation != "llm_generate" {
		t.Errorf("expected Transformation llm_generate, got %s", chain[0].Transformation)
	}
}

func TestLineageTracker_ChainOfThree(t *testing.T) {
	dir := t.TempDir()
	lt := NewLineageTracker(filepath.Join(dir, "lineage.json"))

	ctx := context.Background()

	if err := lt.LogArtifactLineage(ctx, "prompt", "A", "generate", "draft", "B", "agent-1"); err != nil {
		t.Fatalf("log 1: %v", err)
	}
	if err := lt.LogArtifactLineage(ctx, "draft", "B", "refine", "report", "C", "agent-1"); err != nil {
		t.Fatalf("log 2: %v", err)
	}
	if err := lt.LogArtifactLineage(ctx, "report", "C", "format", "pdf", "D", "agent-1"); err != nil {
		t.Fatalf("log 3: %v", err)
	}

	chain, err := lt.GetArtifactLineage(ctx, "D")
	if err != nil {
		t.Fatalf("GetArtifactLineage: %v", err)
	}

	if len(chain) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(chain))
	}

	// Chronological order: A→B first, B→C second, C→D third
	expected := []struct{ src, out string }{
		{"A", "B"},
		{"B", "C"},
		{"C", "D"},
	}
	for i, exp := range expected {
		if chain[i].SourceID != exp.src || chain[i].OutputID != exp.out {
			t.Errorf("chain[%d]: expected %s→%s, got %s→%s", i, exp.src, exp.out, chain[i].SourceID, chain[i].OutputID)
		}
	}
}

func TestLineageTracker_NoLineage(t *testing.T) {
	dir := t.TempDir()
	lt := NewLineageTracker(filepath.Join(dir, "lineage.json"))

	chain, err := lt.GetArtifactLineage(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetArtifactLineage: %v", err)
	}

	if len(chain) != 0 {
		t.Fatalf("expected 0 entries for unknown ID, got %d", len(chain))
	}
}

func TestLineageTracker_CircularProtection(t *testing.T) {
	dir := t.TempDir()
	lt := NewLineageTracker(filepath.Join(dir, "lineage.json"))

	ctx := context.Background()

	if err := lt.LogArtifactLineage(ctx, "type", "A", "loop", "type", "B", "agent-1"); err != nil {
		t.Fatalf("log 1: %v", err)
	}
	if err := lt.LogArtifactLineage(ctx, "type", "B", "loop", "type", "A", "agent-1"); err != nil {
		t.Fatalf("log 2: %v", err)
	}

	chain, err := lt.GetArtifactLineage(ctx, "A")
	if err != nil {
		t.Fatalf("GetArtifactLineage: %v", err)
	}

	// Should trace A→B (A is output of B→A), then stop when it hits B again
	// since B was already visited as a source in the B→A step
	if len(chain) > 3 {
		t.Fatalf("circular protection failed: got %d entries (expected <=3)", len(chain))
	}
}

func TestGovernanceEventTypes(t *testing.T) {
	expected := map[EventType]string{
		EventCapabilityRequested:  "capability_requested",
		EventCapabilityGranted:    "capability_granted",
		EventCapabilityDenied:     "capability_denied",
		EventCapabilityDeferred:   "capability_deferred",
		EventBrokerIntercept:      "broker_intercept",
		EventArtifactCreated:      "artifact_created",
		EventArtifactTransformed:  "artifact_transformed",
		EventArtifactLineageQuery: "artifact_lineage_query",
		EventTeamCreated:          "team_created",
		EventTeamDissolved:        "team_dissolved",
		EventMemberAdded:          "member_added",
		EventMemberRemoved:        "member_removed",
		EventRoleAssigned:         "role_assigned",
	}

	for constant, value := range expected {
		if string(constant) != value {
			t.Errorf("EventType %s: expected %q, got %q", constant, value, string(constant))
		}
	}
}

func TestLineageTracker_FilePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")

	lt := NewLineageTracker(path)
	ctx := context.Background()

	if err := lt.LogArtifactLineage(ctx, "prompt", "X", "transform", "doc", "Y", "agent-1"); err != nil {
		t.Fatalf("LogArtifactLineage: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("lineage file was not created")
	}

	lt2 := NewLineageTracker(path)
	chain, err := lt2.GetArtifactLineage(ctx, "Y")
	if err != nil {
		t.Fatalf("GetArtifactLineage from reloaded tracker: %v", err)
	}

	if len(chain) != 1 {
		t.Fatalf("expected 1 entry after reload, got %d", len(chain))
	}
	if chain[0].SourceID != "X" {
		t.Errorf("expected SourceID X, got %s", chain[0].SourceID)
	}
}
