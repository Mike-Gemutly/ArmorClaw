package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// RPC Error Types
//=============================================================================

const (
	ErrInvalidParams = -32602
	ErrInternal      = -32603
	ErrNotFound      = -32001
	ErrValidation    = -32002
	ErrUnauthorized  = -32003
	ErrConflict      = -32004
)

//=============================================================================
// RPC Request/Response Types
//=============================================================================

// RPCRequest represents a JSON-RPC 2.0 request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	UserID  string          `json:"-"` // Set by auth middleware
}

// RPCResponse represents a JSON-RPC 2.0 response
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Helper functions
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
// Studio RPC Handler
//=============================================================================

// RPCHandler provides JSON-RPC handlers for the Agent Studio
type RPCHandler struct {
	store           Store
	skillRegistry   *SkillRegistry
	piiRegistry     *PIIRegistry
	profileManager  *ProfileManager
	approvalManager *ApprovalManager
	mcpRegistry     McpRegistry
	factory         *AgentFactory
	log             *logger.Logger
}

// RPCHandlerConfig holds configuration for the RPC handler
type RPCHandlerConfig struct {
	Store   Store
	Logger  *logger.Logger
	Factory *AgentFactory
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(cfg RPCHandlerConfig) *RPCHandler {
	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("studio_rpc")
	}

	mcpRegistry := NewMcpRegistry()
	approvalManager := NewApprovalManager(ApprovalManagerConfig{
		Store:    cfg.Store,
		Registry: mcpRegistry,
	})

	return &RPCHandler{
		store:           cfg.Store,
		skillRegistry:   NewSkillRegistry(cfg.Store),
		piiRegistry:     NewPIIRegistry(cfg.Store),
		profileManager:  NewProfileManager(cfg.Store),
		approvalManager: approvalManager,
		mcpRegistry:     mcpRegistry,
		factory:         cfg.Factory,
		log:             log,
	}
}

// Handle routes requests to appropriate handlers
func (h *RPCHandler) Handle(req *RPCRequest) *RPCResponse {
	switch req.Method {
	// Skill Registry
	case "studio.list_skills":
		return h.handleListSkills(req)
	case "studio.get_skill":
		return h.handleGetSkill(req)
	case "studio.register_skill":
		return h.handleRegisterSkill(req)

	// PII Registry
	case "studio.list_pii":
		return h.handleListPII(req)
	case "studio.get_pii":
		return h.handleGetPII(req)
	case "studio.register_pii":
		return h.handleRegisterPII(req)

	// Agent Definitions
	case "studio.create_agent":
		return h.handleCreateAgent(req)
	case "studio.get_agent":
		return h.handleGetAgent(req)
	case "studio.list_agents":
		return h.handleListAgents(req)
	case "studio.update_agent":
		return h.handleUpdateAgent(req)
	case "studio.delete_agent":
		return h.handleDeleteAgent(req)

	// Agent Instances
	case "studio.spawn_agent":
		return h.handleSpawnAgent(req)
	case "studio.get_instance":
		return h.handleGetInstance(req)
	case "studio.list_instances":
		return h.handleListInstances(req)
	case "studio.stop_instance":
		return h.handleStopInstance(req)

	// Statistics
	case "studio.stats":
		return h.handleStats(req)

	// Resource Profiles
	case "studio.list_profiles":
		return h.handleListProfiles(req)

	// MCP Registry
	case "studio.list_mcps":
		return h.handleListMcps(req)
	case "studio.get_mcp":
		return h.handleGetMcp(req)
	case "studio.get_mcp_warning":
		return h.handleGetMcpWarning(req)

	// MCP Approval Workflow
	case "studio.request_mcp_approval":
		return h.handleRequestMcpApproval(req)
	case "studio.list_pending_approvals":
		return h.handleListPendingApprovals(req)
	case "studio.list_my_approvals":
		return h.handleListMyApprovals(req)
	case "studio.approve_mcp_request":
		return h.handleApproveMcpRequest(req)
	case "studio.reject_mcp_request":
		return h.handleRejectMcpRequest(req)

	default:
		return ErrorResponse(ErrNotFound, fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

//=============================================================================
// Skill Registry Handlers
//=============================================================================

// ListSkillsParams is the params for studio.list_skills
type ListSkillsParams struct {
	Category string `json:"category,omitempty"`
}

func (h *RPCHandler) handleListSkills(req *RPCRequest) *RPCResponse {
	var params ListSkillsParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	skills, err := h.skillRegistry.List(params.Category)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list skills: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	})
}

// GetSkillParams is the params for studio.get_skill
type GetSkillParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleGetSkill(req *RPCRequest) *RPCResponse {
	var params GetSkillParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Skill ID is required")
	}

	skill, err := h.skillRegistry.Get(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Skill not found: "+params.ID)
	}

	return SuccessResponse(skill)
}

