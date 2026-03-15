package secretary

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/studio"
)

// Mock implementations for testing

type mockFactory struct {
	spawnFunc     func(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error)
	stopFunc      func(ctx context.Context, instanceID string, timeout time.Duration) error
	removeFunc    func(ctx context.Context, instanceID string) error
	getStatusFunc func(ctx context.Context, instanceID string) (*studio.AgentInstance, error)
	listFunc      func(definitionID string) ([]*studio.AgentInstance, error)
}

func (m *mockFactory) Spawn(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error) {
	if m.spawnFunc != nil {
		return m.spawnFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockFactory) Stop(ctx context.Context, instanceID string, timeout time.Duration) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx, instanceID, timeout)
	}
	return nil
}

func (m *mockFactory) Remove(ctx context.Context, instanceID string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, instanceID)
	}
	return nil
}

func (m *mockFactory) GetStatus(ctx context.Context, instanceID string) (*studio.AgentInstance, error) {
	if m.getStatusFunc != nil {
		return m.getStatusFunc(ctx, instanceID)
	}
	return nil, nil
}

func (m *mockFactory) ListInstances(definitionID string) ([]*studio.AgentInstance, error) {
	if m.listFunc != nil {
		return m.listFunc(definitionID)
	}
	return nil, nil
}

type mockStore struct {
	secretaryStore Store
}

func (m *mockStore) CreateTemplate(ctx context.Context, template *TaskTemplate) error {
	if m.secretaryStore != nil {
		return m.secretaryStore.CreateTemplate(ctx, template)
	}
	return nil
}

func (m *mockStore) GetTemplate(ctx context.Context, templateID string) (*TaskTemplate, error) {
	if m.secretaryStore != nil {
		return m.secretaryStore.GetTemplate(ctx, templateID)
	}
	return nil, nil
}

func (m *mockStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	if m.secretaryStore != nil {
		return m.secretaryStore.ListTemplates(ctx, filter)
	}
	return nil, nil
}

func (m *mockStore) UpdateTemplate(ctx context.Context, template *TaskTemplate) error {
	if m.secretaryStore != nil {
		return m.secretaryStore.UpdateTemplate(ctx, template)
	}
	return nil
}

func (m *mockStore) DeleteTemplate(ctx context.Context, templateID string) error {
	if m.secretaryStore != nil {
		return m.secretaryStore.DeleteTemplate(ctx, templateID)
	}
	return nil
}

func (m *mockStore) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}

func (m *mockStore) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	return nil, nil
}

func (m *mockStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	return nil, nil
}

func (m *mockStore) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}

func (m *mockStore) DeleteWorkflow(ctx context.Context, workflowID string) error {
	return nil
}

func (m *mockStore) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}

func (m *mockStore) GetPolicy(ctx context.Context, policyID string) (*ApprovalPolicy, error) {
	return nil, nil
}

func (m *mockStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	return nil, nil
}

func (m *mockStore) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	return nil
}

func (m *mockStore) DeletePolicy(ctx context.Context, policyID string) error {
	return nil
}

func (m *mockStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	return nil
}

func (m *mockStore) GetScheduledTask(ctx context.Context, taskID string) (*ScheduledTask, error) {
	return nil, nil
}

func (m *mockStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}

func (m *mockStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	return nil
}

func (m *mockStore) DeleteScheduledTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return nil, nil
}

func (m *mockStore) CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}

func (m *mockStore) GetNotificationChannel(ctx context.Context, channelID string) (*NotificationChannel, error) {
	return nil, nil
}

func (m *mockStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	return nil, nil
}

func (m *mockStore) UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}

func (m *mockStore) DeleteNotificationChannel(ctx context.Context, channelID string) error {
	return nil
}

func (m *mockStore) Close() error {
	return nil
}

type mockMatrixAdapter struct {
	sendMessageFunc func(ctx context.Context, roomID, message string) error
}

func (m *mockMatrixAdapter) SendMessage(ctx context.Context, roomID, message string) error {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, roomID, message)
	}
	return nil
}

func (m *mockMatrixAdapter) SendFormattedMessage(ctx context.Context, roomID, plainBody, formattedBody string) error {
	return nil
}

func (m *mockMatrixAdapter) ReplyToEvent(ctx context.Context, roomID, eventID, message string) error {
	return nil
}

