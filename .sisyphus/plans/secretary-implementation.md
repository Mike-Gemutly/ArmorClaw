# Secretary Implementation Plan

> **Status:** Ready for Execution
> **Created:** 2026-03-19
> **Source:** doc/plan/SECRETARY_PLAN.md

---

## TL;DR

> **Quick Summary:** Implement a 6-phase "Secretary" proactive assistant system for ArmorChat, starting with core models and state machine, then adding briefing, triage, privacy, action routing, and voice surfaces.
> 
> **Deliverables:**
> - SecretaryModels.kt, SecretaryState.kt, SecretaryViewModel.kt
> - ProactiveCard.kt, SecretaryAvatar.kt (Compose UI)
> - SecretaryBriefingEngine, SecretaryPolicyEngine
> - PrivacyGuardPolicy, SecretaryActionRouter
> - Voice surfaces (Phase 6)
> 
> **Estimated Effort:** Large (6 phases, 2-3 weeks each)
> **Parallel Execution:** YES - phases sequential, tasks within phases parallel
> **Critical Path:** Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    SECRETARY ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: UI (Compose)                                          │
│  └─ ProactiveCard, SecretaryAvatar, RolodexTierRenderer         │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: Orchestration (SecretaryViewModel)                    │
│  └─ Event analysis, state transitions, cooldown management      │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Local Policy & Context                               │
│  └─ PrivacyGuardPolicy, SecretaryPolicyEngine, ContextProvider  │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: Action Router                                         │
│  └─ SecretaryActionRouter → BridgeSecretaryClient / LocalAction │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Secretary Shell (Foundation)

### Goal
Build the smallest vertical slice proving the Secretary UX loop works.

### Scope
- Core typed models
- State machine
- ViewModel observing MatrixSyncManager
- Two UI primitives (ProactiveCard, SecretaryAvatar)
- One deterministic behavior: urgent Matrix event → proactive card

### Out of Scope
- Bridge RPC execution
- Privacy guard
- Briefing logic
- Voice

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| SecretaryModels.kt | shared/.../secretary | ProactiveCard, SecretaryAction, SecretaryPriority |
| SecretaryState.kt | shared/.../secretary | State machine (Idle, Observing, Thinking, Proposing, Error) |
| SecretaryViewModel.kt | androidApp/.../secretary | State, card list, Matrix event observation |
| ProactiveCard.kt | androidApp/.../secretary/ui | Compose card component |
| SecretaryAvatar.kt | androidApp/.../secretary/ui | State indicator component |

### Acceptance Tests

- [ ] A1: Initial state is Idle, card list empty
- [ ] A2: Urgent Matrix event → Proposing state with one card
- [ ] A3: Dismiss removes card, returns to Idle if none remain
- [ ] B1: Urgent keyword triggers proactive card
- [ ] B2: Non-urgent message does not trigger card
- [ ] B3: VIP sender triggers card without urgent keyword
- [ ] B4: Duplicate event ID ignored
- [ ] C1: Local action emitted on tap
- [ ] D1: ProactiveCard renders title and actions
- [ ] D2: SecretaryAvatar reflects state changes

---

## Phase 2: Briefing and Review

### Goal
Deliver first high-value Secretary summary experience (Morning Briefing, Evening Review).

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| SecretaryBriefingEngine.kt | shared/.../secretary | Deterministic briefing generation |
| SecretaryContextProvider.kt | androidApp/.../secretary | Context aggregation (unread, meetings, approvals) |

### Acceptance Tests

- [ ] Morning briefing appears in configured window
- [ ] No duplicate briefing same day
- [ ] Evening review can be disabled
- [ ] Weekend behavior changes summary
- [ ] No card when context insufficient
- [ ] Summary includes unread count, next meeting, pending approvals
- [ ] One recommendation chip present

---

## Phase 3: Context and Triage

### Goal
Implement deterministic attention management with mode detection and follow-up detection.

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| SecretaryPolicyEngine.kt | shared/.../secretary | Mode/suppression decisions |
| SecretaryMode.kt | shared/.../secretary | MEETING, FOCUS, SLEEP, NORMAL |
| SecretaryTriage.kt | shared/.../secretary | Priority scoring |
| SecretaryFollowUp.kt | shared/.../secretary | Outbound thread follow-up detection |

### Acceptance Tests

- [ ] Meeting mode suppresses non-urgent
- [ ] Focus mode respects whitelist
- [ ] Sleep mode batches normal traffic
- [ ] Urgent keyword raises priority
- [ ] VIP sender raises priority
- [ ] Calendar-linked thread raises priority
- [ ] Follow-up prompt for stale outbound threads
- [ ] Snooze respects cooldown

---

## Phase 4: Privacy Guard and Rolodex Tiers

