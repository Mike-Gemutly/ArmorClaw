package studio

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

//=============================================================================
// Store Interface
//=============================================================================

// Store defines the persistence interface for the Agent Studio
type Store interface {
	// Agent Definitions
	CreateDefinition(def *AgentDefinition) error
	GetDefinition(id string) (*AgentDefinition, error)
	GetDefinitionByName(name string) (*AgentDefinition, error)
	ListDefinitions(activeOnly bool) ([]*AgentDefinition, error)
	UpdateDefinition(def *AgentDefinition) error
	DeleteDefinition(id string) error

	// Skill Registry
	RegisterSkill(skill *Skill) error
	GetSkill(id string) (*Skill, error)
	ListSkills(category string) ([]*Skill, error)
	DeleteSkill(id string) error

	// PII Registry
	RegisterPIIField(field *PIIField) error
	GetPIIField(id string) (*PIIField, error)
	ListPIIFields(sensitivity string) ([]*PIIField, error)
	DeletePIIField(id string) error

	// Agent Instances
	CreateInstance(instance *AgentInstance) error
	GetInstance(id string) (*AgentInstance, error)
	ListInstances(definitionID string, status InstanceStatus) ([]*AgentInstance, error)
	UpdateInstance(instance *AgentInstance) error

	// Statistics
	GetStats() (*StudioStats, error)

	// Lifecycle
	Close() error
}

//=============================================================================
// SQLite Store Implementation
//=============================================================================

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	mu   sync.RWMutex
	db   *sql.DB
	log  *logger.Logger
	path string
}

// StoreConfig holds configuration for the store
type StoreConfig struct {
	Path   string
	Logger *logger.Logger
}

