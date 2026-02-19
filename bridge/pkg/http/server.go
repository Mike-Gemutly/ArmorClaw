// Package http provides an HTTPS server for remote bridge access.
// This enables ArmorTerminal (Android) and web clients to communicate
// with the bridge over the network.
package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/rpc"
	"github.com/gorilla/websocket"
)

// ServerConfig holds configuration for the HTTPS server
type ServerConfig struct {
	Port           int
	CertFile       string
	KeyFile        string
	CertDir        string
	Hostname       string
	EnableCORS     bool
	AllowedOrigins []string
}

// Server is the HTTPS server for the bridge
type Server struct {
	config     ServerConfig
	rpcServer  *rpc.Server
	httpServer *http.Server
	certPEM    []byte
	keyPEM     []byte
	mu         sync.RWMutex
	clients    map[string]*WebSocketClient
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID       string
	DeviceID string
	Send     chan []byte
}

// NewServer creates a new HTTPS server
func NewServer(config ServerConfig, rpcServer *rpc.Server) *Server {
	if config.Port == 0 {
		config.Port = 8443
	}
	if config.CertDir == "" {
		config.CertDir = "/var/lib/armorclaw/certs"
	}
	if config.Hostname == "" {
		config.Hostname = "armorclaw.local"
	}

	return &Server{
		config:    config,
		rpcServer: rpcServer,
		clients:   make(map[string]*WebSocketClient),
	}
}

// Start starts the HTTPS server
func (s *Server) Start() error {
	if err := s.loadOrGenerateCerts(); err != nil {
		return fmt.Errorf("failed to setup certificates: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api", s.handleRPC)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/discover", s.handleDiscover)
	mux.HandleFunc("/fingerprint", s.handleFingerprint)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.corsMiddleware(mux),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
			},
			CipherSuites: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_128_GCM_SHA256,
			},
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("[HTTP] Starting HTTPS server on port %d", s.config.Port)

	err := s.httpServer.ListenAndServeTLS("", "")
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTPS server error: %w", err)
	}

	return nil
}

// Stop stops the HTTPS server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		log.Println("[HTTP] Stopping HTTPS server")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// GetCertificatePEM returns the certificate in PEM format
func (s *Server) GetCertificatePEM() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.certPEM
}

// GetCertificateFingerprint returns the SHA-256 fingerprint of the certificate
func (s *Server) GetCertificateFingerprint() (string, error) {
	block, _ := pem.Decode(s.certPEM)
	if block == nil {
		return "", fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	fingerprint := fmt.Sprintf("%x", cert.Signature)
	return fingerprint[:64], nil
}

func (s *Server) loadOrGenerateCerts() error {
	certFile := s.config.CertFile
	keyFile := s.config.KeyFile

	if certFile == "" {
		certFile = filepath.Join(s.config.CertDir, "bridge.crt")
	}
	if keyFile == "" {
		keyFile = filepath.Join(s.config.CertDir, "bridge.key")
	}

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			certPEM, err := os.ReadFile(certFile)
			if err != nil {
				return fmt.Errorf("failed to read certificate: %w", err)
			}
			keyPEM, err := os.ReadFile(keyFile)
			if err != nil {
				return fmt.Errorf("failed to read key: %w", err)
			}

			s.certPEM = certPEM
			s.keyPEM = keyPEM

			log.Printf("[HTTP] Loaded existing certificate from %s", certFile)
			return nil
		}
	}

	log.Println("[HTTP] Generating new self-signed certificate")

	certPEM, keyPEM, err := s.generateSelfSignedCert()
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	if err := os.MkdirAll(s.config.CertDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	s.certPEM = certPEM
	s.keyPEM = keyPEM

	log.Printf("[HTTP] Generated and saved certificate to %s", certFile)
	return nil
}

func (s *Server) generateSelfSignedCert() ([]byte, []byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	ips, err := getLocalIPs()
	if err != nil {
		ips = []net.IP{net.ParseIP("127.0.0.1")}
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ArmorClaw"},
			CommonName:   s.config.Hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames: []string{
			s.config.Hostname,
			"localhost",
			"*.local",
		},
		IPAddresses: ips,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return certPEM, keyPEM, nil
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, nil, -32700, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Use the RPC server to handle the request
	response := s.rpcServer.HandleRequest(r.Context(), body)

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	ips, _ := getLocalIPs()
	fingerprint, _ := s.GetCertificateFingerprint()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        hostname,
		"hostname":    s.config.Hostname,
		"port":        s.config.Port,
		"ips":         ips,
		"version":     "1.0.0",
		"fingerprint": fingerprint,
		"endpoints": map[string]string{
			"rpc":    "/api",
			"ws":     "/ws",
			"health": "/health",
		},
	})
}

func (s *Server) handleFingerprint(w http.ResponseWriter, r *http.Request) {
	fingerprint, err := s.GetCertificateFingerprint()
	if err != nil {
		http.Error(w, "Failed to get fingerprint", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sha256":  fingerprint,
		"format":  "hex",
	})
}

func (s *Server) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableCORS {
			origin := r.Header.Get("Origin")
			allowed := false

			for _, o := range s.config.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if !allowed && (origin == "http://localhost:3000" ||
				origin == "http://localhost:8080" ||
				origin == "http://127.0.0.1:3000" ||
				origin == "http://127.0.0.1:8080") {
				allowed = true
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// WebSocket handling
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket
	},
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	clientID := generateClientID()
	client := &WebSocketClient{
		ID:   clientID,
		Send: make(chan []byte, 256),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	log.Printf("[WS] Client connected: %s", clientID)

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		close(client.Send)
		log.Printf("[WS] Client disconnected: %s", clientID)
	}()

	go s.writePump(conn, client)
	s.readPump(conn, client)
}

func (s *Server) readPump(conn *websocket.Conn, client *WebSocketClient) {
	conn.SetReadLimit(512 * 1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}

		s.handleWSMessage(client, message)
	}
}

func (s *Server) writePump(conn *websocket.Conn, client *WebSocketClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleWSMessage(client *WebSocketClient, message []byte) {
	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "ping":
		s.sendToClient(client, map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})

	case "register":
		var payload struct {
			DeviceID string `json:"device_id"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			client.DeviceID = payload.DeviceID
			log.Printf("[WS] Client %s registered as device %s", client.ID, client.DeviceID)
			s.sendToClient(client, map[string]interface{}{
				"type":      "registered",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"payload":   map[string]string{"status": "ok"},
			})
		}
	}
}

func (s *Server) sendToClient(client *WebSocketClient, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}

// NotifyDeviceApproved sends approval notification to device
func (s *Server) NotifyDeviceApproved(deviceID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := map[string]interface{}{
		"type":      "device.approved",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"payload":   map[string]string{"status": "approved"},
	}

	data, _ := json.Marshal(msg)

	for _, client := range s.clients {
		if client.DeviceID == deviceID {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// NotifyDeviceRejected sends rejection notification to device
func (s *Server) NotifyDeviceRejected(deviceID string, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := map[string]interface{}{
		"type":      "device.rejected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"payload":   map[string]string{"status": "rejected", "reason": reason},
	}

	data, _ := json.Marshal(msg)

	for _, client := range s.clients {
		if client.DeviceID == deviceID {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		ips = append(ips, net.ParseIP("127.0.0.1"))
	}

	return ips, nil
}

func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
