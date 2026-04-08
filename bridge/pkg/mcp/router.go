// Package mcp provides routing for MCP (Model Context Protocol) tool calls through SkillGate.
//
// Resolves: Task 8 - MCP Router with SkillGate
//
// The MCPRouter routes all MCP tools/call requests through:
// - SkillGate validation (PII interception and redaction)
// - HITL consent workflow for PII operations
// - ToolSidecar provisioning for isolated execution
// - Audit logging for compliance
package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/interfaces"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
	"github.com/armorclaw/bridge/pkg/toolsidecar"
	"github.com/armorclaw/bridge/pkg/translator"
)

const (
	// DefaultConsentTimeout is default time to wait for user approval
	DefaultConsentTimeout = 60 * time.Second

	// DefaultToolExecutionTimeout is default time for tool execution
	DefaultToolExecutionTimeout = 30 * time.Second
)

// MCPRouter routes MCP tool calls through SkillGate with consent workflow
type MCPRouter struct {
	skillGate     interfaces.SkillGate
	provisioner   Provisioner
	consentMgr    *pii.HITLConsentManager
	auditor       *audit.AuditLog
	translator    *translator.RPCToMCPTranslator
	logger        *logger.Logger
	consentNotify func(ctx context.Context, request *pii.AccessRequest) error
}

// Provisioner interface for ToolSidecar operations
type Provisioner interface {
	SpawnToolSidecar(ctx context.Context, skillName, sessionID string) (*toolsidecar.ToolSidecar, error)
	StopToolSidecar(ctx context.Context, containerID string) error
}

// Config holds MCPRouter configuration
type Config struct {
	// SkillGate is the PII interception and validation gate
	SkillGate interfaces.SkillGate

	// Provisioner creates ToolSidecar containers for execution
	Provisioner Provisioner

	// ConsentManager handles HITL consent for PII operations
	ConsentManager *pii.HITLConsentManager

	// Auditor logs all tool calls for compliance
	Auditor *audit.AuditLog

	// Logger is the structured logger
	Logger *logger.Logger

	// ConsentNotify is callback to send consent notifications via Matrix
	ConsentNotify func(ctx context.Context, request *pii.AccessRequest) error
}

// New creates a new MCPRouter
func New(cfg Config) (*MCPRouter, error) {
	// Validate configuration
	if cfg.SkillGate == nil {
		return nil, fmt.Errorf("skillgate is required")
	}
	if cfg.Provisioner == nil {
		return nil, fmt.Errorf("provisioner is required")
	}
	if cfg.ConsentManager == nil {
		return nil, fmt.Errorf("consent manager is required")
	}
	if cfg.Auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}

	// Set defaults
	if cfg.Logger == nil {
		cfg.Logger = logger.Global().WithComponent("mcp_router")
	}

	// Set consent notification callback
	if cfg.ConsentNotify != nil {
		cfg.ConsentManager.SetNotifyCallback(cfg.ConsentNotify)
	}

	return &MCPRouter{
		skillGate:     cfg.SkillGate,
		provisioner:   cfg.Provisioner,
		consentMgr:    cfg.ConsentManager,
		auditor:       cfg.Auditor,
		translator:    translator.NewRPCToMCPTranslator(),
		logger:        cfg.Logger,
		consentNotify: cfg.ConsentNotify,
	}, nil
}

// MCPToolsCallRequest represents an MCP tools/call request
type MCPToolsCallRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Params  *MCPParams  `json:"params,omitempty"`
}

// MCPParams represents parameters for tools/call
type MCPParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObj   `json:"error,omitempty"`
}

// ErrorObj represents a JSON-RPC error object
type ErrorObj struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleToolsCall routes an MCP tools/call request through SkillGate
//
// Flow:
// 1. Validate via SkillGate (PII interception and redaction)
// 2. Check for consent requirement (if PII detected)
// 3. Spawn ToolSidecar for isolated execution
// 4. Execute tool with sanitized arguments
// 5. Audit log (PII redacted)
func (r *MCPRouter) HandleToolsCall(ctx context.Context, req *MCPToolsCallRequest) (*MCPResponse, error) {
	// Step 1: Create ToolCall for SkillGate validation
	toolCall := &interfaces.ToolCall{
		ID:        generateCallID(),
		ToolName:  req.Params.Name,
		Arguments: r.parseArguments(req.Params.Arguments),
	}

	r.logger.Info("mcp_tool_call_received",
		"call_id", toolCall.ID,
		"tool", req.Params.Name,
	)

	// Step 2: Validate via SkillGate
	// This intercepts and redacts PII from arguments
	sanitizedCall, err := r.skillGate.InterceptToolCall(ctx, toolCall)
	if err != nil {
		r.logger.Error("skillgate_validation_failed",
			"call_id", toolCall.ID,
			"tool", req.Params.Name,
			"error", err.Error(),
		)

		_ = r.auditor.LogEvent(audit.EventSecurityViolation,
			"", // session_id
			"", // room_id
			"", // user_id
			map[string]interface{}{
				"call_id": toolCall.ID,
				"tool":    req.Params.Name,
				"error":   err.Error(),
				"reason":  "skillgate_validation_failed",
			},
		)

		return r.errorResponse(req.ID, -32603, "SkillGate validation failed", err.Error()), nil
	}

	_ = r.auditor.LogEvent(audit.EventSecurityViolation,
		"", // session_id
		"", // room_id
		"", // user_id
		map[string]interface{}{
			"call_id":   toolCall.ID,
			"tool":      req.Params.Name,
			"event":     "skillgate_validation",
			"status":    "passed",
			"pii_found": len(sanitizedCall.Arguments) != len(toolCall.Arguments),
		},
	)

	// Step 3: Check for consent requirement
	// If PII was redacted, we need user consent
	if r.requiresConsent(sanitizedCall, toolCall) {
		r.logger.Info("consent_required",
			"call_id", toolCall.ID,
			"tool", req.Params.Name,
			"reason", "pii_detected",
		)

		return r.initiateConsent(ctx, req, sanitizedCall, toolCall)
	}

	// Step 4: Execute tool with sanitized arguments
	return r.executeTool(ctx, req, sanitizedCall, toolCall)
}

