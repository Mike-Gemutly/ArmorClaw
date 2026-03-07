package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/armorclaw/bridge/internal/ai"
	"github.com/armorclaw/bridge/internal/skills"
	"github.com/armorclaw/bridge/pkg/appservice"
	"github.com/armorclaw/bridge/pkg/provisioning"
	"github.com/armorclaw/bridge/pkg/studio"
)

	const (
		JSONRPCVersion   = "2.0"
		BridgeVersion    = "4.6.0"
		ParseError       = -32700
		InvalidRequest   = -32600
		MethodNotFound   = -32601
		InvalidParams    = -32602
		InternalError    = -32603
		TooManyRequests   = -32001
		RequestCancelled  = -32002
	)

	type BridgeManager interface {
		Start() error
		Stop() error
		RegisterAdapter(platform appservice.Platform, adapter interface{}) error
		BridgeChannel(matrixRoomID string, platform appservice.Platform, channelID string) error
		UnbridgeChannel(platform appservice.Platform, channelID string) error
		GetBridgedChannels() []*appservice.BridgedChannel
		GetStats() map[string]interface{}
	}

	type StudioService interface {
		HandleRPCMethod(method string, params json.RawMessage) *studio.RPCResponse
	}

	type ProvisioningManager interface {
		GetUserRole(userID string) provisioning.AdminRole
	}

	type AppService interface {
		GetStats() map[string]interface{}
	}

	type SkillManager interface {
		ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (*skills.SkillResult, error)
	ListEnabled() []*skills.Skill
	GetSkill(skillName string) (*skills.Skill, bool)
	AllowSkill(skillName string) error
	BlockSkill(skillName string) error
	AllowIP(ip string) error
	AllowCIDR(cidr string) error
	GetAllowlist() ([]string, []string)
	GenerateSchema(skill *skills.Skill) interface{}
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}    `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObj   `json:"error,omitempty"`
}

type ErrorObj struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Keystore interface {}
type MatrixAdapter interface {
	SendEvent(roomID string, eventType string, content interface{}) error
}

type HandlerFunc func(ctx context.Context, req *Request) (interface{}, *ErrorObj)

type Server struct {
	keystore  Keystore
	matrix    MatrixAdapter
	aiService *ai.AIService
	aiSemaphore chan struct{}
	aiMaxConcurrent int
	bridgeMgr    BridgeManager
	browserJobs  BrowserJobManager
	studio       StudioService
	appService   AppService
	provisioningMgr ProvisioningManager
	skillMgr     SkillManager
	handlers    map[string]HandlerFunc
}

type Config struct {
	SocketPath       string
	Keystore         Keystore
	Matrix           MatrixAdapter
	AIService        *ai.AIService
	AIMaxConcurrent  int
	BridgeManager    BridgeManager
	BrowserJobs      *BrowserJobManager
	Studio           StudioService
	AppService       AppService
	ProvisioningMgr  ProvisioningManager
	SkillManager     SkillManager
}

func New(cfg Config) (*Server, error) {
	if cfg.AIMaxConcurrent <= 0 {
		cfg.AIMaxConcurrent = 4
	}

	s := &Server{
		keystore: cfg.Keystore,
		matrix: cfg.Matrix,
		aiService: cfg.AIService,
		aiMaxConcurrent: cfg.AIMaxConcurrent,
		aiSemaphore: make(chan struct{}, cfg.AIMaxConcurrent),
		bridgeMgr: cfg.BridgeManager,
		browserJobs: *cfg.BrowserJobs,
		studio: cfg.Studio,
		appService: cfg.AppService,
		provisioningMgr: cfg.ProvisioningMgr,
		skillMgr: cfg.SkillManager,
		handlers: make(map[string]HandlerFunc, 32),
	}
	
	s.registerHandlers()
	return s, nil
}

func (s *Server) Handle(ctx context.Context, req *Request) (resp *Response) {
	defer func() {
		if r := recover(); r != nil {
			var id interface{}
			var method string
			if req != nil {
				id = req.ID
				method = req.Method
			}
			slog.Error(
				"rpc_panic",
				"method", method,
				"id", id,
				"recover", r,
			)
			resp = errorResponse(id, InternalError, "internal server error")
		}
	}()

	if req == nil {
		return errorResponse(nil, InvalidRequest, "request is nil")
	}

	isNotification := req.ID == nil

	if req.JSONRPC != JSONRPCVersion {
		if isNotification {
			return nil
		}
		return errorResponse(req.ID, InvalidRequest, "invalid jsonrpc version")
	}

	if req.Method == "" {
		slog.Warn("rpc_invalid_method")
		if isNotification {
			return nil
		}
		return errorResponse(req.ID, InvalidRequest, "method is required")
	}

	handler, ok := s.handlers[req.Method]
	if !ok {
		if isNotification {
			return nil
		}
		return errorResponse(req.ID, MethodNotFound, "method not found")
	}

	result, rpcErr := handler(ctx, req)

	if isNotification {
		return nil
	}

	if rpcErr != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID: req.ID,
			Error: rpcErr,
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID: req.ID,
		Result: result,
	}
}

func errorResponse(id interface{}, code int, msg string) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID: id,
		Error: &ErrorObj{Code: code, Message: msg},
	}
}

func (s *Server) registerHandlers() {
	h := map[string]HandlerFunc{
		"ai.chat": s.handleAIChat,
		"browser.navigate": s.handleBrowserNavigate,
		"browser.fill": s.handleBrowserFill,
		"browser.click": s.handleBrowserClick,
		"browser.status": s.handleBrowserStatus,
		"browser.wait_for_element": s.handleBrowserWaitForElement,
		"browser.wait_for_captcha": s.handleBrowserWaitForCaptcha,
		"browser.wait_for_2fa": s.handleBrowserWaitFor2FA,
		"browser.complete": s.handleBrowserComplete,
		"browser.fail": s.handleBrowserFail,
		"browser.list": s.handleBrowserList,
		"browser.cancel": s.handleBrowserCancel,
		"bridge.start": s.handleBridgeStart,
		"bridge.stop": s.handleBridgeStop,
		"bridge.status": s.handleBridgeStatus,
		"bridge.channel": s.handleBridgeChannel,
		"bridge.unchannel": s.handleUnbridgeChannel,
		"bridge.list": s.handleListBridgedChannels,
		"bridge.ghost_list": s.handleGhostUserList,
		"bridge.appservice_status": s.handleAppServiceStatus,
		"pii.request": s.handlePIIRequest,
		"pii.approve": s.handlePIIApprove,
		"pii.deny": s.handlePIIDeny,
		"pii.status": s.handlePIIStatus,
		"pii.list_pending": s.handlePIIListPending,
		"pii.stats": s.handlePIIStats,
		"pii.cancel": s.handlePIICancel,
		"pii.fulfill": s.handlePIIFulfill,
		"pii.wait_for_approval": s.handlePIIWaitForApproval,
		"studio.deploy": s.handleStudio,
		"skills.execute": s.handleSkillsExecute,
		"skills.list": s.handleSkillsList,
		"skills.get_schema": s.handleSkillsGetSchema,
		"skills.allow": s.handleSkillsAllow,
		"skills.block": s.handleSkillsBlock,
		"skills.allowlist_add": s.handleSkillsAllowlistAdd,
		"skills.allowlist_remove": s.handleSkillsAllowlistRemove,
		"skills.allowlist_list": s.handleSkillsAllowlistList,
	}
	s.handlers = h
}

func (s *Server) Run(socketPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if present
	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket file: %w", err)
		}
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %w", err)
	}
	defer listener.Close()

	// Set appropriate permissions (mode 660)
	if err := os.Chmod(socketPath, 0660); err != nil {
		slog.Warn("failed to set socket permissions", "error", err)
	}

	// Create channel to signal shutdown
	shutdown := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		slog.Info("shutting down rpc server")
		close(shutdown)
	}()

	// Main event loop
	for {
		select {
		case <-shutdown:
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-shutdown:
					return nil
				default:
					if os.IsTimeout(err) {
						continue
					}
					return fmt.Errorf("accept error: %w", err)
				}
			}

			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read JSON-RPC request
	var req Request
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&req); err != nil {
		slog.Warn("rpc_decode_error", "error", err)
		return
	}

	// Handle request
	resp := s.Handle(context.Background(), &req)

	// Write response
	encoder := json.NewEncoder(conn)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resp); err != nil {
		slog.Warn("rpc_write_error", "error", err)
	}
}

