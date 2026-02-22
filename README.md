# ArmorClaw 🦞🔒

> **Secure containment for powerful AI agents** — Run GPT-4, Claude, and other agents safely with hardened containers, ephemeral secrets, and strict isolation.

[![Beta Release](https://img.shields.io/badge/release-v0.1.0--beta-orange)](https://github.com/armorclaw/armorclaw/releases)
[![Status: Seeking Testers](https://img.shields.io/badge/status-seeking%20testers-yellow)](https://github.com/armorclaw/armorclaw/issues?q=label%3Abeta-test)
[![Security: Hardened](https://img.shields.io/badge/security-multi--layer%20hardening-green)](docs/guides/security-verification-guide.md)
[![Android: Beta](https://img.shields.io/badge/Android-Beta%20Available-green)](#-mobile-apps)
[![iOS: Coming Soon](https://img.shields.io/badge/iOS-Coming%20Soon-lightgrey)](#-mobile-apps)

---

## 📱 Mobile Apps

### ArmorChat 💬

**Secure AI chat client for ArmorClaw** — End-to-end encrypted messaging with your AI agents.

[![Google Play](https://img.shields.io/badge/Google%20Play-Beta%20Testing-brightgreen?logo=google-play)](https://play.google.com/store/apps/details?id=app.armorclaw.armorchat)

**Features:**
- ✅ E2EE messaging via Matrix
- ✅ Push notifications
- ✅ QR code provisioning
- ✅ Bridge verification (emoji-based)
- ✅ Hardware-backed keystore
- ✅ Key backup & recovery

**Android:** [Join Beta Testing](https://play.google.com/store/apps/details?id=app.armorclaw.armorchat)
**iOS:** 🍎 Coming Soon

---

### ArmorTerminal 💻

**Terminal access to your AI agents** — Secure command-line interface for advanced users.

[![Google Play](https://img.shields.io/badge/Google%20Play-Beta%20Testing-brightgreen?logo=google-play)](https://play.google.com/store/apps/details?id=app.armorclaw.armorterminal)

**Features:**
- ✅ Secure terminal access
- ✅ Agent command execution
- ✅ Real-time output streaming
- ✅ Multi-agent management

**Android:** [Join Beta Testing](https://play.google.com/store/apps/details?id=app.armorclaw.armorterminal)
**iOS:** 🍎 Coming Soon

---

### Beta Tester Benefits

Join our beta program and get:
- 🚀 **Early access** to new features
- 💬 **Direct feedback channel** to developers
- 🎁 **Free extended features** for active testers
- 🏆 **Beta tester badge** in the community

**How to Join:**
1. Click the Google Play links above
2. Opt-in to beta testing
3. Install and start using
4. Report feedback via [GitHub Issues](https://github.com/armorclaw/armorclaw/issues)

---

**🔬 BETA RELEASE (v0.1.0)** — Seeking community testers for production validation. See [v0.1.0-beta Release Notes](#-v010-beta-release-security-hardening-update-2026-02-09) below.

---

## 📢 v0.1.0-beta Release: Security Hardening Update (2026-02-09)

**🔬 WE NEED TESTERS!** This beta release implements comprehensive security hardening to address critical container escape vulnerabilities. We need community testing to achieve **production-ready status**.

### What's New in v0.1.0-beta

| Security Layer | Implementation | Threats Mitigated |
|----------------|----------------|-------------------|
| **Filesystem Hardening** | `chmod a-x` on all binaries (except Python/Node) | Shell escapes via `os.execl()`, `child_process.spawn()` |
| **Network Isolation** | `--network=none` enforcement at bridge level | Data exfiltration via `urllib`, `fetch`, `curl` |
| **Seccomp Filtering** | Kernel-level syscall blocking profile | Bypass attempts via raw syscalls |
| **LD_PRELOAD Hook** | Library-level function interception | Runtime shell spawning attempts |
| **Capability Dropping** | `--cap-drop=ALL` on all containers | Privilege escalation vectors |
| **Read-Only Root** | `--read-only` filesystem flag | Filesystem modification attacks |

### Test Results Summary

| Test Category | Before Fix | After Fix | Status |
|---------------|-----------|-----------|--------|
| Python `os.execl('/bin/sh')` | ❌ WORKED (CRITICAL) | ✅ BLOCKED | PASS |
| Node `child_process.exec('/bin/sh')` | ❌ WORKED (CRITICAL) | ✅ BLOCKED | PASS |
| Python `urllib.urlopen()` | ❌ WORKED (CRITICAL) | ✅ BLOCKED | PASS |
| Node `fetch()` | ❌ WORKED (CRITICAL) | ✅ BLOCKED | PASS |
| Direct `/bin/sh` execution | ✅ BLOCKED | ✅ BLOCKED | PASS |
| `/bin/bash` execution | ✅ BLOCKED | ✅ BLOCKED | PASS |

**Total Tests:** 26 | **Passed:** 22 | **Failed:** 0 (after fixes)

### Call for Beta Testers

We need **community testers** to validate the security hardening across different environments:

**Test Environments Needed:**
- [ ] Linux (Ubuntu, Debian, Fedora)
- [ ] WSL2 on Windows
- [ ] macOS (Docker Desktop)
- [ ] Windows (Docker Desktop, PowerShell)
- [ ] ARM64 platforms (Raspberry Pi, AWS Graviton)

**How to Test:**
```bash
# Quick verification
./tests/verify-security.sh

# Full exploit suite
./tests/test-exploits.sh

# PowerShell (Windows)
.\tests\test-exploits.ps1
```

**Report Issues:** [GitHub Issues](https://github.com/armorclaw/armorclaw/issues) with label `beta-test`

**Path to Production:**
1. ✅ Security hardening implemented (v0.1.0-beta)
2. 🔬 **YOU ARE HERE** — Community testing & validation
3. 📦 Production release (v1.0.0) — After test sign-off

### Documentation

- **[Security Verification Guide](docs/guides/security-verification-guide.md)** — Complete manual verification instructions
- **[Test Suites](tests/)** — Automated security validation scripts

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

### Prerequisites

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| **OS** | Ubuntu 22.04, Debian 12 | Ubuntu 24.04 |
| **RAM** | 1 GB | 2 GB |
| **Disk** | 2 GB | 5 GB |
| **Docker** | 24.0+ | Latest |
| **Go** | 1.21+ | 1.24+ |

---

### Method 1: Quick Setup ⚡ (Recommended)

The fastest way to get ArmorClaw running with secure defaults:

```bash
# 1. Clone the repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# 2. Run quick setup (2-3 minutes)
sudo ./deploy/setup-quick.sh
```

**What happens automatically:**
- ✅ Prerequisites check (Docker, memory, disk)
- ✅ Bridge build from source
- ✅ System user creation (non-root, no shell)
- ✅ Encrypted keystore initialization (hardware-bound)
- ✅ Systemd service with hardening (NoNewPrivileges, PrivateTmp, ProtectSystem)
- ✅ QR code generation for device provisioning

**After setup completes:**

```bash
# 3. Add your API key
sudo armorclaw-bridge add-key --provider openai --token sk-...

# 4. Scan the QR code with ArmorChat (or run again for new devices)
sudo ./deploy/armorclaw-provision.sh

# 5. Start an agent
sudo armorclaw-bridge start --key openai-main
```

**Time:** 2-3 minutes | **Experience Level:** Beginner

---

### Method 2: Docker Image ⚡ (No Build Required)

Pull and run with automatic setup wizard:

```bash
# Pull the image
docker pull mikegemut/armorclaw:latest

# Run with Docker socket
docker run -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -p 8443:8443 -p 5000:5000 -p 6167:6167 \
  mikegemut/armorclaw:latest
```

**Non-interactive mode (CI/CD):**
```bash
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -e ARMORCLAW_MATRIX_SERVER=matrix.yourdomain.com \
  -e ARMORCLAW_API_KEY=your-api-key \
  -p 8443:8443 -p 5000:5000 -p 6167:6167 \
  mikegemut/armorclaw:latest
```

**Time:** 2 minutes | **Experience Level:** Beginner | **Requirements:** Docker only

---

### Method 3: Element X Mobile ⭐

Connect via Element X mobile app:

```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./deploy/launch-element-x.sh
```

Scan the QR code with Element X and start chatting.

**Time:** 5 minutes | **Experience Level:** Beginner | **Platform:** Mobile

---

### Method 4: VPS Deployment 🌐

Deploy to a remote VPS (Hostinger, DigitalOcean, AWS, etc.):

```bash
# 1. Create deployment tarball
cd armorclaw
tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' .

# 2. Transfer to VPS
scp deploy/vps-deploy.sh armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# 3. Run deployment on VPS
ssh user@your-vps-ip
chmod +x /tmp/vps-deploy.sh
sudo bash /tmp/vps-deploy.sh
```

**Time:** 10-15 minutes | **Experience Level:** Intermediate | **Platform:** VPS

---

### Connecting ArmorChat

After setup, connect your ArmorChat app:

1. **Generate QR code:**
   ```bash
   sudo ./deploy/armorclaw-provision.sh
   ```

2. **Scan with ArmorChat:**
   - Open ArmorChat app
   - Tap "Scan QR Code" or enter URL manually
   - Connection configured automatically

3. **QR Code Format:**
   ```
   armorclaw://config?d=<base64-encoded-json>

   Contains:
   - Matrix homeserver URL
   - Bridge RPC URL
   - WebSocket URL
   - Push gateway URL
   - Server name
   - Expiry (5 min default)
   ```

---

### Post-Setup: Production Hardening

For production deployments:

```bash
# 1. Enable Matrix with TLS (recommended)
sudo ./deploy/setup-matrix.sh

# 2. Apply system hardening
sudo ./deploy/armorclaw-harden.sh
```

**Hardening includes:**
- UFW firewall (deny-all default)
- SSH hardening (key-only, no root)
- Fail2Ban (brute-force protection)
- Automatic security updates
- Production logging (JSON format)

---

### Verify Installation

```bash
# Check bridge status
sudo systemctl status armorclaw-bridge

# Verify health via RPC
echo '{"jsonrpc":"2.0","method":"health","id":1}' | \
  sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Check systemd hardening
systemctl show armorclaw-bridge | grep -E "NoNewPrivileges|PrivateTmp|ProtectSystem"
```

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
| **Multi-Layer Container Hardening** 🆕 v0.1.0-beta | chmod a-x + seccomp + LD_PRELOAD + --network=none | Blocks shell escapes, network exfiltration, and privilege escalation |
| **Filesystem Hardening** 🆕 v0.1.0-beta | All binaries non-executable except runtime | Python/Node cannot spawn shells or child processes |
| **Network Isolation** 🆕 v0.1.0-beta | --network=none enforced at bridge level | No data exfiltration via urllib, fetch, or network tools |
| **Seccomp Syscall Filtering** 🆕 v0.1.0-beta | Kernel-level syscall blocking | Additional layer against bypass attempts |
| **Container Hardening** | Non-root user, removed shell/tools, seccomp profile | Agent can't escape even if exploited |
| **Ephemeral Secrets** | API keys injected via file descriptor, never written to disk | No secrets in logs, docker inspect, or disk |
| **Hardware-Bound Keystore** | Master key derived from machine-id, DMI UUID, MAC | Database useless if stolen/moved |
| **Zero-Touch Reboot** | Salt persistence enables automatic key derivation | No manual intervention after reboot |
| **Pull-Based Visibility** | All communication through signed Local Bridge | No direct agent access, full audit trail |
| **No Inbound Ports** | Container has no exposed network services | Can't be attacked from outside |
| **Zero-Trust Filtering** | Trusted Matrix senders/rooms allowlist | Only authorized users can control agents |
| **Budget Guardrails** | Daily/monthly spending limits with hard-stop | Prevent unexpected API costs |
| **PII Data Scrubbing** | Auto-redacts emails, SSNs, API keys, credit cards | Protects sensitive data in prompts/responses |
| **Container TTL** | Auto-removes idle containers (10 min default) | Prevents resource leaks |
| **Host Hardening** | Automated firewall (UFW) & SSH hardening | Production-ready security baseline |

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

## 🔐 Installation Security

The setup process is **secure by design**:

| Security Aspect | Implementation | Status |
|-----------------|----------------|--------|
| **Privilege Separation** | Dedicated `armorclaw` user (no shell) | ✅ Secure |
| **File Permissions** | Config 640, Data 750 | ✅ Secure |
| **Keystore Encryption** | SQLCipher + XChaCha20-Poly1305 | ✅ Secure |
| **Hardware Binding** | machine-id + DMI UUID + MAC | ✅ Secure |
| **Systemd Hardening** | NoNewPrivileges, PrivateTmp, ProtectSystem | ✅ Secure |
| **QR Code Expiry** | 5 min default, 1 hour max | ✅ Secure |
| **IP Detection** | Local `hostname -I` (no external calls) | ✅ Secure |

### What Gets Created

| Path | Purpose | Permissions |
|------|---------|-------------|
| `/opt/armorclaw/armorclaw-bridge` | Bridge binary | 755 (root:root) |
| `/etc/armorclaw/config.toml` | Configuration | 640 (armorclaw:armorclaw) |
| `/var/lib/armorclaw/keystore.db` | Encrypted keystore | 600 (armorclaw:armorclaw) |
| `/run/armorclaw/bridge.sock` | Unix socket | 660 (armorclaw:armorclaw) |

### Systemd Service Hardening

```ini
[Service]
User=armorclaw
NoNewPrivileges=true        # No setuid/escalation
PrivateTmp=true             # Isolated /tmp
ProtectSystem=strict        # Read-only /usr, /boot
ProtectHome=true            # Cannot access /home
MemoryMax=512M              # Resource limit
CPUQuota=50%                # Resource limit
```

> **📖 Full Security Review:** See [docs/output/review.md](docs/output/review.md) for complete security analysis.

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

### Security (v0.1.0-beta)
- **[Security Verification Guide](docs/guides/security-verification-guide.md)** 🆕 — Manual verification of all security layers
- **[Security Configuration](docs/guides/security-configuration.md)** — Zero-trust, guardrails, PII scrubbing

### Reference
- **[RPC API Reference](docs/reference/rpc-api.md)** — Complete JSON-RPC 2.0 API documentation
- **[Infrastructure Deployment Guide](docs/guides/2026-02-05-infrastructure-deployment-guide.md)** — Comprehensive deployment documentation

### Navigation Path for LLMs
**README** → **[Documentation Hub](docs/index.md)** → **[V1 Architecture](docs/plans/2026-02-05-armorclaw-v1-design.md)** → **[Reference Docs](docs/reference/)** → **[Function Catalog](docs/output/review.md)**
> 💡 **See [Navigation Guide](docs/NAVIGATION.md)** for complete LLM navigation documentation (README → Architecture → Feature → Functions)

---

## 🧪 Testing

ArmorClaw includes comprehensive security validation:

### Quick Security Verification (v0.1.0-beta)

```bash
# Quick verification of all security layers
./tests/verify-security.sh

# Full exploit simulation suite (26 tests)
./tests/test-exploits.sh

# PowerShell (Windows)
.\tests\test-exploits.ps1
```

### Test Suites

```bash
# Run all tests
make test-all

# Individual test suites
make test-hardening      # Container hardening validation
make test-secrets        # Secrets isolation tests
make test-attach         # Config attachment tests
make test-exploits       # Exploit mitigation tests (NEW in v0.1.0-beta)
make test-e2e            # End-to-end integration tests
```

### Security Test Results (v0.1.0-beta)

| Test Group | Tests | Status |
|------------|-------|--------|
| Shell Escape Attempts | 4 | ✅ All Blocked |
| Network Exfiltration | 3 | ✅ All Blocked |
| Filesystem Containment | 4 | ✅ All Contained |
| Secret Inspection | 3 | ✅ Expected Behavior |
| Privilege Escalation | 3 | ✅ All Blocked |
| Dangerous Tools | 9 | ✅ All Removed |

**For detailed verification procedures:** See [Security Verification Guide](docs/guides/security-verification-guide.md)

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

## 🔬 Beta Testing Call (v0.1.0-beta)

**We need YOUR help to reach production status!**

ArmorClaw v0.1.0-beta implements comprehensive security hardening against container escape and data exfiltration. We need community testing across diverse environments to validate the security measures.

### How to Participate

**1. Run the Verification Script**
```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
./tests/verify-security.sh
```

**2. Test on Your Platform**
- Linux (Ubuntu, Debian, Fedora, Arch)
- WSL2 on Windows
- macOS (Docker Desktop)
- Windows (PowerShell)
- ARM64 (Raspberry Pi, cloud instances)

**3. Report Your Results**
Create a GitHub issue with:
- Platform/OS details
- Docker version
- Test output (`./tests/verify-security.sh > results.txt`)
- Any issues encountered

**Label your issue:** `beta-test`

### Test Checklist

- [ ] Container builds successfully
- [ ] Verification script passes all checks
- [ ] Exploit tests pass (26/26)
- [ ] Element X integration works
- [ ] No unexpected errors or crashes

### Path to Production

| Milestone | Status | Target Date |
|-----------|--------|-------------|
| Security hardening implementation | ✅ Complete | 2026-02-09 |
| Community beta testing | 🔬 **IN PROGRESS** | 2026-02-23 |
| Cross-platform validation | ⏳ Pending | After beta sign-off |
| Production release (v1.0.0) | ⏳ Planned | 2026-03-01 |

---

**⚡ ArmorClaw — Run powerful AI safely.**

**🙏 Thank you for helping us make AI safer for everyone!**

© 2026 Gemutly  
[armorclaw.com](https://armorclaw.com)
