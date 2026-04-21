package invite

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	// inviteCodeLen is the minimum length for generated invite codes.
	inviteCodeLen = 24
)

// base62 alphabet for invite code generation.
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// InviteRecord represents a persisted invite. JSON tags use snake_case per TS contract.
type InviteRecord struct {
	ID              string        `json:"id"`
	Code            string        `json:"code"`
	Role            Role          `json:"role"`
	CreatedBy       string        `json:"created_by"`
	CreatedAt       time.Time     `json:"created_at"`
	ExpiresAt       *time.Time    `json:"expires_at,omitempty"`
	MaxUses         int           `json:"max_uses"`
	UseCount        int           `json:"use_count"`
	Status          InviteStatus  `json:"status"`
	WelcomeMessage  string        `json:"welcome_message,omitempty"`
}

// InviteStore persists invite records via a shared *sql.DB.
type InviteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewInviteStore opens an InviteStore against db, creating the schema if needed.
func NewInviteStore(db *sql.DB) (*InviteStore, error) {
	s := &InviteStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize invite store schema: %w", err)
	}
	return s, nil
}

// initSchema creates the invites table if it does not already exist.
func (s *InviteStore) initSchema() error {
	const ddl = `
	CREATE TABLE IF NOT EXISTS invites (
		id               TEXT PRIMARY KEY,
		code             TEXT UNIQUE NOT NULL,
		role             TEXT NOT NULL,
		created_by       TEXT NOT NULL,
		created_at       DATETIME NOT NULL,
		expires_at       DATETIME,
		max_uses         INTEGER NOT NULL DEFAULT 0,
		use_count        INTEGER NOT NULL DEFAULT 0,
		status           TEXT NOT NULL DEFAULT 'active',
		welcome_message  TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_invites_code   ON invites(code);
	CREATE INDEX IF NOT EXISTS idx_invites_status ON invites(status);
	CREATE INDEX IF NOT EXISTS idx_invites_created_by ON invites(created_by);
	`
	_, err := s.db.Exec(ddl)
	return err
}

// GetInvite returns an invite by ID.
func (s *InviteStore) GetInvite(id string) (*InviteRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queryInvite("WHERE id = ?", id)
}

// GetInviteByCode returns an invite by its code.
func (s *InviteStore) GetInviteByCode(code string) (*InviteRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queryInvite("WHERE code = ?", code)
}

// ListInvites returns all invites ordered by created_at descending.
func (s *InviteStore) ListInvites() ([]*InviteRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, code, role, created_by, created_at, expires_at,
		       max_uses, use_count, status, welcome_message
		FROM invites
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list invites: %w", err)
	}
	defer rows.Close()

	var invites []*InviteRecord
	for rows.Next() {
		inv, err := scanInviteRow(rows)
		if err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate invites: %w", err)
	}
	return invites, nil
}

