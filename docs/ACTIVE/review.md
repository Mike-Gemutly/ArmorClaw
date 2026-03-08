# ArmorClaw Quickstart Review

> **Purpose:** Complete guide to the Docker quickstart process and post-deployment steps
> **Version:** 0.5.0
> **Last Updated:** 2026-03-07
> **Status:** Active Reference

---

## Executive Summary

The ArmorClaw Docker quickstart image (`mikegemut/armorclaw:latest`) provides a **single-command deployment** that bundles everything needed for a complete ArmorClaw installation:

- **Go Bridge** - Core bridge with encrypted keystore, Matrix adapter, JSON-RPC server
- **Matrix Conduit** - Homeserver for E2EE messaging
- **Setup Wizard** - Huh? TUI for interactive configuration (or env vars for non-interactive)
- **Agent Runtime** - Python venv + Node.js for agent execution
- **Agent Studio** - No-code agent factory with skill/PII registries
- **Browser Automation** - Event-based browser control with PII protection
- **MCP Marketplace** - External data connections with approval workflow

**Key Design Principle:** Zero persistent secrets on disk. All credentials are injected into the SQLCipher-encrypted keystore at runtime.

---

## Quickstart Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW QUICKSTART FLOW (v0.4.0)                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: docker run -it -v /var/run/docker.sock:... armorclaw:latest   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. QUICKSTART.SH ENTRYPOINT                                         │   │
│  │    • Initialize logging (/var/log/armorclaw/setup.log)             │   │
│  │    • Check for --help / --version flags                            │   │
│  │    • Verify Docker socket exists                                   │   │
│  │    • Start log backup if ARMORCLAW_DEV_MODE=true                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. GO WIZARD (armorclaw-bridge container-setup)                     │   │
│  │    • Check env vars FIRST (tryNonInteractive)                      │   │
│  │      - If ARMORCLAW_API_KEY set → non-interactive mode             │   │
│  │      - Server name passed from host (ARMORCLAW_SERVER_NAME)        │   │
│  │    • Else: Check terminal (TTY, color support, size)               │   │
│  │    • Launch Huh? TUI wizard if terminal OK                         │   │
│  │      - Step 1 of 2: AI Provider + API Key                          │   │
│  │      - Step 2 of 2: Admin Password + Deploy confirmation           │   │
│  │    • Output: /tmp/armorclaw-wizard.json + env vars for secrets     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. CONTAINER-SETUP.SH (Infrastructure)                              │   │
│  │    • Preflight checks (Docker, network, DNS, disk space)           │   │
│  │    • Progress tracking (10 steps with progress bar)                │   │
│  │    • Create directories (/etc/armorclaw, /var/lib/armorclaw)       │   │
│  │    • Generate self-signed SSL certificate                          │   │
│  │    • Configure Matrix (Conduit)                                    │   │
│  │    • Configure API provider                                        │   │
│  │    • Create admin user                                             │   │
│  │    • Save admin password to /var/lib/armorclaw/.admin_password     │   │
│  │    • Write config.toml                                             │   │
│  │    • Initialize SQLCipher keystore                                 │   │
│  │    • Initialize Agent Studio database (studio.db)                  │   │
│  │    • Start Matrix stack (docker compose up)                        │   │
│  │    • Register bridge + admin users on Conduit                      │   │
│  │    • Create "ArmorClaw Bridge" room with admin as Owner            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 4. POST-SETUP (quickstart.sh continues)                             │   │
│  │    • Start bridge in background                                    │   │
│  │    • Wait for bridge.sock to appear                                │   │
│  │    • Inject API key via RPC (store_key method)                     │   │
│  │    • Auto-claim OWNER role for admin via provisioning.claim        │   │
│  │    • Generate QR code for ArmorChat provisioning                   │   │
│  │    • Display connection info + credentials                         │   │
│  │    • Wait for bridge process                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│                          ARMORCLAW RUNNING                                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Matrix Event Bus Improvements (v0.5.0)

The bridge now uses a high-throughput event bus with zero-allocation receive path and proper context cancellation support.

### Bug Fixes Applied

| Bug | Severity | Fix |
|-----|----------|-----|
| Goroutine leak in WaitForEvents | Critical | Replaced blocking wait with ticker-based polling (25ms intervals) |
| RPC handler timeout ignored | High | Wrapped context with client-specified timeout |
| Polling handler logic broken | Medium | Simplified to single-level timeout with nil guard |
| Status panic on empty buffer | Low | Added empty buffer guard with consistent sequence semantics |

### Architecture

```
Matrix Homeserver → MatrixAdapter → MatrixEventBus → RPC matrix.receive → Agent
```

### Key Features

- **Zero-allocation receive path**: Uses pre-allocated batch buffers
- **Instant wake-up**: Events delivered within 25ms, not polling storms
- **Slow consumer detection**: Cursor guard prevents message loss
- **Context cancellation**: Proper timeout handling prevents indefinite blocking

### RPC Methods

| Method | Description |
|--------|-------------|
| `matrix.status` | Returns connection health and user info |
| `matrix.login` | Dynamic login through bridge |
| `matrix.send` | Message sending via adapter |
| `matrix.receive` | Long-polling with cursor + timeout |

### Container Agent Updates

- Agent now tracks event cursor
- Uses long-polling (25ms) instead of aggressive polling
- Handles cursor reset gracefully
- Reduced CPU and network overhead

---

## Install Script Flow (v0.4.2)

The `install.sh` script orchestrates the entire deployment process:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     INSTALL.SH FLOW (v0.4.2)                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  USER RUNS: curl -fsSL .../install.sh | bash                                │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. PREFLIGHT CHECKS                                                  │   │
│  │    • Verify Docker is installed and running                         │   │
│  │    • Check for root/sudo permissions                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. AUTO-DETECTION (on host, before container)                        │   │
│  │    • Detect available ports (8443, 6167, 5000 or fallback)          │   │
│  │    • Detect server IP: ip route get 1                               │   │
│  │    • Collect env vars from user's shell                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. DOCKER RUN (passes env vars to container)                         │   │
│  │    docker run ... \                                                 │   │
│  │      -e ARMORCLAW_SERVER_NAME=<detected-ip> \                       │   │
│  │      -e ARMORCLAW_API_KEY=<if-set> \                                │   │
│  │      -e ARMORCLAW_API_BASE_URL=<if-set> \                           │   │
│  │      -e ARMORCLAW_PROFILE=<if-set> \                                │   │
│  │      -e ARMORCLAW_ADMIN_PASSWORD=<if-set> \                         │   │
│  │      ...                                                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                 │                                           │
│                                 ▼                                           │
│                     CONTAINER STARTS (see Quickstart Flow)                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why Host-Side IP Detection?

