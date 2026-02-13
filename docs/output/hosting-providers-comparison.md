# Hosting Providers Comparison for ArmorClaw

> **Date:** 2026-02-07
> **Purpose:** Comprehensive evaluation of hosting providers for ArmorClaw deployment
> **Goal:** Maximize ArmorClaw use cases by identifying all viable deployment options

---

## Executive Summary

ArmorClaw has unique hosting requirements due to its architecture:
- **Docker socket access** (for Local Bridge to manage agent containers)
- **Unix socket support** (for Bridge-to-agent IPC)
- **Long-running containers** (agents run for minutes to hours)
- **Background processes** (persistent Bridge service)
- **Low resource footprint** (target: ≤2 GB total)

### Quick Recommendations by Use Case

| Use Case | Recommended Provider | Monthly Cost | Key Advantage |
|----------|---------------------|--------------|---------------|
| **Local Development** | Docker Desktop (local) | Free | Full control, no network latency |
| **Small Production** | Hostinger VPS KVM2 | ~$4-8 | Dedicated resources, full Docker control |
| **Large Production** | AWS Fargate + EFS | ~$15-30/month | Scalable, managed, highly available |
| **Edge/Global Distribution** | Fly.io | ~$5-15/month | Global edge network, Docker support |
| **Cost-Optimized** | DigitalOcean App Platform | ~$5/month | Simple pricing, good for small deployments |
| **GPU/AI Inference** | Vultr GPU | ~$1.85/GPU/hour | Pay-per-use GPU, no commitment |
| **High Availability** | Google Cloud Run + Cloud SQL | ~$10-20/month | Multi-region, managed services |

---

## Provider Comparison Matrix

### Core Requirements Compatibility

| Provider | Docker Socket | Unix Sockets | Long-Running | Background Processes | Min Monthly Cost | Pricing Model |
|----------|--------------|--------------|--------------|----------------------|------------------|---------------|
| **Docker Desktop (Local)** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | Free | N/A |
| **Hostinger VPS** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ~$4-8 | Fixed per tier |
| **Fly.io** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ~$5-15 | Pay-per-use + free allowance |
| **Railway** | ✅ Yes | ⚠️ Limited | ✅ Yes | ✅ Yes | ~$5-20 | Usage-based (free tier: $5/month) |
| **Render** | ✅ Yes | ⚠️ Limited | ✅ Yes | ✅ Yes | Free | Usage-based (free tier available) |
| **DigitalOcean App Platform** | ✅ Yes | ⚠️ Limited | ✅ Yes | ✅ Yes | ~$5 | Fixed per tier |
| **Google Cloud Run** | ✅ Yes | ✅ Yes (Cloud SQL) | ✅ Yes (1 hour) | ✅ Yes (Jobs: 30 min) | ~$0 | Pay-per-use (free tier: 2M requests/month) |
| **AWS Fargate** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ~$9+ | Pay-per-use (Spot: ~$0.83+) |
| **Linode/Akamai** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ~$5 | Fixed per tier |
| **Vultr** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ~$2.50-6 | Fixed per tier |
| **Azure Container Instances** | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ~$0 | Pay-per-second |
| **Cloudflare Workers** | ❌ No | ❌ No | ❌ No (30s limit) | ❌ No | $5 minimum | ❌ **NOT SUITABLE** |

### Feature Comparison

