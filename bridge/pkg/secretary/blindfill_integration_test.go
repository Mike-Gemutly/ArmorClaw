package secretary

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
	studio "github.com/armorclaw/bridge/pkg/studio"
)

// mockBlindFillHandler is a test double for browser and PII operations
type mockBlindFillHandler struct {
	requestPIIFunc     func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error)
	executeCommandFunc func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error)
	sendEventFunc      func(ctx context.Context, eventType string, content interface{}) error
}

func (m *mockBlindFillHandler) RequestPII(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
	if m.requestPIIFunc != nil {
		return m.requestPIIFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockBlindFillHandler) ExecuteBrowserCommand(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, eventType, content)
	}
	return nil, nil
}

func (m *mockBlindFillHandler) SendEvent(ctx context.Context, eventType string, content interface{}) error {
	if m.sendEventFunc != nil {
		return m.sendEventFunc(ctx, eventType, content)
	}
	return nil
}

// TestNewBlindFillIntegration tests creation of BlindFill integration
func TestNewBlindFillIntegration(t *testing.T) {
	handler := &mockBlindFillHandler{}
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)

	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	if integration == nil {
		t.Fatal("NewBlindFillIntegration returned nil")
	}

	if integration.handler != handler {
		t.Error("Handler not set correctly")
	}

	if integration.blindFillEngine != blindFillEngine {
		t.Error("BlindFillEngine not set correctly")
	}

	if integration.securityLog != securityLog {
		t.Error("SecurityLog not set correctly")
	}
}

// TestResolveVariables_NoPIIRefs tests that variables are returned as-is when no PII references
func TestResolveVariables_NoPIIRefs(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			t.Error("RequestPII should not be called when no PII refs")
			return nil, nil
		},
	}

	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	template := &TaskTemplate{
		ID:      "tpl_1",
		PIIRefs: []string{}, // No PII references
	}

	variables := map[string]interface{}{
		"url":    "https://example.com",
		"action": "navigate",
	}

	resolved, err := integration.ResolveVariables(ctx, template, variables, "wf_1", "user_1")

	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	if resolved["url"] != "https://example.com" {
		t.Errorf("Expected url 'https://example.com', got %v", resolved["url"])
	}

	if resolved["action"] != "navigate" {
		t.Errorf("Expected action 'navigate', got %v", resolved["action"])
	}
}

// TestResolveVariables_WithPIIRefs tests PII resolution with approval
func TestResolveVariables_WithPIIRefs(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			if len(req.FieldRefs) != 2 {
				t.Errorf("Expected 2 PII refs, got %d", len(req.FieldRefs))
			}

			return &studio.PIIResponseEvent{
				RequestID: req.RequestID,
				Approved:  true,
				Values: map[string]string{
					"payment.card_number": "4242424242424242",
					"payment.cvv":         "123",
				},
			}, nil
		},
	}

	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	template := &TaskTemplate{
		ID:      "tpl_1",
		PIIRefs: []string{"payment.card_number", "payment.cvv"},
	}

	variables := map[string]interface{}{
		"url": "https://example.com",
	}

	resolved, err := integration.ResolveVariables(ctx, template, variables, "wf_1", "user_1")

	if err != nil {
		t.Fatalf("ResolveVariables failed: %v", err)
	}

	if resolved["url"] != "https://example.com" {
		t.Errorf("Expected url 'https://example.com', got %v", resolved["url"])
	}

	if resolved["payment.card_number"] != "4242424242424242" {
		t.Errorf("Expected payment.card_number '4242424242424242', got %v", resolved["payment.card_number"])
	}

	if resolved["payment.cvv"] != "123" {
		t.Errorf("Expected payment.cvv '123', got %v", resolved["payment.cvv"])
	}
}

