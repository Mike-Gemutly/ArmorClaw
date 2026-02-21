// Package crypto provides cryptographic interfaces for E2EE support
package crypto

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// KeystoreBackedStore implements Store using the encrypted keystore database
// This provides persistent, encrypted storage for Megolm session keys
type KeystoreBackedStore struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// NewKeystoreBackedStore creates a new crypto store backed by SQLCipher
// The dbPath should point to the same encrypted database used by the keystore
func NewKeystoreBackedStore(dbPath string) (*KeystoreBackedStore, error) {
	// Open SQLCipher database
	// Note: The key must be set via PRAGMA key before any operations
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open crypto store database: %w", err)
	}

	store := &KeystoreBackedStore{
		db:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize crypto store schema: %w", err)
	}

	return store, nil
}

// NewKeystoreBackedStoreWithDB creates a store from an existing database connection
// Use this when sharing the same database with the keystore
func NewKeystoreBackedStoreWithDB(db *sql.DB) (*KeystoreBackedStore, error) {
	store := &KeystoreBackedStore{
		db: db,
	}

	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize crypto store schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables for crypto storage
func (s *KeystoreBackedStore) initSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS inbound_group_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id TEXT NOT NULL,
			sender_key TEXT NOT NULL,
			session_id TEXT NOT NULL,
			session_key BLOB NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(room_id, sender_key, session_id)
		);

		CREATE INDEX IF NOT EXISTS idx_inbound_sessions_room
			ON inbound_group_sessions(room_id);

		CREATE INDEX IF NOT EXISTS idx_inbound_sessions_sender
			ON inbound_group_sessions(sender_key);
	`)
	return err
}

// AddInboundGroupSession stores an inbound Megolm session
func (s *KeystoreBackedStore) AddInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string, sessionKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encode session key as base64 for storage
	encodedKey := base64.StdEncoding.EncodeToString(sessionKey)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inbound_group_sessions (room_id, sender_key, session_id, session_key, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(room_id, sender_key, session_id) DO UPDATE SET
			session_key = excluded.session_key,
			updated_at = CURRENT_TIMESTAMP
	`, roomID, senderKey, sessionID, encodedKey)

	if err != nil {
		return fmt.Errorf("failed to store inbound group session: %w", err)
	}

	return nil
}

// GetInboundGroupSession retrieves an inbound Megolm session
func (s *KeystoreBackedStore) GetInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var encodedKey string
	err := s.db.QueryRowContext(ctx, `
		SELECT session_key FROM inbound_group_sessions
		WHERE room_id = ? AND sender_key = ? AND session_id = ?
	`, roomID, senderKey, sessionID).Scan(&encodedKey)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve inbound group session: %w", err)
	}

	sessionKey, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session key: %w", err)
	}

	return sessionKey, nil
}

// HasInboundGroupSession checks if a session exists
func (s *KeystoreBackedStore) HasInboundGroupSession(ctx context.Context, roomID, senderKey, sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inbound_group_sessions
		WHERE room_id = ? AND sender_key = ? AND session_id = ?
	`, roomID, senderKey, sessionID).Scan(&count)

	return err == nil && count > 0
}

// Clear removes all stored sessions
func (s *KeystoreBackedStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM inbound_group_sessions`)
	if err != nil {
		return fmt.Errorf("failed to clear crypto store: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *KeystoreBackedStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetStats returns statistics about the crypto store
func (s *KeystoreBackedStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count sessions
	var sessionCount int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM inbound_group_sessions`).Scan(&sessionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get session count: %w", err)
	}
	stats["session_count"] = sessionCount

	// Count unique rooms
	var roomCount int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT room_id) FROM inbound_group_sessions`).Scan(&roomCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get room count: %w", err)
	}
	stats["room_count"] = roomCount

	// Count unique senders
	var senderCount int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT sender_key) FROM inbound_group_sessions`).Scan(&senderCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender count: %w", err)
	}
	stats["sender_count"] = senderCount

	return stats, nil
}

// DeleteSessionsForRoom removes all sessions for a specific room
func (s *KeystoreBackedStore) DeleteSessionsForRoom(ctx context.Context, roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM inbound_group_sessions WHERE room_id = ?`, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions for room: %w", err)
	}

	return nil
}

// ListSessions returns all session IDs for a room
func (s *KeystoreBackedStore) ListSessions(ctx context.Context, roomID string) ([]SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, sender_key, created_at
		FROM inbound_group_sessions
		WHERE room_id = ?
		ORDER BY created_at DESC
	`, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var info SessionInfo
		info.RoomID = roomID
		if err := rows.Scan(&info.SessionID, &info.SenderKey, &info.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, info)
	}

	return sessions, nil
}

// SessionInfo contains metadata about a stored session
type SessionInfo struct {
	RoomID    string
	SessionID string
	SenderKey string
	CreatedAt string
}

// Verify the implementation satisfies the Store interface
var _ Store = (*KeystoreBackedStore)(nil)
var _ Store = (*MemoryStore)(nil)
