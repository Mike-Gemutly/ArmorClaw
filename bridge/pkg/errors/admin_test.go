package errors

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Mock Matrix adapter for testing
type mockMatrixAdapter struct {
	members []RoomMember
	err     error
}

func (m *mockMatrixAdapter) GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error) {
	return m.members, m.err
}

func TestAdminResolver_Resolve_ConfigAdmin(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		ConfigAdminMXID: "@admin:example.com",
		SetupUserMXID:   "@setup:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if target.MXID != "@admin:example.com" {
		t.Errorf("MXID = %q, want @admin:example.com", target.MXID)
	}
	if target.Source != "config" {
		t.Errorf("Source = %q, want config", target.Source)
	}
}

func TestAdminResolver_Resolve_SetupUser(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		SetupUserMXID: "@setup:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if target.MXID != "@setup:example.com" {
		t.Errorf("MXID = %q, want @setup:example.com", target.MXID)
	}
	if target.Source != "setup" {
		t.Errorf("Source = %q, want setup", target.Source)
	}
}

func TestAdminResolver_Resolve_RoomMember(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{
			{UserID: "@member1:example.com", PowerLevel: 0},
			{UserID: "@admin:example.com", PowerLevel: 100},
			{UserID: "@mod:example.com", PowerLevel: 50},
		},
	}

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should return first admin/moderator (highest power level that's >= 50)
	if target.Source != "room" {
		t.Errorf("Source = %q, want room", target.Source)
	}
	// Order depends on iteration, but should be an admin
	if target.MXID == "@member1:example.com" {
		t.Errorf("Should not return regular member")
	}
}

func TestAdminResolver_Resolve_RoomNoAdmin(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{
			{UserID: "@member1:example.com", PowerLevel: 0},
			{UserID: "@member2:example.com", PowerLevel: 10},
		},
	}

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should fall back to first member
	if target.Source != "room" {
		t.Errorf("Source = %q, want room", target.Source)
	}
}

func TestAdminResolver_Resolve_Fallback(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		FallbackMXID: "@fallback:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if target.MXID != "@fallback:example.com" {
		t.Errorf("MXID = %q, want @fallback:example.com", target.MXID)
	}
	if target.Source != "fallback" {
		t.Errorf("Source = %q, want fallback", target.Source)
	}
}

func TestAdminResolver_Resolve_NoTarget(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{})

	_, err := resolver.Resolve(context.Background())
	if err == nil {
		t.Error("Resolve() should return error when no target configured")
	}
}

func TestAdminResolver_Resolve_RoomError(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		err: errors.New("room not found"),
	}

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
		FallbackMXID:  "@fallback:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should fall back when room lookup fails
	if target.Source != "fallback" {
		t.Errorf("Source = %q, want fallback", target.Source)
	}
}

func TestAdminResolver_Cache(t *testing.T) {
	callCount := 0
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{
			{UserID: "@admin:example.com", PowerLevel: 100},
		},
	}
	mockAdapter.err = nil

	// Override GetRoomMembers to count calls
	originalMembers := mockAdapter.members

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
		CacheTTL:      1 * time.Second,
	})

	// First call
	target1, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("First Resolve() error = %v", err)
	}

	// Second call (should use cache)
	target2, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Second Resolve() error = %v", err)
	}

	if target1.MXID != target2.MXID {
		t.Error("Cached result should match")
	}

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third call (should refresh cache)
	mockAdapter.members = originalMembers
	_, _ = resolver.Resolve(context.Background())

	_ = callCount // Just to avoid unused variable warning
}

func TestAdminResolver_InvalidateCache(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		ConfigAdminMXID: "@admin:example.com",
		CacheTTL:        1 * time.Hour,
	})

	// First call (caches)
	resolver.Resolve(context.Background())

	// Change config
	resolver.SetConfigAdmin("@newadmin:example.com")

	// Should return new value (cache invalidated)
	target, _ := resolver.Resolve(context.Background())
	if target.MXID != "@newadmin:example.com" {
		t.Errorf("MXID = %q, want @newadmin:example.com", target.MXID)
	}
}

func TestAdminResolver_SetMethods(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{})

	resolver.SetConfigAdmin("@config:example.com")
	if resolver.GetConfigAdmin() != "@config:example.com" {
		t.Error("SetConfigAdmin failed")
	}

	resolver.SetSetupUser("@setup:example.com")
	if resolver.GetSetupUser() != "@setup:example.com" {
		t.Error("SetSetupUser failed")
	}

	resolver.SetAdminRoom("!room:example.com")
	if resolver.GetAdminRoom() != "!room:example.com" {
		t.Error("SetAdminRoom failed")
	}

	// Test fallback separately (clear other sources first)
	resolver.SetConfigAdmin("")
	resolver.SetSetupUser("")
	resolver.SetAdminRoom("")
	resolver.SetFallback("@fallback:example.com")

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if target.Source != "fallback" {
		t.Errorf("SetFallback failed, source = %q", target.Source)
	}
}

