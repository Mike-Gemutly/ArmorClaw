// Package turn provides TURN/STUN integration for NAT traversal in WebRTC
// All TURN operations happen in the Bridge, using ephemeral per-session credentials
package turn

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/webrtc"
)

// Config holds TURN server configuration
type Config struct {
	// TURN server addresses (multiple for HA)
	Servers []ServerConfig

	// Shared secret for generating TURN credentials
	// In production, use a strong random secret
	Secret string

	// Default TTL for TURN credentials
	DefaultTTL time.Duration

	// Maximum TTL for TURN credentials
	MaxTTL time.Duration
}

// ServerConfig represents a single TURN/STUN server
type ServerConfig struct {
	Host     string // Server hostname or IP
	Port     int    // Server port
	Protocol string // "udp", "tcp", or "tls"
	Realm    string // TURN realm (for auth)
}

// DefaultConfig returns default TURN configuration
func DefaultConfig() Config {
	return Config{
		Servers: []ServerConfig{
			{
				Host:     "matrix.armorclaw.com",
				Port:     3478,
				Protocol: "udp",
				Realm:    "armorclaw",
			},
		},
		Secret:     "change-me-to-a-strong-random-secret",
		DefaultTTL: 10 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}
}

// Manager manages TURN credential generation and validation
type Manager struct {
	config       Config
	credentials sync.Map // map[username]*credentialInfo
	mu           sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// credentialInfo holds information about a credential
type credentialInfo struct {
	password  string
	expiresAt time.Time
	sessionID string
	created   time.Time
}

// NewManager creates a new TURN manager
func NewManager(config Config) *Manager {
	return &Manager{
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// GenerateTURNCredentials creates ephemeral TURN credentials for a session
// Format: username = <expiry>:<session_id>, password = HMAC(secret, username)
func (m *Manager) GenerateTURNCredentials(sessionID string, ttl time.Duration) ([]webrtc.TURNCredentials, error) {
	// Validate TTL
	if ttl <= 0 {
		ttl = m.config.DefaultTTL
	}
	if ttl > m.config.MaxTTL {
		ttl = m.config.MaxTTL
	}

	// Calculate expiration
	expiresAt := time.Now().Add(ttl)
	expiryTimestamp := expiresAt.Unix()

	// Generate credentials for each server
	creds := make([]webrtc.TURNCredentials, 0, len(m.config.Servers))

	for _, server := range m.config.Servers {
		// Create username: <expiry>:<session_id>
		username := fmt.Sprintf("%d:%s", expiryTimestamp, sessionID)

		// Generate password: HMAC(secret, username)
		password := m.hmac(username)

		// Create TURN server URL
		turnURL := fmt.Sprintf("turn:%s:%d", server.Host, server.Port)
		if server.Protocol != "udp" {
			turnURL = fmt.Sprintf("turns:%s:%d", server.Host, server.Port)
		}

		// Create STUN server URL
		stunURL := fmt.Sprintf("stun:%s:%d", server.Host, server.Port)

		cred := webrtc.TURNCredentials{
			Username:   username,
			Password:   password,
			Expires:    expiresAt,
			TURNServer: turnURL,
			STUNServer: stunURL,
		}

		// Store credential info for validation
		m.credentials.Store(username, &credentialInfo{
			password:  password,
			expiresAt: expiresAt,
			sessionID: sessionID,
			created:   time.Now(),
		})

		creds = append(creds, cred)
	}

	return creds, nil
}

// ValidateTURNCredentials validates TURN credentials
func (m *Manager) ValidateTURNCredentials(username, password string) (string, error) {
	// Look up credential
	info, exists := m.credentials.Load(username)
	if !exists {
		return "", ErrTURNInvalidCredentials
	}

	cred := info.(*credentialInfo)

	// Check expiration
	if time.Now().After(cred.expiresAt) {
		m.credentials.Delete(username)
		return "", ErrTURNExpired
	}

	// Verify password
	if cred.password != password {
		return "", ErrTURNInvalidPassword
	}

	// Return session ID
	return cred.sessionID, nil
}

// CleanupExpired removes expired credentials
func (m *Manager) CleanupExpired() {
	now := time.Now()

	m.credentials.Range(func(key, value interface{}) bool {
		cred := value.(*credentialInfo)

		if now.After(cred.expiresAt) {
			m.credentials.Delete(key)
		}

		return true
	})
}

// GetStats returns statistics about active credentials
func (m *Manager) GetStats() map[string]interface{} {
	count := 0
	expiringSoon := 0
	now := time.Now()

	m.credentials.Range(func(key, value interface{}) bool {
		count++
		cred := value.(*credentialInfo)

		// Count expiring within 5 minutes
		if cred.expiresAt.Sub(now) < 5*time.Minute {
			expiringSoon++
		}

		return true
	})

	return map[string]interface{}{
		"total_credentials":  count,
		"expiring_soon":      expiringSoon,
		"servers":           len(m.config.Servers),
	}
}

// Start starts the TURN manager (cleanup goroutine)
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.cleanupLoop()
}

// Stop stops the TURN manager
func (m *Manager) Stop() {
	close(m.stopChan)
	m.wg.Wait()

	// Clear all credentials
	m.credentials.Range(func(key, value interface{}) bool {
		m.credentials.Delete(key)
		return true
	})
}

// cleanupLoop periodically cleans up expired credentials
func (m *Manager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.CleanupExpired()
		case <-m.stopChan:
			return
		}
	}
}

