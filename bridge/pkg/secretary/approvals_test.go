package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//=============================================================================
// Mock Store (minimal — only policy methods needed for approval tests)
//=============================================================================

type mockApprovalStore struct {
	mu       sync.RWMutex
	policies map[string]*ApprovalPolicy
	listErr  error
}

func newMockApprovalStore() *mockApprovalStore {
	return &mockApprovalStore{policies: make(map[string]*ApprovalPolicy)}
}

func (m *mockApprovalStore) CreatePolicy(_ context.Context, policy *ApprovalPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockApprovalStore) GetPolicy(_ context.Context, id string) (*ApprovalPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.policies[id]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	return p, nil
}

func (m *mockApprovalStore) ListPolicies(_ context.Context) ([]ApprovalPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []ApprovalPolicy
	for _, p := range m.policies {
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockApprovalStore) UpdatePolicy(_ context.Context, policy *ApprovalPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.policies[policy.ID]; !ok {
		return ErrPolicyNotFound
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockApprovalStore) DeletePolicy(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.policies, id)
	return nil
}

// Unused Store methods — approvals tests don't need these.
func (m *mockApprovalStore) CreateTemplate(_ context.Context, _ *TaskTemplate) error { panic("not implemented") }
func (m *mockApprovalStore) GetTemplate(_ context.Context, _ string) (*TaskTemplate, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) GetTemplateByTrigger(_ context.Context, _ string) (*TaskTemplate, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListTemplates(_ context.Context, _ TemplateFilter) ([]TaskTemplate, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) UpdateTemplate(_ context.Context, _ *TaskTemplate) error {
	panic("not implemented")
}
func (m *mockApprovalStore) DeleteTemplate(_ context.Context, _ string) error { panic("not implemented") }
func (m *mockApprovalStore) CreateWorkflow(_ context.Context, _ *Workflow) error { panic("not implemented") }
func (m *mockApprovalStore) GetWorkflow(_ context.Context, _ string) (*Workflow, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListWorkflows(_ context.Context, _ WorkflowFilter) ([]Workflow, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) UpdateWorkflow(_ context.Context, _ *Workflow) error { panic("not implemented") }
func (m *mockApprovalStore) DeleteWorkflow(_ context.Context, _ string) error { panic("not implemented") }
func (m *mockApprovalStore) CreateScheduledTask(_ context.Context, _ *ScheduledTask) error {
	panic("not implemented")
}
func (m *mockApprovalStore) GetScheduledTask(_ context.Context, _ string) (*ScheduledTask, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListScheduledTasks(_ context.Context) ([]ScheduledTask, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) UpdateScheduledTask(_ context.Context, _ *ScheduledTask) error {
	panic("not implemented")
}
func (m *mockApprovalStore) DeleteScheduledTask(_ context.Context, _ string) error {
	panic("not implemented")
}
func (m *mockApprovalStore) ListPendingScheduledTasks(_ context.Context) ([]ScheduledTask, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListDueTasks(_ context.Context) ([]ScheduledTask, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) MarkDispatched(_ context.Context, _ string, _ time.Time) error {
	panic("not implemented")
}
func (m *mockApprovalStore) CreateNotificationChannel(_ context.Context, _ *NotificationChannel) error {
	panic("not implemented")
}
func (m *mockApprovalStore) GetNotificationChannel(_ context.Context, _ string) (*NotificationChannel, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListNotificationChannels(_ context.Context, _ string) ([]NotificationChannel, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) UpdateNotificationChannel(_ context.Context, _ *NotificationChannel) error {
	panic("not implemented")
}
func (m *mockApprovalStore) DeleteNotificationChannel(_ context.Context, _ string) error {
	panic("not implemented")
}
func (m *mockApprovalStore) CreateContact(_ context.Context, _ *Contact) error { panic("not implemented") }
func (m *mockApprovalStore) GetContact(_ context.Context, _ string) (*Contact, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) ListContacts(_ context.Context, _ ContactFilter) ([]Contact, error) {
	panic("not implemented")
}
func (m *mockApprovalStore) UpdateContact(_ context.Context, _ *Contact) error { panic("not implemented") }
func (m *mockApprovalStore) DeleteContact(_ context.Context, _ string) error { panic("not implemented") }
func (m *mockApprovalStore) Close() error { return nil }

// Helper to create a test engine
func newTestEngine(t *testing.T) *ApprovalEngineImpl {
	t.Helper()
	store := newMockApprovalStore()
	engine, err := NewApprovalEngine(ApprovalEngineConfig{Store: store})
	require.NoError(t, err)
	return engine
}

func newTestEngineWithStore(t *testing.T, store *mockApprovalStore) *ApprovalEngineImpl {
	t.Helper()
	engine, err := NewApprovalEngine(ApprovalEngineConfig{Store: store})
	require.NoError(t, err)
	return engine
}

//=============================================================================
// T5: Core Evaluation Tests (15 tests)
//=============================================================================

func TestNewApprovalEngine_NilStore(t *testing.T) {
	_, err := NewApprovalEngine(ApprovalEngineConfig{Store: nil})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store is required")
}

func TestNewApprovalEngine_Success(t *testing.T) {
	engine, err := NewApprovalEngine(ApprovalEngineConfig{Store: newMockApprovalStore()})
	require.NoError(t, err)
	assert.NotNil(t, engine)
}

func TestEvaluate_NoPIIFields(t *testing.T) {
	engine := newTestEngine(t)
	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{},
	})
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.False(t, result.Required)
}

func TestEvaluate_NoMatchingPolicies(t *testing.T) {
	engine := newTestEngine(t)
	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"card_number"},
	})
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.False(t, result.Required)
}

func TestEvaluate_AutoApprovePolicy(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	conditions, _ := json.Marshal([]Condition{
		{Field: "initiator", Operator: "eq", Value: "@admin:example.com"},
	})

	store.policies["pol-1"] = &ApprovalPolicy{
		ID:          "pol-1",
		Name:        "Auto-approve admin",
		PIIFields:   []string{"card_number"},
		AutoApprove: true,
		Conditions:  conditions,
		IsActive:    true,
		CreatedBy:   "@admin:example.com",
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields:  []string{"card_number"},
		Initiator:  "@admin:example.com",
		Variables:  map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.Contains(t, result.ApprovedFields, "card_number")
}

func TestEvaluate_AutoApprovePolicy_ConditionsNotMet(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	conditions, _ := json.Marshal([]Condition{
		{Field: "initiator", Operator: "eq", Value: "@admin:example.com"},
	})

	store.policies["pol-1"] = &ApprovalPolicy{
		ID:          "pol-1",
		Name:        "Auto-approve admin",
		PIIFields:   []string{"card_number"},
		AutoApprove: true,
		Conditions:  conditions,
		IsActive:    true,
		CreatedBy:   "@admin:example.com",
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields:  []string{"card_number"},
		Initiator:  "@other:example.com",
		Variables:  map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.NeedsApproval)
	assert.True(t, result.Required)
}

func TestEvaluate_ManualApprovalPolicy(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	store.policies["pol-1"] = &ApprovalPolicy{
		ID:          "pol-1",
		Name:        "Manual review",
		PIIFields:   []string{"ssn"},
		AutoApprove: false,
		IsActive:    true,
		CreatedBy:   "@admin:example.com",
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"ssn"},
		Variables: map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.NeedsApproval)
	assert.True(t, result.Required)
}

func TestEvaluate_MultiplePolicies(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	store.policies["pol-allow"] = &ApprovalPolicy{
		ID:          "pol-allow",
		PIIFields:   []string{"email"},
		AutoApprove: true,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	store.policies["pol-deny"] = &ApprovalPolicy{
		ID:          "pol-deny",
		PIIFields:   []string{"email"},
		AutoApprove: false,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"email"},
		Variables: map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEvaluate_MultiplePolicies_DelegateTo(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	store.policies["pol-1"] = &ApprovalPolicy{
		ID:          "pol-1",
		PIIFields:   []string{"card_number"},
		AutoApprove: false,
		DelegateTo:  "@manager:example.com",
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"card_number"},
		Variables: map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.NeedsApproval)
	assert.Equal(t, "@manager:example.com", result.DelegateTo)
}

func TestEvaluateStep(t *testing.T) {
	engine := newTestEngine(t)
	workflow := &Workflow{
		ID:        "wf-1",
		Status:    StatusRunning,
		Variables: map[string]interface{}{"key": "value"},
	}
	step := &WorkflowStep{StepID: "step-1", Type: StepAction}

	result, err := engine.EvaluateStep(context.Background(), workflow, nil, step, []string{"email"}, "@user:example.com")
	require.NoError(t, err)
	assert.True(t, result.Approved)
}

func TestEvaluateWorkflow(t *testing.T) {
	engine := newTestEngine(t)
	workflow := &Workflow{
		ID:        "wf-1",
		Status:    StatusRunning,
		Variables: map[string]interface{}{},
	}
	template := &TaskTemplate{
		ID:      "tpl-1",
		PIIRefs: []string{"card_number"},
	}

	result, err := engine.EvaluateWorkflow(context.Background(), workflow, template, "@user:example.com")
	require.NoError(t, err)
	assert.True(t, result.Approved)
}

func TestEvaluateWorkflow_NilTemplate(t *testing.T) {
	engine := newTestEngine(t)
	workflow := &Workflow{
		ID:        "wf-1",
		Status:    StatusRunning,
		Variables: map[string]interface{}{},
	}

	result, err := engine.EvaluateWorkflow(context.Background(), workflow, nil, "@user:example.com")
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.False(t, result.Required)
}

func TestEvaluate_StoreError(t *testing.T) {
	store := newMockApprovalStore()
	store.listErr = fmt.Errorf("database connection lost")
	engine := newTestEngineWithStore(t, store)

	_, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"card_number"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list policies")
}

func TestEvaluate_InactivePolicy(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	store.policies["pol-inactive"] = &ApprovalPolicy{
		ID:        "pol-inactive",
		PIIFields: []string{"card_number"},
		IsActive:  false,
		CreatedAt: time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"card_number"},
		Variables: map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.False(t, result.Required)
}

func TestEvaluate_MultipleConditions_AllMustPass(t *testing.T) {
	engine := newTestEngine(t)
	store := engine.store.(*mockApprovalStore)

	conditions, _ := json.Marshal([]Condition{
		{Field: "initiator", Operator: "eq", Value: "@admin:example.com"},
		{Field: "subject", Operator: "eq", Value: "payment"},
	})

	store.policies["pol-1"] = &ApprovalPolicy{
		ID:          "pol-1",
		PIIFields:   []string{"card_number"},
		AutoApprove: true,
		Conditions:  conditions,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	result, err := engine.Evaluate(context.Background(), EvaluationContext{
		PIIFields: []string{"card_number"},
		Initiator: "@admin:example.com",
		Subject:   "payment",
		Variables: map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, result.Approved)
}

//=============================================================================
// T6: Policy CRUD + Request Management Tests (19 tests)
//=============================================================================

func TestCreatePolicy(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	policy := &ApprovalPolicy{
		ID:        "pol-test-1",
		Name:      "Test Policy",
		PIIFields: []string{"card_number"},
		CreatedBy: "@admin:example.com",
	}

	err := engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)
	assert.True(t, policy.IsActive)
	assert.False(t, policy.CreatedAt.IsZero())
}

func TestCreatePolicy_EmptyID(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	policy := &ApprovalPolicy{
		Name:      "Test Policy",
		PIIFields: []string{"card_number"},
		CreatedBy: "@admin:example.com",
	}

	err := engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)
	assert.NotEmpty(t, policy.ID)
	assert.Contains(t, policy.ID, "policy_")
}

func TestGetPolicy(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	policy := &ApprovalPolicy{
		ID:        "pol-get-1",
		Name:      "Get Test",
		PIIFields: []string{"email"},
		CreatedBy: "@admin:example.com",
	}
	require.NoError(t, engine.CreatePolicy(ctx, policy))

	found, err := engine.GetPolicy(ctx, "pol-get-1")
	require.NoError(t, err)
	assert.Equal(t, "Get Test", found.Name)
}

func TestGetPolicy_NotFound(t *testing.T) {
	engine := newTestEngine(t)
	_, err := engine.GetPolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPolicyNotFound)
}

func TestListPolicies_ActiveOnly(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreatePolicy(ctx, &ApprovalPolicy{ID: "p1", Name: "Active", PIIFields: []string{"a"}, IsActive: true, CreatedBy: "u"})
	engine.CreatePolicy(ctx, &ApprovalPolicy{ID: "p2", Name: "Inactive", PIIFields: []string{"b"}, IsActive: true, CreatedBy: "u"})

	store := engine.store.(*mockApprovalStore)
	store.policies["p2"].IsActive = false

	policies, err := engine.ListPolicies(ctx, true)
	require.NoError(t, err)
	assert.Len(t, policies, 1)
	assert.Equal(t, "Active", policies[0].Name)
}

func TestListPolicies_All(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreatePolicy(ctx, &ApprovalPolicy{ID: "p1", Name: "Active", PIIFields: []string{"a"}, CreatedBy: "u"})
	engine.CreatePolicy(ctx, &ApprovalPolicy{ID: "p2", Name: "Inactive", PIIFields: []string{"b"}, CreatedBy: "u"})

	store := engine.store.(*mockApprovalStore)
	store.policies["p2"].IsActive = false

	policies, err := engine.ListPolicies(ctx, false)
	require.NoError(t, err)
	assert.Len(t, policies, 2)
}

func TestUpdatePolicy(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	policy := &ApprovalPolicy{ID: "p-upd", Name: "Original", PIIFields: []string{"a"}, CreatedBy: "u"}
	require.NoError(t, engine.CreatePolicy(ctx, policy))

	policy.Name = "Updated"
	require.NoError(t, engine.UpdatePolicy(ctx, policy))

	found, err := engine.GetPolicy(ctx, "p-upd")
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.Name)
}

