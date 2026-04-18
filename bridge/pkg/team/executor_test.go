package team

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makePlan creates a minimal TeamPlan from the given goal, subtasks, and roles.
func makePlan(goal string, subtasks []PlanSubtask, roles ...string) *TeamPlan {
	members := make([]PlanMember, len(roles))
	for i, r := range roles {
		members[i] = PlanMember{RoleName: r}
	}
	return &TeamPlan{Goal: goal, Subtasks: subtasks, Members: members}
}

// newTestExecutor builds a TeamPlanExecutor backed by a real in-memory Service.
// Override any func by passing a mutator.
func newTestExecutor(t *testing.T, mut func(cfg *ExecutorConfig)) *TeamPlanExecutor {
	t.Helper()
	svc, _ := newTestService(t)

	cfg := ExecutorConfig{
		Service: svc,
		Spawner: func(ctx context.Context, role, prompt string) (string, error) {
			return fmt.Sprintf("agent-%s", role), nil
		},
		StepExec: func(ctx context.Context, sid, aid, input string) (string, error) {
			return fmt.Sprintf("output-%s", sid), nil
		},
		Escalation: func(ctx context.Context, tid, sid, reason string) (string, error) {
			return "resolved-by-human", nil
		},
	}
	if mut != nil {
		mut(&cfg)
	}
	return NewTeamPlanExecutor(cfg)
}

// ---------------------------------------------------------------------------
// Execute tests
// ---------------------------------------------------------------------------

func TestExecute_SimplePlan(t *testing.T) {
	exec := newTestExecutor(t, nil)

	plan := makePlan("simple goal", []PlanSubtask{
		{ID: "s1", Description: "browse", AssignedRole: "browser_specialist"},
		{ID: "s2", Description: "email", AssignedRole: "email_clerk", DependsOn: []string{"s1"}},
	}, "browser_specialist", "email_clerk")

	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}
	if result.TeamID == "" {
		t.Fatal("expected non-empty team ID")
	}
	if result.Steps[0].SubtaskID != "s1" {
		t.Fatalf("first step should be s1, got %s", result.Steps[0].SubtaskID)
	}
	if result.Steps[1].SubtaskID != "s2" {
		t.Fatalf("second step should be s2, got %s", result.Steps[1].SubtaskID)
	}
}

func TestExecute_ParallelSubtasks(t *testing.T) {
	exec := newTestExecutor(t, nil)

	plan := makePlan("parallel goal", []PlanSubtask{
		{ID: "s1", Description: "browse", AssignedRole: "browser_specialist"},
		{ID: "s2", Description: "docs", AssignedRole: "doc_analyst"},
	}, "browser_specialist", "doc_analyst")

	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}
	// Both steps should succeed independently.
	for _, s := range result.Steps {
		if !s.Success {
			t.Fatalf("step %s should have succeeded: %s", s.SubtaskID, s.Error)
		}
	}
}

func TestExecute_StepFailure_Retry(t *testing.T) {
	var callCount int

	exec := newTestExecutor(t, func(cfg *ExecutorConfig) {
		cfg.StepExec = func(ctx context.Context, sid, aid, input string) (string, error) {
			callCount++
			if callCount <= 2 {
				return "", fmt.Errorf("transient failure %d", callCount)
			}
			return "recovered-output", nil
		}
	})

	plan := makePlan("retry goal", []PlanSubtask{
		{ID: "s1", Description: "flaky step", AssignedRole: "browser_specialist"},
	}, "browser_specialist")

	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success after retries, got: %s", result.Error)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls (2 fails + 1 succeed), got %d", callCount)
	}
	if result.Steps[0].Output != "recovered-output" {
		t.Fatalf("expected recovered-output, got %q", result.Steps[0].Output)
	}
}

