# Linode (Akamai Connected Cloud) Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Linode/Akamai (https://www.linode.com)
> **Best For:** VPS flexibility, Akamai integration, GPU instances
> **Difficulty Level:** Intermediate
> **Estimated Time:** 30-45 minutes

---

## Executive Summary

**Linode** (now part of Akamai Connected Cloud) provides high-performance VPS hosting with competitive pricing and excellent performance. With the Akamai integration, Linode offers enhanced global network capabilities.

### Why Linode for ArmorClaw?

✅ **Competitive Pricing** - From $5/month
✅ **Akamai Integration** - Enhanced global network
✅ **GPU Instances** - Available for AI workloads
✅ **Simple Pricing** - Flat monthly rates
✅ **Full Root Access** - Complete control
✅ **Object Storage** - S3-compatible storage

### Pricing (2026)

| Plan | vCPU | RAM | Storage | Transfer | Monthly Cost |
|------|------|-----|---------|----------|--------------|
| **Nanode 1GB** | 1 | 1 GB | 25 GB | 1 TB | $5 |
| **Dedicated 4GB** ⭐ | 2 | 4 GB | 80 GB | 4 TB | $20 |
| **Dedicated 8GB** | 4 | 8 GB | 160 GB | 8 TB | $40 |
| **Dedicated 16GB** | 6 | 16 GB | 320 GB | 16 TB | $80 |

**GPU Instances:**
- **NVIDIA A40:** From $1.10/hour (~$800/month)
- **NVIDIA A100:** From ~$2.00/hour

---

## Quick Start

### 1. Install Linode CLI

```bash
pip3 install linode-cli
```

### 2. Configure API Key

```bash
linode-cli configure
# Enter API key from: https://cloud.linode.com/profile/tokens
```

### 3. Create Instance

```bash
linode-cli linodes create \
  --type g6-dedicated-4 \
  --region us-east \
  --image linode/ubuntu22.04 \
  --root_pass secure_password \
  --label armorclaw-bridge
```

### 4. SSH and Deploy

```bash
ssh root@your_linode_ip

# Install Docker
curl -fsSL https://get.docker.com | sh

# Deploy ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

## Detailed Deployment

### 1. Choose Instance Type

**Dedicated CPU (Recommended):**
- More consistent performance
- Better for production
- No CPU contention

**Shared CPU:**
- Cheaper
- Good for development
- CPU contention possible

### 2. Create Instance via CLI

```bash
linode-cli linodes create \
  --type g6-dedicated-4 \
  --region us-east \
  --image linode/ubuntu22.04 \
  --root_pass STRONG_PASSWORD_HERE \
  --label armorclaw-bridge \
  --group armorclaw \
  --tags armorclaw,production
```

### 3. Server Setup

**Update System:**
```bash
apt update && apt upgrade -y
```

**Install Docker:**
```bash
curl -fsSL https://get.docker.com | sh
usermod -aG docker $USER
```

**Install Docker Compose:**
```bash
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

### 4. Deploy ArmorClaw

```bash
# Clone repository
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# Build and deploy
docker-compose up -d

# Verify
docker ps
docker logs -f armorclaw-bridge
```

### 5. Configure Firewall

**Via Cloud Manager:**
1. Navigate to **Networking** → **Firewalls**
2. Create firewall: `armorclaw-fw`
3. Add rules:
   - SSH (22) from your IP
   - HTTP (80) from all
   - HTTPS (443) from all
   - Matrix (6167) from all (if needed)
4. Apply to Linode

**Via UFW:**
```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

---

## Object Storage

### 1. Create Bucket

```bash
linode-cli object-storage buckets create \
  --cluster us-east-1 \
  --label armorclaw-backups
```

### 2. Configure Access

**Create Access Key:**
```bash
linode-cli object-storage keys create \
  --label armorclaw-key
```

**Use with AWS CLI:**
```bash
aws configure --profile linode
# Enter access key and secret
# Region: us-east-1
# Output: json

# Add to ~/.aws/config:
[profile linode]
region = us-east-1

# Add to ~/.aws/credentials:
[linode]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

**Upload Files:**
```bash
aws --profile linode --endpoint-url https://us-east-1.linodeobjects.com \
  s3 mb s3://armorclaw-backups

aws --profile linode --endpoint-url https://us-east-1.linodeobjects.com \
  s3 cp backup.tar.gz s3://armorclaw-backups/
```

---

## GPU Deployment

### 1. Create GPU Instance

```bash
linode-cli linodes create \
  --type g6-gpu-1 \
  --region us-east \
  --image linode/ubuntu22.04 \
  --label armorclaw-gpu
```

### 2. Install NVIDIA Drivers

```bash
# Add NVIDIA repository
distribution=$(. /etc/os-release;echo $ID$VERSION_ID | tr -d '.')
wget https://developer.download.nvidia.com/compute/cuda/repos/$distribution/x86_64/cuda-keyring_1.1-1_all.deb
dpkg -i cuda-keyring_1.1-1_all.deb
apt-get update

# Install drivers
apt-get -y install cuda-drivers

# Verify
nvidia-smi
```

### 3. Install NVIDIA Container Toolkit

```bash
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | apt-key add -
curl -s -L https://nvidia.github.com/nvidia-docker/$distribution/nvidia-docker.list | tee /etc/apt/sources.list.d/nvidia-docker.list

apt-get update
apt-get install -y nvidia-container-toolkit
systemctl restart docker
```

### 4. Run GPU Container

```bash
docker run --gpus all -it --rm armorclaw-gpu:latest
```

---

## Managed Databases

### 1. Create Managed PostgreSQL

**Via Cloud Manager:**
1. Navigate to **Databases** → **Create Database**
2. Select **PostgreSQL**
3. Select plan (from $5/month)
4. Select region
5. Create database

**Via CLI:**
```bash
linode-cli databases create \
  --type postgresql \
  --label armorclaw-db \
  --region us-east \
  --engine postgresql 14
```

### 2. Connect to Database

**Connection string:**
```
postgresql://username:password@host:5432/database
```

**Add to ArmorClaw environment:**
```bash
export DATABASE_URL="postgresql://username:password@host:5432/database"
```

---

## Limitations

| Limitation | Details |
|------------|---------|
| **No Auto-Scaling** | Manual scaling required |
| **Transfer Caps** | 1-16 TB/month included |
| **GPU Availability** | Limited regions |
| **No Managed Kubernetes** | Use third-party |

---

## Quick Reference

```bash
# Create instance
linode-cli linodes create --type g6-dedicated-4 --region us-east --image linode/ubuntu22.04

# List instances
linode-cli linodes list

# Reboot instance
linode-cli linodes reboot LINODE_ID

# Delete instance
linode-cli linodes delete LINODE_ID

# Create backup
linode-cli linodes snapshot LINODE_ID
```

---

## Conclusion

Linode (Akamai Connected Cloud) provides excellent VPS hosting with competitive pricing and good performance.

**Best For:**
- Production VPS deployments
- GPU workloads
- Users who prefer predictable flat pricing
- Akamai network benefits

**Next Steps:**
1. Create instance (dedicated CPU recommended)
2. Install Docker and deploy ArmorClaw
3. Configure firewall
4. Set up object storage for backups
5. For GPU: Install NVIDIA drivers

**Related Documentation:**
- [Linode Docs](https://www.linode.com/docs/)
- [Vultr Deployment](docs/guides/vultr-deployment.md)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)

---

**Document Last Updated:** 2026-02-07
**Linode Version:** Based on 2026 pricing
**ArmorClaw Version:** 1.2.0
