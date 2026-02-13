# Hostinger VPS Docker Deployment Guide

> **Last Updated:** 2026-02-07
> **Purpose:** Comprehensive guide for deploying ArmorClaw on Hostinger VPS using Docker
> **Time:** 10-15 minutes (with automated script)
> **Difficulty:** Easy

---

## Quick Start: Automated Deployment ⭐

**Fastest way to deploy ArmorClaw to Hostinger VPS:**

```bash
# From local machine
cd armorclaw
scp deploy/vps-deploy.sh armorclaw-deploy.tar.gz root@YOUR_VPS_IP:/tmp/

# SSH into VPS
ssh root@YOUR_VPS_IP

# Run deployment script
chmod +x /tmp/vps-deploy.sh
sudo bash /tmp/vps-deploy.sh
```

**The automated script includes:**
- ✅ Pre-flight checks (disk, memory, ports)
- ✅ Docker & Docker Compose installation
- ✅ Tarball verification and extraction
- ✅ Interactive configuration
- ✅ Automated deployment with status monitoring

**For detailed manual deployment methods, see below.**

---

## Overview

This guide covers deploying ArmorClaw on Hostinger VPS using Docker, with specific focus on:

- **Method 1:** Hostinger Docker Manager (Web-based, Easiest) ⭐
- **Method 2:** Docker Compose via CLI (Traditional)
- **Method 3:** Manual Docker deployment (Full control)

### Why Hostinger for ArmorClaw?

Hostinger VPS offers excellent value for ArmorClaw deployment:
- **KVM2 Plan:** 8GB RAM, 2 vCPUs - sufficient for full stack
- **NVMe Storage:** Fast database performance
- **Docker Manager:** Web-based container management (no CLI required)
- **One-click Templates:** Quick deployment options
- **Competitive Pricing:** ~$8-12/month for KVM2

### System Requirements

| Component | KVM1 | KVM2 (Recommended) | KVM4+ |
|-----------|------|-------------------|-------|
| **RAM** | 4 GB | 8 GB | 16 GB |
| **CPU** | 1 vCPU | 2 vCPUs | 4 vCPUs |
| **Storage** | 50 GB NVMe | 80 GB NVMe | 160 GB NVMe |
| **ArmorClaw** | ✅ Bare minimum | ✅ Recommended | ✅ Multi-agent |

**Memory Budget Analysis:**
```
Ubuntu minimal:    400 MB
Matrix Conduit:    200 MB
Caddy (SSL):       40 MB
Coturn (TURN):     50 MB
Local Bridge:      50 MB
Agent Container:   800 MB
─────────────────────────────
Total Phase 1:    ~1.54 GB  ✅ Fits in KVM1 (2.54 GB headroom)
Total Phase 4:    ~1.74 GB  ✅ Fits in KVM1 (2.26 GB headroom)
```

**✅ All Hostinger VPS plans can run ArmorClaw comfortably.**

---

## Method 1: Hostinger Docker Manager ⭐ (Easiest)

**Hostinger Docker Manager** is a web-based interface that eliminates the need for command-line Docker operations. Perfect for users who prefer GUI over CLI.

### Advantages
- ✅ No SSH/CLI required
- ✅ Visual project management dashboard
- ✅ One-click deployment from GitHub
- ✅ Real-time container monitoring
- ✅ Built-in log viewer
- ✅ Form Editor for beginners

### Prerequisites

1. **Hostinger VPS** with Docker template installed
2. **Domain name** pointed to your VPS (for Matrix/Element X)
3. **ArmorClaw repository** available on GitHub (public or private)

### Step 1: Prepare VPS with Docker Template

1. **Login to Hostinger hPanel**
   - Navigate to **VPS** → **Manage**

2. **Reinstall with Docker Template**
   - Go to **Operating System** tab
   - Select **Docker on Ubuntu 22.04** or **Docker on Ubuntu 24.04**
   - Click **Reinstall** (⚠️ This will wipe existing data)

3. **Wait for Installation**
   - Installation takes 5-10 minutes
   - You'll receive email when complete

### Step 2: Access Docker Manager

