# ArmorClaw Architecture Review

> **Purpose:** Complete guide to ArmorClaw deployment, architecture, and components
> **Version:** 4.5.0
> **Last Updated:** 2026-03-10
> **Status:** Active Reference

---

## Executive Summary

ArmorClaw v4.5.0 provides a **production-ready AI agent platform** that runs AI agents 24/7 on your VPS, controlled from your phone via Matrix (E2EE) or ArmorChat mobile app.

**Core Components:**

- **ArmorClaw Bridge** - Native Go binary with encrypted keystore, Matrix adapter, JSON-RPC server, Provider Registry
- **Matrix/Conduit** - Integrated homeserver for E2EE messaging (optional, auto-detected and installed)
- **Provider Registry** - Embedded registry with 12+ pre-configured AI providers
- **Agent Studio** - Agent management with skills, PII access control, and MCP approval workflow
- **Browser Automation** - Playwright-based service with anti-detection and PII protection
- **Skills Executor** - Built-in skills (data_analyze, web_extract, email_send, web_search, file_read, slack_message)
- **Memory Store** - Vector-based memory for agents
- **Catwalk Integration** - Dynamic AI provider/model discovery and runtime switching

**Deployment Options:**
- **Quick Install:** Single-command installer with auto-detection
- **Matrix Install:** Integrated Conduit setup for instant QR provisioning
- **Bridge-only:** Install bridge only (existing Matrix or external)

**Key Design Principles:**
- Zero persistent secrets on disk (SQLCipher + XChaCha20-Poly1305)
- Bridge runs as native binary (not in container) for security
- Agent containers isolated with Unix socket communication
- End-to-end encryption for all Matrix communication
- Pull-based architecture (agents request, bridge validates)

---

## Quickstart Flow Diagram (v4.5.0)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW INSTALLATION FLOW (v4.5.0)           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: curl -fsSL https://.../install.sh | bash                     │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. INSTALL.SH (Stage-0 Bootstrap)                            │   │
│  │    • Download installer-v5.sh from GitHub                        │   │
│  │    • Verify GPG signature (A1482657...)                         │   │
│  │    • Verify SHA256 checksum                                      │   │
│  │    • Download bridge binary from GitHub Releases                   │   │
│  │    • Check: curl, sha256sum, gpg, mktemp, sed                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. INSTALLER-V5.SH (Stage-1 Full Installer)                   │   │
│  │    • Detect OS/arch (Linux/ARM64, etc.)                        │   │
│  │    • Check prerequisites (Docker, sudo)                          │   │
│  │    • Detect Docker Compose (compose vs docker-compose)              │   │
│  │    • Download setup scripts (setup-quick.sh, setup-matrix.sh)      │   │
│  │    • Wait for Docker daemon (20s timeout, dual-check)            │   │
│  │    • Lockfile protection (flock)                                 │   │
│  │    • Persistent logging (/var/log/armorclaw/install.log)          │   │
│  │    • Export env vars: DOCKER_COMPOSE, CONDUIT_IMAGE, etc.        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. SETUP-WIZARD.SH (Go TUI or Non-Interactive)                │   │
│  │    • Check env vars (ARMORCLAW_API_KEY triggers non-interactive)   │   │
│  │    • Detect terminal (TTY, color support, size)                   │   │
│  │    • Launch Huh? TUI wizard or use env vars                    │   │
│  │      Step 1: AI Provider selection (with registry)               │   │
│  │      Step 2: API key entry                                     │   │
│  │      Step 3: Admin credentials + deploy confirmation               │   │
│  │    • Generate randomized admin username (armor-admin-xxxxxx)        │   │
│  │    • Generate admin password (or use ARMORCLAW_ADMIN_PASSWORD)     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 4. SETUP-QUICK.SH (Infrastructure Setup)                       │   │
│  │    • Pre-flight checks (Docker, network, disk)                  │   │
│  │    • Create directories (/etc/armorclaw, /var/lib/armorclaw)     │   │
│  │    • Detect if Matrix is running                                 │   │
│  │    • Option: Offer to install Conduit if missing                  │   │
│  │    • Generate self-signed SSL certificate                         │   │
│  │    • Load Provider Registry (12+ providers embedded)              │   │
│  │    • Configure AI provider (OpenAI, Anthropic, Zhipu, etc.)     │   │
│  │    • Create admin user via shared-secret API (zero-touch)          │   │
│  │    • Write config.toml                                          │   │
│  │    • Initialize SQLCipher keystore                                │   │
│  │    • Initialize Agent Studio database (studio.db)                    │   │
│  │    • Start Matrix stack (if installing Conduit)                   │   │
│  │      docker run matrixconduit/matrix-conduit:latest               │   │
│  │    • Register bridge user on Conduit                             │   │
│  │    • Start bridge (native binary on host)                         │   │
│  │      sudo ./armorclaw-bridge                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 5. BOOTSTRAP-ADMIN (Post-Setup)                             │   │
│  │    • Wait for bridge.sock to appear                            │   │
│  │    • Inject API key via RPC (store_key method)                     │   │
│  │    • Create "ArmorClaw Bridge" room on Matrix                     │   │
│  │    • Auto-claim OWNER role for admin via provisioning.claim        │   │
│  │    • Generate QR code for ArmorChat provisioning                   │   │
│  │    • Display connection info + credentials                        │   │
│  │    • Wait for bridge process                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                          ARMORCLAW RUNNING   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Provider Registry Architecture (v4.5.0)

