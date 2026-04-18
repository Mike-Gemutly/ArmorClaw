# ArmorClaw — Project Review

**Version**: v4.8.0 | **State**: Production Ready | **Last Updated**: 2026-04-18

> Entry point for anyone planning or modifying code. Read this before AGENTS.md guardrails take effect.
> See `CHANGELOG.md` for version history, `doc/` for detailed architecture documentation.

## Project Status

| Field | Value |
|-------|-------|
| Released version | v4.8.0 |
| Target version | v0.7.0 |
| Overall health | Stable — production deployed, integration gaps under active development |

## Subsystem Status

| Subsystem | Status | Key Location |
|-----------|--------|-------------|
| Bridge (Go) | Production Ready | `bridge/` |
| Matrix Conduit | Production Ready | Conduit homeserver |
| ArmorChat (Android) | Active Development (v0.7.0) | `applications/ArmorChat/` |
| Admin Panel (React) | Active Development (v0.7.0) | `applications/admin-panel/` |
| Jetski (CDP Proxy) | Production Ready | `jetski/` |
| Rust Sidecar | Production Ready | `sidecar/` |
| Python Sidecar | Production Ready | `sidecar-python/` |
| browser-service (TS/Playwright) | Production Ready | `browser-service/` |
| Secretary / Workflow Engine | Production Ready | `bridge/pkg/secretary/` |
| Email HITL | Production Ready | `bridge/pkg/email/` |
| Voice (WebRTC) | Production Ready | `bridge/pkg/webrtc/` |
| ArmorTerminal | Production Ready | `applications/ArmorTerminal/` |
| OpenClaw (Agent Runtime) | Production Ready | `container/openclaw/` |

## Architectural Decisions

### NetworkMode: none (absolute)

All agent containers run with `NetworkMode: none`. No container networking, no exceptions. This is a non-negotiable security constraint. Do not add IPC channels, bind-mounted event files, or any mechanism that would require network access for containers.

### Crash-only WebSocket design

`EventBus` wiring (`bridge/pkg/eventbus/eventbus.go:146`) uses `log.Fatalf` when WebSocket initialization fails. This is intentional. The bridge must crash rather than run in a degraded state where events are silently lost. Do not add graceful fallback or retry logic to this path without CTO approval.

### Cold-only dispatch

Warm dispatch (`warmDispatch()` in `TaskScheduler`) is architecturally illegal under `NetworkMode: none`. Containers cannot receive inbound connections, so warm dispatch can never work. Deprecated and scheduled for removal in v0.7.0.

### Zero-trust PII

- **BlindFill**: Secrets injected directly into browser form fields via memory-only Unix sockets. Agents request references, never see raw values.
- **Governor-Shield**: Tool call arguments scrubbed before reaching agents.
- **ShadowMap**: PII detection patterns for active masking in transit.

## Deprecation Notices

- **Warm dispatch** — Deprecated in v0.7.0. NetworkMode: none makes it architecturally illegal. Dead code removal in progress.

## Known Gaps (v0.7.0 scope)

See `.sisyphus/plans/v070-master-plan.md` for the full task breakdown.

- Warm dispatch dead code still present in `bridge/pkg/secretary/`
- WebSocket `bridge/pkg/websocket/websocket.go` is a 74-line stub — EventBus cannot publish to clients
- `DeepLinkHandler.kt` missing — notification taps do not navigate to correct screen
- `SecurityConfigViewModel` defined but never wired to `SecurityConfigScreen` — permissions not persisted
- `WorkflowStep` has no `Input` field — sequential step data propagation impossible
- Admin panel uses mock data instead of real Bridge API
- `BridgeRepository` credentials are in-memory only — not persisted across app restarts
- All 12 ArmorChat integration tests are `assertTrue(true)` placeholders

## Active Plan

v0.7.0 master plan: `.sisyphus/plans/v070-master-plan.md`

## Security Constraints

- Do not remove SQLCipher keystore
- Do not bypass Matrix as control plane
- Do not weaken approval flow for payments or critical PII
- Do not introduce direct production secret access
- Prefer minimal patches over rewrites
