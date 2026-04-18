// Package secretary provides core types for Secretary features including templates, workflows, and approvals.
package secretary

import (
	"encoding/json"
	"time"
)

//=============================================================================
// Workflow Status
//=============================================================================

// WorkflowStatus represents the state of a workflow execution
type WorkflowStatus string

const (
	StatusPending   WorkflowStatus = "pending"
	StatusRunning   WorkflowStatus = "running"
	StatusBlocked   WorkflowStatus = "blocked"
	StatusCompleted WorkflowStatus = "completed"
	StatusFailed    WorkflowStatus = "failed"
	StatusCancelled WorkflowStatus = "cancelled"
)

//=============================================================================
// Template Types
//=============================================================================

// TaskTemplate represents a predefined task workflow that can be instantiated.
// Templates define the structure and steps for Secretary tasks.
type TaskTemplate struct {
	// ID is unique identifier for this template
	ID string `json:"id"`

	// Name is human-readable name (e.g., "Payment Approval")
	Name string `json:"name"`

	// Description explains what this template does
	Description string `json:"description,omitempty"`

	// Steps defines the execution flow (JSON array of workflow steps)
	Steps []WorkflowStep `json:"steps"`

	// Variables is JSON schema for input variables
	Variables json.RawMessage `json:"variables"`

	// PIIRefs lists PII field references that will be requested
	PIIRefs []string `json:"pii_refs"`

	// CreatedBy is Matrix user ID who created this template
	CreatedBy string `json:"created_by"`

	// CreatedAt is when this template was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this template was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// IsActive indicates if this template can be instantiated
	IsActive bool `json:"is_active"`
}

// WorkflowStep represents a single step in a workflow template
type WorkflowStep struct {
	// StepID is unique identifier for this step
	StepID string `json:"step_id"`

	// Order is execution order (0-indexed)
	Order int `json:"order"`

	// Type is step type (action, condition, parallel, parallel_split, parallel_merge)
	Type StepType `json:"type"`

	// Name is human-readable name
	Name string `json:"name"`

	// Description explains what this step does
	Description string `json:"description,omitempty"`

	// Config is step-specific configuration (JSON)
	Config json.RawMessage `json:"config,omitempty"`

	// Conditions are conditional rules for step execution
	Conditions json.RawMessage `json:"conditions,omitempty"`

	// NextStepID is the step to execute after this one (empty if last step)
	NextStepID string `json:"next_step_id,omitempty"`

	// Input contains data references from previous steps, resolved before execution.
	// Template variables like {{steps.step_1.data.order_id}} are resolved at runtime.
	// Omit for steps that don't need data from prior steps.
	Input map[string]any `json:"input,omitempty"`

	// AgentIDs are the agents that can execute this step
	AgentIDs []string `json:"agent_ids,omitempty"`

	// TeamID optionally assigns this step to a specific team
	TeamID string `json:"team_id,omitempty"`

	// AssignedMemberID optionally assigns this step to a specific team member.
	// If set, TeamID must also be set.
	AssignedMemberID string `json:"assigned_member_id,omitempty"`
}

// StepType defines the type of workflow step
type StepType string

const (
	StepAction        StepType = "action"         // Execute an action
	StepCondition     StepType = "condition"      // Evaluate a condition
	StepParallel      StepType = "parallel"       // Execute steps in parallel
	StepParallelSplit StepType = "parallel_split" // Split flow into parallel branches
	StepParallelMerge StepType = "parallel_merge" // Merge parallel branches
)

//=============================================================================
// Workflow Types
//=============================================================================

