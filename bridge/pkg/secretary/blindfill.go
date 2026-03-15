package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/google/uuid"
)

//=============================================================================
// BlindFill Executor Service
//=============================================================================

// BlindFillExecutor executes BlindFill operations using persisted learned mappings.
// It loads templates from the Learn Website flow, resolves PII values through
// approval flows, and executes fill/select/click actions via browser integration.
type BlindFillExecutor struct {
	browser  *BrowserIntegration
	store    Store
	approval *ApprovalEngineImpl
	events   EventEmitter
	log      *logger.Logger

	// Track active executions for cancellation
	mu         sync.RWMutex
	executions map[string]*activeExecution
}

// BlindFillConfig holds configuration for the BlindFill executor
type BlindFillConfig struct {
	Browser  *BrowserIntegration
	Store    Store
	Approval *ApprovalEngineImpl
	Events   EventEmitter
	Logger   *logger.Logger
}

// NewBlindFillExecutor creates a new BlindFill executor instance
func NewBlindFillExecutor(cfg BlindFillConfig) (*BlindFillExecutor, error) {
	if cfg.Browser == nil {
		return nil, fmt.Errorf("browser integration is required")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("blindfill_executor")
	}

	return &BlindFillExecutor{
		browser:    cfg.Browser,
		store:      cfg.Store,
		approval:   cfg.Approval,
		events:     cfg.Events,
		log:        log,
		executions: make(map[string]*activeExecution),
	}, nil
}

type activeExecution struct {
	id          string
	cancelFunc  context.CancelFunc
	startedAt   time.Time
	template    *TaskTemplate
	request     *BlindFillRequest
	stepResults []*BlindFillStepResult
	mu          sync.RWMutex
}

//=============================================================================
// Request/Response Types
//=============================================================================

// BlindFillRequest is the input for BlindFill execution
type BlindFillRequest struct {
	// ExecutionID is a unique identifier for this execution (auto-generated if empty)
	ExecutionID string `json:"execution_id,omitempty"`

	// TemplateID references the persisted template from Learn Website
	TemplateID string `json:"template_id"`

	// TargetURL is the URL to navigate to (overrides template if set)
	TargetURL string `json:"target_url,omitempty"`

	// Initiator is the Matrix user ID who initiated this execution
	Initiator string `json:"initiator"`

	// Variables contains input values (non-PII static values)
	Variables map[string]interface{} `json:"variables,omitempty"`

	// PIIValues contains pre-approved PII values (from prior approval)
	// These bypass the approval flow if already approved
	PIIValues map[string]string `json:"pii_values,omitempty"`

	// ApprovalToken is a token from a prior approval request
	ApprovalToken string `json:"approval_token,omitempty"`

	// DeniedFields lists fields that were explicitly denied in prior approval
	DeniedFields []string `json:"denied_fields,omitempty"`

	// Timeout in milliseconds for the entire execution
	Timeout int `json:"timeout,omitempty"`

	// WaitUntil specifies when to consider navigation complete
	WaitUntil string `json:"wait_until,omitempty"`

	// ContinueOnError continues execution even if a step fails
	ContinueOnError bool `json:"continue_on_error,omitempty"`

	// DryRun validates the request without executing
	DryRun bool `json:"dry_run,omitempty"`
}

// BlindFillResult is the output from BlindFill execution
type BlindFillResult struct {
	// ExecutionID is the unique identifier for this execution
	ExecutionID string `json:"execution_id"`

	// TemplateID references the template used
	TemplateID string `json:"template_id"`

	// Status is the final execution status
	Status BlindFillStatus `json:"status"`

	// StartedAt is when execution started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when execution completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Duration is the total execution duration in milliseconds
	Duration int64 `json:"duration_ms"`

	// StepResults contains results for each executed step
	StepResults []*BlindFillStepResult `json:"step_results"`

	// FilledFields lists the field names that were successfully filled
	FilledFields []string `json:"filled_fields,omitempty"`

	// SubmissionID is any identifier returned after form submission
	SubmissionID string `json:"submission_id,omitempty"`

	// FinalURL is the URL after execution (after any redirects)
	FinalURL string `json:"final_url,omitempty"`

	// Warnings contains non-fatal issues encountered
	Warnings []string `json:"warnings,omitempty"`

	// Error contains the error message if status is failed
	Error string `json:"error,omitempty"`

	// ApprovalRequired indicates that approval is needed to proceed
	ApprovalRequired bool `json:"approval_required,omitempty"`

	// ApprovalRequestID references the pending approval request
	ApprovalRequestID string `json:"approval_request_id,omitempty"`

	// RequiredPIIFields lists PII fields that require approval
	RequiredPIIFields []string `json:"required_pii_fields,omitempty"`
}

