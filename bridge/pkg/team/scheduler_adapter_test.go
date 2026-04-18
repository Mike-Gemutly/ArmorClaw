package team

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// ShouldUseTeam
// ---------------------------------------------------------------------------

func TestShouldUseTeam_AutoTeamTrue(t *testing.T) {
	adapter := &SchedulerAdapter{}
	opts := TaskOptions{AutoTeam: true}
	if !adapter.ShouldUseTeam(opts) {
		t.Fatal("expected ShouldUseTeam true when AutoTeam is set")
	}
}

func TestShouldUseTeam_AutoTeamFalse(t *testing.T) {
	adapter := &SchedulerAdapter{}
	opts := TaskOptions{AutoTeam: false}
	if adapter.ShouldUseTeam(opts) {
		t.Fatal("expected ShouldUseTeam false when AutoTeam is unset")
	}
}

// ---------------------------------------------------------------------------
// ScheduleTeamTask
// ---------------------------------------------------------------------------

func TestScheduleTeamTask_Success(t *testing.T) {
	ctx := context.Background()

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return `{"subtasks":[{"id":"s1","description":"do work","assigned_role":"browser_specialist","depends_on":[],"input_artifact":"","output_artifact":""}]}`, nil
		},
	})

	svc, _ := newTestService(t)
	executor := NewTeamPlanExecutor(ExecutorConfig{
		Service: svc,
		Spawner: func(ctx context.Context, role, prompt string) (string, error) {
			return fmt.Sprintf("agent-%s", role), nil
		},
		StepExec: func(ctx context.Context, sid, aid, input string) (string, error) {
			return "output", nil
		},
	})

	adapter := NewSchedulerAdapter(composer, executor)
	result, err := adapter.ScheduleTeamTask(ctx, "search flights", TaskOptions{AutoTeam: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestScheduleTeamTask_ComposeFails(t *testing.T) {
	ctx := context.Background()

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("llm down")
		},
	})

	svc, _ := newTestService(t)
	executor := NewTeamPlanExecutor(ExecutorConfig{Service: svc})
	adapter := NewSchedulerAdapter(composer, executor)

	_, err := adapter.ScheduleTeamTask(ctx, "search flights", TaskOptions{AutoTeam: true})
	if err == nil {
		t.Fatal("expected error when compose fails")
	}
}

func TestScheduleTeamTask_ExecuteFails(t *testing.T) {
	ctx := context.Background()

	composer := NewTeamComposer(TeamComposerConfig{
		LLM: func(_ context.Context, _ string) (string, error) {
			return `{"subtasks":[{"id":"s1","description":"do work","assigned_role":"browser_specialist","depends_on":[],"input_artifact":"","output_artifact":""}]}`, nil
		},
	})

	svc, _ := newTestService(t)
	executor := NewTeamPlanExecutor(ExecutorConfig{
		Service: svc,
		Spawner: func(ctx context.Context, role, prompt string) (string, error) {
			return "", errors.New("executor crashed")
		},
	})

	adapter := NewSchedulerAdapter(composer, executor)

	_, err := adapter.ScheduleTeamTask(ctx, "search flights", TaskOptions{AutoTeam: true})
	if err == nil {
		t.Fatal("expected error when execute fails")
	}
}

// ---------------------------------------------------------------------------
// ScheduleLinearTask
// ---------------------------------------------------------------------------

func TestScheduleLinearTask(t *testing.T) {
	adapter := &SchedulerAdapter{}
	if err := adapter.ScheduleLinearTask(context.Background(), "some goal"); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}
