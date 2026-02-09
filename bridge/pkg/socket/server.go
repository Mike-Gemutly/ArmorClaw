// Package socket provides Unix domain socket utilities for ArmorClaw bridge.
// The socket enables secure communication between the hardened container and the host.
package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
	"golang.org/x/time/rate"
)

const (
	// DefaultSocketPath is the default Unix socket path
	DefaultSocketPath = "/run/armorclaw/bridge.sock"

	// Connection limits
	DefaultMaxConnections = 100
	DefaultConnectionTimeout = 5 * time.Minute

	// Rate limiting (events per second)
	DefaultRateLimit = 10.0
	DefaultRateBurst = 10
)

var (
	ErrServerClosed    = errors.New("server closed")
	ErrInvalidMessage  = errors.New("invalid message format")
	ErrUnauthorized     = errors.New("unauthorized access")
	ErrContainerNotFound = errors.New("container not found")
)

// MessageType identifies the type of JSON-RPC message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeNotification MessageType = "notification"
)

// Message represents a JSON-RPC 2.0 message
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
	Type    MessageType     `json:"type,omitempty"`
}

// RPCError represents a JSON-RPC error object
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorCode constants
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeUnauthorized    = -32000
	CodeContainerNotFound = -32001
)

// Server handles Unix socket connections
type Server struct {
	socketPath string
	listener   net.Listener
	keystore   *keystore.Keystore
	containers map[string]*ContainerSession
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Connection management
	rateLimiter        *rate.Limiter
	maxConnections     int
	activeConnections  int
	connectionTimeout  time.Duration
}

// ContainerSession represents an active container connection
type ContainerSession struct {
	ID       string
	Pid      int
	Endpoint string
	Provider string
	Created  int64
}

// Handler is called for each received message
type Handler func(msg *Message) (*Message, error)

// Config holds server configuration
type Config struct {
	SocketPath string
	Keystore   *keystore.Keystore
}

// New creates a new Unix socket server
func New(cfg Config) (*Server, error) {
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath
	}

	if cfg.Keystore == nil {
		return nil, errors.New("keystore is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		socketPath:        cfg.SocketPath,
		keystore:          cfg.Keystore,
		containers:        make(map[string]*ContainerSession),
		ctx:               ctx,
		cancel:            cancel,
		rateLimiter:       rate.NewLimiter(rate.Limit(DefaultRateLimit), DefaultRateBurst),
		maxConnections:    DefaultMaxConnections,
		connectionTimeout: DefaultConnectionTimeout,
	}, nil
}

// Start starts the Unix socket server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure socket directory exists
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if present
	os.Remove(s.socketPath)

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions (owner + group read/write, no world access)
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the Unix socket server
func (s *Server) Stop() error {
	s.cancel()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return err
		}
	}

	s.wg.Wait()

	// Clean up socket file
	os.Remove(s.socketPath)

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

			// Check connection limit
			s.mu.Lock()
			if s.activeConnections >= s.maxConnections {
				s.mu.Unlock()
				conn.Close()
				// Log: Max connections reached
				continue
			}
			s.activeConnections++
			s.mu.Unlock()

			// Apply rate limit
			if !s.rateLimiter.Allow() {
				s.mu.Lock()
				s.activeConnections--
				s.mu.Unlock()
				conn.Close()
				// Log: Rate limit exceeded
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
	defer func() {
		conn.Close()
		s.mu.Lock()
		s.activeConnections--
		s.mu.Unlock()
	}()

	// Set connection deadline
	if err := conn.SetDeadline(time.Now().Add(s.connectionTimeout)); err != nil {
		// Log error, close connection
		return
	}

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				_ = s.sendError(encoder, nil, CodeParseError, err.Error(), nil)
				continue
			}

			// Extend deadline on activity
			if err := conn.SetDeadline(time.Now().Add(s.connectionTimeout)); err != nil {
				return
			}

			// Route message to handler
			resp := s.handleMessage(&msg)

			// Send response (unless notification)
			if msg.ID != nil || msg.Type != MessageTypeNotification {
				if err := encoder.Encode(resp); err != nil {
					return
				}
			}
		}
	}
}

// handleMessage processes a single message
func (s *Server) handleMessage(msg *Message) *Message {
	// Validate JSON-RPC version
	if msg.JSONRPC != "2.0" {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInvalidRequest,
				Message: "invalid JSON-RPC version",
			},
		}
	}

	// Route to method handler
	switch msg.Method {
	case "status":
		return s.handleStatus(msg)
	case "health":
		return s.handleHealth(msg)
	case "start":
		return s.handleStart(msg)
	case "stop":
		return s.handleStop(msg)
	case "get_credential":
		return s.handleGetCredential(msg)
	case "list_credentials":
		return s.handleListCredentials(msg)
	default:
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("method '%s' not found", msg.Method),
			},
		}
	}
}

