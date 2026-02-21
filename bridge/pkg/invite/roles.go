// Package invite provides role-based invitation management for ArmorClaw.
// Invitations can grant different permission levels to new users.
package invite

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// Role defines the permission level granted by an invite
type Role string

const (
	// RoleAdmin grants full administrative access including security configuration
	RoleAdmin Role = "admin"
	// RoleModerator grants agent management but not security changes
	RoleModerator Role = "moderator"
	// RoleUser grants standard user access to interact with agents
	RoleUser Role = "user"
)

// IsValid checks if a role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleModerator, RoleUser:
		return true
	default:
		return false
	}
}

// ToMatrixPower converts role to Matrix room power level
func (r Role) ToMatrixPower() int {
	switch r {
	case RoleAdmin:
		return 100
	case RoleModerator:
		return 50
	case RoleUser:
		return 0
	default:
		return 0
	}
}

// Permissions returns the permissions granted by this role
func (r Role) Permissions() []string {
	switch r {
	case RoleAdmin:
		return []string{
			"security.read",
			"security.write",
			"adapters.read",
			"adapters.write",
			"skills.read",
			"skills.write",
			"invites.create",
			"invites.revoke",
			"agents.manage",
			"devices.verify",
			"audit.read",
		}
	case RoleModerator:
		return []string{
			"security.read",
			"adapters.read",
			"skills.read",
			"invites.create",
			"agents.manage",
			"audit.read",
		}
	case RoleUser:
		return []string{
			"agents.use",
		}
	default:
		return []string{}
	}
}

// HasPermission checks if the role has a specific permission
func (r Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions() {
		if p == permission {
			return true
		}
	}
	return false
}

// ExpirationOption defines invite expiration durations
type ExpirationOption string

const (
	Expire1Hour   ExpirationOption = "1h"
	Expire6Hours  ExpirationOption = "6h"
	Expire1Day    ExpirationOption = "1d"
	Expire3Days   ExpirationOption = "3d"
	Expire7Days   ExpirationOption = "7d"
	Expire14Days  ExpirationOption = "14d"
	Expire30Days  ExpirationOption = "30d"
	ExpireNever   ExpirationOption = "never"
)

// ToDuration converts expiration option to duration
func (e ExpirationOption) ToDuration() time.Duration {
	switch e {
	case Expire1Hour:
		return 1 * time.Hour
	case Expire6Hours:
		return 6 * time.Hour
	case Expire1Day:
		return 24 * time.Hour
	case Expire3Days:
		return 3 * 24 * time.Hour
	case Expire7Days:
		return 7 * 24 * time.Hour
	case Expire14Days:
		return 14 * 24 * time.Hour
	case Expire30Days:
		return 30 * 24 * time.Hour
	case ExpireNever:
		return 0 // No expiration
	default:
		return 7 * 24 * time.Hour
	}
}

// InviteStatus represents the current state of an invite
type InviteStatus string

const (
	StatusActive   InviteStatus = "active"
	StatusUsed     InviteStatus = "used"
	StatusExpired  InviteStatus = "expired"
	StatusRevoked  InviteStatus = "revoked"
	StatusExhausted InviteStatus = "exhausted"
)

// Invite represents a role-based invitation
type Invite struct {
	mu sync.RWMutex

	ID           string         `json:"id"`
	Code         string         `json:"code"`
	CreatedBy    string         `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	Role         Role           `json:"role"`
	Expiration   ExpirationOption `json:"expiration"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	MaxUses      int            `json:"max_uses"`
	UseCount     int            `json:"use_count"`
	Status       InviteStatus   `json:"status"`
	WelcomMessage string        `json:"welcome_message,omitempty"`

	// Server configuration embedded in invite
	HomeserverURL string   `json:"homeserver_url"`
	BridgeURL     string   `json:"bridge_url"`
	ServerName    string   `json:"server_name"`
	Features      []string `json:"features,omitempty"`
	AutoJoinRooms []string `json:"auto_join_rooms,omitempty"`

	// Tracking
	UsedBy    []InviteUsage `json:"used_by,omitempty"`
	RevokedAt *time.Time    `json:"revoked_at,omitempty"`
	RevokedBy string        `json:"revoked_by,omitempty"`

	// Signature
	Signature string `json:"signature"`
}

// InviteUsage tracks who used an invite
type InviteUsage struct {
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	UsedAt    time.Time `json:"used_at"`
	UserAgent string    `json:"user_agent,omitempty"`
}

// InviteManager handles invitation lifecycle
type InviteManager struct {
	mu      sync.RWMutex
	invites map[string]*Invite
	signingKey []byte
	serverConfig ServerConfig
}

// ServerConfig contains server information embedded in invites
type ServerConfig struct {
	HomeserverURL string
	BridgeURL     string
	ServerName    string
	Features      []string
}

