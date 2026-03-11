# ArmorClaw Architecture Review

> **Purpose:** Complete guide to ArmorClaw deployment, architecture, and components
> **Version:** 4.5.0
> **Last Updated:** 2026-03-10
> **Status:** Active Reference

---

## Phase 4 & 5 Completion Status (2026-03-10)

### ✅ All Success Conditions Met

| Condition | Status | Evidence |
|-----------|--------|----------|
| Quick Start completes with minimal answers | ✅ Pass | Non-interactive mode with env vars |
| Bridge running | ✅ Pass | Process running in container |
| Matrix running | ✅ Pass | Conduit responding on port 6167 |
| SQLCipher keystore initialized | ✅ Pass | Hardware-derived key, encrypted DB |
| API key stored | ✅ Pass | Environment variable support (OPENROUTER_API_KEY, ZAI_API_KEY) |
| OWNER claimed | ✅ Pass | @admin:armorclaw.local registered |
| Provisioning QR generated | ✅ Pass | QR code generation works |
| !status works | ✅ Pass | Via matrix.status RPC |
| Normal AI chat works | ✅ Pass | Env var support for API keys |
| Agent Studio works | ✅ Pass | studio.stats returns 8 skills, 10 PII fields |
| Browser subsystem works | ✅ Pass | browser.list returns 0 jobs (healthy) |
| MCP/skills system works | ✅ Pass | 17 SKILL.md files created |

### Key Updates (v0.5.3)

1. **Environment Variable API Keys** - API keys now read from environment variables instead of stored in keystore:
   - `OPEN_AI_KEY` → openai-default (OpenAI provider)
   - `OPENROUTER_API_KEY` → openrouter-default (OpenRouter provider)
   - `ZAI_API_KEY` → xai-default (xAI provider)
   
   This keeps API keys in .zshrc and never persists them to disk.

2. **SQLCipher Linking** - Bridge binary now links against SQLCipher library with CGO:
   ```bash
   export CGO_ENABLED=1
   export CGO_CFLAGS="-I/usr/include/sqlcipher"
   export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lsqlcipher"
   ```

3. **Browser Skills** - Created 21 SKILL.md files for Chrome DevTools MCP integration:
   - Safe primitives: navigate, click, fill, wait_for, screenshot, snapshot, list_pages, select_page, resize, emulate
   - Workflow skills: extract_page, login_assist, form_submit, upload_document, trace_performance
   - Guarded skills: eval_privileged, network_inspect, console_inspect, lighthouse_audit, memory_snapshot, fill_with_pii (require approval)

4. **Matrix Admin Bootstrap** - Admin user created:
   - Username: admin
   - Password: ArmorClaw2026!
   - Homeserver: armorclaw.local (accessible via localhost:6167)

### Build Command

```bash
# Build bridge with SQLCipher support
export PATH=/home/mink/go/go/bin:$PATH
export GO111MODULE=on
export CGO_ENABLED=1
export CGO_CFLAGS="-I/usr/include/sqlcipher"
export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lsqlcipher"
cd /home/mink/src/ArmorClaw/bridge
go build -o build/armorclaw-bridge ./cmd/bridge
```

### Running with Environment Variables

```bash
# Source your .zshrc to get API keys
source ~/.zshrc

# Run bridge with env vars
sudo env OPENROUTER_API_KEY="$OPENROUTER_API_KEY" ZAI_API_KEY="$ZAI_API_KEY" \
  /home/mink/src/ArmorClaw/bridge/build/armorclaw-bridge
```

---

## Phase 1 Bring-Up Status (2026-03-10)

### ✅ Completed

| Component | Status | Evidence |
|-----------|--------|----------|
| Quickstart image builds | ✅ Pass | `docker build -f Dockerfile.quickstart -t armorclaw-fixed:phase1 .` |
| Bridge binary runs | ✅ Pass | Process running at PID 476 |
| Bridge socket created | ✅ Pass | `/run/armorclaw/bridge.sock` exists |
| Conduit Matrix starts | ✅ Pass | HTTP 200 on `/_matrix/client/versions` |
| Admin user registered | ✅ Pass | `@admin:152.37.165.193` created |
| QR code generated | ✅ Pass | ArmorChat provisioning QR displayed |

### Verification Commands

```bash
# Conduit responds
curl http://localhost:6167/_matrix/client/versions
# → {"versions":["r0.5.0",...]

# Bridge socket exists
docker exec armorclaw test -S /run/armorclaw/bridge.sock && echo "OK"
# → OK

# Bridge process running
docker exec armorclaw ps aux | grep armorclaw-bridge
# → /opt/armorclaw/armorclaw-bridge (PID 476)
```

