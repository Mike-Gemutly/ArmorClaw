# Element X Quick Start Guide

> **Last Updated:** 2026-02-07
> **Time to Complete:** 5-10 minutes
> **Goal:** Connect to ArmorClaw agent via Element X app

---

## Overview

This guide will help you connect to your ArmorClaw agent using the Element X app. You'll be able to:
- Send commands to your agent
- Configure agents remotely
- Receive responses securely through encrypted Matrix protocol

---

## Prerequisites Checklist

Before starting, ensure you have:

- [ ] ArmorClaw Bridge running with Matrix enabled
- [ ] Matrix homeserver accessible (local or remote)
- [ ] Element X app installed on your device
  - **iOS:** [App Store](https://apps.apple.com/app/id1436723410)
  - **Android:** [Play Store](https://play.google.com/store/apps/details?id=im.vector.app)
  - **Desktop:** [Element X Downloads](https://element.io/download)

---

## Method 1: Docker Compose Stack (Easiest) ⭐

This is the fastest way to get started - everything is deployed automatically.

### Step 1: Deploy the Stack

```bash
cd armorclaw
./deploy/launch-element-x.sh
```

The script will:
- ✅ Deploy Matrix Conduit (homeserver)
- ✅ Deploy Caddy (SSL/proxy)
- ✅ Deploy ArmorClaw Bridge
- ✅ Create admin user and room
- ✅ Display connection details

### Step 2: Note Your Connection Details

When deployment completes, you'll see:

```
╔══════════════════════════════════════════════════════╗
║           Deployment Complete!                         ║
╚══════════════════════════════════════════════════════╝

Element X Connection Details:

  Homeserver URL:
    https://your-domain.com

  Username:
    admin

  Password:
    your-generated-password

  Room to Join:
    #agents:your-domain.com
    ArmorClaw Agents
```

**⚠️ IMPORTANT:** Save these credentials! You'll need them for Element X.

### Step 3: Connect with Element X

1. **Open Element X** on your device
2. **Tap "Login"** or "Sign in"
3. **Enter homeserver URL:**
   ```
   https://your-domain.com
   ```
   (Or just enter `your-domain.com` - Element X will add `https://`)

4. **Enter your credentials:**
   - Username: `admin` (or whatever username was created)
   - Password: (the password displayed during deployment)

5. **Tap "Log in"**

6. **Join the agent room:**
   - Tap the `+` button or "Join room"
   - Enter: `#agents:your-domain.com`
   - Tap "Join"

7. **Verify connection:**
   ```
   /ping
   ```

   You should see:
   ```
   ✅ Pong! Agent is running and connected.
   ```

### Troubleshooting Method 1

**"Connection failed" error:**
- Wait 1-2 minutes for SSL certificates to provision
- Check homeserver URL is correct (includes `https://`)
- Verify deployment is running: `docker-compose ps`

**"Room not found" error:**
- Verify room alias is correct: `#agents:your-domain.com`
- Check room was created during provisioning
- Try joining by room ID if alias doesn't work

---

## Method 2: Local Development Setup

For local development or testing without a domain.

### Step 1: Start Local Matrix

```bash
cd armorclaw
./deploy/start-local-matrix.sh
```

This starts:
- Matrix Conduit on `http://localhost:6167`
- Caddy on `http://localhost`
- Bridge with Matrix enabled

### Step 2: Configure /etc/hosts (macOS/Linux)

```bash
sudo nano /etc/hosts
```

Add:
```
127.0.0.1 matrix.local
```

Save and exit.

### Step 3: Create Matrix User

```bash
# Register admin user
curl -X POST http://localhost:6167/_matrix/client/v3/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password",
    "auth": {"type": "m.login.dummy"}
  }'
```

### Step 4: Create Agent Room

```bash
# Create room
curl -X POST http://localhost:6167/_matrix/client/v3/createRoom \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ArmorClaw Agents",
    "room_alias_name": "agents"
  }'
```

### Step 5: Connect Element X

1. Open Element X
2. Login with homeserver: `http://matrix.local` (or `https://matrix.local` for SSL)
3. Use credentials from Step 3
4. Join room: `#agents:matrix.local`

---

## Method 3: Production Infrastructure

For production deployment with your own domain.

### Step 1: Deploy Infrastructure

See [Infrastructure Deployment Guide](2026-02-05-infrastructure-deployment-guide.md) for complete steps.

Quick version:
```bash
# Clone on your server
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Deploy Matrix, Nginx, Coturn
./scripts/deploy-infrastructure.sh

# Start ArmorClaw Bridge
./deploy/setup-wizard.sh
```

### Step 2: Configure DNS

Ensure your domain A record points to your server:
```
matrix.yourdomain.com.  A  your-server-ip
```

Verify: `dig +short matrix.yourdomain.com`

### Step 3: Configure Bridge Matrix Settings

Edit `~/.armorclaw/config.toml`:

```toml
[matrix]
enabled = true
homeserver_url = "https://matrix.yourdomain.com"
username = "bridge-bot"
# Don't set password here - use RPC login
```

### Step 4: Start Bridge

```bash
sudo systemctl start armorclaw-bridge
sudo systemctl status armorclaw-bridge
```

### Step 5: Connect Element X

Use same steps as Method 1, with your domain:
- Homeserver: `https://matrix.yourdomain.com`
- Username: Your created admin user
- Password: Your password

---

## Sending Your First Commands

Once connected, try these commands:

### Basic Commands

```bash
# Check agent status
/status

# Get help
/help

# Ping the agent
/ping
```

### Configuration Commands

```bash
# Attach a simple environment config
/attach_config config.env MODEL=gpt-4

# Attach multiple values
/attach_config settings.env TEMPERATURE=0.7 MAX_TOKENS=4096

# Attach TOML config
/attach_config agent.toml [agent]
model=gpt-4
temperature=0.7
```

### Agent Control

```bash
# Start an agent
/start --provider openai --key openai-default

# Stop an agent
/stop --name my-agent

# List running agents
/agents
```

---

## Understanding Responses

### Successful Response Example

```json
{
  "status": "success",
  "config_id": "config-agent.env-1234567890",
  "name": "agent.env",
  "path": "/run/armorclaw/configs/agent.env",
  "size": 25,
  "type": "env"
}
```

### Error Response Example

```json
{
  "error": "Bridge not connected. Cannot attach config.",
  "code": "BRIDGE_OFFLINE"
}
```

---

## Common Issues

### Issue: "Connection timeout"

**Cause:** Homeserver URL incorrect or server not accessible

**Solution:**
1. Verify homeserver URL in browser first
2. Check firewall allows ports 80, 443, 8448
3. For local: Check `docker ps` shows containers running

### Issue: "Invalid username or password"

**Cause:** Wrong credentials or user not created

**Solution:**
1. Verify credentials from deployment output
2. Check user was created: View logs `docker-compose logs -f matrix`
3. For Docker Compose: Check provision script ran successfully

### Issue: "Room not found"

**Cause:** Room alias incorrect or room not created

**Solution:**
1. Verify room alias: `#agents:your-domain.com`
2. Check room was created during provisioning
3. Try joining by room ID instead

### Issue: "Agent not responding"

**Cause:** Agent not started or bridge not connected

**Solution:**
```bash
# Check bridge status
sudo systemctl status armorclaw-bridge

# Check agent containers
docker ps | grep armorclaw

# Start an agent
sudo ./build/armorclaw-bridge start --key openai-default
```

---

## Security Best Practices

### For Local Development
- Use strong passwords even for local setup
- Don't commit `.env` files
- Stop containers when not in use

### For Production
- ✅ Use HTTPS (SSL certificates)
- ✅ Create separate admin user (don't use "admin")
- ✅ Use strong, unique passwords
- ✅ Enable fail2ban on Matrix server
- ✅ Regular security updates
- ✅ Monitor logs for suspicious activity

---

## Next Steps

After successful connection:

1. **Configure your agent** - Send first config via `/attach_config`
2. **Test agent capabilities** - Send `/status` to verify
3. **Explore commands** - Try `/help` for all available commands
4. **Review security** - Read [Security Best Practices](../plans/2026-02-05-security-hardening.md)
5. **Set up monitoring** - Configure alerts for agent health

---

## Additional Resources

- **Element X App:** [element.io](https://element.io/download)
- **Matrix Protocol:** [matrix.org](https://matrix.org/)
- **Configuration Guide:** [configuration.md](configuration.md)
- **Troubleshooting:** [troubleshooting.md](troubleshooting.md)
- **Error Catalog:** [error-catalog.md](error-catalog.md)

---

**Quick Start Guide Last Updated:** 2026-02-07
**ArmorClaw Version:** 1.0.0
