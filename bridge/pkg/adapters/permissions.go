// Package adapters provides management for external communication adapters
// (Slack, Discord, Teams, WhatsApp) with granular permission controls.
package adapters

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AdapterType identifies different communication platforms
type AdapterType string

const (
	AdapterMatrix   AdapterType = "matrix"
	AdapterSlack    AdapterType = "slack"
	AdapterDiscord  AdapterType = "discord"
	AdapterTeams    AdapterType = "teams"
	AdapterWhatsApp AdapterType = "whatsapp"
)

// AdapterAction defines what operations an adapter can perform
type AdapterAction string

const (
	ActionRead    AdapterAction = "read"
	ActionWrite   AdapterAction = "write"
	ActionSend    AdapterAction = "send"
	ActionReceive AdapterAction = "receive"
	ActionDelete  AdapterAction = "delete"
)

// AdapterStatus represents the current state of an adapter
type AdapterStatus string

const (
	StatusDisconnected AdapterStatus = "disconnected"
	StatusConnecting   AdapterStatus = "connecting"
	StatusConnected    AdapterStatus = "connected"
	StatusError        AdapterStatus = "error"
	StatusDisabled     AdapterStatus = "disabled"
)

// AdapterInfo provides metadata about an adapter
type AdapterInfo struct {
	Type        AdapterType `json:"type"`
	DisplayName string      `json:"display_name"`
	Description string      `json:"description"`
	Encryption  string      `json:"encryption"`
	Status      AdapterStatus `json:"status"`
	Features    []string    `json:"features"`
}

// AllAdapters returns information about all available adapters
func AllAdapters() []AdapterInfo {
	return []AdapterInfo{
		{
			Type:        AdapterMatrix,
			DisplayName: "Matrix",
			Description: "End-to-end encrypted decentralized messaging (primary)",
			Encryption:  "E2EE (Olm/Megolm)",
			Status:      StatusDisconnected,
			Features:    []string{"messages", "rooms", "voice", "video", "files"},
		},
		{
			Type:        AdapterSlack,
			DisplayName: "Slack",
			Description: "Workplace communication platform",
			Encryption:  "TLS",
			Status:      StatusDisabled,
			Features:    []string{"messages", "channels", "files", "threads"},
		},
		{
			Type:        AdapterDiscord,
			DisplayName: "Discord",
			Description: "Community and gaming communication",
			Encryption:  "TLS",
			Status:      StatusDisabled,
			Features:    []string{"messages", "servers", "voice", "files"},
		},
		{
			Type:        AdapterTeams,
			DisplayName: "Microsoft Teams",
			Description: "Enterprise communication platform",
			Encryption:  "TLS",
			Status:      StatusDisabled,
			Features:    []string{"messages", "channels", "voice", "video", "files"},
		},
		{
			Type:        AdapterWhatsApp,
			DisplayName: "WhatsApp",
			Description: "Personal messaging with Signal protocol",
			Encryption:  "Signal Protocol",
			Status:      StatusDisabled,
			Features:    []string{"messages", "voice", "video", "files"},
		},
	}
}

// PermissionConfig defines what data can flow through an adapter
type PermissionConfig struct {
	mu sync.RWMutex

	Enabled         bool                `json:"enabled"`
	AllowedData     map[string]bool     `json:"allowed_data"`     // category -> allowed
	AllowedActions  map[AdapterAction]bool `json:"allowed_actions"`
	TrustedRooms    []string            `json:"trusted_rooms,omitempty"`
	TrustedUsers    []string            `json:"trusted_users,omitempty"`
	BlockedUsers    []string            `json:"blocked_users,omitempty"`
	RateLimit       int                 `json:"rate_limit"`       // messages per minute
	RequireApproval bool                `json:"require_approval"`
	AuditLevel      string              `json:"audit_level"`
	ConfiguredAt    time.Time           `json:"configured_at,omitempty"`
	ConfiguredBy    string              `json:"configured_by,omitempty"`
}

