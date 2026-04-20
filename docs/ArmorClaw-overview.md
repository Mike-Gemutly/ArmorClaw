# ArmorClaw Docker Hub Overview

> **Image:** `mikegemut/armorclaw:latest`
> **Version:** 4.8.0
> **Last Updated:** 2026-04-19

---

## Quick Start (2 minutes)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

**What the wizard asks:**
1. AI Provider (OpenAI, Anthropic, GLM-5, or Custom)
2. API Key (encrypted, never logged)
3. Admin Password (auto-generated if empty)

**What you'll see:**

```
╔══════════════════════════════════════════════════════╗
║        ArmorClaw is Ready!                           ║
╚══════════════════════════════════════════════════════╝

Bridge:  http://192.168.1.50:8443
Matrix:  http://192.168.1.50:6167

Admin:   admin / <generated-password>

[QR CODE - Scan with ArmorChat]
```

**Next:**
1. Install ArmorChat from Google Play
2. Scan the QR code
3. Done - your digital secretary is online

---

## What is ArmorClaw?

ArmorClaw runs AI agents 24/7 on your VPS. They browse websites, fill forms, and manage tasks—while you approve sensitive actions via your phone.

### Why ArmorClaw?

**Traditional AI agents see your passwords.** When you give an AI your credit card, it can log or leak it.

**ArmorClaw agents never see your secrets.** BlindFill™ injects secrets directly into the browser—the agent requests "credit card" but never sees the number.

### Key Features

- **VPS-Based Agents** - Run desktop-class tasks 24/7
- **Mobile Control** - Monitor and approve via ArmorChat
- **End-to-End Encryption** - All messages secured via Matrix
- **BlindFill™ Security** - Secrets decrypted only in memory
- **No-Code Agent Studio** - Define agents via chat
- **Jetski Browser Sidecar** - CDP proxy with Tethered Mode security (PII scrubbing, SQLCipher sessions)
- **Secure Document Pipeline** - Split-Storage RAG, YARA CDR, TTL Proxy Guard
- **Sentinel Mode** - Let's Encrypt TLS, automatic VPS deployment
- **Cloudflare Tunnel/Proxy** - NAT traversal, CDN, DDoS protection

---

## Your First Task

Once connected via ArmorChat:

```
!agent create name="Researcher" skills="web_browsing"
```

Then ask:

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
       │ End-to-end encrypted
       ▼
┌─────────────┐
│   Bridge    │  Your VPS
│ Orchestrator│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Agent    │  Isolated container
│  OpenClaw   │
└──────┬──────┘
       │ CDP (port 9222)
       ▼
┌─────────────┐
│   Jetski    │  Browser sidecar
│ CDP Proxy   │  (Tethered Mode)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Browser    │  Automated tasks
│ Playwright  │
└─────────────┘
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

## Key Components

| Component | Language | Purpose |
|-----------|----------|---------|
| **Go Bridge** | Go | Control plane, JSON-RPC server (51+ methods), Matrix adapter |
| **Rust Sidecar** | Rust | PDF/DOCX extraction, S3 storage, gRPC |
| **Python Sidecar** | Python | XLSX/PPTX/MSG/XLS/DOC/PPT extraction via MarkItDown |
| **Jetski** | Go | CDP proxy with PII scrubbing, SQLCipher sessions, Matrix HITL |
| **browser-service** | TypeScript | Playwright HTTP automation (canonical browser path) |
| **ArmorChat** | Kotlin | Android mobile client, E2EE, biometric keystore |
| **Admin Panel** | React/TypeScript | Browser-based management dashboard |

---

## Deployment Modes

| Mode | Command | Description | Use Case |
|------|---------|-------------|----------|
| **Full Stack** | `bash` (default) | Bridge + Matrix + Push | ArmorChat integration |
| **Bridge-only** | `bash -s -- --bridge-only` | Bridge only, no Matrix | Testing |
| **Sentinel** | Enter domain at prompt | Let's Encrypt TLS, public VPS | Production |
| **Cloudflare Tunnel** | `CF_API_TOKEN=xxx bash` | Outbound tunnel, no public IP | NAT/firewall |
| **Cloudflare Proxy** | `CF_MODE=proxy bash` | CDN, DDoS protection | Existing Cloudflare |
| **Bootstrap** | `bash -s -- --bootstrap` | Generate compose file | GitOps, CI/CD |
| **Show Ports** | `bash -s -- --ports` | Display detected ports | Debugging |