Container-side IP detection returns the **container's internal IP** (172.17.x.x), not the host's public IP. This would break ArmorChat connectivity. The install script detects the IP on the host and passes it via `ARMORCLAW_SERVER_NAME`.

### Non-Interactive Mode

```bash
# Minimal - just API key (IP auto-detected)
export ARMORCLAW_API_KEY=sk-your-key
curl -fsSL https://raw.githubusercontent.com/armorclaw/armorclaw/main/deploy/install.sh | bash

# With explicit server
export ARMORCLAW_API_KEY=sk-your-key
export ARMORCLAW_SERVER_NAME=192.168.1.50
curl -fsSL ... | bash
```

### Bootstrap Mode (GitOps)

```bash
# Generate docker-compose.yml without starting
curl -fsSL ... | bash -s -- --bootstrap
# Output: /opt/armorclaw/docker-compose.yml
```

---

## New Features (v0.4.0)

### Agent Studio

The Agent Studio provides a **no-code interface** for creating and managing AI agents through Matrix chat commands or JSON-RPC.

**Key Components:**
- **Skill Registry** - 8 default skills (browser_navigate, form_filler, pdf_generator, etc.)
- **PII Registry** - 10 default PII fields with sensitivity levels (low/medium/high/critical)
- **Resource Profiles** - 3 tiers (low/medium/high) with memory/CPU limits
- **Agent Definitions** - Combine skills + PII access + resource limits
- **Instance Manager** - Docker container spawning with security hardening

**Matrix Commands:**
```
!agent help              Show command help
!agent list-skills       List available skills
!agent list-pii          List PII fields
!agent create <name>     Start interactive wizard
!agent list              List agent definitions
!agent spawn <id>        Spawn agent instance
!agent stop <instance>   Stop running instance
!agent stats             Show statistics
```

### Browser Skill API v1.1

Event-based protocol for ArmorChat to control a headless browser on the bridge.

**Event Namespace:** `com.armorclaw.browser.*`

**Commands:**
| Event | Purpose |
|-------|---------|
| `navigate` | Load URL with wait conditions |
| `fill` | Form field population |
| `click` | Element interaction |
| `wait` | Conditional pauses |
| `extract` | Data retrieval |
| `screenshot` | Page capture |

**PII Protection:**
- Automatic regex-based redaction (credit cards, SSN, email, phone, API keys)
- BlindFill references for secure PII injection via user approval
- Audit logging for compliance

### MCP Approval Workflow

Role-based access control for external MCP (Model Context Protocol) connections.

**Default MCPs:**
| MCP | Risk Level | Type |
|-----|------------|------|
| filesystem | high | local |
| web-search | low | external |
| database | critical | requires PII |
| github | medium | requires token |
| slack | medium | requires token |

**Approval Flow:**
1. Non-admin user requests MCP access → Creates pending approval
2. Admins notified via Matrix/push
3. Admin reviews and approves/rejects
4. User notified of decision
5. Approved MCPs auto-added to agent definition

**RPC Methods:**
- `studio.list_mcps` - List available MCPs
- `studio.get_mcp_warning` - Get risk assessment (admin)
- `studio.request_mcp_approval` - Request access (non-admin)
- `studio.list_pending_approvals` - View pending requests (admin)
- `studio.approve_mcp_request` - Approve request (admin)
- `studio.reject_mcp_request` - Reject request (admin)

---

## What Happens After `docker run`

### Immediate Actions (0-30 seconds)

When you run the Docker command, these things happen automatically:

1. **Container starts** - The `quickstart.sh` entrypoint runs
2. **Logging initialized** - All output goes to `/var/log/armorclaw/setup.log`
3. **Docker socket verified** - Must be mounted at `/var/run/docker.sock`
4. **Wizard launches** - Either Huh? TUI or non-interactive based on env vars

### Interactive Mode (30 seconds - 2 minutes)

If using the TUI wizard:

| Step | What Happens | User Action |
|------|--------------|-------------|
| AI Provider | Choose OpenAI, Anthropic, GLM-5, or Custom | Arrow keys + Enter |
| API Key | Enter your API key (masked) | Type key + Enter |
| Admin Password | Enter password or press Enter to auto-generate | Type + Enter or just Enter |
| Deploy | Confirm deployment | Select "Deploy" + Enter |

**Note:** Profile selection (Quick/Enterprise) is determined by `ARMORCLAW_PROFILE` env var (defaults to Quick).

### Infrastructure Setup (1-3 minutes)

After wizard completes, `container-setup.sh` runs automatically:

```
[####------] 40% Configuring API provider
[#####-----] 50% Setting up admin user
[######----] 60% Configuring bridge
[#######---] 70% Validating and writing configuration
[########--] 80% Starting Matrix stack
[#########-] 90% Registering users and creating rooms
[##########] 100% Setup Complete!
```

### Post-Setup Automation (30 seconds)

The `quickstart.sh` script then:

1. **Starts the bridge binary** in background
2. **Waits for socket** at `/run/armorclaw/bridge.sock`
3. **Injects API key** via JSON-RPC `store_key` method
4. **Claims OWNER role** for admin user via `provisioning.claim`
5. **Generates QR code** for ArmorChat mobile provisioning

---

## What You MUST Do After Setup

### 1. Save Your Credentials (CRITICAL)

The setup displays credentials **once**. Write them down:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Admin Login (Element X / ArmorChat):
  Username:   @admin:192.168.1.100
  Password:   Xy7kL9mN2pQ4rS8t
  Homeserver: http://192.168.1.100:6167

  Password saved to: /var/lib/armorclaw/.admin_password
  ⚠ Delete this file after first login for security.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Recovery if lost:**