// NewStore creates a new SQLite store
func NewStore(cfg StoreConfig) (*SQLiteStore, error) {
	if cfg.Path == "" {
		cfg.Path = ":memory:"
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("studio")
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
		db:   db,
		log:  log,
		path: cfg.Path,
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
	schema := `
	-- Agent Definitions
	CREATE TABLE IF NOT EXISTS agent_definitions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		skills TEXT NOT NULL,
		pii_access TEXT NOT NULL,
		resource_tier TEXT DEFAULT 'medium',
		created_by TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		is_active INTEGER DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_agent_definitions_created_by ON agent_definitions(created_by);
	CREATE INDEX IF NOT EXISTS idx_agent_definitions_is_active ON agent_definitions(is_active);

	-- Skill Registry
	CREATE TABLE IF NOT EXISTS skill_registry (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT,
		container_image TEXT,
		required_env_vars TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_skill_registry_category ON skill_registry(category);

	-- PII Registry
	CREATE TABLE IF NOT EXISTS pii_registry (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		sensitivity TEXT DEFAULT 'medium',
		keystore_key TEXT,
		requires_approval INTEGER DEFAULT 1,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS pii_registry_sensitivity ON pii_registry(sensitivity);

	-- Resource Profiles
	CREATE TABLE IF NOT EXISTS resource_profiles (
		tier TEXT PRIMARY KEY,
		memory_mb INTEGER NOT NULL,
		cpu_shares INTEGER NOT NULL,
		timeout_seconds INTEGER NOT NULL,
		max_concurrency INTEGER DEFAULT 3,
		description TEXT
	);

	-- Agent Instances
	CREATE TABLE IF NOT EXISTS agent_instances (
		id TEXT PRIMARY KEY,
		definition_id TEXT NOT NULL,
		container_id TEXT,
		status TEXT DEFAULT 'pending',
		task_description TEXT,
		spawned_by TEXT NOT NULL,
		started_at INTEGER,
		completed_at INTEGER,
		exit_code INTEGER,
		error_message TEXT,
		FOREIGN KEY (definition_id) REFERENCES agent_definitions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_agent_instances_definition_id ON agent_instances(definition_id);
	CREATE INDEX IF NOT EXISTS idx_agent_instances_status ON agent_instances(status);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert default data
	if err := s.insertDefaults(); err != nil {
		return fmt.Errorf("failed to insert defaults: %w", err)
	}

	return nil
}

// insertDefaults populates the database with default skills, PII fields, and profiles
func (s *SQLiteStore) insertDefaults() error {
	now := time.Now().UnixMilli()

	// Default skills
	defaultSkills := []struct {
		id, name, description, category string
	}{
		{"browser_navigate", "Browser Navigation", "Navigate to URLs and extract content", "research"},
		{"form_filler", "Form Filler", "Fill web forms with provided data", "automation"},
		{"pdf_generator", "PDF Generator", "Create PDF documents from templates", "document"},
		{"template_filler", "Template Filler", "Fill document templates with data", "document"},
		{"email_sender", "Email Sender", "Send emails via configured SMTP", "communication"},
		{"calendar", "Calendar Manager", "Manage calendar events and scheduling", "productivity"},
		{"web_scraper", "Web Scraper", "Extract data from web pages", "research"},
		{"data_processor", "Data Processor", "Process and transform data files", "data"},
	}

	for _, skill := range defaultSkills {
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO skill_registry (id, name, description, category, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, skill.id, skill.name, skill.description, skill.category, now)
		if err != nil {
			s.log.Warn("failed_to_insert_default_skill", "error", err, "skill", skill.id)
		}
	}

	// Default PII fields
	defaultPII := []struct {
		id, name, description, sensitivity string
		requiresApproval                   bool
	}{
		{"client_name", "Client Name", "Full name of the client", "low", false},
		{"client_email", "Client Email", "Email address of the client", "medium", true},
		{"client_phone", "Client Phone", "Phone number of the client", "medium", true},
		{"client_address", "Client Address", "Physical address of the client", "medium", true},
		{"client_ssn", "Client SSN", "Social Security Number", "critical", true},
		{"client_dob", "Client DOB", "Date of birth", "high", true},
		{"contract_value", "Contract Value", "Financial value of contracts", "high", true},
		{"account_number", "Account Number", "Bank or account numbers", "critical", true},
		{"company_name", "Company Name", "Name of the organization", "low", false},
		{"case_number", "Case Number", "Legal case identifiers", "medium", false},
	}

	for _, field := range defaultPII {
		approvalInt := 0
		if field.requiresApproval {
			approvalInt = 1
		}
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO pii_registry (id, name, description, sensitivity, requires_approval, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, field.id, field.name, field.description, field.sensitivity, approvalInt, now)
		if err != nil {
			s.log.Warn("failed_to_insert_default_pii", "error", err, "field", field.id)
		}
	}

	// Default resource profiles
	defaultProfiles := []ResourceProfile{
		{Tier: "low", MemoryMB: 256, CPUShares: 512, TimeoutSeconds: 300, MaxConcurrency: 5, Description: "Lightweight tasks, quick operations"},
		{Tier: "medium", MemoryMB: 512, CPUShares: 1024, TimeoutSeconds: 600, MaxConcurrency: 3, Description: "Standard tasks, moderate processing"},
		{Tier: "high", MemoryMB: 2048, CPUShares: 2048, TimeoutSeconds: 1800, MaxConcurrency: 1, Description: "Heavy processing, complex operations"},
	}

	for _, profile := range defaultProfiles {
		_, err := s.db.Exec(`
			INSERT OR REPLACE INTO resource_profiles (tier, memory_mb, cpu_shares, timeout_seconds, max_concurrency, description)
			VALUES (?, ?, ?, ?, ?, ?)
		`, profile.Tier, profile.MemoryMB, profile.CPUShares, profile.TimeoutSeconds, profile.MaxConcurrency, profile.Description)
		if err != nil {
			s.log.Warn("failed_to_insert_profile", "error", err, "tier", profile.Tier)
		}
	}

	return nil
}

//=============================================================================
// Agent Definitions
//=============================================================================

// CreateDefinition stores a new agent definition
func (s *SQLiteStore) CreateDefinition(def *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	skillsJSON, err := json.Marshal(def.Skills)
	if err != nil {
		return fmt.Errorf("failed to marshal skills: %w", err)
	}

	piiJSON, err := json.Marshal(def.PIIAccess)
	if err != nil {
		return fmt.Errorf("failed to marshal pii_access: %w", err)
	}

	isActive := 0
	if def.IsActive {
		isActive = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO agent_definitions (id, name, description, skills, pii_access, resource_tier, created_by, created_at, updated_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, def.ID, def.Name, def.Description, string(skillsJSON), string(piiJSON), def.ResourceTier, def.CreatedBy, def.CreatedAt.UnixMilli(), def.UpdatedAt.UnixMilli(), isActive)

	if err != nil {
		return fmt.Errorf("failed to create definition: %w", err)
	}

	s.log.Info("agent_definition_created", "id", def.ID, "name", def.Name, "created_by", def.CreatedBy)
	return nil
}

// GetDefinition retrieves an agent definition by ID
func (s *SQLiteStore) GetDefinition(id string) (*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	def := &AgentDefinition{}
	var skillsJSON, piiJSON string
	var isActive int

	err := s.db.QueryRow(`
		SELECT id, name, description, skills, pii_access, resource_tier, created_by, created_at, updated_at, is_active
		FROM agent_definitions WHERE id = ?
	`, id).Scan(&def.ID, &def.Name, &def.Description, &skillsJSON, &piiJSON, &def.ResourceTier, &def.CreatedBy, &def.CreatedAt, &def.UpdatedAt, &isActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("definition not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get definition: %w", err)
	}

	if err := json.Unmarshal([]byte(skillsJSON), &def.Skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
	}

	if err := json.Unmarshal([]byte(piiJSON), &def.PIIAccess); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pii_access: %w", err)
	}

	def.IsActive = isActive == 1
	// Convert timestamps back to time.Time
	def.CreatedAt = time.UnixMilli(def.CreatedAt.UnixMilli())
	def.UpdatedAt = time.UnixMilli(def.UpdatedAt.UnixMilli())

	return def, nil
}

// GetDefinitionByName retrieves an agent definition by name
func (s *SQLiteStore) GetDefinitionByName(name string) (*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	def := &AgentDefinition{}
	var skillsJSON, piiJSON string
	var isActive int

	err := s.db.QueryRow(`
		SELECT id, name, description, skills, pii_access, resource_tier, created_by, created_at, updated_at, is_active
		FROM agent_definitions WHERE name = ?
	`, name).Scan(&def.ID, &def.Name, &def.Description, &skillsJSON, &piiJSON, &def.ResourceTier, &def.CreatedBy, &def.CreatedAt, &def.UpdatedAt, &isActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("definition not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get definition: %w", err)
	}

	if err := json.Unmarshal([]byte(skillsJSON), &def.Skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
	}

	if err := json.Unmarshal([]byte(piiJSON), &def.PIIAccess); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pii_access: %w", err)
	}

	def.IsActive = isActive == 1
	return def, nil
}

// ListDefinitions returns all agent definitions
func (s *SQLiteStore) ListDefinitions(activeOnly bool) ([]*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, description, skills, pii_access, resource_tier, created_by, created_at, updated_at, is_active FROM agent_definitions`
	if activeOnly {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list definitions: %w", err)
	}
	defer rows.Close()

	var definitions []*AgentDefinition
	for rows.Next() {
		def := &AgentDefinition{}
		var skillsJSON, piiJSON string
		var isActive int

		if err := rows.Scan(&def.ID, &def.Name, &def.Description, &skillsJSON, &piiJSON, &def.ResourceTier, &def.CreatedBy, &def.CreatedAt, &def.UpdatedAt, &isActive); err != nil {
			return nil, fmt.Errorf("failed to scan definition: %w", err)
		}

		if err := json.Unmarshal([]byte(skillsJSON), &def.Skills); err != nil {
			return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
		}

		if err := json.Unmarshal([]byte(piiJSON), &def.PIIAccess); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pii_access: %w", err)
		}

		def.IsActive = isActive == 1
		definitions = append(definitions, def)
	}

	return definitions, nil
}

