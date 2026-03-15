package secretary

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencyValidator(t *testing.T) {
	validator := NewDependencyValidator()
	assert.NotNil(t, validator)
}

func TestValidateTemplate_NilTemplate(t *testing.T) {
	validator := NewDependencyValidator()
	result := validator.ValidateTemplate(nil)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "template is nil")
}

func TestValidateTemplate_EmptySteps(t *testing.T) {
	validator := NewDependencyValidator()
	template := &TaskTemplate{
		ID:    "test-template",
		Name:  "Test Template",
		Steps: []WorkflowStep{},
	}

	result := validator.ValidateTemplate(template)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateTemplate_ValidLinearWorkflow(t *testing.T) {
	validator := NewDependencyValidator()
	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step3"},
			{StepID: "step3", Type: StepAction, Name: "Step 3"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.NotEmpty(t, result.ExecutionOrder)
	assert.Contains(t, result.ExecutionOrder, "step1")
	assert.Contains(t, result.ExecutionOrder, "step2")
	assert.Contains(t, result.ExecutionOrder, "step3")
}

func TestValidateTemplate_CircularDependency(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step3"},
			{StepID: "step3", Type: StepAction, Name: "Step 3", NextStepID: "step1"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrCircularDependency))
	assert.NotEmpty(t, result.Errors[0].CyclePath)
}

func TestValidateTemplate_MissingDependency(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "nonexistent"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrMissingDependency))
	assert.Equal(t, "step1", result.Errors[0].StepID)
	assert.Equal(t, "nonexistent", result.Errors[0].Dependency)
}

func TestValidateTemplate_DuplicateStepID(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1"},
			{StepID: "step1", Type: StepAction, Name: "Duplicate Step"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrDuplicateStepID))
}

func TestValidateTemplate_EmptyStepID(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "", Type: StepAction, Name: "Empty ID Step"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.True(t, errors.Is(result.Errors[0].Err, ErrEmptyStepID))
}

func TestValidateSteps(t *testing.T) {
	validator := NewDependencyValidator()

	steps := []WorkflowStep{
		{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
		{StepID: "step2", Type: StepAction, Name: "Step 2"},
	}

	result := validator.ValidateSteps(steps)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestGetExecutionOrder(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step3"},
			{StepID: "step3", Type: StepAction, Name: "Step 3"},
		},
	}

	order, err := validator.GetExecutionOrder(template)

	require.NoError(t, err)
	assert.Len(t, order, 3)
}

func TestGetExecutionOrder_Circular(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step1"},
		},
	}

	_, err := validator.GetExecutionOrder(template)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircularDependency))
}

func TestGetDependencies(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2"},
		},
	}

	deps, err := validator.GetDependencies("step1", template)

	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "step2")
}

func TestGetDependencies_StepNotFound(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1"},
		},
	}

	_, err := validator.GetDependencies("nonexistent", template)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStepReference))
}

func TestGetDependents(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2"},
		},
	}

	dependents, err := validator.GetDependents("step2", template)

	require.NoError(t, err)
	assert.Len(t, dependents, 1)
	assert.Contains(t, dependents, "step1")
}

func TestGetDependents_StepNotFound(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1"},
		},
	}

	_, err := validator.GetDependents("nonexistent", template)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStepReference))
}

func TestHasCycle_WithCycle(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step1"},
		},
	}

	hasCycle, cyclePath := validator.HasCycle(template)

	assert.True(t, hasCycle)
	assert.NotEmpty(t, cyclePath)
}

func TestHasCycle_NoCycle(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2"},
		},
	}

	hasCycle, _ := validator.HasCycle(template)

	assert.False(t, hasCycle)
}

func TestDependencyValidator_ThreadSafety(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2"},
		},
	}

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				validator.ValidateTemplate(template)
				validator.GetExecutionOrder(template)
				validator.HasCycle(template)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDependencyError_Error(t *testing.T) {
	t.Run("with cycle path", func(t *testing.T) {
		err := &DependencyError{
			Err:       ErrCircularDependency,
			CyclePath: []string{"step1", "step2", "step1"},
		}
		assert.Contains(t, err.Error(), "circular dependency detected")
		assert.Contains(t, err.Error(), "step1")
	})

	t.Run("with dependency", func(t *testing.T) {
		err := &DependencyError{
			Err:        ErrMissingDependency,
			StepID:     "step1",
			Dependency: "step2",
		}
		assert.Contains(t, err.Error(), "missing dependency")
		assert.Contains(t, err.Error(), "step1")
		assert.Contains(t, err.Error(), "step2")
	})

	t.Run("with step only", func(t *testing.T) {
		err := &DependencyError{
			Err:    ErrEmptyStepID,
			StepID: "step1",
		}
		assert.Contains(t, err.Error(), "step ID cannot be empty")
	})
}

func TestValidationResult_DependencyGraph(t *testing.T) {
	validator := NewDependencyValidator()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Steps: []WorkflowStep{
			{StepID: "step1", Type: StepAction, Name: "Step 1", NextStepID: "step2"},
			{StepID: "step2", Type: StepAction, Name: "Step 2", NextStepID: "step3"},
			{StepID: "step3", Type: StepAction, Name: "Step 3"},
		},
	}

	result := validator.ValidateTemplate(template)

	assert.True(t, result.Valid)
	assert.NotNil(t, result.DependencyGraph)
	assert.Contains(t, result.DependencyGraph, "step1")
	assert.Contains(t, result.DependencyGraph, "step2")
	assert.Contains(t, result.DependencyGraph, "step3")
}
