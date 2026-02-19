package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// WebsiteGuard enforces website allowlist rules at runtime
type WebsiteGuard struct {
	config    *SecurityConfig
	auditLog  AuditLogger
	mu        sync.RWMutex

	// Certificate pinning
	pins      map[string][]string // domain -> pins
}

// AuditLogger interface for logging website access
type AuditLogger interface {
	LogWebsiteAccess(entry WebsiteAccessLog)
}

// WebsiteAccessLog represents a website access audit entry
type WebsiteAccessLog struct {
	Timestamp    time.Time   `json:"timestamp"`
	Domain       string      `json:"domain"`
	Path         string      `json:"path"`
	DataCategory DataCategory `json:"data_category"`
	Action       string      `json:"action"`
	Allowed      bool        `json:"allowed"`
	Reason       string      `json:"reason"`
	SessionID    string      `json:"session_id"`
}

// NewWebsiteGuard creates a new website guard
func NewWebsiteGuard(config *SecurityConfig, auditLog AuditLogger) *WebsiteGuard {
	return &WebsiteGuard{
		config:   config,
		auditLog: auditLog,
		pins:     make(map[string][]string),
	}
}

// CheckAccess verifies if data can be used on a website
func (g *WebsiteGuard) CheckAccess(ctx context.Context, rawURL string, category DataCategory, action string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	domain := parsedURL.Hostname()
	path := parsedURL.Path

	// Get category config
	catConfig := g.config.GetCategory(category)
	if catConfig == nil {
		return fmt.Errorf("unknown data category: %s", category)
	}

	// Check if category is allowed
	if !catConfig.IsAllowed() {
		g.logAccess(domain, path, category, action, false, "category_denied")
		return fmt.Errorf("data category %s is not allowed", category)
	}

	// Check website allowlist
	if !catConfig.IsWebsiteAllowed(domain) {
		g.logAccess(domain, path, category, action, false, "website_not_allowed")
		return fmt.Errorf("data category %s not allowed on domain %s", category, domain)
	}

	// Check website config for path restrictions
	if websiteConfig, exists := g.config.Websites[domain]; exists {
		if len(websiteConfig.Subpaths) > 0 {
			pathAllowed := false
			for _, allowedPath := range websiteConfig.Subpaths {
				if strings.HasPrefix(path, allowedPath) {
					pathAllowed = true
					break
				}
			}
			if !pathAllowed {
				g.logAccess(domain, path, category, action, false, "path_not_allowed")
				return fmt.Errorf("path %s not allowed on domain %s", path, domain)
			}
		}
	}

	// Log successful access
	g.logAccess(domain, path, category, action, true, "allowed")

	return nil
}

// VerifyCertificate checks if a website's certificate is valid
func (g *WebsiteGuard) VerifyCertificate(domain string, certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	// Check if we have pinned certificates for this domain
	pins, hasPins := g.pins[domain]
	if !hasPins {
		// No pinning, rely on standard TLS verification
		return nil
	}

	// Verify certificate against pins
	for _, cert := range certs {
		certPin := pinCertificate(cert)
		for _, expectedPin := range pins {
			if certPin == expectedPin {
				return nil
			}
		}
	}

	return fmt.Errorf("certificate does not match pinned certificates for %s", domain)
}

// AddCertificatePin adds a certificate pin for a domain
func (g *WebsiteGuard) AddCertificatePin(domain, pin string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.pins[domain] = append(g.pins[domain], pin)
}

// pinCertificate generates a pin for a certificate
func pinCertificate(cert *x509.Certificate) string {
	// Simplified - in production use SHA-256 of SubjectPublicKeyInfo
	return fmt.Sprintf("%x", cert.SerialNumber)
}

// ValidateURL checks if a URL is valid and safe
func (g *WebsiteGuard) ValidateURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Require HTTPS for production
	if parsedURL.Scheme != "https" {
		// Allow in development mode
		if os.Getenv("ARMORCLAW_DEV_MODE") != "true" {
			return nil, fmt.Errorf("only HTTPS URLs are allowed")
		}
	}

	// Check for suspicious patterns
	if g.isSuspiciousURL(parsedURL) {
		return nil, fmt.Errorf("suspicious URL pattern detected")
	}

	return parsedURL, nil
}

