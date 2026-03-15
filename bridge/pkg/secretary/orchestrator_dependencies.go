package secretary

import (
	"errors"
	"fmt"
	"sync"
)

//=============================================================================
// Dependency Validation Errors
//=============================================================================

var (
	ErrCircularDependency   = errors.New("circular dependency detected")
	ErrMissingDependency    = errors.New("missing dependency")
	ErrInvalidStepReference = errors.New("invalid step reference")
	ErrEmptyStepID          = errors.New("step ID cannot be empty")
	ErrDuplicateStepID      = errors.New("duplicate step ID")
)

type DependencyError struct {
	StepID     string
	Dependency string
	CyclePath  []string
	Err        error
}

func (e *DependencyError) Error() string {
	switch {
	case len(e.CyclePath) > 0:
		return fmt.Sprintf("%v: cycle path: %v", e.Err, e.CyclePath)
	case e.Dependency != "":
		return fmt.Sprintf("%v: step %q depends on %q", e.Err, e.StepID, e.Dependency)
	default:
		return fmt.Sprintf("%v: step %q", e.Err, e.StepID)
	}
}

func (e *DependencyError) Unwrap() error {
	return e.Err
}

//=============================================================================
// Dependency Validator
//=============================================================================

type DependencyValidator struct {
	mu sync.RWMutex
}

func NewDependencyValidator() *DependencyValidator {
	return &DependencyValidator{}
}

type ValidationResult struct {
	Valid           bool
	ExecutionOrder  []string
	Errors          []*DependencyError
	Warnings        []string
	DependencyGraph map[string][]string
}

func (v *DependencyValidator) ValidateTemplate(template *TaskTemplate) *ValidationResult {
	v.mu.Lock()
	defer v.mu.Unlock()

	result := &ValidationResult{
		Valid:           true,
		ExecutionOrder:  make([]string, 0),
		Errors:          make([]*DependencyError, 0),
		Warnings:        make([]string, 0),
		DependencyGraph: make(map[string][]string),
	}

	if template == nil {
		result.Valid = false
		result.Errors = append(result.Errors, &DependencyError{
			Err: errors.New("template is nil"),
		})
		return result
	}

	stepIDs := v.validateStepIDs(template.Steps, result)
	if !result.Valid {
		return result
	}

	v.buildDependencyGraph(template.Steps, stepIDs, result.DependencyGraph, result)

	if !result.Valid {
		return result
	}

	v.detectCycles(result.DependencyGraph, result)
	if !result.Valid {
		return result
	}

	v.topologicalSort(result.DependencyGraph, stepIDs, result)

	return result
}

func (v *DependencyValidator) ValidateSteps(steps []WorkflowStep) *ValidationResult {
	template := &TaskTemplate{
		Steps: steps,
	}
	return v.ValidateTemplate(template)
}

func (v *DependencyValidator) validateStepIDs(steps []WorkflowStep, result *ValidationResult) map[string]bool {
	stepIDs := make(map[string]bool)

	for _, step := range steps {
		if step.StepID == "" {
			result.Valid = false
			result.Errors = append(result.Errors, &DependencyError{
				Err: ErrEmptyStepID,
			})
			continue
		}

		if stepIDs[step.StepID] {
			result.Valid = false
			result.Errors = append(result.Errors, &DependencyError{
				StepID: step.StepID,
				Err:    ErrDuplicateStepID,
			})
			continue
		}

		stepIDs[step.StepID] = true
	}

	return stepIDs
}

func (v *DependencyValidator) buildDependencyGraph(
	steps []WorkflowStep,
	stepIDs map[string]bool,
	graph map[string][]string,
	result *ValidationResult,
) {
	for _, step := range steps {
		if step.StepID == "" {
			continue
		}

		graph[step.StepID] = make([]string, 0)

		if step.NextStepID != "" {
			if !stepIDs[step.NextStepID] {
				result.Valid = false
				result.Errors = append(result.Errors, &DependencyError{
					StepID:     step.StepID,
					Dependency: step.NextStepID,
					Err:        ErrMissingDependency,
				})
			} else {
				graph[step.StepID] = append(graph[step.StepID], step.NextStepID)
			}
		}

		for _, agentID := range step.AgentIDs {
			if agentID == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("step %q has empty agent ID", step.StepID))
			}
		}
	}
}

