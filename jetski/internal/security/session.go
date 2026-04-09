package security

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
)

type Session struct {
	ID        string
	UserAgent string
	Cookies   []byte
	ExpiresAt int64
}

func EncryptSession(session Session, passphrase string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(session); err != nil {
		return nil, fmt.Errorf("failed to encode session: %w", err)
	}

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create scrypt recipient: %w", err)
	}

	encrypted := &bytes.Buffer{}
	w, err := age.Encrypt(encrypted, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encryptor: %w", err)
	}

	return encrypted.Bytes(), nil
}

func DecryptSession(encrypted []byte, passphrase string) (*Session, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create scrypt identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(encrypted), identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryptor: %w", err)
	}

	var session Session
	decoder := gob.NewDecoder(r)
	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

func SaveSession(filePath string, session Session, passphrase string) error {
	encrypted, err := EncryptSession(session, passphrase)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func LoadSession(filePath string, passphrase string) (*Session, error) {
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	return DecryptSession(encrypted, passphrase)
}
