# CLAUDE.md

> **For AI Agents Working on ArmorClaw**
> **Last Updated:** 2026-02-07
> **Documentation Flow:** README → Architecture → Features → Functions

---

# Project-Level Agent Team Configuration

## Team Spawning Instructions
- When a team is requested, spawn the following roles based on the `security-completion-plan.md`:
    - **Script Architect**: Owns `deploy/setup-wizard.sh`.
        - **Security Auditor**: Owns validation of `bridge/pkg/pii/` and `bridge/internal/adapter/matrix.go`.
            - **FinOps Controller**: Owns `bridge/pkg/budget/`.
                - **Documentation Lead**: Owns `docs/` and `README.md`.

                ## Local Constraints
                - Agent teams are enabled via the local `.claude/settings.json`.
                - All teammates must adhere to the "Linear Dependency Chain" defined in the completion plan.

---

## 📚 Documentation Flow (Read Least Possible)

**Follow this order to minimize document reading:**

### Level 1: Project Overview (START HERE)
**Read first:** `README.md`
- What ArmorClaw does
- Quick start guide
- Basic concepts

### Level 2: Architecture & Design
**Read next:** `docs/index.md` → `docs/plans/2026-02-05-armorclaw-v1-design.md`
- System architecture
- Security model
- Component interactions
- Technology decisions

### Level 3: Feature Documentation
**Read as needed:** `docs/guides/` and `docs/reference/`
- 🔍 **Error Catalog:** `docs/guides/error-catalog.md` (search by error text - LLM-friendly!)
- Setup: `docs/guides/setup-guide.md`
- Configuration: `docs/guides/configuration.md`
- RPC API: `docs/reference/rpc-api.md`
- Troubleshooting: `docs/guides/troubleshooting.md`
- Development: `docs/guides/development.md`

### Level 4: Implementation Details
**Read only when coding:** Source files
- `bridge/` - Go bridge implementation
- `container/` - Container runtime
- `tests/` - Test suites

---

## 🎯 Quick Decision Tree

```
What do you need to do?
├── Understand the project?
│   └── Read: README.md
│
├── Design/plan a feature?
│   └── Read: docs/index.md → docs/plans/2026-02-05-armorclaw-v1-design.md
│
├── Implement a feature?
│   ├── Read: docs/guides/development.md (coding standards)
│   ├── Read: docs/reference/rpc-api.md (if adding RPC methods)
│   └── Read: Source files (implementation details)
│
├── Configure/deploy?
│   ├── Read: docs/guides/configuration.md
│   ├── Read: docs/guides/2026-02-05-infrastructure-deployment-guide.md
│   └── Read: docs/guides/troubleshooting.md (if issues)
│
├── Debug an issue?
│   ├── 🔍 **First:** Search error text in docs/guides/error-catalog.md
│   ├── Then: docs/guides/troubleshooting.md (systematic debugging)
│   ├── Then: docs/status/2026-02-05-status.md (known issues)
│   └── Finally: Source code (implementation details)
│
└── Check project status?
    └── Read: docs/PROGRESS/progress.md (milestones)
```

---

## 📁 Documentation Structure

