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
	"github.com/armorclaw/bridge/pkg/capability"
	"github.com/armorclaw/bridge/pkg/interfaces"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
	"github.com/armorclaw/bridge/pkg/toolsidecar"
	"github.com/armorclaw/bridge/pkg/translator"
)

// VaultClient abstracts the vault governance client for tool lifecycle hooks.
// The concrete *vault.VaultGovernanceClient satisfies this interface.
type VaultClient interface {
	// IssueBlindFillToken generates an ephemeral token for blind-fill injection.
	IssueBlindFillToken(ctx context.Context, sessionID, toolName, secret string, ttl time.Duration) (string, error)
	// ZeroizeToolSecrets securely erases all in-memory secrets for a tool/session pair.
	ZeroizeToolSecrets(ctx context.Context, toolName, sessionID string) (uint32, error)
}

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
	vaultClient   VaultClient
	v6Microkernel bool
	v6AuditMode   bool
	authorizer    interfaces.CapabilityBroker
}

// Provisioner interface for ToolSidecar operations
type Provisioner interface {
	SpawnToolSidecar(ctx context.Context, skillName, sessionID string) (*toolsidecar.ToolSidecar, error)
	StopToolSidecar(ctx context.Context, containerID string) error
	ExecuteInSidecar(ctx context.Context, containerID string, toolName string, arguments json.RawMessage) ([]byte, error)
}

