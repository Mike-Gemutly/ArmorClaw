# ArmorClaw Changelog

> **Last Updated:** 2026-02-25
> **Current Version:** 0.3.1

All notable changes to ArmorClaw are documented here with commit references.

---

## [0.3.1] - 2026-02-24 - Docker Quickstart Admin & Hangup Fixes (Pass 6)

### Fixed
- **Bridge user never created on Conduit** — Setup wizard collected bridge credentials but never registered the user; bridge `matrix.Login()` fatally crashed the container on every first run
- **No admin user for Element X / ArmorChat** — `create-matrix-admin.sh` was copied but never called; chicken-and-egg problem with admin token requirement made it unusable
- **No Conduit health check after `docker compose up`** — Replaced `sleep 5` with a 120-second polling loop on `/_matrix/client/versions`, with progress indicators
- **`conduit.toml` server_name hardcoded** — Template always said `matrix.armorclaw.com`; now dynamically updated via `sed` to match user's domain/IP before copying to host
- **`set -e` in both scripts** — Removed from `container-setup.sh` and `quickstart.sh`; transient failures (curl timeouts, docker commands) no longer kill the container
- **`docker compose up -d` errors silently swallowed** — Replaced `2>/dev/null || true` with captured output that shows errors on failure
- **`create-matrix-admin.sh` broken API path** — Rewrote (v2.0.0) to use Conduit's Synapse-compatible shared-secret registration API instead of admin token/CLI dependency
- **Redundant setup flag** — Removed duplicate `touch $SETUP_FLAG` from `container-setup.sh`; `quickstart.sh` already handles it

### Added
- **Shared-secret user registration** — `register_matrix_user()` function uses Conduit's `/_synapse/admin/v1/register` endpoint with HMAC-SHA1 nonce authentication
- **Admin user creation in setup flow** — `configure_admin_user()` prompts for admin credentials; `register_users()` creates both bridge bot and admin user on Conduit
- **Conduit health check loop** — `wait_for_conduit()` polls for up to 120s with progress dots, handles first-run image pulls
- **Admin credentials in final summary** — Setup complete screen now shows Element X / ArmorChat login credentials with homeserver URL
- **Security: shared secret auto-cleanup** — Registration shared secret is injected into conduit.toml for user creation, then removed and Conduit restarted

### Changed
- **`create-matrix-admin.sh`** — Rewritten from v1.0.0 to v2.0.0; now uses shared-secret registration instead of admin token/CLI
- **`final_summary()`** — Now shows admin credentials, homeserver URL, and explicit Element X / ArmorChat login steps
- **`configs/conduit.toml`** — Added documented `registration_shared_secret` placeholder with usage instructions

---

## [0.3.0] - 2026-02-24 - Docker Deployment Hardening (5-Pass Review)

### Fixed
- **LD_PRELOAD security hook** — `security_hook.c` now checks `addr->sa_family` and allows `AF_UNIX` through, preventing bridge socket communication from being silently blocked
- **CGO_ENABLED=0 panic** — Bridge Dockerfile stages now build with `CGO_ENABLED=1` and include `gcc`, `libc-dev`, `libsqlite3-dev` for go-sqlcipher compatibility
- **Dockerfile.openclaw rewrite** — Removed invalid multi-stage references (`COPY --from=openclaw-src`), consolidated to working image
- **Coturn shell expansion** — Replaced YAML command array with shell form entrypoint for runtime expression evaluation
- **OpenClaw healthcheck** — Switched from `.sh` file with Python shebang to inline `CMD ["python3", "-c", "..."]`
- **RPC method mismatch** — Setup scripts used `keystore.add_provider` but bridge only handles `store_key`; fixed method name and added missing `id` param
- **Docker Compose V2 plugin path** — Installed to `/usr/lib/docker/cli-plugins/` instead of invalid symlink with space in name
- **ARMORCLAW_CONFIG env var** — Implemented in `parseFlags()` so documented env var actually works
- **Dockerfile COPY of non-existent file** — Removed `COPY deploy/setup-wizard.sh` that referenced a file that doesn't exist at that path
- **Docker bind mount host path resolution** — Parameterized compose paths with `${ARMORCLAW_CONFIGS:-./configs}` and added host-copy helper
- **External curl calls with no timeout** — All `curl` calls in setup scripts now include `--connect-timeout 5`
- **Socket path string slicing** — Replaced hardcoded length arithmetic with `filepath.Dir()`
- **Missing libsqlite3-0** — Added to runtime stage so bridge binary can dynamically link against SQLite
- **Provider detection lost** — `API_PROVIDER` now persisted to temp file and used in dynamic RPC calls instead of hardcoded `"openai"`
- **MATRIX_SERVER_NAME not exported** — Now exported before `docker compose up` so Conduit gets the user's actual server name
- **TURN_SECRET not generated** — Now generated via `openssl rand -hex 32` instead of using the placeholder default
- **API key shell expansion** — Switched from unquoted heredoc to `printf '%s'` to prevent `$` in keys from being interpreted
- **Interactive prompts exit on invalid input** — All required prompts (Matrix server, provider selection, API key, custom URL) now use retry loops with `print_warning` instead of `exit 1`