### Goal
Implement safe on-device presentation rules for sensitive content.

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| PrivacyGuardPolicy.kt | shared/.../secretary | Visibility/masking decisions |
| SecretarySensitivityTier.kt | shared/.../secretary | PUBLIC_SAFE, PRIVATE_DEVICE, BIOMETRIC_GATED |
| RolodexTierRenderer.kt | androidApp/.../secretary/ui | Tiered contact rendering |
| SensitiveContentRevealController.kt | androidApp/.../secretary | Biometric-gated reveal |

### Acceptance Tests

- [ ] Public-safe content always visible
- [ ] Private-device content masked in untrusted context
- [ ] Biometric-gated requires successful auth
- [ ] Push preview masked in public-risk state
- [ ] TTS suppressed for sensitive in public
- [ ] Missing context falls back safely

---

## Phase 5: Action Routing and Bridge Execution

### Goal
Connect Secretary cards to real ArmorClaw actions through typed routing.

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| SecretaryActionRouter.kt | androidApp/.../secretary | Local vs Bridge routing |
| BridgeSecretaryClient.kt | androidApp/.../secretary | Typed adapter over Bridge RPC |
| SecretaryHitlCoordinator.kt | androidApp/.../secretary | Integration with existing HITL |
| SecretaryExecutionError.kt | shared/.../secretary | Normalized error model |

### Acceptance Tests

- [ ] Local-only action stays local
- [ ] Workflow action routes to Bridge
- [ ] Agent action routes to Bridge
- [ ] HITL approval routes to existing flow
- [ ] Network failure becomes recoverable error
- [ ] Timeout surfaced correctly
- [ ] Unsupported action rejected safely

---

## Phase 6: Voice Surfaces (Optional Enhancement)

### Goal
Add limited voice interaction (mic button → STT → parser → safe TTS).

### Files to Create

| File | Package | Purpose |
|------|---------|---------|
| SecretaryVoiceIntent.kt | shared/.../secretary | Voice intent model |
| SecretaryVoiceParser.kt | shared/.../secretary | Transcript → intent mapping |
| SecretaryVoiceController.kt | androidApp/.../secretary | Mic capture coordination |
| SecretaryTtsCoordinator.kt | androidApp/.../secretary | Privacy-guarded TTS |

### Acceptance Tests

- [ ] Mic button starts explicit capture
- [ ] Cancel returns to idle
- [ ] Transcript maps to known intent
- [ ] Unsupported transcript rejected safely
- [ ] Sensitive content not spoken in public
- [ ] Driving mode uses short summary

---

## Key Principles

1. **Deterministic**: Same input → same output
2. **Local-first**: Offline-capable, no Bridge dependency for core logic
3. **Typed routing**: No raw RPC strings in UI/ViewModel
4. **Reuse existing**: MatrixSyncManager, BiometricAuth, BridgeRpcClient
5. **Test-driven**: Acceptance tests first
6. **No new dependencies**: Use existing libraries

---

## Package Layout

### Shared/KMP
```
shared/src/commonMain/.../secretary/
├── SecretaryModels.kt
├── SecretaryState.kt
├── SecretaryAction.kt
├── SecretaryPriority.kt
├── SecretaryPolicyEngine.kt
├── SecretaryBriefingEngine.kt
├── PrivacyGuardPolicy.kt
├── SecretarySensitivityTier.kt
└── SecretaryVoiceIntent.kt
```

### Android App
```
androidApp/src/main/.../secretary/
├── SecretaryViewModel.kt
├── SecretaryContextProvider.kt
├── SecretaryActionRouter.kt
├── BridgeSecretaryClient.kt
├── SecretaryHitlCoordinator.kt
└── ui/
    ├── ProactiveCard.kt
    ├── SecretaryAvatar.kt
    └── RolodexTierRenderer.kt
```

---

## Integration Points

| Existing Component | Secretary Integration |
|-------------------|----------------------|
| MatrixSyncManager | Real-time event observation |
| BiometricAuthImpl | Gated content reveal |
| BridgeRpcClient | Admin/action operations |
| HitlViewModel | Approval flow routing |
| BackgroundSyncWorker | Periodic reevaluation |
| VoiceRecorder | Phase 6 voice capture |
| NotificationManager | Secretary notification surface |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Secretary becomes noisy | Voice responses shorter than cards; no unsolicited TTS |
| UI duplicates OMO/agent UX | Reuse existing surfaces; no parallel voice mode |
| Privacy mistakes | Privacy Guard before all TTS; conservative fallback |
| RPC drift | Typed adapter; no raw method strings |
| Voice scope blowup | No hotword, no background capture, no continuous loop |

---

## Next Steps

1. **Review and approve this plan**
2. **Start Phase 1 implementation** with TDD approach
3. **Create SecretaryModels.kt and SecretaryState.kt** first
4. **Implement SecretaryViewModel** with MatrixSyncManager integration
5. **Add ProactiveCard and SecretaryAvatar** UI components
6. **Run acceptance tests** before moving to Phase 2

---

*Plan generated from doc/plan/SECRETARY_PLAN.md - Ready for execution via /start-work*
