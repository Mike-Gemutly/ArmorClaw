package lockdown

import (
	"path/filepath"
	"testing"

	"github.com/armorclaw/bridge/pkg/qr"
)

func TestEnterReadmin(t *testing.T) {
	// Create temp file for state
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Follow proper transition chain to reach operational
	// Path: lockdown -> bonding -> configuring -> hardening -> operational -> readmin
	if err := m.Transition(ModeBonding); err != nil {
		t.Fatalf("Failed to set bonding mode: %v", err)
	}
	if err := m.Transition(ModeConfiguring); err != nil {
		t.Fatalf("Failed to set configuring mode: %v", err)
	}
	if err := m.Transition(ModeHardening); err != nil {
		t.Fatalf("Failed to set hardening mode: %v", err)
	}
	if err := m.Transition(ModeOperational); err != nil {
		t.Fatalf("Failed to set operational mode: %v", err)
	}

	// Test: EnterReadmin() succeeds from ModeOperational
	if err := m.EnterReadmin(); err != nil {
		t.Errorf("EnterReadmin() should succeed from operational mode, got: %v", err)
	}

	// Verify mode changed
	if got := m.GetState().Mode; got != ModeReadmin {
		t.Errorf("Expected mode %s, got %s", ModeReadmin, got)
	}
}

func TestEnterReadminFailsFromNonOperational(t *testing.T) {
	// Create temp file for state
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test: EnterReadmin() fails from non-operational modes
	if err := m.EnterReadmin(); err == nil {
		t.Error("EnterReadmin() should fail from non-operational mode")
	}
}

func TestExitReadmin(t *testing.T) {
	// Create temp file for state
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Follow transition chain to reach readmin
	if err := m.Transition(ModeBonding); err != nil {
		t.Fatalf("Failed to set bonding mode: %v", err)
	}
	if err := m.Transition(ModeConfiguring); err != nil {
		t.Fatalf("Failed to set configuring mode: %v", err)
	}
	if err := m.Transition(ModeHardening); err != nil {
		t.Fatalf("Failed to set hardening mode: %v", err)
	}
	if err := m.Transition(ModeOperational); err != nil {
		t.Fatalf("Failed to set operational mode: %v", err)
	}
	if err := m.EnterReadmin(); err != nil {
		t.Fatalf("Failed to enter readmin: %v", err)
	}

	// Test: ExitReadmin() transitions to ModeConfiguring
	if err := m.ExitReadmin(); err != nil {
		t.Errorf("ExitReadmin() should succeed, got: %v", err)
	}

	// Verify mode changed
	if got := m.GetState().Mode; got != ModeConfiguring {
		t.Errorf("Expected mode %s, got %s", ModeConfiguring, got)
	}
}

func TestReadminManagerInit(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create mock dependencies
	qrMgr := qr.NewQRManager([]byte("test"), qr.DefaultQRConfig(), "http://test.com", "http://bridge.com", "test-server")

	mockKS := &mockKeystore{}
	mockMatrix := &mockMatrixPasswordChanger{}

	// Test: NewReadminManager creates a valid manager
	rm := NewReadminManager(m, qrMgr, mockKS, mockMatrix)
	if rm == nil {
		t.Fatal("NewReadminManager returned nil")
	}
	if rm.lockdown == nil {
		t.Error("Lockdown manager not set")
	}
	if rm.qrManager == nil {
		t.Error("QR manager not set")
	}
	if rm.keystore == nil {
		t.Error("Keystore not set")
	}
	if rm.matrix == nil {
		t.Error("Matrix not set")
	}
}

type mockKeystore struct{}

func (m *mockKeystore) WipeAllData() error {
	return nil
}

type mockMatrixPasswordChanger struct{}

func (m *mockMatrixPasswordChanger) ChangePassword(newPassword string, logoutDevices bool) error {
	return nil
}
