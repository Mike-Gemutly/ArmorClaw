package secretary

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSecretaryCommandHandler_HandleCreateWorkflow(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected success message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event1", "!secretary create workflow template_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleListWorkflows(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected workflows list message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event2", "!secretary list workflows")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleWorkflowStatus(t *testing.T) {
	now := time.Now()
	testWorkflow := &Workflow{
		ID:        "workflow_456",
		Name:      "Test Workflow",
		Status:    StatusRunning,
		StartedAt: now,
	}

	store := &mockStoreWithWorkflow{
		workflow: testWorkflow,
	}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected workflow status message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event3", "!secretary workflow status workflow_456")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

type mockStoreWithWorkflow struct {
	mockStore
	workflow *Workflow
}

func (m *mockStoreWithWorkflow) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	if m.workflow != nil && m.workflow.ID == workflowID {
		return m.workflow, nil
	}
	return nil, errors.New("workflow not found")
}

func TestSecretaryCommandHandler_HandleCancelWorkflow(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected cancel confirmation message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event4", "!secretary cancel workflow workflow_456")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleListAgents(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected agents list message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event5", "!secretary list agents")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleDeleteAgent(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected delete confirmation message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event6", "!secretary delete agent instance_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleListTemplates(t *testing.T) {
	store := &mockStore{}
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			if message == "" {
				t.Error("expected templates list message, got empty")
			}
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event7", "!secretary list templates")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}
}

func TestSecretaryCommandHandler_HandleLearnWebsite_NoService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event8", "!secretary learn website https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected error message about service not configured")
	}
}

func TestSecretaryCommandHandler_HandleLearnWebsite_WithService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	learnService := &mockLearnWebsiteService{}
	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:        store,
		Matrix:       matrix,
		LearnWebsite: learnService,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event8", "!secretary learn website https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected message to be sent")
	}
}

func TestSecretaryCommandHandler_HandleRunBlindFill_NoService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event9", "!secretary run blindfill template_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected error message about service not configured")
	}
}

func TestSecretaryCommandHandler_HandleRunBlindFill_WithService(t *testing.T) {
	store := &mockStore{}
	var capturedMessages []string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessages = append(capturedMessages, message)
			return nil
		},
	}

	blindFill := &mockBlindFillExecutor{}
	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:     store,
		Matrix:    matrix,
		BlindFill: blindFill,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event9", "!secretary run blindfill template_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if len(capturedMessages) == 0 {
		t.Error("expected messages to be sent")
	}
}

func TestSecretaryCommandHandler_HandleListTrustPolicies_NoService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event10", "!secretary trust list")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected error message about service not configured")
	}
}

func TestSecretaryCommandHandler_HandleListTrustPolicies_WithService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	trustEngine := &mockTrustEngine{}
	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:       store,
		Matrix:      matrix,
		TrustEngine: trustEngine,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event10", "!secretary trust list")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected trust policies list message")
	}
}

func TestSecretaryCommandHandler_HandleCreateTrustPolicy_WithService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	trustEngine := &mockTrustEngine{}
	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:       store,
		Matrix:      matrix,
		TrustEngine: trustEngine,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event11", "!secretary trust create TestPolicy template=abc123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected trust policy created message")
	}
}

func TestSecretaryCommandHandler_HandleRevokeTrustPolicy_WithService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	trustEngine := &mockTrustEngine{}
	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:       store,
		Matrix:      matrix,
		TrustEngine: trustEngine,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event12", "!secretary trust revoke policy_123 test reason")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected trust policy revoked message")
	}
}

func TestSecretaryCommandHandler_HandleReviewMapping(t *testing.T) {
	store := &mockStoreWithTemplate{
		template: &TaskTemplate{
			ID:          "draft_123",
			Name:        "Test Mapping",
			Description: "Test mapping description",
			IsActive:    true,
			CreatedAt:   time.Now(),
			Steps: []WorkflowStep{
				{StepID: "step_1", Name: "Fill Email", Order: 0},
				{StepID: "step_2", Name: "Submit Form", Order: 1},
			},
			PIIRefs: []string{"user.email"},
		},
	}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event13", "!secretary review mapping draft_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected mapping review message")
	}
}

func TestSecretaryCommandHandler_HandleConfirmMapping_NoService(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event14", "!secretary confirm mapping draft_123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected error message about service not configured")
	}
}