| Provider | Geographic Distribution | GPU Support | Serverless | Auto-Scaling | Managed Services |
|----------|------------------------|-------------|------------|--------------|------------------|
| **Docker Desktop (Local)** | ❌ N/A | ❌ No | ❌ No | ❌ No | ❌ No |
| **Hostinger VPS** | ⚠️ Limited (8 locations) | ❌ No | ❌ No | ❌ No | ❌ No |
| **Fly.io** | ✅ Global (20+ regions) | ❌ No | ✅ Yes | ✅ Yes | ⚠️ Limited (Postgres, Redis) |
| **Railway** | ⚠️ Limited (US/EU) | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes (Postgres, Redis, Redis) |
| **Render** | ⚠️ Limited (US/EU) | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes (Postgres, Redis) |
| **DigitalOcean App Platform** | ⚠️ Limited (12 regions) | ❌ No | ⚠️ Partial | ✅ Yes | ✅ Yes (Managed DBs) |
| **Google Cloud Run** | ✅ Global (20+ regions) | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes (Cloud SQL, etc.) |
| **AWS Fargate** | ✅ Global (multiple regions) | ✅ Yes (via EKS) | ⚠️ Partial | ✅ Yes | ✅ Yes (RDS, EFS, etc.) |
| **Linode/Akamai** | ⚠️ Limited (11 regions) | ✅ Yes | ❌ No | ❌ No | ⚠️ Limited (Object Storage) |
| **Vultr** | ✅ Global (32 regions) | ✅ Yes | ❌ No | ❌ No | ⚠️ Limited (Object Storage) |
| **Azure Container Instances** | ✅ Global (multiple regions) | ✅ Yes | ⚠️ Partial | ✅ Yes | ✅ Yes (Azure SQL, etc.) |
| **Cloudflare Workers** | ✅ Global (200+ cities) | ❌ No | ✅ Yes | ✅ Yes | ⚠️ Limited (KV, Durable Objects) |

---

## Detailed Provider Profiles

### 1. Docker Desktop (Local) - Development Only

**Best For:** Local development, testing, CI/CD pipelines

**Pros:**
- ✅ Full Docker control (socket access, Unix sockets, any container config)
- ✅ Zero cost for development
- ✅ No network latency
- ✅ Complete control over all components

**Cons:**
- ❌ Not suitable for production deployment
- ❌ Requires local machine to be running
- ❌ No geographic distribution
- ❌ Manual scaling and management

**ArmorClaw Fit:** ⭐⭐⭐⭐⭐ (5/5) - Perfect for development and testing

**Cost:** Free (for development)

