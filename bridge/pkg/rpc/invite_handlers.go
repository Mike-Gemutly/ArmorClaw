package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/invite"
)

// handleInviteList returns all invites ordered by created_at descending.
func (s *Server) handleInviteList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.inviteStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "invite store not configured",
		}
	}

	invites, err := s.inviteStore.ListInvites()
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to list invites: " + err.Error(),
		}
	}

	if invites == nil {
		invites = []*invite.InviteRecord{}
	}

	return invites, nil
}

// handleInviteCreate creates a new invite with a crypto/rand code.
func (s *Server) handleInviteCreate(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.inviteStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "invite store not configured",
		}
	}

	var params InviteCreateRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.Role == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "role is required",
		}
	}

	role := invite.Role(params.Role)
	if !role.IsValid() {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("invalid role: %s", params.Role),
		}
	}

	if params.Expiration == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "expiration is required",
		}
	}

	expiresAt, err := invite.ParseExpiration(params.Expiration)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid expiration: " + err.Error(),
		}
	}

	if params.CreatedBy == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "created_by is required",
		}
	}

	record := &invite.InviteRecord{
		Role:           role,
		CreatedBy:      params.CreatedBy,
		ExpiresAt:      expiresAt,
		MaxUses:        params.MaxUses,
		WelcomeMessage: params.WelcomeMessage,
	}

	if err := s.inviteStore.CreateInvite(record); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to create invite: " + err.Error(),
		}
	}

	s.auditGovernanceMutation(audit.EventInviteCreated, params.CreatedBy, map[string]interface{}{
		"invite_id": record.ID,
		"role":      string(record.Role),
		"code":      record.Code,
	})

	s.emitInviteEvent(EventInviteCreated, record.ID, params.CreatedBy, record.Code)

	return record, nil
}

// handleInviteRevoke revokes an active invite.
// Idempotent: revoking an already-revoked invite returns success.
func (s *Server) handleInviteRevoke(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.inviteStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "invite store not configured",
		}
	}

	var params InviteRevokeRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.InviteID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invite_id is required",
		}
	}

	existing, err := s.inviteStore.GetInvite(params.InviteID)
	if err != nil {
		return nil, &ErrorObj{
			Code:    NotFoundError,
			Message: "invite not found",
		}
	}

	// Idempotent: already revoked is a success
	if existing.Status == invite.StatusRevoked {
		return SuccessResponse{Success: true}, nil
	}

	if err := s.inviteStore.RevokeInvite(params.InviteID); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("failed to revoke invite: %s", err),
		}
	}

	s.auditGovernanceMutation(audit.EventInviteRevoked, params.RevokedBy, map[string]interface{}{
		"invite_id": params.InviteID,
	})

	s.emitInviteEvent(EventInviteRevoked, params.InviteID, params.RevokedBy, existing.Code)

	return SuccessResponse{Success: true}, nil
}

// handleInviteValidate validates an invite code and returns the full Invite record.
// Returns error for expired, revoked, or exhausted invites.
func (s *Server) handleInviteValidate(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.inviteStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "invite store not configured",
		}
	}

	var params InviteValidateRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.Code == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "code is required",
		}
	}

	record, err := s.inviteStore.GetInviteByCode(params.Code)
	if err != nil {
		return nil, &ErrorObj{
			Code:    NotFoundError,
			Message: "invite not found",
		}
	}

	if record.Status != invite.StatusActive {
		return nil, &ErrorObj{
			Code:    NotFoundError,
			Message: fmt.Sprintf("invite is %s", string(record.Status)),
		}
	}

	if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, &ErrorObj{
			Code:    NotFoundError,
			Message: "invite has expired",
		}
	}

	if record.MaxUses > 0 && record.UseCount >= record.MaxUses {
		return nil, &ErrorObj{
			Code:    NotFoundError,
			Message: "invite usage limit reached",
		}
	}

	return record, nil
}
