# WebRTC Voice User Guide

> **ArmorClaw WebRTC Voice Call System**
> Last Updated: 2026-02-08
> Version: 1.0.0

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)
4. [API Reference](#api-reference)
5. [Security Features](#security-features)
6. [Budget Management](#budget-management)
7. [Troubleshooting](#troubleshooting)
8. [Examples](#examples)

---

## Overview

ArmorClaw WebRTC Voice provides secure, budget-controlled voice calls through Matrix authorization and WebRTC transport. The system integrates:

- **WebRTC Engine**: Pion-based WebRTC implementation for peer-to-peer audio
- **Matrix Integration**: Call authorization and signaling through Matrix rooms
- **Budget Enforcement**: Token and duration limits for cost control
- **Security Policies**: Rate limiting, E2EE requirements, audit logging
- **TTL Management**: Automatic session expiration and cleanup

### Key Features

- ✅ End-to-end encrypted audio via Opus codec
- ✅ Matrix-based call authorization (not transport)
- ✅ Budget enforcement with configurable limits
- ✅ Security policies (rate limiting, allowlists, concurrent call limits)
- ✅ NAT traversal via TURN/STUN with ephemeral credentials
- ✅ Session TTL management with automatic expiration
- ✅ Comprehensive security audit logging

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         ArmorClaw Bridge                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│  │   Matrix     │      │    WebRTC    │      │   Budget     │  │
│  │   Adapter    │─────▶│    Engine    │─────▶│   Tracker    │  │
│  │              │      │              │      │              │  │
│  └──────────────┘      └──────────────┘      └──────────────┘  │
│         │                       │                       │       │
│         ▼                       ▼                       ▼       │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│  │    Security  │      │     TTL      │      │    Audit     │  │
│  │   Enforcer   │      │   Manager    │      │     Log      │  │
│  └──────────────┘      └──────────────┘      └──────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────┐
                    │  JSON-RPC    │
                    │   Server     │
                    └──────────────┘
```

### Component Overview

| Component | Purpose |
|-----------|---------|
| **Matrix Adapter** | Handles Matrix call events, E2EE verification |
| **WebRTC Engine** | Manages peer connections, ICE candidates, media |
| **Budget Tracker** | Enforces token and duration limits |
| **Security Enforcer** | Applies rate limits, allowlists, concurrent call limits |
| **TTL Manager** | Manages session lifetime and expiration |
| **Audit Log** | Records all security-relevant events |

---

## Configuration

### Enable WebRTC Voice

Edit your `~/.armorclaw/config.toml`:

```toml
[webrtc]
enabled = true

[voice]
enabled = true
```

### Basic Voice Configuration

```toml
[voice.general]
# Call lifetime limits
default_lifetime = "30m"      # Default: 30 minutes
max_lifetime = "2h"           # Maximum: 2 hours

# Call behavior
auto_answer = false           # Require manual answer
require_membership = true     # Bridge must be in room

# Limits
max_concurrent_calls = 5      # Maximum simultaneous calls
```

### Security Configuration

```toml
[voice.security]
# Encryption requirements
require_e2ee = true           # Require E2EE for all calls
min_e2ee_algorithm = "megolm.v1.aes-sha2"

# Rate limiting
rate_limit = 10               # 10 calls/minute per user
rate_limit_burst = 20         # Temporary burst allowance

# Approval workflow
require_approval = false      # Require manual approval for new callers
approval_timeout = "5m"       # Approval decision timeout

# Audit
audit_calls = true            # Log all calls to audit trail
max_participants = 2          # 1 = direct calls only
```

### Budget Configuration

```toml
[voice.budget]
enabled = true

# Per-call limits
default_token_limit = 3600    # ~1 hour of audio
default_duration_limit = "1h"

# Warning threshold
warning_threshold = 0.8       # Warn at 80% usage

# Hard stop
hard_stop = false             # Terminate immediately on limit

# Global limits (optional)
global_token_limit = 0        # 0 = use per-call limits only
global_duration_limit = "0s"  # 0 = unlimited
```

### TTL Configuration

```toml
[voice.ttl]
# Session lifetime
default_ttl = "30m"           # Default session TTL
max_ttl = "2h"                # Maximum allowed TTL

# Enforcement
enforcement_interval = "1m"   # Check for expired sessions every minute

# Warnings
warn_before_expiration = "5m" # Warn 5 minutes before expiration

# Expiration action
on_expiration = "terminate"   # Options: terminate, warn, extend
```

### Room Access Control

```toml
[voice.rooms]
# Whitelist (only allow these rooms)
allowed = [
    "!roomid1:matrix.example.com",
    "!roomid2:matrix.example.com",
]

# Blacklist (block these rooms)
blocked = [
    "!blockedroom:matrix.example.com",
]
```

---

## API Reference

All WebRTC Voice methods are available via the JSON-RPC 2.0 API over the Unix socket at `/run/armorclaw/bridge.sock`.

### Create Call

Initiate a new voice call.

**Method:** `webrtc.create_call`

**Parameters:**

```json
{
  "room_id": "!abc123:matrix.example.com",
  "offer_sdp": "v=0\r\no=- 123456 2 IN IP4 127.0.0.1\r\n...",
  "user_id": "@user:matrix.example.com"
}
```

**Response:**

```json
{
  "call_id": "call-uuid-123",
  "answer_sdp": "v=0\r\no=- 789012 2 IN IP4 127.0.0.1\r\n...",
  "ttl": "30m",
  "token": "jwt-token-for-session"
}
```

**Errors:**

- `ErrMaxConcurrentCallsExceeded`: Maximum concurrent calls limit reached
- `ErrSecurityPolicyViolation`: Security policy check failed
- `ErrBudgetExceeded`: Budget limit exceeded

### Answer Call

Answer an incoming call.

**Method:** `webrtc.answer_call`

**Parameters:**

```json
{
  "call_id": "call-uuid-123",
  "answer_sdp": "v=0\r\no=- 789012 2 IN IP4 127.0.0.1\r\n..."
}
```

**Response:**

```json
{
  "success": true
}
```

**Errors:**

- `ErrCallNotFound`: Call ID not found
- `ErrInvalidState`: Call not in ringing state

### Reject Call

Reject an incoming call.

**Method:** `webrtc.reject_call`

**Parameters:**

```json
{
  "call_id": "call-uuid-123",
  "reason": "user declined"
}
```

**Response:**

```json
{
  "success": true
}
```

### End Call

Terminate an active call.

**Method:** `webrtc.end_call`

**Parameters:**

```json
{
  "call_id": "call-uuid-123",
  "reason": "call completed"
}
```

**Response:**

```json
{
  "success": true
}
```

### Send ICE Candidates

Send ICE candidates for a call.

**Method:** `webrtc.send_candidates`

**Parameters:**

```json
{
  "call_id": "call-uuid-123",
  "candidates": [
    {
      "candidate": "candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host",
      "sdp_mid": "0",
      "sdp_mline_index": 0
    }
  ]
}
```

**Response:**

```json
{
  "success": true
}
```

### Get Call Status

Retrieve the status of a call.

**Method:** `webrtc.get_call_status`

**Parameters:**

```json
{
  "call_id": "call-uuid-123"
}
```

**Response:**

```json
{
  "call_id": "call-uuid-123",
  "state": "connected",
  "duration": "5m23s",
  "tokens_used": 323,
  "budget_remaining": 3277
}
```

### List Calls

List all active calls.

**Method:** `webrtc.list`

**Parameters:** None

**Response:**

```json
{
  "calls": [
    {
      "call_id": "call-uuid-123",
      "room_id": "!abc123:matrix.example.com",
      "state": "connected",
      "duration": "5m23s",
      "tokens_used": 323
    }
  ]
}
```

---

## Security Features

### End-to-End Encryption

All voice calls require E2EE by default. The system verifies:

1. **Device Keys**: All participant devices are verified
2. **Encryption Algorithm**: Minimum `megolm.v1.aes-sha2` required
3. **Room Encryption**: Call must be in an encrypted room

### Rate Limiting

Prevents abuse by limiting call creation rate:

```toml
[voice.security]
rate_limit = 10              # 10 calls per minute
rate_limit_burst = 20        # Temporary burst allowance
```

When rate limit is exceeded:
- Call requests are rejected with `ErrRateLimitExceeded`
- Event is logged to security audit log
- Client should retry after `Retry-After` header

### Concurrent Call Limits

Controls maximum simultaneous calls:

```toml
[voice.general]
max_concurrent_calls = 5
```

When limit is reached:
- New calls are rejected with `ErrMaxConcurrentCallsExceeded`
- Event is logged to security audit log
- Active calls are not affected

### Room Access Control

Whitelist and blacklist specific rooms:

```toml
[voice.rooms]
allowed = ["!trusted:matrix.example.com"]
blocked = ["!blocked:matrix.example.com"]
```

### Security Audit Log

All security-relevant events are logged:

- Call creation, answering, rejection, termination
- Budget warnings and exceedances
- Security policy violations
- Rate limit events
- E2EE verification failures

Access audit log via RPC:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.get_audit_log"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Budget Management

### Token-Based Budgeting

Each call consumes tokens based on audio processing:

- **1 token ≈ 1 second** of audio at default settings
- **Default limit**: 3600 tokens (~1 hour)
- **Warning threshold**: 80% of budget used

### Duration Limits

Separate time-based limits:

```toml
[voice.budget]
default_duration_limit = "1h"  # Hard time limit
```

### Budget Warnings

System warns when approaching limits:

```json
{
  "warning": "budget_warning",
  "call_id": "call-uuid-123",
  "tokens_used": 2880,
  "tokens_remaining": 720,
  "percentage_used": 0.8
}
```

### Hard Stop

Enable immediate termination on limit exceed:

```toml
[voice.budget]
hard_stop = true
```

When `hard_stop = false`:
- Call continues after limit
- Warning logged to audit log
- No new tokens allocated

---

## Troubleshooting

### Call Creation Fails

**Error:** `ErrMaxConcurrentCallsExceeded`

**Solution:** Increase limit or wait for active calls to complete:

```toml
[voice.general]
max_concurrent_calls = 10
```

**Error:** `ErrSecurityPolicyViolation`

**Solution:** Check security policies:
- Verify E2EE is enabled in Matrix room
- Check if room is in blocked list
- Verify rate limits not exceeded

### Call Drops Unexpectedly

**Cause 1:** TTL expired

**Solution:** Increase session TTL:

```toml
[voice.ttl]
default_ttl = "1h"
max_ttl = "4h"
```

**Cause 2:** Budget exceeded

**Solution:** Increase budget limits:

```toml
[voice.budget]
default_token_limit = 7200     # 2 hours
default_duration_limit = "2h"
```

**Cause 3:** TURN credential expired

**Solution:** Check TURN server configuration and credential lifetime

### No Audio in Call

**Check 1:** WebRTC not enabled

```bash
# Check config
grep -A1 "\[webrtc\]" ~/.armorclaw/config.toml

# Should show: enabled = true
```

**Check 2:** ICE candidates not exchanged

Verify ICE candidates are being sent:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"webrtc.list"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock | jq .
```

**Check 3:** Firewall blocking UDP

Open UDP ports for media:

```bash
# Check if ports are open
sudo ufw status
```

### Matrix Events Not Received

**Check 1:** Matrix sync interval

```toml
[matrix]
sync_interval = 5  # seconds
```

**Check 2:** Bridge not in room

```toml
[voice.general]
require_membership = true  # Bridge must be in room
```

**Check 3:** E2EE verification failing

```toml
[voice.security]
require_e2ee = true
min_e2ee_algorithm = "megolm.v1.aes-sha2"
```

---

## Examples

### Basic Voice Call Flow

```bash
# 1. Create a call
CALL_RESPONSE=$(echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "webrtc.create_call",
  "params": {
    "room_id": "!abc123:matrix.example.com",
    "offer_sdp": "v=0\\r\\no=- 123456 2 IN IP4 127.0.0.1\\r\\n...",
    "user_id": "@user:matrix.example.com"
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock)

# Extract call_id and answer_sdp
CALL_ID=$(echo "$CALL_RESPONSE" | jq -r '.result.call_id')
ANSWER_SDP=$(echo "$CALL_RESPONSE" | jq -r '.result.answer_sdp')

# 2. Answer the call
echo '{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "webrtc.answer_call",
  "params": {
    "call_id": "'"$CALL_ID"'",
    "answer_sdp": "v=0\\r\\no=- 789012 2 IN IP4 127.0.0.1\\r\\n..."
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 3. Send ICE candidates
echo '{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "webrtc.send_candidates",
  "params": {
    "call_id": "'"$CALL_ID"'",
    "candidates": [
      {
        "candidate": "candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host",
        "sdp_mid": "0",
        "sdp_mline_index": 0
      }
    ]
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 4. End call when done
echo '{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "webrtc.end_call",
  "params": {
    "call_id": "'"$CALL_ID"'",
    "reason": "call completed"
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Check Call Status

```bash
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "webrtc.get_call_status",
  "params": {
    "call_id": "call-uuid-123"
  }
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock | jq .
```

### List All Active Calls

> ⚠️ **Planned feature - not yet implemented**

```bash
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "webrtc.list"
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock | jq .
```

### Get Security Audit Log

```bash
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "webrtc.get_audit_log"
}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock | jq .
```

---

## Support

For issues or questions:

1. Check the [Error Catalog](error-catalog.md) for specific error solutions
2. Review [Troubleshooting Guide](troubleshooting.md) for systematic debugging
3. Check [Project Status](../status/2026-02-05-status.md) for known issues
4. Create an issue on [GitHub](https://github.com/armorclaw/armorclaw)

---

**Last Updated:** 2026-02-08
**Version:** 1.0.0
