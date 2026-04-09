package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Proxy represents a single proxy server
type Proxy struct {
	URL            *url.URL
	Healthy        bool
	LastCheck      time.Time
	CircuitBreaker *CircuitBreaker
}

// ProxyManager manages a pool of proxies with round-robin rotation
type ProxyManager struct {
	mu             sync.RWMutex
	proxies        []*Proxy
	currentIndex   int
	healthCheckURL string
	healthInterval time.Duration
	checkInterval  *time.Ticker
	client         *http.Client
}

// ProxyManagerConfig contains configuration for the proxy manager
type ProxyManagerConfig struct {
	ProxyList      []string      // List of proxy URLs
	HealthCheckURL string        // URL to check proxy health (default: http://www.google.com)
	HealthInterval time.Duration // Health check interval (default: 60s)
	RequestTimeout time.Duration // Request timeout (default: 30s)
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(config ProxyManagerConfig) (*ProxyManager, error) {
	if config.HealthCheckURL == "" {
		config.HealthCheckURL = "http://www.google.com"
	}
	if config.HealthInterval <= 0 {
		config.HealthInterval = 60 * time.Second
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if len(config.ProxyList) == 0 {
		return nil, fmt.Errorf("proxy list cannot be empty")
	}

	pm := &ProxyManager{
		proxies:        make([]*Proxy, 0),
		currentIndex:   0,
		healthCheckURL: config.HealthCheckURL,
		healthInterval: config.HealthInterval,
		client: &http.Client{
			Timeout: config.RequestTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   config.RequestTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
				DisableKeepAlives:  false,
			},
		},
	}

	for _, proxyStr := range config.ProxyList {
		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %s: %w", proxyStr, err)
		}

		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold:  3,
			ResetTimeout:      30 * time.Second,
			HalfOpenThreshold: 1,
		})

		pm.proxies = append(pm.proxies, &Proxy{
			URL:            proxyURL,
			Healthy:        false,
			LastCheck:      time.Now(),
			CircuitBreaker: cb,
		})
	}

	return pm, nil
}

// StartHealthChecks begins periodic health checking of all proxies
func (pm *ProxyManager) StartHealthChecks(ctx context.Context) {
	pm.checkInterval = time.NewTicker(pm.healthInterval)
	go pm.healthCheckLoop(ctx)
}

// StopHealthChecks stops the health checking loop
func (pm *ProxyManager) StopHealthChecks() {
	if pm.checkInterval != nil {
		pm.checkInterval.Stop()
	}
}

// healthCheckLoop performs periodic health checks on all proxies
func (pm *ProxyManager) healthCheckLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.checkInterval.C:
			pm.CheckAllProxies(ctx)
		}
	}
}

// CheckAllProxies performs health checks on all proxies
func (pm *ProxyManager) CheckAllProxies(ctx context.Context) {
	pm.mu.RLock()
	proxies := make([]*Proxy, len(pm.proxies))
	copy(proxies, pm.proxies)
	pm.mu.RUnlock()

	var wg sync.WaitGroup
	for _, proxy := range proxies {
		wg.Add(1)
		go func(p *Proxy) {
			defer wg.Done()
			pm.CheckProxy(ctx, p)
		}(proxy)
	}
	wg.Wait()
}

// CheckProxy checks the health of a single proxy
func (pm *ProxyManager) CheckProxy(ctx context.Context, proxy *Proxy) {
	if !proxy.CircuitBreaker.Allow() {
		proxy.Healthy = false
		return
	}

	transport := pm.client.Transport.(*http.Transport)
	transport.Proxy = http.ProxyURL(proxy.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", pm.healthCheckURL, nil)
	if err != nil {
		proxy.CircuitBreaker.RecordFailure()
		proxy.Healthy = false
		return
	}

	resp, err := pm.client.Do(req)
	if err != nil {
		proxy.CircuitBreaker.RecordFailure()
		proxy.Healthy = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		proxy.CircuitBreaker.RecordSuccess()
		proxy.Healthy = true
	} else {
		proxy.CircuitBreaker.RecordFailure()
		proxy.Healthy = false
	}

	proxy.LastCheck = time.Now()
}

// GetNextProxy returns the next healthy proxy in round-robin order
func (pm *ProxyManager) GetNextProxy() (*Proxy, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	startIndex := pm.currentIndex
	healthyProxyFound := false

	for i := 0; i < len(pm.proxies); i++ {
		index := (startIndex + i) % len(pm.proxies)
		proxy := pm.proxies[index]

		if proxy.Healthy && proxy.CircuitBreaker.Allow() {
			pm.currentIndex = (index + 1) % len(pm.proxies)
			healthyProxyFound = true
			return proxy, nil
		}
	}

	if !healthyProxyFound {
		pm.currentIndex = (startIndex + 1) % len(pm.proxies)
		return nil, fmt.Errorf("no healthy proxies available")
	}

	return nil, fmt.Errorf("no proxies available")
}

// GetHTTPClient returns an HTTP client configured with the next proxy
func (pm *ProxyManager) GetHTTPClient(ctx context.Context) (*http.Client, error) {
	proxy, err := pm.GetNextProxy()
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Timeout: pm.client.Timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy.URL),
			DialContext: (&net.Dialer{
				Timeout:   pm.client.Timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:       10,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			DisableKeepAlives:  false,
		},
	}, nil
}

// RecordProxyResult records the result of using a proxy
func (pm *ProxyManager) RecordProxyResult(proxy *Proxy, success bool) {
	if success {
		proxy.CircuitBreaker.RecordSuccess()
	} else {
		proxy.CircuitBreaker.RecordFailure()
	}
}

// GetHealthyProxies returns a list of all healthy proxies
func (pm *ProxyManager) GetHealthyProxies() []*Proxy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	healthy := make([]*Proxy, 0)
	for _, proxy := range pm.proxies {
		if proxy.Healthy && proxy.CircuitBreaker.Allow() {
			healthy = append(healthy, proxy)
		}
	}
	return healthy
}

// GetProxyCount returns the total number of proxies
func (pm *ProxyManager) GetProxyCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.proxies)
}

// GetHealthyProxyCount returns the number of healthy proxies
func (pm *ProxyManager) GetHealthyProxyCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count := 0
	for _, proxy := range pm.proxies {
		if proxy.Healthy {
			count++
		}
	}
	return count
}
