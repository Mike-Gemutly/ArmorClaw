# ArmorClaw System Documentation

> **Purpose**: LLM-readable comprehensive documentation describing ArmorClaw's purpose, architecture, features, and functions from feature-level to variable-level detail.
> 
> **Version**: 4.7.0
> 
> **Last Updated**: 2026-04-01

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Architecture](#system-architecture)
3. [Go Bridge Component](#go-bridge-component)
4. [SQLCipher Keystore](#sqlcipher-keystore)
5. [Matrix Conduit Control Plane](#matrix-conduit-control-plane)
6. [Agent Studio](#agent-studio)
7. [Browser Service](#browser-service)
8. [ArmorChat Android Client](#armorchat-android-client)
9. [OpenClaw Agent Runtime](#openclaw-agent-runtime)
10. [Security Architecture](#security-architecture)
11. [Data Flow Patterns](#data-flow-patterns)
12. [Configuration Reference](#configuration-reference)

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

---

## Deployment Modes

### Overview

ArmorClaw v4.7.0 introduces **automatic deployment mode detection** for zero-configuration VPS deployment. The installer automatically detects whether you're deploying locally or on a public VPS and configures the system appropriately.

### Mode Comparison

| Feature | Native Mode | Sentinel Mode |
|---------|-------------|---------------|
| **Use Case** | Development, testing, local use | Production VPS, public access |
| **Communication** | Unix domain socket | TCP + TLS (HTTPS) |
| **Access** | Local-only (`/run/armorclaw/bridge.sock`) | Public (`https://your-domain.com`) |
| **TLS Certificates** | None (local only) | Let's Encrypt (automatic) |
| **Reverse Proxy** | None | Caddy (automatic HTTPS) |
| **Domain Required** | No | Yes |
| **Setup Time** | ~2 minutes | ~5 minutes |

### Mode Selection

The installer automatically determines the mode based on domain input:

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash

# Interactive prompt:
# Enter domain name (leave blank for local-only): 
#   → Leave blank = Native mode (local-only, Unix socket)
#   → Enter domain = Sentinel mode (public, TCP+TLS)
```

---

## Interactive Setup Walkthrough

The installer guides you through setup with interactive prompts:

### Step 1: Download and Run Installer

```bash
curl -fsSL https://raw.githubusercontent.com/Gemutly/ArmorClaw/main/deploy/install.sh | bash
```

### Step 2: Choose Deployment Mode

The installer will prompt:

```
Enter domain name (leave blank for local-only):
```

- **Leave blank** → Native mode (development/local testing)
- **Enter domain** → Sentinel mode (production/public access)

### Step 3: AI Provider Configuration

The installer will prompt for an AI provider:

```
Select AI Provider:
  1) OpenAI
  2) Anthropic
  3) Zhipu AI (Z AI)
  4) DeepSeek
  5) OpenRouter (recommended)
  
Enter choice [1-5]:
```

After selecting, provider, enter your API key when prompted.

> **Note**: API keys are read from environment variables at runtime and are never written to disk. Set them in your `.zshrc` or shell profile:
```bash
# Add to ~/.zshrc (persists across sessions)
export OPENROUTER_API_KEY=sk-or-v1-xxx
export ZAI_API_KEY=xxx
export OPEN_AI_KEY=xxx
```

### Step 4: Admin Credentials

The installer will ask for admin password

```
Admin password (press Enter to auto-generate):
```

- Press **Enter** → Random secure password generated
- Enter password → Use your custom password

### Step 5: Complete Installation

The installer will:
1. Install Docker dependencies
2. Pull Docker images
3. Set up Matrix Conduit homeserver
4. Configure ArmorClaw Bridge
5. Generate QR code for mobile app

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

## Why Cloudflare for Production?

Cloudflare provides critical infrastructure for production deployments:

### Benefits

| Benefit | Description |
|---------|-------------|
| **HTTPS/SSL** | Automatic TLS certificates (no manual cert management) |
| **DDoS Protection** | Built-in protection against distributed denial-of-service attacks |
| **NAT Traversal** | Access your VPS even without a public IP (Tunnel mode) |
| **CDN** | Global content delivery network for faster response times |
| **Firewall** | Web Application Firewall (WAF) for additional security |
| **DNS Management** | Automatic DNS configuration and propagation |

### When Cloudflare is Required

**Cloudflare is REQUIRED for:**
- Production deployments with public HTTPS access
- VPS behind NAT or firewall
- VPS with dynamic IP addresses
- When you need DDoS protection
- When you want CDN acceleration

**Cloudflare is OPTIONAL for:**
- Local development/testing
- VPS with static public IP and direct HTTPS (Let's Encrypt via Caddy)
- Internal network deployments

### Cloudflare Modes

| Mode | Use Case | Public IP Required |
|------|---------|-------------------|
| **Tunnel** | VPS behind NAT/firewall | No (outbound only) |
| **Proxy** | Existing Cloudflare setup | Yes |

### Setup

Cloudflare setup is documented in the main README.md** under the "Cloudflare Setup" section. Choose the mode based on your infrastructure:

---

### Native Mode

**Best for:** Development, testing, machines without public domains

**Configuration:**
- Communication via Unix socket: `/run/armorclaw/bridge.sock`
- No public internet exposure
- No TLS certificates needed
- Matrix accessible at `http://127.0.0.1:6167`

**Environment Variables:**
```bash
ARMORCLAW_SERVER_MODE=native
ARMORCLAW_RPC_TRANSPORT=unix
ARMORCLAW_SOCKET_PATH=/run/armorclaw/bridge.sock
```

**Docker Compose:**
```bash
docker-compose up -d  # No profile needed
```

### Sentinel Mode

**Best for:** Production VPS deployment, remote access

**Configuration:**
- Communication via TCP: `0.0.0.0:8080` (proxied through Caddy)
- Automatic HTTPS with Let's Encrypt
- Public access via domain: `https://your-domain.com`
- Caddy reverse proxy handles TLS termination

**Environment Variables:**
```bash
ARMORCLAW_SERVER_MODE=sentinel
ARMORCLAW_RPC_TRANSPORT=tcp
ARMORCLAW_LISTEN_ADDR=0.0.0.0:8080
ARMORCLAW_PUBLIC_BASE_URL=https://your-domain.com
ARMORCLAW_EMAIL=admin@your-domain.com  # For Let's Encrypt
```

**Docker Compose:**
```bash
docker-compose --profile sentinel up -d  # Activates Caddy proxy
```

### Migration Between Modes

**Native → Sentinel:**
1. Re-run installer with domain
2. Existing secrets are preserved
3. Configuration upgraded to TCP+TLS
4. Caddy provisions Let's Encrypt certificates

**Sentinel → Native:**
1. Re-run installer without domain
2. Secrets preserved
3. Configuration downgraded to Unix socket
4. Caddy profile disabled

**Important:** All existing data (keystore.db, Matrix rooms) is preserved during mode changes. The installer reads existing secrets from `.env` before migration.

### Architecture Differences

#### Native Mode Architecture
```
┌─────────────────────────────────────┐
│           LOCAL MACHINE             │
│                                     │
│  ┌──────────┐     ┌──────────────┐ │
│  │  Bridge  │◄───▶│ Unix Socket  │ │
│  │          │     │ /run/.../    │ │
│  └──────────┘     └──────────────┘ │
│       ▲                             │
│       │ Local Only                  │
│  ┌────┴─────┐                       │
│  │ Matrix   │                       │
│  │ (6167)   │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

#### Sentinel Mode Architecture
```
┌─────────────────────────────────────────────────┐
│                   PUBLIC VPS                     │
│                                                 │
│  ┌──────────┐    ┌───────────┐    ┌──────────┐ │
│  │ Internet │───▶│   Caddy   │───▶│  Bridge  │ │
│  │   :443   │    │  (TLS)    │    │  :8080   │ │
│  └──────────┘    └───────────┘    └──────────┘ │
│                        │               ▲        │
│                        ▼               │        │
│                  Let's Encrypt    TCP Socket   │
│                  Certificates                   │
│                        │                        │
│                  ┌─────┴─────┐                  │
│                  │  Matrix   │                  │
│                  │  (6167)   │                  │
│                  └───────────┘                  │
└─────────────────────────────────────────────────┘
│                                            ▼        ▼        ▼
│                                    Public Access                    HTTPS      │
```
---

## Deployment Skills for AI CLI Tools

### Overview

ArmorClaw v4.7.0 includes **deployment skills** for AI CLI tools (Claude Code, OpenCode, Cursor, etc.). These skills help users deploy and manage ArmorClaw using their AI assistants with cross-platform support (Linux, macOS, Windows).

All skills use **shell variable interpolation** (`${variable}`) for consistency across platforms and AI tool implementations.

### Skills Directory

All deployment skills are located in `.skills/` at the project root:

```
.skills/
├── deploy/
│   ├── deploy.yaml       # Deployment skill configuration
│   └── SKILL.md          # AI-friendly deployment instructions
├── status/
│   ├── status.yaml       # Status checking skill configuration
│   └── SKILL.md          # Status check documentation
├── cloudflare/
│   ├── cloudflare.yaml   # Cloudflare setup skill configuration
│   └── SKILL.md          # Cloudflare configuration guide
├── provision/
│   ├── provision.yaml    # Mobile provisioning skill configuration
│   └── SKILL.md          # Device provisioning guide
├── PLATFORM.md           # Cross-platform detection patterns
├── TEMPLATE.yaml         # Skill schema template
└── README.md             # Skills index
```

### Skill Invocation

Tell your AI CLI tool:

```
"Deploy ArmorClaw to VPS 192.168.1.100 with domain myvps.com"
```

The AI will use the `.skills/deploy.yaml` to execute the deployment.

### Available Skills

| Skill | Purpose | Command | Key Parameters |
|-------|---------|---------|----------------|
| **Deploy** | Deploy ArmorClaw to VPS | `/deploy vps_ip=... ssh_key=... domain=...` | `vps_ip`, `ssh_key`, `domain`, `mode` |
| **Status** | Check deployment health | `/status vps_ip=...` | `vps_ip`, `ssh_key`, `domain`, `verbose` |
| **Cloudflare** | Configure HTTPS | `/cloudflare domain=... mode=tunnel\|proxy` | `domain`, `mode`, `cf_api_token` |
| **Provision** | Connect mobile device | `/provision vps_ip=...` | `vps_ip`, `expiry`, `show_url` |

### Cross-Platform Support

All skills support:
- **Linux** - Native OpenSSH, curl
- **macOS** - Native OpenSSH, curl
- **Windows (PowerShell)** - OpenSSH, Invoke-WebRequest
- **Windows (Git Bash)** - OpenSSH, curl
- **Windows (WSL)** - OpenSSH, curl with `/mnt/c/` paths

### Automation Levels

Each skill step has an automation flag:

| Level | Behavior | Examples |
|-------|----------|----------|
| `auto` | Execute immediately without asking | Health checks, status, OS detection |
| `confirm` | Ask user before executing | SSH connection, running installer |
| `guide` | Provide instructions for user | Account creation, DNS setup, token creation |

### Deployment Modes

Skills support all deployment modes:
- **Native** - Local development (Unix socket)
- **Sentinel** - Production VPS (TCP + Let's Encrypt)
- **Cloudflare Tunnel** - VPS behind NAT/firewall
- **Cloudflare Proxy** - Existing Cloudflare setup

### Using with AI CLI Tools

1. **Navigate to project directory:**
   ```bash
   cd /path/to/armorclaw-omo
   ```

2. **Invoke skill:**
   ```
   "Use the deploy skill to deploy ArmorClaw to my VPS at 5.183.11.149"
   ```

3. **AI reads `.skills/deploy.yaml`** and executes steps with appropriate automation level.

### Example Workflow

```bash
# User tells AI:
"Deploy ArmorClaw to VPS 5.183.11.149 with SSH key ~/.ssh/id_rsa and domain armorclaw.example.com"

# AI uses .skills/deploy.yaml and:
1. Detects your OS (Linux/macOS/Windows)
2. Connects to VPS via SSH (asks for confirmation)
3. Runs install.sh on VPS (asks for confirmation)
4. Waits for services to be ready (auto)
5. Verifies deployment (auto)
6. Displays connection info (auto)
```

### Skill Schema

All skills follow `.skills/TEMPLATE.yaml`:

```yaml
name: skill_name
version: "1.0.0"
description: Brief description of what this skill does

parameters:
  - name: "param_name"
    type: "string"
    required: true
    description: Parameter description
    default: ""

steps:
  - name: "step_name"
    automation: "auto"  # auto | confirm | guide
    description: "What this step does"
    command: |
      # Shell commands use ${variable} syntax
      echo "Processing ${param_name}"

platforms:
  - linux
  - macos
  - windows-gitbash
  - windows-powershell
  - wsl

examples:
  - description: "Example usage"
    command: "/skill-name param1=value1"
```

### Variable Interpolation

All skills use **shell variable syntax** for parameters:

```bash
# Correct - shell variable syntax
command: |
  SSH_KEY="${ssh_key:-~/.ssh/id_ed25519}"
  echo "Connecting to ${vps_ip}..."

# Incorrect - template syntax (not supported)
command: |
  SSH_KEY="{{ssh_key}}"  # DO NOT USE
```

This ensures consistency across all AI CLI tool implementations.

### Related Documentation

- **Skills README**: `.skills/README.md` - Skills index
- **Platform Detection**: `.skills/PLATFORM.md` - Cross-platform patterns
- **Deployment Guide**: This document, [Deployment Modes](#deployment-modes)
- **Individual Skill Docs**: `.skills/*/SKILL.md` - Detailed per-skill documentation

---

## VPS Testing and Verification

### Overview
Comprehensive SSH-based VPS testing infrastructure has been implemented to verify ArmorClaw deployment health, security, and functionality. The testing suite provides automated verification of all critical system components.

### Test Categories
The testing suite includes **10 comprehensive test categories** with **136+ individual tests**:

| Category | Description | Test Count | Status |
|---------|-------------|------------|--------|
| **1. SSH Connectivity** | Key validation, connection, retry logic, network diagnostics | 12 | ✅ Implemented |
| **2. Command Execution** | Remote commands, output capture, timeout handling, exit codes | 8 | ✅ Implemented |
| **3. Container Health** | Container status, logs, restart, isolation, resource usage | 6 | ✅ Implemented |
| **4. API Endpoints** | Bridge RPC, Matrix client, health checks, authentication | 8 | ✅ Implemented |
| **5. Integration** | Cross-component communication, encryption, auth flows | 8 | ✅ Implemented |
| **6. Security** | Firewall, SSH hardening, container isolation, secrets, network policies | 35 | ✅ Implemented |
| **7. Deployment Modes** | Native, Sentinel, Cloudflare Tunnel/Proxy detection | 6 | ✅ Implemented |
| **8. SSL/TLS** | Certificate presence, expiry, chain, HTTPS connectivity | 6 | ✅ Implemented |
| **9. Performance** | SSH speed, API times, container resources, disk I/O | 6 | ✅ Implemented |
| **10. Output Formatting** | JSON console output, error handling, CLI interface | 1 | ✅ Implemented |

### Usage

#### Quick Start
```bash
# Navigate to project directory
cd /home/mink/src/armorclaw-omo

# Run all tests
bash tests/ssh/run_all_tests.sh --all

# Run specific category
bash tests/ssh/run_all_tests.sh --connectivity
bash tests/ssh/run_all_tests.sh --security
```

#### CLI Options
```bash
# Run all tests with verbose output
bash tests/ssh/run_all_tests.sh --all --verbose

# Run all tests with JSON output
bash tests/ssh/run_all_tests.sh --all --output json

# Show help
bash tests/ssh/run_all_tests.sh --help
```

#### Available Options
| Option | Description |
|--------|-------------|
| `-a, --all` | Run all test categories |
| `-c, --connectivity` | SSH connectivity tests only |
| `-x, --command` | Command execution tests only |
| `-h, --health` | Container health tests only |
| `-p, --api` | API endpoint tests only |
| `-i, --integration` | Integration tests only |
| `-s, --security` | Security tests only |
| `-d, --deployment` | Deployment mode tests only |
| `-l, --ssl` | SSL/TLS tests only |
| `-f, --performance` | Performance tests only |
| `-o, --output FORMAT` | Set output format (console\|json) |
| `-v, --verbose` | Enable verbose output |
| `--help` | Show help message |

### Test Results

All test results are saved to the evidence directory:
- **Evidence Location**: `.sisyphus/evidence/`
- **Summary File**: `.sisyphus/evidence/IMPLEMENTATION_SUMMARY.md`

Test result files include:
- **JSON Output**: `task-N-results.json` - Machine-parseable results
- **Console Output**: `task-N-success.txt` - Human-readable summary
- **Evidence Log**: `task-N-evidence.txt` - Detailed findings

### Environment Variables

The test suite reads configuration from `.env` file:

```bash
# Required variables
VPS_IP=5.183.11.149              # VPS IP address
VPS_USER=root                       # SSH username
SSH_KEY_PATH=~/.ssh/openclaw_win   # Path to SSH private key
BRIDGE_PORT=8080                    # Bridge RPC port (default)
MATRIX_PORT=6167                    # Matrix port (default)

# Optional variables
ARMORCLAW_SERVER_MODE=native         # Deployment mode (native\|sentinel)
ARMORCLAW_PUBLIC_BASE_URL=           # Public base URL (for Sentinel mode)
MATRIX_ADMIN_USER=admin             # Matrix admin user
MATRIX_ADMIN_PASSWORD=             # Matrix admin password
```

### Test Categories Detail

#### 1. SSH Connectivity Tests (12 tests)
- SSH key validation (exists, readable, permissions, format)
- SSH connection establishment (10s timeout)
- SSH version verification
- Connection retry logic (3 retries, exponential backoff)
- Connection timeout handling (5s timeout)
- Key-based authentication verification (BatchMode, no password)
- Remote command execution verification
- Connection stability testing (3 quick connections)
- Network diagnostics (ping, traceroute, DNS lookup)

#### 2. Command Execution Tests (8 tests)
- Simple command execution (echo, uptime)
- Commands with arguments (ls -la /tmp)
- Commands with pipes (ps aux | grep docker)
- Output capture (stdout and stderr)
- Exit code handling and validation
- Timeout handling (using timeout command)
- Error message capture

#### 3. Container Health Tests (6 tests)
- Container status (running, exited, restarting)
- Container logs retrieval (docker logs command)
- Container restart handling (simulated restart)
- Container isolation (seccomp, AppArmor, no-new-privileges)
- Resource usage (docker stats for CPU, memory, network)
- Container networking (port bindings, network mode)

#### 4. API Endpoint Tests (8 tests)
- Bridge RPC health check
- Matrix client versions endpoint
- Matrix federation endpoint
- JSON-RPC 2.0 compatibility
- Health endpoint tests (/health, /status)
- Authentication endpoint tests
- Timeout handling (curl --max-time)
- Response format validation (JSON structure)
- Error response handling (401, 404, 500)

#### 5. Integration Tests (8 tests)
- Bridge ↔ Matrix communication
- Bridge → Agent communication
- Agent → Browser communication
- Matrix → Agent messaging
- End-to-end encryption verification
- Authentication flows
- Approval workflows
- Cross-component message routing

#### 6. Security Tests (35 tests)
- Firewall rules (UFW installation, status, default policy)
- SSH hardening (PasswordAuthentication no, PermitRootLogin no, PubkeyAuthentication yes)
- Container isolation (Docker seccomp, AppArmor, no-new-privileges)
- Secret access controls (memory-only, no persistence, keystore permissions)
- Network policies (SYN cookies, IP forwarding, reverse path filtering)
- User permissions (no root on containers, armorclaw user, login shell, sudo)
- SQLCipher keystore (database format, encryption, libraries, integrity)

#### 7. Deployment Mode Tests (6 tests)
- Native mode detection (Unix socket)
- Sentinel mode detection (TCP + Caddy)
- Cloudflare Tunnel detection (cloudflared)
- Cloudflare Proxy detection (CDN)
- Mode switching (ARMORCLAW_SERVER_MODE variable)
- Configuration validation (docker-compose files, env vars)
- Port binding checks (8080, 6167, 8448)

#### 8. SSL/TLS Tests (6 tests)
- Certificate presence (Let's Encrypt on VPS)
- Certificate expiry (30-day warning threshold)
- Certificate chain verification (root to leaf)
- HTTPS connectivity (curl with SSL)
- Certificate renewal (certbot check)
- Let's Encrypt integration (ACME challenge)
- Certificate revocation (CRL/OCSP)

#### 9. Performance Tests (6 tests)
- SSH connection speed (3 connections, average time)
- API response times (curl -w timing)
- Container resource usage (docker stats)
- Concurrent operations (parallel SSH, parallel API)
- Memory usage (free command, container limits)
- CPU usage (top command, container limits)
- Disk I/O (dd read/write speeds)
- Execution time tracking (<60s target)

### Best Practices

#### Running Tests
- **Start from clean state**: Ensure VPS is in expected configuration
- **Review results**: Check evidence files for failures and warnings
- **Investigate failures**: Use verbose mode for detailed error information
- **Fix issues**: Address security vulnerabilities, missing services, configuration problems
- **Re-run tests**: Verify fixes resolve issues

#### Test Failures
Some test failures are **expected** based on VPS state:
- **Missing containers**: Bridge, Browser, Agent may not be running (expected on current VPS)
- **API failures**: Bridge RPC may be inaccessible if container not running
- **Security warnings**: SSH hardening or container isolation may not be configured

These failures indicate **environment state**, not **implementation bugs**.

### Files and Location

All test scripts are located in `tests/ssh/` directory:
- **10 test scripts**: 3,310 total lines
- **CLI tool**: `tests/ssh/run_all_tests.sh` (304 lines)
- **Evidence directory**: `.sisyphus/evidence/`

### Implementation Status

✅ **Complete**: All 15 implementation tasks complete
✅ **Verified**: CLI tool functional and all tests executable
✅ **Documented**: Implementation summary created
✅ **Tested**: Basic connectivity tests verified

### Next Steps

1. **Run full test suite**: `bash tests/ssh/run_all_tests.sh --all`
2. **Review evidence**: Check `.sisyphus/evidence/task-N-*.txt` files
3. **Address issues**: Fix any security vulnerabilities or configuration problems
4. **Schedule regular testing**: Run tests before and after deployments

### Related Documentation

- **Main README**: `README.md` - System overview and quick start
- **Deployment Guide**: `deploy/README.md` - Deployment instructions
- **Architecture**: See [System Architecture](#system-architecture) section above
- **API Reference**: See [Component Overview](#component-overview) and individual component sections

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

### Directory Structure

```
armorclaw-omo/
├── bridge/                    # Go Bridge orchestrator
│   ├── cmd/bridge/main.go    # Primary entry point
│   ├── pkg/                  # Core packages (59 total)
│   │   ├── keystore/         # SQLCipher encrypted storage
│   │   ├── rpc/              # JSON-RPC 2.0 server (47 methods)
│   │   ├── matrix/           # Matrix client integration
│   │   ├── docker/           # Container orchestration
│   │   ├── webrtc/           # Voice/video engine
│   │   ├── trust/            # Zero-trust verification
│   │   ├── audit/            # Tamper-evident logging
│   │   ├── budget/           # Token budget tracking
│   │   ├── secrets/          # BlindFill injection
│   │   ├── pii/              # PHI detection/scrubbing
│   │   ├── studio/           # Agent Studio
│   │   └── [50 more]
│   └── internal/             # Internal implementation
│       ├── adapter/          # Matrix/SDTW adapters
│       ├── sdtw/             # Slack/Discord/Teams/WhatsApp bridges
│       └── skills/           # 21 browser skills
│
├── browser-service/          # TypeScript/Playwright automation
│   ├── src/index.ts         # HTTP API entry
│   └── src/browser.ts       # Playwright wrapper with stealth
│
├── container/openclaw-src/   # OpenClaw agent runtime
│   ├── extensions/          # 39 platform adapters
│   ├── skills/              # Browser skills
│   └── src/                 # Core agent logic
│
├── license-server/           # License validation service
│
├── applications/             # Client applications
│   ├── ArmorChat/           # Android Kotlin client
│   ├── ArmorTerminal/       # Terminal client
│   ├── admin-panel/         # Web dashboard
│   └── setup-wizard/        # Setup UI
│
├── deploy/                   # Deployment scripts
│   └── install.sh           # One-command installer
│
├── docs/                     # Documentation
│   ├── index.md             # Main index
│   ├── guides/              # User guides
│   └── reference/           # API reference
│
└── tests/                    # E2E and integration tests
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

type Config struct {
    SocketPath      string
    Keystore        Keystore
    Matrix          MatrixAdapter
    AIService       *ai.AIService
    AIMaxConcurrent int
    BridgeManager   BridgeManager
    BrowserJobs     *BrowserJobManager
    Studio          StudioService
    AppService      AppService
    ProvisioningMgr ProvisioningManager
    SkillManager    SkillManager
    SkillGate       interfaces.SkillGate
    EventBus        *eventbus.EventBus
    HardeningStore  trust.Store
    Metrics         *Metrics
}
```

### RPC API Surface (47 Methods)

#### Health & System (5 methods)

| Method | Parameters | Returns | File |
|--------|------------|---------|------|
| `health.check` | - | `{status, components}` | `pkg/rpc/server.go` |
| `system.health` | - | `{status, timestamp, uptime, checks}` | `pkg/rpc/public_handlers.go` |
| `system.config` | - | `{version, features, endpoints, limits}` | `pkg/rpc/public_handlers.go` |
| `system.info` | - | `{server, protocol, capabilities}` | `pkg/rpc/public_handlers.go` |
| `system.time` | - | `{server_time, server_time_utc}` | `pkg/rpc/public_handlers.go` |

#### Matrix Integration (6 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `matrix.status` | - | `{enabled, connected, logged_in, homeserver, user_id}` | Check Matrix connection status |
| `matrix.login` | `{username, password}` | `{success, user_id}` | Login to Matrix homeserver |
| `matrix.send` | `{room_id, message, msgtype}` | `{event_id, room_id}` | Send message to room |
| `matrix.receive` | `{cursor, timeout_ms}` | `{events[], cursor, count}` | Receive events (long-poll) |
| `matrix.join_room` | `{room_id, via_servers, reason}` | `{room_id}` | Join Matrix room |
| `events.replay` | `{offset, limit}` | `EventLogRecords[]` | Replay events from log |
| `events.stream` | `{offset, timeout_ms}` | `EventLogRecords[]` | Stream events (long-poll) |

#### AI Chat (1 method)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `ai.chat` | `{messages[], model, temperature, max_tokens, key_id}` | `{id, choices[], model, usage}` | AI chat completion (rate-limited) |

#### Browser Automation (11 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `browser.navigate` | `{url, agent_id, job_id}` | `{job_id, status, url}` | Navigate to URL |
| `browser.fill` | `{job_id, selector, value}` | `{job_id, status, selector, success}` | Fill form field |
| `browser.click` | `{job_id, selector}` | `{job_id, status, selector, success}` | Click element |
| `browser.status` | `{job_id}` | `{job_id, status, url, session}` | Get job status |
| `browser.wait_for_element` | `{job_id, selector, timeout}` | `{job_id, status, success}` | Wait for element |
| `browser.wait_for_captcha` | `{job_id}` | `{job_id, status}` | Wait for captcha |
| `browser.wait_for_2fa` | `{job_id}` | `{job_id, status}` | Wait for 2FA |
| `browser.complete` | `{job_id}` | `{job_id, status, completed_at}` | Mark job complete |
| `browser.fail` | `{job_id, reason}` | `{job_id, status, error}` | Mark job failed |
| `browser.list` | - | `{jobs[], count}` | List all jobs |
| `browser.cancel` | `{job_id}` | `{job_id, status, cancelled_at}` | Cancel job |

#### PII Management (9 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `pii.request` | `{agent_id, skill_id, profile_id, variables[], ttl}` | `{request_id, status, expires_at}` | Request PII access |
| `pii.approve` | `{request_id, user_id, approved_fields[]}` | `{request_id, status, approved_at}` | Approve PII access |
| `pii.deny` | `{request_id, user_id, reason}` | `{request_id, status, denied_at}` | Deny PII access |
| `pii.status` | `{request_id}` | `{request_id, status, fields, ...}` | Get request status |
| `pii.list_pending` | - | `{requests[], count}` | List pending requests |
| `pii.stats` | - | `{stats}` | Get PII statistics |
| `pii.cancel` | `{request_id}` | `{request_id, status}` | Cancel request |
| `pii.fulfill` | `{request_id, resolved_vars}` | `{request_id, status}` | Mark fulfilled |
| `pii.wait_for_approval` | `{request_id, timeout}` | `{request_id, status}` | Wait for approval (poll) |

#### Skills (11 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `skills.execute` | `{skill_name, params}` | `SkillResult` | Execute skill |
| `skills.list` | - | `{skills[], count}` | List all skills |
| `skills.get_schema` | `{skill_name}` | `{skill_name, schema}` | Get skill JSON schema |
| `skills.allow` | `{skill_name}` | `{skill_name, status}` | Allow skill |
| `skills.block` | `{skill_name}` | `{skill_name, status}` | Block skill |
| `skills.allowlist_add` | `{type, value}` | `{type, value, status}` | Add to allowlist |
| `skills.allowlist_remove` | `{type, value}` | `{type, value, status}` | Remove from allowlist |
| `skills.allowlist_list` | - | `{ips[], cidrs[]}` | List allowlist |
| `skills.web_search` | `{params}` | `SkillResult` | Execute web.search |
| `skills.web_extract` | `{params}` | `SkillResult` | Execute web.extract |
| `skills.email_send` | `{params}` | `SkillResult` | Execute email.send |

#### Bridge/AppService (6 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `bridge.start` | - | `{status, message}` | Start bridge manager |
| `bridge.stop` | - | `{status, message}` | Stop bridge manager |
| `bridge.status` | `{user_id}` | `{enabled, status, stats}` | Get bridge status |
| `bridge.channel` | `{matrix_room_id, platform, channel_id}` | `{status, ...}` | Bridge channel |
| `bridge.unchannel` | `{platform, channel_id}` | `{status, ...}` | Remove bridge |
| `bridge.list` | - | `{channels[], count}` | List bridges |

#### Studio/Agent (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `studio.deploy` | `{method_name, params}` | Varies | Delegate to Agent Studio |
| `studio.stats` | - | `{agents, instances, skills}` | Get studio stats |

#### Hardening (3 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `hardening.status` | - | `HardeningState` | Get hardening status |
| `hardening.ack` | `{step}` | `HardeningState` | Acknowledge step |
| `hardening.rotate_password` | `{new_password}` | `{success}` | Rotate password |

#### Provisioning (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `provisioning.start` | - | `{setup_token, qr_data, expires_in}` | Start provisioning |
| `provisioning.claim` | `{setup_token, device_id, device_name}` | `{success, role, device_id}` | Claim device |

#### Keystore (2 methods)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `store_key` | `{id, provider, token, display_name, base_url}` | `{success, id, provider}` | Store API key |
| `get_credential` | `{id}` | `{provider, display_name, token}` | Get credential |

### Data Structures

#### Request/Response (JSON-RPC 2.0)

```go
type Request struct {
    JSONRPC string          `json:"jsonrpc"`  // Must be "2.0"
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *ErrorObj   `json:"error,omitempty"`
}

type ErrorObj struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

#### Error Codes

```go
const (
    ParseError       = -32700  // Invalid JSON
    InvalidRequest   = -32600  // Invalid request object
    MethodNotFound   = -32601  // Method does not exist
    InvalidParams    = -32602  // Invalid method parameters
    InternalError    = -32603  // Internal server error
    TooManyRequests  = -32001  // Rate limit exceeded
    RequestCancelled = -32002  // Request cancelled
)
```

### Initialization Flow

```
main() → parseFlags() → runBridgeServer()
  ├─> config.Load()                    // Load TOML config
  ├─> keystore.New() + Open()          // Initialize SQLCipher
  ├─> adapter.New() + Login()          // Connect to Matrix
  ├─> StartSync()                       // Start Matrix sync loop
  ├─> rpc.New()                         // Create RPC server
  ├─> Run(socketPath)                   // Start Unix socket listener
  └─> Start background services
       ├─> bridgeMgr.Start()
       ├─> browserJobs.CleanupOldJobs()
       └─> eventBus.StartWebSocket()
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

**File**: `bridge/pkg/keystore/keystore.go`

```sql
-- Core tables
CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    token_encrypted BLOB NOT NULL,
    nonce BLOB NOT NULL,
    base_url TEXT,
    display_name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER,
    tags TEXT
);

CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE hardware_binding (
    signature_hash TEXT PRIMARY KEY,
    bound_at INTEGER NOT NULL,
    entropy_sources TEXT NOT NULL
);

CREATE TABLE matrix_refresh_tokens (
    id TEXT PRIMARY KEY,
    token_encrypted BLOB NOT NULL,
    nonce BLOB NOT NULL,
    homeserver_url TEXT NOT NULL,
    user_id TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE user_profiles (
    id TEXT PRIMARY KEY,
    profile_name TEXT NOT NULL,
    profile_type TEXT NOT NULL DEFAULT 'personal',
    data_encrypted BLOB NOT NULL,
    data_nonce BLOB NOT NULL,
    field_schema TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_accessed INTEGER,
    is_default INTEGER DEFAULT 0
);

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

-- Indexes
CREATE INDEX idx_provider ON credentials(provider);
CREATE INDEX idx_expires_at ON credentials(expires_at);
CREATE INDEX idx_profile_type ON user_profiles(profile_type);
CREATE INDEX idx_profile_default ON user_profiles(is_default);
```

### SQLCipher Configuration

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

**Connection String**:
```
file:keystore.db?_pragma_key=x'hex_master_key'&_pragma_cipher_page_size=4096&_pragma_kdf_iter=256000&_pragma_cipher_hmac_algorithm=HMAC_SHA512&_pragma_cipher_kdf_algorithm=PBKDF2_HMAC_SHA512&_foreign_keys=ON
```

### Key Derivation Hierarchy

**Priority order for master key source**:

1. **Environment Variable** - `ARMORCLAW_KEYSTORE_SECRET` (base64-encoded 32 bytes)
2. **Secret File** - `keystore.db.key` (base64-encoded)
3. **Container Key** - Random key persisted to file (if in container)
4. **Hardware-Derived Key** - Entropy from machine markers

**Hardware Entropy Sources**:
```go
func (ks *Keystore) collectEntropy() []byte {
    // 1. Machine ID
    /etc/machine-id, /var/lib/dbus/machine-id
    
    // 2. DMI Product UUID (SMBIOS)
    /sys/class/dmi/id/product_uuid
    
    // 3. Primary MAC address
    First non-loopback interface
    
    // 4. Hostname
    
    // 5. OS/Architecture
    runtime.GOOS, runtime.GOARCH
    
    // 6. CPU info
    /proc/cpuinfo (model name, vendor_id)
}
```

### Core Functions

#### Storage

```go
func (ks *Keystore) Store(cred Credential) error {
    // 1. Validate provider and token format
    // 2. Encrypt token using XChaCha20-Poly1305 AEAD
    //    - Generate 24-byte nonce
    //    - Seal plaintext with authenticated encryption
    // 3. Serialize tags to JSON
    // 4. INSERT OR REPLACE into credentials table
    // 5. Log to audit
}
```

#### Retrieval

```go
func (ks *Keystore) Retrieve(id string) (*Credential, error) {
    // 1. Check environment variables first (OPENROUTER_API_KEY, etc.)
    // 2. Fall back to keystore lookup
    // 3. Query credentials table by ID
    // 4. Verify expiration (if expires_at set)
    // 5. Decrypt using XChaCha20-Poly1305
    // 6. Parse tags from JSON
    // 7. Log access to audit
}
```

#### Encryption

```go
func (ks *Keystore) encrypt(plaintext []byte) (encrypted, nonce []byte, err error) {
    // Generate 24-byte nonce for XChaCha20-Poly1305
    nonce = make([]byte, chacha20poly1305.NonceSizeX)
    io.ReadFull(cryptorand.Reader, nonce)
    
    // Create AEAD cipher
    aead, _ := chacha20poly1305.NewX(ks.masterKey)
    
    // Seal with authentication
    encrypted = aead.Seal(nil, nonce, plaintext, nil)
}

func (ks *Keystore) decrypt(encrypted, nonce []byte) ([]byte, error) {
    // Create AEAD cipher
    aead, _ := chacha20poly1305.NewX(ks.masterKey)
    
    // Open and verify (fails if tampered)
    plaintext, err := aead.Open(nil, nonce, encrypted, nil)
    // Failure indicates tampering or corruption
}
```

### BlindFill™ Architecture

**Secret Injection Flow**:

```
Agent requests "credit_card"
    ↓
Bridge checks approval status
    ↓
Bridge decrypts from keystore
    ↓
Bridge injects directly to form
    ↓
Agent never sees value
```

**Injection Methods**:

1. **Unix Domain Socket** (Recommended):
   - Path: `/run/armorclaw/pii/{container}.pii.sock`
   - Protocol: JSON with 4-byte length prefix
   - TTL: 5 seconds
   - Socket deleted after delivery

2. **Environment Variables** (Caution):
   - Prefix: `PII_`
   - Format: `PII_{field_name}={value}`
   - Warning: May be visible in process listings

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

**Standard Matrix Events**:
- `m.room.message` - Text messages and commands
- `m.room.member` - Membership changes
- `m.room.power_levels` - RBAC (admin=50)
- `m.typing` - Typing notifications
- `m.receipt` - Read receipts

**Voice Call Events**:
- `m.call.invite` - Call initiation
- `m.call.answer` - Call acceptance
- `m.call.hangup` - Call termination
- `m.call.candidates` - ICE candidates
- `m.call.negotiate` - SDP renegotiation

**Custom ArmorClaw Events**:
- `app.armorclaw.alert` - System alerts
- `app.armorclaw.pii_request` - PII access request
- `app.armorclaw.pii_response` - PII access response
- `app.armorclaw.consent.request` - Three-way consent request
- `app.armorclaw.consent.response` - Three-way consent response

### Matrix Adapter Interface

**File**: `bridge/pkg/rpc/server.go`

```go
type MatrixAdapter interface {
    SendMessage(roomID, message, msgType string) (string, error)
    SendEvent(roomID, eventType string, content []byte) error
    Login(username, password string) error
    JoinRoom(ctx context.Context, roomIDOrAlias string, viaServers []string, reason string) (string, error)
    GetUserID() string
    IsLoggedIn() bool
    GetHomeserver() string
    GetEventBus() *events.EventBus
}
```

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
| `!agent list-pii` | List PII fields |
| `!agent list-profiles` | List resource profiles |
| `!agent create name="X"` | Start creation wizard |
| `!agent list` | List agent definitions |
| `!agent show <id>` | Show agent details |
| `!agent spawn <id>` | Spawn agent instance |
| `!agent stop <instance>` | Stop running instance |
| `!agent delete <id>` | Delete agent definition |
| `!agent stats` | Show studio statistics |

### Message Processing Pipeline

```
Sync receives events
    ↓
Filter by bridgeSyncFilter
    ↓
Validate sender (isTrustedSender)
    ↓
Validate room (isTrustedRoom)
    ↓
Zero-trust verification
    ├─> Device fingerprinting
    ├─> Trust score check
    └─> Anomaly detection
    ↓
PII scrubbing
    ↓
Command handling (Studio/Admin)
    ↓
Queue/Publish to eventBus
```

### Sync Filter Optimization

```go
var bridgeSyncFilter = map[string]interface{}{
    "room": map[string]interface{}{
        "timeline": map[string]interface{}{
            "limit": 10,
            "types": []string{
                "m.room.message",
                "m.room.member",
                "m.room.bridge",
                "app.armorclaw.alert",
            },
        },
        "state": map[string]interface{}{
            "lazy_load_members": true,
            "types": []string{
                "m.room.member",
                "m.room.bridge",
                "m.room.name",
                "m.room.power_levels",
            },
        },
    },
}
```

---

## Agent Studio

### Purpose

Agent Studio provides **no-code agent creation and management** with:
- Agent definition storage
- Container lifecycle management
- Skill and PII field registry
- Resource profiles (low/medium/high)
- Interactive Matrix command wizard

### Agent Definition Structure

**File**: `bridge/pkg/studio/types.go`

```go
type AgentDefinition struct {
    ID           string        // Unique identifier
    Name         string        // Human-readable name
    Description  string        // What this agent does
    Skills       []string      // List of skill IDs
    PIIAccess    []string      // PII field IDs
    ResourceTier string        // "low", "medium", "high"
    CreatedBy    string        // Matrix user ID
    CreatedAt    time.Time
    UpdatedAt    time.Time
    IsActive     bool
}

type Skill struct {
    ID              string
    Name            string
    Description     string
    Category        string   // "document", "communication", etc.
    ContainerImage  string
    RequiredEnvVars []string
    CreatedAt       time.Time
}

type PIIField struct {
    ID               string
    Name             string
    Description      string
    Sensitivity      string  // "low", "medium", "high", "critical"
    KeystoreKey      string
    RequiresApproval bool
    CreatedAt        time.Time
}

type ResourceProfile struct {
    Tier            string
    MemoryMB        int
    CPUShares       int
    TimeoutSeconds  int
    MaxConcurrency  int
}
```

### Resource Tiers

| Tier | Memory | CPU Shares | Timeout | Max Concurrency |
|------|--------|------------|---------|-----------------|
| low | 256MB | 512 | 300s | 5 |
| medium | 512MB | 1024 | 600s | 3 |
| high | 2048MB | 2048 | 1800s | 1 |

### Agent Lifecycle

**File**: `bridge/pkg/studio/factory.go`

#### Spawn Flow

```
Get Definition
    ↓
Get ResourceProfile
    ↓
Build Environment
  - AGENT_ID, AGENT_NAME
  - ENABLED_SKILLS
  - RESOURCE_TIER
  - TASK_DESCRIPTION
  - PII fields (if available)
    ↓
Create Container
  - User: 10001:10001
  - Memory: profile.MemoryMB
  - NetworkMode: "none"
  - ReadonlyRootfs: true
  - CapDrop: ["ALL"]
    ↓
Start Container
    ↓
Track Instance (StatusRunning)
    ↓
Return Result
```

#### Instance Statuses

```go
const (
    StatusPending   InstanceStatus = "pending"
    StatusRunning   InstanceStatus = "running"
    StatusPaused    InstanceStatus = "paused"
    StatusCompleted InstanceStatus = "completed"
    StatusFailed    InstanceStatus = "failed"
    StatusCancelled InstanceStatus = "cancelled"
)
```

### RPC Methods

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `studio.list_skills` | - | `{skills[]}` | List available skills |
| `studio.get_skill` | `{skill_id}` | `{skill}` | Get skill details |
| `studio.list_pii` | `{sensitivity}` | `{fields[]}` | List PII fields |
| `studio.list_profiles` | - | `{profiles[]}` | List resource profiles |
| `studio.create_agent` | `{definition}` | `{agent_id}` | Create agent definition |
| `studio.get_agent` | `{agent_id}` | `{agent}` | Get agent definition |
| `studio.list_agents` | - | `{agents[]}` | List all agents |
| `studio.update_agent` | `{agent_id, updates}` | `{agent}` | Update agent |
| `studio.delete_agent` | `{agent_id}` | `{success}` | Delete agent |
| `studio.spawn_agent` | `{agent_id, task}` | `{instance_id}` | Spawn instance |
| `studio.get_instance` | `{instance_id}` | `{instance}` | Get instance details |
| `studio.list_instances` | `{agent_id}` | `{instances[]}` | List instances |
| `studio.stop_instance` | `{instance_id}` | `{success}` | Stop instance |
| `studio.stats` | - | `{stats}` | Get statistics |

### Creation Wizard Flow

```
!agent create name="My Agent"
    ↓
Step 1: Select skills by number
    ↓
Step 2: Select PII fields (optional)
    ↓
Step 3: Select resource tier
    ↓
Step 4: Confirm creation
    ↓
Agent definition saved
```

---

## Browser Service

### Purpose

The browser service provides **Playwright-based headless browser automation** with:
- Anti-detection stealth features
- Human-like typing and mouse movement
- Form filling and navigation
- Screenshot capture
- Data extraction

### TypeScript API

**File**: `browser-service/src/browser.ts`

```typescript
export class BrowserClient {
    async initialize(): Promise<void>
    async navigate(cmd: NavigateCommand): Promise<BrowserResponse>
    async fill(cmd: FillCommand): Promise<BrowserResponse>
    async click(cmd: ClickCommand): Promise<BrowserResponse>
    async wait(cmd: WaitCommand): Promise<BrowserResponse>
    async extract(cmd: ExtractCommand): Promise<BrowserResponse>
    async screenshot(cmd: ScreenshotCommand): Promise<BrowserResponse>
}
```

### Command Types

**File**: `browser-service/src/types.ts`

```typescript
interface NavigateCommand {
    url: string;
    waitUntil?: 'load' | 'domcontentloaded' | 'networkidle';
    timeout?: number;  // Default: 30000ms
}

interface FillCommand {
    fields: FillField[];
    auto_submit?: boolean;
    submit_delay?: number;
}

interface FillField {
    selector: string;
    value?: string;
    value_ref?: string;  // PII reference like "payment.card_number"
}

interface ClickCommand {
    selector: string;
    waitFor?: 'none' | 'navigation' | 'selector';
    timeout?: number;
}

interface WaitCommand {
    condition: 'selector' | 'timeout' | 'url';
    value: string;
    timeout?: number;
}

interface ExtractCommand {
    fields: ExtractField[];
}

interface ExtractField {
    name: string;
    selector: string;
    attribute?: string;  // Default: "textContent"
}

interface ScreenshotCommand {
    fullPage?: boolean;
    selector?: string;
    format?: 'png' | 'jpeg';
}
```

### Stealth Features

- Custom user agent
- `navigator.webdriver` override
- Chrome detection bypass
- Permission query handling
- Plugin/languages overrides
- Human-like delays
- Mouse movement with overshoot
- Reading scroll simulation

### HTTP Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/initialize` | POST | Initialize browser |
| `/close` | POST | Close browser |
| `/session` | GET | Get session info |
| `/navigate` | POST | Navigate to URL |
| `/fill` | POST | Fill form fields |
| `/click` | POST | Click element |
| `/wait` | POST | Wait for condition |
| `/extract` | POST | Extract data |
| `/screenshot` | POST | Take screenshot |
| `/workflow` | POST | Execute multi-step workflow |

---

## ArmorChat Android Client

### Purpose

ArmorChat is the **Android mobile client** providing:
- Bridge discovery and pairing
- E2EE messaging via Matrix
- PII approval workflow
- Push notifications
- Biometric authentication
- Security hardening wizard

### Bridge API Client

**File**: `applications/ArmorChat/app/src/main/java/app/armorclaw/network/BridgeApi.kt`

```kotlin
class BridgeApi(private val baseUrl: String) {
    // JSON-RPC 2.0 communication
    
    // Lockdown & Bonding
    fun getLockdownStatus(): Result<LockdownStatus>
    fun getChallenge(): Result<Challenge>
    fun claimOwnership(...): Result<BondingResponse>
    fun transitionMode(target: String): Result<Map<String, String>>
    
    // Security
    fun getSecurityCategories(): Result<List<DataCategory>>
    fun setSecurityCategory(category: String, permission: String): Result<Map<String, Boolean>>
    fun getSecurityTiers(): Result<List<SecurityTier>>
    fun setSecurityTier(tier: String): Result<Map<String, Boolean>>
    
    // Device Management
    fun listDevices(): Result<List<Device>>
    fun approveDevice(deviceId: String, approvedBy: String): Result<Map<String, Boolean>>
    fun rejectDevice(deviceId: String, rejectedBy: String, reason: String): Result<Map<String, Boolean>>
    
    // Verification
    fun startVerification(userId: String, deviceId: String, roomId: String): Result<VerificationStartResponse>
    fun confirmVerification(transactionId: String): Result<VerificationConfirmResponse>
    fun cancelVerification(transactionId: String, reason: String): Result<Map<String, Boolean>>
    
    // PII Access
    fun approvePiiAccess(requestId: String, approvedFields: List<String>): Result<PiiApproveResponse>
    fun rejectPiiAccess(requestId: String, reason: String): Result<PiiRejectResponse>
    
    // Key Backup
    fun createKeyBackup(passphraseHash: String, algorithm: String): Result<BackupCreateResponse>
    
    // Push Notifications
    fun registerPushToken(deviceId: String, token: String, platform: String): Result<PushTokenResponse>
    fun unregisterPushToken(deviceId: String, token: String): Result<PushTokenResponse>
    
    // Hardening
    fun getHardeningStatus(): Result<HardeningStatus>
    fun acknowledgeHardeningStep(step: String): Result<HardeningStatus>
    fun rotateBootstrapPassword(newPassword: String): Result<Map<String, Boolean>>
}
```

### Data Models

```kotlin
@Serializable
data class LockdownStatus(
    val mode: String,
    val admin_established: Boolean,
    val single_device_mode: Boolean,
    val setup_complete: Boolean,
    val security_configured: Boolean,
    val keystore_initialized: Boolean
)

@Serializable
data class HardeningStatus(
    val user_id: String,
    val password_rotated: Boolean,
    val bootstrap_wiped: Boolean,
    val device_verified: Boolean,
    val recovery_backed_up: Boolean,
    val biometrics_enabled: Boolean,
    val delegation_ready: Boolean,
    val created_at: String,
    val updated_at: String
)

@Serializable
data class VerificationStartResponse(
    val transaction_id: String,
    val emojis: List<EmojiData>,
    val status: String
)
```

---

## OpenClaw Agent Runtime

### Purpose

OpenClaw is the **AI agent runtime** that runs inside containers, providing:
- Multi-platform messaging adapters (39 extensions)
- Browser automation via MCP
- Skill execution
- Agent state management

### Extension System

**Location**: `container/openclaw-src/extensions/`

39 platform adapters including:
- Slack, Discord, Teams, WhatsApp
- Matrix, Telegram, Signal
- iMessage, Google Chat, Feishu
- IRC, Nostr, Twitch
- Voice call support

### Browser Tool Integration

**File**: `container/openclaw-src/src/agents/tools/browser-tool.ts`

```typescript
export function createBrowserTool(opts?: {
    sandboxBridgeUrl?: string,
    allowHostControl?: boolean
}): AnyAgentTool {
    return {
        name: "browser",
        description: "Control browser via OpenClaw's browser control server",
        parameters: BrowserToolSchema,
        execute: async (_toolCallId, args) => {
            switch (action) {
                case "status":     // Check browser state
                case "start":      // Start browser
                case "stop":       // Stop browser
                case "profiles":   // List profiles
                case "tabs":       // List tabs
                case "open":       // Open URL
                case "focus":      // Focus tab
                case "close":      // Close tab/browser
                case "snapshot":   // Get AI-friendly snapshot
                case "screenshot": // Take screenshot
                case "navigate":   // Navigate to URL
                case "console":    // Get console messages
                case "pdf":        // Save as PDF
                case "upload":     // Upload files
                case "dialog":     // Handle alerts
                case "act":       // Execute action
            }
        }
    };
}
```

### 21 Browser Skills

| Skill | Risk | Description |
|-------|------|-------------|
| `browser_click` | medium | Click element |
| `browser_fill` | medium | Fill form fields |
| `browser_fill_with_pii` | **critical** | Fill with PII (requires approval) |
| `browser_screenshot` | low | Take screenshot |
| `browser_snapshot` | low | Get AI-friendly snapshot |
| `browser_navigate` | low | Navigate to URL |
| `browser_console_inspect` | medium | Inspect console |
| `browser_emulate` | low | Emulate device |
| `browser_eval_privileged` | **high** | Execute privileged JS |
| `browser_extract_page` | low | Extract page data |
| `browser_fill_form` | medium | Fill entire form |
| `browser_form_submit` | medium | Submit form |
| `browser_lighthouse_audit` | low | Run Lighthouse audit |
| `browser_list_pages` | low | List open tabs |
| `browser_login_assist` | high | Assist with login |
| `browser_memory_snapshot` | medium | Get storage snapshot |
| `browser_network_inspect` | medium | Inspect network |
| `browser_resize` | low | Resize viewport |
| `browser_select_page` | low | Select tab |
| `browser_trace_performance` | low | Trace performance |
| `browser_upload_document` | medium | Upload file |
| `browser_wait_for` | low | Wait for condition |

---

## Security Architecture

### Container Isolation

```go
// Agent container security configuration
User:           "10001:10001"  // Non-root
Memory:         profile.MemoryMB * 1024 * 1024
MemorySwap:     profile.MemoryMB * 1024 * 1024  // No swap
CPUShares:      profile.CPUShares
NetworkMode:    "none"  // Isolated by default
ReadonlyRootfs: true
SecurityOpt:    ["no-new-privileges:true"]
CapDrop:        ["ALL"]
Privileged:     false
PidsLimit:      100
```

### Hardware Binding

Database bound to specific machine via:
- Machine ID (`/etc/machine-id`)
- DMI Product UUID (SMBIOS)
- Primary MAC address
- Hostname
- CPU info

**Result**: Database cannot be decrypted if moved to different server.

### Approval Workflow

```
Agent requests PII
    ↓
PIIRequestManager.CreateRequest()
  - Status: pending
  - Expires: 5 minutes
    ↓
Matrix event sent to user room
    ↓
User reviews and approves/denies
    ↓
If approved:
  - BlindFillEngine.ResolveVariables()
  - Only approved fields returned
    ↓
PIIInjector.InjectPII()
  - Memory-only delivery
  - Agent never sees raw values
```

### Audit Logging

**Critical events logged**:
- PII access requests
- PII access granted/denied
- PII injected (method, fields)
- Profile created/updated/deleted
- Key access attempts
- Profile access

**Audit log**: `/var/lib/armorclaw/audit.db`
**Max entries**: 10,000 (rotating)

---

## Data Flow Patterns

### Agent Request Flow

```
User (ArmorChat)
    ↓ Matrix message
Matrix Homeserver
    ↓ Event
Bridge (Matrix client)
    ↓ Queue
Agent Container (OpenClaw)
    ↓ Browser command
Browser Service
    ↓ Playwright
Target Website
```

### BlindFill™ Flow

```
Agent requests "credit_card"
    ↓
Bridge checks approval
    ↓
Bridge decrypts from keystore
    ↓
Bridge injects directly to form
    ↓
Agent never sees value
```

### Approval Flow

```
Sensitive action detected
    ↓
Bridge creates approval request
    ↓
Matrix event to user's room
    ↓
User approves (reaction/button)
    ↓
Bridge executes action
    ↓
Audit log updated
```

---

## Configuration Reference

### Environment Variables

#### Core Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENROUTER_API_KEY` | OpenRouter API key | - |
| `OPEN_AI_KEY` | OpenAI API key | - |
| `ZAI_API_KEY` | xAI API key | - |
| `ARMORCLAW_SERVER_NAME` | Server hostname | auto-detected |
| `ARMORCLAW_EXTERNAL_MATRIX` | Use external Matrix | `false` |
| `ARMORCLAW_MATRIX_HOMESERVER_URL` | External Matrix URL | `http://127.0.0.1:6167` |
| `ARMORCLAW_MATRIX_ENABLED` | Enable Matrix | `true` |
| `ARMORCLAW_ADMIN_PASSWORD` | Admin password | auto-generated |
| `ARMORCLAW_KEYSTORE_SECRET` | Keystore master key (base64) | - |
| `CONDUIT_VERSION` | Conduit version | latest |

#### Deployment Mode Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ARMORCLAW_SERVER_MODE` | Server mode: `native` or `sentinel` | `native` |
| `ARMORCLAW_RPC_TRANSPORT` | RPC transport: `unix` or `tcp` | `unix` |
| `ARMORCLAW_SOCKET_PATH` | Unix socket path (native mode) | `/run/armorclaw/bridge.sock` |
| `ARMORCLAW_LISTEN_ADDR` | TCP listen address (sentinel mode) | `0.0.0.0:8080` |
| `ARMORCLAW_PUBLIC_BASE_URL` | Public URL (sentinel mode) | - |
| `ARMORCLAW_ADMIN_TOKEN` | Admin authentication token | auto-generated |
| `ARMORCLAW_MATRIX_SECRET` | Matrix Conduit secret | auto-generated |
| `ARMORCLAW_EMAIL` | Email for Let's Encrypt (sentinel mode) | - |
| `ARMORCLAW_PUBLIC_IP` | Detected public IP | auto-detected |

#### Docker Compose Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ARMORCLAW_BRIDGE_HOST_PORT` | Host port for bridge | `8081` |
| `ARMORCLAW_BRIDGE_CONTAINER_PORT` | Container port for bridge | `8080` |
| `CADDY_HTTP_PORT` | Caddy HTTP port | `80` |
| `CADDY_HTTPS_PORT` | Caddy HTTPS port | `443` |
| `CADDY_CONFIG_PATH` | Caddyfile path | `/etc/armorclaw` |

### Configuration Files

| File | Purpose | Format |
|------|---------|--------|
| `/etc/armorclaw/config.toml` | Bridge configuration | TOML |
| `/var/lib/armorclaw/keystore.db` | Encrypted credentials | SQLCipher |
| `/etc/armorclaw/providers.json` | AI provider registry | JSON |
| `/etc/matrix-conduit/conduit.toml` | Matrix homeserver | TOML |
| `/run/armorclaw/bridge.sock` | RPC socket | Unix socket |

### Docker Compose Services

| Service | Port | Description | Profile |
|---------|------|-------------|---------|
| `armorclaw-sentinel` | 8080/8081 | ArmorClaw orchestrator | - (always) |
| `matrix` | 6167 | Conduit homeserver | - (always) |
| `sygnal` | 5000 | Push notifications | - (always) |
| `caddy` | 80, 443 | Reverse proxy with TLS | `sentinel` |
| `coturn` | 3478 | TURN server for WebRTC | - (always) |

**Starting Services:**
```bash
# Native mode (local only)
docker-compose up -d

# Sentinel mode (public with TLS)
docker-compose --profile sentinel up -d
```

---

## Quick Reference

### Common RPC Calls

```bash
# Health check
echo '{"jsonrpc":"2.0","id":1,"method":"health.check"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Matrix status
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# List API keys
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Studio stats
echo '{"jsonrpc":"2.0","id":1,"method":"studio.stats"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Common Matrix Commands

```
!agent create name="Researcher" skills="web_browsing"
!agent list
!agent spawn <agent-id>
!agent stop <instance-id>
/ai providers
/ai switch openai gpt-4o
```

### File Locations

| Purpose | Path |
|---------|------|
| Bridge binary | `/usr/local/bin/armorclaw-bridge` |
| Config | `/etc/armorclaw/config.toml` |
| Environment | `/etc/armorclaw/.env` |
| Keystore | `/var/lib/armorclaw/keystore.db` |
| Audit log | `/var/lib/armorclaw/audit.db` |
| RPC socket (native) | `/run/armorclaw/bridge.sock` |
| Caddyfile (sentinel) | `/etc/armorclaw/Caddyfile` |
| Install log | `/var/log/armorclaw/install.log` |

### Configuration Files

| File | Purpose | Mode |
|------|---------|------|
| `configs/.env.example` | Environment variable documentation | Both |
| `configs/Caddyfile.template` | Caddy reverse proxy template | Sentinel |

---

## Document Index

### Key Source Files

| Component | Key Files |
|-----------|-----------|
| **Bridge** | `bridge/cmd/bridge/main.go`, `bridge/pkg/rpc/server.go` |
| **Config** | `bridge/pkg/config/config.go`, `bridge/pkg/config/loader.go` |
| **Discovery** | `bridge/pkg/discovery/http.go` |
| **Keystore** | `bridge/pkg/keystore/keystore.go` |
| **Matrix** | `bridge/internal/adapter/matrix.go` |
| **Studio** | `bridge/pkg/studio/types.go`, `bridge/pkg/studio/factory.go` |
| **Browser** | `browser-service/src/browser.ts` |
| **Android** | `applications/ArmorChat/app/src/main/java/app/armorclaw/network/BridgeApi.kt` |
| **OpenClaw** | `container/openclaw-src/src/agents/tools/browser-tool.ts` |
| **Installer** | `deploy/install.sh`, `deploy/installer-v6.sh` |

### Documentation

| Doc | Path |
|-----|------|
| Main README | `/README.md` |
| Guardrails | `/AGENTS.md` |
| Doc Index | `/docs/index.md` |
| RPC API | `/docs/reference/rpc-api.md` |
| Setup Guide | `/docs/guides/setup-guide.md` |
| Troubleshooting | `/docs/guides/troubleshooting.md` |

---

*End of ArmorClaw System Documentation*
