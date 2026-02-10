# ArmorClaw 🦞🔒

> **Secure containment for powerful AI agents** — Run GPT-4, Claude, and other agents safely with hardened containers, ephemeral secrets, and strict isolation.

---

## 🎯 Why ArmorClaw?

**Standard AI setups are risky:**
- Agents run directly on your computer with full filesystem access
- API keys stored in plaintext `.env` files
- Open localhost ports can be exploited
- No audit trail of what agents do

**ArmorClaw 🦞🔒 solves this:**
- ✅ **Hardened container** — Agent runs in a locked Docker container (non-root, no shell)
- ✅ **Ephemeral secrets** — API keys injected into memory only, vanish on shutdown
- ✅ **Strict isolation** — No inbound ports, no Docker socket exposure
- ✅ **Pull-based visibility** — See agent activity through a signed Local Bridge
- ✅ **Zero-trust architecture** — Even if the agent is compromised, it's trapped

**Who uses ArmorClaw 🦞🔒 ?**
- 🔒 **Security teams** — Test AI tools safely without risking company data
- 🏢 **Professionals** — Draft sensitive documents, analyze reports, brainstorm ideas
- 🏠 **Home users** — Run powerful AI locally without exposing your whole computer
- 🎓 **Students & Researchers** — Experiment with AI in a controlled environment

---

## ⚡ Quick Start

### Option 1: Element X (Mobile) ⭐ FASTEST

Connect to your AI agent via Element X mobile app in 5 minutes:

```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./deploy/launch-element-x.sh
```

**That's it!** Scan the QR code with Element X and start chatting.

**Time:** 5 minutes | **Experience Level:** Beginner | **Platform:** Mobile-friendly

---

### Option 2: Interactive Setup Wizard ⭐ (Recommended)

The easiest way to get started on desktop:

```bash
# Clone repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Run the interactive setup wizard
./deploy/setup-wizard.sh
```

The wizard guides you through:
- ✅ System requirements validation
- ✅ Docker installation/verification
- ✅ Container image setup
- ✅ Bridge compilation
- ✅ Encrypted keystore initialization
- ✅ First API key configuration
- ✅ Systemd service setup
- ✅ Post-installation verification

**Time:** 10-15 minutes | **Experience Level:** Beginner

### Option 3: One-Command Install

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy.sh | bash
```

### Option 4: Docker Compose (Quick Test)

```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./deploy/launch-element-x.sh
```

Deploys Matrix, Caddy, and Bridge with auto-provisioning.

**Time:** 5 minutes | **Experience Level:** Intermediate

### Option 5: VPS Deployment 🆓

Deploy to a remote VPS (Hostinger, DigitalOcean, etc.) with automated script:

```bash
# 1. Create deployment tarball
cd armorclaw
tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' .

# 2. Transfer to VPS
scp deploy/vps-deploy.sh armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# 3. Run deployment script on VPS
ssh user@your-vps-ip
chmod +x /tmp/vps-deploy.sh
sudo bash /tmp/vps-deploy.sh
```

The automated script handles:
- ✅ Pre-flight checks (disk, memory, ports)
- ✅ Docker installation
- ✅ Tarball verification and extraction
- ✅ Interactive configuration
- ✅ Automated deployment

**Time:** 10-15 minutes | **Experience Level:** Intermediate | **Platform:** VPS

---

That's it! The agent will be running in a hardened container, and you can connect via **Element X** (Matrix) or any compatible client.

---

## 🆕 New Features (v1.2)

### Shell Completion

Tab completion for bash/zsh - type less, do more:

```bash
./build/armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
source ~/.bash_completion.d/armorclaw-bridge
./build/armorclaw-bridge <TAB>  # Auto-complete commands
```

### Daemon Mode

Run the bridge as a background service:

```bash
./build/armorclaw-bridge daemon start   # Start in background
./build/armorclaw-bridge daemon status  # Check status
```

### Enhanced Help

Every command has detailed help with examples:

```bash
./build/armorclaw-bridge --help            # Overview with examples
./build/armorclaw-bridge add-key --help    # Command-specific guide
```

### Element X Integration

Connect via Element X mobile app - chat with your agent from anywhere:

```bash
./deploy/launch-element-x.sh    # Launch Matrix + Bridge
# Scan QR code with Element X mobile app
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Your Computer                           │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            Local Bridge (Go Binary)                   │  │
│  │  • Unix socket: /run/armorclaw/bridge.sock          │  │
│  │  • Encrypted keystore (hardware-bound)                │  │
│  │  • Matrix client for remote communication           │  │
│  └───────────────┬───────────────────────────────────────┘  │
│                  │ Pull-based communication               │
│                  │ File descriptor passing (secrets)       │
│                  ▼                                         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         Docker Container (Hardened)                   │  │
│  │  • Base: debian:bookworm-slim                        │  │
│  │  • User: UID 10001 (non-root)                          │  │
│  │  • No shell, no network tools, no destructive commands  │  │
│  │  • Secrets: Memory-only (never on disk)                 │  │
│  │  • No Docker socket, no inbound ports                    │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔒 Security Features