// Workflow represents an instance of a template being executed.
type Workflow struct {
	// ID is the unique identifier for this workflow
	ID string `json:"id"`

	// TemplateID references the template definition
	TemplateID string `json:"template_id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description explains what this workflow does
	Description string `json:"description,omitempty"`

	// Status is the current workflow state
	Status WorkflowStatus `json:"status"`

	// Variables is the resolved input variables
	Variables map[string]interface{} `json:"variables"`

	// CurrentStep is the ID of the step currently executing
	CurrentStep string `json:"current_step,omitempty"`

	// AgentIDs are the agents participating in this workflow
	AgentIDs []string `json:"agent_ids"`

	// CreatedBy is the Matrix user ID who started the workflow
	CreatedBy string `json:"created_by"`

	// RoomID is the Matrix room ID associated with this workflow
	RoomID string `json:"room_id"`

	// StartedAt is when the workflow was started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the workflow finished (nil if running)
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ErrorMessage describes any failure
	ErrorMessage string `json:"error_message,omitempty"`

	// StepsData holds accumulated step outputs keyed by step_id.
	// For parallel groups, the value is a map of branch_step_id → data.
	// Available to subsequent steps via {{steps.split_id.data.branch_1.key}} syntax.
	// Populated during execution, not persisted to store.
	StepsData map[string]any `json:"steps_data,omitempty"`
}

// MarshalJSON custom marshals Workflow for API responses
func (w *Workflow) MarshalJSON() ([]byte, error) {
	type Alias Workflow
	return json.Marshal(&struct {
		*Alias
		StartedAt   int64  `json:"started_at"`
		CompletedAt *int64 `json:"completed_at,omitempty"`
	}{
		Alias:       (*Alias)(w),
		StartedAt:   w.StartedAt.UnixMilli(),
		CompletedAt: timeToMillis(w.CompletedAt),
	})
}

//=============================================================================
// Scheduled Task Types
//=============================================================================

// ScheduledTask represents a task to be executed on a schedule
type ScheduledTask struct {
	// ID is the unique identifier for this scheduled task
	ID string `json:"id"`

	// TemplateID references the task template to execute
	TemplateID string `json:"template_id"`

	// DefinitionID references the agent definition for task dispatch
	DefinitionID string `json:"definition_id"`

	// CronExpression defines the execution schedule (cron format)
	CronExpression string `json:"cron_expression"`

	// Timezone defines the timezone for the schedule (default: UTC)
	Timezone string `json:"timezone"`

	// NextRun is the next scheduled execution time
	NextRun *time.Time `json:"next_run"`

	// LastRun is the last time this task was executed
	LastRun *time.Time `json:"last_run"`

	// IsActive indicates if this scheduled task is active
	IsActive bool `json:"is_active"`

	// CreatedBy is the Matrix user ID who created this task
	CreatedBy string `json:"created_by"`

	// Trigger is the event trigger key for event-driven dispatch (e.g., "email:secretary@example.com")
	Trigger string `json:"trigger,omitempty"`

	// OneShot indicates this task should fire once then deactivate
	OneShot bool `json:"one_shot"`
}

//=============================================================================
// Approval Policy Types
//=============================================================================

// ApprovalPolicy defines rules for automatic and manual approval workflows.
type ApprovalPolicy struct {
	// ID is the unique identifier for this policy
	ID string `json:"id"`

	// Name is the human-readable name (e.g., "Credit Card Payment Policy")
	Name string `json:"name"`

	// Description explains the approval rules
	Description string `json:"description,omitempty"`

	// PIIFields is the list of PII field IDs that require approval
	PIIFields []string `json:"pii_fields"`

	// AutoApprove indicates if fields should be auto-approved
	AutoApprove bool `json:"auto_approve"`

	// DelegateTo is the user ID to delegate approval to (empty if direct approval)
	DelegateTo string `json:"delegate_to,omitempty"`

	// Conditions is the JSON schema for approval conditions
	Conditions json.RawMessage `json:"conditions,omitempty"`

	// CreatedBy is the Matrix user ID who created this policy
	CreatedBy string `json:"created_by"`

	// CreatedAt is when this policy was created
	CreatedAt time.Time `json:"created_at"`

	// IsActive indicates if this policy is active
	IsActive bool `json:"is_active"`
}

//=============================================================================
// Notification Types
//=============================================================================

// NotificationChannel defines how notifications are delivered.
type NotificationChannel struct {
	// ID is the unique identifier for this channel
	ID string `json:"id"`

	// UserID is the recipient user ID
	UserID string `json:"user_id"`

	// ChannelType is the delivery method (matrix, email, webhook)
	ChannelType ChannelType `json:"channel_type"`

	// Destination is the channel destination (room ID, email, webhook URL)
	Destination string `json:"destination"`

	// EventTypes is the list of event types that trigger notifications
	EventTypes []NotificationEventType `json:"event_types"`

	// IsActive indicates if this channel is active
	IsActive bool `json:"is_active"`

	// CreatedAt is when this channel was created
	CreatedAt time.Time `json:"created_at"`
}

// ChannelType defines the notification delivery method
type ChannelType string

const (
	ChannelMatrix  ChannelType = "matrix"  // Matrix room notifications
	ChannelEmail   ChannelType = "email"   // Email notifications
	ChannelWebhook ChannelType = "webhook" // Webhook callbacks
)

// NotificationEventType defines the type of notification event
type NotificationEventType string

const (
	EventWorkflowStart     NotificationEventType = "workflow_start"
	EventWorkflowComplete  NotificationEventType = "workflow_complete"
	EventWorkflowFailed    NotificationEventType = "workflow_failed"
	EventApprovalRequested NotificationEventType = "approval_requested"
	EventApprovalApproved  NotificationEventType = "approval_approved"
	EventApprovalDenied    NotificationEventType = "approval_denied"
	EventStepProgress      NotificationEventType = "step_progress"
)

//=============================================================================
// Approval Result Types
//=============================================================================

// ApprovalResult contains the decision outcome of an approval check
type ApprovalResult struct {
	// RequestID is the unique identifier for this approval request
	RequestID string `json:"request_id"`

	// PolicyID is the policy that was evaluated
	PolicyID string `json:"policy_id"`

	// Required indicates if approval is required
	Required bool `json:"required"`

	// Approved indicates if approval was granted
	Approved bool `json:"approved"`

	// ApprovedFields are the fields that were approved
	ApprovedFields []string `json:"approved_fields"`

	// DeniedFields are the fields that were denied
	DeniedFields []string `json:"denied_fields"`

	// NeedsApproval indicates if manual approval is needed
	NeedsApproval bool `json:"needs_approval"`

	// DelegateTo is the user ID to delegate approval to (empty if direct)
	DelegateTo string `json:"delegate_to,omitempty"`

	// Context contains additional evaluation context
	Context map[string]interface{} `json:"context,omitempty"`
}

//=============================================================================
// Interfaces
//=============================================================================

// TemplateExecutor executes workflow templates with input variables.
type TemplateExecutor interface {
	// ExecuteTemplate creates and starts a workflow from a template
	ExecuteTemplate(templateID string, variables map[string]interface{}, createdBy string) (*Workflow, error)

	// GetTemplate retrieves a template by ID
	GetTemplate(templateID string) (*TaskTemplate, error)

	// ListTemplates returns all templates (optionally filtered)
	ListTemplates(activeOnly bool) ([]*TaskTemplate, error)

	// ValidateTemplate checks if a template is valid
	ValidateTemplate(template *TaskTemplate) error

	// UpdateTemplate updates an existing template
	UpdateTemplate(template *TaskTemplate) error
}

// WorkflowOrchestrator manages the execution of workflows and their steps.
type WorkflowOrchestrator interface {
	// StartWorkflow initializes a workflow execution
	StartWorkflow(workflowID string) error

	// GetWorkflow retrieves a workflow by ID
	GetWorkflow(workflowID string) (*Workflow, error)

	// ListWorkflows returns all workflows (optionally filtered)
	ListWorkflows(statusFilter WorkflowStatus, createdBy string) ([]*Workflow, error)

	// AdvanceWorkflow moves to the next step in the workflow
	AdvanceWorkflow(workflowID string, stepID string) error

	// CancelWorkflow terminates a running workflow
	CancelWorkflow(workflowID string, reason string) error

	// GetStepConfig retrieves configuration for a specific step
	GetStepConfig(workflowID string, stepID string) (json.RawMessage, error)
}

// ApprovalEngine evaluates approval policies and decides whether to approve or deny access.
type ApprovalEngine interface {
	// CheckApproval determines if approval is required and who should approve
	CheckApproval(policyID string, piiFields []string, context map[string]interface{}) (*ApprovalResult, error)

	// ApproveField grants approval for specific PII fields
	ApproveField(requestID string, fieldID string, approvedBy string) error

	// DenyField denies approval for specific PII fields
	DenyField(requestID string, fieldID string, deniedBy string) error

	// GetPolicy retrieves an approval policy by ID
	GetPolicy(policyID string) (*ApprovalPolicy, error)

	// ListPolicies returns all policies (optionally filtered)
	ListPolicies(activeOnly bool) ([]*ApprovalPolicy, error)

	// CreatePolicy creates a new approval policy
	CreatePolicy(policy *ApprovalPolicy) error

	// UpdatePolicy updates an existing policy
	UpdatePolicy(policy *ApprovalPolicy) error
}

// NotificationDispatcher sends notifications to registered channels.
type NotificationDispatcher interface {
	// Send sends a notification to a channel
	Send(channelID string, eventType NotificationEventType, message interface{}) error

	// NotifyWorkflowStart notifies of workflow start
	NotifyWorkflowStart(workflow *Workflow) error

	// NotifyWorkflowComplete notifies of workflow completion
	NotifyWorkflowComplete(workflow *Workflow, success bool, message string) error

	// NotifyApprovalRequest notifies of an approval request
	NotifyApprovalRequest(request *ApprovalResult) error

	// NotifyStepProgress notifies of step progress
	NotifyStepProgress(workflowID string, stepID string, progress string) error

	// GetChannel retrieves a notification channel by ID
	GetChannel(channelID string) (*NotificationChannel, error)

	// ListChannels returns all notification channels
	ListChannels() ([]*NotificationChannel, error)

	// CreateChannel creates a new notification channel
	CreateChannel(channel *NotificationChannel) error

	// UpdateChannel updates an existing channel
	UpdateChannel(channel *NotificationChannel) error

	// DeleteChannel removes a notification channel
	DeleteChannel(channelID string) error
}

//=============================================================================
// Step Failover Types
//=============================================================================

// FailoverPolicy controls how step failover behaves when an agent fails.
type FailoverPolicy string

const (
	// FailoverRetry is the default policy: on primary agent failure, attempt
	// fallback agents in order (AgentIDs[1], AgentIDs[2], ...) until one
	// succeeds or all are exhausted.
	FailoverRetry FailoverPolicy = "retry_on_failure"

	// FailoverImmediateFail skips failover entirely — the first agent failure
	// is returned immediately without trying fallback agents.
	FailoverImmediateFail FailoverPolicy = "immediate_fail"
)

//=============================================================================
// Helper Functions
//=============================================================================

//=============================================================================
// Rolodex Contact Types
//=============================================================================

// Contact represents a contact in the Rolodex
type Contact struct {
	// ID is unique identifier for this contact
	ID string `json:"id"`

	// Name is the contact's full name (searchable)
	Name string `json:"name"`

	// Company is the contact's organization (searchable)
	Company string `json:"company,omitempty"`

	// Relationship describes the relationship (e.g., "client", "vendor", "colleague") (searchable)
	Relationship string `json:"relationship,omitempty"`

	// EncryptedData contains sensitive contact details (phone, email, address, notes) as encrypted BLOB
	// This field is stored encrypted in the database
	EncryptedData []byte `json:"-"` // Never sent over JSON

	// EncryptedNonce is the nonce used for encrypting EncryptedData
	// This field is stored alongside encrypted data
	EncryptedNonce []byte `json:"-"` // Never sent over JSON

	// Phone is the contact's phone number (encrypted)
	Phone string `json:"phone,omitempty"`

	// Email is the contact's email address (encrypted)
	Email string `json:"email,omitempty"`

	// Address is the contact's physical address (encrypted)
	Address string `json:"address,omitempty"`

	// Notes contains additional information about the contact (encrypted)
	Notes string `json:"notes,omitempty"`

	// CreatedBy is Matrix user ID who created this contact
	CreatedBy string `json:"created_by"`

	// CreatedAt is when this contact was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this contact was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// ContactFilter filters contact listings
type ContactFilter struct {
	// Name filters by name (partial match)
	Name string

	// Company filters by company (partial match)
	Company string

	// Relationship filters by relationship (exact match)
	Relationship string
}

// timeToMillis converts *time.Time to *int64 (Unix milliseconds) or nil
func timeToMillis(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ms := t.UnixMilli()
	return &ms
}
