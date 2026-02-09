// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
// The bridge exposes methods for container lifecycle and credential management.
package rpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/docker"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/logger"
)

const (
	// DefaultSocketPath is the default Unix socket path
	DefaultSocketPath = "/run/armorclaw/bridge.sock"

	// Centralized path constants for ArmorClaw runtime directories
	DefaultRuntimeDir   = "/run/armorclaw"
	DefaultContainerDir = "/run/armorclaw/containers"
	DefaultSecretsDir   = "/run/armorclaw/secrets"
	DefaultConfigsDir   = "/run/armorclaw/configs"
)

var (
	ErrInvalidRequest    = errors.New("invalid JSON-RPC request")
	ErrMethodNotFound    = errors.New("method not found")
	ErrInvalidParams     = errors.New("invalid parameters")
	ErrContainerTimeout   = errors.New("container operation timed out")
	ErrContainerConflict  = errors.New("container with this name already exists")
)

// Human-readable error messages with helpful suggestions
var errorSuggestions = map[int]string{
	KeyNotFound: `
The API key was not found in the keystore.

Available commands:
  armorclaw-bridge list-keys           # List all stored keys
  armorclaw-bridge add-key --provider openai --token sk-xxx  # Add a new key

Example usage:
  armorclaw-bridge start --key openai-default
`,
	InvalidParams: `
Invalid parameters provided.

Common mistakes:
  - Missing required fields (key_id, container_id, etc.)
  - Invalid JSON format
  - Wrong data types

Use 'armorclaw-bridge --help' for command examples.
`,
	InternalError: `
An internal error occurred.

Troubleshooting:
  1. Check Docker is running: docker ps
  2. Verify bridge status: armorclaw-bridge status
  3. Check logs for detailed error messages

If the problem persists, please report this issue at:
  https://github.com/armorclaw/armorclaw/issues
`,
}

// getHelpfulError returns a human-readable error message with suggestions
func getHelpfulError(code int, message string) string {
	if suggestion, ok := errorSuggestions[code]; ok {
		return message + suggestion
	}
	return message
}

const (
	// Default container operation timeout
	DefaultContainerTimeout = 2 * time.Minute
	// Maximum number of containers
	MaxContainers = 100
)

// JSONRPC 2.0 request/response structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObj   `json:"error,omitempty"`
}

type ErrorObj struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes (JSON-RPC 2.0 + custom)
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603

	// Custom errors
	ContainerRunning = -1
	ContainerStopped = -2
	KeyNotFound       = -3
)

// Server is a JSON-RPC 2.0 server over Unix domain socket
type Server struct {
	socketPath    string
	listener      net.Listener
	keystore      *keystore.Keystore
	matrix        *adapter.MatrixAdapter
	docker        *docker.Client
	containers    map[string]*ContainerInfo
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	securityLog   *logger.SecurityLogger
	wg            sync.WaitGroup
	containerDir  string // Directory for container-specific sockets
}

// ContainerInfo holds information about a running container
type ContainerInfo struct {
	ID       string
	Name     string
	State    string // "running", "stopped", "paused"
	Pid      int
	Created  int64
	Endpoint string // Unix socket for this container
}

// Config holds server configuration
type Config struct {
	SocketPath       string
	Keystore         *keystore.Keystore
	MatrixHomeserver string
	MatrixUsername   string
	MatrixPassword   string
}

