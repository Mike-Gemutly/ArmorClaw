# ArmorClaw Deployment: Build .tar and Transfer to Hostinger VPS

> **Last Updated:** 2026-02-07
> **Purpose:** Deploy ArmorClaw to Hostinger VPS via tarball transfer
> **Time:** 10-15 minutes (with automated script)

---

## Quick Start: Automated Deployment ⭐

**Use the automated deployment script for the fastest, easiest deployment:**

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

**The vps-deploy.sh script handles everything:**
- ✅ Pre-flight checks (disk space, memory, ports)
- ✅ Tarball verification
- ✅ Docker installation (if needed)
- ✅ File extraction with line ending fixes
- ✅ Interactive configuration
- ✅ Automated deployment

**For detailed manual steps, see below.**

---

## Overview

This guide walks you through:
1. Building a ArmorClaw deployment tarball
2. Transferring it to your Hostinger VPS
3. Extracting and deploying on the VPS

---

## Part 1: Build ArmorClaw Tarball

### Option 1: Full Deployment Tarball (Recommended)

Includes everything needed for production deployment:

```bash
# Navigate to ArmorClaw directory
cd armorclaw

# Create deployment tarball
tar -czf armorclaw-deploy.tar.gz \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='*.log' \
    --exclude='.env' \
    --exclude='bridge/build' \
    --exclude='container/venv' \
    .

# Verify contents
tar -tvf armorclaw-deploy.tar.gz | head -20
```

**What's included:**
- ✅ Complete source code
- ✅ Docker configurations
- ✅ Deployment scripts
- ✅ Documentation
- ✅ Configuration templates
- ❌ Build artifacts (rebuild on server)
- ❌ Git history (smaller size)
- ❌ Sensitive data (.env files)

**Size:** ~2-3 MB (compressed)

### Option 2: Minimal Production Tarball

Only production-ready artifacts:

```bash
# Build bridge first (on local machine)
cd armorclaw/bridge
go build -o build/armorclaw-bridge ./cmd/bridge

# Build container image (optional, can rebuild on server)
cd ..
docker build -t armorclaw/agent:v1 .

# Save container image
docker save armorclaw/agent:v1 | gzip > armorclaw-agent.tar.gz

# Create minimal tarball
tar -czf armorclaw-production.tar.gz \
    bridge/build/armorclaw-bridge \
    bridge/config.example.toml \
    docker-compose-stack.yml \
    Dockerfile \
    configs/ \
    deploy/ \
    docs/guides/element-x-quickstart.md \
    LICENSE \
    README.md

# Verify
tar -tvf armorclaw-production.tar.gz
```

**What's included:**
- ✅ Pre-built bridge binary
- ✅ Container image
- ✅ Docker Compose stack
- ✅ Essential configs
- ✅ Deployment scripts
- ✅ Documentation

**Size:** ~100-150 MB (mostly container image)

### Option 3: Source-Only Tarball

For building everything from source on the VPS:

```bash
cd armorclaw

# Create source tarball (no build artifacts)
tar -czf armorclaw-source.tar.gz \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='*.log' \
    --exclude='.env' \
    --exclude='bridge/build' \
    --exclude='container/venv' \
    --exclude='*.tar.gz' \
    .

# Verify
tar -tvf armorclaw-source.tar.gz | head -20
```

**Pros/Cons:**
- ✅ Smallest size (~2 MB)
- ✅ Can rebuild for VPS architecture
- ❌ Requires Go, Docker on VPS
- ❌ Longer deployment time

---

## Part 2: Prepare VPS Before Transfer

### Step 1: Gather VPS Credentials

From Hostinger hPanel, note:
- **VPS IP Address:** (e.g., 123.45.67.89)
- **SSH Port:** Usually `22` or `2222`
- **Root Password:** (from Hostinger email or hPanel)
- **SSH Username:** Usually `root`

### Step 2: Test SSH Connection

```bash
# Test connection
ssh -p 2222 root@your-vps-ip

# If successful, exit
exit
```

### Step 3: Check VPS Requirements

```bash
# After SSH connect, check system
ssh -p 2222 root@your-vps-ip "uname -a && df -h && free -h"
```

**Minimum Requirements:**
- OS: Ubuntu 20.04+ or Debian 11+
- RAM: 2 GB minimum (4 GB recommended)
- Disk: 10 GB free space
- CPU: 1 core minimum (2+ recommended)

---

## Part 3: Transfer Tarball to VPS

### Method 1: SCP (Linux/macOS) ⭐ Recommended

```bash
# Navigate to ArmorClaw directory
cd armorclaw

# Transfer tarball to VPS
scp -P 2222 armorclaw-deploy.tar.gz root@your-vps-ip:/root/

# Show progress for large files
scp -P 2222 -v armorclaw-deploy.tar.gz root@your-vps-ip:/root/
```

