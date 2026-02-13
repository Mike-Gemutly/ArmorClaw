# Robust Local Bridge Specification

**Date:** 2026-02-05
**Status:** Planning
**Version:** 1.0.0-robust
**Supersedes:** `2026-02-05-minimal-bridge-spec.md`

---

## Overview

The **Robust Local Bridge** is an enhanced version of the Minimal Bridge designed to support ArmorClaw's business model with premium features, industry compliance, and multi-protocol support.

### Core Principles

1. **Credential Isolation** - All API keys stored in encrypted keystore
2. **Modular Architecture** - Premium features as pluggable adapters
3. **Business Logic Integration** - License-gated features
4. **Offline Resilience** - Message queueing for network interruptions
5. **Compliance Support** - PII scrubbing, audit logging

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Robust Local Bridge                         │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    JSON-RPC Layer                           │    │
│  │  • Unix socket server (/run/armorclaw/bridge.sock)        │    │
│  │  • Request multiplexing                                     │    │
│  │  • Response routing                                         │    │
│  └──────────────────────────────┬──────────────────────────────┘    │
│                                 │                                  │
│  ┌──────────────────────────────┴──────────────────────────────┐    │
│  │                    Middleware Layer                         │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │    │
│  │  │ Validation  │  │ Queueing    │  │ License Gate        │  │    │
│  │  │             │  │             │  │                     │  │    │
│  │  │ • Regex     │  │ • Offline   │  │ • Feature unlock    │  │    │
│  │  │ • PII scrub │  │ • Buffer    │  │ • Tier enforcement  │  │    │
│  │  │ • Injection │  │ • Ordering  │  │ • Usage tracking    │  │    │
│  │  │   guard     │  │             │  │                     │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘  │    │
│  └──────────────────────────────┬──────────────────────────────┘    │
│                                 │                                  │
│  ┌──────────────────────────────┴──────────────────────────────┐    │
│  │                    Adapter Layer (Pluggable)                │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │    │
│  │  │ Matrix   │  │ Slack    │  │ Discord  │  │ License   │  │    │
│  │  │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Adapter   │  │    │
│  │  │          │  │          │  │          │  │           │  │    │
│  │  │ FREE      │  │ PRO      │  │ PRO      │  │ INTERNAL  │  │    │
│  │  └──────────┘  └──────────┘  └──────────┘  └────────────┘  │    │
│  │                                                               │    │
│  │  Future: WhatsApp, Teams, Salesforce, Jira, SAP             │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                 │                                  │
│  ┌──────────────────────────────┴──────────────────────────────┐    │
│  │                    Keystore Layer                           │    │
│  │  • Encrypted SQLite (SQLCipher)                             │    │
│  │  • Master key derivation (not ENV-only)                     │    │
│  │  • Per-credential AES-256 encryption                         │    │
│  │  • Automatic key rotation support                           │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Persistence Layer                         │    │
│  │  • Message queue (offline buffer)                           │    │
│  │  • Audit log (compliance)                                   │    │
│  │  • License cache (24-hour grace)                            │    │
│  └──────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## File Structure