// New creates a new JSON-RPC server
func New(cfg Config) (*Server, error) {
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath
	}

	if cfg.Keystore == nil {
		return nil, errors.New("keystore is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Docker client
	dockerClient, err := docker.New(docker.Config{
		Scopes: []docker.Scope{docker.ScopeCreate, docker.ScopeExec, docker.ScopeRemove},
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Set container socket directory using centralized constant
	containerDir := DefaultContainerDir

	server := &Server{
		socketPath:   cfg.SocketPath,
		keystore:     cfg.Keystore,
		docker:       dockerClient,
		containers:   make(map[string]*ContainerInfo),
		containerDir: containerDir,
		ctx:          ctx,
		cancel:       cancel,
		securityLog:  logger.NewSecurityLogger(logger.Global().WithComponent("rpc")),
	}

	// Initialize Matrix adapter if homeserver is configured
	if cfg.MatrixHomeserver != "" {
		matrix, err := adapter.New(adapter.Config{
			HomeserverURL: cfg.MatrixHomeserver,
			DeviceID:      "armorclaw-bridge",
		})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create Matrix adapter: %w", err)
		}
		server.matrix = matrix

		// Auto-login if credentials provided
		if cfg.MatrixUsername != "" && cfg.MatrixPassword != "" {
			if err := matrix.Login(cfg.MatrixUsername, cfg.MatrixPassword); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to login to Matrix: %w", err)
			}
			// Start background sync
			matrix.StartSync()
		}
	}

	return server, nil
}

// Start starts the JSON-RPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure socket directory exists
	socketDir := s.socketPath[:len(s.socketPath)-len("/bridge.sock")]
	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if present
	if _, err := os.Stat(s.socketPath); err == nil {
		os.Remove(s.socketPath)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions (owner + group read/write)
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the JSON-RPC server
func (s *Server) Stop() error {
	s.cancel()

	// Close Matrix adapter
	if s.matrix != nil {
		s.matrix.Close()
	}

	// Close Docker client
	if s.docker != nil {
		s.docker.Close()
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return err
		}
	}

	s.wg.Wait()
	return nil
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					// Log error in production
				}
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			var req Request
			if err := decoder.Decode(&req); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				s.sendError(encoder, nil, ParseError, err.Error(), nil)
				continue
			}

			// Process request
			resp := s.handleRequest(&req)
			if err := encoder.Encode(resp); err != nil {
				return
			}

			// Check for notification (no ID = no response expected)
			if req.ID == nil {
				return
			}
		}
	}
}

// handleRequest processes a single JSON-RPC request
func (s *Server) handleRequest(req *Request) *Response {
	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidRequest,
				Message: "invalid JSON-RPC version",
			},
		}
	}

	// Route to handler
	switch req.Method {
	case "status":
		return s.handleStatus(req)
	case "health":
		return s.handleHealth(req)
	case "start":
		return s.handleStart(req)
	case "stop":
		return s.handleStop(req)
	case "list_keys":
		return s.handleListKeys(req)
	case "get_key":
		return s.handleGetKey(req)
	case "matrix.send":
		return s.handleMatrixSend(req)
	case "matrix.receive":
		return s.handleMatrixReceive(req)
	case "matrix.status":
		return s.handleMatrixStatus(req)
	case "matrix.login":
		return s.handleMatrixLogin(req)
	case "attach_config":
		return s.handleAttachConfig(req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("method '%s' not found", req.Method),
			},
		}
	}
}

// handleStatus returns bridge status
func (s *Server) handleStatus(req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"version":     "1.0.0",
		"state":       "running",
		"socket":      s.socketPath,
		"containers":  len(s.containers),
		"container_ids": func() []string {
			ids := make([]string, 0, len(s.containers))
			for _, c := range s.containers {
				ids = append(ids, c.ID)
			}
			return ids
		}(),
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  status,
	}
}

// handleHealth returns health check result
func (s *Server) handleHealth(req *Request) *Response {
	// Check if keystore is accessible
	if s.keystore == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "keystore not initialized",
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status": "healthy",
		},
	}
}

