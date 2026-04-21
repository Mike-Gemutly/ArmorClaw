package trust

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DeviceRecord represents a persisted device. JSON tags use snake_case per TS contract.
type DeviceRecord struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Platform   string     `json:"platform"`
	TrustState TrustState `json:"trust_state"`
	LastSeen   time.Time  `json:"last_seen"`
	FirstSeen  time.Time  `json:"first_seen"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	IsCurrent  bool       `json:"is_current"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
}

// DeviceStore persists device records via a shared *sql.DB.
type DeviceStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDeviceStore opens a DeviceStore against db, creating the schema if needed.
func NewDeviceStore(db *sql.DB) (*DeviceStore, error) {
	s := &DeviceStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize device store schema: %w", err)
	}
	return s, nil
}

// initSchema creates the devices table if it does not already exist.
func (s *DeviceStore) initSchema() error {
	const ddl = `
	CREATE TABLE IF NOT EXISTS devices (
		id          TEXT PRIMARY KEY,
		name        TEXT,
		type        TEXT,
		platform    TEXT,
		trust_state TEXT,
		last_seen   DATETIME,
		first_seen  DATETIME,
		ip_address  TEXT,
		user_agent  TEXT,
		is_current  BOOLEAN,
		verified_at DATETIME,
		created_at  DATETIME,
		updated_at  DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_devices_trust_state ON devices(trust_state);
	`
	_, err := s.db.Exec(ddl)
	return err
}

// GetDevice returns a device by ID.
func (s *DeviceStore) GetDevice(id string) (*DeviceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queryDevice("WHERE id = ?", id)
}

// ListDevices returns all devices ordered by last_seen descending.
func (s *DeviceStore) ListDevices() ([]*DeviceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, type, platform, trust_state,
		       last_seen, first_seen, ip_address, user_agent, is_current,
		       verified_at, created_at, updated_at
		FROM devices
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []*DeviceRecord
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate devices: %w", err)
	}
	return devices, nil
}

// CreateDevice inserts a new device record.
// Sets CreatedAt and UpdatedAt to now if they are zero.
func (s *DeviceStore) CreateDevice(d *DeviceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}

	_, err := s.db.Exec(`
		INSERT INTO devices
			(id, name, type, platform, trust_state,
			 last_seen, first_seen, ip_address, user_agent, is_current,
			 verified_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		d.ID, d.Name, d.Type, d.Platform, string(d.TrustState),
		d.LastSeen, d.FirstSeen, d.IPAddress, d.UserAgent, d.IsCurrent,
		d.VerifiedAt, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}
	return nil
}

// UpdateDevice updates all mutable fields of an existing device.
func (s *DeviceStore) UpdateDevice(d *DeviceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d.UpdatedAt = time.Now().UTC()

	result, err := s.db.Exec(`
		UPDATE devices SET
			name = ?, type = ?, platform = ?, trust_state = ?,
			last_seen = ?, first_seen = ?, ip_address = ?, user_agent = ?,
			is_current = ?, verified_at = ?, updated_at = ?
		WHERE id = ?
	`,
		d.Name, d.Type, d.Platform, string(d.TrustState),
		d.LastSeen, d.FirstSeen, d.IPAddress, d.UserAgent,
		d.IsCurrent, d.VerifiedAt, d.UpdatedAt, d.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found: %s", d.ID)
	}
	return nil
}

// UpdateTrustState updates only the trust_state and verified_at fields.
func (s *DeviceStore) UpdateTrustState(id string, state TrustState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var verifiedAt *time.Time
	if state == StateVerified {
		now := time.Now().UTC()
		verifiedAt = &now
	}

	result, err := s.db.Exec(`
		UPDATE devices SET trust_state = ?, verified_at = ?, updated_at = ?
		WHERE id = ?
	`, string(state), verifiedAt, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update trust state: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found: %s", id)
	}
	return nil
}

// ToJSON serializes a DeviceRecord to JSON bytes.
// Exported for consumers that need raw JSON bytes.
func (d *DeviceRecord) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// DeviceRecordFromJSON deserializes a DeviceRecord from JSON bytes.
func DeviceRecordFromJSON(data []byte) (*DeviceRecord, error) {
	var d DeviceRecord
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device record: %w", err)
	}
	return &d, nil
}

// queryDevice is a helper that selects a single device with an arbitrary WHERE clause.
func (s *DeviceStore) queryDevice(whereClause string, args ...interface{}) (*DeviceRecord, error) {
	query := `
		SELECT id, name, type, platform, trust_state,
		       last_seen, first_seen, ip_address, user_agent, is_current,
		       verified_at, created_at, updated_at
		FROM devices
	` + whereClause

	row := s.db.QueryRow(query, args...)
	d, err := scanRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to query device: %w", err)
	}
	return d, nil
}

// scanDevice scans a single device from the current row position.
func scanDevice(rows *sql.Rows) (*DeviceRecord, error) {
	var d DeviceRecord
	var trustStr string
	var verifiedAt sql.NullTime
	var createdAt, updatedAt sql.NullTime

	err := rows.Scan(
		&d.ID, &d.Name, &d.Type, &d.Platform, &trustStr,
		&d.LastSeen, &d.FirstSeen, &d.IPAddress, &d.UserAgent, &d.IsCurrent,
		&verifiedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan device: %w", err)
	}

	d.TrustState = TrustState(trustStr)
	if verifiedAt.Valid {
		d.VerifiedAt = &verifiedAt.Time
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		d.UpdatedAt = updatedAt.Time
	}
	return &d, nil
}

// scanRow scans a single device from a QueryRow.
func scanRow(row *sql.Row) (*DeviceRecord, error) {
	var d DeviceRecord
	var trustStr string
	var verifiedAt sql.NullTime
	var createdAt, updatedAt sql.NullTime

	err := row.Scan(
		&d.ID, &d.Name, &d.Type, &d.Platform, &trustStr,
		&d.LastSeen, &d.FirstSeen, &d.IPAddress, &d.UserAgent, &d.IsCurrent,
		&verifiedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.TrustState = TrustState(trustStr)
	if verifiedAt.Valid {
		d.VerifiedAt = &verifiedAt.Time
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		d.UpdatedAt = updatedAt.Time
	}
	return &d, nil
}