1. **From hPanel Dashboard:**
   - Navigate to **VPS** → **Manage**
   - Click **Docker Manager** in the left sidebar

2. **Familiarize with Interface:**
   - **Projects Tab:** View all deployed stacks
   - **Containers Tab:** Individual container management
   - **Images Tab:** Manage Docker images
   - **Volumes Tab:** Persistent storage
   - **Networks Tab:** Docker networks

### Step 3: Deploy ArmorClaw from Docker Compose

**Option A: Upload docker-compose-stack.yml manually**

1. **Prepare docker-compose-stack.yml:**
   ```yaml
   version: '3.8'

   services:
     matrix:
       image: matrixconduit/matrix-conduit:latest
       container_name: armorclaw-matrix
       restart: unless-stopped
       ports:
         - "6167:6167"  # Matrix Federation
         - "8448:8448"  # Client API
       volumes:
         - matrix_data:/var/lib/matrix-conduit
       environment:
         - SERVER_NAME=matrix.yourdomain.com
       healthcheck:
         test: ["CMD", "curl", "-f", "http://localhost:6167/_matrix/client/versions"]
         interval: 30s
         timeout: 10s
         retries: 3

     caddy:
       image: caddy:2-alpine
       container_name: armorclaw-caddy
       restart: unless-stopped
       ports:
         - "80:80"
         - "443:443"
       volumes:
         - ./configs/Caddyfile:/etc/caddy/Caddyfile:ro
         - caddy_data:/data
         - caddy_config:/config
       environment:
         - DOMAIN=matrix.yourdomain.com
       depends_on:
         matrix:
           condition: service_healthy

     bridge:
       build:
         context: https://github.com/armorclaw/armorclaw.git#main
         dockerfile: bridge/Dockerfile
       container_name: armorclaw-bridge
       restart: unless-stopped
       volumes:
         - bridge_run:/run/armorclaw
         - bridge_keystore:/var/lib/armorclaw
         - /var/run/docker.sock:/var/run/docker.sock:ro
       environment:
         - ARMORCLAW_KEYSTORE_DB=/var/lib/armorclaw/keystore.db
         - ARMORCLAW_SOCKET_PATH=/run/armorclaw/bridge.sock
       depends_on:
         matrix:
           condition: service_healthy

   volumes:
     matrix_data:
     caddy_data:
     caddy_config:
     bridge_run:
     bridge_keystore:
   ```

2. **In Docker Manager:**
   - Click **New Project**
   - Select **Compose manually**
   - Choose **YAML Editor** (or Form Editor if you prefer)
   - Paste the docker-compose-stack.yml content
   - Replace `yourdomain.com` with your actual domain
   - Click **Create**

3. **Monitor Deployment:**
   - Watch the **Logs** tab for real-time output
   - Wait for all containers to show **"Running"** status
   - This typically takes 3-5 minutes

**Option B: Deploy directly from GitHub (One-Click) ⭐**

1. **Ensure ArmorClaw has docker-compose-stack.yml in repository:**
   - File must be at root level
   - All service configurations included

2. **In Docker Manager:**
   - Click **New Project**
   - Select **Compose from URL**
   - Enter repository URL: `https://github.com/armorclaw/armorclaw.git`
   - Select branch: `main`
   - Click **Import**

3. **Configure Environment Variables:**
   - After import, click **Edit Project**
   - Go to **Environment Variables** tab
   - Add variables:
     ```
     MATRIX_DOMAIN=matrix.yourdomain.com
     MATRIX_ADMIN_USER=admin
     MATRIX_ADMIN_PASSWORD=your-secure-password
     ```

4. **Deploy:**
   - Click **Save & Deploy**
   - Monitor logs for deployment progress

### Step 4: Configure DNS in hPanel

1. **Navigate to:** Domains → **DNS Zone**

2. **Add A Records:**
   ```
   Type: A
   Name: matrix
   Points to: YOUR_VPS_IP
   TTL: 3600

   Type: A
   Name: chat
   Points to: YOUR_VPS_IP
   TTL: 3600
   ```

3. **Verify DNS Propagation:**
   - Use Docker Manager's **Terminal** or SSH
   - Run: `dig +short matrix.yourdomain.com`
   - Should return your VPS IP

