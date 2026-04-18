package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Workflow Orchestrator
//=============================================================================

type Factory interface {
	Spawn(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error)
	Stop(ctx context.Context, instanceID string, timeout time.Duration) error
	Remove(ctx context.Context, instanceID string) error
	GetStatus(ctx context.Context, instanceID string) (*studio.AgentInstance, error)
	ListInstances(definitionID string) ([]*studio.AgentInstance, error)
}

type activeWorkflow struct {
	workflow     *Workflow
	cancelFunc   context.CancelFunc
	startedAt    time.Time
	template     *TaskTemplate
	currentIndex int
}

type OrchestratorConfig struct {
	Store    Store
	Factory  Factory
	EventBus EventEmitter
}

type WorkflowOrchestratorImpl struct {
	mu              sync.RWMutex
	store           Store
	factory         Factory
	eventEmitter    EventEmitter
	activeWorkflows map[string]*activeWorkflow
	ctx             context.Context
	cancel          context.CancelFunc
}

func NewWorkflowOrchestrator(cfg OrchestratorConfig) (*WorkflowOrchestratorImpl, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if cfg.EventBus == nil {
		return nil, fmt.Errorf("event emitter is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkflowOrchestratorImpl{
		store:           cfg.Store,
		factory:         cfg.Factory,
		eventEmitter:    cfg.EventBus,
		activeWorkflows: make(map[string]*activeWorkflow),
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

func (o *WorkflowOrchestratorImpl) StartWorkflow(workflowID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.activeWorkflows[workflowID]; exists {
		return fmt.Errorf("workflow %s is already running", workflowID)
	}

	workflow, err := o.store.GetWorkflow(o.ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := o.validateTransition(workflow.Status, StatusRunning); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	var template *TaskTemplate
	if workflow.TemplateID != "" {
		template, err = o.store.GetTemplate(o.ctx, workflow.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
	}

	workflow.Status = StatusRunning
	workflow.StartedAt = time.Now()

	if template != nil && len(template.Steps) > 0 {
		workflow.CurrentStep = template.Steps[0].StepID
	}

	if err := o.store.UpdateWorkflow(o.ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	workflowCtx, cancel := context.WithCancel(o.ctx)

	startIndex := 0
	if template != nil && len(template.Steps) > 0 {
		startIndex = template.Steps[0].Order
	}

	o.activeWorkflows[workflowID] = &activeWorkflow{
		workflow:     workflow,
		cancelFunc:   cancel,
		startedAt:    time.Now(),
		template:     template,
		currentIndex: startIndex,
	}

	o.eventEmitter.EmitStarted(workflow)

	if template != nil && len(template.Steps) > 0 {
		firstStep := template.Steps[0]
		o.eventEmitter.EmitProgress(workflow, firstStep.StepID, firstStep.Name, 0.0)
	}

	go o.executeWorkflow(workflowCtx, workflowID)

	return nil
}

func (o *WorkflowOrchestratorImpl) GetWorkflow(workflowID string) (*Workflow, error) {
	o.mu.RLock()
	if active, exists := o.activeWorkflows[workflowID]; exists {
		o.mu.RUnlock()
		copy := *active.workflow
		return &copy, nil
	}
	o.mu.RUnlock()

	workflow, err := o.store.GetWorkflow(o.ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	return workflow, nil
}

func (o *WorkflowOrchestratorImpl) ListWorkflows(statusFilter WorkflowStatus, createdBy string) ([]*Workflow, error) {
	filter := WorkflowFilter{}

	if statusFilter != "" {
		filter.Status = &statusFilter
	}
	if createdBy != "" {
		filter.CreatedBy = createdBy
	}

	workflows, err := o.store.ListWorkflows(o.ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	result := make([]*Workflow, len(workflows))
	for i := range workflows {
		result[i] = &workflows[i]
	}

	return result, nil
}

func (o *WorkflowOrchestratorImpl) AdvanceWorkflow(workflowID string, stepID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	if active.workflow.Status != StatusRunning {
		return fmt.Errorf("workflow %s is not running (status: %s)", workflowID, active.workflow.Status)
	}

	if active.template == nil {
		return fmt.Errorf("workflow %s has no template", workflowID)
	}

	if active.workflow.CurrentStep != stepID {
		return fmt.Errorf("workflow %s is not on step %s (current: %s)", workflowID, stepID, active.workflow.CurrentStep)
	}

	currentIdx := active.currentIndex
	var nextStep *WorkflowStep
	for i := range active.template.Steps {
		if active.template.Steps[i].StepID == stepID && i+1 < len(active.template.Steps) {
			nextStep = &active.template.Steps[i+1]
			break
		}
	}

	progress := float64(currentIdx+1) / float64(len(active.template.Steps))
	o.eventEmitter.EmitProgress(active.workflow, stepID, "", progress)

	if nextStep == nil {
		return o.completeWorkflowLocked(workflowID, "all steps completed")
	}

	active.workflow.CurrentStep = nextStep.StepID
	active.currentIndex++

	if err := o.store.UpdateWorkflow(o.ctx, active.workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	stepProgress := float64(active.currentIndex) / float64(len(active.template.Steps))
	o.eventEmitter.EmitProgress(active.workflow, nextStep.StepID, nextStep.Name, stepProgress)

	return nil
}

func (o *WorkflowOrchestratorImpl) CancelWorkflow(workflowID string, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		_, err := o.store.GetWorkflow(o.ctx, workflowID)
		if err != nil {
			return fmt.Errorf("workflow not found: %s", workflowID)
		}
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	if err := o.validateTransition(active.workflow.Status, StatusCancelled); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	active.cancelFunc()

	workflow := active.workflow
	workflow.Status = StatusCancelled
	now := time.Now()
	workflow.CompletedAt = &now
	workflow.ErrorMessage = reason

	_ = o.store.UpdateWorkflow(o.ctx, workflow)

	delete(o.activeWorkflows, workflowID)

	o.eventEmitter.EmitCancelled(workflow, reason)

	return nil
}

// BlockWorkflow transitions a running workflow to blocked status.
// This is called when an agent hits a blocker during execution.
func (o *WorkflowOrchestratorImpl) BlockWorkflow(workflowID, reason, message string, blockerMeta ...map[string]interface{}) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		_, err := o.store.GetWorkflow(o.ctx, workflowID)
		if err != nil {
			return fmt.Errorf("workflow not found: %s", workflowID)
		}
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	if err := o.validateTransition(active.workflow.Status, StatusBlocked); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	workflow := active.workflow
	workflow.Status = StatusBlocked
	workflow.ErrorMessage = message

	_ = o.store.UpdateWorkflow(o.ctx, workflow)

	var meta map[string]interface{}
	if len(blockerMeta) > 0 {
		meta = blockerMeta[0]
	}
	o.eventEmitter.EmitBlocked(workflow, reason, message, meta)

	return nil
}

// UnblockWorkflow transitions a blocked workflow back to running status.
// This is called after a blocker response is received and before re-spawning the agent.
func (o *WorkflowOrchestratorImpl) UnblockWorkflow(workflowID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s is not currently active", workflowID)
	}

	if err := o.validateTransition(active.workflow.Status, StatusRunning); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	workflow := active.workflow
	workflow.Status = StatusRunning
	workflow.ErrorMessage = ""

	if err := o.store.UpdateWorkflow(o.ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

func (o *WorkflowOrchestratorImpl) CompleteWorkflow(workflowID string, result string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.completeWorkflowLocked(workflowID, result)
}

func (o *WorkflowOrchestratorImpl) completeWorkflowLocked(workflowID string, result string) error {
	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	if err := o.validateTransition(active.workflow.Status, StatusCompleted); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	workflow := active.workflow
	workflow.Status = StatusCompleted
	now := time.Now()
	workflow.CompletedAt = &now

	if err := o.store.UpdateWorkflow(o.ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	delete(o.activeWorkflows, workflowID)

	o.eventEmitter.EmitCompleted(workflow, result)

	return nil
}

func (o *WorkflowOrchestratorImpl) FailWorkflow(workflowID string, stepID string, err error, recoverable bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	active, exists := o.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	if validErr := o.validateTransition(active.workflow.Status, StatusFailed); validErr != nil {
		return fmt.Errorf("invalid status transition: %w", validErr)
	}

	workflow := active.workflow
	workflow.Status = StatusFailed
	now := time.Now()
	workflow.CompletedAt = &now
	if err != nil {
		workflow.ErrorMessage = err.Error()
	}

	if storeErr := o.store.UpdateWorkflow(o.ctx, workflow); storeErr != nil {
		return fmt.Errorf("failed to update workflow: %w", storeErr)
	}

	delete(o.activeWorkflows, workflowID)

	o.eventEmitter.EmitFailed(workflow, stepID, err, recoverable)

	return nil
}

func (o *WorkflowOrchestratorImpl) UpdateProgress(workflowID string, stepID string, progress float64) error {
	o.mu.RLock()
	active, exists := o.activeWorkflows[workflowID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workflow %s is not currently running", workflowID)
	}

	var stepName string
	if active.template != nil {
		for _, step := range active.template.Steps {
			if step.StepID == stepID {
				stepName = step.Name
				break
			}
		}
	}

	o.eventEmitter.EmitProgress(active.workflow, stepID, stepName, progress)

	return nil
}

func (o *WorkflowOrchestratorImpl) GetStepConfig(workflowID string, stepID string) (json.RawMessage, error) {
	workflow, err := o.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	template, err := o.store.GetTemplate(o.ctx, workflow.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	for _, step := range template.Steps {
		if step.StepID == stepID {
			return step.Config, nil
		}
	}

	return nil, fmt.Errorf("step %s not found in workflow %s", stepID, workflowID)
}

func (o *WorkflowOrchestratorImpl) Shutdown() {
	o.cancel()

	o.mu.Lock()
	defer o.mu.Unlock()

	for id, active := range o.activeWorkflows {
		active.cancelFunc()

		workflow := active.workflow
		workflow.Status = StatusCancelled
		now := time.Now()
		workflow.CompletedAt = &now
		workflow.ErrorMessage = "orchestrator shutdown"
		_ = o.store.UpdateWorkflow(o.ctx, workflow)

		o.eventEmitter.EmitCancelled(workflow, "orchestrator shutdown")

		delete(o.activeWorkflows, id)
	}
}

func (o *WorkflowOrchestratorImpl) executeWorkflow(ctx context.Context, workflowID string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.mu.RLock()
			active, exists := o.activeWorkflows[workflowID]
			if !exists {
				o.mu.RUnlock()
				return
			}
			workflow := active.workflow
			template := active.template
			currentIdx := active.currentIndex
			o.mu.RUnlock()

			if template == nil || len(template.Steps) == 0 {
				continue
			}

			if currentIdx < len(template.Steps) {
				step := template.Steps[currentIdx]
				progress := float64(currentIdx+1) / float64(len(template.Steps))
				o.eventEmitter.EmitProgress(workflow, step.StepID, step.Name, progress)
			}
		}
	}
}

func (o *WorkflowOrchestratorImpl) GetActiveWorkflowCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.activeWorkflows)
}

func (o *WorkflowOrchestratorImpl) IsRunning() bool {
	select {
	case <-o.ctx.Done():
		return false
	default:
		return true
	}
}

func (o *WorkflowOrchestratorImpl) validateTransition(from, to WorkflowStatus) error {
	if from == to {
		return nil
	}

	validTransitions := map[WorkflowStatus][]WorkflowStatus{
		StatusPending:   {StatusRunning, StatusCancelled},
		StatusRunning:   {StatusCompleted, StatusFailed, StatusCancelled, StatusBlocked},
		StatusBlocked:   {StatusRunning, StatusFailed, StatusCancelled},
		StatusCompleted: {},
		StatusFailed:    {},
		StatusCancelled: {},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("unknown status: %s", from)
	}

	for _, status := range allowed {
		if status == to {
			return nil
		}
	}

	return errors.New("invalid status transition")
}