```bash
# Password file (if still present)
docker exec armorclaw cat /var/lib/armorclaw/.admin_password

# Or reset via Matrix admin API (requires Conduit admin token)
```

### 2. Connect a Matrix Client

**Option A: Element X (Recommended)**

1. Download Element X: https://element.io/download
2. Open Element X
3. Click "Edit" next to homeserver
4. Enter: `http://YOUR_IP:6167` (or your domain)
5. Click "Sign in"
6. Enter username: `admin` (or what you configured)
7. Enter password from setup
8. You should see the "ArmorClaw Bridge" room

**Option B: ArmorChat (Mobile)**

1. Install ArmorChat on your mobile device
2. Scan the QR code displayed in terminal
3. Or manually enter:
   - Homeserver: `http://YOUR_IP:6167`
   - Username: `admin`
   - Password: (from setup)

### 3. Verify Bridge Connection

In the "ArmorClaw Bridge" room, send:

```
!status
```

You should receive a response like:

```
✓ Bridge is running
✓ Matrix connected
✓ Keystore initialized
✓ 1 API key configured
✓ Agent Studio ready
```

### 4. Create Your First Agent

Using Matrix commands:

```
!agent create "Document Processor"
```

Follow the interactive wizard to:
- Select skills (pdf_generator, template_filler)
- Configure PII access (client_name, client_email)
- Set resource tier (medium)

### 5. Test AI Functionality

Send a message to the bridge:

```
Hello, can you help me with something?
```

The bridge should respond using your configured AI provider.

### 6. Delete Password File (Security)

After successful login:

```bash
docker exec armorclaw rm /var/lib/armorclaw/.admin_password
```

---

## Environment Variables Reference

### Required for Non-Interactive Mode

| Variable | Required | Description |
|----------|----------|-------------|
| `ARMORCLAW_API_KEY` | **Yes** | Triggers non-interactive mode. Your AI provider's API key. |
| `ARMORCLAW_SERVER_NAME` | Auto | Server domain or IP. **Auto-detected on host** and passed to container. |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ARMORCLAW_API_BASE_URL` | OpenAI URL | Custom API endpoint (for Anthropic, GLM-5, etc.) |
| `ARMORCLAW_PROFILE` | `quick` | `quick` or `enterprise` |
| `ARMORCLAW_ADMIN_PASSWORD` | (generated) | Admin password for Matrix |
| `ARMORCLAW_HIPAA` | `false` | Enable HIPAA mode (enterprise profile) |
| `ARMORCLAW_QUARANTINE` | `false` | Enable quarantine mode (enterprise profile) |
| `ARMORCLAW_DEBUG` | `false` | Enable debug logging |

### Development/Debugging

| Variable | Description |
|----------|-------------|
| `ARMORCLAW_DEV_MODE` | Enable log backup to `/tmp/armorclaw-logs/` |
| `ARMORCLAW_ACCESSIBLE` | Enable accessible mode for screen readers |
| `ARMORCLAW_LOG_BACKUP_DIR` | Custom log backup directory |

---

## Common Post-Deployment Tasks

### Check System Health

```bash
# Container status
docker ps | grep armorclaw

# Bridge health via RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Matrix homeserver health
curl http://localhost:6167/_matrix/client/versions

# View setup log
docker exec armorclaw cat /var/log/armorclaw/setup.log

# Agent Studio stats
echo '{"jsonrpc":"2.0","id":1,"method":"studio.stats"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Add Another API Key

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"store_key","params":{"id":"anthropic-backup","provider":"anthropic","token":"sk-ant-xxx","display_name":"Anthropic Backup"}}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Generate New ArmorChat QR

The QR code is **automatically generated** at setup completion. It displays:
- ASCII QR code (scan with ArmorChat)
- Deep link (copy/paste to device)
- Web link (for browsers)

To regenerate:

```bash
docker exec armorclaw armorclaw-bridge generate-qr --host <server-ip> --port 8443
```

### View Logs

```bash
# Setup log
docker exec armorclaw view-setup-log

# Follow setup log
docker exec armorclaw view-setup-log --follow

# Errors only
docker exec armorclaw view-setup-log --errors
```

### List Available MCPs

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_mcps"}' | \
  docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Troubleshooting

### Setup Failed

If setup fails, you'll see:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SETUP FAILED (exit code: 1)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Rolling back partial setup...

Log file locations:
  Primary:   /var/log/armorclaw/setup.log
  Backup:    /tmp/armorclaw-logs/setup.log.backup

Last 30 lines of log:
  ...
```

**Steps to recover:**

1. Check the log: `docker cp armorclaw:/var/log/armorclaw/setup.log ./setup.log`
2. Fix the issue (usually Docker socket or network)
3. Remove the container: `docker rm -f armorclaw`
4. Re-run the docker command

### Can't Connect to Matrix

1. **Check port binding:**
   ```bash
   docker port armorclaw
   # Should show: 6167/tcp -> 0.0.0.0:6167
   ```

2. **Check firewall:**
   ```bash
   sudo ufw status
   sudo ufw allow 6167/tcp
   ```

3. **Check from inside container:**
   ```bash
   docker exec armorclaw curl http://localhost:6167/_matrix/client/versions
   ```

### Bridge Not Responding

1. **Check bridge process:**
   ```bash
   docker exec armorclaw ps aux | grep armorclaw-bridge
   ```

2. **Check socket:**
   ```bash
   docker exec armorclaw ls -la /run/armorclaw/bridge.sock
   ```

3. **Check RPC:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

### Agent Studio Issues

1. **Check studio database:**
   ```bash
   docker exec armorclaw ls -la /var/lib/armorclaw/studio.db
   ```

2. **List skills:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_skills"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

3. **Check agent instances:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"studio.list_instances"}' | \
     docker exec -i armorclaw socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
   ```

---

## Architecture Reference

### Container Layout

