# ArmorClaw Security Configuration Guide

> **Last Updated:** 2026-02-08
> **Version:** 1.0.0
> **Status:** Production Ready

This guide explains how to configure and use ArmorClaw's security features to protect your AI agents from unauthorized access, data leakage, and unexpected costs.

---

## Table of Contents

1. [Overview](#overview)
2. [Zero-Trust Matrix Security](#zero-trust-matrix-security)
3. [Budget Guardrails](#budget-guardrails)
4. [PII Data Scrubbing](#pii-data-scrubbing)
5. [Container TTL Management](#container-ttl-management)
6. [Host Hardening](#host-hardening)
7. [Verification](#verification)

---

## Overview

ArmorClaw provides multiple layers of security to protect your AI agents:

| Feature | Protection Level | Default Status |
|---------|-----------------|----------------|
| Zero-Trust Sender Filtering | HIGH | Disabled (opt-in) |
| Budget Guardrails | CRITICAL | Enabled ($5/day, $100/month) |
| PII Scrubbing | MEDIUM | Automatic (17 patterns) |
| Container TTL | MEDIUM | 10 minutes idle |
| Host Hardening | HIGH | Manual (via scripts) |

---

## Zero-Trust Matrix Security

Zero-trust security ensures that only trusted Matrix users and rooms can send commands to your agents. This is **critical for production deployments**.

### Trusted Senders

Restrict agent control to specific Matrix user IDs:

```toml
[matrix.zero_trust]
trusted_senders = [
    "@yourself:example.com",
    "@admin-bot:example.com",
    "*@trusted-domain.com",  # Wildcard: all users from domain
    "*:corporate.com"        # Wildcard: entire homeserver
]
```

**Wildcard Patterns:**
- `@user:domain.com` - Exact user match
- `*@domain.com` - All users from domain
- `*:domain.com` - All users on homeserver

### Trusted Rooms

Restrict agents to only respond in specific rooms:

```toml
[matrix.zero_trust]
trusted_rooms = [
    "!secureRoom:example.com",
    "!adminRoom:example.com"
]
```

### Rejection Messages

Enable helpful rejection messages for untrusted senders:

```toml
[matrix.zero_trust]
reject_untrusted = true  # Send rejection message
```

When enabled, untrusted senders receive:
```
❌ Command rejected: Sender not in trusted list.

Contact the administrator to be added to the trusted senders list.
```

### Configuration via Setup Wizard

The interactive setup wizard guides you through zero-trust configuration:

```bash
sudo ./deploy/setup-wizard.sh
```

During Step 7 (Configuration), you'll be prompted:

```
Enable zero-trust sender/room filtering? [y/N]: y

Enter trusted Matrix user IDs (one per line, empty line to finish):
Format: @user:domain.com, *@trusted.domain.com, or *:domain.com for wildcards

> @alice:example.com
> *@admin-corp.com
> [press Enter]

Enter trusted room IDs (one per line, empty line to finish):
Format: !roomid:domain.com

> !secureOps:example.com
> [press Enter]

Send rejection message to untrusted senders? [Y/n]: y
```

### Security Logging

All authorization events are logged:

```json
{
  "timestamp": "2026-02-08T10:30:45Z",
  "level": "warn",
  "event_type": "access_denied",
  "sender": "@unknown-user:hacker.com",
  "room": "!insecureRoom:example.com",
  "reason": "sender not in trusted list"
}
```

---

## Budget Guardrails

Budget guardrails prevent unexpected API costs by enforcing daily and monthly spending limits.

### Configuration

```toml
[budget]
daily_limit_usd = 5.00
monthly_limit_usd = 100.00
alert_threshold = 80.0     # Warn at 80% of limit
hard_stop = true            # Stop new sessions when exceeded
```

### Budget States

| State | Behavior |
|-------|----------|
| **Normal** | Usage < 80% of limit |
| **Warning** | Usage ≥ 80% of limit (alert logged) |
| **Exceeded** | Usage ≥ 100% of limit (hard-stop if enabled) |

### Per-Model Costs

Configure custom costs for different AI models:

```toml
[budget.provider_costs]
"gpt-4" = 30.00              # $30 per 1M tokens
"gpt-3.5-turbo" = 2.00       # $2 per 1M tokens
"claude-3-opus" = 15.00      # $15 per 1M tokens
"claude-3-sonnet" = 3.00     # $3 per 1M tokens
```

### Budget Confirmation

The setup wizard requires budget confirmation:

```
CRITICAL: Financial Responsibility

ArmorClaw provides token budget tracking, but you MUST set hard limits
in your AI provider dashboard to prevent unexpected charges.

Please verify you have set (or will set) the following:

  1. OpenAI: https://platform.openai.com/settings/limits
  2. Anthropic: https://console.anthropic.com/settings/limits

Set your hard monthly limit to a reasonable amount (e.g., $100)

I have set (or will set) hard limits in my provider dashboard [y/N]: y
```

### Monitoring Budget Usage

Check budget status via RPC:

```bash
echo '{"jsonrpc":"2.0","method":"budget_status","id":1}' | \
  sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

Response:

```json
{
  "jsonrpc": "2.0",
  "result": {
    "daily_usd": 3.45,
    "daily_limit_usd": 5.00,
    "daily_percent": 69.0,
    "monthly_usd": 45.67,
    "monthly_limit_usd": 100.00,
    "monthly_percent": 45.67,
    "status": "warning"
  },
  "id": 1
}
```

---

## PII Data Scrubbing

Personally Identifiable Information (PII) is automatically scrubbed from all agent prompts and responses.

### Default Patterns

ArmorClaw scrubs 17 PII patterns by default:

| Pattern | Example | Scrubbed As |
|---------|---------|-------------|
| Email addresses | `user@example.com` | `████████@█████████` |
| Credit cards | `4111-1111-1111-1111` | `████-████-████-████` |
| SSN | `123-45-6789` | `███-██-████` |
| Phone numbers | `(555) 123-4567` | `(███) ███-████` |
| IP addresses | `192.168.1.1` | `███.███.█.█` |
| API keys | `sk-abc123xyz` | `sk-██████████` |
| Bearer tokens | `Bearer xyz123` | `Bearer ███████` |
| AWS credentials | `AKIAIOSFODNN7EXAMPLE` | `AKIA██████████████` |

### Adding Custom Patterns

Add custom PII patterns via configuration:

```toml
[pii_scrubber]
custom_patterns = [
    "Employee ID: EMP\\d{6}",
    "Project Code: PROJ-[A-Z]{3}-\\d{4}"
]
```

### Testing PII Scrubbing

```python
from bridge.pkg.pii import Scrubber

scrubber = Scrubber()
text = "Contact user@example.com for support"
cleaned = scrubber.scrub(text)
print(cleaned)  # "Contact ████████████████ for support"
```

---

## Container TTL Management

Containers are automatically removed after a period of inactivity to prevent resource leaks.

### Configuration

```toml
[ttl]
idle_timeout = "10m"        # 10 minutes of inactivity
check_interval = "1m"       # Check every minute
```

### Heartbeat Mechanism

Agents must send heartbeat messages to remain active:

```python
# Agent sends heartbeat every 30 seconds
while running:
    bridge.send_heartbeat()
    time.sleep(30)
```

### TTL States

| State | Description | Action |
|-------|-------------|--------|
| **Active** | Container running + heartbeating | None |
| **Idle** | No heartbeat for > `idle_timeout` | Warning logged |
| **Expired** | Idle for > 2 × `idle_timeout` | Container removed |

### Monitoring TTL Status

```bash
echo '{"jsonrpc":"2.0","method":"ttl_status","id":1}' | \
  sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

Response:

```json
{
  "jsonrpc": "2.0",
  "result": {
    "active_containers": 3,
    "idle_containers": 1,
    "containers": [
      {
        "id": "abc123",
        "session_id": "session-456",
        "idle_time": "5m23s",
        "idle_timeout": "10m0s",
        "status": "active"
      }
    ]
  },
  "id": 1
}
```

---

## Host Hardening

ArmorClaw provides automated host hardening scripts for production deployments.

### Firewall Configuration

```bash
sudo ./deploy/setup-firewall.sh
```

**Features:**
- UFW deny-all default policy
- Tailscale VPN auto-detection (whitelisted if present)
- SSH (port 22) rate-limited
- Matrix ports (8448, 443) allowed
- Docker network trusted

### SSH Hardening

```bash
sudo ./deploy/harden-ssh.sh
```

**Features:**
- Root login disabled (`PermitRootLogin no`)
- Password authentication disabled (`PasswordAuthentication no`)
- Key-only authentication required
- Empty passwords prohibited

### VPS Deployment Integration

Host hardening is automatically included in VPS deployment (Step 7):

```bash
sudo ./deploy/vps-deploy.sh
```

---

## Verification

### Configuration Validation

Validate your security configuration:

```bash
sudo /opt/armorclaw/armorclaw-bridge validate /etc/armorclaw/config.toml
```

### Test Zero-Trust Filtering

```bash
# From an untrusted Matrix account:
# Send: /status

# Expected response (if reject_untrusted = true):
# ❌ Command rejected: Sender not in trusted list.
```

### Test Budget Enforcement

```bash
# Set low budget for testing
echo '{"jsonrpc":"2.0","method":"set_budget","params":{"daily_limit_usd":0.01},"id":1}' | \
  sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Try to start agent - should fail with budget error
```

### Test PII Scrubbing

```bash
# Send prompt with PII via Matrix
# "Send email to user@example.com with SSN 123-45-6789"

# Check agent logs - PII should be scrubbed
# "Send email to ████████████████ with SSN ███-██-████"
```

---

## Security Checklist

Before deploying to production:

- [ ] Zero-trust senders configured (if using Matrix control)
- [ ] Budget limits set AND verified in provider dashboard
- [ ] Hard-stop enabled in budget configuration
- [ ] Container TTL configured for your use case
- [ ] Firewall rules applied (`setup-firewall.sh`)
- [ ] SSH hardened (`harden-ssh.sh`)
- [ ] Configuration validated (`armorclaw-bridge validate`)
- [ ] Test agent started successfully
- [ ] PII scrubbing verified with test data
- [ ] Budget tracking verified with low limits

---

## Troubleshooting

### "Sender not in trusted list"

**Problem:** Commands from your own Matrix account are rejected.

**Solution:**
1. Check your exact Matrix user ID: Element X → Settings → Account
2. Verify the format in `config.toml` matches exactly
3. Check for trailing whitespace in user IDs
4. Restart bridge after config change

### "Budget exceeded" errors

**Problem:** Agents won't start despite having budget available.

**Solution:**
1. Check both daily AND monthly limits
2. Verify `hard_stop` isn't blocking new sessions
3. Check budget status: `armorclaw-bridge budget_status`
4. Reset tracking if needed: `armorclaw-bridge budget_reset`

### "Container idle timeout"

**Problem:** Containers are being removed too quickly.

**Solution:**
1. Increase `idle_timeout` in `[ttl]` section
2. Ensure agent is sending heartbeats every 30-60 seconds
3. Check for network issues delaying heartbeats
4. Monitor TTL status: `armorclaw-bridge ttl_status`

---

## Best Practices

1. **Always** set hard limits in your AI provider dashboard (never rely solely on ArmorClaw)
2. **Start** with restrictive zero-trust policies, then expand as needed
3. **Test** PII scrubbing with real data from your use case
4. **Monitor** budget usage daily for the first week
5. **Review** security logs regularly: `journalctl -u armorclaw-bridge -f`
6. **Use** SSH keys only, never password authentication
7. **Keep** container TTL short for untrusted environments
8. **Backup** your keystore database regularly

---

## Additional Resources

- [Configuration Guide](configuration.md) - Complete configuration reference
- [Setup Guide](setup-guide.md) - Installation and setup instructions
- [Error Catalog](error-catalog.md) - Troubleshooting error messages
- [RPC API Reference](../reference/rpc-api.md) - All RPC methods documented

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-08
**Status:** Production Ready ✅