// isSuspiciousURL checks for suspicious URL patterns
func (g *WebsiteGuard) isSuspiciousURL(u *url.URL) bool {
	// Check for data exfiltration patterns
	suspiciousPatterns := []string{
		"pastebin.com",
		"ngrok.io",
		"burpcollaborator.net",
		"webhook.site",
	}

	host := strings.ToLower(u.Host)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(host, pattern) {
			return true
		}
	}

	return false
}

// GetAllowedDomains returns all domains allowed for a category
func (g *WebsiteGuard) GetAllowedDomains(category DataCategory) []string {
	catConfig := g.config.GetCategory(category)
	if catConfig == nil {
		return nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]string, len(catConfig.AllowedWebsites))
	copy(result, catConfig.AllowedWebsites)
	return result
}

// logAccess logs a website access attempt
func (g *WebsiteGuard) logAccess(domain, path string, category DataCategory, action string, allowed bool, reason string) {
	if g.auditLog == nil {
		return
	}

	entry := WebsiteAccessLog{
		Timestamp:    time.Now(),
		Domain:       domain,
		Path:         path,
		DataCategory: category,
		Action:       action,
		Allowed:      allowed,
		Reason:       reason,
	}

	g.auditLog.LogWebsiteAccess(entry)
}

// ExtractDomain extracts the registrable domain from a hostname
func ExtractDomain(hostname string) string {
	// Remove port if present
	if idx := strings.Index(hostname, ":"); idx != -1 {
		hostname = hostname[:idx]
	}

	// Simple domain extraction (in production use publicsuffix)
	parts := strings.Split(hostname, ".")
	if len(parts) <= 2 {
		return hostname
	}

	// Return last two parts (e.g., example.com)
	return strings.Join(parts[len(parts)-2:], ".")
}

// TLSConfig returns a TLS config with certificate verification
func (g *WebsiteGuard) TLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: false,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return nil // Let standard verification handle this
			}

			serverName := state.ServerName
			if serverName == "" && len(state.PeerCertificates) > 0 {
				serverName = state.PeerCertificates[0].Subject.CommonName
			}

			return g.VerifyCertificate(serverName, state.PeerCertificates)
		},
	}
}

// WebsiteAllowlist manages allowed websites per category
type WebsiteAllowlist struct {
	mu       sync.RWMutex
	Category DataCategory    `json:"category"`
	Domains  []DomainRule    `json:"domains"`
	Default  PermissionLevel `json:"default"`
}

// DomainRule defines rules for a specific domain
type DomainRule struct {
	Domain      string   `json:"domain"`
	Pattern     string   `json:"pattern"`      // glob pattern
	Subpaths    []string `json:"subpaths"`     // allowed subpaths
	DataSubsets []string `json:"data_subsets"` // allowed data subsets
	MaxRetention string   `json:"max_retention"`
}

// IsAllowed checks if a domain and path are allowed
func (w *WebsiteAllowlist) IsAllowed(domain, path string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Default == PermissionAllowAll {
		return true
	}

	for _, rule := range w.Domains {
		if matchDomain(rule.Domain, domain) {
			// Domain matches, check path
			if len(rule.Subpaths) == 0 {
				return true // All paths allowed
			}

			for _, allowedPath := range rule.Subpaths {
				if strings.HasPrefix(path, allowedPath) {
					return true
				}
			}
		}
	}

	return false
}

// AddDomain adds a domain to the allowlist
func (w *WebsiteAllowlist) AddDomain(rule DomainRule) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Domains = append(w.Domains, rule)
}

// RemoveDomain removes a domain from the allowlist
func (w *WebsiteAllowlist) RemoveDomain(domain string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, rule := range w.Domains {
		if rule.Domain == domain {
			w.Domains = append(w.Domains[:i], w.Domains[i+1:]...)
			return
		}
	}
}

// ToJSON returns the allowlist as JSON
func (w *WebsiteAllowlist) ToJSON() ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return json.MarshalIndent(w, "", "  ")
}