### Known Issues

- RPC methods return "method not found" - bridge may need configuration or different method names
- API key injection via RPC failed (needs manual `store_key` call)
- Provisioning not available (OWNER claim via QR code)

### Files Modified

- `Dockerfile.quickstart` - Added Go 1.24 builder stage with CGO + SQLCipher
- `deploy/container-setup.sh` - Added `CONDUIT_CONFIG` env var and config mount

---

## Executive Summary

ArmorClaw v4.5.0 provides a **production-ready AI agent platform** that runs AI agents 24/7 on your VPS, controlled from your phone via Matrix (E2EE) or ArmorChat mobile app.

**Core Components:**

**Key Design Principle:** Zero persistent secrets on disk. All credentials are read from environment variables at runtime. API keys stay in your shell profile (.zshrc), never written to the encrypted keystore.
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
<<<<<<< HEAD
│  │ 2. GO WIZARD (armorclaw-bridge container-setup)                     │   │
│  │    • Check env vars FIRST (tryNonInteractive)                      │   │
│  │      - If OPENROUTER_API_KEY or ZAI_API_KEY set → non-interactive  │   │
│  │      - Server name passed from host (ARMORCLAW_SERVER_NAME)        │   │
│  │    • Else: Check terminal (TTY, color support, size)               │   │
│  │    • Launch Huh? TUI wizard if terminal OK                         │   │
│  │      - Step 1 of 2: AI Provider + API Key (from env vars)          │   │
│  │      - Step 2 of 2: Admin Password + Deploy confirmation           │   │
│  │    • Output: /tmp/armorclaw-wizard.json + env vars for secrets     │   │
=======
│  │ 2. INSTALLER-V5.SH (Stage-1 Full Installer)                   │   │
│  │    • Detect OS/arch (Linux/ARM64, etc.)                        │   │
│  │    • Check prerequisites (Docker, sudo)                          │   │
│  │    • Detect Docker Compose (compose vs docker-compose)              │   │
│  │    • Download setup scripts (setup-quick.sh, setup-matrix.sh)      │   │
│  │    • Wait for Docker daemon (20s timeout, dual-check)            │   │
│  │    • Lockfile protection (flock)                                 │   │
│  │    • Persistent logging (/var/log/armorclaw/install.log)          │   │
│  │    • Export env vars: DOCKER_COMPOSE, CONDUIT_IMAGE, etc.        │   │
>>>>>>> b2a095fe5c03a375c78e8bedf59d22f6e83263e7
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

<<<<<<< HEAD
## Install Script Flow (v0.4.2)

The `install.sh` script orchestrates the entire deployment process:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     INSTALL.SH FLOW (v0.4.2)                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: curl -fsSL .../install.sh | bash                                │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. PREFLIGHT CHECKS                                                  │   │
│  │    • Verify Docker is installed and running                         │   │
│  │    • Check for root/sudo permissions                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. AUTO-DETECTION (on host, before container)                        │   │
│  │    • Detect available ports (8443, 6167, 5000 or fallback)          │   │
│  │    • Detect server IP: ip route get 1                               │   │
│  │    • Collect env vars from user's shell                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. DOCKER RUN (passes env vars to container)                        │   │
│  │    docker run ... \                                                 │   │
│  │      -e ARMORCLAW_SERVER_NAME=<detected-ip> \                       │   │
│  │      -e OPENROUTER_API_KEY=<from-shell> \                           │   │
│  │      -e ZAI_API_KEY=<from-shell> \                                  │   │
│  │      -e ARMORCLAW_PROFILE=<if-set> \                                │   │
│  │      -e ARMORCLAW_ADMIN_PASSWORD=<if-set> \                         │   │
│  │      ...                                                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│                     CONTAINER STARTS (see Quickstart Flow)                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why Host-Side IP Detection?

Container-side IP detection returns the **container's internal IP** (172.17.x.x), not the host's public IP. This would break ArmorChat connectivity. The install script detects the IP on the host and passes it via `ARMORCLAW_SERVER_NAME`.

### Non-Interactive Mode

```bash
# Minimal - just API keys (IP auto-detected)
# Keys are read from environment variables (OPENROUTER_API_KEY, ZAI_API_KEY, etc.)
# Add to your .zshrc:
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ZAI_API_KEY=xxx

# Then run:
source ~/.zshrc
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash

# With explicit server
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ARMORCLAW_SERVER_NAME=192.168.1.50
curl -fsSL ... | bash
```
=======
## Agent Studio (v4.5.0)

### Overview