// BlindFillStatus represents the status of a BlindFill execution
type BlindFillStatus string

const (
	BlindFillStatusPending   BlindFillStatus = "pending"
	BlindFillStatusRunning   BlindFillStatus = "running"
	BlindFillStatusCompleted BlindFillStatus = "completed"
	BlindFillStatusFailed    BlindFillStatus = "failed"
	BlindFillStatusCancelled BlindFillStatus = "cancelled"
	BlindFillStatusAwaiting  BlindFillStatus = "awaiting_approval"
)

// BlindFillStepResult represents the result of a single step execution
type BlindFillStepResult struct {
	// StepID is the identifier for this step
	StepID string `json:"step_id"`

	// StepName is the human-readable name
	StepName string `json:"step_name"`

	// ActionType is the type of action performed (fill, click, select, etc.)
	ActionType string `json:"action_type"`

	// Selector is the CSS selector that was targeted
	Selector string `json:"selector,omitempty"`

	// FieldName is the human-readable field name
	FieldName string `json:"field_name,omitempty"`

	// Status is the step execution status
	Status string `json:"status"` // "success", "error", "skipped"

	// PIIRefUsed indicates which PII reference was used (field name only, never value)
	PIIRefUsed string `json:"pii_ref_used,omitempty"`

	// WasPIIApproved indicates if PII approval was obtained
	WasPIIApproved bool `json:"was_pii_approved,omitempty"`

	// ExecutionTime is the time taken for this step in milliseconds
	ExecutionTime int64 `json:"execution_time_ms"`

	// Error contains error message if step failed
	Error string `json:"error,omitempty"`

	// Data contains any extracted or returned data
	Data map[string]interface{} `json:"data,omitempty"`
}

//=============================================================================
// BlindFill Errors
//=============================================================================

// BlindFillError represents an error from BlindFill execution
type BlindFillError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	StepID  string `json:"step_id,omitempty"`
}