```
armorclaw container
├── /opt/armorclaw/
│   ├── armorclaw-bridge     # Go binary
│   ├── quickstart.sh        # Entry point
│   ├── container-setup.sh   # Setup wizard
│   ├── agent/               # Agent runtime
│   └── configs/             # Config templates
├── /etc/armorclaw/
│   ├── config.toml          # Main config
│   ├── ssl/                 # SSL certificates
│   └── .setup_complete      # Setup flag
├── /var/lib/armorclaw/
│   ├── keystore.db          # SQLCipher encrypted
│   ├── studio.db            # Agent Studio database
│   ├── .admin_user          # Admin info for OWNER claim
│   └── .admin_password      # Temp password file
├── /run/armorclaw/
│   └── bridge.sock          # Unix socket for RPC
└── /var/log/armorclaw/
    └── setup.log            # Setup log
```

### Network Topology

```
┌─────────────────────────────────────────────────────────┐
│  Host (VPS/Server)                                      │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │  armorclaw container (mikegemut/armorclaw)      │   │
│  │                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐            │   │
│  │  │ Bridge      │    │ Matrix      │            │   │
│  │  │ (Go binary) │◄──►│ Conduit     │            │   │
│  │  │ :8443/RPC   │    │ :6167       │            │   │
│  │  │             │    │             │            │   │
│  │  │ + Studio    │    └─────────────┘            │   │
│  │  │ + Browser   │                              │   │
│  │  │ + MCP       │                              │   │
│  │  └─────────────┘                              │   │
│  │         │                   │                   │   │
│  │         │    ┌─────────────┐│                   │   │
│  │         └───►│ Sygnal      ││                   │   │
│  │              │ Push :5000  ││                   │   │
│  │              └─────────────┘│                   │   │
│  │                            │                   │   │
│  └────────────────────────────────────────────────┘   │
│                           │                            │
│  Docker Socket ───────────┘ (mounted)                 │
│                                                        │
└────────────────────────────────────────────────────────┘
         │              │              │
      :8443          :6167          :5000
    (HTTPS/RPC)    (Matrix)      (Push)
```

---

## ArmorChat Communication Architecture

### Overview

ArmorChat communicates with ArmorClaw through **Matrix protocol** for all messaging and **JSON-RPC** for direct bridge commands. This architecture provides:

- **End-to-End Encryption (E2EE)** - All messages encrypted via Matrix
- **Push Notifications** - Real-time alerts via Sygnal + FCM
- **Offline Support** - Messages queued and delivered when online
- **Multi-Device** - Same account on multiple devices

### Communication Stack

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW ↔ ARMORCHAT COMMUNICATION                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐                              ┌─────────────────┐       │
│  │ ArmorChat       │                              │ ArmorClaw       │       │
│  │ (Android)       │                              │ (VPS)           │       │
│  │                 │                              │                 │       │
│  │ Matrix SDK      │◄──── E2EE Messages ────────►│ Matrix Conduit  │       │
│  │ (Kotlin)        │                              │ (Rust)          │       │
│  │                 │                              │                 │       │
│  │ FCM Push        │◄──── Push Notifications ────│ Sygnal          │       │
│  │                 │                              │ (Python)        │       │
│  │                 │                              │                 │       │
│  │ JSON-RPC Client │◄──── Direct Commands ──────►│ Bridge RPC      │       │
│  │ (HTTP/HTTPS)    │                              │ (Unix Socket)   │       │
│  └─────────────────┘                              └─────────────────┘       │
│         │                                                │                   │
│         │                                                │                   │
│         │              PROTOCOLS USED                     │                   │
│         │                                                │                   │
│         │  Matrix (CS API):                              │                   │
│         │  - /_matrix/client/v3/                         │                   │
│         │  - /_matrix/media/v3/                          │                   │
│         │  - m.room.message events                       │                   │
│         │  - Custom events (com.armorclaw.*)             │                   │
│         │                                                │                   │
│         │  Push (FCM):                                   │                   │
│         │  - Sygnal → FCM → Device                       │                   │
│         │  - Includes room_id, event_id                  │                   │
│         │                                                │                   │
│         │  JSON-RPC (HTTP):                              │                   │
│         │  - POST to :8443/rpc                           │                   │
│         │  - Auth via Bearer token                       │                   │
│         │                                                │                   │
│         └────────────────────────────────────────────────┘                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Initial Connection Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCHAT PROVISIONING FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Step 1: SETUP COMPLETE (Bridge displays QR)                                │
│  ───────────────────────────────────────                                    │
│                                                                             │
│  Bridge generates provisioning data:                                        │
│  {                                                                          │
│    "server_name": "192.168.1.50:6167",                                      │
│    "setup_token": "armorclaw-setup-abc123",                                │
│    "qr_data": "armorclaw://192.168.1.50:6167?token=abc123"                  │
│  }                                                                          │
│                                                                             │
│  Step 2: USER SCANS QR CODE                                                 │
│  ────────────────────────────                                               │
│                                                                             │
│  ArmorChat parses URI:                                                      │
│    armorclaw://<server>:<port>?token=<setup_token>                          │
│                                                                             │
│  Step 3: ARMORCHAT CONNECTS TO MATRIX                                       │
│  ───────────────────────────────────────                                    │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Conduit     │────►│ Register    │                    │
│  │             │     │ :6167       │     │ Device      │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  1. GET /_matrix/client/versions       │                           │
│         │  2. POST /_matrix/client/v3/register   │                           │
│         │     (with setup_token as device_id)    │                           │
│         │  3. Receive access_token, device_id    │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 4: BRIDGE AUTO-CLAIMS OWNER ROLE                                      │
│  ──────────────────────────────────────                                      │
│                                                                             │
│  Bridge calls: provisioning.claim(setup_token, device_id)                   │
│  → First claim = OWNER role                                                 │
│  → User added to "ArmorClaw Bridge" room                                    │
│                                                                             │
│  Step 5: E2EE SETUP                                                         │
│  ─────────────────                                                          │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Matrix SDK  │────►│ Key         │                    │
│  │             │     │ Crypto       │     │ Exchange    │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  1. Generate device keys               │                           │
│         │  2. Upload keys to server              │                           │
│         │  3. Download bridge's keys             │                           │
│         │  4. Verify via emoji (optional)        │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 6: PUSH NOTIFICATION SETUP                                            │
│  ─────────────────────────────────                                          │
│                                                                             │
│  ArmorChat enables push via Matrix HTTP Pusher:                             │
│    POST /_matrix/client/v3/pushers/set                                      │
│    {                                                                        │
│      "pushkey": "<FCM-token>",                                              │
│      "app_id": "com.armorclaw.armorchat",                                   │
│      "data": { "url": "http://server:5000/_matrix/push/v1/notify" }         │
│    }                                                                        │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                    │
│  │ ArmorChat   │────►│ Conduit     │────►│ Sygnal      │                    │
│  │             │     │             │     │ :5000       │                    │
│  └─────────────┘     └─────────────┘     └─────────────┘                    │
│         │                                        │                           │
│         │  Register pusher with FCM token        │                           │
│         │                                        │                           │
│         ▼                                        ▼                           │
│                                                                             │
│  Step 7: READY TO COMMUNICATE                                               │
│  ─────────────────────────────                                              │
│                                                                             │
│  ArmorChat can now:                                                         │
│  • Send encrypted messages to Bridge room                                   │
│  • Receive push notifications                                               │
│  • Execute commands via !agent, !status                                     │
│  • Control browser via Matrix events                                        │
│  • Approve PII via BlindFill                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Ongoing Communication