### Step 5: Configure Firewall via hPanel

1. **Navigate to:** VPS → **Manage** → **Firewall**

2. **Add Rules:**
   ```
   SSH:        22/tcp    (or your custom port)
   HTTP:       80/tcp
   HTTPS:      443/tcp
   Matrix:     8448/tcp
   Federation: 6167/tcp
   ```

3. **Enable Firewall**

### Step 6: Verify Deployment

1. **Check Container Status in Docker Manager:**
   - All containers should show **"Running"**
   - Green checkmarks indicate healthy services

2. **Test Matrix Server:**
   - Open Docker Manager **Terminal**
   - Run: `curl -I https://matrix.yourdomain.com`
   - Should return HTTP 200

3. **Check Bridge Socket:**
   - In terminal: `ls -la /var/run/docker.sock/volumes/bridge_run/_data/`
   - Should see `bridge.sock`

### Step 7: Connect via Element X

1. **Download Element X app** on your device

2. **Login with:**
   - **Homeserver:** `https://matrix.yourdomain.com`
   - **Username:** `admin`
   - **Password:** (from environment variables)

3. **Join or create room:** `#agents:yourdomain.com`

---

## Method 2: Docker Compose via CLI (Traditional)

For users comfortable with SSH and command-line interface.

### Prerequisites

- Hostinger VPS with Ubuntu 20.04/22.04/24.04
- Root SSH access
- Domain name configured

### Step 1: Install Docker and Docker Compose

**Option A: Hostinger Auto-Installation (Recommended)**

1. **In hPanel:**
   - Navigate to **VPS** → **Manage**
   - Go to **Operating System** tab
   - Select **Docker on Ubuntu 22.04** or **24.04**
   - Click **Reinstall**

2. **Wait for email confirmation** (5-10 minutes)

**Option B: Manual Installation**

```bash
# SSH into your VPS
ssh root@your-vps-ip -p 2222

# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose plugin
apt install docker-compose-plugin -y

# Enable and start Docker
systemctl enable docker
systemctl start docker

# Verify installation
docker --version
docker compose version
```

### Step 2: Create ArmorClaw Directory

```bash
# Create directory
mkdir -p /opt/armorclaw
cd /opt/armorclaw

# Download ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git .
# OR upload tarball (see docs/guides/hostinger-deployment.md)
```

### Step 3: Configure Environment

```bash
# Create .env file
cat > .env <<EOF
MATRIX_DOMAIN=matrix.yourdomain.com
MATRIX_ADMIN_USER=admin
MATRIX_ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
MATRIX_BRIDGE_USER=bridge
MATRIX_BRIDGE_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
ROOM_NAME=ArmorClaw Agents
ROOM_ALIAS=agents
EOF

# Secure the .env file
chmod 600 .env

# Note the admin password
echo "Matrix Admin Password: $(grep MATRIX_ADMIN_PASSWORD .env | cut -d= -f2)"
```

### Step 4: Create Caddy Configuration

```bash
mkdir -p configs

cat > configs/Caddyfile <<EOF
{
    email admin@yourdomain.com
}

matrix.yourdomain.com {
    reverse_proxy matrix:6167
}

chat.yourdomain.com {
    reverse_proxy matrix:8448
}
EOF
```

### Step 5: Deploy Stack

```bash
# Deploy all services
docker compose -f docker-compose-stack.yml up -d

# Check status
docker compose -f docker-compose-stack.yml ps

# View logs
docker compose -f docker-compose-stack.yml logs -f
```

### Step 6: Create Matrix Admin User

```bash
# Access matrix container
docker exec -it armorclaw-matrix bash

# Register admin user
conduit_user = create_admin

# Exit container
exit
```

### Continue with Steps 4-7 from Method 1

(DNS configuration, firewall, Element X connection)

---

## Method 3: Manual Docker Deployment (Full Control)

For users who need maximum customization and control.

### When to Use This Method

- Custom container configurations
- Special network requirements
- Multiple ArmorClaw instances
- Development/testing environments

### Step 1: Install Docker (if not already installed)

```bash
curl -fsSL https://get.docker.com | sh
usermod -aG docker $USER
```