**Quick Start:**
```bash
# Install Docker Desktop
# Clone ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

### 2. Hostinger VPS (KVM2) - Budget Production

**Best For:** Small production deployments, cost-sensitive users, dedicated resources

**Pros:**
- ✅ Full Docker control (root access to VPS)
- ✅ Very cost-effective for small deployments
- ✅ Dedicated resources (not shared/contended)
- ✅ Simple, predictable pricing
- ✅ Good for running Matrix Conduit + Bridge + Agent

**Cons:**
- ❌ Limited geographic coverage (8 data centers)
- ❌ Manual scaling and management
- ❌ No auto-scaling or managed services
- ❌ Requires manual OS updates and security patches

**ArmorClaw Fit:** ⭐⭐⭐⭐ (4/5) - Excellent for small production deployments

**Pricing:**
- KVM2: 2 vCPU, 4 GB RAM, 80 GB SSD = ~$4-8/month
- KVM4: 4 vCPU, 8 GB RAM, 160 GB SSD = ~$8-16/month

**Memory Budget:** KVM2 (4 GB RAM) fits ArmorClaw's target (≤2 GB total)

**Quick Start:** See `docs/guides/hostinger-deployment.md`

---

### 3. Fly.io - Global Edge Distribution

**Best For:** Global deployment, edge computing, low-latency access

**Pros:**
- ✅ Docker support (deploy via `fly.toml` or Dockerfile)
- ✅ Global edge network (20+ regions worldwide)
- ✅ Auto-scaling based on demand
- ✅ Free tier allowance (3 apps, 3 GB volume)
- ✅ Simple CLI deployment
- ✅ Built-in secrets management
- ✅ Supports sidecar containers (for Matrix Conduit)

**Cons:**
- ❌ No GPU support (as of 2026)
- ⚠️ Unix socket support limited to sidecar containers
- ⚠️ Cold starts can add latency (though better than AWS Lambda)
- ❌ Managed services limited (Postgres, Redis only)

**ArmorClaw Fit:** ⭐⭐⭐⭐ (4/5) - Great for global edge deployment

**Pricing:**
- Free tier: 3 apps, 256 MB RAM, 3 GB volume (limited availability)
- Paid: ~$5-15/month for typical ArmorClaw deployment
- CPU: ~$2-4/month per vCPU (shared)
- RAM: ~$1-2/month per GB
- Volume storage: ~$0.15/GB/month

**Quick Start:**
```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Launch ArmorClaw
cd armorclaw
fly launch
```

---

### 4. Railway - Developer Experience

**Best For:** Developers who want simple deployment with good DX

**Pros:**
- ✅ Excellent developer experience (GitHub integration, automatic deploys)
- ✅ Docker support (deploy via Dockerfile or buildpacks)
- ✅ Free tier ($5 one-time credit)
- ✅ Managed services (Postgres, Redis)
- ✅ Simple pricing (pay for what you use)
- ✅ Good for small to medium deployments

**Cons:**
- ❌ Limited geographic distribution (US/EU only)
- ⚠️ Unix socket support not well-documented
- ❌ No GPU support
- ❌ Can get expensive for larger deployments (usage-based pricing)

**ArmorClaw Fit:** ⭐⭐⭐ (3/5) - Good for developers, but may get expensive

**Pricing:**
- Free tier: $5 one-time credit (not recurring)
- Estimated: ~$5-20/month for typical ArmorClaw deployment
- Pricing based on: CPU hours, RAM, network, storage

**Quick Start:**
```bash
# Connect GitHub repo to Railway
# Railway auto-detects Dockerfile and deploys
```

---

### 5. Render - Simple PaaS

**Best For:** Simple deployments, free tier for testing

**Pros:**
- ✅ Free tier for web services (512 MB RAM, 0.1 CPU)
- ✅ Docker support (deploy via Dockerfile)
- ✅ Managed services (Postgres, Redis)
- ✅ Simple pricing
- ✅ Auto-deploys from GitHub
- ✅ Good documentation

**Cons:**
- ❌ Limited geographic distribution (US/EU only)
- ⚠️ Unix socket support limited
- ❌ Free tier spins down after 15 minutes inactivity (not suitable for Bridge)
- ❌ No GPU support
- ❌ Cold starts on free tier

**ArmorClaw Fit:** ⭐⭐ (2/5) - Free tier too limited, paid tier viable but not optimal

**Pricing:**
- Free tier: Limited, spins down after 15 min (❌ not suitable for persistent Bridge)
- Paid: ~$7-25/month for typical ArmorClaw deployment
- CPU: ~$0.01/GB-hour
- RAM: ~$0.005/GB-hour

**Quick Start:**
```bash
# Connect GitHub repo to Render
# Render auto-detects Dockerfile and deploys
```

---

### 6. DigitalOcean App Platform - Simple PaaS

**Best For:** Small production deployments, DO users

**Pros:**
- ✅ Docker support (deploy via Dockerfile or container registry)
- ✅ Simple, predictable pricing ($5/month minimum)
- ✅ Managed databases (Postgres, Redis, MySQL)
- ✅ Good documentation
- ✅ 12 global data center regions
- ✅ Auto-scaling available

**Cons:**
- ❌ No GPU support
- ⚠️ Unix socket support limited to sidecar containers
- ⚠️ App Platform has limitations (e.g., 1 GiB container image size)
- ❌ More expensive than basic DO droplets

**ArmorClaw Fit:** ⭐⭐⭐ (3/5) - Good for simple deployments, but DO Droplets may be better

**Pricing:**
- Basic: $5/month (512 MB RAM, 1 vCPU)
- Standard: $12/month (1 GB RAM, 1 vCPU)
- Professional: $32/month (2 GB RAM, 1 vCPU)

**Memory Budget:** Professional tier (2 GB RAM) meets ArmorClaw's target, but leaves little headroom

**Quick Start:**
```bash
# Push image to DigitalOcean Container Registry
docker registry login registry.digitalocean.com
docker tag armorclaw-agent registry.digitalocean.com/username/armorclaw-agent
docker push registry.digitalocean.com/username/armorclaw-agent

