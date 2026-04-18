package team

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Plan types
// ---------------------------------------------------------------------------

// TeamPlan represents a decomposed team execution plan.
type TeamPlan struct {
	Goal          string
	Subtasks      []PlanSubtask
	Members       []PlanMember
	ArtifactFlows []ArtifactFlow
}

// PlanSubtask is a single step in the team plan.
type PlanSubtask struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	AssignedRole   string   `json:"assigned_role"`
	DependsOn      []string `json:"depends_on"`
	InputArtifact  string   `json:"input_artifact"`
	OutputArtifact string   `json:"output_artifact"`
}

// PlanMember describes a team member to be instantiated.
type PlanMember struct {
	RoleName string
	AgentID  string // assigned during execution
}

// ArtifactFlow describes a typed artifact handoff between subtasks.
type ArtifactFlow struct {
	FromSubtask  string
	ToSubtask    string
	ArtifactType string // e.g., "BrowserResult", "DocumentRef", "EmailDraft"
}

// ---------------------------------------------------------------------------
// LLM function injection
// ---------------------------------------------------------------------------

// LLMCaller invokes an LLM. Injected to avoid AI client imports.
type LLMCaller func(ctx context.Context, prompt string) (string, error)

// TeamComposerConfig holds constructor parameters.
type TeamComposerConfig struct {
	LLM       LLMCaller
	MaxRetries int // default 3
}

// TeamComposer decomposes a goal string into a TeamPlan using an LLM.
type TeamComposer struct {
	llm        LLMCaller
	maxRetries int
}

// NewTeamComposer creates a TeamComposer from the given config.
func NewTeamComposer(cfg TeamComposerConfig) *TeamComposer {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &TeamComposer{
		llm:        cfg.LLM,
		maxRetries: maxRetries,
	}
}

// ---------------------------------------------------------------------------
// Known artifact types (mirrored from capability package to avoid import)
// ---------------------------------------------------------------------------

var knownArtifactTypes = []string{
	"BrowserIntent",
	"BrowserResult",
	"DocumentRef",
	"ExtractedChunkSet",
	"SecretRef",
	"EmailDraft",
	"ApprovalDecision",
}

// ---------------------------------------------------------------------------
// LLM response schema
// ---------------------------------------------------------------------------

// llmPlanResponse is the JSON shape the LLM must return.
type llmPlanResponse struct {
	Subtasks []PlanSubtask `json:"subtasks"`
}

// ---------------------------------------------------------------------------
// Compose
// ---------------------------------------------------------------------------

// Compose decomposes goal into a TeamPlan by calling the LLM, parsing the
// JSON response, and validating the resulting plan.
func (tc *TeamComposer) Compose(ctx context.Context, goal string) (*TeamPlan, error) {
	if strings.TrimSpace(goal) == "" {
		return nil, fmt.Errorf("team: compose: goal must not be empty")
	}

	prompt := buildCompositionPrompt(goal)

	var lastErr error
	for attempt := 0; attempt < tc.maxRetries; attempt++ {
		raw, err := tc.llm(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("team: compose: llm call failed: %w", err)
		}

		plan, err := parseLLMResponse(goal, raw)
		if err != nil {
			lastErr = err
			continue
		}

		if err := validatePlan(plan); err != nil {
			return nil, fmt.Errorf("team: compose: %w", err)
		}

		return plan, nil
	}

	return nil, fmt.Errorf("team: compose: failed after %d retries: %w", tc.maxRetries, lastErr)
}

// parseLLMResponse extracts the plan from the raw LLM output.
func parseLLMResponse(goal, raw string) (*TeamPlan, error) {
	cleaned := stripCodeFences(raw)

	var resp llmPlanResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, fmt.Errorf("team: compose: parse json: %w", err)
	}

	if len(resp.Subtasks) == 0 {
		return nil, fmt.Errorf("team: compose: llm returned zero subtasks")
	}

	plan := &TeamPlan{
		Goal:     goal,
		Subtasks: resp.Subtasks,
	}

	// Derive members (unique role assignments).
	seen := make(map[string]bool)
	for _, st := range resp.Subtasks {
		if !seen[st.AssignedRole] {
			seen[st.AssignedRole] = true
			plan.Members = append(plan.Members, PlanMember{
				RoleName: st.AssignedRole,
			})
		}
	}

	// Derive artifact flows from subtask input/output pairs.
	subtaskMap := make(map[string]PlanSubtask, len(resp.Subtasks))
	for _, st := range resp.Subtasks {
		subtaskMap[st.ID] = st
	}
	for _, st := range resp.Subtasks {
		if st.InputArtifact == "" {
			continue
		}
		for _, depID := range st.DependsOn {
			dep, ok := subtaskMap[depID]
			if !ok {
				continue
			}
			if dep.OutputArtifact == st.InputArtifact {
				plan.ArtifactFlows = append(plan.ArtifactFlows, ArtifactFlow{
					FromSubtask:  depID,
					ToSubtask:    st.ID,
					ArtifactType: dep.OutputArtifact,
				})
			}
		}
	}

	return plan, nil
}