func TestDeletePolicy(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	policy := &ApprovalPolicy{ID: "p-del", Name: "Delete Me", PIIFields: []string{"a"}, CreatedBy: "u"}
	require.NoError(t, engine.CreatePolicy(ctx, policy))

	require.NoError(t, engine.DeletePolicy(ctx, "p-del"))

	_, err := engine.GetPolicy(ctx, "p-del")
	assert.ErrorIs(t, err, ErrPolicyNotFound)
}

func TestCreateRequest(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	req, err := engine.CreateRequest(ctx, ApprovalRequestConfig{
		ID:        "req-1",
		PolicyID:  "pol-1",
		PIIFields: []string{"card_number"},
		Initiator: "@user:example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "req-1", req.ID)
	assert.Equal(t, ApprovalDecision(""), req.Decision)
}

func TestDecide_Approve(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "req-approve", PolicyID: "pol-1", Initiator: "u"})

	req, err := engine.Decide(ctx, "req-approve", DecisionAllow, "@admin:example.com", "looks good")
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, req.Decision)
	assert.Equal(t, "@admin:example.com", req.DecidedBy)
}

func TestDecide_Deny(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "req-deny", PolicyID: "pol-1", Initiator: "u"})

	req, err := engine.Decide(ctx, "req-deny", DecisionDeny, "@admin:example.com", "too risky")
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, req.Decision)
}

