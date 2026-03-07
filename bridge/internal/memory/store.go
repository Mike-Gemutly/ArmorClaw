package memory

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/metrics"
	_ "modernc.org/sqlite"
)

type Message struct {
	ID        string
	RoomID    string
	Role      string
	Content   string
	CreatedAt time.Time
	Metadata  map[string]string
}

type Store struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

type StoreConfig struct {
	Path string
}

func NewStore(cfg StoreConfig) (*Store, error) {
	if cfg.Path == "" {
		cfg.Path = ":memory:"
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:   db,
		path: cfg.Path,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *Store) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			room_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_room ON messages(room_id)`,
		`CREATE TABLE IF NOT EXISTS contexts (
			id TEXT PRIMARY KEY,
			room_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(room_id, key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contexts_room ON contexts(room_id)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}
	return nil
}

func (s *Store) AddMessage(msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics.RecordMemoryOperation("insert")
	_, err := s.db.Exec(
		`INSERT INTO messages (id, room_id, role, content, created_at, metadata)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.RoomID, msg.Role, msg.Content, msg.CreatedAt, serializeMetadata(msg.Metadata),
	)
	return err
}

func (s *Store) GetMessages(roomID string, limit int) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics.RecordMemoryOperation("select")

	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, room_id, role, content, created_at, metadata
		 FROM messages WHERE room_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var metadataStr sql.NullString
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.Role, &msg.Content, &msg.CreatedAt, &metadataStr); err != nil {
			return nil, err
		}
		if metadataStr.Valid {
			msg.Metadata = deserializeMetadata(metadataStr.String)
		}
		messages = append(messages, msg)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (s *Store) SetContext(roomID, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics.RecordMemoryOperation("upsert")
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO contexts (id, room_id, key, value, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(room_id, key) DO UPDATE SET value = ?, updated_at = ?`,
		generateID(), roomID, key, value, now, now, value, now,
	)
	return err
}

func (s *Store) GetContext(roomID, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics.RecordMemoryOperation("select")
	var value string
	err := s.db.QueryRow(
		`SELECT value FROM contexts WHERE room_id = ? AND key = ?`,
		roomID, key,
	).Scan(&value)
	if err != nil {
		return "", false
	}
	return value, true
}

func (s *Store) GetAllContext(roomID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics.RecordMemoryOperation("select")
	rows, err := s.db.Query(
		`SELECT key, value FROM contexts WHERE room_id = ?`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ctx := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		ctx[key] = value
	}
	return ctx, nil
}

func (s *Store) DeleteContext(roomID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics.RecordMemoryOperation("delete")
	_, err := s.db.Exec(
		`DELETE FROM contexts WHERE room_id = ? AND key = ?`,
		roomID, key,
	)
	return err
}

func (s *Store) ClearRoom(roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics.RecordMemoryOperation("delete")
	_, err := s.db.Exec(`DELETE FROM messages WHERE room_id = ?`, roomID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM contexts WHERE room_id = ?`, roomID)
	return err
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func (s *Store) PruneMessages(olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics.RecordMemoryOperation("delete")
	result, err := s.db.Exec(
		`DELETE FROM messages WHERE created_at < ?`,
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) Checkpoint() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func serializeMetadata(m map[string]string) string {
	if m == nil {
		return ""
	}
	result := ""
	for k, v := range m {
		if result != "" {
			result += "|"
		}
		result += k + "=" + v
	}
	return result
}

func deserializeMetadata(str string) map[string]string {
	if str == "" {
		return nil
	}
	m := make(map[string]string)
	pairs := splitPairs(str)
	for _, pair := range pairs {
		k, v := splitKV(pair)
		if k != "" {
			m[k] = v
		}
	}
	return m
}

func splitPairs(s string) []string {
	var pairs []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			pairs = append(pairs, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		pairs = append(pairs, s[start:])
	}
	return pairs
}

func splitKV(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