### Embedded Providers

ArmorClaw includes an **embedded provider registry** with 12+ pre-configured AI providers, eliminating the need for manual setup.

**Default Providers:**

| Provider | ID | Protocol | Base URL | Aliases |
|----------|-----|----------|-----------|----------|
| OpenAI | `openai` | OpenAI | https://api.openai.com/v1 | - |
| Anthropic | `anthropic` | Anthropic | https://api.anthropic.com/v1 | - |
| Google | `google` | OpenAI | https://generativelanguage.googleapis.com/v1 | - |
| xAI | `xai` | OpenAI | https://api.x.ai/v1 | - |
| OpenRouter | `openrouter` | OpenAI | https://openrouter.ai/api/v1 | - |
| **Zhipu AI** | `zhipu` | OpenAI | https://api.z.ai/api/paas/v4 | `zai`, `glm` |
| **DeepSeek** | `deepseek` | OpenAI | https://api.deepseek.com/v1 | - |
| **Moonshot AI** | `moonshot` | OpenAI | https://api.moonshot.ai/v1 | - |
| NVIDIA NIM | `nvidia` | OpenAI | https://integrate.api.nvidia.com/v1 | - |
| **Groq** | `groq` | OpenAI | https://api.groq.com/openai/v1 | - |
| Cloudflare | `cloudflare` | OpenAI | https://gateway.ai.cloudflare.com/v1 | - |
| Ollama | `ollama` | OpenAI | http://localhost:11434/v1 | - |

**Registry Features:**

- **Embedded fallback:** Registry embedded in bridge binary (no external dependency)
- **Dynamic loading:** Can load from `/etc/armorclaw/providers.json` if exists
- **Remote download:** Supports `ARMORCLAW_PROVIDERS_URL` for remote registry
- **Alias resolution:** Users can type `zai`, `zhipu`, or `glm` to use Zhipu AI
- **Protocol abstraction:** All providers expose OpenAI-compatible interface

### Catwalk Integration (Dynamic Discovery)

ArmorClaw integrates with **Catwalk** for dynamic AI provider/model discovery:

**Features:**
- **HTTP client** for querying Catwalk API
- **Dynamic provider list** in quickstart wizard (falls back to embedded if Catwalk unavailable)
- **Runtime AI switching** via Matrix commands:
  - `/ai providers` - List available providers
  - `/ai models <provider>` - List models for a provider
  - `/ai switch <provider> <model>` - Switch provider and model
  - `/ai status` - Show current configuration

**Implementation:**
- `bridge/internal/wizard/catwalk.go` - Catwalk HTTP client
- `bridge/internal/ai/runtime.go` - AI runtime with provider switching
- `bridge/pkg/matrixcmd/handler.go` - `/ai` command handler
- `bridge/Dockerfile` - Downloads Catwalk v0.28.3 at build time

