package rpc

import (
	"context"
	"encoding/json"
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
