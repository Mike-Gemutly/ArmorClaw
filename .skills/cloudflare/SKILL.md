---
name: cloudflare
description: Use when setting up Cloudflare HTTPS for ArmorClaw deployment with tunnel or proxy mode support
---

# ArmorClaw Cloudflare HTTPS Setup Skill

Automated setup of Cloudflare HTTPS for ArmorClaw with support for both Tunnel and Proxy modes.

## Overview

This skill automates Cloudflare HTTPS configuration for ArmorClaw deployments:
- Network environment detection
- Cloudflare mode recommendation (Tunnel vs Proxy)
- Cloudflare API token validation
- Cloudflare Tunnel or Proxy setup
- HTTPS connectivity verification
- SSL certificate validation

## Cloudflare Modes

| Mode | Use Case | Requirements | Benefits | Setup Time |
|------|---------|--------------|----------|------------|
| **Tunnel** | VPS behind NAT/firewall, no public IP | cloudflared authentication | No port forwarding, outbound-only, works with dynamic IP | ~3 min |
| **Proxy** | Static public IP, existing Cloudflare setup | CF_API_TOKEN, ports 80/443 open | Cloudflare CDN, DDoS protection, caching | ~5 min |

### When to Use Tunnel Mode

Use Cloudflare Tunnel when:
- Your VPS is behind NAT or CGNAT
- You have a dynamic IP address
- You cannot open public ports (80/443)
- You want outbound-only connectivity for security
- Running ArmorClaw on home/office network

### When to Use Proxy Mode

Use Cloudflare Proxy when:
- Your VPS has a static public IP
- You can open ports 80 and 443
- Your domain is already configured on Cloudflare
- You want Cloudflare CDN caching and DDoS protection
- You want Cloudflare's reverse proxy features

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `domain` | string | Yes | - | Domain name for Cloudflare setup (e.g., `armorclaw.example.com`) |
| `mode` | string | No | `tunnel` | Cloudflare mode: `tunnel` or `proxy` |
| `cf_api_token` | string | No | - | Cloudflare API token (required for proxy mode, optional for tunnel) |
| `vps_ip` | string | No | - | VPS IP address for network detection and mode recommendation |
| `ssh_key` | string | No | `~/.ssh/id_ed25519` | Path to SSH private key for VPS network checks |

## Usage Examples

### Cloudflare Tunnel Mode (Recommended for NAT/CGNAT)

Set up Cloudflare Tunnel for outbound-only connectivity:

```
/cloudflare domain=armorclaw.example.com mode=tunnel
```

This sets up:
- Cloudflare Tunnel with cloudflared
- No public ports required on VPS
- Automatic Cloudflare SSL certificates
- DNS records created via Tunnel API

### Cloudflare Tunnel with Network Detection

Let the skill detect your network and recommend the optimal mode:

```
/cloudflare domain=armorclaw.example.com mode=tunnel vps_ip=192.168.1.100
```

The skill will:
- Test if ports 80 and 443 are accessible
- Recommend Tunnel or Proxy mode
- Provide clear explanation of why

### Cloudflare Proxy Mode (Requires CF_API_TOKEN)

Set up Cloudflare Proxy with CDN and DDoS protection:

```
/cloudflare domain=armorclaw.example.com mode=proxy cf_api_token=your-token
```

This configures:
- Cloudflare DNS A records with proxy enabled (orange cloud)
- Cloudflare SSL/TLS settings
- Caddy reverse proxy configuration
- CDN caching and DDoS protection

### Full Configuration with Network Check

Complete setup with network detection and API token:

```
/cloudflare domain=armorclaw.example.com mode=proxy cf_api_token=your-token vps_ip=5.183.11.149
```

## Cross-Platform Support

This skill works on all major platforms with the same command syntax.

### Linux / macOS / Git Bash / WSL

Standard commands work identically:

```
/cloudflare domain=armorclaw.example.com mode=tunnel
```

### Windows (PowerShell)

PowerShell support requires full Windows paths:

```
/cloudflare domain=armorclaw.example.com mode=proxy cf_api_token=your-token
```

**Recommendation:** Install Git Bash and use it instead of PowerShell for consistency.

## Automation Levels

Each step has an automation level that determines how it executes:

| Automation Level | Behavior | Used For |
|------------------|----------|----------|
| `auto` | Executes immediately without asking | Network detection, prerequisite checks, HTTPS verification |
| `confirm` | Asks for user confirmation before executing | Running Cloudflare setup script |
| `guide` | Provides instructions for user to perform manually | Cloudflare account/token setup (not needed for Tunnel mode) |

