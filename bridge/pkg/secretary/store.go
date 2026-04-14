package secretary

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// Store Interface
//=============================================================================

// Store defines the persistence interface for Secretary
type Store interface {
	// Task Templates
	CreateTemplate(ctx context.Context, template *TaskTemplate) error
	GetTemplate(ctx context.Context, id string) (*TaskTemplate, error)
	ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error)
	UpdateTemplate(ctx context.Context, template *TaskTemplate) error
	DeleteTemplate(ctx context.Context, id string) error

	// Workflows
	CreateWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) error
	DeleteWorkflow(ctx context.Context, id string) error

	// Approval Policies
	CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error
	GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error)
	ListPolicies(ctx context.Context) ([]ApprovalPolicy, error)
	UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error
	DeletePolicy(ctx context.Context, id string) error

	// Scheduled Tasks
	CreateScheduledTask(ctx context.Context, task *ScheduledTask) error
	GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error)
	ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error)
	UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error
	DeleteScheduledTask(ctx context.Context, id string) error
	ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error)
	ListDueTasks(ctx context.Context) ([]ScheduledTask, error)
	MarkDispatched(ctx context.Context, taskID string, nextRun time.Time) error

	// Notification Channels
	CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error
	GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error)
	ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error
	DeleteNotificationChannel(ctx context.Context, id string) error

	// Contacts (Rolodex)
	CreateContact(ctx context.Context, contact *Contact) error
	GetContact(ctx context.Context, id string) (*Contact, error)
	ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error)
	UpdateContact(ctx context.Context, contact *Contact) error
	DeleteContact(ctx context.Context, id string) error

	// Close closes the database connection
	Close() error
}

//=============================================================================
// SQLite Store Implementation
//=============================================================================

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	mu          sync.RWMutex
	db          *sql.DB
	log         *logger.Logger
	path        string
	auditLogger *AuditLogger
}

// StoreConfig holds configuration for store
type StoreConfig struct {
	Path   string
	Logger *logger.Logger
}

// TemplateFilter filters task template listings
type TemplateFilter struct {
	ActiveOnly bool
}

// WorkflowFilter filters workflow listings
type WorkflowFilter struct {
	Status     *WorkflowStatus
	TemplateID string
	CreatedBy  string
}