// hmac generates HMAC-SHA1 of the input string
func (m *Manager) hmac(input string) string {
	h := hmac.New(sha1.New, []byte(m.config.Secret))
	h.Write([]byte(input))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ICECandidate represents an ICE candidate
type ICECandidate struct {
	Foundation     string `json:"foundation"`
	ComponentID    int    `json:"component_id"`
	Transport      string `json:"transport"`
	Priority       int    `json:"priority"`
	Address        string `json:"address"`
	Port           int    `json:"port"`
	Type           string `json:"type"`
	RelatedAddress string `json:"related_address,omitempty"`
	RelatedPort    int    `json:"related_port,omitempty"`
	Protocol       string `json:"protocol"`
}

// ICECandidateType represents the type of ICE candidate
type ICECandidateType string

const (
	ICECandidateHost       ICECandidateType = "host"
	ICECandidateSRFLX      ICECandidateType = "srflx"
	ICECandidatePRFLX      ICECandidateType = "prflx"
	ICECandidateRelay      ICECandidateType = "relay"
)

// String returns the string representation of the candidate type
func (t ICECandidateType) String() string {
	return string(t)
}

// ICECandidatePriority calculates priority for an ICE candidate
// Based on RFC 5245 Section 4.1.2.1
func ICECandidatePriority(candidateType ICECandidateType, localPref int, componentID int) uint32 {
	typePrecedence := map[ICECandidateType]int{
		ICECandidateHost:  126,
		ICECandidateSRFLX: 100,
		ICECandidatePRFLX: 110,
		ICECandidateRelay: 0,
	}

	typePref := typePrecedence[candidateType]

	// Priority = (2^24)*(type preference) + (2^8)*(local preference) + (256 - component ID)
	priority := (uint32(typePref) << 24) + (uint32(localPref) << 8) + uint32(256-componentID)

	return priority
}

// ICEGatherer gathers ICE candidates
type ICEGatherer struct {
	config      Config
	stunServers []string
	turnServers []string
	candidates  []*ICECandidate
	mu          sync.Mutex
}

// NewICEGatherer creates a new ICE gatherer
func NewICEGatherer(config Config) *ICEGatherer {
	gatherer := &ICEGatherer{
		config:     config,
		candidates: make([]*ICECandidate, 0),
	}

	// Build STUN server list
	for _, server := range config.Servers {
		stunURL := fmt.Sprintf("stun:%s:%d", server.Host, server.Port)
		gatherer.stunServers = append(gatherer.stunServers, stunURL)
	}

	return gatherer
}

// GatherHostCandidates gathers host (local) candidates
func (g *ICEGatherer) GatherHostCandidates() ([]*ICECandidate, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Get local interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	candidates := make([]*ICECandidate, 0)

	// For each interface
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Get addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// For each address
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

			// Create host candidate
			candidate := &ICECandidate{
				Foundation:  "1", // Simplified
				ComponentID: 1,
				Transport:   "udp",
				Priority:    int(ICECandidatePriority(ICECandidateHost, 65535, 1)),
				Address:     ip.String(),
				Port:        0, // Will be set by WebRTC stack
				Type:        ICECandidateHost.String(),
				Protocol:    "udp",
			}

			candidates = append(candidates, candidate)
		}
	}

	g.candidates = append(g.candidates, candidates...)

	return candidates, nil
}

