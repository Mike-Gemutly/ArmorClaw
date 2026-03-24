package rpc

import (
	"errors"
	"testing"

	"github.com/armorclaw/bridge/pkg/trust"
)

type mockGateStore struct {
	isReady bool
	err     error
}

func (m *mockGateStore) Get(userID string) (*trust.UserHardeningState, error) {
	return nil, nil
}

func (m *mockGateStore) Put(state *trust.UserHardeningState) error {
	return nil
}

func (m *mockGateStore) IsDelegationReady(userID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.isReady, nil
}

func (m *mockGateStore) AckStep(userID string, step trust.HardeningStep) error {
	return nil
}

func TestRequireDelegationReady_WhenReady_ReturnsNil(t *testing.T) {
	store := &mockGateStore{isReady: true}
	userID := "test-user-123"

	err := RequireDelegationReady(store, userID)

	if err != nil {
		t.Errorf("Expected nil when delegation ready, got: %v", err)
	}
}

func TestRequireDelegationReady_WhenNotReady_ReturnsErrHardeningRequired(t *testing.T) {
	store := &mockGateStore{isReady: false}
	userID := "test-user-123"

	err := RequireDelegationReady(store, userID)

	if err != ErrHardeningRequired {
		t.Errorf("Expected ErrHardeningRequired when not ready, got: %v", err)
	}
}

func TestRequireDelegationReady_WhenStoreReturnsError_WrapsError(t *testing.T) {
	expectedErr := errors.New("store connection failed")
	store := &mockGateStore{err: expectedErr}
	userID := "test-user-123"

	err := RequireDelegationReady(store, userID)

	if err == nil {
		t.Fatal("Expected error when store returns error, got nil")
	}

	if !errors.Is(err, expectedErr) && err.Error() != "failed to check delegation ready: store connection failed" {
		t.Errorf("Expected wrapped error, got: %v", err)
	}
}

func TestRequireDelegationReady_WhenStoreNil_AllowsAction(t *testing.T) {
	var store trust.Store = nil
	userID := "test-user-123"

	err := RequireDelegationReady(store, userID)

	if err != nil {
		t.Errorf("Expected nil when store is nil (graceful degradation), got: %v", err)
	}
}

func TestExtractUserIDFromParams_WithUserID_ReturnsUserID(t *testing.T) {
	params := []byte(`{"user_id": "@user:example.com", "name": "test"}`)

	userID := extractUserIDFromParams(params)

	if userID != "@user:example.com" {
		t.Errorf("Expected '@user:example.com', got: %s", userID)
	}
}

func TestExtractUserIDFromParams_WithoutUserID_ReturnsEmpty(t *testing.T) {
	params := []byte(`{"name": "test", "other": "value"}`)

	userID := extractUserIDFromParams(params)

	if userID != "" {
		t.Errorf("Expected empty string, got: %s", userID)
	}
}

func TestExtractUserIDFromParams_EmptyParams_ReturnsEmpty(t *testing.T) {
	var params []byte = nil

	userID := extractUserIDFromParams(params)

	if userID != "" {
		t.Errorf("Expected empty string for nil params, got: %s", userID)
	}
}

func TestExtractUserIDFromParams_InvalidJSON_ReturnsEmpty(t *testing.T) {
	params := []byte(`{invalid json`)

	userID := extractUserIDFromParams(params)

	if userID != "" {
		t.Errorf("Expected empty string for invalid JSON, got: %s", userID)
	}
}

// TestDelegationGate is a comprehensive test suite for the delegation gate functionality
func TestDelegationGate(t *testing.T) {
	t.Run("Ready_AllowsAction", func(t *testing.T) {
		store := &mockGateStore{isReady: true}
		err := RequireDelegationReady(store, "test-user")
		if err != nil {
			t.Errorf("Expected nil when ready, got: %v", err)
		}
	})

	t.Run("NotReady_BlocksAction", func(t *testing.T) {
		store := &mockGateStore{isReady: false}
		err := RequireDelegationReady(store, "test-user")
		if err != ErrHardeningRequired {
			t.Errorf("Expected ErrHardeningRequired, got: %v", err)
		}
	})

	t.Run("StoreError_WrapsAndReturns", func(t *testing.T) {
		storeErr := errors.New("store error")
		store := &mockGateStore{err: storeErr}
		err := RequireDelegationReady(store, "test-user")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("NilStore_AllowsAction", func(t *testing.T) {
		err := RequireDelegationReady(nil, "test-user")
		if err != nil {
			t.Errorf("Expected nil for nil store, got: %v", err)
		}
	})
}
