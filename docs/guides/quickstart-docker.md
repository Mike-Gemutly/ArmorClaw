# ArmorClaw Docker Quick Start

> **Description:** Pull, run, and configure ArmorClaw with one Docker command
> **Time to Complete:** ~2 minutes (Quick Start) · ~10 minutes (Enterprise)
> **Last Updated:** 2026-02-26
> **Version:** 0.3.1
> **Image:** `mikegemut/armorclaw:latest`

---

## Overview

The ArmorClaw Docker image is a **self-contained deployment** that bundles the bridge, Matrix homeserver, setup wizard, and agent runtime into a single container. First run launches an interactive wizard that configures everything.

Two deployment profiles are available:

| Profile | Prompts | Time | Best For |
|---------|---------|------|----------|
| **Quick Start** (default) | 2 | ~2 min | Developers, testing, personal use |
| **Enterprise / Compliance** | 6 steps | ~10 min | Regulated industries, healthcare, HIPAA |

---

## Prerequisites

### Host system
- [ ] **OS:** Linux (Ubuntu 22.04+, Debian 12+)
- [ ] **RAM:** 2GB minimum, 4GB recommended
- [ ] **Disk:** 10GB free space
- [ ] **Docker:** 24.0+ with compose plugin

### Network
- [ ] **Ports available:** 8443, 6167, 5000
- [ ] SSH access to host (for VPS deployments)

### Credentials
- [ ] API key from an AI provider (OpenAI, Anthropic, or GLM-5)

> **ℹ️ NOTE:** DNS and domain configuration are optional. The wizard auto-detects your IP for IP-based setups.

---

## Quick Start (Interactive)

### Step 1: Pull and run

```bash
docker run -it --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

You'll see the ArmorClaw banner, then the interactive **Huh? TUI wizard** launches:

```
╔══════════════════════════════════════════════════════╗
║        ArmorClaw Container Setup                     ║
║        Version 0.3.1                                 ║
╚══════════════════════════════════════════════════════╝

✓ Required tools verified (openssl, jq, socat, curl, docker)

  Choose your deployment profile
  This determines the setup flow and default security posture.

  > Quick Start — Fewest questions, running in ~2 minutes
    Enterprise / Compliance — PII/PHI scrubbing, HIPAA, audit logging
```

### Step 2: Complete the setup wizard

The wizard uses **arrow keys** to navigate, **Enter** to confirm, and **Esc** to cancel.

**Page 1: Profile Selection**

Select **Quick Start** (default) and press Enter.

**Page 2: AI Provider + API Key (Step 1 of 2)**

- Use arrow keys to select your AI provider (OpenAI, Anthropic, GLM-5, or Custom)
- Enter your API key (input is masked for security)
- The wizard validates key length before proceeding

**Page 3: Admin & Deployment (Step 2 of 2)**

- Enter an admin password (or press Enter to auto-generate one)
- Select **Deploy** to confirm

After confirmation, the wizard automatically:
1. Auto-detects your server IP
2. Configures the Matrix homeserver (Conduit)
3. Generates a self-signed SSL certificate
4. Starts the Matrix stack, registers users, and creates the bridge room
5. Injects the API key directly into the encrypted keystore (never written to disk as plaintext)

> **⚠️ WARNING:** Save your admin credentials when displayed — the password is not stored after setup.

> **💡 TIP:** For screen reader accessibility, set `ACCESSIBLE=1` as an environment variable. The wizard will use standard prompts instead of the TUI.

### Step 3: Connect a client

After setup completes, you'll see connection details and (optionally) a QR code for ArmorChat.

**Element X:**
1. Open Element X on your device
2. Set homeserver to the URL shown (e.g., `http://YOUR_IP:6167`)
3. Log in with the admin credentials displayed during setup
4. Open the **ArmorClaw Bridge** room (auto-created)
5. Send `!status` to verify the bridge is running

**ArmorChat:**
1. Scan the QR code shown in the terminal
2. Or manually enter the homeserver URL and credentials

> **💡 TIP:** For trusted SSL, ask the agent in ArmorChat: "Set up a cloudflare tunnel."

