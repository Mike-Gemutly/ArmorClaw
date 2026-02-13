# Docker Hub Setup & Hostinger Deployment Guide

> **Last Updated:** 2026-02-08
> **Purpose:** Setup ArmorClaw Docker Hub registry and deploy to Hostinger VPS
> **Time:** 20-30 minutes
> **Difficulty:** Easy

---

## Part 1: Docker Hub Setup

### Step 1: Create Docker Hub Account

1. **Sign up for Docker Hub**
   - Visit: https://hub.docker.com/signup
   - Create account (free tier is sufficient)
   - Verify email address

2. **Plan for your repository:**
   - Repository name: `armorclaw/agent`
   - Visibility: Public (recommended for testing)
   - Description: "ArmorClaw hardened container runtime"

### Step 2: Create Repository

1. **Create new repository**
   - Click **Create Repository**
   - Repository name: `agent` (or full path: `armorclaw/agent`)
   - Visibility: Public
   - Click **Create**

2. **Get your repository URL**
   - Format: `docker.io/yourusername/agent`
   - Example: `docker.io/armorclaw/agent`

### Step 3: Build and Push ArmorClaw Image

**Option A: Build Locally and Push**

```bash
# Clone repository
git clone https://github.com/Mike-Gemutly/ArmorClaw.git
cd ArmorClaw

# Build Docker image
docker build -t armorclaw/agent:latest .

# Tag for Docker Hub (replace YOUR_USERNAME)
docker tag armorclaw/agent:latest YOUR_USERNAME/agent:latest

# Login to Docker Hub
docker login

# Push to Docker Hub
docker push YOUR_USERNAME/agent:latest
```

**Option B: Build using GitHub Actions (Recommended)**

Create `.github/workflows/docker-build.yml`:

```yaml
name: Build and Push to Docker Hub

on:
  push:
    branches: [ main ]
    tags: [ 'v*' ]

env:
  DOCKER_IMAGE: armorclaw/agent
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
2. Add repository secrets:
   - `DOCKER_USERNAME`: Your Docker Hub username
   - `DOCKER_PASSWORD`: Your Docker Hub password (or access token)

**Option C: Automated Builds in Docker Hub**

1. **Connect GitHub to Docker Hub**
   - Go to your repository on Docker Hub
   - Click **Builds** → **Link GitHub Account**
   - Authorize Docker Hub access

2. **Configure build settings**
   - Build Context: `/`
   - Dockerfile Location: `/Dockerfile`
   - Build Triggers: Tag, Active

3. **Build tags**
   - `latest` - triggers on any push to main
   - `v1.0.0` - triggers on git tags

### Step 4: Verify Docker Hub Image

```bash
# Test pull
docker pull YOUR_USERNAME/agent:latest

# Verify image
docker images | grep YOUR_USERNAME/agent

# Test run locally
docker run --rm YOUR_USERNAME/agent:latest --help
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

3. **Configure Firewall in hPanel**
   - Go to **Firewall** tab
   - Add rules:
     ```
     SSH:   22/tcp    (or your custom port)
     HTTP:  80/tcp
     HTTPS: 443/tcp
     Matrix: 8448/tcp  (for federation)
     ```

### Step 2: SSH into VPS

```bash
# SSH to your VPS (replace IP and port)
ssh root@YOUR_VPS_IP -p 2222

# Update system
apt update && apt upgrade -y

# Install Docker Compose (if not pre-installed)
apt install docker-compose-plugin -y

# Verify Docker
docker --version
docker compose version
```

### Step 3: Deploy ArmorClaw from Docker Hub

**Method A: Using Docker Compose (Recommended)**

1. **Create deployment directory**
   ```bash
   mkdir -p /opt/armorclaw
   cd /opt/armorclaw
   ```

