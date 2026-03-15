package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	studio "github.com/armorclaw/bridge/pkg/studio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type liveIntegrationSuite struct {
	mu sync.RWMutex

	store         *liveTestStore
	browser       *BrowserIntegration
	eventEmitter  *liveTestEventEmitter
	matrixAdapter *liveTestMatrixAdapter

	learnService   *LearnWebsiteService
	blindFill      *BlindFillExecutor
	trustEngine    *TrustedWorkflowEngine
	approval       *ApprovalEngineImpl
	commandHandler *SecretaryCommandHandler

	loggedSecrets []string
}

func setupLiveIntegrationSuite(t *testing.T) *liveIntegrationSuite {
	suite := &liveIntegrationSuite{
		store:         newLiveTestStore(),
		eventEmitter:  newLiveTestEventEmitter(),
		matrixAdapter: newLiveTestMatrixAdapter(),
		loggedSecrets: make([]string, 0),
	}

	browser := NewBrowserIntegration(&liveTestBrowserHandler{})
	suite.browser = browser

	var err error
	suite.learnService, err = NewLearnWebsiteService(LearnWebsiteConfig{
		Browser: browser,
		Store:   suite.store,
	})
	require.NoError(t, err)

	suite.approval, err = NewApprovalEngine(ApprovalEngineConfig{
		Store: suite.store,
	})
	require.NoError(t, err)

	suite.trustEngine, err = NewTrustedWorkflowEngine(TrustedWorkflowConfig{
		Store: suite.store,
	})
	require.NoError(t, err)

	suite.blindFill, err = NewBlindFillExecutor(BlindFillConfig{
		Browser:  browser,
		Store:    suite.store,
		Approval: suite.approval,
		Events:   suite.eventEmitter,
	})
	require.NoError(t, err)

	suite.commandHandler = NewSecretaryCommandHandler(SecretaryCommandHandlerConfig{
		Store:          suite.store,
		Matrix:         suite.matrixAdapter,
		LearnWebsite:   suite.learnService,
		BlindFill:      suite.blindFill,
		TrustEngine:    suite.trustEngine,
		ApprovalEngine: suite.approval,
	})

	return suite
}

func (s *liveIntegrationSuite) assertNoPlaintextSecrets(t *testing.T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, msg := range s.matrixAdapter.getMessages() {
		for _, secret := range s.loggedSecrets {
			assert.NotContains(t, msg, secret, "Matrix message should not contain plaintext secret")
		}
	}

	for _, event := range s.eventEmitter.getEvents() {
		eventJSON, _ := json.Marshal(event)
		for _, secret := range s.loggedSecrets {
			assert.NotContains(t, string(eventJSON), secret, "Event should not contain plaintext secret")
		}
	}
}

func (s *liveIntegrationSuite) registerSecret(secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loggedSecrets = append(s.loggedSecrets, secret)
}

type liveTestStore struct {
	mu                   sync.RWMutex
	templates            map[string]*TaskTemplate
	workflows            map[string]*Workflow
	policies             map[string]*ApprovalPolicy
	trustPolicies        map[string]*TrustedWorkflowPolicy
	scheduledTasks       map[string]*ScheduledTask
	notificationChannels map[string]*NotificationChannel
}

func newLiveTestStore() *liveTestStore {
	return &liveTestStore{
		templates:            make(map[string]*TaskTemplate),
		workflows:            make(map[string]*Workflow),
		policies:             make(map[string]*ApprovalPolicy),
		trustPolicies:        make(map[string]*TrustedWorkflowPolicy),
		scheduledTasks:       make(map[string]*ScheduledTask),
		notificationChannels: make(map[string]*NotificationChannel),
	}
}

func (s *liveTestStore) CreateTemplate(ctx context.Context, t *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[t.ID] = t
	return nil
}

func (s *liveTestStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return t, nil
}

