# Getting Started with ArmorClaw

> **Time to First Message:** ~5 minutes
> **Prerequisites:** Docker, Terminal
> **Last Updated:** 2026-02-15

---

## What is ArmorClaw?

ArmorClaw is a **secure bridge** that lets AI agents communicate through encrypted Matrix channels while keeping your API keys and credentials isolated in memory-protected containers.

### Why ArmorClaw?

| Problem | ArmorClaw Solution |
|---------|-------------------|
| API keys exposed in config files | Memory-only secret injection |
| AI agents have full system access | Scoped Docker containers |
| No audit trail for AI actions | Complete operation logging |
| Credentials persist on disk | Ephemeral, encrypted storage |

### Key Features

- **Zero-Trust Security**: API keys never touch disk, never in `docker inspect`
- **E2EE Messaging**: All communication via Matrix with end-to-end encryption
- **Container Isolation**: Each agent runs in a hardened, minimal container
- **Audit Logging**: Every operation is logged with trace IDs
- **Multi-Platform**: Connect Slack, Discord, Teams, WhatsApp through Matrix

---

## Quick Start (5 Minutes)

### Step 1: Prerequisites Check

```bash
# Verify Docker is running
docker ps

# Expected: List of containers (empty is fine)
# If error: Start Docker Desktop or `sudo systemctl start docker`
```

### Step 2: Build the Bridge

```bash
# Clone the repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Build the bridge binary
cd bridge && go build -o build/armorclaw-bridge ./cmd/bridge

# Build the container image
cd .. && docker build -t armorclaw/agent:v1 .
```

### Step 3: Initialize Configuration

```bash
# Create default config
./bridge/build/armorclaw-bridge init

# This creates:
# - ~/.armorclaw/config.toml (configuration)
# - ~/.armorclaw/keystore.db (encrypted credential storage)
```

### Step 4: Add Your API Key

```bash
# Add an OpenAI API key
./bridge/build/armorclaw-bridge add-key \
  --provider openai \
  --token sk-proj-your-key-here \
  --id my-openai-key

# Verify it was stored
./bridge/build/armorclaw-bridge list-keys
```

### Step 5: Start the Bridge

```bash
# Start with default configuration
sudo ./bridge/build/armorclaw-bridge

# In another terminal, test the RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Step 6: Launch an Agent Container

```bash
# Start a container with your API key
echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"my-openai-key"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Response includes container_id and socket endpoint
```

**You're done!** Your first ArmorClaw agent is running with securely injected credentials.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARMORCLAW ARCHITECTURE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐         ┌─────────────────┐                            │
│  │   Your Machine   │         │  Matrix Server  │                            │
│  │                 │         │   (Conduit)     │                            │
│  │  ┌───────────┐  │         │                 │                            │
│  │  │  Bridge   │◄─┼─────────┤  E2EE Messages  │                            │
│  │  │  (Go)     │  │  Matrix │                 │                            │
│  │  └─────┬─────┘  │  Protocol│ ┌─────────────┐│                            │
│  │        │        │         │ │ Element X   ││                            │
│  │  ┌─────▼─────┐  │         │ │ (Mobile/    ││                            │
│  │  │  Docker   │  │         │ │  Desktop)   ││                            │
│  │  │  Socket   │  │         │ └─────────────┘│                            │
│  │  └─────┬─────┘  │         └─────────────────┘                            │
│  │        │        │                                                        │
│  │  ┌─────▼─────────────────────────────────────┐                          │
│  │  │           Hardened Container               │                          │
│  │  │  ┌─────────────┐  ┌─────────────────────┐ │                          │
│  │  │  │   Agent     │  │  Secrets (Memory)   │ │                          │
│  │  │  │   (Python)  │◄─┤  /run/secrets/*.json│ │                          │
│  │  │  └─────────────┘  └─────────────────────┘ │                          │
│  │  │                                             │                          │
│  │  │  • UID 10001 (non-root)                    │                          │
│  │  │  • No shell, no network tools              │                          │
│  │  │  • Read-only root filesystem               │                          │
│  │  │  • Seccomp security profile                │                          │
│  │  └─────────────────────────────────────────────┘                          │
│  │                                                                            │
│  └─────────────────────────────────────────────────────────────────────────────┘
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Components

| Component | Purpose | Technology |
|-----------|---------|------------|
| **Bridge** | Secure host interface | Go 1.24+, JSON-RPC 2.0 |
| **Keystore** | Encrypted credential storage | SQLCipher + XChaCha20 |
| **Container** | Isolated agent runtime | Docker, debian:bookworm-slim |
| **Matrix Adapter** | E2EE messaging | Matrix Protocol |

---

## Security Model

### The Three Pillars

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARMORCLAW SECURITY MODEL                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   PILLAR 1: SECRET ISOLATION                                     │
│   ═══════════════════════════                                     │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐                  │
│   │ API Key  │───▶│ Memory   │───▶│ Container │                  │
│   │          │    │ Only     │    │ /run/...  │                  │
│   └──────────┘    └──────────┘    └──────────┘                  │
│                         │                                        │
│                    Never on disk                                  │
│                    Never in docker inspect                        │
│                    Cleaned up after 10s                           │
│                                                                  │
│   PILLAR 2: CONTAINER HARDENING                                  │
│   ═══════════════════════════════                                 │
│   • No root user (UID 10001)                                     │
│   • No shell access                                               │
│   • No network tools (curl, wget removed)                        │
│   • Read-only root filesystem                                    │
│   • Seccomp profile limits syscalls                              │
│                                                                  │
│   PILLAR 3: ZERO-TRUST ACCESS                                    │
│   ═════════════════════════════                                   │
│   • Unix socket only (no network ports)                          │
│   • Filesystem permissions control access                        │
│   • All Docker operations scoped (create, exec, remove)          │
│   • Full audit logging with trace IDs                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Secret Injection Flow

```
1. User calls "start" RPC with key_id
          │
          ▼
