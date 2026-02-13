# SDTW Adapter Implementation Plan (Enhanced v2.0)

> **Secure Multi-Platform Messaging Integration via ArmorClaw + Matrix Root of Trust**
> **Owner:** Platform / Security Engineering
> **Status:** Design Reviewed, Enhanced & Implementation-Ready
> **Dependencies:** ArmorClaw Phase 1 (Production-Ready as of 2026-02-11)
> **Scope:** Slack, Discord, Microsoft Teams, WhatsApp (SDTW) Adapters
> **Version:** 2.0 (Enhanced Edition)
> **Date:** 2026-02-12
> **Reviewer:** CTO, Component Catch

---

## Executive Summary

This enhanced implementation plan provides a comprehensive blueprint for integrating Slack, Discord, Microsoft Teams, and WhatsApp (SDTW) adapters with ArmorClaw. Building upon the original proposal, this version addresses critical gaps identified during review, including resilience patterns, observability frameworks, disaster recovery, cost management, and compliance considerations. The plan has been expanded to include circuit breaker patterns, message queuing, incident response procedures, threat modeling, and detailed testing strategies that ensure enterprise-grade reliability and security.

**Core Rule:** Element X / ArmorChat (Matrix) remains the sole root of trust. All SDTW platforms are treated as untrusted, downstream transports with zero policy authority, no direct agent control capabilities, and absolutely no permission escalation paths. Every SDTW interaction must be negotiated within Matrix, enforced by ArmorClaw's zero-trust middleware, and logged through the event bus for complete audit trails.

This architecture preserves ArmorClaw's security guarantees including end-to-end encryption (E2EE), PII scrubbing, budget guardrails, and container TTL management, even in worst-case scenarios such as SDTW account compromise or API breach. The plan leverages ArmorClaw's production-ready status (Phase 1 complete with 200+ passing tests, comprehensive documentation, and deployment guides) to ensure rapid, secure iteration with minimal technical debt.

**Key Enhancements in v2.0:**
- **Resilience Patterns:** Circuit breaker, bulkhead isolation, retry with jitter, and graceful degradation
- **Message Reliability:** Internal message queue with persistent buffering and delivery guarantees
- **Observability Framework:** Comprehensive metrics, distributed tracing, alerting, and dashboards
- **Incident Response:** Defined procedures, escalation paths, and automated recovery
- **Cost Management:** Resource estimation, budget tracking, and optimization strategies
- **Compliance Framework:** GDPR considerations, data retention policies, and audit requirements
- **Security Enhancements:** Threat modeling (STRIDE), message signing, and penetration testing scope
- **Testing Expansion:** Mutation testing, chaos engineering, load testing, and contract testing

---

## Project Status Assessment (ArmorClaw)

Based on a thorough review of the ArmorClaw Architecture Review (v1.4.2, dated 2026-02-11) and production readiness evaluation:

### Current Capabilities

**Completion Status:** Phase 1 is fully complete and production-ready. All 8 core components, 11 RPC methods, 5 base security features, and 1 configuration system have been implemented and validated through comprehensive testing.

**Security Enhancements:** Six new security packages are operational with 91+ passing tests:
- Zero-trust middleware with request validation
- Financial guardrails for budget enforcement
- Host hardening configurations
- Container TTL management
- PII scrubbing with pattern recognition
- Setup wizard for secure initialization

The exploit test suite demonstrates 26/26 passing tests with no vulnerabilities detected in critical attack vectors: shell escape attempts, network exfiltration, filesystem access violations, or privilege escalation exploits.

**Integrations Status:** The WebRTC Voice subsystem (459+ lines, 95+ tests) and Event Push system (470+ lines event bus, 450+ lines tests) are complete. Real-time notifications and health monitoring have been integrated into the core infrastructure.

**Infrastructure Readiness:** Docker Hub repository updated (`mikegemut/armorclaw`) with critical fixes for permissions, circular dependencies, and dangerous tools removal. Container image optimized to 393 MB (compressed 98.2 MB) with minimal attack surface.

**Testing & Documentation:** 200+ tests passing across all modules. Comprehensive documentation includes 11 deployment guides, security configuration (480 lines), WebRTC integration guide (600 lines), WebSocket implementation guide (600 lines), and detailed error catalog.

**Performance Baseline:** Low memory footprint (11.4 MB binary, ~50 MB idle runtime), latency targets achieved (<15ms container create, <100ms Matrix send). These metrics provide a solid foundation for SDTW adapter integration.

### Identified Gaps for SDTW Integration

**Technical Debt:** Minor inconsistencies in error handling patterns across modules. Test coverage gaps exist in edge case scenarios. No rate limiting infrastructure currently implemented.

**Missing Features:** Metrics collection and Prometheus integration planned for Phase 2. No distributed tracing capability. Message queuing infrastructure not present.

**Operational Gaps:** No incident response runbooks. Disaster recovery procedures undefined. Cost monitoring not implemented.

### Overall Assessment

ArmorClaw demonstrates high maturity with a strong security posture ideal for SDTW adapter integration. The zero-trust architecture and event bus features directly support untrusted third-party integrations. Proceed with implementation while allocating dedicated resources for integration testing and operational readiness activities.

---

## Core Principles

### Security Principles

**Matrix as Root of Trust:** ArmorClaw's E2EE Matrix messaging (Olm/Megolm encryption), zero-trust validation pipeline, PII scrubbing engine, budget guardrails, TTL cleanup, structured logging, and real-time event bus represent the security perimeter that must never be bypassed. All SDTW permissions originate exclusively in trusted Matrix rooms via Element X / ArmorChat clients.

