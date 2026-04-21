package rpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/armorclaw/bridge/pkg/trust"
)

func newTestDeviceStore(t *testing.T) *trust.DeviceStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := trust.NewDeviceStore(db)
	if err != nil {
		t.Fatalf("new device store: %v", err)
	}
	return store
}

func seedDevice(t *testing.T, store *trust.DeviceStore, id string, state trust.TrustState) *trust.DeviceRecord {
	t.Helper()
	d := &trust.DeviceRecord{
		ID:         id,
		Name:       "Test Device",
		Type:       "phone",
		Platform:   "android",
		TrustState: state,
		LastSeen:   time.Now().UTC(),
		FirstSeen:  time.Now().UTC(),
	}
	if err := store.CreateDevice(d); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	return d
}

func newServerWithDeviceStore(t *testing.T, store *trust.DeviceStore) *Server {
	t.Helper()
	return &Server{
		handlers:    make(map[string]HandlerFunc, 32),
		deviceStore: store,
	}
}

func TestDeviceHandlerRegistration(t *testing.T) {
	server := &Server{}
	server.registerHandlers()

	methods := []string{"device.list", "device.get", "device.approve", "device.reject"}
	for _, m := range methods {
		if _, ok := server.handlers[m]; !ok {
			t.Errorf("handler %q not registered", m)
		}
	}
}

func TestDeviceList(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	s.registerHandlers()

	result, rpcErr := s.handleDeviceList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	devices, ok := result.([]*trust.DeviceRecord)
	if !ok {
		t.Fatalf("expected []*trust.DeviceRecord, got %T", result)
	}
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}

	seedDevice(t, store, "dev-1", trust.StateUnverified)
	seedDevice(t, store, "dev-2", trust.StateVerified)

	result, rpcErr = s.handleDeviceList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	devices = result.([]*trust.DeviceRecord)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
}

func TestDeviceListStoreNil(t *testing.T) {
	s := newServerWithDeviceStore(t, nil)
	_, rpcErr := s.handleDeviceList(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr == nil || rpcErr.Code != InternalError {
		t.Fatalf("expected InternalError, got %v", rpcErr)
	}
}

func TestDeviceGet(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	seedDevice(t, store, "dev-42", trust.StateUnverified)

	result, rpcErr := s.handleDeviceGet(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-42"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	device, ok := result.(*trust.DeviceRecord)
	if !ok {
		t.Fatalf("expected *trust.DeviceRecord, got %T", result)
	}
	if device.ID != "dev-42" {
		t.Fatalf("expected device id dev-42, got %s", device.ID)
	}
}

func TestDeviceGetMissingID(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceGet(context.Background(), &Request{
		Params: json.RawMessage(`{}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestDeviceGetNotFound(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceGet(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"nonexistent"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError (-32000), got %v", rpcErr)
	}
	if rpcErr.Message != "device not found" {
		t.Fatalf("expected 'device not found', got %s", rpcErr.Message)
	}
}

func TestDeviceGetBadParams(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceGet(context.Background(), &Request{
		Params: json.RawMessage(`invalid json`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams, got %v", rpcErr)
	}
}

func TestDeviceApprove(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	seedDevice(t, store, "dev-1", trust.StateUnverified)

	result, rpcErr := s.handleDeviceApprove(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-1","approved_by":"admin"}`),
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

	device, _ := store.GetDevice("dev-1")
	if device.TrustState != trust.StateVerified {
		t.Fatalf("expected state verified, got %s", device.TrustState)
	}
}

func TestDeviceApproveIdempotent(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	seedDevice(t, store, "dev-1", trust.StateVerified)

	result, rpcErr := s.handleDeviceApprove(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-1","approved_by":"admin"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error on idempotent approve: %v", rpcErr)
	}
	resp := result.(SuccessResponse)
	if !resp.Success {
		t.Fatal("expected success=true for idempotent approve")
	}
}

func TestDeviceApproveMissingDeviceID(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceApprove(context.Background(), &Request{
		Params: json.RawMessage(`{"approved_by":"admin"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams for missing device_id, got %v", rpcErr)
	}
}

func TestDeviceApproveMissingApprovedBy(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceApprove(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-1"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams for missing approved_by, got %v", rpcErr)
	}
}

func TestDeviceApproveNotFound(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceApprove(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"nonexistent","approved_by":"admin"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError, got %v", rpcErr)
	}
}

func TestDeviceReject(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	seedDevice(t, store, "dev-1", trust.StateUnverified)

	result, rpcErr := s.handleDeviceReject(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-1","rejected_by":"admin","reason":"suspicious"}`),
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

	device, _ := store.GetDevice("dev-1")
	if device.TrustState != trust.StateRejected {
		t.Fatalf("expected state rejected, got %s", device.TrustState)
	}
}

func TestDeviceRejectIdempotent(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)
	seedDevice(t, store, "dev-1", trust.StateRejected)

	result, rpcErr := s.handleDeviceReject(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"dev-1","rejected_by":"admin"}`),
	})
	if rpcErr != nil {
		t.Fatalf("unexpected error on idempotent reject: %v", rpcErr)
	}
	resp := result.(SuccessResponse)
	if !resp.Success {
		t.Fatal("expected success=true for idempotent reject")
	}
}

func TestDeviceRejectMissingDeviceID(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceReject(context.Background(), &Request{
		Params: json.RawMessage(`{"rejected_by":"admin"}`),
	})
	if rpcErr == nil || rpcErr.Code != InvalidParams {
		t.Fatalf("expected InvalidParams for missing device_id, got %v", rpcErr)
	}
}

func TestDeviceRejectNotFound(t *testing.T) {
	store := newTestDeviceStore(t)
	s := newServerWithDeviceStore(t, store)

	_, rpcErr := s.handleDeviceReject(context.Background(), &Request{
		Params: json.RawMessage(`{"device_id":"nonexistent","rejected_by":"admin"}`),
	})
	if rpcErr == nil || rpcErr.Code != NotFoundError {
		t.Fatalf("expected NotFoundError, got %v", rpcErr)
	}
}