## Steps

### 1. detect_network (auto)

Detects network environment and recommends optimal Cloudflare mode:

- Detects user's platform (Linux, macOS, Windows, WSL)
- Tests VPS connectivity if VPS IP provided
- Checks if ports 80 and 443 are accessible
- Recommends Tunnel or Proxy mode based on network
- Provides clear explanation of recommendation

**Output Example:**
```
Detected platform: linux
Testing connectivity to VPS: 192.168.1.100
Checking if ports 80 and 443 are accessible...
  ✗ Port 80 is NOT accessible
  ✗ Port 443 is NOT accessible

Network Recommendation: Tunnel Mode
  Ports 80/443 are not both accessible
  Tunnel mode works without public ports
  Requires: cloudflared authentication (cloudflared tunnel login)
```

### 2. validate_prerequisites (auto)

Validates Cloudflare prerequisites and API token:

- Checks domain format is valid
- Validates mode is either `tunnel` or `proxy`
- Verifies Cloudflare API token for proxy mode
- Provides clear error messages if prerequisites not met

**Proxy Mode Token Validation:**
- Sends token verification request to Cloudflare API
- Confirms token has required permissions
- Reports specific permission errors if token is invalid

**Tunnel Mode:**
- No API token required
- Uses cloudflared authentication instead
- Provides guidance if cloudflared not installed

### 3. run_setup (confirm)

Executes Cloudflare setup using `deploy/setup-cloudflare.sh`:

- Checks if setup script exists in `deploy/` directory
- Makes script executable
- Builds command with appropriate flags
- Executes setup script with environment variables
- Passes `CF_API_TOKEN` for proxy mode

**Tunnel Mode Setup:**
- Installs cloudflared if not present
- Prompts for `cloudflared tunnel login` if not authenticated
- Creates or reuses existing tunnel
- Configures DNS records via Tunnel API
- Creates systemd service for cloudflared

**Proxy Mode Setup:**
- Verifies domain uses Cloudflare nameservers
- Detects public IP address
- Verifies ports 80 and 443 are accessible
- Creates DNS A records with proxy enabled (orange cloud)
- Generates Cloudflare origin certificate
- Configures Caddy with origin certificate
- Displays SSL/TLS configuration instructions

### 4. verify_https (auto)

Verifies HTTPS connectivity and SSL certificate:

- Waits for DNS propagation (5 seconds)
- Tests HTTPS connectivity using curl
- Checks SSL certificate details
- Tests ArmorClaw health endpoint
- Provides clear status and next steps

**Output Example:**
```
Verifying HTTPS access for: armorclaw.example.com

Waiting for DNS propagation (5 seconds)...

Testing HTTPS connection...
✓ HTTPS is accessible

Checking SSL certificate...
issuer: C=US; O=Cloudflare, Inc.; CN=Cloudflare Inc ECC CA-3

Testing ArmorClaw health endpoint...
✓ ArmorClaw health endpoint is responding

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Cloudflare HTTPS Setup Complete
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your ArmorClaw deployment should now be accessible at:
  https://armorclaw.example.com
```

## Prerequisites

### For All Modes

1. **Domain Name**
   - You must own a domain (e.g., `armorclaw.example.com`)
   - Domain should be active and accessible
   - DNS records will be configured during setup

2. **ArmorClaw Installation**
   - ArmorClaw should be deployed before Cloudflare setup
   - Use `/armorclaw_deploy` skill first
   - Verify ArmorClaw is running: `docker ps`

### For Tunnel Mode