func TestDecide_AlreadyDecided(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "req-decided", PolicyID: "pol-1", Initiator: "u"})
	engine.Decide(ctx, "req-decided", DecisionAllow, "@admin:example.com", "ok")

	_, err := engine.Decide(ctx, "req-decided", DecisionDeny, "@admin:example.com", "changed mind")
	assert.ErrorIs(t, err, ErrRequestAlreadyDecided)
}

func TestDecide_NotFound(t *testing.T) {
	engine := newTestEngine(t)
	_, err := engine.Decide(context.Background(), "nonexistent", DecisionAllow, "@admin:example.com", "ok")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

func TestDecide_InvalidDecision(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "req-invalid", PolicyID: "pol-1", Initiator: "u"})

	_, err := engine.Decide(ctx, "req-invalid", ApprovalDecision("maybe"), "@admin:example.com", "hmm")
	assert.ErrorIs(t, err, ErrInvalidDecision)
}

func TestGetRequest(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{
		ID: "req-get", PolicyID: "pol-1", Initiator: "@user:example.com", PIIFields: []string{"ssn"},
	})

	req, err := engine.GetRequest("req-get")
	require.NoError(t, err)
	assert.Equal(t, "@user:example.com", req.Initiator)
	assert.Equal(t, []string{"ssn"}, req.PIIFields)
}

