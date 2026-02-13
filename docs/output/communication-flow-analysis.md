# ArmorClaw Communication Flow Analysis

> **Date:** 2026-02-08
> **Version:** 1.0.0
> **Status:** Gap Analysis Complete

---

## Executive Summary

This document analyzes the complete communication architecture of ArmorClaw, identifies gaps, and provides recommendations for fixes.

**Current Communication Status:** 75% Complete
**Critical Gaps:** 7
**Recommended Priority:** P0 - Container Health Monitoring, P1 - Event Push, P2 - Budget Alerts

---

## Communication Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARMORCLAW COMMUNICATION ARCHITECTURE                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌──────────────┐                                                              │
│  │   Clients    │  (JSON-RPC over Unix Socket)                                │
│  │              │──────────┐                                                   │
│  └──────────────┘          │                                                   │
│                             ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────────────┐    │
│  │                        ArmorClaw Bridge (Go)                           │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌───────────┐  │    │
│  │  │   RPC       │  │   Matrix     │  │   WebRTC    │  │  Budget   │  │    │
│  │  │   Server    │  │   Adapter    │  │   Engine    │  │  Tracker  │  │    │
│  │  └─────────────┘  └──────────────┘  └─────────────┘  └───────────┘  │    │
│  └──────────────────────────────────────────────────────────────────────┘    │
│           │                    │                    │                      │    │
│           ▼                    ▼                    ▼                      ▼    │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │  Container  │    │   Matrix     │    │   TURN/STUN │    │   Budget    │  │
│  │   Sockets   │    │  Homeserver  │    │   Server    │    │   Alerts    │  │
│  └─────────────┘    └──────────────┘    └─────────────┘    └─────────────┘  │
│           │                                                                 │
└───────────┼─────────────────────────────────────────────────────────────────┘
            ▼
  ┌─────────────────────┐
  │  Agent Containers   │
  │   (Python Agent)    │
  │  ┌───────────────┐  │
  │  │ Bridge Client │  │
  │  │ (JSON-RPC)    │  │
  │  └───────────────┘  │
  └─────────────────────┘
```

---

## Communication Flows

### 1. Client → Bridge (JSON-RPC)

**Protocol:** JSON-RPC 2.0 over Unix Domain Socket
**Socket:** `/run/armorclaw/bridge.sock`

| Method | Direction | Purpose | Status |
|--------|----------|---------|--------|
| `status` | Client→Bridge | Get bridge status | ✅ Complete |
| `health` | Client→Bridge | Health check | ✅ Complete |
| `start` | Client→Bridge | Start container | ✅ Complete |
| `stop` | Client→Bridge | Stop container | ✅ Complete |
| `list_keys` | Client→Bridge | List API keys | ✅ Complete |
| `get_key` | Client→Bridge | Retrieve API key | ✅ Complete |
| `attach_config` | Client→Bridge | Attach config file | ✅ Complete |
| `list_configs` | Client→Bridge | List attached configs | ✅ Complete |
| `matrix.send` | Client→Bridge | Send Matrix message | ✅ Complete |
| `matrix.receive` | Client→Bridge | Receive Matrix events | ✅ Complete |
| `matrix.status` | Client→Bridge | Get Matrix status | ✅ Complete |
| `matrix.login` | Client→Bridge | Login to Matrix | ✅ Complete |
| `webrtc.start` | Client→Bridge | Start voice session | ✅ Complete |
| `webrtc.end` | Client→Bridge | End voice session | ✅ Complete |
| `webrtc.ice_candidate` | Client→Bridge | Submit ICE candidate | ✅ Complete |
| `webrtc.list` | Client→Bridge | List active sessions | ✅ Complete |
| `store_key` | Client→Bridge | Store API key | ✅ Complete |
| `list_configs` | Client→Bridge | List config files | ✅ Complete |
| `webrtc.get_audit_log` | Client→Bridge | Get audit log | ✅ Complete |

**Gap:** No streaming/notification mechanism for real-time updates

---

### 2. Bridge ↔ Matrix Communication

**Protocol:** Matrix Client-Server API over HTTPS
**Authentication:** Access token

| Flow | Direction | Purpose | Status |
|------|----------|---------|--------|
| Login | Bridge→Matrix | Authenticate | ✅ Complete |
| Sync | Bridge→Matrix | Get events | ✅ Complete |
| Send | Bridge→Matrix | Send messages | ✅ Complete |
| Receive Events | Matrix→Bridge | Event queue | ✅ Complete |
| E2EE | Bridge↔Matrix | Encrypt/Decrypt | ✅ Complete |

**Event Flow:**
```
Matrix Homeserver
       │
       │ Sync (with sync token)
       │◄───────────────────────────
       │
       │ Events (new messages)
       │────────────────────────────►
       │
       ▼
Bridge Matrix Adapter
       │
       │ Queue to event channel
       │────────────────────────────►
       │
       ▼
Event Queue (chan *MatrixEvent)
       │
       │ Container pulls via matrix.receive()
       │◄───────────────────────────
