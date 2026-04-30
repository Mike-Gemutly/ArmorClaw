package rpc

import (
	"context"
	"encoding/json"

	"github.com/armorclaw/bridge/pkg/secretary"
)

type secretaryHandlerError struct {
	code    int
	message string
}

func (e *secretaryHandlerError) Error() string {
	return e.message
}

type secretaryRPCHandler interface {
	Handle(method string, params json.RawMessage) (interface{}, error)
}

// secretaryRPCHandlerAdapter wraps a *secretary.RPCHandler to satisfy the
// secretaryRPCHandler interface, bridging the signature mismatch between
// the RPC server's (method, params) style and the secretary package's
// RPCRequest/RPCResponse style.
type secretaryRPCHandlerAdapter struct {
	handler *secretary.RPCHandler
}

// NewSecretaryHandler creates an RPC handler adapter for the secretary
// workflow engine. Pass the returned value as rpc.Config.SecretaryHandler.
func NewSecretaryHandler(h *secretary.RPCHandler) secretaryRPCHandler {
	if h == nil {
		return nil
	}
	return &secretaryRPCHandlerAdapter{handler: h}
}

func (a *secretaryRPCHandlerAdapter) Handle(method string, params json.RawMessage) (interface{}, error) {
	var userID string
	if len(params) > 0 {
		var p map[string]json.RawMessage
		if json.Unmarshal(params, &p) == nil {
			if uid, ok := p["user_id"]; ok {
				_ = json.Unmarshal(uid, &userID)
			}
		}
	}
	if userID == "" {
		userID = "rpc"
	}

	req := &secretary.RPCRequest{
		Method: method,
		Params: params,
		UserID: userID,
	}

	resp := a.handler.Handle(req)
	if resp == nil {
		return nil, nil
	}

	if resp.Error != nil {
		return nil, &secretaryHandlerError{
			code:    resp.Error.Code,
			message: resp.Error.Message,
		}
	}

	return resp.Result, nil
}

func (s *Server) handleSecretaryMethod(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.secretaryHandler == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "secretary service not initialized",
		}
	}

	result, err := s.secretaryHandler.Handle(req.Method, req.Params)
	if err != nil {
		if handlerErr, ok := err.(*secretaryHandlerError); ok {
			return nil, &ErrorObj{
				Code:    handlerErr.code,
				Message: handlerErr.message,
			}
		}
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: err.Error(),
		}
	}

	return result, nil
}