// NewStore creates a new SQLite store
func NewStore(cfg StoreConfig) (*SQLiteStore, error) {
	if cfg.Path == "" {
		cfg.Path = ":memory:"
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("secretary")
	}

	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	store := &SQLiteStore{
		db:          db,
		log:         log,
		path:        cfg.Path,
		auditLogger: NewAuditLogger(slog.Default()),
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	schema := "-- Task Templates" + "\n" +
		"CREATE TABLE IF NOT EXISTS task_templates (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    name TEXT NOT NULL UNIQUE," + "\n" +
		"    description TEXT," + "\n" +
		"    steps TEXT NOT NULL," + "\n" +
		"    variables TEXT," + "\n" +
		"    pii_refs TEXT," + "\n" +
		"    created_by TEXT NOT NULL," + "\n" +
		"    created_at INTEGER NOT NULL," + "\n" +
		"    updated_at INTEGER NOT NULL," + "\n" +
		"    is_active INTEGER DEFAULT 1" + "\n" +
		");" + "\n" +

		"-- Workflows" + "\n" +
		"CREATE TABLE IF NOT EXISTS workflows (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    template_id TEXT," + "\n" +
		"    name TEXT NOT NULL," + "\n" +
		"    description TEXT," + "\n" +
		"    status TEXT DEFAULT 'pending'," + "\n" +
		"    variables TEXT," + "\n" +
		"    current_step INTEGER DEFAULT 0," + "\n" +
		"    agent_ids TEXT," + "\n" +
		"    started_at INTEGER," + "\n" +
		"    completed_at INTEGER," + "\n" +
		"    error_message TEXT," + "\n" +
		"    created_by TEXT NOT NULL," + "\n" +
		"    room_id TEXT DEFAULT ''," + "\n" +
		"    FOREIGN KEY (template_id) REFERENCES task_templates(id) ON DELETE CASCADE" + "\n" +
		");" + "\n" +

		"-- Approval Policies" + "\n" +
		"CREATE TABLE IF NOT EXISTS approval_policies (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    name TEXT NOT NULL," + "\n" +
		"    description TEXT," + "\n" +
		"    pii_fields TEXT NOT NULL," + "\n" +
		"    auto_approve INTEGER DEFAULT 0," + "\n" +
		"    delegate_to TEXT," + "\n" +
		"    conditions TEXT," + "\n" +
		"    created_by TEXT NOT NULL," + "\n" +
		"    created_at INTEGER NOT NULL," + "\n" +
		"    is_active INTEGER DEFAULT 1" + "\n" +
		");" + "\n" +

		"-- Notification Channels" + "\n" +
		"CREATE TABLE IF NOT EXISTS notification_channels (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    user_id TEXT NOT NULL," + "\n" +
		"    channel_type TEXT DEFAULT 'matrix'," + "\n" +
		"    destination TEXT NOT NULL," + "\n" +
		"    event_types TEXT NOT NULL," + "\n" +
		"    is_active INTEGER DEFAULT 1," + "\n" +
		"    created_at INTEGER NOT NULL" + "\n" +
		");" + "\n" +

		"-- Scheduled Tasks" + "\n" +
		"CREATE TABLE IF NOT EXISTS scheduled_tasks (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    template_id TEXT DEFAULT ''," + "\n" +
		"    definition_id TEXT DEFAULT ''," + "\n" +
		"    cron_expression TEXT NOT NULL," + "\n" +
		"    timezone TEXT DEFAULT 'UTC'," + "\n" +
		"    next_run INTEGER," + "\n" +
		"    last_run INTEGER," + "\n" +
		"    is_active INTEGER DEFAULT 1," + "\n" +
		"    created_by TEXT NOT NULL" + "\n" +
		");" + "\n" +

		"-- Contacts (Rolodex)" + "\n" +
		"CREATE TABLE IF NOT EXISTS contacts (" + "\n" +
		"    id TEXT PRIMARY KEY," + "\n" +
		"    name TEXT NOT NULL," + "\n" +
		"    company TEXT," + "\n" +
		"    relationship TEXT," + "\n" +
		"    data_encrypted BLOB NOT NULL," + "\n" +
		"    data_nonce BLOB NOT NULL," + "\n" +
		"    created_by TEXT NOT NULL," + "\n" +
		"    created_at INTEGER NOT NULL," + "\n" +
		"    updated_at INTEGER NOT NULL" + "\n" +
		");" + "\n" +

		"CREATE INDEX IF NOT EXISTS idx_templates_active ON task_templates(is_active);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_workflows_template ON workflows(template_id);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_policies_active ON approval_policies(is_active);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_notification_user ON notification_channels(user_id);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_scheduled_next_run ON scheduled_tasks(next_run);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_contacts_company ON contacts(company);" + "\n" +
		"CREATE INDEX IF NOT EXISTS idx_contacts_relationship ON contacts(relationship);"

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Migration: recreate scheduled_tasks to remove FK constraint on template_id.
	// The FK prevented template_id='' (used by templateless scheduled tasks).
	// SQLite doesn't support DROP CONSTRAINT, so we recreate the table.
	_, migErr := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduled_tasks_v2 (
			id TEXT PRIMARY KEY,
			template_id TEXT DEFAULT '',
			definition_id TEXT DEFAULT '',
			cron_expression TEXT NOT NULL,
			timezone TEXT DEFAULT 'UTC',
			next_run INTEGER,
			last_run INTEGER,
			is_active INTEGER DEFAULT 1,
			created_by TEXT NOT NULL
		)
	`)
	if migErr != nil {
		return fmt.Errorf("failed to create scheduled_tasks_v2: %w", migErr)
	}

	_, migErr = s.db.Exec(`
		INSERT OR IGNORE INTO scheduled_tasks_v2 (id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by)
		SELECT id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by FROM scheduled_tasks
	`)
	if migErr != nil {
		return fmt.Errorf("failed to migrate scheduled_tasks data: %w", migErr)
	}

	_, migErr = s.db.Exec("DROP TABLE scheduled_tasks")
	if migErr != nil {
		return fmt.Errorf("failed to drop old scheduled_tasks: %w", migErr)
	}

	_, migErr = s.db.Exec("ALTER TABLE scheduled_tasks_v2 RENAME TO scheduled_tasks")
	if migErr != nil {
		return fmt.Errorf("failed to rename scheduled_tasks_v2: %w", migErr)
	}

	_, migErr = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_scheduled_next_run ON scheduled_tasks(next_run)")
	if migErr != nil {
		return fmt.Errorf("failed to recreate scheduled_tasks index: %w", migErr)
	}

	_, migErr = s.db.Exec("ALTER TABLE workflows ADD COLUMN room_id TEXT DEFAULT ''")
	if migErr != nil {
		if !strings.Contains(migErr.Error(), "duplicate column name") {
			return fmt.Errorf("failed to add room_id column to workflows: %w", migErr)
		}
	}

	return nil
}

//=============================================================================
// Task Template CRUD
//=============================================================================
// Task Template CRUD
//=============================================================================

// CreateTemplate stores a new task template
func (s *SQLiteStore) CreateTemplate(ctx context.Context, template *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stepsJSON, err := json.Marshal(template.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	isActive := 0
	if template.IsActive {
		isActive = 1
	}

	// Marshal PIIRefs to JSON string
	piiRefsJSON, err := json.Marshal(template.PIIRefs)
	if err != nil {
		return fmt.Errorf("failed to marshal PII refs: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO task_templates (id, name, description, steps, variables, pii_refs, created_by, created_at, updated_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, template.ID, template.Name, template.Description, string(stepsJSON), template.Variables, string(piiRefsJSON), template.CreatedBy, time.Now().UnixMilli(), time.Now().UnixMilli(), isActive)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	// Log to audit
	s.auditLogger.LogOperation(ctx, "template_created", map[string]interface{}{
		"id":         template.ID,
		"name":       template.Name,
		"created_by": template.CreatedBy,
	})

	return nil
}

// GetTemplate retrieves a task template by ID
func (s *SQLiteStore) GetTemplate(ctx context.Context, id string) (*TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	template := &TaskTemplate{}
	var stepsJSON, variablesJSON, piiRefsJSON string
	var isActive int
	var createdAtInt, updatedAtInt sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, name, description, steps, variables, pii_refs, created_by, created_at, updated_at, is_active
		FROM task_templates WHERE id = ?
	`, id).Scan(&template.ID, &template.Name, &template.Description, &stepsJSON, &variablesJSON, &piiRefsJSON, &template.CreatedBy, &createdAtInt, &updatedAtInt, &isActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if createdAtInt.Valid {
		template.CreatedAt = time.UnixMilli(createdAtInt.Int64)
	}
	if updatedAtInt.Valid {
		template.UpdatedAt = time.UnixMilli(updatedAtInt.Int64)
	}

	if err := json.Unmarshal([]byte(stepsJSON), &template.Steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
	}

	template.Variables = json.RawMessage(variablesJSON)
	if err := json.Unmarshal([]byte(piiRefsJSON), &template.PIIRefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PII refs: %w", err)
	}
	template.IsActive = isActive == 1

	return template, nil
}

// ListTemplates returns all task templates, optionally filtered
func (s *SQLiteStore) ListTemplates(ctx context.Context, filter TemplateFilter) ([]TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, description, steps, variables, pii_refs, created_by, created_at, updated_at, is_active FROM task_templates`
	args := []interface{}{}

	if filter.ActiveOnly {
		query += " WHERE is_active = 1"
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []TaskTemplate
	for rows.Next() {
		template := &TaskTemplate{}
		var stepsJSON, variablesJSON, piiRefsJSON string
		var isActive int
		var createdAtInt, updatedAtInt sql.NullInt64

		if err := rows.Scan(&template.ID, &template.Name, &template.Description, &stepsJSON, &variablesJSON, &piiRefsJSON, &template.CreatedBy, &createdAtInt, &updatedAtInt, &isActive); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if createdAtInt.Valid {
			template.CreatedAt = time.UnixMilli(createdAtInt.Int64)
		}
		if updatedAtInt.Valid {
			template.UpdatedAt = time.UnixMilli(updatedAtInt.Int64)
		}

		if err := json.Unmarshal([]byte(stepsJSON), &template.Steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
		}

		template.Variables = json.RawMessage(variablesJSON)
		if err := json.Unmarshal([]byte(piiRefsJSON), &template.PIIRefs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PII refs: %w", err)
		}
		template.IsActive = isActive == 1

		templates = append(templates, *template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing task template
func (s *SQLiteStore) UpdateTemplate(ctx context.Context, template *TaskTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stepsJSON, err := json.Marshal(template.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	variablesJSON := template.Variables // Variables is already json.RawMessage (string)
	var piiRefsJSON string
	if len(template.PIIRefs) > 0 {
		marshaled, err := json.Marshal(template.PIIRefs)
		if err != nil {
			return fmt.Errorf("failed to marshal pii refs: %w", err)
		}
		piiRefsJSON = string(marshaled)
	}

	isActive := 0
	if template.IsActive {
		isActive = 1
	}

	_, err = s.db.Exec(`
		UPDATE task_templates
		SET name = ?, description = ?, steps = ?, variables = ?, pii_refs = ?, updated_at = ?, is_active = ?
		WHERE id = ?
	`, template.Name, template.Description, string(stepsJSON), variablesJSON, piiRefsJSON, time.Now().UnixMilli(), isActive, template.ID)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "template_updated", map[string]interface{}{
		"id":         template.ID,
		"updated_by": template.CreatedBy,
	})

	return nil
}

// DeleteTemplate removes a task template
func (s *SQLiteStore) DeleteTemplate(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM task_templates WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "template_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

//=============================================================================
// Workflow CRUD
//=============================================================================

// CreateWorkflow creates a new workflow instance
func (s *SQLiteStore) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	variablesJSON, _ := json.Marshal(workflow.Variables)
	agentIDsJSON, _ := json.Marshal(workflow.AgentIDs)

	var startedAt *int64 = nil
	if !workflow.StartedAt.IsZero() {
		t := workflow.StartedAt.UnixMilli()
		startedAt = &t
	}

	var completedAt sql.NullInt64
	if workflow.CompletedAt != nil {
		t := workflow.CompletedAt.UnixMilli()
		completedAt.Int64 = t
		completedAt.Valid = true
	}

	var err error
	_, err = s.db.Exec(`
		INSERT INTO workflows (id, template_id, name, description, status, variables, current_step, agent_ids, started_at, completed_at, error_message, created_by, room_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, workflow.ID, workflow.TemplateID, workflow.Name, workflow.Description, string(workflow.Status), string(variablesJSON), workflow.CurrentStep, string(agentIDsJSON), startedAt, completedAt, workflow.ErrorMessage, workflow.CreatedBy, workflow.RoomID)

	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "workflow_created", map[string]interface{}{
		"id":          workflow.ID,
		"template_id": workflow.TemplateID,
		"created_by":  workflow.CreatedBy,
	})

	return nil
}