// NewInviteManager creates a new invite manager
func NewInviteManager(signingKey []byte, config ServerConfig) *InviteManager {
	if len(signingKey) == 0 {
		// Generate a random signing key
		signingKey = securerandom.MustBytes(32)
	}

	return &InviteManager{
		invites:     make(map[string]*Invite),
		signingKey:  signingKey,
		serverConfig: config,
	}
}

// CreateInvite creates a new role-based invitation
func (m *InviteManager) CreateInvite(createdBy string, role Role, expiration ExpirationOption, maxUses int, welcomeMessage string) (*Invite, error) {
	if !role.IsValid() {
		return nil, errors.New("invalid role")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate invite ID and code
	inviteID := "inv_" + securerandom.MustID(16)
	code := securerandom.MustID(8)

	invite := &Invite{
		ID:            inviteID,
		Code:          code,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
		Role:          role,
		Expiration:    expiration,
		MaxUses:       maxUses,
		Status:        StatusActive,
		WelcomMessage: welcomeMessage,

		// Embed server config
		HomeserverURL: m.serverConfig.HomeserverURL,
		BridgeURL:     m.serverConfig.BridgeURL,
		ServerName:    m.serverConfig.ServerName,
		Features:      m.serverConfig.Features,
	}

	// Set expiration
	if expiration != ExpireNever {
		expiresAt := time.Now().Add(expiration.ToDuration())
		invite.ExpiresAt = &expiresAt
	}

	// Generate signature
	invite.Signature = m.signInvite(invite)

	m.invites[inviteID] = invite

	return invite, nil
}

// ValidateInvite validates an invite code and returns the invite if valid
func (m *InviteManager) ValidateInvite(code string) (*Invite, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find invite by code
	var invite *Invite
	for _, inv := range m.invites {
		if inv.Code == code {
			invite = inv
			break
		}
	}

	if invite == nil {
		return nil, errors.New("invite not found")
	}

	invite.mu.RLock()
	defer invite.mu.RUnlock()

	// Verify signature
	expectedSig := m.signInvite(invite)
	if !hmac.Equal([]byte(invite.Signature), []byte(expectedSig)) {
		return nil, errors.New("invalid invite signature")
	}

	// Check status
	switch invite.Status {
	case StatusUsed:
		return nil, errors.New("invite already used")
	case StatusRevoked:
		return nil, errors.New("invite has been revoked")
	case StatusExpired:
		return nil, errors.New("invite has expired")
	}

	// Check expiration
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, errors.New("invite has expired")
	}

	// Check usage limit
	if invite.MaxUses > 0 && invite.UseCount >= invite.MaxUses {
		return nil, errors.New("invite usage limit reached")
	}

	return invite, nil
}

// UseInvite marks an invite as used and returns the role
func (m *InviteManager) UseInvite(code, userID, deviceID, userAgent string) (*Invite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find invite
	var invite *Invite
	for _, inv := range m.invites {
		if inv.Code == code {
			invite = inv
			break
		}
	}

	if invite == nil {
		return nil, errors.New("invite not found")
	}

	invite.mu.Lock()
	defer invite.mu.Unlock()

	// Validate again (status may have changed)
	switch invite.Status {
	case StatusRevoked:
		return nil, errors.New("invite has been revoked")
	}

	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		invite.Status = StatusExpired
		return nil, errors.New("invite has expired")
	}

	// Record usage
	usage := InviteUsage{
		UserID:    userID,
		DeviceID:  deviceID,
		UsedAt:    time.Now(),
		UserAgent: userAgent,
	}
	invite.UsedBy = append(invite.UsedBy, usage)
	invite.UseCount++

	// Update status
	if invite.MaxUses > 0 && invite.UseCount >= invite.MaxUses {
		invite.Status = StatusExhausted
	} else if invite.MaxUses == 1 {
		invite.Status = StatusUsed
	}

	return invite, nil
}

// RevokeInvite revokes an active invite
func (m *InviteManager) RevokeInvite(inviteID, revokedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	invite, exists := m.invites[inviteID]
	if !exists {
		return errors.New("invite not found")
	}

	invite.mu.Lock()
	defer invite.mu.Unlock()

	if invite.Status != StatusActive {
		return errors.New("invite is not active")
	}

	now := time.Now()
	invite.Status = StatusRevoked
	invite.RevokedAt = &now
	invite.RevokedBy = revokedBy

	return nil
}

// GetInvite returns an invite by ID
func (m *InviteManager) GetInvite(inviteID string) (*Invite, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	invite, exists := m.invites[inviteID]
	if !exists {
		return nil, errors.New("invite not found")
	}
	return invite, nil
}

// ListInvites returns all invites, optionally filtered by status
func (m *InviteManager) ListInvites(status *InviteStatus) []*Invite {
	m.mu.RLock()
	defer m.mu.RUnlock()

	invites := make([]*Invite, 0)
	for _, invite := range m.invites {
		if status == nil || invite.Status == *status {
			invites = append(invites, invite)
		}
	}
	return invites
}

