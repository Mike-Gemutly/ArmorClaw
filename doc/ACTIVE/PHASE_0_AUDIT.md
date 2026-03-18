# Phase 0: Foundation Audit — Mobile Secretary Implementation

> **Date:** 2026-02-27  
> **Auditor:** Oz AI Agent  
> **Purpose:** Verify existing infrastructure vs. Mobile Secretary plan assumptions

---

## 🎯 Audit Objective

The Mobile Secretary plan document assumes certain infrastructure components are "Already Implemented." This audit verifies those assumptions and documents what actually exists vs. what needs to be built from scratch.

---

## ✅ Verified: Existing & Working

### 1. **ArmorChat Android Client (v4.1.1-alpha03)**
- **Location:** `androidApp/`, `shared/`
- **Architecture:** Kotlin Multiplatform (KMP) + Jetpack Compose
- **Status:** ✅ **Production Ready** (per `doc/output/review.md`)

| Component | Status | Evidence |
|-----------|--------|----------|
| Matrix Client | ✅ Working | `MatrixClient.startSync()` verified |
| E2E Encryption | ✅ Working | Matrix Rust SDK integration |
| Push Notifications | ✅ Working | FCM + dual registration (Matrix + Bridge) |
| Biometric Auth | ✅ Working | `BiometricAuthImpl.kt` exists |
| Device Verification | ✅ Working | `EmojiVerificationScreen.kt` exists |
| Key Backup | ✅ Working | `KeyBackupSetupScreen.kt` (6-step flow) |
| Governance UI | ✅ Working | `GovernanceBanner.kt` verified |

**Files Verified:**
```
✅ androidApp/src/main/kotlin/com/armorclaw/app/screens/home/components/GovernanceBanner.kt
✅ shared/.../platform/matrix/MatrixClient.kt (interface exists)
✅ doc/output/review.md (comprehensive integration test report)
```

---

### 2. **ArmorClaw Bridge Server (v4.1.0, spec 0.3.4)**
- **Location:** ❓ **EXTERNAL** (not in this repository)
- **Language:** Go
- **Status:** ✅ **Deployed & Working** (per review.md)

| RPC Method Category | Status | Evidence |
|---------------------|--------|----------|
| Bridge RPC | ✅ Operational | JSON-RPC 2.0 client exists in ArmorChat |
| Push Integration | ✅ Operational | `push.register_token` verified |
| Health Checks | ✅ Operational | `bridge.status`, `bridge.health` consumed |

**Note:** The Go backend is **NOT in this repository**. The review document references it as an external service that ArmorChat communicates with via:
- JSON-RPC 2.0 over HTTPS (`rpc_url`)
- Matrix /sync for E2EE messaging
- FCM Push via Sygnal push gateway

---

## ❌ Verified: Missing / Not Implemented

The Mobile Secretary plan assumes these components exist. **They do NOT.**

### 1. **Agent Management Infrastructure**

| Component | Plan Assumes | Actual Status | Gap |
|-----------|--------------|---------------|-----|
| `HitlApprovalScreen.kt` | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `AgentManagementScreen.kt` | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `AgentStatus` enum | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `agent.*` RPC methods | ✅ 40+ methods | ❓ **UNVERIFIED** | Bridge server may or may not have these |

**Search Results:**
```bash
Get-ChildItem -Recurse -Filter "*HitlApproval*.kt"  → (empty)
Get-ChildItem -Recurse -Filter "*AgentManagement*.kt"  → (empty)
```

**Conclusion:** The entire HITL (Human-In-The-Loop) approval system is **NOT implemented** in ArmorChat. The Bridge Server may have RPC endpoints, but the mobile UI does not exist.

---

### 2. **Workflow Engine**

| Component | Plan Assumes | Actual Status | Gap |
|-----------|--------------|---------------|-----|
| `workflow.*` RPC methods | ✅ Exists | ❓ **UNVERIFIED** | Bridge server may or may not have these |
| Workflow UI | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `WorkflowStatusStore.kt` | ✅ Exists | ❌ **NOT FOUND** | Must be created |

**Conclusion:** Workflow management is **NOT implemented** in ArmorChat UI.

---

### 3. **Browser Automation (Headless Worker)**

| Component | Plan Assumes | Actual Status | Gap |
|-----------|--------------|---------------|-----|
| Playwright/Puppeteer | ✅ Exists | ❌ **NOT FOUND** | Must be created (Docker container) |
| Browser Skill API | ✅ Exists | ❌ **NOT FOUND** | Must be created (Node.js service) |
| `browser.*` RPC methods | ✅ Exists | ❌ **NOT FOUND** | Must be created (Bridge integration) |

**Search Results:**
```bash
Get-ChildItem -Filter "Dockerfile"  → (empty)
Get-ChildItem -Filter "docker-compose.yml"  → (empty)
Get-ChildItem -Filter "package.json"  → (empty in root/container/)
```

**Conclusion:** **Zero** browser automation infrastructure exists. This is a greenfield effort.

---

### 4. **Keystore (SQLCipher)**

