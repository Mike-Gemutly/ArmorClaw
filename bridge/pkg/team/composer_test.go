package team

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestCompose_ValidGoal(t *testing.T) {
	llmResponse := llmPlanResponse{
		Subtasks: []PlanSubtask{
			{
				ID:             "step1",
				Description:    "Browse example.com and extract data",
				AssignedRole:   "browser_specialist",
				DependsOn:      []string{},
				InputArtifact:  "",
				OutputArtifact: "BrowserResult",
			},
			{
				ID:             "step2",
				Description:    "Send extracted data via email",
				AssignedRole:   "email_clerk",
				DependsOn:      []string{"step1"},
				InputArtifact:  "BrowserResult",
				OutputArtifact: "EmailDraft",
			},
		},
	}
	raw, _ := json.Marshal(llmResponse)

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return string(raw), nil
		},
	})

	plan, err := composer.Compose(context.Background(), "Research and email results")
	if err != nil {
		t.Fatalf("Compose returned error: %v", err)
	}
	if plan.Goal != "Research and email results" {
		t.Errorf("Goal = %q, want %q", plan.Goal, "Research and email results")
	}
	if len(plan.Subtasks) != 2 {
		t.Fatalf("len(Subtasks) = %d, want 2", len(plan.Subtasks))
	}
	if plan.Subtasks[0].AssignedRole != "browser_specialist" {
		t.Errorf("Subtask[0].AssignedRole = %q, want browser_specialist", plan.Subtasks[0].AssignedRole)
	}
	if plan.Subtasks[1].AssignedRole != "email_clerk" {
		t.Errorf("Subtask[1].AssignedRole = %q, want email_clerk", plan.Subtasks[1].AssignedRole)
	}

	if len(plan.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2", len(plan.Members))
	}

	if len(plan.ArtifactFlows) != 1 {
		t.Fatalf("len(ArtifactFlows) = %d, want 1", len(plan.ArtifactFlows))
	}
	flow := plan.ArtifactFlows[0]
	if flow.FromSubtask != "step1" || flow.ToSubtask != "step2" || flow.ArtifactType != "BrowserResult" {
		t.Errorf("ArtifactFlow = %+v, want FromSubtask=step1 ToSubtask=step2 ArtifactType=BrowserResult", flow)
	}
}

func TestCompose_InvalidRole(t *testing.T) {
	llmResponse := llmPlanResponse{
		Subtasks: []PlanSubtask{
			{
				ID:             "step1",
				Description:    "Do something",
				AssignedRole:   "nonexistent_role",
				DependsOn:      []string{},
				InputArtifact:  "",
				OutputArtifact: "",
			},
		},
	}
	raw, _ := json.Marshal(llmResponse)

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return string(raw), nil
		},
	})

	_, err := composer.Compose(context.Background(), "Some goal")
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("error = %q, want mention of unknown role", err.Error())
	}
}

func TestCompose_CircularDependency(t *testing.T) {
	llmResponse := llmPlanResponse{
		Subtasks: []PlanSubtask{
			{
				ID:             "a",
				Description:    "Step A",
				AssignedRole:   "browser_specialist",
				DependsOn:      []string{"b"},
				InputArtifact:  "",
				OutputArtifact: "",
			},
			{
				ID:             "b",
				Description:    "Step B",
				AssignedRole:   "email_clerk",
				DependsOn:      []string{"a"},
				InputArtifact:  "",
				OutputArtifact: "",
			},
		},
	}
	raw, _ := json.Marshal(llmResponse)

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return string(raw), nil
		},
	})

	_, err := composer.Compose(context.Background(), "Cycle test")
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of circular", err.Error())
	}
}

func TestCompose_EmptyGoal(t *testing.T) {
	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return "{}", nil
		},
	})

	_, err := composer.Compose(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty goal, got nil")
	}
	if !strings.Contains(err.Error(), "goal must not be empty") {
		t.Errorf("error = %q, want mention of empty goal", err.Error())
	}

	_, err = composer.Compose(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only goal, got nil")
	}
}

func TestCompose_LLMError(t *testing.T) {
	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("LLM unavailable")
		},
	})

	_, err := composer.Compose(context.Background(), "Valid goal")
	if err == nil {
		t.Fatal("expected error from LLM failure, got nil")
	}
	if !strings.Contains(err.Error(), "llm call failed") {
		t.Errorf("error = %q, want mention of llm call failed", err.Error())
	}
}

func TestBuildCompositionPrompt(t *testing.T) {
	prompt := buildCompositionPrompt("Book a flight to NYC")

	if !strings.Contains(prompt, "Book a flight to NYC") {
		t.Error("prompt missing goal")
	}

	for _, role := range []string{"team_lead", "browser_specialist", "form_filler", "doc_analyst", "email_clerk", "supervisor"} {
		if !strings.Contains(prompt, role) {
			t.Errorf("prompt missing role %q", role)
		}
	}

	for _, artifact := range knownArtifactTypes {
		if !strings.Contains(prompt, artifact) {
			t.Errorf("prompt missing artifact type %q", artifact)
		}
	}

	if !strings.Contains(prompt, `"subtasks"`) {
		t.Error("prompt missing JSON schema")
	}
}

func TestValidatePlan_ValidPlan(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Do research",
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "Browse", AssignedRole: "browser_specialist", DependsOn: []string{}},
			{ID: "s2", Description: "Send email", AssignedRole: "email_clerk", DependsOn: []string{"s1"}},
		},
	}
	if err := validatePlan(plan); err != nil {
		t.Errorf("validatePlan returned error: %v", err)
	}
}

