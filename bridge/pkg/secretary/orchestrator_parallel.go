package secretary

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

//=============================================================================
// Parallel Execution Types
//=============================================================================

// ParallelErrorPolicy controls how parallel branch failures are handled.
type ParallelErrorPolicy string

const (
	// FailFast cancels remaining branches when any branch fails (default).
	FailFast ParallelErrorPolicy = "fail_fast"

	// CollectAll runs all branches to completion and aggregates errors at merge.
	CollectAll ParallelErrorPolicy = "collect_all"
)

// ParallelConfig configures the parallel execution engine.
type ParallelConfig struct {
	// MaxParallelContainers is the maximum number of containers spawned
	// concurrently. Default is 2 (VPS safety constraint).
	MaxParallelContainers int

	// ErrorPolicy controls failure handling. Default is FailFast.
	ErrorPolicy ParallelErrorPolicy
}

// DefaultParallelConfig returns safe defaults for VPS deployment.
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		MaxParallelContainers: 2,
		ErrorPolicy:           FailFast,
	}
}

// ParallelBranchError holds error details from a failed parallel branch.
type ParallelBranchError struct {
	StepID  string
	AgentID string
	Err     error
}

func (e *ParallelBranchError) Error() string {
	return fmt.Sprintf("parallel branch %s failed: %v", e.StepID, e.Err)
}

func (e *ParallelBranchError) Unwrap() error {
	return e.Err
}

// ParallelGroup identifies a split→branch→merge structure in a workflow.
type ParallelGroup struct {
	// SplitStepID is the StepParallelSplit step.
	SplitStepID string

	// MergeStepID is the StepParallelMerge step (synchronization barrier).
	MergeStepID string

	// BranchStepIDs are the steps to execute concurrently between split and merge.
	BranchStepIDs []string
}

// ParallelResult holds the aggregated outcome of parallel execution.
type ParallelResult struct {
	// BranchResults maps stepID → result for each parallel branch.
	BranchResults map[string]*StepResult

	// Errors collects all branch errors (non-nil only when errors occurred).
	Errors []*ParallelBranchError

	// AllSucceeded is true when every branch completed without error.
	AllSucceeded bool
}

//=============================================================================
// Parallel Executor
//=============================================================================

// ParallelExecutor handles concurrent step execution within the Bridge process.
// It uses errgroup for goroutine pool management and mutex-protected state updates.
type ParallelExecutor struct {
	config   ParallelConfig
	executor *StepExecutor
}

// NewParallelExecutor creates a parallel executor wrapping the given step executor.
func NewParallelExecutor(executor *StepExecutor, cfg ParallelConfig) *ParallelExecutor {
	if cfg.MaxParallelContainers <= 0 {
		cfg.MaxParallelContainers = 2
	}
	if cfg.ErrorPolicy == "" {
		cfg.ErrorPolicy = FailFast
	}
	return &ParallelExecutor{
		config:   cfg,
		executor: executor,
	}
}

// ExecuteParallelGroup runs all branches of a parallel group concurrently.
// It uses errgroup with a bounded goroutine pool (MaxParallelContainers).
// For FailFast policy, the first error cancels remaining branches via context.
// For CollectAll policy, all branches run to completion and errors are aggregated.
func (pe *ParallelExecutor) ExecuteParallelGroup(
	ctx context.Context,
	workflow *Workflow,
	group *ParallelGroup,
	stepMap map[string]WorkflowStep,
	progressOffset float64,
	progressSpan float64,
) (*ParallelResult, error) {
	branchCount := len(group.BranchStepIDs)
	if branchCount == 0 {
		return &ParallelResult{
			BranchResults: make(map[string]*StepResult),
			AllSucceeded:  true,
		}, nil
	}

	// Mutex for concurrent-safe workflow state updates.
	var stateMu sync.Mutex

	result := &ParallelResult{
		BranchResults: make(map[string]*StepResult, branchCount),
		Errors:        make([]*ParallelBranchError, 0),
		AllSucceeded:  true,
	}

	// Build branch steps in stable order.
	branches := make([]WorkflowStep, 0, branchCount)
	for _, id := range group.BranchStepIDs {
		if step, ok := stepMap[id]; ok {
			branches = append(branches, step)
		}
	}

	if pe.config.ErrorPolicy == FailFast {
		return pe.executeFailFast(ctx, workflow, branches, result, &stateMu, progressOffset, progressSpan)
	}
	return pe.executeCollectAll(ctx, workflow, branches, result, &stateMu, progressOffset, progressSpan)
}

