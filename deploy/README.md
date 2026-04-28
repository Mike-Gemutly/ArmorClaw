# ArmorClaw Deployment Guide

## Deployment Modes

| Mode | Use Case | TLS | Command |
|------|----------|-----|---------|
| **Native** | Development, testing | None | Default |
| **Sentinel** | Production VPS, public access | Let's Encrypt | `bash deploy/install.sh` |
| **Cloudflare Tunnel** | VPS behind NAT/firewall | Cloudflare | `CF_API_TOKEN=xxx bash deploy/install.sh` |
| **Cloudflare Proxy** | Existing Cloudflare setup | Cloudflare | `CF_MODE=proxy bash deploy/install.sh` |
| **Self-Hosted** | Home server, LAN-only | Self-signed | `bash deploy/deploy-selfhosted.sh --auto` |

## Email Pipeline (Postfix)

ArmorClaw supports inbound email processing via Postfix:

1. Install Postfix and mta-recv: `sudo bash deploy/postfix/install.sh`
2. Verify setup: `sudo bash deploy/postfix/verify-setup.sh`
3. Start the bridge (creates email-ingest.sock)
4. Email flows: Postfix -> mta-recv -> IngestServer -> YARA -> PII -> Secretary workflow

**Key files:**
- `deploy/postfix/main.cf` -- Postfix configuration
- `deploy/postfix/master.cf` -- armorclaw pipe transport
- `deploy/postfix/transport_maps` -- Email routing rules
- `deploy/postfix/install.sh` -- Idempotent installer
- `deploy/postfix/verify-setup.sh` -- Health check script

## Self-Hosted Appliance

Single VPS / home server setup with mDNS discovery:

```bash
# Quick start
sudo bash deploy/deploy-selfhosted.sh --auto

# With custom hostname
sudo ARMORCLAW_HOSTNAME=myserver.local bash deploy/deploy-selfhosted.sh --auto
```

**Files:**
- `docker-compose.selfhosted.yml` -- Docker Compose for single VPS
- `configs/Caddyfile.selfhosted` -- Reverse proxy config
- `deploy/scripts/generate-certs.sh` -- Self-signed cert generation
- `.env.selfhosted` -- Environment template

**Cert rotation:**
```bash
sudo bash deploy/scripts/generate-certs.sh --rotate --output /etc/armorclaw/certs
```

## Sidecar Binary

Build and test the Rust document processing sidecar:

```bash
cd sidecar
cargo build --bin armorclaw-sidecar           # Dev build
cargo build --release --bin armorclaw-sidecar  # Release build (needs cmake + clang)
cargo test --lib                                # Run tests
```
