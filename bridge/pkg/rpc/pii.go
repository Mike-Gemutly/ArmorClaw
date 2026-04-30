// Package rpc provides PII access control RPC methods.
// These methods implement the "Secretary" flow where agents request sensitive data
// and mobile users approve/deny access.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
)

// handlePIIRequest handles pii.request RPC method
// Creates a PII access request, pauses the agent, and emits Matrix event
func (s *Server) handlePIIRequest(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		AgentID   string                   `json:"agent_id"`
		SkillID   string                   `json:"skill_id"`
		SkillName string                   `json:"skill_name"`
		ProfileID string                   `json:"profile_id"`
		RoomID    string                   `json:"room_id"`
		Context   string                   `json:"context"`
		Variables []map[string]interface{} `json:"variables"`
		TTL       int                      `json:"ttl"` // seconds, 0 = default
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.AgentID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "agent_id is required",
		}
	}

	if params.SkillID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "skill_id is required",
		}
	}

	if params.ProfileID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "profile_id is required",
		}
	}

	// Convert variables to field requests
	fields := make([]keystore.PIIFieldRequest, 0, len(params.Variables))
	for _, v := range params.Variables {
		key, _ := v["key"].(string)
		displayName, _ := v["display_name"].(string)
		if displayName == "" {
			displayName = key
		}
		required, _ := v["required"].(bool)
		sensitive, _ := v["sensitive"].(bool)

		fields = append(fields, keystore.PIIFieldRequest{
			Key:         key,
			DisplayName: displayName,
			Required:    required,
			Sensitive:   sensitive,
		})
	}

	// Calculate TTL
	ttl := time.Duration(params.TTL) * time.Second
	if ttl == 0 {
		ttl = 5 * time.Minute // default
	}

	// Create the request using the PII request manager
	piiMgr := s.getOrCreatePIIRequestManager()

	// Set up callback to emit Matrix event
	piiMgr.SetCallbacks(
		func(ctx context.Context, r *keystore.PIIRequest) error {
			// On request created - emit Matrix event
			return s.emitPIIRequestEvent(ctx, r)
		},
		func(ctx context.Context, r *keystore.PIIRequest) error {
			// On approved - emit approval event
			return s.emitPIIApprovalEvent(ctx, r)
		},
		func(ctx context.Context, r *keystore.PIIRequest) error {
			// On denied - emit denial event
			return s.emitPIIDenialEvent(ctx, r)
		},
		nil, // on expired
	)

	piiReq, err := piiMgr.CreateRequest(
		ctx,
		params.AgentID,
		params.SkillID,
		params.SkillName,
		params.ProfileID,
		fields,
		params.Context,
		params.RoomID,
		ttl,
	)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to create PII request: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"request_id":       piiReq.ID,
		"agent_id":         piiReq.AgentID,
		"skill_id":         piiReq.SkillID,
		"profile_id":       piiReq.ProfileID,
		"requested_fields": fields,
		"status":           string(piiReq.Status),
		"created_at":       piiReq.CreatedAt.Format(time.RFC3339),
		"expires_at":       piiReq.ExpiresAt.Format(time.RFC3339),
		"message":          "PII request created. Agent paused awaiting approval.",
	}, nil
}