**Minimal Adapter Scope:** SDTW adapters handle message relay operations only. They contain no policy logic, no agent control mechanisms, no persistent state storage, and no credential caching. Each adapter operates as a stateless relay within the zero-trust boundary.

**Defense in Depth:** Policies are enforced at multiple layers: ArmorClaw bridge level, zero-trust middleware, and adapter-level validation. Runtime credential injection from the hardware-bound keystore ensures credentials never persist in adapter memory.

**Compliance & Auditing:** Adapters must comply with platform Terms of Service (explicit prohibition of scraping, automation abuse, and unauthorized data collection). All actions are logged via the event bus for complete audit trails supporting compliance requirements.

### Reliability Principles

**Graceful Degradation:** When SDTW platforms experience outages or rate limiting, the system must degrade gracefully without impacting other adapters or the core Matrix infrastructure.

**Circuit Breaker Pattern:** External API failures trigger circuit breaker activation to prevent cascade failures, with automatic recovery testing and gradual traffic restoration.

**Message Durability:** Critical messages are queued with persistent storage, ensuring no data loss during transient failures. Delivery acknowledgments are required before message removal.

**Observability by Default:** Every adapter operation generates metrics, traces, and logs. Proactive alerting identifies issues before they impact users.

### Operational Principles

**Infrastructure as Code:** All adapter configurations, deployment artifacts, and operational procedures are version-controlled and reproducible.

**Zero-Downtime Deployment:** Rolling updates with health checks ensure continuous availability during deployments and rollbacks.

**Cost Awareness:** Resource utilization is monitored and optimized. Budget alerts prevent runaway costs from inefficient adapter operations.

---

## Enhanced Architecture

### High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Element X / ArmorChat (Matrix)                       │
│                                                                         │
│  • E2EE Conversations (Olm/Megolm)     • Permission Negotiation        │
│  • Human Approvals / Revocations       • Real-Time Event Monitoring    │
│  • Admin Command Interface             • Alert & Notification Sink      │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    │
                                    │ JSON-RPC + Event Bus (TLS 1.3)
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ArmorClaw Bridge (Go) - Enhanced                     │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Security Layer                                │   │
│  │  • Zero-Trust Middleware  • Policy Engine (Room-Scoped Rules)   │   │
│  │  • PII Scrubbing          • Message Signing & Verification      │   │
│  │  • Input Validation       • Rate Limiting (Token Bucket)        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Resilience Layer                              │   │
│  │  • Circuit Breaker        • Bulkhead Isolation                  │   │
│  │  • Retry with Jitter      • Message Queue (Persistent)          │   │
│  │  • Timeout Management     • Graceful Degradation                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Observability Layer                           │   │
│  │  • Metrics (Prometheus)   • Distributed Tracing (OpenTelemetry) │   │
│  │  • Structured Logging     • Health Monitoring                   │   │
│  │  • Alert Management       • Performance Dashboards              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Infrastructure Layer                          │   │
│  │  • Hardware-Bound Keystore      • TTL & Budget Enforcement      │   │
│  │  • Real-Time Event Bus          • SDTW Adapter Manager          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            ▼                       ▼                       ▼
┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────────┐
│  Circuit Breaker      │ │  Circuit Breaker      │ │  Circuit Breaker      │
│  [Slack Adapter]      │ │  [Discord Adapter]    │ │  [Teams/WhatsApp]     │
│                       │ │                       │ │                       │
│  • Rate Limit: 1/s    │ │  • Rate Limit: 50/s   │ │  • Platform-Specific  │
│  • Queue: 1000 msgs   │ │  • Queue: 2000 msgs   │ │  • Queue: Configurable│
│  • Timeout: 30s       │ │  • Timeout: 30s       │ │  • Timeout: 30s       │
└───────────┬───────────┘ └───────────┬───────────┘ └───────────┬───────────┘
            │                       │                       │
            ▼                       ▼                       ▼
┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────────┐
│   Slack API           │ │   Discord Gateway     │ │   Teams/WhatsApp API  │
│                       │ │                       │ │                       │
│  • Bot Token Auth     │ │  • Bot Gateway + REST │ │  • Bot Framework      │
│  • Events API         │ │  • Shard Management   │ │  • Business API       │
│  • Webhooks           │ │  • WebSocket Events   │ │  • Webhooks           │
└───────────────────────┘ └───────────────────────┘ └───────────────────────┘
```

### Data Flow Architecture

**Inbound Flow (SDTW → Matrix):**
```
External Event → Adapter.ReceiveEvent()
                     │
                     ▼
              ┌──────────────┐
              │ Rate Limiter │ (Token Bucket Check)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Circuit Breaker│ (State: Closed/Open/HalfOpen)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Input Validator│ (Schema Validation)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Zero-Trust   │ (Policy Enforcement)
              │ Middleware   │
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ PII Scrubber │ (Pattern Detection & Redaction)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Message Signer│ (HMAC-SHA256 Integrity)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Event Bus    │ (Publish to Subscribers)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Matrix Room  │ (E2EE Delivery)
              └──────────────┘
