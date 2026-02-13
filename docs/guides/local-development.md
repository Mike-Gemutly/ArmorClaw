# Docker Desktop (Local Development) Guide for ArmorClaw

> **Last Updated:** 2026-02-07
> **Platform:** Docker Desktop (Local Machine)
> **Best For:** Local development, testing, CI/CD pipelines
> **Difficulty Level:** Beginner
> **Estimated Time:** 10-15 minutes

---

## Executive Summary

**Docker Desktop** is the easiest way to run ArmorClaw locally for development and testing. It provides complete local control with no infrastructure costs.

### Why Docker Desktop for ArmorClaw?

✅ **Free** - Docker Personal is permanently free
✅ **Full Control** - Complete access to all components
✅ **Fast Iteration** - No network latency
✅ **Offline Development** - Works without internet
✅ **Complete Isolation** - Safe testing environment
✅ **Cross-Platform** - Windows, macOS, Linux

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Local Machine                     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Docker Desktop                       │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │   │
│  │  │   Secure    │  │   Matrix    │  │   Caddy   │  │   │
│  │  │   Claw      │  │   Conduit   │  │ (Proxy)   │  │   │
│  │  │   Bridge    │  │              │  │           │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │   │
│  │                                                     │   │
│  │  - Docker Compose orchestration                     │   │
│  │  - Local volume mounts for development              │   │
│  │  - Hot-reload for code changes                      │   │
│  │  - Complete local environment                       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Local Development Tools                  │   │
│  │  - Code editor (VS Code, etc.)                      │   │
│  │  - Git for version control                          │   │
│  │  - Terminal for commands                            │   │
│  │  - Browser for testing                              │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### 1. Install Docker Desktop

**macOS (Intel):**
```bash
brew install --cask docker
```

**macOS (Apple Silicon):**
```bash
brew install --cask docker
```

**Windows:**
- Download from https://www.docker.com/products/docker-desktop
- Run installer
- Restart computer

**Linux:**
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and log back in
```

### 2. Verify Installation

```bash
docker --version
# Docker version 26.x.x

docker-compose --version
# Docker Compose version v2.x.x
```

### 3. Clone ArmorClaw

```bash
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw
```

### 4. Start Services

```bash
docker-compose up -d
```

### 5. Verify Deployment

```bash
docker ps
# Should show running containers

curl http://localhost:8080/health
# Should return health status
```

---

## Detailed Setup

### 1. System Requirements

**macOS:**
- macOS 11 or newer (Big Sur or later)
- Apple Silicon (M1/M2/M3) or Intel processor
- 4 GB RAM minimum (8 GB recommended)
- 500 MB disk space minimum

**Windows:**
- Windows 10 64-bit: Pro, Enterprise, or Education (Build 16299 or later)
- Windows 11 64-bit: Home or Pro version
- 4 GB RAM minimum (8 GB recommended)
- BIOS-level hardware virtualization support

**Linux:**
- 64-bit distribution
- Kernal version 3.10 or higher
- 4 GB RAM minimum (8 GB recommended)

### 2. Docker Desktop Configuration

**macOS Settings:**
1. Open Docker Desktop
2. Navigate to **Settings** → **Resources**
3. Adjust:
   - **Memory:** 4 GB minimum (8 GB recommended)
   - **CPUs:** 2 minimum (4 recommended)
   - **Disk:** 20 GB minimum

**Windows Settings:**
1. Open Docker Desktop
2. Navigate to **Settings** → **Resources**
3. Adjust:
   - **Memory:** 4 GB minimum
   - **CPUs:** 2 minimum
   - **Disk:** 20 GB minimum
4. Enable **WSL 2** integration for better performance

**Linux:**
No configuration needed. Docker Engine runs directly.

### 3. Docker Compose Configuration

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  bridge:
    build:
      context: .
      dockerfile: Dockerfile.bridge
    ports:
      - "8080:8080"
    volumes:
      - ./bridge:/app
      - armorclaw-data:/run/armorclaw
    environment:
      - ARMORCLAW_ENV=development
      - LOG_LEVEL=debug
    restart: unless-stopped

  matrix:
    image: matrixconduit/matrix:latest
    ports:
      - "6167:6167"
    volumes:
      - matrix-data:/var/lib/conduit
    environment:
      - SERVER_NAME=localhost
    restart: unless-stopped

  caddy:
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./configs/Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
      - caddy-config:/config
    restart: unless-stopped

volumes:
  armorclaw-data:
  matrix-data:
  caddy-data:
  caddy-config:
```

### 4. Development Workflow

**Start All Services:**
```bash
docker-compose up -d
```