// sendError sends an error response
func (s *Server) sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) error {
	if encoder == nil {
		return fmt.Errorf("cannot send error: encoder is nil")
	}
	resp := &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to send error response: %w", err)
	}
	return nil
}

// handleStatus returns bridge status
func (s *Server) handleStatus(msg *Message) *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"version":     "1.0.0",
			"state":       "running",
			"socket":      s.socketPath,
			"containers":  len(s.containers),
		},
	}
}

// handleHealth returns health check result
func (s *Server) handleHealth(msg *Message) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"status": "healthy",
		},
	}
}

// handleStart starts a container with credentials
func (s *Server) handleStart(msg *Message) *Message {
	// Check keystore availability
	if s.keystore == nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: "keystore not initialized",
			},
		}
	}

	// Parse parameters
	var params struct {
		KeyID     string `json:"key_id"`
		AgentType string `json:"agent_type"`
		Image     string `json:"image"`
	}

	if len(msg.Params) > 0 {
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeInvalidParams,
					Message: err.Error(),
				},
			}
		}
	}

	// Validate key_id
	if params.KeyID == "" {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInvalidParams,
				Message: "key_id is required",
			},
		}
	}

	// Retrieve credential from keystore
	cred, err := s.keystore.Retrieve(params.KeyID)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeContainerNotFound,
					Message: "key not found",
				},
			}
		}
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: err.Error(),
			},
		}
	}

	// TODO: Implement actual Docker container start
	// For now, create a session record
	containerID := fmt.Sprintf("container-%d", time.Now().Unix())

	s.mu.Lock()
	s.containers[containerID] = &ContainerSession{
		ID:       containerID,
		Provider: string(cred.Provider),
		Created:  time.Now().Unix(),
	}
	s.mu.Unlock()

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"container_id": containerID,
			"status":       "running",
			"endpoint":     fmt.Sprintf("/run/armorclaw/%s.sock", containerID),
		},
	}
}

// handleStop stops a running container
func (s *Server) handleStop(msg *Message) *Message {
	var params struct {
		ContainerID string `json:"container_id"`
	}

	if len(msg.Params) > 0 {
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeInvalidParams,
					Message: err.Error(),
				},
			}
		}
	}

	// Validate container_id
	if params.ContainerID == "" {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInvalidParams,
				Message: "container_id is required",
			},
		}
	}

	// TODO: Implement actual Docker container stop
	s.mu.Lock()
	delete(s.containers, params.ContainerID)
	s.mu.Unlock()

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"status": "stopped",
		},
	}
}

// handleGetCredential retrieves a credential (for agent use)
func (s *Server) handleGetCredential(msg *Message) *Message {
	// Check keystore availability
	if s.keystore == nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: "keystore not initialized",
			},
		}
	}

	var params struct {
		ID string `json:"id"`
	}

	if len(msg.Params) == 0 {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInvalidParams,
				Message: "id parameter required",
			},
		}
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInvalidParams,
				Message: err.Error(),
			},
		}
	}

	cred, err := s.keystore.Retrieve(params.ID)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeContainerNotFound,
					Message: "key not found",
				},
			}
		}
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: err.Error(),
			},
		}
	}

	// Return credential (in production, verify caller is authorized)
	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"provider":     cred.Provider,
			"display_name": cred.DisplayName,
			"token":        cred.Token,
		},
	}
}

// handleListCredentials lists available credentials
func (s *Server) handleListCredentials(msg *Message) *Message {
	// Check keystore availability
	if s.keystore == nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: "keystore not initialized",
			},
		}
	}

	var params struct {
		Provider string `json:"provider"`
	}

	if len(msg.Params) > 0 {
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeInvalidParams,
					Message: err.Error(),
				},
			}
		}
	}

	// Validate provider if specified
	if params.Provider != "" {
		provider := keystore.Provider(params.Provider)
		if provider != keystore.ProviderOpenAI &&
			provider != keystore.ProviderAnthropic &&
			provider != keystore.ProviderOpenRouter &&
			provider != keystore.ProviderGoogle &&
			provider != keystore.ProviderXAI {
			return &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &RPCError{
					Code:    CodeInvalidParams,
					Message: fmt.Sprintf("invalid provider: %s", params.Provider),
				},
			}
		}
	}

	keys, err := s.keystore.List(keystore.Provider(params.Provider))
	if err != nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    CodeInternalError,
				Message: err.Error(),
			},
		}
	}

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

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

// GetSocketPath returns the socket path
func (s *Server) GetSocketPath() string {
	return s.socketPath
}
