package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/capability"
	"github.com/armorclaw/bridge/pkg/governor"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

type mockCapabilityBroker struct {
	deny   bool
	err    error
	called bool
	action string
}

func (m *mockCapabilityBroker) Authorize(_ context.Context, req capability.ActionRequest) (capability.ActionResponse, error) {
	m.called = true
	m.action = req.Action
	if m.err != nil {
		return capability.ActionResponse{Allowed: false, Classification: capability.RiskDeny, Reason: m.err.Error()}, m.err
	}
	if m.deny {
		return capability.ActionResponse{
			Allowed:        false,
			Classification: capability.RiskDeny,
			Reason:         fmt.Sprintf("action %q not permitted", req.Action),
		}, nil
	}
	return capability.ActionResponse{
		Allowed:        true,
		Classification: capability.RiskAllow,
	}, nil
}

func createTestRouterWithAuthorizer(t *testing.T, broker *mockCapabilityBroker) (*MCPRouter, *mockProvisioner) {
	t.Helper()
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_auth.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	mockProv := &mockProvisioner{}
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		Authorizer:     broker,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}
	return router, mockProv
}

func TestHandleToolsCall_AuthorizerDeny(t *testing.T) {
	broker := &mockCapabilityBroker{deny: true}
	router, mockProv := createTestRouterWithAuthorizer(t, broker)

	args, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "auth_deny_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error == nil {
		t.Fatal("Expected error response when broker denies")
	}
	if resp.Error.Message != "Capability denied" {
		t.Errorf("Expected 'Capability denied' message, got %s", resp.Error.Message)
	}
	if !broker.called {
		t.Error("Expected broker.Authorize to be called")
	}
	if broker.action != "test_tool" {
		t.Errorf("Expected action=test_tool, got %s", broker.action)
	}
	if mockProv.spawned {
		t.Error("Expected ToolSidecar NOT to be spawned when broker denies")
	}
}

func TestHandleToolsCall_AuthorizerError(t *testing.T) {
	broker := &mockCapabilityBroker{err: fmt.Errorf("broker internal error")}
	router, _ := createTestRouterWithAuthorizer(t, broker)

	args, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "auth_err_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error == nil {
		t.Fatal("Expected error response when broker errors")
	}
}

func TestHandleToolsCall_NilAuthorizer_BackwardCompat(t *testing.T) {
	mockAuditLog, _ := audit.NewAuditLog(audit.Config{Path: "/tmp/test_audit_no_auth.db"})
	mockGovernor := governor.NewGovernor(nil, logger.Global())
	mockProv := &mockProvisioner{}
	consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{Timeout: 60 * time.Second})

	router, err := New(Config{
		SkillGate:      mockGovernor,
		Provisioner:    mockProv,
		ConsentManager: consentMgr,
		Auditor:        mockAuditLog,
		Logger:         logger.Global(),
		Authorizer:     nil,
	})
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	args, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "no_auth_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success with nil authorizer, got error: %v", resp)
	}
	if !mockProv.spawned {
		t.Error("Expected ToolSidecar to be spawned with nil authorizer (backward compat)")
	}
}

func TestHandleToolsCall_AuthorizerAllow(t *testing.T) {
	broker := &mockCapabilityBroker{}
	router, mockProv := createTestRouterWithAuthorizer(t, broker)

	args, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := &MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "auth_allow_1",
		Params:  &MCPParams{Name: "test_tool", Arguments: args},
	}

	resp, err := router.HandleToolsCall(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil || resp.Error != nil {
		t.Fatalf("Expected success when broker allows, got error: %v", resp)
	}
	if !broker.called {
		t.Error("Expected broker.Authorize to be called")
	}
	if !mockProv.spawned {
		t.Error("Expected ToolSidecar to be spawned when broker allows")
	}
}