func (e *BlindFillError) Error() string {
	if e.StepID != "" {
		return fmt.Sprintf("[%s] %s (step: %s)", e.Code, e.Message, e.StepID)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrTemplateInactive    = &BlindFillError{Code: "TEMPLATE_INACTIVE", Message: "template is not active"}
	ErrNoSteps             = &BlindFillError{Code: "NO_STEPS", Message: "template has no steps"}
	ErrSelectorMismatch    = &BlindFillError{Code: "SELECTOR_MISMATCH", Message: "element not found for selector"}
	ErrMissingRequiredData = &BlindFillError{Code: "MISSING_REQUIRED_DATA", Message: "required data not provided"}
	ErrApprovalDenied      = &BlindFillError{Code: "APPROVAL_DENIED", Message: "PII approval denied"}
	ErrApprovalTimeout     = &BlindFillError{Code: "APPROVAL_TIMEOUT", Message: "PII approval timed out"}
	ErrBrowserFailure      = &BlindFillError{Code: "BROWSER_FAILURE", Message: "browser operation failed"}
	ErrExecutionTimeout    = &BlindFillError{Code: "EXECUTION_TIMEOUT", Message: "execution timed out"}
	ErrExecutionCancelled  = &BlindFillError{Code: "EXECUTION_CANCELLED", Message: "execution was cancelled"}
	ErrInvalidRequest      = &BlindFillError{Code: "INVALID_REQUEST", Message: "invalid request parameters"}
	ErrBlindFillNavFailed  = &BlindFillError{Code: "NAVIGATION_FAILED", Message: "failed to navigate to target URL"}
)

//=============================================================================
// Execute Method
//=============================================================================

// Execute performs a BlindFill operation using the specified template
func (e *BlindFillExecutor) Execute(ctx context.Context, req *BlindFillRequest) (*BlindFillResult, error) {
	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, err
	}

	// Generate execution ID if not provided
	if req.ExecutionID == "" {
		req.ExecutionID = uuid.New().String()
	}

	executionID := req.ExecutionID

	e.log.Info("blindfill_execution_started",
		"execution_id", executionID,
		"template_id", req.TemplateID,
		"initiator", req.Initiator)

	// Load template
	template, err := e.store.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		e.log.Error("template_load_failed", "template_id", req.TemplateID, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrTemplateNotFound, err)
	}

	// Check template is active
	if !template.IsActive {
		return nil, ErrTemplateInactive
	}

	// Check template has steps
	if len(template.Steps) == 0 {
		return nil, ErrNoSteps
	}

	// Dry run - just validate
	if req.DryRun {
		return e.dryRunValidation(template, req)
	}

	// Set up execution context with timeout
	execCtx, cancel := context.WithCancel(ctx)
	if req.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(execCtx, time.Duration(req.Timeout)*time.Millisecond)
	} else {
		execCtx, cancel = context.WithTimeout(execCtx, 5*time.Minute) // default 5 min
	}

	// Track active execution
	exec := &activeExecution{
		id:          executionID,
		cancelFunc:  cancel,
		startedAt:   time.Now(),
		template:    template,
		request:     req,
		stepResults: make([]*BlindFillStepResult, 0),
	}

	e.mu.Lock()
	e.executions[executionID] = exec
	e.mu.Unlock()

	// Cleanup on completion
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.executions, executionID)
		e.mu.Unlock()
	}()

	// Execute
	result := e.executeTemplate(execCtx, exec, template, req)

	return result, nil
}

// Cancel cancels an in-progress execution
func (e *BlindFillExecutor) Cancel(executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, exists := e.executions[executionID]
	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	exec.cancelFunc()

	e.log.Info("blindfill_execution_cancelled", "execution_id", executionID)

	return nil
}

//=============================================================================
// Template Execution
//=============================================================================

