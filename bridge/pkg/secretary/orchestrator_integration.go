package secretary

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/events"
	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Step Execution Errors
//=============================================================================

var (
	ErrNoAgentForStep        = errors.New("no agent available for step")
	ErrStepExecutionFailed   = errors.New("step execution failed")
	ErrAgentSpawnFailed      = errors.New("failed to spawn agent")
	ErrAgentTimeout          = errors.New("agent execution timeout")
	ErrWorkflowNotRunning    = errors.New("workflow not running")
	ErrInvalidExecutionOrder = errors.New("invalid execution order")
)

type StepExecutionError struct {
	StepID      string
	AgentID     string
	Err         error
	Recoverable bool
}

func (e *StepExecutionError) Error() string {
	if e.AgentID != "" {
		return fmt.Sprintf("%v: step %s (agent %s)", e.Err, e.StepID, e.AgentID)
	}
	return fmt.Sprintf("%v: step %s", e.Err, e.StepID)
}

func (e *StepExecutionError) Unwrap() error {
	return e.Err
}

//=============================================================================
// Step Executor
//=============================================================================

type ApprovalChecker interface {
	EvaluateStep(ctx context.Context, workflow *Workflow, template *TaskTemplate, step *WorkflowStep, piiFields []string, initiator string) (*ApprovalResult, error)
}

type StepExecutorConfig struct {
	Factory        *studio.AgentFactory
	Validator      *DependencyValidator
	ApprovalEngine ApprovalChecker
	EventBus       *events.MatrixEventBus
	DefaultTimeout time.Duration
	StepRetryCount int
	StepRetryDelay time.Duration
}

type StepExecutor struct {
	factory        *studio.AgentFactory
	validator      *DependencyValidator
	approvalEngine ApprovalChecker
	eventBus       *events.MatrixEventBus
	defaultTimeout time.Duration
	retryCount     int
	retryDelay     time.Duration

	mu           sync.RWMutex
	runningSteps map[string]*runningStep
}

type runningStep struct {
	workflowID string
	stepID     string
	instanceID string
	startedAt  time.Time
	cancelFunc context.CancelFunc
}

func NewStepExecutor(cfg StepExecutorConfig) *StepExecutor {
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 5 * time.Minute
	}
	if cfg.StepRetryCount == 0 {
		cfg.StepRetryCount = 1
	}
	if cfg.StepRetryDelay == 0 {
		cfg.StepRetryDelay = 1 * time.Second
	}

	return &StepExecutor{
		factory:        cfg.Factory,
		validator:      cfg.Validator,
		approvalEngine: cfg.ApprovalEngine,
		eventBus:       cfg.EventBus,
		defaultTimeout: cfg.DefaultTimeout,
		retryCount:     cfg.StepRetryCount,
		retryDelay:     cfg.StepRetryDelay,
		runningSteps:   make(map[string]*runningStep),
	}
}

func (e *StepExecutor) ExecuteSteps(
	ctx context.Context,
	orchestrator *WorkflowOrchestratorImpl,
	workflow *Workflow,
	template *TaskTemplate,
) error {
	if template == nil || len(template.Steps) == 0 {
		return nil
	}

	validation := e.validator.ValidateTemplate(template)
	if !validation.Valid {
		if len(validation.Errors) > 0 {
			return fmt.Errorf("%w: %v", ErrInvalidExecutionOrder, validation.Errors[0])
		}
		return ErrInvalidExecutionOrder
	}

	stepMap := make(map[string]WorkflowStep)
	for _, step := range template.Steps {
		stepMap[step.StepID] = step
	}

	for i, stepID := range validation.ExecutionOrder {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		step, exists := stepMap[stepID]
		if !exists {
			return fmt.Errorf("%w: step %s not found", ErrInvalidExecutionOrder, stepID)
		}

		progress := float64(i) / float64(len(validation.ExecutionOrder))
		orchestrator.UpdateProgress(workflow.ID, stepID, progress)

		if e.approvalEngine != nil && len(template.PIIRefs) > 0 {
			approvalResult, err := e.approvalEngine.EvaluateStep(ctx, workflow, template, &step, template.PIIRefs, workflow.CreatedBy)
			if err != nil {
				return fmt.Errorf("approval evaluation failed for step %s: %w", stepID, err)
			}

			if approvalResult.Required && !approvalResult.Approved {
				if approvalResult.NeedsApproval {
					if e.eventBus == nil {
						return fmt.Errorf("step %s requires PII approval but no event bus configured", stepID)
					}
					approvedFields, err := PendingApproval(ctx, e.eventBus, workflow.RoomID, stepID, approvalResult.DeniedFields)
					if err != nil {
						return fmt.Errorf("PII approval failed for step %s: %w", stepID, err)
					}
					_ = approvedFields
				} else if len(approvalResult.DeniedFields) > 0 {
					return fmt.Errorf("step %s denied: fields %v blocked by policy", stepID, approvalResult.DeniedFields)
				}
			}
		}

		result := e.executeStepWithRetry(ctx, workflow, step)

		if result.Err != nil {
			if !result.Recoverable {
				return &StepExecutionError{
					StepID:      stepID,
					AgentID:     result.AgentID,
					Err:         result.Err,
					Recoverable: result.Recoverable,
				}
			}
		}

		if err := orchestrator.AdvanceWorkflow(workflow.ID, stepID); err != nil {
			if errors.Is(err, fmt.Errorf("all steps completed")) ||
				(err != nil && err.Error() == "all steps completed") {
				return nil
			}
			return fmt.Errorf("failed to advance workflow: %w", err)
		}
	}

	return nil
}

