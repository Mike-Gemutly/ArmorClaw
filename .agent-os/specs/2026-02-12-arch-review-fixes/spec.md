# Architectural Review Fixes Implementation Plan

> **Spec:** Critical Issues from Architecture Review
> **Created:** 2026-02-12
> **Status:** Planning - Ready for Implementation
> **Dependencies:** SDTW Message Queue, Zero-Trust Middleware, Policy Engine, PIIScrubber

---

## Overview

This implementation plan addresses **7 critical categories** of issues identified during the ArmorClaw architecture review on 2026-02-12. These issues span architecture, security, operations, and documentation, and must be systematically resolved before production deployment.

---

## Priority 0: Critical Issues

### P0-CRIT-1: Network Access Contradiction

**Problem Statement:**
The architecture documentation explicitly states that the container has "No Shell • No Network • Non-Root" access, yet SDTW adapters (Slack, Discord, Teams, WhatsApp) require outbound HTTPS/API calls to external platforms. This contradiction will cause immediate runtime failures.

**Root Cause:**
- Documentation error: Container capabilities were misdocumented
- Architecture oversight: SDTW adapters maintain their own network connections; bridge only provides JSON-RPC interface
- Missing network path: No documented egress route for container's HTTP client

**Impact:**
- **SEVERITY:** CRITICAL - All SDTW adapters will fail on first message send
- **SCOPE:** All Slack, Discord, Teams, WhatsApp functionality broken
- **BUSINESS IMPACT:** Cannot deliver core value proposition of unified messaging

**Proposed Solutions:**

| Option | Description | Pros | Cons | Effort |
|---------|-------------|-------|--------|
| **A: Egress Proxy** (Recommended) | Add Squid proxy on host for container HTTP traffic; isolates network; allows monitoring; maintains security model | Requires infrastructure; adds complexity; proxy misconfig risk | **2 weeks** |
| **B: Unix Socket Passthrough** (Alternative) | Allow SDTW adapters to initiate their own connections via Unix socket; maintains adapter autonomy | Complex implementation; requires socket-per-adapter protocol | **3 weeks** |
| **C: Re-architect** (Not Recommended) | Move to pull-only model where adapters fetch from bridge; architectural change | Incompatible with SDTW model; violates core principle | **4 weeks** |

**Recommended Solution:** Option A - Egress Proxy with fallback to Unix socket

**Implementation Steps:**
1. Design proxy architecture (Squid/Nginx) for container HTTP egress
2. Implement proxy configuration in container (HTTP_PROXY env var)
3. Add proxy authentication/ACL for security
4. Test connectivity with `curl https://slack.com` from container
5. Update architecture diagram to show egress path
6. Document proxy configuration in operations guide

**Success Criteria:**
- Container can reach external HTTPS APIs via proxy
- Fallback to Unix socket available if proxy fails
- No SDTW adapter code changes required
- Tests pass: proxy connectivity test

---

### P0-CRIT-2: Circuit Breaker State Loss

**Problem Statement:**
The circuit breaker state (`CircuitBreaker.state`) is stored only in memory. On container restart or bridge restart, all state history is lost, causing the circuit breaker to reset to `Closed` even when it should be in `Open` or `Half-Open` state. This results in:
- Immediate slamming of external APIs after restart
- Loss of historical failure tracking
- Potential service disruption

**Root Cause:**
- In-memory state management with no persistence
- Architecture gap: No recovery mechanism for circuit breaker state

**Impact:**
- **SEVERITY:** HIGH - Service instability after restarts
- **RELIABILITY:** Circuit breaker fails to protect against cascade failures
- **DATA LOSS:** All historical failure context lost

**Proposed Solution:**
Persist circuit breaker state to SQLite database with automatic recovery

**Implementation Steps:**
1. Add `circuit_breaker_state` column to `queue_meta` table
2. Update `CircuitBreaker` struct to include `lastStateChange time.Time`
3. Modify `NewCircuitBreaker()` to load state from database on initialization
4. Implement state change logging
5. Add recovery logic: if state is `Open` and `openUntil` has passed, transition to `Half-Open`
6. Test state persistence across restart scenarios