# Deploy via App Platform dashboard or CLI
```

---

### 7. Google Cloud Run - Serverless Containers

**Best For:** Serverless deployments, variable workloads, Google Cloud users

**Pros:**
- ✅ Docker support (deploy any container)
- ✅ Serverless scaling (zero to N instances)
- ✅ Free tier (2M requests/month, 360K GB-seconds CPU time/month)
- ✅ Supports sidecar containers (for Cloud SQL with Unix sockets)
- ✅ Long-running containers (up to 1 hour for services, 30 min for Jobs)
- ✅ Excellent Google Cloud integration (Cloud SQL, Cloud Monitoring, etc.)
- ✅ 20+ global regions

**Cons:**
- ❌ Maximum 1 hour execution time (may limit long agent sessions)
- ⚠️ Unix sockets only supported for specific use cases (Cloud SQL proxy)
- ❌ No GPU support (use Vertex AI for GPU workloads)
- ❌ Cold starts (though relatively fast, ~1-2 seconds)
- ❌ Request-based pricing may be expensive for persistent Bridge

**ArmorClaw Fit:** ⭐⭐⭐ (3/5) - Good for serverless, but 1-hour limit may be restrictive

**Pricing:**
- Free tier: 2M requests/month, 360K GB-seconds CPU time/month
- CPU: ~$0.000009/GB-second (after free tier)
- Memory: ~$0.0000004/GB-second (after free tier)
- Requests: $0.40 per million requests (after free tier)
- Estimated: ~$10-30/month for typical ArmorClaw deployment

**Memory Budget:** No explicit limit, but memory-based pricing adds up quickly

**Quick Start:**
```bash
# Deploy to Cloud Run
gcloud run deploy armorclaw-bridge \
  --image gcr.io/PROJECT_ID/armorclaw-bridge \
  --platform managed \
  --region us-central1

# Deploy Cloud Run Job for agent
gcloud run jobs create armorclaw-agent \
  --image gcr.io/PROJECT_ID/armorclaw-agent \
  --region us-central1
```

---

### 8. AWS Fargate - Serverless Containers (AWS)

**Best For:** AWS users, enterprise deployments, highly available setups

**Pros:**
- ✅ Docker support (via ECS/EKS)
- ✅ Serverless scaling (zero to N instances)
- ✅ No VM management
- ✅ Excellent AWS integration (RDS, EFS, CloudWatch, etc.)
- ✅ Long-running containers (no time limit)
- ✅ Fargate Spot for cost savings (up to 70% discount)
- ✅ GPU support (via Fargate for GPU)
- ✅ Multiple availability zones (high availability)

**Cons:**
- ❌ AWS complexity (steep learning curve)
- ❌ Minimum cost is higher than alternatives ($9/month minimum)
- ⚠️ Unix socket support limited to task networking (not across tasks)
- ❌ Can get expensive for always-on Bridge
- ❌ Requires additional AWS services for full functionality (ALB, EFS, etc.)

**ArmorClaw Fit:** ⭐⭐⭐⭐ (4/5) - Great for AWS users, but complex and pricey for small deployments

**Pricing:**
- CPU (Linux x86): $0.04048/vCPU-hour ($29.15/month for 1 vCPU always-on)
- Memory (Linux): $0.0044/GB-hour ($3.17/month for 1 GB always-on)
- **Minimum viable task (0.25 vCPU + 0.5 GB):** ~$9/month
- **Fargate Spot (0.25 vCPU + 0.5 GB):** ~$0.83/month (up to 70% discount)
- Estimated: ~$15-30/month for typical ArmorClaw deployment (with ALB, EFS, CloudWatch)

**Memory Budget:** Flexible, but always-on Bridge adds up quickly

**Quick Start:**
```bash
# Create ECS task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Run task
aws ecs run-task --cluster armorclaw --task-definition armorclaw-bridge
```

---

### 9. Linode/Akamai Connected Cloud - VPS

**Best For:** VPS users who want full control

**Pros:**
- ✅ Full Docker control (root access to VPS)
- ✅ Cost-effective for small deployments
- ✅ Good for running Matrix Conduit + Bridge + Agent
- ✅ 11 global data centers
- ✅ GPU instances available (for AI inference)
- ✅ Simple pricing
- ✅ Good documentation

**Cons:**
- ❌ Manual scaling and management
- ❌ No auto-scaling or managed services (limited offerings)
- ❌ Requires manual OS updates and security patches
- ❌ No serverless options

**ArmorClaw Fit:** ⭐⭐⭐⭐ (4/5) - Excellent for VPS users who want full control

**Pricing:**
- Nanode (1 GB RAM, 1 vCPU, 25 GB SSD): ~$5/month
- Dedicated (2 GB RAM, 1 vCPU, 50 GB SSD): ~$10/month
- Dedicated (4 GB RAM, 2 vCPU, 80 GB SSD): ~$20/month
- GPU (1x A10): ~$1.10/hour (~$800/month if always-on)

**Memory Budget:** Dedicated 4 GB plan meets ArmorClaw's target with room to grow

**Quick Start:**
```bash
# Create Linode
linode-cli linodes create --image linode/ubuntu22.04 --type g6-nanode-1 --region us-east

