# ArmorClaw Docker Hub Overview

> **Image:** `mikegemut/armorclaw:latest`
> **Version:** 4.2.0
> **Last Updated:** 2026-03-05

---

## What is ArmorClaw?

ArmorClaw is a **Zero-Trust VPS orchestration platform** that runs AI agents (OpenClaw) on your server. These agents act as "Digital Secretaries"—browsing websites, filling forms, managing tasks—while you stay mobile via the ArmorChat Android app.

### Key Features

- **VPS-Based Agents** - Run desktop-class tasks 24/7 on your server
- **Mobile Control** - Monitor and approve actions via ArmorChat (Android)
- **End-to-End Encryption** - All messages secured via Matrix protocol
- **BlindFill Security** - Sensitive data decrypted only in memory
- **No-Code Agent Studio** - Define agents via chat, no programming required

---

## Quick Start

### One-Line Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

The wizard asks **4 questions** and deploys in ~2 minutes:

1. **AI Provider** - OpenAI, Anthropic, Google, OpenRouter, xAI, or Skip
2. **API Key** - Your provider's API key (encrypted, never logged)
3. **Admin Username** - Default: admin
4. **Admin Password** - Auto-generated secure password if empty

---

## Deployment Modes

| Mode | Command | Description | Use Case |
|------|---------|-------------|----------|
| **Full Stack** | `bash` (default) | Bridge + Matrix + Push | ArmorChat mobile integration |
| **Bridge-only** | `bash -s -- --bridge-only` | Bridge only, no Matrix | Testing, development |
| **Bootstrap** | `bash -s -- --bootstrap` | Generate compose file | Production planning, GitOps |
| **Show Ports** | `bash -s -- --ports` | Display auto-detected ports | Port conflict debugging |

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
export ARMORCLAW_PROVIDER=openai
export ARMORCLAW_API_KEY=sk-your-api-key
export ARMORCLAW_ADMIN_USER=admin
export ARMORCLAW_ADMIN_PASSWORD=$(openssl rand -base64 24)
export ARMORCLAW_SERVER_NAME=your-domain.com

curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ARMORCLAW_PROVIDER` | No | - | AI provider: `openai`, `anthropic`, `google`, `openrouter`, `xai` |
| `ARMORCLAW_API_KEY` | No* | - | API key for AI provider |
| `ARMORCLAW_ADMIN_USER` | No | `admin` | Admin username for Matrix |
| `ARMORCLAW_ADMIN_PASSWORD` | No | auto-generated | Admin password |
| `ARMORCLAW_SERVER_NAME` | No | auto-detected IP | Server hostname or IP |

*API key can be skipped and added later via ArmorChat or RPC.

---

## Manual Docker Run

For users who prefer direct Docker commands.

### Full Stack

```bash
docker run -it --name armorclaw \
  --restart unless-stopped \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-keystore:/var/lib/armorclaw \
  -p 8443:8443 \
  -p 6167:6167 \
  -p 5000:5000 \
  mikegemut/armorclaw:latest
```

### Bridge-Only

```bash
docker run -d --name armorclaw \
  --restart unless-stopped \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-keystore:/var/lib/armorclaw \
  -p 8443:8443 \
  -e ARMORCLAW_SKIP_MATRIX=true \
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
export ARMORCLAW_API_KEY=sk-your-key
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

- **GitHub:** https://github.com/armorclaw/armorclaw
- **Documentation:** See `docs/` directory
- **Issues:** https://github.com/armorclaw/armorclaw/issues

---

## License

MIT License - See [LICENSE](LICENSE) for details.
