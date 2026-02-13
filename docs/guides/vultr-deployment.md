# Vultr Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Vultr (https://www.vultr.com)
> **Best For:** GPU workloads, cost-effective VPS, global infrastructure
> **Difficulty Level:** Intermediate
> **Estimated Time:** 30-45 minutes

---

## Executive Summary

**Vultr** is a high-performance cloud compute platform with 32 data centers worldwide. Known for competitive pricing and GPU instances, Vultr offers a simple, predictable pricing model.

### Why Vultr for ArmorClaw?

✅ **GPU Instances** - A100, H100, H200 available
✅ **Competitive Pricing** - From $2.50/month
✅ **Global Network** - 32 data centers
✅ **Simple Pricing** - Flat monthly rates, no surprises
✅ **Docker Support** - Pre-installed Docker apps available
✅ **Object Storage** - S3-compatible storage
✅ **One-Click Apps** - Docker, Portainer, etc.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Vultr Cloud Infrastructure               │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Region:    │  │   Region:    │  │   Region:    │      │
│  │   New Jersey │  │   London     │  │   Tokyo      │      │
│  │  (Regular)   │  │  (Regular)   │  │  (GPU)       │      │
│  │              │  │              │  │              │      │
│  │  ┌────────┐  │  │  ┌────────┐  │  │  ┌────────┐  │      │
│  │  │ VPS    │  │  │  │ VPS    │  │  │  │ GPU    │  │      │
│  │  │ Docker │  │  │  │ Docker │  │  │  │ A100   │  │      │
│  │  └────────┘  │  │  └────────┘  │  │  └────────┘  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                          │                                  │
│  ┌───────────────────────┴───────────────────────────────┐   │
│  │              Vultr Object Storage (S3-compatible)       │   │
│  │              Vultr Managed Databases (PostgreSQL)        │   │
│  │              Load Balancers                              │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Prerequisites

