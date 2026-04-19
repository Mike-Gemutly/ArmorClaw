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
	"fmt"
	"sync"
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
		return nil, fmt.Errorf("provisioner intentionally failing")
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

type mockVaultClient struct {
	mu                    sync.Mutex
	issueCalls            []mockIssueCall
	zeroizeCalls          []mockZeroizeCall
	issueErr              error
	zeroizeErr            error
	issueTokenID          string
	zeroizeDestroyedCount uint32
}

type mockIssueCall struct {
	SessionID string
	ToolName  string
	Secret    string
}

type mockZeroizeCall struct {
	ToolName  string
	SessionID string
}

func (m *mockVaultClient) IssueBlindFillToken(_ context.Context, sessionID, toolName, secret string, _ time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issueCalls = append(m.issueCalls, mockIssueCall{SessionID: sessionID, ToolName: toolName, Secret: secret})
	if m.issueErr != nil {
		return "", m.issueErr
	}
	return m.issueTokenID, nil
}

func (m *mockVaultClient) ZeroizeToolSecrets(_ context.Context, toolName, sessionID string) (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zeroizeCalls = append(m.zeroizeCalls, mockZeroizeCall{ToolName: toolName, SessionID: sessionID})
	if m.zeroizeErr != nil {
		return 0, m.zeroizeErr
	}
	return m.zeroizeDestroyedCount, nil
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

func TestExecuteTool_V6MicrokernelIssuesAndZeroizes(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_v6.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{issueTokenID: "tok_abc123", zeroizeDestroyedCount: 2}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"secret": "s3cret_val"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "v6_test_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success, got error: %v", resp)
	}

	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.issueCalls) != 1 {
		t.Errorf("Expected 1 issue call, got %d", len(mv.issueCalls))
	}
	if len(mv.issueCalls) > 0 {
		if mv.issueCalls[0].ToolName != "test_tool" {
			t.Errorf("Expected tool test_tool, got %s", mv.issueCalls[0].ToolName)
		}
		if mv.issueCalls[0].Secret != "s3cret_val" {
			t.Errorf("Expected secret s3cret_val, got %s", mv.issueCalls[0].Secret)
		}
	}
	if len(mv.zeroizeCalls) != 1 {
		t.Errorf("Expected 1 zeroize call, got %d", len(mv.zeroizeCalls))
	}
	if len(mv.zeroizeCalls) > 0 && mv.zeroizeCalls[0].ToolName != "test_tool" {
		t.Errorf("Expected zeroize tool test_tool, got %s", mv.zeroizeCalls[0].ToolName)
	}
}

func TestExecuteTool_V6MicrokernelOffSkipsVault(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_no_v6.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"secret": "s3cret_val"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "no_v6_test",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success, got error: %v", resp)
	}

	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.issueCalls) != 0 {
		t.Errorf("Expected 0 issue calls with v6 off, got %d", len(mv.issueCalls))
	}
	if len(mv.zeroizeCalls) != 0 {
		t.Errorf("Expected 0 zeroize calls with v6 off, got %d", len(mv.zeroizeCalls))
	}
}

func TestExecuteTool_NilVaultClientSkipsGracefully(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_nil_vault.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    nil,
		V6Microkernel:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"secret": "s3cret_val"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "nil_vault_test",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success despite nil vault client, got error: %v", resp)
	}
}

func TestExecuteTool_VaultIssueErrorDegradesGracefully(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_vault_err.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{issueErr: fmt.Errorf("vault unavailable"), zeroizeDestroyedCount: 0}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"secret": "s3cret_val"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "vault_err_test",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success despite vault error, got error: %v", resp)
	}

	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.issueCalls) != 1 {
		t.Errorf("Expected 1 issue call (attempted), got %d", len(mv.issueCalls))
	}
	if len(mv.zeroizeCalls) != 1 {
		t.Errorf("Expected 1 zeroize call (cleanup still runs), got %d", len(mv.zeroizeCalls))
	}
}

