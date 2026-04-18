// Package rpc provides email approval RPC methods.
// These methods implement the HITL approval flow where agents request
// to send outbound emails and mobile users approve/deny them.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/email"
)

// emailApprovalMgr is a package-level EmailApprovalManager used by RPC handlers.
// In production the manager is injected by the bridge; tests can override via
// setEmailApprovalManager.
var emailApprovalMgr *email.EmailApprovalManager

// setEmailApprovalManager stores the manager used by the RPC layer.
func setEmailApprovalManager(mgr *email.EmailApprovalManager) {
	emailApprovalMgr = mgr
}

// getEmailApprovalManager returns the current manager, creating a default one
// if none has been set.
func getEmailApprovalManager() *email.EmailApprovalManager {
	if emailApprovalMgr != nil {
		return emailApprovalMgr
	}
	// Create a default manager (no Matrix callback in default path).
	emailApprovalMgr = email.NewEmailApprovalManager(email.EmailApprovalConfig{
		Timeout: 300 * time.Second,
	})
	return emailApprovalMgr
}

// handleApproveEmail handles approve_email RPC method.
// Approves a pending email approval request, resuming the blocked agent.
func (s *Server) handleApproveEmail(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		ApprovalID string `json:"approval_id"`
		UserID     string `json:"user_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.ApprovalID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "approval_id is required",
		}
	}

	if params.UserID == "" {
		params.UserID = "unknown"
	}

	mgr := getEmailApprovalManager()

	if err := mgr.HandleApprovalResponse(params.ApprovalID, true, params.UserID); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to approve email: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"approval_id": params.ApprovalID,
		"status":      "approved",
		"approved_by": params.UserID,
		"approved_at": time.Now().Format(time.RFC3339),
		"message":     "Email approved. Agent resumed.",
	}, nil
}

// handleDenyEmail handles deny_email RPC method.
// Denies a pending email approval request, cancelling the agent task.
func (s *Server) handleDenyEmail(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		ApprovalID string `json:"approval_id"`
		UserID     string `json:"user_id"`
		Reason     string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.ApprovalID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "approval_id is required",
		}
	}

	if params.UserID == "" {
		params.UserID = "unknown"
	}

	if params.Reason == "" {
		params.Reason = "No reason provided"
	}

	mgr := getEmailApprovalManager()

	if err := mgr.HandleApprovalResponse(params.ApprovalID, false, params.UserID); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to deny email: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"approval_id": params.ApprovalID,
		"status":      "denied",
		"denied_by":   params.UserID,
		"deny_reason": params.Reason,
		"denied_at":   time.Now().Format(time.RFC3339),
		"message":     "Email denied. Agent task cancelled.",
	}, nil
}

// handleEmailApprovalStatus handles email_approval_status RPC method.
// Returns pending approval count.
func (s *Server) handleEmailApprovalStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	mgr := getEmailApprovalManager()

	return map[string]interface{}{
		"pending_count": mgr.PendingCount(),
		"timeout_s":     300,
	}, nil
}

// emitEmailApprovalRequestEvent emits a Matrix event for a new email approval request.
func (s *Server) emitEmailApprovalRequestEvent(roomID, approvalID, emailID, to string, piiFieldCount int, timeoutS int) error {
	if s.matrix == nil {
		return nil
	}

	event := map[string]interface{}{
		"approval_id": approvalID,
		"email_id":    emailID,
		"to":          to,
		"subject":     "[masked]",
		"pii_fields":  piiFieldCount,
		"timeout_s":   timeoutS,
		"event_type":  "email_approval_request",
	}

	content, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal email approval event: %w", err)
	}

	return s.matrix.SendEvent(roomID, "app.armorclaw.email_approval_request", content)
}

// Ensure unused import is consumed.
var _ = fmt.Sprintf
