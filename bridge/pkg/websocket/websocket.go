// Package websocket provides WebSocket server functionality for ArmorClaw.
// NOTE: This is a minimal stub implementation for compatibility.
// Full WebSocket server implementation is planned for future releases.
package websocket

import (
	"fmt"
)

// Config holds WebSocket server configuration
type Config struct {
	Addr           string
	Path           string
	AllowedOrigins []string
}

// Server represents a WebSocket server
type Server struct {
	config Config
	addr   string
}

// NewServer creates a new WebSocket server
func NewServer(cfg Config) *Server {
	return &Server{
		config: cfg,
		addr:   cfg.Addr,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// Stub implementation - WebSocket server not yet implemented
	// This is a placeholder to allow the build to succeed
	return fmt.Errorf("websocket server not yet implemented")
}

// Stop stops the WebSocket server
func (s *Server) Stop() error {
	return nil
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.addr
}

// Path returns the WebSocket path
func (s *Server) Path() string {
	return s.config.Path
}
