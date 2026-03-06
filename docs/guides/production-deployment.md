# Production Deployment Guide

> **Battle-tested deployment** that avoids all common pitfalls.

This guide provides a clean, production-ready Docker Compose stack for ArmorClaw + Conduit.

## Architecture

```
Internet
   │
   ▼
Conduit (Matrix Server)
   port 6167
   │
   ▼
ArmorClaw Bridge
   port 8080, 8443, 5000
```

**Key benefits:**
- Correct Docker networking (no container namespace hacks)
- Conduit exposed externally on port 6167
- ArmorClaw connects internally via Docker DNS
- Persistent storage
- One command startup (~15 seconds)
- Compatible with Element X

## Quick Start

### 1. Create Directory Structure

```bash
mkdir -p ~/armorclaw-stack
cd ~/armorclaw-stack
mkdir -p conduit-data armorclaw-config
```

### 2. Create docker-compose.yml

```yaml
version: "3.9"

services:

  conduit:
    image: matrixconduit/matrix-conduit:latest
    container_name: conduit
    restart: unless-stopped
    ports:
      - "6167:6167"
    volumes:
      - ./conduit.toml:/etc/conduit.toml:ro
      - ./conduit-data:/var/lib/conduit
    environment:
      - CONDUIT_CONFIG=/etc/conduit.toml
    networks:
      - armorclaw-net

  armorclaw:
    image: mikegemut/armorclaw:latest
    container_name: armorclaw
    restart: unless-stopped
    depends_on:
      - conduit
    ports:
      - "8080:8080"
      - "5000:5000"
      - "8443:8443"
    volumes:
      - ./armorclaw-config:/etc/armorclaw
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - ARMORCLAW_API_KEY=${ARMORCLAW_API_KEY:-}
      - ARMORCLAW_MATRIX_ENABLED=true
      - ARMORCLAW_MATRIX_HOMESERVER_URL=http://conduit:6167
      - ARMORCLAW_MATRIX_USERNAME=bridge
      - ARMORCLAW_MATRIX_PASSWORD=bridgepass
    networks:
      - armorclaw-net

networks:
  armorclaw-net:
    driver: bridge
```

**Important:** The `homeserver_url` uses Docker DNS (`http://conduit:6167`) because containers resolve service names automatically.

### 3. Create conduit.toml

```toml
[global]

# Replace with your server IP or domain
server_name = "YOUR_SERVER_IP"

database_backend = "rocksdb"
database_path = "/var/lib/conduit"

address = "0.0.0.0"
port = 6167

max_request_size = 20000000

allow_registration = true
allow_federation = true
allow_check_for_updates = false

trusted_servers = ["matrix.org"]
```

### 4. Start Everything

```bash
# Set your API key
export ARMORCLAW_API_KEY=sk-your-key

# Start the stack
docker compose up -d
```

### 5. Verify Matrix Server

```bash
curl http://localhost:6167/_matrix/client/versions
```

Expected output:
```json
{"versions":["v1.1","v1.2","v1.3"...]}
```

### 6. Create Bridge User

```bash
curl -X POST http://localhost:6167/_matrix/client/v3/register \
  -H "Content-Type: application/json" \
  -d '{
    "username":"bridge",
    "password":"bridgepass",
    "auth":{"type":"m.login.dummy"}
  }'
```

### 7. Connect Element X

1. Open Element X on your device
2. Homeserver URL: `http://YOUR_SERVER_IP:6167`
3. Login with:
   - Username: `bridge`
   - Password: `bridgepass`
4. Your Matrix ID will be: `@bridge:YOUR_SERVER_IP`

## Troubleshooting

### Bridge can't connect to Matrix

**Problem:** Bridge logs show "connection refused"

**Solution:** Verify the Docker network:
```bash
docker network inspect armorclaw-stack_armorclaw-net
```

Both containers should be on the same network.

### Matrix registration fails

**Problem:** `M_UNKNOWN` error during registration

**Solution:** Ensure `allow_registration = true` in conduit.toml and restart Conduit:
```bash
docker compose restart conduit
```

### Port already in use

**Problem:** `port 6167 already allocated`

**Solution:** Stop any existing containers using that port:
```bash
docker ps | grep 6167
docker rm -f $(docker ps -q --filter "publish=6167")
```

## Why This Setup Works

| Issue | Solution |
|-------|----------|
| localhost inside container | Docker DNS (`conduit:6167`) |
| bridge + conduit networking | shared Docker network |
| port conflicts | explicit port mapping |
| Synapse API mismatch | manual user registration |
| container restarts killing services | separate services |

## Production Enhancements

### Add TLS with Caddy

```yaml
  caddy:
    image: caddy:2-alpine
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
    networks:
      - armorclaw-net
    depends_on:
      - conduit
      - armorclaw
```

### Caddyfile

```
matrix.yourdomain.com {
    handle /_matrix/* {
        reverse_proxy conduit:6167
    }

    handle /.well-known/matrix/client {
        header Content-Type application/json
        header Access-Control-Allow-Origin *
        respond `{"m.homeserver":{"base_url":"https://matrix.yourdomain.com"}}` 200
    }
}
```

## Complete Reset

To start completely fresh:

```bash
docker compose down -v
rm -rf conduit-data armorclaw-config
docker compose up -d
```
