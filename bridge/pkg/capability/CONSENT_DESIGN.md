# ConsentProvider Design Document

Version: v0.8.0 (interface only, no implementation)

## 1. Interface Contract

The `ConsentProvider` interface lives in `pkg/interfaces/consent.go` and defines how the capability broker requests human approval for deferred actions.

```go
type ConsentResult struct {
    Approved       bool     `json:"approved"`
    ApprovedFields []string `json:"approved_fields,omitempty"`
    DeniedFields   []string `json:"denied_fields,omitempty"`
    Error          error    `json:"-"`
}

type ConsentProvider interface {
    RequestConsent(ctx context.Context, requestID, reason string, fields []string) (<-chan ConsentResult, error)
}
```

### Channel-based async pattern

`RequestConsent` returns a receive-only channel (`<-chan ConsentResult`) instead of blocking or using callbacks. The channel delivers exactly one result:

- **Approved**: human granted consent, `ApprovedFields` lists which fields are cleared
- **Denied**: human rejected or timeout expired, `DeniedFields` lists blocked fields
- **Error**: non-nil `Error` when something went wrong internally (channel delivery failure, store error)

Callers consume the result with a `select`:

```go
resultCh, err := provider.RequestConsent(ctx, reqID, reason, fields)
if err != nil {
    // synchronous error (queue full, invalid args)
}
select {
case result := <-resultCh:
    if result.Approved {
        // proceed with approved fields
    }
case <-time.After(300 * time.Second):
    // auto-deny: timeout
case <-ctx.Done():
    // caller cancelled
}
```

This pattern gives the caller natural timeout handling via `select`, avoids callback hell, and makes it easy to compose with context cancellation.

## 2. Existing Approval Systems

Three approval systems exist in the codebase today. None of them implement `ConsentProvider` yet. Each will be assessed for integration in future versions.

### A. EmailApprovalManager (`pkg/email/hitl_approval.go`)

The simplest system. 118 lines.

**How it works:**

- `RequestApproval` creates a `pendingApproval` with a buffered channel and a deadline
- Stores it in an in-memory `map[string]*pendingApproval` protected by `sync.RWMutex`
- Sends a Matrix `app.armorclaw.email_approval_request` event with the approval ID, email metadata, and PII field count
- Blocks on the channel with `select` across three cases: response received, timeout (`defaultApprovalTimeout = 300s`), or context cancelled
- `HandleApprovalResponse` writes a decision into the channel when a Matrix response arrives
- Auto-cleanup via `defer` removes the pending entry regardless of outcome

**Mapping to ConsentProvider:**

| EmailApprovalManager | ConsentProvider |
|---------------------|-----------------|
| `RequestApproval(ctx, req)` | `RequestConsent(ctx, requestID, reason, fields)` |
| `ApprovalDecision.Approved` | `ConsentResult.Approved` |
| `ApprovalDecision.DeniedFields` | `ConsentResult.DeniedFields` |
| `pendingApproval.resultCh` | Return `<-chan ConsentResult` |
| 300s default timeout | Caller-side 300s timeout |

**Key difference:** `EmailApprovalManager.RequestApproval` blocks the goroutine. A `ConsentProvider` adapter would return the channel immediately and let the caller manage the blocking `select`.

### B. ApprovalEngineImpl (`pkg/secretary/approvals.go`)

The richest system. 611 lines.

**How it works:**

- Policy-based evaluation backed by a `Store` interface
- `EvaluateStep()` takes a workflow step, task template, and PII fields, then checks active policies
- Each policy returns one of three decisions: `DecisionAllow`, `DecisionDeny`, or `DecisionRequireApproval`
- Fields accumulate across policies: approved fields are subtracted from denied, approval-required fields are subtracted from approved and denied
- If any field requires approval (`NeedsApproval = true`), the workflow pauses for human input
- Supports conditions, auto-approve rules, and delegation (`DelegateTo`)
- `EvaluateWorkflow()` checks the entire workflow before execution starts

**Mapping to ConsentProvider:**