### Step 2: Build ArmorClaw Bridge

```bash
cd /opt/armorclaw/bridge

# Install Go (if not installed)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build bridge
CGO_ENABLED=1 go build -o build/armorclaw-bridge ./cmd/bridge

# Verify
./build/armorclaw-bridge version
```

### Step 3: Build Container Image

```bash
cd /opt/armorclaw

# Build agent container
docker build -t armorclaw/agent:v1 .

# Verify image
docker images | grep armorclaw
```

### Step 4: Create Docker Network

```bash
# Create isolated network
docker network create armorclaw-net
```

### Step 5: Deploy Matrix Conduit

```bash
# Create volume
docker volume create matrix_data

# Run Matrix Conduit
docker run -d \
  --name armorclaw-matrix \
  --network armorclaw-net \
  --restart unless-stopped \
  -p 6167:6167 \
  -p 8448:8448 \
  -v matrix_data:/var/lib/matrix-conduit \
  -e SERVER_NAME=matrix.yourdomain.com \
  matrixconduit/matrix-conduit:latest

# Check status
docker ps | grep armorclaw-matrix
```

### Step 6: Deploy Caddy (SSL Reverse Proxy)

```bash
# Create volumes
docker volume create caddy_data
docker volume create caddy_config

# Create Caddyfile
mkdir -p /opt/armorclaw/configs
cat > /opt/armorclaw/configs/Caddyfile <<EOF
matrix.yourdomain.com {
    reverse_proxy armorclaw-matrix:6167
}
EOF

# Run Caddy
docker run -d \
  --name armorclaw-caddy \
  --network armorclaw-net \
  --restart unless-stopped \
  -p 80:80 \
  -p 443:443 \
  -v caddy_data:/data \
  -v caddy_config:/config \
  -v /opt/armorclaw/configs/Caddyfile:/etc/caddy/Caddyfile:ro \
  caddy:2-alpine
```

### Step 7: Deploy Bridge

```bash
# Create volumes
docker volume create bridge_run
docker volume create bridge_keystore

# Run bridge
docker run -d \
  --name armorclaw-bridge \
  --network armorclaw-net \
  --restart unless-stopped \
  -v bridge_run:/run/armorclaw \
  -v bridge_keystore:/var/lib/armorclaw \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e ARMORCLAW_KEYSTORE_DB=/var/lib/armorclaw/keystore.db \
  -e ARMORCLAW_SOCKET_PATH=/run/armorclaw/bridge.sock \
  -e ARMORCLAW_MATRIX_HOMESERVER=https://matrix.yourdomain.com \
  armorclaw/bridge:latest

# Check logs
docker logs armorclaw-bridge
```

### Step 8: Create Systemd Service (Optional)

```bash
# Create service file
cat > /etc/systemd/system/armorclaw.service <<EOF
[Unit]
Description=ArmorClaw Bridge
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/armorclaw
ExecStart=/usr/bin/docker compose -f docker-compose-stack.yml up -d
ExecStop=/usr/bin/docker compose -f docker-compose-stack.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl enable armorclaw
systemctl start armorclaw
systemctl status armorclaw
```

---

## Hostinger-Specific Optimizations

### Resource Constraints for KVM Plans

**For KVM1 (4GB RAM):**

Edit docker-compose-stack.yml to add memory limits:

```yaml
services:
  matrix:
    image: matrixconduit/matrix-conduit:latest
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M

  caddy:
    image: caddy:2-alpine
    deploy:
      resources:
        limits:
          memory: 128M
        reservations:
          memory: 64M

  bridge:
    deploy:
      resources:
        limits:
          memory: 256M
        reservations:
          memory: 128M
```

**For KVM2 (8GB RAM):**

No additional limits needed - full stack runs comfortably.

### Storage Optimization

Hostinger VPS uses NVMe storage, but it's still good practice:

```yaml
# Use local driver for volumes (default on Hostinger)
volumes:
  matrix_data:
    driver: local
  caddy_data:
    driver: local
```

### Network Optimization

Hostinger VPS has excellent network connectivity:

```yaml
# Use bridge network (best performance on Hostinger)
networks:
  default:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### Backup Strategy

Hostinger provides automated backups, but for critical data:

```bash
# Manual backup script
cat > /opt/armorclaw/backup.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/var/backups/armorclaw"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup volumes
docker run --rm \
  -v matrix_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/matrix_$DATE.tar.gz -C /data .

docker run --rm \
  -v bridge_keystore:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/keystore_$DATE.tar.gz -C /data .

# Keep only last 7 days
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
EOF

chmod +x /opt/armorclaw/backup.sh

# Add to crontab (daily at 2 AM)
crontab -l | { cat; echo "0 2 * * * /opt/armorclaw/backup.sh"; } | crontab -
```

---

## Troubleshooting

### Issue: "Docker Manager won't install Docker template"

**Solution:**
1. Check VPS has enough disk space (minimum 10GB free)
2. Try reinstalling OS without Docker, then install manually
3. Contact Hostinger support if issue persists

### Issue: "Container exits immediately"

**Check logs in Docker Manager:**
1. Go to **Containers** tab
2. Click on problematic container
3. View **Logs** tab

**Common causes:**
- Port conflict (check with `netstat -tulpn`)
- Volume permission issue
- Missing environment variables

### Issue: "Matrix won't federate"

**Solution:**
1. Check firewall allows port 6167
2. Verify DNS A record points to VPS IP
3. Test federation: `curl -I https://matrix.yourdomain.com`

### Issue: "Bridge can't connect to Docker socket"

**Solution:**
```yaml
# Ensure mount is correct in docker-compose.yml:
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

### Issue: "Out of memory on KVM1"

**Solution:**
1. Add resource limits (see Hostinger-Specific Optimizations above)
2. Reduce container log size:
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```
3. Consider upgrading to KVM2

### Issue: "Can't access Docker Manager"

**Solution:**
1. Clear browser cache
2. Try different browser
3. Check VPS is accessible via SSH
4. Restart Docker service: `systemctl restart docker`

---

## Migration from Other Platforms

### Migrating from Local Development

```bash
# Export local volumes
docker save matrix_data caddy_data bridge_keystore -o armorclaw-backup.tar

# Transfer to Hostinger VPS
scp armorclaw-backup.tar root@your-vps-ip:/root/

# On Hostinger VPS
docker load -i /root/armorclaw-backup.tar
```

### Migrating from Another VPS

1. **Backup existing deployment:**
   ```bash
   docker compose --volumes > backup-$(date +%Y%m%d).tar
   ```

2. **Transfer to Hostinger:**
   ```bash
   scp backup-*.tar root@hostinger-vps:/root/
   ```

3. **Restore on Hostinger:**
   ```bash
   docker compose up -d
   docker volume restore backup-*.tar
   ```

---

## Security Best Practices

### 1. Use Strong Passwords

```bash
# Generate secure password
openssl rand -base64 24
```

### 2. Enable Firewall

```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 8448/tcp  # Matrix Client API
ufw allow 6167/tcp  # Matrix Federation
ufw enable
```

### 3. Restrict Docker Socket Access

```bash
# Create docker group
groupadd docker

# Add only specific users
usermod -aG docker armorclaw

# Remove world-readable permissions
chmod 660 /var/run/docker.sock
```

### 4. Regular Updates

```bash
# System updates
apt update && apt upgrade -y

# Docker updates
curl -fsSL https://get.docker.com | sh

# ArmorClaw updates
cd /opt/armorclaw
git pull
docker compose pull
docker compose up -d --build
```

### 5. Monitor Logs

```bash
# View all logs
docker compose logs -f

# View specific service logs
docker compose logs -f matrix
docker compose logs -f bridge
```

---

## Performance Tuning

### Enable Docker BuildKit

```bash
# Add to /etc/docker/daemon.json
{
  "features": {
    "buildkit": true
  }
}

# Restart Docker
systemctl restart docker
```

### Use Multi-Stage Builds

Already implemented in bridge/Dockerfile - reduces final image size.

### Optimize Matrix Conduit

```yaml
# Add to docker-compose-stack.yml
matrix:
  environment:
    - CONDUIT_MAX_REQUEST_SIZE=20_000_000
    - CONDUIT_MAX_CONCURRENT_REQUESTS=500
```

