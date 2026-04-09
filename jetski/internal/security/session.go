package security

import (
	"fmt"
	"strings"
)

type Session struct {
	ID        string
	UserAgent string
	Cookies   []byte
	ExpiresAt int64
}

func SaveSession(filePath string, session Session, passphrase string) error {
	dbPath := strings.TrimSuffix(filePath, ".enc") + ".db"

	store, err := NewSQLCipherSessionStore(dbPath, passphrase)
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

	store, err := NewSQLCipherSessionStore(dbPath, passphrase)
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