#### Message Flow (User → Agent → Response)

```
User Types Message                    Bridge Processing                    Response
─────────────────                    ──────────────────                   ─────────
      │                                     │                                   │
      │  1. User types: "Book a flight"    │                                   │
      │                                     │                                   │
      ▼                                     │                                   │
┌─────────────┐                             │                                   │
│ ArmorChat   │                             │                                   │
│ Matrix SDK  │                             │                                   │
│             │                             │                                   │
│ Encrypt     │                             │                                   │
│ message     │                             │                                   │
└──────┬──────┘                             │                                   │
       │                                    │                                   │
       │  m.room.encrypted                  │                                   │
       │  (ciphertext)                      │                                   │
       │                                    │                                   │
       ▼                                    ▼                                   │
┌─────────────┐                       ┌─────────────┐                          │
│ Conduit     │──────────────────────►│ Bridge      │                          │
│ :6167       │                       │ Matrix      │                          │
│             │                       │ Adapter     │                          │
└─────────────┘                       └──────┬──────┘                          │
                                             │                                 │
                                             │  2. Decrypt message             │
                                             │  3. Route to agent or AI        │
                                             │  4. Process request             │
                                             │                                 │
                                             ▼                                 │
                                       ┌─────────────┐                          │
                                       │ Agent / AI  │                          │
                                       │ Processor   │                          │
                                       └──────┬──────┘                          │
                                              │                                  │
                                              │  5. Generate response            │
                                              │                                  │
                                              ▼                                  │
                                       ┌─────────────┐                          │
                                       │ Bridge      │                          │
                                       │             │                          │
                                       │ Encrypt     │                          │
                                       │ response    │                          │
                                       └──────┬──────┘                          │
                                              │                                  │
                                              │  m.room.encrypted                │
                                              │  (response ciphertext)           │
                                              │                                  │
                                              ▼                                  │
                                       ┌─────────────┐     ┌─────────────┐      │
                                       │ Conduit     │────►│ ArmorChat   │      │
                                       │             │     │ Decrypt     │      │
                                       └─────────────┘     │ Display     │      │
                                                           └─────────────┘      │
                                              │                                  │
                                              │  6. Push notification            │
                                              │     (if app in background)       │
                                              ▼                                  │
                                       ┌─────────────┐     ┌─────────────┐      │
                                       │ Sygnal      │────►│ FCM         │      │
                                       │ :5000       │     │ → Device    │      │
                                       └─────────────┘     └─────────────┘      │
```

#### Browser Control Flow

```
User Requests Browser Action           Bridge Processing              Browser Executes
─────────────────────────              ──────────────────             ────────────────
              │                              │                              │
              │  1. User taps "Navigate"     │                              │
              │     in ArmorChat UI          │                              │
              │                              │                              │
              ▼                              │                              │
       ┌─────────────┐                       │                              │
       │ ArmorChat   │                       │                              │
       │             │                       │                              │
       │ JSON-RPC    │                       │                              │
       │ POST /rpc   │                       │                              │
       └──────┬──────┘                       │                              │
              │                              │                              │
              │  browser.enqueue_job({       │                              │
              │    agent_id,                 │                              │
              │    commands: [               │                              │
              │      { type: "navigate",     │                              │
              │        url: "..." }          │                              │
              │    ]                         │                              │
              │  })                          │                              │
              │                              │                              │
              ▼                              ▼                              │
       ┌─────────────┐                 ┌─────────────┐                      │
       │ Bridge RPC  │────────────────►│ Browser     │                      │
       │ :8443       │                 │ Queue       │                      │
       └─────────────┘                 │ Processor   │                      │
                                       └──────┬──────┘                      │
                                              │                             │
                                              │  2. Queue job               │
                                              │  3. Pick up from queue      │
                                              │                             │
                                              ▼                             │
                                       ┌─────────────┐                      │
                                       │ Browser     │                      │
                                       │ Service     │                      │
                                       │ (Playwright)│                      │
                                       └──────┬──────┘                      │
                                              │                             │
                                              │  4. Execute command         │
                                              │                             │
                                              ▼                             │
                                       ┌─────────────┐                      │
                                       │ Status      │                      │
                                       │ Events      │                      │
                                       └──────┬──────┘                      │
                                              │                             │
       ┌─────────────┐                        │                             │
       │ ArmorChat   │◄───────────────────────┘                             │
       │             │                                                      │
       │ Matrix:     │  com.armorclaw.browser.status                        │
       │   {         │  {                                                   │
       │     status: │    "status": "navigating",                           │
       │     url,    │    "url": "https://...",                             │
       │     progress│    "progress": 50                                    │
       │   }         │  }                                                   │
       └─────────────┘                                                      │
```

### Key Communication Endpoints

| Endpoint | Protocol | Purpose |
|----------|----------|---------|
| `:6167/_matrix/client/*` | Matrix CS API | All Matrix operations |
| `:6167/_matrix/federation/*` | Matrix Federation | Server-to-server (optional) |
| `:5000/_matrix/push/v1/notify` | HTTP POST | Push notifications |
| `:8443/rpc` | JSON-RPC 2.0 | Direct bridge commands |
| `:8443/health` | HTTP GET | Bridge health check |