---

## Monitoring and Maintenance

### Health Checks

```bash
# Check all containers
docker ps

# Check specific container health
docker inspect armorclaw-matrix | grep -A 5 Health

# Bridge health
echo '{"jsonrpc":"2.0","method":"health","id":1}' | nc -U /run/armorclaw/bridge.sock
```

### Resource Monitoring

```bash
# Container resource usage
docker stats

# Disk usage
du -sh /var/lib/docker/volumes/*

# Memory usage
free -h
```

### Log Rotation

```yaml
# Add to docker-compose-stack.yml
services:
  bridge:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

---

## Integration with Hostinger Services

### hPanel DNS Management

1. **Navigate to:** Domains → **DNS Zone**
2. **Add required records:**
   - A record for matrix.yourdomain.com
   - A record for chat.yourdomain.com
3. **TTL:** Set to 3600 (1 hour) for faster propagation

### hPanel SSL Certificates

If not using Caddy's automatic Let's Encrypt:

1. **Navigate to:** Domains → **SSL**
2. **Generate free SSL certificate**
3. **Upload to:**
   ```bash
   /etc/ssl/certs/matrix.yourdomain.com.crt
   /etc/ssl/private/matrix.yourdomain.com.key
   ```

### Hostinger Automated Backups

1. **Navigate to:** VPS → **Manage** → **Backups**
2. **Enable daily backups**
3. **Retention:** 7-30 days (based on plan)

---

## Cost Comparison

| Plan | Monthly | RAM | CPU | Suitable For |
|------|---------|-----|-----|--------------|
| **KVM1** | ~$4-6 | 4 GB | 1 vCPU | Single agent, testing |
| **KVM2** | ~$8-12 | 8 GB | 2 vCPUs | **Recommended** |
| **KVM4** | ~$16-24 | 16 GB | 4 vCPUs | Multi-agent, production |
| **KVM8** | ~$32-48 | 32 GB | 8 vCPUs | Enterprise, scale |

**💡 Tip:** KVM2 offers the best value for ArmorClaw deployment.

---

## Next Steps

After successful deployment:

1. ✅ **Connect via Element X** - See [Element X Quick Start](element-x-quickstart.md)
2. ✅ **Configure API Keys** - Run `./deploy/setup-wizard.sh` in bridge container
3. ✅ **Start First Agent** - Use `docker exec armorclaw-bridge armorclaw-bridge start`
4. ✅ **Set Up Monitoring** - Configure health checks and alerts
5. ✅ **Configure Backups** - Enable automated backups in hPanel

---

## Additional Resources

### Hostinger Documentation
- [How to Install Docker on Ubuntu](https://www.hostinger.com/tutorials/how-to-install-docker-on-ubuntu)
- [What is Docker Compose](https://www.hostinger.com/tutorials/what-is-docker-compose)
- [Docker Manager for VPS](https://www.hostinger.com/support/docker-manager)

### ArmorClaw Documentation
- [Element X Quick Start](element-x-quickstart.md)
- [Hostinger Deployment (Tarball Transfer)](hostinger-deployment.md)
- [Infrastructure Deployment Guide](2026-02-05-infrastructure-deployment-guide.md)
- [Troubleshooting Guide](troubleshooting.md)

### Community Support
- [Hostinger Community](https://www.hostinger.com/community)
- [Docker Community Forums](https://forums.docker.com/)
- [Matrix Community](https://matrix.org/community/)

---

**Sources:**
- [Hostinger Docker Manager](https://www.hostinger.com/support/12040789-hostinger-docker-manager-for-vps-simplify-your-container-deployments/)
- [How to Install Docker on Ubuntu](https://www.hostinger.com/uk/tutorials/how-to-install-docker-on-ubuntu)
- [What is Docker Compose](https://www.hostinger.com/tutorials/what-is-docker-compose)
- [Hostinger Docker VPS Tutorial](https://www.youtube.com/watch?v=BwS2S-mmUG0)
- [HostAdvice Docker VPS Review](https://hk.hostadvice.com/hostinger-company/hostinger-reviews/hostinger-docker-vps-review/)