// Config holds MCPRouter configuration
type Config struct {
	SkillGate      interfaces.SkillGate
	Provisioner    Provisioner
	ConsentManager *pii.HITLConsentManager
	Auditor        *audit.AuditLog
	Logger         *logger.Logger
	ConsentNotify  func(ctx context.Context, request *pii.AccessRequest) error
	VaultClient    VaultClient
	V6Microkernel  bool
	V6AuditMode    bool
	Authorizer     interfaces.CapabilityBroker
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
		vaultClient:   cfg.VaultClient,
		v6Microkernel: cfg.V6Microkernel,
		v6AuditMode:   cfg.V6AuditMode,
		authorizer:    cfg.Authorizer,
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
	toolCall := &interfaces.ToolCall{
		ID:        generateCallID(),
		ToolName:  req.Params.Name,
		Arguments: r.parseArguments(req.Params.Arguments),
	}

	r.logger.Info("mcp_tool_call_received",
		"call_id", toolCall.ID,
		"tool", req.Params.Name,
	)

	if r.authorizer != nil {
		actionReq := capability.ActionRequest{
			Action: req.Params.Name,
			Params: r.parseArguments(req.Params.Arguments),
		}
		resp, err := r.authorizer.Authorize(ctx, actionReq)
		if err != nil || !resp.Allowed {
			reason := "capability denied by broker"
			if resp.Reason != "" {
				reason = resp.Reason
			}
			if err != nil {
				reason = fmt.Sprintf("broker error: %s", err.Error())
			}
			r.logger.Warn("capability_denied",
				"call_id", toolCall.ID,
				"tool", req.Params.Name,
				"reason", reason,
			)
			return r.errorResponse(req.ID, -32603, "Capability denied", reason), nil
		}
	}

	if r.v6AuditMode && r.v6Microkernel {
		return r.handleAuditMode(ctx, req, toolCall)
	}

	sanitizedCall, err := r.skillGate.InterceptToolCall(ctx, toolCall)
	if err != nil {
		r.logger.Error("skillgate_validation_failed",
			"call_id", toolCall.ID,
			"tool", req.Params.Name,
			"error", err.Error(),
		)

		_ = r.auditor.LogEvent(audit.EventSecurityViolation,
			"", "", "",
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
		"", "", "",
		map[string]interface{}{
			"call_id":   toolCall.ID,
			"tool":      req.Params.Name,
			"event":     "skillgate_validation",
			"status":    "passed",
			"pii_found": len(sanitizedCall.Arguments) != len(toolCall.Arguments),
		},
	)

	if r.requiresConsent(sanitizedCall, toolCall) {
		r.logger.Info("consent_required",
			"call_id", toolCall.ID,
			"tool", req.Params.Name,
			"reason", "pii_detected",
		)

		return r.initiateConsent(ctx, req, sanitizedCall, toolCall)
	}

	return r.executeTool(ctx, req, sanitizedCall, toolCall)
}

// handleAuditMode logs every action that would be taken without executing enforcement.
// Tool calls pass through unmodified with original arguments.
func (r *MCPRouter) handleAuditMode(ctx context.Context, req *MCPToolsCallRequest, toolCall *interfaces.ToolCall) (*MCPResponse, error) {
	r.logger.Info("v6_audit_mode_active",
		"call_id", toolCall.ID,
		"tool", toolCall.ToolName,
		"mode", "audit_only",
	)

	auditFields := map[string]interface{}{
		"call_id":    toolCall.ID,
		"tool":       toolCall.ToolName,
		"mode":       "v6_audit",
		"action":     "skillgate_intercept",
		"would_run":  true,
		"args_count": len(toolCall.Arguments),
	}

	violations, err := r.skillGate.ValidateArgs(ctx, toolCall.ToolName, toolCall.Arguments)
	if err != nil {
		auditFields["validation_error"] = err.Error()
		r.logger.Info("v6_audit_skillgate_would_fail",
			"call_id", toolCall.ID,
			"tool", toolCall.ToolName,
			"error", err.Error(),
		)
	} else {
		auditFields["pii_violations"] = len(violations)
		for _, v := range violations {
			r.logger.Info("v6_audit_pii_would_be_intercepted",
				"call_id", toolCall.ID,
				"tool", toolCall.ToolName,
				"field", v.Field,
				"pattern_type", v.PatternType,
				"severity", v.Severity,
			)
		}
	}

	auditFields["would_require_consent"] = r.requiresConsent(toolCall, toolCall) || len(violations) > 0

	r.logger.Info("v6_audit_governance_check",
		"call_id", toolCall.ID,
		"tool", toolCall.ToolName,
		"would_issue_tokens", r.vaultClient != nil,
		"would_spawn_sidecar", true,
		"would_zeroize", r.vaultClient != nil,
	)

	_ = r.auditor.LogEvent(audit.EventSecurityViolation,
		"", "", "",
		auditFields,
	)

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":     "audit_logged",
			"tool":       toolCall.ToolName,
			"call_id":    toolCall.ID,
			"violations": len(violations),
		},
	}, nil
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
	execCtx, cancel := context.WithTimeout(ctx, DefaultToolExecutionTimeout)
	defer cancel()

	sessionID := "mcp_session"
	toolName := sanitizedCall.ToolName

	// Pre-execution: issue ephemeral tokens for blind-fill (security lifecycle hook)
	if r.v6Microkernel && !r.v6AuditMode && r.vaultClient != nil {
		for key, value := range sanitizedCall.Arguments {
			if str, ok := value.(string); ok && len(str) > 0 {
				tokenID, err := r.vaultClient.IssueBlindFillToken(execCtx, sessionID, toolName, str, 10*time.Second)
				if err != nil {
					r.logger.Warn("failed to issue blind fill token",
						"tool", toolName,
						"key", key,
						"error", err,
					)
				} else {
					r.logger.Info("blind fill token issued",
						"tool", toolName,
						"key", key,
						"token_id", tokenID,
					)
				}
			}
		}
	}

	// Post-execution: zeroize secrets regardless of success/failure (security lifecycle hook)
	if r.v6Microkernel && !r.v6AuditMode && r.vaultClient != nil {
		defer func() {
			zCtx, zCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer zCancel()
			count, err := r.vaultClient.ZeroizeToolSecrets(zCtx, toolName, sessionID)
			if err != nil {
				r.logger.Warn("failed to zeroize tool secrets",
					"tool", toolName,
					"error", err,
				)
			} else {
				r.logger.Info("zeroized tool secrets",
					"tool", toolName,
					"count", count,
				)
			}
		}()
	}

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

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cleanupCancel()
		_ = r.provisioner.StopToolSidecar(cleanupCtx, toolsidecar.ID)
	}()

	argsJSON, err := json.Marshal(sanitizedCall.Arguments)
	if err != nil {
		r.logger.Error("failed_to_marshal_arguments",
			"call_id", originalCall.ID,
			"tool", sanitizedCall.ToolName,
			"error", err.Error(),
		)
		return r.errorResponse(req.ID, -32603, "Failed to marshal tool arguments", err.Error()), nil
	}

	output, execErr := r.provisioner.ExecuteInSidecar(execCtx, toolsidecar.ID, sanitizedCall.ToolName, argsJSON)
	if execErr != nil {
		r.logger.Error("tool_execution_failed",
			"call_id", originalCall.ID,
			"tool", sanitizedCall.ToolName,
			"container_id", toolsidecar.ID[:12],
			"error", execErr.Error(),
		)

		_ = r.auditor.LogEvent(audit.EventSidecarQueued,
			"", "", "",
			map[string]interface{}{
				"call_id":      originalCall.ID,
				"tool":         sanitizedCall.ToolName,
				"container_id": toolsidecar.ID[:12],
				"error":        execErr.Error(),
				"status":       "exec_failed",
			},
		)

		return r.errorResponse(req.ID, -32603, "Tool execution failed", execErr.Error()), nil
	}

	r.logger.Info("tool_executed",
		"call_id", originalCall.ID,
		"tool", sanitizedCall.ToolName,
		"container_id", toolsidecar.ID[:12],
	)

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		r.logger.Warn("tool_output_parse_error",
			"call_id", originalCall.ID,
			"tool", sanitizedCall.ToolName,
			"error", err.Error(),
		)
		result = map[string]interface{}{
			"status": "success",
			"output": string(output),
		}
	}

	_ = r.auditor.LogEvent(audit.EventSidecarQueued,
		"", "", "",
		map[string]interface{}{
			"call_id":      originalCall.ID,
			"tool":         sanitizedCall.ToolName,
			"container_id": toolsidecar.ID[:12],
			"arguments":    r.redactPII(originalCall.Arguments),
			"status":       "success",
		},
	)

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