```

**Outbound Flow (Matrix → SDTW):**
```
Matrix Command → Policy Engine
                     │
                     ▼
              ┌──────────────┐
              │ Permission   │ (Room-Scoped Policy Check)
              │ Check        │
              └──────┬───────┘
                     │ Allowed
                     ▼
              ┌──────────────┐
              │ Budget Check │ (Rate Limit & Quota)
              └──────┬───────┘
                     │ Within Limits
                     ▼
              ┌──────────────┐
              │ Message Queue│ (Persistent Buffer)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ Circuit Breaker│ (Adapter Health Check)
              └──────┬───────┘
                     │ Closed
                     ▼
              ┌──────────────┐
              │ Adapter.Send │ (Platform-Specific)
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │ ACK Received │ → Remove from Queue
              └──────────────┘
                     │
                     ▼
              ┌──────────────┐
              │ Event Bus    │ (Log Delivery Success)
              └──────────────┘
```

**Error Handling Flow:**
```
Error Detected
      │
      ├─ Transient Error (Rate Limit, Timeout)
      │       │
      │       ▼
      │   ┌──────────────┐
      │   │ Retry Queue  │ (Exponential Backoff + Jitter)
      │   └──────┬───────┘
      │          │ Max Retries Exceeded
      │          ▼
      │   ┌──────────────┐
      │   │ Dead Letter  │ (Manual Review Queue)
      │   │ Queue        │
      │   └──────┬───────┘
      │
      ├─ Permanent Error (Auth Failure, Invalid Target)
      │       │
      │       ▼
      │   ┌──────────────┐
      │   │ Matrix Alert │ (Immediate Notification)
      │   └──────┬───────┘
      │          │
      │          ▼
      │   ┌──────────────┐
      │   │ Audit Log    │ (Security Event)
      │   └──────────────┘
      │
      └─ Circuit Breaker Trip
              │
              ▼
          ┌──────────────┐
          │ Health Check │ (Periodic Probe)
          │ Mode         │
          └──────┬───────┘
                 │ Recovery
                 ▼
          ┌──────────────┐
          │ Half-Open → │ (Gradual Traffic Restoration)
          │ Closed      │
          └──────────────┘
```

---

## Supported Platforms & Capabilities

### Platform Specifications

| Platform | Auth Model | Direction | Key APIs/Features | Rate Limits | Limitations/Notes |
|----------|------------|-----------|-------------------|-------------|-------------------|
| **Slack** | Bot Token + Events API | Read/Write | Webhooks for events; chat.postMessage; conversations.history | Tier 1: 1 msg/sec; Tier 2: 5 msg/sec; Tier 3: 50 msg/sec | Strict ToS; no automation abuse; workspace approval required for bots |
| **Discord** | Bot Gateway + REST | Read/Write | WebSocket Gateway for real-time; REST for sends; Slash Commands | 50 requests/sec global; 5 req/sec/channel | Guild/channel scoping; shard reconnection handling required |
| **Microsoft Teams** | Bot Framework | Read/Write | Activity handlers; proactive messaging; Adaptive Cards | 1500 msgs/60sec per tenant | Azure AD registration required; app manifest approval |
| **WhatsApp** | Business API | Read/Write (Phase 2) | Webhooks for inbound; POST for outbound; Template Messages | 80 msg/min (free tier); varies by tier | Meta approval required; 24hr window for free-form messages; template pre-approval |

### Credentials Management

**Storage Architecture:**
```
┌─────────────────────────────────────────────────────────────────┐
│                 Hardware-Bound Keystore                          │
│                                                                  │
│  • Encryption: XChaCha20-Poly1305 (256-bit key)                 │
│  • Storage: SQLCipher encrypted database                        │
│  • Binding: TPM/Secure Enclave hardware attestation             │
│  • Access: Ephemeral injection only (no persistent cache)       │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Credential Types Stored:                                 │    │
│  │  • Bot Tokens (Slack, Discord, Teams)                   │    │
│  │  • OAuth Refresh Tokens (where applicable)              │    │
│  │  • API Keys and Secrets                                 │    │
│  │  • Webhook Signing Secrets                              │    │
│  │  • Certificate Private Keys (Teams)                     │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Token Rotation Strategy:**
- **Slack:** Manual rotation via admin UI; 90-day rotation recommended
- **Discord:** No automatic expiration; monitor for suspicious activity
- **Teams:** Azure AD token refresh (automatic); app secrets 180-day max
- **WhatsApp:** Permanent tokens; rotate on compromise detection

---

## Phase Breakdown (Enhanced)

### Phase 1: SDTW Adapter Foundation (2 Weeks)

**Goal:** Create robust adapter infrastructure with resilience patterns, integrated with ArmorClaw's security and event systems.

**Detailed Milestones:**

**Week 1: Core Infrastructure**
- Define and implement `SDTWAdapter` interface with resilience patterns
- Build adapter manager with circuit breaker and bulkhead isolation
- Implement message queue with persistent storage (SQLite/BoltDB)
- Create rate limiter with token bucket algorithm
- Set up input validation framework with schema definitions

**Week 2: Platform Adapters**
- Implement Slack adapter with Events API integration
- Implement Discord adapter with Gateway connection management
- Implement Teams adapter with Bot Framework integration
- Implement WhatsApp adapter stub (full implementation in Phase 2)
- Integrate with keystore for credential injection
- Add comprehensive error handling with categorization

**Enhanced Interface Definition:**