func TestGetRequest_NotFound(t *testing.T) {
	engine := newTestEngine(t)
	_, err := engine.GetRequest("nonexistent")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

func TestListPendingRequests(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r1", PolicyID: "p1", Initiator: "u"})
	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r2", PolicyID: "p1", Initiator: "u"})
	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r3", PolicyID: "p1", Initiator: "u"})
	engine.Decide(ctx, "r1", DecisionAllow, "admin", "ok")

	pending := engine.ListPendingRequests()
	assert.Len(t, pending, 2)
}

func TestListRequestsByWorkflow(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r1", PolicyID: "p1", WorkflowID: "wf-1", Initiator: "u"})
	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r2", PolicyID: "p1", WorkflowID: "wf-2", Initiator: "u"})
	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r3", PolicyID: "p1", WorkflowID: "wf-1", Initiator: "u"})

	result := engine.ListRequestsByWorkflow("wf-1")
	assert.Len(t, result, 2)
}

func TestGetPendingCount(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()

	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r1", PolicyID: "p1", Initiator: "u"})
	engine.CreateRequest(ctx, ApprovalRequestConfig{ID: "r2", PolicyID: "p1", Initiator: "u"})
	engine.Decide(ctx, "r1", DecisionAllow, "admin", "ok")

	assert.Equal(t, 1, engine.GetPendingCount())
}

//=============================================================================
// T7: Condition Evaluation + Helper Tests (20 tests)
//=============================================================================

func TestEvaluateConditions_EmptyConditions(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{}
	assert.True(t, engine.evaluateConditions(nil, evalCtx))
	assert.True(t, engine.evaluateConditions(json.RawMessage(`[]`), evalCtx))
}

func TestEvaluateConditions_InvalidJSON(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{}
	assert.False(t, engine.evaluateConditions(json.RawMessage(`{invalid}`), evalCtx))
}

func TestEvaluateConditions_SingleCondition_Pass(t *testing.T) {
	engine := newTestEngine(t)
	conditions, _ := json.Marshal([]Condition{
		{Field: "initiator", Operator: "eq", Value: "@admin:example.com"},
	})
	evalCtx := EvaluationContext{Initiator: "@admin:example.com"}
	assert.True(t, engine.evaluateConditions(conditions, evalCtx))
}

func TestEvaluateConditions_SingleCondition_Fail(t *testing.T) {
	engine := newTestEngine(t)
	conditions, _ := json.Marshal([]Condition{
		{Field: "initiator", Operator: "eq", Value: "@admin:example.com"},
	})
	evalCtx := EvaluationContext{Initiator: "@other:example.com"}
	assert.False(t, engine.evaluateConditions(conditions, evalCtx))
}

func TestEvaluateCondition_WorkflowStatus(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{
		Workflow: &Workflow{Status: StatusRunning},
	}
	cond := Condition{Field: "workflow.status", Operator: "eq", Value: "running"}
	assert.True(t, engine.evaluateCondition(cond, evalCtx))
}

func TestEvaluateCondition_TemplateFields(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{
		Template: &TaskTemplate{ID: "tpl-1", Name: "Payment Template"},
	}

	assert.True(t, engine.evaluateCondition(Condition{Field: "template.id", Operator: "eq", Value: "tpl-1"}, evalCtx))
	assert.True(t, engine.evaluateCondition(Condition{Field: "template.name", Operator: "eq", Value: "Payment Template"}, evalCtx))
}

func TestEvaluateCondition_Initiator(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{Initiator: "@admin:example.com"}
	cond := Condition{Field: "initiator", Operator: "eq", Value: "@admin:example.com"}
	assert.True(t, engine.evaluateCondition(cond, evalCtx))
}

func TestEvaluateCondition_VariableFallback(t *testing.T) {
	engine := newTestEngine(t)
	evalCtx := EvaluationContext{
		Variables: map[string]interface{}{"region": "us-west"},
	}
	cond := Condition{Field: "region", Operator: "eq", Value: "us-west"}
	assert.True(t, engine.evaluateCondition(cond, evalCtx))
}

func TestCompareValues_Eq(t *testing.T) {
	engine := newTestEngine(t)
	assert.True(t, engine.compareValues("hello", "eq", "hello"))
	assert.True(t, engine.compareValues("hello", "==", "hello"))
	assert.True(t, engine.compareValues("hello", "=", "hello"))
	assert.False(t, engine.compareValues("hello", "eq", "world"))
}

func TestCompareValues_Neq(t *testing.T) {
	engine := newTestEngine(t)
	assert.True(t, engine.compareValues("a", "neq", "b"))
	assert.True(t, engine.compareValues("a", "!=", "b"))
	assert.False(t, engine.compareValues("a", "neq", "a"))
}

func TestCompareValues_In(t *testing.T) {
	engine := newTestEngine(t)
	list := []interface{}{"alpha", "beta", "gamma"}
	assert.True(t, engine.compareValues("beta", "in", list))
}

func TestCompareValues_In_NotFound(t *testing.T) {
	engine := newTestEngine(t)
	list := []interface{}{"alpha", "beta", "gamma"}
	assert.False(t, engine.compareValues("delta", "in", list))
}

func TestCompareValues_Nin(t *testing.T) {
	engine := newTestEngine(t)
	list := []interface{}{"alpha", "beta", "gamma"}
	assert.True(t, engine.compareValues("delta", "nin", list))
	assert.True(t, engine.compareValues("delta", "not_in", list))
	assert.False(t, engine.compareValues("beta", "nin", list))
}

func TestCompareValues_Contains(t *testing.T) {
	engine := newTestEngine(t)
	assert.True(t, engine.compareValues("hello", "contains", "hello"))
	assert.False(t, engine.compareValues("hello", "contains", "world"))
}

func TestCompareValues_UnknownOperator(t *testing.T) {
	engine := newTestEngine(t)
	assert.False(t, engine.compareValues("a", "matches", "a"))
}

func TestPolicyMatchesFields_MatchFound(t *testing.T) {
	engine := newTestEngine(t)
	assert.True(t, engine.policyMatchesFields([]string{"a", "b"}, []string{"b", "c"}))
}

func TestPolicyMatchesFields_NoMatch(t *testing.T) {
	engine := newTestEngine(t)
	assert.False(t, engine.policyMatchesFields([]string{"a"}, []string{"b", "c"}))
}

func TestMergeFields(t *testing.T) {
	engine := newTestEngine(t)
	result := engine.mergeFields([]string{"a", "b"}, []string{"b", "c"})
	assert.ElementsMatch(t, []string{"a", "b", "c"}, result)
}

func TestSubtractFields(t *testing.T) {
	engine := newTestEngine(t)
	result := engine.subtractFields([]string{"a", "b", "c"}, []string{"b"})
	assert.ElementsMatch(t, []string{"a", "c"}, result)
}

func TestSubtractFields_EmptySubtract(t *testing.T) {
	engine := newTestEngine(t)
	result := engine.subtractFields([]string{"a", "b"}, []string{})
	assert.ElementsMatch(t, []string{"a", "b"}, result)
}