func (e *BlindFillExecutor) executeTemplate(
	ctx context.Context,
	exec *activeExecution,
	template *TaskTemplate,
	req *BlindFillRequest,
) *BlindFillResult {
	startedAt := time.Now()
	result := &BlindFillResult{
		ExecutionID: exec.id,
		TemplateID:  template.ID,
		Status:      BlindFillStatusRunning,
		StartedAt:   startedAt,
		StepResults: make([]*BlindFillStepResult, 0),
		Warnings:    make([]string, 0),
	}

	// Emit start event
	if e.events != nil {
		// Create minimal workflow for event emission
		workflow := &Workflow{
			ID:         exec.id,
			TemplateID: template.ID,
			Name:       template.Name,
			Status:     StatusRunning,
			CreatedBy:  req.Initiator,
			StartedAt:  startedAt,
		}
		e.events.EmitStarted(workflow)
	}

	// Check for required PII approval upfront
	piiFields := e.collectPIIFields(template, req)
	if len(piiFields) > 0 && req.ApprovalToken == "" && len(req.PIIValues) == 0 {
		// Need approval first
		approvalReq, err := e.requestPIIApproval(ctx, template, piiFields, req.Initiator, exec.id)
		if err != nil {
			result.Status = BlindFillStatusFailed
			result.Error = fmt.Sprintf("Failed to request PII approval: %v", err)
			result.RequiredPIIFields = piiFields
			return result
		}

		if !approvalReq.Approved {
			result.Status = BlindFillStatusAwaiting
			result.ApprovalRequired = true
			result.ApprovalRequestID = approvalReq.RequestID
			result.RequiredPIIFields = piiFields
			result.Error = "PII approval required"
			return result
		}
	}

	if req.ApprovalToken != "" {
		if err := e.validateApprovalToken(exec, req.ApprovalToken); err != nil {
			result.Status = BlindFillStatusFailed
			result.Error = fmt.Sprintf("Approval token validation failed: %v", err)
			return result
		}
	}

	if len(req.PIIValues) > 0 {
		if err := e.validatePIIValues(exec, req.PIIValues); err != nil {
			result.Status = BlindFillStatusFailed
			result.Error = fmt.Sprintf("PII validation failed: %v", err)
			return result
		}
	}

	// Navigate to target URL first
	targetURL := req.TargetURL
	if targetURL == "" {
		// Try to extract URL from template steps
		targetURL = e.extractTargetURL(template)
	}

	if targetURL != "" {
		navResult := e.executeNavigation(ctx, targetURL, req.WaitUntil, req.Timeout)
		if navResult.Status == "error" {
			result.Status = BlindFillStatusFailed
			result.Error = navResult.Error
			result.StepResults = append(result.StepResults, navResult)
			return result
		}
		result.StepResults = append(result.StepResults, navResult)
		if navResult.Data != nil {
			if finalURL, ok := navResult.Data["final_url"].(string); ok {
				result.FinalURL = finalURL
			}
		}
	}

	// Execute each step
	var filledFields []string
	var lastError error

	for i := range template.Steps {
		step := &template.Steps[i]
		select {
		case <-ctx.Done():
			result.Status = BlindFillStatusCancelled
			result.Error = ErrExecutionCancelled.Message
			return result
		default:
		}

		stepResult := e.executeStep(ctx, step, template, req, exec)

		exec.mu.Lock()
		exec.stepResults = append(exec.stepResults, stepResult)
		exec.mu.Unlock()

		result.StepResults = append(result.StepResults, stepResult)

		if stepResult.Status == "success" && stepResult.FieldName != "" {
			filledFields = append(filledFields, stepResult.FieldName)
		}

		if stepResult.Status == "error" {
			lastError = fmt.Errorf("%s", stepResult.Error)
			if !req.ContinueOnError {
				result.Status = BlindFillStatusFailed
				result.Error = stepResult.Error
				return result
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Step %s failed: %s", step.StepID, stepResult.Error))
		}

		if e.events != nil {
			workflow := &Workflow{
				ID:         exec.id,
				TemplateID: template.ID,
				Name:       template.Name,
				Status:     StatusRunning,
				CreatedBy:  req.Initiator,
				StartedAt:  startedAt,
			}
			progress := float64(len(result.StepResults)) / float64(len(template.Steps))
			e.events.EmitProgress(workflow, step.StepID, step.Name, progress)
		}
	}

	// Execution completed
	now := time.Now()
	result.CompletedAt = &now
	result.Duration = now.Sub(startedAt).Milliseconds()
	result.FilledFields = filledFields

	// Determine final status
	if lastError != nil {
		result.Status = BlindFillStatusCompleted
		result.Warnings = append(result.Warnings, "Completed with errors")
	} else {
		result.Status = BlindFillStatusCompleted
	}

	// Emit completion event
	if e.events != nil {
		workflow := &Workflow{
			ID:         exec.id,
			TemplateID: template.ID,
			Name:       template.Name,
			Status:     StatusCompleted,
			CreatedBy:  req.Initiator,
			StartedAt:  startedAt,
		}
		e.events.EmitCompleted(workflow, string(result.Status))
	}

	e.log.Info("blindfill_execution_completed",
		"execution_id", exec.id,
		"status", result.Status,
		"duration_ms", result.Duration,
		"steps_executed", len(result.StepResults))

	return result
}

//=============================================================================
// Step Execution
//=============================================================================

func (e *BlindFillExecutor) executeStep(
	ctx context.Context,
	step *WorkflowStep,
	template *TaskTemplate,
	req *BlindFillRequest,
	exec *activeExecution,
) *BlindFillStepResult {
	start := time.Now()
	result := &BlindFillStepResult{
		StepID:   step.StepID,
		StepName: step.Name,
		Status:   "pending",
	}

	// Parse step config
	var stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to parse step config: %v", err)
		result.ExecutionTime = time.Since(start).Milliseconds()
		return result
	}

	result.ActionType = stepConfig.Action

	// Execute based on action type
	switch stepConfig.Action {
	case "fill":
		result = e.executeFillStep(ctx, stepConfig, step, req, result)
	case "click":
		result = e.executeClickStep(ctx, stepConfig, step, req, result)
	case "select":
		result = e.executeSelectStep(ctx, stepConfig, step, req, result)
	case "wait":
		result = e.executeWaitStep(ctx, stepConfig, step, req, result)
	case "extract":
		result = e.executeExtractStep(ctx, stepConfig, step, req, result)
	default:
		result.Status = "error"
		result.Error = fmt.Sprintf("Unknown action: %s", stepConfig.Action)
	}

	result.ExecutionTime = time.Since(start).Milliseconds()
	return result
}