| Feature | How It Works | Benefit |
|---------|--------------|---------|
| **Container Hardening** | Non-root user, removed shell/tools, seccomp profile | Agent can't escape even if exploited |
| **Ephemeral Secrets** | API keys injected via file descriptor, never written to disk | No secrets in logs, docker inspect, or disk |
| **Hardware-Bound Keystore** | Master key derived from machine-id, DMI UUID, MAC | Database useless if stolen/moved |
| **Zero-Touch Reboot** | Salt persistence enables automatic key derivation | No manual intervention after reboot |
| **Pull-Based Visibility** | All communication through signed Local Bridge | No direct agent access, full audit trail |
| **No Inbound Ports** | Container has no exposed network services | Can't be attacked from outside |
| **Zero-Trust Filtering** ✅ NEW | Trusted Matrix senders/rooms allowlist | Only authorized users can control agents |
| **Budget Guardrails** ✅ NEW | Daily/monthly spending limits with hard-stop | Prevent unexpected API costs |
| **PII Data Scrubbing** ✅ NEW | Auto-redacts emails, SSNs, API keys, credit cards | Protects sensitive data in prompts/responses |
| **Container TTL** ✅ NEW | Auto-removes idle containers (10 min default) | Prevents resource leaks |
| **Host Hardening** ✅ NEW | Automated firewall (UFW) & SSH hardening | Production-ready security baseline |

> **Build Process Note**: The Docker container build process has been optimized to prevent circular dependencies in security hardening while maintaining all security protections.
>
> **Fix Applied**: Resolved circular dependency bug where the `rm` command was trying to delete itself during the security hardening phase. The fix removes `/bin/rm` from the list of files to delete in the Docker build process (line 88 of Dockerfile).
>
> **Fix Details**: 
> - Previously, line 88 of Dockerfile contained: `RUN /bin/rm -f /bin/bash /bin/sh /bin/dash /bin/mv /bin/find && \` 
> - The `/bin/rm` command was attempting to delete itself while executing, causing the build to fail with exit code 125
> - Solution: Removed `/bin/rm` from the deletion list to prevent the self-deletion loop

**Compliance Ready:** Supports GDPR, HIPAA, SOC 2 requirements through data isolation, audit logging, and access controls.

> **📖 Security Configuration Guide:** See [docs/guides/security-configuration.md](docs/guides/security-configuration.md) for complete security feature documentation.

---

## 🚀 Usage

### Start an Agent

```bash
# Via Local Bridge socket
echo '{"jsonrpc":"2.0","method":"start","params":{"name":"my-agent"},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Check Status

```bash
# Service status
docker-compose -f docker-compose-stack.yml ps

# Bridge status
echo '{"jsonrpc":"2.0","method":"status","id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Stop an Agent

```bash
echo '{"jsonrpc":"2.0","method":"stop","params":{"name":"my-agent"},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## 📚 Documentation

### Quick Start
- **[Quick Start Guide](docs/guides/element-x-quickstart.md)** ⭐ — Get started in 5 minutes with Element X
- **[Setup Guide](docs/guides/setup-guide.md)** — Comprehensive setup with multiple methods
- **[Element X Connection](docs/guides/element-x-quickstart.md)** — Connect to agents via Element X app

### Deployment
- **[Hostinger VPS Deployment](docs/guides/hostinger-deployment.md)** 🆓 — Build .tar and deploy to Hostinger VPS
- **[Hostinger Docker Deployment](docs/guides/hostinger-docker-deployment.md)** 🆓 — Complete Docker deployment guide for Hostinger VPS (Docker Manager + CLI)

### Architecture & Design
- **[Documentation Hub](docs/index.md)** — Central navigation to all documentation
- **[V1 Architecture](docs/plans/2026-02-05-armorclaw-v1-design.md)** — Complete system design and architecture
- **[Architecture Review](docs/output/review.md)** — Implementation snapshot with all functions

### Reference
- **[RPC API Reference](docs/reference/rpc-api.md)** — Complete JSON-RPC 2.0 API documentation
- **[Infrastructure Deployment Guide](docs/guides/2026-02-05-infrastructure-deployment-guide.md)** — Comprehensive deployment documentation

### Navigation Path for LLMs
**README** → **[Documentation Hub](docs/index.md)** → **[V1 Architecture](docs/plans/2026-02-05-armorclaw-v1-design.md)** → **[Reference Docs](docs/reference/)** → **[Function Catalog](docs/output/review.md)**
> 💡 **See [Navigation Guide](docs/NAVIGATION.md)** for complete LLM navigation documentation (README → Architecture → Feature → Functions)

---

## 🧪 Testing

ArmorClaw includes comprehensive tests:

```bash
# Run all tests
make test-all

# Individual test suites
make test-hardening      # Container hardening validation
make test-secrets        # Secrets isolation tests
make test-exploits       # Exploit mitigation tests
make test-e2e            # End-to-end integration tests
```

---

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📜 License

MIT License - see [LICENSE.md](LICENSE.md) file for details.

---

## 🙋 Acknowledgments

- **Container base:** [debian:bookworm-slim](https://hub.docker.com/_/debian/)
- **Matrix homeserver:** [Conduit](https://conduit.rs/)
- **Inspired by:** [OpenClaw](https://github.com/realOpenClaw/openclaw) and security research on container isolation

---

## 📮 Support

- **Issues:** [GitHub Issues](https://github.com/armorclaw/armorclaw/issues)
- **Documentation:** [Full Documentation](docs/index.md)
- **License:** [MIT License](LICENSE.md)

---

**⚡ ArmorClaw — Run powerful AI safely.**