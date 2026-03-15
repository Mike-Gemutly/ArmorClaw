// Package secretary provides BlindFill integration for workflow PII resolution.
// This package integrates Secretary workflows with existing BlindFill and PII injection
// systems for custodial data handling.
package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
	studio "github.com/armorclaw/bridge/pkg/studio"
	"log/slog"
)

// BlindFillIntegration handles PII resolution and injection for workflow steps.
// It integrates with existing PIIResolver, BlindFillEngine, and PIIInjector
// while enforcing approval flows and logging all PII access.
type BlindFillIntegration struct {
	handler         BrowserHandler
	blindFillEngine *pii.BlindFillEngine
	securityLog     *logger.SecurityLogger
	auditLogger     *audit.CriticalOperationLogger
	log             *logger.Logger
}

// NewBlindFillIntegration creates a new BlindFill integration instance.
func NewBlindFillIntegration(
	handler BrowserHandler,
	blindFillEngine *pii.BlindFillEngine,
	securityLog *logger.SecurityLogger,
) *BlindFillIntegration {
	return &BlindFillIntegration{
		handler:         handler,
		blindFillEngine: blindFillEngine,
		securityLog:     securityLog,
		log:             logger.Global().WithComponent("blindfill_integration"),
	}
}

// SetAuditLogger sets the audit logger for compliance tracking.
func (bfi *BlindFillIntegration) SetAuditLogger(auditLogger *audit.CriticalOperationLogger) {
	bfi.auditLogger = auditLogger
}

