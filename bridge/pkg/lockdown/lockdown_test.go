package lockdown

import (
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "lockdown.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() returned error: %v", err)
	}

	if m.GetMode() != ModeLockdown {
		t.Errorf("Expected initial mode %s, got %s", ModeLockdown, m.GetMode())
	}
}

func TestModeTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "lockdown.json")

	cfg := Config{StateFile: stateFile}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() returned error: %v", err)
	}

	// Test valid transition: lockdown -> bonding
	if err := m.Transition(ModeBonding); err != nil {
		t.Errorf("Transition to bonding failed: %v", err)
	}

	// Test valid transition: bonding -> configuring
	if err := m.Transition(ModeConfiguring); err != nil {
		t.Errorf("Transition to configuring failed: %v", err)
	}

	// Test invalid transition: configuring -> lockdown (should fail)
	if err := m.Transition(ModeLockdown); err == nil {
		t.Error("Expected transition to lockdown to fail")
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		current Mode
		target  Mode
		valid   bool
	}{
		{ModeLockdown, ModeBonding, true},
		{ModeLockdown, ModeConfiguring, false},
		{ModeBonding, ModeConfiguring, true},
		{ModeBonding, ModeLockdown, true},
		{ModeConfiguring, ModeHardening, true},
		{ModeHardening, ModeOperational, true},
		{ModeOperational, ModeLockdown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.current)+"->"+string(tt.target), func(t *testing.T) {
			tmpDir := t.TempDir()
			cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
			m, _ := NewManager(cfg)

			// Set the mode directly for testing
			m.state.mu.Lock()
			m.state.Mode = tt.current
			m.state.mu.Unlock()

			err := m.CanTransition(tt.target)
			if tt.valid && err != nil {
				t.Errorf("Expected transition to be valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected transition to be invalid")
			}
		})
	}
}

func TestValidateForOperational_VariableShadowing(t *testing.T) {
	// This test verifies the fix for the variable shadowing bug
	// where 'var errors []string' shadowed the 'errors' package
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	m, _ := NewManager(cfg)

	// Should return an error listing all missing prerequisites
	err := m.ValidateForOperational()
	if err == nil {
		t.Error("Expected error for incomplete setup")
	}

	// The error message should contain validation details
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("Error message should not be empty")
	}
}

func TestStatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "lockdown.json")

	cfg := Config{StateFile: stateFile}
	m1, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() returned error: %v", err)
	}

	// Set admin established
	m1.state.mu.Lock()
	m1.state.AdminEstablished = true
	m1.state.AdminID = "admin_123"
	m1.save() // save() must be called while holding the lock
	m1.state.mu.Unlock()

	// Create new manager and verify state is loaded
	m2, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() on reload returned error: %v", err)
	}

	if !m2.IsAdminEstablished() {
		t.Error("Expected admin to be established after reload")
	}
}

func TestGetState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	m, _ := NewManager(cfg)

	state := m.GetState()

	if state.Mode != ModeLockdown {
		t.Errorf("Expected mode %s, got %s", ModeLockdown, state.Mode)
	}

	if state.SingleDeviceMode != true {
		t.Error("Expected single device mode to be true")
	}
}

func TestIsCommunicationAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{StateFile: filepath.Join(tmpDir, "lockdown.json")}
	m, _ := NewManager(cfg)

	// Unix should be allowed by default
	if !m.IsCommunicationAllowed("unix") {
		t.Error("Expected unix communication to be allowed")
	}

	// TCP should not be allowed by default
	if m.IsCommunicationAllowed("tcp") {
		t.Error("Expected tcp communication to not be allowed")
	}

	// Add tcp and verify
	m.AddAllowedCommunication("tcp")
	if !m.IsCommunicationAllowed("tcp") {
		t.Error("Expected tcp communication to be allowed after adding")
	}
}