2. **Create docker-compose.yml**
   ```bash
   cat > docker-compose.yml <<'EOF'
   version: "3.8"
   
   services:
     # Matrix Conduit
     matrix:
       image: matrixconduit/matrix-conduit:latest
       container_name: armorclaw-matrix
       restart: unless-stopped
       
       environment:
         CONDUIT_SERVER_NAME: "${MATRIX_DOMAIN:-matrix.example.com}"
         CONDUIT_ADDRESS: "0.0.0.0"
         CONDUIT_PORT: "6167"
         CONDUIT_DATABASE_BACKEND: "sqlite"
         CONDUIT_ALLOW_ENCRYPTION: "true"
         CONDUIT_ALLOW_REGISTRATION: "false"
         CONDUIT_ALLOW_FEDERATION: "true"
         CONDUIT_MAX_REQUEST_SIZE: "10485760"
         CONDUIT_LOG: "info"
       
       volumes:
         - matrix_data:/var/lib/matrix-conduit
       
       ports:
         - "6167:6167"
         - "8448:8448"
       
       networks:
         - armorclaw-net
       
       healthcheck:
         test: ["CMD", "curl", "-f", "http://localhost:6167/_matrix/client/versions"]
         interval: 30s
         timeout: 10s
         retries: 3
         start_period: 10s
     
     # Caddy Reverse Proxy (SSL)
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
         - caddy_config:/config
       
       networks:
         - armorclaw-net
       
       depends_on:
         matrix:
           condition: service_healthy
     
     # ArmorClaw Agent Container
     agent:
       image: ${DOCKER_IMAGE:-YOUR_USERNAME/agent:latest}
       container_name: armorclaw-agent
       restart: unless-stopped
       
       environment:
         ARMORCLAW_SECRETS_PATH: "/run/secrets"
         ARMORCLAW_SECRETS_FD: "3"
         PYTHONUNBUFFERED: "1"
         ARMORCLAW_MATRIX_HOMESERVER: "https://${MATRIX_DOMAIN:-matrix.example.com}"
       
       volumes:
         - ./configs/secrets:/run/secrets:ro
         - /tmp:/tmp
       
       networks:
         - armorclaw-net
      
      depends_on:
         - caddy
   
   networks:
     armorclaw-net:
       driver: bridge
   
   volumes:
     matrix_data:
     caddy_data:
     caddy_config:
   EOF
   ```

3. **Create Caddy configuration**
   ```bash
   mkdir -p configs
   
   cat > configs/Caddyfile <<'EOF'
   {
       email admin@example.com
   }
   
   matrix.example.com {
       reverse_proxy matrix:6167
   }
   
   chat.example.com {
       reverse_proxy matrix:8448
   }
   EOF
   ```

4. **Configure environment variables**
   ```bash
   cat > .env <<'EOF'
   DOCKER_IMAGE=YOUR_USERNAME/agent:latest
   MATRIX_DOMAIN=matrix.example.com
   EOF
   
   chmod 600 .env
   ```

5. **Deploy stack**
   ```bash
   docker compose up -d
   
   # Check status
   docker compose ps
   
   # View logs
   docker compose logs -f
   ```

**Method B: Pull and Run Individually**

```bash
# Pull ArmorClaw agent from Docker Hub
docker pull YOUR_USERNAME/agent:latest

# Run Matrix Conduit
docker run -d \
  --name armorclaw-matrix \
  -p 6167:6167 \
  -p 8448:8448 \
  -v matrix_data:/var/lib/matrix-conduit \
  -e SERVER_NAME=matrix.example.com \
  matrixconduit/matrix-conduit:latest

# Run Caddy
docker run -d \
  --name armorclaw-caddy \
  -p 80:80 \
  -p 443:443 \
  -v /opt/armorclaw/configs/Caddyfile:/etc/caddy/Caddyfile:ro \
  -v caddy_data:/data \
  caddy:2-alpine

# Run ArmorClaw Agent
docker run -d \
  --name armorclaw-agent \
  --network host \
  -v /opt/armorclaw/configs/secrets:/run/secrets:ro \
  YOUR_USERNAME/agent:latest
```

### Step 4: Configure DNS

1. **In Hostinger hPanel:**
   - Navigate to **Domains** → **DNS Zone**

2. **Add A records:**
   ```
   Type: A
   Name: matrix
   Points to: YOUR_VPS_IP
   TTL: 3600
   
   Type: A
   Name: chat
   Points to: YOUR_VPS_IP
   TTL: 3600
   ```

3. **Verify DNS propagation:**
   ```bash
   dig +short matrix.example.com
   # Should return your VPS IP
   ```

### Step 5: Verify Deployment

1. **Check container status:**
   ```bash
   docker ps
   ```
   All containers should show "Up" status

2. **Test Matrix server:**
   ```bash
   curl -I https://matrix.example.com
   ```
   Should return HTTP 200

3. **Test ArmorClaw agent:**
   ```bash
   docker logs armorclaw-agent
   ```
   Should show successful startup