```
ArmorClaw/
├── README.md                          # ⭐ START HERE - Project overview
├── CLAUDE.md                          # This file - AI agent guidance
│
├── docs/                              # 📚 ALL DOCUMENTATION
│   ├── index.md                       # 🗂️ Documentation hub (LEVEL 2)
│   │
│   ├── plans/                         # 🏗️ Architecture & Design (LEVEL 2)
│   │   ├── 2026-02-05-armorclaw-v1-design.md
│   │   ├── 2026-02-05-phase1-implementation-tasks.md
│   │   └── 2026-02-05-business-model-architecture.md
│   │
│   ├── guides/                        # 📖 How-to Guides (LEVEL 3)
│   │   ├── error-catalog.md            # 🔍 Every error with solutions (LLM-friendly!)
│   │   ├── setup-guide.md              # Interactive setup wizard
│   │   ├── configuration.md
│   │   ├── troubleshooting.md
│   │   ├── development.md
│   │   ├── element-x-configs.md
│   │   └── 2026-02-05-infrastructure-deployment-guide.md
│   │
│   ├── reference/                     # 📋 Technical Specs (LEVEL 3)
│   │   └── rpc-api.md                 # Complete JSON-RPC 2.0 API
│   │
│   ├── status/                        # 📊 Project Status (LEVEL 2)
│   │   └── 2026-02-05-status.md
│   │
│   ├── PROGRESS/                      # ✅ Milestone Tracking (LEVEL 2)
│   │   └── progress.md                # PRIMARY LOG - Read first!
│   │
│   └── output/                        # 📝 Milestone Reviews (LEVEL 2)
│       ├── review.md                  # Architecture reviews
│       ├── setup-flow-security-analysis.md
│       └── startup-config-analysis.md
│
├── bridge/                            # 🔧 Bridge Implementation (LEVEL 4)
│   ├── cmd/bridge/main.go             # Entry point
│   ├── pkg/
│   │   ├── config/                    # Configuration system
│   │   ├── docker/                    # Docker client
│   │   ├── keystore/                  # Encrypted credential storage
│   │   └── rpc/                       # JSON-RPC server
│   └── internal/adapter/              # Matrix adapter
│
├── container/                         # 🐳 Container Runtime (LEVEL 4)
│   ├── opt/openclaw/
│   │   ├── entrypoint.py              # Container entrypoint
│   │   └── health.sh                  # Health check script
│   └── openclaw/
│       ├── agent.py                   # ArmorClaw agent
│       └── bridge_client.py           # Bridge communication
│
├── tests/                             # 🧪 Test Suites (LEVEL 4)
│   ├── test-secret-passing.sh
│   ├── test-attach-config.sh
│   └── test-hardening.sh
│
├── Dockerfile                         # Container image definition
├── docker-compose.yml                 # Infrastructure stack
└── Makefile                           # Test orchestration
```

---

## 🚀 Getting Started Workflow

### For New AI Agents

1. **Read `README.md`** (5 minutes)
   - Understand what ArmorClaw does
   - Learn basic concepts

2. **Read `docs/index.md`** (3 minutes)
   - Get familiar with documentation structure
   - Find relevant guides

3. **Read `docs/PROGRESS/progress.md`** (2 minutes)
   - Understand current milestone status
   - See what's been completed

4. **Read relevant architecture docs** (as needed)
   - `docs/plans/2026-02-05-armorclaw-v1-design.md` for system design
   - Feature-specific guides for your task

### For Specific Tasks

| Task | Documents to Read | Source Files |
|------|-------------------|--------------|
| Add RPC method | `docs/reference/rpc-api.md` | `bridge/pkg/rpc/server.go` |
| Fix config issue | `docs/guides/configuration.md` | `bridge/pkg/config/` |
| Update container | `docs/plans/2026-02-05-armorclaw-v1-design.md` | `container/`, `Dockerfile` |
| Debug startup | `docs/guides/troubleshooting.md` | `container/opt/openclaw/entrypoint.py` |
| Add tests | `docs/guides/development.md` | `tests/` |

---

## 🔐 Security Principles (CRITICAL)

When working on this codebase:

1. **Never expose the Docker socket** to the container
2. **Never write secrets to disk** — always use memory-only injection
3. **Validate all inputs** through the Local Bridge schema
4. **Principle of least privilege**: Container should have minimal necessary access
5. **All host interaction must be pull-based** — agent requests, bridge validates
6. **No inbound ports** in the default configuration

---

## 🛠️ Common Commands

### Build
```bash
# Build bridge
cd bridge && go build -o build/armorclaw-bridge ./cmd/bridge

# Build container
docker build -t armorclaw/agent:v1 .
```

### Test
```bash
# Run all tests
make test-all

# Individual test suites
make test-hardening
./tests/test-secret-passing.sh
./tests/test-attach-config.sh
```