# SSH and install Docker
ssh root@linode-ip
curl -fsSL https://get.docker.com | sh

# Deploy ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

### 10. Vultr - VPS with GPU Options

**Best For:** VPS users who want GPU support for AI inference

**Pros:**
- ✅ Full Docker control (root access to VPS)
- ✅ Very cost-effective for small deployments
- ✅ Excellent GPU offerings (NVIDIA A100, A40, A10)
- ✅ 32 global data centers
- ✅ Simple pricing
- ✅ Cloud GPU instances available (pay-per-use)
- ✅ Serverless Inference available

**Cons:**
- ❌ Manual scaling and management
- ❌ No auto-scaling or managed services
- ❌ Requires manual OS updates and security patches
- ❌ No serverless options (except GPU inference)

**ArmorClaw Fit:** ⭐⭐⭐⭐⭐ (5/5) for GPU workloads, ⭐⭐⭐⭐ (4/5) for standard deployments

**Pricing:**
- Regular (1 GB RAM, 1 vCPU): ~$2.50/month (cheapest option)
- Regular (2 GB RAM, 1 vCPU): ~$6/month
- Regular (4 GB RAM, 2 vCPU): ~$12/month
- **Cloud GPU (1x A40):** ~$0.73/hour (~$530/month if always-on)
- **Cloud GPU (1x A100):** ~$1.85/hour (~$1,340/month if always-on)
- **Serverless Inference:** Pay-per-use, pricing varies

**Memory Budget:** Regular 4 GB plan meets ArmorClaw's target with room to grow

**Quick Start:**
```bash
# Create Vultr instance
vultr-cli instance create --host 2026-02 --plan vc2-1c-2gb --region ewr --os 1743

# SSH and install Docker
ssh root@instance-ip
curl -fsSL https://get.docker.com | sh

# Deploy ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

### 11. Azure Container Instances - Serverless Containers (Azure)

**Best For:** Azure users, burstable workloads, short-lived containers

**Pros:**
- ✅ Docker support (deploy any container)
- ✅ Serverless (pay-per-second billing)
- ✅ Long-running containers (no time limit)
- ✅ Excellent Azure integration (Azure SQL, Azure Files, etc.)
- ✅ GPU support (via ACI with GPU)
- ✅ Multiple global regions

**Cons:**
- ❌ No background processes (ACI is for containers, not services)
- ❌ Can get expensive for always-on Bridge (no fixed-price tier)
- ❌ No auto-scaling (use Azure Container Apps for that)
- ❌ Unix socket support limited to container group

**ArmorClaw Fit:** ⭐⭐ (2/5) - Not suitable for persistent Bridge, but could work for agent containers

**Pricing:**
- CPU (Linux): ~$0.000012/vCPU-second
- Memory (Linux): ~$0.000004/GB-second
- **Estimated for always-on Bridge (0.5 vCPU + 1 GB):** ~$26/month
- **Estimated for agent containers (intermittent):** ~$5-15/month

**Memory Budget:** Flexible, but always-on Bridge gets expensive quickly

**Quick Start:**
```bash
# Create container group
az container create \
  --resource-group armorclaw-rg \
  --name armorclaw-bridge \
  --image armorclaw/bridge:v1 \
  --cpu 0.5 \
  --memory 1