// NewPermissionConfig creates a new permission config with defaults
func NewPermissionConfig() *PermissionConfig {
	return &PermissionConfig{
		Enabled:        false,
		AllowedData:    make(map[string]bool),
		AllowedActions: make(map[AdapterAction]bool),
		TrustedRooms:   []string{},
		TrustedUsers:   []string{},
		BlockedUsers:   []string{},
		RateLimit:      10, // 10 messages per minute default
		RequireApproval: true,
		AuditLevel:     "standard",
	}
}

// IsEnabled returns whether the adapter is enabled
func (p *PermissionConfig) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Enabled
}

// IsDataAllowed checks if a data category can be used
func (p *PermissionConfig) IsDataAllowed(category string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allowed, exists := p.AllowedData[category]
	if !exists {
		return false
	}
	return allowed
}

// IsActionAllowed checks if an action is permitted
func (p *PermissionConfig) IsActionAllowed(action AdapterAction) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allowed, exists := p.AllowedActions[action]
	if !exists {
		return false
	}
	return allowed
}

// IsRoomTrusted checks if a room/channel is trusted
func (p *PermissionConfig) IsRoomTrusted(roomID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, room := range p.TrustedRooms {
		if room == roomID {
			return true
		}
	}
	return false
}

// IsUserTrusted checks if a user is trusted
func (p *PermissionConfig) IsUserTrusted(userID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, user := range p.TrustedUsers {
		if user == userID {
			return true
		}
	}
	return false
}

// IsUserBlocked checks if a user is blocked
func (p *PermissionConfig) IsUserBlocked(userID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, user := range p.BlockedUsers {
		if user == userID {
			return true
		}
	}
	return false
}

// SetEnabled enables or disables the adapter
func (p *PermissionConfig) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Enabled = enabled
}

// AllowData enables a data category
func (p *PermissionConfig) AllowData(category string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AllowedData[category] = true
}

// DenyData disables a data category
func (p *PermissionConfig) DenyData(category string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AllowedData[category] = false
}

// AllowAction enables an action
func (p *PermissionConfig) AllowAction(action AdapterAction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AllowedActions[action] = true
}

// DenyAction disables an action
func (p *PermissionConfig) DenyAction(action AdapterAction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AllowedActions[action] = false
}

// AddTrustedRoom adds a room to the trusted list
func (p *PermissionConfig) AddTrustedRoom(roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already exists
	for _, room := range p.TrustedRooms {
		if room == roomID {
			return
		}
	}
	p.TrustedRooms = append(p.TrustedRooms, roomID)
}

// RemoveTrustedRoom removes a room from the trusted list
func (p *PermissionConfig) RemoveTrustedRoom(roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, room := range p.TrustedRooms {
		if room == roomID {
			p.TrustedRooms = append(p.TrustedRooms[:i], p.TrustedRooms[i+1:]...)
			return
		}
	}
}

// AddTrustedUser adds a user to the trusted list
func (p *PermissionConfig) AddTrustedUser(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, user := range p.TrustedUsers {
		if user == userID {
			return
		}
	}
	p.TrustedUsers = append(p.TrustedUsers, userID)
}

// BlockUser adds a user to the blocked list
func (p *PermissionConfig) BlockUser(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from trusted if present
	for i, user := range p.TrustedUsers {
		if user == userID {
			p.TrustedUsers = append(p.TrustedUsers[:i], p.TrustedUsers[i+1:]...)
			break
		}
	}

	// Add to blocked
	for _, user := range p.BlockedUsers {
		if user == userID {
			return
		}
	}
	p.BlockedUsers = append(p.BlockedUsers, userID)
}

// ToJSON returns the config as JSON
func (p *PermissionConfig) ToJSON() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return json.MarshalIndent(p, "", "  ")
}

// Manager handles all adapter permissions
type Manager struct {
	mu     sync.RWMutex
	config map[AdapterType]*PermissionConfig
	status map[AdapterType]AdapterStatus
}

// NewManager creates a new adapter manager
func NewManager() *Manager {
	m := &Manager{
		config: make(map[AdapterType]*PermissionConfig),
		status: make(map[AdapterType]AdapterStatus),
	}

	// Initialize all adapters with default config
	for _, info := range AllAdapters() {
		m.config[info.Type] = NewPermissionConfig()
		m.status[info.Type] = info.Status
	}

	return m
}

