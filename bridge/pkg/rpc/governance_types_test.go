package rpc

import (
	"encoding/json"
	"testing"
)

func TestDeviceApproveRequestUnmarshal(t *testing.T) {
	raw := `{"device_id":"dev_123","approved_by":"admin"}`
	var req DeviceApproveRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.DeviceID != "dev_123" {
		t.Errorf("DeviceID = %q, want %q", req.DeviceID, "dev_123")
	}
	if req.ApprovedBy != "admin" {
		t.Errorf("ApprovedBy = %q, want %q", req.ApprovedBy, "admin")
	}
}

func TestInviteCreateRequestUnmarshal(t *testing.T) {
	raw := `{"role":"user","expiration":"7d","max_uses":5,"created_by":"admin"}`
	var req InviteCreateRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Role != "user" {
		t.Errorf("Role = %q, want %q", req.Role, "user")
	}
	if req.Expiration != "7d" {
		t.Errorf("Expiration = %q, want %q", req.Expiration, "7d")
	}
	if req.MaxUses != 5 {
		t.Errorf("MaxUses = %d, want %d", req.MaxUses, 5)
	}
	if req.CreatedBy != "admin" {
		t.Errorf("CreatedBy = %q, want %q", req.CreatedBy, "admin")
	}
}

func TestDeviceRejectRequestUnmarshal(t *testing.T) {
	raw := `{"device_id":"dev_456","rejected_by":"admin","reason":"unrecognized"}`
	var req DeviceRejectRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.DeviceID != "dev_456" {
		t.Errorf("DeviceID = %q, want %q", req.DeviceID, "dev_456")
	}
	if req.RejectedBy != "admin" {
		t.Errorf("RejectedBy = %q, want %q", req.RejectedBy, "admin")
	}
	if req.Reason != "unrecognized" {
		t.Errorf("Reason = %q, want %q", req.Reason, "unrecognized")
	}
}

func TestInviteValidateRequestUnmarshal(t *testing.T) {
	raw := `{"code":"abc-123"}`
	var req InviteValidateRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Code != "abc-123" {
		t.Errorf("Code = %q, want %q", req.Code, "abc-123")
	}
}

func TestInviteRevokeRequestUnmarshal(t *testing.T) {
	raw := `{"invite_id":"inv_789","revoked_by":"admin"}`
	var req InviteRevokeRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.InviteID != "inv_789" {
		t.Errorf("InviteID = %q, want %q", req.InviteID, "inv_789")
	}
	if req.RevokedBy != "admin" {
		t.Errorf("RevokedBy = %q, want %q", req.RevokedBy, "admin")
	}
}

func TestSuccessResponseMarshal(t *testing.T) {
	resp := SuccessResponse{Success: true}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != `{"success":true}` {
		t.Errorf("got %s, want %q", data, `{"success":true}`)
	}
}

func TestDeviceListRequestUnmarshal(t *testing.T) {
	raw := `{}`
	var req DeviceListRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestInviteListRequestUnmarshal(t *testing.T) {
	raw := `{}`
	var req InviteListRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestDeviceGetRequestUnmarshal(t *testing.T) {
	raw := `{"device_id":"dev_999"}`
	var req DeviceGetRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.DeviceID != "dev_999" {
		t.Errorf("DeviceID = %q, want %q", req.DeviceID, "dev_999")
	}
}

func TestInviteCreateRequestWithWelcomeMessage(t *testing.T) {
	raw := `{"role":"admin","expiration":"30d","max_uses":1,"welcome_message":"Welcome!","created_by":"superadmin"}`
	var req InviteCreateRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.WelcomeMessage != "Welcome!" {
		t.Errorf("WelcomeMessage = %q, want %q", req.WelcomeMessage, "Welcome!")
	}
}