// handleStart starts a container with credentials
func (s *Server) handleStart(req *Request) *Response {
	// Parse parameters
	var params struct {
		KeyID     string `json:"key_id"`
		AgentType string `json:"agent_type"`
		Image     string `json:"image"`
	}

	if len(req.Params) > 0 {
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
	}

	// Validate key_id
	if params.KeyID == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "key_id is required",
			},
		}
	}

	// Set defaults
	if params.AgentType == "" {
		params.AgentType = "openclaw"
	}
	if params.Image == "" {
		params.Image = "armorclaw/agent:v1"
	}

	// Check keystore availability
	if s.keystore == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "keystore not initialized",
			},
		}
	}

	// 1. Retrieve credentials from keystore
	cred, err := s.keystore.Retrieve(params.KeyID)
	if err != nil {
		// Log secret access failure
		s.securityLog.LogSecretAccess(s.ctx, params.KeyID, "unknown", slog.String("status", "failed"))
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    KeyNotFound,
				Message: getHelpfulError(KeyNotFound, fmt.Sprintf("Key '%s' not found", params.KeyID)),
			},
		}
	}

	// Log successful secret access
	s.securityLog.LogSecretAccess(s.ctx, params.KeyID, string(cred.Provider), slog.String("status", "success"))

	// 2. Create container-specific socket directory
	if err := os.MkdirAll(s.containerDir, 0750); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to create container directory: %v", err),
			},
		}
	}

	// Generate container name and socket path
	containerName := fmt.Sprintf("armorclaw-%s-%d", params.AgentType, time.Now().Unix())

	// Check for container name collision
	if s.checkContainerNameExists(containerName) {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("container name collision detected: %s", containerName),
			},
		}
	}

	socketPath := filepath.Join(s.containerDir, containerName+".sock")

	// 3. Create secrets file for injection
	// The container will mount this file and read secrets from it
	// Using a file instead of named pipe for cross-platform compatibility
	secretsDir := DefaultSecretsDir
	secretsPath := filepath.Join(secretsDir, containerName+".json")

	// Create secrets directory
	if err := os.MkdirAll(secretsDir, 0750); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to create secrets directory: %v", err),
			},
		}
	}

	// Prepare secrets JSON
	secretsJSON := map[string]interface{}{
		"provider":     cred.Provider,
		"token":        cred.Token,
		"display_name": cred.DisplayName,
	}
	secretsData, err := json.Marshal(secretsJSON)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to marshal secrets: %v", err),
			},
		}
	}

	// Write secrets to file
	if err := os.WriteFile(secretsPath, secretsData, 0640); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to write secrets file: %v", err),
			},
		}
	}

	// Log secret injection
	s.securityLog.LogSecretInject(s.ctx, containerName, params.KeyID,
		slog.String("secrets_path", secretsPath),
		slog.String("image", params.Image),
	)

	// 4. Create container config with secrets injection
	containerConfig := &container.Config{
		Image: params.Image,
		Env: []string{
			fmt.Sprintf("ARMORCLAW_KEY_ID=%s", params.KeyID),
			fmt.Sprintf("ARMORCLAW_ENDPOINT=%s", socketPath),
			"ARMORCLAW_SECRETS_PATH=/run/secrets",
		},
	}

	// Mount secrets file into container at /run/secrets (fixed location)
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/run/secrets:ro", secretsPath),
		},
		AutoRemove: true, // Auto-remove on exit
	}

	// 5. Start the container with timeout
	ctx, cancel := context.WithTimeout(context.Background(), DefaultContainerTimeout)
	defer cancel()

	containerID, err := s.docker.CreateAndStartContainer(
		ctx,
		containerConfig,
		hostConfig,
		nil, // networkingConfig
		nil, // platform
	)
	if err != nil {
		// Rollback: clean up secrets file and socket path
		s.rollbackContainerStart(containerName, secretsPath, socketPath)

		// Check if timeout occurred
		if errors.Is(err, context.DeadlineExceeded) {
			// Log container timeout
			s.securityLog.LogContainerError(s.ctx, containerName, "", "timeout", "container start timed out",
				slog.String("key_id", params.KeyID),
			)
			return &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &ErrorObj{
					Code:    InternalError,
					Message: "container start timed out",
				},
			}
		}

		// Log container error
		s.securityLog.LogContainerError(s.ctx, containerName, "", "start_failed", err.Error(),
			slog.String("key_id", params.KeyID),
		)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to start container: %v", err),
			},
		}
	}

	// 6. Track container
	s.mu.Lock()
	s.containers[containerID] = &ContainerInfo{
		ID:       containerID,
		Name:     containerName,
		State:    "running",
		Created:  time.Now().Unix(),
		Endpoint: socketPath,
	}
	s.mu.Unlock()

	// Log container start success
	s.securityLog.LogContainerStart(s.ctx, containerName, containerID, params.Image,
		slog.String("key_id", params.KeyID),
		slog.String("socket_path", socketPath),
	)

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"container_id": containerID,
			"container_name": containerName,
			"status":       "running",
			"endpoint":     socketPath,
		},
	}
}