// RegisterSkillParams is the params for studio.register_skill
type RegisterSkillParams struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	ContainerImage  string   `json:"container_image,omitempty"`
	RequiredEnvVars []string `json:"required_env_vars,omitempty"`
}

func (h *RPCHandler) handleRegisterSkill(req *RPCRequest) *RPCResponse {
	var params RegisterSkillParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	skill := &Skill{
		ID:              params.ID,
		Name:            params.Name,
		Description:     params.Description,
		Category:        params.Category,
		ContainerImage:  params.ContainerImage,
		RequiredEnvVars: params.RequiredEnvVars,
		CreatedAt:       time.Now(),
	}

	if err := h.skillRegistry.Register(skill); err != nil {
		return ErrorResponse(ErrValidation, err.Error())
	}

	h.log.Info("skill_registered", "id", skill.ID, "name", skill.Name, "by", req.UserID)

	return SuccessResponse(skill)
}

//=============================================================================
// PII Registry Handlers
//=============================================================================

// ListPIIParams is the params for studio.list_pii
type ListPIIParams struct {
	Sensitivity string `json:"sensitivity,omitempty"`
}

func (h *RPCHandler) handleListPII(req *RPCRequest) *RPCResponse {
	var params ListPIIParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	fields, err := h.piiRegistry.List(params.Sensitivity)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list PII fields: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"fields": fields,
		"count":  len(fields),
	})
}

// GetPIIParams is the params for studio.get_pii
type GetPIIParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleGetPII(req *RPCRequest) *RPCResponse {
	var params GetPIIParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "PII field ID is required")
	}

	field, err := h.piiRegistry.Get(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "PII field not found: "+params.ID)
	}

	return SuccessResponse(field)
}

// RegisterPIIParams is the params for studio.register_pii
type RegisterPIIParams struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Sensitivity      string `json:"sensitivity"`
	KeystoreKey      string `json:"keystore_key,omitempty"`
	RequiresApproval bool   `json:"requires_approval"`
}

func (h *RPCHandler) handleRegisterPII(req *RPCRequest) *RPCResponse {
	var params RegisterPIIParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	field := &PIIField{
		ID:               params.ID,
		Name:             params.Name,
		Description:      params.Description,
		Sensitivity:      params.Sensitivity,
		KeystoreKey:      params.KeystoreKey,
		RequiresApproval: params.RequiresApproval,
		CreatedAt:        time.Now(),
	}

	if err := h.piiRegistry.Register(field); err != nil {
		return ErrorResponse(ErrValidation, err.Error())
	}

	h.log.Info("pii_field_registered", "id", field.ID, "name", field.Name, "by", req.UserID)

	return SuccessResponse(field)
}

//=============================================================================
// Agent Definition Handlers
//=============================================================================

// CreateAgentParams is the params for studio.create_agent
type CreateAgentParams struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Skills       []string `json:"skills"`
	PIIAccess    []string `json:"pii_access"`
	ResourceTier string   `json:"resource_tier,omitempty"`
}

