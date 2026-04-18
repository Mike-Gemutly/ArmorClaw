package skills

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRF protection constants
var privateNetworks []*net.IPNet

// Metadata endpoint IPs that should always be blocked
var metadataEndpoints = map[string]bool{
	"169.254.169.254": true, // AWS, GCP, Azure
	"169.254.170.2":   true, // ECS
	"169.254.169.123": true, // Oracle
}

// Initialize private network CIDR ranges
func init() {
	cidrs := []string{
		"127.0.0.0/8",    // localhost
		"10.0.0.0/8",     // private
		"172.16.0.0/12",  // private
		"192.168.0.0/16", // private
		"169.254.0.0/16", // link-local
		"100.64.0.0/10",  // carrier NAT
		"192.0.0.0/24",   // IANA special purpose
		"::1/128",        // IPv6 localhost
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
		"::ffff:0:0/96",  // IPv4-mapped IPv6
		"::/128",         // IPv6 unspecified
		"64:ff9b::/96",   // NAT64
	}

	for _, cidr := range cidrs {
		_, network, _ := net.ParseCIDR(cidr)
		privateNetworks = append(privateNetworks, network)
	}
}

// SSRFValidator provides SSRF protection
type SSRFValidator struct {
	blockPrivateIPs bool
	blockMetadata   bool
	resolveDNSFirst bool
}

// NewSSRFValidator creates a new SSRF validator
func NewSSRFValidator() *SSRFValidator {
	return &SSRFValidator{
		blockPrivateIPs: true,
		blockMetadata:   true,
		resolveDNSFirst: true,
	}
}

// ValidateURL performs SSRF validation on a URL
func (v *SSRFValidator) ValidateURL(urlStr string) error {
	// For Phase 1, simplified validation
	// In production, use proper URL parsing

	// Check if it's an HTTPS URL
	if !strings.HasPrefix(urlStr, "https://") {
		return &SSRFError{
			Type:    "scheme_not_allowed",
			Message: "Only HTTPS URLs are allowed",
			Detail:  urlStr,
		}
	}

	// Extract host (simplified)
	host := v.extractHost(urlStr)
	if host == "" {
		return &SSRFError{
			Type:    "invalid_url",
			Message: "Unable to parse host from URL",
			Detail:  urlStr,
		}
	}

	// Check if it's a metadata endpoint
	if v.blockMetadata && v.isMetadataEndpoint(host) {
		return &SSRFError{
			Type:    "metadata_endpoint_blocked",
			Message: "Cloud metadata endpoint access is blocked",
			Detail:  host,
		}
	}

	// Resolve DNS if required
	if v.resolveDNSFirst {
		ips, err := net.LookupIP(host)
		if err != nil {
			return &SSRFError{
				Type:    "dns_resolution_failed",
				Message: "DNS resolution failed",
				Detail:  err.Error(),
			}
		}

		// Check each resolved IP
		for _, ip := range ips {
			if v.blockPrivateIPs && v.isPrivateIP(ip) {
				return &SSRFError{
					Type:    "private_ip_blocked",
					Message: "Private IP address access is blocked",
					Detail:  ip.String(),
				}
			}
		}
	}

	return nil
}

func (v *SSRFValidator) extractHost(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// isMetadataEndpoint checks if a host is a metadata endpoint
func (v *SSRFValidator) isMetadataEndpoint(host string) bool {
	return metadataEndpoints[host]
}

// isPrivateIP checks if an IP is in a private network range
func (v *SSRFValidator) isPrivateIP(ip net.IP) bool {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// SSRFError represents an SSRF validation error
type SSRFError struct {
	Type    string
	Message string
	Detail  string
}

// Error implements the error interface
func (e *SSRFError) Error() string {
	return e.Message
}

// GetType returns the error type
func (e *SSRFError) GetType() string {
	return e.Type
}

// GetDetail returns error details
func (e *SSRFError) GetDetail() string {
	return e.Detail
}

// IsSSRFError checks if an error is an SSRF error
func IsSSRFError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*SSRFError)
	return ok
}

// GetSSRFErrorType returns the SSRF error type
func GetSSRFErrorType(err error) string {
	if ssrfErr, ok := err.(*SSRFError); ok {
		return ssrfErr.GetType()
	}
	return ""
}

// FormatSSRFError formats an SSRF error for user display
func FormatSSRFError(err error) string {
	if ssrfErr, ok := err.(*SSRFError); ok {
		switch ssrfErr.Type {
		case "private_ip_blocked":
			return fmt.Sprintf("⚠️ Request blocked: %s\n\nThe address %s is inside a private network. ArmorClaw blocks internal network requests to prevent SSRF attacks.", ssrfErr.Message, ssrfErr.Detail)

		case "metadata_endpoint_blocked":
			return fmt.Sprintf("⚠️ Request blocked: %s\n\nCloud metadata endpoint access is blocked for security.", ssrfErr.Message)

		case "scheme_not_allowed":
			return fmt.Sprintf("⚠️ Request blocked: %s\n\nOnly HTTPS URLs are allowed.", ssrfErr.Message)

		default:
			return fmt.Sprintf("⚠️ Request blocked: %s", ssrfErr.Message)
		}
	}
	return "Request blocked for security reasons."
}
