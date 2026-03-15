// Package secretary tests the template engine
package secretary

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()
	assert.NotNil(t, engine)
}

func TestInstantiateTemplate_NilTemplate(t *testing.T) {
	engine := NewTemplateEngine()
	_, err := engine.InstantiateTemplate(context.Background(), nil, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrTemplateNotFound, err)
}

func TestInstantiateTemplate_MissingRequiredVariable(t *testing.T) {
	engine := NewTemplateEngine()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Variables: []byte(`{
			"type": "object",
			"properties": {
				"url": {"type": "string"},
				"username": {"type": "string"}
			},
			"required": ["url"]
		}`),
		Steps: []WorkflowStep{
			{
				StepID: "step1",
				Type:   StepAction,
				Name:   "Test Step",
			},
		},
	}

	variables := map[string]string{
		"username": "testuser",
	}

	_, err := engine.InstantiateTemplate(context.Background(), template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url")
}

func TestSubstituteString_SimpleVariable(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"url":  "https://example.com",
		"user": "testuser",
	}

	result := engine.substituteString("Navigate to ${url}", variables)
	assert.Equal(t, "Navigate to https://example.com", result)

	result = engine.substituteString("User: ${user}", variables)
	assert.Equal(t, "User: testuser", result)
}

func TestSubstituteString_WithDefault(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"url": "https://example.com",
	}

	result := engine.substituteString("User: ${user:defaultuser}", variables)
	assert.Equal(t, "User: defaultuser", result)
}

func TestSubstituteString_NestedField(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"user": `{"name": "John", "email": "john@example.com"}`,
	}

	result := engine.substituteString("${user:email}", variables)
	assert.Equal(t, "john@example.com", result)
}

func TestExtractNestedField(t *testing.T) {
	engine := NewTemplateEngine()

	jsonStr := `{"name": "John", "email": "john@example.com", "age": 30}`

	result := engine.extractNestedField(jsonStr, "name")
	assert.Equal(t, "John", result)

	result = engine.extractNestedField(jsonStr, "email")
	assert.Equal(t, "john@example.com", result)

	result = engine.extractNestedField(jsonStr, "age")
	assert.Equal(t, "30", result)

	result = engine.extractNestedField(jsonStr, "missing")
	assert.Equal(t, "", result)
}

func TestExtractVariableReferences(t *testing.T) {
	engine := NewTemplateEngine()

	result := engine.extractVariableReferences("${url}")
	assert.Equal(t, []string{"url"}, result)

	result = engine.extractVariableReferences("Visit ${url} and ${user}")
	assert.Equal(t, []string{"url", "user"}, result)

	result = engine.extractVariableReferences("No variables here")
	assert.Equal(t, []string(nil), result)
}

func TestDetectCircularReferences(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"a": "${b}",
		"b": "${c}",
		"c": "${d}",
		"d": "${a}", // Cycle!
	}

	err := engine.detectCircularReferences(variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestDetectCircularReferences_NoCycle(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"a": "${b}",
		"b": "${c}",
		"c": "final",
	}

	err := engine.detectCircularReferences(variables)
	assert.NoError(t, err)
}