func (h *RPCHandler) handleCreateAgent(req *RPCRequest) *RPCResponse {
	var params CreateAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	// Validate required fields
	if params.Name == "" {
		return ErrorResponse(ErrInvalidParams, "Agent name is required")
	}
	if len(params.Skills) == 0 {
		return ErrorResponse(ErrInvalidParams, "At least one skill is required")
	}

	// Validate skills
	skillResult := h.skillRegistry.Validate(params.Skills)
	if !skillResult.Valid {
		return ErrorResponse(ErrValidation, "Invalid skills: "+skillResult.Message, skillResult.InvalidIDs)
	}

	// Validate PII fields
	piiResult := h.piiRegistry.Validate(params.PIIAccess)
	if !piiResult.Valid {
		return ErrorResponse(ErrValidation, "Invalid PII fields: "+piiResult.Message, piiResult.InvalidIDs)
	}

	// Validate resource tier
	if params.ResourceTier == "" {
		params.ResourceTier = "medium"
	}
	if err := h.profileManager.Validate(params.ResourceTier); err != nil {
		return ErrorResponse(ErrValidation, err.Error())
	}

	// Check for duplicate name
	if _, err := h.store.GetDefinitionByName(params.Name); err == nil {
		return ErrorResponse(ErrConflict, "Agent with this name already exists")
	}

	// Create definition
	now := time.Now()
	def := &AgentDefinition{
		ID:           generateID("agent"),
		Name:         params.Name,
		Description:  params.Description,
		Skills:       params.Skills,
		PIIAccess:    params.PIIAccess,
		ResourceTier: params.ResourceTier,
		CreatedBy:    req.UserID,
		CreatedAt:    now,
		UpdatedAt:    now,
		IsActive:     true,
	}

	if err := h.store.CreateDefinition(def); err != nil {
		return ErrorResponse(ErrInternal, "Failed to create agent: "+err.Error())
	}

	h.log.Info("agent_created",
		"id", def.ID,
		"name", def.Name,
		"skills", def.Skills,
		"pii_access", def.PIIAccess,
		"by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"agent":             def,
		"requires_approval": piiResult.RequiresApproval,
	})
}

// GetAgentParams is the params for studio.get_agent
type GetAgentParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleGetAgent(req *RPCRequest) *RPCResponse {
	var params GetAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Agent ID is required")
	}

	def, err := h.store.GetDefinition(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Agent not found: "+params.ID)
	}

	return SuccessResponse(def)
}

// ListAgentsParams is the params for studio.list_agents
type ListAgentsParams struct {
	ActiveOnly bool `json:"active_only"`
}

func (h *RPCHandler) handleListAgents(req *RPCRequest) *RPCResponse {
	var params ListAgentsParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	definitions, err := h.store.ListDefinitions(params.ActiveOnly)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list agents: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"agents": definitions,
		"count":  len(definitions),
	})
}