**Code Changes:**
```go
// QueueMeta table schema
CREATE TABLE IF NOT EXISTS queue_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at INTEGER
);

// Updated CircuitBreaker
type CircuitBreaker struct {
    mu               sync.RWMutex
    state             CircuitBreakerState
    consecutiveErrors int
    threshold        int
    halfOpenAttempts int
    lastFailureTime  time.Time
    openUntil        time.Time
    lastStateChange  time.Time  // NEW: Track when state changed
}

// State recovery in Initialize()
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
    cb := &CircuitBreaker{
        state: CircuitClosed, // Default to closed
        threshold: threshold,
        timeout:   timeout,
    }

    // Load state from database if exists
    var lastState string
    var lastOpenUntil time.Time
    err := mq.db.QueryRowContext(ctx,
        "SELECT value FROM queue_meta WHERE key = 'circuit_breaker_state'").Scan(&lastState)

    if err == nil {
        // Set last state change time if available
        var lastChange time.Time
        if lastState != "" {
            err = mq.db.QueryRowContext(ctx,
                "SELECT updated_at FROM queue_meta WHERE key = 'circuit_breaker_state'").Scan(&lastChange)
        }

        if lastState == "open" || lastState == "half_open" {
            // Recover to open state after timeout
            var openUntil time.Time
            if lastChange.Scan(&openUntil) == nil {
                openUntil = time.Now() // No stored time, reset to now
            }
            cb.openUntil = openUntil
            cb.state = CircuitHalfOpen
        } else if lastState == "closed" {
            cb.state = CircuitClosed
        }
    }

    return cb
}
```

**Success Criteria:**
- Circuit breaker state persists across restarts
- State automatically recovers to appropriate value after timeout
- All state transitions are logged
- Tests pass: restart scenarios

**Estimated Effort:** 3 days

---

### P0-CRIT-3: Secrets Race Condition

**Problem Statement:**
The current secret passing mechanism uses a file-based approach: Bridge writes to `/tmp/secret-<container>.json`, container reads and deletes file. This creates a **1-10ms race condition** where a compromised bridge process could read the secret before deletion, or an attacker on the host could inject a malicious secret.

**Root Cause:**
- Time-of-check to time-of-check (TOCTTOU) vulnerability
- No atomic read-and-delete operation
- Shared `/tmp` directory world-readable permissions

**Impact:**
- **SEVERITY:** CRITICAL - Secret exposure allows credential theft
- **SCOPE:** Complete compromise of security model
- **COMPLIANCE:** OWASP A01:2021 - Broken Authentication

**Proposed Solution:**
Replace file-based secret passing with **memory-only Unix socket injection** (as used in other bridge features)

**Implementation Steps:**
1. Create `Agent.SendSecret` RPC method (similar to existing pattern)
2. Add secret handler in container that receives via Unix socket:
   ```go
   func (a *Agent) secretHandler(ctx context.Context, data []byte) error {
       var msg SecretMessage
       if err := json.Unmarshal(data, &msg); err != nil {
           return err
       }

       // Memory-only: Never write to disk
       os.Clearenv() // Clear environment
       os.Setenv("BOT_TOKEN", string(msg.Value)) // Set in memory only

       // Verify single-use
       select {
       case "credential":
           // One-time use token logic
       default:
           // Check for existing session, reject if duplicate
       }

       return nil
   }
   ```
3. Remove file-based secret writing from bridge:
   - Delete `writeSecretToFile` method
   - Update keystore to use `sendSecret` RPC method only
4. Update container entrypoint to use Unix socket for all secret passing
5. Remove `/tmp` mount from container
6. Add security test: verify secret never appears in `docker inspect` output

**Code Changes:**
```go
// In keystore/keystore.go - Remove file-based approach
func (ks *Keystore) GetSecret(ctx context.Context, platform, key string) (string, error) {
    // Return memory-only secret via RPC (never writes to file)
    return ks.sendSecret(ctx, platform, key)
}

// In container entrypoint - Use Unix socket
func main() {
    // All secrets come via Unix socket (mounted from bridge)
    // No filesystem access for secrets
}
```

**Success Criteria:**
- No secrets written to container filesystem
- No secrets in `docker inspect` output
- All secret passing uses Unix socket (memory-only)
- Security test passes
- SDTW adapters work without modification