// GetConfig returns the permission config for an adapter
func (m *Manager) GetConfig(adapter AdapterType) *PermissionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, exists := m.config[adapter]; exists {
		return config
	}
	return NewPermissionConfig()
}

// SetConfig updates the permission config for an adapter
func (m *Manager) SetConfig(adapter AdapterType, config *PermissionConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.ConfiguredAt = time.Now()
	m.config[adapter] = config
}

// GetStatus returns the current status of an adapter
func (m *Manager) GetStatus(adapter AdapterType) AdapterStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, exists := m.status[adapter]; exists {
		return status
	}
	return StatusDisabled
}

// SetStatus updates the status of an adapter
func (m *Manager) SetStatus(adapter AdapterType, status AdapterStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[adapter] = status
}

// IsEnabled returns whether an adapter is enabled
func (m *Manager) IsEnabled(adapter AdapterType) bool {
	config := m.GetConfig(adapter)
	return config.IsEnabled() && m.GetStatus(adapter) != StatusDisabled
}

// CanTransmitData checks if an adapter can transmit a data category
func (m *Manager) CanTransmitData(adapter AdapterType, category string) bool {
	if !m.IsEnabled(adapter) {
		return false
	}
	return m.GetConfig(adapter).IsDataAllowed(category)
}

// CanPerformAction checks if an adapter can perform an action
func (m *Manager) CanPerformAction(adapter AdapterType, action AdapterAction) bool {
	if !m.IsEnabled(adapter) {
		return false
	}
	return m.GetConfig(adapter).IsActionAllowed(action)
}

// ValidateTransmission checks if a transmission is allowed
func (m *Manager) ValidateTransmission(adapter AdapterType, category string, action AdapterAction, roomID, userID string) error {
	if !m.IsEnabled(adapter) {
		return fmt.Errorf("adapter %s is not enabled", adapter)
	}

	config := m.GetConfig(adapter)

	// Check data category
	if !config.IsDataAllowed(category) {
		return fmt.Errorf("data category %s not allowed on adapter %s", category, adapter)
	}

	// Check action
	if !config.IsActionAllowed(action) {
		return fmt.Errorf("action %s not allowed on adapter %s", action, adapter)
	}

	// Check user blocklist
	if userID != "" && config.IsUserBlocked(userID) {
		return fmt.Errorf("user %s is blocked on adapter %s", userID, adapter)
	}

	// Check if approval is required
	if config.RequireApproval && len(config.TrustedRooms) > 0 {
		if roomID != "" && !config.IsRoomTrusted(roomID) {
			return fmt.Errorf("room %s is not trusted and approval is required", roomID)
		}
	}

	return nil
}

// ListEnabled returns all enabled adapters
func (m *Manager) ListEnabled() []AdapterType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var enabled []AdapterType
	for adapter, config := range m.config {
		if config.IsEnabled() {
			enabled = append(enabled, adapter)
		}
	}
	return enabled
}

// ListAll returns all adapter configurations
func (m *Manager) ListAll() map[AdapterType]*PermissionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[AdapterType]*PermissionConfig)
	for k, v := range m.config {
		result[k] = v
	}
	return result
}

// ToJSON returns all configurations as JSON
func (m *Manager) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.MarshalIndent(map[string]interface{}{
		"config": m.config,
		"status": m.status,
	}, "", "  ")
}

// Summary returns a summary of all adapter states
func (m *Manager) Summary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapters := make(map[string]interface{})
	for adapter, config := range m.config {
		allowedData := []string{}
		for cat, allowed := range config.AllowedData {
			if allowed {
				allowedData = append(allowedData, cat)
			}
		}

		adapters[string(adapter)] = map[string]interface{}{
			"enabled":       config.Enabled,
			"status":        m.status[adapter],
			"allowed_data":  allowedData,
			"rate_limit":    config.RateLimit,
			"require_approval": config.RequireApproval,
		}
	}

	return map[string]interface{}{
		"adapters": adapters,
		"count":    len(m.config),
		"enabled":  len(m.ListEnabled()),
	}
}