```go
// SDTWAdapter defines the contract for platform adapters
// All implementations must be thread-safe and support graceful shutdown
type SDTWAdapter interface {
    // Metadata
    Platform() string                    // e.g., "slack", "discord"
    Capabilities() CapabilitySet         // Feature flags
    Version() string                     // Adapter version for compatibility

    // Lifecycle
    Initialize(ctx context.Context, config AdapterConfig) error
    Start(ctx context.Context) error     // Begin processing
    Shutdown(ctx context.Context) error  // Graceful cleanup

    // Core Operations
    SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error)
    ReceiveEvent(event ExternalEvent) error  // Inbound handler

    // Health & Monitoring
    HealthCheck() (HealthStatus, error)
    Metrics() AdapterMetrics              // Prometheus-compatible
    State() AdapterState                  // Circuit breaker state, etc.
}

// CapabilitySet defines adapter feature support
type CapabilitySet struct {
    Read          bool     // Can receive messages
    Write         bool     // Can send messages
    Media         bool     // Supports media attachments
    Reactions     bool     // Supports message reactions
    Threads       bool     // Supports threaded replies
    Edit          bool     // Can edit messages
    Delete        bool     // Can delete messages
    Typing        bool     // Typing indicators
    ReadReceipts  bool     // Read receipt support
}

// Target identifies a message destination
type Target struct {
    Platform    string            // "slack", "discord", etc.
    RoomID      string            // Matrix room mapping
    Channel     string            // Platform channel ID
    UserID      string            // Platform user ID (DMs)
    ThreadID    string            // Thread/timestamp for replies
    Metadata    map[string]string // Platform-specific data
}

// Message represents sanitized message content
type Message struct {
    ID          string            // Unique message identifier
    Content     string            // Text content (PII scrubbed)
    Type        MessageType       // Text, Image, File, etc.
    Attachments []Attachment      // Media attachments
    ReplyTo     string            // Parent message ID (threads)
    Metadata    map[string]string // Platform-specific metadata
    Signature   string            // HMAC-SHA256 integrity
    Timestamp   time.Time         // Message creation time
}

// SendResult provides delivery status
type SendResult struct {
    MessageID   string            // Platform message ID
    Delivered   bool              // Delivery confirmation
    Timestamp   time.Time         // Delivery time
    Error       *AdapterError     // Error details if failed
    Metadata    map[string]string // Platform response data
}

// HealthStatus for monitoring integration
type HealthStatus struct {
    Connected     bool            // Connection state
    LastPing      time.Time       // Last successful ping
    LastMessage   time.Time       // Last message processed
    ErrorRate     float64         // Error percentage (1h window)
    Latency       time.Duration   // Average latency (5m window)
    QueueDepth    int             // Pending messages
    Error         string          // Current error if any
}

// AdapterState for operational visibility
type AdapterState struct {
    CircuitBreaker  CBState       // Closed, Open, HalfOpen
    ConsecutiveFails int          // Failure counter
    LastFailure     time.Time     // Most recent failure
    RateLimitStatus RateLimitInfo // Current rate limit state
}

// AdapterError with categorization for handling
type AdapterError struct {
    Code        ErrorCode      // Classification
    Message     string         // Human-readable error
    Retryable   bool           // Can be retried
    RetryAfter  time.Duration  // Suggested backoff
    Permanent   bool           // Non-recoverable
}

type ErrorCode string

const (
    ErrRateLimited    ErrorCode = "rate_limited"
    ErrAuthFailed     ErrorCode = "auth_failed"
    ErrInvalidTarget  ErrorCode = "invalid_target"
    ErrNetworkError   ErrorCode = "network_error"
    ErrTimeout        ErrorCode = "timeout"
    ErrCircuitOpen    ErrorCode = "circuit_open"
    ErrValidation     ErrorCode = "validation_error"
    ErrPlatformError  ErrorCode = "platform_error"
)
```

**Circuit Breaker Implementation:**

```go
// CircuitBreaker prevents cascade failures
type CircuitBreaker struct {
    maxFailures     int           // Threshold to open (default: 5)
    timeout         time.Duration // Open duration (default: 30s)
    state           CBState
    failureCount    int
    lastFailure     time.Time
    lastStateChange time.Time
    halfOpenCalls   int           // Successful calls in half-open
    requiredSuccess int           // Successes to close (default: 3)
    mu              sync.RWMutex
}

type CBState int

const (
    CBStateClosed CBState = iota   // Normal operation
    CBStateOpen                    // Failing fast
    CBStateHalfOpen                // Testing recovery
)

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case CBStateOpen:
        if time.Since(cb.lastStateChange) < cb.timeout {
            return &AdapterError{
                Code:      ErrCircuitOpen,
                Message:   "circuit breaker is open",
                Retryable: true,
                RetryAfter: cb.timeout - time.Since(cb.lastStateChange),
            }
        }
        cb.state = CBStateHalfOpen
        cb.halfOpenCalls = 0
        cb.lastStateChange = time.Now()

    case CBStateHalfOpen:
        // Limited calls in half-open state
    }

    err := fn()
    if err != nil {
        cb.onFailure()
        return err
    }
    cb.onSuccess()
    return nil
}
```

**Message Queue Implementation:**

> **Decision (2026-02-12): SQLite selected as queue backend**
> **Rationale:** Better concurrency support (WAL mode), mature Go ecosystem, ACID guarantees
> **Detailed Spec:** See `docs/plans/SDTW_MessageQueue_Specification.md`

