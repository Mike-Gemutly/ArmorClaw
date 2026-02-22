# First VPS Deployment Checklist

> **Last Updated:** 2026-02-21
> **Version:** 0.2.0
> **Purpose:** Complete deployment guide for ArmorClaw with OpenClaw, ArmorChat, ArmorTerminal, and Element X
> **Security:** v0.2.0 includes secure provisioning and admin creation

---

## Quick Start Options

### Option A: Docker Image (FASTEST - 2 minutes)

If you want the quickest setup with no dependencies:

```bash
# Pull and run - setup wizard launches automatically
docker pull mikegemut/armorclaw:latest

docker run -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v armorclaw-config:/etc/armorclaw \
  -v armorclaw-data:/var/lib/armorclaw \
  -p 8443:8443 -p 5000:5000 -p 6167:6167 \
  mikegemut/armorclaw:latest
```

**See:** [Docker Quick Start Guide](quickstart-docker.md) for full details.

**Continue to:** Phase 7 (Client Integration) after Docker setup completes.

---

### Option B: Manual Deployment (Full Control)

Follow the phases below for complete control over deployment.

---

## Pre-Deployment Requirements

### VPS Requirements
- [ ] **OS:** Ubuntu 22.04+ or Debian 12+
- [ ] **RAM:** 2GB+ (4GB recommended)
- [ ] **Disk:** 10GB+ free space
- [ ] **CPU:** 2+ cores
- [ ] **Network:** Public IP with open ports 80, 443, 8448, 3478, 5349

### DNS Configuration
- [ ] A record: `matrix.yourdomain.com` → VPS IP
- [ ] A record: `bridge.yourdomain.com` → VPS IP (optional, for HTTPS RPC)
- [ ] SRV record (federation): `_matrix._tcp.yourdomain.com` → `matrix.yourdomain.com:8448`

### Local Prerequisites
- [ ] SSH access to VPS with sudo privileges
- [ ] `socat` installed locally for testing: `sudo apt install socat`
- [ ] Domain with DNS control

---

## Phase 1: VPS Initial Setup

### Step 1.1: Connect and Update
```bash
# Connect to VPS
ssh root@your-vps-ip

# Update system
apt update && apt upgrade -y

# Install prerequisites
apt install -y curl wget git docker.io docker-compose-plugin socat jq unzip
```

### Step 1.2: Configure Firewall
```bash
# Allow essential ports
ufw allow 22/tcp      # SSH
ufw allow 80/tcp      # HTTP (Let's Encrypt)
ufw allow 443/tcp     # HTTPS
ufw allow 8448/tcp    # Matrix Federation
ufw allow 3478/tcp    # STUN
ufw allow 3478/udp    # STUN
ufw allow 5349/tcp    # TURN TLS
ufw allow 5349/udp    # TURN TLS
ufw allow 49152:65535/udp  # TURN relay ports

# Enable firewall
ufw enable
```

### Step 1.3: Clone Repository
```bash
cd /opt
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
```

---

## Phase 2: Build Bridge Binary

### Step 2.1: Install Go
```bash
# Install Go 1.24+
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### Step 2.2: Build Bridge
```bash
cd /opt/armorclaw/bridge
go build -o armorclaw-bridge ./cmd/bridge

# Verify build
./armorclaw-bridge --version
# Expected: ArmorClaw Bridge v7.0.0

# Install to system location
mkdir -p /opt/armorclaw
cp armorclaw-bridge /opt/armorclaw/
chmod +x /opt/armorclaw/armorclaw-bridge
ln -sf /opt/armorclaw/armorclaw-bridge /usr/local/bin/armorclaw-bridge
```

---

## Phase 3: Matrix Stack Deployment

### Step 3.1: Create Configuration Directory
```bash
mkdir -p /etc/armorclaw
mkdir -p /var/lib/armorclaw
mkdir -p /run/armorclaw
mkdir -p /var/log/armorclaw
```

### Step 3.2: Configure Matrix (Conduit)
```bash
# Create conduit config
cat > /opt/armorclaw/configs/conduit.toml << 'EOF'
[global]
server_name = "matrix.yourdomain.com"
address = "0.0.0.0"
port = 6167