// GetWorkflow retrieves a workflow by ID
func (s *SQLiteStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflow := &Workflow{}
	var templateID, variablesJSON, agentIDsJSON string
	var currentStep string
	var startedAt sql.NullInt64
	var completedAt sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, template_id, name, description, status, variables, current_step, agent_ids, started_at, completed_at, error_message, created_by, room_id
		FROM workflows WHERE id = ?
	`, id).Scan(&workflow.ID, &templateID, &workflow.Name, &workflow.Description, &workflow.Status, &variablesJSON, &currentStep, &agentIDsJSON, &startedAt, &completedAt, &workflow.ErrorMessage, &workflow.CreatedBy, &workflow.RoomID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	workflow.CurrentStep = currentStep
	workflow.TemplateID = templateID

	if err := json.Unmarshal([]byte(variablesJSON), &workflow.Variables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	if err := json.Unmarshal([]byte(agentIDsJSON), &workflow.AgentIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent ids: %w", err)
	}

	if startedAt.Valid {
		workflow.StartedAt = time.UnixMilli(startedAt.Int64)
	}

	if completedAt.Valid {
		t := time.UnixMilli(completedAt.Int64)
		workflow.CompletedAt = &t
	}

	return workflow, nil
}

// ListWorkflows returns all workflows, optionally filtered
func (s *SQLiteStore) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, template_id, name, description, status, variables, current_step, agent_ids, started_at, completed_at, error_message, created_by, room_id FROM workflows`
	args := []interface{}{}

	if filter.Status != nil {
		args = append(args, sql.Named("status", string(*filter.Status)))
	}

	if filter.TemplateID != "" {
		args = append(args, sql.Named("template_id", filter.TemplateID))
	}

	if filter.CreatedBy != "" {
		args = append(args, sql.Named("created_by", filter.CreatedBy))
	}

	query += " ORDER BY started_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		workflow := &Workflow{}
		var templateID, variablesJSON, agentIDsJSON string
		var currentStep string
		var startedAt, completedAt sql.NullInt64

		if err := rows.Scan(&workflow.ID, &templateID, &workflow.Name, &workflow.Description, &workflow.Status, &variablesJSON, &currentStep, &agentIDsJSON, &startedAt, &completedAt, &workflow.ErrorMessage, &workflow.CreatedBy, &workflow.RoomID); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		workflow.CurrentStep = currentStep
		workflow.TemplateID = templateID

		if err := json.Unmarshal([]byte(variablesJSON), &workflow.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}

		if err := json.Unmarshal([]byte(agentIDsJSON), &workflow.AgentIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent ids: %w", err)
		}

		if startedAt.Valid {
			workflow.StartedAt = time.UnixMilli(startedAt.Int64)
		}

		if completedAt.Valid {
			t := time.UnixMilli(completedAt.Int64)
			workflow.CompletedAt = &t
		}

		workflows = append(workflows, *workflow)
	}

	return workflows, nil
}