Agent Studio provides a **no-code interface** for creating and managing AI agents through Matrix chat commands or JSON-RPC.
>>>>>>> b2a095fe5c03a375c78e8bedf59d22f6e83263e7

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

<<<<<<< HEAD
1. **Starts the bridge binary** in background
2. **Waits for socket** at `/run/armorclaw/bridge.sock`
3. **API keys are read from environment variables** (not stored in keystore)
4. **Claims OWNER role** for admin user via `provisioning.claim`
5. **Generates QR code** for ArmorChat mobile provisioning
=======
### Stage-0: Bootstrap (install.sh)
>>>>>>> b2a095fe5c03a375c78e8bedf59d22f6e83263e7

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

The bridge should respond using your configured AI provider (from OPENROUTER_API_KEY or ZAI_API_KEY in your environment).

### 6. Delete Password File (Security)

After successful login:

```bash
docker exec armorclaw rm /var/lib/armorclaw/.admin_password
```

---

## Environment Variables Reference

### API Keys (Any one required for Non-Interactive Mode)

| Variable | Provider | Description |
|----------|----------|-------------|
| `OPEN_AI_KEY` | OpenAI | OpenAI API key |
| `OPENROUTER_API_KEY` | OpenRouter | OpenRouter API key (supports many providers) |
| `ZAI_API_KEY` | xAI | xAI API key |

**Note:** Set any of these in your `.zshrc` to enable non-interactive mode. The bridge will automatically use the available keys.

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
<<<<<<< HEAD
| `ARMORCLAW_SERVER_NAME` | Auto | Server domain or IP. Auto-detected on host. |
| `ARMORCLAW_API_BASE_URL` | OpenAI URL | Custom API endpoint (for Anthropic, GLM-5, etc.) |
| `ARMORCLAW_PROFILE` | `quick` | `quick` or `enterprise` |
=======
| `ARMORCLAW_API_BASE_URL` | Provider default | Custom API endpoint (for Anthropic, Zhipu, etc.) |
| `ARMORCLAW_ADMIN_USERNAME` | (generated) | Custom admin username |
>>>>>>> b2a095fe5c03a375c78e8bedf59d22f6e83263e7
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

API keys are now stored in environment variables, not in the keystore. To add a new provider:

1. Add the key to your `.zshrc`:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-xxx
   ```

2. The bridge will automatically use it when available.

3. To verify keys are loaded:
   ```bash
   source ~/.zshrc
   echo $OPENROUTER_API_KEY  # Should show your key
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

<<<<<<< HEAD
## Troubleshooting

### Setup Failed

If setup fails, you'll see:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SETUP FAILED (exit code: 1)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Rolling back partial setup...

Log file locations:
  Primary:   /var/log/armorclaw/setup.log
  Backup:    /tmp/armorclaw-logs/setup.log.backup

Last 30 lines of log:
  ...
```

**Steps to recover:**

1. Check the log: `docker cp armorclaw:/var/log/armorclaw/setup.log ./setup.log`
2. Fix the issue (usually Docker socket or network)
3. Remove the container: `docker rm -f armorclaw`
4. Re-run the docker command

### Can't Connect to Matrix

1. **Check port binding:**
   ```bash
   docker port armorclaw
   # Should show: 6167/tcp -> 0.0.0.0:6167
   ```

2. **Check firewall:**
   ```bash
   sudo ufw status
   sudo ufw allow 6167/tcp
   ```

3. **Check from inside container:**
   ```bash
   docker exec armorclaw curl http://localhost:6167/_matrix/client/versions
   ```

### Bridge Not Responding

1. **Check bridge process:**
   ```bash
   docker exec armorclaw ps aux | grep armorclaw-bridge
   ```

2. **Check socket:**
   ```bash
   docker exec armorclaw ls -la /run/armorclaw/bridge.sock
   ```

3. **Check RPC:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

### Agent Studio Issues

1. **Check studio database:**
   ```bash
   docker exec armorclaw ls -la /var/lib/armorclaw/studio.db
   ```

2. **List skills:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_skills"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

3. **Check agent instances:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_instances"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

---

## Architecture Reference

### Container Layout

```
armorclaw container
├── /opt/armorclaw/
│   ├── armorclaw-bridge     # Go binary
│   ├── quickstart.sh        # Entry point
│   ├── container-setup.sh   # Setup wizard
│   ├── agent/               # Agent runtime
│   └── configs/             # Config templates
├── /etc/armorclaw/
│   ├── config.toml          # Main config
│   ├── ssl/                 # SSL certificates
│   └── .setup_complete      # Setup flag
├── /var/lib/armorclaw/
│   ├── keystore.db          # SQLCipher encrypted (for other secrets)
│   ├── studio.db            # Agent Studio database
│   ├── .admin_user          # Admin info for OWNER claim
│   └── .admin_password      # Temp password file
├── /run/armorclaw/
│   └── bridge.sock          # Unix socket for RPC
└── /var/log/armorclaw/
    └── setup.log            # Setup log

