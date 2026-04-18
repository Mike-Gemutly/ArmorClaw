// Package browser provides the bridge-local handler for browser operations.
//
// The browser_execute handler accepts a BrowserIntent artifact, dispatches
// the requested action through the existing browser Client (Jetski), and
// returns a BrowserResult artifact.
package browser

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/armorclaw/bridge/pkg/capability"
)

// HandlerName is the registry key used to register this handler.
const HandlerName = "browser_execute"

// browserRequest wraps the fields that arrive inside the StepConfig JSON.
// The StepExecutor passes the full step config to bridge-local handlers so
// we extract the browser-specific fields here.
type browserRequest struct {
	// BrowserIntent fields embedded at top level for flexibility.
	URL        string   `json:"url"`
	Action     string   `json:"action"`
	FormFields []string `json:"form_fields,omitempty"`

	// Allow nested intent for callers that wrap in an "intent" object.
	Intent *capability.BrowserIntent `json:"intent,omitempty"`
}

// Handler implements the secretary.BridgeLocalHandler function signature.
// It is exposed as a public function so the setup wiring can register it
// without importing this package from pkg/secretary (avoids cycles).
func Handler(client *Client) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		var req browserRequest
		if err := json.Unmarshal(config, &req); err != nil {
			return nil, fmt.Errorf("browser_execute: parse config: %w", err)
		}

		intent := req.Intent
		if intent == nil {
			intent = &capability.BrowserIntent{
				URL:        req.URL,
				Action:     req.Action,
				FormFields: req.FormFields,
			}
		}

		if err := intent.Validate(); err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		result, err := dispatchAction(ctx, client, intent)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("browser_execute: invalid result: %w", err)
		}

		raw, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: marshal result: %w", err)
		}
		return raw, nil
	}
}

func dispatchAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	switch intent.Action {
	case "navigate":
		return navigateAction(ctx, client, intent)
	case "fill":
		return fillAction(ctx, client, intent)
	case "extract":
		return extractAction(ctx, client, intent)
	case "screenshot":
		return screenshotAction(ctx, client, intent)
	case "workflow":
		return workflowAction(ctx, client, intent)
	default:
		return nil, fmt.Errorf("unsupported browser action: %s", intent.Action)
	}
}

func navigateAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	resp, err := client.Navigate(ctx, ServiceNavigateCommand{URL: intent.URL})
	if err != nil {
		return nil, err
	}
	return serviceResponseToBrowserResult(intent.URL, resp), nil
}

func fillAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	var fields []ServiceFillField
	for _, ff := range intent.FormFields {
		fields = append(fields, ServiceFillField{Selector: ff})
	}
	resp, err := client.Fill(ctx, ServiceFillCommand{Fields: fields})
	if err != nil {
		return nil, err
	}
	return serviceResponseToBrowserResult(intent.URL, resp), nil
}

func extractAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	var fields []ServiceExtractField
	for _, ff := range intent.FormFields {
		fields = append(fields, ServiceExtractField{Selector: ff})
	}
	resp, err := client.Extract(ctx, ServiceExtractCommand{Fields: fields})
	if err != nil {
		return nil, err
	}
	result := serviceResponseToBrowserResult(intent.URL, resp)
	if resp != nil && resp.Data != nil {
		for k, v := range resp.Data {
			result.ExtractedData = append(result.ExtractedData, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return result, nil
}

func screenshotAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	resp, err := client.Screenshot(ctx, ServiceScreenshotCommand{})
	if err != nil {
		return nil, err
	}
	result := serviceResponseToBrowserResult(intent.URL, resp)
	if resp != nil && resp.Screenshot != "" {
		result.Screenshots = append(result.Screenshots, resp.Screenshot)
	}
	return result, nil
}

func workflowAction(ctx context.Context, client *Client, intent *capability.BrowserIntent) (*capability.BrowserResult, error) {
	var steps []ServiceWorkflowStep
	for _, action := range intent.FormFields {
		steps = append(steps, ServiceWorkflowStep{Action: action})
	}
	steps = append([]ServiceWorkflowStep{{Action: "navigate", URL: intent.URL}}, steps...)

	resp, err := client.Workflow(ctx, ServiceWorkflowCommand{Steps: steps})
	if err != nil {
		return nil, err
	}
	result := &capability.BrowserResult{URL: intent.URL}
	if resp != nil && resp.Data != nil {
		for _, step := range resp.Data.Steps {
			if step.Data != nil {
				for k, v := range step.Data {
					result.ExtractedData = append(result.ExtractedData, fmt.Sprintf("%s=%v", k, v))
				}
			}
			if step.Screenshot != "" {
				result.Screenshots = append(result.Screenshots, step.Screenshot)
			}
		}
	}
	return result, nil
}

func serviceResponseToBrowserResult(url string, resp *ServiceResponse) *capability.BrowserResult {
	result := &capability.BrowserResult{URL: url}
	if resp != nil && resp.Data != nil {
		if title, ok := resp.Data["title"].(string); ok {
			result.Title = title
		}
	}
	return result
}