**For production tarball (larger):**
```bash
# Use rsync for better progress and resumability
rsync -avzP -e 'ssh -p 2222' \
    armorclaw-deploy.tar.gz \
    root@your-vps-ip:/root/
```

### Method 2: WinSCP (Windows)

1. **Download WinSCP:** https://winscp.net/

2. **Configure connection:**
   - Host name: `your-vps-ip`
   - Port: `2222` (or `22`)
   - User name: `root`
   - Password: (your VPS password)

3. **Transfer:**
   - Navigate to local `armorclaw-deploy.tar.gz`
   - Navigate to remote `/root/`
   - Drag and drop or F5 to upload

4. **Verify:**
   - Check transfer completed successfully
   - File size matches local

### Method 3: Cloud Storage (For Slow Connections)

**Step 1: Upload to cloud**
- Upload `armorclaw-deploy.tar.gz` to Google Drive, Dropbox, or Mega.nz
- Get shareable download link

**Step 2: Download from VPS**
```bash
# SSH into VPS
ssh -p 2222 root@your-vps-ip

# Install wget
apt update && apt install wget -y

# Download from cloud
wget https://your-cloud-link.com/armorclaw-deploy.tar.gz

# Verify download
ls -lh armorclaw-deploy.tar.gz
```

### Method 4: Direct Upload to hPanel

1. **Login to Hostinger hPanel**
2. **Navigate to:** Files → File Manager
3. **Upload:** `armorclaw-deploy.tar.gz`
4. **Move to deployment directory:**
   - Use file manager to move to `/root/` or `/opt/`

**Note:** hPanel upload may have size limits and is slower.

---

## Part 4: Extract and Deploy on VPS

### Step 1: SSH into VPS

```bash
ssh -p 2222 root@your-vps-ip
```

### Step 2: Verify Transfer

```bash
# Check file exists and size
ls -lh /root/armorclaw-deploy.tar.gz

# Verify tar integrity
tar -tvf /root/armorclaw-deploy.tar.gz | head -20

# Check disk space
df -h
```

### Step 3: Extract Tarball

```bash
# Create deployment directory
mkdir -p /opt/armorclaw
cd /opt/armorclaw

# Extract tarball
tar -xzf /root/armorclaw-deploy.tar.gz

# Verify extraction
ls -la
```

**Expected output:**
```
total 52
drwxr-xr-x 6 root root 4096 Feb  7 12:00 .
drwxr-xr-x 3 root root 4096 Feb  7 12:00 ..
drwxr-xr-x 2 root root 4096 Feb  7 12:00 bridge
drwxr-xr-x 2 root root 4096 Feb  7 12:00 configs
drwxr-xr-x 2 root root 4096 Feb  7 12:00 deploy
drwxr-xr-x 2 root root 4096 Feb  7 12:00 docs
-rw-r--r-- 1 root root 1045 Feb  7 12:00 README.md
```

### Step 4: Install Dependencies

```bash
# Update package lists
apt update

# Install Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
fi

# Install Docker Compose
apt install docker-compose -y

# Install Go (for building bridge)
if ! command -v go &> /dev/null; then
    echo "Installing Go 1.21..."
    wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# Verify installations
docker --version
docker-compose --version
go version
```

### Step 5: Build ArmorClaw Bridge

```bash
# Navigate to bridge directory
cd /opt/armorclaw/bridge

# Build bridge binary
go build -o build/armorclaw-bridge ./cmd/bridge

# Verify build
./build/armorclaw-bridge version
```

**Expected output:**
```
ArmorClaw Bridge v1.0.0
```

### Step 6: Build Container Image

```bash
# Navigate back to ArmorClaw root
cd /opt/armorclaw

# Build container image
docker build -t armorclaw/agent:v1 .

# Verify image
docker images | grep armorclaw
```

**Expected output:**
```
armorclaw/agent   v1   <image-id>   2 minutes ago   393MB
```

### Step 7: Deploy Services

**Option A: Docker Compose Stack (Recommended)**

```bash
# Navigate to ArmorClaw directory
cd /opt/armorclaw

# Create .env file for configuration
cat > .env <<EOF
MATRIX_DOMAIN=your-domain.com
MATRIX_ADMIN_USER=admin
MATRIX_ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
MATRIX_BRIDGE_USER=bridge
MATRIX_BRIDGE_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
ROOM_NAME=ArmorClaw Agents
ROOM_ALIAS=agents
EOF

# Start services
docker-compose -f docker-compose-stack.yml up -d

# Verify services
docker-compose -f docker-compose-stack.yml ps
```

**Option B: Manual Setup**

