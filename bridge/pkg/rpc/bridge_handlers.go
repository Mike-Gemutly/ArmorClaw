// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
// This file contains handlers for AppService-based bridge management.
package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/armorclaw/bridge/pkg/appservice"
)

// BridgeManager interface for RPC server integration
type BridgeManager interface {
	Start() error
	Stop() error
	RegisterAdapter(platform appservice.Platform, adapter interface{}) error
	BridgeChannel(matrixRoomID string, platform appservice.Platform, channelID string) error
	UnbridgeChannel(platform appservice.Platform, channelID string) error
	GetBridgedChannels() []*appservice.BridgedChannel
	GetStats() map[string]interface{}
}

// handleBridgeStart starts the bridge manager (AppService mode)
func (s *Server) handleBridgeStart(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Bridge manager not configured",
			},
		}
	}

	if err := s.bridgeMgr.Start(); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to start bridge: %s", err.Error()),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":  "started",
			"message": "Bridge manager started successfully",
		},
	}
}

// handleBridgeStop stops the bridge manager
func (s *Server) handleBridgeStop(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Bridge manager not configured",
			},
		}
	}

	if err := s.bridgeMgr.Stop(); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to stop bridge: %s", err.Error()),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":  "stopped",
			"message": "Bridge manager stopped successfully",
		},
	}
}

// handleBridgeStatus returns bridge manager status
func (s *Server) handleBridgeStatus(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"enabled": false,
				"status":  "not_configured",
			},
		}
	}

	stats := s.bridgeMgr.GetStats()
	stats["enabled"] = true
	stats["status"] = "running"

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  stats,
	}
}

// handleBridgeChannel creates a bridge between Matrix room and platform channel
func (s *Server) handleBridgeChannel(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Bridge manager not configured",
			},
		}
	}

	var params struct {
		MatrixRoomID string `json:"matrix_room_id"`
		Platform     string `json:"platform"`
		ChannelID    string `json:"channel_id"`
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "matrix_room_id, platform, and channel_id are required",
			},
		}
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: err.Error(),
			},
		}
	}

	if params.MatrixRoomID == "" || params.Platform == "" || params.ChannelID == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "matrix_room_id, platform, and channel_id are required",
			},
		}
	}

	platform := appservice.Platform(params.Platform)
	if err := s.bridgeMgr.BridgeChannel(params.MatrixRoomID, platform, params.ChannelID); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to bridge channel: %s", err.Error()),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":        "bridged",
			"matrix_room":   params.MatrixRoomID,
			"platform":      params.Platform,
			"channel_id":    params.ChannelID,
		},
	}
}

// handleUnbridgeChannel removes a bridge
func (s *Server) handleUnbridgeChannel(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Bridge manager not configured",
			},
		}
	}

	var params struct {
		Platform  string `json:"platform"`
		ChannelID string `json:"channel_id"`
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "platform and channel_id are required",
			},
		}
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: err.Error(),
			},
		}
	}

	if params.Platform == "" || params.ChannelID == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "platform and channel_id are required",
			},
		}
	}

	platform := appservice.Platform(params.Platform)
	if err := s.bridgeMgr.UnbridgeChannel(platform, params.ChannelID); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to unbridge channel: %s", err.Error()),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":     "unbridged",
			"platform":   params.Platform,
			"channel_id": params.ChannelID,
		},
	}
}

// handleListBridgedChannels lists all bridged channels
func (s *Server) handleListBridgedChannels(req *Request) *Response {
	if s.bridgeMgr == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"channels": []*appservice.BridgedChannel{},
				"count":    0,
			},
		}
	}

	channels := s.bridgeMgr.GetBridgedChannels()

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"channels": channels,
			"count":    len(channels),
		},
	}
}

// handleGhostUserList lists registered ghost users
func (s *Server) handleGhostUserList(req *Request) *Response {
	if s.appService == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"ghost_users": []interface{}{},
				"count":       0,
			},
		}
	}

	stats := s.appService.GetStats()

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"ghost_users_count": stats["ghost_users"],
			"appservice_id":     stats["id"],
			"events_processed":  stats["events_processed"],
		},
	}
}

// handleAppServiceStatus returns AppService status
func (s *Server) handleAppServiceStatus(req *Request) *Response {
	if s.appService == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"enabled": false,
				"status":  "not_configured",
			},
		}
	}

	stats := s.appService.GetStats()

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"enabled":         true,
			"status":          "running",
			"id":              stats["id"],
			"homeserver":      stats["homeserver"],
			"ghost_users":     stats["ghost_users"],
			"event_buffer":    stats["event_buffer"],
			"events_processed": stats["events_processed"],
		},
	}
}