[global.database]
path = "/var/lib/matrix-conduit/conduit.db"
backend = "sqlite"

[global.security]
allow_encryption = true
allow_registration = false  # Enable for initial setup, then disable
allow_federation = true

[global.logging]
level = "info"
EOF
```

### Step 3.3: Start Matrix Stack
```bash
cd /opt/armorclaw
docker compose -f docker-compose.matrix.yml up -d

# Wait for services to start
sleep 15

# Verify Matrix is running
curl -f http://localhost:6167/_matrix/client/versions
# Expected: {"versions":["v1.0","v1.1","v1.2","v1.3","v1.4","v1.5","v1.6","v1.7","v1.8","v1.9","v1.10","v1.11"]}
```

### Step 3.4: Create Admin User (SECURE METHOD)

**Do NOT enable registration!** Use the secure admin creation script instead:

```bash
# Use the secure admin creation script (no registration window)
cd /opt/armorclaw
chmod +x deploy/create-matrix-admin.sh
./deploy/create-matrix-admin.sh admin

# Or specify password directly (for automation):
# ./deploy/create-matrix-admin.sh admin "your-secure-password"
```

**Why this matters:** Enabling `allow_registration` creates a window where anyone can register an account on your server. The script creates users via the admin API instead, keeping registration disabled at all times.

**If conduit-admin CLI is not available:**
```bash
# Check if your Conduit image has the CLI
docker exec armorclaw-conduit which conduit-admin

# If not available, you have two options:
# Option 1: Use Conduit with shared secret (recommended)
# Add to configs/conduit.toml:
#   [global.matrix]
#   registration_shared_secret = "your-random-secret"
# Then use: curl -X POST ... with shared_secret

# Option 2: Temporary registration (LESS SECURE)
# Only use if Option 1 doesn't work, and IMMEDIATELY disable after
sed -i 's/allow_registration = false/allow_registration = true/' configs/conduit.toml
docker compose -f docker-compose.matrix.yml restart
# Register via Element X
# IMMEDIATELY:
sed -i 's/allow_registration = true/allow_registration = false/' configs/conduit.toml
docker compose -f docker-compose.matrix.yml restart
```

---

## Phase 4: ArmorClaw Bridge Configuration

### Step 4.1: Run Setup Wizard
```bash
cd /opt/armorclaw
chmod +x deploy/setup-wizard.sh
./deploy/setup-wizard.sh
```

**Follow these choices in the wizard:**

1. **Step 1:** Welcome → Choose "No" for import (fresh install)
2. **Step 2:** Prerequisites → Should pass automatically
3. **Step 3:** Docker → Already installed, verify running
4. **Step 4:** Container → Build from Dockerfile
5. **Step 5:** Bridge → Already built, install to /opt/armorclaw
6. **Step 6:** Budget → Set hard limits in provider dashboard first!
7. **Step 7:** Configuration:
   - Socket path: `/run/armorclaw/bridge.sock`
   - Log level: `info`
   - Log format: `json` (production) or `text` (debugging)
   - Enable Matrix: `yes`
   - Matrix URL: `http://localhost:6167`
   - Matrix username: `bridge`
   - Matrix password: (secure password)
8. **Step 8:** Keystore → Initialize new keystore
9. **Step 9:** API Key → Add your first API key (OpenAI/Anthropic/etc.)
10. **Step 10:** Systemd → Create service file
11. **Step 11:** Verification → Should pass all checks
12. **Step 12:** Advanced Features:
    - WebRTC Voice: `yes` (if using voice)
    - Notifications: `yes` (recommended)
    - Event Bus: `yes` (for ArmorTerminal)
    - Host Hardening: `yes` (recommended)
