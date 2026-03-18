# ArmorChat & ArmorClaw 2026 Roadmap Analysis

> **Document Purpose:** Strategic roadmap for transforming ArmorClaw into a "Hardened Intelligence" platform
> **Created:** 2026-02-11
> **Status:** Planning

---

## Executive Summary

This roadmap outlines a strategic transformation from a basic chat application to a **"Hardened Intelligence" platform** where AI agents and users communicate with Discord-level fluidity and military-grade encryption. The roadmap consists of 5 interconnected epics with clear priorities and dependencies.

---

## Epic Dependency Graph

```
                    ┌─────────────────────────────────────────────────────┐
                    │                    EPIC A                           │
                    │              Full Threads Support                  │
                    │                  (#1 Priority)                     │
                    │                                                     │
                    │   "Blocks voice-in-thread UX"                       │
                    └──────────────────────┬──────────────────────────────┘
                                           │
                                           │ Enables
                                           ▼
                    ┌─────────────────────────────────────────────────────┐
                    │                    EPIC B                           │
                    │         Reliable Voice (MatrixRTC Audio-Only)      │
                    │                  (#2 Priority)                     │
                    │                                                     │
                    │   "Requires Threads 2.0 for full value"            │
                    └──────────────────────┬──────────────────────────────┘
                                           │
                                           │ Requires
                                           ▼
                    ┌─────────────────────────────────────────────────────┐
                    │                    EPIC C                           │
                    │       Improved Session Trust & Verification UI     │
                    │                                                     │
                    │   "Blocks voice trust/adoption"                    │
                    └──────────────────────┬──────────────────────────────┘
                                           │
                                           │ Enhances
                                           ▼
                    ┌─────────────────────────────────────────────────────┐
                    │                    EPIC D                           │
                    │        Push Notifications & Unread Handling        │
                    │                                                     │
                    │   "Critical for call reliability"                  │
                    └──────────────────────┬──────────────────────────────┘
                                           │
                                           │ Supports
                                           ▼
                    ┌─────────────────────────────────────────────────────┐
                    │                    EPIC E                           │
                    │            Diagnostics & Polish                    │
                    │                                                     │
                    │   "Foundation for production readiness"            │
                    └─────────────────────────────────────────────────────┘
```

---

## Epic A: Full Threads Support (ArmorChat 2.0)

| Aspect | Assessment |
|--------|------------|
| **Priority** | #1 - Critical Path |
| **Impact** | Extremely High |
| **Complexity** | Medium-High |

### Problem Statement

Linear message history fragments conversations, making it difficult to follow discussion threads and reducing communication clarity.

### Technical Approach

Using Matrix `m.thread` and `m.relates_to` event types for native thread support.

### Model Updates

```kotlin
// Current Model Addition
data class ThreadInfo(
    val threadUnreadCount: Int,
    val lastThreadMessage: Message?
)

// Suggested Additional Fields
data class ThreadMetadata(
    val threadRootId: String?,          // Reference to thread parent
    val threadParticipants: List<User>, // Active thread participants
    val threadLastActivity: Long        // For sorting active threads
)
```

### Database Schema Changes

```sql
-- Thread support columns
ALTER TABLE messages ADD COLUMN threadRootId TEXT;
ALTER TABLE messages ADD COLUMN isThreadReply INTEGER DEFAULT 0;
ALTER TABLE messages ADD COLUMN threadDepth INTEGER DEFAULT 0;

-- Thread aggregate columns
ALTER TABLE rooms ADD COLUMN threadUnreadCount INTEGER DEFAULT 0;
ALTER TABLE rooms ADD COLUMN activeThreadCount INTEGER DEFAULT 0;

-- Indices for thread queries
CREATE INDEX idx_messages_threadRootId ON messages(threadRootId);
CREATE INDEX idx_messages_threadDepth ON messages(threadDepth);
```

### Alignment with Existing Architecture

- Update `RoomEntity` schema to include thread-related columns
- Extend `MessageEntity` with `threadRootId` and `isThreadReply` fields
- Modify offline sync queue to handle thread operations
- Add `ThreadRepository` interface for thread-specific operations

---