// ---------------------------------------------------------------------------
// Prompt builder
// ---------------------------------------------------------------------------

// buildCompositionPrompt creates the LLM prompt for team plan decomposition.
func buildCompositionPrompt(goal string) string {
	roles := ListRoles()

	var roleDescs []string
	for _, r := range roles {
		roleDescs = append(roleDescs, fmt.Sprintf("  - %s: %s", r.Name, r.Description))
	}

	artifactList := make([]string, len(knownArtifactTypes))
	copy(artifactList, knownArtifactTypes)

	return fmt.Sprintf(`You are a team composition planner. Decompose the following goal into a team execution plan.

GOAL:
%s

AVAILABLE ROLES:
%s

AVAILABLE ARTIFACT TYPES:
%s

INSTRUCTIONS:
1. Break the goal into ordered subtasks.
2. Assign each subtask to exactly one of the available roles.
3. Specify dependencies between subtasks using depends_on (references to other subtask IDs).
4. For each subtask, specify the input_artifact (artifact type expected) and output_artifact (artifact type produced). Use empty string if none.
5. Subtask IDs must be unique and non-empty.

RESPOND WITH JSON ONLY (no markdown, no explanation):
{
  "subtasks": [
    {
      "id": "step1",
      "description": "Description of what this subtask does",
      "assigned_role": "browser_specialist",
      "depends_on": [],
      "input_artifact": "",
      "output_artifact": "BrowserResult"
    },
    {
      "id": "step2",
      "description": "Description of next subtask",
      "assigned_role": "email_clerk",
      "depends_on": ["step1"],
      "input_artifact": "BrowserResult",
      "output_artifact": "EmailDraft"
    }
  ]
}`, goal, strings.Join(roleDescs, "\n"), strings.Join(artifactList, ", "))
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// validatePlan checks the plan for structural correctness.
func validatePlan(plan *TeamPlan) error {
	if plan == nil {
		return fmt.Errorf("team: validate: plan is nil")
	}
	if strings.TrimSpace(plan.Goal) == "" {
		return fmt.Errorf("team: validate: goal must not be empty")
	}
	if len(plan.Subtasks) == 0 {
		return fmt.Errorf("team: validate: at least one subtask is required")
	}

	// Collect subtask IDs for dependency and uniqueness checks.
	ids := make(map[string]bool, len(plan.Subtasks))
	for _, st := range plan.Subtasks {
		if strings.TrimSpace(st.ID) == "" {
			return fmt.Errorf("team: validate: subtask has empty id")
		}
		if ids[st.ID] {
			return fmt.Errorf("team: validate: duplicate subtask id %q", st.ID)
		}
		ids[st.ID] = true

		// Validate assigned role exists in registry.
		if _, err := GetRole(st.AssignedRole); err != nil {
			return fmt.Errorf("team: validate: subtask %q: %w", st.ID, err)
		}
	}

	// Validate depends_on references point to existing subtask IDs.
	for _, st := range plan.Subtasks {
		for _, dep := range st.DependsOn {
			if !ids[dep] {
				return fmt.Errorf("team: validate: subtask %q depends on unknown subtask %q", st.ID, dep)
			}
		}
	}

	// Check for circular dependencies using DFS.
	if err := checkAcyclic(plan.Subtasks); err != nil {
		return err
	}

	return nil
}

// checkAcyclic performs a topological cycle check using DFS.
func checkAcyclic(subtasks []PlanSubtask) error {
	// Build adjacency list: adj[a] = b means a must happen before b.
	adj := make(map[string][]string)
	for _, st := range subtasks {
		for _, dep := range st.DependsOn {
			adj[dep] = append(adj[dep], st.ID)
		}
	}

	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int, len(subtasks))

	var dfs func(id string) error
	dfs = func(id string) error {
		color[id] = gray
		for _, next := range adj[id] {
			switch color[next] {
			case gray:
				return fmt.Errorf("team: validate: circular dependency involving subtask %q", next)
			case white:
				if err := dfs(next); err != nil {
					return err
				}
			}
		}
		color[id] = black
		return nil
	}

	// Sort IDs for deterministic traversal.
	sortedIDs := make([]string, 0, len(subtasks))
	for _, st := range subtasks {
		sortedIDs = append(sortedIDs, st.ID)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		if color[id] == white {
			if err := dfs(id); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stripCodeFences removes ```json ... ``` wrappers that LLMs often add.
func stripCodeFences(raw string) string {
	s := strings.TrimSpace(raw)
	// Remove opening fence.
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	// Remove closing fence.
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
