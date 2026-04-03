---
name: armorclaw_deploy
version: 2.0.0
description: Deploy ArmorClaw to VPS with cross-platform support (Linux, macOS, Windows, PowerShell, Git Bash, WSL)
---

# ArmorClaw Deploy Skill

Deploy ArmorClaw to any VPS with automated OS detection, SSH validation, and service verification.

## Quick Reference

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `vps_ip` | Yes | - | VPS IP address or hostname |
| `ssh_user` | No | `root` | SSH username |
| `ssh_key` | No | `~/.ssh/id_ed25519` | Path to SSH private key |
| `domain` | No | - | Domain for Sentinel/Cloudflare modes |
| `mode` | No | `native` | Deployment mode: native, sentinel, cloudflare |
| `cf_api_token` | No | - | Cloudflare API token (cloudflare mode) |
| `openrouter_api_key` | No | - | OpenRouter API key (recommended) |
| `admin_username` | No | auto | Custom admin username |
| `admin_password` | No | auto | Custom admin password |

## Deployment Modes

| Mode | Use Case | TLS | Setup Time |
|------|---------|-----|------------|
| **native** | Development, local access only | None | ~2 min |
| **sentinel** | Production VPS with public access | Let's Encrypt | ~5 min |
| **cloudflare** | VPS behind NAT/firewall | Cloudflare SSL | ~3 min |

## Usage Examples

### Native Mode (Local Development)

```bash
# Deploy with local access only (no public exposure)
/deploy vps_ip=192.168.1.100
```

### Sentinel Mode (Production with Let's Encrypt)

```bash
# Deploy with automatic HTTPS via Let's Encrypt
/deploy vps_ip=5.183.11.149 domain=armorclaw.example.com mode=sentinel
```

### Cloudflare Tunnel Mode (Behind NAT)

```bash
# Deploy with Cloudflare Tunnel (no port forwarding needed)
/deploy vps_ip=192.168.1.50 domain=armorclaw.example.com mode=cloudflare cf_api_token=your-token
```

### Custom SSH Configuration

```bash
# Deploy with custom SSH key and user
/deploy vps_ip=5.183.11.149 ssh_user=ubuntu ssh_key=~/.ssh/my_key.pem mode=sentinel domain=armorclaw.example.com
```

### From WSL (Windows Subsystem for Linux)

```bash
# Deploy from WSL environment
/deploy vps_ip=5.183.11.149 domain=armorclaw.example.com mode=sentinel
```

## Platform Support

| Platform | Support | Notes |
|---------|---------|-------|
| **Linux** | ✅ Full | Native OpenSSH, curl |
| **macOS** | ✅ Full | Native OpenSSH, curl |
| **Windows (Git Bash)** | ✅ Full | Recommended for Windows |
| **Windows (PowerShell)** | ✅ Full | OpenSSH, Invoke-WebRequest |
| **WSL** | ✅ Full | Full Linux compatibility |

## Automation Levels

| Step | Level | Description |
|------|-------|-------------|
| detect_os | auto | Detect OS and shell environment |
| validate_environment | auto | Validate SSH key and connectivity |
| prepare_cloudflare | guide | Cloudflare setup instructions |
| deploy_installer | confirm | Ask before deploying |
| wait_for_services | auto | Wait for containers to start |
| verify_installation | auto | Verify deployment health |
| get_connection_info | auto | Display connection details |

## Deployment Steps

### 1. detect_os (auto)
Detects user's operating system and shell environment:
- Linux (native)
- macOS (native)
- Windows Git Bash (MSYSTEM/MINGW)
- Windows PowerShell (COMSPEC)
- WSL (WSL_DISTRO_NAME)

### 2. validate_environment (auto)
Validates SSH key and network connectivity:
- Checks SSH key exists at specified path
- Verifies SSH key permissions (600/400)
- Tests SSH connection to VPS
- Provides clear error messages for failures

### 3. prepare_cloudflare (guide)
Guides Cloudflare configuration for cloudflare mode:
- Validates domain is provided
- Provides instructions for creating Cloudflare API token
- Lists required token permissions

### 4. deploy_installer (confirm)
Deploys ArmorClaw installer to VPS:
- Builds SSH command with environment variables
- Passes deployment mode configuration
- Passes admin credentials (if provided)
- Executes remote installer script

