// Package secretary provides template engine for instantiating task templates
// with variable substitution and conditional branching logic.
package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrTemplateNotFound          = errors.New("template not found")
	ErrMissingRequiredVariable   = errors.New("missing required variable")
	ErrCircularVariableReference = errors.New("circular variable reference detected")
	ErrInvalidTemplateStructure  = errors.New("invalid template structure")
)

type InstantiatedTemplate struct {
	TemplateID     string            `json:"template_id"`
	Variables      map[string]string `json:"variables"`
	Steps          []WorkflowStep    `json:"steps"`
	PIIRefs        []string          `json:"pii_refs,omitempty"`
	InstantiatedAt int64             `json:"instantiated_at"`
}

type TemplateEngine struct{}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{}
}

func (e *TemplateEngine) InstantiateTemplate(
	ctx context.Context,
	template *TaskTemplate,
	variables map[string]string,
) (*InstantiatedTemplate, error) {
	if template == nil {
		return nil, ErrTemplateNotFound
	}

	if err := e.validateTemplate(template); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTemplateStructure, err)
	}

	if err := e.validateVariables(template, variables); err != nil {
		return nil, fmt.Errorf("variable validation failed: %w", err)
	}

	mergedVars := e.mergeDefaults(template, variables)
	resolvedVars, err := e.resolveVariables(ctx, mergedVars)
	if err != nil {
		return nil, fmt.Errorf("variable resolution failed: %w", err)
	}

	steps, err := e.evaluateConditions(ctx, template.Steps, resolvedVars)
	if err != nil {
		return nil, fmt.Errorf("condition evaluation failed: %w", err)
	}

	instantiatedSteps, err := e.substituteVariables(steps, resolvedVars)
	if err != nil {
		return nil, fmt.Errorf("variable substitution failed: %w", err)
	}

	piiRefs := e.collectPIIReferences(instantiatedSteps, resolvedVars)

	return &InstantiatedTemplate{
		TemplateID:     template.ID,
		Variables:      resolvedVars,
		Steps:          instantiatedSteps,
		PIIRefs:        piiRefs,
		InstantiatedAt: 0,
	}, nil
}

func (e *TemplateEngine) validateTemplate(template *TaskTemplate) error {
	if template.ID == "" {
		return errors.New("template ID is required")
	}
	if template.Name == "" {
		return errors.New("template name is required")
	}
	if len(template.Steps) == 0 {
		return errors.New("template must have at least one step")
	}

	for i, step := range template.Steps {
		if step.StepID == "" {
			return fmt.Errorf("step %d: step ID is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step %d: step type is required", i)
		}
		if step.Config != nil {
			if _, err := json.Marshal(step.Config); err != nil {
				return fmt.Errorf("step %d: invalid config JSON: %w", i, err)
			}
		}
	}

	return nil
}

func (e *TemplateEngine) validateVariables(template *TaskTemplate, variables map[string]string) error {
	var varDefs map[string]interface{}
	if template.Variables != nil {
		if err := json.Unmarshal(template.Variables, &varDefs); err != nil {
			return fmt.Errorf("invalid variables schema: %w", err)
		}
	}

	if varDefs != nil {
		if required, ok := varDefs["required"].([]interface{}); ok {
			for _, req := range required {
				if key, ok := req.(string); ok {
					if _, exists := variables[key]; !exists {
						return fmt.Errorf("%w: %s", ErrMissingRequiredVariable, key)
					}
				}
			}
		}
	}

	if err := e.detectCircularReferences(variables); err != nil {
		return err
	}

	return nil
}

func (e *TemplateEngine) detectCircularReferences(variables map[string]string) error {
	graph := make(map[string][]string)
	for key, value := range variables {
		refs := e.extractVariableReferences(value)
		graph[key] = refs
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for key := range graph {
		if err := e.detectCycle(key, graph, visited, recStack); err != nil {
			return err
		}
	}

	return nil
}

func (e *TemplateEngine) detectCycle(
	key string,
	graph map[string][]string,
	visited map[string]bool,
	recStack map[string]bool,
) error {
	visited[key] = true
	recStack[key] = true

	for _, dep := range graph[key] {
		if !visited[dep] {
			if err := e.detectCycle(dep, graph, visited, recStack); err != nil {
				return err
			}
		} else if recStack[dep] {
			return fmt.Errorf("%w: %s -> %s", ErrCircularVariableReference, key, dep)
		}
	}

	recStack[key] = false
	return nil
}

func (e *TemplateEngine) extractVariableReferences(value string) []string {
	var refs []string
	re := regexp.MustCompile(`\$\{([^}:]+)(?::[^}]*)?\}`)
	matches := re.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, match[1])
		}
	}
	return refs
}

func (e *TemplateEngine) mergeDefaults(template *TaskTemplate, variables map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range variables {
		merged[k] = v
	}

	var varDefs map[string]interface{}
	if template.Variables != nil {
		if err := json.Unmarshal(template.Variables, &varDefs); err == nil && varDefs != nil {
			if props, ok := varDefs["properties"].(map[string]interface{}); ok {
				for key, prop := range props {
					if propMap, ok := prop.(map[string]interface{}); ok {
						if _, exists := merged[key]; !exists {
							if defVal, ok := propMap["default"].(string); ok {
								merged[key] = defVal
							}
						}
					}
				}
			}
		}
	}

	return merged
}