## Epic B: Reliable ArmorClaw Voice (MatrixRTC Audio-Only)

| Aspect | Assessment |
|--------|------------|
| **Priority** | #2 - High |
| **Impact** | Very High |
| **Complexity** | High |

### Strategic Rationale: Audio-Only

```
┌─────────────────────────────────────────────────────────────────┐
│                Why Audio-Only is Strategic                       │
├─────────────────────────────────────────────────────────────────┤
│  ✅ 5x less bandwidth than video                                │
│  ✅ Lower latency (critical for AI agent interaction)           │
│  ✅ Better battery performance on mobile                        │
│  ✅ Enhanced privacy (no video exposure)                        │
│  ✅ Aligns with "always-available agent" use case               │
└─────────────────────────────────────────────────────────────────┘
```

### Technical Requirements

| Requirement | Specification |
|-------------|---------------|
| Signaling Protocol | MSC3077 (MatrixRTC signaling) |
| VoIP Protocol | MSC3401 (Native Matrix VoIP) |
| Infrastructure | TURN server optimization for audio-only |
| Interoperability | Element X compatibility testing |

### Implementation Components

```kotlin
// Voice Call Manager Interface
interface VoiceCallManager {
    suspend fun initiateCall(roomId: String, participants: List<String>): Result<CallSession>
    suspend fun joinCall(callId: String): Result<CallSession>
    suspend fun leaveCall(callId: String): Result<Unit>
    suspend fun muteAudio(callId: String): Result<Unit>
    suspend fun unmuteAudio(callId: String): Result<Unit>
    fun getActiveCalls(): Flow<List<CallSession>>
    fun getCallState(callId: String): Flow<CallState>
}

data class CallSession(
    val id: String,
    val roomId: String,
    val callerId: String,
    val participants: List<CallParticipant>,
    val state: CallState,
    val startedAt: Long,
    val isAudioOnly: Boolean = true
)

sealed class CallState {
    object Connecting : CallState()
    object Active : CallState()
    object OnHold : CallState()
    data class Error(val code: String, val message: String) : CallState()
    object Ended : CallState()
}
```

### DoD Metrics

| Metric | Target | Feasibility |
|--------|--------|-------------|
| Connection Rate | ≥95% | Challenging (network variability) |
| Call Setup Time | <2s | Achievable with optimized TURN |
| Audio-Only Enforcement | Homeserver-level | Straightforward |
| ICE Connection Success | ≥98% | Requires TURN redundancy |

---

## Epic C: Improved Session Trust & Verification UI

| Aspect | Assessment |
|--------|------------|
| **Priority** | #3 - High |
| **Impact** | High |
| **Complexity** | Medium |

### Trust Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Trust Flow Architecture                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    Cross-Signing     ┌──────────────┐        │
│  │ ArmorChat    │ ◄──────────────────► │ Element X    │        │
│  │ (User)       │                      │ (Interop)    │        │
│  └──────────────┘                      └──────────────┘        │
│         │                                                      │
│         │ Device Trust                                          │
│         ▼                                                      │
│  ┌──────────────┐    Verified Session    ┌──────────────┐      │
│  │ ArmorClaw    │ ◄───────────────────── │ AI Agent     │      │
│  │ Agent        │                        │ Service      │      │
│  └──────────────┘                        └──────────────┘      │
│                                                                  │
│  ⚠️ Cross-signing required before "Sensitive Voice" actions    │
└─────────────────────────────────────────────────────────────────┘
```

### Integration with Existing Security

| Component | Extension |
|-----------|-----------|
| `BiometricAuth` | Verification prompts for trust actions |
| `AndroidKeyStore` | Cross-signing key storage |
| `SecureClipboard` | Verification token handling |

### Verification UI Components

```kotlin
// Verification State Machine
sealed class VerificationState {
    object Unverified : VerificationState()
    object Verifying : VerificationState()
    data class EmojiChallenge(val emojis: List<String>) : VerificationState()
    data class CodeChallenge(val code: String) : VerificationState()
    object Verified : VerificationState()
    data class Failed(val reason: String) : VerificationState()
}

