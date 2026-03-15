package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/google/uuid"
)

//=============================================================================
// Trusted Workflow Policy Types
//=============================================================================

// TrustedWorkflowPolicy defines a policy for automatic execution of approved workflows.
// It allows scoped automatic execution without requiring repeated manual approval.
type TrustedWorkflowPolicy struct {
	// ID is the unique identifier for this trust policy
	ID string `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description explains what this trust policy allows
	Description string `json:"description,omitempty"`

	// Scope defines the boundaries of trust
	Scope TrustScope `json:"scope"`

	// AllowedPIIClasses defines PII field classes that are auto-allowed
	// Examples: "user.profile", "user.contact", "payment.defaults"
	AllowedPIIClasses []string `json:"allowed_pii_classes,omitempty"`

	// AllowedPIIRefs defines specific PII field refs that are auto-allowed
	// Examples: "user.email", "user.phone", "payment.card_last_four"
	AllowedPIIRefs []string `json:"allowed_pii_refs,omitempty"`

	// DeniedPIIRefs defines PII refs that are explicitly denied (overrides allowed)
	DeniedPIIRefs []string `json:"denied_pii_refs,omitempty"`

	// MaxExecutions limits total executions (0 = unlimited)
	MaxExecutions int `json:"max_executions,omitempty"`

	// ExecutionCount tracks how many times this policy has been used
	ExecutionCount int `json:"execution_count"`

	// ExpiresAt is when this trust policy expires (nil = no expiration)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// RevokedAt is when this trust policy was revoked (nil = not revoked)
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	// RevokedBy is who revoked the policy
	RevokedBy string `json:"revoked_by,omitempty"`

	// RevocationReason explains why the policy was revoked
	RevocationReason string `json:"revocation_reason,omitempty"`

	// CreatedBy is the Matrix user ID who created this policy
	CreatedBy string `json:"created_by"`

	// CreatedAt is when this policy was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this policy was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// IsActive indicates if this policy is active
	IsActive bool `json:"is_active"`

	// RequireReconfirmationAfter forces reconfirmation after N executions
	RequireReconfirmationAfter int `json:"require_reconfirmation_after,omitempty"`

	// LastReconfirmedAt is when the policy was last reconfirmed
	LastReconfirmedAt *time.Time `json:"last_reconfirmed_at,omitempty"`
}

// TrustScope defines the boundaries of a trusted workflow policy
type TrustScope struct {
	// TemplateIDs restricts trust to specific template IDs (empty = all)
	TemplateIDs []string `json:"template_ids,omitempty"`

	// WorkflowIDs restricts trust to specific workflow IDs (empty = all)
	WorkflowIDs []string `json:"workflow_ids,omitempty"`

	// Initiators restricts trust to specific initiator user IDs (empty = all)
	Initiators []string `json:"initiators,omitempty"`

	// Subjects restricts trust to specific subjects (empty = all)
	// Subject is typically the target of the workflow (e.g., a specific website)
	Subjects []string `json:"subjects,omitempty"`

	// PIIClassPatterns restricts trust to PII fields matching patterns
	// Patterns use prefix matching: "user.*" matches "user.email", "user.phone"
	PIIClassPatterns []string `json:"pii_class_patterns,omitempty"`

	// MaxAmount restricts trust to operations below a certain value (optional)
	// Useful for payment-related workflows
	MaxAmount *float64 `json:"max_amount,omitempty"`

	// AllowedDomains restricts trust to specific target domains
	AllowedDomains []string `json:"allowed_domains,omitempty"`

	// TimeRestrictions restricts trust to specific time windows
	TimeRestrictions *TimeRestrictions `json:"time_restrictions,omitempty"`
}

// TimeRestrictions defines when a trust policy is active
type TimeRestrictions struct {
	// AllowedHours defines hours of day when trust is active (0-23)
	AllowedHours []int `json:"allowed_hours,omitempty"`

	// AllowedDays defines days of week when trust is active (0=Sunday, 6=Saturday)
	AllowedDays []int `json:"allowed_days,omitempty"`

	// NotBefore is the earliest time trust is active
	NotBefore *time.Time `json:"not_before,omitempty"`

	// NotAfter is the latest time trust is active
	NotAfter *time.Time `json:"not_after,omitempty"`

	// Timezone for time restrictions (default: UTC)
	Timezone string `json:"timezone,omitempty"`
}

//=============================================================================
// Trust Decision Types
//=============================================================================

// TrustDecision represents the outcome of a trust evaluation
type TrustDecision string

const (
	TrustDecisionAllow           TrustDecision = "allow"
	TrustDecisionDeny            TrustDecision = "deny"
	TrustDecisionRequireApproval TrustDecision = "require_approval"
)

// TrustEvaluationResult contains the detailed result of a trust evaluation
type TrustEvaluationResult struct {
	// Decision is the final trust decision
	Decision TrustDecision `json:"decision"`

	// PolicyID references the policy that was evaluated (if any matched)
	PolicyID string `json:"policy_id,omitempty"`

	// PolicyName is the human-readable policy name
	PolicyName string `json:"policy_name,omitempty"`

	// Reason explains why this decision was made
	Reason string `json:"reason"`

	// ReasonCode is a machine-readable reason code
	ReasonCode TrustReasonCode `json:"reason_code"`

	// AllowedFields lists PII fields that were allowed
	AllowedFields []string `json:"allowed_fields,omitempty"`

	// DeniedFields lists PII fields that were denied
	DeniedFields []string `json:"denied_fields,omitempty"`

	// RequireApprovalFields lists fields that need manual approval
	RequireApprovalFields []string `json:"require_approval_fields,omitempty"`

	// ScopeMatched indicates which scope criteria matched
	ScopeMatched ScopeMatchResult `json:"scope_matched"`

	// ExpiresAt indicates when the trust decision expires
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// RemainingExecutions indicates how many executions remain (if limited)
	RemainingExecutions int `json:"remaining_executions,omitempty"`

	// EvaluationTime is when this evaluation was performed
	EvaluationTime time.Time `json:"evaluation_time"`

	// AuditMetadata contains additional audit information
	AuditMetadata map[string]interface{} `json:"audit_metadata,omitempty"`
}

// TrustReasonCode is a machine-readable code for trust decisions
type TrustReasonCode string

const (
	ReasonCodeAllowed                TrustReasonCode = "allowed"
	ReasonCodeDenied                 TrustReasonCode = "denied"
	ReasonCodeExplicitlyDenied       TrustReasonCode = "explicitly_denied"
	ReasonCodeNoMatchingPolicy       TrustReasonCode = "no_matching_policy"
	ReasonCodePolicyExpired          TrustReasonCode = "policy_expired"
	ReasonCodePolicyRevoked          TrustReasonCode = "policy_revoked"
	ReasonCodePolicyInactive         TrustReasonCode = "policy_inactive"
	ReasonCodeExecutionLimitReached  TrustReasonCode = "execution_limit_reached"
	ReasonCodeReconfirmationRequired TrustReasonCode = "reconfirmation_required"
	ReasonCodeScopeMismatch          TrustReasonCode = "scope_mismatch"
	ReasonCodeTimeRestriction        TrustReasonCode = "time_restriction"
	ReasonCodeFieldNotAllowed        TrustReasonCode = "field_not_allowed"
)

// ScopeMatchResult details which scope criteria matched
type ScopeMatchResult struct {
	TemplateMatched   bool   `json:"template_matched,omitempty"`
	WorkflowMatched   bool   `json:"workflow_matched,omitempty"`
	InitiatorMatched  bool   `json:"initiator_matched,omitempty"`
	SubjectMatched    bool   `json:"subject_matched,omitempty"`
	DomainMatched     bool   `json:"domain_matched,omitempty"`
	TimeMatched       bool   `json:"time_matched,omitempty"`
	MatchedPolicyName string `json:"matched_policy_name,omitempty"`
}

//=============================================================================
// Trust Evaluation Request
//=============================================================================

// TrustEvaluationRequest is the input for trust evaluation
type TrustEvaluationRequest struct {
	// TemplateID is the template being executed
	TemplateID string `json:"template_id,omitempty"`

	// WorkflowID is the workflow instance being executed
	WorkflowID string `json:"workflow_id,omitempty"`

	// Initiator is the user who initiated the workflow
	Initiator string `json:"initiator"`

	// Subject is the target of the workflow (e.g., website URL)
	Subject string `json:"subject,omitempty"`

	// PIIFields are the PII fields being requested
	PIIFields []string `json:"pii_fields"`

	// TargetDomain is the domain being targeted
	TargetDomain string `json:"target_domain,omitempty"`

	// Amount is the transaction amount (if applicable)
	Amount *float64 `json:"amount,omitempty"`

	// Context contains additional evaluation context
	Context map[string]interface{} `json:"context,omitempty"`
}

//=============================================================================
// Trusted Workflow Errors
//=============================================================================

// TrustedWorkflowError represents an error in trust evaluation
type TrustedWorkflowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *TrustedWorkflowError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrTrustPolicyNotFound       = &TrustedWorkflowError{Code: "POLICY_NOT_FOUND", Message: "trust policy not found"}
	ErrTrustPolicyExpired        = &TrustedWorkflowError{Code: "POLICY_EXPIRED", Message: "trust policy has expired"}
	ErrTrustPolicyRevoked        = &TrustedWorkflowError{Code: "POLICY_REVOKED", Message: "trust policy has been revoked"}
	ErrTrustPolicyInactive       = &TrustedWorkflowError{Code: "POLICY_INACTIVE", Message: "trust policy is not active"}
	ErrTrustExecutionLimit       = &TrustedWorkflowError{Code: "EXECUTION_LIMIT", Message: "execution limit reached"}
	ErrTrustScopeMismatch        = &TrustedWorkflowError{Code: "SCOPE_MISMATCH", Message: "request does not match trust scope"}
	ErrTrustReconfirmationNeeded = &TrustedWorkflowError{Code: "RECONFIRMATION_NEEDED", Message: "policy requires reconfirmation"}
)

//=============================================================================
// Trusted Workflow Engine
//=============================================================================

// TrustedWorkflowEngine manages trusted workflow policies and evaluations
type TrustedWorkflowEngine struct {
	mu       sync.RWMutex
	store    Store
	log      *logger.Logger
	policies map[string]*TrustedWorkflowPolicy
}

// TrustedWorkflowConfig holds configuration for the engine
type TrustedWorkflowConfig struct {
	Store  Store
	Logger *logger.Logger
}

// NewTrustedWorkflowEngine creates a new trusted workflow engine
func NewTrustedWorkflowEngine(cfg TrustedWorkflowConfig) (*TrustedWorkflowEngine, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("trusted_workflows")
	}

	return &TrustedWorkflowEngine{
		store:    cfg.Store,
		log:      log,
		policies: make(map[string]*TrustedWorkflowPolicy),
	}, nil
}

//=============================================================================
// Trust Evaluation
//=============================================================================

// Evaluate determines if a workflow execution is trusted
func (e *TrustedWorkflowEngine) Evaluate(ctx context.Context, req *TrustEvaluationRequest) (*TrustEvaluationResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	evalTime := time.Now()

	result := &TrustEvaluationResult{
		Decision:              TrustDecisionRequireApproval,
		ReasonCode:            ReasonCodeNoMatchingPolicy,
		Reason:                "No matching trusted policy found",
		EvaluationTime:        evalTime,
		AllowedFields:         make([]string, 0),
		DeniedFields:          make([]string, 0),
		RequireApprovalFields: make([]string, 0),
		AuditMetadata:         make(map[string]interface{}),
	}

	if len(req.PIIFields) == 0 {
		result.Decision = TrustDecisionAllow
		result.ReasonCode = ReasonCodeAllowed
		result.Reason = "No PII fields requested"
		return result, nil
	}

	var matchingPolicies []*TrustedWorkflowPolicy

	for _, policy := range e.policies {
		if !e.isPolicyActive(policy, evalTime) {
			continue
		}

		if e.matchesScope(policy, req) {
			matchingPolicies = append(matchingPolicies, policy)
		}
	}

	if len(matchingPolicies) == 0 {
		result.RequireApprovalFields = req.PIIFields
		return result, nil
	}

	return e.evaluatePolicies(matchingPolicies, req, evalTime)
}

// EvaluateForBlindFill evaluates trust for BlindFill execution
func (e *TrustedWorkflowEngine) EvaluateForBlindFill(
	ctx context.Context,
	templateID string,
	initiator string,
	piiFields []string,
	targetURL string,
) (*TrustEvaluationResult, error) {
	domain := extractDomain(targetURL)

	req := &TrustEvaluationRequest{
		TemplateID:   templateID,
		Initiator:    initiator,
		PIIFields:    piiFields,
		TargetDomain: domain,
		Subject:      targetURL,
	}

	return e.Evaluate(ctx, req)
}

func (e *TrustedWorkflowEngine) evaluatePolicies(
	policies []*TrustedWorkflowPolicy,
	req *TrustEvaluationRequest,
	evalTime time.Time,
) (*TrustEvaluationResult, error) {
	result := &TrustEvaluationResult{
		Decision:              TrustDecisionAllow,
		EvaluationTime:        evalTime,
		AllowedFields:         make([]string, 0),
		DeniedFields:          make([]string, 0),
		RequireApprovalFields: make([]string, 0),
		AuditMetadata:         make(map[string]interface{}),
	}

	var bestPolicy *TrustedWorkflowPolicy
	var bestScore int

	for _, policy := range policies {
		score := e.scorePolicyMatch(policy, req)
		if score > bestScore {
			bestScore = score
			bestPolicy = policy
		}
	}

	if bestPolicy == nil {
		result.Decision = TrustDecisionRequireApproval
		result.ReasonCode = ReasonCodeNoMatchingPolicy
		result.Reason = "No sufficiently matching policy found"
		result.RequireApprovalFields = req.PIIFields
		return result, nil
	}

	result.PolicyID = bestPolicy.ID
	result.PolicyName = bestPolicy.Name
	result.ScopeMatched = e.buildScopeMatchResult(bestPolicy, req)

	if bestPolicy.ExpiresAt != nil {
		result.ExpiresAt = bestPolicy.ExpiresAt
	}

	if bestPolicy.MaxExecutions > 0 {
		result.RemainingExecutions = bestPolicy.MaxExecutions - bestPolicy.ExecutionCount
	}

	for _, field := range req.PIIFields {
		fieldDecision := e.evaluateField(bestPolicy, field)

		switch fieldDecision {
		case TrustDecisionAllow:
			result.AllowedFields = append(result.AllowedFields, field)
		case TrustDecisionDeny:
			result.DeniedFields = append(result.DeniedFields, field)
		case TrustDecisionRequireApproval:
			result.RequireApprovalFields = append(result.RequireApprovalFields, field)
		}
	}

	if len(result.DeniedFields) > 0 {
		result.Decision = TrustDecisionDeny
		result.ReasonCode = ReasonCodeExplicitlyDenied
		result.Reason = fmt.Sprintf("Fields explicitly denied: %s", strings.Join(result.DeniedFields, ", "))
	} else if len(result.RequireApprovalFields) > 0 {
		result.Decision = TrustDecisionRequireApproval
		result.ReasonCode = ReasonCodeFieldNotAllowed
		result.Reason = fmt.Sprintf("Fields require approval: %s", strings.Join(result.RequireApprovalFields, ", "))
	} else {
		result.Decision = TrustDecisionAllow
		result.ReasonCode = ReasonCodeAllowed
		result.Reason = fmt.Sprintf("All fields allowed by policy: %s", bestPolicy.Name)
	}

	result.AuditMetadata["policy_id"] = bestPolicy.ID
	result.AuditMetadata["policy_name"] = bestPolicy.Name
	result.AuditMetadata["allowed_count"] = len(result.AllowedFields)
	result.AuditMetadata["denied_count"] = len(result.DeniedFields)
	result.AuditMetadata["require_approval_count"] = len(result.RequireApprovalFields)

	e.log.Info("trust_evaluation_completed",
		"decision", result.Decision,
		"policy_id", result.PolicyID,
		"allowed_fields", len(result.AllowedFields),
		"denied_fields", len(result.DeniedFields),
		"initiator", req.Initiator)

	return result, nil
}

func (e *TrustedWorkflowEngine) evaluateField(policy *TrustedWorkflowPolicy, field string) TrustDecision {
	for _, denied := range policy.DeniedPIIRefs {
		if denied == field {
			return TrustDecisionDeny
		}
		if strings.HasSuffix(denied, "*") {
			prefix := strings.TrimSuffix(denied, "*")
			if strings.HasPrefix(field, prefix) {
				return TrustDecisionDeny
			}
		}
	}

	for _, allowed := range policy.AllowedPIIRefs {
		if allowed == field {
			return TrustDecisionAllow
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(field, prefix) {
				return TrustDecisionAllow
			}
		}
	}

	for _, class := range policy.AllowedPIIClasses {
		if matchesPIIClass(field, class) {
			return TrustDecisionAllow
		}
	}

	for _, pattern := range policy.Scope.PIIClassPatterns {
		if matchesPIIClass(field, pattern) {
			return TrustDecisionAllow
		}
	}

	return TrustDecisionRequireApproval
}

//=============================================================================
// Policy Matching Helpers
//=============================================================================

func (e *TrustedWorkflowEngine) isPolicyActive(policy *TrustedWorkflowPolicy, now time.Time) bool {
	if !policy.IsActive {
		return false
	}

	if policy.RevokedAt != nil {
		return false
	}

	if policy.ExpiresAt != nil && now.After(*policy.ExpiresAt) {
		return false
	}

	if policy.MaxExecutions > 0 && policy.ExecutionCount >= policy.MaxExecutions {
		return false
	}

	if policy.RequireReconfirmationAfter > 0 && policy.LastReconfirmedAt != nil {
		executionsSinceReconfirm := policy.ExecutionCount
		if executionsSinceReconfirm >= policy.RequireReconfirmationAfter {
			return false
		}
	}

	return true
}

func (e *TrustedWorkflowEngine) matchesScope(policy *TrustedWorkflowPolicy, req *TrustEvaluationRequest) bool {
	scope := policy.Scope

	if len(scope.TemplateIDs) > 0 && req.TemplateID != "" {
		if !containsString(scope.TemplateIDs, req.TemplateID) {
			return false
		}
	}

	if len(scope.WorkflowIDs) > 0 && req.WorkflowID != "" {
		if !containsString(scope.WorkflowIDs, req.WorkflowID) {
			return false
		}
	}

	if len(scope.Initiators) > 0 {
		if !containsString(scope.Initiators, req.Initiator) {
			return false
		}
	}

	if len(scope.Subjects) > 0 && req.Subject != "" {
		if !containsString(scope.Subjects, req.Subject) {
			matched := false
			for _, s := range scope.Subjects {
				if strings.HasSuffix(s, "*") {
					prefix := strings.TrimSuffix(s, "*")
					if strings.HasPrefix(req.Subject, prefix) {
						matched = true
						break
					}
				}
			}
			if !matched {
				return false
			}
		}
	}

	if len(scope.AllowedDomains) > 0 && req.TargetDomain != "" {
		if !containsString(scope.AllowedDomains, req.TargetDomain) {
			matched := false
			for _, d := range scope.AllowedDomains {
				if strings.HasPrefix(d, "*.") {
					suffix := strings.TrimPrefix(d, "*")
					if strings.HasSuffix(req.TargetDomain, suffix) {
						matched = true
						break
					}
				}
			}
			if !matched {
				return false
			}
		}
	}

	if scope.MaxAmount != nil && req.Amount != nil {
		if *req.Amount > *scope.MaxAmount {
			return false
		}
	}

	if scope.TimeRestrictions != nil {
		if !e.matchesTimeRestrictions(scope.TimeRestrictions) {
			return false
		}
	}

	return true
}

func (e *TrustedWorkflowEngine) matchesTimeRestrictions(tr *TimeRestrictions) bool {
	now := time.Now()

	if tr.Timezone != "" {
		loc, err := time.LoadLocation(tr.Timezone)
		if err == nil {
			now = now.In(loc)
		}
	}

	if len(tr.AllowedHours) > 0 {
		hour := now.Hour()
		if !containsInt(tr.AllowedHours, hour) {
			return false
		}
	}

	if len(tr.AllowedDays) > 0 {
		day := int(now.Weekday())
		if !containsInt(tr.AllowedDays, day) {
			return false
		}
	}

	if tr.NotBefore != nil && now.Before(*tr.NotBefore) {
		return false
	}

	if tr.NotAfter != nil && now.After(*tr.NotAfter) {
		return false
	}

	return true
}

func (e *TrustedWorkflowEngine) scorePolicyMatch(policy *TrustedWorkflowPolicy, req *TrustEvaluationRequest) int {
	score := 0

	if len(policy.AllowedPIIRefs) > 0 {
		for _, field := range req.PIIFields {
			for _, allowed := range policy.AllowedPIIRefs {
				if allowed == field {
					score += 10
				}
			}
		}
	}

	if len(policy.Scope.TemplateIDs) > 0 && containsString(policy.Scope.TemplateIDs, req.TemplateID) {
		score += 5
	}

	if len(policy.Scope.Initiators) > 0 && containsString(policy.Scope.Initiators, req.Initiator) {
		score += 5
	}

	if len(policy.Scope.AllowedDomains) > 0 && containsString(policy.Scope.AllowedDomains, req.TargetDomain) {
		score += 3
	}

	return score
}

func (e *TrustedWorkflowEngine) buildScopeMatchResult(policy *TrustedWorkflowPolicy, req *TrustEvaluationRequest) ScopeMatchResult {
	return ScopeMatchResult{
		TemplateMatched:   len(policy.Scope.TemplateIDs) == 0 || containsString(policy.Scope.TemplateIDs, req.TemplateID),
		WorkflowMatched:   len(policy.Scope.WorkflowIDs) == 0 || containsString(policy.Scope.WorkflowIDs, req.WorkflowID),
		InitiatorMatched:  len(policy.Scope.Initiators) == 0 || containsString(policy.Scope.Initiators, req.Initiator),
		SubjectMatched:    len(policy.Scope.Subjects) == 0 || containsString(policy.Scope.Subjects, req.Subject),
		DomainMatched:     len(policy.Scope.AllowedDomains) == 0 || containsString(policy.Scope.AllowedDomains, req.TargetDomain),
		TimeMatched:       policy.Scope.TimeRestrictions == nil || e.matchesTimeRestrictions(policy.Scope.TimeRestrictions),
		MatchedPolicyName: policy.Name,
	}
}

//=============================================================================
// Policy CRUD Operations
//=============================================================================

// CreatePolicy creates a new trusted workflow policy
func (e *TrustedWorkflowEngine) CreatePolicy(ctx context.Context, policy *TrustedWorkflowPolicy) error {
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}

	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	policy.UpdatedAt = now
	policy.IsActive = true
	policy.ExecutionCount = 0

	e.mu.Lock()
	e.policies[policy.ID] = policy
	e.mu.Unlock()

	e.log.Info("trust_policy_created",
		"policy_id", policy.ID,
		"name", policy.Name,
		"created_by", policy.CreatedBy)

	return nil
}

// GetPolicy retrieves a trusted workflow policy by ID
func (e *TrustedWorkflowEngine) GetPolicy(ctx context.Context, id string) (*TrustedWorkflowPolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, exists := e.policies[id]
	if !exists {
		return nil, ErrPolicyNotFound
	}

	copy := *policy
	return &copy, nil
}

// ListPolicies returns all trusted workflow policies
func (e *TrustedWorkflowEngine) ListPolicies(ctx context.Context, activeOnly bool) ([]*TrustedWorkflowPolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TrustedWorkflowPolicy, 0)
	for _, policy := range e.policies {
		if activeOnly && !e.isPolicyActive(policy, time.Now()) {
			continue
		}
		copy := *policy
		result = append(result, &copy)
	}

	return result, nil
}

// UpdatePolicy updates an existing trusted workflow policy
func (e *TrustedWorkflowEngine) UpdatePolicy(ctx context.Context, policy *TrustedWorkflowPolicy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	existing, exists := e.policies[policy.ID]
	if !exists {
		return ErrPolicyNotFound
	}

	policy.UpdatedAt = time.Now()
	policy.CreatedAt = existing.CreatedAt
	policy.CreatedBy = existing.CreatedBy
	policy.ExecutionCount = existing.ExecutionCount

	e.policies[policy.ID] = policy

	e.log.Info("trust_policy_updated",
		"policy_id", policy.ID,
		"name", policy.Name)

	return nil
}

// DeletePolicy removes a trusted workflow policy
func (e *TrustedWorkflowEngine) DeletePolicy(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.policies[id]; !exists {
		return ErrPolicyNotFound
	}

	delete(e.policies, id)

	e.log.Info("trust_policy_deleted", "policy_id", id)

	return nil
}

//=============================================================================
// Expiration and Revocation
//=============================================================================

// RevokePolicy revokes a trusted workflow policy
func (e *TrustedWorkflowEngine) RevokePolicy(ctx context.Context, id string, revokedBy string, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policy, exists := e.policies[id]
	if !exists {
		return ErrPolicyNotFound
	}

	now := time.Now()
	policy.RevokedAt = &now
	policy.RevokedBy = revokedBy
	policy.RevocationReason = reason
	policy.IsActive = false
	policy.UpdatedAt = now

	e.log.Info("trust_policy_revoked",
		"policy_id", id,
		"revoked_by", revokedBy,
		"reason", reason)

	return nil
}

// ReconfirmPolicy reconfirms a trusted workflow policy
func (e *TrustedWorkflowEngine) ReconfirmPolicy(ctx context.Context, id string, confirmedBy string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policy, exists := e.policies[id]
	if !exists {
		return ErrPolicyNotFound
	}

	now := time.Now()
	policy.LastReconfirmedAt = &now
	policy.UpdatedAt = now

	e.log.Info("trust_policy_reconfirmed",
		"policy_id", id,
		"confirmed_by", confirmedBy)

	return nil
}

// RecordExecution records that a policy was used for an execution
func (e *TrustedWorkflowEngine) RecordExecution(ctx context.Context, policyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policy, exists := e.policies[policyID]
	if !exists {
		return ErrPolicyNotFound
	}

	policy.ExecutionCount++
	policy.UpdatedAt = time.Now()

	e.log.Info("trust_policy_execution_recorded",
		"policy_id", policyID,
		"execution_count", policy.ExecutionCount)

	return nil
}

// CleanupExpiredPolicies removes or deactivates expired policies
func (e *TrustedWorkflowEngine) CleanupExpiredPolicies(ctx context.Context) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	count := 0

	for _, policy := range e.policies {
		if policy.ExpiresAt != nil && now.After(*policy.ExpiresAt) && policy.IsActive {
			policy.IsActive = false
			policy.UpdatedAt = now
			count++
		}
	}

	if count > 0 {
		e.log.Info("trust_policies_expired", "count", count)
	}

	return count, nil
}

// GetPolicyStats returns statistics about a policy
func (e *TrustedWorkflowEngine) GetPolicyStats(ctx context.Context, id string) (*PolicyStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, exists := e.policies[id]
	if !exists {
		return nil, ErrPolicyNotFound
	}

	stats := &PolicyStats{
		PolicyID:            policy.ID,
		PolicyName:          policy.Name,
		ExecutionCount:      policy.ExecutionCount,
		RemainingExecutions: -1,
		IsActive:            e.isPolicyActive(policy, time.Now()),
		CreatedAt:           policy.CreatedAt,
		LastReconfirmedAt:   policy.LastReconfirmedAt,
	}

	if policy.MaxExecutions > 0 {
		stats.RemainingExecutions = policy.MaxExecutions - policy.ExecutionCount
	}

	if policy.ExpiresAt != nil {
		stats.ExpiresAt = policy.ExpiresAt
		stats.IsExpired = time.Now().After(*policy.ExpiresAt)
	}

	if policy.RevokedAt != nil {
		stats.IsRevoked = true
		stats.RevokedAt = policy.RevokedAt
		stats.RevokedBy = policy.RevokedBy
	}

	return stats, nil
}

// PolicyStats contains statistics about a policy
type PolicyStats struct {
	PolicyID            string     `json:"policy_id"`
	PolicyName          string     `json:"policy_name"`
	ExecutionCount      int        `json:"execution_count"`
	RemainingExecutions int        `json:"remaining_executions"`
	IsActive            bool       `json:"is_active"`
	IsExpired           bool       `json:"is_expired"`
	IsRevoked           bool       `json:"is_revoked"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	RevokedBy           string     `json:"revoked_by,omitempty"`
	LastReconfirmedAt   *time.Time `json:"last_reconfirmed_at,omitempty"`
}