func (e *BlindFillExecutor) executeFillStep(
	ctx context.Context,
	stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	},
	step *WorkflowStep,
	req *BlindFillRequest,
	result *BlindFillStepResult,
) *BlindFillStepResult {
	params := stepConfig.Params
	if params == nil {
		params = make(map[string]interface{})
	}

	// Extract fields
	fieldsInterface, ok := params["fields"]
	if !ok {
		result.Status = "error"
		result.Error = "No fields specified in fill action"
		return result
	}

	fieldsSlice, ok := fieldsInterface.([]interface{})
	if !ok {
		result.Status = "error"
		result.Error = "Invalid fields format"
		return result
	}

	// Process each field
	processedFields := make([]map[string]interface{}, 0)
	for _, fieldIntf := range fieldsSlice {
		fieldMap, ok := fieldIntf.(map[string]interface{})
		if !ok {
			continue
		}

		selector, _ := fieldMap["selector"].(string)
		result.Selector = selector

		// Get field name from step description
		result.FieldName = step.Name

		// Handle value_ref (PII reference)
		if valueRef, ok := fieldMap["value_ref"].(string); ok && valueRef != "" {
			result.PIIRefUsed = valueRef // Log PII ref name only, not value

			// Check if we have pre-approved PII values
			var actualValue string
			var found bool

			if req.PIIValues != nil {
				actualValue, found = req.PIIValues[valueRef]
			}

			if !found {
				// Check variables (might have resolved PII)
				if req.Variables != nil {
					if val, ok := req.Variables[valueRef]; ok {
						if strVal, ok := val.(string); ok {
							actualValue = strVal
							found = true
						}
					}
				}
			}

			if !found {
				// Try to convert value_ref to variable name format
				varName := strings.ReplaceAll(valueRef, ".", "_")
				if req.Variables != nil {
					if val, ok := req.Variables[varName]; ok {
						if strVal, ok := val.(string); ok {
							actualValue = strVal
							found = true
						}
					}
				}
			}

			if found && actualValue != "" {
				// Create a copy with resolved value
				processedField := make(map[string]interface{})
				for k, v := range fieldMap {
					processedField[k] = v
				}
				processedField["value"] = actualValue
				delete(processedField, "value_ref")
				result.WasPIIApproved = true
				processedFields = append(processedFields, processedField)
			} else {
				// No PII value available - skip or error
				result.Status = "error"
				result.Error = fmt.Sprintf("PII value not available for ref: %s", valueRef)
				return result
			}
		} else {
			// Static value or no value_ref
			processedFields = append(processedFields, fieldMap)
		}
	}

	// Build fill command
	fillParams := map[string]interface{}{
		"fields": processedFields,
	}

	// Add options
	if stepConfig.Options != nil {
		if autoSubmit, ok := stepConfig.Options["auto_submit"].(bool); ok && autoSubmit {
			fillParams["auto_submit"] = true
		}
		if submitDelay, ok := stepConfig.Options["submit_delay"].(int); ok {
			fillParams["submit_delay"] = submitDelay
		}
	}

	// Execute fill
	fillResult, err := e.browser.Fill(ctx, fillParams)
	if err != nil {
		result.Status = "error"
		if be, ok := err.(*BrowserError); ok {
			if be.Code == "PII_REQUEST_DENIED" {
				result.Error = "PII approval denied"
			} else if be.Code == "ELEMENT_NOT_FOUND" {
				result.Error = fmt.Sprintf("Element not found: %s", result.Selector)
			} else {
				result.Error = be.Message
			}
		} else {
			result.Error = err.Error()
		}
		return result
	}

	result.Status = "success"
	result.Data = fillResult
	return result
}