func (v *DependencyValidator) detectCycles(
	graph map[string][]string,
	result *ValidationResult,
) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range graph {
		if !visited[node] {
			cyclePath := v.detectCycleDFS(node, graph, visited, recStack, []string{node})
			if len(cyclePath) > 0 {
				result.Valid = false
				result.Errors = append(result.Errors, &DependencyError{
					CyclePath: cyclePath,
					Err:       ErrCircularDependency,
				})
				return
			}
		}
	}
}

func (v *DependencyValidator) detectCycleDFS(
	node string,
	graph map[string][]string,
	visited map[string]bool,
	recStack map[string]bool,
	path []string,
) []string {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			cycle := v.detectCycleDFS(neighbor, graph, visited, recStack, append(path, neighbor))
			if len(cycle) > 0 {
				return cycle
			}
		} else if recStack[neighbor] {
			cycleStart := 0
			for i, n := range path {
				if n == neighbor {
					cycleStart = i
					break
				}
			}
			return append(path[cycleStart:], neighbor)
		}
	}

	recStack[node] = false
	return nil
}

func (v *DependencyValidator) topologicalSort(
	graph map[string][]string,
	stepIDs map[string]bool,
	result *ValidationResult,
) {
	inDegree := make(map[string]int)
	for node := range graph {
		inDegree[node] = 0
	}

	for _, deps := range graph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	queue := make([]string, 0)
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	sorted := make([]string, 0, len(graph))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(sorted) != len(graph) {
		result.Valid = false
		result.Errors = append(result.Errors, &DependencyError{
			Err: ErrCircularDependency,
		})
		return
	}

	result.ExecutionOrder = v.orderByTemplateSequence(sorted, stepIDs)
}

func (v *DependencyValidator) orderByTemplateSequence(
	topoOrder []string,
	stepIDs map[string]bool,
) []string {
	orderMap := make(map[string]int)
	for i, id := range topoOrder {
		orderMap[id] = i
	}

	result := make([]string, 0, len(topoOrder))
	visited := make(map[string]bool)

	for _, id := range topoOrder {
		if !visited[id] {
			result = append(result, id)
			visited[id] = true
		}
	}

	for id := range stepIDs {
		if !visited[id] {
			result = append(result, id)
		}
	}

	return result
}

func (v *DependencyValidator) GetExecutionOrder(template *TaskTemplate) ([]string, error) {
	result := v.ValidateTemplate(template)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return nil, result.Errors[0]
		}
		return nil, errors.New("validation failed")
	}
	return result.ExecutionOrder, nil
}

func (v *DependencyValidator) GetDependencies(stepID string, template *TaskTemplate) ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if template == nil {
		return nil, errors.New("template is nil")
	}

	for _, step := range template.Steps {
		if step.StepID == stepID {
			deps := make([]string, 0)
			if step.NextStepID != "" {
				deps = append(deps, step.NextStepID)
			}
			return deps, nil
		}
	}

	return nil, fmt.Errorf("%w: step %q not found", ErrInvalidStepReference, stepID)
}

func (v *DependencyValidator) GetDependents(stepID string, template *TaskTemplate) ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if template == nil {
		return nil, errors.New("template is nil")
	}

	dependents := make([]string, 0)
	stepExists := false

	for _, step := range template.Steps {
		if step.StepID == stepID {
			stepExists = true
		}
		if step.NextStepID == stepID {
			dependents = append(dependents, step.StepID)
		}
	}

	if !stepExists {
		return nil, fmt.Errorf("%w: step %q not found", ErrInvalidStepReference, stepID)
	}

	return dependents, nil
}

func (v *DependencyValidator) HasCycle(template *TaskTemplate) (bool, []string) {
	result := v.ValidateTemplate(template)
	for _, err := range result.Errors {
		if errors.Is(err.Err, ErrCircularDependency) {
			return true, err.CyclePath
		}
	}
	return false, nil
}
