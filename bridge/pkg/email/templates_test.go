package email

import (
	"context"
	"strings"
	"testing"
)

func TestTeamTemplateManager_CreateAndGet(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	tmpl := EmailTeamTemplate{
		TeamID:          "team-1",
		Name:            "welcome",
		SubjectTemplate: "Welcome {{name}}!",
		BodyTemplate:    "Hi {{name}}, welcome to {{org}}.",
	}

	err := tm.CreateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	got, err := tm.GetTemplate(context.Background(), "team-1", "welcome")
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.TeamID != "team-1" {
		t.Errorf("TeamID: got %s", got.TeamID)
	}
	if got.Name != "welcome" {
		t.Errorf("Name: got %s", got.Name)
	}
	if got.ID == "" {
		t.Error("ID should be auto-generated")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestTeamTemplateManager_CreateDuplicate(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	tmpl := EmailTeamTemplate{TeamID: "team-1", Name: "greeting", SubjectTemplate: "Hi"}
	_ = tm.CreateTemplate(context.Background(), tmpl)

	err := tm.CreateTemplate(context.Background(), EmailTeamTemplate{
		TeamID: "team-1", Name: "greeting", SubjectTemplate: "Hello",
	})
	if err == nil {
		t.Error("expected error creating duplicate template")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
}

func TestTeamTemplateManager_CreateValidation(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	if err := tm.CreateTemplate(context.Background(), EmailTeamTemplate{Name: "x"}); err == nil {
		t.Error("expected error for missing team_id")
	}
	if err := tm.CreateTemplate(context.Background(), EmailTeamTemplate{TeamID: "t1"}); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestTeamTemplateManager_GetNonExistent(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	_, err := tm.GetTemplate(context.Background(), "team-1", "missing")
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestTeamTemplateManager_ListTemplates(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	templates := []EmailTeamTemplate{
		{TeamID: "team-A", Name: "welcome", SubjectTemplate: "Welcome"},
		{TeamID: "team-A", Name: "goodbye", SubjectTemplate: "Goodbye"},
		{TeamID: "team-B", Name: "welcome", SubjectTemplate: "Hello"},
	}
	for _, tmpl := range templates {
		_ = tm.CreateTemplate(context.Background(), tmpl)
	}

	teamA, err := tm.ListTemplates(context.Background(), "team-A")
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(teamA) != 2 {
		t.Errorf("team-A: expected 2 templates, got %d", len(teamA))
	}

	teamB, _ := tm.ListTemplates(context.Background(), "team-B")
	if len(teamB) != 1 {
		t.Errorf("team-B: expected 1 template, got %d", len(teamB))
	}

	teamC, _ := tm.ListTemplates(context.Background(), "team-C")
	if len(teamC) != 0 {
		t.Errorf("team-C: expected 0 templates, got %d", len(teamC))
	}
}

func TestTeamTemplateManager_RenderTemplate(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	tmpl := EmailTeamTemplate{
		TeamID:          "team-1",
		Name:            "invite",
		SubjectTemplate: "Invitation for {{name}} from {{org}}",
		BodyTemplate:    "Dear {{name}},\n\nYou are invited to join {{org}}.\n\nRegards,\n{{sender}}",
	}
	_ = tm.CreateTemplate(context.Background(), tmpl)

	rendered, err := tm.RenderTemplate(context.Background(), "team-1", "invite", map[string]string{
		"name":   "Alice",
		"org":    "Acme Corp",
		"sender": "Bob",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	if rendered.Subject != "Invitation for Alice from Acme Corp" {
		t.Errorf("subject: got %s", rendered.Subject)
	}
	if !strings.Contains(rendered.BodyText, "Dear Alice,") {
		t.Errorf("body should contain 'Dear Alice,': got %s", rendered.BodyText)
	}
	if !strings.Contains(rendered.BodyText, "join Acme Corp") {
		t.Errorf("body should contain 'join Acme Corp': got %s", rendered.BodyText)
	}
	if !strings.Contains(rendered.BodyText, "Regards,\nBob") {
		t.Errorf("body should contain 'Regards,\\nBob': got %s", rendered.BodyText)
	}
}

func TestTeamTemplateManager_RenderMissingVariable(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	tmpl := EmailTeamTemplate{
		TeamID:          "team-1",
		Name:            "partial",
		SubjectTemplate: "Hello {{name}}, your code is {{code}}",
		BodyTemplate:    "Welcome {{name}}. Missing: {{unknown_var}}",
	}
	_ = tm.CreateTemplate(context.Background(), tmpl)

	rendered, err := tm.RenderTemplate(context.Background(), "team-1", "partial", map[string]string{
		"name": "Eve",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	if !strings.Contains(rendered.Subject, "Eve") {
		t.Errorf("subject should contain Eve: %s", rendered.Subject)
	}
	if !strings.Contains(rendered.Subject, "{{code}}") {
		t.Error("missing variable should remain as-is in subject")
	}
	if !strings.Contains(rendered.BodyText, "{{unknown_var}}") {
		t.Error("missing variable should remain as-is in body")
	}
}

func TestTeamTemplateManager_RenderNonExistent(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	_, err := tm.RenderTemplate(context.Background(), "team-1", "missing", nil)
	if err == nil {
		t.Error("expected error rendering nonexistent template")
	}
}

func TestTeamTemplateManager_PreSetID(t *testing.T) {
	tm := NewTeamTemplateManager(TeamTemplateManagerConfig{})

	tmpl := EmailTeamTemplate{
		ID:              "custom-id-123",
		TeamID:          "team-1",
		Name:            "custom",
		SubjectTemplate: "Custom",
	}
	_ = tm.CreateTemplate(context.Background(), tmpl)

	got, _ := tm.GetTemplate(context.Background(), "team-1", "custom")
	if got.ID != "custom-id-123" {
		t.Errorf("ID should be preserved: got %s", got.ID)
	}
}

func TestGenerateID(t *testing.T) {
	id1, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	id2, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	if id1 == id2 {
		t.Error("two generated IDs should differ")
	}
	if len(id1) != 32 {
		t.Errorf("ID should be 32 hex chars, got %d", len(id1))
	}
}