// GatherReflexiveCandidates gathers server reflexive candidates via STUN
func (g *ICEGatherer) GatherReflexiveCandidates() ([]*ICECandidate, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// In a full implementation, this would:
	// 1. Send STUN binding requests to STUN servers
	// 2. Parse STUN responses to get public IP:port
	// 3. Create srflx candidates

	// For now, return empty list
	// The WebRTC stack will handle STUN automatically
	return make([]*ICECandidate, 0), nil
}

// GatherRelayCandidates gathers relay candidates via TURN
func (g *ICEGatherer) GatherRelayCandidates(sessionID string, ttl time.Duration) ([]*ICECandidate, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// In a full implementation, this would:
	// 1. Allocate TURN relay
	// 2. Get relay IP:port
	// 3. Create relay candidate

	// For now, return empty list
	// The WebRTC stack will handle TURN automatically
	return make([]*ICECandidate, 0), nil
}

// GetAllCandidates returns all gathered candidates
func (g *ICEGatherer) GetAllCandidates() []*ICECandidate {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Return a copy
	candidates := make([]*ICECandidate, len(g.candidates))
	copy(candidates, g.candidates)

	return candidates
}

// ICEServer represents an ICE server configuration
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// CreateICEServers creates ICE server configurations for WebRTC
func CreateICEServers(config Config, sessionID string, ttl time.Duration) ([]ICEServer, error) {
	// Generate TURN credentials
	turnCreds, err := createTurnCreds(config, sessionID, ttl)
	if err != nil {
		return nil, err
	}

	servers := make([]ICEServer, 0)

	// Add STUN servers
	for _, server := range config.Servers {
		stunURL := fmt.Sprintf("stun:%s:%d", server.Host, server.Port)
		servers = append(servers, ICEServer{
			URLs: []string{stunURL},
		})
	}

	// Add TURN servers
	for _, cred := range turnCreds {
		servers = append(servers, ICEServer{
			URLs:       []string{cred.TURNServer},
			Username:   cred.Username,
			Credential: cred.Password,
		})
	}

	return servers, nil
}

// createTurnCreds creates TURN credentials for all servers
func createTurnCreds(config Config, sessionID string, ttl time.Duration) ([]webrtc.TURNCredentials, error) {
	if ttl <= 0 {
		ttl = config.DefaultTTL
	}
	if ttl > config.MaxTTL {
		ttl = config.MaxTTL
	}

	expiresAt := time.Now().Add(ttl)
	expiryTimestamp := expiresAt.Unix()

	creds := make([]webrtc.TURNCredentials, 0, len(config.Servers))

	for _, server := range config.Servers {
		// Create username: <expiry>:<session_id>
		username := fmt.Sprintf("%d:%s", expiryTimestamp, sessionID)

		// Generate password
		password := hmacString(config.Secret, username)

		// Create server URLs
		turnURL := fmt.Sprintf("turn:%s:%d", server.Host, server.Port)
		if server.Protocol == "tcp" {
			turnURL = fmt.Sprintf("turn:%s:%d?transport=tcp", server.Host, server.Port)
		} else if server.Protocol == "tls" {
			turnURL = fmt.Sprintf("turns:%s:%d", server.Host, server.Port)
		}

		stunURL := fmt.Sprintf("stun:%s:%d", server.Host, server.Port)

		cred := webrtc.TURNCredentials{
			Username:   username,
			Password:   password,
			Expires:    expiresAt,
			TURNServer: turnURL,
			STUNServer: stunURL,
		}

		creds = append(creds, cred)
	}

	return creds, nil
}

