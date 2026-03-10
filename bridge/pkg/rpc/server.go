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
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/internal/ai"
	"github.com/armorclaw/bridge/internal/events"
	"github.com/armorclaw/bridge/internal/skills"
	"github.com/armorclaw/bridge/pkg/appservice"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/eventlog"
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
	TooManyRequests  = -32001
	RequestCancelled = -32002
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
	ID      interface{}     `json:"id,omitempty"`
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

type Keystore interface{}
type MatrixAdapter interface {
	SendMessage(roomID, message, msgType string) (string, error)
	SendEvent(roomID, eventType string, content []byte) error
	Login(username, password string) error
	GetUserID() string
	IsLoggedIn() bool
	GetHomeserver() string
}

type MatrixHealthResult struct {
	Enabled    bool   `json:"enabled"`
	Connected  bool   `json:"connected"`
	LoggedIn   bool   `json:"logged_in"`
	Homeserver string `json:"homeserver"`
	UserID     string `json:"user_id,omitempty"`
	LastSync   string `json:"last_sync,omitempty"`
	Error      string `json:"error,omitempty"`
}

type HandlerFunc func(ctx context.Context, req *Request) (interface{}, *ErrorObj)

type Server struct {
	keystore        Keystore
	matrix          MatrixAdapter
	aiService       *ai.AIService
	aiSemaphore     chan struct{}
	aiMaxConcurrent int
	bridgeMgr       BridgeManager
	browserJobs     *BrowserJobManager
	studio          StudioService
	appService      AppService
	provisioningMgr ProvisioningManager
	skillMgr        SkillManager
	eventBus        *eventbus.EventBus
	handlers        map[string]HandlerFunc
}

type Config struct {
	SocketPath      string
	Keystore        Keystore
	Matrix          MatrixAdapter
	AIService       *ai.AIService
	AIMaxConcurrent int
	BridgeManager   BridgeManager
	BrowserJobs     *BrowserJobManager
	Studio          StudioService
	AppService      AppService
	ProvisioningMgr ProvisioningManager
	SkillManager    SkillManager
	EventBus        *eventbus.EventBus
}

func New(cfg Config) (*Server, error) {
	if cfg.AIMaxConcurrent <= 0 {
		cfg.AIMaxConcurrent = 4
	}

	s := &Server{
		keystore:        cfg.Keystore,
		matrix:          cfg.Matrix,
		aiService:       cfg.AIService,
		aiMaxConcurrent: cfg.AIMaxConcurrent,
		aiSemaphore:     make(chan struct{}, cfg.AIMaxConcurrent),
		bridgeMgr:       cfg.BridgeManager,
		browserJobs:     cfg.BrowserJobs,
		studio:          cfg.Studio,
		appService:      cfg.AppService,
		provisioningMgr: cfg.ProvisioningMgr,
		skillMgr:        cfg.SkillManager,
		eventBus:        cfg.EventBus,
		handlers:        make(map[string]HandlerFunc, 32),
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
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func errorResponse(id interface{}, code int, msg string) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   &ErrorObj{Code: code, Message: msg},
	}
}

// Matrix Handlers

func (s *Server) handleMatrixStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	matrix, ok := s.matrix.(*adapter.MatrixAdapter)
	if !ok || matrix == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "matrix adapter not configured",
		}
	}

	result := MatrixHealthResult{
		Enabled:    true,
		Connected:  true,
		LoggedIn:   matrix.IsLoggedIn(),
		Homeserver: matrix.GetHomeserver(),
		UserID:     matrix.GetUserID(),
	}

	if !matrix.IsLoggedIn() {
		result.Error = "not logged in"
	}

	return result, nil
}

func (s *Server) handleMatrixLogin(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Errorf("invalid parameters: %w", err).Error(),
		}
	}

	matrix, ok := s.matrix.(*adapter.MatrixAdapter)
	if !ok || matrix == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "matrix adapter not configured",
		}
	}

	if err := matrix.Login(params.Username, params.Password); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Errorf("login failed: %w", err).Error(),
		}
	}

	// Start sync loop after successful login
	matrix.StartSync()

	return map[string]interface{}{
		"success": true,
		"user_id": matrix.GetUserID(),
	}, nil
}

