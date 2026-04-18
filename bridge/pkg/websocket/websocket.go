// Package websocket provides a WebSocket adapter that bridges EventBus events
// to the HTTP server's existing gorilla/websocket infrastructure.
package websocket

import (
	"sync"
	"time"
)

// MessageHandler handles incoming WebSocket messages
type MessageHandler func(connID string, message []byte) error

// ConnectHandler handles new WebSocket connections
type ConnectHandler func(connID string, conn interface{}) error

// DisconnectHandler handles WebSocket disconnections
type DisconnectHandler func(connID string)

// Config holds WebSocket server configuration
type Config struct {
	Addr              string
	Path              string
	AllowedOrigins    []string
	MaxConnections    int
	InactivityTimeout time.Duration
	MessageHandler    MessageHandler
	ConnectHandler    ConnectHandler
	DisconnectHandler DisconnectHandler
}

// EventBroadcaster is implemented by the HTTP server to push raw JSON
// to all connected gorilla/websocket clients.
type EventBroadcaster interface {
	BroadcastEvent(eventType string, payload []byte)
}

// Server is a WebSocket adapter that delegates broadcasting to the
// HTTP server's gorilla/websocket implementation. It does NOT manage
// its own listener — the HTTP server owns the /ws endpoint.
type Server struct {
	config      Config
	addr        string
	broadcaster EventBroadcaster
	mu          sync.RWMutex
	started     bool
}

// NewServer creates a new WebSocket adapter.
func NewServer(cfg Config) *Server {
	return &Server{
		config: cfg,
		addr:   cfg.Addr,
	}
}

// SetBroadcaster injects the HTTP server's broadcast capability.
// Must be called before Start.
func (s *Server) SetBroadcaster(b EventBroadcaster) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcaster = b
}

// Start marks the adapter as active. Returns an error if no broadcaster
// has been injected — this matches the crash-only contract in eventbus.go:146.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broadcaster == nil {
		return errNoBroadcaster()
	}
	s.started = true
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started = false
	return nil
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Path() string {
	return s.config.Path
}

// Broadcast sends a message to all connected WebSocket clients by
// delegating to the injected EventBroadcaster (the HTTP server).
func (s *Server) Broadcast(message []byte) error {
	s.mu.RLock()
	b := s.broadcaster
	s.mu.RUnlock()

	if b == nil {
		return errNoBroadcaster()
	}

	b.BroadcastEvent("", message)
	return nil
}

// errNoBroadcaster returns the sentinel error used when no broadcaster
// is wired. Kept as a function so the crash-only log.Fatalf in
// eventbus.go:146 fires correctly.
func errNoBroadcaster() error {
	return &noBroadcasterError{}
}

type noBroadcasterError struct{}

func (e *noBroadcasterError) Error() string {
	return "websocket adapter: no EventBroadcaster wired — call SetBroadcaster before Start"
}