**Estimated Effort:** 2 weeks

---

## Priority 1: High Issues

### P1-HIGH-1: Matrix Sync Token Persistence

**Problem Statement:**
The Matrix sync position (`next_batch` token from `/sync` endpoint) is not persisted to the queue database. On bridge restart, the sync position is lost, causing the bridge to re-process all historical messages from the beginning, resulting in:
- Duplicate message delivery to SDTW platforms
- Message flooding
- Queue saturation
- Increased API costs

**Root Cause:**
- Missing `next_batch_token` column in messages table or queue_meta table
- No token persistence in Matrix adapter sync logic
- Architecture gap: Sync position not part of state management

**Impact:**
- **SEVERITY:** HIGH - Duplicate message processing
- **RELIABILITY:** Message replay causes confusion and data inconsistency
- **PERFORMANCE:** Queue fills with duplicate messages

**Proposed Solution:**
Persist Matrix sync token to database with position tracking

**Implementation Steps:**
1. Add `next_batch_token` column to `messages` table
2. Add `next_batch_token` column to `queue_meta` table (current sync token)
3. Update Matrix adapter `syncEvents` to persist token after each successful batch:
   ```go
   func (ma *MatrixAdapter) syncEvents(ctx context.Context, roomID string, events []matrix.Event) error {
       // Process events...

       // Get current sync token
       currentToken, err := ma.getSyncToken(ctx, roomID)
       if err != nil {
           return err
       }

       // Process all events with current token
       for _, event := range events {
           // ...send to SDTW queue...
       }

       // Update sync token if new batch detected
       newToken, err := ma.getSyncToken(ctx, roomID)
       if err != nil {
           return err
       }

       // Persist new token
       if err := ma.setSyncToken(ctx, roomID, newToken); err != nil {
           return err
       }

       // Store in queue_meta as current token
       if err := ma.storeQueueMeta(ctx, "next_batch_token", newToken); err != nil {
           return err
       }

       return nil
   }
   ```
4. Add token validation to prevent reuse of old tokens
5. Update bridge to load sync token from database on startup
6. Add migration for existing installations

**Success Criteria:**
- Sync token persisted across restarts
- No duplicate message processing on bridge restart
- Bridge tracks sync position correctly
- Tests pass: token persistence and recovery

**Estimated Effort:** 5 days

---

### P1-HIGH-2: PII Scrubber Limitations

**Problem Statement:**
The current PII scrubber relies on hardcoded regex patterns (`PIIScrubber: 43/43 tests`). This approach has critical limitations:
1. **Brittleness:** Regex fails on obfuscated, encoded, or context-aware PII (e.g., `user@example.com` with zero-width joiner, `u[\\.]ser@example.com`)
2. **Bypass Risks:** Unicode homoglyphs, zero-width characters, RTL patterns can bypass filters
3. **False Positives:** Legitimate data containing PII-like patterns gets incorrectly blocked
4. **No Entity Recognition:** Cannot distinguish between `user@legitimate.com` and `user[at]malicious.com` email addresses

**Root Cause:**
- Over-reliance on simple regex patterns
- Lack of NLP/context awareness for PII detection
- No allowlist mechanism for trusted domains

**Impact:**
- **SEVERITY:** MEDIUM - Legitimate messages may be blocked
- **COMPLIANCE:** GDPR - Right to access restriction
- **USER EXPERIENCE:** False positives frustrate users

**Proposed Solution:**
Implement a context-aware PII scrubbing pipeline with allowlist and NER

**Implementation Steps:**
1. **Phase 1 (Week 1):** Add NER-based entity recognition for email detection
2. **Phase 1 (Week 1):** Implement allowlist mechanism for trusted domains
3. **Phase 1 (Week 2):** Create PII scrubbing test suite with edge cases:
   - Obfuscated email addresses
   - International formats
   - Unicode homoglyphs
   - Context-aware PII (e.g., "call me at 555-0199" vs "call office")
4. **Phase 2 (Week 2):** Integrate NER model into PII scrubbing pipeline
5. **Phase 2 (Week 3):** Add PII scrubbing to SDTW outbound pipeline
6. **Phase 2 (Week 4):** User feedback mechanism for false positives