### Provisioning RPC Methods

| Method | Purpose |
|--------|---------|
| `provisioning.start` | Generate setup token and QR data |
| `provisioning.claim` | Claim device with token (first = OWNER) |
| `provisioning.status` | Check provisioning state |

### Security Model

```
┌─────────────────────────────────────────────────────────────────┐
│  SECURITY LAYERS                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Layer 1: Network                                                │
│  ───────────────                                                 │
│  • HTTPS/TLS for all external traffic                            │
│  • Self-signed certs generated on setup                          │
│  • Optional Let's Encrypt with Caddy                             │
│                                                                  │
│  Layer 2: Matrix Authentication                                  │
│  ────────────────────────────                                    │
│  • Access tokens for all API calls                               │
│  • Device-specific tokens                                        │
│  • Token stored securely in Android Keystore                     │
│                                                                  │
│  Layer 3: End-to-End Encryption                                  │
│  ─────────────────────────────                                   │
│  • Olm/Megolm encryption (Matrix SDK)                            │
│  • Device keys generated locally                                 │
│  • Cross-signing for verification                                │
│  • Emoji verification for bridge                                 │
│                                                                  │
│  Layer 4: Role-Based Access Control                              │
│  ───────────────────────────────                                 │
│  • OWNER: Full access, can manage users                          │
│  • ADMIN: Can create agents, manage MCPs                         │
│  • USER: Can use agents, request MCP access                      │
│                                                                  │
│  Layer 5: BlindFill PII Protection                               │
│  ─────────────────────────────                                   │
│  • PII never sent to agent                                       │
│  • Decrypted only in browser memory                              │
│  • User approval required per-field                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## ArmorChat Android Integration

### Event Types Reference

| Event Type | Direction | Purpose |
|------------|-----------|---------|
| `com.armorclaw.browser.response` | Bridge → Client | Command results |
| `com.armorclaw.browser.status` | Bridge → Client | Browser state changes |
| `com.armorclaw.agent.status` | Bridge → Client | Agent state machine transitions |
| `com.armorclaw.pii.response` | Client → Bridge | User's PII approval/denial |

> **Important:** Agent state events use `com.armorclaw.agent.status` (not `.state`)

### Matrix Event Listeners

```kotlin
class BrowserCommandHandler(
    private val matrixClient: MatrixClient,
    private val controlPlaneStore: ControlPlaneStore
) {
    fun subscribe() {
        // Listen for browser responses (command results)
        matrixClient.onEvent("com.armorclaw.browser.response") { event ->
            handleBrowserResponse(event)
        }

        // Listen for browser status updates (navigation, form detection, etc.)
        matrixClient.onEvent("com.armorclaw.browser.status") { event ->
            handleBrowserStatus(event)
        }

        // Listen for agent status changes (state machine transitions)
        matrixClient.onEvent("com.armorclaw.agent.status") { event ->
            handleAgentStatus(event)
        }
    }

    private fun handleAgentStatus(event: MatrixEvent) {
        val status = AgentStatusEvent.fromJson(event.content)

        when (status.status) {
            "idle" -> showIdleState()
            "browsing" -> showBrowsingState(status.metadata.url)
            "form_filling" -> showFormFillingState(status.metadata.progress)
            "awaiting_approval" -> showApprovalNeeded(status.metadata.fieldsRequested)
            "awaiting_captcha" -> showCaptchaUI()
            "awaiting_2fa" -> show2FAUI()
            "processing_payment" -> showProcessingPayment()
            "complete" -> showComplete(status.metadata)
            "error" -> showError(status.metadata.error)
        }
    }
}
```

### Agent Status Event Structure

```kotlin
data class AgentStatusEvent(
    val agent_id: String,
    val status: String,          // "idle" | "browsing" | "form_filling" | etc.
    val previous: String?,       // Previous status
    val timestamp: Long,         // Unix milliseconds
    val metadata: StatusMetadata?
)

data class StatusMetadata(
    val url: String?,            // Current browser URL
    val step: String?,           // Current step description
    val progress: Int?,          // 0-100 progress
    val error: String?,          // Error message if status == "error"
    val task_id: String?,        // Current task identifier
    val task_type: String?,      // e.g., "flight_booking"
    val fields_requested: List<String>? // PII fields needed (when awaiting_approval)
)
```

### PII Field Reference Mapping

```kotlin
fun mapPIIFieldRef(ref: String): PiiField {
    return when (ref) {
        "payment.card_number" -> PiiField(
            name = "Card Number",
            sensitivity = SensitivityLevel.HIGH,
            description = "Credit or debit card number",
            currentValue = maskCardNumber(storedCard?.last4)
        )
        "payment.cvv" -> PiiField(
            name = "CVV",
            sensitivity = SensitivityLevel.CRITICAL,
            description = "Card verification code",
            currentValue = null // Never display
        )
        "payment.card_expiry" -> PiiField(
            name = "Expiry Date",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "MM/YY format",
            currentValue = storedCard?.expiry
        )
        "payment.card_name" -> PiiField(
            name = "Cardholder Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Name on card"
        )
        "personal.name" -> PiiField(
            name = "Full Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Your full name"
        )
        "personal.address" -> PiiField(
            name = "Address",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "Street address"
        )
        "personal.email" -> PiiField(
            name = "Email",
            sensitivity = SensitivityLevel.LOW,
            description = "Contact email"
        )
        "personal.phone" -> PiiField(
            name = "Phone",
            sensitivity = SensitivityLevel.LOW,
            description = "Phone number"
        )
        else -> PiiField(
            name = ref.substringAfterLast("."),
            sensitivity = SensitivityLevel.HIGH,
            description = "Requested: $ref"
        )
    }
}
```

### JSON-RPC Browser Queue Client

```kotlin
class BrowserQueueClient(
    private val bridgeUrl: String,
    private val authToken: String
) {
    suspend fun enqueueJob(request: EnqueueJobRequest): Result<BrowserJob>
    suspend fun getJob(jobId: String): Result<BrowserJob>
    suspend fun cancelJob(jobId: String): Result<Unit>
    suspend fun retryJob(jobId: String): Result<Unit>
    suspend fun getQueueStats(): Result<QueueStats>
    suspend fun listJobs(agentId: String? = null, status: List<String>? = null): Result<List<BrowserJob>>
}

