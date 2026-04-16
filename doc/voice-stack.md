# Voice Stack

> Part of the [ArmorClaw System Documentation](armorclaw.md)

## Overview

The voice stack lets ArmorClaw agents make and receive real-time phone calls through the mobile app. It handles everything from audio encoding to NAT traversal entirely inside the Bridge. Audio never touches the agent container directly; the Bridge encodes, decodes, and forwards it between the WebRTC peer and the agent's stdin/stdout or gRPC stream.

The stack is built on four packages: `audio` for PCM and Opus processing, `voice` for call budget enforcement, `webrtc` for peer connection management and session lifecycle, and `turn` for NAT traversal with ephemeral credentials.

## Architecture

Audio flows through a fixed path from the caller's phone to the agent and back. The Bridge sits in the middle, handling codec work, budget checks, and signaling. TURN relays handle NAT punching when direct connections aren't possible.

```
                          ArmorClaw Voice Call Flow

  ┌──────────┐       ┌───────────┐       ┌─────────────────────────────────────┐
  │  Phone   │       │   TURN    │       │            Bridge (VPS)             │
  │ ArmorChat│       │  Relay    │       │                                     │
  │          │       │           │       │  ┌─────────┐  ┌───────┐  ┌───────┐ │
  │ Mic ─────┼──SDP──┼───────────┼──RTP──┼─▶│ webrtc  │─▶│ audio │─▶│ voice │ │
  │          │       │  (NAT     │       │  │ engine  │  │ pcm   │  │ budget│ │
  │ Speaker ◀┼──SDP──┼──traversal│◀─RTP──┼──│ session │◀─│ opus  │◀─│ check │ │
  │          │       │  only)    │       │  └────┬────┘  └───┬───┘  └───┬───┘ │
  └──────────┘       └───────────┘       │       │            │           │     │
                                          │       │            │           │     │
                                          │       ▼            ▼           │     │
                                          │  ┌──────────────────────┐     │     │
                                          │  │    Agent Container   │◀────┘     │
                                          │  │    (AI runtime)      │           │
                                          │  └──────────────────────┘           │
                                          └─────────────────────────────────────┘

  Signaling path (SDP offer/answer, ICE candidates):
    Phone ◀── Matrix E2EE room ──▶ Bridge

  Media path (audio RTP):
    Phone ◀── TURN relay (or direct) ──▶ Bridge

  Budget path:
    Bridge tracks tokens + duration per session, enforces hard stop
```

The signaling layer uses Matrix rooms for SDP exchange and ICE candidate trickling. The media layer runs over RTP through TURN or direct UDP. Budget enforcement runs as a background goroutine that checks every 30 seconds.

## Key Packages

### `bridge/pkg/audio/`

PCM processing and Opus codec support. All audio I/O lives here, not in agent containers.

| File | Purpose |
|------|---------|
| `pcm.go` | `AudioConfig` defaults (48 kHz, mono, 16-bit, 20 ms frames), `AudioStream` bidirectional frame channels, `AudioPipeline` per-session stream pairs, `PCMMixer` for combining multiple streams, `PCMEncoder` with sample rate conversion, `AudioBuffer` circular ring buffer, `WebRTCTrackReader`/`Writer` for Pion track I/O |
| `opus.go` | `OpusEncoder`/`OpusDecoder` for PCM-to-Opus conversion, `OpusConfig` with bitrate/complexity/FEC/DTX tuning, `RTPOpusPacketizer`/`RTPDepacketizer` for RTP framing, `AudioStats` frame/packet/jitter tracking, `AudioLevelMeter` dBFS measurement, `OpusPayloader`/`Depayloader` for Pion integration |

Default audio config: 48 kHz sample rate, mono, 16-bit depth, 960 samples per frame (20 ms), 10-frame buffer (200 ms).

### `bridge/pkg/voice/`

Call budget tracking. Prevents runaway token costs and enforces time limits.

| File | Purpose |
|------|---------|
| `budget.go` | `BudgetTracker` manages per-session limits, `VoiceSessionTracker` tracks token usage (input + output) and duration, `TokenUsage` counters, `Config` with default/duration limits and warning thresholds, background `EnforceLimits` loop (30 s interval), security logging for budget events |

Key defaults:
- Token limit: 100,000 per call
- Duration limit: 30 minutes per call
- Warning threshold: 80% of limit
- Hard stop: enabled by default

The tracker emits `voice_budget_warning` security events when usage crosses the warning threshold and `voice_budget_enforced` when hard-stopping a call.

### `bridge/pkg/webrtc/`

