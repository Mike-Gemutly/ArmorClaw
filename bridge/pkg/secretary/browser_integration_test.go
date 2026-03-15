package secretary

import (
	"context"
	"encoding/json"
	"testing"

	studio "github.com/armorclaw/bridge/pkg/studio"
)

// mockBrowserHandler is a test double for browser automation
type mockBrowserHandler struct {
	executeCommandFunc func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error)
	requestPIIFunc     func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error)
	sendEventFunc      func(ctx context.Context, eventType string, content interface{}) error
}

func (m *mockBrowserHandler) ExecuteBrowserCommand(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, eventType, content)
	}
	return nil, nil
}

func (m *mockBrowserHandler) RequestPII(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
	if m.requestPIIFunc != nil {
		return m.requestPIIFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockBrowserHandler) SendEvent(ctx context.Context, eventType string, content interface{}) error {
	if m.sendEventFunc != nil {
		return m.sendEventFunc(ctx, eventType, content)
	}
	return nil
}

// TestBrowserIntegration_Navigate tests basic navigation functionality
func TestBrowserIntegration_Navigate(t *testing.T) {
	ctx := context.Background()
	handler := &mockBrowserHandler{
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			if eventType != "com.armorclaw.browser.navigate" {
				t.Errorf("Expected navigate event, got %s", eventType)
			}
			return map[string]interface{}{
				"url":   "https://example.com",
				"title": "Example Domain",
			}, nil
		},
		sendEventFunc: func(ctx context.Context, eventType string, content interface{}) error {
			return nil
		},
	}

	integration := NewBrowserIntegration(handler)

	cmd := map[string]interface{}{
		"url": "https://example.com",
	}

	result, err := integration.Navigate(ctx, cmd)
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if result["url"] != "https://example.com" {
		t.Errorf("Expected url 'https://example.com', got %v", result["url"])
	}
}

// TestBrowserIntegration_FillWithPII tests PII resolution in fill operations
func TestBrowserIntegration_FillWithPII(t *testing.T) {
	ctx := context.Background()
	handler := &mockBrowserHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			return &studio.PIIResponseEvent{
				RequestID: "test_req_123",
				Approved:  true,
				Values: map[string]string{
					"payment.card_number": "4242424242424242",
					"payment.cvv":         "123",
				},
			}, nil
		},
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			// Verify PII values were injected
			var fillCmd struct {
				Fields []struct {
					Selector string `json:"selector"`
					Value    string `json:"value"`
				} `json:"fields"`
			}
			if err := json.Unmarshal(content, &fillCmd); err != nil {
				t.Fatalf("Failed to unmarshal fill command: %v", err)
			}

			// Check that card number was resolved
			foundCardNum := false
			for _, field := range fillCmd.Fields {
				if field.Selector == "#card-number" && field.Value == "4242424242424242" {
					foundCardNum = true
				}
			}

			if !foundCardNum {
				t.Error("PII card number was not properly injected")
			}

			return nil, nil
		},
	}

	integration := NewBrowserIntegration(handler)

	cmd := map[string]interface{}{
		"fields": []map[string]interface{}{
			{
				"selector":  "#card-number",
				"value_ref": "payment.card_number",
			},
			{
				"selector":  "#cvv",
				"value_ref": "payment.cvv",
			},
		},
	}

	_, err := integration.Fill(ctx, cmd)
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}
}

// TestBrowserIntegration_Click tests click functionality
func TestBrowserIntegration_Click(t *testing.T) {
	ctx := context.Background()
	handler := &mockBrowserHandler{
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			if eventType != "com.armorclaw.browser.click" {
				t.Errorf("Expected click event, got %s", eventType)
			}
			return nil, nil
		},
	}

	integration := NewBrowserIntegration(handler)

	cmd := map[string]interface{}{
		"selector": "#submit-button",
	}

	_, err := integration.Click(ctx, cmd)
	if err != nil {
		t.Fatalf("Click failed: %v", err)
	}
}

// TestBrowserIntegration_Extract tests data extraction
func TestBrowserIntegration_Extract(t *testing.T) {
	ctx := context.Background()
	handler := &mockBrowserHandler{
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			if eventType != "com.armorclaw.browser.extract" {
				t.Errorf("Expected extract event, got %s", eventType)
			}
			return map[string]interface{}{
				"fields": map[string]string{
					"price": "$19.99",
					"title": "Test Product",
				},
			}, nil
		},
	}

	integration := NewBrowserIntegration(handler)

	cmd := map[string]interface{}{
		"fields": []map[string]interface{}{
			{"name": "price", "selector": ".price"},
			{"name": "title", "selector": "h1"},
		},
	}

	result, err := integration.Extract(ctx, cmd)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	fields, ok := result["fields"].(map[string]string)
	if !ok {
		t.Fatalf("Expected fields map, got %T", result["fields"])
	}

	if fields["price"] != "$19.99" {
		t.Errorf("Expected price '$19.99', got %s", fields["price"])
	}
}

// TestBrowserIntegration_ExecuteStep tests workflow step execution
func TestBrowserIntegration_ExecuteStep(t *testing.T) {
	ctx := context.Background()
	handler := &mockBrowserHandler{
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			if eventType == "com.armorclaw.browser.navigate" {
				return map[string]interface{}{"url": "https://example.com"}, nil
			}
			return nil, nil
		},
	}

	integration := NewBrowserIntegration(handler)

	stepConfig := map[string]interface{}{
		"action":  "navigate",
		"url":     "https://example.com",
		"options": map[string]interface{}{"waitUntil": "load"},
	}

	result, err := integration.ExecuteStep(ctx, &WorkflowStep{
		StepID: "step_1",
		Type:   StepAction,
		Config: mustMarshal(stepConfig),
	})

	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected success status, got %s", result.Status)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