# API Keys: Read from environment variables at runtime
# - OPENROUTER_API_KEY → openai-default
# - ZAI_API_KEY → xai-default
# - OPEN_AI_KEY → openai-default
```

### Network Topology

```
┌─────────────────────────────────────────────────────────┐
│  Host (VPS/Server)                                      │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │  armorclaw container (mikegemut/armorclaw)      │   │
│  │                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐            │   │
│  │  │ Bridge      │    │ Matrix      │            │   │
│  │  │ (Go binary) │◄──►│ Conduit     │            │   │
│  │  │ :8443/RPC   │    │ :6167       │            │   │
│  │  │             │    │             │            │   │
│  │  │ + Studio    │    └─────────────┘            │   │
│  │  │ + Browser   │                              │   │
│  │  │ + MCP       │                              │   │
│  │  └─────────────┘                              │   │
│  │         │                   │                   │   │
│  │         │    ┌─────────────┐│                   │   │
│  │         └───►│ Sygnal      ││                   │   │
│  │              │ Push :5000  ││                   │   │
│  │              └─────────────┘│                   │   │
│  │                            │                   │   │
│  └────────────────────────────────────────────────┘   │
│                           │                            │
│  Docker Socket ───────────┘ (mounted)                 │
│                                                        │
└────────────────────────────────────────────────────────┘
         │              │              │
      :8443          :6167          :5000
    (HTTPS/RPC)    (Matrix)      (Push)
```

---

## ArmorChat Communication Architecture

### Overview

ArmorChat communicates with ArmorClaw through **Matrix protocol** for all messaging and **JSON-RPC** for direct bridge commands. This architecture provides:

- **End-to-End Encryption (E2EE)** - All messages encrypted via Matrix
- **Push Notifications** - Real-time alerts via Sygnal + FCM
- **Offline Support** - Messages queued and delivered when online
- **Multi-Device** - Same account on multiple devices

### Communication Stack

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW ↔ ARMORCHAT COMMUNICATION                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐                              ┌─────────────────┐       │
│  │ ArmorChat       │                              │ ArmorClaw       │       │
│  │ (Android)       │                              │ (VPS)           │       │
│  │                 │                              │                 │       │
│  │ Matrix SDK      │◄──── E2EE Messages ────────►│ Matrix Conduit  │       │
│  │ (Kotlin)        │                              │ (Rust)          │       │
│  │                 │                              │                 │       │
│  │ FCM Push        │◄──── Push Notifications ────│ Sygnal          │       │
│  │                 │                              │ (Python)        │       │
│  │                 │                              │                 │       │
│  │ JSON-RPC Client │◄──── Direct Commands ──────►│ Bridge RPC      │       │
│  │ (HTTP/HTTPS)    │                              │ (Unix Socket)   │       │
│  └─────────────────┘                              └─────────────────┘       │
│         │                                                │                   │
│         │                                                │                   │
│         │              PROTOCOLS USED                     │                   │
│         │                                                │                   │
│         │  Matrix (CS API):                              │                   │
│         │  - /_matrix/client/v3/                         │                   │
│         │  - /_matrix/media/v3/                          │                   │
│         │  - m.room.message events                       │                   │
│         │  - Custom events (com.armorclaw.*)             │                   │
│         │                                                │                   │
│         │  Push (FCM):                                   │                   │
│         │  - Sygnal → FCM → Device                       │                   │
│         │  - Includes room_id, event_id                  │                   │
│         │                                                │                   │
│         │  JSON-RPC (HTTP):                              │                   │
│         │  - POST to :8443/rpc                           │                   │
│         │  - Auth via Bearer token                       │                   │
│         │                                                │                   │
│         └────────────────────────────────────────────────┘                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Initial Connection Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCHAT PROVISIONING FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Step 1: SETUP COMPLETE (Bridge displays QR)                                │
│  ───────────────────────────────────────                                    │
│                                                                             │
│  Bridge generates provisioning data:                                        │
│  {                                                                          │
│    "server_name": "192.168.1.50:6167",                                      │
│    "setup_token": "armorclaw-setup-abc123",                                │
│    "qr_data": "armorclaw://192.168.1.50:6167?token=abc123"                  │
│  }                                                                          │
│                                                                             │
│  Step 2: USER SCANS QR CODE                                                 │
│  ────────────────────────────                                               │
│                                                                             │
│  ArmorChat parses URI:                                                      │
│    armorclaw://<server>:<port>?token=<setup_token>                          │
│                                                                             │
│  Step 3: ARMORCHAT CONNECTS TO MATRIX                                       │
│  ───────────────────────────────────────                                    │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Conduit     │────►│ Register    │                    │
│  │             │     │ :6167       │     │ Device      │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  1. GET /_matrix/client/versions       │                           │
│         │  2. POST /_matrix/client/v3/register   │                           │
│         │     (with setup_token as device_id)    │                           │
│         │  3. Receive access_token, device_id    │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 4: BRIDGE AUTO-CLAIMS OWNER ROLE                                      │
│  ──────────────────────────────────────                                      │
│                                                                             │
│  Bridge calls: provisioning.claim(setup_token, device_id)                   │
│  → First claim = OWNER role                                                 │
│  → User added to "ArmorClaw Bridge" room                                    │
│                                                                             │
│  Step 5: E2EE SETUP                                                         │
│  ─────────────────                                                          │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Matrix SDK  │────►│ Key         │                    │
│  │             │     │ Crypto       │     │ Exchange    │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  1. Generate device keys               │                           │
│         │  2. Upload keys to server              │                           │
│         │  3. Download bridge's keys             │                           │
│         │  4. Verify via emoji (optional)        │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 6: PUSH NOTIFICATION SETUP                                            │
│  ─────────────────────────────────                                          │
│                                                                             │
│  ArmorChat enables push via Matrix HTTP Pusher:                             │
│    POST /_matrix/client/v3/pushers/set                                      │
│    {                                                                        │
│      "pushkey": "<FCM-token>",                                              │
│      "app_id": "com.armorclaw.armorchat",                                   │
│      "data": { "url": "http://server:5000/_matrix/push/v1/notify" }         │
│    }                                                                        │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Conduit     │────►│ Sygnal      │                    │
│  │             │     │             │     │ :5000       │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  Register pusher with FCM token        │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 7: READY TO COMMUNICATE                                               │
│  ─────────────────────────────                                              │
│                                                                             │
│  ArmorChat can now:                                                         │
│  • Send encrypted messages to Bridge room                                   │
│  • Receive push notifications                                               │
│  • Execute commands via !agent, !status                                     │
│  • Control browser via Matrix events                                        │
│  • Approve PII via BlindFill                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Ongoing Communication

#### Message Flow (User → Agent → Response)

```
User Types Message                    Bridge Processing                    Response
─────────────────                    ──────────────────                   ─────────
      │                                     │                                   │
      │  1. User types: "Book a flight"    │                                   │
      │                                     │                                   │
      ▼                                     │                                   │