### Changed
- **nginx** — Added `profiles: ["frontend"]` so it only starts when explicitly requested
- **Setup wizard UX** — Invalid input re-prompts instead of killing the container; no single typo forces full restart

### Documentation
- Created `doc/LESSONS_LEARNED.md` — 18 lessons from all 5 review passes
- Moved `DOCKER-HUB-HOSTINGER-DEPLOYMENT.md` and `GITHUB-ACTIONS-DOCKERHUB-SETUP.md` to `doc/`

---

## [0.2.0] - 2026-02-19 - Phase 5: Zero-Trust Hardening

### Added
- **Zero-Trust System** (`bridge/pkg/trust/`)
  - `zero_trust.go` - Core trust verification engine with device fingerprinting
  - `device.go` - Device identification and tracking
  - `middleware.go` - Operation-level enforcement with configurable policies
  - `zero_trust_test.go` - 15 tests for trust verification

- **Audit System** (`bridge/pkg/audit/`)
  - `tamper_evident.go` - Hash-chain audit logging with integrity verification
  - `compliance.go` - 90-day retention, JSON/CSV export, compliance reports
  - `audit_helper.go` - CriticalOperationLogger for centralized logging
  - Tests: 28 tests for audit functionality

- **Trust Integration**
  - `bridge/internal/adapter/trust_integration.go` - Matrix adapter integration
  - Event verification in Matrix adapter (`matrix.go`)
  - Trust middleware integration in RPC server (`server.go`)

### Security Features
- Trust score levels (Untrusted → Verified, 0-4)
- Default enforcement policies for 8 operation types
- Anomaly detection (IP changes, impossible travel, sensitive access)
- Session lockout after failed verification attempts
- Audit logging for container, key, secret, config, auth operations

### Commits
```
da6b415 docs: Update review.md to 0.2.0 with Phase 5 completion
41a0479 feat(security): Add core trust and audit packages
2b81bad feat(security): Complete Phase 5 audit and zero-trust integration
```

---

## [0.1.5] - 2026-02-18 - Step 4: Push Notification Gateway

### Added
- **Push Gateway** (`bridge/pkg/push/`)
  - `gateway.go` - Multi-platform notification gateway
  - `providers.go` - FCM, APNS, WebPush implementations
  - `sygnal.go` - Matrix Sygnal client
  - `push_test.go` - 15 tests

- **Client Applications** (`applications/`)
  - ArmorChat - Android Matrix client with E2EE
  - ArmorTerminal - Terminal pairing application
  - Admin Panel - React/TypeScript web dashboard
  - Setup Wizard - React/TypeScript setup flow

### Commits
```
6c15fd6 feat: Add client applications and update documentation
```

---

## [0.1.4] - 2026-02-18 - Step 3: Enterprise Enforcement Layer

### Added
- **Enforcement System** (`bridge/pkg/enforcement/`)
  - `enforcement.go` - License-based feature access control
  - `middleware.go` - Request interception and validation
  - `rpc_handlers.go` - RPC methods for enforcement
  - `bridge_integration.go` - AppService hooks
  - Tests: 10 tests

### Commits
```
ca6699c feat: Add adapters, configs, tests, and deployment scripts
```

---

## [0.1.3] - 2026-02-18 - Step 2: Bridge AppService Implementation

### Added
- **AppService** (`bridge/pkg/appservice/`)
  - `appservice.go` - HTTP server for Matrix transactions
  - `bridge.go` - BridgeManager for SDTW adapter coordination
  - `client.go` - Matrix client wrapper
  - Ghost user namespaces (@slack_*, @discord_*, @teams_*, @whatsapp_*)
  - Tests: 16 tests

### Commits
```
ca6699c feat: Add adapters, configs, tests, and deployment scripts
```

---

## [0.1.2] - 2026-02-18 - Step 1: Matrix Infrastructure

### Added
- **Deployment Configs** (`configs/`)
  - `synapse/homeserver.yaml` - Synapse configuration
  - `coturn/turnserver.conf` - TURN/STUN for WebRTC
  - `nginx/matrix.conf` - Reverse proxy with TLS
  - `postgres/postgresql.conf` - Database optimization
  - `appservices/bridge-registration.yaml` - AppService registration