// handleStop stops a running container
func (s *Server) handleStop(req *Request) *Response {
	var params struct {
		ContainerID string `json:"container_id"`
	}

	if len(req.Params) > 0 {
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
	}

	// Validate container_id
	if params.ContainerID == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "container_id is required",
			},
		}
	}

	// Check if container exists
	s.mu.Lock()
	info, exists := s.containers[params.ContainerID]
	if !exists {
		s.mu.Unlock()
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    ContainerStopped,
				Message: "container not found",
			},
		}
	}
	s.mu.Unlock()

	// Remove container with force
	err := s.docker.RemoveContainer(context.Background(), params.ContainerID, true)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to remove container: %v", err),
			},
		}
	}

	// Clean up container-specific socket
	if info.Endpoint != "" {
		os.Remove(info.Endpoint)
	}

	// Remove from tracking
	s.mu.Lock()
	delete(s.containers, params.ContainerID)
	s.mu.Unlock()

	// Log container stop
	s.securityLog.LogContainerStop(s.ctx, info.Name, params.ContainerID, "user_requested",
		slog.String("endpoint", info.Endpoint),
	)

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":        "stopped",
			"container_id":  params.ContainerID,
			"container_name": info.Name,
		},
	}
}

// handleListKeys lists available keys
func (s *Server) handleListKeys(req *Request) *Response {
	var params struct {
		Provider string `json:"provider"`
	}

	if len(req.Params) > 0 {
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
	}

	// Check keystore availability
	if s.keystore == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "keystore not initialized",
			},
		}
	}

	keys, err := s.keystore.List(keystore.Provider(params.Provider))
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	// Convert to []map for JSON serialization
	result := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		result[i] = map[string]interface{}{
			"id":           k.ID,
			"provider":     k.Provider,
			"display_name": k.DisplayName,
			"created_at":   k.CreatedAt,
			"expires_at":   k.ExpiresAt,
			"tags":         k.Tags,
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleGetKey retrieves a key
func (s *Server) handleGetKey(req *Request) *Response {
	var params struct {
		ID string `json:"id"`
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "id parameter required",
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

	// Check keystore availability
	if s.keystore == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "keystore not initialized",
			},
		}
	}

	cred, err := s.keystore.Retrieve(params.ID)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			return &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &ErrorObj{
					Code:    KeyNotFound,
					Message: "key not found",
				},
			}
		}
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"id":           cred.ID,
			"provider":     cred.Provider,
			"token":        cred.Token,
			"display_name": cred.DisplayName,
			"created_at":   cred.CreatedAt,
			"expires_at":   cred.ExpiresAt,
			"tags":         cred.Tags,
		},
	}
}

// handleMatrixSend sends a message to a Matrix room
func (s *Server) handleMatrixSend(req *Request) *Response {
	if s.matrix == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Matrix adapter not configured",
			},
		}
	}

	var params struct {
		RoomID  string `json:"room_id"`
		Message string `json:"message"`
		MsgType string `json:"msgtype"`
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "room_id and message are required",
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

	if params.RoomID == "" || params.Message == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "room_id and message are required",
			},
		}
	}

	// Default to text message
	if params.MsgType == "" {
		params.MsgType = "m.text"
	}

	eventID, err := s.matrix.SendMessage(params.RoomID, params.Message, params.MsgType)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"event_id": eventID,
			"room_id":  params.RoomID,
		},
	}
}

// handleMatrixReceive returns pending Matrix events
func (s *Server) handleMatrixReceive(req *Request) *Response {
	if s.matrix == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Matrix adapter not configured",
			},
		}
	}

	// Collect up to 10 pending events
	events := make([]*adapter.MatrixEvent, 0, 10)
	eventChan := s.matrix.ReceiveEvents()

	for i := 0; i < 10; i++ {
		select {
		case event := <-eventChan:
			if event != nil {
				events = append(events, event)
			}
		default:
			// No more events
			break
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"events": events,
			"count":  len(events),
		},
	}
}

// handleMatrixStatus returns Matrix connection status
func (s *Server) handleMatrixStatus(req *Request) *Response {
	if s.matrix == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"enabled": false,
				"status":  "not_configured",
			},
		}
	}

	userID := s.matrix.GetUserID()
	token := s.matrix.GetAccessToken()

	status := "disconnected"
	if token != "" {
		status = "connected"
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"enabled":    true,
			"status":     status,
			"user_id":    userID,
			"logged_in":  token != "",
		},
	}
}

// handleMatrixLogin logs into Matrix
func (s *Server) handleMatrixLogin(req *Request) *Response {
	if s.matrix == nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: "Matrix adapter not configured",
			},
		}
	}

	var params struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "username and password are required",
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

	if params.Username == "" || params.Password == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "username and password are required",
			},
		}
	}

	if err := s.matrix.Login(params.Username, params.Password); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	// Start background sync
	s.matrix.StartSync()

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":  "logged_in",
			"user_id": s.matrix.GetUserID(),
		},
	}
}