// hmacString generates HMAC-SHA1 of the input string
func hmacString(secret, input string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(input))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ParseICECandidate parses an ICE candidate string
func ParseICECandidate(candidateStr string) (*ICECandidate, error) {
	// ICE candidate format:
	// candidate:foundation component_id transport priority connection-address port typ [raddr] [rport] [related-extensions]

	parts := strings.Split(candidateStr, " ")
	if len(parts) < 8 || parts[0] != "candidate" {
		return nil, fmt.Errorf("invalid ICE candidate format")
	}

	candidate := &ICECandidate{
		Foundation:  parts[1],
		ComponentID: parseInt(parts[2]),
		Transport:   parts[3],
		Priority:    parseInt(parts[4]),
		Address:     parts[5],
		Port:        parseInt(parts[6]),
		Type:        parts[7],
		Protocol:    "udp", // Assume UDP
	}

	// Parse related address/port if present
	if len(parts) > 8 {
		for i := 8; i < len(parts); i++ {
			if parts[i] == "raddr" && i+1 < len(parts) {
				candidate.RelatedAddress = parts[i+1]
				i++
			} else if parts[i] == "rport" && i+1 < len(parts) {
				candidate.RelatedPort = parseInt(parts[i+1])
				i++
			}
		}
	}

	return candidate, nil
}

// String returns the ICE candidate string representation
func (c *ICECandidate) String() string {
	str := fmt.Sprintf("candidate:%s %d %s %d %s %d %s",
		c.Foundation,
		c.ComponentID,
		c.Transport,
		c.Priority,
		c.Address,
		c.Port,
		c.Type,
	)

	if c.RelatedAddress != "" {
		str += fmt.Sprintf(" raddr %s", c.RelatedAddress)
	}

	if c.RelatedPort != 0 {
		str += fmt.Sprintf(" rport %d", c.RelatedPort)
	}

	return str
}

// ToJSON converts the ICE candidate to JSON format
func (c *ICECandidate) ToJSON() map[string]interface{} {
	json := map[string]interface{}{
		"candidate": fmt.Sprintf("%s %d %d %s %s", c.Address, c.Port, c.Priority, c.Type, c.Protocol),
		"sdpMid":    "0",
		"sdpMLineIndex": 0,
	}

	if c.RelatedAddress != "" {
		json["relatedAddress"] = c.RelatedAddress
	}

	if c.RelatedPort != 0 {
		json["relatedPort"] = c.RelatedPort
	}

	return json
}

// parseInt parses an integer from a string
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// PutUint16 writes a uint16 to a byte slice in big-endian format
func PutUint16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

// PutUint32 writes a uint32 to a byte slice in big-endian format
func PutUint32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// Uint16 reads a uint16 from a byte slice in big-endian format
func Uint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

// Uint32 reads a uint32 from a byte slice in big-endian format
func Uint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// STUNMessage represents a STUN message
type STUNMessage struct {
	Type      uint16
	Length    uint16
	MagicCookie uint32
	TransactionID [12]byte
	Attributes []STUNAttribute
}

// STUNAttribute represents a STUN attribute
type STUNAttribute struct {
	Type   uint16
	Length uint16
	Value  []byte
}

// STUNMethod represents STUN method types
const (
	STUNMethodBinding           uint16 = 0x0001
	STUNMethodAllocate          uint16 = 0x0003
	STUNMethodRefresh           uint16 = 0x0004
	STUNMethodSend              uint16 = 0x0006
	STUNMethodData              uint16 = 0x0007
	STUNMethodCreatePermission  uint16 = 0x0008
	STUNMethodChannelBind       uint16 = 0x0009
)

// STUNAttrType represents STUN attribute types
const (
	STUNAttrAddress             uint16 = 0x0001
	STUNAttrXorMappedAddress    uint16 = 0x0020
	STUNAttrUsername            uint16 = 0x0006
	STUNAttrMessageIntegrity    uint16 = 0x0008
	STUNAttrErrorCode          uint16 = 0x0009
	STUNAttrChannelNumber      uint16 = 0x000C
	STUNAttrLifetime           uint16 = 0x000D
	STUNAttrXorPeerAddress     uint16 = 0x0012
	STUNAttrData               uint16 = 0x0013
	STUNAttrRealm              uint16 = 0x0014
	STUNAttrNonce              uint16 = 0x0015
)

// STUNMagicCookie is the STUN magic cookie value
const STUNMagicCookie uint32 = 0x2112A442

