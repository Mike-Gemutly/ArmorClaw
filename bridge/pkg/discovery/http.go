// Package discovery provides HTTP discovery server for ArmorClaw bridge
// This server provides REST API endpoints for ArmorChat discovery
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/rpc"
)

// HTTPServer provides HTTP endpoints for bridge discovery
type HTTPServer struct {
	mu       sync.RWMutex
	server   *http.Server
	info     *BridgeInfo
	metrics  *rpc.Metrics
	running  bool
	shutdown context.CancelFunc
}

// HTTPServerConfig configures the HTTP discovery server
type HTTPServerConfig struct {
	Port             int
	TLS              bool
	InstanceName     string
	MatrixHomeserver string
	PushGateway      string
	APIPath          string
	WSPath           string
	Metrics          *rpc.Metrics
}

// NewHTTPServer creates a new HTTP discovery server
func NewHTTPServer(config HTTPServerConfig) (*HTTPServer, error) {
	port := config.Port
	if port == 0 {
		if config.TLS {
			port = DefaultTLSPort
		} else {
			port = DefaultPort
		}
	}

	instanceName := config.InstanceName
	if instanceName == "" {
		instanceName = "armorclaw"
	}

	apiPath := config.APIPath
	if apiPath == "" {
		apiPath = "/api"
	}

	wsPath := config.WSPath
	if wsPath == "" {
		wsPath = "/ws"
	}

	info := &BridgeInfo{
		Name:             instanceName,
		Port:             port,
		TLS:              config.TLS,
		APIPath:          apiPath,
		WSPath:           wsPath,
		MatrixHomeserver: config.MatrixHomeserver,
		PushGateway:      config.PushGateway,
		Version:          "0.2.0",
		Mode:             "operational",
		TXT:              make(map[string]string),
	}

	return &HTTPServer{
		info:    info,
		metrics: config.Metrics,
	}, nil
}

// Start begins listening for HTTP requests
func (s *HTTPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	s.shutdown = cancel

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/discovery", s.handleDiscovery)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/metrics", s.handleMetrics)

	addr := fmt.Sprintf("0.0.0.0:%d", s.info.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP discovery server starting", "address", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	s.running = true
	slog.Info("HTTP discovery server started", "port", s.info.Port)

	go func() {
		<-ctx.Done()
		s.server.Shutdown(context.Background())
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		slog.Info("HTTP discovery server stopped")
	}()

	return nil
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.shutdown != nil {
		s.shutdown()
	}

	s.running = false
	return nil
}

// Info returns the current bridge info
func (s *HTTPServer) Info() *BridgeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info
}

// handleHealth handles /health endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "armorclaw-discovery",
	})
}

// handleDiscovery handles /api/discovery endpoint
func (s *HTTPServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	info := s.info
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	protocol := "http"
	if info.TLS {
		protocol = "https"
	}

	hostname := info.Name
	if info.Host != "" {
		hostname = info.Host
	}

	response := map[string]interface{}{
		"version":                info.Version,
		"mode":                   info.Mode,
		"service_name":           info.Name,
		"port":                   info.Port,
		"tls":                    info.TLS,
		"api_url":                fmt.Sprintf("%s://%s:%d%s", protocol, hostname, info.Port, info.APIPath),
		"ws_url":                 fmt.Sprintf("%ss://%s:%d%s", map[bool]string{true: "wss", false: "ws"}[info.TLS], hostname, info.Port, info.WSPath),
		"matrix_homeserver":      info.MatrixHomeserver,
		"push_gateway":           info.PushGateway,
		"txt":                    info.TXT,
		"provisioning_available": info.ProvisioningReady,
		"server_name":            info.Name,
	}

	if info.PublicBaseURL != "" {
		response["public_base_url"] = info.PublicBaseURL
	}

	json.NewEncoder(w).Encode(response)
}

// handleStatus handles /api/status endpoint
func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "running",
		"timestamp": time.Now().Unix(),
		"service":   "armorclaw-discovery",
	})
}

func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	if s.metrics != nil {
		s.metrics.UpdateUptime()
		w.Write([]byte(s.metrics.Export()))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("metrics not initialized\n"))
	}
}