---

## Enterprise / Compliance Profile

Select **2** at the profile prompt for a guided 6-step enterprise setup:

| Step | Configuration | Details |
|------|--------------|--------|
| 1 | Matrix homeserver | Domain or IP, bridge user credentials |
| 2 | AI provider | Provider selection + API key |
| 3 | Compliance & security | PII scrubbing, HIPAA mode, quarantine, audit retention |
| 4 | Admin user | Username + password for Element X / ArmorChat login |
| 5 | Bridge configuration | Log level, socket path |
| 6 | Security tier | Auto-set to maximum |

Enterprise profile enables:
- **PII scrubbing** — SSN, credit cards, emails, phone numbers, IPs, API tokens
- **HIPAA mode** (optional) — Medical records, health plans, lab results, diagnoses, prescriptions
- **Quarantine mode** — Blocks critical PII/PHI findings for admin review
- **Audit logging** — Configurable retention (default 90 days), JSON format (SIEM-ready)
- **Buffered response mode** — Full-text scrubbing (no streaming)
- **Maximum security tier** — Seccomp, network isolation, complete audit chain

---

## Non-Interactive Deployment

For automated/CI deployments, pass configuration via environment variables:

```bash
docker run -d --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -e ARMORCLAW_SERVER_NAME=your-domain.com \
  -e ARMORCLAW_API_KEY=sk-your-key \
  -e ARMORCLAW_API_BASE_URL=https://api.openai.com/v1 \
  -e ARMORCLAW_BRIDGE_PASSWORD=secure-password \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

### Environment variable reference

| Variable | Description | Required |
|----------|-------------|----------|
| `ARMORCLAW_SERVER_NAME` | Server domain or IP | Yes* |
| `ARMORCLAW_API_KEY` | AI provider API key | Yes* |
| `ARMORCLAW_API_BASE_URL` | AI provider base URL (default: OpenAI) | No |
| `ARMORCLAW_BRIDGE_PASSWORD` | Bridge Matrix password (auto-generated if omitted) | No |
| `ARMORCLAW_PROFILE` | Deployment profile: `quick` or `enterprise` | No |
| `ARMORCLAW_HIPAA` | Enable HIPAA compliance: `true`/`false` | No |
| `ARMORCLAW_QUARANTINE` | Enable quarantine mode: `true`/`false` | No |
| `ARMORCLAW_AUDIT_RETENTION` | Audit log retention in days | No |
| `ARMORCLAW_LOG_LEVEL` | Log level: `debug`, `info`, `warn` | No |
| `ARMORCLAW_ADMIN_USER` | Admin username (default: `admin`) | No |
| `ARMORCLAW_ADMIN_PASSWORD` | Admin password (auto-generated if omitted) | No |

*At least one of `ARMORCLAW_SERVER_NAME` or `ARMORCLAW_API_KEY` triggers non-interactive mode. If `ARMORCLAW_SERVER_NAME` is omitted, the IP is auto-detected.

### Enterprise non-interactive example

```bash
docker run -d --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -e ARMORCLAW_PROFILE=enterprise \
  -e ARMORCLAW_SERVER_NAME=your-domain.com \
  -e ARMORCLAW_API_KEY=sk-your-key \
  -e ARMORCLAW_HIPAA=true \
  -e ARMORCLAW_QUARANTINE=true \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

---

## API Provider Configuration

### OpenAI (default)
```bash
-e ARMORCLAW_API_KEY=sk-xxx
# ARMORCLAW_API_BASE_URL defaults to https://api.openai.com/v1
```

### Anthropic
```bash
-e ARMORCLAW_API_KEY=sk-ant-xxx
-e ARMORCLAW_API_BASE_URL=https://api.anthropic.com/v1
```

### GLM-5 (Zhipu AI)
```bash
-e ARMORCLAW_API_KEY=your-glm-key
-e ARMORCLAW_API_BASE_URL=https://api.z.ai/api/coding/paas/v4
```

### Custom (OpenAI-compatible)
```bash
-e ARMORCLAW_API_KEY=your-key
-e ARMORCLAW_API_BASE_URL=https://your-provider.com/v1
```

