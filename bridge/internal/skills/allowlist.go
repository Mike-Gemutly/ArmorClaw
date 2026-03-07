package skills

import (
	"net"
	"sync"
)

// AllowlistManager manages IP allowlists for admin overrides
type AllowlistManager struct {
	allowedIPs    map[string]bool
	allowedRanges []*net.IPNet
	mu           sync.RWMutex
}

// NewAllowlistManager creates a new allowlist manager
func NewAllowlistManager() *AllowlistManager {
	return &AllowlistManager{
		allowedIPs:    make(map[string]bool),
		allowedRanges: make([]*net.IPNet, 0),
	}
}

// AllowIP adds a specific IP to the allowlist
func (am *AllowlistManager) AllowIP(ipStr string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &SSRFError{
			Type:    "invalid_ip",
			Message: "Invalid IP address format",
			Detail:  ipStr,
		}
	}

	am.allowedIPs[ip.String()] = true
	return nil
}

// AllowCIDR adds a CIDR range to the allowlist
func (am *AllowlistManager) AllowCIDR(cidr string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return &SSRFError{
			Type:    "invalid_cidr",
			Message: "Invalid CIDR format",
			Detail:  cidr,
		}
	}

	am.allowedRanges = append(am.allowedRanges, ipNet)
	return nil
}

// IsAllowed checks if an IP is in the allowlist
func (am *AllowlistManager) IsAllowed(ip net.IP) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Check exact IP match
	if am.allowedIPs[ip.String()] {
		return true
	}

	// Check CIDR ranges
	for _, ipNet := range am.allowedRanges {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// RemoveAllowedIP removes an IP from the allowlist
func (am *AllowlistManager) RemoveAllowedIP(ipStr string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	ip := net.ParseIP(ipStr)
	if ip != nil {
		delete(am.allowedIPs, ip.String())
	}
}

// RemoveAllowedCIDR removes a CIDR range from the allowlist
func (am *AllowlistManager) RemoveAllowedCIDR(cidr string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, ipNet, err := net.ParseCIDR(cidr)
	if err == nil {
		for i, rangeNet := range am.allowedRanges {
			if ipNet.IP.Equal(rangeNet.IP) && bytesEqual(ipNet.Mask, rangeNet.Mask) {
				am.allowedRanges = append(am.allowedRanges[:i], am.allowedRanges[i+1:]...)
				break
			}
		}
	}
}

// ListAllowed returns all allowed IPs and CIDRs
func (am *AllowlistManager) ListAllowed() ([]string, []string) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	ips := make([]string, 0, len(am.allowedIPs))
	for ip := range am.allowedIPs {
		ips = append(ips, ip)
	}

	cidrs := make([]string, len(am.allowedRanges))
	for i, ipNet := range am.allowedRanges {
		cidrs[i] = ipNet.String()
	}

	return ips, cidrs
}

// ClearAllowlist removes all entries from the allowlist
func (am *AllowlistManager) ClearAllowlist() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.allowedIPs = make(map[string]bool)
	am.allowedRanges = make([]*net.IPNet, 0)
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}