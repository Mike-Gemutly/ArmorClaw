# ArmorClaw: The VPS Secretary Platform

[![Version](https://img.shields.io/badge/version-v4.3.0-blue)](https://github.com/armorclaw/armorclaw)
[![Status](https://img.shields.io/badge/status-production%20ready-green)](https://github.com/armorclaw/armorclaw)

**Run AI agents on your VPS. Control from your phone.**

ArmorClaw runs AI agents 24/7 on your server. They browse websites, fill forms, and manage tasks—while you approve sensitive actions via your phone.

---

## Quick Start (2 minutes)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

**That's it.** The wizard asks 3 questions and sets everything up.

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

[QR CODE - Scan with ArmorChat]
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

### One-Line Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

### Deployment Modes

| Mode | Command | Use Case |
|------|---------|----------|
| **Full Stack** | `bash` (default) | ArmorChat mobile integration |
| **Bridge-only** | `bash -s -- --bridge-only` | Testing, no Matrix |
| **Bootstrap** | `bash -s -- --bootstrap` | Generate docker-compose.yml |

### Non-Interactive (CI/CD)

```bash
# Minimal - auto-detects server IP
export ARMORCLAW_API_KEY=sk-your-key
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash

# Or with explicit server IP/domain
export ARMORCLAW_API_KEY=sk-your-key
export ARMORCLAW_SERVER_NAME=192.168.1.50
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

### Bootstrap Mode (GitOps)

Generates production-ready config:

```bash
curl -fsSL ... | bash -s -- --bootstrap
```

Output: `/opt/armorclaw/docker-compose.yml`

Use for:
- Version-controlled infrastructure
- CI/CD pipelines
- Terraform/GitOps workflows

---

## Advanced / Manual Setup

> **Most users should use the one-liner above.** This section is for advanced customization.

### Generate Config (GitOps/CI/CD)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --bootstrap
```

Creates `/opt/armorclaw/docker-compose.yml` for version control.

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
| **Memory-Only Secrets** | API keys never written to disk |
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

## Build from Source

```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Full stack (Bridge + Matrix + Sygnal + Caddy)
docker compose -f docker-compose-full.yml up -d --build

# Or build just the quickstart image
docker build -t armorclaw/quickstart:latest -f Dockerfile.quickstart .
```

**Note:** Bridge requires Debian-based images for SQLCipher compatibility.

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

---

## Links

* **GitHub:** https://github.com/armorclaw/armorclaw
* **Docker Hub:** https://hub.docker.com/r/mikegemut/armorclaw

## License

[MIT License](LICENSE)
