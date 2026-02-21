// Package discovery provides mDNS/Bonjour service discovery for ArmorClaw bridge
package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

const (
	// ServiceName is the mDNS service type for ArmorClaw bridge
	// Note: Trailing dot is required for FQDN format in mDNS
	ServiceName = "_armorclaw._tcp."

	// ServiceDomain is the mDNS domain
	ServiceDomain = "local."

	// DefaultPort is the default HTTP API port
	DefaultPort = 8080

	// DefaultTLSPort is the default HTTPS API port
	DefaultTLSPort = 8443

	// DiscoveryTimeout is how long to wait for discovery responses
	DiscoveryTimeout = 5 * time.Second
)

// BridgeInfo contains discovered bridge information
type BridgeInfo struct {
	Name             string            `json:"name"`
	Host             string            `json:"host"`
	Port             int               `json:"port"`
	IPs              []net.IP          `json:"ips"`
	TXT              map[string]string `json:"txt"`
	Version          string            `json:"version"`
	Mode             string            `json:"mode"`
	Hardware         string            `json:"hardware,omitempty"`
	MatrixHomeserver string            `json:"matrix_homeserver,omitempty"`
	PushGateway      string            `json:"push_gateway,omitempty"`
	APIPath          string            `json:"api_path,omitempty"`
	WSPath           string            `json:"ws_path,omitempty"`
	TLS              bool              `json:"tls"`
}

// Server represents an mDNS server that advertises the bridge
type Server struct {
	mu       sync.RWMutex
	server   *mdns.Server
	info     *BridgeInfo
	running  bool
	shutdown context.CancelFunc
}

// ServerConfig contains configuration for the mDNS server
type ServerConfig struct {
	// InstanceName is the service instance name (defaults to hostname)
	InstanceName string
	// Port is the HTTP/HTTPS port
	Port int
	// TLS indicates whether HTTPS is enabled
	TLS bool
	// MatrixHomeserver is the Matrix homeserver URL (e.g., https://matrix.example.com)
	MatrixHomeserver string
	// PushGateway is the push gateway URL
	PushGateway string
	// APIPath is the API path (default: /api)
	APIPath string
	// WSPath is the WebSocket path (default: /ws)
	WSPath string
	// ExtraTXT contains additional TXT records
	ExtraTXT map[string]string
}

// NewServer creates a new mDNS advertisement server
// Deprecated: Use NewServerWithConfig for full configuration
func NewServer(instanceName string, port int, extraTXT map[string]string) (*Server, error) {
	return NewServerWithConfig(ServerConfig{
		InstanceName: instanceName,
		Port:         port,
		ExtraTXT:     extraTXT,
	})
}

// NewServerWithConfig creates a new mDNS advertisement server with full configuration
func NewServerWithConfig(config ServerConfig) (*Server, error) {
	port := config.Port
	if port == 0 {
		if config.TLS {
			port = DefaultTLSPort
		} else {
			port = DefaultPort
		}
	}

	// Get hostname if not specified
	instanceName := config.InstanceName
	if instanceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "armorclaw"
		}
		instanceName = hostname
	}

	// Get local IPs
	ips, err := getLocalIPs()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IPs: %w", err)
	}

	// Set defaults
	apiPath := config.APIPath
	if apiPath == "" {
		apiPath = "/api"
	}
	wsPath := config.WSPath
	if wsPath == "" {
		wsPath = "/ws"
	}

	// Build TXT records
	txt := []string{
		fmt.Sprintf("version=0.2.0"),
		fmt.Sprintf("mode=operational"),
		fmt.Sprintf("tls=%v", config.TLS),
		fmt.Sprintf("api_path=%s", apiPath),
		fmt.Sprintf("ws_path=%s", wsPath),
	}
	if config.MatrixHomeserver != "" {
		txt = append(txt, fmt.Sprintf("matrix_homeserver=%s", config.MatrixHomeserver))
	}
	if config.PushGateway != "" {
		txt = append(txt, fmt.Sprintf("push_gateway=%s", config.PushGateway))
	}
	for k, v := range config.ExtraTXT {
		txt = append(txt, fmt.Sprintf("%s=%s", k, v))
	}

	// Create mDNS service
	service, err := mdns.NewMDNSService(
		instanceName,
		ServiceName,
		ServiceDomain,
		"",
		port,
		ips,
		txt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Create server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS server: %w", err)
	}

	info := &BridgeInfo{
		Name:             instanceName,
		Port:             port,
		IPs:              ips,
		TXT:              config.ExtraTXT,
		TLS:              config.TLS,
		APIPath:          apiPath,
		WSPath:           wsPath,
		MatrixHomeserver: config.MatrixHomeserver,
		PushGateway:      config.PushGateway,
	}
	for _, t := range txt {
		parts := strings.SplitN(t, "=", 2)
		if len(parts) == 2 {
			info.TXT[parts[0]] = parts[1]
			switch parts[0] {
			case "version":
				info.Version = parts[1]
			case "mode":
				info.Mode = parts[1]
			case "hardware":
				info.Hardware = parts[1]
			case "matrix_homeserver":
				info.MatrixHomeserver = parts[1]
			case "push_gateway":
				info.PushGateway = parts[1]
			case "api_path":
				info.APIPath = parts[1]
			case "ws_path":
				info.WSPath = parts[1]
			case "tls":
				info.TLS = parts[1] == "true"
			}
		}
	}

	return &Server{
		server:  server,
		info:    info,
		running: false,
	}, nil
}

// Start begins advertising the bridge service
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// The mdns.Server starts advertising immediately on creation
	// We just need to track the running state
	ctx, cancel := context.WithCancel(ctx)
	s.shutdown = cancel
	s.running = true

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	return nil
}