func TestValidatePlan_EmptySubtasks(t *testing.T) {
	plan := &TeamPlan{
		Goal:      "Do stuff",
		Subtasks:  []PlanSubtask{},
	}
	err := validatePlan(plan)
	if err == nil {
		t.Fatal("expected error for empty subtasks")
	}
	if !strings.Contains(err.Error(), "at least one subtask") {
		t.Errorf("error = %q, want mention of at least one subtask", err.Error())
	}
}

func TestValidatePlan_InvalidRole(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Bad role",
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "Hack", AssignedRole: "hacker", DependsOn: []string{}},
		},
	}
	err := validatePlan(plan)
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("error = %q, want mention of unknown role", err.Error())
	}
}

func TestValidatePlan_CircularDependency(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Cycle",
		Subtasks: []PlanSubtask{
			{ID: "a", Description: "A", AssignedRole: "browser_specialist", DependsOn: []string{"b"}},
			{ID: "b", Description: "B", AssignedRole: "email_clerk", DependsOn: []string{"a"}},
		},
	}
	err := validatePlan(plan)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of circular", err.Error())
	}
}

func TestValidatePlan_InvalidDependency(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Bad dep",
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "First", AssignedRole: "browser_specialist", DependsOn: []string{"nonexistent"}},
		},
	}
	err := validatePlan(plan)
	if err == nil {
		t.Fatal("expected error for invalid dependency reference")
	}
	if !strings.Contains(err.Error(), "unknown subtask") {
		t.Errorf("error = %q, want mention of unknown subtask", err.Error())
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain json", `{"subtasks":[]}`, `{"subtasks":[]}`},
		{"json fence", "```json\n{\"subtasks\":[]}\n```", `{"subtasks":[]}`},
		{"bare fence", "```\n{\"subtasks\":[]}\n```", `{"subtasks":[]}`},
		{"no fence with whitespace", `  {"subtasks":[]}  `, `{"subtasks":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("stripCodeFences(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompose_WithCodeFenceResponse(t *testing.T) {
	llmResponse := llmPlanResponse{
		Subtasks: []PlanSubtask{
			{
				ID:           "s1",
				Description:  "Analyze doc",
				AssignedRole: "doc_analyst",
				DependsOn:    []string{},
			},
		},
	}
	raw, _ := json.Marshal(llmResponse)
	fenced := "```json\n" + string(raw) + "\n```"

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return fenced, nil
		},
	})

	plan, err := composer.Compose(context.Background(), "Analyze document")
	if err != nil {
		t.Fatalf("Compose returned error: %v", err)
	}
	if len(plan.Subtasks) != 1 {
		t.Fatalf("len(Subtasks) = %d, want 1", len(plan.Subtasks))
	}
}

func TestNewTeamComposer_DefaultMaxRetries(t *testing.T) {
	c := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) { return "{}", nil },
	})
	if c.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", c.maxRetries)
	}
}

func TestNewTeamComposer_CustomMaxRetries(t *testing.T) {
	c := NewTeamComposer(TeamComposerConfig{
		LLM:        func(_ context.Context, _ string) (string, error) { return "{}", nil },
		MaxRetries: 5,
	})
	if c.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want 5", c.maxRetries)
	}
}

func TestValidatePlan_NilPlan(t *testing.T) {
	if err := validatePlan(nil); err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestValidatePlan_EmptyGoal(t *testing.T) {
	plan := &TeamPlan{
		Goal: "",
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "Step", AssignedRole: "browser_specialist", DependsOn: []string{}},
		},
	}
	if err := validatePlan(plan); err == nil {
		t.Fatal("expected error for empty goal")
	}
}

func TestValidatePlan_DuplicateSubtaskID(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Duplicate IDs",
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "A", AssignedRole: "browser_specialist", DependsOn: []string{}},
			{ID: "s1", Description: "B", AssignedRole: "email_clerk", DependsOn: []string{}},
		},
	}
	err := validatePlan(plan)
	if err == nil {
		t.Fatal("expected error for duplicate subtask ID")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error = %q, want mention of duplicate", err.Error())
	}
}

func TestValidatePlan_EmptySubtaskID(t *testing.T) {
	plan := &TeamPlan{
		Goal: "Empty ID",
		Subtasks: []PlanSubtask{
			{ID: "", Description: "No ID", AssignedRole: "browser_specialist", DependsOn: []string{}},
		},
	}
	if err := validatePlan(plan); err == nil {
		t.Fatal("expected error for empty subtask ID")
	}
}

func TestCompose_DedupMembers(t *testing.T) {
	llmResponse := llmPlanResponse{
		Subtasks: []PlanSubtask{
			{ID: "s1", Description: "Browse A", AssignedRole: "browser_specialist", DependsOn: []string{}},
			{ID: "s2", Description: "Browse B", AssignedRole: "browser_specialist", DependsOn: []string{"s1"}},
		},
	}
	raw, _ := json.Marshal(llmResponse)

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return string(raw), nil
		},
	})

	plan, err := composer.Compose(context.Background(), "Double browse")
	if err != nil {
		t.Fatalf("Compose returned error: %v", err)
	}
	if len(plan.Members) != 1 {
		t.Errorf("len(Members) = %d, want 1 (deduped)", len(plan.Members))
	}
	if plan.Members[0].RoleName != "browser_specialist" {
		t.Errorf("Members[0].RoleName = %q, want browser_specialist", plan.Members[0].RoleName)
	}
}
