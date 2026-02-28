-- Agent Studio Database Schema
-- Version: 1.0.0
-- Created: 2026-02-27

-- =============================================================================
-- Agent Definitions
-- =============================================================================

-- Stores no-code agent configurations
CREATE TABLE IF NOT EXISTS agent_definitions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    skills TEXT NOT NULL,              -- JSON array: ["pdf_generator", "template_filler"]
    pii_access TEXT NOT NULL,          -- JSON array: ["client_name", "contract_value"]
    resource_tier TEXT DEFAULT 'medium', -- "low", "medium", "high"
    created_by TEXT NOT NULL,          -- Matrix user ID
    created_at INTEGER NOT NULL,       -- Unix timestamp (milliseconds)
    updated_at INTEGER NOT NULL,       -- Unix timestamp (milliseconds)
    is_active INTEGER DEFAULT 1        -- Boolean (0 = false, 1 = true)
);

CREATE INDEX IF NOT EXISTS idx_agent_definitions_created_by ON agent_definitions(created_by);
CREATE INDEX IF NOT EXISTS idx_agent_definitions_is_active ON agent_definitions(is_active);
CREATE INDEX IF NOT EXISTS idx_agent_definitions_tier ON agent_definitions(resource_tier);

-- =============================================================================
-- Skill Registry
-- =============================================================================

-- Stores available skills that agents can use
CREATE TABLE IF NOT EXISTS skill_registry (
    id TEXT PRIMARY KEY,               -- "pdf_generator"
    name TEXT NOT NULL,                -- "PDF Generator"
    description TEXT,
    category TEXT,                     -- "document", "communication", "research"
    container_image TEXT,              -- Optional custom image
    required_env_vars TEXT,            -- JSON array of required env var names
    created_at INTEGER NOT NULL        -- Unix timestamp (milliseconds)
);

CREATE INDEX IF NOT EXISTS idx_skill_registry_category ON skill_registry(category);

-- =============================================================================
-- PII Registry
-- =============================================================================

-- Stores available PII fields that agents can access
CREATE TABLE IF NOT EXISTS pii_registry (
    id TEXT PRIMARY KEY,               -- "client_ssn"
    name TEXT NOT NULL,                -- "Client SSN"
    description TEXT,
    sensitivity TEXT DEFAULT 'medium', -- "low", "medium", "high", "critical"
    keystore_key TEXT,                 -- Key path in keystore
    requires_approval INTEGER DEFAULT 1, -- Boolean (0 = false, 1 = true)
    created_at INTEGER NOT NULL        -- Unix timestamp (milliseconds)
);

CREATE INDEX IF NOT EXISTS pii_registry_sensitivity ON pii_registry(sensitivity);
CREATE INDEX IF NOT EXISTS pii_registry_requires_approval ON pii_registry(requires_approval);

-- =============================================================================
-- Resource Profiles
-- =============================================================================

-- Stores resource limit configurations
CREATE TABLE IF NOT EXISTS resource_profiles (
    tier TEXT PRIMARY KEY,             -- "low", "medium", "high"
    memory_mb INTEGER NOT NULL,
    cpu_shares INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    max_concurrency INTEGER DEFAULT 3,
    description TEXT
);

-- Insert default profiles
INSERT OR REPLACE INTO resource_profiles (tier, memory_mb, cpu_shares, timeout_seconds, max_concurrency, description)
VALUES
    ('low', 256, 512, 300, 5, 'Lightweight tasks, quick operations'),
    ('medium', 512, 1024, 600, 3, 'Standard tasks, moderate processing'),
    ('high', 2048, 2048, 1800, 1, 'Heavy processing, complex operations');

-- =============================================================================
-- Agent Instances
-- =============================================================================

-- Tracks running agent instances
CREATE TABLE IF NOT EXISTS agent_instances (
    id TEXT PRIMARY KEY,
    definition_id TEXT NOT NULL,
    container_id TEXT,
    status TEXT DEFAULT 'pending',     -- "pending", "running", "completed", "failed", "cancelled"
    task_description TEXT,
    spawned_by TEXT NOT NULL,          -- Matrix user ID
    started_at INTEGER,                -- Unix timestamp (milliseconds)
    completed_at INTEGER,              -- Unix timestamp (milliseconds)
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (definition_id) REFERENCES agent_definitions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_instances_definition_id ON agent_instances(definition_id);
CREATE INDEX IF NOT EXISTS idx_agent_instances_status ON agent_instances(status);
CREATE INDEX IF NOT EXISTS idx_agent_instances_spawned_by ON agent_instances(spawned_by);
CREATE INDEX IF NOT EXISTS idx_agent_instances_started_at ON agent_instances(started_at);

-- =============================================================================
-- Default Skills
-- =============================================================================

-- Insert built-in skills
INSERT OR IGNORE INTO skill_registry (id, name, description, category, created_at)
VALUES
    ('browser_navigate', 'Browser Navigation', 'Navigate to URLs and extract content', 'research', strftime('%s', 'now') * 1000),
    ('form_filler', 'Form Filler', 'Fill web forms with provided data', 'automation', strftime('%s', 'now') * 1000),
    ('pdf_generator', 'PDF Generator', 'Create PDF documents from templates', 'document', strftime('%s', 'now') * 1000),
    ('template_filler', 'Template Filler', 'Fill document templates with data', 'document', strftime('%s', 'now') * 1000),
    ('email_sender', 'Email Sender', 'Send emails via configured SMTP', 'communication', strftime('%s', 'now') * 1000),
    ('calendar', 'Calendar Manager', 'Manage calendar events and scheduling', 'productivity', strftime('%s', 'now') * 1000),
    ('web_scraper', 'Web Scraper', 'Extract data from web pages', 'research', strftime('%s', 'now') * 1000),
    ('data_processor', 'Data Processor', 'Process and transform data files', 'data', strftime('%s', 'now') * 1000);

-- =============================================================================
-- Default PII Fields
-- =============================================================================

-- Insert common PII fields
INSERT OR IGNORE INTO pii_registry (id, name, description, sensitivity, requires_approval, created_at)
VALUES
    ('client_name', 'Client Name', 'Full name of the client', 'low', 0, strftime('%s', 'now') * 1000),
    ('client_email', 'Client Email', 'Email address of the client', 'medium', 1, strftime('%s', 'now') * 1000),
    ('client_phone', 'Client Phone', 'Phone number of the client', 'medium', 1, strftime('%s', 'now') * 1000),
    ('client_address', 'Client Address', 'Physical address of the client', 'medium', 1, strftime('%s', 'now') * 1000),
    ('client_ssn', 'Client SSN', 'Social Security Number', 'critical', 1, strftime('%s', 'now') * 1000),
    ('client_dob', 'Client DOB', 'Date of birth', 'high', 1, strftime('%s', 'now') * 1000),
    ('contract_value', 'Contract Value', 'Financial value of contracts', 'high', 1, strftime('%s', 'now') * 1000),
    ('account_number', 'Account Number', 'Bank or account numbers', 'critical', 1, strftime('%s', 'now') * 1000),
    ('company_name', 'Company Name', 'Name of the organization', 'low', 0, strftime('%s', 'now') * 1000),
    ('case_number', 'Case Number', 'Legal case identifiers', 'medium', 0, strftime('%s', 'now') * 1000);
