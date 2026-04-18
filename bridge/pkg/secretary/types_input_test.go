package secretary

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowStep_InputField_BackwardCompat_NoInput(t *testing.T) {
	jsonData := `{
		"step_id": "step_1",
		"order": 0,
		"type": "action",
		"name": "Browse Website"
	}`

	var step WorkflowStep
	err := json.Unmarshal([]byte(jsonData), &step)
	require.NoError(t, err)
	assert.Equal(t, "step_1", step.StepID)
	assert.Nil(t, step.Input, "Input should be nil when omitted from JSON")
}

func TestWorkflowStep_InputField_WithInput(t *testing.T) {
	jsonData := `{
		"step_id": "step_2",
		"order": 1,
		"type": "action",
		"name": "Use Previous Data",
		"input": {
			"order_id": "{{steps.step_1.data.order_id}}",
			"total": "{{steps.step_1.data.total}}"
		}
	}`

	var step WorkflowStep
	err := json.Unmarshal([]byte(jsonData), &step)
	require.NoError(t, err)
	assert.Equal(t, "step_2", step.StepID)
	require.NotNil(t, step.Input, "Input should be non-nil when present in JSON")
	assert.Equal(t, "{{steps.step_1.data.order_id}}", step.Input["order_id"])
	assert.Equal(t, "{{steps.step_1.data.total}}", step.Input["total"])
}

func TestWorkflowStep_InputField_EmptyInput(t *testing.T) {
	jsonData := `{
		"step_id": "step_3",
		"order": 2,
		"type": "action",
		"name": "Empty Input Step",
		"input": {}
	}`

	var step WorkflowStep
	err := json.Unmarshal([]byte(jsonData), &step)
	require.NoError(t, err)
	assert.Equal(t, "step_3", step.StepID)
	require.NotNil(t, step.Input, "Input should be non-nil (empty map) when explicitly set to {}")
	assert.Empty(t, step.Input)
}

func TestWorkflowStep_InputField_Omitempty(t *testing.T) {
	step := WorkflowStep{
		StepID: "step_1",
		Order:  0,
		Type:   StepAction,
		Name:   "First Step",
	}

	data, err := json.Marshal(step)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"input"`, "omitempty should exclude nil Input from JSON output")
}

func TestWorkflowStep_InputField_Omitempty_WithValues(t *testing.T) {
	step := WorkflowStep{
		StepID: "step_2",
		Order:  1,
		Type:   StepAction,
		Name:   "Second Step",
		Input: map[string]any{
			"customer_email": "{{steps.step_1.data.email}}",
		},
	}

	data, err := json.Marshal(step)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"input"`)
	assert.Contains(t, string(data), `"customer_email"`)
	assert.Contains(t, string(data), "{{steps.step_1.data.email}}")
}

func TestWorkflowStep_InputField_TemplateVarSyntax(t *testing.T) {
	template := &TaskTemplate{
		ID:   "tpl-order-flow",
		Name: "Order Flow",
		Steps: []WorkflowStep{
			{StepID: "step_1", Order: 0, Type: StepAction, Name: "Place Order"},
			{
				StepID: "step_2",
				Order:  1,
				Type:   StepAction,
				Name:   "Confirm Order",
				Input: map[string]any{
					"order_ref": "{{steps.step_1.data.order_id}}",
					"amount":    "{{steps.step_1.data.total}}",
					"nested": map[string]any{
						"key": "{{steps.step_1.data.nested_value}}",
					},
				},
			},
		},
	}

	data, err := json.Marshal(template)
	require.NoError(t, err)

	var parsed TaskTemplate
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Nil(t, parsed.Steps[0].Input, "First step should have nil Input")
	require.NotNil(t, parsed.Steps[1].Input, "Second step should have Input")
	assert.Equal(t, "{{steps.step_1.data.order_id}}", parsed.Steps[1].Input["order_ref"])

	nested, ok := parsed.Steps[1].Input["nested"].(map[string]interface{})
	require.True(t, ok, "nested should be a map")
	assert.Equal(t, "{{steps.step_1.data.nested_value}}", nested["key"])
}
