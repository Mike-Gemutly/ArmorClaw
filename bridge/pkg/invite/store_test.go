package invite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openMemInviteDB(t *testing.T) (*sql.DB, *InviteStore) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	store, err := NewInviteStore(db)
	if err != nil {
		db.Close()
		t.Fatalf("NewInviteStore: %v", err)
	}
	return db, store
}

func sampleInvite(id, code string) *InviteRecord {
	return &InviteRecord{
		ID:        id,
		Code:      code,
		Role:      RoleAdmin,
		CreatedBy: "@admin:armorclaw.test",
		CreatedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		MaxUses:   5,
		UseCount:  0,
		Status:    StatusActive,
	}
}

// --- CRUD tests ---

func TestInviteStore_CreateAndGet(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "ABC123def456")
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	assertInviteEqual(t, inv, got)
}

func TestInviteStore_GetNotFound(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	_, err := store.GetInvite("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent invite, got nil")
	}
}

func TestInviteStore_GetInviteByCode(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "XYZ789abc012")
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	got, err := store.GetInviteByCode("XYZ789abc012")
	if err != nil {
		t.Fatalf("GetInviteByCode: %v", err)
	}
	if got.ID != "inv_001" {
		t.Errorf("ID: got %q, want %q", got.ID, "inv_001")
	}
}

func TestInviteStore_GetInviteByCodeNotFound(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	_, err := store.GetInviteByCode("nope")
	if err == nil {
		t.Fatal("expected error for nonexistent code, got nil")
	}
}

func TestInviteStore_ListInvites(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv1 := sampleInvite("inv_001", "code001")
	inv1.CreatedAt = time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC)
	inv2 := sampleInvite("inv_002", "code002")
	inv2.CreatedAt = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	if err := store.CreateInvite(inv1); err != nil {
		t.Fatalf("create inv1: %v", err)
	}
	if err := store.CreateInvite(inv2); err != nil {
		t.Fatalf("create inv2: %v", err)
	}

	list, err := store.ListInvites()
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(list))
	}
	if list[0].ID != "inv_002" {
		t.Errorf("expected first invite inv_002 (newer), got %s", list[0].ID)
	}
}

func TestInviteStore_ListInvitesEmpty(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	list, err := store.ListInvites()
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d invites", len(list))
	}
}

func TestInviteStore_RevokeInvite(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	if err := store.RevokeInvite("inv_001"); err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite after revoke: %v", err)
	}
	if got.Status != StatusRevoked {
		t.Errorf("Status: got %q, want %q", got.Status, StatusRevoked)
	}
}

func TestInviteStore_RevokeInviteNotFound(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	if err := store.RevokeInvite("nonexistent"); err == nil {
		t.Fatal("expected error revoking nonexistent invite")
	}
}

func TestInviteStore_RevokeInviteNotActive(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	inv.Status = StatusUsed
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	if err := store.RevokeInvite("inv_001"); err == nil {
		t.Fatal("expected error revoking non-active invite")
	}
}

func TestInviteStore_IncrementUseCount(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	inv.MaxUses = 3
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	if err := store.IncrementUseCount("inv_001"); err != nil {
		t.Fatalf("IncrementUseCount: %v", err)
	}

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.UseCount != 1 {
		t.Errorf("UseCount: got %d, want 1", got.UseCount)
	}
	if got.Status != StatusActive {
		t.Errorf("Status should still be active, got %q", got.Status)
	}
}

func TestInviteStore_IncrementUseCountExhausted(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	inv.MaxUses = 2
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	store.IncrementUseCount("inv_001")
	store.IncrementUseCount("inv_001")

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.UseCount != 2 {
		t.Errorf("UseCount: got %d, want 2", got.UseCount)
	}
	if got.Status != StatusExhausted {
		t.Errorf("Status: got %q, want %q", got.Status, StatusExhausted)
	}
}

func TestInviteStore_IncrementUseCountSingleUse(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	inv.MaxUses = 1
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	store.IncrementUseCount("inv_001")

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.Status != StatusExhausted {
		t.Errorf("Status: got %q, want %q (exhausted when max_uses reached)", got.Status, StatusExhausted)
	}
}

func TestInviteStore_IncrementUseCountNotFound(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	if err := store.IncrementUseCount("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent invite")
	}
}

// --- JSON tests ---

func TestInviteRecord_JSONSnakeCase(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	inv := &InviteRecord{
		ID:        "inv_001",
		Code:      "ABC123",
		Role:      RoleAdmin,
		CreatedBy: "@admin:test",
		CreatedAt: ts,
		MaxUses:   5,
		UseCount:  1,
		Status:    StatusActive,
	}

	data, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to raw map: %v", err)
	}

	snakeCaseFields := []string{
		"created_by", "created_at", "expires_at",
		"max_uses", "use_count", "welcome_message",
	}
	for _, field := range snakeCaseFields {
		if field == "expires_at" || field == "welcome_message" {
			continue // omitempty — may be absent
		}
		if _, ok := raw[field]; !ok {
			t.Errorf("missing snake_case field %q in JSON output", field)
		}
	}

	pascalCaseFields := []string{
		"CreatedBy", "CreatedAt", "ExpiresAt",
		"MaxUses", "UseCount", "WelcomeMessage",
	}
	for _, field := range pascalCaseFields {
		if _, ok := raw[field]; ok {
			t.Errorf("unexpected PascalCase field %q in JSON output", field)
		}
	}
}