// handlePIIApprove handles pii.approve RPC method
// Approves a PII request and resumes the agent with decrypted variables
func (s *Server) handlePIIApprove(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID      string   `json:"request_id"`
		UserID         string   `json:"user_id"`
		ApprovedFields []string `json:"approved_fields"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	if params.UserID == "" {
		params.UserID = "unknown"
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	piiReq, err := piiMgr.ApproveRequest(ctx, params.RequestID, params.UserID, params.ApprovedFields)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to approve PII request: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"request_id":      piiReq.ID,
		"status":          string(piiReq.Status),
		"approved_by":     piiReq.ApprovedBy,
		"approved_fields": piiReq.ApprovedFields,
		"approved_at":     piiReq.ApprovedAt.Format(time.RFC3339),
		"message":         "PII request approved. Agent resumed with approved variables.",
	}, nil
}

// handlePIIDeny handles pii.deny RPC method
// Denies a PII request and cancels the agent task
func (s *Server) handlePIIDeny(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID string `json:"request_id"`
		UserID    string `json:"user_id"`
		Reason    string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	if params.UserID == "" {
		params.UserID = "unknown"
	}

	if params.Reason == "" {
		params.Reason = "No reason provided"
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	piiReq, err := piiMgr.DenyRequest(ctx, params.RequestID, params.UserID, params.Reason)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to deny PII request: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"request_id":  piiReq.ID,
		"status":      string(piiReq.Status),
		"denied_by":   piiReq.DeniedBy,
		"deny_reason": piiReq.DenyReason,
		"denied_at":   piiReq.DeniedAt.Format(time.RFC3339),
		"message":     "PII request denied. Agent task cancelled.",
	}, nil
}

// handlePIIStatus handles pii.status RPC method
// Returns the current status of a PII request
func (s *Server) handlePIIStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	piiReq, err := piiMgr.GetRequest(params.RequestID)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "PII request not found: " + params.RequestID,
		}
	}

	result := map[string]interface{}{
		"request_id":       piiReq.ID,
		"agent_id":         piiReq.AgentID,
		"skill_id":         piiReq.SkillID,
		"skill_name":       piiReq.SkillName,
		"profile_id":       piiReq.ProfileID,
		"status":           string(piiReq.Status),
		"created_at":       piiReq.CreatedAt.Format(time.RFC3339),
		"expires_at":       piiReq.ExpiresAt.Format(time.RFC3339),
		"is_expired":       piiReq.IsExpired(),
		"requested_fields": piiReq.RequestedFields,
	}

	if piiReq.ApprovedAt != nil {
		result["approved_at"] = piiReq.ApprovedAt.Format(time.RFC3339)
		result["approved_by"] = piiReq.ApprovedBy
		result["approved_fields"] = piiReq.ApprovedFields
	}

	if piiReq.DeniedAt != nil {
		result["denied_at"] = piiReq.DeniedAt.Format(time.RFC3339)
		result["denied_by"] = piiReq.DeniedBy
		result["deny_reason"] = piiReq.DenyReason
	}

	if piiReq.FulfilledAt != nil {
		result["fulfilled_at"] = piiReq.FulfilledAt.Format(time.RFC3339)
	}

	return result, nil
}

// handlePIIListPending handles pii.list_pending RPC method
// Lists all pending PII requests
func (s *Server) handlePIIListPending(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	piiMgr := s.getOrCreatePIIRequestManager()

	pending := piiMgr.ListPending()

	requests := make([]map[string]interface{}, 0, len(pending))
	for _, r := range pending {
		requests = append(requests, map[string]interface{}{
			"request_id":       r.ID,
			"agent_id":         r.AgentID,
			"skill_id":         r.SkillID,
			"skill_name":       r.SkillName,
			"profile_id":       r.ProfileID,
			"status":           string(r.Status),
			"created_at":       r.CreatedAt.Format(time.RFC3339),
			"expires_at":       r.ExpiresAt.Format(time.RFC3339),
			"requested_fields": r.RequestedFields,
			"context":          r.Context,
		})
	}

	return map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	}, nil
}

// handlePIIStats handles pii.stats RPC method
// Returns statistics about PII requests
func (s *Server) handlePIIStats(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	piiMgr := s.getOrCreatePIIRequestManager()
	stats := piiMgr.GetStats()

	return stats, nil
}

// getOrCreatePIIRequestManager returns the singleton PII request manager
// initialized during Server creation. Falls back to creating a new one only
// if the Server was constructed outside of New() (should not happen).
func (s *Server) getOrCreatePIIRequestManager() *keystore.PIIRequestManager {
	if s.piiRequestManager != nil {
		return s.piiRequestManager
	}
	// Fallback: should not happen in normal operation
	return keystore.NewPIIRequestManager(keystore.PIIRequestManagerConfig{
		DefaultTTL: 5 * time.Minute,
	})
}

// emitPIIRequestEvent emits a Matrix event for a new PII request
func (s *Server) emitPIIRequestEvent(ctx context.Context, req *keystore.PIIRequest) error {
	if s.matrix == nil {
		return nil
	}

	event := req.ToMatrixEvent()
	eventType := "app.armorclaw.pii_request"

	// Send to the specified room or a default room
	roomID := req.RoomID
	if roomID == "" {
		return nil
	}

	content, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.matrix.SendEvent(roomID, eventType, content)
}

// emitPIIApprovalEvent emits a Matrix event when a PII request is approved
func (s *Server) emitPIIApprovalEvent(ctx context.Context, req *keystore.PIIRequest) error {
	if s.matrix == nil || req.RoomID == "" {
		return nil
	}

	event := map[string]interface{}{
		"request_id":      req.ID,
		"status":          "approved",
		"approved_by":     req.ApprovedBy,
		"approved_fields": req.ApprovedFields,
		"approved_at":     req.ApprovedAt.UnixMilli(),
	}

	content, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.matrix.SendEvent(req.RoomID, "app.armorclaw.pii_response", content)
}

// emitPIIDenialEvent emits a Matrix event when a PII request is denied
func (s *Server) emitPIIDenialEvent(ctx context.Context, req *keystore.PIIRequest) error {
	if s.matrix == nil || req.RoomID == "" {
		return nil
	}

	event := map[string]interface{}{
		"request_id":  req.ID,
		"status":      "denied",
		"denied_by":   req.DeniedBy,
		"deny_reason": req.DenyReason,
		"denied_at":   req.DeniedAt.UnixMilli(),
	}

	content, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.matrix.SendEvent(req.RoomID, "app.armorclaw.pii_response", content)
}

// handlePIICancel handles pii.cancel RPC method
// Cancels a pending PII request
func (s *Server) handlePIICancel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	err := piiMgr.CancelRequest(ctx, params.RequestID)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to cancel PII request: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"request_id": params.RequestID,
		"status":     "cancelled",
		"message":    "PII request cancelled.",
	}, nil
}

// handlePIIFulfill handles pii.fulfill RPC method
// Marks a request as fulfilled after delivering variables to the agent
func (s *Server) handlePIIFulfill(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID  string            `json:"request_id"`
		ResolvedVars map[string]string `json:"resolved_vars"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	err := piiMgr.FulfillRequest(ctx, params.RequestID, params.ResolvedVars)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to fulfill PII request: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"request_id":   params.RequestID,
		"status":       "fulfilled",
		"fields_count": len(params.ResolvedVars),
		"message":      "PII request fulfilled. Variables delivered to agent.",
	}, nil
}

