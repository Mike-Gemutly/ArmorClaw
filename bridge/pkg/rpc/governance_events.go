package rpc

import (
	"encoding/json"
	"log/slog"
	"time"
)

// Governance event type constants emitted as Matrix custom events.
// These are outbound-only (never added to the Matrix sync filter).
const (
	EventDeviceApproved = "app.armorclaw.device.approved"
	EventDeviceRejected = "app.armorclaw.device.rejected"
	EventInviteCreated  = "app.armorclaw.invite.created"
	EventInviteRevoked  = "app.armorclaw.invite.revoked"
)

// emitDeviceEvent emits a governance event for a device mutation.
// Best-effort: logs errors but never fails the calling RPC handler.
func (s *Server) emitDeviceEvent(eventType, deviceID, actor string) {
	if s.matrix == nil || s.governanceRoomID == "" {
		return
	}

	event := map[string]interface{}{
		"device_id":  deviceID,
		"actor":      actor,
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	content, err := json.Marshal(event)
	if err != nil {
		slog.Error("governance event marshal failed",
			slog.String("event_type", eventType),
			slog.String("device_id", deviceID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := s.matrix.SendEvent(s.governanceRoomID, eventType, content); err != nil {
		slog.Error("governance event send failed",
			slog.String("event_type", eventType),
			slog.String("device_id", deviceID),
			slog.String("room_id", s.governanceRoomID),
			slog.String("error", err.Error()),
		)
	}
}

// emitInviteEvent emits a governance event for an invite mutation.
// Best-effort: logs errors but never fails the calling RPC handler.
func (s *Server) emitInviteEvent(eventType, inviteID, actor, code string) {
	if s.matrix == nil || s.governanceRoomID == "" {
		return
	}

	event := map[string]interface{}{
		"invite_id":  inviteID,
		"actor":      actor,
		"code":       code,
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	content, err := json.Marshal(event)
	if err != nil {
		slog.Error("governance event marshal failed",
			slog.String("event_type", eventType),
			slog.String("invite_id", inviteID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := s.matrix.SendEvent(s.governanceRoomID, eventType, content); err != nil {
		slog.Error("governance event send failed",
			slog.String("event_type", eventType),
			slog.String("invite_id", inviteID),
			slog.String("room_id", s.governanceRoomID),
			slog.String("error", err.Error()),
		)
	}
}
