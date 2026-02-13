# SDTW Adapter Interface Specification

> **Spec:** Slack, Discord, Teams, WhatsApp Adapters
> **Created:** 2026-02-12
> **Status:** Design Complete - Ready for Implementation
> **Dependencies:**
> - SDTW Message Queue (foundational)
> - ArmorClaw Phase 1 (Production-Ready)

---

## Overview

This specification defines the **SDTWAdapter** interface that all platform adapters (Slack, Discord, Microsoft Teams, WhatsApp) must implement. The interface provides a contract for message relay operations between ArmorClaw bridge and external messaging platforms, maintaining zero-trust security principles.

### Core Principles

1. **Zero-Trust:** SDTW platforms are untrusted. All policy decisions happen in Matrix rooms via Element X/ArmorChat.
2. **Stateless:** Adapters maintain no persistent state. All state comes from the queue and policy engine.
3. **Pull Model:** Bridge pulls messages; adapters never poll.
4. **Ephemeral Credentials:** Loaded at runtime from hardware-bound keystore, never persisted.
5. **Observability:** All operations generate metrics for monitoring.

---

## Interface Definition

```go
// SDTWAdapter defines the contract for platform adapters
type SDTWAdapter interface {
    // Metadata
    Platform() string                    // e.g., "slack", "discord", "teams", "whatsapp"
    Capabilities() CapabilitySet         // Feature detection
    Version() string                     // Adapter version for compatibility

    // Lifecycle
    Initialize(ctx context.Context, config AdapterConfig) error
    Start(ctx context.Context) error             // Begin processing
    Shutdown(ctx context.Context) error          // Graceful cleanup

    // Core Operations
    SendMessage(ctx context.Context, target Target, msg Message) (*SendResult, error)
    ReceiveEvent(event ExternalEvent) error  // Inbound handler

    // Health & Monitoring
    HealthCheck() (HealthStatus, error)  // Health monitoring
    Metrics() (AdapterMetrics, error)           // Prometheus metrics
}
```

### Capabilities

```go
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
```

### Target

```go
// Target identifies a message destination
type Target struct {
    Platform    string            // "slack", "discord", "teams", "whatsapp"
    RoomID      string            // Matrix room ID (maps to SDTW target)
    Channel     string            // Platform channel ID
    UserID      string            // Platform user ID (for DMs)
    ThreadID    string            // For threaded replies
    Metadata    map[string]string // Platform-specific data
}
```

### Message

```go
// Message represents sanitized message content
type Message struct {
    ID          string            // Unique message identifier
    Content     string            // Text content (PII scrubbed)
    Type        MessageType       // Text, Image, File, Media
    Attachments []Attachment
    ReplyTo     string            // Parent message ID (threads)
    Metadata    map[string]string // Platform-specific metadata
    Timestamp   time.Time         // Message creation time
    Signature   string            // HMAC-SHA256 integrity
}
```

### SendResult

```go
type SendResult struct {
    MessageID   string            // Platform message ID
    Delivered   bool              // Delivery confirmation
    Timestamp   time.Time         // Delivery time
    Error       *AdapterError      // Error details if failed
    Metadata    map[string]string // Platform response data
}
```

### HealthStatus

```go
type HealthStatus struct {
    Connected     bool            // Connection state
    LastPing    time.Time       // Last successful ping
    LastMessage time.Time       // Last message processed
    ErrorRate  float64         // Error percentage (1h window)
    Latency     time.Duration   // Average latency (5m window)
    QueueDepth   int             // Pending message count
    Error       string          // Current error if any
}
```

### AdapterError

```go
type AdapterError struct {
    Code        ErrorCode      // Error classification
    Message     string         // Human-readable error
    Retryable   bool           // Can be retried
    RetryAfter  time.Duration  // Suggested backoff
    Permanent   bool           // Non-recoverable
}
```

### ErrorCode

```go
type ErrorCode string

const (
    ErrRateLimited    ErrorCode = "rate_limited"
    ErrAuthFailed     ErrorCode = "auth_failed"
    ErrInvalidTarget  ErrorCode = "invalid_target"
    ErrNetworkError   ErrorCode = "network_error"
    ErrTimeout        ErrorCode = "timeout"
    ErrCircuitOpen   ErrorCode = "circuit_open"
    ErrValidation     ErrorCode = "validation_error"
    ErrPlatformError ErrorCode = "platform_error"
)
)
```