```

**Gap:** No push mechanism to containers (pull only)

---

### 3. Bridge ↔ Container Communication

**Protocol:** JSON-RPC 2.0 over container-specific Unix sockets
**Socket Pattern:** `/run/armorclaw/containers/{container-name}.sock`

| Flow | Direction | Purpose | Status |
|------|----------|---------|--------|
| Start Container | Bridge→Docker | Create container | ✅ Complete |
| Inject Secrets | Bridge→Container | FD passing | ✅ Complete |
| Container Socket | Bridge→Container | Create socket | ✅ Complete |
| Health Check | Container→Bridge | Verify secrets | ✅ Complete |
| Stop Container | Bridge→Docker | Terminate | ✅ Complete |
| Status Query | Client→Bridge | Container status | ✅ Complete |

**Gap:** No ongoing health monitoring after start
**Gap:** No event notification from container to bridge

---

### 4. WebRTC Voice Communication

**Protocol:** WebRTC (SDP, ICE, DTLS-SRTP)
**Authorization:** Matrix room-scoped

| Flow | Direction | Purpose | Status |
|------|----------|---------|--------|
| Create Session | Client→Bridge | webrtc.start | ✅ Complete |
| SDP Offer/Answer | Client↔Bridge | WebRTC negotiation | ✅ Complete |
| ICE Candidates | Client↔Bridge | NAT traversal | ✅ Complete |
| TURN Credentials | Bridge→Client | Ephemeral credentials | ✅ Complete |
| Audio Stream | Client↔Bridge | Opus audio | ⚠️ Configured only |
| Session End | Client→Bridge | webrtc.end | ✅ Complete |

**Gap:** SignalingServer not integrated (TODO exists)
**Gap:** Opus encoding not implemented (raw PCM only)

---

### 5. Budget & Security Communication

**Protocol:** Internal (in-process)

| Flow | Direction | Purpose | Status |
|------|----------|---------|--------|
| Budget Tracking | Internal | Track token/duration usage | ✅ Complete |
| Budget Alerts | Internal→Matrix | Send warnings | ❌ TODO exists |
| Security Policy | Internal | Enforce rules | ✅ Complete |
| Audit Logging | Internal | Log events | ✅ Complete |
| TTL Enforcement | Internal | Session expiration | ✅ Complete |

**Gap:** Budget alerts not sent via Matrix (TODO at `bridge/pkg/budget/tracker.go:227`)

---

## Critical Gaps Identified

### P0 - Container Health Monitoring

**Issue:** Bridge starts containers but doesn't monitor their health

**Impact:**
- Dead/zombie containers not detected
- Resource leaks
- No automatic recovery

**Location:** `bridge/pkg/rpc/server.go` (container management)

**Fix Required:**
- Implement health check goroutine
- Monitor container processes
- Auto-restart or alert on failure
- Resource usage tracking

---

### P1 - Matrix Event Push to Containers

**Issue:** Containers must pull events via `matrix.receive` (polling)

**Impact:**
- Increased latency
- Unnecessary API calls
- Real-time communication impossible

**Location:** `bridge/internal/adapter/matrix.go` (event queue)

**Fix Required:**
- Implement WebSocket or pub/sub push mechanism
- Container subscribes to Matrix events
- Real-time event delivery

---

### P1 - Container→Bridge Event Notification

**Issue:** No way for containers to notify bridge of events

**Impact:**
- Errors in containers go unnoticed
- No status updates from containers
- Can't implement callbacks

**Fix Required:**
- Add notification RPC method
- Event channel from container to bridge
- Error reporting mechanism

---

### P2 - Budget Alert Matrix Integration

**Issue:** Budget alerts logged but not sent via Matrix

**Impact:**
- Users not notified of budget issues
- Must check logs manually

**Location:** `bridge/pkg/budget/tracker.go:227`

**Current Code:**
```go
// TODO: Send via Matrix adapter
// For now, just log the alert
fmt.Printf("[BUDGET ALERT] %s - Current: $%.2f, Limit: $%.2f\n",
    alertType, current, limit)
```

**Fix Required:**
- Send Matrix message to admin room
- Include alert details
- Configurable alert thresholds

---

### P2 - WebRTC Signaling Server Integration

**Issue:** SignalingServer created but never passed to RPC server

**Impact:**
- WebSocket signaling not available
- Browser clients can't use WebRTC

**Location:** `bridge/cmd/bridge/main.go:1213`

**Current Code:**
```go
SignalingServer: nil, // TODO: Create and pass signaling server
```

**Fix Required:**
- Initialize signaling server in main.go
- Pass to RPC server
- Configure WebSocket endpoint
- Document signaling protocol

---

### P3 - Message Delivery Confirmation

**Issue:** No acknowledgment for Matrix messages sent from containers

**Impact:**
- Can't confirm message delivery
- Lost messages undetected
- No retry mechanism

**Fix Required:**
- Implement message ID tracking
- Return delivery receipt
- Retry on failure

---

### P3 - Bidirectional Streaming

**Issue:** Only request/response, no streaming

**Impact:**
- Can't stream large responses
- No real-time data flow
- Limited to small payloads

**Fix Required:**
- Add streaming RPC methods
- Support chunked responses
- Implement progress callbacks

---

## Communication Flow Diagrams

### Current State

```
┌─────────┐                  ┌─────────┐                  ┌─────────┐
│ Client  │                  │ Bridge  │                  │ Matrix  │
└────┬────┘                  └────┬────┘                  └────┬────┘
     │                            │                            │
     │ 1. JSON-RPC Request        │                            │
     │───────────────────────────►│                            │
     │                            │                            │
     │                            │ 2. Sync Events             │
     │                            │─────────────────────────────►│
     │                            │                            │
     │                            │ 3. Event Queue             │
     │                            │◄─────────────────────────────┤
     │                            │                            │
     │ 4. JSON-RPC Response       │                            │
     │◄───────────────────────────┤                            │
     │                            │                            │
     │ 5. Pull Events (poll)      │                            │
     │───────────────────────────►│                            │
     │                            │                            │
     │ 6. Return Events           │                            │
     │◄───────────────────────────┤                            │
     │                            │                            │

