# Alert Integration Guide

> **Purpose:** Configure proactive monitoring and alerts for ArmorClaw
> **Last Updated:** 2026-02-15
> **Integration:** Matrix notifications via error system

---

## Overview

ArmorClaw provides proactive monitoring through the error notification system. This guide documents how to configure alert rules and receive notifications via Matrix when issues occur.

### Alert Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ALERT ARCHITECTURE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                   │
│  │  Health     │────▶│  Error      │────▶│  Matrix     │                   │
│  │  Monitor    │     │  System     │     │  Notifier   │                   │
│  └─────────────┘     └─────────────┘     └─────────────┘                   │
│         │                   │                   │                            │
│         │                   │                   ▼                            │
│         │                   │           ┌─────────────┐                     │
│         │                   │           │  Admin      │                     │
│         │                   │           │  Room       │                     │
│         │                   │           └─────────────┘                     │
│         │                   │                                                  │
│         ▼                   ▼                                                  │
│  ┌─────────────────────────────────────────────────────────┐                │
│  │                    ALERT RULES                           │                │
│  │  • Container health failures                            │                │
│  │  • Error rate thresholds                                │                │
│  │  • Budget limit warnings                                │                │
│  │  • Queue depth alerts                                   │                │
│  │  • Authentication failures                              │                │
│  └─────────────────────────────────────────────────────────┘                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Alert Severity Levels

| Level | Code Range | Description | Notification |
|-------|------------|-------------|--------------|
| **CRITICAL** | CTX-003, SYS-010, BGT-002 | System unusable, immediate action | Matrix + Log |
| **ERROR** | CTX-001, MAT-001, RPC-010 | Feature degraded, action needed | Matrix + Log |
| **WARNING** | MAT-003, BGT-001, RPC-011 | Potential issue, monitor | Log (Matrix optional) |
| **INFO** | - | Normal operations | Log only |

---

## Built-in Alert Rules

### Container Alerts

| Alert | Trigger | Code | Action |
|-------|---------|------|--------|
| Container Health Failure | 3 consecutive failed health checks | CTX-003 | Notify admin |
| Container Start Failure | Container exits < 5s after start | CTX-001 | Notify admin |
| Container Exec Failure | Exec command fails | CTX-002 | Log + notify on 3rd failure |
| Image Pull Failure | Cannot pull container image | CTX-020 | Notify admin |

### Matrix Alerts

| Alert | Trigger | Code | Action |
|-------|---------|------|--------|
| Connection Failed | Cannot reach homeserver | MAT-001 | Notify admin |
| Authentication Failed | Invalid token/credentials | MAT-002 | Notify admin immediately |
| Sync Timeout | Sync takes > 60s | MAT-003 | Log + notify on 3rd occurrence |
| Message Send Failed | Cannot send message | MAT-021 | Log + retry |

### System Alerts

| Alert | Trigger | Code | Action |
|-------|---------|------|--------|
| Keystore Decryption Failed | Wrong master key | SYS-001 | Notify admin immediately |
| Secret Injection Failed | Cannot inject secrets | SYS-010 | Notify admin immediately |
| Out of Memory | System memory < 10% | SYS-020 | Notify admin |
| Disk Full | Disk usage > 90% | SYS-021 | Notify admin |

### Budget Alerts

| Alert | Trigger | Code | Action |
|-------|---------|------|--------|
| Budget Warning | Usage > 80% of limit | BGT-001 | Log + optional Matrix |
| Budget Exceeded | Usage >= limit | BGT-002 | Notify admin + block |

---

## Configuring Alerts

### Method 1: RPC Configuration

Use the `get_errors` RPC method to monitor for specific error codes:

```bash
# Check for critical errors in last hour
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{
  "severity": "critical",
  "resolved": false
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Check for container errors
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{
  "category": "container",
  "resolved": false
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Method 2: Programmatic Monitoring

The error system automatically sends Matrix notifications for critical and error severity issues when configured:

```go
// In bridge initialization
errorSystem := errors.NewErrorSystem(errors.SystemConfig{
    AdminResolver: resolver,
    MatrixSender:  matrixAdapter,
    EnableNotifications: true,
    NotifySeverities: []errors.Severity{
        errors.SeverityCritical,
        errors.SeverityError,
    },
})
```

### Method 3: Log Monitoring

Monitor security logs for alert conditions:

```bash
# Watch for critical errors
tail -f /var/log/armorclaw/security.log | grep "severity=critical"

# Watch for authentication failures
tail -f /var/log/armorclaw/security.log | grep "MAT-002"
```

---

## Alert Notification Format

When an alert is triggered, the Matrix notification includes:

```
[ArmorClaw Error Trace]
Code: CTX-003
Category: container
Severity: critical
Trace ID: tr_abc123def456
Function: HealthCheck
Timestamp: 2026-02-15T12:00:00Z