---

## Bridge Architecture (v4.5.0)

### Native Binary Deployment

**Critical Design Decision:** The ArmorClaw bridge runs as a **native binary on the host**, not in a Docker container. This is intentional for security:

- **Docker socket access:** Scoped Docker API access for container management
- **Host filesystem access:** Read/write to configs, secrets, logs
- **Unix socket:** `/run/armorclaw/bridge.sock` for agent communication
- **Performance:** Zero Docker overhead for the bridge itself

**To run the bridge:**
```bash
sudo ./bridge/build/armorclaw-bridge
```

### Package Structure

```
bridge/
├── cmd/
│   ├── bridge/           # Main bridge binary (3,000+ lines)
│   ├── bootstrap-admin/   # Admin bootstrap tool
│   └── secureclaw/     # Secure bridge variant
├── pkg/                # Public packages (library code)
│   ├── adapters/         # Multi-protocol adapters (Matrix, Slack, Discord)
│   ├── admin/           # Admin RPC methods
│   ├── agent/           # Agent state machine and integration
│   ├── api/             # HTTP API layer
│   ├── appservice/      # Matrix appservice support
│   ├── audio/           # Audio processing
│   ├── audit/           # Audit logging
│   ├── auth/            # Authentication and authorization
│   ├── browser/         # Browser service client (Go)
│   ├── budget/           # Budget/FinOps controller
│   ├── config/          # Configuration management (TOML)
│   ├── crypto/          # Cryptographic primitives
│   ├── dashboard/        # Admin dashboard
│   ├── discovery/       # Service discovery
│   ├── docker/          # Docker client (scoped)
│   ├── enforcement/      # Policy enforcement
│   ├── errors/          # Error handling
│   ├── eventbus/        # Event bus implementation
│   ├── eventlog/        # Structured event logging
│   ├── ffi/             # Foreign function interfaces
│   ├── ghost/           # Ghost mode (stealth)
│   ├── health/          # Health checks
│   ├── http/            # HTTP client/server utilities
│   ├── invite/          # Invitation management
│   ├── keystore/        # SQLCipher encrypted keystore
│   ├── license/          # License validation
│   ├── logger/          # Logging infrastructure
│   ├── lockdown/        # Lockdown mode enforcement
│   ├── matrix/          # Matrix SDK integration
│   ├── matrixcmd/       # Matrix command handlers
│   ├── notification/    # Push notifications
│   ├── pii/             # PII detection and redaction
│   ├── plugin/          # Plugin system
│   ├── providers/       # Provider registry (NEW v4.5.0)
│   ├── provisioning/    # Provisioning API (QR codes)
│   ├── push/            # Push notification gateway
│   ├── qr/              # QR code generation
│   ├── queue/           # Job queue (browser)
│   ├── recovery/        # Recovery procedures
│   ├── runtime/         # Container runtime (Docker, Firecracker, Containerd)
│   ├── secrets/         # Secrets management
│   ├── securerandom/    # Secure random generation
│   ├── security/        # Security primitives
│   ├── setup/           # Setup utilities
│   ├── socket/          # Unix socket server
│   ├── sso/             # Single sign-on
│   ├── studio/          # Agent Studio (agents, skills, MCP)
│   ├── trust/           # Trust management
│   ├── ttl/             # TTL-based cache
│   ├── turn/            # TURN/STUN configuration
│   ├── voice/           # Voice/WebRTC
│   ├── webrtc/          # WebRTC implementation
│   └── websocket/       # WebSocket support
└── internal/           # Internal modules (bridge-specific)
    ├── adapter/         # Matrix adapter implementation
    ├── ai/              # AI client runtime (multi-provider)
    ├── agent/           # Agent runtime (container management)
    ├── cache/           # LRU cache implementation
    ├── capability/       # Capability detection
    ├── events/          # Matrix event bus
    ├── executor/        # Agent execution engine
    ├── memory/          # Vector-based memory store
    ├── metrics/         # Agent metrics
    ├── petg/            # Catwalk gateway integration
    ├── router/          # Request routing
    ├── skills/          # Skills executor (data_analyze, web_extract, etc.)
    ├── speculative/     # Speculative execution
    ├── trace/           # Distributed tracing
    └── wizard/          # Setup wizard (Go TUI)
```

