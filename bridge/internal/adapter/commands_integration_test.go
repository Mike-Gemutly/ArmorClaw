package adapter

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/armorclaw/bridge/pkg/skills"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func setupLearnedStore(t *testing.T) *skills.LearnedStore {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_pragma_key=test&_pragma_cipher_page_size=4096")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

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
		t.Fatalf("create schema: %v", err)
	}

	return skills.NewLearnedStore(db)
}

func newTestHandler(t *testing.T) *CommandHandler {
	t.Helper()
	store := setupLearnedStore(t)
	return NewCommandHandler(nil, nil, nil, store)
}

func TestAgentSkills_FormatsCorrectly(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.learnedStore.Save(skills.LearnedSkill{
		Name:            "search-flights",
		PatternType:     "web_browsing",
		PatternData:     `{"steps":["navigate","extract"]}`,
		TriggerKeywords: "search flights",
		Confidence:      0.85,
		SuccessCount:    3,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	resp, err := h.handleAgentSkills(context.Background(), "@user:test", []string{"agent_1"})
	if err != nil {
		t.Fatalf("handleAgentSkills: %v", err)
	}

	if !strings.Contains(resp, "📚 Learned Skills for agent_1:") {
		t.Errorf("expected header with agent_id, got: %s", resp)
	}
	if !strings.Contains(resp, "search-flights") {
		t.Errorf("expected skill name, got: %s", resp)
	}
	if !strings.Contains(resp, "0.85") {
		t.Errorf("expected confidence value, got: %s", resp)
	}
	if !strings.Contains(resp, "3 successful") {
		t.Errorf("expected success count, got: %s", resp)
	}
	if strings.Contains(resp, "pattern_data") || strings.Contains(resp, `{"steps"`) {
		t.Errorf("should NOT expose pattern data, got: %s", resp)
	}
}

func TestAgentSkills_EmptyList(t *testing.T) {
	h := newTestHandler(t)

	resp, err := h.handleAgentSkills(context.Background(), "@user:test", []string{"agent_99"})
	if err != nil {
		t.Fatalf("handleAgentSkills: %v", err)
	}

	if !strings.Contains(resp, "No learned skills yet for agent_99") {
		t.Errorf("expected empty message, got: %s", resp)
	}
}

func TestAgentSkills_MissingAgentID(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentSkills(context.Background(), "@user:test", nil)
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected usage error, got: %v", err)
	}
}

func TestAgentForgetSkill_Deletes(t *testing.T) {
	h := newTestHandler(t)

	saved, err := h.learnedStore.Save(skills.LearnedSkill{
		Name:            "search-hotels",
		PatternType:     "web_browsing",
		PatternData:     `{"steps":["navigate"]}`,
		TriggerKeywords: "search hotels",
		Confidence:      0.7,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	resp, err := h.handleAgentForgetSkill(context.Background(), "@user:test", []string{"agent_1", saved.ID})
	if err != nil {
		t.Fatalf("handleAgentForgetSkill: %v", err)
	}

	if !strings.Contains(resp, saved.ID) {
		t.Errorf("expected skill ID in response, got: %s", resp)
	}
	if !strings.Contains(resp, "Forgot skill:") {
		t.Errorf("expected confirmation, got: %s", resp)
	}

	list, _ := h.learnedStore.ListForAgent(10)
	if len(list) != 0 {
		t.Errorf("expected skill to be deleted, found %d", len(list))
	}
}

func TestAgentForgetSkill_MissingArgs(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentForgetSkill(context.Background(), "@user:test", []string{"agent_1"})
	if err == nil {
		t.Fatal("expected error for missing skill_id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected usage error, got: %v", err)
	}
}

func TestAgentForgetSkill_NoArgs(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentForgetSkill(context.Background(), "@user:test", nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestAgentSubcommand_Routes(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentSubcommand(context.Background(), "@user:test", []string{"skills", "agent_1"})
	if err != nil {
		t.Errorf("skills subcommand should route: %v", err)
	}

	_, err = h.handleAgentSubcommand(context.Background(), "@user:test", []string{"forget-skill", "agent_1", "sk_123"})
	if err != nil {
		t.Logf("forget-skill with nonexistent id returns: %v (ok)", err)
	}
}

func TestAgentSubcommand_UnknownSubcommand(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentSubcommand(context.Background(), "@user:test", []string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected unknown error, got: %v", err)
	}
}

func TestAgentSubcommand_NoSubcommand(t *testing.T) {
	h := newTestHandler(t)

	_, err := h.handleAgentSubcommand(context.Background(), "@user:test", nil)
	if err == nil {
		t.Fatal("expected error for no subcommand")
	}
}

func TestAgentSubcommand_NilStore(t *testing.T) {
	h := &CommandHandler{learnedStore: nil}

	_, err := h.handleAgentSubcommand(context.Background(), "@user:test", []string{"skills", "agent_1"})
	if err == nil {
		t.Fatal("expected error when learnedStore is nil")
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Errorf("expected 'not available' error, got: %v", err)
	}
}