// Trust Level
enum class TrustLevel {
    UNVERIFIED,       // No verification performed
    CROSS_SIGNED,     // Verified via cross-signing
    VERIFIED_IN_PERSON, // Manual verification
    KNOWN_UNCOMPRIMISED // Previously verified, still trusted
}
```

---

## Epic D: Push Notifications & Unread Handling

| Aspect | Assessment |
|--------|------------|
| **Priority** | High |
| **Impact** | High |
| **Complexity** | Medium |

### Push Payload Specification

```kotlin
// Call Push Payload Structure
data class CallPushPayload(
    val roomId: String,
    val callerId: String,
    val callerName: String,
    val callType: CallType,           // AUDIO, VIDEO
    val priority: PushPriority,        // HIGH for calls
    val timestamp: Long,
    val isEncrypted: Boolean = true
)

enum class CallType {
    AUDIO, VIDEO
}

enum class PushPriority {
    NORMAL,   // Regular messages
    HIGH,     // Calls, mentions
    URGENT    // Emergency alerts
}
```

### FCM Configuration

```
Priority Levels:
├── Missed Call → FCM HIGH priority + full-screen intent
├── Message → FCM NORMAL priority
├── Thread Reply → FCM NORMAL priority + thread badge
└── Agent Status Change → FCM LOW priority
```

### Notification Channels

```kotlin
object NotificationChannels {
    const val CALLS = "calls"                    // High importance, sound
    const val MESSAGES = "messages"              // High importance
    const val THREAD_REPLIES = "thread_replies"  // Default importance
    const val MENTIONS = "mentions"              // High importance, sound
    const val AGENT_STATUS = "agent_status"      // Low importance
}
```

---

## Epic E: Diagnostics & Polish

| Aspect | Assessment |
|--------|------------|
| **Priority** | Medium-High |
| **Impact** | Medium-High |
| **Complexity** | Low-Medium |

### Error Code Taxonomy

```
Current State: "couldn't reach homeserver" (vague)
                        │
                        ▼
Proposed Error Codes:
├── VOICE_MIC_DENIED           // Microphone permission denied
├── VOICE_CAMERA_DENIED        // Camera permission denied
├── TURN_SERVER_TIMEOUT        // TURN connection failed
├── ICE_CONNECTION_FAILED      // WebRTC ICE failure
├── HOMESERVER_UNREACHABLE     // Network connectivity
├── DEVICE_UNVERIFIED          // Trust verification required
├── SESSION_EXPIRED            // Authentication required
├── ENCRYPTION_KEY_ERROR       // E2EE key issue
└── SYNC_QUEUE_OVERFLOW        // Offline queue full
```

### Error Code Implementation

```kotlin
enum class ArmorClawErrorCode(
    val code: String,
    val userMessage: String,
    val recoverable: Boolean
) {
    VOICE_MIC_DENIED("E001", "Microphone access denied. Please enable in settings.", true),
    TURN_SERVER_TIMEOUT("E002", "Call server unreachable. Please try again.", true),
    ICE_CONNECTION_FAILED("E003", "Network connection failed. Check your internet.", true),
    HOMESERVER_UNREACHABLE("E004", "Server temporarily unavailable.", true),
    DEVICE_UNVERIFIED("E005", "Device verification required for this action.", true),
    SESSION_EXPIRED("E006", "Session expired. Please log in again.", true),
    ENCRYPTION_KEY_ERROR("E007", "Encryption error. Please restart the app.", true),
    SYNC_QUEUE_OVERFLOW("E008", "Too many pending operations. Please wait.", true);
}
```

### Feature Flags for Polish

```kotlin
// Extends ReleaseConfig.kt
object VoiceFeatureFlags {
    // Phase 1 (Q2 2026)
    const val AUDIO_CALLS_ENABLED = true
    const val CALL_NOTIFICATIONS_ENABLED = true

    // Phase 2 (Q3 2026)
    const val AUDIO_VISUALIZER_ENABLED = false
    const val MUTE_HOLD_ENABLED = false
    const val IN_CALL_REACTIONS_ENABLED = false

