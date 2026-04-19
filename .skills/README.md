# ArmorClaw Deployment Skills

Quick reference for AI CLI deployment skills to help you deploy and manage ArmorClaw.

## Quick Reference

| Skill Name | Purpose | Commands |
|-----------|---------|----------|
| **Deploy** | Deploy ArmorClaw to VPS | `/deploy vps_ip=... mode=...` |
| **Status** | Check deployment health | `/status vps_ip=...` |
| **Cloudflare** | Setup HTTPS with Cloudflare | `/cloudflare domain=... mode=tunnel\|proxy` |
| **Provision** | Connect mobile devices | `/provision vps_ip=... expiry=300` |
| **Ops** | Full lifecycle management | `/ops action=health\|redeploy\|logs\|backup\|restore vps_ip=...` |

## Skills

### [Deploy](./deploy/SKILL.md)
Automated VPS deployment with Native, Sentinel, and Cloudflare modes. Supports Linux, macOS, Windows (PowerShell, Git Bash, WSL).

**Key Features:**
- OS and platform detection
- SSH connection validation
- Service health verification
- Connection info retrieval

### [Status](./status/SKILL.md)
Check deployment health with cross-platform support. Verifies containers, services, SSL/TLS, and overall system health.

**Key Features:**
- OS and platform detection
- SSH connection validation
- Container health checks
- Service availability verification
- SSL/TLS certificate validation

### [Cloudflare](./cloudflare/SKILL.md)
Cloudflare HTTPS setup with Tunnel and Proxy modes. Handles network detection and SSL configuration.

**Key Features:**
- Network environment detection
- Cloudflare Tunnel setup (no port forwarding)
- Cloudflare Proxy setup (CDN/DDoS)
- HTTPS connectivity verification

### [Provision](./provision/SKILL.md)
Generate QR codes and deep links for secure ArmorChat/ArmorTerminal mobile app connection.

**Key Features:**
- QR code generation with `qrencode`
- Manual entry alternative
- Configurable token expiry (max: 3600s)
- URL-only display mode for scripts

### [Ops](./ops/SKILL.md)
Consolidated lifecycle management — deploy, health monitoring, remediation, logs, backup, and restore. Single entry point for all ArmorClaw operations.

**Key Features:**
- Six action paths: deploy, health, redeploy, logs, backup, restore
- Dependency-aware restart ordering
- Topology auto-detection
- Progressive disclosure (simple default, verbose for details)

## Platform Support

| Platform | Support | Notes |
|---------|---------|-------|
| **Linux** | ✅ Full | Native support |
| **macOS** | ✅ Full | Native support |
| **Windows (Git Bash)** | ✅ Full | Recommended for Windows |
| **Windows (PowerShell)** | ✅ Full | Alternative to Git Bash |
| **WSL** | ✅ Full | Full support |

## Additional Documentation

- **[Platform Guide](./PLATFORM.md)** - Cross-platform patterns and examples
- **[Skill Template](./TEMPLATE.yaml)** - Reusable skill schema template

## Usage in AI CLI Tools

Skills are discovered automatically by AI CLI tools from the `.skills/` directory. Invoke skills by name:

```
/deploy vps_ip=192.168.1.100 mode=sentinel domain=armorclaw.example.com
/status vps_ip=192.168.1.100
/cloudflare domain=armorclaw.example.com mode=tunnel cf_api_token=...
/provision vps_ip=192.168.1.100 expiry=600
/ops action=health vps_ip=192.168.1.100
/ops action=redeploy vps_ip=192.168.1.100 service=bridge
```

## Deployment Modes

| Mode | Use Case | TLS | Setup Time |
|------|---------|-----|------------|
| **native** | Development, local | None | ~2 min |
| **sentinel** | Production VPS | Let's Encrypt | ~5 min |
| **cloudflare-tunnel** | VPS behind NAT | Cloudflare SSL | ~3 min |
| **cloudflare-proxy** | Static IP + CDN | Cloudflare SSL | ~5 min |

---
**ArmorClaw v4.7.0** - [Project Home](../README.md)