// ResolveVariables resolves template variable references to actual PII values.
// It requests PII approval through the store and returns resolved values.
//
// This method:
// 1. Detects PII references in workflow variables
// 2. Requests approval via HITL flow
// 3. Returns resolved values or error if denied
// 4. Logs all PII access (field names only)
func (bfi *BlindFillIntegration) ResolveVariables(
	ctx context.Context,
	template *TaskTemplate,
	variables map[string]interface{},
	workflowID string,
	requestedBy string,
) (map[string]interface{}, error) {
	// Check for PII references in template
	if len(template.PIIRefs) == 0 {
		// No PII references, return variables as-is
		return variables, nil
	}

	// Log PII request attempt
	bfi.log.Info("pii_resolution_requested",
		"workflow_id", workflowID,
		"pii_refs_count", len(template.PIIRefs),
		"requested_by", requestedBy,
		"pii_fields", template.PIIRefs, // Field names only
	)

	// Create PII request event
	requestID := pii.GenerateRequestID()
	piiReq := &studio.PIIRequestEvent{
		RequestID: requestID,
		FieldRefs: template.PIIRefs,
		Context:   fmt.Sprintf("Workflow %s requires PII for template %s", workflowID, template.ID),
		Timeout:   300, // 5 minutes
	}

	// Request PII approval via handler
	piiResp, err := bfi.handler.RequestPII(ctx, piiReq)
	if err != nil {
		bfi.log.Error("pii_request_failed",
			"workflow_id", workflowID,
			"request_id", requestID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("PII request failed: %w", err)
	}

	// Check if approved
	if !piiResp.Approved {
		bfi.log.Warn("pii_request_denied",
			"workflow_id", workflowID,
			"request_id", requestID,
		)
		return nil, fmt.Errorf("PII access denied for workflow %s", workflowID)
	}

	// Merge resolved PII values into variables
	resolved := make(map[string]interface{})
	for k, v := range variables {
		resolved[k] = v
	}

	// Add PII values (already approved by user)
	for key, value := range piiResp.Values {
		resolved[key] = value
	}

	// Log successful resolution (field names only, never values)
	bfi.log.Info("pii_resolution_success",
		"workflow_id", workflowID,
		"request_id", requestID,
		"fields_count", len(piiResp.Values),
	)

	// Security logging
	if bfi.securityLog != nil {
		bfi.securityLog.LogPIIAccessGranted(ctx, requestID, template.Name, requestedBy, template.PIIRefs,
			slog.String("workflow_id", workflowID),
		)
	}

	return resolved, nil
}

// DetectPIIRefs checks workflow step config for PII references.
// It looks for "value_ref" fields which indicate BlindFill references.
func (bfi *BlindFillIntegration) DetectPIIRefs(step *WorkflowStep) []string {
	var refs []string

	var stepConfig struct {
		Action string                 `json:"action"`
		Params map[string]interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
		bfi.log.Warn("step_config_parse_failed",
			"step_id", step.StepID,
			"error", err.Error(),
		)
		return nil
	}

	// Only check "fill" actions for PII references
	if stepConfig.Action != "fill" {
		return nil
	}

	// Extract fields from params
	params := stepConfig.Params
	if params == nil {
		return nil
	}

	fieldsInterface, ok := params["fields"]
	if !ok {
		return nil
	}

	fieldsSlice, ok := fieldsInterface.([]interface{})
	if !ok {
		return nil
	}

	// Check each field for value_ref
	for _, fieldIntf := range fieldsSlice {
		fieldMap, ok := fieldIntf.(map[string]interface{})
		if !ok {
			continue
		}

		if valueRef, ok := fieldMap["value_ref"].(string); ok && valueRef != "" {
			refs = append(refs, valueRef)
		}
	}

	return refs
}

// ExecuteStep executes a workflow step with PII resolution integration.
// It wraps BrowserIntegration.ExecuteStep with automatic PII resolution.
func (bfi *BlindFillIntegration) ExecuteStep(
	ctx context.Context,
	step *WorkflowStep,
	workflowID string,
	requestedBy string,
) (*StepExecutionResult, error) {
	// Detect PII references in this step
	piiRefs := bfi.DetectPIIRefs(step)

	if len(piiRefs) == 0 {
		// No PII references, delegate to browser integration directly
		browserIntegration := NewBrowserIntegration(bfi.handler)
		return browserIntegration.ExecuteStep(ctx, step)
	}

	// PII references found - request approval
	bfi.log.Info("pii_refs_detected_in_step",
		"workflow_id", workflowID,
		"step_id", step.StepID,
		"pii_refs", piiRefs,
	)

	// Create PII request
	requestID := pii.GenerateRequestID()
	piiReq := &studio.PIIRequestEvent{
		RequestID: requestID,
		FieldRefs: piiRefs,
		Context:   fmt.Sprintf("Step %s in workflow %s requires PII", step.StepID, workflowID),
		Timeout:   300, // 5 minutes
	}

	// Request PII approval
	piiResp, err := bfi.handler.RequestPII(ctx, piiReq)
	if err != nil {
		bfi.log.Error("pii_request_failed",
			"workflow_id", workflowID,
			"step_id", step.StepID,
			"request_id", requestID,
			"error", err.Error(),
		)
		return &StepExecutionResult{
			Status: "error",
			Error:  fmt.Sprintf("PII request failed: %v", err),
			StepID: step.StepID,
		}, nil
	}

	// Check if approved
	if !piiResp.Approved {
		bfi.log.Warn("pii_request_denied",
			"workflow_id", workflowID,
			"step_id", step.StepID,
			"request_id", requestID,
		)
		return &StepExecutionResult{
			Status: "error",
			Error:  "PII access denied for this step",
			StepID: step.StepID,
		}, nil
	}

	// Log successful PII access (field names only)
	bfi.log.Info("pii_access_granted_for_step",
		"workflow_id", workflowID,
		"step_id", step.StepID,
		"request_id", requestID,
		"fields_count", len(piiResp.Values),
	)

	// Security logging
	if bfi.securityLog != nil {
		bfi.securityLog.LogPIIAccessGranted(ctx, requestID, step.Name, requestedBy, piiRefs,
			slog.String("workflow_id", workflowID),
		)
	}

	// Inject PII values into step config
	modifiedConfig := bfi.injectPIIIntoConfig(step.Config, piiResp.Values)

	// Execute step with modified config
	browserIntegration := NewBrowserIntegration(bfi.handler)
	modifiedStep := *step // Copy step
	modifiedStep.Config = modifiedConfig

	return browserIntegration.ExecuteStep(ctx, &modifiedStep)
}

// injectPIIIntoConfig replaces value_ref fields with actual PII values.
// This modifies the step config JSON to include resolved PII values.
func (bfi *BlindFillIntegration) injectPIIIntoConfig(
	config json.RawMessage,
	piiValues map[string]string,
) json.RawMessage {
	var stepConfig map[string]interface{}
	if err := json.Unmarshal(config, &stepConfig); err != nil {
		bfi.log.Error("config_unmarshal_failed",
			"error", err.Error(),
		)
		return config // Return original on error
	}

	// Check for params
	params, ok := stepConfig["params"].(map[string]interface{})
	if !ok {
		return config
	}

	// Check for fields
	fieldsIntf, ok := params["fields"]
	if !ok {
		return config
	}

	fieldsSlice, ok := fieldsIntf.([]interface{})
	if !ok {
		return config
	}

	// Replace value_ref with actual values
	for _, fieldIntf := range fieldsSlice {
		field, ok := fieldIntf.(map[string]interface{})
		if !ok {
			continue
		}

		if valueRef, ok := field["value_ref"].(string); ok && valueRef != "" {
			if actualValue, exists := piiValues[valueRef]; exists {
				field["value"] = actualValue
				delete(field, "value_ref")
			}
		}
	}

	// Re-marshal modified config
	modified, err := json.Marshal(stepConfig)
	if err != nil {
		bfi.log.Error("config_marshal_failed",
			"error", err.Error(),
		)
		return config // Return original on error
	}

	return modified
}

// ValidateResolution checks if resolved variables are still valid (not expired).
func (bfi *BlindFillIntegration) ValidateResolution(resolved map[string]interface{}) error {
	// Check if we have a timestamp to validate
	ts, ok := resolved["_resolved_at"]
	if !ok {
		return nil // No timestamp, can't validate
	}

	tsFloat, ok := ts.(float64)
	if !ok {
		return nil // Invalid timestamp format, skip validation
	}

	// Convert to time.Time
	resolvedAt := time.UnixMilli(int64(tsFloat))
	expiredAt := resolvedAt.Add(5 * time.Minute) // 5 minute expiry

	if time.Now().After(expiredAt) {
		return fmt.Errorf("PII resolution expired at %s", expiredAt.Format(time.RFC3339))
	}

	return nil
}