type StepResult struct {
	StepID      string
	AgentID     string
	InstanceID  string
	Err         error
	Recoverable bool
}

func (e *StepExecutor) executeStepWithRetry(ctx context.Context, workflow *Workflow, step WorkflowStep) *StepResult {
	var lastErr error
	var lastAgentID string

	for attempt := 0; attempt <= e.retryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &StepResult{
					StepID:      step.StepID,
					Err:         ctx.Err(),
					Recoverable: false,
				}
			case <-time.After(e.retryDelay):
			}
		}

		result := e.executeStep(ctx, workflow, step)
		lastErr = result.Err
		lastAgentID = result.AgentID

		if result.Err == nil {
			return result
		}

		if !result.Recoverable {
			return result
		}
	}

	return &StepResult{
		StepID:      step.StepID,
		AgentID:     lastAgentID,
		Err:         lastErr,
		Recoverable: false,
	}
}

func (e *StepExecutor) executeStep(ctx context.Context, workflow *Workflow, step WorkflowStep) *StepResult {
	if len(step.AgentIDs) == 0 {
		return &StepResult{
			StepID:      step.StepID,
			Err:         ErrNoAgentForStep,
			Recoverable: false,
		}
	}

	agentID := step.AgentIDs[0]

	taskDesc := fmt.Sprintf("Workflow %s - Step: %s", workflow.ID, step.Name)

	spawnReq := &studio.SpawnRequest{
		DefinitionID:    agentID,
		TaskDescription: taskDesc,
		UserID:          workflow.CreatedBy,
		RoomID:          workflow.RoomID,
		Config:          step.Config,
	}

	spawnCtx, cancel := context.WithTimeout(ctx, e.defaultTimeout)
	defer cancel()

	result, err := e.factory.Spawn(spawnCtx, spawnReq)
	if err != nil {
		return &StepResult{
			StepID:      step.StepID,
			AgentID:     agentID,
			Err:         fmt.Errorf("%w: %v", ErrAgentSpawnFailed, err),
			Recoverable: true,
		}
	}

	instanceID := result.Instance.ID

	e.mu.Lock()
	stepCtx, stepCancel := context.WithCancel(ctx)
	e.runningSteps[instanceID] = &runningStep{
		workflowID: workflow.ID,
		stepID:     step.StepID,
		instanceID: instanceID,
		startedAt:  time.Now(),
		cancelFunc: stepCancel,
	}
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.runningSteps, instanceID)
		e.mu.Unlock()
	}()

	execResult := e.waitForCompletion(stepCtx, instanceID)

	return &StepResult{
		StepID:      step.StepID,
		AgentID:     agentID,
		InstanceID:  instanceID,
		Err:         execResult,
		Recoverable: execResult != nil && !errors.Is(execResult, context.Canceled),
	}
}

func (e *StepExecutor) waitForCompletion(ctx context.Context, instanceID string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = e.factory.Stop(context.Background(), instanceID, 10*time.Second)
			return ctx.Err()
		case <-ticker.C:
			instance, err := e.factory.GetStatus(ctx, instanceID)
			if err != nil {
				return fmt.Errorf("failed to get agent status: %w", err)
			}

			switch instance.Status {
			case studio.StatusCompleted:
				return nil
			case studio.StatusFailed:
				return fmt.Errorf("%w: agent exited with failure", ErrStepExecutionFailed)
			case studio.StatusRunning:
				continue
			default:
				continue
			}
		}
	}
}

func (e *StepExecutor) CancelStep(instanceID string) error {
	e.mu.Lock()
	running, exists := e.runningSteps[instanceID]
	e.mu.Unlock()

	if !exists {
		return nil
	}

	running.cancelFunc()

	return e.factory.Stop(context.Background(), instanceID, 10*time.Second)
}