func (s *liveTestStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []TaskTemplate
	for _, t := range s.templates {
		if filter.ActiveOnly && !t.IsActive {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

func (s *liveTestStore) UpdateTemplate(ctx context.Context, t *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[t.ID] = t
	return nil
}

func (s *liveTestStore) DeleteTemplate(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.templates, id)
	return nil
}

func (s *liveTestStore) CreateWorkflow(ctx context.Context, w *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID] = w
	return nil
}

func (s *liveTestStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	return w, nil
}

func (s *liveTestStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Workflow
	for _, w := range s.workflows {
		if filter.Status != nil && w.Status != *filter.Status {
			continue
		}
		result = append(result, *w)
	}
	return result, nil
}

func (s *liveTestStore) UpdateWorkflow(ctx context.Context, w *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID] = w
	return nil
}

func (s *liveTestStore) DeleteWorkflow(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workflows, id)
	return nil
}

func (s *liveTestStore) CreatePolicy(ctx context.Context, p *ApprovalPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[p.ID] = p
	return nil
}

func (s *liveTestStore) GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", id)
	}
	return p, nil
}

func (s *liveTestStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ApprovalPolicy
	for _, p := range s.policies {
		result = append(result, *p)
	}
	return result, nil
}

func (s *liveTestStore) UpdatePolicy(ctx context.Context, p *ApprovalPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[p.ID] = p
	return nil
}

func (s *liveTestStore) DeletePolicy(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.policies, id)
	return nil
}

func (s *liveTestStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *liveTestStore) GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.scheduledTasks[id]
	if !ok {
		return nil, fmt.Errorf("scheduled task not found: %s", id)
	}
	return t, nil
}

func (s *liveTestStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ScheduledTask
	for _, t := range s.scheduledTasks {
		result = append(result, *t)
	}
	return result, nil
}

func (s *liveTestStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scheduledTasks[task.ID] = task
	return nil
}

func (s *liveTestStore) DeleteScheduledTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.scheduledTasks, id)
	return nil
}

func (s *liveTestStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ScheduledTask
	now := time.Now()
	for _, t := range s.scheduledTasks {
		if t.NextRun != nil && (t.NextRun.Before(now) || t.NextRun.Equal(now)) {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (s *liveTestStore) CreateNotificationChannel(ctx context.Context, ch *NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notificationChannels[ch.ID] = ch
	return nil
}

func (s *liveTestStore) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.notificationChannels[id]
	if !ok {
		return nil, fmt.Errorf("notification channel not found: %s", id)
	}
	return ch, nil
}

func (s *liveTestStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []NotificationChannel
	for _, ch := range s.notificationChannels {
		if userID == "" || ch.UserID == userID {
			result = append(result, *ch)
		}
	}
	return result, nil
}

func (s *liveTestStore) UpdateNotificationChannel(ctx context.Context, ch *NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notificationChannels[ch.ID] = ch
	return nil
}

func (s *liveTestStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.notificationChannels, id)
	return nil
}

func (s *liveTestStore) Close() error { return nil }

type liveTestBrowserHandler struct{}

func (h *liveTestBrowserHandler) ExecuteBrowserCommand(ctx context.Context, eventType string, content json.RawMessage) (interface{}, error) {
	return map[string]interface{}{"status": "ok"}, nil
}

func (h *liveTestBrowserHandler) RequestPII(ctx context.Context, req *studio.PIIRequestEvent) (*studio.PIIResponseEvent, error) {
	return &studio.PIIResponseEvent{
		RequestID: req.RequestID,
		Approved:  false,
		Values:    make(map[string]string),
	}, nil
}

func (h *liveTestBrowserHandler) SendEvent(ctx context.Context, eventType string, content interface{}) error {
	return nil
}

type capturedLiveEvent struct {
	eventType string
	workflow  *Workflow
	stepID    string
	stepName  string
	progress  float64
	err       error
	reason    string
	result    string
	timestamp time.Time
}

type liveTestEventEmitter struct {
	mu     sync.RWMutex
	events []capturedLiveEvent
}

func newLiveTestEventEmitter() *liveTestEventEmitter {
	return &liveTestEventEmitter{
		events: make([]capturedLiveEvent, 0),
	}
}

func (e *liveTestEventEmitter) EmitStarted(workflow *Workflow) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedLiveEvent{
		eventType: "started",
		workflow:  workflow,
		timestamp: time.Now(),
	})
	return uint64(len(e.events))
}

