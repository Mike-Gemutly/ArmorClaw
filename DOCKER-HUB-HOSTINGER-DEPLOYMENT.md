# Docker Hub Setup & Hostinger Deployment Guide

> **Last Updated:** 2026-02-08
> **Purpose:** Setup ArmorClaw Docker Hub registry and deploy to Hostinger VPS
> **Time:** 20-30 minutes
> **Difficulty:** Easy

---

## Quick Start

```bash
# 1. Push image to Docker Hub
export DOCKERHUB_USERNAME=yourusername
./scripts/push-dockerhub.sh

# 2. Deploy to Hostinger VPS
ssh root@your-vps-ip -p 2222
export DOCKERHUB_USERNAME=yourusername
export MATRIX_DOMAIN=matrix.example.com
./scripts/deploy-hostinger-dockerhub.sh
```

---

## Part 1: Docker Hub Setup

### Step 1: Create Docker Hub Account

1. **Sign up for Docker Hub**
   - Visit: https://hub.docker.com/signup
   - Create account (free tier is sufficient)
   - Verify email address

2. **Create repository**
   - Repository name: `agent` (or full path: `yourusername/agent`)
   - Visibility: Public (recommended for testing)

### Step 2: Build and Push Image

**Option A: Use automated script**
```bash
export DOCKERHUB_USERNAME=yourusername
./scripts/push-dockerhub.sh v1.0.0
```

**Option B: Manual build and push**
```bash
# Build Docker image
docker build -t yourusername/agent:latest .

# Login to Docker Hub
docker login

# Push to Docker Hub
docker push yourusername/agent:latest
```

**Option C: GitHub Actions (Recommended)**

Add `.github/workflows/docker-build.yml`:

```yaml
name: Build and Push to Docker Hub

on:
  push:
    branches: [ main ]
    tags: [ 'v*' ]

env:
  DOCKER_IMAGE: yourusername/agent
  PLATFORMS: linux/amd64,linux/arm64

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./Dockerfile
          push: true
          tags: |
            ${{ env.DOCKER_IMAGE }}:latest
            ${{ env.DOCKER_IMAGE }}:${{ github.sha }}
          platforms: ${{ env.PLATFORMS }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

**Set up GitHub Secrets:**
1. Go to repository **Settings** → **Secrets and variables** → **Actions**
2. Add:
   - `DOCKER_USERNAME`: Your Docker Hub username
   - `DOCKER_PASSWORD`: Your Docker Hub access token

### Step 3: Verify Docker Hub Image

```bash
# Test pull
docker pull yourusername/agent:latest

# Verify image
docker images | grep yourusername/agent
```

---

## Part 2: Hostinger VPS Deployment

### Step 1: Prepare Hostinger VPS

1. **Login to Hostinger hPanel**
   - Navigate to **VPS** → **Manage**

2. **Reinstall with Docker Template**
   - Go to **Operating System** tab
   - Select **Docker on Ubuntu 22.04** (or 24.04)
   - Click **Reinstall** (⚠️ This wipes existing data)
   - Wait 5-10 minutes

3. **Configure Firewall**
   - Go to **Firewall** tab
   - Add rules:
     ```
     SSH:   22/tcp
     HTTP:  80/tcp
     HTTPS: 443/tcp
     Matrix: 8448/tcp
     ```

### Step 2: Deploy Using Script

**Option A: Use automated deployment script**
```bash
# SSH into VPS
ssh root@your-vps-ip -p 2222

# Export configuration
export DOCKERHUB_USERNAME=yourusername
export MATRIX_DOMAIN=matrix.example.com

# Run deployment script
./scripts/deploy-hostinger-dockerhub.sh
```

**Option B: Manual deployment**

```bash
# Create deployment directory
mkdir -p /opt/armorclaw
cd /opt/armorclaw

# Pull image from Docker Hub
docker pull yourusername/agent:latest