// handleAttachConfig attaches a configuration file for use in containers
// This allows sending configs via Matrix that can be injected into containers
func (s *Server) handleAttachConfig(req *Request) *Response {
	// Parse parameters
	var params struct {
		Name     string `json:"name"`               // Config filename
		Content  string `json:"content"`            // File content (base64 or raw)
		Encoding string `json:"encoding,omitempty"` // "base64" or "raw" (default: raw)
		Type     string `json:"type,omitempty"`     // "env", "toml", "yaml", "json", etc.
		Metadata map[string]string `json:"metadata,omitempty"` // Additional metadata
	}

	if len(req.Params) == 0 {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "name and content are required",
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

	// Validate required parameters
	if params.Name == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "name is required",
			},
		}
	}

	if params.Content == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "content is required",
			},
		}
	}

	// Default encoding to raw if not specified
	if params.Encoding == "" {
		params.Encoding = "raw"
	}

	// Validate config name (prevent path traversal)
	cleanName := filepath.Clean(params.Name)
	if cleanName != params.Name || filepath.IsAbs(params.Name) {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: "invalid config name (path traversal not allowed)",
			},
		}
	}

	// Decode content if base64 encoded
	var contentBytes []byte
	var err error

	if params.Encoding == "base64" {
		contentBytes, err = decodeBase64(params.Content)
		if err != nil {
			return &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &ErrorObj{
					Code:    InvalidParams,
					Message: fmt.Sprintf("failed to decode base64 content: %v", err),
				},
			}
		}
	} else {
		contentBytes = []byte(params.Content)
	}

	// Validate content size (max 1MB)
	const maxConfigSize = 1 * 1024 * 1024
	if len(contentBytes) > maxConfigSize {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InvalidParams,
				Message: fmt.Sprintf("config content too large (max %d MB)", maxConfigSize/(1024*1024)),
			},
		}
	}

	// Create config directory if it doesn't exist using centralized constant
	configDir := DefaultConfigsDir
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to create config directory: %v", err),
			},
		}
	}

	// Write config file
	configPath := filepath.Join(configDir, params.Name)
	if err := os.WriteFile(configPath, contentBytes, 0640); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("failed to write config file: %v", err),
			},
		}
	}

	// Build result
	result := map[string]interface{}{
		"config_id": generateConfigID(params.Name),
		"name":      params.Name,
		"path":      configPath,
		"size":      len(contentBytes),
		"type":      params.Type,
		"encoding":  params.Encoding,
	}

	// Add metadata if provided
	if params.Metadata != nil {
		result["metadata"] = params.Metadata
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// sendError sends an error response
func (s *Server) sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObj{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	encoder.Encode(resp)
}

// decodeBase64 decodes a base64 string
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// generateConfigID generates a unique config ID
func generateConfigID(name string) string {
	// Simple ID generation: config-<name>-<timestamp>
	// In production, this could use UUID or hash
	return fmt.Sprintf("config-%s-%d", name, time.Now().Unix())
}

// checkContainerNameExists checks if a container with the given name already exists
func (s *Server) checkContainerNameExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.containers {
		if c.Name == name {
			return true
		}
	}
	return false
}

// rollbackContainerStart cleans up resources when container start fails
func (s *Server) rollbackContainerStart(containerName, secretsPath, socketPath string) {
	// Clean up secrets file
	if secretsPath != "" {
		if err := os.Remove(secretsPath); err != nil && !os.IsNotExist(err) {
			// Log error but don't fail the rollback
			fmt.Printf("[ArmorClaw] Failed to remove secrets file: %v\n", err)
		} else {
			// Log successful secret cleanup
			s.securityLog.LogSecretCleanup(s.ctx, containerName, "rollback",
				slog.String("reason", "container_start_failed"),
			)
		}
	}

	// Clean up socket path
	if socketPath != "" {
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			// Log error but don't fail the rollback
			fmt.Printf("[ArmorClaw] Failed to remove socket path: %v\n", err)
		}
	}

	// Log container rollback
	logger.Warn("container start rolled back",
		"container_name", containerName,
		"secrets_path", secretsPath,
	)
}