---

## Verification

### Check bridge health

```bash
# Via Docker exec
docker exec armorclaw armorclaw-bridge --health

# Via JSON-RPC over Unix socket
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.health"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Check Matrix homeserver

```bash
curl http://localhost:6167/_matrix/client/versions
```

### Check push gateway

```bash
curl http://localhost:5000/_matrix/push/v1/notify
```

### Check container status

```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep armorclaw
```

---

## Re-running Setup

To re-run the setup wizard (e.g., to change API provider or profile):

```bash
# Stop and remove the container
docker stop armorclaw && docker rm armorclaw

# Clear the setup flag
docker run --rm -v armorclaw-config:/etc/armorclaw alpine rm -f /etc/armorclaw/.setup_complete

# Run again
docker run -it --name armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -p 8443:8443 -p 6167:6167 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

> **ℹ️ NOTE:** Your keystore data persists in the `armorclaw-data` volume. Only the setup flag is cleared.

---

## Troubleshooting

### "Docker socket not found"

**Cause:** Container started without Docker socket mount.

**Fix:**
```bash
# Ensure -v flag includes the Docker socket
docker run -v /var/run/docker.sock:/var/run/docker.sock ...
```

### "Permission denied"

**Cause:** Current user is not in the `docker` group.

**Fix:**
```bash
sudo usermod -aG docker $USER
# Log out and back in for group change to take effect
```

### "Setup wizard won't start" (on restart)

**Cause:** Setup flag exists from a previous run.

**Fix:**
```bash
docker run --rm -v armorclaw-config:/etc/armorclaw alpine rm /etc/armorclaw/.setup_complete
```

### "API key not detected"

**Cause:** API key was not entered during wizard or environment variable is missing.

**Fix:**
```bash
# Pass API key via environment
docker run -e ARMORCLAW_API_KEY=your-key ...

# Or inject via RPC after bridge is running
echo '{"jsonrpc":"2.0","id":1,"method":"store_key","params":{"id":"openai-default","provider":"openai","token":"sk-xxx","display_name":"Default API"}}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### "Conduit did not become ready"

**Cause:** Matrix homeserver is still starting (common on first run with image pulls).

**Fix:** Wait up to 2 minutes. The wizard polls automatically. If it times out:
```bash
# Check Conduit logs
docker logs armorclaw-conduit

# Verify port is bound
docker exec armorclaw curl -sf http://localhost:6167/_matrix/client/versions
```

### "Bridge exited with code 1"

**Cause:** Configuration missing required fields or bridge socket creation failed.

**Fix:**
```bash
# Check config
docker exec armorclaw cat /etc/armorclaw/config.toml

# Check bridge logs
docker logs armorclaw
```

---

## Production: systemd Service

Create `/etc/systemd/system/armorclaw.service`:

```ini
[Unit]
Description=ArmorClaw Bridge
After=docker.service
Requires=docker.service

[Service]
Type=simple
ExecStart=/usr/bin/docker start -a armorclaw
ExecStop=/usr/bin/docker stop armorclaw
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable armorclaw
sudo systemctl start armorclaw
```

---

## Port Reference

| Port | Service | Protocol |
|------|---------|----------|
| `8443` | ArmorClaw Bridge (HTTPS/RPC) | TCP |
| `6167` | Matrix Conduit (homeserver) | TCP |
| `5000` | Sygnal (push gateway) | TCP |

---

## Next Steps

1. **[Configuration Guide](configuration.md)** — Customize bridge settings
2. **[Element X Quick Start](element-x-quickstart.md)** — Detailed Element X setup
3. **[First Deployment Checklist](first-deployment-checklist.md)** — Full manual deployment
4. **[Error Catalog](error-catalog.md)** — Complete error reference
5. **[Getting Started](getting-started.md)** — Architecture overview and build-from-source path

---

## CLI Reference

```bash
# Show help
docker run --rm mikegemut/armorclaw:latest --help

# Show version
docker run --rm mikegemut/armorclaw:latest --version
```

---

**Guide Version:** 3.0.0
**Last Updated:** 2026-02-26