### Key Internal Modules

**Skills Executor** (`internal/skills/`)
- **Built-in skills:**
  - `data_analyze` - Structured data analysis
  - `web_extract` - Web content extraction
  - `email_send` - Email sending
  - `web_search` - Web search queries
  - `file_read` - Secure file reading
  - `slack_message` - Slack integration
- **Policy engine:** Allowlist, denylist, rate limiting
- **SSRF protection:** Prevent server-side request forgery
- **Schema validation:** Input validation for all skills

**Memory Store** (`internal/memory/`)
- **Checkpoint system:** Agent state snapshots
- **Batch processing:** Efficient vector operations
- **Store interface:** Pluggable backends

**Agent Runtime** (`internal/agent/`)
- **State machine:** Agent lifecycle management
- **Container orchestration:** Spawn, monitor, stop agents
- **Resource limits:** CPU, memory, disk quotas

**Executor Engine** (`internal/executor/`)
- **Parallel execution:** Concurrent task execution
- **Error handling:** Retry logic with exponential backoff
- **Circuit breakers:** Prevent cascading failures

**Router** (`internal/router/`)
- **Request routing:** Matrix commands → handlers
- **Cache layer:** LRU cache for common queries
- **Middleware:** Logging, auth, rate limiting

**Capabilities** (`internal/capability/`)
- **Capability detection:** Agent feature detection
- **Policy enforcement:** What agents can/cannot do

---

## Matrix Integration (v4.5.0)

### Conduit Setup Options

**Option A: Integrated Setup (Recommended)**

The installer auto-detects if Matrix is running and offers to install Conduit:

```bash
# Quick install with Matrix
export ARMORCLAW_API_KEY=sk-your-key
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**What happens:**
1. Installer detects no Matrix server running
2. Offers to install Conduit homeserver
3. Runs `docker run matrixconduit/matrix-conduit:latest`
4. Configures bridge to connect to local Conduit
5. Generates QR code for ArmorChat provisioning

**Option B: Bridge-Only Mode**

If you have an existing Matrix server:

```bash
# Connect to existing Matrix
export ARMORCLAW_API_KEY=sk-your-key
export MATRIX_SERVER_URL=https://your.matrix.server
export MATRIX_ACCESS_TOKEN=your-token
curl -fsSL ... | bash
```

### Matrix Event Bus Improvements

The bridge uses a **high-throughput event bus** with zero-allocation receive path:

**Key Features:**
- **Zero-allocation receive path:** Pre-allocated batch buffers
- **Instant wake-up:** Events delivered within 25ms, not polling storms
- **Slow consumer detection:** Cursor guard prevents message loss
- **Context cancellation:** Proper timeout handling prevents indefinite blocking

**RPC Methods:**
| Method | Description |
|--------|-------------|
| `matrix.status` | Returns connection health and user info |
| `matrix.login` | Dynamic login through bridge |
| `matrix.send` | Message sending via adapter |
| `matrix.receive` | Long-polling with cursor + timeout |

---

## Agent Studio (v4.5.0)

### Overview

Agent Studio provides a **no-code interface** for creating and managing AI agents through Matrix chat commands or JSON-RPC.

### Components

**Skill Registry** (`bridge/pkg/studio/registry.go`)
- 8+ built-in skills (data_analyze, web_extract, email_send, web_search, file_read, slack_message)
- Custom skill support via JSON schema
- Skill metadata (name, description, parameters, security level)

**PII Registry** (`bridge/pkg/studio/`)
- 10+ default PII fields with sensitivity levels (low/medium/high/critical)
- Automatic regex-based redaction (credit cards, SSN, email, phone, API keys)
- BlindFill references for secure PII injection

**Resource Profiles**
- 3 tiers: low, medium, high
- Memory/CPU limits per tier
- Disk quota enforcement

**Agent Factory** (`bridge/pkg/studio/factory.go`)
- Agent definition creation
- Instance spawning with Docker
- Security hardening applied automatically

**MCP Approval Workflow** (`bridge/pkg/studio/mcp_approval.go`)
- Role-based access control for external MCP connections
- Pending approval queue
- Admin approval/rejection workflow

### Matrix Commands

```
!agent help              Show command help
!agent create <name>     Start interactive wizard
!agent list-skills       List available skills
!agent list-pii          List PII fields
!agent create <name>     Create agent definition
!agent list              List agent definitions
!agent spawn <id>        Spawn agent instance
!agent stop <instance>   Stop running instance
!agent stats             Show statistics
```

### RPC Methods

| Method | Description |
|--------|-------------|
| `studio.create_agent` | Create agent definition |
| `studio.list_skills` | List available skills |
| `studio.list_agents` | List agent definitions |
| `studio.spawn_agent` | Spawn agent instance |
| `studio.stop_agent` | Stop agent instance |
| `studio.stats` | Show studio statistics |
| `studio.list_mcps` | List available MCPs |
| `studio.request_mcp_approval` | Request MCP access |
| `studio.approve_mcp_request` | Approve MCP request (admin) |
| `studio.reject_mcp_request` | Reject MCP request (admin) |

---

## Browser Automation (v4.5.0)

### Architecture

The browser automation system consists of three main components:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BROWSER AUTOMATION ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │ ArmorChat       │     │ Bridge          │     │ Browser Service │       │
│  │ (Android)       │     │ (Go)            │     │ (TypeScript)    │       │
│  │                 │     │                 │     │                 │       │
│  │ JSON-RPC        │────►│ Browser Client  │────►│ Playwright      │       │
│  │ Matrix Events   │     │ Queue Processor │     │ Stealth Mode    │       │
│  │                 │     │                 │     │                 │       │
│  └─────────────────┘     └─────────────────┘     └─────────────────┘       │
│         │                       │                       │                   │
│         │                       ▼                       │                   │
│         │              ┌─────────────────┐              │                   │
│         │              │ Job Queue       │              │                   │
│         │              │ (SQLite)        │              │                   │
│         │              └─────────────────┘              │                   │
│         │                       │                       │                   │
│         └───────────────────────┴───────────────────────┘                   │
│                     Matrix Events (status, response)                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Browser Service (TypeScript/Playwright)

**Location:** `browser-service/`

**Features:**
- Playwright-based headless browser automation
- Anti-detection / stealth mode
- Screenshot capture with element cropping
- Form filling with PII injection
- Cookie and session management
- Proxy rotation support

**API Endpoints:**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Service health check |
| `/navigate` | POST | Navigate to URL |
| `/fill` | POST | Fill form fields |
| `/click` | POST | Click element |
| `/extract` | POST | Extract page data |
| `/screenshot` | POST | Capture screenshot |
| `/status` | GET | Current browser state |

### Browser Client (Go)

**Location:** `bridge/pkg/browser/`

**Components:**
- `client.go` - HTTP client for browser-service API
- `processor.go` - Job queue processor with retry logic
- `browser.go` - Core browser types and interfaces

**Configuration:**
```toml
[browser]
enabled = true
service_url = "http://localhost:3001"
timeout = 30
max_retries = 3
retry_delay = 2

[browser.stealth]
enabled = true
fingerprint_seed = ""

[browser.queue]
max_workers = 3
max_depth = 100
```

### Deployment

**Docker Compose:** `deploy/browser/docker-compose.browser.yml`

```yaml
services:
  browser-service:
    build: ../../browser-service
    ports:
      - "3001:3001"
    environment:
      - NODE_ENV=production
      - STEALTH_MODE=true
    cap_add:
      - SYS_ADMIN
    security_opt:
      - seccomp:unconfined