// CreateInvite inserts a new invite record.
// Sets CreatedAt to now if zero. Generates a code and ID if empty.
func (s *InviteStore) CreateInvite(inv *InviteRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now().UTC()
	}
	if inv.ID == "" {
		inv.ID = "inv_" + mustGenerateID(16)
	}
	if inv.Code == "" {
		code, err := GenerateInviteCode()
		if err != nil {
			return fmt.Errorf("failed to generate invite code: %w", err)
		}
		inv.Code = code
	}
	if inv.Status == "" {
		inv.Status = StatusActive
	}

	_, err := s.db.Exec(`
		INSERT INTO invites
			(id, code, role, created_by, created_at, expires_at,
			 max_uses, use_count, status, welcome_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		inv.ID, inv.Code, string(inv.Role), inv.CreatedBy, inv.CreatedAt,
		inv.ExpiresAt, inv.MaxUses, inv.UseCount, string(inv.Status),
		inv.WelcomeMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to create invite: %w", err)
	}
	return nil
}

// RevokeInvite sets status to "revoked" for the given invite ID.
func (s *InviteStore) RevokeInvite(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`
		UPDATE invites SET status = ? WHERE id = ? AND status = ?
	`, string(StatusRevoked), id, string(StatusActive))
	if err != nil {
		return fmt.Errorf("failed to revoke invite: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("invite not found or not active: %s", id)
	}
	return nil
}

// IncrementUseCount atomically increments use_count and updates status
// to "exhausted" when max_uses > 0 and use_count reaches max_uses,
// or "used" when max_uses == 1.
func (s *InviteStore) IncrementUseCount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var useCount, maxUses int
	var status string
	err := s.db.QueryRow(`
		SELECT use_count, max_uses, status FROM invites WHERE id = ?
	`, id).Scan(&useCount, &maxUses, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invite not found: %s", id)
		}
		return fmt.Errorf("failed to query invite: %w", err)
	}

	useCount++
	newStatus := InviteStatus(status)
	if maxUses > 0 && useCount >= maxUses {
		newStatus = StatusExhausted
	} else if maxUses == 1 {
		newStatus = StatusUsed
	}

	_, err = s.db.Exec(`
		UPDATE invites SET use_count = ?, status = ? WHERE id = ?
	`, useCount, string(newStatus), id)
	if err != nil {
		return fmt.Errorf("failed to increment use count: %w", err)
	}
	return nil
}

// GenerateInviteCode produces a cryptographically random base62 string of
// inviteCodeLen characters (minimum 16) using crypto/rand.
func GenerateInviteCode() (string, error) {
	b := make([]byte, inviteCodeLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Alphabet))))
		if err != nil {
			return "", fmt.Errorf("crypto/rand failed: %w", err)
		}
		b[i] = base62Alphabet[n.Int64()]
	}
	return string(b), nil
}

// ParseExpiration parses a human-friendly expiration string into a *time.Time.
// Supported values: "1h", "6h", "1d", "7d", "30d", "never".
// Returns nil time pointer for "never".
func ParseExpiration(s string) (*time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1h":
		return expirationFromNow(1 * time.Hour), nil
	case "6h":
		return expirationFromNow(6 * time.Hour), nil
	case "1d":
		return expirationFromNow(24 * time.Hour), nil
	case "7d":
		return expirationFromNow(7 * 24 * time.Hour), nil
	case "30d":
		return expirationFromNow(30 * 24 * time.Hour), nil
	case "never":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported expiration: %q (valid: 1h, 6h, 1d, 7d, 30d, never)", s)
	}
}

func expirationFromNow(d time.Duration) *time.Time {
	t := time.Now().UTC().Add(d)
	return &t
}

// queryInvite is a helper that selects a single invite with an arbitrary WHERE clause.
func (s *InviteStore) queryInvite(whereClause string, args ...interface{}) (*InviteRecord, error) {
	query := `
		SELECT id, code, role, created_by, created_at, expires_at,
		       max_uses, use_count, status, welcome_message
		FROM invites
	` + whereClause

	row := s.db.QueryRow(query, args...)
	inv, err := scanInviteRowSingle(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invite not found")
		}
		return nil, fmt.Errorf("failed to query invite: %w", err)
	}
	return inv, nil
}

// scanInviteRow scans a single invite from the current rows position.
func scanInviteRow(rows *sql.Rows) (*InviteRecord, error) {
	var inv InviteRecord
	var roleStr, statusStr string
	var expiresAt sql.NullTime

	err := rows.Scan(
		&inv.ID, &inv.Code, &roleStr, &inv.CreatedBy, &inv.CreatedAt,
		&expiresAt, &inv.MaxUses, &inv.UseCount, &statusStr,
		&inv.WelcomeMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan invite: %w", err)
	}

	inv.Role = Role(roleStr)
	inv.Status = InviteStatus(statusStr)
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	return &inv, nil
}

// scanInviteRowSingle scans a single invite from a QueryRow.
func scanInviteRowSingle(row *sql.Row) (*InviteRecord, error) {
	var inv InviteRecord
	var roleStr, statusStr string
	var expiresAt sql.NullTime

	err := row.Scan(
		&inv.ID, &inv.Code, &roleStr, &inv.CreatedBy, &inv.CreatedAt,
		&expiresAt, &inv.MaxUses, &inv.UseCount, &statusStr,
		&inv.WelcomeMessage,
	)
	if err != nil {
		return nil, err
	}

	inv.Role = Role(roleStr)
	inv.Status = InviteStatus(statusStr)
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	return &inv, nil
}

// ToJSON serializes an InviteRecord to JSON bytes.
func (inv *InviteRecord) ToJSON() ([]byte, error) {
	return json.Marshal(inv)
}

// InviteRecordFromJSON deserializes an InviteRecord from JSON bytes.
func InviteRecordFromJSON(data []byte) (*InviteRecord, error) {
	var inv InviteRecord
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal invite record: %w", err)
	}
	return &inv, nil
}

// mustGenerateID produces a hex-encoded random string of the given byte length.
// Panics on failure (mirrors existing securerandom.MustID pattern).
func mustGenerateID(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return fmt.Sprintf("%x", b)
}