func TestInviteRecord_JSONStatusValues(t *testing.T) {
	statuses := []InviteStatus{StatusActive, StatusUsed, StatusExpired, StatusRevoked, StatusExhausted}
	for _, status := range statuses {
		inv := &InviteRecord{ID: "inv_001", Status: status}
		data, err := json.Marshal(inv)
		if err != nil {
			t.Fatalf("Marshal with status %s: %v", status, err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		got, ok := raw["status"].(string)
		if !ok {
			t.Fatalf("status not a string: %T", raw["status"])
		}
		if got != string(status) {
			t.Errorf("status: got %q, want %q", got, status)
		}
	}
}

func TestInviteRecord_JSONNullExpiresAt(t *testing.T) {
	inv := &InviteRecord{
		ID:        "inv_001",
		Code:      "ABC",
		Role:      RoleUser,
		CreatedBy: "@admin:test",
		CreatedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		ExpiresAt: nil,
		MaxUses:   0,
		UseCount:  0,
		Status:    StatusActive,
	}

	data, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	matched := regexp.MustCompile(`"expires_at"`).MatchString(string(data))
	if matched {
		t.Error("expires_at should be omitted when nil (omitempty), but was present")
	}
}

func TestInviteRecord_JSONExpiresAtISO8601(t *testing.T) {
	ts := time.Date(2025, 6, 16, 10, 30, 0, 0, time.UTC)
	inv := &InviteRecord{
		ID:        "inv_001",
		Code:      "ABC",
		CreatedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		ExpiresAt: &ts,
	}

	data, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	val, ok := raw["expires_at"].(string)
	if !ok {
		t.Fatalf("expires_at not a string: %T", raw["expires_at"])
	}
	if _, err := time.Parse(time.RFC3339, val); err != nil {
		t.Errorf("expires_at = %q is not ISO 8601 / RFC3339: %v", val, err)
	}
}

func TestInviteRecord_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	exp := ts.Add(24 * time.Hour)
	original := &InviteRecord{
		ID:        "inv_001",
		Code:      "roundtrip_code",
		Role:      RoleModerator,
		CreatedBy: "@admin:test",
		CreatedAt: ts,
		ExpiresAt: &exp,
		MaxUses:   10,
		UseCount:  3,
		Status:    StatusActive,
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	recovered, err := InviteRecordFromJSON(data)
	if err != nil {
		t.Fatalf("InviteRecordFromJSON: %v", err)
	}
	assertInviteEqual(t, original, recovered)
}

// --- Expiration parsing ---

func TestParseExpiration_AllValues(t *testing.T) {
	tests := []struct {
		input   string
		wantDur time.Duration
		wantNil bool
	}{
		{"1h", 1 * time.Hour, false},
		{"6h", 6 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"never", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseExpiration(tt.input)
			if err != nil {
				t.Fatalf("ParseExpiration(%q): %v", tt.input, err)
			}
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
			} else {
				if got == nil {
					t.Fatal("expected non-nil, got nil")
				}
				remaining := time.Until(*got)
				if remaining < 0 || remaining > tt.wantDur+time.Second {
					t.Errorf("expiration off: remaining=%v, wantDur=%v", remaining, tt.wantDur)
				}
			}
		})
	}
}

func TestParseExpiration_Invalid(t *testing.T) {
	_, err := ParseExpiration("2w")
	if err == nil {
		t.Fatal("expected error for unsupported expiration")
	}
}

func TestParseExpiration_CaseInsensitive(t *testing.T) {
	got, err := ParseExpiration("NEVER")
	if err != nil {
		t.Fatalf("ParseExpiration(\"NEVER\"): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for NEVER, got %v", got)
	}
}

func TestParseExpiration_TrimSpace(t *testing.T) {
	got, err := ParseExpiration("  1h  ")
	if err != nil {
		t.Fatalf("ParseExpiration with spaces: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
}

// --- Code generation ---

func TestGenerateInviteCode_Length(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode: %v", err)
	}
	if len(code) < 16 {
		t.Errorf("code too short: %d chars, want >= 16", len(code))
	}
	if len(code) != 24 {
		t.Errorf("code length: got %d, want 24", len(code))
	}
}

func TestGenerateInviteCode_Base62(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode: %v", err)
	}
	matched, _ := regexp.MatchString(`^[0-9A-Za-z]+$`, code)
	if !matched {
		t.Errorf("code contains non-base62 chars: %q", code)
	}
}