```

---

## Docker Compose Architecture (v4.5.0)

### Topology Separation

ArmorClaw uses a **two-tier network topology** for security:

```
┌─────────────────────────────────────────────────────────────┐
│                        HOST MACHINE                        │
│                                                        │
│  ┌─────────────────┐                                    │
│  │ ArmorClaw Bridge│ ← Unix Socket /run/armorclaw/bridge.sock│
│  │ (Native Binary) │                                    │
│  └────────┬────────┘                                    │
│           │                                               │
│  ┌────────▼────────────────────────────────────────────────┐   │
│  │              matrix-net (172.20.0.0/24)              │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │   Nginx     │  │  Conduit    │  │   Coturn    │     │   │
│  │  │  (Proxy)    │  │ (Matrix HS) │  │  (TURN/STUN)│     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  └──────────────────────────────────────────────────────────┘   │
│           │                                               │
│  ┌────────▼────────────────────────────────────────────────┐   │
│  │              bridge-net (172.21.0.0/24) [internal]      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │   Sygnal    │  │ mautrix-*   │  │   Agents    │     │   │
│  │  │ (Push GW)   │  │  (Bridges)  │  │  (Docker)   │     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Network Security

- **matrix-net:** Exposed via Nginx (80/443) for public Matrix access
- **bridge-net:** Internal only (no external access) for services
- **Bridge on host:** Unix socket only (no network exposure)

### Docker Compose Files

**File:** `docker-compose.yml` (Meta-composition)
- Includes both `docker-compose.matrix.yml` and `docker-compose.bridge.yml`
- Provides convenient presets for full stack

**Matrix Stack:** `docker-compose.matrix.yml`
- Services: Nginx, Coturn
- Optional profile: `frontend` (Nginx)
- Network: `matrix-net` (172.20.0.0/24)

**Bridge Stack:** `docker-compose.bridge.yml`
- Services: Sygnal, mautrix-* (optional profiles)
- Network: `bridge-net` (172.21.0.0/24, internal)
- External: `matrix-net` (for Matrix API access)

### Optional Bridges

ArmorClaw supports **Matrix-to-Service bridges** via profiles:

| Bridge | Profile | Purpose |
|---------|----------|---------|
| mautrix-slack | `slack` | Slack integration |
| mautrix-discord | `discord` | Discord integration |
| mautrix-telegram | `telegram` | Telegram integration |
| mautrix-whatsapp | `whatsapp` | WhatsApp integration |

**Usage:**
```bash
docker-compose --profile slack up -d
docker-compose --profile discord up -d
```

---

## Installation Script Flow (v4.5.0)

### Stage-0: Bootstrap (install.sh)

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**Steps:**
1. Download `installer-v5.sh`, SHA256, GPG signature, and public key
2. Normalize line endings to LF (critical for GPG verification)
3. Verify SHA256 checksum
4. Verify GPG signature (A1482657223EAFE1C481B74A8F535F90685749E0)
5. Download bridge binary from GitHub Releases
6. Execute `installer-v5.sh`

### Stage-1: Full Installer (installer-v5.sh)

**Features:**
- **Lockfile protection:** flock to prevent parallel installs
- **Docker daemon readiness:** Dual-check (info + ps) with 20s timeout
- **Persistent logging:** /var/log/armorclaw/install.log
- **Environment passthrough:** All vars passed to setup scripts
- **Docker Compose detection:** Supports both `docker compose` and `docker-compose`
- **Conduit image control:** Via CONDUIT_VERSION environment variable

**Configuration:**
```bash
INSTALL_MODE=quick|matrix  # Choose installation type
DOCKER_COMPOSE           # Auto-detected
CONDUIT_VERSION           # Default: latest
CONDUIT_IMAGE             # Default: matrixconduit/matrix-conduit:$CONDUIT_VERSION
```

### Stage-2: Setup Scripts

**Quick Setup** (`setup-quick.sh`):
- Interactive wizard or non-interactive mode
- AI provider configuration with registry
- Admin creation (zero-touch with shared-secret API)
- Bridge configuration
- Matrix detection and optional Conduit setup

**Matrix Setup** (`setup-matrix.sh`):
- Deploy Matrix/Conduit stack
- Configure Nginx reverse proxy
- Setup Coturn for voice/video
- Generate SSL certificates

---

## What You MUST Do After Setup

### 1. Save Your Credentials (CRITICAL)

