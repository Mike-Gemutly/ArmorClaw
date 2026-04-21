package rpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/armorclaw/bridge/pkg/invite"
)

func newTestInviteStore(t *testing.T) *invite.InviteStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := invite.NewInviteStore(db)
	if err != nil {
		t.Fatalf("new invite store: %v", err)
	}
	return store
}

func seedInvite(t *testing.T, store *invite.InviteStore, role invite.Role, status invite.InviteStatus) *invite.InviteRecord {
	t.Helper()
	record := &invite.InviteRecord{
		Role:      role,
		CreatedBy: "@admin:test.com",
		MaxUses:   10,
	}
	if err := store.CreateInvite(record); err != nil {
		t.Fatalf("seed invite: %v", err)
	}
	if status != "" && status != invite.StatusActive {
		if status == invite.StatusRevoked {
			_ = store.RevokeInvite(record.ID)
		}
	}
	return record
}

func newServerWithInviteStore(t *testing.T, store *invite.InviteStore) *Server {
	t.Helper()
	return &Server{
		handlers:    make(map[string]HandlerFunc, 32),
		inviteStore: store,
	}
}

func TestInviteHandlerRegistration(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	methods := []string{"invite.list", "invite.create", "invite.revoke", "invite.validate"}
	for _, m := range methods {
		if _, ok := server.handlers[m]; !ok {
			t.Errorf("handler %q not registered", m)
		}
	}
}

func TestInviteList(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)
	s.registerHandlers()

	result, rpcErr := s.handleInviteList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	invites, ok := result.([]*invite.InviteRecord)
	if !ok {
		t.Fatalf("expected []*invite.InviteRecord, got %T", result)
	}
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites, got %d", len(invites))
	}

	seedInvite(t, store, invite.RoleUser, "")
	seedInvite(t, store, invite.RoleAdmin, "")

	result, rpcErr = s.handleInviteList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	invites = result.([]*invite.InviteRecord)
	if len(invites) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(invites))
	}
}

func TestInviteListStoreNil(t *testing.T) {
	s := newServerWithInviteStore(t, nil)
	_, rpcErr := s.handleInviteList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr == nil || rpcErr.Code != InternalError {
		t.Fatalf("expected InternalError, got %v", rpcErr)
	}
}

func TestInviteCreate(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	result, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"admin","expiration":"7d","max_uses":5,"created_by":"@admin:test.com","welcome_message":"Welcome!"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	record, ok := result.(*invite.InviteRecord)
	if !ok {
		t.Fatalf("expected *invite.InviteRecord, got %T", result)
	}
	if record.Code == "" {
		t.Fatal("expected non-empty code")
	}
	if record.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if record.Role != invite.RoleAdmin {
		t.Fatalf("expected role admin, got %s", record.Role)
	}
	if record.CreatedBy != "@admin:test.com" {
		t.Fatalf("expected created_by @admin:test.com, got %s", record.CreatedBy)
	}
	if record.ExpiresAt == nil {
		t.Fatal("expected expires_at to be set")
	}
	if record.MaxUses != 5 {
		t.Fatalf("expected max_uses 5, got %d", record.MaxUses)
	}
}

func TestInviteCreateNeverExpires(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	result, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"user","expiration":"never","max_uses":0,"created_by":"@admin:test.com"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	record := result.(*invite.InviteRecord)
	if record.ExpiresAt != nil {
		t.Fatal("expected nil expires_at for 'never' expiration")
	}
}

func TestInviteCreateMissingRole(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"expiration":"7d","created_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteCreateInvalidRole(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"superadmin","expiration":"7d","created_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteCreateMissingExpiration(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"user","created_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteCreateInvalidExpiration(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"user","expiration":"2w","created_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteCreateMissingCreatedBy(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`{"role":"user","expiration":"7d"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteCreateBadParams(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteCreate(context.Background(), &Request{
		Params: json.RawMessage(`invalid json`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteRevoke(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)
	record := seedInvite(t, store, invite.RoleUser, "")

	result, rpcErr := s.handleInviteRevoke(context.Background(), &Request{
		Params: json.RawMessage(`{"invite_id":"` + record.ID + `","revoked_by":"@admin:test.com"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	resp, ok := result.(SuccessResponse)
	if !ok {
		t.Fatalf("expected SuccessResponse, got %T", result)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestInviteRevokeIdempotent(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)
	record := seedInvite(t, store, invite.RoleUser, invite.StatusRevoked)

	result, rpcErr := s.handleInviteRevoke(context.Background(), &Request{
		Params: json.RawMessage(`{"invite_id":"` + record.ID + `","revoked_by":"@admin:test.com"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error on idempotent revoke: %v", rpcErr)
	}
	resp := result.(SuccessResponse)
	if !resp.Success {
		t.Fatal("expected success=true for idempotent revoke")
	}
}

func TestInviteRevokeMissingInviteID(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteRevoke(context.Background(), &Request{
		Params: json.RawMessage(`{"revoked_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteRevokeNotFound(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteRevoke(context.Background(), &Request{
		Params: json.RawMessage(`{"invite_id":"nonexistent","revoked_by":"@admin:test.com"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError, got %v", rpcErr)
	}
}

func TestInviteValidate(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)
	record := seedInvite(t, store, invite.RoleUser, "")

	result, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{"code":"` + record.Code + `"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	validated, ok := result.(*invite.InviteRecord)
	if !ok {
		t.Fatalf("expected *invite.InviteRecord, got %T", result)
	}
	if validated.Code != record.Code {
		t.Fatalf("expected code %s, got %s", record.Code, validated.Code)
	}
	if validated.CreatedBy != "@admin:test.com" {
		t.Fatalf("expected created_by in response, got %s", validated.CreatedBy)
	}
}

func TestInviteValidateMissingCode(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestInviteValidateNotFound(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	_, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{"code":"nonexistent"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError, got %v", rpcErr)
	}
}

func TestInviteValidateExpired(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	past := time.Now().UTC().Add(-1 * time.Hour)
	record := &invite.InviteRecord{
		Role:      invite.RoleUser,
		CreatedBy: "@admin:test.com",
		ExpiresAt: &past,
		MaxUses:   10,
	}
	if err := store.CreateInvite(record); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	_, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{"code":"` + record.Code + `"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError for expired invite, got %v", rpcErr)
	}
	if rpcErr.Message != "invite has expired" {
		t.Fatalf("expected 'invite has expired', got %s", rpcErr.Message)
	}
}

func TestInviteValidateRevoked(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)
	record := seedInvite(t, store, invite.RoleUser, invite.StatusRevoked)

	_, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{"code":"` + record.Code + `"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError for revoked invite, got %v", rpcErr)
	}
}

func TestInviteValidateExhausted(t *testing.T) {
	store := newTestInviteStore(t)
	s := newServerWithInviteStore(t, store)

	record := &invite.InviteRecord{
		Role:      invite.RoleUser,
		CreatedBy: "@admin:test.com",
		MaxUses:   1,
	}
	if err := store.CreateInvite(record); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	if err := store.IncrementUseCount(record.ID); err != nil {
		t.Fatalf("increment use count: %v", err)
	}

	_, rpcErr := s.handleInviteValidate(context.Background(), &Request{
		Params: json.RawMessage(`{"code":"` + record.Code + `"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError for exhausted invite, got %v", rpcErr)
	}
}
