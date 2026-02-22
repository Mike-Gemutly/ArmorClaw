# ArmorClaw Documentation

> **Version:** 0.2.0 | **Last Updated:** 2026-02-22 | **Status:** Production Testing

---

## Quick Start

| What | Link | Time |
|------|------|------|
| **Quick Setup** ⚡ | [Express Installation](guides/setup-guide.md#method-1-quick-setup--recommended-for-beginners) | 2 min |
| **Docker Quick Start** | [Docker Image Guide](guides/quickstart-docker.md) | 2 min |
| First VPS deployment? | [First Deployment Checklist](guides/first-deployment-checklist.md) ⭐ | 30 min |
| New to ArmorClaw? | [Getting Started Guide](guides/getting-started.md) | 5 min |
| Deploy to production | [Hostinger VPS Deployment](guides/hostinger-vps-deployment.md) | 15 min |

## Post-Setup Scripts

| Script | Purpose | Command |
|--------|---------|---------|
| **Quick Setup** | 2-minute express installation | `sudo ./deploy/setup-quick.sh` |
| **Setup Wizard** | Mode-based setup (Quick/Standard/Expert) | `sudo ./deploy/setup-wizard.sh` |
| **Matrix Setup** | Enable Matrix communication | `sudo ./deploy/setup-matrix.sh` |
| **Production Hardening** | Firewall, SSH, Fail2Ban, logging | `sudo ./deploy/armorclaw-harden.sh` |
| **Device Provisioning** | Generate QR codes for ArmorChat | `sudo ./deploy/armorclaw-provision.sh` |

## Additional Resources

| What | Link | Time |
|------|------|------|
| Setup client discovery | [Discovery Deployment Guide](guides/discovery-deployment.md) | 10 min |
| Secure provisioning | [Provisioning Protocol](plans/2026-02-21-secure-provisioning-protocol.md) | 10 min |
| Security gap analysis | [Security Gap Analysis](plans/2026-02-21-security-gap-analysis.md) | - |
| Connect via Element X | [Element X Quickstart](guides/element-x-quickstart.md) | 5 min |
| Connect ArmorChat | [ArmorChat Documentation](output/ArmorChat.md) | 10 min |
| Connect ArmorTerminal | [ArmorTerminal Documentation](output/ArmorTerminal.md) | 10 min |
| Run OpenClaw inside | [OpenClaw Integration](guides/openclaw-integration.md) | 10 min |
| Troubleshoot issues | [Error Catalog](guides/error-catalog.md) | - |

---

## Feature Directory

### Core Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **Encrypted Keystore** | SQLCipher + XChaCha20-Poly1305 credential storage | [Config Guide](guides/configuration.md) | `bridge/pkg/keystore` |
| **Docker Client** | Scoped container operations (create, exec, remove) | [Development Guide](guides/development.md) | `bridge/pkg/docker` |
| **Matrix Adapter** | E2EE-capable Matrix bridge | [Matrix Deployment](guides/matrix-homeserver-deployment.md) | `bridge/internal/adapter` |
| **JSON-RPC Server** | Unix socket API with 24 methods | [RPC API Reference](reference/rpc-api.md) | `bridge/pkg/rpc` |
| **Configuration System** | TOML + environment variables | [Configuration Guide](guides/configuration.md) | `bridge/pkg/config` |
| **Secret Injection** | Memory-only, never on disk | [Security Config](guides/security-configuration.md) | `bridge/pkg/secrets` |

### Security Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **Zero-Trust Verification** | Device fingerprinting, trust scoring | [Security Config](guides/security-configuration.md) | `bridge/pkg/trust` |
| **Audit Logging** | Tamper-evident hash-chain logs | [Security Config](guides/security-configuration.md) | `bridge/pkg/audit` |
| **HIPAA Compliance** | PHI detection and scrubbing | [Security Config](guides/security-configuration.md) | `bridge/pkg/pii` |
| **Media PHI Scanner** | OCR-based PHI detection in images/PDFs | [Security Config](guides/security-configuration.md) | `bridge/pkg/pii` |
| **Budget Guardrails** | Token tracking, cost controls, workflow states | [Security Config](guides/security-configuration.md) | `bridge/pkg/budget` |
| **Security Tiers** | Essential → Enhanced → Maximum | [Tier Upgrade Guide](guides/security-tier-upgrade.md) | `bridge/pkg/lockdown` |
| **Bridge Security Warnings** | E2EE downgrade detection and alerts | [Security Config](guides/security-configuration.md) | `ui/components/BridgeSecurityWarning.kt` |
| **Resource Governor** | CPU/memory limits, violation monitoring | [Security Config](guides/security-configuration.md) | `bridge/pkg/docker` |
| **Secure Provisioning** | QR-based device setup with signed config | [Provisioning Protocol](plans/2026-02-21-secure-provisioning-protocol.md) | `bridge/pkg/provisioning` |

### Communication Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **WebRTC Voice** | Matrix-to-Matrix voice/video with TURN relay | [Voice Guide](guides/webrtc-voice-guide.md) | `bridge/pkg/webrtc` |
| **WebSocket Client** | Real-time Matrix event push | [WebSocket Guide](guides/websocket-client-guide.md) | `bridge/pkg/websocket` |
| **Push Notifications** | FCM, APNS, WebPush via Sygnal | - | `bridge/pkg/push` |
| **SDTW Adapters** | Slack, Discord, Teams, WhatsApp bridges with reactions, edits, deletes | - | `bridge/internal/sdtw` |

> **Note:** Voice/video calls work only between Matrix users. Cross-platform voice bridging (e.g., Slack Huddles ↔ Matrix) is not currently supported.

### Mobile Client (ArmorChat Android)

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **E2EE Support** | Matrix SDK crypto with bridge verification | - | `applications/ArmorChat/app/src/main/java/app/armorclaw` |
| **Key Backup** | SSSS recovery passphrase setup/recovery | - | `ui/security/KeyBackupScreen.kt` |
| **Push Manager** | Native Matrix HTTP Pusher with FCM | - | `push/MatrixPusherManager.kt` |
| **Identity** | Namespace-aware user management | - | `data/repository/UserRepository.kt` |
| **Feature Suppression** | Capability-aware UI for bridged rooms | - | `ui/components/MessageActions.kt` |
| **Migration** | v2.5 → v4.6 upgrade flow | - | `ui/migration/MigrationScreen.kt` |
| **System Alerts** | Distinct UI for bridge alerts (budget, license, security) | - | `ui/components/SystemAlertMessage.kt` |
| **Bridge Verification** | Emoji verification flow for E2EE trust | - | `ui/verification/BridgeVerificationScreen.kt` |
| **Bridge Security Warning** | E2EE downgrade alerts for bridged rooms | - | `ui/components/BridgeSecurityWarning.kt` |
| **Context Transfer Warning** | Cost estimation before agent context transfer | - | `ui/components/ContextTransferDialog.kt` |

### Enterprise Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **License Server** | PostgreSQL-backed license validation | - | `license-server/` |
| **License State Manager** | Runtime polling, grace period handling | - | `bridge/pkg/license` |
| **SSO Integration** | SAML 2.0 and OIDC authentication | - | `bridge/pkg/sso` |
| **Web Dashboard** | Embedded management interface | - | `bridge/pkg/dashboard` |
| **Error Handling** | Structured codes, tracking, alerting | [Error Catalog](guides/error-catalog.md) | `bridge/pkg/errors` |
| **Recovery System** | BIP39 phrase, 48-hour window | [Multi-Device UX](guides/multi-device-ux.md) | `bridge/pkg/recovery` |
| **Ghost User Manager** | Lifecycle management for bridged users | - | `bridge/pkg/ghost` |
| **OpenClaw Integration** | Run OpenClaw AI assistant in hardened container | [OpenClaw Integration](guides/openclaw-integration.md) | `container/openclaw/` |

---

## Feature Details

### Keystore (`bridge/pkg/keystore`)

**Purpose:** Encrypted credential storage with hardware-bound keys

**Key Files:**
- `keystore.go` - Core keystore implementation
- `keystore.go` (internal) - SQLCipher database operations

**Critical Functions:**
| Function | Purpose |
|----------|---------|
| `NewKeystore()` | Initialize encrypted database |
| `Store(cred)` | Store encrypted credential |
| `Retrieve(id)` | Retrieve and decrypt credential |
| `Delete(id)` | Securely delete credential |
| `List()` | List all stored credential IDs |

**Dependencies:**
- `github.com/mutecomm/go-sqlcipher/v4` - SQLCipher driver
- Hardware identifiers for key binding

**Configuration:**
```toml
[keystore]
path = "~/.armorclaw/keystore.db"
```

---

### Crypto Store (`bridge/pkg/crypto`)

**Purpose:** Persistent encrypted storage for E2EE Megolm session keys

**Key Files:**
| File | Purpose |
|------|---------|
| `store.go` | Crypto store interface definition |
| `keystore_store.go` | SQLCipher-backed implementation |

**Critical Functions:**
| Function | Purpose |
|----------|---------|
| `NewKeystoreBackedStoreWithDB(db)` | Create store from existing keystore DB |
| `AddInboundGroupSession()` | Store Megolm session key |
| `GetInboundGroupSession()` | Retrieve session for decryption |
| `HasInboundGroupSession()` | Check if session exists |
| `DeleteSessionsForRoom()` | Clear sessions for a room |

**Integration:**
- Uses same SQLCipher database as keystore
- Keys encrypted at rest with hardware-bound master key
- Enables bridge to decrypt historical messages after restart

---

### System Alerts (`bridge/pkg/notification` + Android)

**Purpose:** Distinct UI treatment for critical bridge alerts

**Key Files:**
| File | Platform | Purpose |
|------|----------|---------|
| `notification/alert_types.go` | Go | Alert manager and sender |
| `data/model/SystemAlert.kt` | Kotlin | Alert types and content model |
| `ui/components/SystemAlertMessage.kt` | Kotlin | Alert card and banner UI |

**Alert Types:**
| Category | Types |
|----------|-------|
| Budget | BUDGET_WARNING, BUDGET_EXCEEDED |
| License | LICENSE_EXPIRING, LICENSE_EXPIRED, LICENSE_INVALID |
| Security | SECURITY_EVENT, TRUST_DEGRADED, VERIFICATION_REQUIRED |
| System | BRIDGE_ERROR, BRIDGE_RESTARTING, MAINTENANCE |

**Event Structure:**
```json
{
  "type": "app.armorclaw.alert",
  "content": {
    "alert_type": "BUDGET_WARNING",
    "severity": "WARNING",
    "title": "Budget Warning",
    "message": "Token usage is at 80%...",
    "action": "View Usage",
    "action_url": "armorclaw://dashboard/budget"
  }
}
```

---

### Zero-Trust System (`bridge/pkg/trust`)

**Purpose:** Continuous verification with device fingerprinting

**Key Files:**
| File | Purpose |
|------|---------|
| `zero_trust.go` | ZeroTrustManager - core verification engine |
| `device.go` | Device fingerprinting and tracking |
| `middleware.go` | Operation-level enforcement |

**Critical Types:**
```go
type TrustScore int  // 0=Untrusted, 1=Low, 2=Medium, 3=High, 4=Verified

type ZeroTrustRequest struct {
    SessionID         string
    UserID            string
    DeviceFingerprint DeviceFingerprintInput
    IPAddress         string
    Action            string
    Resource          string
}

type ZeroTrustResult struct {
    Passed          bool
    TrustLevel      TrustScore
    RiskScore       int      // 0-100
    AnomalyFlags    []string
    RequiredActions []string
}
```

**Default Policies:**
| Operation | Min Trust | Max Risk | MFA | Verified Device |
|-----------|-----------|----------|-----|-----------------|
| container_create | Medium (2) | 40 | No | No |
| secret_access | High (3) | 25 | Yes | Yes |
| admin_access | Verified (4) | 15 | Yes | Yes |

---

### Audit System (`bridge/pkg/audit`)

**Purpose:** Tamper-evident logging with compliance reporting

**Key Files:**
| File | Purpose |
|------|---------|
| `tamper_evident.go` | Hash-chain audit log |
| `compliance.go` | 90-day retention, exports |
| `audit_helper.go` | CriticalOperationLogger |

**Critical Functions:**
```go
// Log an audit entry
func (l *TamperEvidentLog) LogEntry(eventType string, actor Actor, action string,
    resource Resource, details map[string]interface{}, compliance ComplianceFlags) (*AuditEntry, error)

// Verify chain integrity
func (l *TamperEvidentLog) VerifyChain() ([]int, error)

// Export for compliance
func (l *TamperEvidentLog) ExportJSON(start, end time.Time) ([]byte, error)
```

**Audit Categories:**
| Category | Retention | Example Events |
|----------|-----------|----------------|
| container_lifecycle | 90 days | start, stop, error |
| key_access | 90 days | access, create, delete |
| secret_management | 90 days | injection, cleanup |
| phi_access | 6 years | read, write (HIPAA) |

---

### WebRTC/Voice (`bridge/pkg/webrtc`)

**Purpose:** Real-time voice communication with NAT traversal

**Key Files:**
| File | Purpose |
|------|---------|
| `engine.go` | Pion-based WebRTC engine |
| `session.go` | Session management |
| `token.go` | Authorization tokens |

**Dependencies:**
- `github.com/pion/webrtc/v3` - WebRTC implementation
- TURN server (Coturn) for NAT traversal

**Configuration:**
```toml
[webrtc]
enabled = true
stun_servers = ["stun:stun.l.google.com:19302"]
turn_server = "turn:your-server.com:3478"
```

---

### Budget-Aware Workflows (`bridge/pkg/budget`)

**Purpose:** Token budget tracking with workflow state management

**Key Files:**
| File | Purpose |
|------|---------|
| `tracker.go` | Core budget tracker with workflow states |
| `persistence.go` | WAL-based durability for crash recovery |

**Workflow States:**
| State | Description | User Action |
|-------|-------------|-------------|
| `RUNNING` | Active, processing requests | None required |
| `PAUSED` | User-initiated pause | Resume button enabled |
| `PAUSED_INSUFFICIENT_FUNDS` | Budget exhaustion | Resume disabled, top-up required |
| `COMPLETED` | Successfully finished | None |
| `FAILED` | Terminated with error | Retry or contact support |

**Critical Functions:**
```go
// Get current workflow state
func (b *BudgetTracker) GetWorkflowState() WorkflowState

// Check if workflow can resume
func (b *BudgetTracker) CanResumeWorkflow() error

// Record token usage
func (b *BudgetTracker) RecordUsage(record UsageRecord) error

// Check if new session can start
func (b *BudgetTracker) CanStartSession() error
```

**Configuration:**
```toml
[budget]
daily_limit_usd = 10.0
monthly_limit_usd = 100.0
alert_threshold = 80.0  # Alert at 80% of limit
hard_stop = true        # Block requests when exceeded
```

---

### Bridge Security Warnings (`ui/components/BridgeSecurityWarning.kt`)

**Purpose:** Alert users when E2EE rooms are bridged to non-E2EE platforms

**Key Components:**
| Component | Purpose |
|-----------|---------|
| `BridgeSecurityWarningBanner` | In-room warning card |
| `BridgeSecurityIndicator` | Room list badge |
| `BridgeSecurityInfoDialog` | Full explanation modal |
| `PreJoinBridgeSecurityWarning` | Pre-join consent screen |

**Security Levels:**
| Level | Description | UI Treatment |
|-------|-------------|--------------|
| `NATIVE_E2EE` | Native Matrix E2EE | Lock icon |
| `BRIDGED_SECURE` | Bridged to E2EE platform (WhatsApp, Signal) | Lock icon |
| `BRIDGED_INSECURE` | Bridged to non-E2EE (Slack, Discord, Teams) | Red warning banner |
| `UNKNOWN` | Status unknown | Gray indicator |

**Platform E2EE Support:**
| Platform | E2EE | Notes |
|----------|------|-------|
| Matrix | ✅ | Native Megolm |
| Signal | ✅ | Signal Protocol |
| WhatsApp | ✅ | Signal Protocol |
| Slack | ❌ | Enterprise compliance |
| Discord | ❌ | No encryption API |
| Teams | ❌ | Enterprise compliance |

**Integration:**
- Emits `BRIDGE_SECURITY_DOWNGRADE` system alert when E2EE room bridges to non-E2EE platform
- Requires explicit user consent before joining insecurely-bridged encrypted rooms

---

### Ghost User Manager (`bridge/pkg/ghost`)

**Purpose:** Lifecycle management for Matrix ghost users representing external platform users

**Key Files:**
| File | Purpose |
|------|---------|
| `manager.go` | GhostUserManager - lifecycle orchestrator |

**Event Types:**
| Event | Trigger | Action |
|-------|---------|--------|
| `USER_JOINED` | User joins external platform | Create/activate Ghost User |
| `USER_LEFT` | User leaves external platform | Deactivate, append "[Left Platform]" |
| `USER_DELETED` | User account deleted | Same as LEFT (preserve history) |
| `USER_UPDATED` | Profile updated | Update display name |

**Critical Functions:**
```go
// Handle user lifecycle events
func (m *Manager) HandleUserEvent(ctx context.Context, event UserEvent) error

// Sync platform roster against Matrix ghost users
func (m *Manager) SyncPlatform(ctx context.Context, platform string) error

// Start periodic sync (default: 24 hours)
func (m *Manager) StartSync()
```

**Retention Policy:**
- Historical messages preserved (never redacted)
- Display name updated with `[Left Platform]` suffix
- Future logins prevented via Matrix deactivation

---

### License State Manager (`bridge/pkg/license`)

**Purpose:** Runtime license state management with grace period handling

**Key Files:**
| File | Purpose |
|------|---------|
| `state_manager.go` | LicenseStateManager - state machine + polling |

**License States:**
| State | Behavior | Description |
|-------|----------|-------------|
| `VALID` | Normal | All operations allowed |
| `DEGRADED` | Limited | Warning state, admin ops blocked |
| `GRACE_PERIOD` | Degraded/ReadOnly | 7-day grace after expiry |
| `EXPIRED` | Blocked/ReadOnly | Grace period ended |
| `INVALID` | Blocked | License malformed or revoked |

**Critical Functions:**
```go
// Initialize on boot - returns license state
func (m *StateManager) Initialize(ctx context.Context) (*LicenseInfo, error)

// Check if operation is allowed
func (m *StateManager) CanPerformOperation(op Operation) (bool, string)

// Get current behavior
func (m *StateManager) GetBehavior() RuntimeBehavior

// Start runtime polling (default: 24h interval)
func (m *StateManager) StartPolling()
```

**Configuration:**
```toml
[license]
grace_period_days = 7
poll_interval_hours = 24
alert_thresholds = [30, 14, 7, 1]  # Days before expiry
block_on_expired = true
read_only_on_grace = false
```

**Boot Sequence:**
- Step 10: License State Check
- `VALID` → Start normally
- `GRACE_PERIOD` → Start + send CRITICAL alert
- `EXPIRED` → Block RPC, show dashboard error

---

### Context Transfer Quota (`ui/components/ContextTransferDialog.kt`)

**Purpose:** Cost estimation and warnings before agent context transfers

**Key Components:**
| Component | Purpose |
|-----------|---------|
| `ContextTransferWarningDialog` | Main warning dialog |
| `ContextTransferEstimate` | Cost estimate data class |
| `RiskLevel` | LOW/MEDIUM/HIGH/CRITICAL enum |

**Risk Levels:**
| Level | Budget Remaining | UI Treatment |
|-------|-----------------|--------------|
| LOW | > 80% | Green indicator |
| MEDIUM | 20-80% | Yellow indicator |
| HIGH | < 20% | Orange, emphasize warning |
| CRITICAL | Would exhaust | Red, transfer blocked |

**Token Estimation:**
| Content Type | Multiplier | Formula |
|--------------|------------|---------|
| Text | 1.0 | `chars / 4` |
| Code | 1.2 | `chars / 3.5` |
| PDF | 2.0 | `bytes / 2` |
| File | 1.5 | `bytes / 4` |

---

### Error Handling (`bridge/pkg/errors`)

**Purpose:** Structured error tracking with admin alerting

**Error Codes:**
| Prefix | Category | Example |
|--------|----------|---------|
| CTX-XXX | Container | CTX-003: Health timeout |
| MAT-XXX | Matrix | MAT-001: Connection failed |
| RPC-XXX | RPC/API | RPC-010: Socket failed |
| SYS-XXX | System | SYS-010: Secret inject failed |
| BGT-XXX | Budget | BGT-002: Budget exceeded |
| VOX-XXX | Voice | VOX-001: WebRTC failed |

**RPC Methods:**
- `get_errors` - Query error history
- `resolve_error` - Mark error as resolved

---

## Deployment Options

### Recommended (Budget-Friendly)
| Platform | Cost | Guide |
|----------|------|-------|
| Hostinger VPS | $4-8/mo | [Deployment Guide](guides/hostinger-vps-deployment.md) |
| Vultr | $2.50/mo+ | [Deployment Guide](guides/vultr-deployment.md) |
| DigitalOcean | $5/mo+ | [Deployment Guide](guides/digitalocean-deployment.md) |

### Enterprise
| Platform | Features | Guide |
|----------|----------|-------|
| Fly.io | Global edge (35+ regions) | [Deployment Guide](guides/flyio-deployment.md) |
| AWS Fargate | Enterprise serverless | [Deployment Guide](guides/aws-fargate-deployment.md) |
| GCP Cloud Run | Free tier available | [Deployment Guide](guides/gcp-cloudrun-deployment.md) |

### Development
| Platform | Purpose | Guide |
|----------|---------|-------|
| Docker Desktop | Local development | [Local Dev Guide](guides/local-development.md) |
| Railway | Quick prototyping | [Deployment Guide](guides/railway-deployment.md) |
| Render | Free tier testing | [Deployment Guide](guides/render-deployment.md) |

---

## Reference Documentation

| Document | Purpose |
|----------|---------|
| [RPC API Reference](reference/rpc-api.md) | Complete JSON-RPC 2.0 API (24 methods) |
| [Error Catalog](guides/error-catalog.md) | Every error with solutions |
| [Security Configuration](guides/security-configuration.md) | Zero-trust, budget, PII |
| [Troubleshooting Guide](guides/troubleshooting.md) | Systematic debugging |
| [Configuration Guide](guides/configuration.md) | TOML config and env vars |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      ARMORCLAW ARCHITECTURE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   External Platforms          Bridge Components                  │
│   ┌─────────────┐            ┌─────────────────────┐           │
│   │    Slack    │───────────▶│   SDTW Adapters     │           │
│   │   Discord   │            │   (Slack/Discord/   │           │
│   │    Teams    │            │    Teams/WhatsApp)  │           │
│   │  WhatsApp   │            └──────────┬──────────┘           │
│   └─────────────┘                       │                       │
│                                         ▼                       │
│                              ┌─────────────────────┐           │
│                              │   Message Queue     │           │
│                              │   (SQLite + WAL)    │           │
│                              └──────────┬──────────┘           │
│                                         │                       │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    BRIDGE BINARY                         │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ │   │
│   │  │ Keystore  │ │  Trust    │ │   Audit   │ │  RPC    │ │   │
│   │  │(Encrypted)│ │ Middleware│ │   Log     │ │ Server  │ │   │
│   │  └───────────┘ └───────────┘ └───────────┘ └─────────┘ │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ │   │
│   │  │  Budget   │ │  WebRTC   │ │  Errors   │ │  Health │ │   │
│   │  │  Tracker  │ │  Engine   │ │  System   │ │ Monitor │ │   │
│   │  └───────────┘ └───────────┘ └───────────┘ └─────────┘ │   │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐             │   │
│   │  │  Ghost    │ │  License  │ │  Crypto   │             │   │
│   │  │  Manager  │ │   State   │ │   Store   │             │   │
│   │  └───────────┘ └───────────┘ └───────────┘             │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│              ┌───────────────┼───────────────┐                  │
│              ▼               ▼               ▼                  │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│   │    Matrix    │  │   Container  │  │    TURN      │         │
│   │  Homeserver  │  │   Runtime    │  │   Server     │         │
│   │  (Conduit)   │  │   (Docker)   │  │  (Coturn)    │         │
│   └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Project History

See [CHANGELOG.md](../CHANGELOG.md) for complete version history with commit references.

---

## Support

- **Issues:** https://github.com/armorclaw/armorclaw/issues
- **Documentation Issues:** Create issue with `docs:` label
- **Bug Reports:** Create issue with `bug:` label

---

**Documentation Version:** 5.3.0 | **Last Updated:** 2026-02-21