### Run
```bash
# Start bridge
sudo ./bridge/build/armorclaw-bridge

# Test RPC
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## 📦 Component Overview

### Bridge (Go)
- **Location:** `bridge/`
- **Purpose:** Secure interface between host and containers
- **Protocol:** JSON-RPC 2.0 over Unix socket (`/run/armorclaw/bridge.sock`)
- **Key Features:**
  - Encrypted keystore (SQLCipher + XChaCha20-Poly1305)
  - Scoped Docker client (create, exec, remove only)
  - Matrix adapter (E2EE support)
  - Configuration system (TOML + env vars)

### Container (Hardened Docker)
- **Location:** `container/`
- **Base:** `debian:bookworm-slim`
- **User:** UID 10001 (claw, non-root)
- **Hardening:** No shell, no network tools, no destructive tools
- **Entry:** `container/opt/openclaw/entrypoint.py`
- **Agent:** `container/openclaw/agent.py`

### Secret Injection
- **Method:** File-based (cross-platform compatible)
- **Path:** `/run/armorclaw/secrets/<container>.json`
- **Lifecycle:** Created → Mounted → Read → Cleaned up (10s)
- **Exposure:** Memory only, never in `docker inspect`

---

## 📋 Documentation Rules

### Rule 1: Root Directory Stays Clean
- **No loose documentation files** in root
- Root contains only: `CLAUDE.md`, `README.md`, config files, code directories
- All project documentation under `docs/`

### Rule 2: Update Progress First
- **PRIMARY:** `docs/PROGRESS/progress.md`
- Update after every milestone
- Include date, status, artifacts

### Rule 3: Update Related Docs
- Feature docs: `docs/guides/`
- API docs: `docs/reference/`
- Status: `docs/status/`
- Architecture: `docs/output/review.md`

### Rule 4: Keep docs/index.md Synchronized
- All new documentation must be linked from index
- Check links are working

---

## 🎯 Technology Decisions (Locked for v1)

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Container | Docker | Industry standard, mature tooling |
| Base Image | debian:bookworm-slim | Minimal, well-maintained |
| Bridge | Go 1.24+ | Performance, strong crypto, static binaries |
| Keystore | SQLCipher | Encrypted at rest, SQLite simplicity |
| Encryption | XChaCha20-Poly1305 | Hardware-bound, modern AEAD |
| Protocol | JSON-RPC 2.0 | Simple, widely supported |
| Transport | Unix socket | No network exposure, minimal overhead |
| Communication | Matrix (Conduit) | E2EE, decentralized, rich ecosystem |

---

## 🔍 Finding Things Quickly

### Need to understand the project?
→ `README.md`

### Need to know what's implemented?
→ `docs/PROGRESS/progress.md`

### Need to understand the architecture?
→ `docs/plans/2026-02-05-armorclaw-v1-design.md`

### Need to use the RPC API?
→ `docs/reference/rpc-api.md`

### Need to configure the bridge?
→ `docs/guides/configuration.md`

### Need to deploy?
→ `docs/guides/2026-02-05-infrastructure-deployment-guide.md`

### Need to troubleshoot?
→ `docs/guides/troubleshooting.md`

### Need to contribute code?
→ `docs/guides/development.md`

### Need to understand a specific component?
→ Read the source code in `bridge/` or `container/`

---

## 📌 Current Status (2026-02-24)

**Phase:** Phase 5 Complete - Docker Deployment Hardened (v4.1.0)

**Bridge Core (Complete):**
- ✅ Encrypted Keystore (SQLCipher + XChaCha20-Poly1305)
- ✅ Docker Client (scoped operations + seccomp)
- ✅ Matrix Adapter (E2EE support)
- ✅ JSON-RPC Server (24+ methods)
- ✅ Configuration System (TOML + env vars)
- ✅ Container Entrypoint (secrets validation + fail-fast)
- ✅ Agent Integration (bridge client + ArmorClawAgent)

**ArmorChat Android (Feature Complete):**
- ✅ E2EE Support (Matrix SDK crypto)
- ✅ Push Notifications (Matrix HTTP Pusher + FCM)
- ✅ Key Backup/Recovery (SSSS passphrase)
- ✅ Bridge Verification (emoji verification)
- ✅ Identity Management (namespace-aware)
- ✅ Feature Suppression (capability-aware UI)
- ✅ Migration Path (v2.5 → v4.6)

**Infrastructure (Complete):**
- ✅ Topology Separation (docker-compose.matrix.yml + docker-compose.bridge.yml)
- ✅ Health Check Script (deploy/health-check.sh)
- ✅ Sygnal Push Gateway
- ✅ FFI Boundary Tests (Kotlin + Go)

**Docker Deployment (Hardened):**
- ✅ Dockerfile.quickstart — 19 fixes across 5 review passes
- ✅ container-setup.sh — All prompts retry on invalid input (no stuck users)
- ✅ Docker Compose — Parameterized paths, env var passthrough, V2 plugin support
- ✅ Bridge build — CGO_ENABLED=1, libsqlite3-0 in runtime stage
- ✅ Security hooks — AF_UNIX allowed through LD_PRELOAD

**Next:**
- ⏳ Production deployment and integration testing
- ⏳ End-to-end E2EE verification with real devices

---

## 📞 Support

- **GitHub:** https://github.com/armorclaw/armorclaw
- **Documentation Issues:** Create issue with `docs:` label
- **Bug Reports:** Create issue with `bug:` label

---

**CLAUDE.md Last Updated:** 2026-02-24
**Phase:** Phase 5 Complete - Docker Deployment Hardened (v4.1.0)