- **Deployment Scripts** (`deploy/matrix/`)
  - `deploy-matrix.sh` - Automated Matrix deployment
  - `docker-compose.matrix.yml` - Production compose

### Commits
```
ca6699c feat: Add adapters, configs, tests, and deployment scripts
```

---

## [0.1.1] - 2026-02-18 - Critical Bug Fixes

### Fixed
- **LLM Response PHI Scrubbing** - Tier-dependent compliance for outbound responses
  - `bridge/pkg/pii/llm_compliance.go` - LLM response compliance handler
  - `bridge/pkg/pii/errors.go` - Structured compliance error types

- **License Activation Race Condition** - Transaction-based activation
  - `license-server/main.go` - SELECT FOR UPDATE pattern

- **Budget Tracker Persistence** - WAL-based durability
  - `bridge/pkg/budget/persistence.go` - Write-ahead logging
  - `bridge/pkg/budget/tracker.go` - Fsync on critical writes

- **Quarantine Notifications** - Callback support
  - `bridge/pkg/pii/hipaa.go` - QuarantineNotifier callback

- **Code Quality** - Atomic operations, structured errors
  - Replaced sync.RWMutex with atomic operations
  - Added component context to all logs

### Commits
```
2205375 chore: Update core packages with Phase 4/5 changes
```

---

## [3.0.0] - 2026-02-17 - Phase 4: Enterprise Features

### Added
- **License Server** (`license-server/`)
  - PostgreSQL-backed license validation
  - Rate limiting, grace periods, machine binding
  - Tests: 15 tests

- **HIPAA Compliance** (`bridge/pkg/pii/`)
  - `hipaa.go` - PHI detection (SSN, CC, MRN patterns)
  - Severity-based action routing
  - Tests: 15 tests

- **SSO Integration** (`bridge/pkg/sso/`)
  - `sso.go` - SAML 2.0 and OIDC authentication
  - Role mapping from attributes
  - Tests: 13 tests

- **Web Dashboard** (`bridge/pkg/dashboard/`)
  - `dashboard.go` - Embedded HTTP server
  - Static HTML templates
  - Tests: 19 tests

### Commits
```
c6e7efa feat(enterprise): Add Phase 4 enterprise packages
```

---

## [2.0.0] - 2026-02-15 - Sprint 2: All Gaps Resolved

### Added
- **Error Handling System** (`bridge/pkg/errors/`)
  - Structured error codes (CTX-XXX, MAT-XXX, RPC-XXX, SYS-XXX, BGT-XXX, VOX-XXX)
  - Component-scoped event tracking with ring buffers
  - SQLite persistence
  - 3-tier admin resolution chain
  - RPC methods: `get_errors`, `resolve_error`