### Step 6: Configure Matrix Admin User

1. **Access Matrix container:**
   ```bash
   docker exec -it armorclaw-matrix bash
   ```

2. **Create admin user:**
   ```bash
   # Register admin user
   curl -X POST http://localhost:6167/_matrix/client/v3/register \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin",
       "password": "YOUR_SECURE_PASSWORD",
       "auth": {"type": "m.login.dummy"}
     }'
   ```

3. **Enable registration temporarily** (if needed):
   Edit `docker-compose.yml` and change:
   ```yaml
   CONDUIT_ALLOW_REGISTRATION: "true"
   ```

### Step 7: Connect with Element X

1. **Download Element X** (mobile or desktop)
2. **Login:**
   - Homeserver: `https://matrix.example.com`
   - Username: `admin`
   - Password: (your secure password)
3. **Join or create room:** `#agents:matrix.example.com`

---

## Update Strategy

### Update from Docker Hub

```bash
# Pull latest image
docker compose pull

# Recreate containers
docker compose up -d

# Or force rebuild
docker compose up -d --force-recreate
```

### Automated Updates with Watchtower

```yaml
# Add to docker-compose.yml
  watchtower:
    image: containrrr/watchtower
    container_name: armorclaw-watchtower
    restart: unless-stopped
    
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    
    environment:
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_POLL_INTERVAL=86400  # 24 hours
    
    networks:
      - armorclaw-net
```

---

## Troubleshooting

### Issue: "Permission denied on Docker socket"

**Solution:**
```bash
# Fix Docker socket permissions
sudo chown $USER:$USER /var/run/docker.sock
sudo usermod -aG docker $USER
```

### Issue: "Image pull failed"

**Solution:**
```bash
# Check Docker Hub login
docker logout
docker login

# Manually pull
docker pull YOUR_USERNAME/agent:latest
```

### Issue: "Container exits immediately"

**Solution:**
```bash
# Check logs
docker logs armorclaw-agent

# Check health
docker inspect armorclaw-agent | grep -A 10 Health
```

### Issue: "Caddy SSL certificate fails"

**Solution:**
```bash
# Check Caddy logs
docker logs armorclaw-caddy

# Ensure DNS is correct
dig +short matrix.example.com

# Verify port 80/443 are open
ufw status
```

---

## Security Best Practices

1. **Use Docker Hub Access Tokens**
   - Don't use your password for CI/CD
   - Generate access tokens in Docker Hub settings
   - Use read/write scope for push access

2. **Scan Images for Vulnerabilities**
   ```bash
   docker scan YOUR_USERNAME/agent:latest
   ```

3. **Enable Docker Content Trust**
   ```bash
   export DOCKER_CONTENT_TRUST=1
   docker pull YOUR_USERNAME/agent:latest
   ```

4. **Regular Updates**
   ```bash
   # Auto-update script
   cat > /opt/armorclaw/update.sh <<'EOF'
   #!/bin/bash
   cd /opt/armorclaw
   docker compose pull
   docker compose up -d
   docker image prune -f
   EOF
   
   chmod +x /opt/armorclaw/update.sh
   
   # Add to crontab (daily at 3 AM)
   crontab -l | { cat; echo "0 3 * * * /opt/armorclaw/update.sh"; } | crontab -
   ```

---

## Cost Comparison

| Deployment Method | Docker Hub | Build Time | Storage | Cost |
|-----------------|-------------|-------------|----------|-------|
| Local build & push | Required | 5-10 min | Local | Free |
| GitHub Actions | Required | 2-5 min | None | Free (2000 min/month) |
| Docker Hub Auto-Build | Built-in | 5-15 min | None | Free (1 concurrent) |

**Recommendation:** GitHub Actions for faster builds and better control.

---

## Next Steps

1. ✅ Set up automated builds on GitHub Actions
2. ✅ Configure CI/CD pipeline for testing
3. ✅ Deploy to Hostinger VPS
4. ✅ Set up monitoring and alerts
5. ✅ Configure backup strategy

---

## Resources

- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
- [GitHub Actions for Docker](https://docs.github.com/en/actions/publishing-packages/publishing-docker-images)
- [Hostinger VPS Documentation](https://www.hostinger.com/tutorials/vps-tutorial)
- [ArmorClaw Repository](https://github.com/Mike-Gemutly/ArmorClaw)