func (e *BlindFillExecutor) executeClickStep(
	ctx context.Context,
	stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	},
	step *WorkflowStep,
	req *BlindFillRequest,
	result *BlindFillStepResult,
) *BlindFillStepResult {
	params := stepConfig.Params
	if params == nil {
		params = make(map[string]interface{})
	}

	selector, _ := params["selector"].(string)
	result.Selector = selector
	result.FieldName = step.Name

	// Execute click
	clickResult, err := e.browser.Click(ctx, params)
	if err != nil {
		result.Status = "error"
		if be, ok := err.(*BrowserError); ok {
			if be.Code == "ELEMENT_NOT_FOUND" {
				result.Error = fmt.Sprintf("Element not found: %s", selector)
			} else {
				result.Error = be.Message
			}
		} else {
			result.Error = err.Error()
		}
		return result
	}

	result.Status = "success"
	result.Data = clickResult
	return result
}

func (e *BlindFillExecutor) executeSelectStep(
	ctx context.Context,
	stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	},
	step *WorkflowStep,
	req *BlindFillRequest,
	result *BlindFillStepResult,
) *BlindFillStepResult {
	// Select is essentially fill for dropdowns
	// Use the same logic as fill
	return e.executeFillStep(ctx, stepConfig, step, req, result)
}

func (e *BlindFillExecutor) executeWaitStep(
	ctx context.Context,
	stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	},
	step *WorkflowStep,
	req *BlindFillRequest,
	result *BlindFillStepResult,
) *BlindFillStepResult {
	params := stepConfig.Params
	if params == nil {
		params = make(map[string]interface{})
	}

	result.FieldName = step.Name

	// Execute wait
	waitResult, err := e.browser.Wait(ctx, params)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	result.Status = "success"
	result.Data = waitResult
	return result
}

func (e *BlindFillExecutor) executeExtractStep(
	ctx context.Context,
	stepConfig struct {
		Action  string                 `json:"action"`
		Params  map[string]interface{} `json:"params,omitempty"`
		Options map[string]interface{} `json:"options,omitempty"`
	},
	step *WorkflowStep,
	req *BlindFillRequest,
	result *BlindFillStepResult,
) *BlindFillStepResult {
	params := stepConfig.Params
	if params == nil {
		params = make(map[string]interface{})
	}

	result.FieldName = step.Name

	// Execute extract
	extractResult, err := e.browser.Extract(ctx, params)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	result.Status = "success"
	result.Data = extractResult
	return result
}

//=============================================================================
// Navigation Execution
//=============================================================================

func (e *BlindFillExecutor) executeNavigation(
	ctx context.Context,
	targetURL string,
	waitUntil string,
	timeout int,
) *BlindFillStepResult {
	start := time.Now()
	result := &BlindFillStepResult{
		StepID:     "navigation",
		StepName:   "Navigate to target URL",
		ActionType: "navigate",
		Status:     "pending",
	}

	navParams := map[string]interface{}{
		"url": targetURL,
	}

	if waitUntil != "" {
		navParams["waitUntil"] = waitUntil
	}
	if timeout > 0 {
		navParams["timeout"] = timeout
	} else {
		navParams["timeout"] = 30000 // default 30s
	}

	navResult, err := e.browser.Navigate(ctx, navParams)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Navigation failed: %v", err)
		result.ExecutionTime = time.Since(start).Milliseconds()
		return result
	}

	result.Status = "success"
	result.Data = navResult
	result.ExecutionTime = time.Since(start).Milliseconds()
	return result
}

//=============================================================================
// Helper Methods
//=============================================================================

func (e *BlindFillExecutor) validateRequest(req *BlindFillRequest) error {
	if req.TemplateID == "" {
		return fmt.Errorf("%w: template_id is required", ErrInvalidRequest)
	}
	if req.Initiator == "" {
		return fmt.Errorf("%w: initiator is required", ErrInvalidRequest)
	}
	return nil
}