1. **Cloudflare Account**
   - Sign up at [cloudflare.com](https://cloudflare.com) (free tier works)
   - Add your domain to Cloudflare

2. **Cloudflare Tunnel Access**
   - No API token required for tunnel mode
   - Authentication handled via `cloudflared tunnel login`
   - During setup, you'll be prompted to authenticate

3. **Network Requirements**
   - No port forwarding required
   - Works behind NAT/CGNAT
   - Outbound connectivity to Cloudflare only

### For Proxy Mode

1. **Cloudflare Account**
   - Sign up at [cloudflare.com](https://cloudflare.com)
   - Add your domain to Cloudflare
   - Update nameservers to Cloudflare's nameservers

2. **Cloudflare API Token**
   - Create API token with specific permissions
   - Store securely as environment variable: `export CF_API_TOKEN=your-token`
   - Token is never persisted to disk

3. **API Token Permissions**
   - Visit: https://dash.cloudflare.com/profile/api-tokens
   - Click "Create Token"
   - Use "Custom token" template with:
     - **Zone: DNS - Edit**
     - **Zone: SSL/TLS - Edit**
   - Set zone resources to include your domain
   - Copy token and export: `export CF_API_TOKEN=your-token`

4. **Network Requirements**
   - VPS must have static public IP
   - Ports 80 and 443 must be accessible from internet
   - Firewall rules must allow incoming connections
   - DNS A records must point to VPS IP

### Creating a Cloudflare API Token (Proxy Mode)

1. **Navigate to API Tokens:**
   - Go to Cloudflare Dashboard
   - Click on your profile (top right)
   - Select "My Profile"
   - Click "API Tokens" tab
   - Click "Create Token"

2. **Use Custom Token Template:**
   - Select "Custom token" template
   - Set permissions:
     - **Zone** → DNS → Edit
     - **Zone** → SSL/TLS → Edit
   - Set Account Resources: Include → All accounts (or specific account)
   - Set Zone Resources: Include → Specific zone → your-domain.com
   - Click "Continue to summary"
   - Click "Create Token"

3. **Copy and Store Token:**
   - Copy the token (you won't see it again!)
   - Store securely as environment variable:
     ```bash
     export CF_API_TOKEN=your-actual-api-token-here
     ```
   - Add to your shell profile (`.zshrc`, `.bashrc`) for persistence

4. **Verify Token:**
   ```bash
   curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json"
   ```
   Expected response: `{"success":true,...}`

## Troubleshooting

### Tunnel Mode Issues

**cloudflared Not Installed:**

**Error:** `cloudflared: command not found`

**Solution:** The setup script automatically installs cloudflared. If it fails:

```bash
# Linux/macOS
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o cloudflared
sudo chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# Verify installation
cloudflared --version
```

**Authentication Failed:**

**Error:** `cloudflared tunnel login failed`

**Solution:**
1. Ensure you have a Cloudflare account
2. Ensure your domain is added to Cloudflare
3. Try manual authentication:
   ```bash
   cloudflared tunnel login
   ```
4. Follow browser prompt to authorize

**Tunnel Not Connecting:**

**Error:** `Tunnel shows inactive in Cloudflare Dashboard`

**Solution:**
1. Check cloudflared service status:
   ```bash
   sudo systemctl status cloudflared
   ```
2. View tunnel logs:
   ```bash
   sudo journalctl -u cloudflared -f
   ```
3. Restart cloudflared service:
   ```bash
   sudo systemctl restart cloudflared
   ```
4. Verify ArmorClaw Bridge is running on port 8080:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker ps | grep armorclaw'
   ```

### Proxy Mode Issues

**API Token Invalid:**

**Error:** `Cloudflare API token validation failed`

**Solution:**
1. Verify token is set:
   ```bash
   echo $CF_API_TOKEN
   ```
2. Test token manually:
   ```bash
   curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json"
   ```
3. Check token permissions:
   - Zone: DNS - Edit ✓
   - Zone: SSL/TLS - Edit ✓
   - Zone Resources: Your domain ✓
4. Regenerate token if needed (old token may have expired)

**DNS Not Propagating:**

**Error:** `HTTPS not accessible yet`

**Solution:**
1. Check DNS A record points to correct IP:
   ```bash
   dig +short armorclaw.example.com
   ```
2. Check Cloudflare Dashboard → DNS → Records
3. Verify proxy is enabled (orange cloud icon)
4. Wait 1-2 minutes for DNS propagation

**"522" Connection Timed Out:**

**Error:** Cloudflare returns error 522

**Solution:**
1. Check if Caddy is listening on port 8080:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'netstat -tlnp | grep 8080'
   ```
2. Check firewall rules on VPS:
   ```bash
   sudo ufw status
   sudo ufw allow 8080/tcp
   ```
3. Verify ArmorClaw is running:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker ps'
   ```

**"1016" Origin DNS Error:**

**Error:** Cloudflare returns error 1016

**Solution:**
1. Verify DNS A record points to correct VPS IP
2. Check if ArmorClaw Bridge is running:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker ps | grep armorclaw'
   ```
3. Test local access:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'curl http://localhost:8080/health'
   ```

**SSL Certificate Issues:**

**Error:** SSL certificate errors in browser

**Solution:**
1. Check SSL/TLS mode in Cloudflare Dashboard:
   - Visit: https://dash.cloudflare.com
   - Select your domain
   - Navigate to SSL/TLS → Overview
   - Set to: **Full (strict)**
2. Restart Caddy to apply new configuration:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker restart caddy'
   ```
3. Verify certificate generation:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker logs caddy | grep certificate'
   ```

### HTTPS Verification Issues

**Health Endpoint Not Accessible:**

**Error:** `ArmorClaw health endpoint not accessible`

**This is normal for first-time setup.** ArmorClaw may take 1-2 minutes to fully initialize.

**Solution:**
1. Wait 1-2 minutes and test again:
   ```bash
   curl https://your-domain.com/health
   ```
2. Check if ArmorClaw is running:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker ps'
   ```
3. Check ArmorClaw logs:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker logs -f armorclaw'
   ```

**DNS Propagation Delay:**

**Error:** `HTTPS not yet accessible (may still be propagating)`

**Solution:**
1. Check DNS propagation:
   ```bash
   dig +short your-domain.com
   ```
2. Verify Cloudflare DNS records:
   - Visit: https://dash.cloudflare.com
   - Select your domain
   - Navigate to DNS → Records
   - Verify A record exists and points to correct IP
3. Wait 1-2 minutes for DNS to propagate globally

## Advanced Usage

### Switching Modes

If you need to switch between Tunnel and Proxy mode:

```bash
# From Tunnel to Proxy
/cloudflare domain=armorclaw.example.com mode=proxy cf_api_token=your-token

# From Proxy to Tunnel
/cloudflare domain=armorclaw.example.com mode=tunnel
```

**Note:** Clean up previous configuration before switching modes.

### Multiple Domains

Set up Cloudflare for multiple domains:

```bash
# Production
/cloudflare domain=prod.armorclaw.com mode=proxy cf_api_token=your-token

# Staging
/cloudflare domain=staging.armorclaw.com mode=proxy cf_api_token=your-token

# Development (local only)
/cloudflare domain=dev.armorclaw.com mode=tunnel
```

### Dry Run Mode

Test setup without making changes (supported by underlying script):

```bash
# Test Tunnel mode setup
bash deploy/setup-cloudflare.sh --domain armorclaw.example.com --mode tunnel --dry-run

# Test Proxy mode setup
bash deploy/setup-cloudflare.sh --domain armorclaw.example.com --mode proxy --dry-run
```

## After Cloudflare Setup

### Verify Installation

```bash
# Test HTTPS connectivity
curl https://your-domain.com/health

# Check SSL certificate
curl -vI https://your-domain.com 2>&1 | grep -i "ssl\|certificate"

# For Tunnel mode: Check cloudflared service
ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'sudo systemctl status cloudflared'

# For Proxy mode: Check Caddy service
ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker ps | grep caddy'
```

### Configure Cloudflare SSL/TLS (Proxy Mode)

1. Visit: https://dash.cloudflare.com
2. Select your domain
3. Navigate to SSL/TLS → Overview
4. Set SSL/TLS encryption mode to: **Full (strict)**
5. Restart Caddy to apply:
   ```bash
   ssh -i ~/.ssh/id_ed25519 root@your-vps-ip 'docker restart caddy'
   ```

### Connect Your Phone

1. Install ArmorChat from Google Play
2. Open app and scan QR code displayed during ArmorClaw installation
3. Set up biometrics for secure keystore access
4. Your digital secretary is online

### Create Your First Agent

Once connected via ArmorChat:

```
!agent create name="Researcher" skills="web_browsing"
```

Then ask it to do something:

```
Research the best restaurants in NYC for a birthday dinner
```

## Security Notes

- **Never log or store API tokens:** The CF_API_TOKEN is read from environment variables and never written to disk or logs
- **Token display:** Display token once for confirmation, then clear
- **Source from env:** Token should come from environment variable `CF_API_TOKEN`
- **Tunnel mode security:** Uses outbound-only connection (no public ports exposed)
- **Proxy mode security:** Cloudflare provides DDoS protection and Web Application Firewall

## References

- **Main Documentation:** [ARMORCLAW.md](../../ARMORCLAW.md#cloudflare-setup)
- **Setup Script:** [deploy/setup-cloudflare.sh](../../deploy/setup-cloudflare.sh)
- **Platform Support:** [.skills/PLATFORM.md](../PLATFORM.md)
- **Deploy Skill:** [.skills/deploy/SKILL.md](../deploy/SKILL.md)
- **Cloudflare API Docs:** https://developers.cloudflare.com/api/
- **Cloudflare Tunnel Docs:** https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/
