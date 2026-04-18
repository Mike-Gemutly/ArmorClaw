package capability

import (
	"context"
	"errors"
	"testing"
)

const validTeamPlanJSON = `{
  "goal": "Book a flight to NYC",
  "subtasks": [
    {
      "id": "st-1",
      "description": "Search for flights",
      "assigned_role": "browser_specialist",
      "depends_on": []
    },
    {
      "id": "st-2",
      "description": "Fill payment form",
      "assigned_role": "form_filler",
      "depends_on": ["st-1"]
    }
  ],
  "role_assignments": [
    {"role_name": "browser_specialist", "subtask_ids": ["st-1"]},
    {"role_name": "form_filler", "subtask_ids": ["st-2"]}
  ]
}`

func newTestParser(llm LLMCaller) *StructuredOutputParser {
	return NewStructuredOutputParser(StructuredOutputConfig{LLM: llm})
}

func TestParse_ValidJSON(t *testing.T) {
	p := newTestParser(nil)
	out, err := p.Parse(context.Background(), validTeamPlanJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Goal != "Book a flight to NYC" {
		t.Errorf("goal = %q, want %q", out.Goal, "Book a flight to NYC")
	}
	if len(out.Subtasks) != 2 {
		t.Fatalf("subtasks = %d, want 2", len(out.Subtasks))
	}
	if out.Subtasks[0].ID != "st-1" {
		t.Errorf("subtask[0].ID = %q, want %q", out.Subtasks[0].ID, "st-1")
	}
	if out.Subtasks[1].AssignedRole != "form_filler" {
		t.Errorf("subtask[1].AssignedRole = %q, want %q", out.Subtasks[1].AssignedRole, "form_filler")
	}
	if len(out.RoleAssignments) != 2 {
		t.Fatalf("role_assignments = %d, want 2", len(out.RoleAssignments))
	}
	if out.RoleAssignments[0].RoleName != "browser_specialist" {
		t.Errorf("role_assignment[0].RoleName = %q, want %q", out.RoleAssignments[0].RoleName, "browser_specialist")
	}
}

func TestParse_ValidJSONInCodeFence(t *testing.T) {
	p := newTestParser(nil)
	fenced := "```json\n" + validTeamPlanJSON + "\n```"
	out, err := p.Parse(context.Background(), fenced)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Goal != "Book a flight to NYC" {
		t.Errorf("goal = %q, want %q", out.Goal, "Book a flight to NYC")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	p := newTestParser(nil)
	_, err := p.Parse(context.Background(), "not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParse_MissingGoal(t *testing.T) {
	p := newTestParser(nil)
	noGoal := `{
		"subtasks": [{"id":"st-1","description":"do it","assigned_role":"team_lead"}],
		"role_assignments": [{"role_name":"team_lead","subtask_ids":["st-1"]}]
	}`
	_, err := p.Parse(context.Background(), noGoal)
	if err == nil {
		t.Fatal("expected error for missing goal")
	}
}

func TestParse_EmptySubtasks(t *testing.T) {
	p := newTestParser(nil)
	noSubtasks := `{
		"goal": "do something",
		"subtasks": [],
		"role_assignments": [{"role_name":"team_lead","subtask_ids":[]}]
	}`
	_, err := p.Parse(context.Background(), noSubtasks)
	if err == nil {
		t.Fatal("expected error for empty subtasks")
	}
}

func TestParse_EmptyRoleAssignments(t *testing.T) {
	p := newTestParser(nil)
	noRoles := `{
		"goal": "do something",
		"subtasks": [{"id":"st-1","description":"do it","assigned_role":"team_lead"}],
		"role_assignments": []
	}`
	_, err := p.Parse(context.Background(), noRoles)
	if err == nil {
		t.Fatal("expected error for empty role_assignments")
	}
}

func TestParseWithRetries_FirstTrySuccess(t *testing.T) {
	calls := 0
	llm := func(ctx context.Context, prompt string) (string, error) {
		calls++
		return validTeamPlanJSON, nil
	}
	p := newTestParser(llm)
	out, err := p.ParseWithRetries(context.Background(), "plan a trip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("LLM calls = %d, want 1", calls)
	}
	if out.Goal != "Book a flight to NYC" {
		t.Errorf("goal = %q, want %q", out.Goal, "Book a flight to NYC")
	}
}

func TestParseWithRetries_RetryThenSuccess(t *testing.T) {
	calls := 0
	llm := func(ctx context.Context, prompt string) (string, error) {
		calls++
		if calls < 2 {
			return "bad json", nil
		}
		return validTeamPlanJSON, nil
	}
	p := newTestParser(llm)
	out, err := p.ParseWithRetries(context.Background(), "plan a trip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("LLM calls = %d, want 2", calls)
	}
	if out.Goal != "Book a flight to NYC" {
		t.Errorf("goal = %q, want %q", out.Goal, "Book a flight to NYC")
	}
}

func TestParseWithRetries_AllRetriesFail(t *testing.T) {
	calls := 0
	llm := func(ctx context.Context, prompt string) (string, error) {
		calls++
		return "always bad", nil
	}
	p := newTestParser(llm)
	out, err := p.ParseWithRetries(context.Background(), "plan a trip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("LLM calls = %d, want 3", calls)
	}
	if out.Goal != "plan a trip" {
		t.Errorf("fallback goal = %q, want %q", out.Goal, "plan a trip")
	}
	if len(out.Subtasks) != 3 {
		t.Errorf("fallback subtasks = %d, want 3", len(out.Subtasks))
	}
	if len(out.RoleAssignments) != 3 {
		t.Errorf("fallback role_assignments = %d, want 3", len(out.RoleAssignments))
	}
}

func TestParseWithRetries_ContextCancelled(t *testing.T) {
	llm := func(ctx context.Context, prompt string) (string, error) {
		return "", nil
	}
	p := newTestParser(llm)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.ParseWithRetries(ctx, "plan a trip")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestDefaultTeamPlan(t *testing.T) {
	p := newTestParser(nil)
	plan := p.DefaultTeamPlan("search for hotels")

	if plan.Goal != "search for hotels" {
		t.Errorf("goal = %q, want %q", plan.Goal, "search for hotels")
	}
	if len(plan.Subtasks) != 3 {
		t.Fatalf("subtasks = %d, want 3", len(plan.Subtasks))
	}
	roles := make(map[string]bool)
	for _, ra := range plan.RoleAssignments {
		roles[ra.RoleName] = true
	}
	for _, want := range []string{"team_lead", "browser_specialist", "form_filler"} {
		if !roles[want] {
			t.Errorf("missing role %q in role_assignments", want)
		}
	}
	if err := plan.Validate(); err != nil {
		t.Errorf("default plan should validate: %v", err)
	}
}

func TestNewStructuredOutputParser_DefaultMaxRetries(t *testing.T) {
	p := NewStructuredOutputParser(StructuredOutputConfig{
		LLM:        func(ctx context.Context, prompt string) (string, error) { return "", nil },
		MaxRetries: 0,
	})
	if p.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", p.maxRetries)
	}
}

func TestNewStructuredOutputParser_CustomMaxRetries(t *testing.T) {
	p := NewStructuredOutputParser(StructuredOutputConfig{
		LLM:        func(ctx context.Context, prompt string) (string, error) { return "", nil },
		MaxRetries: 5,
	})
	if p.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want 5", p.maxRetries)
	}
}
