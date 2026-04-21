package trust

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openMemDB(t *testing.T) (*sql.DB, *DeviceStore) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	store, err := NewDeviceStore(db)
	if err != nil {
		db.Close()
		t.Fatalf("NewDeviceStore: %v", err)
	}
	return db, store
}

func sampleDevice(id string) *DeviceRecord {
	return &DeviceRecord{
		ID:         id,
		Name:       "Test Phone",
		Type:       "phone",
		Platform:   "Android 14",
		TrustState: StateUnverified,
		LastSeen:   time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		FirstSeen:  time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC),
		IPAddress:  "192.168.1.42",
		UserAgent:  "ArmorChat/1.0",
		IsCurrent:  true,
	}
}

func TestDeviceStore_CreateAndGet(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d := sampleDevice("dev_001")
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := store.GetDevice("dev_001")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	assertDeviceEqual(t, d, got)
}

func TestDeviceStore_GetNotFound(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	_, err := store.GetDevice("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent device, got nil")
	}
}

func TestDeviceStore_ListDevices(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d1 := sampleDevice("dev_001")
	d1.LastSeen = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	d2 := sampleDevice("dev_002")
	d2.LastSeen = time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC)

	if err := store.CreateDevice(d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if err := store.CreateDevice(d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	list, err := store.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(list))
	}
	if list[0].ID != "dev_002" {
		t.Errorf("expected first device dev_002 (newer last_seen), got %s", list[0].ID)
	}
}

func TestDeviceStore_ListDevicesEmpty(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	list, err := store.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d devices", len(list))
	}
}

func TestDeviceStore_UpdateDevice(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d := sampleDevice("dev_001")
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	d.Name = "Updated Phone"
	d.Platform = "Android 15"
	d.IsCurrent = false
	if err := store.UpdateDevice(d); err != nil {
		t.Fatalf("UpdateDevice: %v", err)
	}

	got, err := store.GetDevice("dev_001")
	if err != nil {
		t.Fatalf("GetDevice after update: %v", err)
	}
	if got.Name != "Updated Phone" {
		t.Errorf("Name: got %q, want %q", got.Name, "Updated Phone")
	}
	if got.Platform != "Android 15" {
		t.Errorf("Platform: got %q, want %q", got.Platform, "Android 15")
	}
	if got.IsCurrent {
		t.Error("IsCurrent should be false after update")
	}
}

func TestDeviceStore_UpdateDeviceNotFound(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d := sampleDevice("nonexistent")
	if err := store.UpdateDevice(d); err == nil {
		t.Fatal("expected error updating nonexistent device")
	}
}

func TestDeviceStore_UpdateTrustState(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d := sampleDevice("dev_001")
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	if err := store.UpdateTrustState("dev_001", StateVerified); err != nil {
		t.Fatalf("UpdateTrustState: %v", err)
	}

	got, err := store.GetDevice("dev_001")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.TrustState != StateVerified {
		t.Errorf("TrustState: got %q, want %q", got.TrustState, StateVerified)
	}
	if got.VerifiedAt == nil {
		t.Error("VerifiedAt should be set when state becomes verified")
	}
}

func TestDeviceStore_UpdateTrustStateNotFound(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	if err := store.UpdateTrustState("nonexistent", StateVerified); err == nil {
		t.Fatal("expected error for nonexistent device")
	}
}

func TestDeviceStore_UpdateTrustStateRejected(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	d := sampleDevice("dev_001")
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	if err := store.UpdateTrustState("dev_001", StateRejected); err != nil {
		t.Fatalf("UpdateTrustState: %v", err)
	}

	got, err := store.GetDevice("dev_001")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.TrustState != StateRejected {
		t.Errorf("TrustState: got %q, want %q", got.TrustState, StateRejected)
	}
	if got.VerifiedAt != nil {
		t.Error("VerifiedAt should remain nil for rejected state")
	}
}

func TestDeviceRecord_JSONSnakeCase(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	d := &DeviceRecord{
		ID:         "dev_001",
		Name:       "Pixel 8",
		Type:       "phone",
		Platform:   "Android 14",
		TrustState: StateVerified,
		LastSeen:   ts,
		FirstSeen:  ts,
		IPAddress:  "10.0.0.1",
		UserAgent:  "ArmorChat/2.0",
		IsCurrent:  true,
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to raw map: %v", err)
	}

	snakeCaseFields := []string{
		"trust_state", "last_seen", "first_seen", "ip_address",
		"user_agent", "is_current",
	}
	for _, field := range snakeCaseFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing snake_case field %q in JSON output", field)
		}
	}

	pascalCaseFields := []string{
		"TrustState", "LastSeen", "FirstSeen", "IPAddress",
		"UserAgent", "IsCurrent", "trustState", "lastSeen",
	}
	for _, field := range pascalCaseFields {
		if _, ok := raw[field]; ok {
			t.Errorf("unexpected PascalCase/camelCase field %q in JSON output", field)
		}
	}
}