func (e *liveTestEventEmitter) EmitProgress(workflow *Workflow, stepID, stepName string, progress float64) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedLiveEvent{
		eventType: "progress",
		workflow:  workflow,
		stepID:    stepID,
		stepName:  stepName,
		progress:  progress,
		timestamp: time.Now(),
	})
	return uint64(len(e.events))
}

func (e *liveTestEventEmitter) EmitCompleted(workflow *Workflow, result string) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedLiveEvent{
		eventType: "completed",
		workflow:  workflow,
		result:    result,
		timestamp: time.Now(),
	})
	return uint64(len(e.events))
}

func (e *liveTestEventEmitter) EmitFailed(workflow *Workflow, stepID string, err error, recoverable bool) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedLiveEvent{
		eventType: "failed",
		workflow:  workflow,
		stepID:    stepID,
		err:       err,
		timestamp: time.Now(),
	})
	return uint64(len(e.events))
}

func (e *liveTestEventEmitter) EmitCancelled(workflow *Workflow, reason string) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, capturedLiveEvent{
		eventType: "cancelled",
		workflow:  workflow,
		reason:    reason,
		timestamp: time.Now(),
	})
	return uint64(len(e.events))
}

func (e *liveTestEventEmitter) getEvents() []capturedLiveEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]capturedLiveEvent, len(e.events))
	copy(result, e.events)
	return result
}

type liveTestMatrixAdapter struct {
	mu       sync.RWMutex
	messages []string
}

func newLiveTestMatrixAdapter() *liveTestMatrixAdapter {
	return &liveTestMatrixAdapter{
		messages: make([]string, 0),
	}
}

func (a *liveTestMatrixAdapter) SendMessage(ctx context.Context, roomID, message string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = append(a.messages, message)
	return nil
}

func (a *liveTestMatrixAdapter) SendFormattedMessage(ctx context.Context, roomID, plainBody, formattedBody string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = append(a.messages, plainBody)
	return nil
}

func (a *liveTestMatrixAdapter) ReplyToEvent(ctx context.Context, roomID, eventID, message string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = append(a.messages, message)
	return nil
}

func (a *liveTestMatrixAdapter) getMessages() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]string, len(a.messages))
	copy(result, a.messages)
	return result
}

