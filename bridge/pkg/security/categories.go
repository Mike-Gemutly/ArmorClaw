// Package security provides data category management, website allowlists,
// and security tier configuration for ArmorClaw.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DataCategory represents types of sensitive information
type DataCategory string

const (
	// CategoryBanking - Account numbers, routing numbers, balances
	CategoryBanking DataCategory = "banking"
	// CategoryPII - SSN, driver's license, passport, tax IDs
	CategoryPII DataCategory = "pii"
	// CategoryMedical - Health records, prescriptions, diagnoses
	CategoryMedical DataCategory = "medical"
	// CategoryResidential - Home address, phone, email
	CategoryResidential DataCategory = "residential"
	// CategoryNetwork - IP, MAC, hostname, DNS info
	CategoryNetwork DataCategory = "network"
	// CategoryIdentity - Name, DOB, photo
	CategoryIdentity DataCategory = "identity"
	// CategoryLocation - GPS, city, country
	CategoryLocation DataCategory = "location"
	// CategoryCredentials - Usernames, passwords, API keys
	CategoryCredentials DataCategory = "credentials"
)

// CategoryInfo provides metadata about a data category
type CategoryInfo struct {
	Name        DataCategory `json:"name"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	Examples    []string     `json:"examples"`
	RiskLevel   string       `json:"risk_level"` // high, medium, low
}

// AllCategories returns all available data categories with metadata
func AllCategories() []CategoryInfo {
	return []CategoryInfo{
		{
			Name:        CategoryBanking,
			DisplayName: "Banking Information",
			Description: "Financial account details and transaction data",
			Examples:    []string{"account numbers", "routing numbers", "balances", "credit card numbers"},
			RiskLevel:   "high",
		},
		{
			Name:        CategoryPII,
			DisplayName: "Personally Identifiable Information",
			Description: "Government-issued identifiers and personal documents",
			Examples:    []string{"SSN", "driver's license", "passport", "tax ID"},
			RiskLevel:   "high",
		},
		{
			Name:        CategoryMedical,
			DisplayName: "Medical Information",
			Description: "Health records and medical history",
			Examples:    []string{"diagnoses", "prescriptions", "lab results", "insurance info"},
			RiskLevel:   "high",
		},
		{
			Name:        CategoryResidential,
			DisplayName: "Residential Information",
			Description: "Physical address and contact details",
			Examples:    []string{"home address", "phone number", "personal email"},
			RiskLevel:   "medium",
		},
		{
			Name:        CategoryNetwork,
			DisplayName: "Network Information",
			Description: "Network identifiers and infrastructure details",
			Examples:    []string{"IP address", "MAC address", "hostname", "DNS records"},
			RiskLevel:   "medium",
		},
		{
			Name:        CategoryIdentity,
			DisplayName: "Identity Information",
			Description: "Personal identity attributes",
			Examples:    []string{"full name", "date of birth", "photo", "signature"},
			RiskLevel:   "medium",
		},
		{
			Name:        CategoryLocation,
			DisplayName: "Location Information",
			Description: "Geographic location data",
			Examples:    []string{"GPS coordinates", "city", "country", "timezone"},
			RiskLevel:   "low",
		},
		{
			Name:        CategoryCredentials,
			DisplayName: "Credentials",
			Description: "Authentication and access credentials",
			Examples:    []string{"usernames", "passwords", "API keys", "tokens"},
			RiskLevel:   "high",
		},
	}
}

// PermissionLevel defines how a category can be used
type PermissionLevel string

const (
	// PermissionDeny - Category cannot be used at all
	PermissionDeny PermissionLevel = "deny"
	// PermissionAllow - Category can be used with restrictions
	PermissionAllow PermissionLevel = "allow"
	// PermissionAllowAll - Category can be used without restrictions (not recommended)
	PermissionAllowAll PermissionLevel = "allow_all"
)

// AuditLevel defines how much to log
type AuditLevel string

const (
	// AuditNone - No audit logging
	AuditNone AuditLevel = "none"
	// AuditMinimal - Log access only
	AuditMinimal AuditLevel = "minimal"
	// AuditStandard - Log access and actions
	AuditStandard AuditLevel = "standard"
	// AuditVerbose - Log everything including data hashes
	AuditVerbose AuditLevel = "verbose"
)

// SubsetConfig defines permissions for a subset of a category
type SubsetConfig struct {
	Permission     PermissionLevel `json:"permission"`
	AllowedActions []string        `json:"allowed_actions,omitempty"`
}

// CategoryConfig defines how a data category can be handled
type CategoryConfig struct {
	mu sync.RWMutex

	Permission       PermissionLevel          `json:"permission"`
	AllowedWebsites  []string                 `json:"allowed_websites,omitempty"`
	BlockedWebsites  []string                 `json:"blocked_websites,omitempty"`
	AllowedAdapters  []string                 `json:"allowed_adapters,omitempty"`
	DataSubsets      map[string]SubsetConfig  `json:"data_subsets,omitempty"`
	RequiresApproval bool                     `json:"requires_approval"`
	AuditLevel       AuditLevel               `json:"audit_level"`
	MaxRetention     string                   `json:"max_retention,omitempty"` // Duration string
	ConfiguredAt     time.Time                `json:"configured_at,omitempty"`
	ConfiguredBy     string                   `json:"configured_by,omitempty"`
}

// IsAllowed checks if the category can be used
func (c *CategoryConfig) IsAllowed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Permission != PermissionDeny
}

// IsWebsiteAllowed checks if a website can receive this data
func (c *CategoryConfig) IsWebsiteAllowed(domain string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Permission == PermissionAllowAll {
		return true
	}

	// Check blocked list first
	for _, blocked := range c.BlockedWebsites {
		if matchDomain(blocked, domain) {
			return false
		}
	}

	// If no allowlist, deny by default
	if len(c.AllowedWebsites) == 0 {
		return false
	}

	// Check allowlist
	for _, allowed := range c.AllowedWebsites {
		if matchDomain(allowed, domain) {
			return true
		}
	}

	return false
}

// IsAdapterAllowed checks if an adapter can transmit this data
func (c *CategoryConfig) IsAdapterAllowed(adapter string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Permission == PermissionAllowAll {
		return true
	}

	for _, allowed := range c.AllowedAdapters {
		if allowed == adapter {
			return true
		}
	}

	return false
}

// IsSubsetAllowed checks if a data subset can be used
func (c *CategoryConfig) IsSubsetAllowed(subset string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Permission == PermissionAllowAll {
		return true
	}

	if c.DataSubsets == nil {
		return c.Permission != PermissionDeny
	}

	subsetConfig, exists := c.DataSubsets[subset]
	if !exists {
		return c.Permission != PermissionDeny
	}

	return subsetConfig.Permission != PermissionDeny
}

// matchDomain checks if a pattern matches a domain
func matchDomain(pattern, domain string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	if pattern == domain {
		return true
	}
	// Wildcard subdomain matching (*.example.com)
	if len(pattern) > 2 && pattern[:2] == "*." {
		suffix := pattern[1:] // .example.com
		if len(domain) >= len(suffix) && domain[len(domain)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

// SecurityConfig represents the complete security configuration
type SecurityConfig struct {
	mu sync.RWMutex

	Version     string                     `json:"version"`
	AdminID     string                     `json:"admin_id"`
	ConfiguredAt time.Time                 `json:"configured_at"`
	Tier        SecurityTier               `json:"tier"`

	Categories  map[DataCategory]*CategoryConfig `json:"categories"`
	Adapters    map[string]AdapterConfig         `json:"adapters"`
	Websites    map[string]WebsiteConfig         `json:"websites"`
	Skills      map[string]SkillConfig           `json:"skills"`

	configFile string
}

// SecurityTier represents the overall security level
type SecurityTier string

const (
	// TierParanoid - Most restrictive
	TierParanoid SecurityTier = "paranoid"
	// TierStrict - Allowlist-based
	TierStrict SecurityTier = "strict"
	// TierBalanced - Context-aware (recommended)
	TierBalanced SecurityTier = "balanced"
	// TierPermissive - Blocklist-based
	TierPermissive SecurityTier = "permissive"
	// TierOpen - Minimal restrictions (not recommended)
	TierOpen SecurityTier = "open"
)

// AdapterConfig defines adapter permissions
type AdapterConfig struct {
	Enabled         bool           `json:"enabled"`
	AllowedData     []DataCategory `json:"allowed_data"`
	AllowedActions  []string       `json:"allowed_actions"`
	TrustedRooms    []string       `json:"trusted_rooms,omitempty"`
	TrustedUsers    []string       `json:"trusted_users,omitempty"`
	RateLimit       int            `json:"rate_limit"`
	RequireApproval bool           `json:"require_approval"`
	AuditLevel      AuditLevel     `json:"audit_level"`
}

// WebsiteConfig defines website-specific rules
type WebsiteConfig struct {
	Domain       string   `json:"domain"`
	AllowedData  []DataCategory `json:"allowed_data"`
	Subpaths     []string `json:"subpaths,omitempty"`
	MaxRetention string   `json:"max_retention,omitempty"`
	RequiresHTTPS bool    `json:"requires_https"`
}

// SkillConfig defines skill permissions
type SkillConfig struct {
	Enabled         bool           `json:"enabled"`
	RequiredData    []DataCategory `json:"required_data,omitempty"`
	RequiresApproval bool          `json:"requires_approval"`
	AuditLevel      AuditLevel     `json:"audit_level"`
}

// NewSecurityConfig creates a new security configuration
func NewSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Version:    "1.0.0",
		Tier:       TierBalanced,
		Categories: make(map[DataCategory]*CategoryConfig),
		Adapters:   make(map[string]AdapterConfig),
		Websites:   make(map[string]WebsiteConfig),
		Skills:     make(map[string]SkillConfig),
	}
}

// LoadSecurityConfig loads configuration from a file
func LoadSecurityConfig(path string) (*SecurityConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSecurityConfig(), nil
		}
		return nil, err
	}

	config := &SecurityConfig{configFile: path}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	return config, nil
}

// SetConfigFile sets the path for saving configuration
func (s *SecurityConfig) SetConfigFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configFile = path
}

// GetCategory returns the configuration for a data category
func (s *SecurityConfig) GetCategory(category DataCategory) *CategoryConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if config, exists := s.Categories[category]; exists {
		return config
	}

	// Return default (deny) configuration
	return &CategoryConfig{
		Permission:       PermissionDeny,
		RequiresApproval: true,
		AuditLevel:       AuditStandard,
	}
}

// SetCategory updates the configuration for a data category
func (s *SecurityConfig) SetCategory(category DataCategory, config *CategoryConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.ConfiguredAt = time.Now()
	s.Categories[category] = config

	return s.save()
}

// SetTier updates the security tier
func (s *SecurityConfig) SetTier(tier SecurityTier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Tier = tier

	// Apply tier defaults
	switch tier {
	case TierParanoid:
		s.applyParanoidDefaults()
	case TierStrict:
		s.applyStrictDefaults()
	case TierBalanced:
		s.applyBalancedDefaults()
	case TierPermissive:
		s.applyPermissiveDefaults()
	case TierOpen:
		s.applyOpenDefaults()
	}

	return s.save()
}

// GetAdapter returns the configuration for an adapter
func (s *SecurityConfig) GetAdapter(name string) AdapterConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if config, exists := s.Adapters[name]; exists {
		return config
	}

	return AdapterConfig{
		Enabled:         false,
		RequireApproval: true,
		AuditLevel:      AuditStandard,
	}
}

// SetAdapter updates the configuration for an adapter
func (s *SecurityConfig) SetAdapter(name string, config AdapterConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Adapters[name] = config
	return s.save()
}

// GetSkill returns the configuration for a skill
func (s *SecurityConfig) GetSkill(name string) SkillConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if config, exists := s.Skills[name]; exists {
		return config
	}

	return SkillConfig{
		Enabled:          false,
		RequiresApproval: true,
		AuditLevel:       AuditStandard,
	}
}

// SetSkill updates the configuration for a skill
func (s *SecurityConfig) SetSkill(name string, config SkillConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Skills[name] = config
	return s.save()
}

// IsDataAllowed checks if a data category can be used in a given context
func (s *SecurityConfig) IsDataAllowed(category DataCategory, context DataUsageContext) error {
	catConfig := s.GetCategory(category)

	if !catConfig.IsAllowed() {
		return fmt.Errorf("data category %s is not allowed", category)
	}

	// Check website if provided
	if context.Website != "" && !catConfig.IsWebsiteAllowed(context.Website) {
		return fmt.Errorf("data category %s not allowed on website %s", category, context.Website)
	}

	// Check adapter if provided
	if context.Adapter != "" && !catConfig.IsAdapterAllowed(context.Adapter) {
		return fmt.Errorf("data category %s not allowed on adapter %s", category, context.Adapter)
	}

	// Check subset if provided
	if context.Subset != "" && !catConfig.IsSubsetAllowed(context.Subset) {
		return fmt.Errorf("data subset %s of category %s is not allowed", context.Subset, category)
	}

	// Check if approval is required
	if catConfig.RequiresApproval && !context.Approved {
		return fmt.Errorf("data category %s requires approval", category)
	}

	return nil
}

// DataUsageContext describes where and how data is being used
type DataUsageContext struct {
	Website  string `json:"website,omitempty"`
	Adapter  string `json:"adapter,omitempty"`
	Subset   string `json:"subset,omitempty"`
	Action   string `json:"action,omitempty"`
	Approved bool   `json:"approved"`
}

// save writes configuration to disk
func (s *SecurityConfig) save() error {
	if s.configFile == "" {
		return nil
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.configFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	tmpFile := s.configFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0640); err != nil {
		return err
	}

	return os.Rename(tmpFile, s.configFile)
}

// Tier default applications

func (s *SecurityConfig) applyParanoidDefaults() {
	// Deny all by default, require approval for everything
	for _, cat := range AllCategories() {
		s.Categories[cat.Name] = &CategoryConfig{
			Permission:       PermissionDeny,
			RequiresApproval: true,
			AuditLevel:       AuditVerbose,
		}
	}
}

func (s *SecurityConfig) applyStrictDefaults() {
	// Deny all, must explicitly allow
	for _, cat := range AllCategories() {
		s.Categories[cat.Name] = &CategoryConfig{
			Permission:       PermissionDeny,
			RequiresApproval: true,
			AuditLevel:       AuditStandard,
		}
	}
}

func (s *SecurityConfig) applyBalancedDefaults() {
	// Allow low-risk categories, deny high-risk
	for _, cat := range AllCategories() {
		permission := PermissionDeny
		if cat.RiskLevel == "low" {
			permission = PermissionAllow
		}

		s.Categories[cat.Name] = &CategoryConfig{
			Permission:       permission,
			RequiresApproval: cat.RiskLevel == "high",
			AuditLevel:       AuditStandard,
		}
	}
}

func (s *SecurityConfig) applyPermissiveDefaults() {
	// Allow most, block explicit
	for _, cat := range AllCategories() {
		s.Categories[cat.Name] = &CategoryConfig{
			Permission:       PermissionAllow,
			RequiresApproval: false,
			AuditLevel:       AuditMinimal,
		}
	}
}

func (s *SecurityConfig) applyOpenDefaults() {
	// Allow all with minimal logging
	for _, cat := range AllCategories() {
		s.Categories[cat.Name] = &CategoryConfig{
			Permission:       PermissionAllowAll,
			RequiresApproval: false,
			AuditLevel:       AuditMinimal,
		}
	}
}

// ToJSON returns the configuration as JSON
func (s *SecurityConfig) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s, "", "  ")
}

// Summary returns a summary of the current configuration
func (s *SecurityConfig) Summary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categoryStatus := make(map[string]string)
	for name, config := range s.Categories {
		categoryStatus[string(name)] = string(config.Permission)
	}

	return map[string]interface{}{
		"version":      s.Version,
		"tier":         s.Tier,
		"configured_at": s.ConfiguredAt,
		"categories":   categoryStatus,
		"adapters_enabled": countEnabledAdapters(s.Adapters),
		"skills_enabled":   countEnabledSkills(s.Skills),
	}
}

func countEnabledAdapters(m map[string]AdapterConfig) int {
	count := 0
	for _, v := range m {
		if v.Enabled {
			count++
		}
	}
	return count
}

func countEnabledSkills(m map[string]SkillConfig) int {
	count := 0
	for _, v := range m {
		if v.Enabled {
			count++
		}
	}
	return count
}

// Clone creates a deep copy of the configuration
func (s *SecurityConfig) Clone() *SecurityConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &SecurityConfig{
		Version:     s.Version,
		AdminID:     s.AdminID,
		ConfiguredAt: s.ConfiguredAt,
		Tier:        s.Tier,
		Categories:  make(map[DataCategory]*CategoryConfig),
		Adapters:    make(map[string]AdapterConfig),
		Websites:    make(map[string]WebsiteConfig),
		Skills:      make(map[string]SkillConfig),
		configFile:  s.configFile,
	}

	for k, v := range s.Categories {
		// Create a copy without copying the mutex
		copied := &CategoryConfig{
			Permission:       v.Permission,
			AllowedWebsites:  append([]string(nil), v.AllowedWebsites...),
			BlockedWebsites:  append([]string(nil), v.BlockedWebsites...),
			AllowedAdapters:  append([]string(nil), v.AllowedAdapters...),
			DataSubsets:      make(map[string]SubsetConfig),
			RequiresApproval: v.RequiresApproval,
			AuditLevel:       v.AuditLevel,
			MaxRetention:     v.MaxRetention,
			ConfiguredAt:     v.ConfiguredAt,
			ConfiguredBy:     v.ConfiguredBy,
		}
		for sk, sv := range v.DataSubsets {
			copied.DataSubsets[sk] = sv
		}
		clone.Categories[k] = copied
	}

	for k, v := range s.Adapters {
		clone.Adapters[k] = v
	}

	for k, v := range s.Websites {
		clone.Websites[k] = v
	}

	for k, v := range s.Skills {
		clone.Skills[k] = v
	}

	return clone
}
