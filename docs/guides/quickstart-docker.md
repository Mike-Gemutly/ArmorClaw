# ArmorClaw Quick Start Guide

> **Docker Image Quick Start** - Pull, Run, Configure
> **Last Updated:** 2026-02-21
> **Image:** `mikegemut/armorclaw:latest`

---

## Overview

The ArmorClaw Docker image is a **self-contained deployment** that includes:
- ArmorClaw Bridge (Go binary)
- Setup Wizard (interactive configuration)
- Agent Runtime (hardened container)
- Configuration Templates

**First run** automatically launches the setup wizard to configure:
- Matrix homeserver connection
- API keys for AI providers
- Push notification settings
- Bridge security settings

---

## Requirements

### Host System
- **OS:** Linux (Ubuntu 22.04+, Debian 12+)
- **RAM:** 2GB minimum, 4GB recommended
- **Disk:** 10GB free space
- **Docker:** 24.0+ with compose plugin

### Network
- **Ports:** 80, 443, 8448, 5000, 6167
- **DNS:** A record for `matrix.yourdomain.com`

### Before You Start
- Domain with DNS control
- API key from AI provider (OpenAI, Anthropic, or GLM-5)
- SSH access to VPS

---

## Quick Start Commands

### 1. Pull the Image

```bash
docker pull mikegemut/armorclaw:latest
```

### 2. Run Setup Wizard

```bash
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -v armorclaw-logs:/var/log/armorclaw \
  -p 8443:8443 \
  -p 5000:5000 \
  -p 6167:6167 \
  mikegemut/armorclaw:latest
```

**The setup wizard will guide you through:**
1. Prerequisites check
2. Docker configuration
3. Matrix homeserver setup
4. API key configuration
5. Bridge security settings
6. Service startup

### 3. Re-run Setup (if needed)

```bash
# Remove setup flag to re-trigger wizard
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  mikegemut/armorclaw:latest \
  rm /etc/armorclaw/.setup_complete && /opt/armorclaw/setup-wizard.sh
```

---

## Environment Variables (Non-Interactive)

For automated deployments, use environment variables:

```bash
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -e ARMORCLAW_MATRIX_SERVER=matrix.yourdomain.com \
  -e ARMORCLAW_MATRIX_URL=http://conduit:6167 \
  -e ARMORCLAW_API_KEY=your-api-key \
  -e ARMORCLAW_API_BASE_URL=https://api.openai.com/v1 \
  -e ARMORCLAW_BRIDGE_PASSWORD=secure-password \
  -p 8443:8443 -p 5000:5000 -p 6167:6167 \
  mikegemut/armorclaw:latest
```

### Available Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ARMORCLAW_MATRIX_SERVER` | Matrix server domain | Yes |
| `ARMORCLAW_MATRIX_URL` | Matrix homeserver URL | Yes |
| `ARMORCLAW_API_KEY` | AI provider API key | Yes |
| `ARMORCLAW_API_BASE_URL` | AI provider base URL | No* |
| `ARMORCLAW_BRIDGE_PASSWORD` | Bridge Matrix password | Yes |
| `ARMORCLAW_LOG_LEVEL` | Log level (debug/info/warn) | No |
| `ARMORCLAW_PUSH_ENABLED` | Enable push notifications | No |

*Required if using non-OpenAI provider

---

## API Provider Configuration

### OpenAI (Default)
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

### Custom Provider
```bash
-e ARMORCLAW_API_KEY=your-key
-e ARMORCLAW_API_BASE_URL=https://your-provider.com/v1
```

---

## Full Stack Deployment

For a complete deployment with Matrix, push gateway, and bridge:

```bash
# Create network
docker network create armorclaw-net

# Start Matrix stack
docker compose -f docker-compose.matrix.yml up -d

# Run ArmorClaw (connects to Matrix)
docker run -d \
  --name armorclaw \
  --network armorclaw-net \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -e ARMORCLAW_MATRIX_URL=http://matrix:6167 \
  -p 8443:8443 -p 5000:5000 \
  mikegemut/armorclaw:latest
```

---

## Connecting ArmorChat

After setup completes:

1. **Open ArmorChat app** on your device
2. **Enter homeserver:** `https://matrix.yourdomain.com`
3. **Create account** or sign in
4. **Start DM** with `@bridge:matrix.yourdomain.com`
5. **Send `!status`** to verify connection

---

## Verification

### Check Bridge Health

```bash
# Via Docker exec
docker exec armorclaw armorclaw-bridge --health

# Via socket
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.health"}' | \
  socat - UNIX-CONNECT:/var/lib/armorclaw/bridge.sock
```

### Check Matrix

```bash
curl http://localhost:6167/_matrix/client/versions
```

### Check Push Gateway

```bash
curl http://localhost:5000/_matrix/push/v1/notify
```

---

## Troubleshooting

### "Docker socket not found"
```bash
# Ensure you mount the Docker socket
docker run -v /var/run/docker.sock:/var/run/docker.sock ...
```

### "Permission denied"
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Log out and back in
```

### "Setup wizard won't start"
```bash
# Remove setup flag
docker run --rm -v armorclaw-config:/etc/armorclaw alpine rm /etc/armorclaw/.setup_complete
```

### "API key not detected"
```bash
# Pass API key via environment
docker run -e ARMORCLAW_API_KEY=your-key ...
```

---

## systemd Service (Production)

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

Enable:
```bash
sudo systemctl daemon-reload
sudo systemctl enable armorclaw
sudo systemctl start armorclaw
```

---

## Support

- **Documentation:** `/opt/armorclaw/docs/` (in container)
- **GitHub Issues:** https://github.com/armorclaw/armorclaw/issues
- **Error Catalog:** `docs/guides/error-catalog.md`

---

**Quick Start Guide Version:** 1.0.0
**Last Updated:** 2026-02-21
