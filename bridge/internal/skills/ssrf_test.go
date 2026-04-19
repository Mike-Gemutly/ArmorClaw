package skills

import (
	"net"
	"testing"
)

func TestSSRF_IPv6Multicast(t *testing.T) {
	v := NewSSRFValidator()

	multicastIPs := []string{
		"ff00::1",
		"ff02::1",
		"ff0e::1",
		"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
	}

	for _, addr := range multicastIPs {
		ip := net.ParseIP(addr)
		if ip == nil {
			t.Fatalf("failed to parse IP %q", addr)
		}
		if !v.isPrivateIP(ip) {
			t.Errorf("isPrivateIP(%s) = false, want true (IPv6 multicast)", addr)
		}
	}
}

func TestSSRF_ExistingIPv6Ranges(t *testing.T) {
	v := NewSSRFValidator()

	blocked := []struct {
		addr string
		name string
	}{
		{"::1", "localhost"},
		{"fc00::1", "unique local"},
		{"fe80::1", "link-local"},
	}

	for _, tc := range blocked {
		ip := net.ParseIP(tc.addr)
		if ip == nil {
			t.Fatalf("failed to parse IP %q", tc.addr)
		}
		if !v.isPrivateIP(ip) {
			t.Errorf("isPrivateIP(%s) = false, want true (%s)", tc.addr, tc.name)
		}
	}

	publicIP := net.ParseIP("2001:db8::1")
	if publicIP == nil {
		t.Fatal("failed to parse 2001:db8::1")
	}
	if v.isPrivateIP(publicIP) {
		t.Errorf("isPrivateIP(2001:db8::1) = true, want false (public IPv6)")
	}
}
