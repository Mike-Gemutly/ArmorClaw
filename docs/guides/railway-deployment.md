# Railway Deployment Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Provider:** Railway (https://railway.com)
> **Best For:** Quick deployment, excellent DX, GitHub integration
> **Difficulty Level:** Beginner
> **Estimated Time:** 15-30 minutes

---

## Executive Summary

**Railway** is a modern application deployment platform focused on developer experience. It provides a simple, intuitive interface for deploying apps with minimal configuration.

### Why Railway for ArmorClaw?

✅ **Excellent DX** - Beautiful UI, fast deployments
✅ **GitHub Integration** - Automatic deploys on push
✅ **Built-in Services** - Postgres, Redis, MySQL
✅ **Simple Pricing** - Usage-based, predictable
✅ **Fast Setup** - Deploy in minutes
✅ **Volumes** - Persistent storage included

### Pricing (2026)

| Plan | Base Cost | Includes |
|------|-----------|----------|
| **Free Trial** | $0 (one-time) | $5 credit for 30 days |
| **Hobby** | $5/month | 512 MB RAM, 0.5 vCPU |
| **Pro** | $20/month | 2 GB RAM, 1 vCPU |

---

## Quick Start

### 1. Connect GitHub

1. Visit https://railway.com
2. Click **"New Project"**
3. Select **"Deploy from GitHub"**
4. Authorize Railway
5. Select `armorclaw/armorclaw` repository

### 2. Configure Build

**Railway automatically detects:**
- Dockerfile
- package.json (Node.js)
- requirements.txt (Python)
- go.mod (Go)

**Or configure manually:**
1. Click **"New Service"**
2. Select **"Dockerfile"**
3. Configure build settings

### 3. Set Environment Variables

1. Navigate to **Variables** tab
2. Add variables:
   ```
   ARMORCLAW_ENV=production
   DATABASE_URL=...
   API_KEY=...
   ```

### 4. Deploy

1. Click **"Deploy"**
2. Wait for build (1-3 minutes)
3. Access at: `https://your-app.up.railway.app`

---

## Detailed Deployment

### 1. Dockerfile Deployment

**Create Dockerfile:**
```dockerfile
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go build -o armorclaw-bridge ./bridge/cmd/bridge

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/armorclaw-bridge /app/
EXPOSE 8080
CMD ["/app/armorclaw-bridge"]
```

**Commit and Push:**
```bash
git add Dockerfile
git commit -m "Add Railway Dockerfile"
git push
```

**Railway auto-deploys on push.**

### 2. Add Database

**Create PostgreSQL:**
1. Click **"New Service"**
2. Select **"Database"** → **"PostgreSQL"**
3. Railway provisions database
4. Connection URL added to variables: `DATABASE_URL`

**Connect Bridge:**
```bash
# In Bridge config, use:
DATABASE_URL=${DATABASE_URL}
```

### 3. Persistent Storage

**Add Volume:**
1. Click **"Variables"** tab
2. Click **"New Volume"**
3. Name: `armorclaw-data`
4. Mount path: `/data`
5. Volume persists across deployments

### 4. Custom Domain

**Add Domain:**
1. Click **"Settings"** → **"Domains"**
2. Click **"Add Domain"**
3. Enter: `armorclaw.your-domain.com`
4. Configure DNS (CNAME: `cname.railway.app`)
5. SSL provisioned automatically

---

## Pricing Calculation

**Example: Small Deployment**

```
Components:
- Bridge (Hobby): $5/month
- Postgres: $7/month
- Volume (1 GB): $0.50/month
- Bandwidth: ~$2/month

Total: ~$15/month
```

---

## Limitations

| Limitation | Details |
|------------|---------|
| **Execution Time** | No hard limit |
| **Container Size** | Up to 16 GB (Pro plan) |
| **Regions** | Limited (US/EU only) |
| **GPU Support** | Not available |

---

## Quick Reference

```bash
# Install Railway CLI
npm install -g railway

# Login
railway login

# Initialize project
railway init

# Deploy
railway up

# Open project in browser
railway open
```

---

## Conclusion

Railway is excellent for developers who want to deploy quickly with minimal configuration.

**Best For:**
- Quick prototypes
- Small to medium deployments
- Developers who love good UX
- GitHub-based workflows

**Related Documentation:**
- [Railway Docs](https://docs.railway.com/)
- [DigitalOcean Deployment](docs/guides/digitalocean-deployment.md)

---

**Document Last Updated:** 2026-02-07
**Railway Version:** Based on 2026 pricing
**ArmorClaw Version:** 1.2.0