data class EnqueueJobRequest(
    val id: String? = null,           // Optional, auto-generated if omitted
    val agent_id: String,
    val room_id: String,
    val definition_id: String? = null,
    val commands: List<BrowserCommandJson>,
    val priority: Int = 5,            // 1-10, higher = processed first
    val timeout: Int = 300,           // Seconds
    val max_retries: Int = 2,
    val expires_in: Int? = null       // Seconds from now
)

data class BrowserJob(
    val id: String,
    val agent_id: String,
    val room_id: String,
    val user_id: String?,
    val definition_id: String?,
    val status: JobStatus,            // "pending" | "running" | "completed" | "failed" | "cancelled"
    val priority: Int,
    val attempts: Int,
    val current_step: Int,
    val total_steps: Int,
    val error: String?,
    val created_at: Long,
    val started_at: Long?,
    val completed_at: Long?,
    val result: Map<String, Any?>?
)

data class QueueStats(
    val total: Int,
    val pending: Int,
    val running: Int,
    val completed: Int,
    val failed: Int,
    val cancelled: Int,
    val awaiting_pii: Int,
    val active_workers: Int,
    val queue_depth: Int
)
```

### PII Response Handling

```kotlin
// When user approves/denies in BlindFill dialog:
fun respondToPIIRequest(requestId: String, approved: Boolean, values: Map<String, String>?) {
    // Send PII response as Matrix event
    matrixClient.sendEvent(roomId, mapOf(
        "type" to "com.armorclaw.pii.response",
        "content" to mapOf(
            "request_id" to requestId,
            "approved" to approved,
            "values" to (values ?: emptyMap<String, String>())
        )
    ))

    // Clear pending request
    controlPlaneStore.removePiiRequest(requestId)
}

// In ChatViewModel - connect to existing approvePiiRequest
fun approvePiiRequest(approvedFields: Set<String>) {
    val request = _pendingPiiRequest.value ?: return

    val values = mutableMapOf<String, String>()
    approvedFields.forEach { fieldName ->
        when (fieldName) {
            "Card Number" -> values["payment.card_number"] = secureStorage.getCardNumber()
            "CVV" -> values["payment.cvv"] = secureStorage.getCVV()
            "Expiry Date" -> values["payment.card_expiry"] = secureStorage.getCardExpiry()
            "Cardholder Name" -> values["payment.card_name"] = userPrefs.getCardName()
            "Full Name" -> values["personal.name"] = userPrefs.getFullName()
            "Address" -> values["personal.address"] = userPrefs.getAddress()
            "Email" -> values["personal.email"] = userPrefs.getEmail()
            "Phone" -> values["personal.phone"] = userPrefs.getPhone()
        }
    }

    respondToPIIRequest(request.requestId, approved = true, values = values)
}
```

### Complete Checkout Flow Example

```kotlin
fun startCheckout(url: String, agent: AgentDefinition) = scope.launch {
    // 1. Create browser job via JSON-RPC
    val job = queueClient.enqueueJob(
        EnqueueJobRequest(
            agent_id = agent.id,
            room_id = currentRoomId,
            definition_id = agent.definitionId,
            commands = listOf(
                BrowserCommandJson("navigate", mapOf("url" to url, "waitUntil" to "load")),
                BrowserCommandJson("fill", mapOf(
                    "fields" to listOf(
                        mapOf("selector" to "#email", "value" to userPrefs.email),
                        mapOf("selector" to "#address", "value_ref" to "personal.address")
                    )
                )),
                BrowserCommandJson("click", mapOf("selector" to "#continue-btn", "waitFor" to "navigation"))
            ),
            priority = 5,
            timeout = 300
        )
    ).getOrThrow()

    // 2. Subscribe to status updates
    matrixService.onEvent("com.armorclaw.agent.status")
        .filter { it.content.agent_id == agent.id }
        .collect { event ->
            val status = AgentStatusEvent.fromJson(event.content)
            when (status.status) {
                "awaiting_approval" -> {
                    showBlindFillDialog(status.metadata?.fields_requested ?: emptyList())
                }
                "complete" -> {
                    showCheckoutComplete(status.metadata)
                }
                "error" -> {
                    showError(status.metadata?.error ?: "Unknown error")
                }
            }
        }
}
```

---

## Browser Service Architecture (v0.4.1)

### Components

The browser automation system consists of three main components:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BROWSER AUTOMATION ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │ ArmorChat       │     │ Bridge          │     │ Browser Service │       │
│  │ (Android)       │     │ (Go)            │     │ (TypeScript)    │       │
│  │                 │     │                 │     │                 │       │
│  │ JSON-RPC        │────►│ Browser Client  │────►│ Playwright      │       │
│  │ Matrix Events   │     │ Queue Processor │     │ Stealth Mode    │       │
│  │                 │     │                 │     │                 │       │
│  └─────────────────┘     └─────────────────┘     └─────────────────┘       │
│         │                       │                       │                   │
│         │                       ▼                       │                   │
│         │              ┌─────────────────┐              │                   │
│         │              │ Job Queue       │              │                   │
│         │              │ (SQLite)        │              │                   │
│         │              └─────────────────┘              │                   │
│         │                       │                       │                   │
│         └───────────────────────┴───────────────────────┘                   │
│                     Matrix Events (status, response)                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Browser Service (TypeScript/Playwright)

**Location:** `browser-service/`

**Features:**
- Playwright-based headless browser automation
- Anti-detection / stealth mode
- Screenshot capture with element cropping
- Form filling with PII injection
- Cookie and session management
- Proxy rotation support

**API Endpoints:**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Service health check |
| `/navigate` | POST | Navigate to URL |
| `/fill` | POST | Fill form fields |
| `/click` | POST | Click element |
| `/extract` | POST | Extract page data |
| `/screenshot` | POST | Capture screenshot |
| `/status` | GET | Current browser state |

### Browser Client (Go)

**Location:** `bridge/pkg/browser/`

**Components:**
- `client.go` - HTTP client for browser-service API
- `processor.go` - Job queue processor with retry logic
- `browser.go` - Core browser types and interfaces

**Configuration:**
```toml
[browser]
enabled = true
service_url = "http://localhost:3001"
timeout = 30
max_retries = 3
retry_delay = 2