func TestExecute_StepFailure_AllRetriesFail(t *testing.T) {
	var escalationCalled bool

	exec := newTestExecutor(t, func(cfg *ExecutorConfig) {
		cfg.StepExec = func(ctx context.Context, sid, aid, input string) (string, error) {
			return "", fmt.Errorf("permanent failure")
		}
		cfg.Escalation = func(ctx context.Context, tid, sid, reason string) (string, error) {
			escalationCalled = true
			return "human-resolution", nil
		}
	})

	plan := makePlan("escalation goal", []PlanSubtask{
		{ID: "s1", Description: "doomed step", AssignedRole: "browser_specialist"},
	}, "browser_specialist")

	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success after escalation, got: %s", result.Error)
	}
	if !escalationCalled {
		t.Fatal("expected escalation handler to be called")
	}
	if result.Steps[0].Output != "human-resolution" {
		t.Fatalf("expected human-resolution, got %q", result.Steps[0].Output)
	}
}

func TestExecute_TopologicalSort_Cycle(t *testing.T) {
	_, err := topologicalSort([]PlanSubtask{
		{ID: "s1", DependsOn: []string{"s2"}},
		{ID: "s2", DependsOn: []string{"s1"}},
	})
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle in error message, got: %v", err)
	}
}

func TestExecute_TopologicalSort_Valid(t *testing.T) {
	sorted, err := topologicalSort([]PlanSubtask{
		{ID: "s1"},
		{ID: "s2", DependsOn: []string{"s1"}},
		{ID: "s3", DependsOn: []string{"s2"}},
	})
	if err != nil {
		t.Fatalf("topologicalSort: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sorted))
	}
	// Verify order: s1 before s2 before s3.
	ids := make([]string, len(sorted))
	for i, s := range sorted {
		ids[i] = s.ID
	}
	if ids[0] != "s1" || ids[1] != "s2" || ids[2] != "s3" {
		t.Fatalf("expected [s1 s2 s3], got %v", ids)
	}
}

func TestExecute_TeamCreationFailure(t *testing.T) {
	svc, store := newTestService(t)
	store.Close() // force DB failure

	exec := NewTeamPlanExecutor(ExecutorConfig{
		Service: svc,
		Spawner: func(ctx context.Context, role, prompt string) (string, error) {
			return "agent", nil
		},
		StepExec:   func(ctx context.Context, sid, aid, input string) (string, error) { return "", nil },
		Escalation: func(ctx context.Context, tid, sid, reason string) (string, error) { return "", nil },
	})

	plan := makePlan("fail goal", []PlanSubtask{
		{ID: "s1", AssignedRole: "browser_specialist"},
	}, "browser_specialist")

	_, err := exec.Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error when team creation fails")
	}
	if !strings.Contains(err.Error(), "create team") {
		t.Fatalf("expected 'create team' in error, got: %v", err)
	}
}

func TestExecute_AgentSpawnFailure(t *testing.T) {
	exec := newTestExecutor(t, func(cfg *ExecutorConfig) {
		cfg.Spawner = func(ctx context.Context, role, prompt string) (string, error) {
			return "", fmt.Errorf("spawn failed for %s", role)
		}
	})

	plan := makePlan("spawn fail goal", []PlanSubtask{
		{ID: "s1", AssignedRole: "browser_specialist"},
	}, "browser_specialist")

	_, err := exec.Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error when agent spawn fails")
	}
	if !strings.Contains(err.Error(), "spawn agent") {
		t.Fatalf("expected 'spawn agent' in error, got: %v", err)
	}
}

func TestExecute_EscalationFailure(t *testing.T) {
	exec := newTestExecutor(t, func(cfg *ExecutorConfig) {
		cfg.StepExec = func(ctx context.Context, sid, aid, input string) (string, error) {
			return "", fmt.Errorf("step failure")
		}
		cfg.Escalation = func(ctx context.Context, tid, sid, reason string) (string, error) {
			return "", fmt.Errorf("human unavailable")
		}
	})

	plan := makePlan("escalation fail goal", []PlanSubtask{
		{ID: "s1", AssignedRole: "browser_specialist"},
	}, "browser_specialist")

	result, err := exec.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure when escalation also fails")
	}
	if !strings.Contains(result.Error, "escalation failed") {
		t.Fatalf("expected 'escalation failed' in result.Error, got: %s", result.Error)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Success {
		t.Fatal("step should have failed")
	}
}