```go
// MessageQueue provides reliable delivery with persistence
type MessageQueue struct {
    db          *sql.DB           // SQLite database with WAL mode
    config      QueueConfig       // Platform-specific configuration
    backoff     BackoffStrategy    // Exponential + jitter
    metrics     *QueueMetrics      // Prometheus metrics collector
    mu          sync.RWMutex
}

type QueueItem struct {
    ID          string
    Message     Message
    Target      Target
    Platform    string
    Attempts    int
    LastAttempt time.Time
    NextRetry   time.Time
    Error       string
    CreatedAt   time.Time
}

func (q *MessageQueue) Enqueue(ctx context.Context, platform string, target Target, msg Message) error {
    item := &QueueItem{
        ID:        uuid.New().String(),
        Message:   msg,
        Target:    target,
        Platform:  platform,
        CreatedAt: time.Now(),
        NextRetry: time.Now(),
    }
    return q.storage.Store(item)
}

func (q *MessageQueue) ProcessNext(ctx context.Context) (*QueueItem, error) {
    // Get next eligible item (NextRetry <= now)
    // Mark as inflight
    // Return for processing
}

func (q *MessageQueue) Ack(ctx context.Context, id string) error {
    // Remove from queue after successful delivery
    return q.storage.Delete(id)
}

func (q *MessageQueue) Nack(ctx context.Context, id string, err error) error {
    // Handle failure: increment attempts, calculate next retry
    // If max retries exceeded, move to dead letter queue
    item, _ := q.storage.Get(id)
    item.Attempts++

    if item.Attempts >= q.maxRetries {
        return q.deadLetter.Add(item, err)
    }

    item.NextRetry = q.backoff.Next(item.Attempts)
    item.Error = err.Error()
    return q.storage.Update(item)
}
```

**Testing Requirements:**
- Unit tests: 85% coverage minimum for adapter core
- Mock API servers for each platform
- Circuit breaker state transition tests
- Message queue persistence tests
- Rate limiter boundary tests

**Resource Estimate:** 2 engineers, 600-800 LOC core + 200 LOC per adapter

---

### Phase 2: Policy Engine & Security (3 Weeks)

**Goal:** Implement comprehensive policy management with security controls and audit capabilities.

**Detailed Milestones:**

**Week 1: Policy Model & Storage**
- Design policy schema with versioning support
- Implement policy storage in encrypted keystore
- Create policy CRUD operations with validation
- Build Matrix command parser for policy management

**Week 2: Enforcement Pipeline**
- Implement policy evaluation engine
- Build permission check middleware
- Create TTL integration for policy expiration
- Add revocation handling with immediate effect

**Week 3: Security Enhancements**
- Implement message signing (HMAC-SHA256)
- Add comprehensive audit logging
- Build threat detection patterns
- Create security event notifications

**Enhanced Policy Model:**

```go
// AdapterPolicy with full audit trail
type AdapterPolicy struct {
    ID          string            // Unique policy identifier
    Version     int               // Schema version for migrations
    Platform    string            // Target platform
    RoomID      string            // Matrix room scope
    SDTWTarget  Target            // SDTW-specific target

    Permissions []Permission      // Granted permissions
    Constraints PolicyConstraints // Additional restrictions

    CreatedAt   time.Time
    CreatedBy   string            // Matrix user ID
    ExpiresAt   time.Time         // TTL integration
    RevokedAt   *time.Time        // Revocation timestamp
    RevokedBy   string            // Revoking user

    AuditLog    []AuditEntry      // Complete history
    Metadata    map[string]string // Extension point
}

type Permission string

const (
    PermRead      Permission = "read"       // Receive messages
    PermWrite     Permission = "write"      // Send messages
    PermMedia     Permission = "media"      // Send/receive media
    PermReact     Permission = "react"      // Message reactions
    PermThread    Permission = "thread"     // Thread operations
    PermMention   Permission = "mention"    // @mention users
    PermAdmin     Permission = "admin"      // Policy management
)

type PolicyConstraints struct {
    MaxMessageLength int              // Character limit
    AllowedMentions  []string         // Permitted @mention targets
    BlockedPatterns  []string         // Regex patterns to block
    RateLimit        RateLimitConfig  // Per-policy rate limits
    AllowedHours     TimeWindow       // Operating hours
    AllowedUsers     []string         // Whitelist of users
    BlockedUsers     []string         // Blacklist of users
}

type RateLimitConfig struct {
    MessagesPerMinute int
    MessagesPerHour   int
    MessagesPerDay    int
}

type TimeWindow struct {
    StartHour   int     // 0-23
    EndHour     int     // 0-23
    DaysOfWeek  []int   // 0-6 (Sunday=0)
    Timezone    string  // IANA timezone
}

type AuditEntry struct {
    Timestamp   time.Time
    Action      AuditAction
    Actor       string            // User or system
    Details     map[string]string
    IPAddress   string            // If applicable
    UserAgent   string            // Client info
}

type AuditAction string

const (
    AuditCreated   AuditAction = "created"
    AuditModified  AuditAction = "modified"
    AuditAccessed  AuditAction = "accessed"
    AuditRevoked   AuditAction = "revoked"
    AuditExpired   AuditAction = "expired"
    AuditViolated  AuditAction = "violated"
)
```

**Policy Engine Implementation:**