- **Documentation** (11 GAPs resolved)
  - Getting Started guide (GAP #1)
  - Platform deployment guides (GAP #2)
  - API key validation (GAP #3)
  - QR scanning flow (GAP #4)
  - Multi-device UX (GAP #5)
  - Account recovery (GAP #6)
  - Error escalation (GAP #7)
  - Platform onboarding (GAP #8)
  - Slack adapter (GAP #9)
  - Alert integration (GAP #10)
  - Security tier upgrade (GAP #11)

### Commits
```
3c0493e docs: Update progress.md with Sprint 2 Complete milestone
b7781d8 docs: Update review.md to v2.0.0 - All Gaps Resolved
519d8fd docs: Add security tier upgrade guide (GAP #11 resolved)
fd2fb5c docs: Add QR scanning flow guide (GAP #4 resolved)
a9b3780 docs: Add API key validation guide (GAP #3 resolved)
ba9bbbd docs: Add multi-device UX guide (GAP #5 resolved)
62aaca8 docs: Add alert integration guide (GAP #10 resolved)
e0fc171 docs: Add Getting Started guide (GAP #1 resolved)
5b07d54 docs: Update user journey gap analysis (v2.0.0)
c8cee3c docs: Add error handling system documentation
dde4b7f feat(errors): integrate error system with Matrix adapter
2d1cbb4 feat(errors): integrate error system with docker client
a0c7023 feat: Add get_errors and resolve_error RPC methods
a00b577 feat: Integrate error handling system with bridge main
a183a63 feat(errors): Add package initialization and documentation
d9a1a35 feat(errors): Add notification sender
7727492 feat(errors): Add error persistence store
3051240 feat(errors): Add admin resolution chain
ec1d940 feat(errors): Add smart sampling registry
203afd0 feat(errors): Add component-scoped event tracking
aac1d4e feat(errors): Add core error types and builder
d2f3542 docs: Add error handling system design
```

---

## [1.5.0] - 2026-02-12 - P0 Critical Fixes

### Fixed
- **P0-CRIT-3**: Memory-only Unix socket injection
  - `bridge/pkg/secrets/socket.go` - SecretInjector with Unix domain sockets
  - Secrets never written to disk, transmitted via socket only
  - RPC method: `send_secret`

- **P0-CRIT-1**: Egress proxy for SDTW adapters
  - Outbound traffic through proxy only
  - No direct container internet access

### Commits
```
9f2b13d feat: Add Agent.SendSecret RPC method (P0-CRIT-3)
458e80c feat: Implement memory-only Unix socket injection (P0-CRIT-3)
db86c8e feat: Implement egress proxy for SDTW adapters (P0-CRIT-1)
```

---

## [1.4.0] - 2026-02-11 - SDTW Message Queue

### Added
- **Message Queue** (`bridge/internal/queue/`)
  - SQLite-based persistent queue with WAL
  - Retry logic with exponential backoff
  - Dead letter queue (DLQ) support
  - Circuit breaker pattern

### Commits
```
c2132e0 Add production-ready features to SDTW message queue
3dfc6cc Implement full SQLite message queue with retry logic and DLQ support
441f177 Fix SDTW queue package compilation errors
9fe233c Add SDTW message queue stub implementation
dcad27d Add SDTW Message Queue specification and tasks
```

---

## [1.3.0] - 2026-02-08 - WebRTC Voice Implementation

### Added
- **WebRTC Engine** (`bridge/pkg/webrtc/`)
  - `engine.go` - Pion-based WebRTC implementation
  - `session.go` - Session management
  - `token.go` - Token-based authorization

- **TURN Server** (`bridge/pkg/turn/`)
  - `turn.go` - Coturn integration
  - NAT traversal support

### Commits
```
ec813c4 fix(voice): Resolve structural issues in voice package
```

---

## [1.2.0] - 2026-02-07 - Security Enhancements

### Added
- **Zero-Trust Middleware** - Trusted senders/rooms validation
- **Budget Guardrails** - Token-aware budget tracking
- **Container TTL** - Auto-cleanup with heartbeat

### Commits
```
[Historical commits for security enhancements]
```

---

## [1.1.0] - 2026-02-06 - Setup Wizard

### Added
- **Interactive Setup** (`deploy/setup-wizard.sh`)
  - 10-step guided installation
  - System requirements validation
  - Docker installation/verification
  - Keystore initialization

---

## [1.0.0] - 2026-02-05 - Phase 1 Complete

### Core Features
- **Encrypted Keystore** - SQLCipher + XChaCha20-Poly1305
- **Docker Client** - Scoped operations (create, exec, remove)
- **Matrix Adapter** - E2EE support with Conduit/Synapse
- **JSON-RPC Server** - 11 core methods
- **Configuration System** - TOML + environment variables
- **Container Entrypoint** - Secrets validation, fail-fast

### Build Artifacts
- Bridge binary: 31 MB
- Container image: 393 MB (98.2 MB compressed)

---

## Historical Plans (Superseded)

The following planning documents have been superseded by implementation:

| Document | Status | Superseded By |
|----------|--------|---------------|
| `2026-02-05-minimal-bridge-spec.md` | Implemented | v1.0.0 |
| `2026-02-05-robust-bridge-spec.md` | Implemented | v1.0.0 |
| `2026-02-05-secureclaw-v1-design.md` | Renamed | ArmorClaw |
| `2026-02-05-local-bridge-matrix-gateway.md` | Implemented | v1.2.0 |
| `2026-02-05-license-server-api.md` | Implemented | v3.0.0 |
| `2026-02-07-swarmclaw-design.md` | Deferred | Future |
| `2026-02-08-armorclaw-renaming-plan.md` | Complete | N/A |
| `2026-02-08-webrtc-voice-implementation.md` | Implemented | v1.3.0 |
| `2026-02-10-armorclaw-fix-plan.md` | Complete | v2.0.0 |
| `2026-02-11-missing-rpc-methods-implementation.md` | Complete | v2.0.0 |
| `2026-02-15-error-handling-system-design.md` | Implemented | v2.0.0 |
| `2026-02-15-error-handling-implementation-tasks.md` | Complete | v2.0.0 |
| `2026-02-16-first-boot-security-configuration.md` | Implemented | v3.2.0 |
| `SDTW_Adapter_Implementation_Plan_v2.0.md` | In Progress | - |
| `SDTW_MessageQueue_Specification.md` | Implemented | v1.4.0 |

---

**Changelog Last Updated:** 2026-02-19