13. **Step 13:** Start Agent → Optional, can start later

### Step 4.2: Verify Bridge Configuration
```bash
# Check config file
cat /etc/armorclaw/config.toml

# Start bridge
systemctl start armorclaw-bridge
systemctl status armorclaw-bridge

# Test bridge RPC
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.health"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

**Expected response:**
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "version":"7.0.0",
    "supports_e2ee":true,
    "supports_recovery":true,
    "supports_agents":true,
    "supports_workflows":true,
    "status":"healthy"
  }
}
```

---

## Phase 5: Sygnal Push Gateway

### Step 5.1: Configure Sygnal
```bash
# Create sygnal config
cat > /opt/armorclaw/configs/sygnal.yaml << 'EOF'
## Sygnal Push Gateway Configuration

matrix:
  homeserver: "matrix.yourdomain.com"
  url: "http://matrix-conduit:6167"

db:
  name: /app/data/sygnal.db

http:
  bind: "0.0.0.0"
  port: 5000

logging:
  version: 1
  formatters:
    precise:
      format: '%(asctime)s - %(name)s - %(lineno)d - %(levelname)s - %(message)s'
  handlers:
    console:
      class: logging.StreamHandler
      formatter: precise
  loggers:
    sygnal:
      handlers: [console]
      level: INFO
    sygnal.access_token:
      level: INFO
EOF
```

### Step 5.2: Start Sygnal
```bash
cd /opt/armorclaw
docker compose -f docker-compose.bridge.yml up -d sygnal

# Verify Sygnal
curl -f http://localhost:5000/_matrix/push/v1/notify
# Expected: 400 Bad Request (normal, needs body) or 200
```

---

## Phase 6: OpenClaw Agent Container

### Step 6.1: Build OpenClaw Container
```bash
cd /opt/armorclaw
docker build -t armorclaw/agent:v1 .
```

### Step 6.2: Verify Container
```bash
# Test container runs
docker run --rm armorclaw/agent:v1 --version
# Expected: ArmorClaw v1.0.0

# Test container hardening
docker run --rm armorclaw/agent:v1 id
# Expected: uid=10001(claw) gid=10001(claw)
```

---

## Phase 7: Client Integration

### 7.1: Element X Configuration

**Installation:**
1. Download Element X: https://element.io/download
2. Install on your device (desktop/mobile)

**Configuration:**
1. Open Element X
2. Click "Edit" next to homeserver
3. Enter: `https://matrix.yourdomain.com`
4. Create account or sign in

**Verify E2EE:**
1. Start a DM with the bridge bot: `@bridge:matrix.yourdomain.com`
2. Send a message: `!status`
3. Verify encrypted message indicator (🔒)

### 7.2: ArmorChat Configuration

**Prerequisites:**
- Android device with Google Play Services (for FCM)
- ArmorChat APK installed

**Setup Steps:**
1. Open ArmorChat
2. Enter homeserver: `https://matrix.yourdomain.com`
3. Create account or sign in
4. Navigate to Settings → Bridge
5. Enter bridge URL: `https://bridge.yourdomain.com` (or local IP)
6. Verify bridge connection via `bridge.health` check

**Verify Push Notifications:**
```bash
# Check push registration
echo '{"jsonrpc":"2.0","id":1,"method":"push.register_token","params":{"token":"test","platform":"fcm"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### 7.3: ArmorTerminal Configuration

**Prerequisites:**
- ArmorTerminal APK installed

**Setup Steps:**
1. Open ArmorTerminal
2. Configure bridge:
   - RPC URL: `https://bridge.yourdomain.com/rpc`
   - WebSocket URL: `wss://bridge.yourdomain.com/ws`
3. Authenticate with Matrix credentials
4. Verify capabilities via `bridge.health` check

