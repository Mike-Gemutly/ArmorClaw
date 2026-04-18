package team

import (
	"context"
	"fmt"
	"sort"
)

// ---------------------------------------------------------------------------
// Result types
// ---------------------------------------------------------------------------

// ExecutorStepResult holds the result of executing a single subtask.
type ExecutorStepResult struct {
	SubtaskID string
	Success   bool
	Output    string // JSON artifact output
	Error     string
}

// ExecutorResult holds the result of executing an entire team plan.
type ExecutorResult struct {
	PlanID  string
	TeamID  string
	Steps   []ExecutorStepResult
	Success bool
	Error   string
}

// ---------------------------------------------------------------------------
// Function injection types
// ---------------------------------------------------------------------------

// AgentSpawner creates an agent with role specialization. Returns agent ID.
type AgentSpawner func(ctx context.Context, roleName, systemPrompt string) (agentID string, err error)

// StepExecutor runs a single subtask. Returns output artifact.
type StepExecutor func(ctx context.Context, subtaskID, agentID string, inputArtifact string) (output string, err error)

// EscalationHandler handles escalation to humans. Returns resolution or error.
type EscalationHandler func(ctx context.Context, teamID, subtaskID, reason string) (resolution string, err error)

// ---------------------------------------------------------------------------
// Config & Executor
// ---------------------------------------------------------------------------

// ExecutorConfig holds constructor parameters for TeamPlanExecutor.
type ExecutorConfig struct {
	Service          *Service
	Spawner          AgentSpawner
	StepExec         StepExecutor
	Escalation       EscalationHandler
	MaxRetries       int // default 3
	MaxAgentFailover int // default 2
}

// TeamPlanExecutor orchestrates execution of a TeamPlan: creates a team,
// spawns agents, runs subtasks in topological order with retry / failover /
// escalation, and dissolves the team when done.
type TeamPlanExecutor struct {
	service          *Service
	spawner          AgentSpawner
	stepExec         StepExecutor
	escalation       EscalationHandler
	maxRetries       int
	maxAgentFailover int
}