func TestSecretaryCommandHandler_Help(t *testing.T) {
	store := &mockStore{}
	var capturedMessage string
	matrix := &mockMatrixAdapter{
		sendMessageFunc: func(ctx context.Context, roomID, message string) error {
			capturedMessage = message
			return nil
		},
	}

	handler := NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:  store,
		Matrix: matrix,
	})

	ctx := context.Background()
	handled, err := handler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event15", "!secretary help")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !handled {
		t.Fatal("expected command to be handled")
	}

	if capturedMessage == "" {
		t.Error("expected help message")
	}
}

type mockLearnWebsiteService struct{}

func (m *mockLearnWebsiteService) Learn(ctx context.Context, req *LearnWebsiteRequest) (*LearnWebsiteResult, error) {
	return &LearnWebsiteResult{
		RequestID:    "req_123",
		TargetURL:    req.TargetURL,
		DiscoveredAt: time.Now(),
		Fields: []DiscoveredField{
			{ID: "field_1", Selector: "#email", TagName: "input", LabelText: "Email"},
		},
		MappingDraft: &MappingDraft{
			ID:        "draft_123",
			TargetURL: req.TargetURL,
		},
	}, nil
}

func (m *mockLearnWebsiteService) ConfirmMapping(ctx context.Context, mapping *ConfirmedMapping) error {
	return nil
}

type mockBlindFillExecutor struct{}

func (m *mockBlindFillExecutor) Execute(ctx context.Context, req *BlindFillRequest) (*BlindFillResult, error) {
	now := time.Now()
	return &BlindFillResult{
		ExecutionID:  "exec_123",
		TemplateID:   req.TemplateID,
		Status:       BlindFillStatusCompleted,
		StartedAt:    now,
		CompletedAt:  &now,
		Duration:     100,
		StepResults:  []*BlindFillStepResult{},
		FilledFields: []string{},
	}, nil
}

func (m *mockBlindFillExecutor) Cancel(executionID string) error {
	return nil
}

func (m *mockBlindFillExecutor) GetExecutionStatus(executionID string) (*BlindFillResult, bool) {
	return nil, false
}

func (m *mockBlindFillExecutor) ListActiveExecutions() []string {
	return []string{}
}

func (m *mockBlindFillExecutor) GetActiveExecutionCount() int {
	return 0
}

func (m *mockBlindFillExecutor) ResumeWithApproval(ctx context.Context, executionID string, approvalToken string, piiValues map[string]string) (*BlindFillResult, error) {
	return nil, nil
}

type mockTrustEngine struct{}

func (m *mockTrustEngine) Evaluate(ctx context.Context, req *TrustEvaluationRequest) (*TrustEvaluationResult, error) {
	return &TrustEvaluationResult{
		Decision:       TrustDecisionAllow,
		EvaluationTime: time.Now(),
		AllowedFields:  req.PIIFields,
	}, nil
}

func (m *mockTrustEngine) CreatePolicy(ctx context.Context, policy *TrustedWorkflowPolicy) error {
	policy.ID = "policy_123"
	return nil
}

func (m *mockTrustEngine) GetPolicy(ctx context.Context, id string) (*TrustedWorkflowPolicy, error) {
	return nil, nil
}

func (m *mockTrustEngine) ListPolicies(ctx context.Context, activeOnly bool) ([]*TrustedWorkflowPolicy, error) {
	return []*TrustedWorkflowPolicy{
		{
			ID:        "policy_123",
			Name:      "Test Policy",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}, nil
}

func (m *mockTrustEngine) UpdatePolicy(ctx context.Context, policy *TrustedWorkflowPolicy) error {
	return nil
}

func (m *mockTrustEngine) DeletePolicy(ctx context.Context, id string) error {
	return nil
}

func (m *mockTrustEngine) RevokePolicy(ctx context.Context, id string, revokedBy string, reason string) error {
	return nil
}

func (m *mockTrustEngine) ReconfirmPolicy(ctx context.Context, id string, confirmedBy string) error {
	return nil
}

func (m *mockTrustEngine) RecordExecution(ctx context.Context, policyID string) error {
	return nil
}

func (m *mockTrustEngine) CleanupExpiredPolicies(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockTrustEngine) GetPolicyStats(ctx context.Context, id string) (*PolicyStats, error) {
	return nil, nil
}