WebRTC peer connection management and session lifecycle. This is where Matrix rooms, agent containers, TURN allocations, and budget sessions are bound together.

| File | Purpose |
|------|---------|
| `engine.go` | `Engine` creates and manages `PeerConnectionWrapper` instances, registers Opus codec, handles SDP offer/answer exchange, writes audio to local tracks, reads RTP from remote tracks, integrates with `turn.Manager` for ephemeral credentials |
| `session.go` | `SessionManager` handles the full lifecycle of `Session` objects (pending, active, ended, failed, expired), TTL enforcement with 1-minute cleanup interval, binds session to container ID, Matrix room ID, TURN credentials, and budget session |

Session states: `pending` (created, not connected) → `active` (media flowing) → `ended` (normal close), `failed` (error), or `expired` (TTL hit).

Default TTL: 10 minutes. Max TTL: 1 hour. Session IDs use `sess_` prefix with 16 hex chars from `crypto/rand`.

### `bridge/pkg/turn/`

NAT traversal with ephemeral per-session TURN credentials. No static passwords.

| File | Purpose |
|------|---------|
| `turn.go` | `Manager` generates time-limited TURN credentials using HMAC-SHA1, `TURNCredentials` with `<expiry>:<session_id>` username format, `ICEGatherer` for host candidate gathering (reflexive/relay gathering return empty — delegated to WebRTC stack), `ICECandidate` parsing and serialization, `STUNMessage` builder/parser for STUN binding requests, `CreateICEServers` helper for Pion integration |

Credential format: username is `<unix_expiry>:<session_id>`, password is `base64(HMAC-SHA1(secret, username))`. Credentials are scoped to a single session and auto-expire. A cleanup goroutine runs every minute to purge stale entries.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TURN_SECRET` | **Required.** Shared secret for HMAC-SHA1 credential generation. Bridge refuses to start if empty. | _(none — must be set)_ |
| `TURN_HOST` | TURN relay hostname or IP | `matrix.armorclaw.com` |
| `TURN_PORT` | TURN relay port | `3478` |
| `TURN_PROTOCOL` | Transport protocol: `udp`, `tcp`, or `tls` | `udp` |
| `TURN_REALM` | Authentication realm | `armorclaw` |
| `TURN_DEFAULT_TTL` | Credential lifetime | `10m` |
| `TURN_MAX_TTL` | Maximum credential lifetime | `1h` |

### Budget Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `DefaultTokenLimit` | 100,000 | Max tokens per call |
| `DefaultDurationLimit` | 30 min | Max call duration |
| `WarningThreshold` | 0.8 (80%) | Emit warning at this usage fraction |
| `HardStop` | true | Terminate call when limit is hit |
| `DefaultLifetime` | 10 min | Default session TTL |
| `MaxLifetime` | 1 hour | Maximum session TTL |

### Audio Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `SampleRate` | 48,000 Hz | Opus standard sample rate |
| `Channels` | 1 (mono) | Voice calls use mono |
| `BitDepth` | 16 bit | PCM16 format |
| `FrameSize` | 960 samples | 20 ms frames at 48 kHz |
| `BufferSize` | 10 frames | 200 ms jitter buffer |
| `Bitrate` | 64,000 bps | Opus target bitrate |
| `Complexity` | 5 (0-10) | Encoder complexity |
| `FEC` | enabled | Forward error correction |
| `DTX` | disabled | Discontinuous transmission |

## Integration Points

### Matrix Rooms

Signaling uses the existing Matrix E2EE infrastructure. The Bridge sends SDP offers and answers as Matrix events, and ICE candidates are trickled through the same encrypted channel. Each voice session is bound to a Matrix room ID, and the `RequireMembership` config flag (on by default) ensures only room members can initiate calls.

### Budget System

The `voice.BudgetTracker` integrates with the Bridge's security logger. Every session start, end, budget warning, and enforcement action is logged as a security event. Token usage from the AI model's input (speech-to-text) and output (text-to-speech) is tracked per call and checked against limits every 30 seconds.

### Agent Runtime

Audio frames flow between the Bridge and agent containers through the `AudioPipeline`. The pipeline creates a `StreamPair` (inbound + outbound) for each session. The agent container receives decoded PCM audio and produces PCM audio back, without needing to know about WebRTC, Opus, or TURN. The Bridge handles all codec and protocol work.

### TURN Infrastructure

The `turn.Manager` generates ephemeral credentials scoped to individual sessions. The WebRTC engine calls `SetTURNServersWithManager` before each peer connection is created, getting fresh TURN URLs and credentials that expire with the session. This avoids static credentials and limits the blast radius of any leak.
