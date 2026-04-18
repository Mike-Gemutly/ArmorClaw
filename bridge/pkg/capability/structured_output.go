package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

// LLMCaller invokes an LLM and returns the raw text response.
// Injected to avoid importing AI client packages.
type LLMCaller func(ctx context.Context, prompt string) (string, error)

// TeamPlanOutput represents the parsed LLM response for team composition.
type TeamPlanOutput struct {
	Goal            string           `json:"goal"`
	Subtasks        []SubtaskDef     `json:"subtasks"`
	RoleAssignments []RoleAssignment `json:"role_assignments"`
}

// Validate ensures required fields are present and consistent.
func (t *TeamPlanOutput) Validate() error {
	if t.Goal == "" {
		return fmt.Errorf("capability: TeamPlanOutput: field goal is required")
	}
	if len(t.Subtasks) == 0 {
		return fmt.Errorf("capability: TeamPlanOutput: at least one subtask is required")
	}
	if len(t.RoleAssignments) == 0 {
		return fmt.Errorf("capability: TeamPlanOutput: at least one role_assignment is required")
	}
	for i, s := range t.Subtasks {
		if s.ID == "" {
			return fmt.Errorf("capability: TeamPlanOutput: subtask[%d]: id is required", i)
		}
		if s.Description == "" {
			return fmt.Errorf("capability: TeamPlanOutput: subtask[%d]: description is required", i)
		}
		if s.AssignedRole == "" {
			return fmt.Errorf("capability: TeamPlanOutput: subtask[%d]: assigned_role is required", i)
		}
	}
	for i, r := range t.RoleAssignments {
		if r.RoleName == "" {
			return fmt.Errorf("capability: TeamPlanOutput: role_assignment[%d]: role_name is required", i)
		}
	}
	return nil
}

// SubtaskDef represents a single subtask in a team plan.
type SubtaskDef struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	AssignedRole string   `json:"assigned_role"`
	DependsOn    []string `json:"depends_on,omitempty"`
}

// RoleAssignment maps a subtask to a role.
type RoleAssignment struct {
	RoleName   string   `json:"role_name"`
	SubtaskIDs []string `json:"subtask_ids"`
}

// StructuredOutputConfig holds configuration for NewStructuredOutputParser.
type StructuredOutputConfig struct {
	LLM        LLMCaller
	MaxRetries int
}

// StructuredOutputParser parses LLM JSON responses into Go structs.
type StructuredOutputParser struct {
	llm        LLMCaller
	maxRetries int
	fallback   *TeamPlanOutput
}

// NewStructuredOutputParser creates a parser with the given config.
func NewStructuredOutputParser(cfg StructuredOutputConfig) *StructuredOutputParser {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &StructuredOutputParser{
		llm:        cfg.LLM,
		maxRetries: maxRetries,
		fallback:   nil, // lazily set via DefaultTeamPlan
	}
}

// jsonBlockRe extracts JSON from markdown code fences or raw JSON.
var jsonBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)```")

// extractJSON finds a JSON object within the LLM response. It first looks
// for fenced code blocks; if none found it treats the entire string as JSON.
func extractJSON(raw string) string {
	if m := jsonBlockRe.FindStringSubmatch(raw); len(m) >= 2 {
		return m[1]
	}
	return raw
}

// Parse parses a raw LLM response string into a TeamPlanOutput.
// It tolerates responses wrapped in markdown code fences.
func (p *StructuredOutputParser) Parse(ctx context.Context, llmResponse string) (*TeamPlanOutput, error) {
	body := extractJSON(llmResponse)

	var out TeamPlanOutput
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return nil, fmt.Errorf("capability: structured_output: invalid JSON: %w", err)
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

// ParseWithRetries calls the LLM, tries Parse, and retries on malformed output
// up to maxRetries times. On all retries exhausted, returns the fallback plan.
func (p *StructuredOutputParser) ParseWithRetries(ctx context.Context, prompt string) (*TeamPlanOutput, error) {
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Check for cancellation before each attempt.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("capability: structured_output: context cancelled: %w", err)
		}

		raw, err := p.llm(ctx, prompt)
		if err != nil {
			continue // retry on LLM error
		}

		result, parseErr := p.Parse(ctx, raw)
		if parseErr == nil {
			return result, nil
		}
		// malformed output — retry
	}

	// All retries exhausted — return fallback.
	return p.DefaultTeamPlan(prompt), nil
}

// DefaultTeamPlan returns a sensible default: team_lead + browser_specialist +
// form_filler for any goal.
func (p *StructuredOutputParser) DefaultTeamPlan(goal string) *TeamPlanOutput {
	return &TeamPlanOutput{
		Goal: goal,
		Subtasks: []SubtaskDef{
			{
				ID:           "st-1",
				Description:  "Coordinate the team and plan approach",
				AssignedRole: "team_lead",
				DependsOn:    nil,
			},
			{
				ID:           "st-2",
				Description:  "Browse web pages and extract information",
				AssignedRole: "browser_specialist",
				DependsOn:    []string{"st-1"},
			},
			{
				ID:           "st-3",
				Description:  "Fill forms with user-provided data",
				AssignedRole: "form_filler",
				DependsOn:    []string{"st-2"},
			},
		},
		RoleAssignments: []RoleAssignment{
			{RoleName: "team_lead", SubtaskIDs: []string{"st-1"}},
			{RoleName: "browser_specialist", SubtaskIDs: []string{"st-2"}},
			{RoleName: "form_filler", SubtaskIDs: []string{"st-3"}},
		},
	}
}
