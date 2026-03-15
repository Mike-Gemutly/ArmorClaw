package secretary

import (
	"context"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Studio Integration for Secretary Workflows
//=============================================================================

// AgentFactoryAdapter wraps studio.AgentFactory for Secretary agent lifecycle
type AgentFactoryAdapter interface {
	Spawn(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error)
	Stop(ctx context.Context, instanceID string, timeout time.Duration) error
	Remove(ctx context.Context, instanceID string) error
	GetStatus(ctx context.Context, instanceID string) (*studio.AgentInstance, error)
	ListInstances(definitionID string) ([]*studio.AgentInstance, error)
}

// StudioAgentFactory wraps studio.AgentFactory to implement AgentFactoryAdapter
type StudioAgentFactory struct {
	factory *studio.AgentFactory
}

func NewStudioAgentFactory(factory *studio.AgentFactory) *StudioAgentFactory {
	return &StudioAgentFactory{
		factory: factory,
	}
}

func (s *StudioAgentFactory) Spawn(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error) {
	return s.factory.Spawn(ctx, req)
}

func (s *StudioAgentFactory) Stop(ctx context.Context, instanceID string, timeout time.Duration) error {
	return s.factory.Stop(ctx, instanceID, timeout)
}

func (s *StudioAgentFactory) Remove(ctx context.Context, instanceID string) error {
	return s.factory.Remove(ctx, instanceID)
}

func (s *StudioAgentFactory) GetStatus(ctx context.Context, instanceID string) (*studio.AgentInstance, error) {
	return s.factory.GetStatus(ctx, instanceID)
}

func (s *StudioAgentFactory) ListInstances(definitionID string) ([]*studio.AgentInstance, error) {
	return s.factory.ListInstances(definitionID)
}

// SecretaryStore defines minimal store interface for studio integration
type SecretaryStore interface {
	GetTemplate(ctx context.Context, templateID string) (*TaskTemplate, error)
	ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error)
	CreateWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) error
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error)
	Close() error
}

//=============================================================================
// Studio Integration
//=============================================================================

// StudioIntegration manages Secretary workflow integration with Agent Studio
type StudioIntegration struct {
	agentFactory   AgentFactoryAdapter
	secretaryStore SecretaryStore
	matrixAdapter  studio.MatrixAdapter
}

// StudioIntegrationConfig holds configuration for studio integration
type StudioIntegrationConfig struct {
	AgentFactory   AgentFactoryAdapter
	SecretaryStore SecretaryStore
	MatrixAdapter  studio.MatrixAdapter
}

// NewStudioIntegration creates a new studio integration
func NewStudioIntegration(cfg StudioIntegrationConfig) *StudioIntegration {
	return &StudioIntegration{
		agentFactory:   cfg.AgentFactory,
		secretaryStore: cfg.SecretaryStore,
		matrixAdapter:  cfg.MatrixAdapter,
	}
}

// SpawnSecretaryAgentRequest contains parameters for spawning a secretary agent
type SpawnSecretaryAgentRequest struct {
	WorkflowID      string
	TaskDescription string
	UserID          string
	RoomID          string
}

// SpawnSecretaryAgentResult contains the result of spawning a secretary agent
type SpawnSecretaryAgentResult struct {
	Instance   *studio.AgentInstance
	Definition *studio.AgentDefinition
	WorkflowID string
}

// SpawnSecretaryAgent creates a new agent instance for Secretary workflow execution
// Uses studio.AgentFactory to spawn containers with proper configuration
func (s *StudioIntegration) SpawnSecretaryAgent(ctx context.Context, req *SpawnSecretaryAgentRequest) (*SpawnSecretaryAgentResult, error) {
	if s.agentFactory == nil {
		return nil, fmt.Errorf("agent factory not configured")
	}

	spawnReq := &studio.SpawnRequest{
		DefinitionID:    "secretary_workflow_agent",
		TaskDescription: req.TaskDescription,
		UserID:          req.UserID,
		RoomID:          req.RoomID,
	}

	result, err := s.agentFactory.Spawn(ctx, spawnReq)
	if err != nil {
		return nil, fmt.Errorf("failed to spawn secretary agent: %w", err)
	}

	return &SpawnSecretaryAgentResult{
		Instance:   result.Instance,
		Definition: result.Definition,
		WorkflowID: req.WorkflowID,
	}, nil
}

// ListSecretaryAgents lists all secretary workflow agent instances
func (s *StudioIntegration) ListSecretaryAgents(ctx context.Context) ([]*studio.AgentInstance, error) {
	if s.agentFactory == nil {
		return nil, fmt.Errorf("agent factory not configured")
	}

	instances, err := s.agentFactory.ListInstances("secretary_workflow_agent")
	if err != nil {
		return nil, fmt.Errorf("failed to list secretary agents: %w", err)
	}

	return instances, nil
}

// DeleteSecretaryAgent stops and removes a secretary agent instance
func (s *StudioIntegration) DeleteSecretaryAgent(ctx context.Context, instanceID string) error {
	if s.agentFactory == nil {
		return fmt.Errorf("agent factory not configured")
	}

	if err := s.agentFactory.Stop(ctx, instanceID, 30*time.Second); err != nil {
		return fmt.Errorf("failed to stop secretary agent: %w", err)
	}

	if err := s.agentFactory.Remove(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to remove secretary agent: %w", err)
	}

	return nil
}