// TestVaultClient verifies that a non-nil VaultClient is wired through Config
// and that the router calls IssueBlindFillToken and ZeroizeToolSecrets when v6 is enabled.
func TestVaultClient(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_vault_wiring.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{issueTokenID: "tok_wire_42", zeroizeDestroyedCount: 1}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create router with VaultClient: %v", err)
	}

	if router.vaultClient == nil {
		t.Fatal("Expected vaultClient to be non-nil on router after construction")
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"api_key": "sk-test-123"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "vault_wiring_test",
		Params:  &MCPParams{Name: "fill_form", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success, got error: %v", resp)
	}

	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.issueCalls) != 1 {
		t.Errorf("Expected 1 IssueBlindFillToken call, got %d", len(mv.issueCalls))
	}
	if len(mv.zeroizeCalls) != 1 {
		t.Errorf("Expected 1 ZeroizeToolSecrets call, got %d", len(mv.zeroizeCalls))
	}
	if len(mv.issueCalls) > 0 && mv.issueCalls[0].Secret != "sk-test-123" {
		t.Errorf("Expected secret 'sk-test-123', got %s", mv.issueCalls[0].Secret)
	}
}

func TestAuditMode_ToolCallsPassThroughUnmodified(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_mode.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{issueTokenID: "tok_audit", zeroizeDestroyedCount: 0}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  true,
		V6AuditMode:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{
		"email": "user@example.com",
		"ssn":   "123-45-6789",
	})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "audit_test_1",
		Params:  &MCPParams{Name: "pii_request", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success in audit mode, got error: %v", resp)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if result["status"] != "audit_logged" {
		t.Errorf("Expected status 'audit_logged', got %v", result["status"])
	}

	mv.mu.Lock()
	if len(mv.issueCalls) != 0 {
		t.Errorf("Expected 0 vault issue calls in audit mode, got %d", len(mv.issueCalls))
	}
	if len(mv.zeroizeCalls) != 0 {
		t.Errorf("Expected 0 vault zeroize calls in audit mode, got %d", len(mv.zeroizeCalls))
	}
	mv.mu.Unlock()

	if mockProv.spawned {
		t.Error("Expected NO ToolSidecar to be spawned in audit mode")
	}

	entries, _ := router.auditor.Query(audit.QueryParams{Limit: 10})
	if len(entries) == 0 {
		t.Error("Expected audit log entries in audit mode")
	}
}

func TestAuditMode_RequiresV6Microkernel(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_no_mk.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{issueTokenID: "tok_no_mk", zeroizeDestroyedCount: 0}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  false,
		V6AuditMode:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "no_mk_audit",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success on legacy path, got error: %v", resp)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if result["status"] == "audit_logged" {
		t.Error("Audit mode should NOT activate when v6_microkernel=false")
	}

	if !mockProv.spawned {
		t.Error("Expected ToolSidecar to be spawned on legacy path")
	}
}

func TestAuditMode_BothFlagsFalse_LegacyUnchanged(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_both_false.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})
	mockProv := &mockProvisioner{}
	mv := &mockVaultClient{}

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		VaultClient:    mv,
		V6Microkernel:  false,
		V6AuditMode:    false,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	args, _ := json.Marshal(map[string]interface{}{"query": "legacy test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "legacy_both_false",
		Params:  &MCPParams{Name: "legacy_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success on legacy path, got error: %v", resp)
	}

	mv.mu.Lock()
	if len(mv.issueCalls) != 0 {
		t.Errorf("Expected 0 vault calls with both flags false, got %d issue calls", len(mv.issueCalls))
	}
	if len(mv.zeroizeCalls) != 0 {
		t.Errorf("Expected 0 vault calls with both flags false, got %d zeroize calls", len(mv.zeroizeCalls))
	}
	mv.mu.Unlock()

	if !mockProv.spawned {
		t.Error("Expected ToolSidecar to be spawned on legacy path with both flags false")
	}
}