// ListInvitesByCreator returns invites created by a specific user
func (m *InviteManager) ListInvitesByCreator(createdBy string) []*Invite {
	m.mu.RLock()
	defer m.mu.RUnlock()

	invites := make([]*Invite, 0)
	for _, invite := range m.invites {
		if invite.CreatedBy == createdBy {
			invites = append(invites, invite)
		}
	}
	return invites
}

// CleanupExpired removes expired invites
func (m *InviteManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0

	for id, invite := range m.invites {
		invite.mu.RLock()
		if invite.ExpiresAt != nil && now.After(*invite.ExpiresAt) && invite.Status == StatusActive {
			invite.mu.RUnlock()
			invite.mu.Lock()
			invite.Status = StatusExpired
			invite.mu.Unlock()
			count++
		} else {
			invite.mu.RUnlock()
		}

		// Remove old expired/used/revoked invites (30+ days)
		if invite.Status != StatusActive {
			// Keep for audit purposes, but could delete here
			_ = id
		}
	}

	return count
}

// signInvite generates a signature for an invite
func (m *InviteManager) signInvite(invite *Invite) string {
	data := strings.Join([]string{
		invite.ID,
		invite.Code,
		string(invite.Role),
		invite.CreatedBy,
		invite.CreatedAt.Format(time.RFC3339),
	}, ":")

	h := hmac.New(sha256.New, m.signingKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateInviteURL generates a shareable invite URL
func (m *InviteManager) GenerateInviteURL(invite *Invite) string {
	return fmt.Sprintf("https://armorclaw.app/invite/%s", invite.Code)
}

// GenerateDeepLink generates a deep link for ArmorChat
func (m *InviteManager) GenerateDeepLink(invite *Invite) string {
	return fmt.Sprintf("armorclaw://invite?code=%s&server=%s", invite.Code, invite.HomeserverURL)
}

// ToJSON returns the invite as JSON
func (i *Invite) ToJSON() ([]byte, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return json.MarshalIndent(i, "", "  ")
}

// Summary returns a summary of the invite
func (i *Invite) Summary() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return map[string]interface{}{
		"id":           i.ID,
		"code":         i.Code,
		"role":         string(i.Role),
		"status":       string(i.Status),
		"created_at":   i.CreatedAt,
		"expires_at":   i.ExpiresAt,
		"max_uses":     i.MaxUses,
		"use_count":    i.UseCount,
		"server_name":  i.ServerName,
	}
}

// IsExpired checks if the invite is expired
func (i *Invite) IsExpired() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// IsExhausted checks if the invite has reached its usage limit
func (i *Invite) IsExhausted() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.MaxUses <= 0 {
		return false
	}
	return i.UseCount >= i.MaxUses
}

// RemainingUses returns the number of remaining uses
func (i *Invite) RemainingUses() int {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.MaxUses <= 0 {
		return -1 // Unlimited
	}
	remaining := i.MaxUses - i.UseCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ParseInviteFromURL extracts invite code from a URL
func ParseInviteFromURL(url string) (string, error) {
	// Handle armorclaw://invite?code=XXX
	if strings.HasPrefix(url, "armorclaw://") {
		parts := strings.Split(url, "?")
		if len(parts) < 2 {
			return "", errors.New("invalid deep link format")
		}
		params := strings.Split(parts[1], "&")
		for _, param := range params {
			kv := strings.Split(param, "=")
			if len(kv) == 2 && kv[0] == "code" {
				return kv[1], nil
			}
		}
	}

	// Handle https://armorclaw.app/invite/XXX
	if strings.Contains(url, "/invite/") {
		parts := strings.Split(url, "/invite/")
		if len(parts) == 2 {
			code := strings.Split(parts[1], "?")[0]
			code = strings.Split(code, "/")[0]
			return code, nil
		}
	}

	// Assume it's just a code
	if len(url) >= 8 && len(url) <= 32 {
		return url, nil
	}

	return "", errors.New("invalid invite URL format")
}

// Stats returns invite statistics
func (m *InviteManager) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     len(m.invites),
		"active":    0,
		"used":      0,
		"expired":   0,
		"revoked":   0,
		"exhausted": 0,
	}

	byRole := map[string]int{
		"admin":     0,
		"moderator": 0,
		"user":      0,
	}

	for _, invite := range m.invites {
		switch invite.Status {
		case StatusActive:
			stats["active"]++
		case StatusUsed:
			stats["used"]++
		case StatusExpired:
			stats["expired"]++
		case StatusRevoked:
			stats["revoked"]++
		case StatusExhausted:
			stats["exhausted"]++
		}
		byRole[string(invite.Role)]++
	}

	return map[string]interface{}{
		"status": stats,
		"by_role": byRole,
	}
}

// UserMaxInvites limits how many invites a user can create
func (m *InviteManager) UserMaxInvites(createdBy string, maxActive int) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := 0
	for _, invite := range m.invites {
		if invite.CreatedBy == createdBy && invite.Status == StatusActive {
			active++
		}
	}

	if active >= maxActive {
		return fmt.Errorf("maximum active invites (%d) reached", maxActive)
	}

	return nil
}
