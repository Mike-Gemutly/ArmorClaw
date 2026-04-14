# Communication Infrastructure

> Part of the [ArmorClaw System Documentation](armorclaw.md)

## Overview

**These are independent subsystems.** They do not share a common architecture. Each solves a distinct communication problem within ArmorClaw. They were initially grouped under "operational governance" but code analysis confirmed zero cross-dependencies between them. Treat each one as a standalone module with its own lifecycle, configuration, and error handling.

The six subsystems covered here:

| Subsystem | Package | Purpose |
|-----------|---------|---------|
| Push Notifications | `pkg/push/` | Mobile push via FCM, APNS, Web Push, and Matrix Sygnal |
| Single Sign-On | `pkg/sso/` | Enterprise SAML 2.0 and OIDC authentication |
| WebSocket Server | `pkg/websocket/` | Real-time event streaming to clients |
| Event Bus Internals | `pkg/eventbus/` | In-process pub/sub with optional durable log |
| Platform Adapters | `internal/adapter/` | Matrix and Slack message routing |
| Additional Platform Adapters | `internal/sdtw/` | Discord, Telegram, Teams, WhatsApp adapters |

## Subsystems

### Push Notifications (`pkg/push/`)

The push subsystem delivers mobile and browser notifications to ArmorClaw users. It centers on the `Gateway` type, which manages device registrations and routes notifications to platform-specific providers.

**Supported platforms:**

- **FCM** (Firebase Cloud Messaging) for Android and iOS via Google's infrastructure
- **APNS** (Apple Push Notification Service) for iOS devices
- **Web Push** (VAPID) for browser notifications using ECDSA P-256 key pairs
- **Unified Push** as an abstract platform identifier

**How it works:**

1. Devices register with the `Gateway` via `RegisterDevice`, associating a user ID, platform, and device token.
2. The gateway maintains an in-memory map of user to device list, protected by a read-write mutex.
3. To send a notification, call `SendToUser` with a user ID and a `Notification` struct. The gateway fans out to all registered devices for that user, selecting the correct provider per device.
4. Each send attempt includes configurable retries (default 3) with a configurable delay (default 5 seconds).
5. The gateway keeps the last 1,000 `PushResult` entries for stats.

**Matrix Sygnal integration:**

The `CreateMatrixPushNotification` method converts a Matrix room event (room ID, event ID, sender, content) into a set of per-device `Notification` structs. This is the bridge between Matrix message events and native mobile push. The `SygnalURL` and `SygnalAPIKey` fields in `Config` allow the gateway to delegate delivery to a Sygnal push gateway when configured.

**Provider interface:**

Each provider implements `PushProvider` with three methods: `Send`, `ValidateToken`, and `Platform`. Token validation is platform-specific (FCM tokens are long strings, APNS tokens are exactly 64 hex characters, Web Push subscription endpoints are long URLs). A `MockProvider` exists for testing.

**Key types:** `Gateway`, `Notification`, `DeviceRegistration`, `PushResult`, `PushProvider` interface.

### Single Sign-On (`pkg/sso/`)

The SSO subsystem handles enterprise authentication via SAML 2.0 and OpenID Connect. It's managed by `SSOManager`, which wraps an `SSOProvider` implementation and a `StateStore` for OAuth state parameters.

**Authentication flow:**

1. `BeginAuth(redirect)` generates a cryptographic state parameter, stores it (with a 10-minute TTL), and returns the IdP authorization URL.
2. After the user authenticates at the IdP, the callback handler calls `HandleCallback(ctx, code, state)`.
3. The state is validated (one-time use, then deleted). The provider-specific handler processes the code or SAML response.
4. User attributes are extracted and mapped to ArmorClaw roles via `RoleMapping` config. Unmapped users get the default "user" role.
5. The resulting `SSOSession` is stored in memory and can be validated or refreshed.

**SAML 2.0 provider:**

Loads IdP metadata from a URL or file, builds `AuthnRequest` XML, and redirects to the IdP's SSO URL. Callbacks decode and validate the base64-encoded `SAMLResponse`, extract the `NameID` and assertion attributes. SAML sessions cannot be natively refreshed; users must re-authenticate. Supports Single Logout (SLO) when the IdP provides a SLO URL.

**OIDC provider:**

Discovers endpoints automatically from the issuer's `/.well-known/openid-configuration`. Supports PKCE (S256 challenge) for public clients. Token exchange, userinfo retrieval, and refresh token flows are all implemented. Sessions can be refreshed using the stored refresh token. Logout redirects to the IdP's end-session endpoint.