- Vultr account (sign up at https://www.vultr.com)
- API key (from Settings → API)
- SSH client
- Basic Linux knowledge

### 1. Install Vultr CLI

**macOS:**
```bash
brew install vultr-cli
```

**Linux:**
```bash
curl -L https://github.com/vultr/vultr-cli/releases/latest/download/vultr-cli-linux-amd64.tar.gz | tar xz
sudo mv vultr-cli /usr/local/bin/vultr
```

**Windows:**
```bash
scoop install vultr-cli
```

### 2. Configure API Key

```bash
export VULTR_API_KEY="your_api_key_here"

# Or configure
vultr configure
```

### 3. Create Instance

```bash
vultr instance create \
  --region "ewr" \
  --plan "vc2-1c-1gb" \
  --os 1743 \
  --hostname "armorclaw" \
  --tag "ArmorClaw"
```

### 4. SSH and Deploy

```bash
# Get instance IP
INSTANCE_IP=$(vultr instance list --format json | jq -r '.[0].main_ip')

# SSH in
ssh root@$INSTANCE_IP

# Install Docker
curl -fsSL https://get.docker.com | sh

# Deploy ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

## Detailed Deployment

### 1. Instance Plans

**Regular Cloud Compute:**

| Plan | vCPU | RAM | Storage | Bandwidth | Monthly Cost |
|------|------|-----|---------|-----------|--------------|
| **vc2-1c-0.5gb** | 1 | 0.5 GB | 10 GB | 0.5 TB | $2.50/month |
| **vc2-1c-1gb** | 1 | 1 GB | 25 GB | 1 TB | $5.00/month |
| **vc2-1c-2gb** | 1 | 2 GB | 55 GB | 2 TB | $10.00/month |
| **vc2-2c-4gb** ⭐ | 2 | 4 GB | 80 GB | 3 TB | $20.00/month |
| **vc2-4c-8gb** | 4 | 8 GB | 160 GB | 5 TB | $40.00/month |

**High Frequency Compute:**

| Plan | vCPU | RAM | Storage | Bandwidth | Monthly Cost |
|------|------|-----|---------|-----------|--------------|
| **vhf-1c-2gb** | 1 | 2 GB | 32 GB | 2 TB | $12.00/month |
| **vhf-2c-4gb** | 2 | 4 GB | 64 GB | 3 TB | $24.00/month |

**Cloud GPU (NVIDIA A100):**

| Plan | GPU | vCPU | RAM | Storage | Hourly Cost |
|------|-----|------|-----|---------|-------------|
| **gpu-a100-40gb** | 1x A100 | 8 | 60 GB | 350 GB | ~$1.00/hour |
| **gpu-a100-s4** | 4x A100 | 60 | 480 GB | 3 TB | ~$4.00/hour |

**Cloud GPU (NVIDIA H100):**

| Plan | GPU | vCPU | RAM | Storage | Hourly Cost |
|------|-----|------|-----|---------|-------------|
| **gpu-h100-80gb** | 1x H100 | 16 | 120 GB | 500 GB | ~$2.50/hour |

### 2. Create Instance via CLI

**Regular Instance:**
```bash
vultr instance create \
  --region "ewr" \
  --plan "vc2-2c-4gb" \
  --os 1743 \
  --hostname "armorclaw-bridge" \
  --tag "ArmorClaw" \
  --ssh-keys "ssh_key_id" \
  --enable-ipv6 \
  --attach-private-network "pn_id"
```

**GPU Instance:**
```bash
vultr instance create \
  --region "ewr" \
  --plan "gpu-a100-40gb" \
  --os 387 \
  --hostname "armorclaw-gpu" \
  --tag "ArmorClaw GPU"
```

### 3. Server Setup

**SSH into Instance:**
```bash
ssh root@your_instance_ip
```

**Update System:**
```bash
apt update && apt upgrade -y
```

**Install Docker:**
```bash
curl -fsSL https://get.docker.com | sh
usermod -aG docker ubuntu
```

**Install Docker Compose:**
```bash
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

### 4. Deploy ArmorClaw

**Clone Repository:**
```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
```

**Build and Deploy:**
```bash
docker-compose up -d
```

**Verify Deployment:**
```bash
docker ps
docker logs -f armorclaw-bridge
```

### 5. Networking

**Configure Firewall:**
```bash
# Via Vultr web panel: Settings → Firewall
# Or via CLI:
vultr firewall create \
  --description "ArmorClaw Firewall"
```

**Firewall Rules:**
```bash
vultr firewall-rule create FIREWALL_ID \
  --protocol "tcp" \
  --port "22" \
  --ip "your_ip/32"

vultr firewall-rule create FIREWALL_ID \
  --protocol "tcp" \
  --port "80" \
  --ip "0.0.0.0/0"

vultr firewall-rule create FIREWALL_ID \
  --protocol "tcp" \
  --port "443" \
  --ip "0.0.0.0/0"
```

---

## GPU Deployment

### 1. Create GPU Instance

**A100 Instance:**
```bash
vultr instance create \
  --region "ewr" \
  --plan "gpu-a100-40gb" \
  --os 387 \
  --hostname "armorclaw-gpu"
```

### 2. Install NVIDIA Drivers

```bash
# Add NVIDIA repository
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | tee /etc/apt/sources.list.d/nvidia-docker.list

# Install drivers
apt update
apt install -y nvidia-driver-535 nvidia-utils-535

# Verify
nvidia-smi
```

### 3. Install NVIDIA Container Toolkit

```bash
curl -s -L https://nvidia.github.io/nvidia-container-runtime/gpgkey | apt-key add -
curl -s -L https://nvidia.github.io/nvidia-container-runtime/$distribution/nvidia-container-runtime.list | tee /etc/apt/sources.list.d/nvidia-container-runtime.list

apt update
apt install -y nvidia-container-toolkit

# Restart Docker
systemctl restart docker
```

### 4. Run GPU Container

```bash
docker run --gpus all -it --rm armorclaw-gpu:latest
```

---

## Object Storage

### 1. Create Object Storage Bucket

```bash
vultr object-storage create \
  --label "armorclaw-storage" \
  --region "ewr" \
  --cluster-id "cluster_id"
```

### 2. Configure S3-Compatible Access

**Install AWS CLI:**
```bash
pip install awscli
```

**Configure:**
```bash
aws configure --profile vultr
# Enter access key and secret from Vultr panel
# Region: us-east-1 (or your cluster region)
# Output: json

# Add to ~/.aws/config:
[profile vultr]
region = us-east-1
output = json

# Add to ~/.aws/credentials:
[vultr]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

**Upload Files:**
```bash
aws --profile vultr --endpoint-url https://ewr1.vultrobjects.com \
  s3 mb s3://armorclaw-backups

aws --profile vultr --endpoint-url https://ewr1.vultrobjects.com \
  s3 cp backup.tar.gz s3://armorclaw-backups/
```

---

## Pricing Details

### Regular Instances (2026)

| Plan | vCPU | RAM | Storage | Bandwidth | Monthly Cost |
|------|------|-----|---------|-----------|--------------|
| **vc2-1c-0.5gb** | 1 | 0.5 GB | 10 GB | 0.5 TB | $2.50 |
| **vc2-1c-1gb** | 1 | 1 GB | 25 GB | 1 TB | $5.00 |
| **vc2-1c-2gb** | 1 | 2 GB | 55 GB | 2 TB | $10.00 |
| **vc2-2c-4gb** ⭐ | 2 | 4 GB | 80 GB | 3 TB | $20.00 |
| **vc2-4c-8gb** | 4 | 8 GB | 160 GB | 5 TB | $40.00 |

### GPU Instances

| Plan | GPU | Hourly Cost | Monthly Cost (24/7) |
|------|-----|-------------|---------------------|
| **A100 (40GB)** | 1x A100 | ~$1.00 | ~$720 |
| **H100 (80GB)** | 1x H100 | ~$2.50 | ~$1,800 |

### Additional Services

| Service | Cost |
|---------|------|
| **Object Storage** | $0.01/GB/month |
| **Load Balancer** | $0.015/hour (~$10/month) |
| **Managed PostgreSQL** | From $15/month |

---

## Limitations

### Platform Limitations

| Limitation | Details |
|------------|---------|
| **No Auto-Scaling** | Manual scaling required |
| **Bandwidth Caps** | 0.5-5 TB/month included, overage charged |
| **GPU Availability** | Limited regions, can sell out |
| **No Native Kubernetes** | Use third-party or self-host |

### ArmorClaw Considerations

✅ **Fully Supported:**
- Docker container deployment
- GPU compute for AI workloads
- Long-running containers
- Root access for security hardening
- Object storage for backups

⚠️ **Considerations:**
- No auto-scaling (manual intervention required)
- GPU instances can be expensive for 24/7 operation
- Limited managed services

---

## Quick Reference

### Essential Commands

```bash
# Create instance
vultr instance create --region ewr --plan vc2-2c-4gb --os 1743

# List instances
vultr instance list

# Reboot instance
vultr instance reboot INSTANCE_ID

# Destroy instance
vultr instance delete INSTANCE_ID

# Create firewall
vultr firewall create --description "ArmorClaw"

# Create snapshot
vultr snapshot create INSTANCE_ID --description "Backup"
```

---

## Conclusion

Vultr provides excellent value for ArmorClaw deployments with:

✅ **Competitive Pricing** - From $2.50/month
✅ **GPU Instances** - A100, H100 available
✅ **Global Network** - 32 data centers
✅ **Simple Pricing** - Flat monthly rates
✅ **Docker Support** - Pre-installed apps available

**Best For:**
- GPU-powered AI agent workloads
- Cost-sensitive deployments
- Global presence requirements
- Users who prefer simple, predictable pricing

**Next Steps:**
1. Create instance (regular or GPU)
2. Install Docker and deploy ArmorClaw
3. Configure firewall and networking
4. Set up object storage for backups
5. For GPU: Install NVIDIA drivers and toolkit

**Related Documentation:**
- [Vultr Documentation](https://www.vultr.com/docs/)
- [Object Storage Guide](https://www.vultr.com/docs/vultr-object-storage/)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)

---

**Document Last Updated:** 2026-02-07
**Vultr Version:** Based on 2026 pricing and features
**ArmorClaw Version:** 1.2.0
