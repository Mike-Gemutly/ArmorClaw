# Jetski - ArmorClaw Management Workspace

## Purpose

This workspace manages the ArmorClaw secure communication system deployed on a remote VPS. It contains no local source code - all operations target the remote deployment via SSH tunneling and JSON-RPC interfaces.

## Architecture Overview

ArmorClaw consists of:
- **Bridge**: JSON-RPC 2.0 interface (SQLCipher + Matrix adapters)
- **ArmorChat**: Android mobile client with hardware-backed keystore
- **Governor-Shield**: Metadata scrubbing layer
- **Deployment**: Docker Compose on remote VPS

## Environment Setup

Required environment variables (defined in `.env` on the host machine):
- `VPS_IP` - Remote server IP address
- `CONNECT_VPS` - SSH connection command
- `AI_API_KEY` - API key for AI provider
- `AI_PROVIDER` - AI provider identifier

## Agent Roles

This workspace uses specialized OpenCode agents for different operational contexts:

### Infrastructure Management
- **sentinel-ops** - Docker Compose deployment, environment handling, service diagnostics
- **claw-ssh** - SSH tunnel establishment, RPC bridge access via localhost:4096

### Testing & Validation
- **armor-test-pilot** - RPC health checks, privacy audits, code analysis
- **chat-adb** - Android app debugging via ADB, Matrix sync validation

## Key Operational Patterns

### SSH Tunnel Setup
```bash
ssh -L 4096:127.0.0.1:4096 -o IdentityAgent=none -i ~/.ssh/openclaw_win root@${VPS_IP}
```

### Bridge Health Check
```bash
curl -X POST http://127.0.0.1:4096/api -d '{"jsonrpc":"2.0","method":"bridge.status","id":1}'
```

### Deployment Protocol
1. Export `AI_API_KEY` and `AI_PROVIDER` in shell
2. Run `docker-compose up -d` to enforce topology
3. Verify `INFO RPC transport` is `tcp`, not `unix`
4. Confirm `ARMORCLAW_CONTAINER_MODE=false` for tunnel stability

## Common Issues

### "Empty Reply" from Bridge
- **Cause**: Bridge bound to Unix socket instead of TCP
- **Fix**: Check `ARMORCLAW_CONTAINER_MODE` and restart with TCP transport

### SSH Connection Errors (curl 52/35)
- **Cause**: Identity firewall or tunnel not established
- **Fix**: Verify SSH tunnel is active, check identity key path

### Android Sync Failures
- **Check**: `adb logcat` for `M_BAD_JSON` or `M_INVALID_USERNAME`
- **Verify**: `SyncStatusBar` transitions to green when VPS bridge is active

## Agent Invocation

Invoke agents based on operational context:
- Deploy/infrastructure changes → `@sentinel-ops`
- SSH tunnel issues → `@claw-ssh`
- RPC testing/audits → `@armor-test-pilot`
- Android app debugging → `@chat-adb`