// UpdateWorkflow updates an existing workflow
func (s *SQLiteStore) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	variablesJSON, _ := json.Marshal(workflow.Variables)
	agentIDsJSON, _ := json.Marshal(workflow.AgentIDs)

	startedAt := workflow.StartedAt.UnixMilli()
	var completedAt sql.NullInt64
	if workflow.CompletedAt != nil {
		t := workflow.CompletedAt.UnixMilli()
		completedAt.Int64 = t
		completedAt.Valid = true
	}

	result, err := s.db.Exec(`
		UPDATE workflows
		SET template_id = ?, name = ?, description = ?, status = ?, variables = ?, current_step = ?, agent_ids = ?, started_at = ?, completed_at = ?, error_message = ?, room_id = ?
		WHERE id = ?
	`, workflow.TemplateID, workflow.Name, workflow.Description, string(workflow.Status), string(variablesJSON), workflow.CurrentStep, string(agentIDsJSON), startedAt, completedAt, workflow.ErrorMessage, workflow.RoomID, workflow.ID)

	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}

	s.auditLogger.LogOperation(ctx, "workflow_updated", map[string]interface{}{
		"id":         workflow.ID,
		"updated_by": workflow.CreatedBy,
	})

	return nil
}

// DeleteWorkflow removes a workflow
func (s *SQLiteStore) DeleteWorkflow(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM workflows WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "workflow_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

//=============================================================================
// Approval Policy CRUD
//=============================================================================

// CreatePolicy stores a new approval policy
func (s *SQLiteStore) CreatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	piiFieldsJSON, _ := json.Marshal(policy.PIIFields)
	conditionsJSON := policy.Conditions

	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		INSERT INTO approval_policies (id, name, pii_fields, auto_approve, delegate_to, conditions, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policy.ID, policy.Name, string(piiFieldsJSON), policy.AutoApprove, policy.DelegateTo, conditionsJSON, policy.CreatedBy, now)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "policy_created", map[string]interface{}{
		"id":         policy.ID,
		"name":       policy.Name,
		"created_by": policy.CreatedBy,
	})

	return nil
}

