package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/trust"
)

// handleDeviceList returns all registered devices.
func (s *Server) handleDeviceList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.deviceStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "device store not configured",
		}
	}

	devices, err := s.deviceStore.ListDevices()
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to list devices: " + err.Error(),
		}
	}

	if devices == nil {
		devices = []*trust.DeviceRecord{}
	}

	return devices, nil
}

// handleDeviceGet returns a single device by ID.
func (s *Server) handleDeviceGet(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.deviceStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "device store not configured",
		}
	}

	var params DeviceGetRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.DeviceID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "device_id is required",
		}
	}

	device, err := s.deviceStore.GetDevice(params.DeviceID)
	if err != nil {
		if err.Error() == "device not found" {
			return nil, &ErrorObj{
				Code:    NotFoundError,
				Message: "device not found",
			}
		}
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to get device: " + err.Error(),
		}
	}

	return device, nil
}

// handleDeviceApprove approves a device, setting its trust state to verified.
// Idempotent: approving an already-verified device returns success.
func (s *Server) handleDeviceApprove(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.deviceStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "device store not configured",
		}
	}

	var params DeviceApproveRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.DeviceID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "device_id is required",
		}
	}

	if params.ApprovedBy == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "approved_by is required",
		}
	}

	device, err := s.deviceStore.GetDevice(params.DeviceID)
	if err != nil {
		if err.Error() == "device not found" {
			return nil, &ErrorObj{
				Code:    NotFoundError,
				Message: "device not found",
			}
		}
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to get device: " + err.Error(),
		}
	}

	// Idempotent: already verified is a success
	if device.TrustState == trust.StateVerified {
		return SuccessResponse{Success: true}, nil
	}

	if err := s.deviceStore.UpdateTrustState(params.DeviceID, trust.StateVerified); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("failed to approve device: %s", err),
		}
	}

	s.auditGovernanceMutation(audit.EventDeviceApproved, params.ApprovedBy, map[string]interface{}{
		"device_id": params.DeviceID,
	})

	return SuccessResponse{Success: true}, nil
}

// handleDeviceReject rejects a device, setting its trust state to rejected.
// Idempotent: rejecting an already-rejected device returns success.
func (s *Server) handleDeviceReject(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.deviceStore == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "device store not configured",
		}
	}

	var params DeviceRejectRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.DeviceID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "device_id is required",
		}
	}

	device, err := s.deviceStore.GetDevice(params.DeviceID)
	if err != nil {
		if err.Error() == "device not found" {
			return nil, &ErrorObj{
				Code:    NotFoundError,
				Message: "device not found",
			}
		}
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "failed to get device: " + err.Error(),
		}
	}

	// Idempotent: already rejected is a success
	if device.TrustState == trust.StateRejected {
		return SuccessResponse{Success: true}, nil
	}

	if err := s.deviceStore.UpdateTrustState(params.DeviceID, trust.StateRejected); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("failed to reject device: %s", err),
		}
	}

	s.auditGovernanceMutation(audit.EventDeviceRejected, params.RejectedBy, map[string]interface{}{
		"device_id": params.DeviceID,
		"reason":    params.Reason,
	})

	return SuccessResponse{Success: true}, nil
}

func (s *Server) auditGovernanceMutation(eventType audit.EventType, userID string, details interface{}) {
	if s.auditLog == nil {
		return
	}
	if err := s.auditLog.LogEvent(eventType, "", "", userID, details); err != nil {
		slog.Error("governance audit write failed",
			slog.String("event_type", string(eventType)),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}
}