func (e *TemplateEngine) resolveVariables(ctx context.Context, variables map[string]string) (map[string]string, error) {
	resolved := make(map[string]string)
	maxIterations := len(variables) + 1

	for i := 0; i < maxIterations; i++ {
		resolved = make(map[string]string)
		for key, value := range variables {
			resolved[key] = e.substituteString(value, variables)
		}
		variables = resolved
	}

	return resolved, nil
}

func (e *TemplateEngine) substituteString(input string, variables map[string]string) string {
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := strings.TrimPrefix(match, "${")
		parts = strings.TrimSuffix(parts, "}")

		colonIndex := strings.Index(parts, ":")
		if colonIndex == -1 {
			key := parts
			if val, exists := variables[key]; exists {
				return val
			}
			return match
		}

		key := parts[:colonIndex]
		spec := parts[colonIndex+1:]

		if val, exists := variables[key]; exists && val != "" {
			if e.isJSONObject(val) {
				return e.extractNestedField(val, spec)
			}
		}

		if val, exists := variables[key]; exists {
			return val
		}

		return spec
	})
}

func (e *TemplateEngine) isJSONObject(val string) bool {
	var data interface{}
	return json.Unmarshal([]byte(val), &data) == nil
}

func (e *TemplateEngine) extractNestedField(jsonStr, fieldPath string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	parts := strings.Split(fieldPath, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			if next, exists := v[part]; exists {
				current = next
			} else {
				return ""
			}
		default:
			return ""
		}
	}

	switch v := current.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (e *TemplateEngine) evaluateConditions(
	ctx context.Context,
	steps []WorkflowStep,
	variables map[string]string,
) ([]WorkflowStep, error) {
	filtered := make([]WorkflowStep, 0, len(steps))

	for _, step := range steps {
		if len(step.Conditions) == 0 {
			filtered = append(filtered, step)
			continue
		}

		var conds []Condition
		if err := json.Unmarshal(step.Conditions, &conds); err != nil {
			return nil, fmt.Errorf("step %s: failed to parse conditions: %w", step.StepID, err)
		}

		varMap := make(map[string]interface{}, len(variables))
		for k, v := range variables {
			varMap[k] = v
		}

		evalCtx := EvaluationContext{
			Step:      &step,
			Variables: varMap,
			Timestamp: time.Now(),
		}

		allConditionsPass := true
		for _, cond := range conds {
			if !e.evaluateCondition(cond, evalCtx) {
				allConditionsPass = false
				break
			}
		}

		if allConditionsPass {
			filtered = append(filtered, step)
		}
	}

	return filtered, nil
}

func (e *TemplateEngine) evaluateCondition(cond Condition, evalCtx EvaluationContext) bool {
	var fieldValue interface{}

	switch cond.Field {
	case "step.type":
		if evalCtx.Step != nil {
			fieldValue = string(evalCtx.Step.Type)
		}
	case "step.id":
		if evalCtx.Step != nil {
			fieldValue = evalCtx.Step.StepID
		}
	case "step.name":
		if evalCtx.Step != nil {
			fieldValue = evalCtx.Step.Name
		}
	default:
		if evalCtx.Variables != nil {
			fieldValue = evalCtx.Variables[cond.Field]
		}
	}

	return e.compareValues(fieldValue, cond.Operator, cond.Value)
}

func (e *TemplateEngine) compareValues(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "==", "=":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "neq", "!=":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "in":
		if arr, ok := expected.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", v) {
					return true
				}
			}
			return false
		}
		return false
	case "nin", "not_in":
		if arr, ok := expected.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", v) {
					return false
				}
			}
			return true
		}
		return true
	case "contains":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	default:
		return false
	}
}

func (e *TemplateEngine) substituteVariables(steps []WorkflowStep, variables map[string]string) ([]WorkflowStep, error) {
	result := make([]WorkflowStep, len(steps))

	for i, step := range steps {
		stepCopy := step

		if step.Config != nil {
			var configData map[string]interface{}
			if err := json.Unmarshal(step.Config, &configData); err == nil {
				substituted, err := e.substituteValue(configData, variables)
				if err != nil {
					return nil, fmt.Errorf("step %s: %w", step.StepID, err)
				}
				if newConfig, err := json.Marshal(substituted); err == nil {
					stepCopy.Config = newConfig
				}
			}
		}

		stepCopy.Description = e.substituteString(step.Description, variables)

		result[i] = stepCopy
	}

	return result, nil
}

func (e *TemplateEngine) substituteValue(value interface{}, variables map[string]string) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return e.substituteString(v, variables), nil

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			substituted, err := e.substituteValue(val, variables)
			if err != nil {
				return nil, err
			}
			result[key] = substituted
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			substituted, err := e.substituteValue(val, variables)
			if err != nil {
				return nil, err
			}
			result[i] = substituted
		}
		return result, nil

	default:
		return value, nil
	}
}

func (e *TemplateEngine) collectPIIReferences(steps []WorkflowStep, variables map[string]string) []string {
	refs := make(map[string]bool)
	piiPattern := regexp.MustCompile(`\bpii\.[a-zA-Z_][a-zA-Z0-9_]*\b`)

	for _, step := range steps {
		if step.Config != nil {
			matches := piiPattern.FindAllString(string(step.Config), -1)
			for _, match := range matches {
				refs[match] = true
			}
		}
	}

	for _, val := range variables {
		matches := piiPattern.FindAllString(val, -1)
		for _, match := range matches {
			refs[match] = true
		}
	}

	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}
