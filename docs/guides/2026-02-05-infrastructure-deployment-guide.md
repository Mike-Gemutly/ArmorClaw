# ArmorClaw Infrastructure Deployment Guide

**Date:** 2026-02-05
**Purpose:** Deploy Matrix Conduit + Nginx + Coturn on Hostinger KVM2
**Prerequisites:** Hostinger KVM2 instance, Ubuntu 22.04/24.04

---

## Overview

This guide covers **Stage 1** of the Hybrid Implementation Strategy: deploying the communication infrastructure that the ArmorClaw Bridge will connect to.

### What You're Deploying

```
Hostinger KVM2 (4 GB RAM)
├── Matrix Conduit (Docker)    → Matrix homeserver
├── Nginx (Docker)              → Reverse proxy + SSL
├── Coturn (Docker)             → STUN/TURN for voice/video
└── (Later) ArmorClaw Bridge   → Runs on host, not in Docker
```

### Memory Allocation

| Component | RAM | Purpose |
|-----------|-----|---------|
| Ubuntu OS | 400 MB | Base system |
| Matrix Conduit | 200 MB | Homeserver |
| Nginx | 40 MB | Reverse proxy |
| Coturn | 50 MB | TURN relay |
| **Subtotal** | **690 MB** | Infrastructure |
| Headroom for Bridge + Agent | ~1.3 GB | For later stages |

---

## Prerequisites

### 1. DNS Configuration

Ensure your domain A record points to your Hostinger IP:

```
matrix.armorclaw.com.  A  123.45.67.89
chat.armorclaw.com.    A  123.45.67.89
```

Verify with:
```bash
dig +short matrix.armorclaw.com
```

### 2. Server Access

SSH into your Hostinger KVM2:
```bash
ssh root@your-server-ip
```

### 3. System Update

```bash
apt update && apt upgrade -y
```

---

## Deployment Steps

### Step 1: Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install docker-compose
apt install docker-compose -y

# Add your user to docker group (optional)
usermod -aG docker $USER

