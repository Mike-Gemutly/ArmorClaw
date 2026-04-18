// Package team provides the SQLCipher-backed store for multi-agent teams.
//
// The store manages three tables (teams, team_members, team_roles) and exposes
// CRUD methods. All write operations use transactions for concurrent safety.
// Optimistic locking is provided via a version column on the teams table.
package team

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrTeamNotFound    = errors.New("team: team not found")
	ErrVersionConflict = errors.New("team: version conflict")
	ErrMemberNotFound  = errors.New("team: member not found")
)

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

const schemaDDL = `
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT DEFAULT '',
    shared_context TEXT DEFAULT '',
    lifecycle_state TEXT NOT NULL DEFAULT 'active',
    budgets_json TEXT DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id TEXT NOT NULL REFERENCES teams(id),
    agent_id TEXT NOT NULL,
    role_name TEXT NOT NULL,
    allowed_tools TEXT DEFAULT '[]',
    allowed_secret_prefixes TEXT DEFAULT '[]',
    browser_context_id TEXT DEFAULT '',
    priority INTEGER DEFAULT 0,
    PRIMARY KEY (team_id, agent_id)
);

CREATE TABLE IF NOT EXISTS team_roles (
    team_id TEXT NOT NULL REFERENCES teams(id),
    role_name TEXT NOT NULL,
    capabilities_json TEXT NOT NULL DEFAULT '{}',
    description TEXT DEFAULT '',
    PRIMARY KEY (team_id, role_name)
);
`

// ---------------------------------------------------------------------------
// TeamStore
// ---------------------------------------------------------------------------

// TeamStore is a SQLCipher-backed persistence layer for teams.
type TeamStore struct {
	db *sql.DB
}

// NewTeamStore opens (or creates) a SQLCipher database at dbPath using the
// given passphrase. Pass an empty passphrase to use plain SQLite. The schema
// is created automatically if it does not exist.
func NewTeamStore(dbPath string, passphrase string) (*TeamStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("team: create db directory: %w", err)
	}

	var dsn string
	if passphrase != "" {
		dsn = fmt.Sprintf(
			"file:%s?_pragma_key=%s&_pragma_cipher_page_size=4096&_pragma_kdf_iter=256000&_pragma_cipher_hmac_algorithm=HMAC_SHA512&_pragma_cipher_kdf_algorithm=PBKDF2_HMAC_SHA512&_foreign_keys=ON",
			dbPath, passphrase,
		)
	} else {
		dsn = fmt.Sprintf("file:%s?_foreign_keys=ON", dbPath)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("team: open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("team: ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("team: set WAL mode: %w", err)
	}

	if _, err := db.Exec(schemaDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("team: schema migration: %w", err)
	}

	return &TeamStore{db: db}, nil
}

// NewTeamStoreFromDB wraps an existing *sql.DB. The caller is responsible for
// ensuring the schema is applied (via InitSchema) and for closing the DB.
func NewTeamStoreFromDB(db *sql.DB) (*TeamStore, error) {
	if _, err := db.Exec(schemaDDL); err != nil {
		return nil, fmt.Errorf("team: schema migration: %w", err)
	}
	return &TeamStore{db: db}, nil
}

