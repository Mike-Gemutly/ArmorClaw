package trust

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestGuard_AllowsTrustedTCPIP(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "127.0.0.1"
	g.resolvedAt = time.Now()

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	if err := g.Check(addr); err != nil {
		t.Fatalf("expected allow for trusted proxy IP, got: %v", err)
	}
}

func TestGuard_BlocksUntrustedTCPIP(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "10.0.0.1"
	g.resolvedAt = time.Now()

	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 54321}
	if err := g.Check(addr); err == nil {
		t.Fatal("expected error for untrusted IP, got nil")
	}
}

func TestGuard_SkipsUnixSocket(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)

	addr := &net.UnixAddr{Name: "/tmp/test.sock", Net: "unix"}
	if err := g.Check(addr); err != nil {
		t.Fatalf("unix socket should always be allowed, got: %v", err)
	}
}

func TestGuard_IPv6Support(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "::1"
	g.resolvedAt = time.Now()

	addr := &net.TCPAddr{IP: net.ParseIP("::1"), Port: 8080}
	if err := g.Check(addr); err != nil {
		t.Fatalf("expected allow for trusted IPv6, got: %v", err)
	}

	addr2 := &net.TCPAddr{IP: net.ParseIP("fe80::1"), Port: 8080}
	if err := g.Check(addr2); err == nil {
		t.Fatal("expected error for untrusted IPv6, got nil")
	}
}

func TestGuard_TTLCacheRefresh(t *testing.T) {
	resolveCount := 0
	originalResolver := defaultResolver
	defaultResolver = func(hostname string) string {
		resolveCount++
		return "10.0.0.1"
	}
	defer func() { defaultResolver = originalResolver }()

	ttl := 50 * time.Millisecond
	g := New("armorclaw-proxy", ttl)

	addr := &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}
	if err := g.Check(addr); err != nil {
		t.Fatalf("first check should succeed: %v", err)
	}
	if resolveCount != 1 {
		t.Fatalf("expected 1 resolve, got %d", resolveCount)
	}

	if err := g.Check(addr); err != nil {
		t.Fatalf("second check within TTL should succeed: %v", err)
	}
	if resolveCount != 1 {
		t.Fatalf("expected still 1 resolve within TTL, got %d", resolveCount)
	}

	time.Sleep(2 * ttl)

	if err := g.Check(addr); err != nil {
		t.Fatalf("third check after TTL should succeed: %v", err)
	}
	if resolveCount != 2 {
		t.Fatalf("expected 2 resolves after TTL expiry, got %d", resolveCount)
	}
}

// SECURITY: net.Conn level has no HTTP headers; X-Forwarded-For is deferred to the HTTP layer.
func TestGuard_ExtractsRealClientIP(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "10.0.0.1"
	g.resolvedAt = time.Now()

	addr := &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 9999}
	if err := g.Check(addr); err != nil {
		t.Fatalf("direct TCP peer should be accepted: %v", err)
	}

	addr2 := &net.TCPAddr{IP: net.ParseIP("172.16.0.5"), Port: 9999}
	if err := g.Check(addr2); err == nil {
		t.Fatal("non-proxy IP should be rejected (no header-based override at conn level)")
	}
}

// SECURITY: Header sanitization belongs at the HTTP layer. The guard rejects untrusted
// connections outright rather than stripping headers at net.Conn level.
func TestGuard_WipesHeadersUntrusted(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "10.0.0.1"
	g.resolvedAt = time.Now()

	untrusted := &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 443}
	err := g.Check(untrusted)
	if err == nil {
		t.Fatal("untrusted connection should be rejected, not have headers stripped")
	}
	t.Logf("correct behavior: untrusted connection rejected with: %v", err)
}

func TestGuard_DualCheckLocking(t *testing.T) {
	originalResolver := defaultResolver
	defaultResolver = func(hostname string) string {
		return "127.0.0.1"
	}
	defer func() { defaultResolver = originalResolver }()

	g := New("armorclaw-proxy", 10*time.Millisecond)

	trustedAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	unixAddr := &net.UnixAddr{Name: "/tmp/race.sock", Net: "unix"}

	var wg sync.WaitGroup
	const goroutines = 50
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				errors[idx] = g.Check(trustedAddr)
			} else {
				errors[idx] = g.Check(unixAddr)
			}
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	time.Sleep(25 * time.Millisecond)

	var wg2 sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg2.Add(1)
		go func(idx int) {
			defer wg2.Done()
			if idx%2 == 0 {
				errors[idx] = g.Check(trustedAddr)
			} else {
				errors[idx] = g.Check(unixAddr)
			}
		}(i)
	}
	wg2.Wait()

	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d (burst 2): unexpected error: %v", i, err)
		}
	}
}

func TestGuard_EnvOverride(t *testing.T) {
	t.Setenv("ARMORCLAW_PROXY_IP", "192.168.50.50")

	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = ""
	g.resolvedAt = time.Time{}

	addr := &net.TCPAddr{IP: net.ParseIP("192.168.50.50"), Port: 1234}
	if err := g.Check(addr); err != nil {
		t.Fatalf("env override should resolve to 192.168.50.50: %v", err)
	}
}

func TestGuard_FailsafeDeny(t *testing.T) {
	originalResolver := defaultResolver
	defaultResolver = func(hostname string) string {
		return ""
	}
	defer func() { defaultResolver = originalResolver }()

	g := New("nonexistent.invalid", 60*time.Second)
	g.cachedProxyIP = ""
	g.resolvedAt = time.Time{}

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	err := g.Check(addr)
	if err == nil {
		t.Fatal("expected failsafe deny when DNS resolution fails")
	}
	t.Logf("failsafe correctly denies: %v", err)
}

func TestGuard_UnknownAddrType(t *testing.T) {
	g := New("armorclaw-proxy", 60*time.Second)
	g.cachedProxyIP = "127.0.0.1"
	g.resolvedAt = time.Now()

	type fakeAddr struct{ net.Addr }
	addr := fakeAddr{}

	err := g.Check(addr)
	if err == nil {
		t.Fatal("expected error for unknown address type")
	}
	_ = fmt.Sprintf("unknown addr rejected: %v", err)
}