[browser.stealth]
enabled = true
fingerprint_seed = ""

[browser.queue]
max_workers = 3
max_depth = 100
```

### Deployment

**Docker Compose:** `deploy/browser/docker-compose.browser.yml`

```yaml
services:
  browser-service:
    build: ../../browser-service
    ports:
      - "3001:3001"
    environment:
      - NODE_ENV=production
      - STEALTH_MODE=true
    cap_add:
      - SYS_ADMIN
    security_opt:
      - seccomp=unconfined
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.5.0 | 2026-03-07 | **Matrix event bus stability**: Fixed goroutine leak, proper timeout handling, zero-allocation receive path, slow consumer detection |
| 0.4.2 | 2026-03-05 | **Install script fixes**: Host-side IP detection, env var pass-through, corrected wizard flow docs |
| 0.4.1 | 2026-02-28 | Browser Service (TypeScript/Playwright), Browser Client, Queue Processor |
| 0.4.0 | 2026-02-28 | Agent Studio, Browser Skill API, MCP Approval Workflow |
| 0.3.3 | 2026-02-26 | Preflight checks, progress indication, rollback, password file |
| 0.3.2 | 2026-02-25 | Crash handler, logging, dev mode log backup |
| 0.3.1 | 2026-02-24 | Initial Docker Hub release |

---

## Related Documentation

- **[quickstart-docker.md](../guides/quickstart-docker.md)** - Full quickstart guide
- **[error-catalog.md](../guides/error-catalog.md)** - Error codes and solutions
- **[configuration.md](../guides/configuration.md)** - Post-setup configuration
- **[troubleshooting.md](../guides/troubleshooting.md)** - Detailed troubleshooting
- **[rpc-api.md](../reference/rpc-api.md)** - Complete JSON-RPC API reference
- **[agent-studio-rpc-api.md](../reference/agent-studio-rpc-api.md)** - Agent Studio RPC methods

---

**Document Version:** 1.3.0
**Last Updated:** 2026-03-05
**Maintainer:** ArmorClaw Team

## 2026-03-08 - Installer Hardening Review

### Completed Work

**Installer Hardening (Phase 6)**

Successfully implemented production-grade installer with 19 changes across 4 scripts:

1. **Docker Daemon Readiness Checks**
   - wait_for_docker() function in all installers
   - Dual-check: docker info && docker ps
   - 20-second timeout with 2-second intervals
   - Handles Docker startup race conditions

2. **Installer Lockfile**
   - Uses flock for parallel install prevention
   - EXIT trap with flock -u 2>/dev/null || true
   - Prevents race conditions
   - Skipped gracefully if flock not available

3. **Persistent Logging**
   - /var/log/armorclaw/install.log
   - Fallback to /tmp/armorclaw if /var/log unavailable
   - Uses exec > >(tee -a "$LOG_FILE") 2>&1
   - Captures both stdout and stderr

4. **Environment Variable Passthrough**
   - DOCKER_COMPOSE, CONDUIT_VERSION, CONDUIT_IMAGE exported to sub-scripts
   - env -S bash for proper inheritance
   - All variables accessible in setup-quick.sh and setup-matrix.sh

5. **Docker Compose Detection**
   - Detects both 'docker compose' and 'docker-compose'
   - Exported DOCKER_COMPOSE for use in all scripts
   - Fallback mechanism in sub-installers
   - Uses quoted "$DOCKER_COMPOSE" with || fail

6. **Conduit Image Unification**
   - CONDUIT_IMAGE variable with fallback to matrixconduit/matrix-conduit:latest
   - Supports CONDUIT_VERSION environment variable
   - Consistent across all installers
   - Self-contained with proper defaults

7. **Data Directory Consistency**
   - Fixed database_path = "/var/lib/conduit" in deploy-infra.sh
   - Was: /var/lib/matrix-conduit
   - Now: /var/lib/conduit (matches Docker mount)

8. **Safety Improvements**
   - Command -v checks before use
   - Proper error messages to stderr
   - || true for idempotent operations
   - Greedy matching for pattern detection
   - POSIX-compliant bash constructs

### Test Coverage

Created comprehensive test suite with 8 tests:

```
✓ Lockfile functionality (skip if flock not available)
✓ Docker wait loop (dual-check with info + ps)
✓ Environment variable passthrough
✓ Docker Compose detection
✓ CONDUIT_IMAGE fallback
✓ Syntax validation of all installers
✓ wait_for_docker function existence
✓ Variable ordering
```

**Test Results:** All 8 tests passed

### Files Modified

| File | Changes |
|------|---------|
| deploy/install.sh | 10 (lockfile, logging, wait, env passthrough, CONDUIT_IMAGE) |
| deploy/setup-matrix.sh | 3 (wait, DOCKER_COMPOSE, CONDUIT_IMAGE) |
| deploy/quickstart-entrypoint.sh | 2 (wait, CONDUIT_IMAGE) |
| deploy/deploy-infra.sh | 4 (wait, DOCKER_COMPOSE, CONDUIT_IMAGE, database_path fix) |
| tests/integration/test-installer-hardening.sh | New test suite (8 tests) |

### Benefits

- **No race conditions** - Lockfile prevents parallel installs
- **Safe re-runs** - Idempotent design with container reuse
- **Docker-startup resilient** - Waits for daemon to be fully ready
- **Portable** - Works with both docker compose and docker-compose
- **Debuggable** - Persistent logs for troubleshooting
- **Consistent** - Single homeserver, single data directory
- **Self-contained** - Fallbacks prevent installation failures

### Commits

1. `6e5c1f77dd30eb35d49804176a04875322c038d4` - fix(installer): harden installation flow
2. Tests and documentation pending commit

### Next Steps

- Add to CI/CD pipeline
- Integration test with actual Docker daemon
- Document in setup guide
- Add to release notes