# Verify
docker --version
docker-compose --version
```

### Step 2: Deploy ArmorClaw Infrastructure

```bash
# Clone repository (if not already)
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Make scripts executable
chmod +x scripts/*.sh

# Run deployment script
./scripts/deploy-infrastructure.sh
```

The deployment script will:
1. Create directory structure
2. Validate DNS configuration
3. Pull Docker images
4. Start Matrix Conduit + Nginx + Coturn
5. Optionally obtain SSL certificates
6. Run health checks

### Step 3: Validate Deployment

```bash
# Run validation script
./scripts/validate-infrastructure.sh
```

Expected output:
```
========================================
ArmorClaw Infrastructure Validation
========================================

1. Docker Environment
Testing Docker daemon... ✓ PASS
Testing docker-compose... ✓ PASS

2. Container Status
Testing Matrix Conduit container... ✓ PASS
Testing Nginx container... ✓ PASS
Testing Coturn container... ✓ PASS

3. Service Endpoints
Testing Matrix Conduit API... ✓ PASS
Testing Nginx health check... ✓ PASS

...
```

---

## Configuration

### Matrix Conduit

Edit `configs/conduit.toml`:

```toml
[global]
server_name = "matrix.armorclaw.com"

[global.matrix]
allow_registration = true   # Enable for initial setup
```

Restart to apply:
```bash
docker-compose restart matrix-conduit
```

### Nginx SSL

SSL certificates are stored in `configs/nginx/ssl/`:
- `fullchain.pem` - Certificate chain
- `privkey.pem` - Private key
- `chain.pem` - CA chain

To obtain/renew certificates:
```bash
sudo certbot certonly --standalone -d matrix.armorclaw.com
sudo cp /etc/letsencrypt/live/matrix.armorclaw.com/*.pem configs/nginx/ssl/
docker-compose restart nginx
```

---

## Initial Setup

### Create Admin Account

1. Open Element Web: `https://app.element.io`
2. Click "Edit" next to "Homeserver"
3. Enter: `https://matrix.armorclaw.com`
4. Click "Continue"
5. Register your admin account

### Disable Open Registration

Edit `configs/conduit.toml`:
```toml
[global.matrix]
allow_registration = false
```

```bash
docker-compose restart matrix-conduit
```

---

## Testing

### Test Matrix API

```bash
# Check versions
curl http://localhost:6167/_matrix/client/versions

# Expected response:
# {"versions":["r0.6.0","v1.1","v1.2","v1.3","v1.4","v1.5"]}
```

### Test Nginx

```bash
# Health check
curl http://localhost/health

# Expected response:
# OK
```

### Test with Element

1. Go to `https://app.element.io`
2. Set homeserver to `https://matrix.armorclaw.com`
3. Login with your admin account
4. Create a test room
5. Note the room ID (e.g., `!abc123:matrix.armorclaw.com`)

---

## Service Management

### Start/Stop Services

```bash
# Start all
docker-compose up -d

# Stop all
docker-compose down

# Restart specific service
docker-compose restart matrix-conduit

# View logs
docker-compose logs -f matrix-conduit
```

### Update Services

```bash
# Pull latest images
docker-compose pull

# Recreate containers
docker-compose up -d --force-recreate
```

---

## Troubleshooting

### Issue: Container Won't Start

```bash
# Check logs
docker-compose logs matrix-conduit

# Check resource usage
docker stats
```

### Issue: DNS Not Resolving

```bash
# Verify DNS
dig +short matrix.armorclaw.com

# Check local hosts file
cat /etc/hosts
```

### Issue: SSL Certificate Errors

```bash
# Check certificate expiry
openssl x509 -in configs/nginx/ssl/fullchain.pem -noout -dates

# Renew certificate
sudo certbot renew
```

### Issue: High Memory Usage

```bash
# Check container memory
docker stats --no-stream

# Limit memory (add to docker-compose.yml)
deploy:
  resources:
    limits:
      memory: 400M
```

---

## Security Considerations

### Firewall Configuration

```bash
# Configure UFW
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

### TURN Server Authentication

Edit `configs/turnserver.conf`:

```conf
# Generate strong secret
static-auth-secret=$(openssl rand -hex 32)

# Add to .env
TURN_SECRET=your-generated-secret
```

### Regular Updates

```bash
# Update system
apt update && apt upgrade -y

# Update Docker images
docker-compose pull
docker-compose up -d --force-recreate
```

---

## Performance Tuning

### Matrix Conduit

For small teams (< 50 users), default settings are sufficient.

For larger deployments, edit `configs/conduit.toml`:

```toml
[global]
max_concurrent_requests = 500
```

### Nginx

Edit `configs/nginx.conf`:

```nginx
worker_processes auto;
worker_connections 512;
```

---

## Monitoring

### Health Checks

```bash
# Quick health check
curl http://localhost/health

# Detailed status
docker-compose ps
```

### Log Monitoring

```bash
# Follow all logs
docker-compose logs -f

# Specific service
docker-compose logs -f matrix-conduit

# Last 100 lines
docker-compose logs --tail=100 matrix-conduit
```

---

## Next Steps

Once infrastructure is deployed and validated:

1. ✅ **Stage 1 Complete** - Infrastructure is running
2. ⏳ **Stage 2:** Initialize Go Bridge Repository
   ```bash
   ./scripts/init-bridge-repo.sh
   ```
3. ⏳ **Stage 3:** Implement Bridge Core (Phase 1)
   - Task 1.2: Configuration System
   - Task 1.3: Encrypted Keystore
   - Task 1.4: JSON-RPC Server
   - Task 1.5: Matrix Client

---

## Validation Checklist

Before proceeding to Stage 2, confirm:

- [ ] Matrix Conduit is responding on port 6167
- [ ] Nginx is responding on port 80/443
- [ ] SSL certificate is valid (if enabled)
- [ ] Admin account created in Element
- [ ] Open registration disabled
- [ ] All containers show "Up" status
- [ ] Memory usage < 700 MB total
- [ ] No errors in logs

Run validation:
```bash
./scripts/validate-infrastructure.sh
```

---

## Success Criteria

Stage 1 is complete when:

✅ Matrix Conduit API returns valid JSON
✅ Nginx serves requests on HTTPS (if SSL enabled)
✅ Element Web can connect and register/login
✅ Test room can be created
✅ Room ID is available for bridge testing

**Expected URL for testing:** `https://matrix.armorclaw.com/_matrix/client/versions`

---

**Stage 1 Status:** ✅ Ready to deploy
**Estimated Time:** 30-45 minutes
**Difficulty:** Beginner
