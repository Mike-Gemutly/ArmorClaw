package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// RPC Error Codes
//=============================================================================

const (
	ErrInvalidParams = -32602
	ErrInternal      = -32603
	ErrNotFound      = -32001
	ErrValidation    = -32002
)

//=============================================================================
// RPC Request/Response Types
//=============================================================================

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	UserID  string          `json:"-"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(result interface{}) *RPCResponse {
	return &RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
	}
}

func ErrorResponse(code int, message string, data ...interface{}) *RPCResponse {
	resp := &RPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	if len(data) > 0 {
		resp.Error.Data = data[0]
	}
	return resp
}

//=============================================================================
// Secretary RPC Handler
//=============================================================================

type RPCHandler struct {
	orchestrator *WorkflowOrchestratorImpl
	store        Store
	log          *logger.Logger
}

type RPCHandlerConfig struct {
	Orchestrator *WorkflowOrchestratorImpl
	Store        Store
	Logger       *logger.Logger
}

func NewRPCHandler(cfg RPCHandlerConfig) *RPCHandler {
	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("secretary_rpc")
	}

	return &RPCHandler{
		orchestrator: cfg.Orchestrator,
		store:        cfg.Store,
		log:          log,
	}
}

func (h *RPCHandler) Handle(req *RPCRequest) *RPCResponse {
	switch req.Method {
	case "secretary.start_workflow":
		return h.handleStartWorkflow(req)
	case "secretary.get_workflow":
		return h.handleGetWorkflow(req)
	case "secretary.get_template":
		return h.handleGetTemplate(req)
	case "secretary.delete_template":
		return h.handleDeleteTemplate(req)
	case "secretary.cancel_workflow":
		return h.handleCancelWorkflow(req)
	case "secretary.create_template":
		return h.handleCreateTemplate(req)
	case "secretary.advance_workflow":
		return h.handleAdvanceWorkflow(req)
	case "secretary.get_active_count":
		return h.handleGetActiveCount(req)
	case "secretary.is_running":
		return h.handleIsRunning(req)
	case "secretary.list_templates":
		return h.handleListTemplates(req)
	case "secretary.update_template":
		return h.handleUpdateTemplate(req)
	case "secretary.shutdown":
		return h.handleShutdown(req)
	default:
		return ErrorResponse(ErrNotFound, fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

//=============================================================================
// Workflow Handlers
//=============================================================================

type CreateTemplateParams struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Steps       []WorkflowStep  `json:"steps"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	PIIRefs     []string        `json:"pii_refs,omitempty"`
	CreatedBy   string          `json:"created_by"`
}

type GetTemplateParams struct {
	TemplateID string `json:"template_id"`
}

type DeleteTemplateParams struct {
	TemplateID string `json:"template_id"`
}