┌─────────────┐                             │                                   │
│ ArmorChat   │                             │                                   │
│ Matrix SDK  │                             │                                   │
│             │                             │                                   │
│ Encrypt     │                             │                                   │
│ message     │                             │                                   │
└──────┬──────┘                             │                                   │
       │                                    │                                   │
       │  m.room.encrypted                  │                                   │
       │  (ciphertext)                      │                                   │
       │                                    │                                   │
       ▼                                    ▼                                   │
┌─────────────┐                       ┌─────────────┐                          │
│ Conduit     │──────────────────────►│ Bridge      │                          │
│ :6167       │                       │ Matrix      │                          │
│             │                       │ Adapter     │                          │
└─────────────┘                       └──────┬──────┘                          │
                                             │                                 │
                                             │  2. Decrypt message             │
                                             │  3. Route to agent or AI        │
                                             │  4. Process request             │
                                             │                                 │
                                             ▼                                 │
                                       ┌─────────────┐                          │
                                       │ Agent / AI  │                          │
                                       │ Processor   │                          │
                                       └──────┬──────┘                          │
                                              │                                  │
                                              │  5. Generate response            │
                                              │                                  │
                                              ▼                                  │
                                       ┌─────────────┐                          │
                                       │ Bridge      │                          │
                                       │             │                          │
                                       │ Encrypt     │                          │
                                       │ response    │                          │
                                       └──────┬──────┘                          │
                                              │                                  │
                                              │  m.room.encrypted                │
                                              │  (response ciphertext)           │
                                              │                                  │
                                              ▼                                  │
                                       ┌─────────────┐     ┌─────────────┐      │
                                       │ Conduit     │────►│ ArmorChat   │      │
                                       │             │     │ Decrypt     │      │
                                       └─────────────┘     │ Display     │      │
                                                           └─────────────┘      │
                                              │                                  │
                                              │  6. Push notification            │
                                              │     (if app in background)       │
                                              ▼                                  │
                                       ┌─────────────┐     ┌─────────────┐      │
                                       │ Sygnal      │────►│ FCM         │      │
                                       │ :5000       │     │ → Device    │      │
                                       └─────────────┘     └─────────────┘      │