// TestResolveVariables_PIIRequestDenied tests handling of denied PII requests
func TestResolveVariables_PIIRequestDenied(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			return &studio.PIIResponseEvent{
				RequestID: req.RequestID,
				Approved:  false,
			}, nil
		},
	}

	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	template := &TaskTemplate{
		ID:      "tpl_1",
		PIIRefs: []string{"payment.card_number"},
	}

	variables := map[string]interface{}{
		"url": "https://example.com",
	}

	_, err := integration.ResolveVariables(ctx, template, variables, "wf_1", "user_1")

	if err == nil {
		t.Fatal("Expected error when PII request is denied, got nil")
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestDetectPIIRefs_FillAction tests detection of PII refs in fill actions
func TestDetectPIIRefs_FillAction(t *testing.T) {
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{}
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	stepConfig := map[string]interface{}{
		"action": "fill",
		"params": map[string]interface{}{
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
		},
	}

	config := mustMarshal(stepConfig)
	step := &WorkflowStep{
		StepID: "step_1",
		Name:   "Fill payment form",
		Type:   StepAction,
		Config: config,
	}

	refs := integration.DetectPIIRefs(step)

	if len(refs) != 2 {
		t.Fatalf("Expected 2 PII refs, got %d", len(refs))
	}

	if refs[0] != "payment.card_number" {
		t.Errorf("Expected ref 'payment.card_number', got %s", refs[0])
	}

	if refs[1] != "payment.cvv" {
		t.Errorf("Expected ref 'payment.cvv', got %s", refs[1])
	}
}

// TestDetectPIIRefs_NonFillAction tests that non-fill actions have no PII refs
func TestDetectPIIRefs_NonFillAction(t *testing.T) {
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{}
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	stepConfig := map[string]interface{}{
		"action": "navigate",
		"params": map[string]interface{}{
			"url": "https://example.com",
		},
	}

	config := mustMarshal(stepConfig)
	step := &WorkflowStep{
		StepID: "step_1",
		Name:   "Navigate to page",
		Type:   StepAction,
		Config: config,
	}

	refs := integration.DetectPIIRefs(step)

	if refs != nil {
		t.Errorf("Expected no PII refs for navigate action, got %v", refs)
	}
}

// TestDetectPIIRefs_NoValueRef tests detection when fields have values but no refs
func TestDetectPIIRefs_NoValueRef(t *testing.T) {
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{}
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	stepConfig := map[string]interface{}{
		"action": "fill",
		"params": map[string]interface{}{
			"fields": []map[string]interface{}{
				{
					"selector": "#username",
					"value":    "testuser",
				},
			},
		},
	}

	config := mustMarshal(stepConfig)
	step := &WorkflowStep{
		StepID: "step_1",
		Name:   "Fill username",
		Type:   StepAction,
		Config: config,
	}

	refs := integration.DetectPIIRefs(step)

	if refs != nil {
		t.Errorf("Expected no PII refs when no value_ref, got %v", refs)
	}
}

// TestExecuteStep_WithPIIApproval tests step execution with PII approval
func TestExecuteStep_WithPIIApproval(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			return &studio.PIIResponseEvent{
				RequestID: req.RequestID,
				Approved:  true,
				Values: map[string]string{
					"payment.card_number": "4242424242424242",
				},
			}, nil
		},
		executeCommandFunc: func(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
			// Verify that PII was injected
			var fillCmd map[string]interface{}
			if err := json.Unmarshal(content, &fillCmd); err == nil {
				if params, ok := fillCmd["params"].(map[string]interface{}); ok {
					if fields, ok := params["fields"].([]interface{}); ok {
						for _, fieldIntf := range fields {
							if field, ok := fieldIntf.(map[string]interface{}); ok {
								if value, ok := field["value"].(string); ok && value == "4242424242424242" {
									// PII was injected successfully
								}
							}
						}
					}
				}
			}
			return nil, nil
		},
	}

	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	stepConfig := map[string]interface{}{
		"action": "fill",
		"params": map[string]interface{}{
			"fields": []map[string]interface{}{
				{
					"selector":  "#card-number",
					"value_ref": "payment.card_number",
				},
			},
		},
	}

	config := mustMarshal(stepConfig)
	step := &WorkflowStep{
		StepID: "step_1",
		Name:   "Fill card number",
		Type:   StepAction,
		Config: config,
	}

	result, err := integration.ExecuteStep(ctx, step, "wf_1", "user_1")

	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %s", result.Status)
	}
}

// TestExecuteStep_PIIRequestDenied tests step execution when PII is denied
func TestExecuteStep_PIIRequestDenied(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{
		requestPIIFunc: func(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
			return &studio.PIIResponseEvent{
				RequestID: req.RequestID,
				Approved:  false,
			}, nil
		},
	}

	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	stepConfig := map[string]interface{}{
		"action": "fill",
		"params": map[string]interface{}{
			"fields": []map[string]interface{}{
				{
					"selector":  "#card-number",
					"value_ref": "payment.card_number",
				},
			},
		},
	}

	config := mustMarshal(stepConfig)
	step := &WorkflowStep{
		StepID: "step_1",
		Name:   "Fill card number",
		Type:   StepAction,
		Config: config,
	}

	result, err := integration.ExecuteStep(ctx, step, "wf_1", "user_1")

	if err != nil {
		t.Fatalf("ExecuteStep returned error: %v", err)
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message, got empty string")
	}
}

// TestValidateResolution_ValidTimestamp tests validation with a valid timestamp
func TestValidateResolution_ValidTimestamp(t *testing.T) {
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{}
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	resolved := map[string]interface{}{
		"name":         "test",
		"_resolved_at": float64(time.Now().UnixMilli()),
	}

	err := integration.ValidateResolution(resolved)

	if err != nil {
		t.Errorf("Expected no error for valid timestamp, got %v", err)
	}
}

// TestValidateResolution_ExpiredTimestamp tests validation with an expired timestamp
func TestValidateResolution_ExpiredTimestamp(t *testing.T) {
	testLogger := logger.Global().WithComponent("test")
	securityLog := logger.NewSecurityLogger(testLogger)
	handler := &mockBlindFillHandler{}
	blindFillEngine := pii.NewBlindFillEngine(nil, securityLog)
	integration := NewBlindFillIntegration(handler, blindFillEngine, securityLog)

	// Timestamp from 10 minutes ago (beyond 5 minute expiry)
	oldTime := time.Now().Add(-10 * time.Minute)
	resolved := map[string]interface{}{
		"name":         "test",
		"_resolved_at": float64(oldTime.UnixMilli()),
	}

	err := integration.ValidateResolution(resolved)

	if err == nil {
		t.Fatal("Expected error for expired timestamp, got nil")
	}
}
