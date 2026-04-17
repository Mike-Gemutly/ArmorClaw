-- Secretary Database Schema
-- Version: 1.0.0
-- Created: 2026-03-13

-- =============================================================================
-- Task Templates
-- =============================================================================

-- Stores reusable task definitions with steps and variable schemas
CREATE TABLE IF NOT EXISTS task_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    steps TEXT NOT NULL,           -- JSON array of workflow steps
    variables TEXT,                -- JSON schema for variables
    pii_refs TEXT,                 -- JSON array of PII field references
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    is_active INTEGER DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_templates_active ON task_templates(is_active);

-- =============================================================================
-- Workflows
-- =============================================================================

-- Stores running workflow instances
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    template_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending',  -- pending, running, completed, failed, cancelled
    variables TEXT,                 -- JSON resolved variables
    current_step INTEGER DEFAULT 0,
    agent_ids TEXT,                 -- JSON array of spawned agent IDs
    started_at INTEGER,
    completed_at INTEGER,
    error_message TEXT,
    created_by TEXT NOT NULL,
    FOREIGN KEY (template_id) REFERENCES task_templates(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_template ON workflows(template_id);

-- =============================================================================
-- Approval Policies
-- =============================================================================

-- Stores approval rules for PII-sensitive operations
CREATE TABLE IF NOT EXISTS approval_policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    pii_fields TEXT NOT NULL,
    auto_approve INTEGER DEFAULT 0,
    delegate_to TEXT,
    conditions TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

-- =============================================================================
-- Notification Channels
-- =============================================================================

-- Stores notification destinations for users
CREATE TABLE IF NOT EXISTS notification_channels (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    channel_type TEXT NOT NULL,     -- matrix, push, email
    destination TEXT NOT NULL,      -- room ID, device token, email
    event_types TEXT NOT NULL,      -- JSON array of event types
    is_active INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_user ON notification_channels(user_id);

-- =============================================================================
-- Scheduled Tasks
-- =============================================================================

-- Stores scheduled task executions
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    next_run INTEGER,
    last_run INTEGER,
    is_active INTEGER DEFAULT 1,
    created_by TEXT NOT NULL,
    FOREIGN KEY (template_id) REFERENCES task_templates(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scheduled_next_run ON scheduled_tasks(next_run);

CREATE TABLE IF NOT EXISTS user_contacts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    company TEXT,
    email TEXT,
    phone TEXT,
    notes TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    is_active INTEGER DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_contacts_user ON user_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON user_contacts(name);
CREATE INDEX IF NOT EXISTS idx_contacts_active ON user_contacts(is_active);

CREATE TABLE IF NOT EXISTS contact_details (
    id TEXT PRIMARY KEY,
    contact_id TEXT NOT NULL,
    contact_data BLOB NOT NULL,
    encryption_key_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (contact_id) REFERENCES user_contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_contact_details_contact ON contact_details(contact_id);

CREATE TABLE IF NOT EXISTS contact_relationships (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    contact_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    last_contacted_at INTEGER,
    notes TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    is_active INTEGER DEFAULT 1,
    FOREIGN KEY (contact_id) REFERENCES user_contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_relationships_user ON contact_relationships(user_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON contact_relationships(relationship_type);
CREATE INDEX IF NOT EXISTS idx_relationships_last_contacted ON contact_relationships(last_contacted_at);
CREATE INDEX IF NOT EXISTS idx_relationships_active ON contact_relationships(is_active);

-- =============================================================================
-- Email Pipeline Migrations (v2)
-- =============================================================================

-- Add trigger column to task_templates for event-driven dispatch
ALTER TABLE task_templates ADD COLUMN trigger TEXT DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_templates_trigger ON task_templates(trigger);

-- Add default_definition_id to task_templates for one-shot email workflows
ALTER TABLE task_templates ADD COLUMN default_definition_id TEXT DEFAULT '';

-- Add one_shot column to scheduled_tasks for fire-once tasks
ALTER TABLE scheduled_tasks ADD COLUMN one_shot INTEGER DEFAULT 0;

-- Add trigger column to scheduled_tasks for event-driven dispatch
ALTER TABLE scheduled_tasks ADD COLUMN trigger TEXT DEFAULT '';