func (s *Server) handleMatrixSend(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		RoomID  string `json:"room_id"`
		Message string `json:"message"`
		MsgType string `json:"msgtype"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Errorf("invalid parameters: %w", err).Error(),
		}
	}

	matrix, ok := s.matrix.(*adapter.MatrixAdapter)
	if !ok || matrix == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "matrix adapter not configured",
		}
	}

	eventID, err := matrix.SendMessage(params.RoomID, params.Message, params.MsgType)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Errorf("send failed: %w", err).Error(),
		}
	}

	return map[string]interface{}{
		"event_id": eventID,
		"room_id":  params.RoomID,
	}, nil
}

func (s *Server) handleMatrixReceive(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		Cursor    string `json:"cursor"`
		TimeoutMs int    `json:"timeout_ms"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Errorf("invalid parameters: %w", err).Error(),
		}
	}

	if params.TimeoutMs <= 0 {
		params.TimeoutMs = 30000
	}

	matrix, ok := s.matrix.(*adapter.MatrixAdapter)
	if !ok || matrix == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "matrix adapter not configured",
		}
	}

	// If event bus is configured, use it for streaming
	if matrix.GetEventBus() != nil {
		return s.handleMatrixReceiveWithEventBus(ctx, req, &params, matrix)
	}

	// Fallback to polling old event queue
	return s.handleMatrixReceivePolling(ctx, req, &params, matrix)
}

func (s *Server) handleMatrixReceiveWithEventBus(ctx context.Context, req *Request, params *struct {
	Cursor    string `json:"cursor"`
	TimeoutMs int    `json:"timeout_ms"`
}, matrix *adapter.MatrixAdapter) (interface{}, *ErrorObj) {
	cursor, err := strconv.ParseUint(params.Cursor, 10, 64)
	if err != nil {
		cursor = 0
	}

	eventBus := matrix.GetEventBus()
	if eventBus == nil {
		return map[string]interface{}{
			"events":       []events.MatrixEvent{},
			"cursor":       strconv.FormatUint(cursor, 10),
			"count":        0,
			"cursor_reset": false,
		}, nil
	}

	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	recvCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	evs, next, reset := eventBus.WaitForEvents(recvCtx, cursor)

	if recvCtx.Err() != nil && len(evs) == 0 && !reset {
		return map[string]interface{}{
			"events":       []events.MatrixEvent{},
			"cursor":       strconv.FormatUint(cursor, 10),
			"count":        0,
			"cursor_reset": false,
		}, nil
	}

	return map[string]interface{}{
		"events":       evs,
		"cursor":       strconv.FormatUint(next, 10),
		"count":        len(evs),
		"cursor_reset": reset,
	}, nil
}

func (s *Server) handleMatrixReceivePolling(ctx context.Context, req *Request, params *struct {
	Cursor    string `json:"cursor"`
	TimeoutMs int    `json:"timeout_ms"`
}, matrix *adapter.MatrixAdapter) (interface{}, *ErrorObj) {
	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, &ErrorObj{
			Code:    RequestCancelled,
			Message: ctx.Err().Error(),
		}
	case <-time.After(timeout):
		// Poll the event queue (backward compatibility)
		events := make([]*adapter.MatrixEvent, 0, 10)
		timeoutChan := time.After(timeout)

		// Try to get events
		select {
		case e, ok := <-matrix.ReceiveEvents():
			if ok {
				events = append(events, e)
			}
		case <-timeoutChan:
			// Timed out
		}

		return map[string]interface{}{
			"events": events,
			"count":  len(events),
		}, nil
	}
}

func (s *Server) handleEventsReplay(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.eventBus == nil {
		return nil, &ErrorObj{Code: InternalError, Message: "event bus not initialized"}
	}

	log := s.eventBus.GetLog()
	if log == nil {
		return nil, &ErrorObj{Code: InternalError, Message: "durable log not enabled"}
	}

	var params struct {
		Offset uint64 `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{Code: InvalidParams, Message: err.Error()}
	}

	records, err := log.ReadFrom(params.Offset, params.Limit)
	if err != nil {
		return nil, &ErrorObj{Code: InternalError, Message: err.Error()}
	}

	return records, nil
}

func (s *Server) handleEventsStream(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.eventBus == nil {
		return nil, &ErrorObj{Code: InternalError, Message: "event bus not initialized"}
	}

	log := s.eventBus.GetLog()
	if log == nil {
		return nil, &ErrorObj{Code: InternalError, Message: "durable log not enabled"}
	}

	var params struct {
		Offset    uint64 `json:"offset"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{Code: InvalidParams, Message: err.Error()}
	}

	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Use long-polling
	lctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	records, err := log.WaitForEvents(lctx, params.Offset)
	if err != nil {
		if err == context.DeadlineExceeded {
			return map[string]interface{}{
				"events": []*eventlog.Record{},
				"count":  0,
			}, nil
		}
		return nil, &ErrorObj{Code: InternalError, Message: err.Error()}
	}

	return map[string]interface{}{
		"events": records,
		"count":  len(records),
	}, nil
}