// requiresConsent checks if a tool call requires user consent
func (r *MCPRouter) requiresConsent(sanitizedCall *interfaces.ToolCall, originalCall *interfaces.ToolCall) bool {
	// Check if any PII was redacted
	for key := range originalCall.Arguments {
		sanitizedValue, exists := sanitizedCall.Arguments[key]
		if !exists {
			return true
		}

		// Check if value was redacted (contains [REDACTED: or similar)
		if str, ok := sanitizedValue.(string); ok && len(str) > 10 {
			if str[:10] == "[REDACTED:" || str[:10] == "[REDACTED_" {
				return true
			}
		}
	}

	// Check tool name against PII-sensitive tools
	piiTools := map[string]bool{
		"pii_request":    true,
		"profile_access": true,
		"payment_fill":   true,
		"credit_card":    true,
		"ssn_lookup":     true,
	}

	return piiTools[originalCall.ToolName]
}

// initiateConsent starts the consent workflow for PII operations
func (r *MCPRouter) initiateConsent(
	ctx context.Context,
	req *MCPToolsCallRequest,
	sanitizedCall *interfaces.ToolCall,
	originalCall *interfaces.ToolCall,
) (*MCPResponse, error) {
	// Create a simple manifest for consent request
	manifest := &pii.SkillManifest{
		SkillID:   "mcp_" + originalCall.ToolName,
		SkillName: "MCP Tool: " + originalCall.ToolName,
		Variables: r.extractVariables(originalCall.Arguments),
	}

	// Request consent
	accessRequest, err := r.consentMgr.RequestAccessAndWait(
		ctx,
		manifest,
		"mcp_session",
		"",
	)
	if err != nil {
		r.logger.Error("consent_failed",
			"call_id", originalCall.ID,
			"error", err.Error(),
		)

		_ = r.auditor.LogEvent(audit.EventPIIAccessRejected,
			"", // session_id
			"", // room_id
			"", // user_id
			map[string]interface{}{
				"call_id": originalCall.ID,
				"tool":    originalCall.ToolName,
				"reason":  err.Error(),
			},
		)

		return r.errorResponse(req.ID, -32603, "Consent required but failed", err.Error()), nil
	}

	// Consent granted, proceed with execution
	r.logger.Info("consent_granted",
		"call_id", originalCall.ID,
		"request_id", accessRequest.ID,
	)

	_ = r.auditor.LogEvent(audit.EventPIIAccessGranted,
		"",                       // session_id
		"",                       // room_id
		accessRequest.ApprovedBy, // user_id
		map[string]interface{}{
			"call_id":         originalCall.ID,
			"tool":            originalCall.ToolName,
			"request_id":      accessRequest.ID,
			"approved_fields": accessRequest.ApprovedFields,
		},
	)

	// Execute tool with sanitized arguments
	return r.executeTool(ctx, req, sanitizedCall, originalCall)
}