// Close closes the underlying database connection.
func (s *TeamStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB returns the underlying *sql.DB.
func (s *TeamStore) DB() *sql.DB { return s.db }

// ---------------------------------------------------------------------------
// CRUD: Teams
// ---------------------------------------------------------------------------

// CreateTeam inserts a new team after validation. The version is set to 1.
func (s *TeamStore) CreateTeam(ctx context.Context, t *Team) error {
	if err := t.Validate(); err != nil {
		return err
	}

	budgetsJSON := "{}"
	if t.Budgets != nil {
		b, err := json.Marshal(t.Budgets)
		if err != nil {
			return fmt.Errorf("team: marshal budgets: %w", err)
		}
		budgetsJSON = string(b)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("team: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO teams (id, name, template_id, shared_context, lifecycle_state, budgets_json, version)
		VALUES (?, ?, ?, ?, ?, ?, 1)`,
		t.ID, t.Name, t.TemplateID, t.SharedContext,
		string(t.LifecycleState), budgetsJSON,
	)
	if err != nil {
		return fmt.Errorf("team: insert team: %w", err)
	}

	t.Version = 1
	return tx.Commit()
}

// GetTeam fetches a team by ID including all its members.
func (s *TeamStore) GetTeam(ctx context.Context, id string) (*Team, error) {
	var t Team
	var lifecycle string
	var budgetsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, template_id, shared_context, lifecycle_state, budgets_json, version
		FROM teams WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.TemplateID, &t.SharedContext, &lifecycle, &budgetsJSON, &t.Version)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTeamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("team: query team: %w", err)
	}

	t.LifecycleState = LifecycleState(lifecycle)

	if budgetsJSON != "" && budgetsJSON != "{}" {
		var budgets TeamBudgets
		if err := json.Unmarshal([]byte(budgetsJSON), &budgets); err == nil {
			t.Budgets = &budgets
		}
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT team_id, agent_id, role_name, allowed_tools, allowed_secret_prefixes, browser_context_id, priority
		FROM team_members WHERE team_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("team: query members: %w", err)
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var m TeamMember
		var toolsJSON, prefixesJSON string
		if err := rows.Scan(&m.TeamID, &m.AgentID, &m.RoleName, &toolsJSON, &prefixesJSON, &m.BrowserContextID, &m.Priority); err != nil {
			return nil, fmt.Errorf("team: scan member: %w", err)
		}
		_ = json.Unmarshal([]byte(toolsJSON), &m.AllowedTools)
		_ = json.Unmarshal([]byte(prefixesJSON), &m.AllowedSecretPrefixes)
		members = append(members, m)
	}

	t.Members = members
	return &t, nil
}

// ListTeams returns all teams ordered by creation time (without members).
func (s *TeamStore) ListTeams(ctx context.Context) ([]Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, template_id, shared_context, lifecycle_state, budgets_json, version
		FROM teams ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("team: list teams: %w", err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		var lifecycle string
		var budgetsJSON string
		if err := rows.Scan(&t.ID, &t.Name, &t.TemplateID, &t.SharedContext, &lifecycle, &budgetsJSON, &t.Version); err != nil {
			return nil, fmt.Errorf("team: scan team: %w", err)
		}
		t.LifecycleState = LifecycleState(lifecycle)

		if budgetsJSON != "" && budgetsJSON != "{}" {
			var budgets TeamBudgets
			if err := json.Unmarshal([]byte(budgetsJSON), &budgets); err == nil {
				t.Budgets = &budgets
			}
		}

		teams = append(teams, t)
	}

	return teams, rows.Err()
}

// UpdateTeam applies optimistic-locking updates. Returns ErrVersionConflict
// when the stored version does not match t.Version.
func (s *TeamStore) UpdateTeam(ctx context.Context, t *Team) error {
	if err := t.Validate(); err != nil {
		return err
	}

	budgetsJSON := "{}"
	if t.Budgets != nil {
		b, err := json.Marshal(t.Budgets)
		if err != nil {
			return fmt.Errorf("team: marshal budgets: %w", err)
		}
		budgetsJSON = string(b)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("team: begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE teams
		SET name = ?, template_id = ?, shared_context = ?, lifecycle_state = ?,
		    budgets_json = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND version = ?`,
		t.Name, t.TemplateID, t.SharedContext, string(t.LifecycleState),
		budgetsJSON, t.ID, t.Version,
	)
	if err != nil {
		return fmt.Errorf("team: update team: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("team: rows affected: %w", err)
	}
	if n == 0 {
		return ErrVersionConflict
	}

	t.Version++
	return tx.Commit()
}

// ---------------------------------------------------------------------------
// CRUD: Members
// ---------------------------------------------------------------------------

// AddMember adds a member to a team after validating the team exists and is
// not dissolved.
func (s *TeamStore) AddMember(ctx context.Context, m *TeamMember) error {
	if err := m.Validate(); err != nil {
		return err
	}

	toolsJSON, err := json.Marshal(m.AllowedTools)
	if err != nil {
		return fmt.Errorf("team: marshal allowed_tools: %w", err)
	}
	prefixesJSON, err := json.Marshal(m.AllowedSecretPrefixes)
	if err != nil {
		return fmt.Errorf("team: marshal allowed_secret_prefixes: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("team: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Verify team exists and is not dissolved.
	var state string
	err = tx.QueryRow("SELECT lifecycle_state FROM teams WHERE id = ?", m.TeamID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTeamNotFound
	}
	if err != nil {
		return fmt.Errorf("team: check team: %w", err)
	}
	if LifecycleState(state) == LifecycleDissolved {
		return fmt.Errorf("team: cannot add member to dissolved team")
	}

	_, err = tx.Exec(`
		INSERT INTO team_members (team_id, agent_id, role_name, allowed_tools, allowed_secret_prefixes, browser_context_id, priority)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.TeamID, m.AgentID, m.RoleName, string(toolsJSON), string(prefixesJSON),
		m.BrowserContextID, m.Priority,
	)
	if err != nil {
		return fmt.Errorf("team: insert member: %w", err)
	}

	return tx.Commit()
}

// RemoveMember removes a member. If zero members remain, the team is
// automatically dissolved.
func (s *TeamStore) RemoveMember(ctx context.Context, teamID, agentID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("team: begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec("DELETE FROM team_members WHERE team_id = ? AND agent_id = ?", teamID, agentID)
	if err != nil {
		return fmt.Errorf("team: delete member: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}

	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM team_members WHERE team_id = ?", teamID).Scan(&count)
	if err != nil {
		return fmt.Errorf("team: count members: %w", err)
	}

	if count == 0 {
		_, err = tx.Exec(`
			UPDATE teams SET lifecycle_state = 'dissolved', updated_at = CURRENT_TIMESTAMP, version = version + 1
			WHERE id = ?`, teamID)
		if err != nil {
			return fmt.Errorf("team: auto-dissolve: %w", err)
		}
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Dissolve
// ---------------------------------------------------------------------------

// DissolveTeam marks a team as dissolved. Data is preserved for audit.
// Idempotent: calling on an already-dissolved team is a no-op.
func (s *TeamStore) DissolveTeam(ctx context.Context, teamID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("team: begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE teams SET lifecycle_state = 'dissolved', updated_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = ? AND lifecycle_state != 'dissolved'`, teamID)
	if err != nil {
		return fmt.Errorf("team: dissolve team: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		var exists bool
		err = tx.QueryRow("SELECT 1 FROM teams WHERE id = ?", teamID).Scan(&exists)
		if err != nil {
			return ErrTeamNotFound
		}
		return nil
	}

	return tx.Commit()
}