2. Bridge retrieves encrypted key from keystore
          │
          ▼
3. Bridge decrypts key in memory (XChaCha20-Poly1305)
          │
          ▼
4. Bridge writes to /run/armorclaw/secrets/<container>.json
          │
          ▼
5. Bridge mounts file to container at /run/secrets:ro
          │
          ▼
6. Container entrypoint reads secrets, sets env vars
          │
          ▼
7. Agent starts with credentials in memory
          │
          ▼
8. Bridge deletes secrets file (10s timeout)
```

---

## Common Use Cases

### 1. Local AI Assistant

Run a secure AI assistant that can communicate via Matrix:

```bash
# Add your API key
./build/armorclaw-bridge add-key --provider openai --token sk-xxx --id assistant

# Start with Matrix enabled
./build/armorclaw-bridge -matrix-enabled \
  -matrix-homeserver https://matrix.example.com \
  -matrix-username my-bot \
  -matrix-password secret

# Connect from Element X and chat with your assistant
```

### 2. Multi-Platform Bot

Connect Slack and Discord through Matrix:

```bash
# Use platform.connect RPC to add platforms
echo '{"jsonrpc":"2.0","id":1,"method":"platform.connect","params":{
  "platform": "slack",
  "workspace_id": "T0XXXXXXXX",
  "access_token": "xoxb-xxx",
  "matrix_room": "!bot:matrix.example.com",
  "channels": ["C0XXXXXXXX"]
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

See [Platform Onboarding Guide](platform-onboarding.md) for detailed setup.

### 3. Development Testing

Quick testing without full Matrix setup:

```bash
# Start bridge without Matrix
./build/armorclaw-bridge

# Start a container
echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"test-key"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Check status
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## RPC Methods Quick Reference

| Method | Purpose | Example |
|--------|---------|---------|
| `status` | Check bridge status | `{"method":"status"}` |
| `health` | Health check | `{"method":"health"}` |
| `start` | Start container | `{"method":"start","params":{"key_id":"xxx"}}` |
| `stop` | Stop container | `{"method":"stop","params":{"container_id":"xxx"}}` |
| `list_keys` | List stored keys | `{"method":"list_keys"}` |
| `store_key` | Store new key | `{"method":"store_key","params":{...}}` |
| `get_errors` | Query errors | `{"method":"get_errors"}` |
| `matrix.send` | Send Matrix message | `{"method":"matrix.send","params":{...}}` |

See [RPC API Reference](../reference/rpc-api.md) for complete documentation.

---

## Error Handling

ArmorClaw uses structured error codes for easy debugging:

| Prefix | Category | Example |
|--------|----------|---------|
| CTX-XXX | Container | CTX-001: container start failed |
| MAT-XXX | Matrix | MAT-002: authentication failed |
| RPC-XXX | RPC/API | RPC-010: socket connection failed |
| SYS-XXX | System | SYS-001: keystore decryption failed |

Query errors via RPC:

```bash
# Get all unresolved errors
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Get specific error category
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{"category":"container"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

See [Error Catalog](error-catalog.md) for complete error reference.

---

## Next Steps

### Recommended Reading

1. **[Configuration Guide](configuration.md)** - Customize your setup
2. **[Platform Onboarding](platform-onboarding.md)** - Connect Slack/Discord/Teams
3. **[Security Configuration](security-configuration.md)** - Advanced security options
4. **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions

### Deployment Options

| Platform | Guide | Best For |
|----------|-------|----------|
| Hostinger | [Guide](hostinger-deployment.md) | Budget VPS |
| DigitalOcean | [Guide](digitalocean-deployment.md) | Simple cloud |
| AWS Fargate | [Guide](aws-fargate-deployment.md) | Enterprise |
| Fly.io | [Guide](flyio-deployment.md) | Edge deployment |

### Getting Help

- **Documentation:** [docs/index.md](../index.md)
- **Issues:** https://github.com/armorclaw/armorclaw/issues
- **Security Issues:** security@armorclaw.com

---

## Quick Reference Card

```bash
# === LIFECYCLE ===
./build/armorclaw-bridge init              # Initialize config
./build/armorclaw-bridge add-key ...       # Add API key
./build/armorclaw-bridge list-keys         # List keys
sudo ./build/armorclaw-bridge              # Start bridge

# === RPC (via socat) ===
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"my-key"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

echo '{"jsonrpc":"2.0","id":1,"method":"stop","params":{"container_id":"xxx"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# === TROUBLESHOOTING ===
./build/armorclaw-bridge -log-level=debug  # Debug mode
docker logs <container_id>                  # Container logs
./build/armorclaw-bridge validate          # Validate config
```

---

**Getting Started Guide Last Updated:** 2026-02-15
