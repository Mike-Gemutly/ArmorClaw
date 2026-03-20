# ArmorClaw: The VPS Secretary Platform

[![Version](https://img.shields.io/badge/version-v4.6.0-blue)](https://github.com/Gemutly/ArmorClaw)
[![Status](https://img.shields.io/badge/status-production%20ready-green)](https://github.com/Gemutly/ArmorClaw)

**Run AI agents on your VPS. Control from your phone.**

ArmorClaw runs AI agents 24/7 on your server. They browse websites, fill forms, and manage tasks—while you approve sensitive actions via your phone.

**🛡️ v4.6.0 Highlights:** **Environment Variable API Keys** (keys stored in .zshrc, never persisted to disk), **SQLCipher-linked bridge**, and **21 Browser Skills** for Chrome DevTools MCP integration.

---

## Quick Install (All-in-One)

The recommended way to start is the bootstrap installer, which now includes an optional **Matrix/Conduit** setup for instant QR provisioning:

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**v4.6.0 Improvements:**
- **🔐 Environment API Keys** - Keys from env vars (OPENROUTER_API_KEY, ZAI_API_KEY, OPEN_AI_KEY), never persisted to disk
- **🔗 SQLCipher Linked** - Bridge binary links against SQLCipher for encrypted keystore
- **🌐 21 Browser Skills** - Chrome DevTools MCP skills (navigate, click, fill, screenshot, etc.)
- **🛡️ Hardened Security** - GPG-verified bootstrap, lockfile protection, and persistent logging

### Install Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | VERSION=v4.6.0 bash
```

---

## Non-Interactive Setup (CI/CD)

API keys are read from environment variables - set these in your shell profile (.zshrc):

```bash
# Add to your .zshrc
export OPENROUTER_API_KEY=sk-or-v1-xxx   # For OpenRouter (recommended)
export ZAI_API_KEY=xxx                   # For xAI (Grok)
export OPEN_AI_KEY=xxx                   # For OpenAI

# Optional settings
export ARMORCLAW_ADMIN_USERNAME=admin    # Custom admin username
export CONDUIT_VERSION=v1.0.0            # Specify Conduit version

# Run installer
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh -o /tmp/install.sh
sudo bash /tmp/install.sh
```

**v4.4.0+ Installer Features:**

The production-grade installer (install.sh) now includes:

- **🛡️ Lockfile Protection** - Prevents parallel installs via flock
- **⏳ Docker Daemon Readiness** - Dual-check (info + ps) with 20-second timeout
- **📝 Persistent Logging** - Logs to /var/log/armorclaw/install.log (fallback to /tmp)
- **🚀 Docker Compose Detection** - Supports both `docker compose` and `docker-compose`
- **📦 Environment Passthrough** - All vars passed to setup scripts
- **🖼️ Conduit Image Control** - Via CONDUIT_VERSION environment variable
- **✅ Comprehensive Tests** - 8 integration tests covering all features

**Environment Variables:**
```bash
# API Keys (any one required - stored in .zshrc, never persisted to disk)
OPENROUTER_API_KEY         # OpenRouter (recommended - supports many providers)
OPEN_AI_KEY               # OpenAI
ZAI_API_KEY               # xAI (Grok)

# Optional
ARMORCLAW_ADMIN_USERNAME  # Custom admin username (optional)
ARMORCLAW_ADMIN_PASSWORD  # Custom admin password (optional)
CONDUIT_VERSION           # Conduit version (default: latest)
CONDUIT_IMAGE             # Custom image (default: matrixconduit/matrix-conduit:latest)
DOCKER_COMPOSE           # Docker Compose command (auto-detected)
INSTALL_MODE              # quick | matrix (default: quick)
```

Run tests:
```bash
bash tests/integration/test-installer-hardening.sh
```

### Production-Grade Bootstrap

ArmorClaw v4.3.1+ includes **zero-touch admin creation** with production-grade security:

- **No manual registration** - Admin user created automatically via shared-secret API
- **No open registration window** - `allow_registration = false` at all times
- **Randomized usernames** - Generates `armor-admin-xxxxxxxx` by default (negligible collision risk)
- **One-time password display** - Password shown once, never persisted to disk
- **Conflict detection** - Automatically retries with alternative username if conflict detected
- **Input validation** - Environment variables sanitized to prevent injection attacks

See [Production Deployment Guide](docs/guides/production-grade-deployment.md) for details.

### What You'll See

```
╔══════════════════════════════════════════════════════╗
║        ArmorClaw Quick Setup                         ║
╚══════════════════════════════════════════════════════╝

Step 1 of 2: AI Provider Configuration
  AI Provider: [OpenAI ▾]
              (OpenAI, Anthropic, GLM-5, or Custom)
  API Key: ••••••••

Step 2 of 2: Admin & Deployment
  Admin Password: (press Enter to auto-generate)
  Ready to deploy? [Yes]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ArmorClaw is Ready!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Bridge:  http://192.168.1.50:8443
Matrix:  http://192.168.1.50:6167
Admin:   admin / <generated-password>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ArmorChat Mobile App Connection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Configuration:
  Server:    192.168.1.50
  Port:      8443
  Matrix:    http://192.168.1.50:6167
  Valid:     24 hours

Deep Link: armorclaw://config?d=...

██████████████████████████████████
██████████████████████████████████
██ ▄▄▄▄▄ █▀▄█▀▄▀█ ▄▀█ █ ▄▄▄▄▄ ██
... (ASCII QR - scan with ArmorChat)
```

### Next: Connect Your Phone

1. **Install ArmorChat** from Google Play
2. **Scan the QR code** displayed in terminal
3. **Set up biometrics** for secure keystore access
4. **Done** - your digital secretary is online

---

## Your First Task

Once connected via ArmorChat:

```
!agent create name="Researcher" skills="web_browsing"
```

The Bridge provisions an isolated agent and invites you to its room.

Then ask it to do something:

```
Research the best restaurants in NYC for a birthday dinner
```

Watch it browse, gather information, and report back.

---

## How It Works

```
┌─────────────┐
│   Phone     │  Your pocket
│ ArmorChat   │
└──────┬──────┘
       │ End-to-end encrypted (Matrix)
       ▼
┌─────────────┐
│   Bridge    │  Your VPS
│ Orchestrator│
└──────┬──────┘
       │ Secure RPC
       ▼
┌─────────────┐
│    Agent    │  Isolated container
│  OpenClaw   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Browser    │  Automated tasks
│ Playwright  │
└─────────────┘
```

**Key components:**
- **ArmorClaw Bridge** - Orchestrator on your VPS
- **OpenClaw** - Agent runtime (runs in isolated containers)
- **ArmorChat** - Mobile control app

---

## Why ArmorClaw?

### Traditional AI Agents See Your Passwords

When you give an AI agent your credit card or password, it can log, store, or leak that data.

### ArmorClaw Agents Never See Your Secrets

**BlindFill™** injects secrets directly into the browser. The agent requests "credit card" but never sees the actual number—it goes straight from encrypted storage to the form field.

```
Agent says:     "I need payment.card_number"
Bridge checks:  User approved? ✓
Bridge injects: 4242... → Browser form
Agent sees:     (nothing - it's blind)
```

This is why ArmorClaw is safe for sensitive tasks.

---

## Production-Grade Installer (v4.4.0)

ArmorClaw v4.4.0+ includes a hardened installer with production-grade reliability:

### Security Features

| Feature | Implementation |
|---------|----------------|
| **Lockfile** | Uses `flock` to prevent parallel installs |
| **Docker Readiness** | Dual-check (docker info + docker ps) with 20s timeout |
| **Persistent Logs** | `/var/log/armorclaw/install.log` with fallback |
| **Compose Detection** | Auto-detects `docker compose` vs `docker-compose` |
| **Env Passthrough** | All vars exported to sub-scripts |
| **Version Control** | `CONDUIT_VERSION` environment variable |

### Idempotent & Safe

- **Safe to re-run** - Detects existing containers, never duplicates
- **Migration support** - Migrates from old `/var/lib/matrix-conduit` paths
- **Graceful failures** - Clear error messages, logs to stderr
- **Self-contained** - Fallbacks prevent installation failures

### Test Coverage

Comprehensive test suite with 8 tests:
- Lockfile functionality
- Docker wait loop
- Environment variable passthrough
- Docker Compose detection
- CONDUIT_IMAGE fallback
- Syntax validation
- Wait function existence
- Variable ordering

**All tests passing:** ✅

Run tests:
```bash
bash tests/integration/test-installer-hardening.sh
```

---

## Supported AI Providers (v4.5.0+)

> **Full model list**: See [docs/reference/models.md](docs/reference/models.md) for all available models per provider.

ArmorClaw now uses a **Provider Registry** pattern. The installer provides a selection menu, and the Bridge resolves aliases (like `zai` or `glm`) automatically.

### Included in Registry:
| Provider | ID | Protocol | Description |
|----------|----|----------|-------------|
| **OpenAI** | `openai` | openai | GPT-4o, o1, etc. |
| **Anthropic** | `anthropic` | anthropic | Claude 3.5 Sonnet/Opus |
| **Zhipu AI (Z AI)** | `zhipu` | openai | `api.z.ai` (aliases: `zai`, `glm`) |
| **DeepSeek** | `deepseek` | openai | DeepSeek R1, V3 |
| **Moonshot AI** | `moonshot` | openai | Moonshot/Kimi |
| **Google Gemini** | `google` | openai | Gemini 1.5 Pro/Flash |
| **xAI** | `xai` | openai | Grok-1, Grok-2 |
| **OpenRouter** | `openrouter` | openai | Multi-provider aggregator |
| **Groq** | `groq` | openai | Ultra-fast Llama/Mixtral |
| **NVIDIA NIM** | `nvidia` | openai | NVIDIA-hosted models |
| **Cloudflare** | `cloudflare` | openai | AI Gateway |
| **Ollama** | `ollama` | openai | Local models via `localhost:11434` |

### Adding API Keys

API keys are read from environment variables at runtime - no need to store them in the keystore:

```bash
# Add to your .zshrc or environment
export OPENROUTER_API_KEY=sk-or-v1-xxx   # Recommended - access to many providers
export ZAI_API_KEY=xxx                   # xAI (Grok)
export OPEN_AI_KEY=xxx                   # OpenAI

# The bridge automatically uses available keys
# No CLI command needed - keys are read from environment at startup
```

### Adding Custom Providers:

You can add any OpenAI-compatible provider by editing `/etc/armorclaw/providers.json`. See the [Provider Registry Guide](docs/reference/provider-registry.md) for details.

### Dynamic Provider Discovery (Catwalk)

ArmorClaw integrates with [Catwalk](https://github.com/charmbracelet/catwalk) for dynamic provider and model discovery:

- **Quickstart** automatically discovers providers from the local Catwalk registry
- **Runtime switching** via Matrix commands — no restart needed
- **Fallback** to hardcoded list if Catwalk unavailable

**Matrix Commands:**
```
/ai              — Show help
/ai providers    — List available providers
/ai models <p>  — List models for a provider
/ai switch <p> <m> — Switch provider and model
/ai status      — Show current configuration
```

Example:
```
/ai switch openai gpt-4o
```

---

## System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| CPU | 1 core | 2+ cores |
| RAM | 2 GB | 4 GB |
| Disk | 10 GB | 20 GB |
| OS | Linux (Ubuntu 20.04+) | Ubuntu 22.04 |
| Docker | 24.0+ | Latest |

---

## Installation Options

### Option 1: Quickstart (Recommended for First-Time Users)

The quickstart container includes everything in one image:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh)"
```

**What it includes:**
- ArmorClaw Bridge
- Matrix Conduit homeserver
- Automatic user registration
- QR code for mobile app setup

### Option 2: Production Deployment (Docker Compose)

For production, use the full stack compose file with all services as peers:

```bash
# Clone the repository
git clone https://github.com/Gemutly/ArmorClaw.git
cd ArmorClaw

# Create config directory
mkdir -p configs

# Copy example configs (or create your own)
cp configs/conduit.toml.example configs/conduit.toml 2>/dev/null || true

# Set your server IP/domain and API keys
export MATRIX_DOMAIN=your-server.example.com
export OPENROUTER_API_KEY=sk-or-v1-xxx  # Add to .zshrc for persistence

# Start the stack
docker compose -f docker-compose-full.yml up -d
```

**Services started:**
- `matrix` - Conduit homeserver (port 6167)
- `sygnal` - Push notifications (port 5000)
- `caddy` - Reverse proxy with auto-SSL (ports 80, 443)
- `bridge` - ArmorClaw orchestrator (port 8443)

**Create the bridge Matrix user:**
```bash
curl -X POST http://localhost:6167/_matrix/client/v3/register \
  -H "Content-Type: application/json" \
  -d '{"username":"bridge","password":"bridgepass","auth":{"type":"m.login.dummy"}}'
```

### Option 3: Bridge-Only (No Matrix)

For testing or when using an external Matrix server:

```bash
# Using quickstart with --bridge-only flag
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash -s -- --bridge-only

# Or with external Matrix (add keys to .zshrc first)
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ARMORCLAW_EXTERNAL_MATRIX=true
export ARMORCLAW_MATRIX_HOMESERVER_URL=http://your-matrix-server:6167
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### Deployment Modes Summary

| Mode | Command | Use Case |
|------|---------|----------|
| **Quickstart** | `bash install.sh` | First-time users, all-in-one |
| **Production** | `docker compose -f docker-compose-full.yml up -d` | Production, scalable |
| **Bridge-only** | `bash install.sh --bridge-only` | Testing, external Matrix |
| **External Matrix** | `ARMORCLAW_EXTERNAL_MATRIX=true` | Use existing Matrix server |

### Non-Interactive (CI/CD)

```bash
# Minimal - auto-detects server IP (add API keys to .zshrc)
# Recommended: use OPENROUTER_API_KEY for access to many providers
source ~/.zshrc  # Load API keys
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# Or with explicit server IP/domain
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ARMORCLAW_SERVER_NAME=192.168.1.50
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### Bootstrap Mode (GitOps)

Generates production-ready config:

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash -s -- --bootstrap
```

Output: `/opt/armorclaw/docker-compose.yml`

Use for:
- Version-controlled infrastructure
- CI/CD pipelines
- Terraform/GitOps workflows

---

## Advanced / Manual Setup

> **Most users should use the options above.** This section is for advanced customization.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENROUTER_API_KEY` | OpenRouter API key (recommended) | (from .zshrc) |
| `OPEN_AI_KEY` | OpenAI API key | (from .zshrc) |
| `ZAI_API_KEY` | xAI API key | (from .zshrc) |
| `ARMORCLAW_SERVER_NAME` | Server hostname or IP | auto-detected |
| `ARMORCLAW_EXTERNAL_MATRIX` | Use external Matrix server | `false` |
| `ARMORCLAW_MATRIX_HOMESERVER_URL` | External Matrix URL | `http://127.0.0.1:6167` |
| `ARMORCLAW_MATRIX_ENABLED` | Enable Matrix integration | `true` |
| `ARMORCLAW_ADMIN_PASSWORD` | Admin password | auto-generated |

### Manual Docker Run

If you need full control over ports, volumes, or environment:

```bash
docker run -it --name armorclaw \
  --restart unless-stopped \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-data:/etc/armorclaw \
  -v armorclaw-keystore:/var/lib/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  -e OPENROUTER_API_KEY=sk-or-v1-xxx \
  -e ARMORCLAW_EXTERNAL_MATRIX=true \
  mikegemut/armorclaw:latest
```

### Why Docker Socket?

ArmorClaw uses the Docker socket to create isolated agent containers. This is safe because:

- Agent containers run with restricted privileges (no root, limited capabilities)
- Agents never receive Docker socket access
- Seccomp/AppArmor profiles prevent escape attempts
- Each agent runs in its own isolated container

---

## Deployment Profiles

| Profile | Runtime | Security | Best For |
|---------|---------|----------|----------|
| **Quick** | Docker | Standard hardening | Developers, testing |
| **Advanced** | Docker | Enhanced profiles | Production teams |
| **Enterprise** | Docker/containerd/Firecracker | Maximum isolation | HIPAA, SOC2, regulated |

**Enterprise runtime options:**
- **Docker hardened** - Default, maximum Docker security
- **containerd** (v5.0) - Kubernetes-native, reduced attack surface
- **Firecracker** (on request) - MicroVM isolation

---

## Security Features

| Feature | Description |
|---------|-------------|
| **E2EE Messaging** | All communication via Matrix protocol |
| **BlindFill™** | Secrets injected directly, agents never see values |
| **Memory-Only Secrets** | API keys read from environment variables, never written to disk |
| **SQLCipher Keystore** | Encrypted local storage for other secrets |
| **Container Isolation** | Each agent in hardened, isolated container |
| **HITL Approval** | Human-in-the-loop for sensitive operations |
| **Audit Logging** | All operations logged for compliance |

### Agent Container Security

Each agent runs with:
```
--cap-drop=ALL
--security-opt=no-new-privileges
--read-only
--pids-limit=100
--memory=512M
```

Plus seccomp profiles, AppArmor policies, and no network tools.

---

## Observability

### Check Status

```bash
# Container status
docker ps

# Bridge logs
docker logs -f armorclaw

# Test Bridge RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Test Matrix API
curl http://localhost:6167/_matrix/client/versions
```

---

## Testing

### GitHub Actions CI/CD

ArmorClaw uses GitHub Actions for automated testing and deployment:

**View all workflows:** [Actions](https://github.com/Gemutly/ArmorClaw/actions)

**Current Workflows:**
- **Test:** Comprehensive CI test suite (container hardening, secrets, exploits, E2E)
- **Build Release:** Multi-platform binary builds (Linux, macOS, Windows)
- **DockerHub:** Automatic Docker image builds and pushes

### Installer Test Suite (v4.4.0+)

Run the comprehensive test suite to verify installer hardening:

```bash
bash tests/integration/test-installer-hardening.sh
```

**Tests cover:**
- Lockfile functionality
- Docker daemon readiness
- Environment variable passthrough
- Docker Compose detection
- CONDUIT_IMAGE fallback
- Syntax validation
- Wait function existence

**Example output:**
```
=========================================
Running Installer Test Suite
=========================================
[TEST] Test 1: Lockfile functionality
[TEST] Docker wait loop
[TEST] Environment variable passthrough
[TEST] Docker Compose detection
[TEST] CONDUIT_IMAGE fallback
[TEST] Syntax validation
[TEST] wait_for_docker function
[TEST] Variable ordering
=========================================
All tests passed!
```

### Integration Tests

For CI/CD pipelines, run:
```bash
# Quick check
bash -n deploy/install.sh deploy/setup-matrix.sh deploy/quickstart-entrypoint.sh deploy/deploy-infra.sh

# Full test suite
bash tests/integration/test-installer-hardening.sh
```

### RPC Test Coverage

The bridge includes comprehensive RPC tests to catch issues before Docker builds:

**Unit Tests** (`bridge/pkg/rpc/*_test.go`):
- Method registration coverage (catches missing handlers)
- Matrix v3 login format validation
- Handler response structure tests

**Integration Tests** (`tests/test-rpc-methods.sh`):
- Socket communication validation
- Critical method availability
- Error handling verification

Run locally:
```bash
# Unit tests
cd bridge && go test -v ./pkg/rpc/...

# Integration tests (requires socat)
./tests/test-rpc-methods.sh
```

CI Pipeline:
```
precheck → rpc-unit-tests → rpc-integ-tests → docker-build → docker-smoke → docker-push
```

All RPC tests must pass before Docker images are built and pushed to Docker Hub.

---

## Build from Source

### Quickstart Image (All-in-One)

```bash
git clone https://github.com/Gemutly/ArmorClaw.git
cd ArmorClaw

# Build the quickstart image
docker build -t armorclaw/quickstart:latest -f Dockerfile.quickstart .
```

### Production Stack (Multi-Service)

```bash
git clone https://github.com/Gemutly/ArmorClaw.git
cd ArmorClaw

# Configure your server
export MATRIX_DOMAIN=your-server.example.com
export ARMORCLAW_API_KEY=sk-your-key

# Start all services (Matrix, Sygnal, Caddy, Bridge)
docker compose -f docker-compose-full.yml up -d --build
```

**Services:**
- `matrix` - Conduit homeserver (port 6167)
- `sygnal` - Push notifications (port 5000)
- `caddy` - Reverse proxy with auto-SSL (ports 80, 443)
- `bridge` - ArmorClaw orchestrator (port 8443)

### Bridge-Only Build

```bash
cd ArmorClaw/bridge

# Install Go 1.24+ and SQLCipher development libraries
# Build with SQLCipher support
export CGO_ENABLED=1
export CGO_CFLAGS="-I/usr/include/sqlcipher"
export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lsqlcipher"

go build -o armorclaw-bridge ./cmd/bridge
```

---

## Create Your First Agent (No-Code)

In the Matrix room:

```
!agent create name="Travel Booker" skills="web_browsing, form_filling"
```

The Bridge provisions the agent and invites you to its dedicated room.

---

## Architecture

```
┌───────────────────────────────────────────────────────────────────────┐
│                         THE VPS (Office)                              │
│                                                                       │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐           │
│  │ ArmorClaw   │◀────▶│  OpenClaw   │◀────▶│  Playwright │           │
│  │ Bridge      │      │  (Agent)    │      │  (Browser)  │           │
│  │ (Orchestr.) │      │             │      │             │           │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘           │
│         │                    │                     │                   │
│         │   BlindFill Engine │                     │                   │
│         │   (Memory-Only)    │                     │                   │
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

---

## Key Features

* **VPS-Based Secretary:** Agents run headless on your server, performing desktop-class tasks
* **Mobile-First Control:** Monitor status, review screenshots, approve PII via ArmorChat
* **No-Code Agent Studio:** Define agents via chat or Dashboard—no coding required
* **BlindFill™ Security:** Agents request sensitive data via references, never see raw values
* **Secure Browser Automation:** Remote control via Matrix protocol

---

## Documentation

* **Setup Guide:** [docs/guides/setup-guide.md](docs/guides/setup-guide.md)
* **Configuration:** [docs/guides/configuration.md](docs/guides/configuration.md)
* **RPC API:** [docs/reference/rpc-api.md](docs/reference/rpc-api.md)
* **Troubleshooting:** [docs/guides/troubleshooting.md](docs/guides/troubleshooting.md)
* **Full Index:** [docs/index.md](docs/index.md)
* **Tests:** [tests/integration/](tests/integration/) (v4.4.0+)
* **GitHub Actions:** [Actions](https://github.com/Gemutly/ArmorClaw/actions) (CI/CD pipelines)

---

## Links

* **GitHub:** https://github.com/Gemutly/ArmorClaw
* **Docker Hub:** https://hub.docker.com/r/mikegemut/armorclaw

## License

[MIT License](LICENSE)
