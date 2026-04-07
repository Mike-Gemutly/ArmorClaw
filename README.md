# ArmorClaw: The VPS Secretary Platform

[![Version](https://img.shields.io/badge/version-v4.7.0-blue)](https://github.com/Gemutly/ArmorClaw)
[![Status](https://img.shields.io/badge/status-production%20 ready-green)](https://github.com/Gemutly/ArmorClaw)

**Run AI agents on your VPS. Control from your phone.**

ArmorClaw runs AI agents 24/7 on your server. They browse websites, fill forms, and manage tasks—while you approve sensitive actions via your phone.

**🛡️ v4.7.0 Highlights:** **Sentinel Mode** (automatic VPS deployment with Let's Encrypt TLS), **Native/Sentinel deployment modes**, and **21 Browser Skills** for Chrome DevTools MCP integration.

---

> ⚠️ **Requirements Before You Start**
>
> Before installing ArmorClaw, make sure you have:
>
> - **AI Provider API Key** - At least one of:
>   - OpenRouter (recommended - supports 100+ models) - [Get key](https://openrouter.ai)
>   - OpenAI API key - [Get key](https://platform.openai.com)
>   - xAI (Grok) API key - [Get key](https://x.ai)
>   - Anthropic API key - [Get key](https://console.anthropic.com)
>   - See [Supported AI Providers](#supported-ai-providers-v450) for full list
>
> - **For Production (Public Access):**
>   - **Domain** - You must own a domain (e.g., `armorclaw.example.com`)
>   - **Cloudflare Account** - Free tier works, required for:
>     - Automatic HTTPS/TLS certificates
>     - DDoS protection
>     - NAT/firewall traversal (Tunnel mode)
>     - CDN caching
>
> - **System Requirements:**
>   - Linux VPS (Ubuntu 20.04+) or local machine
>   - Docker 24.0+
>   - 2GB RAM minimum, 4GB recommended
>   - 10GB disk minimum
>
> **Local/Development Mode** works without a domain - uses Unix sockets only.

---

## Quick Install (All-in-One)

The recommended way to start is the bootstrap installer, which now includes an optional **Matrix/Conduit** setup for instant QR provisioning:

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

**v4.7.0 Improvements:**
- **🌐 Sentinel Mode** - Automatic VPS deployment with Let's Encrypt TLS
- **🔄 Deployment Modes** - Native (local) vs Sentinel (public) auto-detection
- **🔐 Environment API Keys** - Keys from env vars (OPENROUTER_API_KEY, ZAI_API_KEY, OPEN_AI_KEY), never persisted to disk
- **🛡️ Secure Migrations** - Existing secrets preserved during mode upgrades

---

## Deployment Modes

ArmorClaw v4.7.0 introduces **automatic deployment mode detection** for zero-configuration VPS deployment.

| Mode | Use Case | Communication | TLS | Setup Time |
|------|---------|---------------|-----|------------|
| **Native** | Development, testing, local | Unix socket | None | ~2 min |
| **Sentinel** ⚡ | Production VPS, public access | TCP + HTTPS | Let's Encrypt | ~5 min |
| **Cloudflare Tunnel** | VPS behind NAT/firewall, no public IP | cloudflared tunnel | Cloudflare | ~3 min |
| **Cloudflare Proxy** | Existing Cloudflare proxy, CDN | HTTP(S) | Cloudflare | ~5 min |

### Native Mode (Default)
- **Communication:** Unix domain socket (`/run/armorclaw/bridge.sock`)
- **Access:** Local-only (no public exposure)
- **Best for:** Development, testing, machines without public IP

### Sentinel Mode (Production)
- **Communication:** TCP (`0.0.0.0:8080`) with Caddy reverse proxy
- **Access:** Public via `https://your-domain.com`
- **TLS:** Automatic Let's Encrypt certificates
- **Best for:** Production VPS, public access

**Activating Sentinel Mode:**
```bash
# During installation
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
# When prompted, enter your domain (e.g., armorclaw.example.com)
# Leave blank for Native mode
```

### Cloudflare Tunnel Mode
- **Communication:** Outbound via Cloudflare Tunnel (cloudflared)
- **Access:** Public via `https://your-domain.com` through Cloudflare network
- **TLS:** Automatic Cloudflare SSL certificates
- **Best for:** VPS behind NAT/firewall, no public IP, dynamic IP
- **Requirements:** Cloudflare account with API token

**Activating Cloudflare Tunnel Mode:**
```bash
# Set environment variables before installation
export CF_API_TOKEN=your-cloudflare-api-token
export CF_TUNNEL_DOMAIN=armorclaw.example.com

# Run installer
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### Cloudflare Proxy Mode
- **Communication:** HTTP(S) through Cloudflare proxy
- **Access:** Public via `https://your-domain.com` with Cloudflare CDN
- **TLS:** Cloudflare SSL (Flexible/Full/Full Strict)
- **Best for:** Existing Cloudflare setup, CDN caching, DDoS protection
- **Requirements:** Cloudflare account with domain configured

**Activating Cloudflare Proxy Mode:**
```bash
# Set environment variables before installation
export CF_API_TOKEN=your-cloudflare-api-token
export CF_MODE=proxy
export CF_ZONE_ID=your-zone-id

# Run installer
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### Environment Variables

| Variable | Native Mode | Sentinel Mode | Cloudflare |
|----------|-------------|---------------|------------|
| `ARMORCLAW_SERVER_MODE` | `native` | `sentinel` | `cloudflare` |
| `ARMORCLAW_RPC_TRANSPORT` | `unix` | `tcp` | `tcp` |
| `ARMORCLAW_LISTEN_ADDR` | - | `0.0.0.0:8080` | `0.0.0.0:8080` |
| `ARMORCLAW_PUBLIC_BASE_URL` | - | `https://your-domain.com` | `https://your-domain.com` |
| `ARMORCLAW_EMAIL` | - | `admin@your-domain.com` | `admin@your-domain.com` |
| `CF_API_TOKEN` | - | - | Cloudflare API token |
| `CF_TUNNEL_DOMAIN` | - | - | Tunnel domain |
| `CF_MODE` | - | - | `tunnel` or `proxy` |
| `CF_ZONE_ID` | - | - | Zone ID (proxy mode only) |

---

## Deployment Skills for AI CLI Tools

**What it is:** Built-in skills for AI assistants (Claude Code, OpenCode, Cursor) to deploy and manage ArmorClaw on your VPS.

**How to use:** Navigate to the project directory and tell your AI assistant what you want to do. Skills are auto-discovered from `.skills/`.

### Available Skills

| Skill | Purpose | Command |
|-------|---------|---------|
| **Deploy** | Deploy to VPS | `/deploy vps_ip=5.183.11.149 domain=armorclaw.example.com` |
| **Status** | Check health | `/status vps_ip=5.183.11.149` |
| **Cloudflare** | Setup HTTPS | `/cloudflare domain=example.com mode=tunnel` |
| **Provision** | Connect mobile | `/provision vps_ip=5.183.11.149` |

### Quick Example

```bash
cd /path/to/armorclaw-omo

# Tell your AI:
"Deploy ArmorClaw to VPS 5.183.11.149 with domain armorclaw.example.com"

# The AI will:
# 1. Read .skills/deploy.yaml
# 2. Ask for confirmation before SSH
# 3. Run the installer
# 4. Verify services are ready
# 5. Display connection info
```

**Platforms:** Linux, macOS, Windows (PowerShell, Git Bash, WSL)  
**Docs:** `.skills/README.md` | `doc/armorclaw.md`

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

# Deployment Mode (auto-detected from domain input)
ARMORCLAW_SERVER_MODE        # native | sentinel | cloudflare (auto-detected)

# Cloudflare (for cloudflare mode)
CF_API_TOKEN             # Cloudflare API token
CF_TUNNEL_DOMAIN         # Domain for Cloudflare Tunnel
CF_MODE                  # cloudflare mode: tunnel | proxy
CF_ZONE_ID               # Cloudflare Zone ID (proxy mode only)

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
| **Cloudflare Tunnel** | `CF_API_TOKEN=xxx CF_TUNNEL_DOMAIN=xxx bash install.sh` | VPS behind NAT, no public IP |
| **Cloudflare Proxy** | `CF_API_TOKEN=xxx CF_MODE=proxy CF_ZONE_ID=xxx bash install.sh` | Existing Cloudflare setup |
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

# With Cloudflare Tunnel
export OPENROUTER_API_KEY=sk-or-v1-xxx
export CF_API_TOKEN=your-cloudflare-token
export CF_TUNNEL_DOMAIN=armorclaw.example.com
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# With Cloudflare Proxy
export OPENROUTER_API_KEY=sk-or-v1-xxx
export CF_API_TOKEN=your-cloudflare-token
export CF_MODE=proxy
export CF_ZONE_ID=your-zone-id
export MATRIX_DOMAIN=armorclaw.example.com
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
| `CF_API_TOKEN` | Cloudflare API token for Tunnel/Proxy mode | - |
| `CF_TUNNEL_DOMAIN` | Domain for Cloudflare Tunnel | - |
| `CF_MODE` | Cloudflare mode: `tunnel` or `proxy` | - |
| `CF_ZONE_ID` | Cloudflare Zone ID (required for proxy mode) | - |

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

## Cloudflare Setup

### Cloudflare Tunnel Mode

Cloudflare Tunnel provides secure, outbound-only access to your ArmorClaw server without opening public ports or configuring firewall rules.

#### Prerequisites

1. **Cloudflare Account** - Sign up at [cloudflare.com](https://cloudflare.com) (free tier available)
2. **Domain** - Add your domain to Cloudflare
3. **API Token** - Create an API token with:
   - Zone: DNS - Edit
   - Zone: Cloudflare Tunnel - Edit
   - Account: Cloudflare Tunnel - Read

   Create token at: Cloudflare Dashboard → My Profile → API Tokens → Create Token

#### Manual Setup

```bash
# 1. Create Cloudflare API token
#    Visit: https://dash.cloudflare.com/profile/api-tokens
#    Create token with Zone DNS, Cloudflare Tunnel permissions

# 2. Set environment variables
export CF_API_TOKEN=your-cloudflare-api-token
export CF_TUNNEL_DOMAIN=armorclaw.yourdomain.com

# 3. Run installer
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# 4. Installer will:
#    - Install cloudflared
#    - Create tunnel
#    - Configure DNS records
#    - Start tunnel service
```

#### Troubleshooting Cloudflare Tunnel

**Tunnel not connecting:**
```bash
# Check cloudflared service status
sudo systemctl status cloudflared

# View tunnel logs
sudo journalctl -u cloudflared -f

# Verify tunnel exists in Cloudflare Dashboard
#    Visit: https://dash.cloudflare.com/ -> Access -> Tunnels
```

**DNS not propagating:**
```bash
# Check DNS records
dig +short armorclaw.yourdomain.com

# If no result, verify DNS record in Cloudflare Dashboard
#    Visit: https://dash.cloudflare.com/ -> DNS -> Records
```

**API token permissions error:**
```bash
# Verify token has correct permissions:
#    - Zone: DNS - Edit (for your zone)
#    - Zone: Cloudflare Tunnel - Edit (for your zone)
#    - Account: Cloudflare Tunnel - Read

# Test token with curl:
curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
  -H "Authorization: Bearer $CF_API_TOKEN" \
  -H "Content-Type: application/json"
```

**Tunnel status shows inactive:**
```bash
# Restart cloudflared service
sudo systemctl restart cloudflared

# Check if port 8080 is listening (Bridge)
sudo netstat -tlnp | grep 8080

# Verify Bridge is running
docker ps | grep armorclaw
```

### Cloudflare Proxy Mode

Cloudflare Proxy mode uses Cloudflare's reverse proxy and CDN to serve your ArmorClaw instance through existing Cloudflare infrastructure.

#### Prerequisites

1. **Cloudflare Account** with your domain already configured
2. **Zone ID** - Find at Cloudflare Dashboard → Select domain → Overview → Right sidebar
3. **API Token** - Create token with:
   - Zone: DNS - Edit
   - Zone: Cloudflare SSL/TLS - Edit

#### Manual Setup

```bash
# 1. Get your Zone ID
#    Visit: https://dash.cloudflare.com/ -> Select your domain
#    Zone ID is in the right sidebar

# 2. Set environment variables
export CF_API_TOKEN=your-cloudflare-api-token
export CF_MODE=proxy
export CF_ZONE_ID=your-zone-id

# 3. Run installer
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# 4. Installer will:
#    - Configure Caddy to listen on 0.0.0.0:8080
#    - Set up Cloudflare DNS records
#    - Configure SSL mode
```

#### Troubleshooting Cloudflare Proxy

**"522" Connection Timed Out:**
```bash
# Check if Caddy is listening on port 8080
sudo netstat -tlnp | grep 8080

# Check firewall rules
sudo ufw status
sudo ufw allow 8080/tcp

# Verify Caddy logs
docker logs caddy
```

**"1016" Origin DNS Error:**
```bash
# Verify DNS A record points to correct IP
dig +short armorclaw.yourdomain.com

# Check if ArmorClaw Bridge is running
docker ps | grep armorclaw

# Test local access
curl http://localhost:8080/health
```

**SSL Certificate Issues:**
```bash
# Check SSL/TLS mode in Cloudflare Dashboard
#    Visit: https://dash.cloudflare.com/ -> SSL/TLS -> Overview
#    Recommended: Full (strict) mode

# Verify Caddy is generating certificates
docker logs caddy | grep "certificate"
```

**Cloudflare caching static content:**
```bash
# Purge cache if experiencing stale content
#    Visit: https://dash.cloudflare.com/ -> Caching -> Configuration
#    Or use Cloudflare API to purge cache
```

### Creating a Cloudflare API Token

1. **Navigate to API Tokens:**
   - Go to Cloudflare Dashboard
   - Click on your profile (top right)
   - Select "My Profile"
   - Click "API Tokens" tab
   - Click "Create Token"

2. **Use Template:**
   - Select "Custom token" template
   - Set permissions:
     - **Account** → Cloudflare Tunnel → Read
     - **Zone** → DNS → Edit
     - **Zone** → Cloudflare Tunnel → Edit (for proxy mode: SSL/TLS → Edit)
   - Set Account Resources: Include → All accounts (or specific account)
   - Set Zone Resources: Include → Specific zone → your-domain.com
   - Click "Continue to summary"
   - Click "Create Token"

3. **Copy Token:**
   - Copy the token (you won't see it again!)
   - Store securely: `export CF_API_TOKEN=xxx`

### Example: Complete Cloudflare Tunnel Setup

```bash
#!/bin/bash
# Complete Cloudflare Tunnel setup for ArmorClaw

# 1. Set your configuration
export CF_API_TOKEN="your-actual-api-token-here"
export CF_TUNNEL_DOMAIN="armorclaw.example.com"
export OPENROUTER_API_KEY="your-openrouter-key"

# 2. Verify token
echo "Verifying Cloudflare API token..."
curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
  -H "Authorization: Bearer $CF_API_TOKEN" \
  -H "Content-Type: application/json" | jq .

# 3. Run ArmorClaw installer with Cloudflare Tunnel
echo "Installing ArmorClaw with Cloudflare Tunnel..."
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# 4. Verify installation
echo ""
echo "Checking services..."
docker ps
sudo systemctl status cloudflared --no-pager

# 5. Test tunnel
echo ""
echo "Testing tunnel connection..."
sleep 5
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" https://$CF_TUNNEL_DOMAIN/health

echo ""
echo "Setup complete! Access ArmorClaw at: https://$CF_TUNNEL_DOMAIN"
```

### Example: Complete Cloudflare Proxy Setup

```bash
#!/bin/bash
# Complete Cloudflare Proxy setup for ArmorClaw

# 1. Set your configuration
export CF_API_TOKEN="your-actual-api-token-here"
export CF_ZONE_ID="your-zone-id-here"
export CF_MODE="proxy"
export MATRIX_DOMAIN="armorclaw.example.com"
export OPENROUTER_API_KEY="your-openrouter-key"

# 2. Verify token
echo "Verifying Cloudflare API token..."
curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
  -H "Authorization: Bearer $CF_API_TOKEN" \
  -H "Content-Type: application/json" | jq .

# 3. Run ArmorClaw installer with Cloudflare Proxy
echo "Installing ArmorClaw with Cloudflare Proxy..."
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# 4. Verify DNS setup
echo ""
echo "Checking DNS propagation..."
sleep 10
dig +short $MATRIX_DOMAIN

# 5. Verify services
echo ""
echo "Checking services..."
docker ps

# 6. Test proxy
echo ""
echo "Testing proxy connection..."
sleep 5
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" https://$MATRIX_DOMAIN/health

echo ""
echo "Setup complete! Access ArmorClaw at: https://$MATRIX_DOMAIN"
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

The + ### Go Bridge (Control Plane)
    + The Go Bridge is the control plane that orchestrates all operations:
    + - Manages Matrix Conduit connections
    + - Handles user authentication via Matrix
    + - Routes requests to appropriate AI providers
    + - Implements audit logging for all operations
    + - Manages browser automation via Playwright
    + - Coordinates with Rust Sidecar for heavy I/O

    + ### Rust Office Sidecar (Data Plane)
    + The Rust sidecar handles heavy I/O operations:
    + - **Storage**: S3 upload/download/list/delete with streaming
    + - **Documents**: PDF text extraction, split, merge
    + - **Documents**: DOCX text extraction, editing
    + - **Security**: Ephemeral token auth (30 min TTL)
    + - **Reliability**: Circuit breaker, rate limiting
    + - **Performance**: Handles files up to 5GB with streaming
    + - **Communication**: gRPC over Unix Domain Socket
    + - **Memory**: Bounded to ~2MB for download streams
    + - **Integration**: PII detection and redaction

    + **Key Features:**
    + - Zero-copy streaming (no buffering)
    + - Single-pass SHA256 hashing
    + - 1MB chunk size for downloads
    + - Circuit breaker (5 failures → open, 30s recovery)
    + - Rate limiting (100 req/s)
    + - In-memory request queueing (graceful degradation)
    + - Prometheus metrics endpoint

    + **Security:**
    + - No persistent credential storage
    + - No credential caching beyond request lifecycle
    + - No direct cloud API calls without Go Bridge interception
    + - All operations logged in Go Bridge audit.db
    + - PII interception before sidecar calls
    + - Unix domain socket with 0600 permissions


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