// GetPolicy retrieves an approval policy by ID
func (s *SQLiteStore) GetPolicy(ctx context.Context, id string) (*ApprovalPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy := &ApprovalPolicy{}
	var piiFieldsJSON, conditionsJSON string

	err := s.db.QueryRow(`
		SELECT id, name, pii_fields, auto_approve, delegate_to, conditions, created_by, created_at
		FROM approval_policies WHERE id = ?
	`, id).Scan(&policy.ID, &policy.Name, &piiFieldsJSON, &policy.AutoApprove, &policy.DelegateTo, &conditionsJSON, &policy.CreatedBy, &policy.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if err := json.Unmarshal([]byte(piiFieldsJSON), &policy.PIIFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pii fields: %w", err)
	}

	policy.Conditions = json.RawMessage(conditionsJSON)
	policy.CreatedAt = time.UnixMilli(policy.CreatedAt.UnixMilli())

	return policy, nil
}

// ListPolicies returns all approval policies
func (s *SQLiteStore) ListPolicies(ctx context.Context) ([]ApprovalPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, pii_fields, auto_approve, delegate_to, conditions, created_by, created_at FROM approval_policies ORDER BY name ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer rows.Close()

	var policies []ApprovalPolicy
	for rows.Next() {
		policy := &ApprovalPolicy{}
		var piiFieldsJSON, conditionsJSON string

		if err := rows.Scan(&policy.ID, &policy.Name, &piiFieldsJSON, &policy.AutoApprove, &policy.DelegateTo, &conditionsJSON, &policy.CreatedBy, &policy.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		if err := json.Unmarshal([]byte(piiFieldsJSON), &policy.PIIFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pii fields: %w", err)
		}

		policy.Conditions = json.RawMessage(conditionsJSON)
		policy.CreatedAt = time.UnixMilli(policy.CreatedAt.UnixMilli())

		policies = append(policies, *policy)
	}

	return policies, nil
}

// UpdatePolicy updates an existing approval policy
func (s *SQLiteStore) UpdatePolicy(ctx context.Context, policy *ApprovalPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	piiFieldsJSON, _ := json.Marshal(policy.PIIFields)
	conditionsJSON := policy.Conditions

	_, err := s.db.Exec(`
		UPDATE approval_policies
		SET name = ?, pii_fields = ?, auto_approve = ?, delegate_to = ?, conditions = ?, created_at = ?
		WHERE id = ?
	`, policy.Name, string(piiFieldsJSON), policy.AutoApprove, policy.DelegateTo, conditionsJSON, time.Now().UnixMilli(), policy.ID)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "policy_updated", map[string]interface{}{
		"id":         policy.ID,
		"updated_by": policy.CreatedBy,
	})

	return nil
}

// DeletePolicy removes an approval policy
func (s *SQLiteStore) DeletePolicy(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM approval_policies WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "policy_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

//=============================================================================
// Scheduled Task CRUD
//=============================================================================

// CreateScheduledTask stores a new scheduled task
func (s *SQLiteStore) CreateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	isActive := 0
	if task.IsActive {
		isActive = 1
	}

	var nextRunVal sql.NullInt64
	if task.NextRun != nil {
		nextRunVal.Int64 = task.NextRun.UnixMilli()
		nextRunVal.Valid = true
	}

	var lastRunVal sql.NullInt64
	if task.LastRun != nil {
		lastRunVal.Int64 = task.LastRun.UnixMilli()
		lastRunVal.Valid = true
	}

	_, err := s.db.Exec(`
		INSERT INTO scheduled_tasks (id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.TemplateID, task.DefinitionID, task.CronExpression, task.Timezone, nextRunVal, lastRunVal, isActive, task.CreatedBy)

	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "scheduled_task_created", map[string]interface{}{
		"id":          task.ID,
		"template_id": task.TemplateID,
		"created_by":  task.CreatedBy,
	})

	return nil
}

// GetScheduledTask retrieves a scheduled task by ID
func (s *SQLiteStore) GetScheduledTask(ctx context.Context, id string) (*ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task := &ScheduledTask{}
	var nextRun, lastRun sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by
		FROM scheduled_tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.TemplateID, &task.DefinitionID, &task.CronExpression, &task.Timezone, &nextRun, &lastRun, &task.IsActive, &task.CreatedBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduled task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled task: %w", err)
	}

	if nextRun.Valid {
		t := time.UnixMilli(nextRun.Int64)
		task.NextRun = &t
	}

	if lastRun.Valid {
		t := time.UnixMilli(lastRun.Int64)
		task.LastRun = &t
	}

	return task, nil
}

// ListScheduledTasks returns all scheduled tasks
func (s *SQLiteStore) ListScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by FROM scheduled_tasks ORDER BY next_run ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		task := &ScheduledTask{}
		var nextRun, lastRun sql.NullInt64

		if err := rows.Scan(&task.ID, &task.TemplateID, &task.DefinitionID, &task.CronExpression, &task.Timezone, &nextRun, &lastRun, &task.IsActive, &task.CreatedBy); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}

		if nextRun.Valid {
			t := time.UnixMilli(nextRun.Int64)
			task.NextRun = &t
		}

		if lastRun.Valid {
			t := time.UnixMilli(lastRun.Int64)
			task.LastRun = &t
		}

		tasks = append(tasks, *task)
	}

	return tasks, nil
}

// UpdateScheduledTask updates an existing scheduled task
func (s *SQLiteStore) UpdateScheduledTask(ctx context.Context, task *ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var nextRunVal sql.NullInt64
	if task.NextRun != nil {
		nextRunVal.Int64 = task.NextRun.UnixMilli()
		nextRunVal.Valid = true
	}

	var lastRun sql.NullInt64
	if task.LastRun != nil {
		lastRun.Int64 = task.LastRun.UnixMilli()
		lastRun.Valid = true
	}

	var err error
	_, err = s.db.Exec(`
		UPDATE scheduled_tasks
		SET template_id = ?, definition_id = ?, cron_expression = ?, timezone = ?, next_run = ?, last_run = ?, is_active = ?
		WHERE id = ?
	`, task.TemplateID, task.DefinitionID, task.CronExpression, task.Timezone, nextRunVal, lastRun, task.IsActive, task.ID)

	if err != nil {
		return fmt.Errorf("failed to update scheduled task: %w", err)
	}

	return nil
}

// DeleteScheduledTask removes a scheduled task
func (s *SQLiteStore) DeleteScheduledTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM scheduled_tasks WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "scheduled_task_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

// ListPendingScheduledTasks returns tasks due to run
func (s *SQLiteStore) ListPendingScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by FROM scheduled_tasks WHERE is_active = 1 AND next_run <= ? ORDER BY next_run ASC`
	args := []interface{}{sql.Named("next_run", time.Now().UnixMilli())}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		task := &ScheduledTask{}
		var nextRun, lastRun sql.NullInt64

		if err := rows.Scan(&task.ID, &task.TemplateID, &task.DefinitionID, &task.CronExpression, &task.Timezone, &nextRun, &lastRun, &task.IsActive, &task.CreatedBy); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}

		if nextRun.Valid {
			t := time.UnixMilli(nextRun.Int64)
			task.NextRun = &t
		}

		if lastRun.Valid {
			t := time.UnixMilli(lastRun.Int64)
			task.LastRun = &t
		}

		tasks = append(tasks, *task)
	}

	return tasks, nil
}