```
bridge/
├── cmd/
│   └── armorclaw-bridge/
│       └── main.go                     # Entry point
├── internal/
│   ├── bridge/
│   │   ├── server.go                   # JSON-RPC server
│   │   ├── socket.go                   # Unix socket handler
│   │   └── router.go                   # Request routing
│   ├── keystore/
│   │   ├── keystore.go                 # Encrypted credential storage
│   │   ├── sqlcipher.go                # SQLCipher wrapper
│   │   └── master_key.go               # Key derivation
│   ├── adapter/
│   │   ├── adapter.go                  # Adapter interface
│   │   ├── matrix.go                   # Matrix adapter (FREE)
│   │   ├── slack.go                    # Slack adapter (PRO)
│   │   ├── discord.go                  # Discord adapter (PRO)
│   │   └── registry.go                 # Adapter registry
│   ├── middleware/
│   │   ├── validator.go                # Message validation
│   │   ├── pii_scrubber.go             # PII redaction
│   │   ├── injection_guard.go          # Prompt injection detection
│   │   ├── queue.go                    # Offline queueing
│   │   └── license_gate.go             # License checking
│   ├── license/
│   │   ├── client.go                   # License validation client
│   │   ├── cache.go                    # Offline license cache
│   │   └── types.go                    # License data structures
│   └── config/
│       ├── config.go                   # Configuration loader
│       └── defaults.go
├── pkg/
│   └── api/
│       └── v1/
│           └── bridge.proto           # gRPC definitions (optional)
├── configs/
│   ├── bridge.toml                     # Main configuration
│   ├── adapters.toml                   # Adapter configurations
│   └── armorclaw-bridge.service       # systemd service
├── scripts/
│   ├── install.sh                      # Installation script
│   ├── setup-keystore.sh               # Initial keystore setup
│   └── migrate.sh                      # Data migration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core Interfaces

### Adapter Interface

```go
package adapter

import (
    "context"
    "time"
)

// Message is the universal message format
type Message struct {
    ID        string                 `json:"id"`
    Timestamp time.Time              `json:"timestamp"`
    Protocol  string                 `json:"protocol"`  // "matrix", "slack", etc.
    Target    string                 `json:"target"`    // Room ID, channel, etc.
    Sender    string                 `json:"sender"`
    Content   map[string]interface{} `json:"content"`
    Metadata  map[string]string      `json:"metadata"`
}

