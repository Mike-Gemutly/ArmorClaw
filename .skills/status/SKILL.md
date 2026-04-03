---
name: armorclaw_status
version: 1.0.0
description: Check ArmorClaw deployment health and status
---

# ArmorClaw Status Skill

Check deployment health on VPS with cross-platform support.

## Quick Reference

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `vps_ip` | Yes | - | VPS IP address or hostname |
| `ssh_user` | No | `root` | SSH username |
| `ssh_key` | No | `~/.ssh/id_ed25519` | Path to SSH private key |
| `domain` | No | - | Domain for SSL checks |
| `verbose` | No | `false` | Enable detailed output |

## Usage

```bash
# Basic health check
/status vps_ip=192.168.1.100

# With custom SSH key
/status vps_ip=192.168.1.100 ssh_key=~/.ssh/custom_key

# With domain for SSL verification
/status vps_ip=192.168.1.100 domain=armorclaw.example.com

# Verbose output
/status vps_ip=192.168.1.100 verbose=true
```

## Health Checks Performed

### 1. Docker Status
- Docker daemon running
- ArmorClaw containers status

### 2. Matrix Stack
- Matrix Conduit responding
- Matrix federation endpoint

### 3. Bridge Stack
- Bridge HTTP health endpoint
- Bridge Unix socket RPC

### 4. Additional Services
- Sygnal push gateway
- Catwalk AI service
- Nginx proxy

### 5. SSL/TLS (if domain provided)
- HTTPS connectivity
- Certificate validity

### 6. Network & Volumes
- Docker networks
- Docker volumes

## Platform Support

| Platform | Support | Notes |
|---------|---------|-------|
| **Linux** | ✅ Full | Native support |
| **macOS** | ✅ Full | Native support |
| **Windows (Git Bash)** | ✅ Full | Recommended |
| **Windows (PowerShell)** | ✅ Full | Alternative |
| **WSL** | ✅ Full | Full support |

## Related Skills

- **[Deploy](../deploy/SKILL.md)** - Deploy ArmorClaw to VPS
- **[Cloudflare](../cloudflare/SKILL.md)** - Configure HTTPS
- **[Provision](../provision/SKILL.md)** - Connect mobile devices

## Reference

- Health check script: `deploy/health-check.sh`
- Documentation: `doc/armorclaw.md`
