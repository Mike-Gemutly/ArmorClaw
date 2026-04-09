package trust

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type resolveFunc func(hostname string) string

var defaultResolver resolveFunc = resolveProxyIP

func resolveProxyIP(hostname string) string {
	if ip := os.Getenv("ARMORCLAW_PROXY_IP"); ip != "" {
		return ip
	}
	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) == 0 {
		return ""
	}
	return ips[0].String()
}

type TrustedProxyGuard struct {
	cachedProxyIP string
	resolvedAt    time.Time
	cacheMu       sync.RWMutex
	dnsTTL        time.Duration
	proxyHostname string
	resolver      resolveFunc
}

func New(proxyHostname string, dnsTTL time.Duration) *TrustedProxyGuard {
	if dnsTTL <= 0 {
		dnsTTL = 60 * time.Second
	}
	if proxyHostname == "" {
		proxyHostname = "armorclaw-proxy"
	}
	return &TrustedProxyGuard{
		proxyHostname: proxyHostname,
		dnsTTL:        dnsTTL,
		resolver:      defaultResolver,
	}
}

func (g *TrustedProxyGuard) ensureResolved() {
	g.cacheMu.RLock()
	if !g.resolvedAt.IsZero() && time.Since(g.resolvedAt) < g.dnsTTL {
		g.cacheMu.RUnlock()
		return
	}
	g.cacheMu.RUnlock()

	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()

	if !g.resolvedAt.IsZero() && time.Since(g.resolvedAt) < g.dnsTTL {
		return
	}

	ip := g.resolver(g.proxyHostname)
	g.cachedProxyIP = ip
	g.resolvedAt = time.Now()
}

func (g *TrustedProxyGuard) Check(remoteAddr net.Addr) error {
	switch addr := remoteAddr.(type) {
	case *net.UnixAddr:
		return nil
	case *net.TCPAddr:
		g.ensureResolved()

		g.cacheMu.RLock()
		trustedIP := g.cachedProxyIP
		g.cacheMu.RUnlock()

		if trustedIP == "" {
			return fmt.Errorf("trusted proxy guard: DNS resolution failed, denying all TCP connections (failsafe)")
		}

		remoteIP := addr.IP.String()
		if remoteIP == trustedIP {
			return nil
		}

		parsedTrusted := net.ParseIP(trustedIP)
		if parsedTrusted != nil && parsedTrusted.Equal(addr.IP) {
			return nil
		}

		return fmt.Errorf("trusted proxy guard: rejected untrusted TCP connection from %s (trusted proxy: %s)", addr.IP, trustedIP)
	default:
		return fmt.Errorf("trusted proxy guard: unsupported address type %T", remoteAddr)
	}
}