// executeFailFast uses errgroup to cancel remaining branches on first failure.
func (pe *ParallelExecutor) executeFailFast(
	ctx context.Context,
	workflow *Workflow,
	branches []WorkflowStep,
	result *ParallelResult,
	stateMu *sync.Mutex,
	progressOffset float64,
	progressSpan float64,
) (*ParallelResult, error) {
	g, gCtx := errgroup.WithContext(ctx)

	// Limit concurrency to MaxParallelContainers.
	g.SetLimit(pe.config.MaxParallelContainers)

	for i, step := range branches {
		i, step := i, step // capture loop variables

		g.Go(func() error {
			// Check if another branch already failed.
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			progress := progressOffset + (float64(i)/float64(len(branches)))*progressSpan

			stateMu.Lock()
			result.BranchResults[step.StepID] = &StepResult{StepID: step.StepID}
			stateMu.Unlock()

			stepResult := pe.executor.executeStepWithRetry(gCtx, workflow, step)

			stateMu.Lock()
			result.BranchResults[step.StepID] = stepResult
			if stepResult.Err != nil {
				result.AllSucceeded = false
				result.Errors = append(result.Errors, &ParallelBranchError{
					StepID:  step.StepID,
					AgentID: stepResult.AgentID,
					Err:     stepResult.Err,
				})
			}
			stateMu.Unlock()

			_ = progress // progress tracking is best-effort in parallel

			if stepResult.Err != nil && !stepResult.Recoverable {
				return stepResult.Err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Context cancelled by FailFast — return first error.
		if len(result.Errors) > 0 {
			return result, result.Errors[0].Err
		}
		return result, err
	}

	if !result.AllSucceeded {
		return result, fmt.Errorf("parallel execution completed with %d branch failures", len(result.Errors))
	}

	return result, nil
}

// executeCollectAll runs all branches to completion regardless of individual failures.
func (pe *ParallelExecutor) executeCollectAll(
	ctx context.Context,
	workflow *Workflow,
	branches []WorkflowStep,
	result *ParallelResult,
	stateMu *sync.Mutex,
	progressOffset float64,
	progressSpan float64,
) (*ParallelResult, error) {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(pe.config.MaxParallelContainers)

	for i, step := range branches {
		i, step := i, step

		g.Go(func() error {
			// Check parent context (not group context — we never cancel siblings).
			select {
			case <-gCtx.Done():
				return nil // parent cancelled, not sibling failure
			default:
			}

			progress := progressOffset + (float64(i)/float64(len(branches)))*progressSpan

			stateMu.Lock()
			result.BranchResults[step.StepID] = &StepResult{StepID: step.StepID}
			stateMu.Unlock()

			stepResult := pe.executor.executeStepWithRetry(gCtx, workflow, step)

			stateMu.Lock()
			result.BranchResults[step.StepID] = stepResult
			if stepResult.Err != nil {
				result.AllSucceeded = false
				result.Errors = append(result.Errors, &ParallelBranchError{
					StepID:  step.StepID,
					AgentID: stepResult.AgentID,
					Err:     stepResult.Err,
				})
			}
			stateMu.Unlock()

			_ = progress

			// Never return error — collect all results.
			return nil
		})
	}

	_ = g.Wait() // always nil since goroutines never return errors

	if !result.AllSucceeded {
		return result, fmt.Errorf("parallel execution completed with %d branch failures", len(result.Errors))
	}

	return result, nil
}

// ExecuteStandaloneParallel executes sub-operations of a single StepParallel step
// concurrently. This is used when a step of type "parallel" has multiple AgentIDs,
// and each agent runs the step concurrently.
func (pe *ParallelExecutor) ExecuteStandaloneParallel(
	ctx context.Context,
	workflow *Workflow,
	step WorkflowStep,
) (*ParallelResult, error) {
	if len(step.AgentIDs) == 0 {
		return nil, ErrNoAgentForStep
	}

	if len(step.AgentIDs) == 1 {
		// Single agent — no parallelism needed, just run normally.
		stepResult := pe.executor.executeStepWithRetry(ctx, workflow, step)
		result := &ParallelResult{
			BranchResults: map[string]*StepResult{step.StepID: stepResult},
			AllSucceeded:  stepResult.Err == nil,
		}
		if stepResult.Err != nil {
			result.Errors = append(result.Errors, &ParallelBranchError{
				StepID:  step.StepID,
				AgentID: stepResult.AgentID,
				Err:     stepResult.Err,
			})
		}
		return result, stepResult.Err
	}

	// Create virtual branch steps — one per agent.
	branches := make([]WorkflowStep, len(step.AgentIDs))
	for i, agentID := range step.AgentIDs {
		branchID := fmt.Sprintf("%s#agent-%d", step.StepID, i)
		branches[i] = WorkflowStep{
			StepID:   branchID,
			Order:    step.Order,
			Type:     StepAction,
			Name:     fmt.Sprintf("%s (agent %d)", step.Name, i),
			Config:   step.Config,
			AgentIDs: []string{agentID},
		}
	}

	result := &ParallelResult{
		BranchResults: make(map[string]*StepResult, len(branches)),
		AllSucceeded:  true,
	}

	var stateMu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(pe.config.MaxParallelContainers)

	for _, branch := range branches {
		branch := branch
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			stepResult := pe.executor.executeStepWithRetry(gCtx, workflow, branch)

			stateMu.Lock()
			result.BranchResults[branch.StepID] = stepResult
			if stepResult.Err != nil {
				result.AllSucceeded = false
				result.Errors = append(result.Errors, &ParallelBranchError{
					StepID:  branch.StepID,
					AgentID: stepResult.AgentID,
					Err:     stepResult.Err,
				})
			}
			stateMu.Unlock()

			if stepResult.Err != nil && !stepResult.Recoverable {
				return stepResult.Err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if len(result.Errors) > 0 {
			return result, result.Errors[0].Err
		}
		return result, err
	}

	if !result.AllSucceeded {
		return result, fmt.Errorf("parallel step %s completed with %d branch failures", step.StepID, len(result.Errors))
	}

	return result, nil
}

//=============================================================================
// Parallel Group Identification
//=============================================================================

// IdentifyParallelGroups scans a template's steps and identifies split→merge
// groups using dependency edges. A parallel group is defined by:
//
//   - A StepParallelSplit step that spawns branches
//   - Branch steps that have no NextStepID (or point to the Merge)
//   - A StepParallelMerge step that acts as a synchronization barrier
//
// Steps between a Split and Merge that share the same Merge dependency are
// parallel branches.
func IdentifyParallelGroups(steps []WorkflowStep) []*ParallelGroup {
	stepMap := make(map[string]WorkflowStep, len(steps))
	for _, s := range steps {
		stepMap[s.StepID] = s
	}

	var groups []*ParallelGroup

	for _, step := range steps {
		if step.Type != StepParallelSplit {
			continue
		}

		// Find the corresponding merge step.
		// The merge step is the StepParallelMerge that appears after this split.
		mergeStepID := ""
		for _, candidate := range steps {
			if candidate.Type == StepParallelMerge && candidate.Order > step.Order {
				// Take the first merge after this split (closest one).
				if mergeStepID == "" || candidate.Order < stepMap[mergeStepID].Order {
					mergeStepID = candidate.StepID
				}
			}
		}

		if mergeStepID == "" {
			continue // no matching merge — skip this split
		}

		// Identify branches: steps between split and merge (by Order).
		var branchIDs []string
		mergeStep := stepMap[mergeStepID]
		for _, s := range steps {
			if s.Order > step.Order && s.Order < mergeStep.Order &&
				s.Type != StepParallelSplit && s.Type != StepParallelMerge {
				branchIDs = append(branchIDs, s.StepID)
			}
		}

		groups = append(groups, &ParallelGroup{
			SplitStepID:   step.StepID,
			MergeStepID:   mergeStepID,
			BranchStepIDs: branchIDs,
		})
	}

	return groups
}

// HasParallelSteps returns true if the template contains any parallel-type steps.
func HasParallelSteps(steps []WorkflowStep) bool {
	for _, s := range steps {
		if s.Type == StepParallel || s.Type == StepParallelSplit || s.Type == StepParallelMerge {
			return true
		}
	}
	return false
}

// BuildParallelGroupIndex creates a lookup map: splitStepID → ParallelGroup.
func BuildParallelGroupIndex(groups []*ParallelGroup) map[string]*ParallelGroup {
	idx := make(map[string]*ParallelGroup, len(groups))
	for _, g := range groups {
		idx[g.SplitStepID] = g
	}
	return idx
}

// BuildStepToGroupMap creates a reverse lookup: branch/merge stepID → ParallelGroup.
// Useful for detecting whether a step belongs to a parallel group.
func BuildStepToGroupMap(groups []*ParallelGroup) map[string]*ParallelGroup {
	m := make(map[string]*ParallelGroup)
	for _, g := range groups {
		m[g.MergeStepID] = g
		for _, id := range g.BranchStepIDs {
			m[id] = g
		}
	}
	return m
}

//=============================================================================
// Error Aggregation
//=============================================================================

// AggregateParallelErrors combines multiple ParallelBranchErrors into a single error.
func AggregateParallelErrors(errs []*ParallelBranchError) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	var agg []error
	for _, e := range errs {
		agg = append(agg, e.Err)
	}
	return errors.Join(agg...)
}