**Security measures:**

- State parameters are one-time use with automatic cleanup.
- Redirect URLs are validated against dangerous schemes (`javascript:`, `data:`, `vbscript:`) and control characters.
- Issuer URLs must use HTTPS except for localhost development.
- Client IDs are length-limited and checked for control characters.

**Key types:** `SSOManager`, `SSOSession`, `SSOProvider` interface, `SAMLProvider`, `OIDCProvider`, `StateStore`.

### WebSocket Server (`pkg/websocket/`)

This is a minimal stub. The `Server` type accepts a `Config` with address, path, allowed origins, max connections, inactivity timeout, and handler callbacks. The `Start` method currently returns an error because the full implementation is deferred to a future release. `Broadcast` is a no-op.

The event bus (`pkg/eventbus`) consumes this package. When the WebSocket server is eventually implemented, the event bus will use it to stream events to connected clients in real time.

**Key types:** `Server`, `Config`, `MessageHandler`, `ConnectHandler`, `DisconnectHandler`.

### Event Bus Internals (`pkg/eventbus/`)

The event bus is the in-process pub/sub backbone for real-time Matrix event distribution. It enables containers and internal components to receive events without polling.

> The main doc's Event Bus Patterns section covers event types (matrix.message, agent.started, workflow.progress, hitl.pending, budget.alert, etc.). This section covers the internal mechanism.

**Core mechanism:**

1. Subscribers call `Subscribe(filter)` and receive a `Subscriber` with a buffered channel (capacity 100).
2. Publishers call `Publish(event)` with a `MatrixEvent`. The bus walks all subscribers, checks each filter (room ID, sender ID, event type), and sends matching events to subscriber channels.
3. If a subscriber's channel is full (slow consumer), the event is dropped and logged. The bus does not block publishers on slow subscribers.
4. A background goroutine cleans up subscribers inactive for more than 30 minutes.
5. Optionally, a durable log (`eventlog.Log`) appends every published event for replay or auditing.
6. Optionally, an embedded WebSocket server streams events to external clients.

**Filter model:**

`EventFilter` has three optional fields: `RoomID`, `SenderID`, and `EventType` (slice). Empty fields match everything. All filter conditions must match (AND logic).

**Bridge events:**

`PublishBridgeEvent` handles non-Matrix events (agent, workflow, HITL, budget, platform). These events implement the `BridgeEvent` interface (`EventType()`, `Timestamp()`, `ToJSON()`). They are wrapped, serialized, and broadcast to WebSocket clients if the server is enabled. Broadcast failures are logged but don't fail the publish operation.

**Error handling:**

The `errors.go` file defines a structured error system. Every error has a domain, code, severity, message, source location, stack trace, and optional context hints. Error codes are organized by category:

| Range | Category | Examples |
|-------|----------|---------|
| E001-E099 | Publisher | nil event, wrap failed, serialize failed, broadcast failed |
| E101-E199 | Subscriber | not found, inactive, channel full, closed |
| E201-E299 | WebSocket | not enabled, connect failed, message failed |
| E301-E399 | Filter | invalid filter |

Each error code is registered in `ErrorRegistry` with a description and resolution hint. The `IsErrorCode` and `AsError` helpers integrate with Go's `errors.Is`/`errors.As` patterns.

**Key types:** `EventBus`, `Subscriber`, `EventFilter`, `MatrixEventWrapper`, `EventError`, `ErrorCode`, `ErrorDomain`.

### Platform Adapters (`internal/adapter/`)

This package contains the primary platform adapters that route messages between external services and the ArmorClaw Bridge.

**Matrix adapter (`matrix.go`):**

The central adapter. `MatrixAdapter` is a full Matrix client that syncs with the homeserver, processes events, and publishes them to the event bus. It handles:

- Login and access token management with token refresh
- Long polling via `/sync` with server-side sync filters for performance
- Event queue for inbound Matrix events
- PII scrubbing via the `pii.Scrubber`
- Zero-trust verification via `TrustVerifier` (sender trust scoring, device fingerprinting)
- Audit logging via `audit.TamperEvidentLog`
- Integration with the high-throughput `events.MatrixEventBus` for agent streaming
- Sync performance metrics (`SyncMetrics`)

**Slack adapter (`slack.go`):**

Routes messages between Slack workspaces and Matrix rooms. `SlackAdapter` supports:

- Bot token authentication (`xoxb-` tokens)
- Socket Mode for real-time events (optional)
- Channel monitoring and forwarding to a configured Matrix room
- User and channel caching
- Rate limiting
- Message queue integration via `queue.MessageQueue`

**Command handling (`commands_integration.go`):**

`CommandHandler` processes Matrix messages that start with `/` as admin commands. It integrates with `admin.ClaimManager` and `lockdown.Manager` for administrative operations like lockdown mode.

**Trust integration (`trust_integration.go`):**

`TrustVerifier` applies zero-trust principles to every Matrix event. It checks sender trust scores, device fingerprints, and IP reputation. Events from untrusted sources are rejected based on a configurable minimum trust level.

**Additional files:**

- `key_ingestion.go` handles cryptographic key ingestion for the Matrix adapter
- `pii_consent.go` manages PII consent tracking per user

### Additional Platform Adapters (`internal/sdtw/`)

The `sdtw` package (named after the platforms it supports: Slack, Discord, Teams, WhatsApp) provides a uniform adapter interface for external messaging platforms. Each adapter implements `SDTWAdapter` and embeds `BaseAdapter` for shared functionality.

**Adapter interface (`adapter.go`):**

`SDTWAdapter` defines the contract every platform adapter must follow:

- **Lifecycle:** `Initialize`, `Start`, `Shutdown`
- **Core:** `SendMessage`, `ReceiveEvent`
- **Reactions:** `SendReaction`, `RemoveReaction`, `GetReactions`
- **Mutation:** `EditMessage`, `DeleteMessage`, `GetMessageHistory`
- **Health:** `HealthCheck`, `Metrics`

Each adapter declares a `CapabilitySet` (read, write, media, reactions, threads, edit, delete, typing, read receipts) so callers can check what operations are supported before attempting them.

Messages carry HMAC-SHA256 signatures for integrity verification. `SignMessage` and `VerifySignature` handle signing and validation.

**Discord adapter (`discord.go`):**

Connects to Discord via bot token. Uses the Discord Gateway protocol with opcodes for heartbeat, identify, resume, and dispatch. Supports rich embeds, message references (threaded replies), and guild-scoped operation.

**WhatsApp adapter (`whatsapp.go`):**

Uses the WhatsApp Business Cloud API. Authenticates with an access token and phone number ID. Supports text, template, image, document, audio, and video message types. Includes a token-bucket rate limiter to respect WhatsApp API limits.

**Microsoft Teams adapter (`teams.go`):**

Authenticates via Azure AD (client ID, client secret, tenant ID). Processes Teams messages via webhook. Supports OAuth token refresh. Includes HMAC-SHA256 webhook verification for inbound request validation.

**Slack adapter (`slack.go`):**

A second Slack adapter in the `sdtw` package, separate from `internal/adapter/slack.go`. This one follows the `SDTWAdapter` interface contract, supports webhook-based message delivery, and handles Slack's URL verification challenge for event subscriptions.

## Integration Points

These subsystems connect to the Bridge core at specific, well-defined points. They don't connect to each other.

```
Bridge Core
  |
  +-- Matrix Adapter (internal/adapter)
  |     |-- Receives Matrix events via /sync
  |     |-- Publishes to EventBus (pkg/eventbus)
  |     |-- Applies TrustVerifier and PII scrubbing
  |
  +-- EventBus (pkg/eventbus)
  |     |-- Receives events from Matrix adapter and internal components
  |     |-- Distributes to in-process subscribers
  |     |-- Optionally broadcasts via WebSocket (pkg/websocket)
  |     |-- Optionally appends to durable log
  |
  +-- Push Gateway (pkg/push)
  |     |-- Called when Matrix messages need mobile notification
  |     |-- Delegates to FCM, APNS, or Web Push providers
  |     |-- Integrates with Sygnal for Matrix push
  |
  +-- SSO Manager (pkg/sso)
  |     |-- Called during authentication flow
  |     |-- Independent of other subsystems
  |
  +-- Platform Adapters (internal/adapter, internal/sdtw)
        |-- Route messages between external platforms and Matrix
        |-- Each adapter is independently configured and started
```

The key takeaway: each subsystem is wired into the Bridge core independently. There is no shared subsystem-to-subsystem wiring. The event bus is the closest thing to a shared dependency, but only the Matrix adapter publishes to it directly. Other subsystems don't depend on it.
