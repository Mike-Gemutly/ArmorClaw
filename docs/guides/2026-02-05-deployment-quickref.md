# ArmorClaw Deployment Quick Reference

**Stage 1: Infrastructure Deployment**

---

## Pre-Deployment Checklist

- [ ] DNS A record configured → Hostinger IP
- [ ] SSH access to server
- [ ] 4 GB RAM available
- [ ] Docker installed (`curl -fsSL https://get.docker.com | sh`)
- [ ] docker-compose installed (`apt install docker-compose`)

---

## Deployment Commands

```bash
# 1. Navigate to project
cd /path/to/ArmorClaw

# 2. Copy environment file
cp .env.example .env
nano .env  # Edit your domain, email, etc.

# 3. Deploy infrastructure
./scripts/deploy-infrastructure.sh

# 4. Validate deployment
./scripts/validate-infrastructure.sh
```

---

## Critical URLs

| Service | URL | Purpose |
|---------|-----|---------|
| Matrix API | `http://localhost:6167/_matrix/client/versions` | Health check |
| Nginx Health | `http://localhost/health` | Health check |
| Element Web | `https://app.element.io` | Register admin |
| Federation | `https://matrix.armorclaw.com/.well-known/matrix/server` | Delegation |

---

## Service Management

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# Restart specific service
docker-compose restart matrix-conduit

# View logs
docker-compose logs -f matrix-conduit

# Check status
docker-compose ps
```

---

## Initial Setup Flow

```
1. Set allow_registration = true in configs/conduit.toml
2. docker-compose restart matrix-conduit
3. Open https://app.element.io
4. Set homeserver: https://matrix.armorclaw.com
5. Register admin account
6. Set allow_registration = false
7. docker-compose restart matrix-conduit
8. Create test room → Note room ID for bridge testing
```

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Container won't start | `docker-compose logs matrix-conduit` |
| DNS not resolving | `dig +short matrix.armorclaw.com` |
| SSL error | `sudo certbot certonly --standalone -d matrix.armorclaw.com` |
| High memory | `docker stats` (should be < 700 MB total) |
| Can't connect | Check UFW: `ufw allow 80/tcp && ufw allow 443/tcp` |

---

## Success Validation

```bash
# All checks should pass
./scripts/validate-infrastructure.sh

# Expected output:
# ✓ Matrix Conduit is responding
# ✓ Nginx is responding
# ✓ SSL certificate valid (if enabled)
# All checks passed!
```

---

## Next Stage: Bridge Development

After infrastructure is validated:

```bash
# Initialize Go repository
./scripts/init-bridge-repo.sh

# Begin implementation
cd bridge
make deps
make build
```

---

**Memory Budget:** 690 MB allocated for infrastructure, ~1.3 GB remaining for bridge + agent