func TestGenerateInviteCode_100CodesUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			t.Fatalf("GenerateInviteCode %d: %v", i, err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate code at iteration %d: %q", i, code)
		}
		seen[code] = struct{}{}
	}
}

// --- Auto-generation in CreateInvite ---

func TestInviteStore_CreateInviteAutoGenerate(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := &InviteRecord{
		Role:      RoleUser,
		CreatedBy: "@admin:test",
		MaxUses:   1,
	}
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	if inv.ID == "" {
		t.Error("expected auto-generated ID")
	}
	if inv.Code == "" {
		t.Error("expected auto-generated code")
	}
	if inv.Status != StatusActive {
		t.Errorf("Status: got %q, want %q", inv.Status, StatusActive)
	}
	if inv.CreatedAt.IsZero() {
		t.Error("expected auto-set CreatedAt")
	}

	got, err := store.GetInviteByCode(inv.Code)
	if err != nil {
		t.Fatalf("GetInviteByCode: %v", err)
	}
	if got.ID != inv.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, inv.ID)
	}
}

// --- Persistence test ---

func TestInviteStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)"

	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	store1, err := NewInviteStore(db1)
	if err != nil {
		db1.Close()
		t.Fatalf("NewInviteStore: %v", err)
	}

	inv := sampleInvite("inv_persist", "persist_code")
	inv.WelcomeMessage = "Welcome!"
	if err := store1.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if err := store1.IncrementUseCount("inv_persist"); err != nil {
		t.Fatalf("IncrementUseCount: %v", err)
	}
	db1.Close()

	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()
	store2, err := NewInviteStore(db2)
	if err != nil {
		t.Fatalf("NewInviteStore db2: %v", err)
	}

	got, err := store2.GetInvite("inv_persist")
	if err != nil {
		t.Fatalf("GetInvite after reopen: %v", err)
	}
	if got.Code != "persist_code" {
		t.Errorf("Code: got %q, want %q", got.Code, "persist_code")
	}
	if got.UseCount != 1 {
		t.Errorf("UseCount: got %d, want 1", got.UseCount)
	}
	if got.WelcomeMessage != "Welcome!" {
		t.Errorf("WelcomeMessage: got %q, want %q", got.WelcomeMessage, "Welcome!")
	}
}

func TestInviteStore_InitSchemaIdempotent(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	if err := store.initSchema(); err != nil {
		t.Fatalf("second initSchema: %v", err)
	}

	inv := sampleInvite("inv_idem", "idem_code")
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite after double initSchema: %v", err)
	}
}

// --- ExpiresAt storage ---

func TestInviteStore_ExpiresAtNullable(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	inv := sampleInvite("inv_001", "code001")
	inv.ExpiresAt = nil
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	got, err := store.GetInvite("inv_001")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil, got %v", got.ExpiresAt)
	}
}

func TestInviteStore_ExpiresAtSet(t *testing.T) {
	db, store := openMemInviteDB(t)
	defer db.Close()

	exp := time.Date(2025, 7, 15, 10, 30, 0, 0, time.UTC)
	inv := sampleInvite("inv_002", "code002")
	inv.ExpiresAt = &exp
	if err := store.CreateInvite(inv); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	got, err := store.GetInvite("inv_002")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt should not be nil")
	}
	if !got.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt: got %v, want %v", *got.ExpiresAt, exp)
	}
}

// --- helpers ---

func assertInviteEqual(t *testing.T, want, got *InviteRecord) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if got.Code != want.Code {
		t.Errorf("Code: got %q, want %q", got.Code, want.Code)
	}
	if got.Role != want.Role {
		t.Errorf("Role: got %q, want %q", got.Role, want.Role)
	}
	if got.CreatedBy != want.CreatedBy {
		t.Errorf("CreatedBy: got %q, want %q", got.CreatedBy, want.CreatedBy)
	}
	if got.MaxUses != want.MaxUses {
		t.Errorf("MaxUses: got %d, want %d", got.MaxUses, want.MaxUses)
	}
	if got.UseCount != want.UseCount {
		t.Errorf("UseCount: got %d, want %d", got.UseCount, want.UseCount)
	}
	if got.Status != want.Status {
		t.Errorf("Status: got %q, want %q", got.Status, want.Status)
	}
	if got.WelcomeMessage != want.WelcomeMessage {
		t.Errorf("WelcomeMessage: got %q, want %q", got.WelcomeMessage, want.WelcomeMessage)
	}
	if (want.ExpiresAt == nil) != (got.ExpiresAt == nil) {
		t.Errorf("ExpiresAt nil mismatch: want=%v got=%v", want.ExpiresAt, got.ExpiresAt)
	}
	if want.ExpiresAt != nil && got.ExpiresAt != nil && !got.ExpiresAt.Equal(*want.ExpiresAt) {
		t.Errorf("ExpiresAt: got %v, want %v", *got.ExpiresAt, *want.ExpiresAt)
	}
}

var _ = fmt.Sprintf