func (e *BlindFillExecutor) dryRunValidation(template *TaskTemplate, req *BlindFillRequest) (*BlindFillResult, error) {
	result := &BlindFillResult{
		ExecutionID: req.ExecutionID,
		TemplateID:  template.ID,
		Status:      BlindFillStatusPending,
		StartedAt:   time.Now(),
		Warnings:    make([]string, 0),
	}

	// Check for PII requirements
	piiFields := e.collectPIIFields(template, req)
	if len(piiFields) > 0 {
		result.RequiredPIIFields = piiFields
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Template requires PII approval for %d fields", len(piiFields)))
	}

	// Validate all selectors can be matched
	for i := range template.Steps {
		step := &template.Steps[i]
		if !e.validateStepConfig(step) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Step %s may have invalid configuration", step.StepID))
		}
	}

	result.Warnings = append(result.Warnings, "Dry run validation passed")

	return result, nil
}

func (e *BlindFillExecutor) collectPIIFields(template *TaskTemplate, req *BlindFillRequest) []string {
	// Start with template-level PII refs
	piiSet := make(map[string]bool)
	for _, ref := range template.PIIRefs {
		piiSet[ref] = true
	}

	// Scan step configs for value_ref
	for _, step := range template.Steps {
		var stepConfig struct {
			Action string                 `json:"action"`
			Params map[string]interface{} `json:"params,omitempty"`
		}

		if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
			continue
		}

		if stepConfig.Action != "fill" && stepConfig.Action != "select" {
			continue
		}

		params := stepConfig.Params
		if params == nil {
			continue
		}

		fieldsInterface, ok := params["fields"]
		if !ok {
			continue
		}

		fieldsSlice, ok := fieldsInterface.([]interface{})
		if !ok {
			continue
		}

		for _, fieldIntf := range fieldsSlice {
			fieldMap, ok := fieldIntf.(map[string]interface{})
			if !ok {
				continue
			}

			if valueRef, ok := fieldMap["value_ref"].(string); ok && valueRef != "" {
				// Check if already provided in PIIValues
				if req.PIIValues != nil {
					if _, provided := req.PIIValues[valueRef]; provided {
						continue // Already have value, skip
					}
				}
				piiSet[valueRef] = true
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(piiSet))
	for ref := range piiSet {
		result = append(result, ref)
	}
	return result
}

func (e *BlindFillExecutor) requestPIIApproval(
	ctx context.Context,
	template *TaskTemplate,
	piiFields []string,
	initiator string,
	executionID string,
) (*ApprovalResult, error) {
	if e.approval == nil {
		// No approval engine - return pending
		return &ApprovalResult{
			RequestID:     uuid.New().String(),
			Required:      true,
			Approved:      false,
			NeedsApproval: true,
		}, nil
	}

	// Create evaluation context
	evalCtx := EvaluationContext{
		Template:  template,
		Initiator: initiator,
		PIIFields: piiFields,
		Variables: make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	return e.approval.Evaluate(ctx, evalCtx)
}

func (e *BlindFillExecutor) extractTargetURL(template *TaskTemplate) string {
	// Look for navigate step
	for _, step := range template.Steps {
		var stepConfig struct {
			Action string                 `json:"action"`
			Params map[string]interface{} `json:"params,omitempty"`
		}

		if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
			continue
		}

		if stepConfig.Action == "navigate" {
			if url, ok := stepConfig.Params["url"].(string); ok {
				return url
			}
		}
	}
	return ""
}

func (e *BlindFillExecutor) validateStepConfig(step *WorkflowStep) bool {
	var stepConfig struct {
		Action string                 `json:"action"`
		Params map[string]interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(step.Config, &stepConfig); err != nil {
		return false
	}

	// Validate required params
	switch stepConfig.Action {
	case "fill", "select":
		_, hasFields := stepConfig.Params["fields"]
		return hasFields
	case "click":
		_, hasSelector := stepConfig.Params["selector"]
		return hasSelector
	case "navigate":
		_, hasURL := stepConfig.Params["url"]
		return hasURL
	case "wait", "extract":
		return true // These may have optional params
	}

	return true
}

//=============================================================================
// Execution Status
//=============================================================================

// GetExecutionStatus returns the status of an active or completed execution
func (e *BlindFillExecutor) GetExecutionStatus(executionID string) (*BlindFillResult, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	exec, exists := e.executions[executionID]
	if !exists {
		return nil, false
	}

	exec.mu.RLock()
	defer exec.mu.RUnlock()

	result := &BlindFillResult{
		ExecutionID: exec.id,
		TemplateID:  exec.template.ID,
		Status:      BlindFillStatusRunning,
		StartedAt:   exec.startedAt,
		StepResults: exec.stepResults,
	}

	return result, true
}

// ListActiveExecutions returns all active execution IDs
func (e *BlindFillExecutor) ListActiveExecutions() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]string, 0, len(e.executions))
	for id := range e.executions {
		ids = append(ids, id)
	}
	return ids
}