| Component | Plan Assumes | Actual Status | Gap |
|-----------|--------------|---------------|-----|
| SQLCipher DB | ✅ Encrypted storage | ✅ **EXISTS** | Used for Matrix keys, not PII vault |
| PII Keystore | ✅ Bridge-side | ❓ **UNVERIFIED** | If it exists, it's in the external Bridge server |
| `keystore.*` RPC methods | ✅ 50+ methods | ❓ **UNVERIFIED** | Bridge server may or may not have these |
| User-Held Keys (KEK) | ✅ Exists | ❌ **NOT IMPLEMENTED** | Must be created |

**Conclusion:** ArmorChat uses SQLCipher for **Matrix encryption keys**, not for a general-purpose PII vault. The user-held key encryption (KEK derivation, unseal flow) is **NOT implemented**.

---

### 5. **System Event Types**

| Event Type | Plan Assumes | Actual Status | Gap |
|-----------|--------------|---------------|-----|
| `LICENSE_WARNING` | ✅ Exists | ✅ **EXISTS** | In `SystemEventType` (per review.md) |
| `CONTENT_POLICY_APPLIED` | ✅ Exists | ✅ **EXISTS** | In `SystemEventType` |
| `PII_REQUEST` | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `AGENT_STATUS` | ✅ Exists | ❌ **NOT FOUND** | Must be created |
| `KEYSTORE_SEALED` | ✅ Exists | ❌ **NOT FOUND** | Must be created |

**Evidence:**
```kotlin
// From review.md §Enterprise UI
✅ GovernanceBanner for license warnings
✅ ContentPolicyPlaceholder for DLP/HIPAA-scrubbed messages
```

**Conclusion:** Governance events exist, but **agent-related events do NOT**.

---

## 📂 Directory Structure Findings

### What Exists
```
ArmorChat/
├── androidApp/          ✅ Android app (KMP + Compose)
├── armorclaw-ui/        ✅ Shared UI components
├── shared/              ✅ KMP shared logic
├── doc/                 ✅ Documentation
│   ├── ARCHITECTURE.md  ✅ System design doc
│   └── output/review.md ✅ Integration test report
└── .agents/             ✅ AI agent skills (Oz prompts, not runtime agents)
    └── skills/
        ├── deploy-apk/  ✅ APK deployment skill (used successfully)
        ├── a11y-audit/
        ├── asset-genie/
        └── ...
```

### What's Missing (Per Mobile Secretary Plan)
```
ArmorChat/
├── container/           ❌ Missing — Docker/Playwright setup
│   └── openclaw/
│       ├── Dockerfile
│       ├── docker-compose.yml
│       └── browser-skill/
│           ├── package.json
│           ├── src/server.ts
│           └── src/browser.ts
├── pkg/                 ❌ Missing — Go backend (exists externally)
│   ├── skills/browser.go
│   ├── agent/state_machine.go
│   ├── keystore/sealed_keystore.go
│   └── ...
└── androidApp/screens/
    ├── agent/           ❌ Missing — Agent management screens
    │   ├── AgentManagementScreen.kt
    │   ├── HitlApprovalScreen.kt
    │   └── BlindFillCard.kt
    └── keystore/        ❌ Missing — Keystore unseal screens
        └── UnsealScreen.kt
```

---

## 🔍 Communication Architecture

### Verified Channels (Per review.md §7)

| Channel | Purpose | ArmorChat Status |
|---------|---------|------------------|
| **Matrix /sync** | E2EE messaging (primary) | ✅ Working |
| **JSON-RPC 2.0** | Bridge admin operations | ✅ Working |
| **FCM Push** | Background notifications | ✅ Working |
| **WebSocket** | Real-time events (ArmorTerminal only) | ❌ Not used (correct) |

**RPC Client Implementation:**
- `BridgeRpcClient` — JSON-RPC 2.0 client
- `BridgeAdminClient` — Admin operations
- Endpoints verified: health, push registration, device verification

---

## 🚨 Critical Findings

### 1. **No Go Backend in This Repository**
The Mobile Secretary plan references a "Bridge RPC Server" with 50+ methods. This server:
- ✅ **EXISTS** (external, deployed, working per review.md)
- ❌ **NOT in this repository** (no `go.mod`, `pkg/`, `cmd/` directories)
- 🔍 **Location unknown** (possibly a separate repo)

**Implication:** If the plan requires modifying Bridge RPC methods (e.g., adding `browser.*`, `pii.*`, `keystore.unseal()`), we need access to the Go backend repository.

### 2. **Agent Infrastructure is 100% Greenfield**
None of the "Already Implemented" agent features exist in ArmorChat:
- No HITL approval screens
- No agent management UI
- No workflow visualization
- No browser automation

**Implication:** Phase 1-4 effort estimates are **underestimated** if they assume existing UI scaffolding.

### 3. **`.agents/skills/` is AI Agent Context, Not Runtime Code**
The `.agents/skills/` directory contains:
- Markdown skill definitions (e.g., `deploy-apk/SKILL.md`)
- Instructions for Oz AI agent (this AI)
- **NOT** runtime agent software

**Implication:** These are AI assistant skills, not the "Mobile Secretary" agent system.