The setup displays credentials **once**. Write them down:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Admin Login (Element X / ArmorChat):
  Username:   @admin:192.168.1.100
  Password:   Xy7kL9mN2pQ4rS8t
  Homeserver: http://192.168.1.100:6167

  Password saved to: /var/lib/armorclaw/.admin_password
  ⚠ Delete this file after first login for security.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Recovery if lost:**
```bash
# Password file (if still present)
docker exec armorclaw cat /var/lib/armorclaw/.admin_password

# Or reset via Matrix admin API (requires Conduit admin token)
```

### 2. Connect a Matrix Client

**Option A: Element X (Recommended)**

1. Download Element X: https://element.io/download
2. Open Element X
3. Click "Edit" next to homeserver
4. Enter: `http://YOUR_IP:6167` (or your domain)
5. Click "Sign in"
6. Enter username: `admin` (or what was configured)
7. Enter password from setup
8. You should see the "ArmorClaw Bridge" room

**Option B: ArmorChat (Mobile)**

1. Install ArmorChat on your mobile device
2. Scan the QR code displayed in terminal
3. Or manually enter:
   - Homeserver: `http://YOUR_IP:6167`
   - Username: `admin`
   - Password: (from setup)

### 3. Verify Bridge Connection

In the "ArmorClaw Bridge" room, send:

```
!status
```

You should receive a response like:

```
✓ Bridge is running
✓ Matrix connected
✓ Keystore initialized
✓ 1 API key configured
✓ Agent Studio ready
```

### 4. Create Your First Agent

Using Matrix commands:

```
!agent create "Document Processor"
```

Follow the interactive wizard to:
- Select skills (data_analyze, web_extract)
- Configure PII access (client_name, client_email)
- Set resource tier (medium)

### 5. Test AI Functionality

Send a message to the bridge:

```
Hello, can you help me with something?
```

The bridge should respond using your configured AI provider.

### 6. Delete Password File (Security)

After successful login:

```bash
docker exec armorclaw rm /var/lib/armorclaw/.admin_password
```

---

## Environment Variables Reference

### Required for Non-Interactive Mode

| Variable | Required | Description |
|----------|----------|-------------|
| `ARMORCLAW_API_KEY` | **Yes** | Triggers non-interactive mode. Your AI provider's API key. |
| `ARMORCLAW_SERVER_NAME` | Auto | Server domain or IP. **Auto-detected on host** and passed to container. |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ARMORCLAW_API_BASE_URL` | Provider default | Custom API endpoint (for Anthropic, Zhipu, etc.) |
| `ARMORCLAW_ADMIN_USERNAME` | (generated) | Custom admin username |
| `ARMORCLAW_ADMIN_PASSWORD` | (generated) | Admin password for Matrix |
| `ARMORCLAW_PROFILE` | `quick` | `quick` or `matrix` |
| `CONDUIT_VERSION` | `latest` | Conduit version to install |
| `CONDUIT_IMAGE` | matrixconduit/matrix-conduit:latest | Custom Conduit image |
| `DOCKER_COMPOSE` | (auto-detected) | Docker Compose command |

### Development/Debugging

| Variable | Description |
|----------|-------------|
| `ARMORCLAW_DEBUG` | Enable debug logging |
| `ARMORCLAW_DEV_MODE` | Enable log backup to `/tmp/armorclaw-logs/` |
| `ARMORCLAW_PROVIDERS_URL` | Remote provider registry URL |

---

## Common Post-Deployment Tasks

### Check System Health

```bash
# Container status
docker ps | grep armorclaw

# Bridge health via RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Matrix homeserver health
curl http://localhost:6167/_matrix/client/versions

# View setup log
docker exec armorclaw cat /var/log/armorclaw/setup.log

# Agent Studio stats
echo '{"jsonrpc":"2.0","id":1,"method":"studio.stats"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Add Another API Key

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"store_key","params":{"id":"anthropic-backup","provider":"anthropic","token":"sk-ant-xxx","display_name":"Anthropic Backup"}}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Generate New ArmorChat QR

The QR code is **automatically generated** at setup completion. To regenerate:

```bash
docker exec armorclaw armorclaw-bridge generate-qr --host <server-ip> --port 8443
```

### View Logs

```bash
# Setup log
docker exec armorclaw view-setup-log

# Follow setup log
docker exec armorclaw view-setup-log --follow

# Errors only
docker exec armorclaw view-setup-log --errors
```

### List Available MCPs

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_mcps"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 4.5.0 | 2026-03-10 | **Provider Registry:** Embedded 12+ providers (Zhipu, Moonshot, DeepSeek, Groq). **Matrix Integration:** Auto-detect and install Conduit. **Catwalk:** Dynamic provider/model discovery with `/ai` commands. |
| 4.4.0 | 2026-03-08 | **Systemd Hardening:** Type=simple, RuntimeDirectory, deterministic startup. **Installer:** Lockfile, Docker readiness, persistent logging, GPG verification v1.4.3. |
| 4.3.0 | 2026-03-06 | **Catwalk AI:** Dynamic provider discovery, runtime switching via Matrix commands, Catwalk v0.28.3 integration. |
| 4.2.0 | 2026-03-07 | **Matrix Event Bus:** Zero-allocation receive path, slow consumer detection, context cancellation. |
| 4.1.0 | 2026-02-28 | **Browser Service:** TypeScript/Playwright automation, Browser Client (Go), Queue Processor. |
| 4.0.0 | 2026-02-28 | **Agent Studio:** Skills registry, PII registry, Resource profiles, MCP approval workflow. |

---

## Recent Reviews

### 2026-03-08 - Installer Hardening Review

**Completed Work:**

**Installer Hardening (Phase 6)**

1. **Docker Daemon Readiness Checks**
   - wait_for_docker() function in all installers
   - Dual-check: docker info && docker ps
   - 20-second timeout with 2-second intervals

2. **Installer Lockfile**
   - Uses flock for parallel install prevention
   - EXIT trap with flock -u 2>/dev/null || true

3. **Persistent Logging**
   - /var/log/armorclaw/install.log
   - Fallback to /tmp/armorclaw if /var/log unavailable

4. **Environment Variable Passthrough**
   - DOCKER_COMPOSE, CONDUIT_VERSION, CONDUIT_IMAGE exported
   - env -S bash for proper inheritance

5. **Docker Compose Detection**
   - Detects both 'docker compose' and 'docker-compose'
   - Fallback mechanism in sub-installers

**Test Coverage:** 8 tests (all passing ✅)

### 2026-03-08 - Systemd Service Hardening Review

**Completed Work:**

**Systemd Service Hardening (Phase 6b)**

1. **Eliminated Startup Timeouts**
   - Changed Type=notify to Type=simple
   - Service active immediately upon process launch

2. **Automated Runtime Management**
   - RuntimeDirectory=armorclaw and RuntimeDirectoryMode=0755
   - Systemd creates/cleans up /run/armorclaw automatically

3. **Improved Reliability**
   - After=network-online.target and Wants=network-online.target
   - LimitNOFILE=65536 for socket scalability
   - Restart=always with RestartSec=5

4. **Security Hardening**
   - ProtectKernelTunables=true and ProtectControlGroups=true

**Test Coverage:** 9 tests (all passing ✅)

---

## Related Documentation

- **[quickstart-docker.md](../guides/quickstart-docker.md)** - Full quickstart guide
- **[error-catalog.md](../guides/error-catalog.md)** - Error codes and solutions
- **[configuration.md](../guides/configuration.md)** - Post-setup configuration
- **[troubleshooting.md](../guides/troubleshooting.md)** - Detailed troubleshooting
- **[rpc-api.md](../reference/rpc-api.md)** - Complete JSON-RPC API reference
- **[agent-studio-rpc-api.md](../reference/agent-studio-rpc-api.md)** - Agent Studio RPC methods
- **[provider-registry-complete.md](../PROGRESS/provider-registry-complete.md)** - Provider registry implementation
- **[progress.md](../PROGRESS/progress.md)** - Complete progress log

---

**Document Version:** 4.5.0
**Last Updated:** 2026-03-10
**Maintainer:** ArmorClaw Team
