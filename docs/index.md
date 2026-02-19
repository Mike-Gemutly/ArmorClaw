# ArmorClaw Documentation

> **Version:** 4.0.0 | **Last Updated:** 2026-02-19 | **Status:** Production Ready

---

## Quick Start

| What | Link | Time |
|------|------|------|
| New to ArmorClaw? | [Getting Started Guide](guides/getting-started.md) | 5 min |
| Deploy to production | [Hostinger VPS Deployment](guides/hostinger-vps-deployment.md) | 15 min |
| Connect via Element X | [Element X Quickstart](guides/element-x-quickstart.md) | 5 min |
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
| **Budget Guardrails** | Token tracking and cost controls | [Security Config](guides/security-configuration.md) | `bridge/pkg/budget` |
| **Security Tiers** | Essential → Enhanced → Maximum | [Tier Upgrade Guide](guides/security-tier-upgrade.md) | `bridge/pkg/lockdown` |

### Communication Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **WebRTC Voice** | Real-time voice with TURN relay | [Voice Guide](guides/webrtc-voice-guide.md) | `bridge/pkg/webrtc` |
| **WebSocket Client** | Real-time Matrix event push | [WebSocket Guide](guides/websocket-client-guide.md) | `bridge/pkg/websocket` |
| **Push Notifications** | FCM, APNS, WebPush via Sygnal | - | `bridge/pkg/push` |
| **SDTW Adapters** | Slack, Discord, Teams, WhatsApp bridges | - | `bridge/internal/sdtw` |

### Enterprise Features

| Feature | Description | Docs | Package |
|---------|-------------|------|---------|
| **License Server** | PostgreSQL-backed license validation | - | `license-server/` |
| **SSO Integration** | SAML 2.0 and OIDC authentication | - | `bridge/pkg/sso` |
| **Web Dashboard** | Embedded management interface | - | `bridge/pkg/dashboard` |
| **Error Handling** | Structured codes, tracking, alerting | [Error Catalog](guides/error-catalog.md) | `bridge/pkg/errors` |
| **Recovery System** | BIP39 phrase, 48-hour window | [Multi-Device UX](guides/multi-device-ux.md) | `bridge/pkg/recovery` |

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

**Documentation Version:** 2.0.0 | **Last Updated:** 2026-02-19
