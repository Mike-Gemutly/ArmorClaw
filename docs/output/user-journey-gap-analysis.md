# User Journey Gap Analysis

> **Date:** 2026-02-15
> **Version:** 2.0.0
> **Status:** Updated After Sprint 1 + Error Handling Implementation

---

## Executive Summary

This analysis maps the complete user journey across ArmorClaw features and identifies gaps in the user experience.

**Previous State (2026-02-14):**
- 27 documented user stories
- 11 critical gaps
- Journey Health: ⚠️ NEEDS ATTENTION

**Current State (2026-02-15):**
- 27 documented user stories
- **5 gaps resolved** (GAP #2, #6, #7, #8, #9)
- **6 gaps remain** (GAP #1, #3, #4, #5, #10, #11)
- Journey Health: ✅ **IMPROVED** (81% → 89%)

---

## Resolved Gaps Summary

### ✅ GAP #2: Platform Support Documentation
**Resolved:** 2026-02-14
**Resolution:** 12 platform deployment guides created

| Platform | Guide | Status |
|----------|-------|--------|
| AWS Fargate | `aws-fargate-deployment.md` | ✅ |
| Azure | `azure-deployment.md` | ✅ |
| DigitalOcean | `digitalocean-deployment.md` | ✅ |
| Fly.io | `flyio-deployment.md` | ✅ |
| GCP Cloud Run | `gcp-cloudrun-deployment.md` | ✅ |
| Hostinger KVM | `hostinger-deployment.md` | ✅ |
| Hostinger Docker | `hostinger-docker-deployment.md` | ✅ |
| Hostinger VPS | `hostinger-vps-deployment.md` | ✅ |
| Linode | `linode-deployment.md` | ✅ |
| Railway | `railway-deployment.md` | ✅ |
| Render | `render-deployment.md` | ✅ |
| Vultr | `vultr-deployment.md` | ✅ |

---

### ✅ GAP #6: Account Recovery Flow
**Resolved:** 2026-02-14
**Resolution:** Complete recovery system implemented

**Implementation:**
- `bridge/pkg/recovery/recovery.go` - Recovery manager
- 12-word BIP39-style recovery phrase
- Encrypted storage (ChaCha20-Poly1305)
- 48-hour recovery window
- Device invalidation on completion

**RPC Methods:**
1. `recovery.generate_phrase` - Generate new phrase
2. `recovery.store_phrase` - Store encrypted phrase
3. `recovery.verify` - Start recovery
4. `recovery.status` - Check recovery status
5. `recovery.complete` - Finalize recovery
6. `recovery.is_device_valid` - Check device validity

---

### ✅ GAP #7: Error Escalation Flow
**Resolved:** 2026-02-15
**Resolution:** Complete error handling system implemented

**Implementation:**
- `bridge/pkg/errors/` - Full error handling package
- Structured error codes (CTX-XXX, MAT-XXX, RPC-XXX, SYS-XXX, BGT-XXX, VOX-XXX)
- Component-scoped event tracking
- Smart sampling with rate limiting
- 3-tier admin resolution chain
- SQLite persistence
- LLM-friendly notification format

**RPC Methods:**
1. `get_errors` - Query errors with filters
2. `resolve_error` - Mark errors resolved

**Integration Points:**
- Docker client (CTX-XXX errors)
- Matrix adapter (MAT-XXX errors)
- Bridge main (initialization)

---

### ✅ GAP #8: Platform Onboarding Wizard
**Resolved:** 2026-02-14
**Resolution:** Complete platform onboarding documentation

**Implementation:**
- `docs/guides/platform-onboarding.md` - Platform setup guide
- Step-by-step Slack integration
- Step-by-step Discord integration
- Step-by-step Microsoft Teams integration
- Step-by-step WhatsApp Business API integration
- OAuth flow documentation
- Security considerations

**RPC Methods:**
1. `platform.connect` - Connect external platform
2. `platform.disconnect` - Disconnect platform
3. `platform.list` - List connected platforms
4. `platform.status` - Check platform status
5. `platform.test` - Test platform connection

---

### ✅ GAP #9: Slack Adapter Implementation
**Resolved:** 2026-02-14
**Resolution:** Complete Slack adapter implemented

**Implementation:**
- `bridge/internal/adapter/slack.go` - Slack adapter
- Slack Web API integration
- Bot authentication (xoxb- tokens)
- Channel listing and management
- Message sending with blocks/attachments
- Conversation history retrieval
- User info caching
- Rate limit handling
- Background sync loop

---

## Updated User Journey Map

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ARMORCLAW USER JOURNEY MAP                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PHASE 1: DISCOVERY & SETUP                                                  │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │ Discover │───▶│ Install  │───▶│ Configure│───▶│ Deploy  │                  │
│  │         │    │ Bridge   │    │ API Keys │    │Container│                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘                  │
│       │              │              │              │                         │
│       ▼              ▼              ▼              ▼                         │
│   [GAP #1]       [✅ OK]       [✅ OK]       [✅ OK]                        │
│   Entry         Platform      Setup          Pre-validation                 │
│   point         docs now      wizard         implemented                    │
│   missing       complete      exists                                        │
│                                                                              │
│  PHASE 2: CONNECTION & VERIFICATION                                          │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │ Connect │───▶│ Verify  │───▶│ Device  │───▶│ Trust   │                  │
│  │ Matrix  │    │Device   │    │ Setup   │    │ Anchor  │                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘                  │
│       │              │              │              │                         │
│       ▼              ▼              ▼              ▼                         │
│   [✅ OK]       [GAP #4]      [GAP #5]      [✅ OK]                        │
│   Element X     QR Scanning   Multi-device  Recovery                       │
│   quickstart    incomplete    UX missing    implemented                    │
│                                                                              │
│  PHASE 3: DAILY USAGE                                                        │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │  Chat   │───▶│ Sync    │───▶│ Status  │───▶│ Error   │                  │
│  │ Messages│    │ Messages│    │ Display │    │ Handling│                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘                  │
│       │              │              │              │                         │
│       ▼              ▼              ▼              ▼                         │
│   [✅ OK]       [✅ OK]       [✅ OK]       [✅ OK]                         │
│   E2EE chat     Queue system  Visual sync   Error system                   │
│   working       implemented  status spec    complete                       │
│                                                                              │
│  PHASE 4: MULTI-PLATFORM (SDTW)                                              │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │ Connect │───▶| Platform │───▶| Message │───▶| Monitor │                  │
│  │Platform │    │Adapter   │    │Queue    │    │Health   │                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘                  │
│       │              │              │              │                         │
│       ▼              ▼              ▼              ▼                         │
│   [✅ OK]       [✅ OK]        [✅ OK]       [GAP #10]                      │
│   Platform      Slack         Queue         Alert                           │
│   onboarding    adapter       complete      integration                     │
│   complete      implemented                 missing                         │
│                                                                              │
│  PHASE 5: SECURITY & MAINTENANCE                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │ Upgrade │───▶| Audit   │───▶| Device  │───▶| Account │                  │
│  │Security │    │Trail    │    │Remove   │    │Recovery │                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘                  │
│       │              │              │              │                         │
│       ▼              ▼              ▼              ▼                         │
│   [GAP #11]     [✅ OK]       [✅ OK]       [✅ OK]                        │
│   Security      Logging       Device        Recovery                       │
│   tier UX       complete      removal       implemented                    │
│   missing                     spec'd                                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Remaining Gap Analysis

### GAP #1: Clear Entry Point for New Users
**Location:** Discovery → Setup
**Severity:** HIGH
**Status:** ⚠️ NOT RESOLVED

**Problem:** No "Getting Started" guide for new users

**Current State:**
- README.md provides technical overview
- Setup guide exists but assumes technical knowledge
- No interactive onboarding

**Impact:**
- Users may not understand the value proposition
- High drop-off rate during initial setup
- Confusion about what ArmorClaw does vs ArmorChat

**Recommendation:**
Create `docs/guides/getting-started.md` that:
1. Explains the security model visually
2. Shows the architecture diagram
3. Provides a "Try it in 5 minutes" quickstart
4. Links to video tutorials

---

### GAP #3: Pre-Validation Implementation Status
**Location:** Configure → Deploy
**Severity:** MEDIUM
**Status:** ⚠️ PARTIALLY RESOLVED

**Current State:**
- ✅ Setup wizard validates API keys
- ✅ Memory-only secret injection implemented
- ✅ Unix socket-based secret delivery working
- ⚠️ Test API call validation not fully implemented

**Remaining Work:**
1. Add actual API call validation (lightweight models endpoint)
2. Add key expiration date detection
3. Add key usage quota warnings

---

### GAP #4: QR Scanning Flow Incomplete
**Location:** Verify Device
**Severity:** HIGH
**Status:** ⚠️ NOT RESOLVED

**Current State:**
- QR code generation implemented
- QR scanning UI spec exists
- Implementation NOT complete

**Impact:**
- Users must manually enter configuration
- Slower device pairing
- Higher error rate during setup

**Recommendation:**
1. Implement camera permission handling
2. Add QR validation feedback
3. Add fallback to manual setup

---

### GAP #5: Multi-Device UX Missing
**Location:** Device Setup → Trust Anchor
**Severity:** MEDIUM
**Status:** ⚠️ NOT RESOLVED

**Current State:**
- Device listing exists
- Device verification stories defined
- Cross-device trust flow incomplete

**Impact:**
- Users don't understand device relationships
- No visual indication of trust chain
- Confusion when adding 2nd/3rd device

**Recommendation:**
1. Create device trust visualization
2. Add "Add another device" flow
3. Implement verification code exchange
4. Show trust anchor indicator

---

### GAP #10: Alert Integration Missing
**Location:** Monitor Health
**Severity:** MEDIUM
**Status:** ⚠️ NOT RESOLVED

**Current State:**
- Prometheus metrics defined
- Metrics collection implemented
- No alert rules defined
- No notification integration

**Note:** Error system can send Matrix notifications, but alert rules not configured.

**Impact:**
- Issues detected only after user complaints
- No proactive monitoring
- DLQ growth goes unnoticed

**Recommendation:**
1. Define alert rules for key metrics
2. Integrate with existing Matrix notification system
3. Create alert severity levels
4. Document runbooks for common alerts

---

### GAP #11: Security Tier Upgrade UX Missing
**Location:** Upgrade Security
**Severity:** LOW
**Status:** ⚠️ NOT RESOLVED

**Current State:**
- Progressive security tiers defined
- No user-facing upgrade notifications
- No tier benefits explanation
- Manual tier management

**Impact:**
- Users unaware of enhanced security options
- Lower overall security posture
- Missed opportunity for security education

**Recommendation:**
1. Add in-app notifications for tier eligibility
2. Create tier benefits comparison
3. Add one-tap upgrade flow

---

## Journey Transition Matrix (Updated)

| From Phase | To Phase | Transition Story | Status |
|------------|----------|------------------|--------|
| Discovery | Setup | "I found ArmorClaw, how do I start?" | ⚠️ GAP #1 |
| Setup | Connection | "Bridge installed, how do I connect?" | ✅ Complete |
| Connection | Verification | "Connected, how do I verify?" | ⚠️ GAP #4 |
| Verification | Daily Usage | "Verified, ready to chat!" | ✅ Complete |
| Daily Usage | Multi-Platform | "Can I connect Slack too?" | ✅ Complete |
| Multi-Platform | Security | "How do I improve security?" | ⚠️ GAP #11 |
| Security | Recovery | "Lost my devices, help!" | ✅ Complete |
| Any | Error | "Something went wrong" | ✅ Complete |
| Monitoring | Alerts | "How do I know when things break?" | ⚠️ GAP #10 |

---

## Priority Recommendations (Updated)

### P0 - Critical (Block Production) - ALL RESOLVED ✅

| Gap | Status | Resolution Date |
|-----|--------|-----------------|
| GAP #6: Account Recovery | ✅ Resolved | 2026-02-14 |
| GAP #8: Platform Onboarding | ✅ Resolved | 2026-02-14 |
| GAP #9: Adapter Implementation | ✅ Resolved | 2026-02-14 |

### P1 - High (Degraded Experience)

| Gap | Status | Priority |
|-----|--------|----------|
| GAP #1: Entry Point | ⚠️ Open | HIGH |
| GAP #4: QR Scanning | ⚠️ Open | HIGH |
| GAP #7: Error Escalation | ✅ Resolved | 2026-02-15 |

### P2 - Medium (Improvement)

| Gap | Status | Priority |
|-----|--------|----------|
| GAP #2: Platform Support | ✅ Resolved | 2026-02-14 |
| GAP #5: Multi-Device UX | ⚠️ Open | MEDIUM |
| GAP #10: Alert Integration | ⚠️ Open | MEDIUM |
| GAP #3: Pre-Validation | ⚠️ Partial | MEDIUM |

### P3 - Low (Nice to Have)

| Gap | Status | Priority |
|-----|--------|----------|
| GAP #11: Security Tier UX | ⚠️ Open | LOW |

---

## Feature Connection Analysis

### Critical Feature Chains

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     FEATURE CONNECTION MAP                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CHAIN 1: Setup → First Message (PRIMARY FLOW)                               │
│  ══════════════════════════════════════════                                  │
│  [Config] → [Keystore] → [Docker] → [Container] → [Matrix] → [Message]      │
│     ✅          ✅          ✅         ✅            ✅          ✅           │
│                                                                              │
│  CHAIN 2: Error Detection → Resolution (NEW)                                 │
│  ═════════════════════════════════════                                        │
│  [Error] → [Sampling] → [Tracking] → [Persist] → [Notify] → [RPC Query]     │
│     ✅         ✅          ✅          ✅          ✅          ✅              │
│                                                                              │
│  CHAIN 3: Recovery Flow (NEW)                                                │
│  ══════════════════════════                                                  │
│  [Lost Device] → [Recovery Phrase] → [Verify] → [Restore Access]            │
│        ✅              ✅               ✅            ✅                      │
│                                                                              │
│  CHAIN 4: Multi-Platform (SDTW)                                              │
│  ══════════════════════════════                                              │
│  [Connect] → [OAuth] → [Adapter] → [Queue] → [Bridge] → [Matrix]            │
│     ✅         ✅        ✅         ✅         ✅         ✅                   │
│                                                                              │
│  CHAIN 5: Monitoring & Alerts (PARTIAL)                                      │
│  ═════════════════════════════════════                                       │
│  [Metrics] → [Collection] → [Storage] → [Alert Rules] → [Notify]            │
│     ✅           ✅             ✅          ❌            ✅                   │
│                                                          ↑                   │
│                                              GAP #10: Rules Missing          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Updated Summary

| Category | Previous | Current | Change |
|----------|----------|---------|--------|
| Total Gaps | 11 | 6 | -5 (45% reduction) |
| Critical (P0) | 3 | 0 | -3 |
| High (P1) | 3 | 2 | -1 |
| Medium (P2) | 3 | 4 | +1 |
| Low (P3) | 2 | 0 | -2 |
| Stories with Implementation | 16 (59%) | 24 (89%) | +30% |

**Overall Journey Health:** ✅ **IMPROVED**

### What's Working Well:
1. ✅ Core bridge functionality is solid
2. ✅ Error handling system is complete
3. ✅ Recovery flow prevents permanent lockouts
4. ✅ Multi-platform support via Slack adapter
5. ✅ Platform deployment options are documented
6. ✅ Matrix E2EE messaging is functional

### Remaining Concerns:
1. ⚠️ New user onboarding experience needs improvement
2. ⚠️ QR scanning flow incomplete
3. ⚠️ Multi-device UX needs polish
4. ⚠️ Alert rules not configured for proactive monitoring

---

## Next Steps (Sprint 2)

### Priority 1: User Onboarding (GAP #1)
1. Create `docs/guides/getting-started.md`
2. Add architecture diagram to README
3. Create 5-minute quickstart guide

### Priority 2: Device Experience (GAP #4, #5)
1. Implement QR scanning UI
2. Add device trust visualization
3. Implement verification code flow

### Priority 3: Monitoring (GAP #10)
1. Define alert rules for critical metrics
2. Integrate with Matrix notification system
3. Create operational runbooks

---

**Document Last Updated:** 2026-02-15
**Next Review:** After Sprint 2 completion
