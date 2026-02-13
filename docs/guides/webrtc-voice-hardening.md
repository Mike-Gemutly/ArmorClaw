# WebRTC Voice Hardening Guide

> **Last Updated:** 2026-02-08
> **Component:** Voice Call System
> **Phase:** 5 - Hardening

## Overview

This guide describes the security hardening, budget enforcement, and monitoring features of the WebRTC voice system. These features protect against abuse, enforce resource limits, and ensure system stability under load.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Bridge                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐    │
│  │  Voice Manager │──│  Security      │──│  Budget        │    │
│  │                │  │  Enforcer      │  │  Tracker       │    │
│  └────────────────┘  └────────────────┘  └────────────────┘    │
│         │                     │                     │            │
│         ▼                     ▼                     ▼            │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐    │
│  │  Matrix        │  │  TTL Manager   │  │  Audit Log     │    │
│  │  Integration   │  │                │  │                │    │
│  └────────────────┘  └────────────────┘  └────────────────┘    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Security Features

### 1. Security Policies

The `SecurityPolicy` struct defines comprehensive security rules:

```go
type SecurityPolicy struct {
    // Maximum concurrent calls
    MaxConcurrentCalls int

    // Maximum call duration
    MaxCallDuration time.Duration

    // Allowed users (empty = all users allowed)
    AllowedUsers map[string]bool

    // Blocked users
    BlockedUsers map[string]bool

    // Allowed rooms (empty = all rooms allowed)
    AllowedRooms map[string]bool

    // Blocked rooms
    BlockedRooms map[string]bool

    // Require E2EE for calls
    RequireE2EE bool

    // Require TLS for signaling
    RequireSignalingTLS bool

    // Rate limiting (max calls per user per time window)
    RateLimitCalls     int
    RateLimitWindow  time.Duration
}
```

#### Default Security Policy

```go
policy := DefaultSecurityPolicy()
// MaxConcurrentCalls:     10
// MaxCallDuration:        1 hour
// RequireE2EE:            true
// RequireSignalingTLS:    true
// RateLimitCalls:         10 per hour
```

### 2. Access Control

#### User Allowlisting

```toml
[voice]
# Only allow specific users
allowed_users = ["@user1:example.com", "@user2:example.com"]
```

#### User Blocklisting

```toml
[voice]
# Block specific users
blocked_users = ["@abusive:example.com"]
```

#### Room Allowlisting

```toml
[voice]
# Only allow calls in specific rooms
allowed_rooms = ["!private:example.com", "!verified:example.com"]
```

#### Room Blocklisting

```toml
[voice]
# Block specific rooms
blocked_rooms = ["!spam:example.com"]
```

### 3. Rate Limiting

Rate limiting prevents users from making excessive calls:

```go
// Configure rate limit
policy.RateLimitCalls = 10          // Max 10 calls
policy.RateLimitWindow = 1 * time.Hour  // Per hour

// Check before starting call
err := enforcer.CheckStartCall(userID, roomID)
if err == voice.ErrRateLimitExceeded {
    // User has exceeded rate limit
}
```

### 4. Concurrent Call Limits

Limit the total number of active calls:

```go
// Configure concurrent call limit
policy.MaxConcurrentCalls = 10

// Check current usage
activeCount := enforcer.getActiveCallCount()
if activeCount >= policy.MaxConcurrentCalls {
    // Reject new calls
}
```

### 5. Call Duration Limits

Enforce maximum call duration:

```go
// Configure duration limit
policy.MaxCallDuration = 1 * time.Hour

// Validate during call
if time.Since(call.StartTime) > policy.MaxCallDuration {
    // Terminate call
    manager.EndCall(call.ID, "duration_exceeded")
}
```

## Budget Enforcement

### Token Budget

Track and limit token usage for AI-powered voice features:

```go
// Create budget tracker
config := voice.DefaultConfig()
config.DefaultTokenLimit = 100000  // 100k tokens per call
tracker := voice.NewBudgetTracker(config, budgetMgr)

// Start session with budget
session, _ := tracker.StartSession(
    sessionID,
    callID,
    roomID,
    tokenLimit,      // Custom limit
    durationLimit,   // Custom duration
)

// Record usage
tracker.RecordTokenUsage(
    sessionID,
    inputTokens,   // From user speech
    outputTokens,  // From TTS
    model,
)
```

### Duration Budget

Enforce maximum call duration:

```go
// Configure duration limit
config.DefaultDurationLimit = 30 * time.Minute

// Check periodically
err := tracker.CheckDuration(sessionID)
if err == voice.ErrDurationExceeded {
    // Call exceeded duration limit
    manager.EndCall(call.ID, "duration_limit_exceeded")
}
```

### Budget Warning Threshold

Warn before limits are exceeded:

```go
// Configure warning threshold
config.WarningThreshold = 0.8  // Warn at 80%

// Check automatically during usage recording
if usagePercentage >= config.WarningThreshold {
    // Emit warning log
    securityLog.LogSecurityEvent("voice_budget_warning", ...)
}
```

### Hard Stop vs Soft Limit

Configure behavior when limits are exceeded:

```go
config.HardStop = true  // Terminate call immediately
// OR
config.HardStop = false // Log warning, allow continuation
```

## TTL Management

### Session TTL

Automatically expire inactive sessions:

```go
// Configure TTL
config := voice.DefaultTTLConfig()
config.DefaultTTL = 10 * time.Minute
config.MaxTTL = 1 * time.Hour
config.HardStop = true

// Create TTL manager
sessions := webrtc.NewSessionManager(10 * time.Minute)
ttlMgr := voice.NewTTLManager(sessions, config)

// Start enforcement
ctx := context.Background()
ttlMgr.StartEnforcement(ctx)
```

### TTL Warning

Warn before TTL expires:

```go
config.WarningThreshold = 0.9  // Warn at 90% of TTL

// Automatic warning logs
// "voice_session_ttl_warning" with remaining time
```

### TTL Enforcement

```go
// Manual enforcement
err := ttlMgr.EnforceTTL(ctx)
if err != nil {
    // Sessions were expired
    fmt.Printf("%d sessions expired", expiredCount)
}

// Get statistics
stats := ttlMgr.GetTTLStats()
// "total_sessions", "expiring_soon", "average_remaining"
```

## Audit and Compliance

### Call Auditing

Generate audit records for every call:

```go
// Create security auditor
audit := voice.NewSecurityAudit(policy)

// Audit a call
record, _ := audit.AuditCall(call)

// Record contains:
// - Call metadata (ID, room, users, state)
// - Duration and timestamps
// - Token usage
// - Security events
// - Policy violations
```

### Audit Record

```go
type AuditRecord struct {
    CallID         string
    RoomID         string
    CallerID       string
    CalleeID       string
    State          string
    StartTime      time.Time
    EndTime        time.Time
    Duration       time.Duration
    TokenUsage     TokenUsage
    PolicyVersion  string
    Events         []AuditEvent
    Violations     []string
}
```

### Security Reports

Generate comprehensive security reports:

```go
// Generate report
report := audit.GenerateReport()

// Report contains:
// - Total calls
// - Violation counts
// - Recent calls (last 100)
// - Policy version
```

## Configuration

### Full Voice Configuration

```toml
# Voice call configuration
[voice]
# Limits
max_concurrent_calls = 10
max_call_duration = "1h"

# Access control
allowed_users = ["@user1:example.com"]
blocked_users = ["@abusive:example.com"]
allowed_rooms = ["!verified:example.com"]
blocked_rooms = ["!spam:example.com"]

# Security requirements
require_e2ee = true
require_signaling_tls = true

# Rate limiting
rate_limit_calls = 10
rate_limit_window = "1h"

# Budget enforcement
default_token_limit = 100000
default_duration_limit = "30m"
warning_threshold = 0.8
hard_stop = true

# TTL configuration
[voice.ttl]
default_ttl = "10m"
max_ttl = "1h"
enforcement_interval = "30s"
warning_threshold = 0.9
hard_stop = true
```

## Testing

### Load Testing

Test system performance under load:

```bash
# Run load test
cd tests/voice
./load-test.sh

# Configure load test
CONCURRENT_CALLS=50 \
CALLS_PER_SECOND=10 \
TEST_DURATION=120 \
./load-test.sh
```

#### Load Test Metrics

- Call creation time
- ICE candidate handling
- Session listing performance
- Sustained load handling
- Success rate

### Soak Testing

Test long-running stability:

```bash
# Run soak test (1 hour)
cd tests/voice
TEST_DURATION=3600 \
CALL_ROTATION=5 \
./soak-test.sh

# Run overnight test (8 hours)
TEST_DURATION=28800 \
CALL_ROTATION=10 \
./soak-test.sh
```

#### Soak Test Metrics