| ApprovalEngineImpl | ConsentProvider |
|-------------------|-----------------|
| `EvaluateStep(ctx, workflow, template, step, piiFields, initiator)` | `RequestConsent(ctx, requestID, reason, fields)` |
| `ApprovalResult.ApprovedFields` | `ConsentResult.ApprovedFields` |
| `ApprovalResult.DeniedFields` | `ConsentResult.DeniedFields` |
| `ApprovalResult.NeedsApproval` | Triggers the channel block |
| Store-backed policy evaluation | Internal to the implementation |

**Key difference:** `ApprovalEngineImpl` evaluates policies synchronously and returns a result struct. The `ConsentProvider` adapter would evaluate policies, and if `NeedsApproval` is true, create a channel that blocks until the human responds. If policies auto-approve or auto-deny, the channel delivers immediately.

### C. PendingApproval (`pkg/secretary/pending_approval.go`)

Wraps the Matrix event bus for PII approval. 154 lines.

**How it works:**

- Package-level globals: `pendingApps map[string]chan piiResponse`, `ApprovalTimeout`, `piiAlertDispatcher`
- `PendingApproval()` publishes an `app.armorclaw.pii_request` Matrix event with step ID, required fields, and timestamp
- Fires a push notification via `piiAlertDispatcher` (set during Bridge startup)
- Blocks on `select` with `ApprovalTimeout` (default 120s, max 900s) or context cancellation
- `HandlePIIResponse()` delivers the human's response into the channel
- Auto-cleanup via `defer` removes the pending entry

**Mapping to ConsentProvider:**

| PendingApproval | ConsentProvider |
|----------------|-----------------|
| `PendingApproval(ctx, eventBus, roomID, stepID, fields)` | `RequestConsent(ctx, requestID, reason, fields)` |
| `piiResponse.Approved` | `ConsentResult.Approved` |
| `piiResponse.Fields` | `ConsentResult.ApprovedFields` |
| `pendingApps` map | Internal to the implementation |
| `ApprovalTimeout` (120s default) | Caller-side 300s timeout |

**Key difference:** Uses package-level globals instead of struct state. Any `ConsentProvider` adapter would need to encapsulate these globals into a struct instance. The task spec flags this for v0.9.0 cleanup.

## 3. Timeout Policy

- **DEFER timeout**: 300 seconds (5 minutes)
- **On timeout**: auto-deny. Send a Matrix `m.notice` notification to the requesting agent's room explaining the denial reason ("consent request expired after 300s")
- **Implementation**: the caller uses `select` with `time.After(300 * time.Second)`. The ConsentProvider itself does not enforce this timeout; the contract documents that callers MUST use 300s.
- **Rationale**: 300s balances giving the user enough time to review and respond (especially on mobile) with keeping workflows live. Shorter timeouts cause false denials. Longer timeouts let workflows stall indefinitely.

```go
select {
case result := <-resultCh:
    // handle result
case <-time.After(300 * time.Second):
    // auto-deny: notify agent room, log the timeout
}
```

The existing systems already use similar patterns:
- `EmailApprovalManager`: 300s default (`defaultApprovalTimeout`)
- `PendingApproval`: 120s default, 900s max (`ApprovalTimeout`)

Consolidating on 300s as the standard DEFER timeout gives consistency across all approval paths.

## 4. Queue Depth Limit

- **Maximum concurrent DEFERRED actions**: 50
- **When limit is exceeded**: new `RequestConsent` calls return an error immediately with "consent queue full" as the reason. The caller treats this as a denial.
- **Purpose**: prevents resource exhaustion if an agent (or set of agents) triggers mass-DEFER scenarios. Without a limit, hundreds of blocked goroutines waiting on human responses could exhaust memory and stall the Bridge.

**Implementation sketch:**

```go
type consentManager struct {
    mu       sync.Mutex
    pending  int
    maxQueue int // 50
}

func (m *consentManager) RequestConsent(...) (<-chan ConsentResult, error) {
    m.mu.Lock()
    if m.pending >= m.maxQueue {
        m.mu.Unlock()
        return nil, fmt.Errorf("consent queue full (%d pending)", m.pending)
    }
    m.pending++
    m.mu.Unlock()
    // ... create channel, register pending
}
```

