package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// Approval Decision Types
//=============================================================================

type ApprovalDecision string

const (
	DecisionAllow           ApprovalDecision = "allow"
	DecisionDeny            ApprovalDecision = "deny"
	DecisionRequireApproval ApprovalDecision = "require_approval"
)

//=============================================================================
// Approval Request Types
//=============================================================================

type ApprovalRequest struct {
	ID             string                 `json:"id"`
	PolicyID       string                 `json:"policy_id"`
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	TemplateID     string                 `json:"template_id,omitempty"`
	StepID         string                 `json:"step_id,omitempty"`
	PIIFields      []string               `json:"pii_fields"`
	Initiator      string                 `json:"initiator"`
	Subject        string                 `json:"subject,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	Decision       ApprovalDecision       `json:"decision"`
	DecidedAt      *time.Time             `json:"decided_at,omitempty"`
	DecidedBy      string                 `json:"decided_by,omitempty"`
	DecisionReason string                 `json:"decision_reason,omitempty"`
}

type ApprovalRequestConfig struct {
	ID         string
	PolicyID   string
	WorkflowID string
	TemplateID string
	StepID     string
	PIIFields  []string
	Initiator  string
	Subject    string
	Context    map[string]interface{}
}

//=============================================================================
// Policy Evaluation Context
//=============================================================================

type EvaluationContext struct {
	Workflow  *Workflow              `json:"workflow,omitempty"`
	Template  *TaskTemplate          `json:"template,omitempty"`
	Step      *WorkflowStep          `json:"step,omitempty"`
	Initiator string                 `json:"initiator"`
	Subject   string                 `json:"subject,omitempty"`
	PIIFields []string               `json:"pii_fields"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

//=============================================================================
// Policy Engine Errors
//=============================================================================

var (
	ErrPolicyNotFound        = fmt.Errorf("policy not found")
	ErrRequestNotFound       = fmt.Errorf("approval request not found")
	ErrRequestAlreadyDecided = fmt.Errorf("approval request already decided")
	ErrInvalidDecision       = fmt.Errorf("invalid decision")
	ErrNoMatchingPolicy      = fmt.Errorf("no matching policy found")
)

//=============================================================================
// Approval Engine Implementation
//=============================================================================

type ApprovalEngineImpl struct {
	mu       sync.RWMutex
	store    Store
	requests map[string]*ApprovalRequest
	log      *logger.Logger
	ctx      context.Context
}

type ApprovalEngineConfig struct {
	Store  Store
	Logger *logger.Logger
}

func NewApprovalEngine(cfg ApprovalEngineConfig) (*ApprovalEngineImpl, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("approval_engine")
	}

	return &ApprovalEngineImpl{
		store:    cfg.Store,
		requests: make(map[string]*ApprovalRequest),
		log:      log,
		ctx:      context.Background(),
	}, nil
}

//=============================================================================
// Policy Evaluation
//=============================================================================

func (e *ApprovalEngineImpl) Evaluate(ctx context.Context, evalCtx EvaluationContext) (*ApprovalResult, error) {
	if len(evalCtx.PIIFields) == 0 {
		return &ApprovalResult{
			RequestID: generateApprovalID("req"),
			Required:  false,
			Approved:  true,
			Context:   evalCtx.Variables,
		}, nil
	}

	policies, err := e.store.ListPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	var matchingPolicies []ApprovalPolicy
	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}
		if e.policyMatchesFields(policy.PIIFields, evalCtx.PIIFields) {
			matchingPolicies = append(matchingPolicies, policy)
		}
	}

	if len(matchingPolicies) == 0 {
		return &ApprovalResult{
			RequestID:      generateApprovalID("req"),
			Required:       false,
			Approved:       true,
			ApprovedFields: evalCtx.PIIFields,
			Context:        evalCtx.Variables,
		}, nil
	}

	return e.evaluatePolicies(ctx, matchingPolicies, evalCtx)
}

func (e *ApprovalEngineImpl) EvaluateStep(ctx context.Context, workflow *Workflow, template *TaskTemplate, step *WorkflowStep, piiFields []string, initiator string) (*ApprovalResult, error) {
	evalCtx := EvaluationContext{
		Workflow:  workflow,
		Template:  template,
		Step:      step,
		Initiator: initiator,
		PIIFields: piiFields,
		Variables: workflow.Variables,
		Timestamp: time.Now(),
	}

	return e.Evaluate(ctx, evalCtx)
}

func (e *ApprovalEngineImpl) EvaluateWorkflow(ctx context.Context, workflow *Workflow, template *TaskTemplate, initiator string) (*ApprovalResult, error) {
	var piiFields []string
	if template != nil {
		piiFields = template.PIIRefs
	}

	evalCtx := EvaluationContext{
		Workflow:  workflow,
		Template:  template,
		Initiator: initiator,
		PIIFields: piiFields,
		Variables: workflow.Variables,
		Timestamp: time.Now(),
	}

	return e.Evaluate(ctx, evalCtx)
}

