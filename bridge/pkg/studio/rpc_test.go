package studio

import (
	"encoding/json"
	"testing"
)

//=============================================================================
// RPC Handler Tests (CGO-free)
//=============================================================================

func TestSuccessResponse(t *testing.T) {
	result := map[string]string{"id": "test-123"}
	resp := SuccessResponse(result)

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got: %s", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Error("expected no error in success response")
	}

	if resp.Result == nil {
		t.Error("expected result in success response")
	}
}

func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse(ErrNotFound, "Not found", "extra data")

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got: %s", resp.JSONRPC)
	}

	if resp.Error == nil {
		t.Fatal("expected error in error response")
	}

	if resp.Error.Code != ErrNotFound {
		t.Errorf("expected error code %d, got: %d", ErrNotFound, resp.Error.Code)
	}

	if resp.Error.Message != "Not found" {
		t.Errorf("expected message 'Not found', got: %s", resp.Error.Message)
	}

	if resp.Error.Data != "extra data" {
		t.Errorf("expected data 'extra data', got: %v", resp.Error.Data)
	}
}

func TestErrorResponseWithoutData(t *testing.T) {
	resp := ErrorResponse(ErrInvalidParams, "Invalid params")

	if resp.Error.Data != nil {
		t.Errorf("expected no data, got: %v", resp.Error.Data)
	}
}

//=============================================================================
// Handle Routing Tests
//=============================================================================

func TestHandle_UnknownMethod(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown.method",
		Params:  json.RawMessage("{}"),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}

	if resp.Error.Code != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %d", resp.Error.Code)
	}
}

func TestHandle_ListProfiles(t *testing.T) {
	// This test doesn't require a store
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.list_profiles",
		Params:  json.RawMessage("{}"),
	}

	resp := handler.Handle(req)

	if resp.Error != nil {
		t.Errorf("expected no error, got: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}

	profiles, ok := result["profiles"].(map[string]ResourceProfile)
	if !ok {
		t.Fatal("expected profiles map")
	}

	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles, got: %d", len(profiles))
	}
}

//=============================================================================
// Parameter Validation Tests
//=============================================================================