# Create docker-compose.yml
cat > docker-compose.yml <<'EOF'
version: "3.8"

services:
  matrix:
    image: matrixconduit/matrix-conduit:latest
    container_name: armorclaw-matrix
    restart: unless-stopped
    
    environment:
      CONDUIT_SERVER_NAME: "matrix.example.com"
      CONDUIT_ADDRESS: "0.0.0.0"
      CONDUIT_PORT: "6167"
      CONDUIT_DATABASE_BACKEND: "sqlite"
      CONDUIT_ALLOW_ENCRYPTION: "true"
      CONDUIT_ALLOW_FEDERATION: "true"
    
    volumes:
      - matrix_data:/var/lib/matrix-conduit
    
    ports:
      - "6167:6167"
      - "8448:8448"
    
    networks:
      - armorclaw-net
  
  caddy:
    image: caddy:2-alpine
    container_name: armorclaw-caddy
    restart: unless-stopped
    
    ports:
      - "80:80"
      - "443:443"
    
    volumes:
      - ./configs/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
    
    networks:
      - armorclaw-net
  
  agent:
    image: "yourusername/agent:latest"
    container_name: armorclaw-agent
    restart: unless-stopped
    
    environment:
      ARMORCLAW_MATRIX_HOMESERVER: "https://matrix.example.com"
    
    volumes:
      - ./configs/secrets:/run/secrets:ro
    
    networks:
      - armorclaw-net

networks:
  armorclaw-net:
    driver: bridge

volumes:
  matrix_data:
  caddy_data:
EOF

# Deploy
docker compose up -d
```

### Step 3: Configure DNS

In Hostinger hPanel → **DNS Zone**:

```
Type: A
Name: matrix
Points to: YOUR_VPS_IP
TTL: 3600
```

### Step 4: Verify Deployment

```bash
# Check containers
docker ps

# Test Matrix
curl -I https://matrix.example.com

# View logs
docker logs armorclaw-agent
```

### Step 5: Connect via Element X

1. **Download Element X** app
2. **Login:**
   - Homeserver: `https://matrix.example.com`
   - Create account
3. **Join room:** `#agents`

---

## Update Strategy

```bash
# Pull latest
docker compose pull

# Recreate containers
docker compose up -d --force-recreate
```

---

## Troubleshooting

### Image pull failed
```bash
# Check login
docker login

# Manually pull
docker pull yourusername/agent:latest
```

### Container won't start
```bash
# Check logs
docker logs armorclaw-agent

# Check health
docker inspect armorclaw-agent | grep -A 10 Health
```

### SSL certificate issues
```bash
# Check Caddy logs
docker logs armorclaw-caddy

# Verify DNS
dig +short matrix.example.com
```

---

## Security Best Practices

1. **Use Docker Hub Access Tokens** instead of passwords
2. **Scan images** for vulnerabilities:
   ```bash
   docker scan yourusername/agent:latest
   ```
3. **Enable firewall** in Hostinger hPanel
4. **Regular updates** using automated scripts

---

## Cost Comparison

| Deployment Method | Cost | Build Time | Storage |
|-----------------|-------|-------------|----------|
| Local build & push | Free | 5-10 min | Local |
| GitHub Actions | Free (2000 min) | 2-5 min | None |
| Docker Hub Auto-Build | Free (1 concurrent) | 5-15 min | None |

**Recommendation:** GitHub Actions for faster builds.

---

## Next Steps

1. ✅ Set up GitHub Actions for automated builds
2. ✅ Deploy to Hostinger VPS
3. ✅ Configure DNS and firewall
4. ✅ Connect via Element X
5. ✅ Set up monitoring

---

## Resources

- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
- [GitHub Actions for Docker](https://docs.github.com/en/actions/publishing-packages/publishing-docker-images)
- [Hostinger VPS Documentation](https://www.hostinger.com/tutorials/vps-tutorial)
