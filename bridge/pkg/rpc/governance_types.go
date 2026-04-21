package rpc

// Device governance request types.

// DeviceListRequest is the request for device.list (no parameters).
type DeviceListRequest struct{}

// DeviceGetRequest is the request for device.get.
type DeviceGetRequest struct {
	DeviceID string `json:"device_id"`
}

// DeviceApproveRequest is the request for device.approve.
type DeviceApproveRequest struct {
	DeviceID   string `json:"device_id"`
	ApprovedBy string `json:"approved_by"`
}

// DeviceRejectRequest is the request for device.reject.
type DeviceRejectRequest struct {
	DeviceID   string `json:"device_id"`
	RejectedBy string `json:"rejected_by"`
	Reason     string `json:"reason"`
}

// Invite governance request types.

// InviteCreateRequest is the request for invite.create.
type InviteCreateRequest struct {
	Role           string `json:"role"`
	Expiration     string `json:"expiration"`
	MaxUses        int    `json:"max_uses"`
	WelcomeMessage string `json:"welcome_message,omitempty"`
	CreatedBy      string `json:"created_by"`
}

// InviteListRequest is the request for invite.list (no parameters).
type InviteListRequest struct{}

// InviteRevokeRequest is the request for invite.revoke.
type InviteRevokeRequest struct {
	InviteID  string `json:"invite_id"`
	RevokedBy string `json:"revoked_by"`
}

// InviteValidateRequest is the request for invite.validate.
type InviteValidateRequest struct {
	Code string `json:"code"`
}

// SuccessResponse is a generic success response for governance methods.
type SuccessResponse struct {
	Success bool `json:"success"`
}