The limit is configurable at startup but defaults to 50. This matches reasonable expectations: a single VPS user is unlikely to have more than 50 pending approval requests simultaneously.

## 5. Notification Delivery

When a DEFER occurs, the user must be notified promptly so they can respond before the 300s timeout expires.

### Timeline

| Event | Target | Max Latency |
|-------|--------|-------------|
| DEFER decision made | Bridge internal | 0ms |
| Matrix `m.notice` event sent to agent's room | Matrix (Conduit) | 2 seconds |
| Push notification delivered to ArmorChat via Sygnal | Phone | 5 seconds total |

### Notification content

The `m.notice` event body includes:

```json
{
    "action_type": "defer",
    "risk_class": "high",
    "requesting_agent": "agent-abc123",
    "request_id": "req_xxxxx",
    "fields": ["payment.card_number", "payment.cvv"],
    "reason": "Agent requesting payment card for checkout form",
    "timestamp": 1700000000000,
    "expires_in_s": 300
}
```

### Push notification flow

1. Bridge publishes the `m.notice` event to the agent's Matrix room
2. Conduit receives the event and pushes to Sygnal
3. Sygnal sends a push notification to the user's device via FCM/APNs
4. ArmorChat receives the notification, displays the approval UI
5. User approves or denies via ArmorChat (biometric-protected)
6. ArmorChat sends an `app.armorclaw.pii_response` event back through Matrix
7. Bridge receives the response, delivers it into the pending channel

### Latency requirements

- **Matrix event within 2s**: this is achievable because the Bridge and Conduit are on the same VPS (local network, no internet round-trip for the publish step)
- **Push notification within 5s total**: Sygnal to FCM/APNs adds roughly 1-2s. The 5s budget accounts for this plus any event processing overhead.

## 6. Implementation Timeline

| Version | Scope |
|---------|-------|
| **v0.8.0** | Interface only (`pkg/interfaces/consent.go`). No implementations. This design document defines the contract. |
| **v0.9.0** | `HITLConsentManager` implements `ConsentProvider`. Wraps `EmailApprovalManager` as the first real implementation. Caps package-level globals in `pending_approval.go` behind a struct. |
| **v1.0.0** | Consolidation assessment. Evaluate whether `ApprovalEngineImpl`, `EmailApprovalManager`, and `PendingApproval` should merge into a single `ConsentProvider` implementation, or remain separate with a router/dispatcher. |

## 7. Design Decisions

### Channel-based, not callback

Channels integrate naturally with Go's `select` statement. This gives the caller built-in timeout handling, context cancellation, and composability with other channels. Callbacks require separate timeout infrastructure, error propagation chains, and are harder to test.

### Fail-closed

Timeout equals deny, never allow. If the user doesn't respond in time, the action is denied. This is the conservative choice for a security-sensitive system where the actions being approved involve PII, payments, or sensitive operations.

The alternative (fail-open, timeout equals allow) would be dangerous: a user who steps away from their phone would silently approve every pending action.

### No consolidation in v0.8.0

Three separate approval systems exist. They serve different purposes (email HITL, policy-based workflow approval, PII field approval) and have different interfaces. Consolidating them requires careful design to avoid breaking existing workflows. v0.8.0 defines the target interface; v0.9.0 adds the first adapter; v1.0.0 assesses full unification.

### ConsentProvider is separate from CapabilityBroker

The `CapabilityBroker` decides whether an action is allowed, denied, or deferred. The `ConsentProvider` handles the deferral: it manages the human-in-the-loop flow when the broker decides to defer. These are separate responsibilities:

- **Broker**: evaluates risk, policy, and makes a decision
- **ConsentProvider**: when the broker says DEFER, asks the human and returns the answer

This separation means the consent mechanism can be swapped (Matrix HITL, email, SMS, auto-approve for testing) without changing the broker logic.