- Memory usage over time
- Session stability
- Call success rate
- Memory leak detection
- Resource cleanup

### Unit Testing

Run all voice tests:

```bash
cd bridge
go test ./pkg/voice/... -v
```

## Monitoring

### Security Events

Security events are logged with structured data:

```go
// Access denied
securityLog.LogAccessDenied(
    ctx,
    "voice_call_start",
    roomID,
    slog.String("user_id", userID),
    slog.String("reason", "user_blocked"),
)

// Security event
securityLog.LogSecurityEvent(
    "voice_budget_warning",
    slog.String("session_id", sessionID),
    slog.Float64("usage_percent", 85),
)
```

### Key Metrics

Monitor these metrics for system health:

1. **Active Calls**: Current number of active calls
2. **Call Success Rate**: Percentage of successful calls
3. **Average Duration**: Mean call duration
4. **Token Usage**: Total tokens consumed
5. **Memory Usage**: Bridge memory footprint
6. **Session Expirations**: Number of expired sessions

### Alerting

Set up alerts for:

- High failure rate (> 5%)
- Memory growth (> 100MB)
- Concurrent call limit reached
- Rate limit exceeded frequently
- Session expiration anomalies

## Troubleshooting

### Issue: Calls Failing with "rate_limit_exceeded"

**Solution**: Check rate limit configuration and increase if needed:

```toml
[voice]
rate_limit_calls = 20  # Increase from default 10
rate_limit_window = "1h"
```

### Issue: Calls Terminating After Default Duration

**Solution**: Configure appropriate duration limit:

```toml
[voice]
max_call_duration = "2h"  # Increase from default 1h
```

### Issue: Memory Growth Over Time

**Solution**: Run soak test to identify leaks:

```bash
cd tests/voice
./soak-test.sh
```

Check for:
- Session leaks (unterminated calls)
- Buffer leaks (audio buffers not released)
- Goroutine leaks (background processes)

### Issue: High CPU Usage

**Solution**: Check enforcement intervals:

```toml
[voice.ttl]
enforcement_interval = "60s"  # Reduce from default 30s
```

## Security Best Practices

1. **Enable E2EE**: Always require E2EE in production
2. **Require TLS**: Use WSS for signaling
3. **Set Reasonable Limits**: Configure limits based on expected load
4. **Monitor Usage**: Regularly review audit logs
5. **Test Limits**: Run load tests before production
6. **Budget Enforcement**: Enable hard stops for production
7. **Regular Audits**: Review security reports weekly

## Production Checklist

Before deploying to production:

- [ ] Configure appropriate concurrent call limits
- [ ] Set up user/room allowlists if needed
- [ ] Configure rate limiting
- [ ] Enable budget enforcement with hard stop
- [ ] Set up TTL enforcement
- [ ] Configure audit logging
- [ ] Set up monitoring and alerting
- [ ] Run load tests with expected traffic
- [ ] Run soak test for 24+ hours
- [ ] Review security audit reports
- [ ] Document escalation procedures

## API Reference

### SecurityEnforcer Methods

```go
// Check if a call can be started
CheckStartCall(userID, roomID string) error

// Register an active call
RegisterCall(roomID, userID string) error

// Unregister a call
UnregisterCall(roomID, userID string) error

// Validate call parameters
ValidateCallParameters(roomID, userID, sessionID string) error

// Audit a call
AuditCall(call *MatrixCall) *AuditRecord
```

### BudgetTracker Methods

```go
// Start tracking a session
StartSession(sessionID, callID, roomID string, tokenLimit uint64, durationLimit time.Duration) (*VoiceSessionTracker, error)

// Record token usage
RecordTokenUsage(sessionID string, inputTokens, outputTokens uint64, model string) error

// Check duration limit
CheckDuration(sessionID string) error

// End a session
EndSession(sessionID string) error

// Get usage statistics
GetUsage(sessionID string) (*TokenUsage, time.Duration, error)

// Enforce limits on all sessions
EnforceLimits(ctx context.Context) error
```

### TTLManager Methods

```go
// Enforce TTL on all sessions
EnforceTTL(ctx context.Context) error

// Start enforcement loop
StartEnforcement(ctx context.Context) error

// Get TTL statistics
GetTTLStats() map[string]interface{}
```

## Further Reading

- [WebRTC Voice Implementation Plan](../plans/2026-02-08-webrtc-voice-implementation.md)
- [Development Guide](development.md)
- [Troubleshooting Guide](troubleshooting.md)
