# ArmorClaw System Documentation

> **Purpose**: LLM-readable comprehensive documentation for ArmorClaw architecture, components, APIs, and security.
>
> **Version**: 4.7.0
>
> **Last Updated**: 2026-04-04

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Deployment Skills for AI CLI Tools](#deployment-skills-for-ai-cli-tools)
3. [System Architecture](#system-architecture)
4. [Go Bridge Component](#go-bridge-component)
5. [SQLCipher Keystore](#sqlcipher-keystore)
6. [Rust Vault Sidecar](#rust-vault-sidecar)
7. [Matrix Conduit Control Plane](#matrix-conduit-control-plane)
8. [Security Architecture](#security-architecture)
9. [Component Integration Patterns](#component-integration-patterns)
10. [Agent Studio](#agent-studio)
11. [Browser Service](#browser-service)
12. [Rust Office Sidecar](#rust-office-sidecar)
13. [ArmorChat Android Client](#armorchat-android-client)
14. [OpenClaw Agent Runtime](#openclaw-agent-runtime)
15. [RPC API Reference](#rpc-api-reference)
16. [Event Types Reference](#event-types-reference)
17. [Configuration Reference](#configuration-reference)
18. [Deployment Modes](#deployment-modes)
19. [Testing & Verification](#testing--verification)

---

## Executive Summary

### What is ArmorClaw?

ArmorClaw is a **VPS-based AI secretary platform** that runs AI agents 24/7 on your server, controlled from your phone. It enables automated web browsing, form filling, and task management with **human-in-the-loop approval** for sensitive operations.

### Core Value Proposition

**Problem**: Traditional AI agents see your passwords and credit cards when you give them access to perform tasks.

**Solution**: ArmorClaw's **BlindFill™** technology injects secrets directly into the browser. The agent requests "credit card" but never sees the actual number—it goes straight from encrypted storage to the form field.

### Key Features

| Feature | Description |
|---------|-------------|
| **BlindFill™** | Memory-only secret injection, agents never see raw values |
| **E2EE Messaging** | All communication via Matrix protocol with Megolm encryption |
| **Container Isolation** | Each agent runs in hardened Docker container |
| **Human-in-the-Loop** | Mobile approval for sensitive operations (payments, PII) |
| **SQLCipher Keystore** | Hardware-bound encrypted credential storage |
| **No-Code Agent Studio** | Define agents via chat commands or dashboard |
| **21 Browser Skills** | Chrome DevTools MCP integration for web automation |
| **Sentinel Mode** | Automatic VPS deployment with Let's Encrypt TLS |

### Component Overview

| Component | Language | Purpose | Entry Point |
|-----------|----------|---------|-------------|
| **Go Bridge** | Go | Central orchestrator, RPC server, container manager | `bridge/cmd/bridge/main.go` |
| **SQLCipher Keystore** | Go | Encrypted credential storage with hardware binding | `bridge/pkg/keystore/keystore.go` |
| **Matrix Conduit** | Rust | Matrix homeserver for E2EE messaging | Conduit binary |
| **Browser Service** | TypeScript | Playwright-based browser automation | `browser-service/src/index.ts` |
| **OpenClaw Runtime** | TypeScript/Node | AI agent runtime in containers | `container/openclaw-src/openclaw.mjs` |
| **License Server** | Go | Enterprise license validation | `license-server/main.go` |
| **ArmorChat** | Kotlin | Android mobile client | `applications/ArmorChat/` |

---

## Deployment Skills for AI CLI Tools

### Overview

ArmorClaw includes **built-in deployment skills** that let coding agents like Claude Code, OpenCode, Cursor, and Crush deploy and manage your VPS secretary platform.

All skills use **shell variable interpolation** (`${variable}`) for consistency across platforms.

### Available Skills

| Skill | Purpose | Command | Key Parameters |
|-------|---------|---------|----------------|
| **Deploy** | Deploy ArmorClaw to VPS | `/deploy vps_ip=...` | `vps_ip`, `ssh_key`, `domain`, `mode` |
| **Status** | Check deployment health | `/status vps_ip=...` | `vps_ip`, `ssh_key`, `domain`, `verbose` |
| **Cloudflare** | Configure HTTPS | `/cloudflare domain=...` | `domain`, `mode`, `cf_api_token` |
| **Provision** | Connect mobile device | `/provision vps_ip=...` | `vps_ip`, `expiry`, `show_url` |

### Automation Levels

| Level | Behavior | Examples |
|-------|----------|----------|
| `auto` | Execute immediately | Health checks, status, OS detection |
| `confirm` | Ask user before executing | SSH connection, running installer |
| `guide` | Provide instructions | Account creation, DNS setup |

### Skills Directory

```
.skills/
├── deploy.yaml          # Deployment skill definition
├── deploy/SKILL.md      # AI-friendly instructions
├── status.yaml          # Status check skill
├── status/SKILL.md      # Status documentation
├── cloudflare.yaml      # Cloudflare setup skill
├── cloudflare/SKILL.md  # Cloudflare guide
├── provision.yaml       # Mobile provisioning skill
├── provision/SKILL.md   # Provisioning guide
├── PLATFORM.md          # Cross-platform patterns
├── TEMPLATE.yaml        # Schema for new skills
└── README.md            # Skills index
```

---

## System Architecture

### High-Level Architecture Diagram

```
┌───────────────────────────────────────────────────────────────────────┐
│                         THE VPS (Office)                              │
│                                                                       │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐           │
│  │ ArmorClaw   │◀────▶│  OpenClaw   │◀────▶│  Playwright │           │
│  │ Bridge      │      │  (Agent)    │      │  (Browser)  │           │
│  │ (Orchestr.) │      │             │      │             │           │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘           │
│         │                    │  ▲                  │                   │
│         │                    │  │                  │                   │
│         │   BlindFill Engine │  │                  │                   │
│         │   (Memory-Only)    │  │                  │                   │
│         │                    │  │                  │                   │
│         │    ┌───────────────┘  │                  │                   │
│         │    │ Rust Vault Sidecar│                  │                   │
│         │    │ (gRPC/Unix Socket)│                  │                   │
│         │    │ - Zeroization     │                  │                   │
│         │    │ - Network BlindFill│                 │                   │
│         │    │ - Circuit Breaking │                 │                   │
│         │    └───────────────────┘                  │                   │
└─────────┼────────────────────┼─────────────────────┼───────────────────┘
          │                    │                     │
          │ Secure Matrix Tunnel (E2EE)             │
          │                    │                     │
┌─────────▼────────────────────▼─────────────────────▼───────────────────┐
│                         USER (Mobile)                                 │
│   ArmorChat App                                                      │
│   "Book a flight to NYC"  [Approve Credit Card] 🔐                   │
└───────────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
armorclaw-omo/
├── bridge/                    # Go Bridge orchestrator (60 packages)
│   ├── cmd/bridge/main.go    # Primary entry point (3,389 lines)
│   ├── pkg/                  # Public packages
│   │   ├── rpc/              # JSON-RPC 2.0 server (47 methods)
│   │   ├── keystore/         # SQLCipher encrypted storage
│   │   ├── pii/              # BlindFill engine
│   │   ├── studio/           # Agent container management
│   │   ├── eventbus/         # Event broadcasting
│   │   ├── matrix/           # Matrix client
│   │   ├── browser/          # Browser automation
│   │   ├── provisioning/     # Mobile provisioning
│   │   ├── trust/            # Zero-trust verification
│   │   ├── audit/            # Audit logging
│   │   └── ... (50 more)
│   └── internal/             # Internal implementation (19 packages)
│       ├── adapter/          # Matrix/Slack adapters
│       ├── ai/               # AI service
│       ├── skills/           # Built-in skills
│       └── agent/            # Agent runtime
│
├── browser-service/          # TypeScript/Playwright automation
│   └── src/                  # HTTP API + Playwright wrapper
│
├── container/openclaw-src/   # OpenClaw agent runtime
│   ├── extensions/           # 39 platform adapters
│   └── skills/               # Browser skills
│
├── applications/             # Client applications
│   ├── ArmorChat/           # Android Kotlin client
│   ├── ArmorTerminal/       # Terminal client
│   └── admin-panel/         # Web dashboard
│
├── deploy/                   # Deployment scripts (32 scripts)
│   └── install.sh           # One-command installer
│
├── .skills/                  # AI CLI deployment skills
│   ├── deploy.yaml
│   ├── status.yaml
│   ├── cloudflare.yaml
│   └── provision.yaml
│
└── tests/ssh/               # VPS testing suite (10 categories)
```

### Communication Patterns

| Pattern | Protocol | Purpose | Port/Path |
|---------|----------|---------|-----------|
| **Matrix Protocol** | Matrix Client-Server API v3 | E2EE messaging, control plane | 6167 |
| **JSON-RPC 2.0 (Native)** | Unix domain socket | Internal component communication | `/run/armorclaw/bridge.sock` |
| **JSON-RPC 2.0 (Sentinel)** | TCP | Public API access (via Caddy proxy) | `0.0.0.0:8080` |
| **Docker Socket** | Docker Engine API | Container lifecycle management | `/var/run/docker.sock` |
| **HTTP/WebSocket** | REST + WebSocket | Health checks, metrics, real-time events | 8080 |
| **WebRTC** | ICE/STUN/TURN | Voice/video calls | Dynamic |

---

## Go Bridge Component

### Purpose

The Go Bridge is the **central orchestrator** that coordinates between the host system and isolated AI agent containers. It provides:
- Secure credential management via SQLCipher
- JSON-RPC 2.0 API (47 methods across 8 domains)
- Matrix integration for encrypted messaging
- Browser automation job queue
- Skill execution with allowlist control
- PII approval workflow

### Main Structure

**File**: `bridge/pkg/rpc/server.go`

```go
type Server struct {
    // Core communication
    handlers map[string]HandlerFunc  // 47 registered methods
    socketPath string
    listener net.Listener
    
    // Rate limiting
    aiMaxConcurrent int           // Default: 4
    aiSemaphore chan struct{}
    heartbeats sync.Map           // UserID -> time.Time
    
    // Integration dependencies
    keystore        Keystore
    matrix          MatrixAdapter
    aiService       *ai.AIService
    bridgeMgr       BridgeManager
    browserJobs     *BrowserJobManager
    studio          StudioService
    appService      AppService
    provisioningMgr ProvisioningManager
    skillMgr        SkillManager
    skillGate       interfaces.SkillGate
    eventBus        *eventbus.EventBus
    hardeningStore  trust.Store
    metrics         *Metrics
}
```

### Package Index

#### Control Plane
| Package | Purpose |
|---------|---------|
| `pkg/rpc/` | JSON-RPC 2.0 server with all method handlers |
| `pkg/eventbus/` | Event broadcasting to WebSocket clients |
| `pkg/config/` | TOML configuration management |
| `pkg/logger/` | Structured logging |

#### AI & Skills
| Package | Purpose |
|---------|---------|
| `internal/ai/` | AI provider clients (OpenAI, Anthropic, OpenRouter, etc.) |
| `internal/skills/` | Built-in skills (web_search, calendar, email, data_analyze) |
| `pkg/skills/` | Skill registry and management |
| `pkg/interfaces/skillgate.go` | PII interception interface |

#### Security & Trust
| Package | Purpose |
|---------|---------|
| `pkg/pii/` | BlindFill engine for secure PII injection |
| `pkg/keystore/` | SQLCipher encrypted credential storage |
| `pkg/trust/` | Zero-trust device verification |
| `pkg/security/` | Website guard and security policies |
| `pkg/enforcement/` | License validation and enforcement |
| `pkg/lockdown/` | Admin reset mode |

#### Communication
| Package | Purpose |
|---------|---------|
| `internal/adapter/` | Matrix, Slack adapters (messaging platforms) |
| `pkg/matrix/` | Matrix client library |
| `pkg/appservice/` | Matrix AppService bridges |
| `pkg/provisioning/` | Mobile device provisioning via QR |
| `pkg/ghost/` | Ghost user management |

#### Container & Runtime
| Package | Purpose |
|---------|---------|
| `pkg/studio/` | Agent container lifecycle (Docker) |
| `pkg/browser/` | Browser automation interface |
| `pkg/queue/` | Job queue for browser tasks |
| `pkg/docker/` | Docker client wrapper |
| `internal/agent/` | Agent runtime state machine |
| `internal/executor/` | Task execution engine |

#### Observability & Governance
| Package | Purpose |
|---------|---------|
| `pkg/audit/` | Critical operation audit logging |
| `pkg/budget/` | AI spend budget tracking |
| `pkg/governor/` | Rate limiting and throttling |
| `pkg/metrics/` | Metrics collection |

### Initialization Flow

**CLI Commands:**
```
init              → Generate example config
validate          → Validate configuration
setup             → Interactive setup wizard
daemon            → Daemon management (start/stop/restart/status)
add-key           → Add API key to keystore
generate-qr       → Generate QR for mobile app
(no command)      → Start bridge server
```

**Server Initialization Sequence:**
1. Parse CLI flags and load configuration
2. Setup logging from config
3. Pre-flight Docker availability check
4. Create runtime directory (`/run/armorclaw/`)
5. Initialize encrypted keystore (with recovery for corruption)
6. Initialize audit logger
7. Create Matrix adapter (if enabled)
8. Initialize AI service with keystore
9. Initialize event bus
10. Initialize browser job manager
11. Initialize studio (agent factory)
12. Initialize provisioning manager
13. Initialize skill manager
14. Register RPC handlers
15. Start RPC server (Unix socket or TCP)
16. Start event bus broadcaster
17. Start Matrix sync loop
18. Wait for shutdown signal
19. Graceful shutdown

### Key Interfaces

```go
// Bridge management
type BridgeManager interface {
    Start() error
    Stop() error
    RegisterAdapter(platform, adapter) error
    BridgeChannel(roomID, platform, channelID) error
    GetBridgedChannels() []*BridgedChannel
}

// PII interception
type SkillGate interface {
    InterceptToolCall(ctx, call) (*ToolCall, error)
    InterceptPrompt(ctx, prompt) (string, *PIIMapping, error)
    RestoreOutput(ctx, output, mapping) (string, error)
    ValidateArgs(ctx, toolName, args) ([]PIIViolation, error)
}

// Matrix communication
type MatrixAdapter interface {
    SendMessage(roomID, message, msgType) (string, error)
    SendEvent(roomID, eventType, content) error
    Login(username, password) error
    JoinRoom(ctx, roomIDOrAlias, viaServers, reason) (string, error)
    GetUserID() string
    IsLoggedIn() bool
}
```

---

## SQLCipher Keystore

### Purpose

The keystore provides **zero-knowledge encrypted credential storage** using SQLCipher with hardware-bound master keys. It enables:
- Secure API key storage (never persisted to disk as plaintext)
- BlindFill™ secret injection (agents never see raw values)
- Hardware binding (database useless if stolen)
- Zero-touch reboot (no password required)

### Database Schema

**Database Path**: `/var/lib/armorclaw/keystore.db` (encrypted)
**Encryption**: SQLCipher with XChaCha20-Poly1305 AEAD

```sql
-- API Credentials
CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,                    -- openai, anthropic, cloudflare, etc.
    token_encrypted BLOB NOT NULL,             -- XChaCha20-Poly1305 encrypted
    nonce BLOB NOT NULL,                       -- AEAD nonce
    base_url TEXT,                             -- Custom endpoint
    display_name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER,                        -- Token expiry (optional)
    tags TEXT                                  -- JSON array
);

CREATE INDEX idx_provider ON credentials(provider);
CREATE INDEX idx_expires_at ON credentials(expires_at);

-- User Profiles (BlindFill PII)
CREATE TABLE user_profiles (
    id TEXT PRIMARY KEY,
    profile_name TEXT NOT NULL,
    profile_type TEXT NOT NULL DEFAULT 'personal',
    data_encrypted BLOB NOT NULL,              -- JSON-serialized PII (encrypted)
    data_nonce BLOB NOT NULL,
    field_schema TEXT NOT NULL,                -- JSON schema of fields
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_accessed INTEGER,
    is_default INTEGER DEFAULT 0
);

CREATE INDEX idx_profile_type ON user_profiles(profile_type);
CREATE INDEX idx_profile_default ON user_profiles(is_default);

-- Matrix Refresh Tokens
CREATE TABLE matrix_refresh_tokens (
    id TEXT PRIMARY KEY,
    token_encrypted BLOB NOT NULL,
    nonce BLOB NOT NULL,
    homeserver_url TEXT NOT NULL,
    user_id TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

-- Hardening State
CREATE TABLE hardening_state (
    user_id TEXT PRIMARY KEY,
    password_rotated INTEGER DEFAULT 0,
    bootstrap_wiped INTEGER DEFAULT 0,
    device_verified INTEGER DEFAULT 0,
    recovery_backed_up INTEGER DEFAULT 0,
    biometrics_enabled INTEGER DEFAULT 0,
    delegation_ready INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Hardware Binding
CREATE TABLE hardware_binding (
    signature_hash TEXT PRIMARY KEY,
    bound_at INTEGER NOT NULL,
    entropy_sources TEXT NOT NULL             -- JSON of sources used
);

-- Metadata
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

### Key Derivation Hierarchy

**Priority order for master key source:**
1. `ARMORCLAW_KEYSTORE_SECRET` environment variable (base64-encoded 32 bytes)
2. `keystore.db.key` file (base64-encoded)
3. Container-persisted random key
4. Hardware-derived key (fallback)

**Hardware Entropy Sources:**
```go
// CollectEntropy() gathers from:
1. /etc/machine-id, /var/lib/dbus/machine-id
2. /sys/class/dmi/id/product_uuid (SMBIOS)
3. Primary MAC address (first non-loopback)
4. Hostname
5. OS/Architecture (runtime.GOOS, runtime.GOARCH)
6. /proc/cpuinfo (model name, vendor_id)
```

### Encryption Configuration

```go
const (
    saltLength       = 32
    pbkdf2Iterations = 256000  // SQLCipher default
    keyLength        = 32
    cipherPageSize   = 4096
    cipherKdfIter    = 256000
    cipherHmacAlg    = "HMAC_SHA512"
    cipherKdfAlgorithm = "PBKDF2_HMAC_SHA512"
)
```

**Connection String:**
```
file:keystore.db?_pragma_key=x'hex_master_key'&_pragma_cipher_page_size=4096&_pragma_kdf_iter=256000&_pragma_cipher_hmac_algorithm=HMAC_SHA512&_pragma_cipher_kdf_algorithm=PBKDF2_HMAC_SHA512&_foreign_keys=ON
```

### Supported Providers

```go
const (
    ProviderOpenAI     Provider = "openai"
    ProviderAnthropic  Provider = "anthropic"
    ProviderCloudflare Provider = "cloudflare"
    ProviderDeepSeek   Provider = "deepseek"
    ProviderGoogle     Provider = "google"
    ProviderGroq       Provider = "groq"
    ProviderMoonshot   Provider = "moonshot"
    ProviderNvidia     Provider = "nvidia"
    ProviderOllama     Provider = "ollama"
    ProviderOpenRouter Provider = "openrouter"
    ProviderXAI        Provider = "xai"
    ProviderZhipu      Provider = "zhipu"
)
```

### Environment Fallback

`Retrieve()` checks environment variables first:
- `OPENROUTER_API_KEY`
- `ZAI_API_KEY`
- `OPEN_AI_KEY`

---

## Rust Vault Sidecar

### Purpose

The Rust Vault is a **security-hardened cryptographic enclave** that provides heavy I/O operations for ArmorClaw with enhanced security features. It implements:

- **State Bifurcation** - Separate persistent secrets (vault.db) from ephemeral crypto state (matrix_state.db)
- **Network-Layer BlindFill** - Inject secrets at network layer via Chrome DevTools Protocol
- **gRPC Circuit Breaking** - Rate limit and protect against DoS attacks
- **Zeroization** - All secrets zeroized in memory after use
- **mTLS Authentication** - gRPC over Unix domain sockets with certificate validation

### Architecture

```
┌───────────────────────────────────────────────────────────────────────┐
│                         THE VPS (Office)                              │
│                                                                       │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐           │
│  │ ArmorClaw   │◀────▶│  Rust Vault   │◀────▶│  Playwright │           │
│  │ Bridge      │ gRPC │  (Sidecar)    │ CDP  │  (Browser)  │           │
│  │ (Orchestr.) │ Unix │             │      │             │           │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘           │
│         │                    │                     │                   │
│         │                    │                     │                   │
│         │                    │   BlindFill Engine │                   │
│         │                    │   (Memory-Only)    │                   │
│         │                    │                     │                   │
└─────────┼────────────────────┼─────────────────────┼───────────────────┘
          │                    │                     │
          │                    │                     │
          │ Secure Matrix Tunnel (E2EE)             │
          │                    │                     │
┌─────────▼────────────────────▼─────────────────────▼───────────────────┐
│                         USER (Mobile)                                 │
│   ArmorChat App                                                      │
│   "Book a flight to NYC"  [Approve Credit Card] 🔐                   │
└───────────────────────────────────────────────────────────────────────┘
```

### Integration with ArmorClaw

**Go Bridge → Rust Vault:**
- gRPC over Unix Domain Socket (`/run/armorclaw/rust-vault.sock`)
- mTLS authentication with certificate validation
- Keystore proxy API for secret retrieval
- Rate limiting (100 requests/second) with atomic operations
- Concurrency limiting (10 concurrent requests)

**Rust Vault → Playwright/Browser:**
- Chrome DevTools Protocol (CDP) interception
- Filters XHR and Fetch requests only (not wildcard)
- Placeholder resolution: `{{secret:payment.card_number}}`
- Network-layer injection (secrets never reach agent)

**Security Features:**

1. **State Bifurcation**
   - `vault.db` - Persistent secrets (SQLCipher encrypted)
   - `matrix_state.db` - Ephemeral crypto state (SQLCipher encrypted)
   - Separate databases prevent cross-contamination

2. **Network-Layer BlindFill**
   - CDP interceptor filters by resourceType (XHR, Fetch only)
   - Placeholder format: `{{secret:name}}` (flat lookups only)
   - Secrets injected at network layer, never accessible to agent
   - Zeroized immediately after request completes

3. **gRPC Security**
   - Unix domain socket with 0600 permissions
   - mTLS authentication (certificate validation)
   - Rate limiting: 100 req/s with atomic operations (no mutex)
   - Concurrency limiting: 10 concurrent requests with semaphore

4. **Memory Safety**
   - All secrets use `Zeroizing<String>` from zeroize crate
   - Secrets zeroized on drop
   - No secret caching beyond request lifecycle
   - No secret values in logs

5. **Key Derivation**
   - PBKDF2-HMAC-SHA512 with 256,000 iterations
   - 32-byte salt for each database
   - Compatible with Go Bridge implementation

6. **SQLCipher Configuration**
   - `cipher_plaintext_header_size=32` for performance
   - `synchronous=NORMAL` for durability
   - Separate encryption keys for vault.db and matrix_state.db

7. **Logging**
   - Basic logging only (no comprehensive observability)
   - No secret values in logs
   - No circuit breakers or advanced retry logic

### Configuration

**Environment Variables:**

```bash
# Rust Vault Configuration
RUST_VAULT_ENABLED=true
RUST_VAULT_SOCKET_PATH=/run/armorclaw/rust-vault.sock
RUST_VAULT_TLS_ENABLED=true
RUST_VAULT_TLS_CERT_PATH=/etc/armorclaw/rust-vault.crt
RUST_VAULT_TLS_KEY_PATH=/etc/armorclaw/rust-vault.key
RUST_VAULT_TLS_CA_PATH=/etc/armorclaw/ca.crt

# Rate Limiting
RUST_VAULT_RATE_LIMIT=100              # Requests per second
RUST_VAULT_BURST_SIZE=20               # Burst capacity

# Concurrency
RUST_VAULT_MAX_CONCURRENT=10           # Max concurrent requests

# BlindFill
RUST_VAULT_CDP_ENABLED=true            # Enable CDP interception
```

**Default Configuration:**

```rust
pub struct VaultConfig {
    // Socket Configuration
    pub keystore_socket_path: PathBuf,
    pub use_tls: bool,
    pub tls: Option<TlsConfig>,
    
    // Rate Limiting
    pub rate_limit: u32,           // Default: 100
    pub burst_size: u32,           // Default: 20
    
    // Concurrency
    pub max_concurrent: usize,     // Default: 10
    
    // BlindFill
    pub cdp_enabled: bool,         // Default: true
}
```

### API Reference

**gRPC Methods (via Unix Socket):**

```protobuf
service Keystore {
    // Secret Management
    rpc StoreSecret(StoreSecretRequest) returns (StoreSecretResponse);
    rpc RetrieveSecret(RetrieveSecretRequest) returns (RetrieveSecretResponse);
    rpc DeleteSecret(DeleteSecretRequest) returns (DeleteSecretResponse);
    rpc ListSecrets(ListSecretsRequest) returns (ListSecretsResponse);
    
    // Matrix State
    rpc StoreMatrixState(StoreMatrixStateRequest) returns (StoreMatrixStateResponse);
    rpc RetrieveMatrixState(RetrieveMatrixStateRequest) returns (RetrieveMatrixStateResponse);
}
```

**CDP Interception:**

```json
{
  "method": "Fetch.enable",
  "params": {
    "patterns": [
      {
        "urlPattern": "*",
        "resourceType": "XHR",
        "requestStage": "Request"
      },
      {
        "urlPattern": "*",
        "resourceType": "Fetch",
        "requestStage": "Request"
      }
    ]
  }
}
```

### Testing

**Test Coverage: 118 tests across 13 test files**

- **Config Tests** (5) - Configuration validation
- **Error Tests** (15) - Error handling
- **DB Pool Tests** (5) - SQLCipher connection pooling
- **Vault Tests** (7) - Secret storage and zeroization
- **Matrix State Tests** (5) - Ephemeral state management
- **Placeholder Tests** (34) - Placeholder parsing and resolution
- **CDP Interceptor Tests** (6) - Network-layer filtering
- **BlindFill Integration Tests** (4) - End-to-end secret injection
- **gRPC Server Tests** (4) - Unix socket and permissions
- **mTLS Auth Tests** (10) - Certificate validation
- **Integration Tests** (1) - Project compilation
- **Doc Tests** (1) - Documentation examples

**Run Tests:**

```bash
cd rust-vault
cargo test --all
cargo clippy -- -D warnings
```

### Security Considerations

**Guardrails Respected:**

- ✅ No wildcard URL patterns (resourceType filtering instead)
- ✅ No WebSocket interception
- ✅ No document.write() or innerHTML interception
- ✅ No comprehensive observability (basic logging only)
- ✅ No circuit breakers or advanced retry logic
- ✅ No secret caching beyond request lifecycle
- ✅ No secret values in logs
- ✅ No advanced placeholder features (conditionals, loops, nesting)

**Production Checklist:**

- [ ] Generate TLS certificates for mTLS
- [ ] Set Unix socket permissions to 0600
- [ ] Configure SQLCipher encryption keys
- [ ] Enable rate limiting and concurrency limits
- [ ] Test CDP interception with real browser
- [ ] Verify zeroization in memory dumps
- [ ] Audit logs for secret exposure

### Performance Characteristics

- **Memory**: ~2MB bounded for download streams
- **Rate Limiting**: 100 req/s with atomic operations
- **Concurrency**: 10 concurrent requests with semaphore
- **Key Derivation**: 256,000 iterations (compatible with Go Bridge)
- **Zeroization**: Immediate on drop, no caching
- **Socket**: Unix domain socket (0600 permissions)

### Troubleshooting

**Common Issues:**

1. **Socket Permission Denied**
   ```bash
   ls -la /run/armorclaw/rust-vault.sock
   # Should show: srw------- 1 root root 0 ... rust-vault.sock
   chmod 0600 /run/armorclaw/rust-vault.sock
   ```

2. **mTLS Authentication Failed**
   ```bash
   # Verify certificates exist
   ls -la /etc/armorclaw/rust-vault.{crt,key} /etc/armorclaw/ca.crt
   
   # Check certificate expiry
   openssl x509 -in /etc/armorclaw/rust-vault.crt -text -noout | grep "Not After"
   ```

3. **SQLCipher Key Derivation Mismatch**
   ```bash
   # Ensure PBKDF2-HMAC-SHA512 with 256,000 iterations
   # Check Go Bridge compatibility
   grep -r "PBKDF2" bridge/pkg/keystore/
   ```

4. **CDP Interception Not Working**
   ```bash
   # Verify CDP is enabled
   curl http://localhost:9222/json/list
   
   # Check resourceType filtering
   # Should only intercept XHR and Fetch requests
   ```

---

## Matrix Conduit Control Plane

### Purpose

Matrix serves as the **primary control plane** for ArmorClaw, providing:
- End-to-end encrypted messaging
- Real-time event streaming
- Admin command processing
- Agent control commands
- Voice call signaling

### Event Types

**Standard Matrix Events:**
- `m.room.message` - Text messages and commands
- `m.room.member` - Membership changes
- `m.room.power_levels` - RBAC (admin=50)
- `m.typing` - Typing notifications
- `m.receipt` - Read receipts

**Voice Call Events:**
- `m.call.invite` - Call initiation
- `m.call.answer` - Call acceptance
- `m.call.hangup` - Call termination
- `m.call.candidates` - ICE candidates
- `m.call.negotiate` - SDP renegotiation

**Custom ArmorClaw Events:**
- `app.armorclaw.alert` - System alerts
- `app.armorclaw.pii_request` - PII access request
- `app.armorclaw.pii_response` - PII access response
- `app.armorclaw.consent.request` - Three-way consent request
- `app.armorclaw.consent.response` - Three-way consent response

### Control Plane Commands

#### Admin Commands (`/` prefix)

| Command | Description |
|---------|-------------|
| `/claim_admin [device_name]` | Claim admin rights (lockdown mode only) |
| `/status` | Show system status |
| `/verify <code>` | Verify challenge code |
| `/approve <claim_id>` | Approve admin claim |
| `/reject <claim_id> [reason]` | Reject claim |
| `/help` | Show available commands |

#### AI Management Commands (`/ai` prefix)

| Command | Description |
|---------|-------------|
| `/ai providers` | List available AI providers |
| `/ai models <provider>` | List models for provider |
| `/ai switch <provider> <model>` | Switch runtime provider/model |
| `/ai status` | Show current AI configuration |

#### Agent Studio Commands (`!agent` prefix)

| Command | Description |
|---------|-------------|
| `!agent help` | Show help |
| `!agent list-skills` | List available skills |
| `!agent create name=... skills=...` | Create new agent |
| `!agent list` | List all agents |
| `!agent start <agent_id>` | Start agent |
| `!agent stop <agent_id>` | Stop agent |
| `!agent remove <agent_id>` | Remove agent |

---

## Security Architecture

### BlindFill™ Secret Injection

**Core Principle**: Agents request PII by reference name, never see actual values. Secrets are injected directly into browser/containers via memory-only methods.

**Flow Architecture:**
```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Agent     │────▶│ Approval     │────▶│   PII       │
│ Requests    │     │ Engine       │     │ Injector    │
│ "card_num"  │     │ Evaluate     │     │ (Socket)    │
└─────────────┘     │ Policy       │     └─────────────┘
                   │ Returns:     │            │
                   │ ["card_num"] │            │
                   └──────────────┘            │
                                                 ▼
                                          ┌─────────────┐
                                          │   Browser/  │
                                          │ Container   │
                                          │ Receives    │
                                          │ 4242...     │
                                          │ (not agent) │
                                          └─────────────┘
```

**Injection Methods:**

1. **Unix Domain Socket** (Primary, memory-only):
   - Path: `/run/armorclaw/pii/{container}.pii.sock`
   - Permissions: 0600 (owner only)
   - TTL: 5 seconds
   - Socket deleted after delivery

2. **Environment Variables** (Fallback):
   - Prefix: `PII_`
   - Format: `PII_{field_name}={value}`
   - Warning: May be visible in process listings

### PII Approval Workflow

**States:**
- `pending` — Awaiting user approval (default: 5 min TTL)
- `approved` — User approved specific fields
- `denied` — User denied request
- `expired` — Request timed out
- `cancelled` — Agent cancelled request
- `fulfilled` — Approved data delivered

**PII Request Structure:**
```go
type PIIRequest struct {
    ID              string
    AgentID         string
    SkillID         string
    ProfileID       string
    RequestedFields []PIIFieldRequest
    Context         string              // Reason shown to user
    RoomID          string              // Matrix room for events
    Status          PIIRequestStatus
    CreatedAt       time.Time
    ExpiresAt       time.Time           // Default: +5 min
    ApprovedFields  []string
    ApprovedBy      string
    DeniedReason    string
}

type PIIFieldRequest struct {
    Key         string
    DisplayName string
    Required    bool
    Sensitive   bool
}
```

**Approval Engine Decision Types:**
- `DecisionAllow` — Auto-approve
- `DecisionDeny` — Block
- `DecisionRequireApproval` — Ask user

### Hardening State Management

**Mandatory Steps** (all must be true for `delegation_ready`):
```go
type UserHardeningState struct {
    UserID           string
    PasswordRotated  bool   // Changed initial password
    BootstrapWiped   bool   // Cleaned temp files
    DeviceVerified   bool   // Device is trusted
    RecoveryBackedUp bool   // Recovery keys backed up
    BiometricsEnabled bool   // Optional
    DelegationReady  bool   // Computed: all mandatory steps complete
}
```

### Audit Logging

**Three-Tier Audit System:**

#### Tier 1: Basic Audit
```go
type Entry struct {
    Timestamp   time.Time
    EventType   EventType
    SessionID   string
    RoomID      string
    UserID      string
    Details     interface{}
}
```

#### Tier 2: Compliance Audit
```go
type ComplianceEntry struct {
    ID           string
    Timestamp    time.Time
    EventType    EventType
    UserID       string
    Source       string          // Component
    IPAddress    string
    UserAgent    string
    Action       string          // create, read, update, delete
    Resource     string
    Status       string          // success, failure, denied
    PreviousHash string          // Hash chain
    EntryHash    string
}
```

**Compliance Levels:**
- `standard` — 30-day retention
- `extended` — 90-day retention
- `full` — 1-year retention
- `hipaa` — 6-year retention

#### Tier 3: Tamper-Evident Audit
```go
type TamperEvidentEntry struct {
    Sequence     int64
    Timestamp    time.Time
    EventType    string
    Actor        Actor
    Action       string
    Resource     Resource
    Hash         string
    PreviousHash string
    Signature    string          // Optional: high-security mode
    Compliance   ComplianceFlags
}
```

### Zero-Trust Device Verification

**Trust Score Calculation:**
- Base score from verification count, device status, IP history
- Anomalies add: +30 (new device), +20 (unverified), +15 (unknown IP), +25 (>3 failures)

**Device States:**
```go
const (
    StateUnverified        = "unverified"
    StatePendingApproval   = "pending_approval"
    StateAwaitingSecondFactor = "awaiting_second_factor"
    StateVerified          = "verified"
    StateRejected          = "rejected"
    StateExpired           = "expired"
)
```

**Verification Methods:**
- `admin_approval` — Admin must manually approve
- `second_factor` — Existing device confirms
- `wait_period` — Auto-approve after delay
- `automatic` — Not recommended

---

## Component Integration Patterns

### Bridge ↔ Matrix Conduit

**Communication Pattern**: HTTP-based Matrix Client API with long-poll sync

**Key Components:**
- **Matrix Client** (`bridge/pkg/matrix/client.go`): Login, incremental sync, message sending
- **Authentication** (`bridge/pkg/auth/matrix_auth.go`): Token-based auth with power level RBAC
- **Command Handlers** (`bridge/pkg/matrixcmd/handler.go`): Regex-based command parsing

**Data Flow:**
```
Bridge → Conduit: POST /_matrix/client/v3/login, PUT /rooms/{id}/send
Bridge ← Conduit: GET /_matrix/client/v3/sync?filter={}&since={token}
```

### Bridge ↔ Browser Service

**Communication Pattern**: Event-based job queue with status emissions

**Key Components:**
- **Browser Queue** (`bridge/pkg/queue/browser_queue.go`): Priority queue with workers
- **Browser Skill** (`bridge/pkg/browser/browser.go`): Status tracking interface
- **Studio Protocol** (`bridge/pkg/studio/browser_skill.go`): Event namespace `com.armorclaw.browser.*`

**Job States**: PENDING → RUNNING → COMPLETED/FAILED/CANCELLED/AWAITING_PII

### Bridge ↔ OpenClaw Agents

**Communication Pattern**: Factory interface with container lifecycle management

**Key Components:**
- **Agent Integration** (`bridge/pkg/agent/integration.go`): StateMachine + HITLConsentManager
- **State Machine** (`bridge/pkg/agent/state_machine.go`): 9-state lifecycle
- **Orchestrator Factory** (`bridge/pkg/secretary/orchestrator.go`): Spawn, Stop, Remove, GetStatus

**Agent States**: OFFLINE, IDLE, BROWSING, FORM_FILLING, AWAITING_APPROVAL, AWAITING_CAPTCHA, AWAITING_2FA, PROCESSING_PAYMENT, COMPLETE, ERROR

### Bridge ↔ ArmorChat Mobile

**Communication Pattern**: QR code deep link + Matrix messaging + RPC with bearer tokens

**Key Components:**
- **Provisioning Manager** (`bridge/pkg/provisioning/manager.go`): QR token lifecycle with HMAC-SHA256 signatures
- **Token Structure**: `armorclaw://config?d={base64(json)}` with signature
- **Role Persistence**: Roles saved to `provisioning_roles.json`

**Admin Levels**: NONE, MODERATOR, ADMIN, OWNER

### Event Bus Patterns

**Communication Pattern**: Pub/sub with WebSocket push

**Event Types:**
- **Matrix**: `matrix.message`, `matrix.receipt`, `matrix.typing`, `matrix.presence`
- **Agent**: `agent.started`, `agent.stopped`, `agent.status_changed`, `agent.command`, `agent.error`
- **Workflow**: `workflow.started`, `workflow.progress`, `workflow.completed`, `workflow.failed`, `workflow.cancelled`
- **HITL**: `hitl.pending`, `hitl.approved`, `hitl.rejected`, `hitl.expired`, `hitl.escalated`
- **Budget**: `budget.alert`, `budget.limit`, `budget.updated`
- **Platform**: `platform.connected`, `platform.disconnected`, `platform.message`, `platform.error`

---

## Agent Studio

### Purpose

Agent Studio provides **no-code agent creation and management**. Users can define, deploy, and manage AI agents through Matrix chat commands or the web dashboard.

### Agent Definition

```yaml
name: "Travel Booker"
skills:
  - web_browsing
  - form_filling
  - email
provider: openrouter
model: anthropic/claude-3.5-sonnet
system_prompt: |
  You are a travel booking assistant...
constraints:
  - require_approval_for: [payment, pii]
  - max_budget: 10.00  # USD per day
```

### Agent Lifecycle

```
Create → Deploy → Start → [Running] → Stop → Remove
         │                   │
         └─── Containers ────┘
              (Docker)
```

### Studio Service Interface

```go
type StudioService interface {
    HandleRPCMethod(method string, params json.RawMessage) *RPCResponse
}

// Methods: studio.deploy, studio.stats
```

---

## Browser Service

### Purpose

The Browser Service provides **Playwright-based browser automation** for web browsing, form filling, and data extraction.

### Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Bridge    │────▶│ Browser Job  │────▶│ Playwright  │
│   RPC       │     │ Queue        │     │ Browser     │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Job State    │
                    │ Machine      │
                    └──────────────┘
```

### Browser Skills

| Skill | Description |
|-------|-------------|
| `navigate` | Navigate to URL |
| `fill` | Fill form fields |
| `click` | Click element |
| `wait_for_element` | Wait for element |
| `wait_for_captcha` | Wait for CAPTCHA |
| `wait_for_2fa` | Wait for 2FA |
| `extract` | Extract data |
| `screenshot` | Take screenshot |

### Job States

```
PENDING → RUNNING → COMPLETED
                 → FAILED
                 → CANCELLED
                 → AWAITING_PII
```

---

## Rust Office Sidecar

### Purpose

The Rust Office Sidecar is a **high-performance data plane component** for heavy I/O operations, separate from the Rust Vault security enclave. It handles:

- **Cloud Storage Access** - S3, SharePoint, Azure Blob operations
- **Document Processing** - PDF text extraction, DOCX parsing
- **Data Transformation** - Heavy computational work

### Current Implementation State

**Status:** Library Compiles - Binary Compiles - Tests Have Compilation Errors

| Component | Status | Notes |
|-----------|--------|-------|
| **Library Core** | ✅ Compiles | 27 warnings, 0 errors |
| **Binary** | ✅ Compiles | 3 warnings, 0 errors |
| **Security Module** | ✅ Works | Token validation, HMAC, rate limiting |
| **Connectors** | ❌ Disabled | AWS SDK v2 migration needed (21 errors) |
| **Document Processing** | ✅ Stubs | Placeholder implementations |
| **Test Suite** | ❌ 45 errors | Test code needs updates |

**Build Commands:**
```bash
cd sidecar
cargo build --lib    # ✅ Works
cargo build          # ✅ Works
cargo test --lib     # ❌ Test code has errors
```

### Using the Library

The library core compiles and can be used for security operations:

```rust
use armorclaw_sidecar::{
    security::validate_token,
    error::{SidecarError, Result},
};

// Token validation (working)
let token_info = validate_token(&token, &shared_secret)?;
if is_token_expired(&token_info) {
    return Err(SidecarError::AuthenticationFailed("Token expired".to_string()));
}
```

### Architecture

```
┌─────────────────┐
│   Go Bridge     │ (Control Plane - Security Sovereignty)
│   Unix Socket   │
└────────┬────────┘
         │
         │ gRPC over Unix Socket
         │
┌────────▼────────┐
│  Rust Sidecar   │ (Data Plane - Heavy I/O)
│  ┌────────────┐ │
│  │ Connectors │ │ S3, SharePoint, Azure Blob
│  └────────────┘ │
│  ┌────────────┐ │
│  │ Documents  │ │ PDF, DOCX, XLSX, OCR
│  └────────────┘ │
│  ┌────────────┐ │
│  │  Security  │ │ Token Validation, HMAC
│  └────────────┘ │
└─────────────────┘
```

### What's Working

#### ✅ Security Module
- Token validation with HMAC-SHA256
- Timestamp validation (5-minute max age)
- Rate limiting with token bucket algorithm
- Circuit breaker patterns

#### ✅ Configuration
- Environment variable configuration
- TOML configuration support

#### ✅ Error Types
- Comprehensive error taxonomy
- `SidecarError` enum with 15+ variants

### What's Not Working

#### ❌ S3 Connector (22 errors remaining)
**Root Causes:**
1. AWS SDK v2 API changes - fluent builder pattern requires different method calls
2. `client.get_object(request)` → `client.get_object().bucket(b).key(k).send()`
3. `ByteStream::new()` signature changed
4. `into_service_error()` pattern changed
5. `output.contents()` returns `Option<&[Object]>` not `&[Object]`

#### ❌ Document Processing (stubs only)
- PDF text extraction: Not implemented
- DOCX parsing: Not implemented
- XLSX extraction: Stub only
- OCR processing: Stub only

### Recommended Path Forward

**Option A: Complete AWS SDK v2 Migration** (8-12 hours)
- Update S3 connector to use fluent builder pattern
- Fix `ByteStream` handling
- Update error handling patterns
- Full binary compilation

**Option B: Disable S3, Ship Security Module** (2 hours)
- Disable connectors module in lib.rs
- Keep security, config, error modules working
- Minimal working binary for token validation

**Option C: Use Library Directly** (0 hours) ✅ **CURRENT STATE**
- Import library in Go Bridge via FFI
- Use security module for token validation
- Defer S3/document processing to Go implementation

### Integration with Go Bridge

**Planned Integration:**
- gRPC over Unix domain socket
- Token-based authentication
- Rate limiting and circuit breaking
- Separate from Rust Vault (different purpose)

**Not Yet Integrated:**
- Binary doesn't compile
- gRPC service not functional
- Use library directly instead

### Files Requiring Fixes

**Critical Files (22 errors total):**
- `sidecar/src/connectors/aws_s3.rs` - AWS SDK v2 API migration needed
  - Line 302: `upload_internal` method signature
  - Line 348: `ByteStream::new()` type mismatch
  - Line 524: `get_object()` takes 0 args
  - Line 663: `list_objects_v2()` takes 0 args
  - Line 681: `contents()` returns Option, not slice

**Working Files:**
- `sidecar/src/security/` - ✅ Compiles, tests pass
- `sidecar/src/config.rs` - ✅ Compiles
- `sidecar/src/error.rs` - ✅ Compiles
- `sidecar/src/grpc/` - ✅ Compiles (needs protoc for full generation)
- `sidecar/src/document/pdf.rs` - ✅ Compiles (stub implementation)
- `sidecar/src/reliability.rs` - ✅ Compiles

### Documentation

- **Implementation State**: `sidecar/IMPLEMENTATION_STATE.md`
- **README**: `sidecar/README.md`
- **Code**: `sidecar/src/`

### Summary

The Rust Office Sidecar library core is **functional** for security operations (token validation, rate limiting, circuit breakers). The binary compilation is blocked by 22 errors in the S3 connector due to AWS SDK v2 API changes. 

**For immediate use:**
- Import the library directly for security module functionality
- Use Go Bridge for S3/document operations (existing implementation)

**For full Rust sidecar:**
- Invest 8-12 hours to complete AWS SDK v2 migration
- Document processing remains stubbed (PDF/DOCX text extraction)
- Full binary compilation will unlock gRPC server

---

## ArmorChat Android Client

### Purpose

ArmorChat is the **Android mobile client** that provides:
- End-to-end encrypted messaging with agents
- Human-in-the-loop approval for sensitive operations
- QR code provisioning
- Push notifications

### Key Features

| Feature | Description |
|---------|-------------|
| **E2EE Messaging** | Megolm encryption via Matrix |
| **QR Provisioning** | Scan to connect to VPS |
| **PII Approval** | Approve/deny sensitive data access |
| **Push Notifications** | Real-time alerts via Sygnal |
| **Biometric Auth** | Secure keystore access |

### Provisioning Flow

```
Bridge: generate QR with setup_token
   ↓
ArmorChat: scan QR
   ↓
ArmorChat: POST /provisioning.claim
   ↓
Bridge: return admin_token
   ↓
ArmorChat: store credentials, connect to Matrix
```

---

## OpenClaw Agent Runtime

### Purpose

OpenClaw is the **agent runtime** that executes inside isolated Docker containers. It provides:
- AI model integration
- Skill execution
- Browser automation
- Secure PII handling

### Container Security

```yaml
security_opt:
  - no-new-privileges:true
  - seccomp:seccomp-profile.json
  - apparmor:armorclaw-agent
cap_drop:
  - ALL
read_only: true
pids_limit: 100
memory: 512M
```

### Skills

OpenClaw includes **21 browser skills** for web automation:
- Navigation, form filling, clicking
- Data extraction, screenshots
- CAPTCHA/2FA handling
- File operations

---

## RPC API Reference

### Health & System (7 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `health.check` | - | `{status, components}` | Health check |
| `mobile.heartbeat` | `{user_id}` | `{acknowledged}` | Mobile heartbeat |
| `system.health` | - | `{status, timestamp, uptime, checks}` | System health |
| `system.config` | - | `{version, features, endpoints, limits}` | System configuration |
| `system.info` | - | `{server, protocol, capabilities}` | System info |
| `system.time` | - | `{server_time, server_time_utc}` | System time |
| `device.validate` | `{device_id}` | `{valid, trust_level}` | Device validation |

### AI (1 method)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `ai.chat` | `{messages[], model, temperature, max_tokens, key_id}` | `{id, choices[], model, usage}` | Chat completion (rate-limited) |

### Matrix (5 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `matrix.status` | - | `{enabled, connected, logged_in, homeserver, user_id}` | Connection status |
| `matrix.login` | `{username, password}` | `{success, user_id}` | Login to homeserver |
| `matrix.send` | `{room_id, message, msgtype}` | `{event_id, room_id}` | Send message |
| `matrix.receive` | `{cursor, timeout_ms}` | `{events[], cursor, count}` | Receive events (long-poll) |
| `matrix.join_room` | `{room_id, via_servers, reason}` | `{room_id}` | Join room |

### Browser Automation (11 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `browser.navigate` | `{url, agent_id, job_id}` | `{job_id, status, url}` | Navigate to URL |
| `browser.fill` | `{job_id, selector, value}` | `{job_id, status, selector, success}` | Fill form field |
| `browser.click` | `{job_id, selector}` | `{job_id, status, selector, success}` | Click element |
| `browser.status` | `{job_id}` | `{job_id, status, url, session}` | Get job status |
| `browser.wait_for_element` | `{job_id, selector, timeout}` | `{job_id, status, success}` | Wait for element |
| `browser.wait_for_captcha` | `{job_id}` | `{job_id, status}` | Wait for CAPTCHA |
| `browser.wait_for_2fa` | `{job_id}` | `{job_id, status}` | Wait for 2FA |
| `browser.complete` | `{job_id}` | `{job_id, status, completed_at}` | Mark complete |
| `browser.fail` | `{job_id, reason}` | `{job_id, status, error}` | Mark failed |
| `browser.list` | - | `{jobs[], count}` | List all jobs |
| `browser.cancel` | `{job_id}` | `{job_id, status, cancelled_at}` | Cancel job |

### PII Management (9 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `pii.request` | `{agent_id, skill_id, profile_id, variables[], ttl}` | `{request_id, status, expires_at}` | Request PII access |
| `pii.approve` | `{request_id, user_id, approved_fields[]}` | `{request_id, status, approved_at}` | Approve access |
| `pii.deny` | `{request_id, user_id, reason}` | `{request_id, status, denied_at}` | Deny access |
| `pii.status` | `{request_id}` | `{request_id, status, fields}` | Get request status |
| `pii.list_pending` | - | `{requests[], count}` | List pending requests |
| `pii.stats` | - | `{stats}` | PII statistics |
| `pii.cancel` | `{request_id}` | `{request_id, status}` | Cancel request |
| `pii.fulfill` | `{request_id, resolved_vars}` | `{request_id, status}` | Mark fulfilled |
| `pii.wait_for_approval` | `{request_id, timeout}` | `{request_id, status}` | Wait for approval |

### Skills (14 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `skills.execute` | `{skill_name, params}` | `SkillResult` | Execute skill |
| `skills.list` | - | `{skills[], count}` | List enabled skills |
| `skills.get_schema` | `{skill_name}` | `{skill_name, schema}` | Get skill schema |
| `skills.allow` | `{skill_name}` | `{skill_name, status}` | Allow skill |
| `skills.block` | `{skill_name}` | `{skill_name, status}` | Block skill |
| `skills.allowlist_add` | `{type, value}` | `{type, value, status}` | Add to allowlist |
| `skills.allowlist_remove` | `{type, value}` | `{type, value, status}` | Remove from allowlist |
| `skills.allowlist_list` | - | `{ips[], cidrs[]}` | List allowlist |
| `skills.web_search` | `{params}` | `SkillResult` | Web search |
| `skills.web_extract` | `{params}` | `SkillResult` | Web extraction |
| `skills.email_send` | `{params}` | `SkillResult` | Send email |
| `skills.slack_message` | `{params}` | `SkillResult` | Slack message |
| `skills.file_read` | `{params}` | `SkillResult` | Read file |
| `skills.data_analyze` | `{params}` | `SkillResult` | Data analysis |

### Bridge Management (9 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `bridge.start` | - | `{status, message}` | Start bridge |
| `bridge.stop` | - | `{status, message}` | Stop bridge |
| `bridge.status` | `{user_id}` | `{enabled, status, stats}` | Get bridge status |
| `bridge.channel` | `{matrix_room_id, platform, channel_id}` | `{status}` | Bridge channel |
| `bridge.unchannel` | `{platform, channel_id}` | `{status}` | Unbridge channel |
| `bridge.list` | - | `{channels[], count}` | List bridges |
| `bridge.ghost_list` | - | `{ghosts[], count}` | List ghost users |
| `bridge.appservice_status` | - | `{status}` | AppService status |
| `store_key` | `{id, provider, token, display_name, base_url}` | `{success, id}` | Store API key |

### Events (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `events.replay` | `{offset, limit}` | `EventLogRecords[]` | Replay events |
| `events.stream` | `{offset, timeout_ms}` | `EventLogRecords[]` | Stream events (long-poll) |

### Studio (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `studio.deploy` | `{method_name, params}` | Varies | Deploy agent |
| `studio.stats` | - | `{agents, instances, skills}` | Studio statistics |

### Provisioning (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `provisioning.start` | - | `{setup_token, qr_data, expires_in}` | Start provisioning |
| `provisioning.claim` | `{setup_token, device_id, device_name}` | `{success, role, device_id}` | Claim device |

### Hardening (3 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `hardening.status` | - | `HardeningState` | Hardening status |
| `hardening.ack` | `{step}` | `HardeningState` | Acknowledge step |
| `hardening.rotate_password` | `{new_password}` | `{success}` | Rotate password |

---

## Event Types Reference

### Matrix Events
| Event | Description |
|-------|-------------|
| `matrix.message` | New message received |
| `matrix.receipt` | Read receipt |
| `matrix.typing` | Typing notification |
| `matrix.presence` | Presence update |

### Agent Events
| Event | Description |
|-------|-------------|
| `agent.started` | Agent started |
| `agent.stopped` | Agent stopped |
| `agent.status_changed` | Status transition |
| `agent.command` | Command received |
| `agent.error` | Error occurred |

### Workflow Events
| Event | Description |
|-------|-------------|
| `workflow.started` | Workflow started |
| `workflow.progress` | Progress update |
| `workflow.completed` | Workflow completed |
| `workflow.failed` | Workflow failed |
| `workflow.cancelled` | Workflow cancelled |
| `workflow.paused` | Workflow paused |
| `workflow.resumed` | Workflow resumed |

### HITL Events
| Event | Description |
|-------|-------------|
| `hitl.pending` | Approval pending |
| `hitl.approved` | Approval granted |
| `hitl.rejected` | Approval rejected |
| `hitl.expired` | Approval expired |
| `hitl.escalated` | Approval escalated |

### Budget Events
| Event | Description |
|-------|-------------|
| `budget.alert` | Budget alert |
| `budget.limit` | Budget limit reached |
| `budget.updated` | Budget updated |

### Platform Events
| Event | Description |
|-------|-------------|
| `platform.connected` | Platform connected |
| `platform.disconnected` | Platform disconnected |
| `platform.message` | Platform message |
| `platform.error` | Platform error |

---

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ARMORCLAW_KEYSTORE_SECRET` | Base64-encoded keystore master key | - |
| `OPENROUTER_API_KEY` | OpenRouter API key | - |
| `ZAI_API_KEY` | xAI API key | - |
| `OPEN_AI_KEY` | OpenAI API key | - |
| `ARMORCLAW_SERVER_MODE` | Deployment mode (native/sentinel) | `native` |
| `ARMORCLAW_RPC_TRANSPORT` | RPC transport (unix/tcp) | `unix` |
| `ARMORCLAW_SOCKET_PATH` | Unix socket path | `/run/armorclaw/bridge.sock` |
| `ARMORCLAW_LISTEN_ADDR` | TCP listen address | `0.0.0.0:8080` |
| `ARMORCLAW_PUBLIC_BASE_URL` | Public base URL | - |
| `ARMORCLAW_EMAIL` | Admin email (Let's Encrypt) | - |
| `CF_API_TOKEN` | Cloudflare API token | - |
| `CF_TUNNEL_DOMAIN` | Cloudflare tunnel domain | - |

### TOML Configuration

**File**: `/etc/armorclaw/config.toml`

```toml
[server]
mode = "native"                    # native | sentinel
socket_path = "/run/armorclaw/bridge.sock"
listen_addr = "0.0.0.0:8080"
public_base_url = "https://your-domain.com"

[matrix]
enabled = true
homeserver_url = "http://localhost:6167"
username = "bridge"
password = ""

[ai]
default_provider = "openrouter"
default_model = "anthropic/claude-3.5-sonnet"
max_concurrent = 4

[keystore]
path = "/var/lib/armorclaw/keystore.db"

[logging]
level = "info"
format = "json"

[audit]
enabled = true
retention_days = 30
```

---

## Deployment Modes

### Mode Comparison

| Feature | Native | Sentinel | Cloudflare Tunnel | Cloudflare Proxy |
|---------|--------|----------|-------------------|------------------|
| **Use Case** | Development | Production VPS | NAT/firewall | Existing CF |
| **Communication** | Unix socket | TCP + TLS | cloudflared tunnel | HTTP(S) proxy |
| **Access** | Local-only | Public | Public | Public |
| **TLS** | None | Let's Encrypt | Cloudflare SSL | Cloudflare SSL |
| **Public IP Required** | No | Yes | No | Yes |
| **Setup Time** | ~2 min | ~5 min | ~3 min | ~5 min |

### Native Mode

**Configuration:**
```bash
ARMORCLAW_SERVER_MODE=native
ARMORCLAW_RPC_TRANSPORT=unix
ARMORCLAW_SOCKET_PATH=/run/armorclaw/bridge.sock
```

### Sentinel Mode

**Configuration:**
```bash
ARMORCLAW_SERVER_MODE=sentinel
ARMORCLAW_RPC_TRANSPORT=tcp
ARMORCLAW_LISTEN_ADDR=0.0.0.0:8080
ARMORCLAW_PUBLIC_BASE_URL=https://your-domain.com
ARMORCLAW_EMAIL=admin@your-domain.com
```

### Cloudflare Tunnel Mode

**Configuration:**
```bash
CF_API_TOKEN=your-token
CF_TUNNEL_DOMAIN=armorclaw.example.com
```

---

## Testing & Verification

### Test Categories

The testing suite includes **10 comprehensive test categories** with **136+ individual tests**:

| Category | Description | Test Count |
|----------|-------------|------------|
| **SSH Connectivity** | Key validation, connection, retry logic | 12 |
| **Command Execution** | Remote commands, output capture, exit codes | 8 |
| **Container Health** | Container status, logs, resource usage | 6 |
| **API Endpoints** | Bridge RPC, Matrix client, health checks | 8 |
| **Integration** | Cross-component communication | 8 |
| **Security** | Firewall, SSH hardening, container isolation | 35 |
| **Deployment Modes** | Native, Sentinel, Cloudflare detection | 6 |
| **SSL/TLS** | Certificate presence, expiry, chain | 6 |
| **Performance** | SSH speed, API times, container resources | 6 |
| **Output Formatting** | JSON console output, error handling | 1 |

### Running Tests

```bash
# Run all tests
bash tests/ssh/run_all_tests.sh --all

# Run specific category
bash tests/ssh/run_all_tests.sh --security

# Run with verbose output
bash tests/ssh/run_all_tests.sh --all --verbose

# Run with JSON output
bash tests/ssh/run_all_tests.sh --all --output json
```

### Test Results Location

- **Evidence Directory**: `.sisyphus/evidence/`
- **Summary File**: `.sisyphus/evidence/IMPLEMENTATION_SUMMARY.md`
- **JSON Output**: `task-N-results.json`
- **Console Output**: `task-N-success.txt`

---

## Document Index

### Primary Documentation
- **README.md** - System overview and quick start
- **ARMORCLAW.md** - AI-powered deployment skills introduction
- **doc/armorclaw.md** - This document (comprehensive architecture)

### Skills Documentation
- `.skills/README.md` - Skills index
- `.skills/PLATFORM.md` - Cross-platform patterns
- `.skills/*/SKILL.md` - Individual skill documentation

### Deployment Documentation
- `deploy/README.md` - Deployment scripts reference
- `deploy/install.sh` - One-command installer

### Review Documentation
- `review.md` - Code review findings
- `applications/ArmorChat-review.md` - Android client review
- `applications/ArmorTerminal-review.md` - Terminal client review

---

**End of Documentation**