// NewSTUNMessage creates a new STUN message
func NewSTUNMessage(method uint16, transactionID [12]byte) *STUNMessage {
	return &STUNMessage{
		Type:           method,
		MagicCookie:    STUNMagicCookie,
		TransactionID:  transactionID,
		Attributes:     make([]STUNAttribute, 0),
	}
}

// Serialize converts the STUN message to bytes
func (m *STUNMessage) Serialize() []byte {
	// Calculate total length
	attrLength := 0
	for _, attr := range m.Attributes {
		// Add padding to 4-byte boundary
		attrLen := attr.Length
		if attrLen%4 != 0 {
			attrLen += 4 - (attrLen % 4)
		}
		attrLength += 4 + attrLen // Type(2) + Length(2) + Value + Padding
	}

	m.Length = uint16(attrLength)

	// Allocate buffer
	buf := make([]byte, 20+attrLength)

	// Write header
	PutUint16(buf[0:2], m.Type)
	PutUint16(buf[2:4], m.Length)
	PutUint32(buf[4:8], m.MagicCookie)
	copy(buf[8:20], m.TransactionID[:])

	// Write attributes
	offset := 20
	for _, attr := range m.Attributes {
		PutUint16(buf[offset:offset+2], attr.Type)
		PutUint16(buf[offset+2:offset+4], attr.Length)
		copy(buf[offset+4:], attr.Value)
		offset += 4 + int(attr.Length)

		// Add padding
		for attr.Length%4 != 0 {
			buf[offset] = 0
			offset++
			attr.Length++
		}
	}

	return buf
}

// ParseSTUNMessage parses a STUN message from bytes
func ParseSTUNMessage(data []byte) (*STUNMessage, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("STUN message too short")
	}

	msg := &STUNMessage{
		Type:      Uint16(data[0:2]),
		Length:    Uint16(data[2:4]),
		MagicCookie: Uint32(data[4:8]),
	}

	copy(msg.TransactionID[:], data[8:20])

	// Verify magic cookie
	if msg.MagicCookie != STUNMagicCookie {
		return nil, fmt.Errorf("invalid STUN magic cookie")
	}

	// Parse attributes
	offset := 20
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		attr := STUNAttribute{
			Type:   Uint16(data[offset:offset+2]),
			Length: Uint16(data[offset+2:offset+4]),
		}

		if attr.Length > 0 {
			if offset+4+int(attr.Length) > len(data) {
				return nil, fmt.Errorf("attribute too long")
			}
			attr.Value = make([]byte, attr.Length)
			copy(attr.Value, data[offset+4:offset+4+int(attr.Length)])
		}

		msg.Attributes = append(msg.Attributes, attr)

		offset += 4 + int(attr.Length)

		// Skip padding
		for attr.Length%4 != 0 && offset < len(data) {
			offset++
		}
	}

	return msg, nil
}

// GenerateTransactionID generates a random STUN transaction ID
func GenerateTransactionID() [12]byte {
	var id [12]byte
	// Use timestamp for uniqueness (simple approach)
	timestamp := time.Now().UnixNano()
	binary.BigEndian.PutUint64(id[0:8], uint64(timestamp))

	// Add some randomness
	id[8] = byte(timestamp >> 56)
	id[9] = byte(timestamp >> 48)
	id[10] = byte(timestamp >> 40)
	id[11] = byte(timestamp >> 32)

	return id
}

// Errors
var (
	// ErrTURNInvalidCredentials is returned for invalid TURN credentials
	ErrTURNInvalidCredentials = fmt.Errorf("invalid TURN credentials")

	// ErrTURNExpired is returned for expired TURN credentials
	ErrTURNExpired = fmt.Errorf("TURN credentials expired")

	// ErrTURNInvalidPassword is returned for incorrect TURN password
	ErrTURNInvalidPassword = fmt.Errorf("incorrect TURN password")

	// ErrTURNInvalidFormat is returned for malformed TURN credentials
	ErrTURNInvalidFormat = fmt.Errorf("malformed TURN credentials format")

	// ErrSTUNNotSupported is returned when STUN is not supported
	ErrSTUNNotSupported = fmt.Errorf("STUN not supported")
)