func TestHandle_GetSkill_MissingID(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.get_skill",
		Params:  json.RawMessage("{}"),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for missing skill ID")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

func TestHandle_GetPII_MissingID(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.get_pii",
		Params:  json.RawMessage("{}"),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for missing PII ID")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

func TestHandle_GetAgent_MissingID(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.get_agent",
		Params:  json.RawMessage("{}"),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for missing agent ID")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

func TestHandle_CreateAgent_MissingName(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.create_agent",
		Params:  json.RawMessage(`{"skills": ["browser_navigate"], "pii_access": ["client_name"]}`),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for missing name")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

func TestHandle_CreateAgent_MissingSkills(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.create_agent",
		Params:  json.RawMessage(`{"name": "Test Agent", "pii_access": ["client_name"]}`),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for missing skills")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

func TestHandle_CreateAgent_InvalidParams(t *testing.T) {
	handler := NewRPCHandler(RPCHandlerConfig{Store: nil})
	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "studio.create_agent",
		Params:  json.RawMessage(`invalid json`),
	}

	resp := handler.Handle(req)

	if resp.Error == nil {
		t.Error("expected error for invalid JSON")
	}

	if resp.Error.Code != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams, got: %d", resp.Error.Code)
	}
}

//=============================================================================
// ID Generation Tests
//=============================================================================

func TestGenerateID(t *testing.T) {
	id1 := generateID("agent")
	id2 := generateID("agent")

	// Should have prefix
	if len(id1) < 10 {
		t.Errorf("ID too short: %s", id1)
	}

	// Should be unique
	if id1 == id2 {
		t.Error("expected unique IDs")
	}

	// Should start with prefix
	if id1[:6] != "agent_" {
		t.Errorf("expected ID to start with 'agent_', got: %s", id1[:6])
	}
}

//=============================================================================
// Type Marshaling Tests
//=============================================================================

func TestRPCError_MarshalJSON(t *testing.T) {
	rpcErr := &RPCError{
		Code:    ErrNotFound,
		Message: "Not found",
		Data:    map[string]string{"id": "123"},
	}

	data, err := json.Marshal(rpcErr)
	if err != nil {
		t.Fatalf("failed to marshal RPCError: %v", err)
	}

	var unmarshaled RPCError
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal RPCError: %v", err)
	}

	if unmarshaled.Code != ErrNotFound {
		t.Errorf("expected code %d, got: %d", ErrNotFound, unmarshaled.Code)
	}
}

func TestRPCResponse_MarshalJSON(t *testing.T) {
	resp := &RPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]string{"status": "ok"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal RPCResponse: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got: %v", unmarshaled["jsonrpc"])
	}
}

//=============================================================================
// Resource Profile Validation Tests
//=============================================================================

func TestProfileManager_Validate(t *testing.T) {
	pm := NewProfileManager(nil)

	// Valid tiers
	if err := pm.Validate("low"); err != nil {
		t.Errorf("expected low tier to be valid, got: %v", err)
	}
	if err := pm.Validate("medium"); err != nil {
		t.Errorf("expected medium tier to be valid, got: %v", err)
	}
	if err := pm.Validate("high"); err != nil {
		t.Errorf("expected high tier to be valid, got: %v", err)
	}
	if err := pm.Validate(""); err != nil {
		t.Errorf("expected empty tier to be valid (defaults), got: %v", err)
	}

	// Invalid tier
	if err := pm.Validate("invalid"); err == nil {
		t.Error("expected invalid tier to fail validation")
	}
}

//=============================================================================
// Request Type Tests
//=============================================================================

func TestListSkillsParams_Parse(t *testing.T) {
	params := json.RawMessage(`{"category": "document"}`)

	var lsp ListSkillsParams
	if err := json.Unmarshal(params, &lsp); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if lsp.Category != "document" {
		t.Errorf("expected category 'document', got: %s", lsp.Category)
	}
}

func TestCreateAgentParams_Parse(t *testing.T) {
	params := json.RawMessage(`{
		"name": "Test Agent",
		"description": "A test agent",
		"skills": ["browser_navigate", "pdf_generator"],
		"pii_access": ["client_name", "client_email"],
		"resource_tier": "high"
	}`)

	var cap CreateAgentParams
	if err := json.Unmarshal(params, &cap); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if cap.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got: %s", cap.Name)
	}

	if len(cap.Skills) != 2 {
		t.Errorf("expected 2 skills, got: %d", len(cap.Skills))
	}

	if cap.ResourceTier != "high" {
		t.Errorf("expected tier 'high', got: %s", cap.ResourceTier)
	}
}

func TestSpawnAgentParams_Parse(t *testing.T) {
	params := json.RawMessage(`{
		"id": "agent-123",
		"task_description": "Process contracts"
	}`)

	var sap SpawnAgentParams
	if err := json.Unmarshal(params, &sap); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if sap.ID != "agent-123" {
		t.Errorf("expected ID 'agent-123', got: %s", sap.ID)
	}

	if sap.TaskDescription != "Process contracts" {
		t.Errorf("expected description 'Process contracts', got: %s", sap.TaskDescription)
	}
}

//=============================================================================
// Update Agent Params Tests
//=============================================================================

func TestUpdateAgentParams_Parse(t *testing.T) {
	params := json.RawMessage(`{
		"id": "agent-123",
		"name": "Updated Name",
		"is_active": false
	}`)

	var uap UpdateAgentParams
	if err := json.Unmarshal(params, &uap); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if uap.ID != "agent-123" {
		t.Errorf("expected ID 'agent-123', got: %s", uap.ID)
	}

	if uap.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got: %s", uap.Name)
	}

	if uap.IsActive == nil || *uap.IsActive != false {
		t.Errorf("expected is_active to be false, got: %v", uap.IsActive)
	}
}
