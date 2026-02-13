# Render Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Render (https://render.com)
> **Best For:** Free tier for testing, simple deployment from Git
> **Difficulty Level:** Beginner
> **Estimated Time:** 15-30 minutes

---

## Executive Summary

**Render** is a modern cloud platform that makes it easy to deploy and manage apps. With a generous free tier and excellent Git integration, it's ideal for testing and development.

### Why Render for ArmorClaw?

✅ **Free Tier** - Limited but usable for testing
✅ **Git Integration** - Deploy on push
✅ **SSL Included** - Automatic HTTPS
✅ **Multiple Services** - Web services, workers, cron jobs
✅ **Simple Pricing** - No surprise bills

### Pricing (2026)

| Plan | Price | Includes |
|------|-------|----------|
| **Free** | $0 | 750 hours/month (spins down after 15 min inactivity) |
| **Starter** | $7/month | Always on, 0.5 GB RAM |
| **Standard** | $25/month | 2 GB RAM, more CPU |
| **Pro** | $85/month | 8 GB RAM, dedicated CPU |

---

## Quick Start

### 1. Create Render Account

1. Visit https://render.com
2. Sign up with GitHub, GitLab, or email
3. Authorize Render to access your repositories

### 2. Create New Web Service

1. Click **"New"** → **"Web Service"**
2. Connect GitHub/GitLab repository
3. Select `armorclaw` repository
4. Select branch (default: `main`)
5. Configure build settings

### 3. Configure Build

**For Dockerfile deployment:**
- Runtime: **Docker**
- Dockerfile Path: `./Dockerfile`
- Docker Context: `/`

**For deployment from registry:**
- Runtime: **Docker**
- Image URL: `your-dockerhub-username/armorclaw:latest`

### 4. Configure Service

- **Name:** `armorclaw-bridge`
- **Region:** Oregon (us-west) or Frankfurt (eu-central)
- **Branch:** `main`
- **Root Directory:** `/`
- **Command:** (leave blank for Docker CMD)

### 5. Set Environment Variables

**Add in Environment tab:**
```
PORT=8080
ARMORCLAW_ENV=production
```

### 6. Deploy

Click **"Create Web Service"**

Render builds and deploys. URL: `https://armorclaw-bridge.onrender.com`

---

## Detailed Deployment

### 1. Dockerfile Configuration

**Create Dockerfile:**
```dockerfile
FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY . .
RUN go build -o armorclaw-bridge ./bridge/cmd/bridge

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/armorclaw-bridge /app/
EXPOSE 8080
CMD ["/app/armorclaw-bridge", "--server-addr=:8080"]
```

**Important:** Must expose port 8080 (or use PORT env variable).

### 2. Background Workers

**Create Background Worker:**
1. Click **"New"** → **"Cron Job"**
2. Connect repository
3. Configure: `* * * * *` (every minute)
4. Command: `/app/worker`

### 3. Databases

**Create PostgreSQL:**
1. Click **"New"** → **"PostgreSQL"**
2. Name: `armorclaw-db`
3. Database: `armorclaw`
4. User: `armorclaw`
5. Region: Same as web service
6. Plan: Free (limited) or Starter ($20/month)

**Connection:**
- Internal URL: `postgresql://user:pass@host:5432/dbname`
- External URL: Available in dashboard

### 4. Persistent Disk

**Add Disk:**
1. Navigate to service
2. Click **"Advanced"** → **"Add Disk"**
3. Name: `armorclaw-data`
4. Mount path: `/data`
5. Size: 1 GB minimum

**Note:** Disks not available on free tier.

### 5. Custom Domain

**Add Domain:**
1. Navigate to service
2. Click **"Custom Domain"**
3. Enter: `armorclaw.your-domain.com`
4. Add CNAME record: `cname.render.com`
5. SSL provisioned automatically

---

## Limitations

| Limitation | Free Tier | Paid Plans |
|------------|-----------|------------|
| **Spin Down** | After 15 min inactivity | No |
| **Execution Time** | No limit | No limit |
| **Container Size** | 512 MB RAM | Up to 8 GB |
| **Disks** | Not available | Yes |

**Note:** Free tier NOT suitable for persistent Bridge (spins down).

---

## Pricing Calculation

**Example: Small Deployment**

```
Components:
- Web Service (Starter): $7/month
- PostgreSQL (Starter): $20/month
- Disk (1 GB): $1/month

Total: ~$28/month
```

---

## Quick Reference

```bash
# Install Render CLI
npm install -g render-cli

# Login
render login

# Deploy
render deploy

# View logs
render logs
```

---

## Conclusion

Render offers a simple deployment option with a free tier for testing.

**Best For:**
- Testing and development
- Small production deployments
- Users who want Git-based deployment
- Projects with simple requirements

**Caveats:**
- Free tier spins down (not suitable for persistent Bridge)
- Limited regions (US and EU only)
- More expensive than alternatives for production

**Related Documentation:**
- [Render Docs](https://render.com/docs)
- [Railway Deployment](docs/guides/railway-deployment.md)

---

**Document Last Updated:** 2026-02-07
**Render Version:** Based on 2026 pricing
**ArmorClaw Version:** 1.2.0
