# ArmorClaw Documentation Index

> **Last Updated:** 2026-02-12
> **Version:** 1.2.1
> **Phase:** SDTW Planning Complete ✅ | Next: SDTW Implementation

---

## 📚 Quick Navigation

This index is the central hub for all ArmorClaw documentation. Start here for any AI agent working on this project.

### 🚀 Getting Started (Read This First)
1. [Project Overview](#project-overview) - What ArmorClaw does and why
2. [Current Status](#current-status) - What's implemented and what's next
3. [Quick Start](#quick-start) - Get up and running in 5 minutes
4. [Architecture](#architecture) - How the system works

### 📋 Reference Documentation
- [🔍 Error Catalog](docs/guides/error-catalog.md) - Every error with solutions (search by error text)
- [🔒 Security Verification Guide](docs/guides/security-verification-guide.md) 🆕 - Manual verification of all security hardening measures
- [🔒 Security Configuration](docs/guides/security-configuration.md) ⭐ - Zero-trust, budget guardrails, PII scrubbing
- [Element X Quick Start](docs/guides/element-x-quickstart.md) ⭐ - Connect to agents via Element X in 5 minutes
- [WebRTC Voice Guide](docs/guides/webrtc-voice-guide.md) 🆕 - Secure voice calls with Matrix authorization
- [WebRTC Voice Hardening](docs/guides/webrtc-voice-hardening.md) 🆕 - Security hardening for voice calls
- [WebSocket Client Guide](docs/guides/websocket-client-guide.md) 🆕 - Real-time Matrix event push via WebSocket
- [Communication Flow Analysis](docs/output/communication-flow-analysis.md) 🆕 - Complete communication architecture documentation
- [Setup Guide](docs/guides/setup-guide.md) - Interactive setup wizard and manual setup
- [Troubleshooting Guide](docs/guides/troubleshooting.md) - Systematic debugging procedures
- [RPC API Reference](docs/reference/rpc-api.md) - Complete JSON-RPC 2.0 API
- [Configuration Guide](docs/guides/configuration.md) - TOML config and env vars
- [Element X Configs](docs/guides/element-x-configs.md) - Sending configs via Element X
- [Developer Guide](docs/guides/development.md) - Development environment and contribution

### 🚀 Deployment Guides

#### Budget-Friendly Options
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md) 🆓 - Complete VPS setup from $4-8/month (Recommended)
- [Hostinger Docker Deployment](docs/guides/hostinger-docker-deployment.md) 🆓 - Docker Manager focus with GUI management
- [DockerHub + Hostinger Deployment](docs/guides/dockerhub-hostinger-deployment.md) 🆓 - Deploy via DockerHub with automatic updates
- [Vultr Deployment](docs/guides/vultr-deployment.md) - VPS with GPU options from $2.50/month

#### PaaS & Serverless
- [DigitalOcean App Platform](docs/guides/digitalocean-deployment.md) - Simple PaaS from $5/month
- [Railway Deployment](docs/guides/railway-deployment.md) - Quick deployment with excellent DX
- [Render Deployment](docs/guides/render-deployment.md) - Free tier for testing

#### Enterprise & Global
- [Fly.io Deployment](docs/guides/flyio-deployment.md) - Global edge distribution (35+ regions)
- [Google Cloud Run Deployment](docs/guides/gcp-cloudrun-deployment.md) - Serverless with free tier (2M requests/month)
- [AWS Fargate Deployment](docs/guides/aws-fargate-deployment.md) - Enterprise serverless with Spot pricing

#### Additional Options
- [Linode/Akamai Deployment](docs/guides/linode-deployment.md) - VPS with Akamai integration
- [Azure Container Instances](docs/guides/azure-deployment.md) - Per-second billing for burstable workloads
- [Local Development Guide](docs/guides/local-development.md) - Docker Desktop setup for local development

### Infrastructure Deployment
- [Infrastructure Deployment Guide](docs/guides/2026-02-05-infrastructure-deployment-guide.md) - General infrastructure setup

### 📐 Planning Documents
- [Phase 1 Tasks](docs/plans/2026-02-05-phase1-implementation-tasks.md) - Implementation roadmap
- [Business Model](docs/plans/2026-02-05-business-model-architecture.md) - Product tiers and revenue
- [ArmorClaw Evolution Design](docs/plans/2026-02-07-armorclaw-evolution-design.md) - Multi-agent collaboration platform 🆓
- [Security Enhancements](docs/plans/2026-02-07-security-enhancements.md) 🆕 - Zero-trust middleware, guardrails, hardening
- **NEW:** [SDTW Adapter Implementation Plan v2.0](docs/plans/SDTW_Adapter_Implementation_Plan_v2.0.md) 🆓 - Slack, Discord, Teams, WhatsApp integration
- **NEW:** [SDTW Message Queue Specification](docs/plans/SDTW_MessageQueue_Specification.md) 🆓 - SQLite-based persistent queue

### 📊 Status & Progress
- [Project Status](docs/status/2026-02-05-status.md) - Detailed status tracking
- [Milestone Progress](docs/PROGRESS/progress.md) - Completed milestones

### 🔬 Research & Analysis
- [Hosting Providers Comparison](docs/output/hosting-providers-comparison.md) 🆓 - Comprehensive evaluation of 11+ hosting options (Fly.io, AWS, GCP, DigitalOcean, Vultr, etc.)
- [Cloudflare Workers Analysis](docs/output/cloudflare-workers-analysis.md) - Platform evaluation for ArmorClaw fit ❌

---

## Project Overview

**ArmorClaw** is a local containment system for AI agents that prevents prompt injection from exposing API keys and secrets.

### Core Promise
> API keys are injected ephemerally via file descriptor passing. They exist only in memory inside the isolated container, are never written to disk, and are not exposed in Docker metadata or container inspection.

### What It Does
- **Isolates** AI agents in hardened Docker containers (non-root, no shell)
- **Protects** API keys with hardware-bound encryption (SQLCipher + XChaCha20-Poly1305)
- **Communicates** via Matrix protocol with E2EE support
- **Contains** blast radius if agent is compromised

### What It Doesn't Do (v1)
- Does NOT prevent in-memory misuse during active session
- Does NOT prevent side-channel attacks (memory scraping)
- Does NOT prevent host-level compromise

---

## Beyond ArmorClaw: ArmorClaw Evolution

**ArmorClaw Evolution** is a planned evolution of ArmorClaw that enables **secure multi-agent collaboration** while maintaining the same security boundaries.

### Key Enhancements

| Feature | ArmorClaw | ArmorClaw Evolution |
|---------|-----------|-----------|
| **Agent Communication** | Raw Matrix messages | Agent-to-Agent (A2A) Protocol |
| **Tool Discovery** | Custom JSON-RPC | Model Context Protocol (MCP) |
| **Memory** | Agent-local only | Shared encrypted epistemic memory |
| **Observability** | Minimal logging | Causal dependency graphs (CDG) |
| **Governance** | Schema validation | Real-time policy engine (OPA) |

### Use Cases

- **Collaborative Problem Solving:** Agents delegate tasks to specialists
- **Ensemble Decision Making:** Multiple agents analyze and vote
- **Map-Reduce Processing:** Distribute work across agent swarm
- **Knowledge Sharing:** Agents learn from each other's discoveries

### Planning Status

🆕 **Design Complete (v2.0):** [ArmorClaw Evolution Design Document](docs/plans/2026-02-07-armorclaw-evolution-design.md)

- ✅ Architecture specification (50+ sections)
- ✅ Technical modifications detailed (20+ components)
- ✅ Security analysis completed
- ✅ Gap analysis addressed (identity, fault tolerance, monitoring, etc.)
- ✅ Implementation phases defined (10 phases, 24-34 weeks)
- ⏳ Awaiting stakeholder approval

---

## Current Status

### Phase 1: Standard Bridge ✅ COMPLETE

| Component | Status | Binary Location |
|-----------|--------|-----------------|
| Encrypted Keystore | ✅ Complete | `bridge/pkg/keystore/` |
| Docker Client | ✅ Complete | `bridge/pkg/docker/` |
| Matrix Adapter | ✅ Complete | `bridge/internal/adapter/` |
| JSON-RPC Server | ✅ Complete | `bridge/pkg/rpc/` |
| Configuration System | ✅ Complete | `bridge/pkg/config/` |
| Shell Completion | ✅ Complete | `bridge/completions/` |
| Daemon Mode | ✅ Complete | `bridge/cmd/bridge/main.go` |
| Enhanced Help | ✅ Complete | `bridge/cmd/bridge/main.go` |

**Bridge Binary:** `bridge/build/armorclaw-bridge` (11 MB)

### UX Achievement: 8/10 ✅ TARGET REACHED

| Aspect | Rating | Status |
|--------|--------|--------|
| First-run experience | 9/10 | ✅ Excellent |
| Daily use | 9/10 | ✅ Excellent |
| Error recovery | 7/10 | ✅ Good |
| Documentation | 9/10 | ✅ Excellent |

**Recent Milestones:**
- ✅ Milestone 16: Error Documentation for LLMs
- ✅ Milestone 17: Comprehensive UX Assessment
- ✅ Milestone 18: P2 Polish Items (shell completion, daemon mode, enhanced help)
- ✅ Milestone 19: Element X UX Improvements

### Next Steps
- ✅ Initial integration testing complete (container hardening validated)
- ⏳ Full integration testing with Matrix Conduit
- ⏳ End-to-end testing with agent containers
- ⏳ Infrastructure deployment on Hostinger KVM2
- ⏳ ArmorClaw Evolution Phase 1 implementation (if approved)

---

## Quick Start

### Method 1: Element X Integration ⭐ NEW (Fastest)

Connect to your AI agents via Element X mobile app in 5 minutes:

```bash
# 1. Launch the infrastructure stack
cd armorclaw && ./deploy/launch-element-x.sh

# 2. Scan the QR code with Element X mobile app
# 3. Start chatting with your agent!
```

Perfect for mobile users - no local installation required beyond Docker.

**📖 Full Guide:** [Element X Quick Start](docs/guides/element-x-quickstart.md)

---

### Method 2: Interactive Setup Wizard ⭐ (Recommended)

The setup wizard automates installation and configuration in 10-15 minutes:

```bash
cd armorclaw
./deploy/setup-wizard.sh
```

The wizard handles:
- System requirements validation
- Docker installation/verification
- Container image building
- Bridge compilation and installation
- Keystore initialization (with security best practices)
- Configuration file generation
- First API key setup
- Systemd service setup
- Post-installation verification

### Method 3: Interactive CLI Wizard

For users who prefer to build first, then configure:

```bash
# Build the bridge
cd bridge && go build -o build/armorclaw-bridge ./cmd/bridge

# Run the interactive setup wizard
./build/armorclaw-bridge setup
```

This guides you through:
1. Docker availability check
2. Configuration location
3. AI provider selection (OpenAI, Anthropic, etc.)
4. API key entry (stored securely)
5. Optional Matrix configuration
6. Automatic configuration generation

### Method 4: Manual Setup (For advanced users)

### 1. Build the Bridge
```bash
cd bridge
go build -o build/armorclaw-bridge ./cmd/bridge
```

### 2. Initialize Configuration
```bash
# Option 1: Interactive setup (NEW - Recommended for first-time users)
./build/armorclaw-bridge setup

# Option 2: Quick config init
./build/armorclaw-bridge init
✓ Example configuration written to: ~/.armorclaw/config.toml
```

### 3. Add Your API Key (NEW - Much Easier!)
```bash
# Option 1: CLI command (recommended)
./build/armorclaw-bridge add-key --provider openai --token sk-proj-...

# Option 2: OpenClaw-style (environment variable)
export ARMORCLAW_API_KEY="sk-proj-..."
./build/armorclaw-bridge
```

### 4. Start an Agent (NEW - Much Easier!)
```bash
# List your keys first
./build/armorclaw-bridge list-keys

# Start with a specific key
./build/armorclaw-bridge start --key openai-default
```

### 5. Check Status
```bash
# List all containers
./build/armorclaw-bridge status

# Or use RPC (advanced)
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### Method 5: VPS Deployment 🆓

Deploy ArmorClaw to a remote VPS (Hostinger, DigitalOcean, etc.):

```bash
# From local machine
cd armorclaw
scp deploy/vps-deploy.sh armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# SSH into VPS
ssh user@your-vps-ip

# Run deployment script
chmod +x /tmp/vps-deploy.sh
sudo bash /tmp/vps-deploy.sh
```

**The automated script handles:**
- Pre-flight checks (disk, memory, ports)
- Docker installation (if needed)
- Tarball verification and extraction
- Interactive configuration
- Automated deployment

**📖 Full Guides:**
- [Setup Guide - VPS Deployment](docs/guides/setup-guide.md#method-5-vps-deployment-via-tarball-)
- [Hostinger VPS Deployment](docs/guides/hostinger-deployment.md)
- [Hostinger Docker Deployment](docs/guides/hostinger-docker-deployment.md)

---

## 🆕 New Features (v1.1.0)

### Shell Completion

Tab completion for bash and zsh makes daily usage more efficient:

```bash
# Generate completion
./build/armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
source ~/.bash_completion.d/armorclaw-bridge

# Use completion
./build/armorclaw-bridge <TAB>              # Show commands
./build/armorclaw-bridge add-key --<TAB>     # Show flags
./build/armorclaw-bridge start --key <TAB>   # Show available keys
```

### Daemon Mode

Run the bridge as a background service:

```bash
./build/armorclaw-bridge daemon start   # Start in background
./build/armorclaw-bridge daemon status  # Check status
./build/armorclaw-bridge daemon logs    # View logs
./build/armorclaw-bridge daemon stop    # Stop daemon
```

### Enhanced CLI Help

Better help with examples for all commands:

```bash
./build/armorclaw-bridge --help            # Main help with examples
./build/armorclaw-bridge add-key --help    # Command-specific help
```

### Element X Integration

Connect to your agents via Element X mobile app - no local installation required:

```bash
./deploy/launch-element-x.sh    # Launch Matrix + Caddy + Bridge
# Scan QR code with Element X mobile app
```

**📖 Full Guide:** [Element X Quick Start](docs/guides/element-x-quickstart.md)

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Host Machine                            │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Local Bridge (Go)                                     │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │  │
│  │  │   Keystore   │  │   Docker     │  │   Matrix  │  │  │
│  │  │  (SQLCipher) │  │   Client     │  │   Adapter  │  │  │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │  │
│  │                                                       │  │
│  │  JSON-RPC 2.0 Server                                 │  │
│  │  Socket: /run/armorclaw/bridge.sock                 │  │
│  └───────────────────────────────────────────────────────┘  │
│                          ↕ JSON-RPC + FD passing             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  ArmorClaw Container (Hardened Docker)               │  │
│  │  ┌─────────────────────────────────────────────────┐   │  │
│  │  │  OpenClaw Agent + Matrix Skill                  │   │  │
│  │  │  - User: UID 10001 (claw)                       │   │  │
│  │  │  - No shell, no network tools                   │   │  │
│  │  │  - Secrets in memory only (FD 3)                │   │  │
│  │  └─────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                           ↕ Matrix Protocol
┌─────────────────────────────────────────────────────────────┐
│              Matrix Conduit (Docker)                         │
│  - Homeserver: https://matrix.armorclaw.com               │
│  - Port: 6167 (API), 8448 (Client)                          │
│  - E2EE: Olm/Megolm                                         │
└─────────────────────────────────────────────────────────────┘
```

### Key Technologies
- **Container:** Docker with seccomp profiles
- **Encryption:** SQLCipher + XChaCha20-Poly1305
- **Communication:** JSON-RPC 2.0 over Unix socket
- **Protocol:** Matrix (Conduit homeserver)
- **Language:** Go 1.24+ (Bridge), Python (Agent)

---

## File Structure (LLM-Optimized)

```
ArmorClaw/
├── docs/                          # ALL DOCUMENTATION (start here)
│   ├── index.md                   # THIS FILE - Documentation hub
│   ├── plans/                     # Architecture & design documents
│   │   ├── 2026-02-05-armorclaw-v1-design.md
│   │   ├── 2026-02-05-phase1-implementation-tasks.md
│   │   ├── 2026-02-05-license-server-api.md
│   │   └── 2026-02-05-business-model-architecture.md
│   ├── guides/                    # How-to guides
│   │   └── 2026-02-05-deployment-quickref.md
│   ├── reference/                 # Technical specifications (TODO)
│   │   └── rpc-api.md             # Complete RPC API reference
│   ├── status/                    # Project status tracking
│   │   └── 2026-02-05-status.md
│   ├── PROGRESS/                  # Milestone tracking (TODO)
│   │   └── progress.md
│   └── output/                    # Milestone reviews (TODO)
│       └── review.md
├── bridge/                        # Go Local Bridge (Phase 1 complete)
│   ├── cmd/bridge/main.go         # Entry point
│   ├── pkg/
│   │   ├── config/                # Configuration system
│   │   ├── docker/                # Docker client (scoped)
│   │   ├── keystore/              # Encrypted credential storage
│   │   ├── logger/                # Structured logging (slog + security events)
│   │   └── rpc/                   # JSON-RPC 2.0 server
│   ├── internal/adapter/          # Matrix adapter
│   ├── build/armorclaw-bridge    # Compiled binary (11 MB)
│   └── go.mod
├── container/                     # Container runtime files
│   └── opt/openclaw/
├── tests/                         # Test suites
├── docker-compose.yml             # Infrastructure stack
├── Dockerfile                      # Hardened container image
├── CLAUDE.md                       # AI agent guidance
└── README.md                       # User-facing documentation
```

---

## Documentation for Specific Tasks

### For AI Agents Working On:
- **New Features:** Read `docs/plans/2026-02-05-phase1-implementation-tasks.md`
- **Docker Integration:** Read `bridge/pkg/docker/client.go` source
- **Matrix Integration:** Read `bridge/internal/adapter/matrix.go` source
- **Security:** Read security principles in `CLAUDE.md`
- **Testing:** Read `tests/` directory

### For Understanding:
- **Architecture:** `docs/plans/2026-02-05-armorclaw-v1-design.md`
- **Business Model:** `docs/plans/2026-02-05-business-model-architecture.md`
- **Communication:** `docs/plans/2026-02-05-communication-server-options.md`

---

## Key Decisions (Locked for v1)

| Decision | Rationale | Date |
|----------|-----------|------|
| Hybrid Bridge Strategy | Fast time-to-market + premium upsell | 2026-02-05 |
| Matrix Conduit for comm | E2EE, lightweight, rich ecosystem | 2026-02-05 |
| SQLCipher for keystore | Encrypted at rest, SQLite simplicity | 2026-02-05 |
| Unix socket for bridge | No network exposure, minimal overhead | 2026-02-05 |
| Scoped Docker client | Permission checks + seccomp hardening | 2026-02-05 |

---

## Memory Budget

**Target:** ≤ 2 GB on Hostinger KVM2 (4 GB available)

| Component | Phase 1 | Phase 4 |
|-----------|---------|---------|
| Ubuntu (minimal) | 400 MB | 400 MB |
| Nginx | 40 MB | 40 MB |
| Matrix Conduit | 200 MB | 200 MB |
| Coturn (TURN) | 50 MB | 50 MB |
| Local Bridge | 50 MB | 250 MB |
| OpenClaw Agent | 800 MB | 800 MB |
| **TOTAL** | **~1.54 GB** | **~1.74 GB** |
| **HEADROOM** | **~460 MB** | **~260 MB** |

✅ Both phases under 2 GB target

---

## Dependencies

### Runtime (Required)
- Docker Desktop or Docker Daemon
- Linux or WSL2 (v1 target)

### Build (For Development)
- Go 1.24+ (for Local Bridge)
- Python 3.x (for OpenClaw compatibility)
- CGo-enabled compiler (for SQLCipher)

### Go Dependencies
- `github.com/docker/docker` - Docker API client
- `github.com/mutecomm/go-sqlcipher/v4` - Encrypted SQLite
- `golang.org/x/crypto` - XChaCha20-Poly1305 encryption

---

## Security Posture

### What ArmorClaw Prevents
- ✅ Secrets persisting to disk
- ✅ Secrets in Docker metadata
- ✅ Direct filesystem escape
- ✅ Long-term secret retention

### What ArmorClaw Does NOT Prevent (v1)
- ⚠️ In-memory misuse during active session
- ⚠️ Side-channel attacks (memory scraping)
- ⚠️ Host-level compromise

**Our containment is blast radius reduction, not perfect secrecy.**

---

## Quick Reference Commands

### Bridge Operations
```bash
# Build bridge
cd bridge && go build -o build/armorclaw-bridge ./cmd/bridge

# Initialize config
./build/armorclaw-bridge init

# Validate config
./build/armorclaw-bridge validate

# Start bridge
sudo ./build/armorclaw-bridge

# Start with Matrix
sudo ./build/armorclaw-bridge -matrix-enabled \
  -matrix-homeserver https://matrix.armorclaw.com \
  -matrix-username bridge-bot \
  -matrix-password secret
```

### Shell Completion
```bash
# Bash completion
./build/armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
source ~/.bash_completion.d/armorclaw-bridge

# Zsh completion
./build/armorclaw-bridge completion zsh > ~/.zsh/completions/_armorclaw-bridge
```

### Daemon Mode
```bash
# Start as background daemon
./build/armorclaw-bridge daemon start

# Check daemon status
./build/armorclaw-bridge daemon status

# View daemon logs
./build/armorclaw-bridge daemon logs

# Stop daemon
./build/armorclaw-bridge daemon stop
```

### Container Operations
```bash
# Build container
docker build -t armorclaw/agent:v1 .

# Run hardening tests
make test-hardening

# Run all tests
make test-all
```

---

## License

MIT License - See [LICENSE](LICENSE) file

---

## Support & Contribution

- **GitHub:** https://github.com/armorclaw/armorclaw
- **Documentation Issues:** Create issue with `docs:` label
- **Bug Reports:** Create issue with `bug:` label

---

**Index Last Updated:** 2026-02-07
**Phase:** Phase 1 Complete - Production Ready