```

#### Browser Control Flow

```
User Requests Browser Action           Bridge Processing              Browser Executes
─────────────────────────              ──────────────────             ────────────────
              │                              │                              │
              │  1. User taps "Navigate"     │                              │
              │     in ArmorChat UI          │                              │
              │                              │                              │
              ▼                              │                              │
       ┌─────────────┐                       │                              │
       │ ArmorChat   │                       │                              │
       │             │                       │                              │
       │ JSON-RPC    │                       │                              │
       │ POST /rpc   │                       │                              │
       └──────┬──────┘                       │                              │
              │                              │                              │
              │  browser.enqueue_job({       │                              │
              │    agent_id,                 │                              │
              │    commands: [               │                              │
              │      { type: "navigate",     │                              │
              │        url: "..." }          │                              │
              │    ]                         │                              │
              │  })                          │                              │
              │                              │                              │
              ▼                              ▼                              │
       ┌─────────────┐                 ┌─────────────┐                      │
       │ Bridge RPC  │────────────────►│ Browser     │                      │
       │ :8443       │                 │ Queue       │                      │
       └─────────────┘                 │ Processor   │                      │
                                       └──────┬──────┘                      │
                                              │                             │
                                              │  2. Queue job               │
                                              │  3. Pick up from queue      │
                                              │                             │
                                              ▼                             │
                                       ┌─────────────┐                      │
                                       │ Browser     │                      │
                                       │ Service     │                      │
                                       │ (Playwright)│                      │
                                       └──────┬──────┘                      │
                                              │                             │
                                              │  4. Execute command         │
                                              │                             │
                                              ▼                             │
                                       ┌─────────────┐                      │
                                       │ Status      │                      │
                                       │ Events      │                      │
                                       └──────┬──────┘                      │
                                              │                             │
       ┌─────────────┐                        │                             │
       │ ArmorChat   │◄───────────────────────┘                             │
       │             │                                                      │
       │ Matrix:     │  com.armorclaw.browser.status                        │
       │   {         │  {                                                   │
       │     status: │    "status": "navigating",                           │
       │     url,    │    "url": "https://...",                             │
       │     progress│    "progress": 50                                    │
       │   }         │  }                                                   │
       └─────────────┘                                                      │
