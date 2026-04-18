package email

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultStorageDir = "/var/lib/armorclaw/email-files"

type EmailStorage interface {
	StoreEmail(emailID string, rawEmail []byte) error
	StoreAttachment(emailID, filename string, content []byte) (fileID string, err error)
	GetAttachment(fileID string) ([]byte, error)
	DeleteEmail(emailID string) error
	StoreAttachmentText(emailID, filename, text string) error
}

type LocalFSEmailStorage struct {
	baseDir string
}

func NewLocalFSEmailStorage(baseDir string) *LocalFSEmailStorage {
	if baseDir == "" {
		baseDir = defaultStorageDir
	}
	return &LocalFSEmailStorage{baseDir: baseDir}
}

func (s *LocalFSEmailStorage) StoreEmail(emailID string, rawEmail []byte) error {
	dir := filepath.Join(s.baseDir, "emails", emailID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create email dir: %w", err)
	}
	path := filepath.Join(dir, "raw.eml")
	if err := os.WriteFile(path, rawEmail, 0600); err != nil {
		return fmt.Errorf("write raw email: %w", err)
	}
	return nil
}

func (s *LocalFSEmailStorage) StoreAttachment(emailID, filename string, content []byte) (string, error) {
	hash := sha256.Sum256(content)
	fileID := hex.EncodeToString(hash[:16])

	dir := filepath.Join(s.baseDir, "attachments", emailID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create attachment dir: %w", err)
	}

	safeName := sanitizeFilename(filename)
	if safeName == "" {
		safeName = fileID
	}
	path := filepath.Join(dir, safeName)
	if err := os.WriteFile(path, content, 0600); err != nil {
		return "", fmt.Errorf("write attachment: %w", err)
	}

	meta := filepath.Join(dir, fileID+".meta")
	metaContent := fmt.Sprintf(`{"file_id":"%s","filename":"%s","size":%d,"stored_at":%d}`,
		fileID, safeName, len(content), time.Now().Unix())
	if err := os.WriteFile(meta, []byte(metaContent), 0600); err != nil {
		return "", fmt.Errorf("write meta: %w", err)
	}

	return fileID, nil
}

func (s *LocalFSEmailStorage) GetAttachment(fileID string) ([]byte, error) {
	attachDir := filepath.Join(s.baseDir, "attachments")
	entries, err := os.ReadDir(attachDir)
	if err != nil {
		return nil, fmt.Errorf("read attachments dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(attachDir, entry.Name(), fileID+".meta")
		if _, err := os.Stat(metaPath); err != nil {
			continue
		}

		files, err := os.ReadDir(filepath.Join(attachDir, entry.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() == fileID+".meta" {
				continue
			}
			fullPath := filepath.Join(attachDir, entry.Name(), f.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("read attachment: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("attachment %s not found", fileID)
}

func (s *LocalFSEmailStorage) DeleteEmail(emailID string) error {
	emailDir := filepath.Join(s.baseDir, "emails", emailID)
	if err := os.RemoveAll(emailDir); err != nil {
		return fmt.Errorf("delete email: %w", err)
	}
	attachDir := filepath.Join(s.baseDir, "attachments", emailID)
	_ = os.RemoveAll(attachDir)
	return nil
}

func (s *LocalFSEmailStorage) StoreAttachmentText(emailID, filename, text string) error {
	dir := filepath.Join(s.baseDir, "extracted-text", emailID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create extracted-text dir: %w", err)
	}
	safeName := sanitizeFilename(filename)
	if safeName == "" {
		safeName = "unnamed"
	}
	path := filepath.Join(dir, safeName+".txt")
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		return fmt.Errorf("write extracted text: %w", err)
	}
	return nil
}

func (s *LocalFSEmailStorage) StoreRaw(emailID string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read raw email: %w", err)
	}
	return s.StoreEmail(emailID, data)
}

func sanitizeFilename(name string) string {
	replaced := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\x00' {
			return '_'
		}
		return r
	}, name)
	if len(replaced) > 255 {
		replaced = replaced[:255]
	}
	return replaced
}
