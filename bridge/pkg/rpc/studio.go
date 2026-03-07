// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
package rpc

import (
	"context"

	"github.com/armorclaw/bridge/pkg/studio"
)

// handleStudio routes all studio.* methods to the Agent Studio integration
func (s *Server) handleStudio(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	// Check if studio is initialized
	if s.studio == nil {
		return nil, &ErrorObj{
			Code:    MethodNotFound,
			Message: "Agent Studio not initialized",
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
	return studio.IsStudioMethod(method)
}