```bash
# Initialize bridge config
cd /opt/armorclaw/bridge
./build/armorclaw-bridge init

# Add API key
./build/armorclaw-bridge add-key --provider openai --token "sk-proj-..."

# Start bridge
./build/armorclaw-bridge &
```

### Step 8: Verify Deployment

```bash
# Check bridge status
cd /opt/armorclaw/bridge
./build/armorclaw-bridge status

# Check health
echo '{"jsonrpc":"2.0","method":"health","id":1}' | nc -U /run/armorclaw/bridge.sock

# Check containers
docker ps | grep armorclaw
```

---

## Part 5: Post-Deployment Configuration

### Configure DNS (for Matrix/Element X)

1. **In Hostinger hPanel:**
   - Go to Domains → DNS Zone
   - Add A records:
     ```
     matrix.yourdomain.com.  A  your-vps-ip
     chat.yourdomain.com.    A  your-vps-ip
     ```

2. **Verify DNS propagation:**
   ```bash
   dig +short matrix.yourdomain.com
   ```

### Configure Firewall

```bash
# Allow SSH
ufw allow 2222/tcp

# Allow HTTP/HTTPS
ufw allow 80/tcp
ufw allow 443/tcp

# Allow Matrix ports
ufw allow 8448/tcp
ufw allow 6167/tcp

# Enable firewall
ufw enable

# Check status
ufw status
```

### Set up SSL (for Matrix)

```bash
# Caddy in docker-compose-stack.yml will handle SSL automatically
# Just ensure DNS is configured correctly

# Verify SSL
curl -I https://matrix.yourdomain.com
```

### Test Element X Connection

1. **Download Element X app** on your device
2. **Login with:**
   - Homeserver: `https://matrix.yourdomain.com`
   - Username: `admin`
   - Password: (from .env file)

3. **Join room:** `#agents:yourdomain.com`

4. **Test commands:**
   ```
   /ping
   /status
   ```

---

## Troubleshooting

### Issue: "Permission denied on socket"

```bash
# Create socket directory with correct permissions
mkdir -p /run/armorclaw
chown $USER:$USER /run/armorclaw
chmod 770 /run/armorclaw
```

### Issue: "Bridge won't start"

```bash
# Check logs
cd /opt/armorclaw/bridge
./build/armorclaw-bridge -log-level=debug

# Check if Docker is running
docker ps
```

### Issue: "Docker build failed"

```bash
# Check Docker is running
systemctl status docker

# Check disk space
df -h

# Check available RAM
free -h
```

### Issue: "Transfer incomplete"

```bash
# Verify file integrity
tar -tvf armorclaw-deploy.tar.gz

# If corrupted, re-transfer using rsync
rsync -avzP --partial -e 'ssh -p 2222' \
    armorclaw-deploy.tar.gz \
    root@your-vps-ip:/root/
```

---

## Quick Reference Commands

### From local machine:

```bash
# 1. Build tarball
cd armorclaw
tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' .

# 2. Transfer to VPS
scp -P 2222 armorclaw-deploy.tar.gz root@YOUR_VPS_IP:/root/

# 3. SSH into VPS
ssh -p 2222 root@YOUR_VPS_IP

# 4. Extract and deploy
mkdir -p /opt/armorclaw
cd /opt/armorclaw
tar -xzf /root/armorclaw-deploy.tar.gz
./deploy/setup-wizard.sh
```

### All-in-one deployment script:

```bash
# Run from local machine
cd armorclaw
tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' . && \
scp -P 2222 armorclaw-deploy.tar.gz root@YOUR_VPS_IP:/root/ && \
ssh -p 2222 root@YOUR_VPS_IP "mkdir -p /opt/armorclaw && cd /opt/armorclaw && tar -xzf /root/armorclaw-deploy.tar.gz && ls -la"
```

---

## Alternative: Direct Git Clone on VPS

If VPS has internet access, you can skip tarball transfer:

```bash
# SSH into VPS
ssh -p 2222 root@your-vps-ip

# Install Git
apt update && apt install git -y

# Clone ArmorClaw repository
cd /opt
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Continue from Step 4 above
```

---

## Summary Checklist

- [ ] Build tarball on local machine
- [ ] Test tar integrity with `tar -tvf`
- [ ] Transfer tarball to VPS
- [ ] Verify transfer on VPS
- [ ] Extract tarball
- [ ] Install dependencies (Docker, Go)
- [ ] Build bridge binary
- [ ] Build container image
- [ ] Deploy services
- [ ] Configure DNS
- [ ] Configure firewall
- [ ] Test Element X connection
- [ ] Run verification tests

---

**Next Steps:**
- See [Element X Quick Start](docs/guides/element-x-quickstart.md)
- See [Infrastructure Deployment Guide](docs/guides/2026-02-05-infrastructure-deployment-guide.md)
- See [Troubleshooting Guide](docs/guides/troubleshooting.md)