// handlePIIWaitForApproval handles pii.wait_for_approval RPC method
// Waits for a PII request to be approved or denied (long-poll)
func (s *Server) handlePIIWaitForApproval(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RequestID string `json:"request_id"`
		Timeout   int    `json:"timeout"` // seconds
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.RequestID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "request_id is required",
		}
	}

	if params.Timeout == 0 {
		params.Timeout = 60 // default 60 seconds
	}

	piiMgr := s.getOrCreatePIIRequestManager()

	// Poll for status change
	timeout := time.Duration(params.Timeout) * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		piiReq, err := piiMgr.GetRequest(params.RequestID)
		if err != nil {
			return nil, &ErrorObj{
				Code:    InternalError,
				Message: "PII request not found",
			}
		}

		if piiReq.Status == keystore.StatusApproved {
			return map[string]interface{}{
				"request_id":      piiReq.ID,
				"status":          "approved",
				"approved_by":     piiReq.ApprovedBy,
				"approved_fields": piiReq.ApprovedFields,
			}, nil
		}

		if piiReq.Status == keystore.StatusDenied {
			return map[string]interface{}{
				"request_id":  piiReq.ID,
				"status":      "denied",
				"denied_by":   piiReq.DeniedBy,
				"deny_reason": piiReq.DenyReason,
			}, nil
		}

		if piiReq.IsExpired() {
			return map[string]interface{}{
				"request_id": piiReq.ID,
				"status":     "expired",
			}, nil
		}

		// Wait before next poll
		time.Sleep(500 * time.Millisecond)
	}

	// Timeout - still pending
	return map[string]interface{}{
		"request_id": params.RequestID,
		"status":     "pending",
		"message":    fmt.Sprintf("Request still pending after %d seconds", params.Timeout),
	}, nil
}