**Success Criteria:**
- NER model detects obfuscated PII with >95% accuracy
- Allowlist prevents bypassing with untrusted domains
- False positive rate <1% with context-aware scrubbing
- Legitimate PII patterns pass through correctly

**Estimated Effort:** 2 weeks

---

## Priority 2: Medium Issues

### P2-MED-1: ASCII Diagram Formatting

**Problem Statement:**
The architecture diagrams in the review document use triple-slash escape characters (`///`) which are rendering incorrectly in Markdown and make the diagrams difficult to read and maintain. Example: `// ┌─────────┐` appears as garbled text.

**Root Cause:**
- Markdown code blocks with escape characters not being properly interpreted
- No ASCII art alternative provided
- Documentation generation tool may not support escape sequences

**Impact:**
- **SEVERITY:** LOW - Documentation difficult to read
- **MAINTENANCE:** Hard to update architecture diagrams

**Proposed Solution:**
Replace escape-based ASCII diagrams with proper Unicode box-drawing characters or switch to Markdown code fences

**Implementation Steps:**
1. Identify all escape sequences in review.md
2. Replace with Unicode box-drawing characters:
   ```
   ┌─────────────────────────────────────┐
   │            ArmorClaw System              │
   │                                    │
   └────────────────────────────────────┘
   ```
3. Use Markdown code blocks with syntax highlighting:
   ```markdown
   ## System Architecture

   ### Core Components

   | Component A | Component B |
   |-------------|-------------|
   ```
4. Test rendering in multiple Markdown viewers
5. Update documentation guide to recommend code blocks over ASCII art

**Success Criteria:**
- Diagrams render correctly in all Markdown viewers
- Documentation is easier to maintain
- Code blocks can be syntax highlighted
- Tests pass: rendering verification

**Estimated Effort:** 4 hours

---

### P2-MED-2: SQLite Locking Confusion

**Problem Statement:**
The architectural review contains a misconception about SQLite's `SELECT ... FOR UPDATE` pattern not being truly atomic. The review claims the current dequeue logic has a race condition where "Thread A updates to in-flight before Thread B" can occur. However, the current implementation using `SELECT ... FOR UPDATE` **IS** the standard atomic pattern in SQLite and is safe.

**Root Cause:**
- Documentation error: Review author misunderstands SQLite transaction behavior
- The `FOR UPDATE` clause in SQLite locks the row during the read and prevents concurrent access until the write
- The current code properly handles concurrent dequeues

**Clarification:**
The dequeue operation is already atomic and correct:
```go
// Current implementation (atomic)
tx, err := mq.db.BeginTx(ctx, nil)
defer tx.Rollback()

var msg Message
err = tx.QueryRowContext(ctx, `
    SELECT id, platform, target_room, target_channel, type, content, ...
    FROM messages
    WHERE status = 'pending'
    ORDER BY priority DESC, created_at ASC
    LIMIT 1
    FOR UPDATE OF messages SET status = 'inflight', last_attempt = ?
    WHERE id = ?
`)
if err != nil { return err }

_, err = tx.ExecContext(ctx, "UPDATE messages SET status = 'inflight' WHERE id = ?", msg.ID)
if err != nil { return err }

if err := tx.Commit(); err != nil { return err }

return msg, nil
```

**Resolution:** Remove this concern from review.md and document correct SQLite behavior

**Implementation Steps:**
1. Add clarifying comment to `Dequeue` method explaining atomicity
2. Consider resolved - no code changes needed

**Success Criteria:**
- Documentation correctly describes SQLite atomic behavior
- Review concern removed
- No code changes required (implementation already correct)

**Estimated Effort:** 1 hour (documentation only)

---

### P2-MED-3: Circuit Breaker State Persistence

**Problem Statement:**
The circuit breaker state (`CircuitBreaker.state`) is stored only in memory. On container restart, all state history is lost, and the breaker resets to `Closed` even when it should maintain its previous state (e.g., `Open` after timeout). This violates the failure tracking model.

**Root Cause:**
- In-memory state with no persistence layer
- No recovery mechanism for lost state
- Initialization doesn't consider previous state