// CreateWorkflowFromTemplate creates a new workflow instance from a template
func (s *StudioIntegration) CreateWorkflowFromTemplate(ctx context.Context, templateID string, variables map[string]interface{}, createdBy string) (*Workflow, error) {
	if s.secretaryStore == nil {
		return nil, fmt.Errorf("secretary store not configured")
	}

	template, err := s.secretaryStore.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %s: %w", templateID, err)
	}

	if template == nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	if !template.IsActive {
		return nil, fmt.Errorf("template is inactive: %s", templateID)
	}

	now := time.Now()
	workflow := &Workflow{
		ID:          fmt.Sprintf("workflow_%d", now.UnixMilli()),
		TemplateID:  templateID,
		Name:        template.Name,
		Description: template.Description,
		Status:      StatusPending,
		Variables:   variables,
		AgentIDs:    []string{},
		CreatedBy:   createdBy,
		StartedAt:   now,
		CompletedAt: nil,
	}

	if err := s.secretaryStore.CreateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

//=============================================================================
// Secretary Agent Registration
//=============================================================================

// RegisterSecretaryAgent registers the Secretary agent type in the Studio store
// This creates a default agent definition for Secretary workflows
func RegisterSecretaryAgent(store studio.Store) error {
	now := time.Now()

	// Check if secretary agent definition already exists
	_, err := store.GetDefinitionByName("Secretary Workflow Agent")
	if err == nil {
		// Already exists, skip registration
		return nil
	}

	// Create the Secretary agent definition
	def := &studio.AgentDefinition{
		ID:          "secretary_workflow_agent",
		Name:        "Secretary Workflow Agent",
		Description: "Agent for executing Secretary workflow templates including document processing, form filling, and approval workflows",
		Skills: []string{
			"workflow_executor",
			"template_filler",
			"document_processor",
			"approval_checker",
		},
		PIIAccess: []string{
			"client_name",
			"client_email",
			"client_address",
			"client_phone",
		},
		ResourceTier: "medium",
		CreatedBy:    "system",
		CreatedAt:    now,
		UpdatedAt:    now,
		IsActive:     true,
	}

	if err := store.CreateDefinition(def); err != nil {
		return fmt.Errorf("failed to register secretary agent: %w", err)
	}

	return nil
}

// RegisterSecretarySkill registers Secretary-specific skills in the Studio skill registry
func RegisterSecretarySkill(skillRegistry *studio.SkillRegistry) error {
	now := time.Now()

	// Define Secretary-specific skills
	skills := []*studio.Skill{
		{
			ID:              "workflow_executor",
			Name:            "Workflow Executor",
			Description:     "Executes Secretary workflow templates with variable substitution and step-by-step processing",
			Category:        "automation",
			RequiredEnvVars: []string{},
			CreatedAt:       now,
		},
		{
			ID:              "template_filler",
			Name:            "Template Filler",
			Description:     "Fills document templates with variable values and outputs formatted documents",
			Category:        "document",
			RequiredEnvVars: []string{},
			CreatedAt:       now,
		},
		{
			ID:              "document_processor",
			Name:            "Document Processor",
			Description:     "Processes and transforms documents including PDF generation and format conversion",
			Category:        "document",
			RequiredEnvVars: []string{},
			CreatedAt:       now,
		},
		{
			ID:              "approval_checker",
			Name:            "Approval Checker",
			Description:     "Checks approval policies and manages approval workflows for sensitive PII access",
			Category:        "automation",
			RequiredEnvVars: []string{},
			CreatedAt:       now,
		},
	}

	// Register each skill
	for _, skill := range skills {
		if err := skillRegistry.Register(skill); err != nil {
			// Check if it's a duplicate error (skill already exists)
			if !skillRegistry.Exists(skill.ID) {
				return fmt.Errorf("failed to register skill %s: %w", skill.ID, err)
			}
		}
	}

	return nil
}

// RegisterSecretaryPIIFields registers Secretary-specific PII fields in the Studio PII registry
func RegisterSecretaryPIIFields(piiRegistry *studio.PIIRegistry) error {
	now := time.Now()

	// Define Secretary-specific PII fields
	fields := []*studio.PIIField{
		{
			ID:               "contract_id",
			Name:             "Contract ID",
			Description:      "Unique identifier for contracts",
			Sensitivity:      "low",
			RequiresApproval: false,
			CreatedAt:        now,
		},
		{
			ID:               "contract_status",
			Name:             "Contract Status",
			Description:      "Current status of a contract",
			Sensitivity:      "low",
			RequiresApproval: false,
			CreatedAt:        now,
		},
		{
			ID:               "workflow_owner",
			Name:             "Workflow Owner",
			Description:      "User who owns/created the workflow",
			Sensitivity:      "low",
			RequiresApproval: false,
			CreatedAt:        now,
		},
		{
			ID:               "approval_delegate",
			Name:             "Approval Delegate",
			Description:      "User delegated to approve requests",
			Sensitivity:      "medium",
			RequiresApproval: true,
			CreatedAt:        now,
		},
	}

	// Register each field
	for _, field := range fields {
		if err := piiRegistry.Register(field); err != nil {
			// Check if it's a duplicate error (field already exists)
			if !piiRegistry.Exists(field.ID) {
				return fmt.Errorf("failed to register PII field %s: %w", field.ID, err)
			}
		}
	}

	return nil
}