// Stop stops advertising the bridge service
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.shutdown != nil {
		s.shutdown()
	}

	if s.server != nil {
		return s.server.Shutdown()
	}

	s.running = false
	return nil
}

// Info returns the current bridge info being advertised
func (s *Server) Info() *BridgeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info
}

// UpdateTXT updates the TXT records being advertised
func (s *Server) UpdateTXT(txt map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update stored info
	for k, v := range txt {
		s.info.TXT[k] = v
		switch k {
		case "version":
			s.info.Version = v
		case "mode":
			s.info.Mode = v
		case "hardware":
			s.info.Hardware = v
		case "matrix_homeserver":
			s.info.MatrixHomeserver = v
		case "push_gateway":
			s.info.PushGateway = v
		case "api_path":
			s.info.APIPath = v
		case "ws_path":
			s.info.WSPath = v
		case "tls":
			s.info.TLS = v == "true"
		}
	}

	// Note: mDNS doesn't support dynamic TXT updates easily
	// For a production system, you'd need to restart the service
	// or use a more advanced mDNS library

	return nil
}

// Client discovers ArmorClaw bridges on the network
type Client struct {
	timeout time.Duration
}

// NewClient creates a new discovery client
func NewClient() *Client {
	return &Client{
		timeout: DiscoveryTimeout,
	}
}

// SetTimeout sets the discovery timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Discover finds all ArmorClaw bridges on the local network
func (c *Client) Discover(ctx context.Context) ([]BridgeInfo, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	var bridges []BridgeInfo
	var mu sync.Mutex

	go func() {
		for entry := range entriesCh {
			if strings.HasPrefix(entry.Name, ServiceName) ||
				strings.Contains(entry.Name, "armorclaw") {
				info := parseEntry(entry)
				mu.Lock()
				bridges = append(bridges, info)
				mu.Unlock()
			}
		}
	}()

	// Run discovery
	params := &mdns.QueryParam{
		Service:             ServiceName,
		Domain:              ServiceDomain,
		Timeout:             c.timeout,
		Entries:             entriesCh,
		WantUnicastResponse: false,
		DisableIPv4:         false,
		DisableIPv6:         false,
	}

	if err := mdns.Query(params); err != nil {
		close(entriesCh)
		return nil, fmt.Errorf("mDNS query failed: %w", err)
	}

	close(entriesCh)

	// Wait for responses
	select {
	case <-time.After(c.timeout):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if len(bridges) == 0 {
		return nil, fmt.Errorf("no ArmorClaw bridges found")
	}

	return bridges, nil
}

// DiscoverOne finds a single bridge (the first one discovered)
func (c *Client) DiscoverOne(ctx context.Context) (*BridgeInfo, error) {
	bridges, err := c.Discover(ctx)
	if err != nil {
		return nil, err
	}

	return &bridges[0], nil
}

// parseEntry converts an mDNS entry to BridgeInfo
func parseEntry(entry *mdns.ServiceEntry) BridgeInfo {
	info := BridgeInfo{
		Name: entry.Name,
		Port: entry.Port,
		IPs:  []net.IP{},
		TXT:  make(map[string]string),
		// Set defaults
		APIPath: "/api",
		WSPath:  "/ws",
		TLS:     true, // Default to TLS for security
	}

	// Add IPv4 address if present
	if entry.AddrV4 != nil {
		info.IPs = append(info.IPs, entry.AddrV4)
		info.Host = entry.AddrV4.String()
	}

	// Add IPv6 address if present
	if entry.AddrV6 != nil {
		info.IPs = append(info.IPs, entry.AddrV6)
		if info.Host == "" {
			info.Host = entry.AddrV6.String()
		}
	}

	if entry.InfoFields != nil {
		for _, field := range entry.InfoFields {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) == 2 {
				info.TXT[parts[0]] = parts[1]
				switch parts[0] {
				case "version":
					info.Version = parts[1]
				case "mode":
					info.Mode = parts[1]
				case "hardware":
					info.Hardware = parts[1]
				case "matrix_homeserver":
					info.MatrixHomeserver = parts[1]
				case "push_gateway":
					info.PushGateway = parts[1]
				case "api_path":
					info.APIPath = parts[1]
				case "ws_path":
					info.WSPath = parts[1]
				case "tls":
					info.TLS = parts[1] == "true"
				}
			}
		}
	}

	return info
}

// getLocalIPs returns all local IP addresses
func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces that are down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
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

			// Skip loopback and link-local
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no suitable IP addresses found")
	}

	return ips, nil
}

// ManualConnection represents a manually specified bridge connection
type ManualConnection struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Validate checks if the manual connection is valid
func (m *ManualConnection) Validate() error {
	if m.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Validate IP or hostname
	if ip := net.ParseIP(m.Host); ip == nil {
		// Not an IP, validate hostname
		if len(m.Host) > 253 {
			return fmt.Errorf("hostname too long")
		}
		// Basic hostname validation
		for _, part := range strings.Split(m.Host, ".") {
			if len(part) == 0 || len(part) > 63 {
				return fmt.Errorf("invalid hostname segment")
			}
		}
	}

	if m.Port <= 0 || m.Port > 65535 {
		m.Port = DefaultPort
	}

	return nil
}

// ToBridgeInfo converts a manual connection to BridgeInfo
func (m *ManualConnection) ToBridgeInfo() *BridgeInfo {
	return &BridgeInfo{
		Name: m.Host,
		Host: m.Host,
		Port: m.Port,
		TXT:  map[string]string{"manual": "true"},
	}
}
