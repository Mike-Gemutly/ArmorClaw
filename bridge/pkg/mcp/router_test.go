// Package mcp provides routing for MCP (Model Context Protocol) tool calls through SkillGate.
//
// Resolves: Task 8 - MCP Router with SkillGate
//
// Test suite covers:
// - SkillGate integration
// - Consent workflow for PII operations
// - ToolSidecar provisioning
// - Audit logging with PII redaction
// - Error handling
package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/governor"
	"github.com/armorclaw/bridge/pkg/interfaces"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
	"github.com/armorclaw/bridge/pkg/toolsidecar"
)

type mockProvisioner struct {
	shouldFail bool
	spawned    bool
	toolName   string
	sessionID  string
}

 func (m *mockProvisioner) SpawnToolSidecar(ctx context.Context, skillName, sessionID string) (*toolsidecar.ToolSidecar, error) {
	if m.shouldFail {
        return nil
    }
}
	m.spawned = true
	m.toolName = skillName
	m.sessionID = sessionID
	return &toolsidecar.ToolSidecar{
		ID:        "mock_container_id_123456789012",
		SkillName: skillName,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		Status:    "running",
	}, nil
}

func (m *mockProvisioner) StopToolSidecar(ctx context.Context, containerID string) error {
	return nil
}

// createTestRouter creates a router with all dependencies properly initialized
func createTestRouter(t *testing.T) (*MCPRouter, *mockProvisioner) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	mockProv := &mockProvisioner{}

	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{
		Timeout: 60 * time.Second,
	})

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	return router, mockProv
}

// TestNewRouter validates router creation with proper config
func TestNewRouter_MissingSkillGate(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	mockProv := &mockProvisioner{}

	// Test missing SkillGate
	_, err := New(Config{
		SkillGate:   nil,
		Provisioner: mockProv,
		ConsentManager: pii.NewHITLConsentManager(pii.HITLConfig{
			Timeout: 60 * time.Second,
		}),
		Auditor: mockAuditLog,
		Logger:  logger.Global(),
	})
	if err == nil {
		t.Error("Expected error for missing SkillGate")
	}

	// Test missing Provisioner
	_, err = New(Config{
		SkillGate:   mockGovernor,
		Provisioner: nil,
		ConsentManager: pii.NewHITLConsentManager(pii.HITLConfig{
			Timeout: 60 * time.Second,
		}),
		Auditor: mockAuditLog,
		Logger:  logger.Global(),
	})
	if err == nil {
		t.Error("Expected error for missing Provisioner")
	}

	// Test missing ConsentManager
	_, err = New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    &mockProvisioner{},
		ConsentManager: nil,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
	})
	if err == nil {
		t.Error("Expected error for missing ConsentManager")
	}

	// Test missing Auditor
	_, err = New(Config{
		SkillGate:   mockGovernor,
		Provisioner: &mockProvisioner{},
		ConsentManager: pii.NewHITLConsentManager(pii.HITLConfig{
			Timeout: 60 * time.Second,
		}),
		Auditor: nil,
		Logger:  logger.Global(),
	})
	if err == nil {
		t.Error("Expected error for missing Auditor")
	}
}

// TestHandleToolsCall_SkillGateValidation tests the SkillGate integration with tool execution
func TestHandleToolsCall_SkillGateValidation(t *testing.T) {
	router, mockProv := createTestRouter(t)
	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"query": "test query"})

	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "test_id_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil || resp.JSONRPC != "2.0" || resp.Error != nil {
		t.Error("Expected successful response")
	}

	if !mockProv.spawned || mockProv.toolName != "test_tool" {
		t.Error("Expected ToolSidecar to be provisioned")
	}

	entries, _ := router.auditor.Query(audit.QueryParams{Limit: 10})
	if len(entries) == 0 {
		t.Error("Expected audit entries")
	}
}

// TestHandleToolsCall_ToolSidecarProvisioning tests ToolSidecar provisioning
func TestHandleToolsCall_ToolSidecarProvisioning(t *testing.T) {
	router, mockProv := createTestRouter(t)
	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"query": "test query"})

	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "test_id_2",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil || resp.JSONRPC != "2.0" || resp.Error != nil {
		t.Error("Expected successful response")
	}

	if !mockProv.spawned || mockProv.toolName != "test_tool" {
		t.Error("Expected ToolSidecar to be provisioned correctly")
	}
}

// TestHandleToolsCall_ToolSidecarFailure tests ToolSidecar failure handling
func TestHandleToolsCall_ToolSidecarFailure(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{
		Timeout: 60 * time.Second,
	})

	// Create a failing mock provisioner
	failingProv := &mockProvisioner{shouldFail: true}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    failingProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"query": "test query"})

	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "test_id_3",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil || resp.Error == nil || resp.Error.Message != "Failed to spawn tool container" {
		t.Error("Expected error response")
	}
}

// TestRequiresConsent tests consent requirement logic
func TestRequiresConsent(t *testing.T) {
	router, _ := createTestRouter(t)

	tests := []struct {
		name            string
		sanitizedArgs   map[string]interface{}
		originalArgs    map[string]interface{}
		toolName        string
		expectedConsent bool
	}{
		{
			name:            "No PII - no consent required",
			sanitizedArgs:   map[string]interface{}{"query": "test"},
			originalArgs:    map[string]interface{}{"query": "test"},
			toolName:        "search",
			expectedConsent: false,
		},
		{
			name:            "Redacted PII - consent required",
			sanitizedArgs:   map[string]interface{}{"email": "[REDACTED:abc123]"},
			originalArgs:    map[string]interface{}{"email": "user@example.com"},
			toolName:        "pii_tool",
			expectedConsent: true,
		},
		{
			name:            "PII-sensitive tool - consent required",
			sanitizedArgs:   map[string]interface{}{"query": "test"},
			originalArgs:    map[string]interface{}{"query": "test"},
			toolName:        "pii_request",
			expectedConsent: true,
		},
		{
			name:            "Payment tool - consent required",
			sanitizedArgs:   map[string]interface{}{"amount": 100},
			originalArgs:    map[string]interface{}{"amount": 100},
			toolName:        "payment_fill",
			expectedConsent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizedCall := &interfaces.ToolCall{
				ID:        "test_id",
				ToolName:  tt.toolName,
				Arguments: tt.sanitizedArgs,
			}
			originalCall := &interfaces.ToolCall{
				ID:        "test_id",
				ToolName:  tt.toolName,
				Arguments: tt.originalArgs,
			}

			result := router.requiresConsent(sanitizedCall, originalCall)
			if result != tt.expectedConsent {
				t.Errorf("Expected consent=%v, got %v", tt.expectedConsent, result)
			}
		})
	}
}

// TestDetermineSensitivity tests sensitivity level determination
func TestDetermineSensitivity(t *testing.T) {
	router, _ := createTestRouter(t)

	tests := []struct {
		name     string
		key      string
		expected pii.SensitivityLevel
	}{
		{"Email", "email", pii.SensitivityMedium},
		{"SSN", "ssn", pii.SensitivityCritical},
		{"Credit Card", "credit_card", pii.SensitivityCritical},
		{"Card Number", "card_number", pii.SensitivityCritical},
		{"Password", "password", pii.SensitivityCritical},
		{"API Key", "api_key", pii.SensitivityCritical},
		{"Unknown", "unknown_field", pii.SensitivityLow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.determineSensitivity(tt.key, "value")
			if result != tt.expected {
				t.Errorf("Expected sensitivity %s, got %s", tt.expected, result)
			}
		})
	}
}