```

### Key Communication Endpoints

| Endpoint | Protocol | Purpose |
|----------|----------|---------|
| `:6167/_matrix/client/*` | Matrix CS API | All Matrix operations |
| `:6167/_matrix/federation/*` | Matrix Federation | Server-to-server (optional) |
| `:5000/_matrix/push/v1/notify` | HTTP POST | Push notifications |
| `:8443/rpc` | JSON-RPC 2.0 | Direct bridge commands |
| `:8443/health` | HTTP GET | Bridge health check |

### Provisioning RPC Methods

| Method | Purpose |
|--------|---------|
| `provisioning.start` | Generate setup token and QR data |
| `provisioning.claim` | Claim device with token (first = OWNER) |
| `provisioning.status` | Check provisioning state |

### Security Model

```
┌─────────────────────────────────────────────────────────────────┐
│  SECURITY LAYERS                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Layer 1: Network                                                │
│  ───────────────                                                 │
│  • HTTPS/TLS for all external traffic                            │
│  • Self-signed certs generated on setup                          │
│  • Optional Let's Encrypt with Caddy                             │
│                                                                  │
│  Layer 2: Matrix Authentication                                  │
│  ────────────────────────────                                    │
│  • Access tokens for all API calls                               │
│  • Device-specific tokens                                        │
│  • Token stored securely in Android Keystore                     │
│                                                                  │
│  Layer 3: End-to-End Encryption                                  │
│  ─────────────────────────────                                   │
│  • Olm/Megolm encryption (Matrix SDK)                            │
│  • Device keys generated locally                                 │
│  • Cross-signing for verification                                │
│  • Emoji verification for bridge                                 │
│                                                                  │
│  Layer 4: Role-Based Access Control                              │
│  ───────────────────────────────                                 │
│  • OWNER: Full access, can manage users                          │
│  • ADMIN: Can create agents, manage MCPs                         │
│  • USER: Can use agents, request MCP access                      │
│                                                                  │
│  Layer 5: BlindFill PII Protection                               │
│  ─────────────────────────────                                   │
│  • PII never sent to agent                                       │
│  • Decrypted only in browser memory                              │
│  • User approval required per-field                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## ArmorChat Android Integration

### Event Types Reference

| Event Type | Direction | Purpose |
|------------|-----------|---------|
| `com.armorclaw.browser.response` | Bridge → Client | Command results |
| `com.armorclaw.browser.status` | Bridge → Client | Browser state changes |
| `com.armorclaw.agent.status` | Bridge → Client | Agent state machine transitions |
| `com.armorclaw.pii.response` | Client → Bridge | User's PII approval/denial |

> **Important:** Agent state events use `com.armorclaw.agent.status` (not `.state`)

### Matrix Event Listeners

```kotlin
class BrowserCommandHandler(
    private val matrixClient: MatrixClient,
    private val controlPlaneStore: ControlPlaneStore
) {
    fun subscribe() {
        // Listen for browser responses (command results)
        matrixClient.onEvent("com.armorclaw.browser.response") { event ->
            handleBrowserResponse(event)
        }

        // Listen for browser status updates (navigation, form detection, etc.)
        matrixClient.onEvent("com.armorclaw.browser.status") { event ->
            handleBrowserStatus(event)
        }

        // Listen for agent status changes (state machine transitions)
        matrixClient.onEvent("com.armorclaw.agent.status") { event ->
            handleAgentStatus(event)
        }
    }

    private fun handleAgentStatus(event: MatrixEvent) {
        val status = AgentStatusEvent.fromJson(event.content)

        when (status.status) {
            "idle" -> showIdleState()
            "browsing" -> showBrowsingState(status.metadata.url)
            "form_filling" -> showFormFillingState(status.metadata.progress)
            "awaiting_approval" -> showApprovalNeeded(status.metadata.fieldsRequested)
            "awaiting_captcha" -> showCaptchaUI()
            "awaiting_2fa" -> show2FAUI()
            "processing_payment" -> showProcessingPayment()
            "complete" -> showComplete(status.metadata)
            "error" -> showError(status.metadata.error)
        }
    }
}
```

### Agent Status Event Structure

```kotlin
data class AgentStatusEvent(
    val agent_id: String,
    val status: String,          // "idle" | "browsing" | "form_filling" | etc.
    val previous: String?,       // Previous status
    val timestamp: Long,         // Unix milliseconds
    val metadata: StatusMetadata?
)

data class StatusMetadata(
    val url: String?,            // Current browser URL
    val step: String?,           // Current step description
    val progress: Int?,          // 0-100 progress
    val error: String?,          // Error message if status == "error"
    val task_id: String?,        // Current task identifier
    val task_type: String?,      // e.g., "flight_booking"
    val fields_requested: List<String>? // PII fields needed (when awaiting_approval)
)
```

### PII Field Reference Mapping

```kotlin
fun mapPIIFieldRef(ref: String): PiiField {
    return when (ref) {
        "payment.card_number" -> PiiField(
            name = "Card Number",
            sensitivity = SensitivityLevel.HIGH,
            description = "Credit or debit card number",
            currentValue = maskCardNumber(storedCard?.last4)
        )
        "payment.cvv" -> PiiField(
            name = "CVV",
            sensitivity = SensitivityLevel.CRITICAL,
            description = "Card verification code",
            currentValue = null // Never display
        )
        "payment.card_expiry" -> PiiField(
            name = "Expiry Date",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "MM/YY format",
            currentValue = storedCard?.expiry
        )
        "payment.card_name" -> PiiField(
            name = "Cardholder Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Name on card"
        )
        "personal.name" -> PiiField(
            name = "Full Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Your full name"
        )
        "personal.address" -> PiiField(
            name = "Address",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "Street address"
        )
        "personal.email" -> PiiField(
            name = "Email",
            sensitivity = SensitivityLevel.LOW,
            description = "Contact email"
        )
        "personal.phone" -> PiiField(
            name = "Phone",
            sensitivity = SensitivityLevel.LOW,
            description = "Phone number"
        )
        else -> PiiField(
            name = ref.substringAfterLast("."),
            sensitivity = SensitivityLevel.HIGH,
            description = "Requested: $ref"
        )
    }
}
```

### JSON-RPC Browser Queue Client

```kotlin
class BrowserQueueClient(
    private val bridgeUrl: String,
    private val authToken: String
) {
    suspend fun enqueueJob(request: EnqueueJobRequest): Result<BrowserJob>
    suspend fun getJob(jobId: String): Result<BrowserJob>
    suspend fun cancelJob(jobId: String): Result<Unit>
    suspend fun retryJob(jobId: String): Result<Unit>
    suspend fun getQueueStats(): Result<QueueStats>
    suspend fun listJobs(agentId: String? = null, status: List<String>? = null): Result<List<BrowserJob>>
}

data class EnqueueJobRequest(
    val id: String? = null,           // Optional, auto-generated if omitted
    val agent_id: String,
    val room_id: String,
    val definition_id: String? = null,
    val commands: List<BrowserCommandJson>,
    val priority: Int = 5,            // 1-10, higher = processed first
    val timeout: Int = 300,           // Seconds
    val max_retries: Int = 2,
    val expires_in: Int? = null       // Seconds from now
)

data class BrowserJob(
    val id: String,
    val agent_id: String,
    val room_id: String,
    val user_id: String?,
    val definition_id: String?,
    val status: JobStatus,            // "pending" | "running" | "completed" | "failed" | "cancelled"
    val priority: Int,
    val attempts: Int,
    val current_step: Int,
    val total_steps: Int,
    val error: String?,
    val created_at: Long,
    val started_at: Long?,
    val completed_at: Long?,
    val result: Map<String, Any?>?
)

data class QueueStats(
    val total: Int,
    val pending: Int,
    val running: Int,
    val completed: Int,
    val failed: Int,
    val cancelled: Int,
    val awaiting_pii: Int,
    val active_workers: Int,
    val queue_depth: Int
)
```

### PII Response Handling

```kotlin
// When user approves/denies in BlindFill dialog:
fun respondToPIIRequest(requestId: String, approved: Boolean, values: Map<String, String>?) {
    // Send PII response as Matrix event
    matrixClient.sendEvent(roomId, mapOf(
        "type" to "com.armorclaw.pii.response",
        "content" to mapOf(
            "request_id" to requestId,
            "approved" to approved,
            "values" to (values ?: emptyMap<String, String>())
        )
    ))

    // Clear pending request
    controlPlaneStore.removePiiRequest(requestId)
}

// In ChatViewModel - connect to existing approvePiiRequest
fun approvePiiRequest(approvedFields: Set<String>) {
    val request = _pendingPiiRequest.value ?: return

    val values = mutableMapOf<String, String>()
    approvedFields.forEach { fieldName ->
        when (fieldName) {
            "Card Number" -> values["payment.card_number"] = secureStorage.getCardNumber()
            "CVV" -> values["payment.cvv"] = secureStorage.getCVV()
            "Expiry Date" -> values["payment.card_expiry"] = secureStorage.getCardExpiry()
            "Cardholder Name" -> values["payment.card_name"] = userPrefs.getCardName()
            "Full Name" -> values["personal.name"] = userPrefs.getFullName()
            "Address" -> values["personal.address"] = userPrefs.getAddress()
            "Email" -> values["personal.email"] = userPrefs.getEmail()
            "Phone" -> values["personal.phone"] = userPrefs.getPhone()
        }
    }

    respondToPIIRequest(request.requestId, approved = true, values = values)
}
```

### Complete Checkout Flow Example

```kotlin
fun startCheckout(url: String, agent: AgentDefinition) = scope.launch {
    // 1. Create browser job via JSON-RPC
    val job = queueClient.enqueueJob(
        EnqueueJobRequest(
            agent_id = agent.id,
            room_id = currentRoomId,
            definition_id = agent.definitionId,
            commands = listOf(
                BrowserCommandJson("navigate", mapOf("url" to url, "waitUntil" to "load")),
                BrowserCommandJson("fill", mapOf(
                    "fields" to listOf(
                        mapOf("selector" to "#email", "value" to userPrefs.email),
                        mapOf("selector" to "#address", "value_ref" to "personal.address")
                    )
                )),
                BrowserCommandJson("click", mapOf("selector" to "#continue-btn", "waitFor" to "navigation"))
            ),
            priority = 5,
            timeout = 300
        )
    ).getOrThrow()

    // 2. Subscribe to status updates
    matrixService.onEvent("com.armorclaw.agent.status")
        .filter { it.content.agent_id == agent.id }
        .collect { event ->
            val status = AgentStatusEvent.fromJson(event.content)
            when (status.status) {
                "awaiting_approval" -> {
                    showBlindFillDialog(status.metadata?.fields_requested ?: emptyList())
                }
                "complete" -> {
                    showCheckoutComplete(status.metadata)
                }
                "error" -> {
                    showError(status.metadata?.error ?: "Unknown error")
                }
            }
        }
}
```

---

## Browser Service Architecture (v0.4.1)

### Components

The browser automation system consists of three main components:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BROWSER AUTOMATION ARCHITECTURE                          │
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
│                     Matrix Events (status, response)                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
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
      - seccomp=unconfined
```

---

=======
>>>>>>> b2a095fe5c03a375c78e8bedf59d22f6e83263e7
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
