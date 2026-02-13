# DigitalOcean App Platform Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** DigitalOcean (https://www.digitalocean.com)
> **Best For:** Simple PaaS deployment, developer-friendly, predictable pricing
> **Difficulty Level:** Beginner to Intermediate
> **Estimated Time:** 30-45 minutes

---

## Executive Summary

**DigitalOcean App Platform** is a fully managed Platform-as-a-Service (PaaS) for deploying containerized applications. It offers a simple, straightforward approach to deploying apps without managing infrastructure.

### Why DigitalOcean for ArmorClaw?

✅ **Simple Pricing** - Flat monthly rates, no surprises
✅ **Easy Deployment** - Push to deploy from GitHub or Docker registry
✅ **Managed Services** - Databases, object storage, load balancers
✅ **Developer-Friendly** - Clean UI, good documentation
✅ **Global Network** - 12 data centers worldwide
✅ **Good Free Tier** - 3 static sites free

### Pricing (2026)

| Plan | RAM | CPU | Price |
|------|-----|-----|-------|
| **Basic** | 512 MB | 1 | $5/month |
| **Professional** | 1 GB | 1 | $12/month |
| **Professional** | 2 GB | 1 | $32/month |

---

## Quick Start

### 1. Install doctl CLI

```bash
brew install doctl  # macOS
# Or download from https://github.com/digitalocean/doctl/releases
```

### 2. Authenticate

```bash
doctl auth init
```

### 3. Create App

```bash
doctl apps create --spec spec.yaml
```

**spec.yaml:**
```yaml
name: armorclaw-bridge
region: nyc
services:
  - name: bridge
    image:
      registry_type: DOCKER_HUB
      repository: armorclaw/bridge
      tag: latest
    http_port: 8080
    instance_count: 1
    instance_size_slug: basic-xxs
    env_vars:
      - key: ARMORCLAW_ENV
        value: production
```

---

## Detailed Deployment

### 1. Push from Docker Hub

**Build and Push Image:**
```bash
docker build -t your-dockerhub-username/armorclaw:latest .
docker push your-dockerhub-username/armorclaw:latest
```

**Create App from Dashboard:**
1. Navigate to **Apps** → **Create App**
2. Select **Docker Hub**
3. Enter image: `your-dockerhub-username/armorclaw`
4. Select region
5. Configure components
6. Click **Create**

### 2. Push from GitHub

**Connect Repository:**
1. Navigate to **Apps** → **Create App**
2. Select **GitHub**
3. Authorize DigitalOcean
4. Select repository
5. Select branch
6. Configure build settings
7. Deploy

### 3. Configure Environment Variables

**Via Dashboard:**
1. Navigate to **App Settings** → **Environment Variables**
2. Add variables:
   - `ARMORCLAW_ENV=production`
   - `DATABASE_URL=...`
   - `API_KEY=...`

**Via CLI:**
```bash
doctl apps update APP_ID --env-vars ARMORCLAW_ENV=production
```

### 4. Add Database

**Create Managed Database:**
1. Navigate to **Databases** → **Create Database**
2. Select **PostgreSQL**
3. Select plan (Basic: $15/month)
4. Select region (same as app)
5. Create database

**Connect App to Database:**
1. Navigate to **App Settings** → **Components**
2. Add component
3. Select **Database**
4. Link to existing database
5. Connection string added to environment variables

### 5. Custom Domain

**Add Domain:**
1. Navigate to **App Settings** → **Domains**
2. Click **Add Domain**
3. Enter domain: `armorclaw.your-domain.com`
4. Configure DNS (CNAME record provided)
5. SSL provisioned automatically

---

## Limitations

| Limitation | Details |
|------------|---------|
| **Container Size** | Max 1 GiB for Basic plan |
| **Build Time** | 15 minutes max |
| **Execution Time** | No timeout limit (unlike Cloud Run) |
| **GPU Support** | Not available |

---

## Quick Reference

```bash
# Create app
doctl apps create --spec spec.yaml

# List apps
doctl apps list

# Get app info
doctl apps get APP_ID

# Update app
doctl apps update APP_ID --spec spec.yaml

# Delete app
doctl apps delete APP_ID
```

---

## Conclusion

DigitalOcean App Platform is ideal for developers who want simple, predictable PaaS deployment. Good for small to medium ArmorClaw deployments.

**Best For:**
- Beginners
- Small production deployments
- Developers who prefer UI over CLI
- Predictable monthly costs

**Related Documentation:**
- [DigitalOcean Docs](https://docs.digitalocean.com/products/app-platform/)
- [Hostinger VPS Deployment](docs/guides/hostinger-vps-deployment.md)

---

**Document Last Updated:** 2026-02-07
**DigitalOcean Version:** Based on 2026 pricing
**ArmorClaw Version:** 1.2.0
