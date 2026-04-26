//go:build !cgo

package security

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func init() {
	isEncrypted = false
}

type plaintextSessionStore struct {
	db *sql.DB
}

func newCipherSessionStore(dbPath, passphrase string) (SessionStore, error) {
	if passphrase == "" {
		return nil, ErrEmptyPassphrase
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := initPlaintextSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &plaintextSessionStore{db: db}, nil
}

func (s *plaintextSessionStore) CreateSession(session Session) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_agent, cookies, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		session.ID, session.UserAgent, session.Cookies, session.ExpiresAt, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (s *plaintextSessionStore) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(
		"SELECT id, user_agent, cookies, expires_at FROM sessions WHERE id = ?", id,
	)

	var session Session
	err := row.Scan(&session.ID, &session.UserAgent, &session.Cookies, &session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionNotFound, err)
	}

	return &session, nil
}

func (s *plaintextSessionStore) CloseSession(id string) error {
	res, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (s *plaintextSessionStore) ListSessions() ([]Session, error) {
	rows, err := s.db.Query("SELECT id, user_agent, cookies, expires_at FROM sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.UserAgent, &sess.Cookies, &sess.ExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}

	return sessions, rows.Err()
}

func (s *plaintextSessionStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initPlaintextSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_agent TEXT,
			cookies BLOB,
			expires_at INTEGER,
			created_at INTEGER
		)
	`)
	return err
}