func TestCollectPIIReferences(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"url": "${pii.card_number}"}`),
		},
	}

	variables := map[string]string{
		"url": "https://example.com",
	}

	refs := engine.collectPIIReferences(steps, variables)
	assert.Len(t, refs, 1)
	assert.Contains(t, refs, "pii.card_number")
}

func TestSubstituteValue(t *testing.T) {
	engine := NewTemplateEngine()

	variables := map[string]string{
		"url": "https://example.com",
	}

	result, err := engine.substituteValue("Navigate to ${url}", variables)
	require.NoError(t, err)
	assert.Equal(t, "Navigate to https://example.com", result)

	result, err = engine.substituteValue(map[string]interface{}{
		"url":  "${url}",
		"user": "static",
	}, variables)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"url":  "https://example.com",
		"user": "static",
	}, result)

	result, err = engine.substituteValue([]interface{}{"${url}", "static"}, variables)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"https://example.com", "static"}, result)

	result, err = engine.substituteValue(123, variables)
	require.NoError(t, err)
	assert.Equal(t, 123, result)
}

func TestEvaluateConditions_NoConditions(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{StepID: "step1", Type: StepAction, Name: "Test Step 1"},
		{StepID: "step2", Type: StepAction, Name: "Test Step 2"},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_AllConditionsPass(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Name:   "Test Step 1",
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "eq", "value": "approved"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Name:   "Test Step 2",
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "eq", "value": "pending"}]}`),
		},
	}

	variables := map[string]string{"workflow.status": "approved"}

	result, err := engine.evaluateConditions(context.Background(), steps, variables)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_SomeConditionsFail(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Name:   "Test Step 1",
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "eq", "value": "approved"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Name:   "Test Step 2",
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "eq", "value": "approved"}]}`),
		},
		{
			StepID: "step3",
			Type:   StepAction,
			Name:   "Test Step 3",
		},
	}

	variables := map[string]string{"workflow.status": "pending"}

	result, err := engine.evaluateConditions(context.Background(), steps, variables)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestEvaluateConditions_OperatorEq(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "eq", "value": "action"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "eq", "value": "approval"}]}`),
		},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_OperatorNeq(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "neq", "value": "approval"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "neq", "value": "action"}]}`),
		},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_OperatorIn(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.id", "operator": "in", "value": ["step1", "step3"]}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.id", "operator": "in", "value": ["step1", "step2"]}]}`),
		},
		{
			StepID: "step3",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.id", "operator": "in", "value": ["step3"]}]}`),
		},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestEvaluateConditions_OperatorNin(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.id", "operator": "nin", "value": ["step2", "step3"]}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "step.id", "operator": "nin", "value": ["step1"]}]}`),
		},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_OperatorContains(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "initiator", "operator": "contains", "value": "admin"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "subject", "operator": "contains", "value": "payment"}]}`),
		},
	}

	variables := map[string]string{"initiator": "admin_user", "subject": "payment_processing"}

	result, err := engine.evaluateConditions(context.Background(), steps, variables)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEvaluateConditions_StepTypeCondition(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Name:   "Action Step",
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "eq", "value": "action"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepCondition,
			Name:   "Condition Step",
			Config: []byte(`{"conditions": [{"field": "step.type", "operator": "eq", "value": "condition"}]}`),
		},
		{
			StepID: "step3",
			Type:   StepAction,
			Name:   "Another Action Step",
		},
	}

	result, err := engine.evaluateConditions(context.Background(), steps, nil)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestEvaluateConditions_VariableResolution(t *testing.T) {
	engine := NewTemplateEngine()

	steps := []WorkflowStep{
		{
			StepID: "step1",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "eq", "value": "approved"}, {"field": "amount", "operator": "eq", "value": "100"}]}`),
		},
		{
			StepID: "step2",
			Type:   StepAction,
			Config: []byte(`{"conditions": [{"field": "workflow.status", "operator": "neq", "value": "pending"}]}`),
		},
	}

	variables := map[string]string{"workflow.status": "approved", "amount": "100"}

	result, err := engine.evaluateConditions(context.Background(), steps, variables)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestInstantiateTemplate_Success(t *testing.T) {
	engine := NewTemplateEngine()

	template := &TaskTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Variables: []byte(`{
			"type": "object",
			"properties": {
				"url": {"type": "string", "default": "https://default.com"}
			}
		}`),
		Steps: []WorkflowStep{
			{
				StepID: "step1",
				Type:   StepAction,
				Name:   "Test Step",
				Config: []byte(`{"url": "${url}"}`),
			},
		},
	}

	variables := map[string]string{} // Empty variables should use defaults

	result, err := engine.InstantiateTemplate(context.Background(), template, variables)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, template.ID, result.TemplateID)
	assert.Len(t, result.Steps, 1)
	assert.Equal(t, "https://default.com", result.Variables["url"])
}
