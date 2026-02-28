# ArmorClaw Quickstart Review

> **Purpose:** Complete guide to the Docker quickstart process and post-deployment steps
> **Version:** 0.4.0
> **Last Updated:** 2026-02-28
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
│  │      - Auto-detect server name if not provided                     │   │
│  │    • Else: Check terminal (TTY, color support, size)               │   │
│  │    • Launch Huh? TUI wizard if terminal OK                         │   │
│  │      - Page 1: Profile selection (Quick/Enterprise)                │   │
│  │      - Page 2: AI provider + API key                               │   │
│  │      - Page 3: Admin password + Deploy confirmation                │   │
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
| Profile | Select Quick Start or Enterprise | Arrow keys + Enter |
| AI Provider | Choose OpenAI, Anthropic, GLM-5, or Custom | Arrow keys + Enter |
| API Key | Enter your API key (masked) | Type key + Enter |
| Admin Password | Enter password or press Enter to auto-generate | Type + Enter or just Enter |
| Deploy | Confirm deployment | Select "Deploy" + Enter |

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
| `ARMORCLAW_SERVER_NAME` | Auto | Server domain or IP. Auto-detected if omitted. |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ARMORCLAW_PROFILE` | `quick` | `quick` or `enterprise` |
| `ARMORCLAW_API_BASE_URL` | OpenAI | Provider API URL |
| `ARMORCLAW_ADMIN_USER` | `admin` | Admin username |
| `ARMORCLAW_ADMIN_PASSWORD` | (generated) | Admin password |
| `ARMORCLAW_HIPAA` | `false` | Enable HIPAA mode |
| `ARMORCLAW_QUARANTINE` | `false` | Enable quarantine mode |
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

```bash
docker exec armorclaw armorclaw-bridge generate-qr
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

## Version History

| Version | Date | Changes |
|---------|------|---------|
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

**Document Version:** 1.1.0
**Last Updated:** 2026-02-28
**Maintainer:** ArmorClaw Team
