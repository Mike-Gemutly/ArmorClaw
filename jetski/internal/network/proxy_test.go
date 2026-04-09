package network

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestNewProxyManager(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
		},
		HealthCheckURL: "http://example.com",
		HealthInterval: 60 * time.Second,
		RequestTimeout: 30 * time.Second,
	}

	pm, err := NewProxyManager(config)

	if err != nil {
		t.Fatalf("NewProxyManager returned error: %v", err)
	}

	if pm == nil {
		t.Fatal("NewProxyManager returned nil")
	}

	if pm.GetProxyCount() != 2 {
		t.Errorf("Expected 2 proxies, got %d", pm.GetProxyCount())
	}
}

func TestNewProxyManager_EmptyProxyList(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{},
	}

	_, err := NewProxyManager(config)

	if err == nil {
		t.Error("Expected error for empty proxy list")
	}
}

func TestNewProxyManager_InvalidProxyURL(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			":invalid-url",
		},
	}

	_, err := NewProxyManager(config)

	if err == nil {
		t.Error("Expected error for invalid proxy URL")
	}
}

func TestNewProxyManager_DefaultConfig(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{"http://proxy.example.com:8080"},
	}

	pm, err := NewProxyManager(config)

	if err != nil {
		t.Fatalf("NewProxyManager returned error: %v", err)
	}

	if pm.healthCheckURL != "http://www.google.com" {
		t.Errorf("Expected default health check URL, got %s", pm.healthCheckURL)
	}

	if pm.healthInterval != 60*time.Second {
		t.Errorf("Expected default health interval 60s, got %v", pm.healthInterval)
	}
}

func TestProxyManager_CheckProxy_HealthyProxy(t *testing.T) {
	pm, _ := NewProxyManager(ProxyManagerConfig{
		ProxyList: []string{"http://proxy.example.com:8080"},
	})

	proxy := &Proxy{
		URL:     &url.URL{Scheme: "http", Host: "proxy.example.com:8080"},
		Healthy: true,
		CircuitBreaker: NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 3,
			ResetTimeout:     30 * time.Second,
		}),
	}

	ctx := context.Background()
	pm.CheckProxy(ctx, proxy)
}

func TestProxyManager_CheckProxy_CircuitBreakerOpen(t *testing.T) {
	pm, _ := NewProxyManager(ProxyManagerConfig{
		ProxyList: []string{"http://proxy.example.com:8080"},
	})

	proxy := &Proxy{
		URL:     &url.URL{Scheme: "http", Host: "proxy.example.com:8080"},
		Healthy: true,
		CircuitBreaker: NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     30 * time.Second,
		}),
	}

	proxy.CircuitBreaker.RecordFailure()

	ctx := context.Background()
	pm.CheckProxy(ctx, proxy)

	if proxy.Healthy {
		t.Error("Expected proxy to be unhealthy when circuit breaker is open")
	}
}

func TestProxyManager_GetNextProxy_RoundRobin(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
			"http://proxy3.example.com:8080",
		},
	}

	pm, _ := NewProxyManager(config)

	for i := range pm.proxies {
		pm.proxies[i].Healthy = true
		pm.proxies[i].CircuitBreaker.RecordSuccess()
	}

	proxy1, _ := pm.GetNextProxy()
	proxy2, _ := pm.GetNextProxy()
	proxy3, _ := pm.GetNextProxy()
	proxy4, _ := pm.GetNextProxy()

	if proxy1 == proxy2 {
		t.Error("Expected different proxies for consecutive calls")
	}

	if proxy2 == proxy3 {
		t.Error("Expected different proxies for consecutive calls")
	}

	if proxy1 != proxy4 {
		t.Error("Expected proxy rotation to cycle back to first proxy")
	}
}

func TestProxyManager_GetNextProxy_NoHealthyProxies(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
		},
	}

	pm, _ := NewProxyManager(config)

	_, err := pm.GetNextProxy()

	if err == nil {
		t.Error("Expected error when no healthy proxies available")
	}
}

func TestProxyManager_GetNextProxy_SkipsUnhealthy(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
			"http://proxy3.example.com:8080",
		},
	}

	pm, _ := NewProxyManager(config)

	pm.proxies[0].Healthy = true
	pm.proxies[1].Healthy = false
	pm.proxies[2].Healthy = true

	proxy1, _ := pm.GetNextProxy()
	proxy2, _ := pm.GetNextProxy()

	if !proxy1.Healthy {
		t.Error("Expected returned proxy to be healthy")
	}

	if !proxy2.Healthy {
		t.Error("Expected returned proxy to be healthy")
	}

	if proxy1 == proxy2 {
		t.Error("Expected different proxies (should skip unhealthy)")
	}
}

func TestProxyManager_RecordProxyResult(t *testing.T) {
	pm, _ := NewProxyManager(ProxyManagerConfig{
		ProxyList: []string{"http://proxy.example.com:8080"},
	})

	proxy := &Proxy{
		URL:     &url.URL{Scheme: "http", Host: "proxy.example.com:8080"},
		Healthy: true,
		CircuitBreaker: NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 3,
			ResetTimeout:     30 * time.Second,
		}),
	}

	pm.RecordProxyResult(proxy, true)

	state := proxy.CircuitBreaker.GetState()
	if state != StateClosed {
		t.Errorf("Expected circuit to be CLOSED after success, got %v", state)
	}

	pm.RecordProxyResult(proxy, false)

	failureCount := proxy.CircuitBreaker.GetFailureCount()
	if failureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", failureCount)
	}
}

func TestProxyManager_GetHealthyProxies(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
			"http://proxy3.example.com:8080",
		},
	}

	pm, _ := NewProxyManager(config)

	pm.proxies[0].Healthy = true
	pm.proxies[1].Healthy = false
	pm.proxies[2].Healthy = true

	healthy := pm.GetHealthyProxies()

	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy proxies, got %d", len(healthy))
	}
}

func TestProxyManager_GetHealthyProxyCount(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{
			"http://proxy1.example.com:8080",
			"http://proxy2.example.com:8080",
			"http://proxy3.example.com:8080",
		},
	}

	pm, _ := NewProxyManager(config)

	pm.proxies[0].Healthy = true
	pm.proxies[1].Healthy = false
	pm.proxies[2].Healthy = true

	count := pm.GetHealthyProxyCount()

	if count != 2 {
		t.Errorf("Expected healthy proxy count 2, got %d", count)
	}
}

func TestProxyManager_StartHealthChecks(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList:      []string{"http://proxy.example.com:8080"},
		HealthInterval: 100 * time.Millisecond,
	}

	pm, _ := NewProxyManager(config)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	pm.StartHealthChecks(ctx)

	time.Sleep(150 * time.Millisecond)

	pm.StopHealthChecks()
}

func TestProxyManager_GetHTTPClient(t *testing.T) {
	config := ProxyManagerConfig{
		ProxyList: []string{"http://proxy.example.com:8080"},
	}

	pm, _ := NewProxyManager(config)

	pm.proxies[0].Healthy = true

	ctx := context.Background()
	client, err := pm.GetHTTPClient(ctx)

	if err != nil {
		t.Fatalf("GetHTTPClient returned error: %v", err)
	}

	if client == nil {
		t.Error("Expected non-nil HTTP client")
	}
}