**View Logs:**
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f bridge
```

**Restart Service:**
```bash
docker-compose restart bridge
```

**Stop All Services:**
```bash
docker-compose down
```

**Rebuild Service:**
```bash
docker-compose up -d --build bridge
```

### 5. Hot Reload Development

**For Bridge Development:**

1. Edit code locally
2. Rebuild and restart:
```bash
docker-compose up -d --build bridge
```

**For Live Reload (requires inotify-tools):**

```bash
# Install inotify-tools (Linux)
sudo apt-get install inotify-tools

# Watch for changes and rebuild
while inotifywait -e modify -r bridge/; do
  docker-compose up -d --build bridge
done
```

---

## Troubleshooting

### Issue 1: Port Already in Use

**Symptoms:**
```
Error: bind: address already in use
```

**Solutions:**

**Find process using port:**
```bash
# macOS/Linux
lsof -i :8080

# Windows
netstat -ano | findstr :8080
```

**Kill process:**
```bash
# macOS/Linux
kill -9 <PID>

# Windows
taskkill /PID <PID> /F
```

### Issue 2: Container Won't Start

**Symptoms:**
```
Container exited with code 1
```

**Solutions:**

**Check logs:**
```bash
docker logs armorclaw-bridge
```

**Check resource usage:**
```bash
docker stats
```

**Increase Docker resources:**
- Docker Desktop → Settings → Resources → Increase Memory/CPU

### Issue 3: Permission Denied

**Symptoms:**
```
Permission denied (unix:///var/run/docker.sock)
```

**Solutions:**

**Add user to docker group (Linux):**
```bash
sudo usermod -aG docker $USER
# Log out and log back in
```

### Issue 4: Volume Mount Issues

**Symptoms:**
```
Error: volume not found
```

**Solutions:**

**Create volume:**
```bash
docker volume create armorclaw-data
```

**Check volumes:**
```bash
docker volume ls
```

---

## Development Tips

### 1. Use .env File

**Create .env:**
```bash
cat > .env << EOF
ARMORCLAW_ENV=development
LOG_LEVEL=debug
DATABASE_URL=postgresql://user:pass@localhost:5432/armorclaw
API_KEY=sk-test-...
EOF
```

**Use in docker-compose.yml:**
```yaml
services:
  bridge:
    env_file:
      - .env
```

### 2. Local Code Mounts

**Mount local directory:**
```yaml
services:
  bridge:
    volumes:
      - ./bridge:/app/bridge
      - ./pkg:/app/pkg
```

**Code changes reflected immediately** (after restart).

### 3. Debugging

**Attach to container:**
```bash
docker exec -it armorclaw-bridge bash
```

**Run in interactive mode:**
```bash
docker run -it --rm armorclaw-bridge:latest bash
```

### 4. Clean Up

**Remove all containers:**
```bash
docker container prune -f
```

**Remove all images:**
```bash
docker image prune -a -f
```

**Remove all volumes:**
```bash
docker volume prune -f
```

**Factory reset:**
```bash
docker system prune -a --volumes -f
```

---

## Testing

### 1. Unit Tests

```bash
cd bridge
go test ./...
```

### 2. Integration Tests

```bash
./tests/test-secrets.sh
./tests/test-exploits.sh
```

### 3. End-to-End Tests

```bash
./tests/test-e2e.sh
```

---

## Production Deployment

When ready for production, deploy to:
- **Hostinger VPS:** `docs/guides/hostinger-vps-deployment.md`
- **Fly.io:** `docs/guides/flyio-deployment.md`
- **Google Cloud Run:** `docs/guides/gcp-cloudrun-deployment.md`
- **AWS Fargate:** `docs/guides/aws-fargate-deployment.md`

---

## Quick Reference

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs
docker-compose logs -f

# Restart service
docker-compose restart bridge

# Rebuild service
docker-compose up -d --build bridge

# Execute command in container
docker exec -it armorclaw-bridge bash

# Clean up
docker system prune -a --volumes -f
```

---

## Conclusion

Docker Desktop provides the perfect local development environment for ArmorClaw.

**Best For:**
- Local development and testing
- Learning Docker
- Pre-production testing
- CI/CD pipelines
- Offline development

**Advantages:**
- Free for personal use
- Complete control
- Fast iteration
- No infrastructure costs
- Safe isolated environment

**Next Steps:**
1. Install Docker Desktop
2. Clone ArmorClaw repository
3. Start services with `docker-compose up -d`
4. Test and develop locally
5. Deploy to production when ready

**Related Documentation:**
- [Docker Desktop Docs](https://docs.docker.com/desktop/)
- [Production Deployment Guides](docs/guides/)

---

**Document Last Updated:** 2026-02-07
**Docker Version:** 26.x.x
**ArmorClaw Version:** 1.2.0
