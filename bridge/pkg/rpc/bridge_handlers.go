// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
// This file contains handlers for AppService-based bridge management.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/armorclaw/bridge/pkg/appservice"
)



// handleBridgeStart starts the bridge manager (AppService mode)
func (s *Server) handleBridgeStart(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.bridgeMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Bridge manager not configured",
		}
	}

	if err := s.bridgeMgr.Start(); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("failed to start bridge: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"status":  "started",
		"message": "Bridge manager started successfully",
	}, nil
}

// handleBridgeStop stops the bridge manager
func (s *Server) handleBridgeStop(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.bridgeMgr == nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: "Bridge manager not configured",
		}
	}

	if err := s.bridgeMgr.Stop(); err != nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: "Bridge manager not configured",
		}
	}

	return map[string]interface{}{
		"status":  "stopped",
		"message": "Bridge manager stopped successfully",
	}, nil
}

// handleBridgeStatus returns bridge manager status
func (s *Server) handleBridgeStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var stats map[string]interface{}

if s.bridgeMgr == nil {
		stats = map[string]interface{}{
			"enabled": false,
			"status":  "not_configured",
		}
	} else {
		stats = s.bridgeMgr.GetStats()
		stats["enabled"] = true
		stats["status"] = "running"
	}

	// Always include user_role from provisioning manager (ArmorChat fallback path)
	if s.provisioningMgr != nil {
		var params struct {
			UserID string `json:"user_id"`
		}
		if len(req.Params) > 0 {
			json.Unmarshal(req.Params, &params)
		}
		if params.UserID != "" {
			stats["user_role"] = string(s.provisioningMgr.GetUserRole(params.UserID))
		}
	}

	return stats, nil
}

// handleBridgeChannel creates a bridge between Matrix room and platform channel
func (s *Server) handleBridgeChannel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.bridgeMgr == nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: "Bridge manager not configured",
		}
	}

	var params struct {
		MatrixRoomID string `json:"matrix_room_id"`
		Platform     string `json:"platform"`
		ChannelID    string `json:"channel_id"`
	}

if len(req.Params) == 0 {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: "matrix_room_id, platform, and channel_id are required",
		}
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: err.Error(),
		}
	}

if params.MatrixRoomID == "" || params.Platform == "" || params.ChannelID == "" {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: "matrix_room_id, platform, and channel_id are required",
		}
	}

platform := appservice.Platform(params.Platform)
	if err := s.bridgeMgr.BridgeChannel(params.MatrixRoomID, platform, params.ChannelID); err != nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: "Bridge manager not configured",
		}
	}

	return map[string]interface{}{
		"status":        "bridged",
		"matrix_room":   params.MatrixRoomID,
		"platform":      params.Platform,
		"channel_id":    params.ChannelID,
	}, nil
}

// handleUnbridgeChannel removes a bridge
func (s *Server) handleUnbridgeChannel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
if s.bridgeMgr == nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: "Bridge manager not configured",
		}
	}

	var params struct {
		Platform  string `json:"platform"`
		ChannelID string `json:"channel_id"`
	}

if len(req.Params) == 0 {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: "platform and channel_id are required",
		}
	}

if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: err.Error(),
		}
	}

	if params.Platform == "" || params.ChannelID == "" {
		return nil, &ErrorObj{
			Code: InvalidParams,
			Message: "platform and channel_id are required",
		}
	}

platform := appservice.Platform(params.Platform)
	if err := s.bridgeMgr.UnbridgeChannel(platform, params.ChannelID); err != nil {
		return nil, &ErrorObj{
			Code: InternalError,
			Message: fmt.Sprintf("failed to unbridge channel: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"status":     "unbridged",
		"platform":   params.Platform,
		"channel_id": params.ChannelID,
	}, nil
}

// handleListBridgedChannels lists all bridged channels
func (s *Server) handleListBridgedChannels(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
if s.bridgeMgr == nil {
		return map[string]interface{}{
			"channels": []*appservice.BridgedChannel{},
			"count":    0,
		}, nil
	}

	channels := s.bridgeMgr.GetBridgedChannels()

	return map[string]interface{}{
		"channels": channels,
		"count":    len(channels),
}, nil
}

// handleGhostUserList lists registered ghost users
func (s *Server) handleGhostUserList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
if s.appService == nil {
		return map[string]interface{}{
			"ghost_users": []interface{}{},
			"count":       0,
		}, nil
	}

	stats := s.appService.GetStats()

	return map[string]interface{}{
		"ghost_users_count": stats["ghost_users"],
		"appservice_id":     stats["id"],
		"events_processed":  stats["events_processed"],
	}, nil
}

// handleAppServiceStatus returns AppService status
func (s *Server) handleAppServiceStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
if s.appService == nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "not_configured",
		}, nil
	}

	stats := s.appService.GetStats()

	return map[string]interface{}{
		"enabled":         true,
		"status":          "running",
		"id":              stats["id"],
		"homeserver":      stats["homeserver"],
		"ghost_users":     stats["ghost_users"],
		"event_buffer":    stats["event_buffer"],
		"events_processed": stats["events_processed"],
	}, nil
}
