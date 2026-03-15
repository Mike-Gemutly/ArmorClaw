package secretary

import (
	"context"
	"encoding/json"

	"github.com/armorclaw/bridge/pkg/pii"
	studio "github.com/armorclaw/bridge/pkg/studio"
)

// BrowserHandler defines the interface for browser automation operations
type BrowserHandler interface {
	ExecuteBrowserCommand(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error)
	RequestPII(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error)
	SendEvent(ctx context.Context, eventType string, content interface{}) error
}

// BrowserIntegration provides browser automation methods for Secretary workflows
type BrowserIntegration struct {
	handler BrowserHandler
}

// NewBrowserIntegration creates a new browser integration instance
func NewBrowserIntegration(handler BrowserHandler) *BrowserIntegration {
	return &BrowserIntegration{
		handler: handler,
	}
}

// Navigate to a URL
func (bi *BrowserIntegration) Navigate(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	content, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	result, err := bi.handler.ExecuteBrowserCommand(ctx, string(studio.BrowserNavigate), content)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, nil
}

// Fill form fields with PII resolution support
func (bi *BrowserIntegration) Fill(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	var fillCmd struct {
		Fields []struct {
			Selector string `json:"selector"`
			Value    string `json:"value,omitempty"`
			ValueRef string `json:"value_ref,omitempty"`
		} `json:"fields"`
	}

	content, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(content, &fillCmd); err != nil {
		return nil, err
	}

	// Check for PII references
	var piiRefs []string
	for _, field := range fillCmd.Fields {
		if field.ValueRef != "" {
			piiRefs = append(piiRefs, field.ValueRef)
		}
	}

	// Request PII approval if needed
	if len(piiRefs) > 0 {
		piiResp, err := bi.handler.RequestPII(ctx, &studio.PIIRequestEvent{
			RequestID: pii.GenerateRequestID(),
			FieldRefs: piiRefs,
			Context:   "Form fill requires sensitive fields",
			Timeout:   300,
		})

		if err != nil || !piiResp.Approved {
			return nil, &BrowserError{
				Code:    "PII_REQUEST_DENIED",
				Message: "PII request denied or timed out",
			}
		}

		// Inject PII values into fields
		for i, field := range fillCmd.Fields {
			if field.ValueRef != "" {
				if val, ok := piiResp.Values[field.ValueRef]; ok {
					fillCmd.Fields[i].Value = val
					fillCmd.Fields[i].ValueRef = ""
				}
			}
		}

		// Re-marshal with resolved values
		content, err = json.Marshal(fillCmd)
		if err != nil {
			return nil, err
		}
	}

	result, err := bi.handler.ExecuteBrowserCommand(ctx, string(studio.BrowserFill), content)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, nil
}

// Click an element
func (bi *BrowserIntegration) Click(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	content, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	result, err := bi.handler.ExecuteBrowserCommand(ctx, string(studio.BrowserClick), content)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, nil
}

// Extract data from the page
func (bi *BrowserIntegration) Extract(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	content, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	result, err := bi.handler.ExecuteBrowserCommand(ctx, string(studio.BrowserExtract), content)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, nil
}

// Wait for a condition to be met
func (bi *BrowserIntegration) Wait(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	content, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	result, err := bi.handler.ExecuteBrowserCommand(ctx, string(studio.BrowserWait), content)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, nil
}

// StepExecutionResult represents the result of a workflow step execution
type StepExecutionResult struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Error  string                 `json:"error,omitempty"`
	StepID string                 `json:"step_id"`
}

// BrowserError represents a browser operation error
type BrowserError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *BrowserError) Error() string {
	return e.Message
}

// ExecuteStep executes a workflow step with browser automation
func (bi *BrowserIntegration) ExecuteStep(ctx context.Context, step *WorkflowStep) (*StepExecutionResult, error) {
	var stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
		return nil, err
	}

	var result map[string]interface{}
	var err error

	switch stepConfig.Action {
	case "navigate":
		params := stepConfig.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		if options := stepConfig.Options; options != nil {
			for k, v := range options {
				params[k] = v
			}
		}
		result, err = bi.Navigate(ctx, params)
	case "fill":
		params := stepConfig.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		result, err = bi.Fill(ctx, params)
	case "click":
		params := stepConfig.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		result, err = bi.Click(ctx, params)
	case "extract":
		params := stepConfig.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		result, err = bi.Extract(ctx, params)
	case "wait":
		params := stepConfig.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		result, err = bi.Wait(ctx, params)
	default:
		return &StepExecutionResult{
			Status: "error",
			Error:  "Unknown action: " + stepConfig.Action,
			StepID: step.StepID,
		}, nil
	}

	if err != nil {
		return &StepExecutionResult{
			Status: "error",
			Error:  err.Error(),
			StepID: step.StepID,
		}, nil
	}

	return &StepExecutionResult{
		Status: "success",
		Data:   result,
		StepID: step.StepID,
	}, nil
}