func TestDeviceRecord_JSONTrustStateLowercase(t *testing.T) {
	states := []TrustState{StateVerified, StateUnverified, StatePendingApproval, StateRejected}
	for _, state := range states {
		d := &DeviceRecord{ID: "dev_001", TrustState: state}
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal with state %s: %v", state, err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		got, ok := raw["trust_state"].(string)
		if !ok {
			t.Fatalf("trust_state not a string: %T", raw["trust_state"])
		}
		if got != string(state) {
			t.Errorf("trust_state: got %q, want %q", got, state)
		}
	}
}

func TestDeviceRecord_JSONTimestampsISO8601(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	d := &DeviceRecord{
		ID:        "dev_001",
		FirstSeen: ts,
		LastSeen:  ts,
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, field := range []string{"first_seen", "last_seen"} {
		val, ok := raw[field].(string)
		if !ok {
			t.Fatalf("%s not a string", field)
		}
		if _, err := time.Parse(time.RFC3339, val); err != nil {
			t.Errorf("%s = %q is not ISO 8601 / RFC3339: %v", field, val, err)
		}
	}
}

func TestDeviceRecord_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	original := &DeviceRecord{
		ID:         "dev_001",
		Name:       "Pixel 8",
		Type:       "phone",
		Platform:   "Android 14",
		TrustState: StatePendingApproval,
		LastSeen:   ts,
		FirstSeen:  ts,
		IPAddress:  "10.0.0.1",
		UserAgent:  "ArmorChat/2.0",
		IsCurrent:  true,
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	recovered, err := DeviceRecordFromJSON(data)
	if err != nil {
		t.Fatalf("DeviceRecordFromJSON: %v", err)
	}
	assertDeviceEqual(t, original, recovered)
}

func TestDeviceStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")

	dsn := dbPath + "?_pragma=journal_mode(WAL)"
	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	store1, err := NewDeviceStore(db1)
	if err != nil {
		db1.Close()
		t.Fatalf("NewDeviceStore: %v", err)
	}

	d := sampleDevice("dev_persist")
	d.VerifiedAt = nil
	if err := store1.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	if err := store1.UpdateTrustState("dev_persist", StateVerified); err != nil {
		t.Fatalf("UpdateTrustState: %v", err)
	}
	db1.Close()

	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()
	store2, err := NewDeviceStore(db2)
	if err != nil {
		t.Fatalf("NewDeviceStore db2: %v", err)
	}

	got, err := store2.GetDevice("dev_persist")
	if err != nil {
		t.Fatalf("GetDevice after reopen: %v", err)
	}
	if got.ID != "dev_persist" {
		t.Errorf("ID: got %q, want %q", got.ID, "dev_persist")
	}
	if got.TrustState != StateVerified {
		t.Errorf("TrustState: got %q, want %q", got.TrustState, StateVerified)
	}
	if got.Name != "Test Phone" {
		t.Errorf("Name: got %q, want %q", got.Name, "Test Phone")
	}
	if got.VerifiedAt == nil {
		t.Error("VerifiedAt should survive persistence")
	}
}

func TestDeviceStore_InitSchemaIdempotent(t *testing.T) {
	db, store := openMemDB(t)
	defer db.Close()

	if err := store.initSchema(); err != nil {
		t.Fatalf("second initSchema: %v", err)
	}

	d := sampleDevice("dev_idem")
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("CreateDevice after double initSchema: %v", err)
	}
}

func assertDeviceEqual(t *testing.T, want, got *DeviceRecord) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Type != want.Type {
		t.Errorf("Type: got %q, want %q", got.Type, want.Type)
	}
	if got.Platform != want.Platform {
		t.Errorf("Platform: got %q, want %q", got.Platform, want.Platform)
	}
	if got.TrustState != want.TrustState {
		t.Errorf("TrustState: got %q, want %q", got.TrustState, want.TrustState)
	}
	if got.IPAddress != want.IPAddress {
		t.Errorf("IPAddress: got %q, want %q", got.IPAddress, want.IPAddress)
	}
	if got.UserAgent != want.UserAgent {
		t.Errorf("UserAgent: got %q, want %q", got.UserAgent, want.UserAgent)
	}
	if got.IsCurrent != want.IsCurrent {
		t.Errorf("IsCurrent: got %v, want %v", got.IsCurrent, want.IsCurrent)
	}
}