func (e *ApprovalEngineImpl) evaluatePolicies(ctx context.Context, policies []ApprovalPolicy, evalCtx EvaluationContext) (*ApprovalResult, error) {
	result := &ApprovalResult{
		RequestID:     generateApprovalID("req"),
		Required:      false,
		Approved:      false,
		NeedsApproval: false,
		Context:       evalCtx.Variables,
	}

	var approvedFields []string
	var deniedFields []string
	var requireApprovalFields []string
	var delegateTo string

	for _, policy := range policies {
		policyDecision := e.evaluateSinglePolicy(policy, evalCtx)

		switch policyDecision {
		case DecisionAllow:
			approvedFields = e.mergeFields(approvedFields, policy.PIIFields)
		case DecisionDeny:
			deniedFields = e.mergeFields(deniedFields, policy.PIIFields)
		case DecisionRequireApproval:
			requireApprovalFields = e.mergeFields(requireApprovalFields, policy.PIIFields)
			if policy.DelegateTo != "" {
				delegateTo = policy.DelegateTo
			}
			result.NeedsApproval = true
			result.Required = true
		}
	}

	approvedFields = e.subtractFields(approvedFields, deniedFields)
	requireApprovalFields = e.subtractFields(requireApprovalFields, approvedFields)
	requireApprovalFields = e.subtractFields(requireApprovalFields, deniedFields)

	result.ApprovedFields = approvedFields
	result.DeniedFields = deniedFields

	if len(requireApprovalFields) > 0 {
		result.NeedsApproval = true
		result.Required = true
		result.DelegateTo = delegateTo
	} else if len(deniedFields) > 0 && len(approvedFields) == 0 {
		result.Approved = false
	} else {
		result.Approved = true
	}

	return result, nil
}

func (e *ApprovalEngineImpl) evaluateSinglePolicy(policy ApprovalPolicy, evalCtx EvaluationContext) ApprovalDecision {
	if policy.AutoApprove {
		if e.evaluateConditions(policy.Conditions, evalCtx) {
			return DecisionAllow
		}
		return DecisionRequireApproval
	}

	if len(policy.Conditions) > 0 {
		if e.evaluateConditions(policy.Conditions, evalCtx) {
			return DecisionAllow
		}
	}

	return DecisionRequireApproval
}

func (e *ApprovalEngineImpl) evaluateConditions(conditions json.RawMessage, evalCtx EvaluationContext) bool {
	if len(conditions) == 0 {
		return true
	}

	var conds []Condition
	if err := json.Unmarshal(conditions, &conds); err != nil {
		e.log.Warn("failed_to_parse_conditions", "error", err.Error())
		return false
	}

	for _, cond := range conds {
		if !e.evaluateCondition(cond, evalCtx) {
			return false
		}
	}

	return true
}

type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

func (e *ApprovalEngineImpl) evaluateCondition(cond Condition, evalCtx EvaluationContext) bool {
	var fieldValue interface{}

	switch cond.Field {
	case "workflow.status":
		if evalCtx.Workflow != nil {
			fieldValue = string(evalCtx.Workflow.Status)
		}
	case "workflow.created_by":
		if evalCtx.Workflow != nil {
			fieldValue = evalCtx.Workflow.CreatedBy
		}
	case "template.id":
		if evalCtx.Template != nil {
			fieldValue = evalCtx.Template.ID
		}
	case "template.name":
		if evalCtx.Template != nil {
			fieldValue = evalCtx.Template.Name
		}
	case "step.type":
		if evalCtx.Step != nil {
			fieldValue = string(evalCtx.Step.Type)
		}
	case "step.id":
		if evalCtx.Step != nil {
			fieldValue = evalCtx.Step.StepID
		}
	case "initiator":
		fieldValue = evalCtx.Initiator
	case "subject":
		fieldValue = evalCtx.Subject
	default:
		if evalCtx.Variables != nil {
			fieldValue = evalCtx.Variables[cond.Field]
		}
	}

	return e.compareValues(fieldValue, cond.Operator, cond.Value)
}

func (e *ApprovalEngineImpl) compareValues(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "==", "=":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "neq", "!=":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "in":
		if arr, ok := expected.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", v) {
					return true
				}
			}
			return false
		}
		return false
	case "nin", "not_in":
		if arr, ok := expected.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", v) {
					return false
				}
			}
			return true
		}
		return true
	case "contains":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	default:
		return false
	}
}

//=============================================================================
// Approval Request Management
//=============================================================================

func (e *ApprovalEngineImpl) CreateRequest(ctx context.Context, cfg ApprovalRequestConfig) (*ApprovalRequest, error) {
	req := &ApprovalRequest{
		ID:         cfg.ID,
		PolicyID:   cfg.PolicyID,
		WorkflowID: cfg.WorkflowID,
		TemplateID: cfg.TemplateID,
		StepID:     cfg.StepID,
		PIIFields:  cfg.PIIFields,
		Initiator:  cfg.Initiator,
		Subject:    cfg.Subject,
		Context:    cfg.Context,
		CreatedAt:  time.Now(),
		Decision:   "",
	}

	if req.ID == "" {
		req.ID = generateApprovalID("req")
	}

	e.mu.Lock()
	e.requests[req.ID] = req
	e.mu.Unlock()

	e.log.Info("approval_request_created",
		"request_id", req.ID,
		"policy_id", req.PolicyID,
		"workflow_id", req.WorkflowID,
		"initiator", req.Initiator,
		"pii_fields", req.PIIFields)

	return req, nil
}