// UpdateDefinition updates an existing agent definition
func (s *SQLiteStore) UpdateDefinition(def *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	skillsJSON, err := json.Marshal(def.Skills)
	if err != nil {
		return fmt.Errorf("failed to marshal skills: %w", err)
	}

	piiJSON, err := json.Marshal(def.PIIAccess)
	if err != nil {
		return fmt.Errorf("failed to marshal pii_access: %w", err)
	}

	isActive := 0
	if def.IsActive {
		isActive = 1
	}

	def.UpdatedAt = time.Now()

	result, err := s.db.Exec(`
		UPDATE agent_definitions
		SET name = ?, description = ?, skills = ?, pii_access = ?, resource_tier = ?, updated_at = ?, is_active = ?
		WHERE id = ?
	`, def.Name, def.Description, string(skillsJSON), string(piiJSON), def.ResourceTier, def.UpdatedAt.UnixMilli(), isActive, def.ID)

	if err != nil {
		return fmt.Errorf("failed to update definition: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("definition not found: %s", def.ID)
	}

	s.log.Info("agent_definition_updated", "id", def.ID, "name", def.Name)
	return nil
}

// DeleteDefinition removes an agent definition
func (s *SQLiteStore) DeleteDefinition(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM agent_definitions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete definition: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("definition not found: %s", id)
	}

	s.log.Info("agent_definition_deleted", "id", id)
	return nil
}

