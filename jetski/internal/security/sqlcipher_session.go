//go:build cgo

package security

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
	"golang.org/x/crypto/pbkdf2"
)

const (
	cipherPageSize     = 4096
	cipherKdfIter      = 256000
	cipherHmacAlg      = "HMAC_SHA512"
	cipherKdfAlgorithm = "PBKDF2_HMAC_SHA512"
	derivedKeyLength   = 32
	saltLength         = 32
)

type SQLCipherSessionStore struct {
	db *sql.DB
}

func NewSQLCipherSessionStore(dbPath, passphrase string) (*SQLCipherSessionStore, error) {
	if passphrase == "" {
		return nil, ErrEmptyPassphrase
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	salt, err := loadOrGenerateSalt(dbPath + ".salt")
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate salt: %w", err)
	}

	key := pbkdf2.Key([]byte(passphrase), salt, cipherKdfIter, derivedKeyLength, sha512.New)

	dsn := fmt.Sprintf(
		"file:%s?_pragma_key=x'%s'&_pragma_cipher_page_size=%d&_pragma_kdf_iter=%d&_pragma_cipher_hmac_algorithm=%s&_pragma_cipher_kdf_algorithm=%s&_foreign_keys=ON",
		dbPath, hex.EncodeToString(key), cipherPageSize, cipherKdfIter, cipherHmacAlg, cipherKdfAlgorithm,
	)

	for i := range key {
		key[i] = 0
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &SQLCipherSessionStore{db: db}, nil
}

func (s *SQLCipherSessionStore) CreateSession(session Session) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_agent, cookies, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		session.ID, session.UserAgent, session.Cookies, session.ExpiresAt, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (s *SQLCipherSessionStore) GetSession(id string) (*Session, error) {
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

func (s *SQLCipherSessionStore) CloseSession(id string) error {
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

func (s *SQLCipherSessionStore) ListSessions() ([]Session, error) {
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

func (s *SQLCipherSessionStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func loadOrGenerateSalt(saltPath string) ([]byte, error) {
	existing, err := os.ReadFile(saltPath)
	if err == nil && len(existing) == saltLength {
		return existing, nil
	}

	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	if err := os.WriteFile(saltPath, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write salt file: %w", err)
	}

	return salt, nil
}

func initSchema(db *sql.DB) error {
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