```go
// PolicyEngine evaluates permissions for all SDTW operations
type PolicyEngine struct {
    store       PolicyStorage
    cache       *lru.Cache       // Hot policy cache
    eventBus    *EventBus
    metrics     *Metrics
    mu          sync.RWMutex
}

func (pe *PolicyEngine) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyResult, error) {
    // 1. Get applicable policies for room/platform
    policies := pe.store.GetPolicies(ctx, req.RoomID, req.Platform)

    // 2. Check policy validity (not expired, not revoked)
    validPolicies := pe.filterValid(policies)

    // 3. Check permissions
    for _, p := range validPolicies {
        if pe.matchesTarget(p, req.Target) && pe.hasPermission(p, req.Permission) {
            // 4. Check constraints
            if !pe.checkConstraints(p, req) {
                pe.eventBus.Publish(EventPolicyViolation, req)
                return &PolicyResult{
                    Allowed:   false,
                    Reason:    "constraint violation",
                    PolicyID:  p.ID,
                }, nil
            }

            // 5. Log access
            pe.auditAccess(p, req)

            return &PolicyResult{
                Allowed:   true,
                PolicyID:  p.ID,
                ExpiresAt: p.ExpiresAt,
            }, nil
        }
    }

    // Default deny
    pe.eventBus.Publish(EventPolicyDenied, req)
    return &PolicyResult{
        Allowed: false,
        Reason:  "no matching policy",
    }, nil
}
```

**Message Signing:**

```go
// MessageSigner provides integrity verification
type MessageSigner struct {
    key         []byte          // Signing key from keystore
    algorithm   string          // "HMAC-SHA256"
}

func (ms *MessageSigner) Sign(msg *Message) error {
    h := hmac.New(sha256.New, ms.key)

    // Sign canonical representation
    canonical := fmt.Sprintf("%s:%s:%s:%d",
        msg.ID, msg.Type, msg.Content, msg.Timestamp.Unix())

    h.Write([]byte(canonical))
    msg.Signature = hex.EncodeToString(h.Sum(nil))
    return nil
}

func (ms *MessageSigner) Verify(msg *Message) bool {
    expected := msg.Signature
    msg.Signature = "" // Clear for verification

    ms.Sign(msg)
    return hmac.Equal([]byte(expected), []byte(msg.Signature))
}
```

**Matrix Commands (Enhanced):**

| Command | Description | Example |
|---------|-------------|---------|
| `/sdtw allow` | Grant permissions | `/sdtw allow slack #incident read write --expire 24h --max-rate 60/h` |
| `/sdtw deny` | Revoke permissions | `/sdtw deny discord #gaming write` |
| `/sdtw list` | Show policies | `/sdtw list slack --room #incident` |
| `/sdtw revoke` | Emergency revoke | `/sdtw revoke all --reason "security incident"` |
| `/sdtw status` | Adapter health | `/sdtw status slack` |
| `/sdtw audit` | View audit log | `/sdtw audit --platform slack --last 24h` |

**Testing Requirements:**
- Policy CRUD operations: 30+ tests
- Enforcement scenarios: 50+ tests including edge cases
- Constraint validation: 20+ tests
- Message signing: 15+ tests
- Audit logging: 20+ tests
- End-to-end policy flows: 25+ tests

---

### Phase 3: Integration & Comprehensive Testing (2 Weeks)

**Goal:** Full system integration, performance validation, security audit, and operational readiness.

**Week 1: Integration & Performance**
- Wire adapters into ArmorClaw bridge
- Implement RPC methods (`sdtw.send`, `sdtw.policy`, `sdtw.status`)
- Performance benchmarking
- Load testing

**Week 2: Security & Resilience**
- Security audit with penetration testing scope
- Chaos engineering experiments
- Incident response simulation
- Documentation finalization

**Testing Strategy Matrix:**

| Test Type | Tools | Coverage Target | Artifacts |
|-----------|-------|-----------------|-----------|
| Unit Tests | Go test, testify | 85% lines | Coverage report |
| Integration Tests | Docker Compose, mock APIs | 100% critical paths | Test matrix |
| Contract Tests | Pact, OpenAPI validator | 100% API contracts | Contract specs |
| Load Tests | k6, Locust | 10x expected load | Performance report |
| Chaos Tests | Chaos Mesh, Litmus | 15 failure scenarios | Resilience report |
| Security Tests | Custom exploit suite | OWASP Top 10 | Security report |
| Mutation Tests | go-mutesting | 80% mutation score | Quality report |

**Performance Benchmarks:**

```yaml
benchmarks:
  latency:
    p50: 50ms
    p95: 100ms
    p99: 200ms
    max_acceptable: 500ms

  throughput:
    messages_per_minute: 1000
    concurrent_adapters: 4
    queue_depth_max: 10000

  resource_limits:
    memory_per_adapter: 50MB
    cpu_per_adapter: 0.5 cores
    goroutines_max: 1000

  reliability:
    uptime_target: 99.9%
    message_delivery_rate: 99.99%
    recovery_time: 30s
```

**Chaos Engineering Scenarios:**

| Scenario | Expected Behavior | Validation |
|----------|-------------------|------------|
| SDTW API timeout | Circuit breaker opens, queue buffers | No message loss |
| SDTW API 5xx errors | Retry with backoff, eventual DLQ | Audit trail intact |
| Network partition | Graceful degradation, alerts sent | Matrix notifications |
| Memory pressure | Backpressure, queue spill to disk | No OOM crashes |
| Credential expiry | Immediate alert, adapter pauses | No credential leaks |
| Concurrent policy conflict | Deterministic resolution | Audit log shows decision |
| Message queue full | Oldest messages dropped with alerts | Critical messages preserved |

---

### Phase 4: Deployment & Operations (1-2 Weeks)

**Goal:** Production rollout with operational tooling and documentation.

**Deployment Artifacts:**