**Test Agent Lifecycle:**
```bash
# Start agent
echo '{"jsonrpc":"2.0","id":1,"method":"agent.start","params":{"name":"Test","type":"assistant","capabilities":["chat"]}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Check agent status
echo '{"jsonrpc":"2.0","id":1,"method":"agent.list"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Phase 8: Verification Checklist

### Bridge Verification
```bash
# Run health check script
./deploy/health-check.sh

# Manual verification
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.health"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

### Matrix Verification
```bash
# Client API
curl -f http://localhost:6167/_matrix/client/versions

# Federation API
curl -f http://localhost:8448/_matrix/federation/v1/version

# Via Nginx proxy
curl -f http://localhost/_matrix/client/versions
```

### Push Gateway Verification
```bash
curl -f http://localhost:5000/_matrix/push/v1/notify
```

### Docker Containers
```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

**Expected running containers:**
- `armorclaw-conduit` (healthy)
- `armorclaw-nginx` (healthy)
- `armorclaw-coturn` (running)
- `armorclaw-sygnal` (healthy)

---

## Phase 9: Post-Deployment Security

### Step 9.1: Verify Registration Disabled
```bash
# Verify registration is disabled (should be false)
grep 'allow_registration' /opt/armorclaw/configs/conduit.toml
# Expected: allow_registration = false
```

### Step 9.2: Enable HTTPS (Recommended)
```bash
# Install Certbot
apt install -y certbot python3-certbot-nginx

# Get certificate
certbot --nginx -d matrix.yourdomain.com
certbot --nginx -d bridge.yourdomain.com

# Auto-renewal
systemctl enable certbot.timer
```

### Step 9.3: Enable Bridge Service
```bash
systemctl enable armorclaw-bridge
```

---

## Troubleshooting

### Bridge Not Starting
```bash
# Check logs
journalctl -u armorclaw-bridge -n 50

# Check socket permissions
ls -la /run/armorclaw/bridge.sock
# Should be: srw------- or srw-rw----

# Check config
cat /etc/armorclaw/config.toml
```

### Matrix Not Responding
```bash
# Check container logs
docker logs armorclaw-conduit

# Check port binding
netstat -tlnp | grep 6167

# Test internal connection
docker exec armorclaw-conduit curl localhost:6167/_matrix/client/versions
```

### Push Notifications Not Working
```bash
# Check Sygnal logs
docker logs armorclaw-sygnal

# Verify FCM config
cat /opt/armorclaw/configs/sygnal.yaml | grep -A5 fcm

# Test push endpoint
curl -X POST http://localhost:5000/_matrix/push/v1/notify \
  -H "Content-Type: application/json" \
  -d '{"notification":{"event_id":"test","room_id":"test","type":"m.room.message"}}'
```

### Clients Can't Connect
1. **Check firewall:** `ufw status`
2. **Check DNS:** `nslookup matrix.yourdomain.com`
3. **Check TLS:** `curl -v https://matrix.yourdomain.com`
4. **Check Nginx:** `docker logs armorclaw-nginx`

---

## Quick Reference Commands

```bash
# Start all services
docker compose up -d
systemctl start armorclaw-bridge

# Stop all services
systemctl stop armorclaw-bridge
docker compose down

# View logs
journalctl -u armorclaw-bridge -f
docker compose logs -f matrix-conduit
docker compose logs -f sygnal

# Health check
./deploy/health-check.sh

# Test RPC
echo '{"jsonrpc":"2.0","id":1,"method":"bridge.health"}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Restart bridge
systemctl restart armorclaw-bridge

# Rebuild bridge
cd /opt/armorclaw/bridge && go build -o /opt/armorclaw/armorclaw-bridge ./cmd/bridge
systemctl restart armorclaw-bridge
```

---

## Support

- **Documentation:** `/opt/armorclaw/docs/`
- **GitHub Issues:** https://github.com/armorclaw/armorclaw/issues
- **Error Catalog:** `docs/guides/error-catalog.md`
- **Health Check:** `./deploy/health-check.sh`

---

**Checklist Version:** 0.2.0
**Last Updated:** 2026-02-21