// UpdateAgentParams is the params for studio.update_agent
type UpdateAgentParams struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	PIIAccess    []string `json:"pii_access,omitempty"`
	ResourceTier string   `json:"resource_tier,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
}

func (h *RPCHandler) handleUpdateAgent(req *RPCRequest) *RPCResponse {
	var params UpdateAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Agent ID is required")
	}

	// Get existing definition
	def, err := h.store.GetDefinition(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Agent not found: "+params.ID)
	}

	// Update fields
	if params.Name != "" {
		// Check for duplicate name
		if existing, err := h.store.GetDefinitionByName(params.Name); err == nil && existing.ID != params.ID {
			return ErrorResponse(ErrConflict, "Agent with this name already exists")
		}
		def.Name = params.Name
	}

	if params.Description != "" {
		def.Description = params.Description
	}

	if len(params.Skills) > 0 {
		skillResult := h.skillRegistry.Validate(params.Skills)
		if !skillResult.Valid {
			return ErrorResponse(ErrValidation, "Invalid skills: "+skillResult.Message)
		}
		def.Skills = params.Skills
	}

	if params.PIIAccess != nil {
		piiResult := h.piiRegistry.Validate(params.PIIAccess)
		if !piiResult.Valid {
			return ErrorResponse(ErrValidation, "Invalid PII fields: "+piiResult.Message)
		}
		def.PIIAccess = params.PIIAccess
	}

	if params.ResourceTier != "" {
		if err := h.profileManager.Validate(params.ResourceTier); err != nil {
			return ErrorResponse(ErrValidation, err.Error())
		}
		def.ResourceTier = params.ResourceTier
	}

	if params.IsActive != nil {
		def.IsActive = *params.IsActive
	}

	if err := h.store.UpdateDefinition(def); err != nil {
		return ErrorResponse(ErrInternal, "Failed to update agent: "+err.Error())
	}

	h.log.Info("agent_updated", "id", def.ID, "name", def.Name, "by", req.UserID)

	return SuccessResponse(def)
}

// DeleteAgentParams is the params for studio.delete_agent
type DeleteAgentParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleDeleteAgent(req *RPCRequest) *RPCResponse {
	var params DeleteAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Agent ID is required")
	}

	// Check for running instances
	instances, err := h.store.ListInstances(params.ID, StatusRunning)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to check instances: "+err.Error())
	}
	if len(instances) > 0 {
		return ErrorResponse(ErrConflict, "Cannot delete agent with running instances")
	}

	if err := h.store.DeleteDefinition(params.ID); err != nil {
		return ErrorResponse(ErrInternal, "Failed to delete agent: "+err.Error())
	}

	h.log.Info("agent_deleted", "id", params.ID, "by", req.UserID)

	return SuccessResponse(map[string]string{
		"id":      params.ID,
		"message": "Agent deleted successfully",
	})
}

//=============================================================================
// Agent Instance Handlers
//=============================================================================

// SpawnAgentParams is the params for studio.spawn_agent
type SpawnAgentParams struct {
	ID              string `json:"id"`
	TaskDescription string `json:"task_description,omitempty"`
}

func (h *RPCHandler) handleSpawnAgent(req *RPCRequest) *RPCResponse {
	var params SpawnAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Agent ID is required")
	}

	// Get definition
	def, err := h.store.GetDefinition(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Agent not found: "+params.ID)
	}

	if !def.IsActive {
		return ErrorResponse(ErrValidation, "Agent is not active")
	}

	// Check for PII fields requiring approval
	requiresApproval := h.piiRegistry.GetFieldsRequiringApproval(def.PIIAccess)
	if len(requiresApproval) > 0 {
		return SuccessResponse(map[string]interface{}{
			"status":            "approval_required",
			"message":           "Agent requires approval for sensitive PII access",
			"requires_approval": requiresApproval,
			"definition":        def,
			"instruction":       "Use pii.request to request access, then retry spawn",
		})
	}

	// Create instance record
	now := time.Now()
	instance := &AgentInstance{
		ID:              generateID("inst"),
		DefinitionID:    def.ID,
		Status:          StatusPending,
		TaskDescription: params.TaskDescription,
		SpawnedBy:       req.UserID,
		StartedAt:       &now,
	}

	if err := h.store.CreateInstance(instance); err != nil {
		return ErrorResponse(ErrInternal, "Failed to create instance: "+err.Error())
	}

	h.log.Info("agent_spawned",
		"instance_id", instance.ID,
		"definition_id", def.ID,
		"name", def.Name,
		"by", req.UserID)

	// Spawn container via AgentFactory (Phase 4)
	if h.factory != nil {
		spawnResult, spawnErr := h.factory.Spawn(context.Background(), &SpawnRequest{
			DefinitionID:    def.ID,
			TaskDescription: params.TaskDescription,
			UserID:          req.UserID,
		})
		if spawnErr != nil {
			instance.Status = StatusFailed
			now := time.Now()
			instance.CompletedAt = &now
			h.store.UpdateInstance(instance)
			return ErrorResponse(ErrInternal, "Failed to spawn container: "+spawnErr.Error())
		}

		// Use the real instance from the factory (it has the real ContainerID)
		instance = spawnResult.Instance

		// Build response with any factory warnings
		warnings := []string{}
		if len(spawnResult.Warnings) > 0 {
			warnings = spawnResult.Warnings
		}

		return SuccessResponse(map[string]interface{}{
			"instance":   instance,
			"definition": def,
			"profile":    GetProfile(def.ResourceTier),
			"warnings":   warnings,
			"message":    "Agent instance spawned successfully",
		})
	}

	// Fallback: no factory available (development mode)
	h.log.Warn("agent_spawn_no_factory",
		"instance_id", instance.ID,
		"definition_id", def.ID,
		"message", "AgentFactory not configured, using stub spawn")
	instance.Status = StatusRunning
	instance.ContainerID = "pending-container-" + instance.ID
	h.store.UpdateInstance(instance)

	return SuccessResponse(map[string]interface{}{
		"instance":   instance,
		"definition": def,
		"profile":    GetProfile(def.ResourceTier),
		"message":    "Agent instance spawned successfully",
	})
}

// GetInstanceParams is the params for studio.get_instance
type GetInstanceParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleGetInstance(req *RPCRequest) *RPCResponse {
	var params GetInstanceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Instance ID is required")
	}

	instance, err := h.store.GetInstance(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Instance not found: "+params.ID)
	}

	// Get definition for context
	def, _ := h.store.GetDefinition(instance.DefinitionID)

	return SuccessResponse(map[string]interface{}{
		"instance":   instance,
		"definition": def,
	})
}

// ListInstancesParams is the params for studio.list_instances
type ListInstancesParams struct {
	DefinitionID string         `json:"definition_id,omitempty"`
	Status       InstanceStatus `json:"status,omitempty"`
}

func (h *RPCHandler) handleListInstances(req *RPCRequest) *RPCResponse {
	var params ListInstancesParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	instances, err := h.store.ListInstances(params.DefinitionID, params.Status)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list instances: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"instances": instances,
		"count":     len(instances),
	})
}

// StopInstanceParams is the params for studio.stop_instance
type StopInstanceParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleStopInstance(req *RPCRequest) *RPCResponse {
	var params StopInstanceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "Instance ID is required")
	}

	instance, err := h.store.GetInstance(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "Instance not found: "+params.ID)
	}

	if instance.Status != StatusRunning && instance.Status != StatusPending {
		return ErrorResponse(ErrValidation, "Instance is not running")
	}

	// Stop container via AgentFactory (Phase 4)
	if h.factory != nil {
		stopErr := h.factory.Stop(context.Background(), instance.ID, 30*time.Second)
		if stopErr != nil {
			h.log.Error("container_stop_failed", "instance_id", instance.ID, "error", stopErr)
		}
	}

	now := time.Now()
	instance.Status = StatusCancelled
	instance.CompletedAt = &now

	if err := h.store.UpdateInstance(instance); err != nil {
		return ErrorResponse(ErrInternal, "Failed to stop instance: "+err.Error())
	}

	h.log.Info("instance_stopped", "id", params.ID, "by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"instance": instance,
		"message":  "Instance stopped successfully",
	})
}

//=============================================================================
// Statistics Handler
//=============================================================================

func (h *RPCHandler) handleStats(req *RPCRequest) *RPCResponse {
	stats, err := h.store.GetStats()
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to get stats: "+err.Error())
	}

	return SuccessResponse(stats)
}

//=============================================================================
// Resource Profiles Handler
//=============================================================================

func (h *RPCHandler) handleListProfiles(req *RPCRequest) *RPCResponse {
	profiles := h.profileManager.List()
	return SuccessResponse(map[string]interface{}{
		"profiles": profiles,
	})
}

//=============================================================================
// MCP Registry Handlers
//=============================================================================

// ListMcpsParams is the params for studio.list_mcps
type ListMcpsParams struct {
	Category   string       `json:"category,omitempty"`
	RiskLevel  McpRiskLevel `json:"risk_level,omitempty"`
	ActiveOnly bool         `json:"active_only,omitempty"`
}

func (h *RPCHandler) handleListMcps(req *RPCRequest) *RPCResponse {
	var params ListMcpsParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
		}
	}

	filter := &McpFilter{
		Category:   params.Category,
		RiskLevel:  params.RiskLevel,
		ActiveOnly: params.ActiveOnly,
	}

	mcps, err := h.mcpRegistry.ListMcps(filter)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list MCPs: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"mcps":  mcps,
		"count": len(mcps),
	})
}

// GetMcpParams is the params for studio.get_mcp
type GetMcpParams struct {
	ID string `json:"id"`
}

func (h *RPCHandler) handleGetMcp(req *RPCRequest) *RPCResponse {
	var params GetMcpParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ID == "" {
		return ErrorResponse(ErrInvalidParams, "MCP ID is required")
	}

	mcp, err := h.mcpRegistry.GetMcp(params.ID)
	if err != nil {
		return ErrorResponse(ErrNotFound, "MCP not found: "+params.ID)
	}

	return SuccessResponse(mcp)
}

// GetMcpWarningParams is the params for studio.get_mcp_warning
type GetMcpWarningParams struct {
	MCPId string `json:"mcp_id"`
}

func (h *RPCHandler) handleGetMcpWarning(req *RPCRequest) *RPCResponse {
	var params GetMcpWarningParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.MCPId == "" {
		return ErrorResponse(ErrInvalidParams, "MCP ID is required")
	}

	// Get risk assessment
	assessment, err := h.approvalManager.GetMcpRiskAssessment(params.MCPId)
	if err != nil {
		return ErrorResponse(ErrNotFound, "MCP not found: "+params.MCPId)
	}

	// Log the warning view for audit purposes
	_ = LogMcpAction("VIEW_WARNING", params.MCPId, req.UserID, RoleAdmin)
	h.log.Info("mcp_warning_viewed",
		"mcp_id", params.MCPId,
		"user_id", req.UserID,
		"risk_level", assessment.MCP.RiskLevel)

	return SuccessResponse(&McpWarningResponse{
		MCP:            assessment.MCP,
		RiskAssessment: assessment,
		AuditLogged:    true,
	})
}

//=============================================================================
// MCP Approval Workflow Handlers
//=============================================================================

// RequestMcpApprovalParams is the params for studio.request_mcp_approval
type RequestMcpApprovalParams struct {
	MCPId     string `json:"mcp_id"`
	AgentName string `json:"agent_name"`
	Reason    string `json:"reason,omitempty"`
}

func (h *RPCHandler) handleRequestMcpApproval(req *RPCRequest) *RPCResponse {
	var params RequestMcpApprovalParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.MCPId == "" {
		return ErrorResponse(ErrInvalidParams, "MCP ID is required")
	}
	if params.AgentName == "" {
		return ErrorResponse(ErrInvalidParams, "Agent name is required")
	}
	if req.UserID == "" {
		return ErrorResponse(ErrInvalidParams, "User ID is required")
	}

	approvalReq := &McpApprovalRequest{
		MCPId:       params.MCPId,
		AgentName:   params.AgentName,
		Reason:      params.Reason,
		RequestedBy: req.UserID,
	}

	result, err := h.approvalManager.CreateApprovalRequest(nil, approvalReq)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to create approval request: "+err.Error())
	}

	h.log.Info("mcp_approval_requested",
		"approval_id", result.ID,
		"mcp_id", params.MCPId,
		"agent_name", params.AgentName,
		"user_id", req.UserID)

	return SuccessResponse(&McpApprovalResponse{
		ApprovalID: result.ID,
		Status:     result.Status,
		MCPId:      result.MCPId,
		MCPName:    result.MCPName,
		CreatedAt:  result.CreatedAt,
		ExpiresAt:  result.ExpiresAt,
	})
}

// ListPendingApprovalsParams is the params for studio.list_pending_approvals
type ListPendingApprovalsParams struct{}

func (h *RPCHandler) handleListPendingApprovals(req *RPCRequest) *RPCResponse {
	approvals, err := h.approvalManager.ListPendingApprovals(nil)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list pending approvals: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"approvals": approvals,
		"count":     len(approvals),
	})
}

// ListMyApprovalsParams is the params for studio.list_my_approvals
type ListMyApprovalsParams struct{}

func (h *RPCHandler) handleListMyApprovals(req *RPCRequest) *RPCResponse {
	if req.UserID == "" {
		return ErrorResponse(ErrInvalidParams, "User ID is required")
	}

	approvals, err := h.approvalManager.ListUserApprovals(nil, req.UserID)
	if err != nil {
		return ErrorResponse(ErrInternal, "Failed to list user approvals: "+err.Error())
	}

	return SuccessResponse(map[string]interface{}{
		"approvals": approvals,
		"count":     len(approvals),
	})
}

// ApproveMcpRequestParams is the params for studio.approve_mcp_request
type ApproveMcpRequestParams struct {
	ApprovalID string `json:"approval_id"`
	Notes      string `json:"notes,omitempty"`
}

func (h *RPCHandler) handleApproveMcpRequest(req *RPCRequest) *RPCResponse {
	var params ApproveMcpRequestParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ApprovalID == "" {
		return ErrorResponse(ErrInvalidParams, "Approval ID is required")
	}
	if req.UserID == "" {
		return ErrorResponse(ErrInvalidParams, "User ID is required")
	}

	if err := h.approvalManager.ApproveRequest(nil, params.ApprovalID, req.UserID, params.Notes); err != nil {
		return ErrorResponse(ErrInternal, "Failed to approve request: "+err.Error())
	}

	h.log.Info("mcp_approval_approved",
		"approval_id", params.ApprovalID,
		"reviewed_by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"approval_id": params.ApprovalID,
		"status":      "APPROVED",
		"message":     "MCP access request approved",
	})
}

// RejectMcpRequestParams is the params for studio.reject_mcp_request
type RejectMcpRequestParams struct {
	ApprovalID string `json:"approval_id"`
	Notes      string `json:"notes,omitempty"`
}

func (h *RPCHandler) handleRejectMcpRequest(req *RPCRequest) *RPCResponse {
	var params RejectMcpRequestParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(ErrInvalidParams, "Invalid params: "+err.Error())
	}

	if params.ApprovalID == "" {
		return ErrorResponse(ErrInvalidParams, "Approval ID is required")
	}
	if req.UserID == "" {
		return ErrorResponse(ErrInvalidParams, "User ID is required")
	}

	if err := h.approvalManager.RejectRequest(nil, params.ApprovalID, req.UserID, params.Notes); err != nil {
		return ErrorResponse(ErrInternal, "Failed to reject request: "+err.Error())
	}

	h.log.Info("mcp_approval_rejected",
		"approval_id", params.ApprovalID,
		"reviewed_by", req.UserID)

	return SuccessResponse(map[string]interface{}{
		"approval_id": params.ApprovalID,
		"status":      "REJECTED",
		"message":     "MCP access request rejected",
	})
}

//=============================================================================
// Helper Functions
//=============================================================================

var idCounter int64

func generateID(prefix string) string {
	// Use counter for uniqueness within same millisecond
	idCounter++
	return fmt.Sprintf("%s_%d_%d_%s", prefix, time.Now().UnixMilli(), idCounter, randomSuffix(4))
}

func randomSuffix(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	// Use nanosecond variation for randomness
	ns := time.Now().Nanosecond()
	for i := range b {
		b[i] = letters[(ns+i*17+int(idCounter)*31)%len(letters)]
	}
	return string(b)
}