// ListDueTasks returns active tasks where next_run <= now AND definition_id != ”
func (s *SQLiteStore) ListDueTasks(ctx context.Context) ([]ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, template_id, definition_id, cron_expression, timezone, next_run, last_run, is_active, created_by FROM scheduled_tasks WHERE is_active = 1 AND next_run <= ? AND definition_id != '' ORDER BY next_run ASC`

	rows, err := s.db.Query(query, time.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to list due tasks: %w", err)
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		task := &ScheduledTask{}
		var nextRun, lastRun sql.NullInt64

		if err := rows.Scan(&task.ID, &task.TemplateID, &task.DefinitionID, &task.CronExpression, &task.Timezone, &nextRun, &lastRun, &task.IsActive, &task.CreatedBy); err != nil {
			return nil, fmt.Errorf("failed to scan due task: %w", err)
		}

		if nextRun.Valid {
			t := time.UnixMilli(nextRun.Int64)
			task.NextRun = &t
		}
		if lastRun.Valid {
			t := time.UnixMilli(lastRun.Int64)
			task.LastRun = &t
		}

		tasks = append(tasks, *task)
	}

	return tasks, nil
}

// MarkDispatched updates a task's last_run and next_run after dispatch
func (s *SQLiteStore) MarkDispatched(ctx context.Context, taskID string, nextRun time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	nextRunMillis := nextRun.UnixMilli()

	_, err := s.db.Exec(`
		UPDATE scheduled_tasks
		SET last_run = ?, next_run = ?
		WHERE id = ?
	`, now, nextRunMillis, taskID)

	if err != nil {
		return fmt.Errorf("failed to mark task dispatched: %w", err)
	}

	return nil
}

//=============================================================================
// Notification Channel CRUD
//=============================================================================

// CreateNotificationChannel stores a new notification channel
func (s *SQLiteStore) CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventTypesJSON, _ := json.Marshal(channel.EventTypes)

	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		INSERT INTO notification_channels (id, user_id, channel_type, destination, event_types, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.UserID, channel.ChannelType, channel.Destination, string(eventTypesJSON), channel.IsActive, now)

	if err != nil {
		return fmt.Errorf("failed to create notification channel: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "notification_channel_created", map[string]interface{}{
		"id":      channel.ID,
		"user_id": channel.UserID,
	})

	return nil
}

// GetNotificationChannel retrieves a notification channel by ID
func (s *SQLiteStore) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channel := &NotificationChannel{}
	var eventTypesJSON string

	err := s.db.QueryRow(`
		SELECT id, user_id, channel_type, destination, event_types, is_active, created_at
		FROM notification_channels WHERE id = ?
	`, id).Scan(&channel.ID, &channel.UserID, &channel.ChannelType, &channel.Destination, &eventTypesJSON, &channel.IsActive, &channel.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification channel not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}

	if err := json.Unmarshal([]byte(eventTypesJSON), &channel.EventTypes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event types: %w", err)
	}

	channel.CreatedAt = time.UnixMilli(channel.CreatedAt.UnixMilli())

	return channel, nil
}

// ListNotificationChannels returns all notification channels for a user
func (s *SQLiteStore) ListNotificationChannels(ctx context.Context, userID string) ([]NotificationChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, user_id, channel_type, destination, event_types, is_active, created_at FROM notification_channels WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		channel := &NotificationChannel{}
		var eventTypesJSON string

		if err := rows.Scan(&channel.ID, &channel.UserID, &channel.ChannelType, &channel.Destination, &eventTypesJSON, &channel.IsActive, &channel.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification channel: %w", err)
		}

		if err := json.Unmarshal([]byte(eventTypesJSON), &channel.EventTypes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event types: %w", err)
		}

		channel.CreatedAt = time.UnixMilli(channel.CreatedAt.UnixMilli())

		channels = append(channels, *channel)
	}

	return channels, nil
}

// UpdateNotificationChannel updates an existing notification channel
func (s *SQLiteStore) UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventTypesJSON, _ := json.Marshal(channel.EventTypes)

	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		UPDATE notification_channels
		SET channel_type = ?, destination = ?, event_types = ?, is_active = ?, created_at = ?
		WHERE id = ?
	`, channel.ChannelType, channel.Destination, string(eventTypesJSON), channel.IsActive, now, channel.ID)

	if err != nil {
		return fmt.Errorf("failed to update notification channel: %w", err)
	}

	return nil
}