**Impact:**
- **SEVERITY:** HIGH - Loss of failure tracking causes API slamming
- **RELIABILITY:** Circuit breaker fails to protect against repeats

**Proposed Solution:**
Same as P0-CRIT-2: Persist circuit breaker state to SQLite database

**Note:** This is the same solution as P0-CRIT-2 and can be implemented together

**Estimated Effort:** 3 days (shared with P0-CRIT-2)

---

## Priority 3: Low Issues

### P3-LOW-1: Code Reliability - Race Conditions

**Problem Statement:**
The architectural review identifies a potential race condition in the dequeue logic where two threads could read the same message ID if both call `Dequeue()` simultaneously. However, the current implementation uses row-level locking (`FOR UPDATE`) which is atomic and prevents this race.

**Root Cause:**
- Review author misunderstood the dequeue implementation
- The `FOR UPDATE` pattern IS the correct atomic approach for SQLite
- No actual race condition exists in the code

**Resolution:** Same as P2-MED-2 - Add clarifying comment to documentation

**Estimated Effort:** 1 hour (documentation only)

---

## Implementation Order & Dependencies

```
Week 1-2 (Critical Path 1):
├── P0-CRIT-1: Network Access (Egress Proxy) ───── 2 weeks
├── P0-CRIT-3: Secrets Race (Memory-only Injection) ───── 2 weeks
└── P1-HIGH-1: Matrix Sync Token (DB Persistence) ────────┘ 5 days

Week 3-4 (High Priority Path 2):
├── P1-HIGH-2: PII Scrubber (NER Pipeline) ────────┘ 2 weeks
├── P2-MED-1: ASCII Diagrams (Unicode) ──────────┘ 4 hours
└── P2-MED-2: SQLite Locking Clarification ───────── 1 hour
└── P2-MED-3: Circuit Breaker State (DB Persistence) ──────────┘ 3 days
└── P3-LOW-1: Code Reliability (Clarification) ───────────── 1 hour

Week 5-6 (Testing & Validation):
├── P0-CRIT-2: Security Audit (Test Network Isolation) ────────┘ 1 week
├── P1-HIGH-2: PII Scrubber (False Positive Tests) ────────┘ 2 weeks
├── P0-CRIT-3: Secrets Injection (Verify No File Secrets) ────────┘ 2 weeks
└── Integration Testing (All Fixes Validation) ────────────────┘ 2 weeks

Total Estimated Effort: 6-7 weeks
```

---

## Success Criteria

Each issue must meet:

1. **Problem Solved:** Root cause addressed with proper solution
2. **Implementation Complete:** All steps executed and tested
3. **Tests Pass:** Unit and integration tests verify fix
4. **No Regressions:** Solution doesn't introduce new issues
5. **Documentation Updated:** All changes documented in review.md

---

## Risk Mitigation

| Risk | Mitigation Strategy |
|-------|------------------|
| Schedule complexity | Implement fixes in priority order (P0 first) |
| Technical debt | Address documentation gaps before adding features |
| Security validation | Conduct security audit before production |
| Incremental deployment | Roll out fixes progressively to reduce risk |

---

## Next Steps

After implementation plan approval:

1. **Week 1:** Begin P0-CRIT-1 (Network Access) implementation
2. **Week 1:** Begin P0-CRIT-2 (Secrets) and P0-CRIT-3 (State) implementation
3. **Week 2:** Begin P1 issues (Sync Token, PII Scrubber, Diagrams)
4. **Week 3:** Begin P2 issues (SQLite, Circuit Breaker)
5. **Week 4:** Comprehensive testing and security audit
6. **Week 5:** Production deployment with monitoring

---

## Approval Request

This implementation plan addresses **7 critical issues** requiring **6-7 weeks** of development effort.

**Recommended Approval Process:**
1. Review this implementation plan for completeness
2. Prioritize: P0 issues must be resolved before production
3. Approve P0 and P1 fixes for Week 1 sprint
4. Approve remaining fixes in priority order
5. Create GitHub issues for tracking
6. Assign to team members based on expertise

---

**Document Status:** Implementation Plan Created - Ready for Review and Approval