// Response is the universal response format
type Response struct {
    MessageID string            `json:"message_id"`
    Status    string            `json:"status"`    // "sent", "queued", "failed"
    Error     error             `json:"error,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}

// Adapter defines the interface for protocol adapters
type Adapter interface {
    // Name returns the adapter name
    Name() string

    // Protocol returns the protocol identifier
    Protocol() string

    // Initialize the adapter with credentials
    Init(creds map[string]string) error

    // Send a message through this adapter
    Send(ctx context.Context, msg Message) (Response, error)

    // Receive returns a channel of incoming messages
    Receive(ctx context.Context) (<-chan Message, error)

    // Status returns adapter health
    Status() Status

    // Close cleanup resources
    Close() error
}

// Status represents adapter health
type Status struct {
    Connected   bool      `json:"connected"`
    Healthy     bool      `json:"healthy"`
    LastError   error     `json:"last_error,omitempty"`
    LastActivity time.Time `json:"last_activity"`
}
```

### Validator Interface

```go
package middleware

import "armorclaw/bridge/internal/adapter"

// Validator checks and transforms messages
type Validator interface {
    // Validate checks if a message is acceptable
    Validate(msg *adapter.Message) error

    // Transform optionally modifies the message
    Transform(msg *adapter.Message) (*adapter.Message, error)
}

// PIIScrubber redacts personally identifiable information
type PIIScrubber struct {
    // Patterns for PII detection
    emailPattern    *regexp.Regexp
    phonePattern    *regexp.Regexp
    ssnPattern      *regexp.Regexp
    creditPattern   *regexp.Regexp
}

// PromptInjectionGuard detects prompt injection attempts
type PromptInjectionGuard struct {
    // Suspicious patterns
    injectionPatterns []string
    maxLength        int
}

// QueueManager handles offline message buffering
type QueueManager struct {
    storage    QueueStorage
    maxSize    int64
    flushInterval time.Duration
}

// LicenseGate checks feature availability
type LicenseGate struct {
    client  *license.Client
    cache   *license.Cache
}
```

---

## Configuration

### bridge.toml

```toml
# /etc/armorclaw/bridge.toml

[bridge]
# Identity
name = "armorclaw-bridge"
version = "1.0.0"

# Networking
socket_path = "/run/armorclaw/bridge.sock"
socket_permissions = "0660"

# Process
user = "armorclaw"
group = "armorclaw"

# Logging
log_level = "info"           # debug, info, warn, error
log_file = "/var/log/armorclaw/bridge.log"
log_max_size = 100           # MB
log_max_backups = 3
log_max_age = 30             # days

# Performance
max_concurrent_requests = 100
request_timeout = "30s"
queue_max_size = 10000
queue_flush_interval = "5s"

[keystore]
# Encrypted credential storage
path = "/var/lib/armorclaw/keystore.db"

# Master key derivation (preferred over ENV)
# Uses system-specific values to derive encryption key
derivation_method = "system"  # "system", "env", "file"
# If "env", requires ARMORCLAW_MASTER_KEY environment variable
# If "file", reads from specified path
# If "system", derives from hostname + MAC + user secret

# Key rotation
auto_rotate = true
rotate_after_days = 90
keep_old_keys = 1

[license]
# License server configuration
enabled = true
endpoint = "https://api.armorclaw.com/v1/licenses"
cache_ttl = "24h"
grace_period = "72h"  # Allow offline operation
instance_id = "auto"   # Auto-generate if "auto"

[adapters]
# Available protocol adapters
enabled = ["matrix"]

# Adapter-specific configurations
[adapters.matrix]
homeserver_url = "http://localhost:6167"
device_id = "ARMORCLAW_BRIDGE"
sync_timeout = "30s"
retry_limit = 3
retry_delay = "5s"

[adapters.slack]
# PRO feature - requires license
enabled = false

[adapters.discord]
# PRO feature - requires license
enabled = false

[middleware]
# Message validation and transformation

[middleware.validator]
enabled = true
max_message_size = 1048576  # 1 MB
max_message_length = 10000   # characters

[middleware.pii_scrubber]
enabled = false  # PRO feature
redaction_char = "*"
log_redactions = true

[middleware.injection_guard]
enabled = true
log_attempts = true
block_on_detection = true
max_violations = 10
block_duration = "5m"

[middleware.queue]
enabled = true
storage_path = "/var/lib/armorclaw/queue.db"
max_memory_bytes = 104857600  # 100 MB
max_disk_bytes = 1073741824   # 1 GB
persist_to_disk = true

[audit]
# Compliance and security logging
enabled = false  # PRO feature
log_path = "/var/log/armorclaw/audit.log"
include_payloads = false
include_headers = true
retention_days = 90
```

---

## Keystore Implementation

### SQLCipher Schema

```sql
-- keystore.db schema

CREATE TABLE credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    adapter TEXT NOT NULL,
    type TEXT NOT NULL,  -- "token", "api_key", "password", etc.
    account TEXT,        -- e.g., "@user:server.com"
    encrypted_value BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    expires_at INTEGER,
    tags TEXT            -- JSON array of tags
);

CREATE TABLE encryption_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key_id TEXT NOT NULL UNIQUE,
    encrypted_key BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_credentials_name ON credentials(name);
CREATE INDEX idx_credentials_adapter ON credentials(adapter);
CREATE INDEX idx_credentials_tags ON credentials(tags);

-- Master key is derived, not stored
-- Each credential is encrypted with a unique data encryption key
-- Data keys are encrypted with the master key
```

### Key Derivation (Preferred Approach)

```go
package keystore

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "net"
    "os"
    "runtime"
    "golang.org/x/crypto/pbkdf2"
)

// MasterKeyDerivation derives the master key from system properties
func MasterKeyDerivation(userSecret string) ([]byte, error) {
    // Collect system-specific entropy
    hostname, _ := os.Hostname()

    // Get first MAC address
    interfaces, _ := net.Interfaces()
    mac := ""
    for _, iface := range interfaces {
        if len(iface.HardwareAddr) > 0 {
            mac = iface.HardwareAddr.String()
            break
        }
    }

    // OS and arch
    osArch := runtime.GOOS + "/" + runtime.GOARCH

    // Combine entropy sources
    salt := hostname + mac + osArch + userSecret

    // Derive 256-bit key using PBKDF2
    // Using a fixed iteration count for reproducibility
    key := pbkdf2.Key([]byte(salt), []byte("ArmorClawBridge"), 100000, 32, sha256.New)

    return key, nil
}

// GenerateMasterKey creates a new random master key (for setup)
func GenerateMasterKey() ([]byte, error) {
    key := make([]byte, 32)
    _, err := rand.Read(key)
    return key, err
}

// MasterKeyToString converts key to hex for backup/export
func MasterKeyToString(key []byte) string {
    return hex.EncodeToString(key)
}

// MasterKeyFromString parses hex string to key
func MasterKeyFromString(s string) ([]byte, error) {
    return hex.DecodeString(s)
}
```

---

## License System

### License Client with Caching

```go
package license

import (
    "encoding/json"
    "sync"
    "time"
)

type Client struct {
    endpoint   string
    instanceID string
    httpClient *http.Client
    cache      *Cache
    cacheMutex sync.RWMutex
}

type Cache struct {
    licenses map[string]*CachedLicense
    mutex     sync.RWMutex
}

type CachedLicense struct {
    Valid      bool
    Features   []string
    Tier       string
    ExpiresAt  time.Time
    CachedAt   time.Time
    GraceUntil time.Time  // For offline operation
}

func (c *Client) Validate(feature string) (bool, error) {
    // Check cache first
    if cached := c.cache.Get(feature); cached != nil {
        if cached.IsValid() {
            return true, nil
        }
        // Within grace period?
        if time.Now().Before(cached.GraceUntil) {
            return cached.Valid, nil
        }
    }

    // Fresh check from server
    result, err := c.checkServer(feature)
    if err != nil {
        // Server error - use cached value if in grace period
        if cached := c.cache.Get(feature); cached != nil {
            if time.Now().Before(cached.GraceUntil) {
                return cached.Valid, nil
            }
        }
        return false, err
    }

    // Update cache
    c.cache.Set(feature, &CachedLicense{
        Valid:      result.Valid,
        Features:   result.Features,
        Tier:       result.Tier,
        ExpiresAt:  time.Now().Add(24 * time.Hour),
        CachedAt:   time.Now(),
        GraceUntil: time.Now().Add(72 * time.Hour),  // 3-day grace
    })

    return result.Valid, nil
}

func (c *Cache) Get(feature string) *CachedLicense {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    return c.licenses[feature]
}

func (c *CachedLicense) IsValid() bool {
    return c.Valid && time.Now().Before(c.ExpiresAt)
}
```

---

## Memory Usage Optimization

### Memory Budget Compliance

```go
package bridge

import (
    "runtime"
    "time"
)

// MemoryMonitor ensures we stay within budget
type MemoryMonitor struct {
    maxMemory    uint64  // bytes
    checkInterval time.Duration
}

func (m *MemoryMonitor) Start() {
    ticker := time.NewTicker(m.checkInterval)
    go func() {
        for range ticker.C {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)

            if m.Alloc > m.maxMemory {
                log.Printf("WARNING: Memory usage %.2f MB exceeds budget %.2f MB",
                    float64(m.Alloc)/1024/1024,
                    float64(m.maxMemory)/1024/1024)

                // Trigger memory reduction
                m.reduceMemory()
            }
        }
    }()
}

func (m *MemoryMonitor) reduceMemory() {
    // Clear message queues if needed
    // Compact offline queue
    // Trigger GC
    runtime.GC()
    runtime.GC()  // Twice for effectiveness
}
```

---

## JSON-RPC API (Extended)

### Core Methods (Free)

```json
// Send message (adapter-agnostic)
{
  "jsonrpc": "2.0",
  "method": "bridge.send",
  "params": {
    "protocol": "matrix",
    "target": "!room:id:server.com",
    "content": {"msgtype": "m.text", "body": "Hello"}
  },
  "id": 1
}

// Receive messages
{
  "jsonrpc": "2.0",
  "method": "bridge.receive",
  "params": {"timeout": 30},
  "id": 2
}

// List available adapters
{
  "jsonrpc": "2.0",
  "method": "bridge.adapters",
  "params": {},
  "id": 3
}

// Bridge status
{
  "jsonrpc": "2.0",
  "method": "bridge.status",
  "params": {},
  "id": 4
}
```

### Premium Methods (License-Gated)

```json
// Enable premium adapter
{
  "jsonrpc": "2.0",
  "method": "bridge.enable_adapter",
  "params": {
    "adapter": "slack",
    "credentials": {...}
  },
  "id": 10
}

// PII scrubbing (PRO)
{
  "jsonrpc": "2.0",
  "method": "bridge.scrub_pii",
  "params": {
    "text": "Call me at 555-123-4567"
  },
  "id": 11
}

// Audit log query (Enterprise)
{
  "jsonrpc": "2.0",
  "method": "audit.query",
  "params": {
    "start": "2026-02-01T00:00:00Z",
    "end": "2026-02-05T23:59:59Z"
  },
  "id": 12
}
```

---

## Implementation Phases

### Phase 1: Core Bridge (Week 1-2)
- [ ] Basic JSON-RPC server
- [ ] Encrypted keystore
- [ ] Matrix adapter
- [ ] Unix socket communication

### Phase 2: Middleware (Week 3)
- [ ] Validator framework
- [ ] Prompt injection guard
- [ ] Offline queueing
- [ ] Message persistence

### Phase 3: License System (Week 4)
- [ ] License client
- [ ] Offline caching
- [ ] Feature gating
- [ ] Grace period handling

### Phase 4: Premium Adapters (Week 5-6)
- [ ] Slack adapter
- [ ] Discord adapter
- [ ] PII scrubber
- [ ] Audit logging

### Phase 5: Testing & Docs (Week 7-8)
- [ ] Unit tests
- [ ] Integration tests
- [ ] Performance benchmarks
- [ ] User documentation

---

## Deployment

### Installation Script

```bash
#!/bin/bash
# scripts/install.sh

set -e

echo "Installing ArmorClaw Robust Bridge..."

# Create user
sudo useradd --system \
  --user-group \
  --home-dir /var/lib/armorclaw \
  --shell /bin/bash \
  armorclaw

# Create directories
sudo mkdir -p /etc/armorclaw
sudo mkdir -p /var/lib/armorclaw
sudo mkdir -p /var/log/armorclaw
sudo mkdir -p /run/armorclaw

# Set permissions
sudo chown -R armorclaw:armorclaw /var/lib/armorclaw
sudo chown -R armorclaw:armorclaw /var/log/armorclaw
sudo chown -R armorclaw:armorclaw /run/armorclaw
sudo chmod 750 /var/lib/armorclaw
sudo chmod 750 /var/log/armorclaw

# Install binary
sudo cp armorclaw-bridge /usr/local/bin/
sudo chmod +x /usr/local/bin/armorclaw-bridge

# Install configs
sudo cp configs/bridge.toml /etc/armorclaw/
sudo chown armorclaw:armorclaw /etc/armorclaw/bridge.toml
sudo chmod 640 /etc/armorclaw/bridge.toml

# Install systemd service
sudo cp configs/armorclaw-bridge.service /etc/systemd/system/
sudo systemctl daemon-reload

echo "Installation complete!"
echo "Run: sudo systemctl enable armorclaw-bridge"
echo "Then: sudo ./scripts/setup-keystore.sh"
```

---

## Success Criteria

| Criteria | Target | Measurement |
|----------|--------|-------------|
| **Memory usage** | < 250 MB | `ps aux | grep armorclaw-bridge` |
| **Message latency** | < 50ms p95 | End-to-end timing |
| **Offline durability** | 100% | Queue survives restart |
| **License uptime** | > 99.9% | Grace period covers outages |
| **Concurrent agents** | 50+ | Load testing |

---

**Next:** Create detailed implementation tasks for Phase 1 (Core Bridge).