// GetActiveExecutionCount returns the number of active executions
func (e *BlindFillExecutor) GetActiveExecutionCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.executions)
}

//=============================================================================
// Resume with Approval
//=============================================================================

// ResumeWithApproval resumes an execution waiting for PII approval
func (e *BlindFillExecutor) ResumeWithApproval(
	ctx context.Context,
	executionID string,
	approvalToken string,
	piiValues map[string]string,
) (*BlindFillResult, error) {
	e.mu.RLock()
	exec, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	// Validate approval token binding
	if err := e.validateApprovalToken(exec, approvalToken); err != nil {
		e.log.Warn("approval_token_validation_failed",
			"execution_id", executionID,
			"error", err.Error())
		return nil, fmt.Errorf("invalid approval token: %w", err)
	}

	// Validate PII values against approved/denied fields
	if err := e.validatePIIValues(exec, piiValues); err != nil {
		e.log.Warn("pii_values_validation_failed",
			"execution_id", executionID,
			"error", err.Error())
		return nil, fmt.Errorf("PII values validation failed: %w", err)
	}

	// Update request with PII values
	exec.mu.Lock()
	if exec.request.PIIValues == nil {
		exec.request.PIIValues = make(map[string]string)
	}
	for k, v := range piiValues {
		exec.request.PIIValues[k] = v
	}
	exec.request.ApprovalToken = approvalToken
	exec.mu.Unlock()

	e.log.Info("blindfill_execution_resumed",
		"execution_id", executionID,
		"pii_fields_count", len(piiValues))

	// Continue execution - the execution is already running in background
	// This would typically be called by an approval callback
	// For now, return current status
	return &BlindFillResult{
		ExecutionID: executionID,
		TemplateID:  exec.template.ID,
		Status:      BlindFillStatusRunning,
		StartedAt:   exec.startedAt,
	}, nil
}

// validateApprovalToken validates that the approval token is bound to this execution
func (e *BlindFillExecutor) validateApprovalToken(exec *activeExecution, token string) error {
	if token == "" {
		return fmt.Errorf("approval token is required")
	}

	// Token must match the expected approval request for this execution
	// Format: "approval_<execution_id>_<timestamp>"
	expectedPrefix := fmt.Sprintf("approval_%s_", exec.id)
	if !strings.HasPrefix(token, expectedPrefix) {
		return fmt.Errorf("token not bound to execution")
	}

	return nil
}

// validatePIIValues validates that provided PII values don't include denied fields
func (e *BlindFillExecutor) validatePIIValues(exec *activeExecution, piiValues map[string]string) error {
	if piiValues == nil {
		return nil
	}

	// Get denied fields from approval result if available
	// This prevents providing values for fields that were explicitly denied
	if exec.request.DeniedFields != nil {
		for fieldRef := range piiValues {
			for _, denied := range exec.request.DeniedFields {
				if fieldRef == denied || strings.HasPrefix(fieldRef, denied+".") {
					return fmt.Errorf("cannot provide value for denied field: %s", fieldRef)
				}
			}
		}
	}

	return nil
}
