package security

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrEmptyPassphrase    = errors.New("passphrase must not be empty")
	ErrDuplicateSession   = errors.New("session already exists")
	ErrInvalidCredentials = errors.New("invalid credentials or corrupted database")

	isEncrypted bool
)

type Session struct {
	ID        string
	UserAgent string
	Cookies   []byte
	ExpiresAt int64
}

type SessionStore interface {
	CreateSession(session Session) error
	GetSession(id string) (*Session, error)
	CloseSession(id string) error
	ListSessions() ([]Session, error)
	Close() error
}

func NewSessionStore(dbPath, passphrase string) (SessionStore, error) {
	store, err := newCipherSessionStore(dbPath, passphrase)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func SaveSession(filePath string, session Session, passphrase string) error {
	dbPath := strings.TrimSuffix(filePath, ".enc") + ".db"

	store, err := NewSessionStore(dbPath, passphrase)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.CreateSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func LoadSession(filePath string, passphrase string) (*Session, error) {
	dbPath := strings.TrimSuffix(filePath, ".enc") + ".db"

	store, err := NewSessionStore(dbPath, passphrase)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	sessions, err := store.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return nil, ErrSessionNotFound
	}

	return &sessions[0], nil
}

func init() {
	if !isEncrypted {
		log.Println("[security] WARNING: SQLCipher unavailable (CGO disabled). Sessions stored as plaintext SQLite. Do not use in production with sensitive data.")
	}
}
