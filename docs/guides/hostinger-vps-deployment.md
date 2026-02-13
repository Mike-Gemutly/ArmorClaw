# Hostinger VPS Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Hostinger (https://www.hostinger.com)
> **Best For:** Budget production deployments, dedicated resources, full Docker control
> **Difficulty Level:** Intermediate
> **Estimated Time:** 30-45 minutes

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Account Setup](#account-setup)
4. [VPS Creation](#vps-creation)
5. [Initial Server Setup](#initial-server-setup)
6. [Docker Installation](#docker-installation)
7. [ArmorClaw Deployment](#armorclaw-deployment)
8. [Network Configuration](#network-configuration)
9. [Domain and SSL Setup](#domain-and-ssl-setup)
10. [Persistent Storage](#persistent-storage)
11. [Monitoring and Maintenance](#monitoring-and-maintenance)
12. [Scaling Considerations](#scaling-considerations)
13. [Pricing Details](#pricing-details)
14. [Limitations](#limitations)
15. [Troubleshooting](#troubleshooting)
16. [Security Best Practices](#security-best-practices)

---

## Overview

**Hostinger VPS** provides an excellent balance of cost and performance for ArmorClaw deployments. With plans starting at $1.99/month (promotional), it offers dedicated resources, full root access, and complete Docker control.

### Why Hostinger for ArmorClaw?

✅ **Dedicated Resources** - No noisy neighbors, consistent performance
✅ **Full Docker Control** - Docker Manager GUI + CLI access
✅ **Root Access** - Complete system control for security hardening
✅ **Budget-Friendly** - Starting at $1.99/month (promotional)
✅ **AI Log Analysis** - Built-in Docker log analysis with AI
✅ **Global Data Centers** - Multiple geographic regions

### Recommended Plans

| Plan | vCPU | RAM | Storage | Bandwidth | Monthly Cost |
|------|------|-----|---------|-----------|--------------|
| **VPS 1** | 1 | 1 GB | 20 GB SSD | 1 TB | $1.99* (promo) |
| **VPS 2** | 2 | 2 GB | 40 GB SSD | 2 TB | ~$3-5* |
| **VPS 4** ⭐ | 2 | 4 GB | 80 GB SSD | 4 TB | ~$4-8* |
| **VPS 6** | 4 | 8 GB | 160 GB SSD | 8 TB | ~$8-16* |

*Promotional pricing requires upfront payment (monthly rate reflects total cost divided by months)

**Recommendation:** VPS 4 or higher for ArmorClaw to meet the 2 GB memory budget target.

---

## Prerequisites

### Before You Begin

- **Active Hostinger Account:** Sign up at https://www.hostinger.com
- **Payment Method:** Credit card or PayPal required
- **Domain Name (Optional):** For custom domain access
- **SSH Client:** Terminal (Linux/macOS) or PuTTY (Windows)
- **Basic Linux Knowledge:** Comfortable with command line

### System Requirements

- **Operating System:** Ubuntu 22.04 LTS or 24.04 LTS (recommended)
- **Minimum Resources:** 2 GB RAM, 20 GB storage, 1 vCPU
- **Recommended Resources:** 4 GB RAM, 80 GB storage, 2 vCPU
- **Network:** Public IP address with ports 80, 443, 6167 available

---

## Account Setup

### Step 1: Create Hostinger Account

1. Visit https://www.hostinger.com
2. Click **"Get Started"** or **"Sign Up"**
3. Choose signup method:
   - Email + password
   - Google account
   - Facebook account
4. Complete email verification (sent to registered email)
5. Log in to hPanel (Hostinger control panel)

### Step 2: Claim Credits (If Available)

- **New User Credits:** Check email for welcome credits
- **Promotional Codes:** Apply during checkout
- **Referral Credits:** If referred by existing user

### Step 3: Access VPS Dashboard

1. In hPanel, locate **"VPS"** section
2. Click **"VPS Hosting"**
3. Review available VPS plans
4. Proceed to [VPS Creation](#vps-creation)

---

## VPS Creation

### Step 1: Choose VPS Plan

1. Select desired plan (VPS 4 recommended for ArmorClaw)
2. **Billing Cycle:**
   - Monthly (higher rate, flexible)
   - 12 months (best promotional rate)
   - 24 months (maximum discount)
   - 48 months (long-term commitment)

3. **Example Pricing (VPS 4):**
   - Monthly: ~$8-12/month
   - 12 months: ~$4-6/month (total paid upfront)
   - 24 months: ~$3-4/month (total paid upfront)

### Step 2: Configure VPS

#### Operating System Selection

**Recommended:** Ubuntu 22.04 LTS or 24.04 LTS

**Options:**
- Ubuntu 22.04 LTS "Jammy Jellyfish" (recommended, stable)
- Ubuntu 24.04 LTS "Noble Numbat" (latest LTS)
- Debian 11 "Bullseye" (stable, minimal)
- CentOS Stream (not recommended for ArmorClaw)

#### Data Center Selection

**Available Regions:** Choose geographically closest to your users

- **North America:**
  - USA (multiple locations)
  - Canada
- **Europe:**
  - United Kingdom
  - Netherlands (Amsterdam)
  - Germany
  - France
  - Lithuania
- **Asia:**
  - Singapore
  - India
- **South America:**
  - Brazil
- **Australia:** Available

### Step 3: Additional Services (Optional)

- **Domain Registration:** Add during checkout
- **SSL Certificate:** Let's Encrypt (free) available after deployment
- **Backup Service:** Automated backups (recommended)
- **IP Address:** Additional IPs available

### Step 4: Complete Purchase

1. Review order summary
2. Apply promo codes (if available)
3. Complete payment
4. Wait for VPS provisioning (typically 1-5 minutes)
5. Receive welcome email with credentials

---

## Initial Server Setup

### Step 1: Access VPS Dashboard

1. In hPanel, locate your new VPS
2. Click **"Manage"** or **"Dashboard"**
3. Note the following information:
   - **IP Address:** xxx.xxx.xxx.xxx
   - **Username:** root
   - **Password:** (found in hPanel under "SSH Keys" or emailed)

### Step 2: SSH Into VPS

**Linux/macOS:**
```bash
ssh root@your-vps-ip
```

**Windows (PowerShell):**
```powershell
ssh root@your-vps-ip
```

**Windows (PuTTY):**
1. Download PuTTY from https://www.putty.org
2. Enter VPS IP address
3. Port: 22
4. Click "Open"
5. Login as: root
6. Password: (paste from clipboard, right-click)

**First Login Tips:**
- Password doesn't appear as you type (normal Linux behavior)
- Copy password from hPanel: **VPS → Dashboard → SSH Keys**
- Or check welcome email

### Step 3: Update System

```bash
# Update package list
apt update

# Upgrade installed packages
apt upgrade -y

# Auto-remove unnecessary packages
apt autoremove -y

# Reboot if kernel updated (optional)
# reboot
```

### Step 4: Set Hostname (Optional)

```bash
# Set hostname
hostnamectl set-hostname armorclaw

# Edit hosts file
nano /etc/hosts

# Add line:
127.0.1.1 armorclaw

# Save: Ctrl+O, Enter, Ctrl+X
```

### Step 5: Set Timezone

```bash
# List timezones
timedatectl list-timezones

# Set timezone (example: America/New_York)
timedatectl set-timezone America/New_York

# Verify
timedatectl
```

### Step 6: Create Non-Root User (Security Best Practice)

```bash
# Create user
adduser armorclaw

# Add user to sudo group
usermod -aG sudo armorclaw

# Switch to user
su - armorclaw
```

**Continue rest of guide as `armorclaw` user** (use `sudo` for administrative commands).

---

## Docker Installation

### Option A: Using Docker Manager (Hostinger GUI) ⭐ EASIEST

**Advantages:**
- Web-based GUI, no SSH required
- AI-powered log analysis
- One-click Docker templates
- Real-time monitoring

#### Step 1: Access Docker Manager

1. In hPanel, navigate to **VPS → Manage**
2. Click **"Docker Manager"** tab
3. Wait for Docker Manager installation (first time only)

#### Step 2: Install Docker Templates

**Available Templates:**
- **Docker Registry** - Private container registry
- **Portainer** - Docker management GUI
- **Dockge** - Docker Compose manager
- **Popular Apps** - Pre-configured containers

#### Step 3: Deploy ArmorClaw via Docker Manager

1. Click **"Create Container"**
2. Choose deployment method:
   - **From Image:** Pull from Docker Hub
   - **From Dockerfile:** Build from source
3. Configure container:
   - Name: `armorclaw-bridge`
   - Image: `armorclaw/bridge:latest`
   - Ports: `80:80`, `443:443`, `6167:6167`
   - Volumes: Configure persistent storage
4. Click **"Deploy"**

#### Step 4: Monitor with AI Log Analysis

1. Click on deployed container
2. Navigate to **"Logs"** tab
3. AI analyzes logs for:
   - Errors and warnings
   - Performance bottlenecks
   - Security anomalies
   - Optimization suggestions

---

### Option B: Manual Docker Installation (Full Control)

**Advantages:**
- Latest Docker version
- Complete control over configuration
- Access to Docker Compose
- Scriptable/automatable

#### Step 1: Install Prerequisites

```bash
# Update package index
sudo apt update

# Install required packages
sudo apt install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release
```

#### Step 2: Add Docker's Official GPG Key

```bash
# Create directory for GPG key
sudo mkdir -p /etc/apt/keyrings

# Add Docker GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
```

#### Step 3: Set Up Docker Repository

```bash
# Add Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Update package index
sudo apt update
```

#### Step 4: Install Docker Engine

```bash
# Install Docker Engine, CLI, and Compose plugin
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Verify installation
docker --version
# Output: Docker version 26.x.x, build xxxxxxx

# Test Docker (hello-world)
sudo docker run hello-world
```

#### Step 5: Manage Docker as Non-Root User

```bash
# Create docker group (if not exists)
sudo groupadd docker

# Add user to docker group
sudo usermod -aG docker $USER

# Log out and log back in for changes to take effect
exit
# SSH back in

# Verify docker works without sudo
docker run hello-world
```

#### Step 6: Enable Docker on Boot

```bash
# Enable Docker service
sudo systemctl enable docker.service

# Enable container service
sudo systemctl enable containerd.service

# Check Docker status
sudo systemctl status docker
```

---

## ArmorClaw Deployment

### Step 1: Clone ArmorClaw Repository

```bash
# Install git (if not installed)
sudo apt install -y git

# Clone repository
git clone https://github.com/armorclaw/armorclaw.git

# Navigate to project directory
cd armorclaw

# List contents
ls -la
```

**Expected Output:**
```
drwxr-xr-x  2 armorclaw armorclaw 4096 Feb  7 12:00 bridge
drwxr-xr-x  2 armorclaw armorclaw 4096 Feb  7 12:00 container
drwxr-xr-x  2 armorclaw armorclaw 4096 Feb  7 12:00 configs
drwxr-xr-x  2 armorclaw armorclaw 4096 Feb  7 12:00 deploy
-rw-r--r--  1 armorclaw armorclaw 3241 Feb  7 12:00 docker-compose.yml
-rw-r--r--  1 armorclaw armorclaw 2456 Feb  7 12:00 Dockerfile
```

### Step 2: Build ArmorClaw Bridge

```bash
# Navigate to bridge directory
cd bridge

# Build bridge binary
go build -o build/armorclaw-bridge ./cmd/bridge

# Verify build
ls -lh build/
# Output: armorclaw-bridge (11 MB)

# Return to project root
cd ..
```

### Step 3: Build Container Image

```bash
# Build ArmorClaw agent image
docker build -t armorclaw/agent:v1 .

# Verify image
docker images | grep armorclaw
# Output: armorclaw/agent v1 [image-id] 393MB ago xx minutes
```

### Step 4: Deploy Infrastructure Stack

```bash
# Deploy all services (Matrix, Nginx, Caddy, etc.)
docker-compose up -d

# Check running containers
docker ps

# Expected output:
# CONTAINER ID   IMAGE                    STATUS
# xxxxxxxxxxxx   matrixconduit/matrix:    Up 2 minutes
# xxxxxxxxxxxx   caddy:latest             Up 2 minutes
# xxxxxxxxxxxx   armorclaw/bridge:latest  Up 2 minutes
```

### Step 5: Verify Deployment

```bash
# Check bridge status
./bridge/build/armorclaw-bridge status

# Check health
./bridge/build/armorclaw-bridge health

# View logs
docker-compose logs -f bridge
```

---

## Network Configuration

### Step 1: Configure Cloud Firewall (Hostinger)

**Access Cloud Firewall:**
1. Navigate to **VPS → Manage → Firewall**
2. Default rules may block all traffic
3. Add necessary rules

**Required Firewall Rules:**

| Rule | Protocol | Port | Source | Description |
|------|----------|------|--------|-------------|
| **SSH** | TCP | 22 | Your IP (recommended) or 0.0.0.0/0 | SSH access |
| **HTTP** | TCP | 80 | 0.0.0.0/0 | Web access (redirect to HTTPS) |
| **HTTPS** | TCP | 443 | 0.0.0.0/0 | Secure web access |
| **Matrix** | TCP | 6167 | 0.0.0.0/0 or specific | Matrix client-server API |
| **Matrix Federation** | TCP | 8448 | 0.0.0.0/0 or specific | Matrix federation |

**Adding Rules:**
1. Click **"Add Rule"**
2. Select protocol (TCP)
3. Enter port
4. Select source (0.0.0.0/0 for all, or specific IP)
5. Click **"Save"**

### Step 2: Configure UFW (Optional - Additional Security)

```bash
# Install UFW (if not installed)
sudo apt install -y ufw

# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (IMPORTANT: Do this first!)
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow Matrix
sudo ufw allow 6167/tcp
sudo ufw allow 8448/tcp

# Enable UFW
sudo ufw enable

# Check status
sudo ufw status
```

**⚠️ WARNING:** Always allow SSH (port 22) before enabling UFW, or you'll lock yourself out!

### Step 3: Test Connectivity

```bash
# Test HTTP
curl -I http://your-vps-ip

# Test HTTPS (if SSL configured)
curl -I https://your-vps-ip

# Test Matrix (from local machine)
telnet your-vps-ip 6167
```

---

## Domain and SSL Setup

### Step 1: Point Domain to VPS

**Option A: Use Hostinger-Registered Domain**

1. In hPanel, navigate to **Domains**
2. Select domain
3. Click **"DNS / Nameservers"**
4. Add **A Record:**
   - Type: A
   - Name: @ (root domain) or subdomain (e.g., matrix)
   - IPv4 Address: Your VPS IP
   - TTL: 3600 (1 hour)

**Option B: External Domain (e.g., Cloudflare, GoDaddy)**

1. Access DNS management at domain registrar
2. Add **A Record:**
   - Name: @ or subdomain
   - Value: Your VPS IP
   - TTL: 3600

**Propagation Time:** 1-24 hours (typically 1-2 hours)

### Step 2: Verify DNS Propagation

```bash
# Check from local machine
dig your-domain.com +short
# Expected output: Your VPS IP

# Or use nslookup
nslookup your-domain.com
```

### Step 3: Configure SSL with Let's Encrypt

**Option A: Using Caddy (Automatic SSL)** ⭐ RECOMMENDED

Caddy automatically obtains and renews SSL certificates.

```bash
# Edit Caddyfile
nano ~/armorclaw/configs/Caddyfile

# Add configuration:
your-domain.com {
    reverse_proxy bridge:8080
}

matrix.your-domain.com {
    reverse_proxy conduit:6167
}

# Save and restart Caddy
docker-compose restart caddy
```

**Option B: Using Certbot (Manual)**

```bash
# Install Certbot
sudo apt install -y certbot

# Obtain certificate
sudo certbot certonly --standalone -d your-domain.com -d matrix.your-domain.com

# Certificates installed to:
# /etc/letsencrypt/live/your-domain.com/fullchain.pem
# /etc/letsencrypt/live/your-domain.com/privkey.pem

# Auto-renewal is configured automatically
sudo certbot renew --dry-run
```

**Option C: Using Hostinger SSL**

1. In hPanel, navigate to **SSL**
2. Click **"Get Started"** for Let's Encrypt
3. Select domain
4. Click **"Install"**
5. Hostinger handles auto-renewal

---

## Persistent Storage

### Step 1: Create Docker Volumes

```bash
# Create volume for bridge data
docker volume create armorclaw-bridge-data

# Create volume for Matrix data
docker volume create armorclaw-matrix-data

# Create volume for Caddy data
docker volume create armorclaw-caddy-data

# List volumes
docker volume ls
```

### Step 2: Mount Volumes in docker-compose.yml

```yaml
version: '3.8'

services:
  bridge:
    volumes:
      - armorclaw-bridge-data:/run/armorclaw
      - armorclaw-keystore:/var/lib/armorclaw

  matrix:
    volumes:
      - armorclaw-matrix-data:/var/lib/conduit

  caddy:
    volumes:
      - armorclaw-caddy-data:/data
      - armorclaw-caddy-config:/config

volumes:
  armorclaw-bridge-data:
  armorclaw-keystore:
  armorclaw-matrix-data:
  armorclaw-caddy-data:
  armorclaw-caddy-config:
```

### Step 3: Backup Data

**Manual Backup:**
```bash
# Backup Docker volumes
docker run --rm \
  -v armorclaw-bridge-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/armorclaw-backup-$(date +%Y%m%d).tar.gz /data

# List backups
ls -lh armorclaw-backup-*.tar.gz
```

**Automated Backup Script:**
```bash
# Create backup script
nano ~/backup-armorclaw.sh

# Add content:
#!/bin/bash
BACKUP_DIR="/home/armorclaw/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup each volume
for volume in armorclaw-bridge-data armorclaw-matrix-data armorclaw-caddy-data; do
  docker run --rm \
    -v ${volume}:/data \
    -v ${BACKUP_DIR}:/backup \
    alpine tar czf /backup/${volume}-${DATE}.tar.gz /data
done

# Clean backups older than 7 days
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: $DATE"

# Save and make executable
chmod +x ~/backup-armorclaw.sh

# Add to crontab (daily at 2 AM)
crontab -e
# Add line:
0 2 * * * /home/armorclaw/backup-armorclaw.sh >> /home/armorclaw/backup.log 2>&1
```

---

## Monitoring and Maintenance

### Step 1: Monitor Resource Usage

**Via hPanel:**
1. Navigate to **VPS → Manage**
2. View **Resource Usage** dashboard
3. Metrics displayed:
   - CPU usage (%)
   - RAM usage (%)
   - Disk usage (%)
   - Bandwidth (GB/month)

**Via Command Line:**
```bash
# Check CPU and memory
htop

# Check disk usage
df -h

# Check Docker stats
docker stats

# Check container resource usage
docker stats --no-stream
```

### Step 2: View Logs

**Docker Logs:**
```bash
# View bridge logs
docker logs -f armorclaw-bridge

# View Matrix logs
docker logs -f armorclaw-matrix

# View all logs
docker-compose logs -f
```

**Journal Logs:**
```bash
# View system logs
sudo journalctl -f

# View Docker daemon logs
sudo journalctl -u docker -f
```

### Step 3: Set Up Monitoring with Prometheus (Optional)

```bash
# Deploy Prometheus stack
cd ~/armorclaw
docker-compose -f docker-compose.monitoring.yml up -d

# Access Grafana
# URL: http://your-vps-ip:3000
# Default credentials: admin/admin
```

### Step 4: Regular Maintenance Tasks

**Update System:**
```bash
# Weekly updates
sudo apt update && sudo apt upgrade -y && sudo apt autoremove -y
```

**Clean Docker:**
```bash
# Remove unused containers
docker container prune -f

# Remove unused images
docker image prune -a -f

# Remove unused volumes
docker volume prune -f

# Remove unused build cache
docker builder prune -f
```

**Reboot VPS:**
```bash
# Schedule reboot (monthly)
sudo reboot
```

---

## Scaling Considerations

### Vertical Scaling (Upgrade VPS Plan)

**When to Scale Up:**
- CPU usage consistently > 80%
- Memory usage consistently > 90%
- Disk usage > 80%
- Need more bandwidth

**How to Scale Up:**
1. In hPanel, navigate to **VPS → Manage**
2. Click **"Upgrade"**
3. Select higher plan
4. Choose additional options (storage, bandwidth)
5. Complete upgrade
6. VPS rebooted automatically with new resources

**Temporary CPU Limit Removal:**
- Hostinger allows removing CPU limits **once per week** for testing
- Useful for performance testing
- Navigate to **VPS → Manage → CPU**

### Horizontal Scaling (Multiple VPS Instances)

**Load Balancer Setup:**
1. Deploy multiple ArmorClaw instances on separate VPS
2. Configure load balancer (Hostinger offers load balancer add-on)
3. Distribute traffic across instances
4. Share state via external database (e.g., managed PostgreSQL)

---

## Pricing Details

### VPS Plans (2026 Pricing)

| Plan | vCPU | RAM | Storage | Bandwidth | Monthly (Promo)* | Monthly (Regular) |
|------|------|-----|---------|-----------|------------------|------------------|
| **VPS 1** | 1 | 1 GB | 20 GB | 1 TB | $1.99 | ~$5-7 |
| **VPS 2** | 2 | 2 GB | 40 GB | 2 TB | ~$3-5 | ~$7-10 |
| **VPS 4** ⭐ | 2 | 4 GB | 80 GB | 4 TB | ~$4-8 | ~$10-15 |
| **VPS 6** | 4 | 8 GB | 160 GB | 8 TB | ~$8-16 | ~$15-25 |
| **VPS 8** | 6 | 12 GB | 240 GB | 12 TB | ~$12-24 | ~$20-35 |
| **VPS 12** | 8 | 16 GB | 320 GB | 16 TB | ~$16-32 | ~$25-45 |

*Promotional pricing requires upfront payment (12, 24, or 48 months)

### Additional Costs

| Service | Cost |
|---------|------|
| **Domain Registration** | $9-15/year |
| **SSL Certificate** | Free (Let's Encrypt) |
| **Backup Service** | ~$1-2/month |
| **Additional Storage** | ~$1-2/10 GB/month |
| **Bandwidth Overage** | Varies by plan |
| **Load Balancer** | ~$5-10/month |

### Cost Optimization Tips

1. **Choose Longer Billing Cycles** - Maximum discount on 48-month plans
2. **Use Promotional Codes** - Check Hostinger coupons page
3. **Start with VPS 4** - Meets ArmorClaw requirements, upgrade later
4. **Monitor Bandwidth** - Avoid overage charges
5. **Use Free SSL** - Let's Encrypt is free and auto-renews

---

## Limitations

### Platform Limitations

| Limitation | Impact | Workaround |
|------------|--------|------------|
| **CPU Throttling** | Reported at 20% utilization | Upgrade plan or test with temporary CPU limit removal |
| **Resource Limits** | Processes, RAM, IO enforced | Monitor usage, upgrade if needed |
| **Bandwidth Caps** | Overage charges apply | Monitor bandwidth, upgrade plan |
| **No Auto-Scaling** | Manual scaling required | Plan capacity ahead, use load balancer |
| **No Managed Databases** | Self-managed only | Use external managed DB or Docker-based |
| **MySQL Packet Size** | 16MB limit | Configure smaller packets or use PostgreSQL |
| **MySQL Query Time** | 60 seconds max | Optimize queries, use PostgreSQL |

### ArmorClaw-Specific Considerations

✅ **Fully Supported:**
- Docker socket access
- Unix socket communication
- Long-running containers
- Background processes
- Root access for security hardening

⚠️ **Partial Support:**
- Auto-scaling (manual upgrade required)
- High availability (manual failover required)
- Geographic distribution (single VPS per plan)

---

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Unable to SSH into VPS

**Symptoms:**
- `Connection refused` error
- `Connection timed out` error

**Solutions:**

1. **Check Cloud Firewall:**
   - Ensure SSH (port 22) is allowed
   - Check source IP restriction (if set)

2. **Check UFW:**
   ```bash
   sudo ufw status
   sudo ufw allow 22/tcp
   ```

3. **Restart SSH Service:**
   ```bash
   sudo systemctl restart sshd
   ```

4. **Use VNC Console:**
   - Access via hPanel → VPS → Manage → VNC Console
   - Login directly from browser

#### Issue 2: Docker Service Not Starting

**Symptoms:**
- `docker: command not found`
- `Cannot connect to Docker daemon`

**Solutions:**

1. **Check Docker Service:**
   ```bash
   sudo systemctl status docker
   ```

2. **Start Docker:**
   ```bash
   sudo systemctl start docker
   sudo systemctl enable docker
   ```

3. **Check Docker Logs:**
   ```bash
   sudo journalctl -u docker -n 50
   ```

4. **Reinstall Docker:**
   ```bash
   sudo apt remove docker docker-engine docker.io containerd runc
   sudo apt update
   sudo apt install docker-ce docker-ce-cli containerd.io
   ```

#### Issue 3: Container Exits Immediately

**Symptoms:**
- `docker ps` shows no running containers
- Container status: `Exited (1)`

**Solutions:**

1. **Check Container Logs:**
   ```bash
   docker logs armorclaw-bridge
   ```

2. **Check Resource Usage:**
   ```bash
   docker stats --no-stream
   ```

3. **Verify Configuration:**
   ```bash
   docker-compose config
   ```

4. **Run Interactively:**
   ```bash
   docker run -it --rm armorclaw/agent:v1 /bin/bash
   ```

#### Issue 4: 504 Gateway Timeout

**Symptoms:**
- Browser shows `504 Gateway Timeout`
- Nginx/Caddy error logs

**Solutions:**

1. **Check Backend Service:**
   ```bash
   docker ps | grep bridge
   ```

2. **Check Service Health:**
   ```bash
   docker exec armorclaw-bridge /health.sh
   ```

3. **Increase Timeout:**
   ```nginx
   # In nginx.conf
   proxy_read_timeout 300;
   proxy_connect_timeout 300;
   ```

4. **Restart Services:**
   ```bash
   docker-compose restart
   ```

#### Issue 5: High CPU Usage

**Symptoms:**
- hPanel shows CPU > 90%
- VPS slow to respond

**Solutions:**

1. **Identify Process:**
   ```bash
   top
   # Or
   htop
   ```

2. **Check Docker Stats:**
   ```bash
   docker stats
   ```

3. **Restart Heavy Containers:**
   ```bash
   docker restart armorclaw-bridge
   ```

4. **Upgrade VPS Plan:**
   - If consistently high CPU, upgrade to higher tier

#### Issue 6: Out of Memory

**Symptoms:**
- OOM killer log entries
- Containers terminated unexpectedly

**Solutions:**

1. **Check Memory Usage:**
   ```bash
   free -h
   docker stats --no-stream
   ```

2. **Add Swap Space:**
   ```bash
   # Create 2GB swap file
   sudo fallocate -l 2G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile

   # Make permanent
   echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
   ```

3. **Limit Container Memory:**
   ```yaml
   # In docker-compose.yml
   services:
     bridge:
       deploy:
         resources:
           limits:
             memory: 512M
   ```

4. **Upgrade VPS Plan:**
   - If consistently out of memory, upgrade to higher RAM plan

#### Issue 7: Disk Space Full

**Symptoms:**
- `No space left on device` error
- Containers fail to start

**Solutions:**

1. **Check Disk Usage:**
   ```bash
   df -h
   du -sh /var/lib/docker
   ```

2. **Clean Docker:**
   ```bash
   docker system prune -a --volumes
   ```

3. **Clean Old Logs:**
   ```bash
   sudo journalctl --vacuum-time=7d
   ```

4. **Upgrade Storage:**
   - Add additional storage via hPanel

### Getting Help

**Hostinger Resources:**
- **Knowledge Base:** https://support.hostinger.com
- **Community Forum:** https://community.hostinger.com
- **Live Chat:** Available 24/7 in hPanel
- **Email:** support@hostinger.com

**ArmorClaw Resources:**
- **Documentation:** https://github.com/armorclaw/armorclaw
- **Issues:** https://github.com/armorclaw/armorclaw/issues
- **Troubleshooting Guide:** `docs/guides/troubleshooting.md`

---

## Security Best Practices

### 1. SSH Hardening

```bash
# Edit SSH config
sudo nano /etc/ssh/sshd_config

# Disable root login
PermitRootLogin no

# Disable password authentication (use keys only)
PasswordAuthentication no

# Change default port (optional)
Port 2222

# Restart SSH
sudo systemctl restart sshd
```

### 2. Firewall Configuration

```bash
# Default deny all incoming
sudo ufw default deny incoming

# Allow only necessary ports
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 6167/tcp  # Matrix

# Enable logging
sudo ufw logging on

# Enable firewall
sudo ufw enable
```

### 3. Fail2Ban Installation

```bash
# Install Fail2Ban
sudo apt install -y fail2ban

# Enable SSH protection
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Check status
sudo fail2ban-client status sshd
```

### 4. Automatic Security Updates

```bash
# Install unattended-upgrades
sudo apt install -y unattended-upgrades

# Configure automatic updates
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 5. Regular Backups

- Enable Hostinger backup service
- Set up automated off-site backups
- Test restore procedure regularly

### 6. Monitor Logs

```bash
# Monitor auth logs
sudo tail -f /var/log/auth.log

# Monitor system logs
sudo journalctl -f
```

### 7. Keep Software Updated

```bash
# Weekly updates
sudo apt update && sudo apt upgrade -y
```

---

## Quick Reference

### Essential Commands

```bash
# SSH into VPS
ssh armorclaw@your-vps-ip

# Start ArmorClaw
cd ~/armorclaw && docker-compose up -d

# Stop ArmorClaw
docker-compose down

# View logs
docker-compose logs -f

# Check status
docker ps

# Restart service
docker-compose restart bridge

# Update ArmorClaw
cd ~/armorclaw && git pull && docker-compose up -d --build
```

### Useful Paths

```
ArmorClaw Project:  /home/armorclaw/armorclaw
Bridge Binary:      /home/armorclaw/armorclaw/bridge/build/armorclaw-bridge
Docker Compose:     /home/armorclaw/armorclaw/docker-compose.yml
Configuration:      /home/armorclaw/armorclaw/configs/
Backups:           /home/armorclaw/backups/
```

---

## Conclusion

Hostinger VPS provides an excellent platform for ArmorClaw deployment with:

✅ Budget-friendly pricing (from $1.99/month promotional)
✅ Full Docker control with Docker Manager
✅ Dedicated resources for consistent performance
✅ AI-powered log analysis
✅ Global data center options

**Next Steps:**
1. Complete deployment using this guide
2. Configure domain and SSL
3. Set up monitoring and backups
4. Test ArmorClaw functionality
5. Monitor resource usage and scale as needed

**Related Documentation:**
- [Troubleshooting Guide](docs/guides/troubleshooting.md)
- [Configuration Guide](docs/guides/configuration.md)
- [Element X Quick Start](docs/guides/element-x-quickstart.md)
- [Hostinger Docker Deployment](docs/guides/hostinger-docker-deployment.md) - Docker Manager focus

---

**Document Last Updated:** 2026-02-07
**Hostinger VPS Version:** Based on 2026 pricing and features
**ArmorClaw Version:** 1.2.0
