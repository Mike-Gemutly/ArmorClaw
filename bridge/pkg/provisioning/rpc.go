package provisioning

import (
	"context"
	"encoding/json"
	"fmt"
)

// RPCHandler provides JSON-RPC handlers for provisioning
type RPCHandler struct {
	manager *Manager
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(manager *Manager) *RPCHandler {
	return &RPCHandler{manager: manager}
}

// Method names
const (
	MethodStart          = "provisioning.start"
	MethodStatus         = "provisioning.status"
	MethodCancel         = "provisioning.cancel"
	MethodClaim          = "provisioning.claim"
	MethodRotateSecret   = "provisioning.rotate_secret"
	MethodList           = "provisioning.list"
	MethodGetQR          = "provisioning.get_qr"
)

// RPCRequest represents a generic RPC request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// RPCResponse represents a generic RPC response
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StartRequest is the request for provisioning.start
type StartRequest struct {
	ExpirySeconds int    `json:"expires_in,omitempty"`
	DeviceName    string `json:"device_name,omitempty"`
}

// StartResponse is the response for provisioning.start
type StartResponse struct {
	TokenID    string `json:"token_id"`
	DeepLink   string `json:"deep_link"`
	QRData     string `json:"qr_data"`
	ExpiresAt  int64  `json:"expires_at"`
	ExpiresIn  int    `json:"expires_in"`
	ServerName string `json:"server_name"`
}

// StatusResponse is the response for provisioning.status
type StatusResponse struct {
	TokenID    string       `json:"token_id"`
	Status     string       `json:"status"`
	ExpiresAt  int64        `json:"expires_at"`
	ClaimedBy  *DeviceInfo  `json:"claimed_by,omitempty"`
	Config     *Config      `json:"config,omitempty"`
}

// ClaimRequest is the request for provisioning.claim
type ClaimRequest struct {
	TokenID    string `json:"token_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// ClaimResponse is the response for provisioning.claim
type ClaimResponse struct {
	Claimed          bool   `json:"claimed"`
	DeviceRegistered bool   `json:"device_registered"`
	TokenID          string `json:"token_id"`
	MatrixHomeserver string `json:"matrix_homeserver"`
}

// Handle handles an RPC request
func (h *RPCHandler) Handle(ctx context.Context, req *RPCRequest) *RPCResponse {
	var result interface{}
	var err error

	switch req.Method {
	case MethodStart:
		result, err = h.handleStart(ctx, req.Params)
	case MethodStatus:
		result, err = h.handleStatus(ctx, req.Params)
	case MethodCancel:
		result, err = h.handleCancel(ctx, req.Params)
	case MethodClaim:
		result, err = h.handleClaim(ctx, req.Params)
	case MethodRotateSecret:
		result, err = h.handleRotateSecret(ctx, req.Params)
	case MethodList:
		result, err = h.handleList(ctx, req.Params)
	case MethodGetQR:
		result, err = h.handleGetQR(ctx, req.Params)
	default:
		return &RPCResponse{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32601, Message: "method not found"},
			ID:      req.ID,
		}
	}

	if err != nil {
		return &RPCResponse{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32000, Message: err.Error()},
			ID:      req.ID,
		}
	}

	return &RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func (h *RPCHandler) handleStart(ctx context.Context, params json.RawMessage) (*StartResponse, error) {
	var req StartRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	opts := &StartTokenOptions{
		ExpirySeconds: req.ExpirySeconds,
		DeviceName:    req.DeviceName,
	}

	token, err := h.manager.StartToken(ctx, opts)
	if err != nil {
		return nil, err
	}

	qrData, err := h.manager.GetQRData(token)
	if err != nil {
		return nil, err
	}

	expiresIn := int(token.ExpiresAt.Sub(token.CreatedAt).Seconds())

	return &StartResponse{
		TokenID:    token.ID,
		DeepLink:   qrData,
		QRData:     qrData,
		ExpiresAt:  token.ExpiresAt.Unix(),
		ExpiresIn:  expiresIn,
		ServerName: token.Config.ServerName,
	}, nil
}

func (h *RPCHandler) handleStatus(ctx context.Context, params json.RawMessage) (*StatusResponse, error) {
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	token, err := h.manager.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}

	return &StatusResponse{
		TokenID:   token.ID,
		Status:    string(token.Status),
		ExpiresAt: token.ExpiresAt.Unix(),
		ClaimedBy: token.ClaimedBy,
		Config:    token.Config,
	}, nil
}

func (h *RPCHandler) handleCancel(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if err := h.manager.CancelToken(req.TokenID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"cancelled": true,
		"token_id":  req.TokenID,
	}, nil
}

func (h *RPCHandler) handleClaim(ctx context.Context, params json.RawMessage) (*ClaimResponse, error) {
	var req ClaimRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if req.TokenID == "" || req.DeviceID == "" {
		return nil, fmt.Errorf("token_id and device_id are required")
	}

	device := &DeviceInfo{
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		UserAgent:  req.UserAgent,
	}

	token, err := h.manager.ClaimToken(ctx, req.TokenID, device)
	if err != nil {
		return nil, err
	}

	return &ClaimResponse{
		Claimed:          true,
		DeviceRegistered: true,
		TokenID:          token.ID,
		MatrixHomeserver: token.Config.MatrixHomeserver,
	}, nil
}

func (h *RPCHandler) handleRotateSecret(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		NewSecret string `json:"new_secret"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if req.NewSecret == "" {
		return nil, fmt.Errorf("new_secret is required")
	}

	if err := h.manager.RotateSecret(req.NewSecret); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"rotated": true,
		"warning": "Existing unclaimed tokens have been invalidated",
	}, nil
}

func (h *RPCHandler) handleList(ctx context.Context, params json.RawMessage) ([]*StatusResponse, error) {
	tokens := h.manager.ListTokens()

	var result []*StatusResponse
	for _, token := range tokens {
		result = append(result, &StatusResponse{
			TokenID:   token.ID,
			Status:    string(token.Status),
			ExpiresAt: token.ExpiresAt.Unix(),
		})
	}

	return result, nil
}

func (h *RPCHandler) handleGetQR(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	token, err := h.manager.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}

	qrData, err := h.manager.GetQRData(token)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token_id":   token.ID,
		"qr_data":    qrData,
		"deep_link":  qrData,
		"expires_at": token.ExpiresAt.Unix(),
		"status":     token.Status,
	}, nil
}