---

## Mode Details

### 1. Full Stack Mode (Default)

The complete ArmorClaw experience with Matrix messaging for ArmorChat.

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

**What it starts:**
- ArmorClaw Bridge (port 8443 or auto-detected)
- Matrix Conduit homeserver (port 6167 or auto-detected)
- Sygnal push gateway (port 5000 or auto-detected)

**Best for:** Production use with ArmorChat mobile app.

---

### 2. Bridge-Only Mode

Runs just the bridge without Matrix. No mobile app integration.

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --bridge-only
```

**What it starts:**
- ArmorClaw Bridge only (port 8443)

**Best for:**
- Local testing and development
- CI/CD pipelines
- Headless automation without mobile control

---

### 3. Bootstrap Mode

Generates a `docker-compose.yml` file without starting any containers.

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --bootstrap
```

**Output:** `/opt/armorclaw/docker-compose.yml`

```yaml
services:
  armorclaw:
    image: mikegemut/armorclaw:latest
    container_name: armorclaw
    restart: unless-stopped
    ports:
      - "8443:8443"
      - "6167:6167"
      - "5000:5000"
    # ... auto-detected ports if defaults are busy
```

**Best for:**
- Reviewing configuration before deployment
- Customizing ports, volumes, environment
- Version controlling your infrastructure
- GitOps workflows (ArgoCD, Flux)
- Multi-environment setups (staging, production)

**Deploy after customization:**
```bash
docker compose -f /opt/armorclaw/docker-compose.yml up -d
```

---

### 4. Show Ports Mode

Displays auto-detected available ports without installing.

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --ports
```

**Output:**
```
Ports detected:
  Bridge:  8443
  Matrix:  6167
  Push:    5000
```

**Best for:** Debugging port conflicts before deployment.

---

## Non-Interactive Deployment

Deploy without any prompts using environment variables.

```bash
# Minimal - auto-detects server IP
export OPENROUTER_API_KEY=sk-your-api-key
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash

# Or specify IP/domain explicitly
export OPENROUTER_API_KEY=sk-your-api-key
export ARMORCLAW_SERVER_NAME=192.168.1.50  # or your-domain.com
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENROUTER_API_KEY` | Yes* | - | OpenRouter API key (recommended, supports many providers) |
| `OPEN_AI_KEY` | No | - | OpenAI API key |
| `ZAI_API_KEY` | No | - | xAI (Grok) API key |
| `ARMORCLAW_API_BASE_URL` | No | OpenAI URL | Custom API endpoint (for Anthropic, GLM-5, etc.) |
| `ARMORCLAW_PROFILE` | No | `quick` | Deployment profile: `quick` or `enterprise` |
| `ARMORCLAW_SERVER_NAME` | No | auto-detected IP | Server IP or hostname (omit to auto-detect) |
| `ARMORCLAW_ADMIN_PASSWORD` | No | auto-generated | Admin password for Matrix |
| `CF_API_TOKEN` | No | - | Cloudflare API token (for Tunnel/Proxy modes) |
| `CF_TUNNEL_DOMAIN` | No | - | Domain for Cloudflare Tunnel |
| `CF_MODE` | No | - | `tunnel` or `proxy` |

*At least one API key is required for non-interactive mode. Without it, the interactive wizard runs instead.

---

## Advanced / Manual Setup

> **Most users should use the one-liner at the top.** This section is for advanced customization.

### Bootstrap Mode (Generate docker-compose.yml)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash -s -- --bootstrap
```

Creates `/opt/armorclaw/docker-compose.yml` for GitOps, CI/CD, or Terraform workflows.

### Manual Docker Run

For full control over configuration:

```bash
docker run -it --name armorclaw \
  --restart unless-stopped \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-keystore:/var/lib/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

### Required Flags

| Flag | Purpose |
|------|---------|
| `--user root` | Docker socket access |
| `-v /var/run/docker.sock` | Spawn agent containers |
| `-v armorclaw-config` | Persistent configuration |
| `-v armorclaw-keystore` | Encrypted credential storage |

---

## Auto Port Detection

ArmorClaw automatically detects available ports if defaults are in use.

| Service | Default | Fallback Range |
|---------|---------|----------------|
| Bridge RPC | 8443 | 30000-40000 |
| Matrix | 6167 | 30000-40000 |
| Push Gateway | 5000 | 30000-40000 |

No manual port configuration needed.

---

## Post-Deployment

### Connect with ArmorChat (Android)

1. Install ArmorChat from Google Play (link in README)
2. Open the app and scan the QR code displayed in container logs
3. Set up biometric authentication for the keystore

### Connect with Element X (Any Platform)

1. Open Element X
2. Enter homeserver: `http://YOUR-SERVER:6167`
3. Login with admin credentials from setup