func (e *StepExecutor) CancelAllForWorkflow(workflowID string) error {
	e.mu.RLock()
	var toCancel []string
	for instanceID, running := range e.runningSteps {
		if running.workflowID == workflowID {
			toCancel = append(toCancel, instanceID)
		}
	}
	e.mu.RUnlock()

	var lastErr error
	for _, instanceID := range toCancel {
		if err := e.CancelStep(instanceID); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (e *StepExecutor) GetRunningSteps(workflowID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var steps []string
	for _, running := range e.runningSteps {
		if running.workflowID == workflowID {
			steps = append(steps, running.stepID)
		}
	}
	return steps
}

//=============================================================================
// Orchestrator Integration
//=============================================================================

type OrchestratorIntegration struct {
	orchestrator        *WorkflowOrchestratorImpl
	executor            *StepExecutor
	store               Store
	approvalEngine      *ApprovalEngineImpl
	notificationService *NotificationService

	mu               sync.RWMutex
	executionCancels map[string]context.CancelFunc
}

type IntegrationConfig struct {
	Orchestrator        *WorkflowOrchestratorImpl
	Executor            *StepExecutor
	Store               Store
	ApprovalEngine      *ApprovalEngineImpl
	NotificationService *NotificationService
}

func NewOrchestratorIntegration(cfg IntegrationConfig) *OrchestratorIntegration {
	return &OrchestratorIntegration{
		orchestrator:        cfg.Orchestrator,
		executor:            cfg.Executor,
		store:               cfg.Store,
		approvalEngine:      cfg.ApprovalEngine,
		notificationService: cfg.NotificationService,
		executionCancels:    make(map[string]context.CancelFunc),
	}
}

func (i *OrchestratorIntegration) StartWorkflowExecution(workflowID string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.executionCancels[workflowID]; exists {
		return fmt.Errorf("workflow %s is already executing", workflowID)
	}

	workflow, err := i.orchestrator.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.Status != StatusRunning {
		return fmt.Errorf("%w: current status %s", ErrWorkflowNotRunning, workflow.Status)
	}

	var template *TaskTemplate
	if workflow.TemplateID != "" {
		template, err = i.store.GetTemplate(context.Background(), workflow.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	i.executionCancels[workflowID] = cancel

	go i.runWorkflow(ctx, workflowID, workflow, template)

	return nil
}

func (i *OrchestratorIntegration) runWorkflow(
	ctx context.Context,
	workflowID string,
	workflow *Workflow,
	template *TaskTemplate,
) {
	defer func() {
		i.mu.Lock()
		delete(i.executionCancels, workflowID)
		i.mu.Unlock()
	}()

	if i.notificationService != nil && workflow != nil {
		i.notificationService.NotifyWorkflowStarted(workflow, template)
	}

	err := i.executor.ExecuteSteps(ctx, i.orchestrator, workflow, template)

	if err != nil {
		var stepErr *StepExecutionError
		recoverable := false
		stepID := ""

		if errors.As(err, &stepErr) {
			recoverable = stepErr.Recoverable
			stepID = stepErr.StepID
		}

		if errors.Is(err, context.Canceled) {
			i.orchestrator.CancelWorkflow(workflowID, "execution cancelled")
			if i.notificationService != nil {
				updatedWf, _ := i.orchestrator.GetWorkflow(workflowID)
				i.notificationService.NotifyWorkflowCancelled(updatedWf, "execution cancelled", template)
			}
			return
		}

		i.orchestrator.FailWorkflow(workflowID, stepID, err, recoverable)
		if i.notificationService != nil {
			updatedWf, _ := i.orchestrator.GetWorkflow(workflowID)
			i.notificationService.NotifyWorkflowFailed(updatedWf, stepID, err, recoverable, template)
		}
		return
	}

	i.orchestrator.CompleteWorkflow(workflowID, "all steps completed successfully")
	if i.notificationService != nil {
		updatedWf, _ := i.orchestrator.GetWorkflow(workflowID)
		i.notificationService.NotifyWorkflowCompleted(updatedWf, "all steps completed successfully", template)
	}
}

func (i *OrchestratorIntegration) CancelWorkflowExecution(workflowID string) error {
	i.mu.Lock()
	cancel, exists := i.executionCancels[workflowID]
	i.mu.Unlock()

	if !exists {
		return nil
	}

	cancel()

	return i.executor.CancelAllForWorkflow(workflowID)
}

func (i *OrchestratorIntegration) GetExecutionStatus(workflowID string) *ExecutionStatus {
	var runningSteps []string
	if i.executor != nil {
		runningSteps = i.executor.GetRunningSteps(workflowID)
	}

	i.mu.RLock()
	_, isExecuting := i.executionCancels[workflowID]
	i.mu.RUnlock()

	return &ExecutionStatus{
		WorkflowID:   workflowID,
		IsExecuting:  isExecuting,
		RunningSteps: runningSteps,
	}
}

type ExecutionStatus struct {
	WorkflowID   string
	IsExecuting  bool
	RunningSteps []string
}
