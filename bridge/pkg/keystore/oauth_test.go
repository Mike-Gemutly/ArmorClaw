//go:build cgo

package keystore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func createOAuthTestKeystore(t *testing.T) *Keystore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "oauth_test.db")
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	ks, err := New(Config{DBPath: dbPath, MasterKey: masterKey})
	if err != nil {
		t.Fatalf("New keystore: %v", err)
	}
	if err := ks.Open(); err != nil {
		t.Fatalf("Open keystore: %v", err)
	}
	if err := ks.initOAuthTable(); err != nil {
		t.Fatalf("initOAuthTable: %v", err)
	}
	return ks
}

func TestOAuth_StoreAndGetRoundTrip(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()
	ctx := context.Background()

	id, err := ks.StoreOAuthToken(ctx, "gmail", "user@example.com", "my-secret-refresh-token")
	if err != nil {
		t.Fatalf("StoreOAuthToken: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if !strings.HasPrefix(id, "oauth_") {
		t.Errorf("ID should start with oauth_, got %s", id)
	}

	rec, err := ks.GetOAuthRefreshToken(ctx, "gmail")
	if err != nil {
		t.Fatalf("GetOAuthRefreshToken: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.Provider != "gmail" {
		t.Errorf("Provider = %q, want gmail", rec.Provider)
	}
	if rec.AccountEmail != "user@example.com" {
		t.Errorf("AccountEmail = %q, want user@example.com", rec.AccountEmail)
	}
	if rec.RefreshToken != "my-secret-refresh-token" {
		t.Errorf("RefreshToken not decrypted correctly, got %q", rec.RefreshToken)
	}
	if rec.Status != "active" {
		t.Errorf("Status = %q, want active", rec.Status)
	}
}

func TestOAuth_GetTokenNotFound(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()

	rec, err := ks.GetOAuthRefreshToken(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetOAuthRefreshToken: %v", err)
	}
	if rec != nil {
		t.Error("expected nil for nonexistent provider")
	}
}

func TestOAuth_RevokeToken(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()
	ctx := context.Background()

	id, _ := ks.StoreOAuthToken(ctx, "outlook", "user@outlook.com", "token-to-revoke")

	if err := ks.RevokeOAuthToken(ctx, id); err != nil {
		t.Fatalf("RevokeOAuthToken: %v", err)
	}

	rec, err := ks.GetOAuthRefreshToken(ctx, "outlook")
	if err != nil {
		t.Fatalf("GetOAuthRefreshToken after revoke: %v", err)
	}
	if rec != nil {
		t.Error("revoked token should not be returned (GetOAuthRefreshToken returns active only)")
	}
}

func TestOAuth_RevokeNonexistentToken(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()

	err := ks.RevokeOAuthToken(context.Background(), "nonexistent-id")
	if err != nil {
		t.Errorf("revoking nonexistent token should not error: %v", err)
	}
}

func TestOAuth_ListTokens(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()
	ctx := context.Background()

	ks.StoreOAuthToken(ctx, "gmail", "user@gmail.com", "gmail-token")
	ks.StoreOAuthToken(ctx, "outlook", "user@outlook.com", "outlook-token")

	tokens, err := ks.ListOAuthTokens(ctx)
	if err != nil {
		t.Fatalf("ListOAuthTokens: %v", err)
	}
	if len(tokens) < 2 {
		t.Errorf("expected >= 2 tokens, got %d", len(tokens))
	}

	providers := make(map[string]bool)
	for _, tok := range tokens {
		providers[tok.Provider] = true
		if tok.AccountEmail == "" {
			t.Error("AccountEmail should not be empty")
		}
	}
	if !providers["gmail"] {
		t.Error("expected gmail in list")
	}
	if !providers["outlook"] {
		t.Error("expected outlook in list")
	}
}

func TestOAuth_EncryptionAtRest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "encryption_test.db")
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	ks, _ := New(Config{DBPath: dbPath, MasterKey: masterKey})
	ks.Open()
	ks.initOAuthTable()
	ctx := context.Background()
	ks.StoreOAuthToken(ctx, "gmail", "user@test.com", "super-secret-token-12345")
	ks.Close()

	rawData, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read db file: %v", err)
	}
	if strings.Contains(string(rawData), "super-secret-token-12345") {
		t.Error("plaintext token found in database file — encryption failed!")
	}
}

func TestOAuth_TokenInfoFields(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()
	ctx := context.Background()

	before := time.Now().Truncate(time.Second)
	ks.StoreOAuthToken(ctx, "gmail", "user@test.com", "token")
	tokens, _ := ks.ListOAuthTokens(ctx)
	after := time.Now().Add(time.Second)

	if len(tokens) == 0 {
		t.Fatal("expected at least 1 token")
	}
	info := tokens[0]
	if info.ID == "" {
		t.Error("ID should not be empty")
	}
	if info.CreatedAt.Before(before) || info.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, expected between %v and %v", info.CreatedAt, before, after)
	}
	if info.Status != "active" {
		t.Errorf("Status = %q, want active", info.Status)
	}
}

func TestOAuth_ClosedKeystore(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	ks.Close()

	ctx := context.Background()

	_, err := ks.StoreOAuthToken(ctx, "gmail", "user@test.com", "token")
	if err == nil {
		t.Error("expected error storing to closed keystore")
	}

	_, err = ks.GetOAuthRefreshToken(ctx, "gmail")
	if err == nil {
		t.Error("expected error getting from closed keystore")
	}

	err = ks.RevokeOAuthToken(ctx, "some-id")
	if err == nil {
		t.Error("expected error revoking in closed keystore")
	}

	_, err = ks.ListOAuthTokens(ctx)
	if err == nil {
		t.Error("expected error listing from closed keystore")
	}
}

func TestOAuth_MultipleProviders(t *testing.T) {
	ks := createOAuthTestKeystore(t)
	defer ks.Close()
	ctx := context.Background()

	ks.StoreOAuthToken(ctx, "gmail", "user@gmail.com", "gmail-rt")
	ks.StoreOAuthToken(ctx, "outlook", "user@outlook.com", "outlook-rt")

	gmail, err := ks.GetOAuthRefreshToken(ctx, "gmail")
	if err != nil {
		t.Fatalf("get gmail token: %v", err)
	}
	if gmail.RefreshToken != "gmail-rt" {
		t.Errorf("gmail token = %q, want gmail-rt", gmail.RefreshToken)
	}

	outlook, err := ks.GetOAuthRefreshToken(ctx, "outlook")
	if err != nil {
		t.Fatalf("get outlook token: %v", err)
	}
	if outlook.RefreshToken != "outlook-rt" {
		t.Errorf("outlook token = %q, want outlook-rt", outlook.RefreshToken)
	}
}