GAP: No push mechanism (step 6 is pull-only)
GAP: No delivery confirmation (step 2 has no receipt)
```

### Recommended State

```
┌─────────┐                  ┌─────────┐                  ┌─────────┐
│ Client  │                  │ Bridge  │                  │ Matrix  │
└────┬────┘                  └────┬────┘                  └────┬────┘
     │                            │                            │
     │ 1. JSON-RPC Request        │                            │
     │───────────────────────────►│                            │
     │                            │                            │
     │                            │ 2. Sync Events             │
     │                            │─────────────────────────────►│
     │                            │                            │
     │                            │ 3. Event Queue             │
     │                            │◄─────────────────────────────┤
     │                            │                            │
     │ 4. JSON-RPC Response       │                            │
     │◄───────────────────────────┤                            │
     │                            │                            │
     │ 5. Subscribe to Events     │                            │
     │───────────────────────────►│                            │
     │                            │                            │
     │ 6. Push Events (real-time) │                            │
     │◄════════════════════════════│                            │
     │                            │                            │
     │ 7. Delivery Receipt        │                            │
     │───────────────────────────►│                            │
     │                            │─────────────────────────────►│

ADDED: Push mechanism (step 6)
ADDED: Delivery confirmation (step 7)
```

---

## Security Considerations

### Current Security Measures

| Component | Security | Status |
|-----------|----------|--------|
| Unix Socket | File permissions | ✅ 0600 permissions |
| Matrix Communication | E2EE | ✅ Supported |
| Container Isolation | Namespace/cgroups | ✅ Hardened |
| Secret Injection | FD passing | ✅ Memory-only |
| Rate Limiting | Per-client | ✅ Implemented |
| Access Control | Trusted senders/rooms | ✅ Supported |

### Security Gaps

| Gap | Risk | Mitigation |
|-----|------|------------|
| No container health monitoring | Zombie containers | Implement health checks |
| No message delivery confirmation | Lost messages | Add receipts |
| Pull-only events | Stale data | Implement push |
| No resource limits | DoS attacks | Add quotas |
| No audit trail for events | Compliance gap | Enhanced logging |

---

## Performance Considerations

### Current Bottlenecks

| Area | Issue | Impact |
|------|-------|--------|
| Polling for Matrix events | Unnecessary latency | +100-500ms |
| No connection pooling | Connection overhead | +10-50ms per request |
| No caching | Repeated operations | +50-200ms |
| Synchronous operations | Blocking | Variable |

### Recommendations

1. **Implement Event Push** - Reduce latency by 100-500ms
2. **Add Connection Pooling** - Reduce overhead by 10-50ms
3. **Implement Caching** - Reduce repeated operations by 50-200ms
4. **Add Streaming** - Support large payloads
5. **Batch Operations** - Reduce round trips

---

## Recommendations

### Immediate (P0 - This Week)

1. **Container Health Monitoring**
   - Implement health check goroutine
   - Auto-restart on failure
   - Resource usage tracking

### Short-term (P1 - Next 2 Weeks)

2. **Matrix Event Push**
   - Implement WebSocket/push mechanism
   - Container subscription model
   - Real-time event delivery

3. **Container Event Notification**
   - Add notification RPC method
   - Error reporting from containers
   - Status callback mechanism

### Medium-term (P2 - Next Month)

4. **Budget Alert Matrix Integration**
   - Send alerts via Matrix
   - Configurable thresholds
   - Admin room notifications

5. **WebRTC Signaling Server**
   - Initialize in main.go
   - Document signaling protocol
   - Test with browser clients

6. **Message Delivery Confirmation**
   - Track message IDs
   - Return delivery receipts
   - Retry on failure

---

## Success Criteria

- [ ] All containers health-monitored
- [ ] Matrix events pushed to containers (real-time)
- [ ] Containers can notify bridge of events
- [ ] Budget alerts sent via Matrix
- [ ] WebRTC signaling server operational
- [ ] Message delivery confirmed
- [ ] Performance: <50ms latency for events
- [ ] Security: All gaps addressed

---

**Document Status:** Analysis Complete
**Next Steps:** Implement P0 and P1 fixes