func (e *ApprovalEngineImpl) Decide(ctx context.Context, requestID string, decision ApprovalDecision, decidedBy string, reason string) (*ApprovalRequest, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	req, exists := e.requests[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}

	if req.Decision != "" {
		return nil, ErrRequestAlreadyDecided
	}

	if decision != DecisionAllow && decision != DecisionDeny {
		return nil, ErrInvalidDecision
	}

	now := time.Now()
	req.Decision = decision
	req.DecidedAt = &now
	req.DecidedBy = decidedBy
	req.DecisionReason = reason

	e.log.Info("approval_request_decided",
		"request_id", requestID,
		"decision", decision,
		"decided_by", decidedBy,
		"reason", reason)

	return req, nil
}

func (e *ApprovalEngineImpl) Approve(ctx context.Context, requestID string, decidedBy string, reason string) (*ApprovalRequest, error) {
	return e.Decide(ctx, requestID, DecisionAllow, decidedBy, reason)
}

func (e *ApprovalEngineImpl) Deny(ctx context.Context, requestID string, decidedBy string, reason string) (*ApprovalRequest, error) {
	return e.Decide(ctx, requestID, DecisionDeny, decidedBy, reason)
}

func (e *ApprovalEngineImpl) GetRequest(requestID string) (*ApprovalRequest, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	req, exists := e.requests[requestID]
	if !exists {
		return nil, ErrRequestNotFound
	}

	copy := *req
	return &copy, nil
}

func (e *ApprovalEngineImpl) ListPendingRequests() []*ApprovalRequest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range e.requests {
		if req.Decision == "" {
			copy := *req
			pending = append(pending, &copy)
		}
	}
	return pending
}

func (e *ApprovalEngineImpl) ListRequestsByWorkflow(workflowID string) []*ApprovalRequest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*ApprovalRequest
	for _, req := range e.requests {
		if req.WorkflowID == workflowID {
			copy := *req
			result = append(result, &copy)
		}
	}
	return result
}

func (e *ApprovalEngineImpl) GetPendingCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	count := 0
	for _, req := range e.requests {
		if req.Decision == "" {
			count++
		}
	}
	return count
}

//=============================================================================
// Policy Management (delegates to store)
//=============================================================================

func (e *ApprovalEngineImpl) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	if policy.ID == "" {
		policy.ID = generateApprovalID("policy")
	}
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}
	policy.IsActive = true

	if err := e.store.CreatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	e.log.Info("policy_created", "policy_id", policy.ID, "name", policy.Name)
	return nil
}

func (e *ApprovalEngineImpl) GetPolicy(ctx context.Context, policyID string) (*ApprovalPolicy, error) {
	return e.store.GetPolicy(ctx, policyID)
}

func (e *ApprovalEngineImpl) ListPolicies(ctx context.Context, activeOnly bool) ([]*ApprovalPolicy, error) {
	policies, err := e.store.ListPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	result := make([]*ApprovalPolicy, 0, len(policies))
	for i := range policies {
		if activeOnly && !policies[i].IsActive {
			continue
		}
		result = append(result, &policies[i])
	}
	return result, nil
}

func (e *ApprovalEngineImpl) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	if err := e.store.UpdatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	e.log.Info("policy_updated", "policy_id", policy.ID)
	return nil
}

func (e *ApprovalEngineImpl) DeletePolicy(ctx context.Context, policyID string) error {
	if err := e.store.DeletePolicy(ctx, policyID); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	e.log.Info("policy_deleted", "policy_id", policyID)
	return nil
}

//=============================================================================
// Helper Methods
//=============================================================================

func (e *ApprovalEngineImpl) policyMatchesFields(policyFields []string, requestedFields []string) bool {
	for _, pf := range policyFields {
		for _, rf := range requestedFields {
			if pf == rf {
				return true
			}
		}
	}
	return false
}

func (e *ApprovalEngineImpl) mergeFields(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, f := range a {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	for _, f := range b {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	return result
}

func (e *ApprovalEngineImpl) subtractFields(from, subtract []string) []string {
	subtractSet := make(map[string]bool)
	for _, f := range subtract {
		subtractSet[f] = true
	}

	result := make([]string, 0)
	for _, f := range from {
		if !subtractSet[f] {
			result = append(result, f)
		}
	}
	return result
}

//=============================================================================
// ID Generation
//=============================================================================

var approvalIDCounter int64

func generateApprovalID(prefix string) string {
	approvalIDCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), approvalIDCounter)
}
