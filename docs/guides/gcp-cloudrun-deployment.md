# Google Cloud Run Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Google Cloud Run (https://cloud.google.com/run)
> **Best For:** Serverless container deployment, auto-scaling, managed infrastructure
> **Difficulty Level:** Intermediate to Advanced
> **Estimated Time:** 60-90 minutes

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Account Setup](#account-setup)
4. [Project Setup](#project-setup)
5. [CLI Installation](#cli-installation)
6. [Authentication](#authentication)
7. [Container Build](#container-build)
8. [Cloud Run Deployment](#cloud-run-deployment)
9. [Configuration Management](#configuration-management)
10. [Database Integration](#database-integration)
11. [Networking and Security](#networking-and-security)
12. [Scaling and Performance](#scaling-and-performance)
13. [Monitoring and Logging](#monitoring-and-logging)
14. [Pricing Details](#pricing-details)
15. [Limitations](#limitations)
16. [Troubleshooting](#troubleshooting)
17. [Best Practices](#best-practices)

---

## Overview

**Google Cloud Run** is a fully managed compute platform for deploying and scaling containerized applications. Cloud Run abstracts away all infrastructure management, letting you focus on building applications.

### Why Cloud Run for ArmorClaw?

✅ **Serverless** - No infrastructure management
✅ **Auto-Scaling** - Zero to N instances automatically
✅ **Generous Free Tier** - 2M requests/month free
✅ **Global Network** - 20+ regions worldwide
✅ **Integrated Services** - Cloud SQL, Cloud Storage, Memorystore
✅ **Pay-Per-Use** - Pay only for what you use
✅ **SSL/TLS Included** - Automatic HTTPS

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Google Cloud Platform                    │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                 Cloud Run Service                     │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │   │
│  │  │   Revision 1 │  │   Revision 2 │  │  Latest   │  │   │
│  │  │  (Bridge)    │  │  (New Ver)   │  │ (Active)  │  │   │
│  │  │  100%        │  │  0%          │  │  100%     │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │   │
│  │                                                     │   │
│  │  Auto-Scaling (0 → N instances)                    │   │
│  │  Request Load Balancer                              │   │
│  │  HTTPS Endpoint (automatic)                         │   │
│  └──────────────────────────────────────────────────────┘   │
│                          │                                   │
│  ┌───────────────────────┼───────────────────────────────┐   │
│  │                       │                               │   │
│  ▼                       ▼                               ▼   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Cloud SQL   │  │ Cloud Storage│  │  Secret Mgr  │   │
│  │  (PostgreSQL)│  │   (Files)    │  │  (API Keys)  │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Cloud Logging & Monitoring               │   │
│  │  (Logs, Metrics, Error Reporting, Tracing)            │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

### Before You Begin

- **Google Cloud Account:** Sign up at https://console.cloud.google.com
- **Payment Method:** Credit card required (even for free tier)
- **Basic GCP Knowledge:** Familiarity with Google Cloud Console
- **Docker Installed:** For local testing
- **ArmorClaw Repository:** Cloned locally

### System Requirements

**Local Machine:**
- **Operating System:** Windows, macOS, or Linux
- **Docker:** Latest version installed
- **Disk Space:** 500 MB for ArmorClaw code
- **Network:** Stable internet connection

**Google Cloud Account:**
- **Free Tier:** $300 credit for new customers (30 days)
- **Monthly Free Usage:**
  - 240,000 vCPU-seconds
  - 450,000 GiB-seconds memory
  - 2 million requests
- **Credit Card:** Required for activation

---

## Account Setup

### Step 1: Create Google Cloud Account

1. Visit https://console.cloud.google.com
2. Click **"Get started for free"**
3. Sign in with Google account
4. Agree to terms of service
5. Select country/region (affects pricing)
6. Add payment method (credit/debit card)

**Initial Credits:**
- **New Customers:** $300 USD credit (30 days)
- **Free Tier Usage:** Monthly free limits apply after credit

### Step 2: Create Project

**Via Console:**
1. In Google Cloud Console, click project selector
2. Click **"New Project"**
3. Enter project name: `armorclaw-deployment`
4. Select organization (if applicable)
5. Click **"Create"**

**Via CLI:**
```bash
gcloud projects create armorclaw-deployment
```

### Step 3: Enable Cloud Run API

**Via Console:**
1. Navigate to **APIs & Services** → **Library**
2. Search for "Cloud Run API"
3. Click **"Enable"**

**Via CLI:**
```bash
gcloud services enable run.googleapis.com
```

### Step 4: Enable Required APIs

Enable additional APIs for ArmorClaw:

**Via Console:**
- Cloud SQL API
- Cloud Storage API
- Secret Manager API
- Cloud Build API
- Artifact Registry API

**Via CLI:**
```bash
gcloud services enable \
  sqladmin.googleapis.com \
  storage.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com
```

---

## Project Setup

### Step 1: Set Default Project

```bash
gcloud config set project armorclaw-deployment

# Verify
gcloud config get-value project
# Output: armorclaw-deployment
```

### Step 2: Set Default Region

```bash
# Set region (choose closest to users)
gcloud config set run/region us-central1

# Common regions:
# us-central1 (Iowa)       - Default, cheap
# us-east1 (South Carolina) - Popular
# us-west1 (Oregon)        - West coast US
# europe-west1 (Belgium)   - Europe
# asia-east1 (Taiwan)      - Asia
# australia-southeast1 (Sydney) - Australia

# Verify
gcloud config get-value run/region
```

### Step 3: Set Default Zone

```bash
# Set zone within region
gcloud config set run/zone us-central1-a

# Verify
gcloud config get-value run/zone
```

---

## CLI Installation

### Step 1: Install Google Cloud SDK

**macOS (Homebrew):**
```bash
brew install google-cloud-sdk
```

**macOS (Manual):**
```bash
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
gcloud init
```

**Linux (Debian/Ubuntu):**
```bash
# Add Google Cloud SDK distribution URI
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list

# Import Google Cloud public key
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -

# Update and install
sudo apt-get update && sudo apt-get install google-cloud-cli

# Optional: Install optional components
sudo apt-get install google-cloud-cli-extra-python
```

**Windows:**
- Download installer from https://cloud.google.com/sdk/docs/install
- Run installer
- Restart terminal

### Step 2: Verify Installation

```bash
gcloud --version
# Expected output: Google Cloud SDK 4xx.x.x

gcloud info
# Shows installation details
```

### Step 3: Install Components

```bash
# Install additional components
gcloud components install \
  cloud-run-proxy \
  gcloud-crc32c

# Update all components
gcloud components update
```

---

## Authentication

### Step 1: Authenticate GCP Account

**Interactive Login:**
```bash
gcloud auth login
```

This will:
1. Open browser for authentication
2. Request OAuth consent
3. Save authorization token locally

**Service Account (for automation):**
```bash
# Create service account
gcloud iam service-accounts create armorclaw-sa \
  --display-name="ArmorClaw Service Account"

# Grant necessary roles
gcloud projects add-iam-policy-binding armorclaw-deployment \
  --member="serviceAccount:armorclaw-sa@armorclaw-deployment.iam.gserviceaccount.com" \
  --role="roles/run.invoker"

gcloud projects add-iam-policy-binding armorclaw-deployment \
  --member="serviceAccount:armorclaw-sa@armorclaw-deployment.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Create and download key
gcloud iam service-accounts keys create armorclaw-sa.json \
  --iam-account=armorclaw-sa@armorclaw-deployment.iam.gserviceaccount.com

# Authenticate with service account
export GOOGLE_APPLICATION_CREDENTIALS=$(pwd)/armorclaw-sa.json
```

### Step 2: Configure Docker Authentication

```bash
gcloud auth configure-docker us-central1-docker.pkg.dev
# Or for all regions:
gcloud auth configure-docker
```

This allows pushing/pulling images to Google Artifact Registry.

### Step 3: Verify Authentication

```bash
gcloud auth list
# Shows authenticated accounts

gcloud auth list --filter=status:ACTIVE
# Shows active account
```

---

## Container Build

### Step 1: Create Dockerfile

**Navigate to ArmorClaw directory:**
```bash
cd path/to/armorclaw
```

**Create Dockerfile.bridge:**
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
RUN CGO_ENABLED=1 go build -o armorclaw-bridge ./bridge/cmd/bridge

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/armorclaw-bridge /app/armorclaw-bridge

# Create runtime directories
RUN mkdir -p /run/armorclaw /var/lib/armorclaw

# Expose Cloud Run port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run bridge
CMD ["/app/armorclaw-bridge", "--server-addr=:8080"]
```

### Step 2: Build Container Locally (Optional)

```bash
# Build locally for testing
docker build -f Dockerfile.bridge -t armorclaw-bridge:test .

# Test locally
docker run -p 8080:8080 armorclaw-bridge:test
```

### Step 3: Create Artifact Registry Repository

```bash
# Create Docker repository
gcloud artifacts repositories create armorclaw-repo \
  --repository-format=docker \
  --location=us-central1 \
  --description="ArmorClaw container registry"
```

### Step 4: Build and Push to Artifact Registry

```bash
# Set variables
PROJECT_ID=$(gcloud config get-value project)
REGION=$(gcloud config get-value run/region)
IMAGE_NAME="armorclaw-bridge"
IMAGE_URI="${REGION}-docker.pkg.dev/${PROJECT_ID}/armorclaw-repo/${IMAGE_NAME}"

# Build image with Cloud Build
gcloud builds submit \
  --tag "${IMAGE_URI}:latest" \
  --tag "${IMAGE_URI}:v1.2.0" \
  .

# Or build locally and push
docker build -f Dockerfile.bridge -t "${IMAGE_URI}:latest" .
docker push "${IMAGE_URI}:latest"

# Verify image
gcloud artifacts docker images list "${IMAGE_URI}" --include-tags
```

---

## Cloud Run Deployment

### Step 1: Deploy to Cloud Run

**Basic Deployment:**
```bash
gcloud run deploy armorclaw-bridge \
  --image="${IMAGE_URI}:latest" \
  --platform=managed \
  --region=us-central1 \
  --allow-unauthenticated \
  --port=8080
```

**Deployment with Options:**
```bash
gcloud run deploy armorclaw-bridge \
  --image="${IMAGE_URI}:v1.2.0" \
  --platform=managed \
  --region=us-central1 \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --timeout=300 \
  --concurrency=80 \
  --set-env-vars="ARMORCLAW_ENV=production"
```

**Expected Output:**
```
Deploying container to Cloud Run service [armorclaw-bridge] in project [armorclaw-deployment] region [us-central1]

✓ Deploying...
✓ Serving...
✓ Setting IAM policy...

Done.
Service [armorclaw-bridge] revision [armorclaw-bridge-00001-xxx] has been deployed and is serving 100 percent of traffic.
Service URL: https://armorclaw-bridge-xxxxx-uc.a.run.app
```

### Step 2: Verify Deployment

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --format='value(status.url)')

echo "Service URL: ${SERVICE_URL}"

# Test health endpoint
curl "${SERVICE_URL}/health"
```

### Step 3: Check Service Status

```bash
gcloud run services describe armorclaw-bridge \
  --platform=managed \
  --region=us-central1
```

**Expected Output:**
```
metadata:
  name: armorclaw-bridge
  namespace: '123456789'
spec:
  template:
    spec:
      containers:
      - image: us-central1-docker.pkg.dev/armorclaw-deployment/armorclaw-repo/armorclaw-bridge@sha256:xxxx
        ports:
        - name: http1
          containerPort: 8080
status:
  latestReadyRevision: armorclaw-bridge-00001-xxx
  observedGeneration: 1
  traffic:
  - revisionName: armorclaw-bridge-00001-xxx
    percent: 100
  url:
    https://armorclaw-bridge-xxxxx-uc.a.run.app
```

---

## Configuration Management

### Step 1: Environment Variables

**Set Environment Variables:**
```bash
gcloud run services update armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --set-env-vars="ARMORCLAW_ENV=production,LOG_LEVEL=info"
```

**Update Individual Variables:**
```bash
gcloud run services update armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --update-env-vars="NEW_VAR=value"
```

**Remove Environment Variables:**
```bash
gcloud run services update armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --remove-env-vars="OLD_VAR"
```

### Step 2: Secret Management

**Create Secret:**
```bash
echo -n "sk-proj-xxx" | \
  gcloud secrets create ARMORCLAW_API_KEY \
  --data-file=-
```

**Grant Access to Secret:**
```bash
# Get service account number
PROJECT_NUMBER=$(gcloud projects describe armorclaw-deployment --format='value(projectNumber)')

# Grant secret accessor role
gcloud secrets add-iam-policy-binding ARMORCLAW_API_KEY \
  --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
  --role='roles/secretmanager.secretAccessor'
```

**Mount Secret in Cloud Run:**
```bash
gcloud run services update armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --update-secrets="ARMORCLAW_API_KEY=ARMORCLAW_API_KEY:latest"
```

**Access Secret in Container:**
```bash
# Secret is available as environment variable
# Or mounted as file at /etc/secrets/
```

### Step 3: Configuration Files

**For larger configurations, use Cloud Storage:**

```bash
# Upload config to Cloud Storage
gsutil mb gs://armorclaw-configs
gsutil cp config.toml gs://armorclaw-configs/

# Grant service account access
gsutil iam ch serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com:objectViewer \
  gs://armorclaw-configs

# In Cloud Run, download config at startup
# (Add to Dockerfile or startup script)
```

---

## Database Integration

### Step 1: Create Cloud SQL Instance

**Via Console:**
1. Navigate to **SQL** in Cloud Console
2. Click **"Create Instance"**
3. Choose **PostgreSQL** (recommended for ArmorClaw)
4. Configure:
   - Instance ID: `armorclaw-db`
   - Password: (generate strong password)
   - Region: Same as Cloud Run service
   - Zone: Same as Cloud Run service
   - Machine type: `db-f1-micro` (free tier) or `db-g1-small`
5. Click **"Create"**

**Via CLI:**
```bash
gcloud sql instances create armorclaw-db \
  --database-version=POSTGRES_14 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --storage-auto-increase \
  --backup-start-time=03:00
```

### Step 2: Create Database

```bash
gcloud sql databases create armorclaw \
  --instance=armorclaw-db
```

### Step 3: Create Database User

```bash
gcloud sql users create armorclaw-user \
  --instance=armorclaw-db \
  --password=STRONG_PASSWORD_HERE
```

### Step 4: Connect Cloud Run to Cloud SQL

**Via Cloud SQL Proxy (Recommended):**

```bash
gcloud run services update armorclaw-bridge \
  --platform=managed \
  --region=us-central1 \
  --add-cloudsql-instances=armorclaw-deployment:us-central1:armorclaw-db \
  --update-env-vars="DATABASE_HOST=/cloudsql/armorclaw-deployment:us-central1:armorclaw-db,DATABASE_NAME=armorclaw,DATABASE_USER=armorclaw-user,DATABASE_PASSWORD=PASSWORD"
```

**Connection String:**
```
postgresql://armorclaw-user:PASSWORD@/cloudsql/armorclaw-deployment:us-central1:armorclaw-db/armorclaw
```

---

## Networking and Security

### Step 1: Default Domain

**Automatic Domain:**
- Format: `https://<service-name>-<hash>-<region>.a.run.app`
- Example: `https://armorclaw-bridge-xxxxx-uc.a.run.app`
- SSL: Automatic, Google-managed certificate

### Step 2: Custom Domain

**Via Console:**
1. Navigate to **Cloud Run** → **armorclaw-bridge**
2. Click **"Manage Custom Domains"**
3. Click **"Add Custom Domain"**
4. Enter domain: `armorclaw.your-domain.com`
5. Verify domain ownership (DNS record)
6. Wait for SSL certificate provisioning

**Via CLI:**
```bash
gcloud run domain-mappings create \
  --service=armorclaw-bridge \
  --domain=armorclaw.your-domain.com \
  --region=us-central1
```

**DNS Configuration:**
1. Access DNS management at your domain registrar
2. Add CNAME record:
   - Name: `armorclaw`
   - Value: `ghs.googlehosted.com`
   - TTL: 3600

### Step 3: VPC Connector (Private Resources)

**Create VPC Connector:**
```bash
gcloud compute networks vpc-access connectors create armorclaw-connector \
  --region=us-central1 \
  --range=10.8.0.0/28
```

**Deploy with VPC Connector:**
```bash
gcloud run deploy armorclaw-bridge \
  --image="${IMAGE_URI}:latest" \
  --vpc-connector=armorclaw-connector \
  --vpc-egress=private-ranges-only
```

### Step 4: Ingress Settings

**Restrict to Specific IPs:**
```bash
gcloud run services update armorclaw-bridge \
  --ingress=internal \
  --network=projects/armorclaw-deployment/global/networks/default
```

**Ingress Options:**
- `all` - Public and internal
- `internal` - Internal only (VPC/Cloud Load Balancing)
- `internal-and-cloud-load-balancing` - Internal + Cloud Load Balancing

### Step 5: Security Best Practices

**Enable Binary Authorization:**
```bash
gcloud run services update armorclaw-bridge \
  --binary-authorization=breakglass
```

**Require Authentication:**
```bash
gcloud run services update armorclaw-bridge \
  --no-allow-unauthenticated
```

**Set IAM Policy:**
```bash
# Allow specific user
gcloud run services add-iam-policy-binding armorclaw-bridge \
  --member=user:user@example.com \
  --role=roles/run.invoker \
  --region=us-central1 \
  --platform=managed
```

---

## Scaling and Performance

### Step 1: Configure Instance Settings

**Min/Max Instances:**
```bash
gcloud run services update armorclaw-bridge \
  --min-instances=1 \
  --max-instances=10
```

**Min Instances Options:**
- `0` - Scale to zero (cold starts)
- `1` - Always one instance running
- `N` - Always N instances running (more expensive, no cold starts)

### Step 2: Configure CPU and Memory

```bash
gcloud run services update armorclaw-bridge \
  --cpu=1 \
  --memory=512Mi
```

**CPU Options:**
- `1` - 1 vCPU
- `2` - 2 vCPUs
- `4` - 4 vCPUs (may require increase in quota)

**Memory Options:**
- `512Mi` - 512 MB
- `1Gi` - 1 GB
- `2Gi` - 2 GB
- `4Gi` - 4 GB
- Up to `32Gi` (may require quota increase)

### Step 3: Configure Concurrency

```bash
gcloud run services update armorclaw-bridge \
  --concurrency=80 \
  --concurrency-target=50
```

**Concurrency:** Number of requests per instance

### Step 4: Configure Timeout

```bash
gcloud run services update armorclaw-bridge \
  --timeout=300
```

**Timeout Options:**
- Minimum: 1 second
- Maximum: 3600 seconds (60 minutes)
- Default: 300 seconds (5 minutes)

### Step 5: Auto-Scaling Behavior

Cloud Run automatically scales based on:

- **Incoming requests** - More requests = more instances
- **CPU utilization** - Scale up if CPU > 60-80%
- **Memory utilization** - Scale up if memory > 80%

**No configuration needed** - Cloud Run handles automatically.

---

## Monitoring and Logging

### Step 1: View Logs

**Via Console:**
1. Navigate to **Logging** → **Logs Explorer**
2. Filter: `resource.labels.service_name="armorclaw-bridge"`
3. View logs in real-time

**Via CLI:**
```bash
# Stream logs
gcloud logging tail "resource.labels.service_name=armorclaw-bridge"

# View recent logs
gcloud logging read "resource.labels.service_name=armorclaw-bridge" \
  --limit=50 \
  --freshness=1h
```

### Step 2: Cloud Monitoring Metrics

**Enable Cloud Monitoring:**
```bash
gcloud run services update armorclaw-bridge \
  --update-labels=cloud.googleapis.com/location=us-central1
```

**View Metrics:**
```bash
# Via Console: Monitoring → Metrics Explorer
# Metrics available:
# - request_count
# - request_latencies
# - instance_count
# - cpu/utilization
# - memory/utilization
```

### Step 3: Create Dashboards

**Via Console:**
1. Navigate to **Monitoring** → **Dashboards**
2. Click **"Create Dashboard"**
3. Add charts for:
   - Request count
   - Request latency
   - Error rate
   - CPU utilization
   - Memory utilization

### Step 4: Alert Policies

**Create Alert:**
```bash
gcloud alpha monitoring policies create \
  --policy-from-file=alert-policy.yaml
```

**Example Alert Policy:**
```yaml
# alert-policy.yaml
displayName: "ArmorClaw High Error Rate"
conditions:
  - displayName: "Error rate > 5%"
    conditionThreshold:
      filter: >
        resource.type="cloud_run_revision"
        resource.labels.service_name="armorclaw-bridge"
        metric.type="run.googleapis.com/request_count"
        metric.labels.response_code_class="5"
      aggregation:
        alignmentPeriod: 300s
        perSeriesAligner: ALIGN_RATE
        crossSeriesReducer: REDUCE_SUM
      comparison:
        comparisonOperation: COMPARISON_GT
        thresholdValue: 0.05
      duration: 300s
```

### Step 5: Error Reporting

**Enable Error Reporting:**
```bash
# Errors are automatically reported to Cloud Error Reporting
# Access via: Error Reporting in Cloud Console
```

---

## Pricing Details

### Pricing Model (2026)

**Free Tier Monthly:**
- 240,000 vCPU-seconds
- 450,000 GiB-seconds memory
- 2 million requests
- 1 GB network egress from North America

**Beyond Free Tier (us-central1 pricing):**

| Resource | Price |
|----------|-------|
| **CPU** | $0.00000995 per vCPU-second |
| **Memory** | $0.0000004 per GiB-second |
| **Requests** | $0.40 per million |
| **Networking** | $0.12 per GB egress |

### Cost Calculator

**Example: Always-On Bridge (1 vCPU, 512 MB, 30 days)**

```
CPU:
  1 vCPU * 30 days * 24 hours * 3600 seconds = 2,592,000 vCPU-seconds
  Free: 240,000 vCPU-seconds
  Billable: 2,352,000 vCPU-seconds
  Cost: 2,352,000 * $0.00000995 = $23.40

Memory:
  0.5 GiB * 30 days * 24 hours * 3600 seconds = 1,296,000 GiB-seconds
  Free: 450,000 GiB-seconds
  Billable: 846,000 GiB-seconds
  Cost: 846,000 * $0.0000004 = $0.34

Requests:
  1 million requests (example)
  Free: 2 million requests
  Billable: 0
  Cost: $0

Networking:
  10 GB egress (example)
  Free: 1 GB
  Billable: 9 GB
  Cost: 9 * $0.12 = $1.08

Total: ~$25/month
```

**Example: Scale-to-Zero Bridge (rarely used)**

```
CPU:
  1 vCPU * 1 hour/day * 30 days = 108,000 vCPU-seconds
  Free: 240,000 vCPU-seconds
  Billable: 0
  Cost: $0

Memory:
  0.5 GiB * 1 hour/day * 30 days = 54,000 GiB-seconds
  Free: 450,000 GiB-seconds
  Billable: 0
  Cost: $0

Requests:
  10,000 requests/month
  Free: 2 million requests
  Billable: 0
  Cost: $0

Networking:
  1 GB egress
  Free: 1 GB
  Billable: 0
  Cost: $0

Total: $0/month (within free tier)
```

### Regional Pricing Variations

**Tier 1 Regions (cheapest):**
- us-central1 (Iowa)
- us-east1 (South Carolina)
- us-west1 (Oregon)

**Tier 2 Regions (~20% more expensive):**
- europe-west1 (Belgium)
- asia-east1 (Taiwan)
- australia-southeast1 (Sydney)

---

## Limitations

### Platform Limitations

| Limitation | Value | Impact |
|------------|-------|--------|
| **Maximum Request Timeout** | 60 minutes | Long-running agent sessions limited |
| **Maximum Memory** | 32 GiB | Large memory workloads |
| **Maximum CPU** | 8 vCPUs | CPU-intensive workloads |
| **Request Size** | 32 MB | Large payloads limited |
| **Response Size** | 32 MB | Large responses limited |
| **Concurrent Requests** | 1000 per instance | High concurrency needs multiple instances |
| **Cold Start Time** | 1-5 seconds | Scale-to-zero has latency |

### ArmorClaw-Specific Considerations

✅ **Fully Supported:**
- Docker container deployment
- HTTP/HTTPS communication
- Auto-scaling (zero to N instances)
- Environment variables and secrets
- Cloud SQL integration
- Cloud Storage integration
- Custom domains with SSL

⚠️ **Partial Support:**
- Long-running agent sessions (60-minute timeout)
- Unix socket communication (use Cloud SQL proxy)
- Background processes (use Cloud Run Jobs)

❌ **Not Supported:**
- Docker socket access (not needed on Cloud Run)
- Direct filesystem access (use Cloud Storage)
- Inbound TCP connections (use Cloud Tasks)
- GPU compute (use Vertex AI)

---

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Container Not Starting

**Symptoms:**
- `503 Service Unavailable`
- Revision in `REVISION_STATUS_NOT_SERVING` state

**Solutions:**

1. **Check Revision Logs:**
```bash
gcloud logging read "resource.labels.service_name=armorclaw-bridge" \
  --limit=50 \
  --format="value(textPayload)"
```

2. **Check Resource Limits:**
```bash
gcloud run services describe armorclaw-bridge \
  --format='yaml(spec.template.spec.containers)'
```

3. **Increase Memory/CPU:**
```bash
gcloud run services update armorclaw-bridge \
  --memory=1Gi \
  --cpu=2
```

#### Issue 2: 502/503 Errors

**Symptoms:**
- `502 Bad Gateway`
- `503 Service Unavailable`

**Solutions:**

1. **Check Instance Count:**
```bash
gcloud run services describe armorclaw-bridge \
  --format='value(status.latestReadyRevisionName)'
```

2. **Increase Min Instances:**
```bash
gcloud run services update armorclaw-bridge \
  --min-instances=1
```

3. **Check Health Checks:**
```bash
# Container must respond to HTTP requests on configured port
curl https://armorclaw-bridge-xxxxx-uc.a.run.app/health
```

#### Issue 3: Cold Starts Too Slow

**Symptoms:**
- First request takes 5-30 seconds
- Subsequent requests fast

**Solutions:**

1. **Set Min Instances:**
```bash
gcloud run services update armorclaw-bridge \
  --min-instances=1
```

2. **Optimize Container Image:**
- Reduce image size
- Minimize dependencies
- Use lighter base image

3. **Use CPU Allocation:**
```bash
gcloud run services update armorclaw-bridge \
  --cpu=1 \
  --memory=512Mi
```

#### Issue 4: Database Connection Failures

**Symptoms:**
- `connection refused`
- `timeout connecting to database`

**Solutions:**

1. **Verify Cloud SQL Instance:**
```bash
gcloud sql instances describe armorclaw-db
```

2. **Check VPC Connector:**
```bash
gcloud compute networks vpc-access connectors describe armorclaw-connector \
  --region=us-central1
```

3. **Verify Connection String:**
```
# Correct format:
postgresql://user:password@/cloudsql/project:region:instance/database
```

#### Issue 5: Out of Quota

**Symptoms:**
- `Quota exceeded`
- Deployment fails

**Solutions:**

1. **Check Quotas:**
```bash
gcloud compute regions describe us-central1 \
  --format="value(quotas)"
```

2. **Request Quota Increase:**
- Via Console: IAM & Admin → Quotas
- Select quota type
- Click **"Edit Quotas"**
- Submit increase request

#### Issue 6: Permission Denied

**Symptoms:**
- `Permission 'run.services.update' denied`
- `IAM permission denied`

**Solutions:**

1. **Verify IAM Roles:**
```bash
gcloud projects get-iam-policy armorclaw-deployment \
  --filter="user:your-email@example.com"
```

2. **Grant Necessary Roles:**
```bash
gcloud projects add-iam-policy-binding armorclaw-deployment \
  --member="user:your-email@example.com" \
  --role="roles/run.developer"
```

### Getting Help

**Google Cloud Resources:**
- **Documentation:** https://cloud.google.com/run/docs
- **Support:** https://cloud.google.com/support
- **Community:** https://cloud.google.com/run/community
- **Stack Overflow:** Tag questions with `google-cloud-run`

**ArmorClaw Resources:**
- **Documentation:** https://github.com/armorclaw/armorclaw
- **Issues:** https://github.com/armorclaw/armorclaw/issues

---

## Best Practices

### 1. Optimize for Scale-to-Zero

**Keep containers lightweight:**
- Minimal base images
- Only necessary dependencies
- Efficient startup time

**Set appropriate min/max:**
```bash
--min-instances=0    # Scale to zero
--max-instances=100  # Handle bursts
```

### 2. Use Secrets Sensitive Data

```bash
# Store API keys in Secret Manager
gcloud secrets create API_KEY --data-file=- < key.txt

# Mount in Cloud Run
gcloud run services update SERVICE \
  --update-secrets="API_KEY=API_KEY:latest"
```

### 3. Implement Health Checks

```dockerfile
# In Dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
```

### 4. Use Concurrent Requests

```bash
# Handle multiple requests per instance
--concurrency=80  # Default
```

### 5. Monitor and Alert

- Create Cloud Monitoring dashboards
- Set up alert policies for errors and latency
- Use Cloud Logging for troubleshooting

### 6. Optimize Costs

- Use scale-to-zero for development
- Set appropriate min/max instances
- Monitor usage with Cloud Monitoring
- Use free tier effectively

### 7. Use Cloud Run Jobs for Batch Work

```bash
gcloud run jobs create backup-job \
  --image="${IMAGE_URI}:latest" \
  --command="/app/backup.sh" \
  --memory=1Gi \
  --tasks=1000
```

### 8. Implement Graceful Shutdown

```go
// Handle SIGTERM for graceful shutdown
signal.Notify(context, syscall.SIGTERM)
```

---

## Quick Reference

### Essential Commands

```bash
# Deploy service
gcloud run deploy SERVICE --image=IMAGE

# Update service
gcloud run services update SERVICE --KEY=VALUE

# List services
gcloud run services list

# Describe service
gcloud run services describe SERVICE

# View logs
gcloud logging tail "resource.labels.service_name=SERVICE"

# Set secret
echo "value" | gcloud secrets create SECRET --data-file=-

# Delete service
gcloud run services delete SERVICE
```

### Useful URLs

```
Cloud Console:  https://console.cloud.google.com
Service URL:    https://SERVICE-xxxxx-REGION.a.run.app
Monitoring:     https://console.cloud.google.com/monitoring
Logging:        https://console.cloud.google.com/logging
```

---

## Conclusion

Google Cloud Run provides an excellent serverless platform for ArmorClaw with:

✅ **Generous free tier** - 2M requests/month
✅ **Auto-scaling** - Zero to N instances
✅ **Managed services** - Cloud SQL, Cloud Storage, Secret Manager
✅ **Global network** - 20+ regions
✅ **Pay-per-use** - Only pay for what you use
✅ **Integrated monitoring** - Cloud Logging and Monitoring

**Next Steps:**
1. Complete deployment using this guide
2. Configure Cloud SQL for persistent storage
3. Set up monitoring and alerting
4. Configure custom domain
5. Test auto-scaling behavior

**Related Documentation:**
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud SQL for PostgreSQL](https://cloud.google.com/sql/docs/postgres)
- [Secret Manager](https://cloud.google.com/secret-manager/docs)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)

---

**Document Last Updated:** 2026-02-07
**Google Cloud Run Version:** Based on 2026 pricing and features
**ArmorClaw Version:** 1.2.0