func TestStudioIntegration_SpawnSecretaryAgent(t *testing.T) {
	// Given: A studio integration with mock factory
	mockFactory := &mockFactory{
		spawnFunc: func(ctx context.Context, req *studio.SpawnRequest) (*studio.SpawnResult, error) {
			// Verify the spawn request is properly formatted for Secretary agent
			if req.DefinitionID != "secretary_workflow_agent" {
				t.Errorf("expected DefinitionID to be 'secretary_workflow_agent', got %s", req.DefinitionID)
			}
			if req.TaskDescription != "Execute secretary workflow" {
				t.Errorf("expected TaskDescription 'Execute secretary workflow', got %s", req.TaskDescription)
			}
			if req.UserID != "@user:example.com" {
				t.Errorf("expected UserID '@user:example.com', got %s", req.UserID)
			}

			// Return a successful spawn result
			return &studio.SpawnResult{
				Instance: &studio.AgentInstance{
					ID:              "instance_123",
					DefinitionID:    "secretary_workflow_agent",
					ContainerID:     "container_abc",
					Status:          studio.StatusRunning,
					TaskDescription: req.TaskDescription,
					SpawnedBy:       req.UserID,
					StartedAt:       func() *time.Time { t := time.Now(); return &t }(),
				},
				Definition: &studio.AgentDefinition{
					ID:     "secretary_workflow_agent",
					Name:   "Secretary Workflow Agent",
					Skills: []string{"workflow_orchestration"},
				},
			}, nil
		},
	}

	integration := NewStudioIntegration(StudioIntegrationConfig{
		AgentFactory: mockFactory,
	})

	// When: Spawning a secretary agent
	ctx := context.Background()
	req := &SpawnSecretaryAgentRequest{
		WorkflowID:      "workflow_456",
		TaskDescription: "Execute secretary workflow",
		UserID:          "@user:example.com",
		RoomID:          "!room:example.com",
	}

	result, err := integration.SpawnSecretaryAgent(ctx, req)

	// Then: Should succeed with proper instance and definition
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Instance == nil {
		t.Fatal("expected Instance to be set")
	}

	if result.Instance.ID != "instance_123" {
		t.Errorf("expected instance ID 'instance_123', got %s", result.Instance.ID)
	}

	if result.Definition == nil {
		t.Fatal("expected Definition to be set")
	}

	if result.Definition.ID != "secretary_workflow_agent" {
		t.Errorf("expected definition ID 'secretary_workflow_agent', got %s", result.Definition.ID)
	}
}

func TestStudioIntegration_ListSecretaryAgents(t *testing.T) {
	// Given: A studio integration with mock factory
	factory := &mockFactory{
		listFunc: func(definitionID string) ([]*studio.AgentInstance, error) {
			return []*studio.AgentInstance{}, nil
		},
	}

	integration := NewStudioIntegration(StudioIntegrationConfig{
		AgentFactory: factory,
	})

	// When: Listing secretary agents
	agents, err := integration.ListSecretaryAgents(context.Background())

	// Then: Should return list of agents
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if agents == nil {
		t.Fatal("expected agents list, got nil")
	}
}

func TestStudioIntegration_DeleteSecretaryAgent(t *testing.T) {
	// Given: A studio integration with mock factory
	factory := &mockFactory{}

	integration := NewStudioIntegration(StudioIntegrationConfig{
		AgentFactory: factory,
	})

	// When: Deleting a secretary agent
	err := integration.DeleteSecretaryAgent(context.Background(), "instance_123")

	// Then: Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStudioIntegration_CreateWorkflowFromTemplate(t *testing.T) {
	// Given: A studio integration with a store that returns a valid template
	store := &mockStoreWithTemplate{
		template: &TaskTemplate{
			ID:          "template_123",
			Name:        "Test Template",
			Description: "A test template",
			IsActive:    true,
		},
	}
	integration := NewStudioIntegration(StudioIntegrationConfig{
		SecretaryStore: store,
	})

	// When: Creating a workflow from template
	workflow, err := integration.CreateWorkflowFromTemplate(context.Background(), "template_123", map[string]interface{}{
		"input": "test_value",
	}, "@user:example.com")

	// Then: Should create workflow with proper structure
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if workflow == nil {
		t.Fatal("expected workflow to be created")
	}

	if workflow.TemplateID != "template_123" {
		t.Errorf("expected template ID 'template_123', got %s", workflow.TemplateID)
	}

	if workflow.CreatedBy != "@user:example.com" {
		t.Errorf("expected created by '@user:example.com', got %s", workflow.CreatedBy)
	}

	if workflow.Status != StatusPending {
		t.Errorf("expected status 'pending', got %s", workflow.Status)
	}
}

type mockStoreWithTemplate struct {
	mockStore
	template *TaskTemplate
}

func (m *mockStoreWithTemplate) GetTemplate(ctx context.Context, templateID string) (*TaskTemplate, error) {
	if m.template != nil && m.template.ID == templateID {
		return m.template, nil
	}
	return nil, fmt.Errorf("template not found: %s", templateID)
}