### Verify Deployment

```bash
# Check container status
docker ps

# View logs
docker logs -f armorclaw

# Test Bridge RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Test Matrix API
curl http://localhost:6167/_matrix/client/versions
```

---

## Deployment Profiles

ArmorClaw supports three deployment profiles for different use cases:

### Quick Setup (Default)

For developers and testing.

```
Runtime: Docker
Security: Standard hardening
Setup Time: ~2 minutes
```

**Includes:**
- Docker runtime with seccomp profiles
- Standard container isolation
- Matrix + Bridge + Push gateway

### Advanced Setup

For production teams.

```
Runtime: Docker (hardened)
Security: Enhanced hardening
Setup Time: ~5 minutes
```

**Includes:**
- Docker runtime with custom security profiles
- Resource limits (CPU, memory, PIDs)
- Audit logging
- Custom network policies

### Enterprise Setup

For regulated environments (HIPAA, SOC2, ISO27001).

```
Runtime: Docker / containerd / Firecracker
Security: Maximum isolation
Setup Time: ~10 minutes
```

**Runtime Options:**

| Runtime | Status | Use Case |
|---------|--------|----------|
| Docker hardened | ✅ Available | Default enterprise option |
| containerd | 🔜 v5.0 | Kubernetes-native environments |
| Firecracker | 🔜 On request | Maximum isolation (microVM) |

**Enterprise Features:**
- Audit logging with 90-day retention
- Compliance configurations
- Enhanced security profiles
- Memory-only secret injection
- Network isolation

---

## Enterprise Runtime Options

### Docker Hardened (Default)

Standard Docker runtime with maximum security hardening:

```yaml
security:
  - seccomp: armorclaw-enterprise
  - apparmor: armorclaw-enterprise
  - read_only_root_fs: true
  - no_new_privileges: true
  - cap_drop: ["ALL"]
resources:
  - memory: 256MB
  - cpu: 0.5
  - pids: 50
network:
  - disabled: true  # Must proxy through bridge
```

### containerd (v5.0)

For Kubernetes-native and reduced attack surface:

- No Docker daemon required
- Smaller attack surface
- Native container runtime
- Better for production infrastructure

### Firecracker (On Request)

For maximum isolation with microVMs:

- VM-grade isolation between agents
- Prevents container escape attacks
- Strong multi-tenant separation
- Used by AWS Lambda, Fly.io

**Requirements:**
- Linux host with KVM (`/dev/kvm`)
- Additional ~200ms startup latency
- Higher memory overhead

---

## Common Issues

### Port Already in Use

ArmorClaw auto-detects available ports. If you see port conflicts:

```bash
# Check what's using the port
ss -ltn | grep 8443

# Let ArmorClaw pick available ports
curl -fsSL ... | bash
```

### Docker Socket Permission Denied

```bash
# Run with --user root
docker run --user root ...
```

### Terminal Not Supported (Wizard Crashes)

Use environment variables instead:

```bash
export OPENROUTER_API_KEY=sk-your-key
curl -fsSL ... | bash
```

---

## Upgrading

```bash
# Stop and remove old container
docker rm -f armorclaw

# Pull latest image
docker pull mikegemut/armorclaw:latest

# Re-run install (preserves data volumes)
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

Data is preserved in Docker volumes:
- `armorclaw-config` - Configuration files
- `armorclaw-keystore` - Encrypted credentials

---

## Uninstalling

```bash
# Stop and remove container
docker rm -f armorclaw

# Remove volumes (WARNING: deletes all data)
docker volume rm armorclaw-config armorclaw-keystore

# Remove generated compose file
rm /opt/armorclaw/docker-compose.yml
```

---

## Support

- **GitHub:** https://github.com/Gemutly/ArmorClaw
- **Documentation:** See `docs/` directory
- **Issues:** https://github.com/Gemutly/ArmorClaw/issues

---

## License

MIT License - See [LICENSE](LICENSE) for details.