```toml
# config.toml - SDTW Configuration Section
[sdtw]
enabled = true
max_queue_depth = 10000
default_timeout = "30s"
circuit_breaker_threshold = 5
circuit_breaker_timeout = "30s"

[sdtw.slack]
enabled = true
workspace_id = "TXXXXXXXX"
# Credentials injected from keystore at runtime
rate_limit_tier = 2
queue_depth = 1000

[sdtw.discord]
enabled = true
guild_id = "XXXXXXXXXXXXX"
shard_count = 1
queue_depth = 2000

[sdtw.teams]
enabled = true
tenant_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
app_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
queue_depth = 1500

[sdtw.whatsapp]
enabled = false  # Phase 2
phone_number_id = ""
business_account_id = ""
```

**Operational Runbooks:**

1. **Adapter Failure Recovery**
   - Identify failed adapter via health endpoint
   - Check circuit breaker state
   - Review error logs in event bus
   - Manual recovery steps if needed

2. **Policy Emergency Revocation**
   - Execute `/sdtw revoke all --reason "..."`
   - Verify revocation in audit log
   - Confirm adapters stopped processing
   - Document incident

3. **Rate Limit Handling**
   - Monitor rate limit metrics
   - Adjust queue depth if needed
   - Contact platform for limit increase
   - Implement user-level throttling

4. **Credential Rotation**
   - Generate new credentials in platform
   - Update keystore entry
   - Trigger adapter restart
   - Verify old credentials invalidated

---

## Observability Framework

### Metrics (Prometheus)

```yaml
# Adapter Metrics
sdtw_adapter_messages_sent_total{platform, status}
sdtw_adapter_messages_received_total{platform}
sdtw_adapter_latency_seconds{platform, operation}
sdtw_adapter_errors_total{platform, error_type}
sdtw_adapter_circuit_breaker_state{platform}
sdtw_adapter_queue_depth{platform}
sdtw_adapter_rate_limit_remaining{platform}

# Policy Metrics
sdtw_policy_evaluations_total{platform, result}
sdtw_policy_violations_total{platform, violation_type}
sdtw_policy_cache_hit_ratio

# Security Metrics
sdtw_security_events_total{event_type, severity}
sdtw_message_signing_failures_total
sdtw_pii_scrubbed_total{platform}
```

### Alerts

```yaml
# Critical Alerts (Immediate Response)
- alert: SDTWAdapterCircuitBreakerOpen
  expr: sdtw_adapter_circuit_breaker_state == 2
  for: 1m
  severity: critical
  annotations:
    summary: "Adapter {{ $labels.platform }} circuit breaker is open"

- alert: SDTWQueueDepthHigh
  expr: sdtw_adapter_queue_depth > 8000
  for: 5m
  severity: warning
  annotations:
    summary: "High message queue depth for {{ $labels.platform }}"

- alert: SDTWPolicyViolation
  expr: increase(sdtw_policy_violations_total[5m]) > 10
  severity: warning
  annotations:
    summary: "Multiple policy violations detected"

- alert: SDTWCredentialExpiring
  expr: sdtw_credential_expires_in_hours < 168
  severity: warning
  annotations:
    summary: "Credentials for {{ $labels.platform }} expire within a week"
```

### Dashboards

- **SDTW Overview:** All adapters health, message throughput, error rates
- **Per-Platform Detail:** Platform-specific metrics, rate limit status, queue depth
- **Security Dashboard:** Policy violations, audit events, PII scrubbing stats
- **Operations Dashboard:** Circuit breaker states, queue depths, latency trends

---

## Cost Management

### Resource Estimation

| Resource | Estimate | Cost Impact |
|----------|----------|-------------|
| Memory (4 adapters) | 200 MB | $5-10/month (cloud) |
| CPU (average) | 0.5 cores | $15-20/month |
| Persistent Queue | 1 GB disk | $0.10/month |
| Network egress | 10 GB/month | $1-2/month |
| Monitoring (Prometheus) | 50 MB metrics | $5/month |
| Logging (retained 30d) | 5 GB | $2-5/month |

**Total Monthly Estimate:** $30-50 for standard deployment

### Cost Optimization Strategies

1. **Queue TTL:** Auto-expire old messages to prevent storage bloat
2. **Metric Sampling:** Reduce resolution for non-critical metrics
3. **Log Sampling:** Sample routine operations, keep all errors
4. **Adapter Scheduling:** Disable adapters during low-activity periods

---

## Compliance & Audit

### Data Handling

| Data Type | Retention | Encryption | Access Control |
|-----------|-----------|------------|----------------|
| Message Content | 7 days (configurable) | At-rest + in-transit | Policy-scoped |
| Audit Logs | 1 year minimum | At-rest | Admin only |
| Credentials | Until rotated | Hardware-bound keystore | Zero-access |
| Metrics | 90 days | Optional | Operations team |
| Dead Letter Queue | 30 days | At-rest | Admin review |

### Compliance Checklist

- [ ] GDPR: Data subject access requests supported via audit export
- [ ] GDPR: Right to erasure via message deletion APIs
- [ ] SOC 2: Audit trail integrity with signed logs
- [ ] SOC 2: Access control with principle of least privilege
- [ ] Platform ToS: Rate limiting prevents automation abuse
- [ ] Platform ToS: No unauthorized data scraping

---

## Risk Register (Enhanced)

| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|--------|------------|-------|
| SDTW API rate limits | High | Medium | Token bucket + queue + backoff | Adapter Team |
| Platform ToS changes | Medium | High | Relay-only design, ToS monitoring | Compliance |
| Credential compromise | Low | Critical | Keystore-only, rotation procedures | Security |
| Platform API deprecation | Medium | Medium | Version pinning, deprecation alerts | Adapter Team |
| Message queue overflow | Medium | Medium | TTL, spill to disk, alerts | Operations |
| Memory exhaustion | Low | High | Resource limits, backpressure | Platform |
| Policy evaluation bottleneck | Low | Medium | Caching, async evaluation | Platform |
| Incident response delay | Medium | High | Runbooks, on-call procedures | Operations |
| Compliance violation | Low | Critical | Audit trails, policy enforcement | Compliance |

