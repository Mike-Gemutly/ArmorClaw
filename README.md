# ArmorClaw 🦞🔒

## The Secure Containment Layer for AI Agents

> **Deploy AI agents in production — without exposing your infrastructure.**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)
[![Status](https://img.shields.io/badge/status-v4.1.0--beta-orange.svg)](https://github.com/armorclaw/armorclaw/releases)
[![Security](https://img.shields.io/badge/security-hardened-green.svg)](docs/guides/security-verification-guide.md)
[![Docker](https://img.shields.io/badge/docker-24%2B-blue.svg)](https://docs.docker.com/)
[![Matrix](https://img.shields.io/badge/protocol-Matrix%20E2EE-purple.svg)](https://matrix.org/)

---

## 🚨 Why ArmorClaw Exists

AI agents are powerful. They can:

* Execute shell commands
* Access local files
* Make outbound network requests
* Interact with APIs using sensitive keys

Most teams run agents:

* Directly on their machine
* Inside lightly configured Docker containers
* With plaintext `.env` API keys
* With open localhost ports
* With no audit trail

**This is a production security risk.**

ArmorClaw adds a **hardened containment boundary** between your AI agent and your infrastructure.

---

## 🛡 What ArmorClaw Provides

### 1️⃣ Hardened Container Runtime

Each agent runs in a locked-down environment with:

| Security Layer            | Protection                            |
| ------------------------- | ------------------------------------- |
| `--network=none`          | Blocks outbound data exfiltration     |
| `--cap-drop=ALL`          | Removes privilege escalation vectors  |
| `--read-only` root        | Prevents filesystem modification      |
| Seccomp syscall filtering | Kernel-level system call restrictions |
| `chmod a-x` on binaries   | Prevents shell execution              |
| LD_PRELOAD hooks          | Blocks runtime process spawning       |

**Result:** Even if an agent is jailbroken, it is trapped.

### 2️⃣ Ephemeral Secret Injection

* API keys injected in-memory only
* No plaintext `.env` files
* Secrets vanish on shutdown
* Hardware-backed keystore support

**Result:** Compromised agent ≠ compromised credentials.

### 3️⃣ Zero-Trust Architecture

* No inbound ports
* No Docker socket exposure
* Pull-based activity visibility
* Strict isolation boundaries

ArmorClaw assumes the agent may fail — and designs for containment.

### 4️⃣ Secure Mobile Access

#### ArmorChat 💬

**End-to-end encrypted messaging with your AI agents.**
Powered by Matrix protocol — server cannot read message content.

#### ArmorTerminal 💻

**Secure terminal access with real-time streaming.**
Multi-agent management from your mobile device.

#### Client Applications

| Platform      | Client         | E2EE | Status      |
|---------------|----------------|------|-------------|
| Android       | ArmorChat      | ✅   | Beta        |
| Android       | ArmorTerminal  | ✅   | Beta        |
| iOS           | ArmorChat      | ✅   | Coming Soon |
| Desktop       | ArmorTerminal  | ✅   | In Dev      |
| Any OS        | Element X      | ✅   | ✅ Works    |
| Browser       | Element Web    | ✅   | ✅ Works    |

Mobile access extends control without exposing your infrastructure.

---

## 🎯 Who ArmorClaw Is For

ArmorClaw is built for teams running AI agents in production workflows:

### 🔒 Security & Compliance Teams

* Testing AI safely before internal rollout
* Enforcing containment policies
* Reducing data exfiltration risk

### 🚀 AI-Native Startups

* Running RAG systems with internal documents
* Deploying agents connected to company APIs
* Needing security approval before production use

### 🏢 Engineering Teams

* Building internal copilots
* Connecting agents to databases
* Handling sensitive business logic

**If your AI agents access proprietary data, you need containment.**

---

## 🧪 Current Status

**v4.1.0-beta — Docker Deployment Hardening Release**

This release implements multi-layer defense against:

* Shell escape attempts
* Process spawning
* Outbound network exfiltration
* Filesystem abuse
* Raw syscall bypass attempts

### Test Results

| Test Category                 | Result    |
| ----------------------------- | --------- |
| Python shell spawn            | ✅ Blocked |
| Node process spawn            | ✅ Blocked |
| urllib/fetch exfiltration     | ✅ Blocked |
| Direct shell execution        | ✅ Blocked |
| Privilege escalation attempts | ✅ Blocked |

Community validation in progress before v1.0 production release.

---

## ⚡ Quick Start

### Prerequisites

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| **OS** | Ubuntu 22.04, Debian 12 | Ubuntu 24.04 |
| **RAM** | 1 GB | 2 GB |
| **Docker** | 24.0+ | Latest |
| **Go** | 1.21+ | 1.24+ |

### Install (2-3 minutes)

```bash
# 1. Clone the repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# 2. Run quick setup
sudo ./deploy/setup-quick.sh

# 3. Add your API key
sudo armorclaw-bridge add-key --provider openai --token sk-...

# 4. Start an agent
sudo armorclaw-bridge start --key openai-main
```

### Verify Installation

```bash
# Check bridge status
sudo systemctl status armorclaw-bridge

# Test health via RPC
echo '{"jsonrpc":"2.0","method":"health","id":1}' | \
  sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

**Time to production:** ~3 minutes.

---

## 🌐 Deployment Options

| Method       | Domain Required | SSL      | Use Case           |
|--------------|-----------------|----------|---------------------|
| **Docker One-Command** | No | Self-signed | Simplest setup |
| Quick Setup  | No              | Self-signed | Development, testing |
| Standard     | Yes             | Let's Encrypt | Production        |
| Docker Stack | Optional        | Built-in | Full infrastructure |
| VPS Deploy   | Optional        | Configurable | Remote hosting    |

### Docker One-Command (Simplest)

**No YAML files. No manual config. Just run and answer questions.**

```bash
docker run -it --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-data:/etc/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

**The setup wizard asks:**
1. Server name (domain or IP)
2. API provider and key (OpenAI, Anthropic, GLM-5, or custom)
3. Matrix enabled? (optional for remote access)

All prompts have retry-on-error — a typo re-prompts instead of killing the container.

**After setup completes:**
- ✅ Bridge running
- ✅ QR code displayed for ArmorChat
- ✅ All configs auto-generated

**Non-interactive (CI/CD):**
```bash
docker run -d --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-data:/etc/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  -e ARMORCLAW_SERVER_NAME=your-domain.com \
  -e ARMORCLAW_API_KEY=sk-your-key \
  mikegemut/armorclaw:latest
```

### IP-Only Deployment (No Domain Required)

ArmorClaw supports deployment without a domain:
- Uses HTTP mode for IP addresses
- Auto-generates self-signed certificates
- QR provisioning works with IP address

```bash
# During setup, enter IP instead of domain
# Example: 123.45.67.89 instead of matrix.example.com
```

---

## 🏗 Architecture Overview

ArmorClaw separates responsibilities across secure layers:

```
┌─────────────────────────────────────────────────────────────┐
│                     CLIENT LAYER                             │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │  ArmorChat    │  │ ArmorTerminal │  │  Element X    │   │
│  │  (Android)    │  │  (Desktop)    │  │  (Any OS)     │   │
│  │  ✅ Full E2EE │  │  ✅ Full E2EE  │  │  ✅ Full E2EE  │   │
│  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘   │
└──────────┼──────────────────┼──────────────────┼───────────┘
           │                  │                  │
           ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                  COMMUNICATION LAYER                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Matrix Protocol (E2EE) + JSON-RPC 2.0 + WebSocket  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────┐
│                   LOCAL BRIDGE (Go)                          │
│  • Unix socket: /run/armorclaw/bridge.sock                  │
│  • Encrypted keystore (hardware-bound)                       │
│  • Docker client (scoped operations)                         │
│  • Matrix adapter (E2EE support)                             │
└──────────────────────────┬──────────────────────────────────┘
                           │ Pull-based communication
                           │ File descriptor passing (secrets)
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              HARDENED CONTAINER (Docker)                     │
│  • Base: debian:bookworm-slim                               │
│  • User: UID 10001 (non-root)                               │
│  • No shell, no network tools, no destructive commands      │
│  • Secrets: Memory-only (never on disk)                     │
│  • No Docker socket, no inbound ports                       │
└─────────────────────────────────────────────────────────────┘
```

**No direct system access. No exposed sockets. No unsafe defaults.**

---

## 📚 Documentation

### Getting Started
- **[Setup Guide](docs/guides/setup-guide.md)** — Get started in 2-3 minutes
- **[Element X Quick Start](docs/guides/element-x-quickstart.md)** — Connect via mobile app

### Architecture & Design
- **[Architecture Overview](docs/plans/2026-02-05-armorclaw-v1-design.md)** — How it works
- **[Architecture Review](docs/output/review.md)** — Implementation snapshot

### API Reference
- **[RPC API Reference](docs/reference/rpc-api.md)** — Complete JSON-RPC 2.0 API
- **[Error Catalog](docs/guides/error-catalog.md)** — Search errors by text (LLM-friendly)

### Deployment
- **[VPS Deployment Guide](docs/guides/2026-02-05-infrastructure-deployment-guide.md)** — Deploy to any VPS
- **[Hostinger Deployment](docs/guides/hostinger-deployment.md)** — Step-by-step for Hostinger

### Security
- **[Security Verification Guide](docs/guides/security-verification-guide.md)** — Manual verification
- **[Security Configuration](docs/guides/security-configuration.md)** — Zero-trust setup

---

## 🛣 Roadmap to v1.0

| Version | Feature                    | Target    | Status |
|---------|---------------------------|-----------|--------|
| v0.1.0  | Multi-layer security hardening | 2026-02-09 | ✅ Complete |
| v4.0.0  | Zero-trust & audit system | 2026-02-19 | ✅ Complete |
| v4.1.0  | Docker deployment hardening | 2026-02-24 | ✅ Complete |
| v0.5.0  | Policy engine             | Q1 2026   | 🚧 In Progress |
| v1.0.0  | Enterprise ready          | Q3 2026   | 📋 Planned |

ArmorClaw is evolving from **secure runtime** → **enterprise AI containment platform**.

---

## 💰 Licensing

| Tier | Use Case | Price |
|------|----------|-------|
| **Community** | Local development, testing | Free |
| **Pro** | Individual builders | $29/mo |
| **Team** | Startups & small orgs | $99/mo |
| **Enterprise** | Security & compliance teams | Custom |

**[Join the Beta Program](#-beta-program)** — Free lifetime licenses for first 100 testers.

---

## 🎟️ Beta Program — Limited Licenses Available

We're offering **free lifetime licenses** to our first batch of beta testers. Spots are limited!

**Beta Tester Benefits:**
- 🎁 **Free lifetime license** (first 100 accepted testers per app)
- 🚀 **Early access** to all new features
- 💬 **Direct feedback channel** to developers
- 🏆 **Beta tester badge** and community recognition
- 🔐 **Priority support** for setup and issues

**How to Join:**

1. Star the repo: [github.com/armorclaw/armorclaw](https://github.com/armorclaw/armorclaw)
2. Open an issue with title: `Beta Test Request: ArmorChat` or `Beta Test Request: ArmorTerminal`
3. Add the label: `beta-request`

**Current Beta Status:**

| App | Slots Filled | Slots Remaining |
|-----|--------------|-----------------|
| ArmorChat | 23/100 | 77 available |
| ArmorTerminal | 12/100 | 88 available |

> ⚡ **Act fast!** Once we hit 100 testers per app, beta access will require a paid license.

---

## 📦 Why Not Just Use Docker?

Docker provides isolation.
ArmorClaw provides:

| Feature | Docker | ArmorClaw |
|---------|--------|-----------|
| AI-specific hardening | ❌ | ✅ |
| Multi-layer defense | ❌ | ✅ |
| Exploit-tested containment | ❌ | ✅ |
| Secret lifecycle control | ❌ | ✅ |
| Secure mobile interface | ❌ | ✅ |
| E2EE messaging | ❌ | ✅ |
| Agent-focused security model | ❌ | ✅ |

ArmorClaw is **purpose-built for AI agents** — not generic containers.

---

## 🤝 Contributing

We are actively seeking:

* Security researchers
* AI engineers running agents in production
* Compliance-focused teams

**Open an issue** labeled `beta-test` to participate.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 🧪 Testing

```bash
# Quick security verification
./tests/verify-security.sh

# Full exploit suite (26 tests)
./tests/test-exploits.sh

# Run all tests
make test-all
```

### Security Test Results

| Test Group | Tests | Status |
|------------|-------|--------|
| Shell Escape Attempts | 4 | ✅ All Blocked |
| Network Exfiltration | 3 | ✅ All Blocked |
| Filesystem Containment | 4 | ✅ All Contained |
| Secret Inspection | 3 | ✅ Expected Behavior |
| Privilege Escalation | 3 | ✅ All Blocked |
| Dangerous Tools | 9 | ✅ All Removed |

---

## 📮 Support

- **Issues:** [GitHub Issues](https://github.com/armorclaw/armorclaw/issues)
- **Documentation:** [docs/index.md](docs/index.md)
- **License:** [MIT License](LICENSE.md)

---

## 🔐 Final Thought

AI agents are not static tools.

They reason.
They execute.
They improvise.

**ArmorClaw ensures they cannot escape.**

---

**⚡ ArmorClaw — Run powerful AI safely.**

**🙏 Thank you for helping us make AI safer for everyone!**

© 2026 Gemutly
[armorclaw.com](https://armorclaw.com)