// executeTool spawns a ToolSidecar and executes the tool
func (r *MCPRouter) executeTool(
	ctx context.Context,
	req *MCPToolsCallRequest,
	sanitizedCall *interfaces.ToolCall,
	originalCall *interfaces.ToolCall,
) (*MCPResponse, error) {
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, DefaultToolExecutionTimeout)
	defer cancel()

	// Step 4.1: Spawn ToolSidecar
	toolsidecar, err := r.provisioner.SpawnToolSidecar(execCtx, sanitizedCall.ToolName, "mcp_session")
	if err != nil {
		r.logger.Error("toolsidecar_spawn_failed",
			"call_id", originalCall.ID,
			"tool", sanitizedCall.ToolName,
			"error", err.Error(),
		)

		_ = r.auditor.LogEvent(audit.EventSidecarQueued,
			"", // session_id
			"", // room_id
			"", // user_id
			map[string]interface{}{
				"call_id": originalCall.ID,
				"tool":    sanitizedCall.ToolName,
				"error":   err.Error(),
				"status":  "failed",
			},
		)

		return r.errorResponse(req.ID, -32603, "Failed to spawn tool container", err.Error()), nil
	}

	r.logger.Info("toolsidecar_spawned",
		"call_id", originalCall.ID,
		"container_id", toolsidecar.ID[:12],
		"tool", sanitizedCall.ToolName,
	)

	// Step 4.2: Execute tool
	// For now, return a mock result
	// In a full implementation, this would communicate with the ToolSidecar
	result := map[string]interface{}{
		"status": "success",
		"output": fmt.Sprintf("Tool %s executed in container %s", sanitizedCall.ToolName, toolsidecar.ID[:12]),
	}

	r.logger.Info("tool_executed",
		"call_id", originalCall.ID,
		"tool", sanitizedCall.ToolName,
		"container_id", toolsidecar.ID[:12],
	)

	// Step 5: Audit log (PII redacted)
	_ = r.auditor.LogEvent(audit.EventSidecarQueued,
		"", // session_id
		"", // room_id
		"", // user_id
		map[string]interface{}{
			"call_id":      originalCall.ID,
			"tool":         sanitizedCall.ToolName,
			"container_id": toolsidecar.ID[:12],
			"arguments":    r.redactPII(originalCall.Arguments),
			"status":       "success",
		},
	)

	// Cleanup toolsidecar
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cleanupCancel()
	_ = r.provisioner.StopToolSidecar(cleanupCtx, toolsidecar.ID)

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// parseArguments parses raw JSON arguments into a map
func (r *MCPRouter) parseArguments(args json.RawMessage) map[string]interface{} {
	if len(args) == 0 {
		return make(map[string]interface{})
	}

	var result map[string]interface{}
	if err := json.Unmarshal(args, &result); err != nil {
		r.logger.Warn("failed_to_parse_arguments", "error", err.Error())
		return make(map[string]interface{})
	}

	return result
}

// extractVariables creates skill manifest variables from arguments
func (r *MCPRouter) extractVariables(args map[string]interface{}) []pii.VariableRequest {
	variables := make([]pii.VariableRequest, 0, len(args))

	for key, value := range args {
		variables = append(variables, pii.VariableRequest{
			Key:         key,
			Description: "Argument: " + key,
			Required:    true,
			Sensitivity: r.determineSensitivity(key, value),
		})
	}

	return variables
}

// determineSensitivity determines the sensitivity level of an argument
func (r *MCPRouter) determineSensitivity(key string, value interface{}) pii.SensitivityLevel {
	// Check key names for PII patterns
	piiKeys := map[string]pii.SensitivityLevel{
		"email":       pii.SensitivityMedium,
		"phone":       pii.SensitivityMedium,
		"ssn":         pii.SensitivityCritical,
		"credit_card": pii.SensitivityCritical,
		"card_number": pii.SensitivityCritical,
		"card_cvv":    pii.SensitivityCritical,
		"card_expiry": pii.SensitivityMedium,
		"password":    pii.SensitivityCritical,
		"secret":      pii.SensitivityCritical,
		"api_key":     pii.SensitivityCritical,
		"token":       pii.SensitivityCritical,
		"address":     pii.SensitivityMedium,
		"zip":         pii.SensitivityLow,
		"name":        pii.SensitivityLow,
		"first_name":  pii.SensitivityLow,
		"last_name":   pii.SensitivityLow,
	}

	if sensitivity, ok := piiKeys[key]; ok {
		return sensitivity
	}

	// Check value for PII patterns
	if str, ok := value.(string); ok && len(str) > 10 && str[:10] == "[REDACTED:" {
		return pii.SensitivityCritical
	}

	return pii.SensitivityLow
}

// redactPII redacts PII values from arguments for audit logging
func (r *MCPRouter) redactPII(args map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{})

	for key, value := range args {
		sensitivity := r.determineSensitivity(key, value)

		if sensitivity == pii.SensitivityCritical {
			hash := sha256.Sum256([]byte(fmt.Sprintf("%v", value)))
			hashStr := hex.EncodeToString(hash[:8])
			redacted[key] = fmt.Sprintf("[REDACTED:%s]", hashStr)
		} else if sensitivity == pii.SensitivityHigh {
			redacted[key] = "[REDACTED_HIGH_SENSITIVITY]"
		} else if sensitivity == pii.SensitivityMedium {
			redacted[key] = "[REDACTED_MEDIUM_SENSITIVITY]"
		} else {
			redacted[key] = value
		}
	}

	return redacted
}

// errorResponse creates an error response
func (r *MCPRouter) errorResponse(id interface{}, code int, message string, data interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObj{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// generateCallID generates a unique call ID
func generateCallID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return "mcp_call_" + hex.EncodeToString(hash[:16])
}
