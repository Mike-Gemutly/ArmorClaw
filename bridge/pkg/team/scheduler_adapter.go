package team

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// TaskOptions
// ---------------------------------------------------------------------------

// TaskOptions controls how the scheduler adapter handles a task.
type TaskOptions struct {
	AutoTeam bool              `json:"auto_team"`
	TeamSize int               `json:"team_size,omitempty"` // 0 = auto-determine
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// SchedulerAdapter
// ---------------------------------------------------------------------------

// SchedulerAdapter provides team-aware task scheduling. When a task has the
// auto_team flag set, it delegates to TeamComposer for decomposition instead
// of linear workflow execution.
type SchedulerAdapter struct {
	composer *TeamComposer
	executor *TeamPlanExecutor
}

// NewSchedulerAdapter creates a SchedulerAdapter wired to the given composer
// and executor.
func NewSchedulerAdapter(composer *TeamComposer, executor *TeamPlanExecutor) *SchedulerAdapter {
	return &SchedulerAdapter{
		composer: composer,
		executor: executor,
	}
}

// ShouldUseTeam returns true when the task options request automatic team
// composition.
func (a *SchedulerAdapter) ShouldUseTeam(opts TaskOptions) bool {
	return opts.AutoTeam
}

// ScheduleTeamTask composes a team plan for the goal and then executes it.
func (a *SchedulerAdapter) ScheduleTeamTask(ctx context.Context, goal string, opts TaskOptions) (*ExecutorResult, error) {
	plan, err := a.composer.Compose(ctx, goal)
	if err != nil {
		return nil, fmt.Errorf("team: scheduler: compose: %w", err)
	}

	result, err := a.executor.Execute(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("team: scheduler: execute: %w", err)
	}

	return result, nil
}

// ScheduleLinearTask is a passthrough placeholder that will eventually hand off
// to the existing secretary linear workflow. Currently returns nil to indicate
// success without side effects.
func (a *SchedulerAdapter) ScheduleLinearTask(_ context.Context, _ string) error {
	// Future: delegate to existing secretary workflow.
	return nil
}