```

---

## Use Case Recommendations

### 1. Local Development

**Recommended:** Docker Desktop (local)

**Why:**
- Zero cost for development
- Full control over all components
- Fast iteration (no network latency)
- Perfect for testing before deploying

**Setup:**
```bash
# Install Docker Desktop
# Clone ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
docker-compose up -d
```

---

### 2. Small Production Deployment (< 10 users)

**Recommended:** Hostinger VPS (KVM2) or Vultr (Regular)

**Why:**
- Dedicated resources at low cost (~$4-8/month)
- Full Docker control
- Predictable pricing
- Easy to manage

**Setup:** See `docs/guides/hostinger-deployment.md`

---

### 3. Large Production Deployment (10+ users)

**Recommended:** AWS Fargate (with Spot) or Google Cloud Run

**Why:**
- Auto-scaling based on demand
- Highly available (multiple AZs/regions)
- Managed services (RDS, EFS, Cloud SQL, etc.)
- Pay for what you use

**Setup:** Use ECS/EKS (AWS) or Cloud Run (GCP) with managed databases

---

### 4. Edge/Global Distribution

**Recommended:** Fly.io

**Why:**
- Global edge network (20+ regions)
- Auto-scaling
- Good free tier
- Simple CLI deployment

**Setup:**
```bash
cd armorclaw
fly launch
```

---

### 5. GPU/AI Inference Workloads

**Recommended:** Vultr (Cloud GPU or Serverless Inference)

**Why:**
- Pay-per-use GPU (no commitment)
- Multiple GPU options (A10, A40, A100)
- Serverless Inference for variable workloads
- Cost-effective for occasional GPU use

**Setup:** Deploy ArmorClaw on Vultr VPS, use Cloud GPU for agent inference

---

### 6. Cost-Optimized Deployment

**Recommended:** Hostinger VPS (KVM2) or Vultr (Regular)

**Why:**
- Lowest fixed monthly cost (~$2.50-8/month)
- Dedicated resources
- Full Docker control
- Predictable pricing

**Setup:** See `docs/guides/hostinger-deployment.md`

---

### 7. High Availability Deployment

**Recommended:** Google Cloud Run + Cloud SQL (multi-region)

**Why:**
- Multi-region deployment
- Managed services (Cloud SQL, Cloud Monitoring)
- Auto-scaling
- 99.99% uptime SLA

**Setup:** Deploy Bridge and Agent on Cloud Run, use Cloud SQL for keystore

---

## Cost Comparison Table

### Typical ArmorClaw Deployment (Bridge + Agent + Matrix + Keystore)

| Provider | Configuration | Monthly Cost | Notes |
|----------|--------------|--------------|-------|
| **Docker Desktop** | Local machine | Free | Development only |
| **Hostinger KVM2** | 2 vCPU, 4 GB RAM | ~$4-8 | Dedicated resources |
| **Vultr Regular** | 1-2 vCPU, 2-4 GB RAM | ~$6-12 | Good value |
| **Linode** | 1-2 vCPU, 2-4 GB RAM | ~$5-20 | Similar to Vultr |
| **Fly.io** | 1-2 vCPU, 1-2 GB RAM | ~$5-15 | Pay-per-use, free allowance |
| **Railway** | Usage-based | ~$5-20 | Free $5 credit, then usage-based |
| **Render** | Usage-based | ~$7-25 | Free tier spins down (not suitable) |
| **DigitalOcean App Platform** | 1 vCPU, 1-2 GB RAM | ~$12-32 | More expensive than DO Droplets |
| **Google Cloud Run** | Serverless | ~$10-30 | Pay-per-use, free tier |
| **AWS Fargate** | 0.5-1 vCPU, 1-2 GB RAM | ~$15-30 (or ~$5-10 with Spot) | Plus ALB, EFS, CloudWatch |
| **Azure Container Instances** | 0.5 vCPU, 1 GB RAM | ~$26+ | Pay-per-second, gets expensive |

### GPU Workloads (for AI inference)

| Provider | GPU | Hourly Cost | Monthly Cost (always-on) |
|----------|-----|-------------|-------------------------|
| **Vultr** | 1x A10 | ~$0.73 | ~$530 |
| **Vultr** | 1x A100 | ~$1.85 | ~$1,340 |
| **Linode** | 1x A40 | ~$1.10 | ~$800 |
| **AWS Fargate** | 1x A10 | ~$1.00+ | ~$720+ |
| **Google Cloud** | 1x A100 | ~$2.93+ | ~$2,100+ |

**Note:** For GPU workloads, consider serverless inference (pay-per-use) instead of always-on instances.

---

## Deployment Guides

### Top 3 Recommended Providers

#### 1. Hostinger VPS (Budget Production)

**Full Guide:** `docs/guides/hostinger-deployment.md`

**Quick Start:**
```bash
# 1. Create Hostinger VPS (KVM2 plan)
# 2. SSH into VPS
ssh root@your-vps-ip

