// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
package rpc

import (
	"context"
	"encoding/json"

	"github.com/armorclaw/bridge/pkg/studio"
)

func requiresDelegationGate(method string) bool {
	switch method {
	case "studio.create_agent",
		"studio.update_agent",
		"studio.delete_agent",
		"studio.spawn_agent",
		"studio.stop_instance":
		return true
	default:
		return false
	}
}

// handleStudio routes all studio.* methods to the Agent Studio integration
func (s *Server) handleStudio(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	// Check if studio is initialized
	if s.studio == nil {
		return nil, &ErrorObj{
			Code:    MethodNotFound,
			Message: "Agent Studio not initialized",
		}
	}

	if requiresDelegationGate(req.Method) {
		// Extract userID from request params (from Matrix user_id field)
		userID := extractUserIDFromParams(req.Params)
		if userID != "" {
			if err := RequireDelegationReady(s.hardeningStore, userID); err != nil {
				return nil, &ErrorObj{
					Code:    InternalError,
					Message: err.Error(),
				}
			}
		}
	}

	// Handle via studio integration
	studioResp := s.studio.HandleRPCMethod(req.Method, req.Params)

	// Convert back to server response type
	resp := &Response{
		JSONRPC: studioResp.JSONRPC,
		ID:      req.ID,
	}
	if studioResp.Error != nil {
		resp.Error = &ErrorObj{
			Code:    studioResp.Error.Code,
			Message: studioResp.Error.Message,
			Data:    studioResp.Error.Data,
		}
	} else {
		resp.Result = studioResp.Result
	}

	return resp, nil
}

// GetStudio returns the studio integration for Matrix command handling
func (s *Server) GetStudio() *studio.StudioIntegration {
	if s.studio == nil {
		return nil
	}
	return s.studio.(*studio.StudioIntegration)
}

// StudioMethodList returns all available studio methods
func StudioMethodList() []string {
	return studio.StudioMethods
}

// IsStudioMethod checks if a method is a studio method
func IsStudioMethod(method string) bool {
	return len(method) > 7 && method[:7] == "studio."
}

// extractUserIDFromParams attempts to extract the user ID from request params
// In the Matrix integration, the user_id is passed in the request params
func extractUserIDFromParams(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}

	var p struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return ""
	}
	return p.UserID
}
