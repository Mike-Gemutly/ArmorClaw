# Fly.io Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Fly.io (https://fly.io)
> **Best For:** Global edge distribution, low-latency access worldwide
> **Difficulty Level:** Intermediate to Advanced
> **Estimated Time:** 45-60 minutes

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Account Setup](#account-setup)
4. [CLI Installation](#cli-installation)
5. [Authentication](#authentication)
6. [Project Setup](#project-setup)
7. [Docker Deployment](#docker-deployment)
8. [Configuration Management](#configuration-management)
9. [Volumes and Persistent Storage](#volumes-and-persistent-storage)
10. [Networking and Domains](#networking-and-domains)
11. [Scaling and Regions](#scaling-and-regions)
12. [Monitoring and Logs](#monitoring-and-logs)
13. [Pricing Details](#pricing-details)
14. [Limitations](#limitations)
15. [Troubleshooting](#troubleshooting)
16. [Advanced Features](#advanced-features)

---

## Overview

**Fly.io** is a platform for running apps close to users around the world. With 35+ global regions, it's ideal for ArmorClaw deployments requiring low-latency access worldwide.

### Why Fly.io for ArmorClaw?

✅ **Global Edge Network** - 35+ regions worldwide
✅ **Anycast Network** - Automatic global routing
✅ **Native Docker Support** - Deploy any container
✅ **Automatic SSL** - Free TLS certificates
✅ **Private Networking** - Secure service-to-service communication
✅ **Flexible Storage** - Volumes, PostgreSQL, Redis, MySQL
✅ **Simple CLI** - Easy deployment and management

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Fly.io Anycast Network                  │
│                   (Global Edge Distribution)                 │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Region:    │  │   Region:    │  │   Region:    │      │
│  │   ord        │  │   lhr        │  │   nrt        │      │
│  │  (Chicago)   │  │  (London)    │  │  (Tokyo)     │      │
│  │              │  │              │  │              │      │
│  │  ┌────────┐  │  │  ┌────────┐  │  │  ┌────────┐  │      │
│  │  │Bridge  │  │  │  │Bridge  │  │  │  │Bridge  │  │      │
│  │  │App     │  │  │  │App     │  │  │  │App     │  │      │
│  │  │Volumes │  │  │  │Volumes │  │  │  │Volumes │  │      │
│  │  └────────┘  │  │  └────────┘  │  │  └────────┘  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                  │                  │             │
│         └──────────────────┴──────────────────┘             │
│                            │                                │
│                     Fly.io Platform                        │
│                (Anycast, SSL, Monitoring)                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

### Before You Begin

- **Active Fly.io Account:** Sign up at https://fly.io
- **Payment Method:** Credit card required (for paid plans)
- **GitHub Account:** For deployment integration (optional)
- **Docker Installed:** For local testing
- **Basic CLI Knowledge:** Comfortable with terminal commands
- **ArmorClaw Repository:** Cloned locally

### System Requirements

**Local Machine:**
- **Operating System:** Windows, macOS, or Linux
- **Docker:** Latest version installed
- **Disk Space:** 500 MB for ArmorClaw code
- **Network:** Stable internet connection

**Fly.io Account:**
- **Free Trial:** 2 hours machine runtime OR 7 days access (whichever first)
- **Credit:** One-time $5 credit for new users
- **Credit Card:** Required for any usage beyond free trial

---

## Account Setup

### Step 1: Create Fly.io Account

1. Visit https://fly.io
2. Click **"Sign Up"**
3. Choose signup method:
   - GitHub account (recommended)
   - Email + password
4. Complete registration
5. Verify email (if using email signup)

### Step 2: Claim Free Credit

**New User Benefits:**
- **Free Trial:** 2 hours machine runtime OR 7 days access
- **One-Time Credit:** $5 USD
- **Credit Applied:** Automatically after account creation

### Step 3: Add Payment Method

**Required for:**
- Usage beyond free trial
- Paid plans ($5/month minimum)
- Volume storage beyond 3 GB free

**Steps:**
1. Navigate to https://fly.io/dashboard
2. Click **"Billing"**
3. Add credit card
4. Billing is per-second, pay-as-you-go

---

## CLI Installation

### Step 1: Install flyctl CLI

**macOS (Homebrew):**
```bash
brew install flyctl
```

**macOS (Manual):**
```bash
curl -L https://fly.io/install.sh | sh
```

**Linux (curl):**
```bash
curl -L https://fly.io/install.sh | sh
```

**Windows ( scoop):**
```powershell
scoop bucket add fly-io
scoop install flyctl
```

**Windows (chocolatey):**
```powershell
choco install flyctl
```

**Windows (manual installer):**
- Download from https://github.com/superfly/flyctl/releases
- Run installer
- Add to PATH

### Step 2: Verify Installation

```bash
flyctl version
# Expected output: flyctl v0.x.x

flyctl help
# Shows all available commands
```

### Step 3: Configure Shell Completion (Optional)

**Bash:**
```bash
echo 'source <(flyctl completion bash)' >> ~/.bashrc
source ~/.bashrc
```

**Zsh:**
```bash
echo 'source <(flyctl completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

**Fish:**
```bash
flyctl completion fish | source
```

---

## Authentication

### Step 1: Authenticate Fly.io

**Interactive Login:**
```bash
flyctl auth signup
```

This will:
1. Open browser for authentication
2. Request GitHub OAuth (if using GitHub)
3. Request API token
4. Save token locally

**Token-Based Login:**
```bash
# Generate token at https://fly.io/dashboard/personal/access-tokens
flyctl auth token

# Or set environment variable
export FLY_API_TOKEN=your_token_here
```

### Step 2: Verify Authentication

```bash
flyctl auth whoami
# Expected output: <your-email or username>

flyctl orgs open
# Expected output: Lists your organizations
```

### Step 3: Docker Registry Authentication (Optional)

For pushing/pulling images from Fly.io registry:

```bash
flyctl auth docker
```

This configures Docker to authenticate with `registry.fly.io`.

---

## Project Setup

### Step 1: Initialize Fly.io App

**Navigate to ArmorClaw directory:**
```bash
cd path/to/armorclaw
```

**Launch app:**
```bash
flyctl launch
```

**Interactive Prompts:**

1. **App Name:** `armorclaw-bridge` (or custom)
   - Must be unique across Fly.io
   - Used in URL: `https://armorclaw-bridge.fly.dev`

2. **Region:** Choose nearest to users
   - `ord` (Chicago) - Default
   - `lhr` (London)
   - `nrt` (Tokyo)
   - `syd` (Sydney)
   - Or any of 35+ regions

3. **Deploy Now:** `n` (we'll configure first)

**Files Created:**
```
fly.toml              # Fly.io configuration
Dockerfile            # Container build instructions
.fly/                 # Fly.io metadata directory
```

### Step 2: Configure fly.toml

**Edit fly.toml:**
```bash
nano fly.toml
```

**ArmorClaw Bridge Configuration:**
```toml
# fly.toml
app = "armorclaw-bridge"
primary_region = "ord"

[build]
  dockerfile = "Dockerfile.bridge"

[env]
  ARMORCLAW_ENV = "production"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false  # Keep bridge running
  min_machines_running = 1    # Always have 1 instance

  [[http_service.checks]]
    interval = "15s"
    timeout = "10s"
    grace_period = "5s"
    method = "GET"
    path = "/health"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512
```

**Save:** `Ctrl+O`, `Enter`, `Ctrl+X`

---

## Docker Deployment

### Step 1: Create Dockerfile.bridge

**Create Dockerfile for Bridge:**
```bash
nano Dockerfile.bridge
```

**Content:**
```dockerfile
# Multi-stage build for ArmorClaw Bridge
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy source code
COPY bridge/ ./bridge/
COPY go.mod go.sum ./

# Build bridge
RUN go build -o armorclaw-bridge ./bridge/cmd/bridge

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/armorclaw-bridge /app/armorclaw-bridge

# Create runtime directories
RUN mkdir -p /run/armorclaw /var/lib/armorclaw

# Expose health check port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD /app/armorclaw-bridge health || exit 1

# Run bridge
CMD ["/app/armorclaw-bridge"]
```

### Step 2: Deploy Application

**First Deployment:**
```bash
flyctl deploy
```

**What Happens:**
1. Builds Docker image remotely
2. Pushes to Fly.io registry
3. Creates Machines in specified region(s)
4. Runs health checks
5. Allocates public URL

**Expected Output:**
```
==> Verifying app config
--> Validating fly.toml
--> Validating app configuration
--> App configuration is valid!

==> Building image
--> Searching for image 'registry.fly.io/armorclaw-bridge:deployment-xxxxxxxx'
--> Remote image registry.fly.io/armorclaw-bridge:deployment-xxxxxxxx found
==> Creating release
--> Release v1 created

==> Deploying release
--> Release v1 deployed successfully
--> You can now access your app at https://armorclaw-bridge.fly.dev
```

### Step 3: Verify Deployment

**Check App Status:**
```bash
flyctl status
```

**Expected Output:**
```
App
  Name: armorclaw-bridge
  Owner: <your-email>
  Version: 1
  Status: running
  Hostname: armorclaw-bridge.fly.dev

Machines
  ID              State   Region  Checks                Updated
  xxxxxxxxxxxx    started ord     1 total, 1 passing    1m30s ago
```

**Test Health Endpoint:**
```bash
curl https://armorclaw-bridge.fly.dev/health
```

---

## Configuration Management

### Step 1: Set Environment Variables

**Individual Secrets:**
```bash
# Set API key
flyctl secrets set ARMORCLAW_API_KEY="sk-proj-..."

# Set database URL
flyctl secrets set DATABASE_URL="postgres://..."

# Set Matrix credentials
flyctl secrets set MATRIX_HOMESERVER="https://matrix.armorclaw.com"
flyctl secrets set MATRIX_USERNAME="bridge-bot"
flyctl secrets set MATRIX_PASSWORD="secret"
```

**Batch Secrets from File:**
```bash
# Create secrets file
cat > secrets.txt << EOF
ARMORCLAW_API_KEY=sk-proj-xxx
DATABASE_URL=postgres://user:pass@host:5432/db
MATRIX_HOMESERVER=https://matrix.armorclaw.com
MATRIX_USERNAME=bridge-bot
MATRIX_PASSWORD=secret
EOF

# Set all secrets
flyctl secrets set --stage < secrets.txt

# Delete secrets file
rm secrets.txt
```

**List Secrets:**
```bash
flyctl secrets list
```

**Delete Secret:**
```bash
flyctl secrets delete ARMORCLAW_API_KEY
```

### Step 2: Configuration Files

**Upload Configuration File:**
```bash
# Upload config to app
flyctl ssh sftp shell
# Then: put /path/to/config.toml /app/config/config.toml
```

**Or use Secrets for Small Configs:**
```bash
flyctl secrets set CONFIG_TOML="$(cat config.toml | base64)"
```

**In application, decode:**
```go
configBase64 := os.Getenv("CONFIG_TOML")
configData, _ := base64.StdEncoding.DecodeString(configBase64)
```

---

## Volumes and Persistent Storage

### Step 1: Create Volume

**Create 1 GB Volume:**
```bash
flyctl volumes create armorclaw_data --region ord --size 1
```

**Expected Output:**
```
        ID: vol_xxxxxxxxxxxxxxxxx
      Name: armorclaw_data
      Size: 1 GB
    Region: ord
   Encrypted: true
Created at: 07 Feb 26 12:00 UTC
```

### Step 2: Attach Volume to App

**Edit fly.toml:**
```bash
nano fly.toml
```

**Add volume mount:**
```toml
[[mounts]]
  source = "armorclaw_data"
  destination = "/data"
```

**Full fly.toml with volumes:**
```toml
app = "armorclaw-bridge"
primary_region = "ord"

[build]
  dockerfile = "Dockerfile.bridge"

[[mounts]]
  source = "armorclaw_data"
  destination = "/data"

[env]
  ARMORCLAW_DATA_DIR = "/data"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false
  min_machines_running = 1

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512
```

### Step 3: Redeploy with Volume

```bash
flyctl deploy
```

### Step 4: Verify Volume

**SSH into Machine:**
```bash
flyctl ssh shell
```

**Check Volume:**
```bash
df -h | grep data
# Expected output: /dev/vdb    1.0G    0    1.0G   0%  /data

ls -la /data
# Expected output: (empty or contains data)
```

### Step 5: Backup Volume

**Create Snapshot:**
```bash
flyctl volumes snapshot armorclaw_data
```

**List Snapshots:**
```bash
flyctl volumes snapshots armorclaw_data
```

**Restore from Snapshot:**
```bash
flyctl volumes restore armorclaw_data --snapshot-id <snapshot-id>
```

---

## Networking and Domains

### Step 1: Default Domain

**Automatic Domain:**
- Format: `https://<app-name>.fly.dev`
- Example: `https://armorclaw-bridge.fly.dev`
- SSL: Automatic, free Let's Encrypt certificate

### Step 2: Custom Domain

**Add Custom Domain:**
```bash
flyctl certs add your-domain.com
```

**Expected Output:**
```
Could not find a CNAME or A record for your-domain.com.
Please verify your DNS configuration.
```

**Configure DNS:**
1. Access DNS management at your domain registrar
2. Add CNAME record:
   - Name: `@` (root) or subdomain
   - Value: `armorclaw-bridge.fly.dev`
   - TTL: 3600

**Verify DNS Propagation:**
```bash
dig your-domain.com +short
# Expected output: CNAME armorclaw-bridge.fly.dev
```

**Re-run Certificate Command:**
```bash
flyctl certs add your-domain.com
```

**Expected Output:**
```
The certificate for your-domain.com has been issued.
```

### Step 3: Wildcard Certificate

**Add Wildcard Domain:**
```bash
flyctl certs add "*.your-domain.com"
```

**DNS Configuration:**
- Add CNAME: `*.your-domain.com` → `armorclaw-bridge.fly.dev`

### Step 4: Private Networking

**Services can communicate privately:**

```bash
# Create Matrix service
flyctl launch --name armorclaw-matrix --region ord

# Services communicate via internal DNS
# Matrix accessible at: http://armorclaw-matrix.internal:6167
```

**In fly.toml:**
```toml
[env]
  MATRIX_URL="http://armorclaw-matrix.internal:6167"
```

---

## Scaling and Regions

### Step 1: Multi-Region Deployment

**Add Additional Region:**
```bash
flyctl regions add lhr
```

**Deploy to New Region:**
```bash
flyctl deploy --region lhr
```

**Status After Multi-Region:**
```bash
flyctl status
```

**Expected Output:**
```
App
  Name: armorclaw-bridge
  Hostname: armorclaw-bridge.fly.dev

Machines
  ID              State   Region  Checks    Updated
  xxxxxxxxxxxx    started ord     1/1       2m ago
  yyyyyyyyyyyy    started lhr     1/1       1m ago
```

### Step 2: Scale Machines

**Scale to 3 Machines:**
```bash
flyctl scale count 3
```

**Scale with Region Distribution:**
```bash
# 2 in Chicago, 1 in London
flyctl scale count 2 --region ord
flyctl scale count 1 --region lhr
```

**Scale Zero-to-One (Scale to Zero):**
```bash
flyctl scale count 0
# App stops, no charges

flyctl scale count 1
# App starts on first request (cold start)
```

### Step 3: Auto-Scaling

**Enable Auto-Scaling:**
```bash
flyctl autoscale set --min 1 --max 10
```

**Auto-Scale Based on CPU:**
```bash
flyctl autoscale set --min 1 --max 10 --cpu 80
```

**Auto-Scale Based on Memory:**
```bash
flyctl autoscale set --min 1 --max 10 --memory 80
```

### Step 4: Machine Sizes

**List Available Sizes:**
```bash
flyctl platform sizes
```

**Common Sizes:**

| Size | CPU | Memory | Cost (approx) |
|------|-----|--------|---------------|
| **shared-cpu-1x** | 1 (shared) | 256 MB | $2-4/month |
| **shared-cpu-2x** | 2 (shared) | 512 MB | $4-8/month |
| **dedicated-cpu-1x** | 1 (dedicated) | 2 GB | $30-50/month |
| **dedicated-cpu-2x** | 2 (dedicated) | 4 GB | $60-100/month |

**Change Machine Size:**
```bash
flyctl scale vm shared-cpu-2x
```

**In fly.toml:**
```toml
[[vm]]
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 1024
```

---

## Monitoring and Logs

### Step 1: View Logs

**Live Logs:**
```bash
flyctl logs
```

**Follow Logs (tail):**
```bash
flyctl logs --tail
```

**Logs for Specific Machine:**
```bash
flyctl logs --machine xxxxxxxxxxxx
```

**Logs with Metadata:**
```bash
flyctl logs --format json
```

### Step 2: Metrics

**Real-Time Metrics:**
```bash
flyctl metrics
```

**Expected Output:**
```
Metrics for armorclaw-bridge (last 5m)

CPU Usage:
  avg: 15%
  max: 45%

Memory Usage:
  avg: 256 MB
  max: 380 MB

Request Rate:
  avg: 10 req/s
  max: 25 req/s
```

### Step 3: Health Checks

**Check Health Status:**
```bash
flyctl status
```

**Manual Health Check:**
```bash
curl https://armorclaw-bridge.fly.dev/health
```

### Step 4: Monitoring Integration

**Prometheus Metrics:**

Fly.io doesn't include built-in Prometheus, but you can:

1. **Self-Hosted Monitoring:**
   - Deploy Prometheus + Grafana on Fly.io
   - Use `/metrics` endpoint if exposed

2. **External Services:**
   - Datadog
   - New Relic
   - Honeycomb

---

## Pricing Details

### Pricing Model (2026)

**Free Trial:**
- Duration: 2 hours machine runtime OR 7 days (whichever first)
- One-time credit: $5
- **No permanent free tier**

### Pay-As-You-Go Pricing

**Compute (per-second billing):**

| Plan | vCPU | Memory | Price (approx/month) |
|------|------|--------|---------------------|
| **shared-cpu-1x** | 1 (shared) | 256 MB | $2-4 |
| **shared-cpu-2x** | 2 (shared) | 512 MB | $4-8 |
| **shared-cpu-4x** | 4 (shared) | 1 GB | $8-16 |
| **dedicated-cpu-1x** | 1 (dedicated) | 2 GB | $30-50 |
| **dedicated-cpu-2x** | 2 (dedicated) | 4 GB | $60-100 |

**Storage:**
- Volumes: $0.15/GB/month (~$5/30 days per GB)
- Free: 3 GB per volume
- Beyond free: $0.15/GB/month

**Networking:**
- Outbound data: Free 1 GB/month (NA)
- Beyond free: Variable by region

### Cost Calculator

**Example: Small ArmorClaw Deployment**

```
Components:
- Bridge (shared-cpu-2x, 512 MB): $5/month
- Matrix (shared-cpu-1x, 256 MB): $3/month
- Volume (5 GB): $0.75/month (after 3 GB free)
- Outbound Data (10 GB): $2/month

Total: ~$10-12/month
```

**Example: Large ArmorClaw Deployment**

```
Components:
- Bridge (dedicated-cpu-1x, 2 GB): $40/month
- Matrix (shared-cpu-2x, 512 MB): $5/month
- Volume (20 GB): $2.55/month (after 3 GB free)
- Outbound Data (100 GB): $15/month

Total: ~$60-65/month
```

### Pricing Optimization

1. **Use shared-cpu** for non-critical services
2. **Enable scale-to-zero** for development environments
3. **Optimize regions** - some regions are cheaper
4. **Monitor usage** - avoid over-provisioning
5. **Use volumes efficiently** - delete unused volumes

---

## Limitations

### Platform Limitations

| Limitation | Details | Impact |
|------------|---------|--------|
| **No permanent free tier** | Only 2-hour trial | Not suitable for free hosting |
| **Cold starts** | Scale-to-zero has latency | ~5-30s delay on first request |
| **Volume size limit** | 3 GB free per volume | Additional storage costs $0.15/GB/month |
| **CPU throttling** | Shared CPUs are shared | Performance varies by load |
| **Platform stability** | Occasional issues reported | May experience downtime |
| **No GPU support** (as of 2026) | CPU-only compute | Not suitable for GPU workloads |
| **Region gaps** | Not all countries covered | Some regions far from users |

### ArmorClaw-Specific Considerations

✅ **Fully Supported:**
- Docker container deployment
- Unix socket communication (within machine)
- Long-running containers (no timeout)
- Background processes (persistent machines)
- Custom domains with SSL
- Private networking between services
- Volume storage for persistence

⚠️ **Partial Support:**
- Unix socket across machines (use HTTP/private networking)
- Auto-scaling (requires manual configuration)
- Multi-region (manual region selection)

❌ **Not Supported:**
- GPU compute (use external GPU service)
- Docker socket access (not needed on Fly.io)

---

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Deployment Fails with Build Error

**Symptoms:**
- `Error: failed to fetch`
- Build timeouts

**Solutions:**

1. **Check Dockerfile:**
```bash
# Test Dockerfile locally
docker build -f Dockerfile.bridge -t test .
```

2. **Increase Build Timeout:**
```bash
flyctl deploy --build-timeout 600
```

3. **Check Build Logs:**
```bash
flyctl logs --build
```

#### Issue 2: App Not Responding (502/503)

**Symptoms:**
- `502 Bad Gateway`
- `503 Service Unavailable`

**Solutions:**

1. **Check Machine Status:**
```bash
flyctl status
```

2. **Restart Machine:**
```bash
flyctl machines restart xxxxxxxxxxxx
```

3. **Check Health Checks:**
```bash
curl https://armorclaw-bridge.fly.dev/health
```

4. **Review Logs:**
```bash
flyctl logs --tail
```

#### Issue 3: Volume Mount Fails

**Symptoms:**
- `Failed to mount volume`
- App won't start

**Solutions:**

1. **Verify Volume Exists:**
```bash
flyctl volumes list
```

2. **Check Volume Region:**
```bash
flyctl volumes list --region ord
```

3. **Ensure Region Match:**
```toml
# In fly.toml
primary_region = "ord"  # Must match volume region
```

4. **Redeploy:**
```bash
flyctl deploy --force
```

#### Issue 4: Certificate Error

**Symptoms:**
- `ERR_CERT_AUTHORITY_INVALID`
- Certificate not issued

**Solutions:**

1. **Verify DNS Configuration:**
```bash
dig your-domain.com +short
```

2. **Wait for DNS Propagation:**
```bash
# May take 1-24 hours
watch dig your-domain.com +short
```

3. **Re-issue Certificate:**
```bash
flyctl certs remove your-domain.com
flyctl certs add your-domain.com
```

4. **Check Certificate Status:**
```bash
flyctl certs list
```

#### Issue 5: Out of Memory

**Symptoms:**
- App crashes
- OOM killed

**Solutions:**

1. **Check Memory Usage:**
```bash
flyctl metrics
```

2. **Increase Memory:**
```bash
flyctl scale vm shared-cpu-2x --memory 1024
```

3. **In fly.toml:**
```toml
[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 1024  # Increase from 512
```

4. **Redeploy:**
```bash
flyctl deploy
```

#### Issue 6: High Latency

**Symptoms:**
- Slow response times
- Users far from region

**Solutions:**

1. **Deploy Closer to Users:**
```bash
flyctl regions add lhr  # London
flyctl regions add nrt  # Tokyo
```

2. **Use Anycast:**
- Automatically routes to nearest region

3. **Enable Caching:**
```toml
# In fly.toml
[http_service]
  [http_service.headers]
    X-Cache-Status = "HIT"
```

### Getting Help

**Fly.io Resources:**
- **Documentation:** https://fly.io/docs/
- **Community Forum:** https://community.fly.io
- **Status Page:** https://status.fly.io
- **GitHub:** https://github.com/superfly/flyctl

**ArmorClaw Resources:**
- **Documentation:** https://github.com/armorclaw/armorclaw
- **Issues:** https://github.com/armorclaw/armorclaw/issues

---

## Advanced Features

### Step 1: Cron Jobs

**Scheduled Tasks:**

Fly.io doesn't have native cron, but you can:

1. **Use External Cron:**
   - Set up cron job elsewhere
   - Trigger Fly.io app via HTTP

2. **Self-Hosted Cron:**
```bash
# Deploy separate cron service
flyctl launch --name armorclaw-cron

# In app, implement cron logic
# e.g., using github.com/robfig/cron
```

### Step 2: Blue-Green Deployment

**Zero-Downtime Deployment:**

Fly.io supports rolling updates by default:

1. **Deploy New Version:**
```bash
flyctl deploy --strategy immediate
```

2. **Custom Strategy:**
```toml
# In fly.toml
[deploy]
  strategy = "canary"
```

### Step 3: WireGuard VPN

**Private Network Access:**

```bash
# Install WireGuard
brew install wireguard-tools  # macOS
# Or: apt install wireguard  # Linux

# Create WireGuard peer
flyctl wireguard create <token>

# Connect to VPN
wg-quick up wg0

# Access services privately
curl http://armorclaw-bridge.internal:8080
```

### Step 4: CI/CD Integration

**GitHub Actions:**

```yaml
# .github/workflows/deploy.yml
name: Deploy to Fly.io

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Deploy to Fly.io
        uses: superfly/flyctl-actions@master
        with:
          args: "deploy --remote-only"
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

### Step 5: Multiple Environments

**Staging and Production:**

```bash
# Create staging app
flyctl launch --name armorclaw-bridge-staging

# Create production app
flyctl launch --name armorclaw-bridge-prod

# Deploy to staging
flyctl deploy --app armorclaw-bridge-staging

# Deploy to production
flyctl deploy --app armorclaw-bridge-prod
```

---

## Quick Reference

### Essential Commands

```bash
# Launch new app
flyctl launch

# Deploy app
flyctl deploy

# View app status
flyctl status

# View logs
flyctl logs --tail

# SSH into machine
flyctl ssh shell

# Scale machines
flyctl scale count 3

# Add region
flyctl regions add lhr

# Set secrets
flyctl secrets set KEY=value

# List volumes
flyctl volumes list

# View metrics
flyctl metrics

# Restart machine
flyctl machines restart <machine-id>

# Destroy app
flyctl apps destroy
```

### Useful URLs

```
App Dashboard:  https://fly.io/dashboard
App URL:        https://<app-name>.fly.dev
Metrics:        flyctl metrics
Logs:           flyctl logs --tail
Status Page:    https://status.fly.io
```

---

## Conclusion

Fly.io provides an excellent platform for global ArmorClaw deployment with:

✅ **35+ global regions** for low-latency access
✅ **Automatic SSL certificates** for all domains
✅ **Private networking** for secure service communication
✅ **Flexible storage** (volumes, databases)
✅ **Simple CLI** for easy deployment
✅ **Per-second billing** for cost optimization

**Next Steps:**
1. Complete deployment using this guide
2. Configure custom domain
3. Set up volumes for persistent storage
4. Deploy to multiple regions if needed
5. Monitor metrics and logs

**Related Documentation:**
- [Fly.io Documentation](https://fly.io/docs/)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)
- [Troubleshooting Guide](docs/guides/troubleshooting.md)

---

**Document Last Updated:** 2026-02-07
**Fly.io Version:** Based on 2026 pricing and features
**ArmorClaw Version:** 1.2.0
