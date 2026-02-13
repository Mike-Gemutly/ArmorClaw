# ArmorClaw WebRTC Voice Support — Corrected Implementation Plan

**Date:** 2026-02-08  
**Status:** Revised / Approved for Implementation  
**Priority:** P1 – High  
**Estimated Duration:** 10–14 days  

---

## Executive Summary

This plan adds **real-time voice support** to ArmorClaw using **WebRTC**, while preserving ArmorClaw’s core principles:

- Zero-trust by default  
- Hardened, device-free containers  
- Strong lifecycle control (TTL, budget, cleanup)  
- Matrix as **authorization layer** (not transport)

**Key correction:**  
👉 **All WebRTC, audio I/O, and signaling live in the ArmorClaw Bridge — not in containers.**

---

## Goals

- Low-latency, bidirectional audio (<200ms)
- End-to-end encrypted media (DTLS-SRTP)
- Mobile-friendly (Android / iOS)
- Clean integration with:
  - TTL Manager
  - Budget Tracker
  - Zero-Trust Matrix Adapter
- No audio persisted to disk

---

## Non-Goals (v1)

- Video calls
- Multi-party calls
- Audio recording
- Background call continuity across app suspension
- Cross-container voice routing

---

## Corrected Architecture

```text
Mobile App (ArmorChat / Element X)
        ↕ WebRTC (audio, DTLS-SRTP)
ArmorClaw Bridge
  - WebRTC (Pion)
  - Audio capture / playback
  - TURN credential issuance
  - Session + TTL management
  - Budget enforcement
        ↕ PCM frames (RPC / stream)
Agent Container
  - NO audio devices
  - NO WebRTC
  - Logic + inference only
```
## Key Architectural Rules

- Containers **never** access microphones or speakers
- Containers **never** handle SDP, ICE, or TURN
- The **Bridge is the only WebRTC peer**
- Matrix is used for **authorization**, not transport

---

## Component Overview

### 1. WebRTC Engine (Bridge – Go)

**Location:**  
`bridge/pkg/webrtc/`

**Responsibilities**
- WebRTC `PeerConnection` (audio-only)
- SDP offer / answer negotiation
- ICE candidate handling
- Media encryption (DTLS-SRTP)
- Audio frame forwarding to and from the agent

**Dependencies**
- `github.com/pion/webrtc/v3`
- `github.com/gorilla/websocket`

## 2. Signaling Server (Bridge – Go)

**Transport:**  
Secure WebSocket (`wss://`)

**Purpose:**  
Session negotiation only (no media transport)

### Message Types
```json
{
  "type": "offer | answer | ice | bye",
  "session_id": "uuid",
  "payload": {}
}
```
## Security

- TLS required for all signaling traffic
- Call session token required (**not** raw Matrix token)
- Rate-limited to prevent abuse

---

## 3. Call Session Manager (Bridge – Go)

### Purpose
Single source of truth for the entire voice call lifecycle.

### Binds Together
- Matrix room
- Agent container
- WebRTC peer
- TURN allocation
- Budget session

### Session States
```text
Pending → Active → Ended | Failed | Expired
```
## TTL Rules

- Default TTL: **10 minutes**
- Call end → container heartbeat stops
- Container death → call forcibly ends

---

## 4. Call Session Token (NEW)

### Purpose
- Avoid overloading Matrix authentication
- Provide short-lived, scoped credentials

### Properties
```json
{
  "session_id": "uuid",
  "room_id": "!matrixRoom",
  "expires_at": "timestamp"
}
```
### Used For
- WebRTC signaling
- TURN credential derivation

---

## 5. TURN / STUN Integration (Ephemeral Credentials)

### TURN
- Coturn (existing infrastructure reused)
- REST authentication (RFC 5766)

### Credential Format
```text
username = <expiry>:<session_id>
password = HMAC(secret, username)
```

### Benefits
- Automatic expiration
- Per-session abuse containment
- No shared static secrets

---

## 6. Audio Pipeline (Bridge Only)

### Input
- WebRTC incoming audio
- Decoded to PCM

### Processing
- Forward PCM frames to agent container
- Receive synthesized PCM audio from agent

### Output
- Encode PCM → Opus
- Send via WebRTC audio track

> **Note:** No ALSA, PulseAudio, or `/dev/snd` inside containers.

---

## 7. Agent Container Interface (Updated)

### Removed
- `pyaudio`
- `aiortc`
- Any audio device access

### New Interface
```text
AudioFrameIn  (PCM bytes)  → agent
AudioFrameOut (PCM bytes)  ← agent
```

## Transport Options

- JSON-RPC streaming
- Shared memory / pipes (implementation choice)

---

## RPC Additions (Bridge)

```text
WebRTCStart(room_id) → {
  session_id,
  sdp_answer,
  turn_credentials
}

WebRTCIceCandidate(session_id, candidate)

WebRTCEnd(session_id)
```

## RPC Enforcement

All RPC calls must:
- Enforce zero-trust sender and room validation
- Emit structured security events

---

## Budget Integration (Required)

Voice sessions are **billable activity**.

### Rules
- Open a budget session on call start
- Track:
  - Duration
  - Model used
  - Tokens consumed
- Enforce a hard stop if limits are exceeded mid-call

### New Field
```text
SessionType = "text" | "voice"
```
## Failure & Mobile Reality Handling

### Explicit Rules
- App backgrounded → call ends
- Network change → attempt one ICE restart
- WebRTC failure → fallback to text
- TTL expiry → hard termination

### User-Visible States
- Connecting
- Live
- Reconnecting
- Ended
- Failed

---

## Security Considerations

- DTLS-SRTP mandatory
- No audio written to disk
- No cross-session audio access
- TURN usage rate-limited
- All signaling events logged as security events

---

## Implementation Phases (Revised)

### Phase 1 — Core Infrastructure (Days 1–3)
- WebRTC engine in bridge
- Signaling server
- Session manager
- Call tokens
- Unit tests

### Phase 2 — Audio Pipeline (Days 4–6)
- PCM streaming bridge ↔ agent
- Opus encode / decode
- Audio loopback tests

### Phase 3 — TURN + NAT (Days 7–8)
- Ephemeral TURN credentials
- ICE handling
- NAT traversal testing

### Phase 4 — Matrix Integration (Days 9–10)
- Room-scoped authorization
- Call start / end via Matrix messages
- Element X / ArmorChat testing

### Phase 5 — Hardening (Days 11–14)
- Budget enforcement
- TTL enforcement
- Security audit
- Load and soak tests
- Documentation

---

## Explicitly Out of Scope (v1)

- Video
- Call recording
- Multi-party calls
- Background call persistence
- Cross-agent audio

---

## Final Assessment

| Area | Status |
|------|--------|
| Security | ✅ Strong |
| Container isolation | ✅ Preserved |
| Mobile compatibility | ✅ Realistic |
| TURN safety | ✅ Fixed |
| Budget control | ✅ Integrated |
| TTL correctness | ✅ Unified |

---

## Approval Statement

This revised plan:
- Fits ArmorClaw’s existing architecture
- Avoids unsafe container practices
- Is implementable without refactors
- Keeps future expansion (video, multi-party) open

**Status:** ✅ Approved for implementation