// DeleteNotificationChannel removes a notification channel
func (s *SQLiteStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM notification_channels WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "notification_channel_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

//=============================================================================
// Contact CRUD (Rolodex)
//=============================================================================

// CreateContact stores a new contact
func (s *SQLiteStore) CreateContact(ctx context.Context, contact *Contact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if contact.EncryptedData == nil || contact.EncryptedNonce == nil {
		return fmt.Errorf("encrypted data and nonce are required")
	}

	_, err := s.db.Exec(`
		INSERT INTO contacts (id, name, company, relationship, data_encrypted, data_nonce, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, contact.ID, contact.Name, contact.Company, contact.Relationship,
		contact.EncryptedData, contact.EncryptedNonce, contact.CreatedBy,
		time.Now().UnixMilli(), time.Now().UnixMilli())

	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "contact_created", map[string]interface{}{
		"id":         contact.ID,
		"name":       contact.Name,
		"created_by": contact.CreatedBy,
	})

	return nil
}

// GetContact retrieves a contact by ID
func (s *SQLiteStore) GetContact(ctx context.Context, id string) (*Contact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contact := &Contact{}
	var createdAt, updatedAt int64

	err := s.db.QueryRow(`
		SELECT id, name, company, relationship, data_encrypted, data_nonce, created_by, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id).Scan(&contact.ID, &contact.Name, &contact.Company, &contact.Relationship,
		&contact.EncryptedData, &contact.EncryptedNonce, &contact.CreatedBy,
		&createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contact not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	contact.CreatedAt = time.UnixMilli(createdAt)
	contact.UpdatedAt = time.UnixMilli(updatedAt)

	return contact, nil
}

// ListContacts returns all contacts, optionally filtered
func (s *SQLiteStore) ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, company, relationship, data_encrypted, data_nonce, created_by, created_at, updated_at FROM contacts`
	args := []interface{}{}

	if filter.Name != "" {
		query += " WHERE name LIKE ?"
		args = append(args, "%"+filter.Name+"%")
	}

	if filter.Company != "" {
		if filter.Name != "" {
			query += " AND"
		} else {
			query += " WHERE"
		}
		query += " company LIKE ?"
		args = append(args, "%"+filter.Company+"%")
	}

	if filter.Relationship != "" {
		if filter.Name != "" || filter.Company != "" {
			query += " AND"
		} else {
			query += " WHERE"
		}
		query += " relationship = ?"
		args = append(args, filter.Relationship)
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		contact := &Contact{}
		var createdAt, updatedAt int64
		if err := rows.Scan(&contact.ID, &contact.Name, &contact.Company, &contact.Relationship,
			&contact.EncryptedData, &contact.EncryptedNonce, &contact.CreatedBy,
			&createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contact.CreatedAt = time.UnixMilli(createdAt)
		contact.UpdatedAt = time.UnixMilli(updatedAt)

		contacts = append(contacts, *contact)
	}

	return contacts, nil
}

// UpdateContact updates an existing contact
func (s *SQLiteStore) UpdateContact(ctx context.Context, contact *Contact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if contact.EncryptedData == nil || contact.EncryptedNonce == nil {
		return fmt.Errorf("encrypted data and nonce are required")
	}

	result, err := s.db.Exec(`
		UPDATE contacts
		SET name = ?, company = ?, relationship = ?, data_encrypted = ?, data_nonce = ?, updated_at = ?
		WHERE id = ?
	`, contact.Name, contact.Company, contact.Relationship, contact.EncryptedData,
		contact.EncryptedNonce, time.Now().UnixMilli(), contact.ID)

	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("contact not found: %s", contact.ID)
	}

	s.auditLogger.LogOperation(ctx, "contact_updated", map[string]interface{}{
		"id":         contact.ID,
		"updated_by": contact.CreatedBy,
	})

	return nil
}

// DeleteContact removes a contact
func (s *SQLiteStore) DeleteContact(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM contacts WHERE id = ?`, id)

	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	s.auditLogger.LogOperation(ctx, "contact_deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Close()
}