# 3. Install Docker
curl -fsSL https://get.docker.com | sh

# 4. Clone ArmorClaw
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw

# 5. Deploy infrastructure
./deploy/launch-element-x.sh
```

---

#### 2. Fly.io (Global Edge)

**Quick Start:**
```bash
# 1. Install flyctl
curl -L https://fly.io/install.sh | sh

# 2. Authenticate
fly auth signup
fly auth login

# 3. Create fly.toml
cat > fly.toml << 'EOF'
app = "armorclaw-bridge"

[build]
  dockerfile = "Dockerfile.bridge"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]
EOF

# 4. Deploy
fly launch
fly deploy
```

---

#### 3. Google Cloud Run (Serverless)

**Quick Start:**
```bash
# 1. Build and push image
gcloud builds submit --tag gcr.io/PROJECT_ID/armorclaw-bridge

# 2. Deploy Bridge to Cloud Run
gcloud run deploy armorclaw-bridge \
  --image gcr.io/PROJECT_ID/armorclaw-bridge \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated

# 3. Create Cloud Run Job for agents
gcloud run jobs create armorclaw-agent \
  --image gcr.io/PROJECT_ID/armorclaw-agent \
  --region us-central1

# 4. Execute agent job
gcloud run jobs execute armorclaw-agent --region us-central1
```

---

## Conclusion

### Summary of Recommendations

| Use Case | Recommended Provider | Why |
|----------|---------------------|-----|
| **Development** | Docker Desktop (local) | Free, full control |
| **Small Production** | Hostinger VPS (KVM2) | Cheap, dedicated resources |
| **Large Production** | AWS Fargate (Spot) | Scalable, cost-effective |
| **Global Edge** | Fly.io | Global network, simple deployment |
| **GPU/AI** | Vultr (Cloud GPU) | Pay-per-use GPU |
| **Serverless** | Google Cloud Run | Good free tier, managed services |
| **Cost-Optimized** | Hostinger VPS | Lowest fixed monthly cost |
| **High Availability** | Google Cloud Run + Cloud SQL | Multi-region, managed |

### Key Takeaways

1. **For most users, Hostinger VPS (KVM2) offers the best balance of cost, performance, and simplicity.**

2. **For global distribution, Fly.io is the best choice** due to its edge network and simple deployment.

3. **For AWS/GCP users, use the serverless options** (Fargate, Cloud Run) for scalability and managed services.

4. **For GPU workloads, Vultr offers the most flexible and cost-effective options.**

5. **Cloudflare Workers is NOT suitable for ArmorClaw** due to fundamental platform limitations (no Docker socket, no Unix sockets, 30-second limit).

### Next Steps

1. **Choose a provider** based on your use case and budget
2. **Follow the deployment guide** for your chosen provider
3. **Test the deployment** using the test suites in `tests/`
4. **Monitor resource usage** and adjust as needed

### Additional Resources

- **Hostinger Deployment:** `docs/guides/hostinger-deployment.md`
- **Docker Deployment:** `docs/guides/hostinger-docker-deployment.md`
- **Element X Quick Start:** `docs/guides/element-x-quickstart.md`
- **Troubleshooting:** `docs/guides/troubleshooting.md`
- **Status Tracking:** `docs/PROGRESS/progress.md`

---

**Document Last Updated:** 2026-02-07
**ArmorClaw Version:** 1.2.0
**Phase:** Phase 1 Complete - Production Ready
