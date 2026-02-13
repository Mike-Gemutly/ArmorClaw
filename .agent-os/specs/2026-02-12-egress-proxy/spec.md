# Egress Proxy Implementation Plan

> Created: 2026-02-12
> Status: Ready for Implementation
> Dependencies: SDTW Message Queue, Zero-Trust Middleware, Policy Engine

---

## Overview

Implement Squid/Nginx egress proxy support for ArmorClaw containers to enable SDTW adapters to make HTTPS requests to external platforms (Slack, Discord, Teams, WhatsApp APIs).

---

## Phase 1: Proxy Infrastructure (Week 1)

### Step 1: Add Squid Proxy Container
**File:** `deploy/squid-compose.yml` (new)
**Purpose:** HTTP/HTTPS proxy for Docker containers
**Components:**
- Squid container (ephemeral)
- Nginx configuration (reverse proxy)
- Docker network bridge network

**Configuration:**
```toml
[squid]
enabled = true
image = "squid:latest"
network_mode = "bridge"  # bridge, host, none

[network.bridge]
driver = "bridge"
enabled = true

[proxy.egress]
# Map container hostname to proxy
container1 = "sdtw-slack"
container2 = "sdtw-discord"
container3 = "sdtw-teams"
container4 = "sdtw-whatsapp"
```

**Implementation Steps:**
1. Create Squid Docker Compose file
2. Add Docker network `bridge`
3. Configure proxy environment variables
4. Update container entrypoint to use HTTP_PROXY
5. Test connectivity from container

### Step 2: Add Proxy Support to Bridge
**File:** `bridge/pkg/proxy/proxy.go` (new)
**Purpose:** HTTP client that supports proxy via environment variable
**Implementation:**
```go
package proxy

import (
    "context"
    "io"
    "net"
    "net/http"
    "os"
    "strings"
)

// ProxyClient wraps HTTP Client with proxy support
type ProxyClient struct {
    client *http.Client
    proxyURL *url.URL // nil for direct connection
}

func NewProxyClient() *ProxyClient {
    return &ProxyClient{
        client: &http.Client{},
    }
}

func (pc *ProxyClient) Do(req *http.Request) (*http.Response, error) {
    // Check for proxy environment variable
    proxyURL := os.Getenv("HTTP_PROXY")
    if proxyURL != "" {
        proxy, err := url.Parse(proxyURL)
        if err != nil {
            return nil, err
        }

        req.URL = proxy
        req.Header.Set("Proxy-Connection", "close") // Keep connection alive
    }

    // Forward client request
    return pc.client.Do(req)
}

func (pc *ProxyClient) Get(url string) (*url.URL, error) {
    if pc.proxyURL != "" {
        proxy, err := url.Parse(pc.proxyURL)
        if err != nil {
            return nil, err
        }

        return url.Parse(proxyURL + url)
    }

    return pc.client.Get(url.String())
}
```

6. Update container entrypoint to pass environment variables:
```python
# In opt/openclaw/entrypoint.py, update run_agent_command()
def run_agent_command(cmd):
    if cmd in ['start', 'run']:
        env = os.environ.copy()
        # Add HTTP_PROXY to environment for SDTW adapters
        if 'HTTP_PROXY' not in os.environ:
            env['HTTP_PROXY'] = 'http://squid:3128:8080'
        # Set OPENAI_API_KEY, etc. from config for SDTW adapters
        env['OPENAI_API_KEY'] = os.getenv('OPENAI_API_KEY', '')
        # Pass all env vars to container
```

**Estimated Effort:** 3-5 days

### Step 3: Add Circuit Breaker State Persistence
**File:** `internal/queue/queue.go` (modify)
**Purpose:** Persist circuit breaker state to SQLite database
**Implementation:**
- Already implemented (see `queue_meta` table in database schema)
- Update `CircuitBreaker` struct to include `lastStateChange time.Time`
- Modify initialization to load state from database
- Add state change logging

**Estimated Effort:** 2 days

---

## Success Criteria

- Squid proxy container starts successfully
- SDTW adapters can reach external HTTPS APIs
- Circuit breaker state persists across restarts
- No secrets exposed in `/run/secrets/`
- Tests pass: proxy connectivity, state persistence

**Estimated Total Effort:** 1-2 weeks