// NewTeamPlanExecutor creates a TeamPlanExecutor from the given config.
func NewTeamPlanExecutor(cfg ExecutorConfig) *TeamPlanExecutor {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	maxFailover := cfg.MaxAgentFailover
	if maxFailover <= 0 {
		maxFailover = 2
	}
	return &TeamPlanExecutor{
		service:          cfg.Service,
		spawner:          cfg.Spawner,
		stepExec:         cfg.StepExec,
		escalation:       cfg.Escalation,
		maxRetries:       maxRetries,
		maxAgentFailover: maxFailover,
	}
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

// Execute runs a team plan end-to-end:
//  1. Create team via service
//  2. Spawn agents for each unique role, add as members
//  3. Topological-sort subtasks
//  4. Execute in order with retry / failover / escalation
//  5. Dissolve team
//  6. Return ExecutorResult
func (e *TeamPlanExecutor) Execute(ctx context.Context, plan *TeamPlan) (*ExecutorResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("team: executor: plan is nil")
	}

	result := &ExecutorResult{
		PlanID:  plan.Goal,
		Success: true,
	}

	// 1. Create team.
	team, err := e.service.CreateTeam(ctx, plan.Goal, "")
	if err != nil {
		return nil, fmt.Errorf("team: executor: create team: %w", err)
	}
	result.TeamID = team.ID

	// Always dissolve on exit.
	defer func() {
		_ = e.service.DissolveTeam(ctx, team.ID)
	}()

	// 2. Spawn agents and register as members.
	roleAgents := make(map[string][]string) // roleName → agent IDs
	for i := range plan.Members {
		member := &plan.Members[i]

		agentID, err := e.spawner(ctx, member.RoleName, "")
		if err != nil {
			return nil, fmt.Errorf("team: executor: spawn agent for role %q: %w", member.RoleName, err)
		}
		member.AgentID = agentID

		if _, err := e.service.AddMember(ctx, team.ID, agentID, member.RoleName); err != nil {
			return nil, fmt.Errorf("team: executor: add member: %w", err)
		}

		roleAgents[member.RoleName] = append(roleAgents[member.RoleName], agentID)
	}

	// 3. Topological sort.
	sorted, err := topologicalSort(plan.Subtasks)
	if err != nil {
		return nil, err
	}

	// 4. Execute subtasks in dependency order.
	completedOutputs := make(map[string]string) // subtaskID → output artifact

	for _, subtask := range sorted {
		// Resolve input from upstream dependency output.
		var input string
		for _, depID := range subtask.DependsOn {
			if out, ok := completedOutputs[depID]; ok {
				input = out
				break
			}
		}

		sr := e.executeSubtask(ctx, team.ID, subtask, roleAgents, input)
		result.Steps = append(result.Steps, sr)

		if sr.Success {
			completedOutputs[subtask.ID] = sr.Output
		} else {
			result.Success = false
			result.Error = sr.Error
			break
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Retry / failover / escalation chain
// ---------------------------------------------------------------------------

// executeSubtask runs the full retry → failover → lead → human chain for one subtask.
func (e *TeamPlanExecutor) executeSubtask(
	ctx context.Context,
	teamID string,
	subtask PlanSubtask,
	roleAgents map[string][]string,
	input string,
) ExecutorStepResult {
	agents := roleAgents[subtask.AssignedRole]

	// Phase 1 — retry with primary agent.
	if len(agents) > 0 {
		if sr, _ := e.executeWithRetry(ctx, subtask, agents[0], input); sr.Success {
			return sr
		}
	}

	// Phase 2 — failover to other agents sharing the same role.
	maxFO := e.maxAgentFailover
	if maxFO > len(agents)-1 {
		maxFO = len(agents) - 1
	}
	for i := 1; i <= maxFO; i++ {
		if sr, _ := e.executeWithRetry(ctx, subtask, agents[i], input); sr.Success {
			return sr
		}
	}

	// Phase 3 — escalate to team_lead (if different role and lead exists).
	if subtask.AssignedRole != "team_lead" {
		if leads := roleAgents["team_lead"]; len(leads) > 0 {
			if sr, _ := e.executeWithRetry(ctx, subtask, leads[0], input); sr.Success {
				return sr
			}
		}
	}

	// Phase 4 — escalate to human.
	resolution, err := e.escalate(ctx, teamID, subtask.ID,
		fmt.Sprintf("all retries exhausted for subtask %q", subtask.ID))
	if err != nil {
		return ExecutorStepResult{
			SubtaskID: subtask.ID,
			Success:   false,
			Error:     fmt.Sprintf("escalation failed: %v", err),
		}
	}
	return ExecutorStepResult{
		SubtaskID: subtask.ID,
		Success:   true,
		Output:    resolution,
	}
}

// executeWithRetry retries a single subtask with the given agent up to maxRetries times.
func (e *TeamPlanExecutor) executeWithRetry(
	ctx context.Context,
	subtask PlanSubtask,
	agentID, input string,
) (ExecutorStepResult, error) {
	var lastErr error
	for attempt := 0; attempt < e.maxRetries; attempt++ {
		output, err := e.stepExec(ctx, subtask.ID, agentID, input)
		if err == nil {
			return ExecutorStepResult{
				SubtaskID: subtask.ID,
				Success:   true,
				Output:    output,
			}, nil
		}
		lastErr = err
	}
	return ExecutorStepResult{
		SubtaskID: subtask.ID,
		Success:   false,
		Error:     lastErr.Error(),
	}, lastErr
}

// escalate calls the configured EscalationHandler.
func (e *TeamPlanExecutor) escalate(ctx context.Context, teamID, subtaskID, reason string) (string, error) {
	if e.escalation == nil {
		return "", fmt.Errorf("team: executor: no escalation handler configured")
	}
	return e.escalation(ctx, teamID, subtaskID, reason)
}

// ---------------------------------------------------------------------------
// Topological sort (Kahn's algorithm)
// ---------------------------------------------------------------------------

// topologicalSort returns subtasks in execution order respecting DependsOn.
// Returns an error if a cycle is detected.
func topologicalSort(subtasks []PlanSubtask) ([]PlanSubtask, error) {
	inDegree := make(map[string]int, len(subtasks))
	adj := make(map[string][]string)
	byID := make(map[string]PlanSubtask, len(subtasks))

	for _, st := range subtasks {
		byID[st.ID] = st
		if _, exists := inDegree[st.ID]; !exists {
			inDegree[st.ID] = 0
		}
		for _, dep := range st.DependsOn {
			adj[dep] = append(adj[dep], st.ID)
			inDegree[st.ID]++
		}
	}

	// Seed queue with zero-in-degree nodes, sorted for determinism.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []PlanSubtask
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, byID[curr])

		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
		sort.Strings(queue) // keep deterministic
	}

	if len(result) != len(subtasks) {
		return nil, fmt.Errorf("team: executor: cycle detected in subtask dependencies")
	}

	return result, nil
}