//=============================================================================
// Skill Registry
//=============================================================================

// RegisterSkill adds a skill to the registry
func (s *SQLiteStore) RegisterSkill(skill *Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	envVarsJSON, err := json.Marshal(skill.RequiredEnvVars)
	if err != nil {
		return fmt.Errorf("failed to marshal env vars: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO skill_registry (id, name, description, category, container_image, required_env_vars, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, skill.ID, skill.Name, skill.Description, skill.Category, skill.ContainerImage, string(envVarsJSON), skill.CreatedAt.UnixMilli())

	if err != nil {
		return fmt.Errorf("failed to register skill: %w", err)
	}

	s.log.Info("skill_registered", "id", skill.ID, "name", skill.Name)
	return nil
}

// GetSkill retrieves a skill by ID
func (s *SQLiteStore) GetSkill(id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill := &Skill{}
	var envVarsJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, description, category, container_image, required_env_vars, created_at
		FROM skill_registry WHERE id = ?
	`, id).Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category, &skill.ContainerImage, &envVarsJSON, &skill.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	if envVarsJSON.Valid && envVarsJSON.String != "" {
		if err := json.Unmarshal([]byte(envVarsJSON.String), &skill.RequiredEnvVars); err != nil {
			return nil, fmt.Errorf("failed to unmarshal env vars: %w", err)
		}
	}

	return skill, nil
}

// ListSkills returns all skills, optionally filtered by category
func (s *SQLiteStore) ListSkills(category string) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, description, category, container_image, required_env_vars, created_at FROM skill_registry`
	args := []interface{}{}

	if category != "" {
		query += " WHERE category = ?"
		args = append(args, category)
	}
	query += " ORDER BY category, name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	defer rows.Close()

	var skills []*Skill
	for rows.Next() {
		skill := &Skill{}
		var envVarsJSON sql.NullString

		if err := rows.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category, &skill.ContainerImage, &envVarsJSON, &skill.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}

		if envVarsJSON.Valid && envVarsJSON.String != "" {
			if err := json.Unmarshal([]byte(envVarsJSON.String), &skill.RequiredEnvVars); err != nil {
				return nil, fmt.Errorf("failed to unmarshal env vars: %w", err)
			}
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// DeleteSkill removes a skill from the registry
func (s *SQLiteStore) DeleteSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM skill_registry WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("skill not found: %s", id)
	}

	s.log.Info("skill_deleted", "id", id)
	return nil
}

//=============================================================================
// PII Registry
//=============================================================================

// RegisterPIIField adds a PII field to the registry
func (s *SQLiteStore) RegisterPIIField(field *PIIField) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	requiresApproval := 0
	if field.RequiresApproval {
		requiresApproval = 1
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO pii_registry (id, name, description, sensitivity, keystore_key, requires_approval, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, field.ID, field.Name, field.Description, field.Sensitivity, field.KeystoreKey, requiresApproval, field.CreatedAt.UnixMilli())

	if err != nil {
		return fmt.Errorf("failed to register PII field: %w", err)
	}

	s.log.Info("pii_field_registered", "id", field.ID, "name", field.Name)
	return nil
}

// GetPIIField retrieves a PII field by ID
func (s *SQLiteStore) GetPIIField(id string) (*PIIField, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	field := &PIIField{}
	var requiresApproval int

	err := s.db.QueryRow(`
		SELECT id, name, description, sensitivity, keystore_key, requires_approval, created_at
		FROM pii_registry WHERE id = ?
	`, id).Scan(&field.ID, &field.Name, &field.Description, &field.Sensitivity, &field.KeystoreKey, &requiresApproval, &field.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("PII field not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PII field: %w", err)
	}

	field.RequiresApproval = requiresApproval == 1
	return field, nil
}

// ListPIIFields returns all PII fields, optionally filtered by sensitivity
func (s *SQLiteStore) ListPIIFields(sensitivity string) ([]*PIIField, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, description, sensitivity, keystore_key, requires_approval, created_at FROM pii_registry`
	args := []interface{}{}

	if sensitivity != "" {
		query += " WHERE sensitivity = ?"
		args = append(args, sensitivity)
	}
	query += " ORDER BY sensitivity, name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list PII fields: %w", err)
	}
	defer rows.Close()

	var fields []*PIIField
	for rows.Next() {
		field := &PIIField{}
		var requiresApproval int

		if err := rows.Scan(&field.ID, &field.Name, &field.Description, &field.Sensitivity, &field.KeystoreKey, &requiresApproval, &field.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan PII field: %w", err)
		}

		field.RequiresApproval = requiresApproval == 1
		fields = append(fields, field)
	}

	return fields, nil
}

// DeletePIIField removes a PII field from the registry
func (s *SQLiteStore) DeletePIIField(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM pii_registry WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete PII field: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("PII field not found: %s", id)
	}

	s.log.Info("pii_field_deleted", "id", id)
	return nil
}

//=============================================================================
// Agent Instances
//=============================================================================

// CreateInstance stores a new agent instance
func (s *SQLiteStore) CreateInstance(instance *AgentInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var startedAt, completedAt sql.NullInt64
	if instance.StartedAt != nil {
		startedAt.Int64 = instance.StartedAt.UnixMilli()
		startedAt.Valid = true
	}
	if instance.CompletedAt != nil {
		completedAt.Int64 = instance.CompletedAt.UnixMilli()
		completedAt.Valid = true
	}

	var exitCode sql.NullInt64
	if instance.ExitCode != nil {
		exitCode.Int64 = int64(*instance.ExitCode)
		exitCode.Valid = true
	}

	_, err := s.db.Exec(`
		INSERT INTO agent_instances (id, definition_id, container_id, status, task_description, spawned_by, started_at, completed_at, exit_code, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, instance.ID, instance.DefinitionID, instance.ContainerID, instance.Status, instance.TaskDescription, instance.SpawnedBy, startedAt, completedAt, exitCode, instance.ErrorMessage)

	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	s.log.Info("agent_instance_created", "id", instance.ID, "definition_id", instance.DefinitionID)
	return nil
}

// GetInstance retrieves an agent instance by ID
func (s *SQLiteStore) GetInstance(id string) (*AgentInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance := &AgentInstance{}
	var startedAt, completedAt, exitCode sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, definition_id, container_id, status, task_description, spawned_by, started_at, completed_at, exit_code, error_message
		FROM agent_instances WHERE id = ?
	`, id).Scan(&instance.ID, &instance.DefinitionID, &instance.ContainerID, &instance.Status, &instance.TaskDescription, &instance.SpawnedBy, &startedAt, &completedAt, &exitCode, &instance.ErrorMessage)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if startedAt.Valid {
		t := time.UnixMilli(startedAt.Int64)
		instance.StartedAt = &t
	}
	if completedAt.Valid {
		t := time.UnixMilli(completedAt.Int64)
		instance.CompletedAt = &t
	}
	if exitCode.Valid {
		code := int(exitCode.Int64)
		instance.ExitCode = &code
	}

	return instance, nil
}

// ListInstances returns agent instances, optionally filtered
func (s *SQLiteStore) ListInstances(definitionID string, status InstanceStatus) ([]*AgentInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, definition_id, container_id, status, task_description, spawned_by, started_at, completed_at, exit_code, error_message FROM agent_instances WHERE 1=1`
	args := []interface{}{}

	if definitionID != "" {
		query += " AND definition_id = ?"
		args = append(args, definitionID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY started_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*AgentInstance
	for rows.Next() {
		instance := &AgentInstance{}
		var startedAt, completedAt, exitCode sql.NullInt64

		if err := rows.Scan(&instance.ID, &instance.DefinitionID, &instance.ContainerID, &instance.Status, &instance.TaskDescription, &instance.SpawnedBy, &startedAt, &completedAt, &exitCode, &instance.ErrorMessage); err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}

		if startedAt.Valid {
			t := time.UnixMilli(startedAt.Int64)
			instance.StartedAt = &t
		}
		if completedAt.Valid {
			t := time.UnixMilli(completedAt.Int64)
			instance.CompletedAt = &t
		}
		if exitCode.Valid {
			code := int(exitCode.Int64)
			instance.ExitCode = &code
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

// UpdateInstance updates an agent instance
func (s *SQLiteStore) UpdateInstance(instance *AgentInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var startedAt, completedAt sql.NullInt64
	if instance.StartedAt != nil {
		startedAt.Int64 = instance.StartedAt.UnixMilli()
		startedAt.Valid = true
	}
	if instance.CompletedAt != nil {
		completedAt.Int64 = instance.CompletedAt.UnixMilli()
		completedAt.Valid = true
	}

	var exitCode sql.NullInt64
	if instance.ExitCode != nil {
		exitCode.Int64 = int64(*instance.ExitCode)
		exitCode.Valid = true
	}

	result, err := s.db.Exec(`
		UPDATE agent_instances
		SET container_id = ?, status = ?, started_at = ?, completed_at = ?, exit_code = ?, error_message = ?
		WHERE id = ?
	`, instance.ContainerID, instance.Status, startedAt, completedAt, exitCode, instance.ErrorMessage, instance.ID)

	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("instance not found: %s", instance.ID)
	}

	return nil
}

//=============================================================================
// Statistics
//=============================================================================

// GetStats returns overview statistics
func (s *SQLiteStore) GetStats() (*StudioStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StudioStats{
		ByTier: make(map[string]int),
	}

	// Total definitions
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_definitions`).Scan(&stats.TotalDefinitions); err != nil {
		return nil, err
	}

	// Active definitions
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_definitions WHERE is_active = 1`).Scan(&stats.ActiveDefinitions); err != nil {
		return nil, err
	}

	// Total instances
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_instances`).Scan(&stats.TotalInstances); err != nil {
		return nil, err
	}

	// Running instances
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_instances WHERE status = 'running'`).Scan(&stats.RunningInstances); err != nil {
		return nil, err
	}

	// Skills count
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM skill_registry`).Scan(&stats.SkillsAvailable); err != nil {
		return nil, err
	}

	// PII fields count
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pii_registry`).Scan(&stats.PIIFieldsAvailable); err != nil {
		return nil, err
	}

	// By tier
	rows, err := s.db.Query(`SELECT resource_tier, COUNT(*) FROM agent_definitions GROUP BY resource_tier`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tier string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			return nil, err
		}
		stats.ByTier[tier] = count
	}

	return stats, nil
}

//=============================================================================
// Lifecycle
//=============================================================================

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