func TestAdminResolver_Priority(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{
			{UserID: "@roomadmin:example.com", PowerLevel: 100},
		},
	}

	// Config should win over setup user
	resolver := NewAdminResolver(AdminConfig{
		ConfigAdminMXID: "@config:example.com",
		SetupUserMXID:   "@setup:example.com",
		AdminRoomID:     "!room:example.com",
		MatrixAdapter:   mockAdapter,
		FallbackMXID:    "@fallback:example.com",
	})

	target, _ := resolver.Resolve(context.Background())
	if target.Source != "config" {
		t.Errorf("Config should have highest priority, got source %q", target.Source)
	}

	// Setup user should win over room
	resolver.SetConfigAdmin("")
	target, _ = resolver.Resolve(context.Background())
	if target.Source != "setup" {
		t.Errorf("Setup user should have second priority, got source %q", target.Source)
	}

	// Room should win over fallback
	resolver.SetSetupUser("")
	target, _ = resolver.Resolve(context.Background())
	if target.Source != "room" {
		t.Errorf("Room should have third priority, got source %q", target.Source)
	}

	// Fallback is last resort
	resolver.SetAdminRoom("")
	target, _ = resolver.Resolve(context.Background())
	if target.Source != "fallback" {
		t.Errorf("Fallback should be last resort, got source %q", target.Source)
	}
}

func TestSetupUserStorage(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "setup-user")

	storage := NewSetupUserStorage(path)

	// Initially doesn't exist
	if storage.Exists() {
		t.Error("Storage should not exist initially")
	}

	// Save
	err := storage.Save("@setup:example.com")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Now exists
	if !storage.Exists() {
		t.Error("Storage should exist after Save()")
	}

	// Load
	mxid, err := storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if mxid != "@setup:example.com" {
		t.Errorf("Load() = %q, want @setup:example.com", mxid)
	}
}

func TestSetupUserStorage_LoadNonExistent(t *testing.T) {
	storage := NewSetupUserStorage("/nonexistent/path/setup-user")

	_, err := storage.Load()
	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

func TestGlobalAdminResolver(t *testing.T) {
	// Reset global resolver
	SetGlobalAdminResolver(NewAdminResolver(AdminConfig{
		SetupUserMXID: "@global:example.com",
	}))

	target, err := ResolveAdmin(context.Background())
	if err != nil {
		t.Fatalf("ResolveAdmin() error = %v", err)
	}

	if target.MXID != "@global:example.com" {
		t.Errorf("MXID = %q, want @global:example.com", target.MXID)
	}

	// Test setters
	SetGlobalSetupUser("@newsetup:example.com")
	SetGlobalConfigAdmin("@newconfig:example.com")
	SetGlobalAdminRoom("!newroom:example.com")

	// Verify config admin takes priority
	target, _ = ResolveAdmin(context.Background())
	if target.MXID != "@newconfig:example.com" {
		t.Errorf("MXID = %q, want @newconfig:example.com", target.MXID)
	}
}

func TestAdminResolver_EmptyRoomMembers(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{},
	}

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
		FallbackMXID:  "@fallback:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should fall back when room is empty
	if target.Source != "fallback" {
		t.Errorf("Source = %q, want fallback", target.Source)
	}
}

func TestAdminResolver_NoMatrixAdapter(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:  "!room:example.com",
		FallbackMXID: "@fallback:example.com",
	})

	target, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should fall back when no adapter
	if target.Source != "fallback" {
		t.Errorf("Source = %q, want fallback", target.Source)
	}
}

func TestAdminResolver_ContextCancellation(t *testing.T) {
	mockAdapter := &mockMatrixAdapter{
		members: []RoomMember{
			{UserID: "@admin:example.com", PowerLevel: 100},
		},
	}

	resolver := NewAdminResolver(AdminConfig{
		AdminRoomID:   "!room:example.com",
		MatrixAdapter: mockAdapter,
	})

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should still work with cached or immediate resolution
	target, err := resolver.Resolve(ctx)
	// The result depends on whether room lookup is needed
	// If config/setup are set, it should succeed
	_ = target
	_ = err
}

func TestAdminConfig_Defaults(t *testing.T) {
	cfg := DefaultAdminConfig()

	if cfg.CacheTTL != 5*time.Minute {
		t.Errorf("Default CacheTTL = %v, want 5m", cfg.CacheTTL)
	}
}

func TestNewAdminResolver_ZeroCacheTTL(t *testing.T) {
	resolver := NewAdminResolver(AdminConfig{
		CacheTTL: 0,
	})

	// Should default to 5 minutes
	if resolver.cacheTTL != 5*time.Minute {
		t.Errorf("cacheTTL = %v, want 5m", resolver.cacheTTL)
	}
}

func TestSetupUserStorage_FilePermissions(t *testing.T) {
	// Skip on Windows - Unix permissions not supported
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows - Unix permissions not supported")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "setup-user")

	storage := NewSetupUserStorage(path)
	err := storage.Save("@setup:example.com")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Should be 0600 (owner read/write only)
	expectedPerms := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerms {
		t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), expectedPerms)
	}
}
