package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	cron "github.com/robfig/cron/v3"
)

//=============================================================================
// Task Scheduler - Stateless Dispatcher
//=============================================================================

// FactoryInterface defines the studio factory methods the scheduler needs.
// This bridges the two-database architecture (secretary store + studio store)
// without importing the studio package directly (avoids circular dependency).
type FactoryInterface interface {
	Spawn(ctx context.Context, req *SpawnRequestRef) (*SpawnResultRef, error)
}

// AgentInstanceRef is a lightweight reference to an agent instance.
// The scheduler doesn't need the full AgentInstance — just the room_id for dispatch.
type AgentInstanceRef struct {
	ID     string
	RoomID string
	Status string
}

// SpawnRequestRef is a lightweight spawn request.
type SpawnRequestRef struct {
	DefinitionID    string
	TaskDescription string
	UserID          string
	RoomID          string
}

// SpawnResultRef is a lightweight spawn result.
type SpawnResultRef struct {
	InstanceID string
	RoomID     string
}

// MatrixAdapter defines the interface for sending Matrix events.
type MatrixAdapter interface {
	SendEvent(ctx context.Context, roomID string, eventType string, payload interface{}) error
}

// TaskScheduler dispatches due tasks via cold-start (ephemeral container spawn).
// It is a stateless dispatcher — reads due tasks from DB, dispatches, updates next_run.
type TaskScheduler struct {
	store        Store
	factory      FactoryInterface
	matrix       MatrixAdapter
	log          *logger.Logger
	orchestrator *WorkflowOrchestratorImpl
	integration  *OrchestratorIntegration
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewTaskScheduler creates a new task scheduler
func NewTaskScheduler(store Store, factory FactoryInterface, matrix MatrixAdapter, log *logger.Logger, orchestrator *WorkflowOrchestratorImpl, integration *OrchestratorIntegration) *TaskScheduler {
	if log == nil {
		log = logger.Global().WithComponent("scheduler")
	}
	return &TaskScheduler{
		store:        store,
		factory:      factory,
		matrix:       matrix,
		log:          log,
		orchestrator: orchestrator,
		integration:  integration,
		stopCh:       make(chan struct{}),
	}
}

// Start launches the scheduler loop in a background goroutine
func (ts *TaskScheduler) Start() {
	ts.wg.Add(1)
	go ts.run()
	ts.log.Info("task_scheduler_started", "tick_interval", "15s")
}

// Stop signals the scheduler to stop and waits for the goroutine to exit
func (ts *TaskScheduler) Stop() {
	close(ts.stopCh)
	ts.wg.Wait()
	ts.log.Info("task_scheduler_stopped")
}

// run is the main scheduler loop
func (ts *TaskScheduler) run() {
	defer ts.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Process immediately on start (catches missed tasks after bridge restart)
	ts.tick()

	for {
		select {
		case <-ts.stopCh:
			return
		case <-ticker.C:
			ts.tick()
		}
	}
}

// tick processes all due tasks
func (ts *TaskScheduler) tick() {
	if ts.store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasks, err := ts.store.ListDueTasks(ctx)
	if err != nil {
		ts.log.Error("failed_to_list_due_tasks", "error", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	ts.log.Info("processing_due_tasks", "count", len(tasks))

	for _, task := range tasks {
		ts.dispatchTask(ctx, task)
	}
}

// dispatchTask handles a single due task
func (ts *TaskScheduler) dispatchTask(ctx context.Context, task ScheduledTask) {
	// Route through workflow engine if template is set
	if strings.TrimSpace(task.TemplateID) != "" {
		ts.templateDispatch(ctx, task)
		return
	}

	// Skip tasks with empty definition_id and no template (shouldn't happen due to ListDueTasks filter, but guard)
	if strings.TrimSpace(task.DefinitionID) == "" && strings.TrimSpace(task.TemplateID) == "" {
		ts.log.Warn("skipping_task_no_definition_id", "task_id", task.ID)
		// Deactivate to prevent re-processing
		task.IsActive = false
		if err := ts.store.UpdateScheduledTask(ctx, &task); err != nil {
			ts.log.Error("failed_to_deactivate_task", "task_id", task.ID, "error", err)
		}
		return
	}

	// COLD START: all dispatch is cold-only (ephemeral container spawn).
	// NetworkMode: none is a hard architectural constraint.
	ts.coldDispatch(ctx, task)
}

func (ts *TaskScheduler) templateDispatch(ctx context.Context, task ScheduledTask) {
	template, err := ts.store.GetTemplate(ctx, task.TemplateID)
	if err != nil {
		ts.log.Error("template_dispatch_failed_get_template", "task_id", task.ID, "template_id", task.TemplateID, "error", err)
		return
	}
	if template == nil || !template.IsActive {
		ts.log.Error("template_dispatch_template_not_found", "task_id", task.ID, "template_id", task.TemplateID)
		return
	}

	now := time.Now()
	var variables map[string]interface{}
	if template.Variables != nil {
		if err := json.Unmarshal(template.Variables, &variables); err != nil {
			ts.log.Error("template_dispatch_failed_unmarshal_variables", "task_id", task.ID, "template_id", task.TemplateID, "error", err)
			variables = nil
		}
	}
	workflow := &Workflow{
		ID:          fmt.Sprintf("workflow_%d", now.UnixMilli()),
		TemplateID:  task.TemplateID,
		Name:        template.Name,
		Description: template.Description,
		Status:      StatusPending,
		Variables:   variables,
		AgentIDs:    []string{},
		CreatedBy:   task.CreatedBy,
		RoomID:      "",
		StartedAt:   now,
	}

	if err := ts.store.CreateWorkflow(ctx, workflow); err != nil {
		ts.log.Error("template_dispatch_failed_create_workflow", "task_id", task.ID, "template_id", task.TemplateID, "error", err)
		return
	}

	if ts.orchestrator == nil {
		ts.log.Error("template_dispatch_no_orchestrator", "task_id", task.ID)
		return
	}
	if err := ts.orchestrator.StartWorkflow(workflow.ID); err != nil {
		ts.log.Error("template_dispatch_failed_start_workflow", "task_id", task.ID, "workflow_id", workflow.ID, "error", err)
		return
	}

	if ts.integration != nil {
		if err := ts.integration.StartWorkflowExecution(workflow.ID); err != nil {
			ts.log.Error("template_dispatch_failed_start_execution", "task_id", task.ID, "workflow_id", workflow.ID, "error", err)
			return
		}
	}

	ts.log.Info("template_dispatched_task", "task_id", task.ID, "template_id", task.TemplateID, "workflow_id", workflow.ID)
	ts.updateAfterDispatch(ctx, task)
}

// coldDispatch spawns a new container for the task
func (ts *TaskScheduler) coldDispatch(ctx context.Context, task ScheduledTask) {
	result, err := ts.factory.Spawn(ctx, &SpawnRequestRef{
		DefinitionID:    task.DefinitionID,
		TaskDescription: fmt.Sprintf("Scheduled task: %s", task.ID),
		UserID:          task.CreatedBy,
	})
	if err != nil {
		ts.log.Error("failed_to_cold_start", "task_id", task.ID, "definition_id", task.DefinitionID, "error", err)
		return
	}

	ts.log.Info("cold_dispatched_task", "task_id", task.ID, "instance_id", result.InstanceID)
	ts.updateAfterDispatch(ctx, task)
}

// updateAfterDispatch calculates next_run and marks the task dispatched
func (ts *TaskScheduler) updateAfterDispatch(ctx context.Context, task ScheduledTask) {
	// One-shot task (no cron expression): deactivate
	if strings.TrimSpace(task.CronExpression) == "" {
		task.IsActive = false
		if err := ts.store.UpdateScheduledTask(ctx, &task); err != nil {
			ts.log.Error("failed_to_deactivate_oneshot", "task_id", task.ID, "error", err)
		}
		ts.log.Info("deactivated_oneshot_task", "task_id", task.ID)
		return
	}

	// Calculate next_run from cron expression
	loc, err := time.LoadLocation(task.Timezone)
	if err != nil {
		ts.log.Error("invalid_timezone", "task_id", task.ID, "timezone", task.Timezone, "error", err)
		loc = time.UTC
	}

	sched, err := cron.ParseStandard(task.CronExpression)
	if err != nil {
		ts.log.Error("invalid_cron_expression", "task_id", task.ID, "cron", task.CronExpression, "error", err)
		// Deactivate to prevent infinite retry
		task.IsActive = false
		if updateErr := ts.store.UpdateScheduledTask(ctx, &task); updateErr != nil {
			ts.log.Error("failed_to_deactivate_invalid_cron", "task_id", task.ID, "error", updateErr)
		}
		return
	}

	nextRun := sched.Next(time.Now().In(loc))
	if nextRun.IsZero() {
		ts.log.Error("cron_next_run_is_zero", "task_id", task.ID, "cron", task.CronExpression)
		task.IsActive = false
		if err := ts.store.UpdateScheduledTask(ctx, &task); err != nil {
			ts.log.Error("failed_to_deactivate_zero_next_run", "task_id", task.ID, "error", err)
		}
		return
	}

	// Mark dispatched: updates last_run and next_run
	if err := ts.store.MarkDispatched(ctx, task.ID, nextRun); err != nil {
		ts.log.Error("failed_to_mark_dispatched", "task_id", task.ID, "error", err)
		return
	}

	ts.log.Info("updated_task_next_run", "task_id", task.ID, "next_run", nextRun.Format(time.RFC3339))
}

func (ts *TaskScheduler) DispatchNow(ctx context.Context, task *ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if err := ts.store.CreateScheduledTask(ctx, task); err != nil {
		return fmt.Errorf("create scheduled task: %w", err)
	}
	ts.dispatchTask(ctx, *task)
	return nil
}