type UpdateTemplateParams struct {
	TemplateID  string          `json:"template_id"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Steps       []WorkflowStep  `json:"steps,omitempty"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	PIIRefs     []string        `json:"pii_refs,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
}

type GetWorkflowParams struct {
	WorkflowID string `json:"workflow_id"`
}

func (h *RPCHandler) handleCreateTemplate(req *RPCRequest) *RPCResponse {
	var params CreateTemplateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	// Validate required fields
	if params.Name == "" {
		return ErrorResponse(ErrValidation, "name is required")
	}
	if len(params.Steps) == 0 {
		return ErrorResponse(ErrValidation, "steps must contain at least one step")
	}
	if req.UserID == "" {
		return ErrorResponse(ErrValidation, "user_id is required")
	}

	now := time.Now()
	template := &TaskTemplate{
		ID:          fmt.Sprintf("tpl_%d", now.UnixMilli()),
		Name:        params.Name,
		Description: params.Description,
		Steps:       params.Steps,
		Variables:   params.Variables,
		PIIRefs:     params.PIIRefs,
		CreatedBy:   req.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsActive:    true,
	}

	if err := h.store.CreateTemplate(context.Background(), template); err != nil {
		return ErrorResponse(ErrInternal, "Failed to create template: "+err.Error())
	}

	h.log.Info("template_created_via_rpc", "template_id", template.ID, "name", template.Name, "created_by", req.UserID)

	return SuccessResponse(template)
}

type StartWorkflowParams struct {
	WorkflowID string `json:"workflow_id"`
}

func (h *RPCHandler) handleStartWorkflow(req *RPCRequest) *RPCResponse {
	var params StartWorkflowParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.WorkflowID == "" {
		return ErrorResponse(ErrInvalidParams, "workflow_id is required")
	}

	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	if err := h.orchestrator.StartWorkflow(params.WorkflowID); err != nil {
		return ErrorResponse(ErrInternal, "Failed to start workflow: "+err.Error())
	}

	workflow, _ := h.orchestrator.GetWorkflow(params.WorkflowID)

	h.log.Info("workflow_started_via_rpc", "workflow_id", params.WorkflowID, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"workflow_id": params.WorkflowID,
		"status":      "started",
		"workflow":    workflow,
	})
}

func (h *RPCHandler) handleGetTemplate(req *RPCRequest) *RPCResponse {
	var params GetTemplateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.TemplateID == "" {
		return ErrorResponse(ErrInvalidParams, "template_id is required")
	}

	template, err := h.store.GetTemplate(context.Background(), params.TemplateID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Template not found: "+params.TemplateID)
	}

	h.log.Info("template_retrieved_via_rpc", "template_id", template.ID, "name", template.Name, "by", req.UserID)

	return SuccessResponse(template)
}

func (h *RPCHandler) handleDeleteTemplate(req *RPCRequest) *RPCResponse {
	var params DeleteTemplateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.TemplateID == "" {
		return ErrorResponse(ErrInvalidParams, "template_id is required")
	}

	if err := h.store.DeleteTemplate(context.Background(), params.TemplateID); err != nil {
		return ErrorResponse(ErrInternal, "Failed to delete template: "+err.Error())
	}

	h.log.Info("template_deleted_via_rpc", "template_id", params.TemplateID, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"template_id": params.TemplateID,
		"deleted":     true,
	})
}

func (h *RPCHandler) handleUpdateTemplate(req *RPCRequest) *RPCResponse {
	var params UpdateTemplateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.TemplateID == "" {
		return ErrorResponse(ErrInvalidParams, "template_id is required")
	}

	template, err := h.store.GetTemplate(context.Background(), params.TemplateID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Template not found: "+params.TemplateID)
	}

	if params.Name != "" {
		template.Name = params.Name
	}

	if params.Description != "" {
		template.Description = params.Description
	}

	if params.Steps != nil && len(params.Steps) > 0 {
		template.Steps = params.Steps
	}

	if params.Variables != nil {
		template.Variables = params.Variables
	}

	if params.PIIRefs != nil && len(params.PIIRefs) > 0 {
		template.PIIRefs = params.PIIRefs
	}

	if params.IsActive != nil {
		template.IsActive = *params.IsActive
	}

	template.UpdatedAt = time.Now()

	if err := h.store.UpdateTemplate(context.Background(), template); err != nil {
		return ErrorResponse(ErrInternal, "Failed to update template: "+err.Error())
	}

	h.log.Info("template_updated_via_rpc", "template_id", template.ID, "name", template.Name, "by", req.UserID)

	return SuccessResponse(template)
}

//=============================================================================
// Workflow Handlers
//=============================================================================

func (h *RPCHandler) handleGetWorkflow(req *RPCRequest) *RPCResponse {
	var params GetWorkflowParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.WorkflowID == "" {
		return ErrorResponse(ErrInvalidParams, "workflow_id is required")
	}

	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	workflow, err := h.orchestrator.GetWorkflow(params.WorkflowID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Workflow not found: "+params.WorkflowID)
	}

	return SuccessResponse(workflow)
}

type ListWorkflowsParams struct {
	Status    WorkflowStatus `json:"status,omitempty"`
	CreatedBy string         `json:"created_by,omitempty"`
}

func (h *RPCHandler) handleListWorkflows(req *RPCRequest) *RPCResponse {
	var params ListWorkflowsParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	workflows, err := h.orchestrator.ListWorkflows(params.Status, params.CreatedBy)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list workflows: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

type ListTemplatesParams struct {
	ActiveOnly bool `json:"active_only,omitempty"`
}

func (h *RPCHandler) handleListTemplates(req *RPCRequest) *RPCResponse {
	var params ListTemplatesParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	templates, err := h.store.ListTemplates(context.Background(), TemplateFilter{
		ActiveOnly: params.ActiveOnly,
	})
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list templates: "+err.Error())
	}

	h.log.Info("templates_listed_via_rpc", "count", len(templates), "active_only", params.ActiveOnly, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

type CancelWorkflowParams struct {
	WorkflowID string `json:"workflow_id"`
	Reason     string `json:"reason,omitempty"`
}

func (h *RPCHandler) handleCancelWorkflow(req *RPCRequest) *RPCResponse {
	var params CancelWorkflowParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.WorkflowID == "" {
		return ErrorResponse(ErrInvalidParams, "workflow_id is required")
	}

	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	reason := params.Reason
	if reason == "" {
		reason = "cancelled via rpc"
	}

	if err := h.orchestrator.CancelWorkflow(params.WorkflowID, reason); err != nil {
		return ErrorResponse(ErrInternal, "Failed to cancel workflow: "+err.Error())
	}

	h.log.Info("workflow_cancelled_via_rpc", "workflow_id", params.WorkflowID, "reason", reason, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"workflow_id": params.WorkflowID,
		"status":      "cancelled",
		"reason":      reason,
	})
}

type AdvanceWorkflowParams struct {
	WorkflowID string `json:"workflow_id"`
	StepID     string `json:"step_id"`
}

func (h *RPCHandler) handleAdvanceWorkflow(req *RPCRequest) *RPCResponse {
	var params AdvanceWorkflowParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.WorkflowID == "" {
		return ErrorResponse(ErrInvalidParams, "workflow_id is required")
	}
	if params.StepID == "" {
		return ErrorResponse(ErrInvalidParams, "step_id is required")
	}

	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	if err := h.orchestrator.AdvanceWorkflow(params.WorkflowID, params.StepID); err != nil {
		return ErrorResponse(ErrInternal, "Failed to advance workflow: "+err.Error())
	}

	workflow, _ := h.orchestrator.GetWorkflow(params.WorkflowID)

	h.log.Info("workflow_advanced_via_rpc", "workflow_id", params.WorkflowID, "step_id", params.StepID, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"workflow_id":    params.WorkflowID,
		"completed_step": params.StepID,
		"workflow":       workflow,
	})
}

func (h *RPCHandler) handleGetActiveCount(req *RPCRequest) *RPCResponse {
	if h.orchestrator == nil {
		return ErrorResponse(ErrInternal, "Orchestrator not configured")
	}

	count := h.orchestrator.GetActiveWorkflowCount()

	return SuccessResponse(map[string]interface{}{
		"active_count": count,
	})
}

func (h *RPCHandler) handleIsRunning(req *RPCRequest) *RPCResponse {
	if h.orchestrator == nil {
		return SuccessResponse(map[string]interface{}{
			"running": false,
		})
	}

	return SuccessResponse(map[string]interface{}{
		"running": h.orchestrator.IsRunning(),
	})
}

func (h *RPCHandler) handleShutdown(req *RPCRequest) *RPCResponse {
	if h.orchestrator == nil {
		return SuccessResponse(map[string]interface{}{
			"message": "No orchestrator to shutdown",
		})
	}

	h.orchestrator.Shutdown()

	h.log.Info("orchestrator_shutdown_via_rpc", "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"message": "Orchestrator shutdown complete",
	})
}