//=============================================================================
// Helper Functions
//=============================================================================

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func matchesPIIClass(field string, class string) bool {
	if class == "*" {
		return true
	}

	if strings.HasSuffix(class, "*") {
		prefix := strings.TrimSuffix(class, "*")
		return strings.HasPrefix(field, prefix)
	}

	parts := strings.Split(field, ".")
	classParts := strings.Split(class, ".")

	if len(classParts) > len(parts) {
		return false
	}

	for i, cp := range classParts {
		if cp != parts[i] && cp != "*" {
			return false
		}
	}

	return true
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}

	if idx := strings.Index(url, ":"); idx > 0 {
		url = url[:idx]
	}

	return url
}

//=============================================================================
// Integration with Approval Engine
//=============================================================================

// EvaluateWithApprovalEngine combines trust evaluation with approval engine
func (e *TrustedWorkflowEngine) EvaluateWithApprovalEngine(
	ctx context.Context,
	req *TrustEvaluationRequest,
	approvalEngine *ApprovalEngineImpl,
) (*TrustEvaluationResult, error) {
	trustResult, err := e.Evaluate(ctx, req)
	if err != nil {
		return nil, err
	}

	if trustResult.Decision == TrustDecisionAllow {
		return trustResult, nil
	}

	if trustResult.Decision == TrustDecisionDeny {
		return trustResult, nil
	}

	if approvalEngine == nil {
		return trustResult, nil
	}

	evalCtx := EvaluationContext{
		Initiator: req.Initiator,
		PIIFields: trustResult.RequireApprovalFields,
		Variables: req.Context,
		Timestamp: time.Now(),
	}

	approvalResult, err := approvalEngine.Evaluate(ctx, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("approval evaluation failed: %w", err)
	}

	if approvalResult.Approved {
		trustResult.Decision = TrustDecisionAllow
		trustResult.ReasonCode = ReasonCodeAllowed
		trustResult.Reason = "Approved through approval engine"
		trustResult.AllowedFields = append(trustResult.AllowedFields, trustResult.RequireApprovalFields...)
		trustResult.RequireApprovalFields = nil
		trustResult.AuditMetadata["approval_request_id"] = approvalResult.RequestID
	} else {
		trustResult.Decision = TrustDecisionRequireApproval
		trustResult.Reason = "Waiting for manual approval"
		trustResult.AuditMetadata["approval_required"] = true
		trustResult.AuditMetadata["approval_request_id"] = approvalResult.RequestID
	}

	return trustResult, nil
}

//=============================================================================
// JSON Serialization Helpers
//=============================================================================

// MarshalPolicy serializes a policy to JSON
func MarshalPolicy(policy *TrustedWorkflowPolicy) ([]byte, error) {
	return json.Marshal(policy)
}

// UnmarshalPolicy deserializes a policy from JSON
func UnmarshalPolicy(data []byte) (*TrustedWorkflowPolicy, error) {
	var policy TrustedWorkflowPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}
	return &policy, nil
}