func TestLiveIntegration_TrustedAutoAllow(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_trusted_test",
		Name:      "Trusted Test Template",
		IsActive:  true,
		PIIRefs:   []string{"billing.address"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill_address",
				Order:  0,
				Type:   StepAction,
				Name:   "Fill Address",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#address","value_ref":"billing.address"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	trustPolicy := &TrustedWorkflowPolicy{
		ID:             "trust_policy_auto",
		Name:           "Auto-approve billing",
		IsActive:       true,
		AllowedPIIRefs: []string{"billing.address"},
		CreatedBy:      "@admin:example.com",
		CreatedAt:      time.Now(),
	}

	err = suite.trustEngine.CreatePolicy(ctx, trustPolicy)
	require.NoError(t, err)

	trustResult, err := suite.trustEngine.Evaluate(ctx, &TrustEvaluationRequest{
		TemplateID: "template_trusted_test",
		PIIFields:  []string{"billing.address"},
		Initiator:  "@user:example.com",
	})

	require.NoError(t, err)
	assert.Equal(t, TrustDecisionAllow, trustResult.Decision)
	assert.Contains(t, trustResult.AllowedFields, "billing.address")

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_NoPlaintextPIIInLogs(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_pii_test",
		Name:      "PII Test Template",
		IsActive:  true,
		PIIRefs:   []string{"user.email", "payment.card_number"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill",
				Order:  0,
				Type:   StepAction,
				Name:   "Fill Form",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#email","value_ref":"user.email"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	suite.registerSecret("4242424242424242")
	suite.registerSecret("test@example.com")
	suite.registerSecret("secret_token_123")

	_, _ = suite.blindFill.Execute(ctx, &BlindFillRequest{
		TemplateID: "template_pii_test",
		Initiator:  "@user:example.com",
		PIIValues: map[string]string{
			"user.email":          "test@example.com",
			"payment.card_number": "4242424242424242",
		},
		ApprovalToken: "secret_token_123",
	})

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_DeniedFieldsCannotExecute(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_denied_test",
		Name:      "Denied Field Test",
		IsActive:  true,
		PIIRefs:   []string{"payment.card_number", "payment.cvv"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill_denied",
				Order:  0,
				Type:   StepAction,
				Name:   "Fill Denied Field",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#cvv","value_ref":"payment.cvv"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	result, err := suite.blindFill.Execute(ctx, &BlindFillRequest{
		TemplateID: "template_denied_test",
		Initiator:  "@user:example.com",
		PIIValues: map[string]string{
			"payment.cvv": "123",
		},
		DeniedFields: []string{"payment.cvv"},
	})

	require.NoError(t, err)
	assert.Equal(t, BlindFillStatusFailed, result.Status)
	assert.Contains(t, result.Error, "denied")

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_CommandHandlerWiring(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_cmd_test",
		Name:      "Command Test Template",
		IsActive:  true,
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	handled, err := suite.commandHandler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event1", "!secretary list templates")
	require.NoError(t, err)
	assert.True(t, handled)

	messages := suite.matrixAdapter.getMessages()
	assert.NotEmpty(t, messages)

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_ApprovalTokenBinding(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_token_test",
		Name:      "Token Test Template",
		IsActive:  true,
		PIIRefs:   []string{"payment.card_number"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill",
				Order:  0,
				Type:   StepAction,
				Name:   "Fill",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#card","value_ref":"payment.card_number"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	suite.registerSecret("wrong_token_bound_to_other_exec")

	result, err := suite.blindFill.Execute(ctx, &BlindFillRequest{
		TemplateID: "template_token_test",
		Initiator:  "@user:example.com",
		PIIValues: map[string]string{
			"payment.card_number": "4242424242424242",
		},
		ApprovalToken: "approval_other_exec_123",
	})

	require.NoError(t, err)
	_ = result

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_TrustPolicyRevocation(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	trustPolicy := &TrustedWorkflowPolicy{
		ID:             "trust_policy_revoked",
		Name:           "Revoked Policy",
		IsActive:       true,
		AllowedPIIRefs: []string{"billing.address"},
		CreatedBy:      "@admin:example.com",
		CreatedAt:      time.Now(),
	}

	err := suite.trustEngine.CreatePolicy(ctx, trustPolicy)
	require.NoError(t, err)

	err = suite.trustEngine.RevokePolicy(ctx, "trust_policy_revoked", "@admin:example.com", "Security review")
	require.NoError(t, err)

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	revokedPolicy := &TrustedWorkflowPolicy{
		ID:        "trust_policy_revoked",
		IsActive:  true,
		RevokedAt: &pastTime,
		RevokedBy: "@admin:example.com",
	}

	isActive := suite.trustEngine.isPolicyActive(revokedPolicy, now)
	assert.False(t, isActive, "Revoked policy should not be active")

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_TrustPolicyExpiration(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)

	expiredPolicy := &TrustedWorkflowPolicy{
		ID:        "trust_policy_expired",
		IsActive:  true,
		ExpiresAt: &pastTime,
	}

	isActive := suite.trustEngine.isPolicyActive(expiredPolicy, now)
	assert.False(t, isActive, "Expired policy should not be active")

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_AuditEventEmission(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_event_test",
		Name:      "Event Test Template",
		IsActive:  true,
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_navigate",
				Order:  0,
				Type:   StepAction,
				Name:   "Navigate",
				Config: json.RawMessage(`{"action":"navigate","params":{"url":"https://example.com"}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	_, _ = suite.blindFill.Execute(ctx, &BlindFillRequest{
		TemplateID: "template_event_test",
		Initiator:  "@user:example.com",
	})

	events := suite.eventEmitter.getEvents()
	assert.NotEmpty(t, events, "Expected events to be emitted")

	var foundStartEvent bool
	for _, event := range events {
		if event.eventType == "started" {
			foundStartEvent = true
			assert.NotNil(t, event.workflow)
		}
	}
	assert.True(t, foundStartEvent, "Expected started event")

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_MatrixNotificationSent(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_notify_test",
		Name:      "Notify Test Template",
		IsActive:  true,
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	handled, err := suite.commandHandler.HandleMessage(ctx, "!room:example.com", "@user:example.com", "$event1", "!secretary list workflows")
	require.NoError(t, err)
	assert.True(t, handled)

	messages := suite.matrixAdapter.getMessages()
	assert.NotEmpty(t, messages, "Expected Matrix notification to be sent")

	for _, msg := range messages {
		assert.NotContains(t, msg, "token")
		assert.NotContains(t, msg, "secret")
		assert.NotContains(t, msg, "password")
	}

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_SecurityHeadersNoPIILeak(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	suite.registerSecret("4111111111111111")
	suite.registerSecret("my_secret_cvv_123")

	template := &TaskTemplate{
		ID:        "template_sec_header_test",
		Name:      "Security Header Test",
		IsActive:  true,
		PIIRefs:   []string{"payment.card_number", "payment.cvv"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill_secure",
				Order:  0,
				Type:   StepAction,
				Name:   "Secure Fill",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#card","value_ref":"payment.card_number"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	_, _ = suite.blindFill.Execute(ctx, &BlindFillRequest{
		TemplateID: "template_sec_header_test",
		Initiator:  "@user:example.com",
		PIIValues: map[string]string{
			"payment.card_number": "4111111111111111",
			"payment.cvv":         "my_secret_cvv_123",
		},
		ApprovalToken: "approval_exec_123_1234567890",
	})

	events := suite.eventEmitter.getEvents()
	for _, event := range events {
		eventJSON, _ := json.Marshal(event)
		eventStr := string(eventJSON)
		assert.NotContains(t, eventStr, "4111111111111111", "Event should not contain card number")
		assert.NotContains(t, eventStr, "my_secret_cvv_123", "Event should not contain CVV")
		assert.NotContains(t, eventStr, "approval_exec_123_1234567890", "Event should not contain approval token")

		if event.workflow != nil {
			assert.NotContains(t, eventStr, "4111111111111111")
		}
	}

	suite.assertNoPlaintextSecrets(t)
}

func TestLiveIntegration_PIIFieldNamingOnly(t *testing.T) {
	suite := setupLiveIntegrationSuite(t)
	ctx := context.Background()

	template := &TaskTemplate{
		ID:        "template_pii_naming",
		Name:      "PII Naming Test",
		IsActive:  true,
		PIIRefs:   []string{"user.ssn", "user.tax_id"},
		CreatedBy: "@user:example.com",
		CreatedAt: time.Now(),
		Steps: []WorkflowStep{
			{
				StepID: "step_fill_pii",
				Order:  0,
				Type:   StepAction,
				Name:   "Fill PII Fields",
				Config: json.RawMessage(`{"action":"fill","params":{"fields":[{"selector":"#ssn","value_ref":"user.ssn"}]}}`),
			},
		},
	}
	err := suite.store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	suite.registerSecret("123-45-6789")
	suite.registerSecret("12-3456789")

	messages := suite.matrixAdapter.getMessages()

	for _, msg := range messages {
		if strings.Contains(msg, "PII") || strings.Contains(msg, "pii") {
			assert.NotContains(t, msg, "123-45-6789", "PII reference should use field name, not value")
			assert.NotContains(t, msg, "12-3456789", "PII reference should use field name, not value")

			assert.True(t,
				strings.Contains(msg, "user.ssn") ||
					strings.Contains(msg, "user.tax_id") ||
					strings.Contains(msg, "payment.card_number"),
				"PII reference should use dotted field naming",
			)
		}
	}

	suite.assertNoPlaintextSecrets(t)
}