---

## Threat Model (STRIDE)

| Threat Type | Scenario | Mitigation |
|-------------|----------|------------|
| **Spoofing** | Attacker forges SDTW webhook | Webhook signature verification |
| **Tampering** | Message modified in transit | HMAC-SHA256 message signing |
| **Repudiation** | User denies sending message | Immutable audit log with signatures |
| **Information Disclosure** | PII leaked to unauthorized party | PII scrubbing, zero-trust enforcement |
| **Denial of Service** | Flood of messages overwhelms system | Rate limiting, queue depth limits, circuit breaker |
| **Elevation of Privilege** | SDTW user gains policy control | Matrix-only policy authority, no SDTW escalation path |

---

## Rollback Strategy

### Rollback Triggers

- Critical security vulnerability discovered
- Data loss or corruption detected
- Performance degradation > 50% baseline
- Platform API breaking changes

### Rollback Procedure

1. **Immediate:** Disable SDTW adapters via config flag
2. **Verify:** Confirm no message processing
3. **Communicate:** Alert stakeholders via Matrix
4. **Investigate:** Root cause analysis
5. **Restore:** Revert to previous container image
6. **Validate:** Health checks pass, messages resume
7. **Post-Mortem:** Document incident and lessons learned

### Rollback Artifacts

- Previous container image tagged and retained
- Database backup before each deployment
- Config rollback script prepared
- Communication template ready

---

## Timeline & Resources (Revised)

| Phase | Duration | Milestones | Resources | Dependencies |
|-------|----------|------------|-----------|--------------|
| 1: Adapter Foundation | 2 Weeks | Interface + Resilience + 4 Adapters | 2 Engineers | ArmorClaw keystore, event bus |
| 2: Policy Engine | 3 Weeks | Model + Enforcement + Security | 2 Engineers + Security Review | Phase 1 complete |
| 3: Integration & Testing | 2 Weeks | Benchmarks + Audit + Chaos | 1 Engineer + QA + Security | Phase 2 complete |
| 4: Deployment & Operations | 1-2 Weeks | Guides + Runbooks + Release | 1 Engineer + Ops | Phase 3 complete |

**Total Timeline:** 8-9 Weeks

**Budget Estimate:** Low (leveraging Phase 1 infrastructure)

**Risk Buffer:** 1-2 weeks for unforeseen platform API issues

---

## Success Criteria

### Technical Metrics

- [ ] All 4 adapters operational with <100ms p95 latency
- [ ] Circuit breaker prevents cascade failures in all test scenarios
- [ ] Zero message loss in chaos engineering tests
- [ ] 85%+ test coverage across all modules
- [ ] 99.9% uptime during 30-day stabilization period

### Security Metrics

- [ ] All STRIDE threats mitigated
- [ ] Penetration test passes with no critical findings
- [ ] Audit trail integrity verified
- [ ] No credential exposure in logs/memory dumps

### Operational Metrics

- [ ] Runbooks cover all identified failure scenarios
- [ ] Mean time to recovery <15 minutes
- [ ] Alerts are actionable with zero false positives (critical alerts)
- [ ] Documentation rated comprehensive by operations team

---

## Appendices

### A. API Contract Specifications

OpenAPI 3.0 specifications for each SDTW adapter interface available in `/docs/api/`:
- `slack-adapter.yaml`
- `discord-adapter.yaml`
- `teams-adapter.yaml`
- `whatsapp-adapter.yaml`

### B. Configuration Reference

Complete configuration schema documented in `/docs/config-reference.md`

### C. Runbook Index

1. Adapter Failure Recovery
2. Policy Emergency Revocation
3. Rate Limit Handling
4. Credential Rotation
5. Queue Overflow Management
6. Circuit Breaker Manual Reset
7. Incident Response Procedure

### D. Testing Artifacts

- Unit test coverage reports
- Integration test matrix
- Load test results
- Chaos engineering experiment results
- Security audit findings

---

## Conclusion

This enhanced implementation plan addresses all identified gaps from the original proposal while maintaining the security-first approach required for SDTW integration. The addition of resilience patterns, comprehensive observability, and detailed operational procedures ensures enterprise-grade reliability.

**Key Improvements from Original:**
1. **Circuit Breaker & Bulkhead:** Prevents cascade failures from external API issues
2. **Message Queue:** Ensures reliable delivery with persistent buffering
3. **Observability Framework:** Complete metrics, tracing, and alerting infrastructure
4. **Incident Response:** Defined procedures and runbooks for operational readiness
5. **Threat Modeling:** STRIDE analysis ensures comprehensive security coverage
6. **Compliance Framework:** GDPR and audit requirements explicitly addressed
7. **Cost Management:** Resource estimation and optimization strategies defined
8. **Rollback Strategy:** Explicit procedures for safe rollback if needed

**Recommended Next Steps:**
1. Assign engineering resources for Phase 1 kickoff
2. Schedule security review session for threat model validation
3. Establish monitoring infrastructure (Prometheus, Grafana)
4. Create development environment with mock SDTW APIs
5. Begin adapter interface implementation

**Approved for Implementation.**

---

*Document Version: 2.0*
*Last Updated: 2026-02-12*
*Review Cycle: Quarterly*
