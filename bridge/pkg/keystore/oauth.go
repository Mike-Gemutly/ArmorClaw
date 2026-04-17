package keystore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type OAuthTokenRecord struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"`
	AccountEmail    string    `json:"account_email"`
	RefreshToken    string    `json:"-"` // Never serialized to JSON
	Scopes          string    `json:"scopes"`
	CreatedAt       time.Time `json:"created_at"`
	LastRefreshedAt time.Time `json:"last_refreshed_at,omitempty"`
	Status          string    `json:"status"`
}

type OAuthTokenInfo struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	AccountEmail string    `json:"account_email"`
	Scopes       string    `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

func (ks *Keystore) initOAuthTable() error {
	_, err := ks.db.Exec(`
		CREATE TABLE IF NOT EXISTS oauth_tokens (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			account_email TEXT NOT NULL,
			refresh_token_encrypted BLOB NOT NULL,
			refresh_token_nonce BLOB NOT NULL,
			scopes TEXT DEFAULT '',
			created_at INTEGER NOT NULL,
			last_refreshed_at INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active'
		)
	`)
	if err != nil {
		return fmt.Errorf("create oauth_tokens table: %w", err)
	}

	_, err = ks.db.Exec(`CREATE INDEX IF NOT EXISTS idx_oauth_provider ON oauth_tokens(provider)`)
	if err != nil {
		return fmt.Errorf("create oauth index: %w", err)
	}

	return nil
}

func (ks *Keystore) StoreOAuthToken(ctx context.Context, provider, email, refreshToken string) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return "", fmt.Errorf("keystore is not open")
	}

	id := generateOAuthID()
	encrypted, nonce, err := ks.encrypt([]byte(refreshToken))
	if err != nil {
		return "", fmt.Errorf("encrypt refresh token: %w", err)
	}

	now := time.Now().Unix()
	_, err = ks.db.Exec(`
		INSERT INTO oauth_tokens (id, provider, account_email, refresh_token_encrypted, refresh_token_nonce, scopes, created_at, status)
		VALUES (?, ?, ?, ?, ?, '', ?, 'active')
		ON CONFLICT(id) DO UPDATE SET
			refresh_token_encrypted = excluded.refresh_token_encrypted,
			refresh_token_nonce = excluded.refresh_token_nonce,
			status = 'active'
	`, id, provider, email, encrypted, nonce, now)
	if err != nil {
		return "", fmt.Errorf("store oauth token: %w", err)
	}

	return id, nil
}

func (ks *Keystore) GetOAuthRefreshToken(ctx context.Context, provider string) (*OAuthTokenRecord, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, fmt.Errorf("keystore is not open")
	}

	row := ks.db.QueryRow(`
		SELECT id, provider, account_email, refresh_token_encrypted, refresh_token_nonce, scopes, created_at, last_refreshed_at, status
		FROM oauth_tokens
		WHERE provider = ? AND status = 'active'
		ORDER BY created_at DESC LIMIT 1
	`, provider)

	var rec OAuthTokenRecord
	var encrypted, nonce []byte
	var createdAt, lastRefreshed int64

	err := row.Scan(&rec.ID, &rec.Provider, &rec.AccountEmail, &encrypted, &nonce, &rec.Scopes, &createdAt, &lastRefreshed, &rec.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query oauth token: %w", err)
	}

	plaintext, err := ks.decrypt(encrypted, nonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	rec.RefreshToken = string(plaintext)
	rec.CreatedAt = time.Unix(createdAt, 0)
	if lastRefreshed > 0 {
		rec.LastRefreshedAt = time.Unix(lastRefreshed, 0)
	}

	return &rec, nil
}

func (ks *Keystore) RevokeOAuthToken(ctx context.Context, id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return fmt.Errorf("keystore is not open")
	}

	_, err := ks.db.Exec(`UPDATE oauth_tokens SET status = 'revoked' WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke oauth token: %w", err)
	}
	return nil
}

func (ks *Keystore) ListOAuthTokens(ctx context.Context) ([]OAuthTokenInfo, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, fmt.Errorf("keystore is not open")
	}

	rows, err := ks.db.Query(`
		SELECT id, provider, account_email, scopes, created_at, status
		FROM oauth_tokens
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list oauth tokens: %w", err)
	}
	defer rows.Close()

	var tokens []OAuthTokenInfo
	for rows.Next() {
		var info OAuthTokenInfo
		var createdAt int64
		if err := rows.Scan(&info.ID, &info.Provider, &info.AccountEmail, &info.Scopes, &createdAt, &info.Status); err != nil {
			continue
		}
		info.CreatedAt = time.Unix(createdAt, 0)
		tokens = append(tokens, info)
	}
	return tokens, nil
}

func generateOAuthID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "oauth_" + hex.EncodeToString(b)
}