    // Phase 3 (Q4 2026)
    const val SPACES_2_0_ENABLED = false
    const val VIDEO_CALLS_ENABLED = false
}
```

---

## Implementation Priorities Matrix

| Feature | ArmorClaw Platform | ArmorChat App | Complexity |
|---------|-------------------|---------------|------------|
| **Threads** | Database schema for `m.thread` relationships | UI for nested message lists and thread badges | Medium |
| **Voice** | MatrixRTC signaling + TURN optimization | Audio-only WebRTC + permissions UI | High |
| **Trust** | Cross-signing + AndroidKeyStore | "Secure Session" UI + biometric prompts | Medium |
| **Sync** | Queue prioritization for signaling | Real-time updates + optimistic UI | Low |
| **Notifications** | FCM integration + payload handling | Call notifications + badge updates | Medium |

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| MatrixRTC spec instability | Medium | High | Pin to specific MSC versions; maintain fallback |
| Element X interop issues | Medium | High | Establish compatibility test suite |
| Cross-signing UX complexity | High | Medium | Simplified verification flow; progressive disclosure |
| FCM background delivery | Medium | High | Implement fallback poll mechanism |
| Thread sync conflicts | Medium | Medium | Extend existing `ConflictResolver` |
| Audio codec compatibility | Low | Medium | Use Opus as primary; fallback to AAC |

---

## Execution Timeline

### Phase 1 (Q1 2026): Foundation

```
Week 1-4: Epic A - Threads Support
├── Week 1: Update Room/Message models
├── Week 2: Implement m.thread schema
├── Week 3: Thread repository implementation
└── Week 4: Thread UI in ArmorChat

Week 2-4 (parallel): Epic E - Diagnostics
└── Error code taxonomy implementation
```

### Phase 2 (Q2 2026): Voice Infrastructure

```
Week 5-8: Epic B - MatrixRTC Voice
├── Week 5: Audio-only WebRTC stack setup
├── Week 6: TURN server optimization
├── Week 7: Call UI implementation
└── Week 8: Element X interop testing

Week 6-8 (parallel): Epic C - Trust & Verification
├── Week 6: Cross-signing implementation
├── Week 7: Voice Trust panel UI
└── Week 8: Biometric verification flow
```

### Phase 3 (Q3 2026): Polish & Reliability

```
Week 9-12: Epic D - Push Notifications
├── Week 9: Call payload specification
├── Week 10: FCM integration
├── Week 11: Background delivery testing
└── Week 12: Notification UI polish

Week 10-12 (parallel): Epic E - Polish
├── Week 10: Audio visualizers
├── Week 11: In-call reactions
└── Week 12: Spaces 2.0 foundation
```

### Phase 4 (Q4 2026): Production Hardening

```
Week 13-16: Final Preparations
├── Week 13: End-to-end testing
├── Week 14: Performance optimization
├── Week 15: Documentation updates
└── Week 16: Production deployment
```

---

## Summary

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Strategic Clarity** | ⭐⭐⭐⭐⭐ | Clear vision of "Hardened Intelligence" platform |
| **Priority Alignment** | ⭐⭐⭐⭐⭐ | Dependencies correctly identified |
| **Technical Feasibility** | ⭐⭐⭐⭐☆ | MatrixRTC adds complexity but is manageable |
| **DoD Specificity** | ⭐⭐⭐⭐☆ | Good metrics, some need quantification |
| **Documentation** | ⭐⭐⭐⭐☆ | Comprehensive specifications |

**Overall Roadmap Score: 4.4/5.0**

This roadmap provides a solid foundation for transforming ArmorClaw into a production-ready AI agent communication platform. The phased approach with clear dependencies will enable systematic delivery while maintaining quality.

---

## Appendix: Matrix MSC References

| MSC | Title | Status | Relevance |
|-----|-------|--------|-----------|
| MSC3077 | MatrixRTC signaling | Stable | Voice calls |
| MSC3401 | Native Matrix VoIP | Stable | Call protocol |
| MSC3440 | Thread relations | Stable | Threads support |
| MSC3676 | Threaded notifications | Stable | Thread notifications |
| MSC3786 | Audio-only calls | Proposed | Audio enforcement |

---

*Last Updated: 2026-02-11*
*Document Status: Planning*