---

## Platform-Specific Requirements

### Slack Adapter

**API Methods:**
- `chat.postMessage` - Send message to channel
- `conversations.info` - Get channel info
- Webhooks for inbound events

**Authentication:** Bot token (injected from keystore)
**Rate Limits:** Tier 1: 1 msg/s, Tier 2: 5 msg/s, Tier 3: 50 msg/s

**Message Format:**
```json
{
  "channel": "C12345678",
  "text": "Hello from ArmorClaw!",
  "username": "armorclaw-bot",
  "icon_emoji": ":robot_face:",
  "attachments": []
}
```

**Webhook Configuration:**
```
{
  "token": "verification_token",
  "challenge": "challenge_string_to_hash",
  "url": "https://hooks.slack.com/services/YOUR_SERVICE"
  "event": {"url": "https://hooks.slack.com/events/YOUR_EVENT"}
}
```

### Discord Adapter

**API Methods:**
- Gateway WebSocket connection for real-time events
- REST API for message sends
- Slash command support

**Authentication:** Bot token (injected from keystore)
**Rate Limits:** 50 requests/second global, 5 requests/second per channel
**Message Format:**
```json
{
  "content": "Hello from ArmorClaw!",
  "tts": "unique_message_id",
  "embeds": [{"type": "rich", "title": "ArmorClaw Integration"}]
}
```

### Microsoft Teams Adapter

**API Methods:**
- Bot Framework for activity handling
- Proactive messaging for outbound
- Adaptive Cards for rich formatting

**Authentication:** Azure AD app registration + bot token
**Rate Limits:** 1500 messages/60sec per tenant

### WhatsApp Adapter

**API Methods:**
- Business API for sending messages
- Webhooks for inbound events (requires Meta review)

**Authentication:** Business API credentials (injected from keystore)
**Rate Limits:** 80 messages/minute (free tier)
**Template Restrictions:** 24-hour window for free-form messages

**Message Format:**
```json
{
  "to": "15551234567",
  "type": "text",
  "text": {"body": "Hello from ArmorClaw!"}
}
```

---

## Configuration

```toml
[sdtw.slack]
enabled = true
bot_token = ""  # Injected from keystore
default_channel = "#general"
webhook_url = ""  # Auto-generated or configured

[sdtw.discord]
enabled = true
bot_token = ""  # Injected from keystore
guild_id = ""  # Optional, for multi-guild bots
command_prefix = "!"

[sdtw.teams]
enabled = false  # Phase 2
app_id = ""
tenant_id = ""
app_secret = ""

[sdtw.whatsapp]
enabled = false # Phase 2
phone_number_id = ""
business_account_id = ""
access_token = ""
```

---

## Dependencies

**Requires:**
- Message queue implementation (queue package)
- Zero-trust middleware (zerotrust package)
- Policy engine (policy package)
- PII scrubber (piiscrubbing package)
- Hardware-bound keystore (keystore package)
- JSON-RPC server (rpc package with SDTW methods)

---

## Implementation Priority

**Phase 1: Foundation (Week 1-2)**
1. Adapter interface definition
2. Slack adapter implementation
3. Discord adapter implementation
4. Teams adapter stub (placeholder)
5. WhatsApp adapter stub (placeholder)

**Phase 2: Integration (Week 3)**
1. Wire adapters into ArmorClaw bridge RPC
2. Implement policy engine checks
3. Add monitoring and metrics

**Estimated Effort:** 4-6 weeks total

---

## Success Criteria

- [ ] All adapters compile without errors
- [ ] Can send and receive messages
- [ ] Health checks return accurate status
- [ ] Metrics exported correctly
- [ ] Policies enforced on all operations
- [ ] Credentials never persisted
- [ ] Zero-trust model maintained

---

**Next Steps:**
1. Implement queue package fully (includes SQLite integration)
2. Create sdtw adapter package structure
3. Implement Slack adapter
4. Implement Discord adapter
5. Add RPC methods to bridge server
6. Create comprehensive tests

---

**Document Status:** Design Complete - Ready for Implementation