### 5. wait_for_services (auto)
Waits for ArmorClaw services to start:
- Polls Docker containers (max 30 retries)
- 2-second intervals between checks
- Reports timeout warnings if services slow to start

### 6. verify_installation (auto)
Verifies ArmorClaw installation:
- Checks ArmorClaw containers are running
- Tests health endpoint (HTTP or HTTPS based on mode)
- Reports any issues found

### 7. get_connection_info (auto)
Displays connection information:
- Public URL (for sentinel/cloudflare modes)
- Bridge URL and Matrix URL (for native mode)
- Commands for checking logs and status
- Next steps for mobile app setup

## Prerequisites

### For All Modes

1. **VPS Requirements**
   - Ubuntu 20.04+ or similar Linux distribution
   - Docker 24.0+ (installed by script if missing)
   - 2GB RAM minimum, 4GB recommended
   - 10GB disk minimum

2. **SSH Access**
   - SSH key-based authentication (recommended)
   - Root or sudo access on VPS

3. **AI Provider API Key** (one of)
   - OpenRouter (recommended): https://openrouter.ai
   - OpenAI: https://platform.openai.com
   - xAI (Grok): https://x.ai
   - Anthropic: https://console.anthropic.com

### For Sentinel Mode

- Domain name pointing to VPS IP
- Ports 80 and 443 accessible from internet

### For Cloudflare Mode

- Domain configured on Cloudflare
- Cloudflare API token with DNS and Tunnel permissions

## Troubleshooting

### SSH Connection Failed

**Error:** `Cannot connect to VPS`

**Solutions:**
1. Verify VPS IP is correct
2. Check SSH key path and permissions: `chmod 600 ~/.ssh/id_ed25519`
3. Verify SSH user has access: `ssh -i ~/.ssh/id_ed25519 user@ip`
4. Check VPS firewall allows SSH (port 22)

### Docker Not Starting

**Error:** `Docker daemon not running`

**Solutions:**
1. SSH into VPS: `ssh root@your-vps-ip`
2. Start Docker: `systemctl start docker`
3. Enable Docker: `systemctl enable docker`
4. Check status: `systemctl status docker`

### Health Endpoint Not Responding

**Error:** `Health endpoint not accessible yet`

**This is normal for first-time setup.** Wait 1-2 minutes and:
1. Check container status: `docker ps`
2. Check logs: `docker logs armorclaw`
3. Test manually: `curl http://localhost:8443/health`

### Cloudflare Mode Issues

**Error:** `Cloudflare mode requires CF_API_TOKEN`

**Solution:**
1. Create token at: https://dash.cloudflare.com/profile/api-tokens
2. Required permissions: Zone DNS Edit, Zone Cloudflare Tunnel Edit
3. Set environment: `export CF_API_TOKEN=your-token`
4. Run deploy again

## After Deployment

### Verify Installation

```bash
# Check container status
ssh root@your-vps-ip 'docker ps'

# Check Bridge health
curl http://your-vps-ip:8443/health

# Check Matrix
curl http://your-vps-ip:6167/_matrix/client/versions
```

### Connect Mobile App

1. Install ArmorChat from Google Play
2. Run provision skill: `/provision vps_ip=your-vps-ip`
3. Scan QR code or enter deep link manually
4. Set up biometrics for keystore access

### Create Your First Agent

Once connected via ArmorChat:

```
!agent create name="Researcher" skills="web_browsing"
```

## Security Notes

- **API keys** are read from environment variables, never persisted to disk
- **Admin credentials** are auto-generated if not provided
- **SSH keys** should use ed25519 algorithm (recommended)
- **Cloudflare tokens** are passed via environment, not logged

## Related Skills

- **[Status](../status/SKILL.md)** - Check deployment health
- **[Cloudflare](../cloudflare/SKILL.md)** - Configure HTTPS
- **[Provision](../provision/SKILL.md)** - Connect mobile devices

## References

- Main documentation: `doc/armorclaw.md`
- Installer script: `deploy/install.sh`
- Health check: `deploy/health-check.sh`
- Platform patterns: `.skills/PLATFORM.md`