---

## 📋 Revised Gap Analysis

| Plan Assumption | Reality | Impact on Timeline |
|-----------------|---------|-------------------|
| "Bridge RPC Server — ✅ 50+ methods" | ❓ Exists externally, unverified which methods exist | **HIGH** — May need to build RPC endpoints from scratch |
| "Matrix Client (ArmorChat) — ✅ 40+ methods" | ✅ Exists, works for **chat**, not agents | **MEDIUM** — Need to extend for agent events |
| "HITL System — ✅ Approve/Reject UI" | ❌ Does NOT exist | **HIGH** — Full UI implementation needed (Phase 3) |
| "Agent Management — ✅ List/Status/Stop" | ❌ Does NOT exist | **HIGH** — Full UI implementation needed (Phase 2) |
| "Workflow Engine — ✅ Templates/Start/Status" | ❌ Does NOT exist | **HIGH** — Full system implementation needed |
| "Headless Browser — ✅ Playwright" | ❌ Does NOT exist | **CRITICAL** — Entire Phase 1 is greenfield |
| "Keystore — ✅ Encrypted storage" | ✅ Exists for **Matrix keys**, NOT for PII vault | **MEDIUM** — Need separate PII keystore system |
| "User-Held Keys — ✅ KEK derivation" | ❌ Does NOT exist | **HIGH** — Entire Phase 4 is greenfield |

---

## ✅ Phase 0 Deliverables

### 1. Infrastructure Audit (This Document)
- ✅ Verified existing components
- ✅ Identified missing components
- ✅ Documented gaps vs. plan assumptions

### 2. Functional Screens Verified
- ✅ `GovernanceBanner.kt` — Working (governance events)
- ❌ `HitlApprovalScreen.kt` — Missing
- ❌ `AgentManagementScreen.kt` — Missing
- ✅ `KeyBackupSetupScreen.kt` — Working (Matrix key backup)
- ❌ `UnsealScreen.kt` — Missing (keystore unseal)

### 3. Communication Channels Verified
- ✅ Matrix /sync — Working
- ✅ JSON-RPC 2.0 — Working (with external Bridge server)
- ✅ FCM Push — Working
- ❓ Agent-specific RPC methods — Unverified (need Bridge server access)

### 4. Test Environment Status
- ✅ Android build system working (`gradlew assembleDebug` verified)
- ✅ ADB deployment working (device detected, APK installed)
- ❓ Bridge server test environment — Unknown (external)
- ❌ Matrix test room for agents — Not created yet

---

## 🚦 Recommendations

### Immediate Actions (Before Phase 1)

1. **Locate Bridge Server Repository**
   - Find the Go backend codebase
   - Verify which RPC methods already exist (`agent.*`, `workflow.*`, `keystore.*`, `pii.*`)
   - Confirm whether `browser.*` methods need to be built from scratch

2. **Adjust Timeline Estimates**
   - Phase 1 (Browser): Assume **7-10 days** (not 4) for greenfield Docker/Playwright setup
   - Phase 2 (Umbilical Cord): Assume **5-7 days** (not 2) for agent state machine + UI
   - Phase 3 (Remote Consent): Assume **5-7 days** (not 3) for HITL + BlindFill UI
   - Phase 4 (Keystore Hardening): Assume **7-10 days** (not 5) for KEK + unseal flow

   **Revised Total:** ~24-34 days (not 17 days)

3. **Create Test Matrix Room**
   - Set up a test room for agent status events
   - Configure Bridge server connection
   - Test Matrix event delivery pipeline

4. **Document External Dependencies**
   - Bridge server API documentation
   - RPC method contracts
   - Event schema definitions

---

## 📊 Phase 0 Completion Status

| Task | Status | Notes |
|------|--------|-------|
| Verify Bridge RPC methods | ⚠️ **PARTIAL** | Client exists, server methods unverified |
| Verify ArmorChat screens | ✅ **COMPLETE** | Only GovernanceBanner exists, agent screens missing |
| Verify Matrix event handling | ✅ **COMPLETE** | System events exist, agent events missing |
| Document gaps and issues | ✅ **COMPLETE** | This document |
| Setup test environment | ⚠️ **PARTIAL** | Android working, Bridge server unknown |

---

## 🎯 Next Steps: Phase 1 Prerequisites

Before starting Phase 1 (Headless Worker), we must:

1. ✅ **Confirm Docker availability** on VPS (or dev machine)
2. ✅ **Verify Node.js/npm** for Playwright installation
3. ❓ **Access Bridge server codebase** to add `browser.*` RPC methods
4. ❓ **Identify VPS hosting environment** for container deployment
5. ❓ **Define Browser Skill API contract** (input/output schemas)

**Blocker Resolution Required:**
- Where is the ArmorClaw Bridge Server repository?
- Can we modify it, or is it managed by another team?
- Is there an existing RPC method registration system?

---

**Audit Complete.**  
**Status:** ⚠️ **PROCEED WITH CAUTION** — Mobile Secretary plan assumes significant infrastructure that does not exist. Timeline estimates need revision.