Message: container health check timeout
Help: Container may be hung; check logs and consider restart

Inputs:
  container_id: abc123
  container_name: armorclaw-openclaw-1738864000

State:
  status: unhealthy
  failure_count: 3

Component Events:
  [docker] health_check - FAILED at 2026-02-15T12:00:00Z
[/ArmorClaw Error Trace]
```

---

## Operational Runbooks

### Runbook: Container Health Failure (CTX-003)

**Symptoms:**
- Container marked unhealthy
- Repeated health check failures

**Investigation:**
```bash
# 1. Check container logs
docker logs <container_id>

# 2. Check container status
docker inspect <container_id> | jq '.[0].State'

# 3. Check resource usage
docker stats <container_id> --no-stream
```

**Resolution:**
```bash
# If container is hung, restart it
echo '{"jsonrpc":"2.0","id":1,"method":"stop","params":{"container_id":"<id>"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Then start a new instance
echo '{"jsonrpc":"2.0","id":1,"method":"start","params":{"key_id":"my-key"}}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### Runbook: Matrix Connection Failed (MAT-001)

**Symptoms:**
- Cannot send/receive Matrix messages
- Sync failing

**Investigation:**
```bash
# 1. Check Matrix server status
curl -s https://matrix.example.com/_matrix/client/versions

# 2. Check bridge Matrix status
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.status"}' | \
  socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# 3. Check network connectivity
ping matrix.example.com
```

**Resolution:**
```bash
# Re-authenticate to Matrix
echo '{"jsonrpc":"2.0","id":1,"method":"matrix.login","params":{
  "username": "bridge-bot",
  "password": "secret"
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

### Runbook: Budget Exceeded (BGT-002)

**Symptoms:**
- Operations blocked
- Budget limit reached

**Investigation:**
```bash
# 1. Check current budget status
# (Via application-specific metrics)

# 2. Review recent usage
echo '{"jsonrpc":"2.0","id":1,"method":"get_errors","params":{
  "category": "budget"
}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

**Resolution:**
1. Review usage patterns
2. Increase budget limits if appropriate
3. Wait for budget reset window
4. Consider rate limiting heavy users

---

### Runbook: Secret Injection Failed (SYS-010)

**Symptoms:**
- Container cannot start
- API keys not available

**Investigation:**
```bash
# 1. Check secrets directory
ls -la /run/armorclaw/secrets/

# 2. Verify key exists in keystore
./build/armorclaw-bridge list-keys

# 3. Check bridge logs
journalctl -u armorclaw-bridge -n 100
```

**Resolution:**
```bash
# Re-add the API key if corrupted
./build/armorclaw-bridge add-key \
  --provider openai \
  --token sk-xxx \
  --id my-key \
  --force
```

---

## Alert Rule Configuration File

Create `/etc/armorclaw/alerts.toml`:

```toml
# Alert configuration for ArmorClaw

[notifications]
enabled = true
# Matrix room for admin notifications
admin_room = "!admin:matrix.example.com"
# Minimum severity to notify (debug, info, warning, error, critical)
min_severity = "error"

[alert_rules.container_health]
enabled = true
# Failures before alert
threshold = 3
# Time window for counting failures
window = "5m"
severity = "critical"

[alert_rates.matrix_auth]
enabled = true
# Alert on any auth failure
threshold = 1
severity = "critical"

[alert_rates.matrix_sync]
enabled = true
# Alert after 3 sync failures
threshold = 3
window = "10m"
severity = "warning"

[alert_rates.budget]
enabled = true
# Warning at 80%
warning_threshold = 0.8
# Critical at 100%
critical_threshold = 1.0
```

---

## Integration with External Monitoring

### Prometheus Export (Future)

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'armorclaw'
    static_configs:
      - targets: ['localhost:9090']
```

### Grafana Dashboard

Key metrics to display:
- Container health status
- Error rate by category
- Budget usage percentage
- Matrix connection status
- Queue depth (SDTW)

---

## Best Practices

1. **Set up a dedicated Matrix room** for alerts to avoid noise in main channels
2. **Configure rate limiting** to avoid alert fatigue
3. **Document runbooks** for each alert type
4. **Test alerts regularly** to ensure notifications work
5. **Review alert thresholds** periodically and adjust based on operational experience

---

## Alert Frequency Limits

To prevent notification spam:

| Category | Max Alerts | Window |
|----------|------------|--------|
| Critical | 5 | 1 hour |
| Error | 10 | 1 hour |
| Warning | 20 | 1 hour |

After limits are reached, additional alerts are logged but not sent to Matrix until the window resets.

---

**Alert Integration Guide Last Updated:** 2026-02-15