func (s *Server) registerHandlers() {
	h := map[string]HandlerFunc{
		"ai.chat":                  s.handleAIChat,
		"browser.navigate":         s.handleBrowserNavigate,
		"browser.fill":             s.handleBrowserFill,
		"browser.click":            s.handleBrowserClick,
		"browser.status":           s.handleBrowserStatus,
		"browser.wait_for_element": s.handleBrowserWaitForElement,
		"browser.wait_for_captcha": s.handleBrowserWaitForCaptcha,
		"browser.wait_for_2fa":     s.handleBrowserWaitFor2FA,
		"browser.complete":         s.handleBrowserComplete,
		"browser.fail":             s.handleBrowserFail,
		"browser.list":             s.handleBrowserList,
		"browser.cancel":           s.handleBrowserCancel,
		"bridge.start":             s.handleBridgeStart,
		"bridge.stop":              s.handleBridgeStop,
		"bridge.status":            s.handleBridgeStatus,
		"bridge.channel":           s.handleBridgeChannel,
		"bridge.unchannel":         s.handleUnbridgeChannel,
		"bridge.list":              s.handleListBridgedChannels,
		"bridge.ghost_list":        s.handleGhostUserList,
		"bridge.appservice_status": s.handleAppServiceStatus,
		"pii.request":              s.handlePIIRequest,
		"pii.approve":              s.handlePIIApprove,
		"pii.deny":                 s.handlePIIDeny,
		"pii.status":               s.handlePIIStatus,
		"pii.list_pending":         s.handlePIIListPending,
		"pii.stats":                s.handlePIIStats,
		"pii.cancel":               s.handlePIICancel,
		"pii.fulfill":              s.handlePIIFulfill,
		"pii.wait_for_approval":    s.handlePIIWaitForApproval,
		"skills.execute":           s.handleSkillsExecute,
		"skills.list":              s.handleSkillsList,
		"skills.get_schema":        s.handleSkillsGetSchema,
		"skills.allow":             s.handleSkillsAllow,
		"skills.block":             s.handleSkillsBlock,
		"skills.allowlist_add":     s.handleSkillsAllowlistAdd,
		"skills.allowlist_remove":  s.handleSkillsAllowlistRemove,
		"skills.allowlist_list":    s.handleSkillsAllowlistList,
		"skills.web_search":        s.handleSkillsWebSearch,
		"skills.web_extract":       s.handleSkillsWebExtract,
		"skills.email_send":        s.handleSkillsEmailSend,
		"skills.slack_message":     s.handleSkillsSlackMessage,
		"skills.file_read":         s.handleSkillsFileRead,
		"skills.data_analyze":      s.handleSkillsDataAnalyze,
		"matrix.status":            s.handleMatrixStatus,
		"matrix.login":             s.handleMatrixLogin,
		"matrix.send":              s.handleMatrixSend,
		"matrix.receive":           s.handleMatrixReceive,
		"events.replay":            s.handleEventsReplay,
		"events.stream":            s.handleEventsStream,
		"studio.deploy":            s.handleStudio,
	}

	s.handlers = h
}

// GetMatrixAdapter returns the Matrix adapter for external integration
func (s *Server) GetMatrixAdapter() MatrixAdapter {
	return s.matrix
}

func (s *Server) Run(socketPath string) error {
	var listener net.Listener
	var err error

	if runtime.GOOS == "windows" {
		// Use TCP fallback on Windows
		addr := "127.0.0.1:6168"
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to create TCP listener on %s: %w", addr, err)
		}
		slog.Info("RPC transport: tcp (windows fallback)", "address", addr)
	} else {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}

		// Remove existing socket file if present (cleanup stale socket)
		if _, err := os.Stat(socketPath); err == nil {
			if err := os.Remove(socketPath); err != nil {
				return fmt.Errorf("failed to remove existing socket file: %w", err)
			}
		}

		// Create Unix domain socket
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("failed to create Unix socket: %w", err)
		}
		slog.Info("RPC transport: unix socket", "path", socketPath)

		// Set appropriate permissions (mode 660)
		if err := os.Chmod(socketPath, 0660); err != nil {
			slog.Warn("failed to set socket permissions", "error", err)
		}
	}
	defer listener.Close()

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
